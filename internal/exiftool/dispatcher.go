package exiftool

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vincentchyu/photools/internal/domain"
)

// GPSWritePayload 统领全库所有 GPS 写入操作的数据载荷
type GPSWritePayload struct {
	// SourcePhotoPath 机身全量克隆源（优先：无损克隆 13+ 项物理 GPS 字段）
	SourcePhotoPath string

	// 坐标与海拔数值（当 SourcePhotoPath 为空或无效时的回退/推算载荷）
	Latitude  float64
	Longitude float64
	Altitude  float64

	// Provenance 溯源审计指纹
	Provenance domain.GPSProvenance
}

// HasCloneSource 是否具备可直接克隆的源照片
func (p GPSWritePayload) HasCloneSource() bool {
	if p.SourcePhotoPath == "" {
		return false
	}
	info, err := os.Stat(p.SourcePhotoPath)
	return err == nil && !info.IsDir()
}

// FindAssetGroupForPath 辅助函数：根据任意单个文件路径智能发现同目录下配对的 RAW/JPG/XMP 伴随文件，构造完整的 AssetGroup
func FindAssetGroupForPath(filePath string) domain.AssetGroup {
	if filePath == "" {
		return domain.AssetGroup{}
	}

	cleanPath := filepath.Clean(filePath)
	dir := filepath.Dir(cleanPath)
	ext := filepath.Ext(cleanPath)

	// 处理复合扩展名（例如 DSC_001.nef.xmp）
	nameWithoutExt := strings.TrimSuffix(filepath.Base(cleanPath), ext)
	subExt := filepath.Ext(nameWithoutExt)
	baseNoExt := nameWithoutExt
	if subExt != "" {
		baseNoExt = strings.TrimSuffix(nameWithoutExt, subExt)
	}

	group := domain.AssetGroup{
		BaseName: baseNoExt,
		Dir:      dir,
	}

	isRaw := func(e string) bool {
		switch strings.ToLower(strings.TrimPrefix(e, ".")) {
		case "nef", "cr3", "cr2", "arw", "dng", "raf", "rw2", "orf":
			return true
		default:
			return false
		}
	}

	isJPG := func(e string) bool {
		norm := strings.ToLower(strings.TrimPrefix(e, "."))
		return norm == "jpg" || norm == "jpeg"
	}

	isXMP := func(e string) bool {
		return strings.ToLower(strings.TrimPrefix(e, ".")) == "xmp"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if isRaw(ext) {
			group.RawPath = cleanPath
		} else if isJPG(ext) {
			group.JPGPath = cleanPath
		} else if isXMP(ext) {
			group.XMPPath = cleanPath
		}
		return group
	}

	targetBaseLower := strings.ToLower(baseNoExt)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		currExt := filepath.Ext(name)
		currNameNoExt := strings.TrimSuffix(name, currExt)
		currSubExt := filepath.Ext(currNameNoExt)
		currBase := currNameNoExt
		if currSubExt != "" {
			currBase = strings.TrimSuffix(currNameNoExt, currSubExt)
		}

		if strings.ToLower(currBase) != targetBaseLower {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if isRaw(currExt) {
			group.RawPath = fullPath
		} else if isJPG(currExt) {
			group.JPGPath = fullPath
		} else if isXMP(currExt) {
			// 如果已有 XMP，且当前是复合后缀 (如 .nef.xmp)，优先保留更具体的匹配
			if group.XMPPath == "" || strings.HasSuffix(strings.ToLower(name), ".nef.xmp") {
				group.XMPPath = fullPath
			}
		} else {
			group.CompanionPaths = append(group.CompanionPaths, fullPath)
		}
	}

	return group
}

// VerifyGPSTags 执行二次读取校验，确认目标文件（媒体或 XMP）中确实已写入有效 GPSPosition 或 GPSLatitude
func VerifyGPSTags(runner CommandRunner, filePath string) error {
	if filePath == "" {
		return errors.New("二次校验路径为空")
	}
	checkOut, _ := runner.CombinedOutput("exiftool", "-m", "-q", "-s3", "-GPSPosition", filePath)
	if len(bytes.TrimSpace(checkOut)) == 0 {
		checkLat, _ := runner.CombinedOutput("exiftool", "-m", "-q", "-s3", "-GPSLatitude", filePath)
		if len(bytes.TrimSpace(checkLat)) == 0 {
			return fmt.Errorf("GPS 二次校验失败：未在目标文件 %s 中检测到有效 GPS 坐标", filePath)
		}
	}
	return nil
}

// VerifyLocationTags 执行二次读取校验，确认目标文件（媒体或 XMP）中确实已写入有效国家/城市等派生地名，且无重复标签树
func VerifyLocationTags(runner CommandRunner, filePath string) error {
	if filePath == "" {
		return errors.New("二次校验路径为空")
	}
	checkOut, _ := runner.CombinedOutput("exiftool", "-m", "-q", "-s3", "-City", "-Country", "-State", filePath)
	if len(bytes.TrimSpace(checkOut)) == 0 {
		return fmt.Errorf("地名二次校验失败：未在目标文件 %s 中检测到有效城市或国家元数据", filePath)
	}

	tags, err := ReadLocationTags(runner, filePath)
	if err == nil && len(tags.HierarchicalSubject) > 1 {
		seen := make(map[string]bool)
		for _, tree := range tags.HierarchicalSubject {
			if seen[tree] {
				return fmt.Errorf("地名二次校验失败：目标文件 %s 存在重复分层标签树: %s", filePath, tree)
			}
			seen[tree] = true
		}
	}

	return nil
}

// WriteGPS 全局唯一第二层修正事实调度引擎：将 GPS 元数据严格按照 SidecarPolicy 调度写入摄影资产组
func WriteGPS(
	runner CommandRunner, asset domain.AssetGroup, payload GPSWritePayload, policy domain.SidecarPolicy,
) ([]string, error) {
	primary := asset.PrimaryPath()
	if primary == "" {
		return nil, errors.New("资产组缺少有效的主媒体文件")
	}

	normPolicy := domain.NormalizePolicy(string(policy))
	canClone := payload.HasCloneSource()
	src := payload.SourcePhotoPath

	// 原子操作辅助函数
	writeMedia := func(target string) error {
		if canClone && src != target {
			return CloneAllGPSMetadata(runner, src, target)
		}
		return WriteCoordinates(runner, target, payload.Latitude, payload.Longitude, payload.Altitude)
	}

	writeXMP := func(targetXMP string) error {
		if canClone && src != "" {
			return SyncGPSToXMPWithProvenance(runner, src, targetXMP, payload.Provenance)
		}
		return WriteCoordinatesToXMPWithProvenance(
			runner, targetXMP, payload.Latitude, payload.Longitude, payload.Altitude, payload.Provenance,
		)
	}

	var modified []string
	record := func(path string) {
		if path != "" && !slices.Contains(modified, path) {
			modified = append(modified, path)
		}
	}

	switch normPolicy {
	case domain.PolicySidecarOnly:
		// 纯侧车模式：RAW/JPG 原图 100% 保持只读！全部只写配套 .xmp 侧车文件
		targetXMP := asset.SidecarPathFor(primary)
		if asset.HasXMP() {
			targetXMP = asset.XMPPath
		}
		if err := writeXMP(targetXMP); err != nil {
			return nil, fmt.Errorf("写入 XMP 侧车失败: %w", err)
		}
		record(targetXMP)

		// 二次校验侧车
		if err := VerifyGPSTags(runner, targetXMP); err != nil {
			return nil, err
		}

	case domain.PolicyEmbedOnly:
		// 纯原图内嵌模式：仅写 RAW/JPG，严禁触碰或生成任何 XMP
		if asset.HasRaw() {
			if err := writeMedia(asset.RawPath); err != nil {
				return nil, fmt.Errorf("写入 RAW GPS 失败: %w", err)
			}
			record(asset.RawPath)
		}
		if asset.HasJPG() {
			if err := writeMedia(asset.JPGPath); err != nil {
				return nil, fmt.Errorf("写入 JPG GPS 失败: %w", err)
			}
			record(asset.JPGPath)
		}
		if !asset.HasRaw() && !asset.HasJPG() {
			if err := writeMedia(primary); err != nil {
				return nil, fmt.Errorf("写入原图 GPS 失败: %w", err)
			}
			record(primary)
		}

		// 二次校验主文件
		if err := VerifyGPSTags(runner, primary); err != nil {
			return nil, err
		}

	case domain.PolicyEmbedAndSidecar:
		// 双写模式：写入 RAW/JPG + 同步更新 XMP
		if asset.HasRaw() {
			if err := writeMedia(asset.RawPath); err != nil {
				return nil, fmt.Errorf("写入 RAW GPS 失败: %w", err)
			}
			record(asset.RawPath)
		}
		if asset.HasJPG() {
			if err := writeMedia(asset.JPGPath); err != nil {
				return nil, fmt.Errorf("写入 JPG GPS 失败: %w", err)
			}
			record(asset.JPGPath)
		}
		if !asset.HasRaw() && !asset.HasJPG() {
			if err := writeMedia(primary); err != nil {
				return nil, fmt.Errorf("写入原图 GPS 失败: %w", err)
			}
			record(primary)
		}

		targetXMP := asset.XMPPath
		if targetXMP == "" {
			targetXMP = asset.SidecarPathFor(primary)
		}
		if err := writeXMP(targetXMP); err != nil {
			return nil, fmt.Errorf("同步 XMP 侧车失败: %w", err)
		}
		record(targetXMP)

		// 二次校验主文件
		if err := VerifyGPSTags(runner, primary); err != nil {
			return nil, err
		}

	default: // PolicySmart (智能分层模式，黄金平衡点)
		// 1. RAW EXIF 头部：写入/克隆全量高精度标准 GPS (RAW 永久拥有物理 GPS)
		if asset.HasRaw() {
			if err := writeMedia(asset.RawPath); err != nil {
				return nil, fmt.Errorf("写入 RAW GPS 失败: %w", err)
			}
			record(asset.RawPath)
		}

		// 2. 伴随/独立 JPG：直接内嵌写入/克隆 (跨设备即开即看)
		if asset.HasJPG() {
			if err := writeMedia(asset.JPGPath); err != nil {
				return nil, fmt.Errorf("写入 JPG GPS 失败: %w", err)
			}
			record(asset.JPGPath)
		}

		if !asset.HasRaw() && !asset.HasJPG() {
			if err := writeMedia(primary); err != nil {
				return nil, fmt.Errorf("写入原图 GPS 失败: %w", err)
			}
			record(primary)
		}

		// 3. 配套 XMP 侧车：
		// - 若包含 RAW：必定写入/生成 .nef.xmp 侧车并留下指纹 (RAW 原生级别标准工作流)
		// - 若仅单 JPG：仅在原本已存在 XMP 侧车时同步更新，避免为单 JPG 凭空产生垃圾 .jpg.xmp
		shouldWriteXMP := asset.HasRaw() || asset.HasXMP()
		if shouldWriteXMP {
			targetXMP := asset.XMPPath
			if targetXMP == "" {
				targetXMP = asset.SidecarPathFor(primary)
			}
			if err := writeXMP(targetXMP); err != nil {
				return nil, fmt.Errorf("写入带指纹的 XMP 侧车失败: %w", err)
			}
			record(targetXMP)
		}

		// 4. 二次读取校验主文件
		if err := VerifyGPSTags(runner, primary); err != nil {
			return nil, err
		}
	}

	return modified, nil
}

// WriteLocation 全局唯一第三层派生信息调度引擎：将逆地理中文地名严格按照 SidecarPolicy 调度写入摄影资产组
func WriteLocation(
	runner CommandRunner, asset domain.AssetGroup, loc domain.LocationInfo, policy domain.SidecarPolicy,
) ([]string, error) {
	primary := asset.PrimaryPath()
	if primary == "" {
		return nil, errors.New("资产组缺少有效的主媒体文件")
	}

	normPolicy := domain.NormalizePolicy(string(policy))
	var modified []string
	record := func(path string) {
		if path != "" && !slices.Contains(modified, path) {
			modified = append(modified, path)
		}
	}

	switch normPolicy {
	case domain.PolicySidecarOnly:
		// 纯侧车模式：原图 100% 只读，地名仅写入 .xmp 侧车
		targetXMP := asset.SidecarPathFor(primary)
		if asset.HasXMP() {
			targetXMP = asset.XMPPath
		}
		if err := WriteLocationToXMP(runner, targetXMP, loc); err != nil {
			return nil, fmt.Errorf("写入 XMP 地名失败 (%s): %w", targetXMP, err)
		}
		record(targetXMP)

		if asset.HasRaw() && asset.HasJPG() {
			jpgXMP := asset.SidecarPathFor(asset.JPGPath)
			if jpgXMP != targetXMP {
				_ = WriteLocationToXMP(runner, jpgXMP, loc)
				record(jpgXMP)
			}
		}

	case domain.PolicyEmbedOnly:
		// 纯内嵌模式：地名直接写入主文件与伴随 JPG，不写 XMP
		if err := WriteLocationToMedia(runner, primary, loc); err != nil {
			return nil, fmt.Errorf("写入主文件地名失败 (%s): %w", primary, err)
		}
		record(primary)

		if asset.HasRaw() && asset.HasJPG() {
			if err := SyncLocationToJPG(runner, asset.RawPath, asset.JPGPath); err != nil {
				return nil, fmt.Errorf("同步地名到 JPG 失败: %w", err)
			}
			record(asset.JPGPath)
		}

	case domain.PolicyEmbedAndSidecar:
		// 双写模式：内嵌写入主文件 + 同步写入 XMP
		if err := WriteLocationToMedia(runner, primary, loc); err != nil {
			return nil, fmt.Errorf("写入主文件地名失败 (%s): %w", primary, err)
		}
		record(primary)

		if asset.HasRaw() && asset.HasJPG() {
			if err := SyncLocationToJPG(runner, asset.RawPath, asset.JPGPath); err != nil {
				return nil, fmt.Errorf("同步地名到 JPG 失败: %w", err)
			}
			record(asset.JPGPath)
		}

		targetXMP := asset.XMPPath
		if targetXMP == "" {
			targetXMP = asset.SidecarPathFor(primary)
		}
		if err := WriteLocationToXMP(runner, targetXMP, loc); err != nil {
			return nil, fmt.Errorf("同步地名到 XMP 失败 (%s): %w", targetXMP, err)
		}
		record(targetXMP)

	default: // PolicySmart (智能分层模式：第三层派生地名绝不触碰 RAW，RAW 只写 XMP，JPG 内嵌)
		if asset.HasRaw() {
			targetXMP := asset.SidecarPathFor(asset.RawPath)
			if asset.HasXMP() {
				targetXMP = asset.XMPPath
			}
			if err := WriteLocationToXMP(runner, targetXMP, loc); err != nil {
				return nil, fmt.Errorf("RAW 写入 XMP 地名失败 (%s): %w", targetXMP, err)
			}
			record(targetXMP)

			if asset.HasJPG() {
				if err := WriteLocationToMedia(runner, asset.JPGPath, loc); err != nil {
					return nil, fmt.Errorf("写入 JPG 地名失败 (%s): %w", asset.JPGPath, err)
				}
				record(asset.JPGPath)
			}
		} else {
			if err := WriteLocationToMedia(runner, primary, loc); err != nil {
				return nil, fmt.Errorf("写入 JPG 地名失败 (%s): %w", primary, err)
			}
			record(primary)
		}
	}

	return modified, nil
}
