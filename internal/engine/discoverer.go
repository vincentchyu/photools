package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vincentchyu/photools/internal/domain"
)

// Discoverer 负责扫描指定目录并聚合 AssetGroup
type Discoverer struct {
	rawExtensions map[string]struct{}
}

// NewDiscoverer 创建 Discoverer 实例
func NewDiscoverer(rawExts []string) *Discoverer {
	rawMap := make(map[string]struct{}, len(rawExts))
	for _, ext := range rawExts {
		clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
		if clean != "" {
			rawMap[clean] = struct{}{}
		}
	}
	return &Discoverer{rawExtensions: rawMap}
}

// 忽略的非摄影/文档/日志等文件扩展名（不作为伴随文件与独立资产处理）
var ignoredDocExtensions = map[string]struct{}{
	"md": {}, "log": {}, "txt": {}, "json": {}, "yaml": {}, "yml": {},
	"csv": {}, "pdf": {}, "doc": {}, "docx": {}, "zip": {}, "tar": {},
	"gz": {}, "7z": {}, "rar": {}, "bak": {}, "tmp": {}, "ds_store": {},
	"toml": {}, "sh": {}, "bat": {},
}

func isIgnoredDocExt(ext string) bool {
	_, ok := ignoredDocExtensions[strings.ToLower(ext)]
	return ok
}

// IsIgnoredDir 判断是否为应屏蔽扫描的系统/备份/归档/轨迹/日志目录
func IsIgnoredDir(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, ".") {
		return true
	}
	if lower == "inbox_bak" || lower == "bak" || lower == "backup" || lower == "backups" ||
		strings.HasSuffix(lower, "_bak") || strings.HasSuffix(lower, "_backup") {
		return true
	}
	if lower == "processed" || lower == "gpx" || lower == "logs" || lower == "node_modules" || lower == "dist" {
		return true
	}
	return false
}

// Discover 递归扫描 sourceDir 并返回按目录和 BaseName 排序的 AssetGroup
func (d *Discoverer) Discover(sourceDir string) ([]domain.AssetGroup, error) {
	fi, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("待处理照片源目录不存在: %s (请检查工作区设置)", sourceDir)
		}
		return nil, fmt.Errorf("无法访问照片源目录 %s: %w", sourceDir, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("照片源路径不是有效目录: %s", sourceDir)
	}

	groups := map[string]*domain.AssetGroup{}
	cleanSourceDir := filepath.Clean(sourceDir)

	err = filepath.WalkDir(
		sourceDir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// 如果是源根目录本身，允许进入
				if filepath.Clean(path) == cleanSourceDir {
					return nil
				}
				// 忽略隐藏目录与备份/归档/轨迹/日志等目录
				if IsIgnoredDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			// 忽略隐藏文件与系统文件
			if strings.HasPrefix(entry.Name(), ".") {
				return nil
			}

			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(entry.Name()), "."))
			if isIgnoredDocExt(ext) {
				return nil
			}

			baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			dir := filepath.Dir(path)
			key := dir + "::" + strings.ToLower(baseName)

			group, ok := groups[key]
			if !ok {
				group = &domain.AssetGroup{
					BaseName: baseName,
					Dir:      dir,
				}
				groups[key] = group
			}

			switch {
			case d.isRawExt(ext):
				group.RawPath = path
			case ext == "jpg" || ext == "jpeg":
				group.JPGPath = path
			default:
				if strings.EqualFold(ext, "xmp") {
					group.XMPPath = path
				}
				group.CompanionPaths = append(group.CompanionPaths, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	var assets []domain.AssetGroup
	for _, group := range groups {
		// 摄影资产必须至少包含 RAW 或 JPG 主文件
		if !group.HasRaw() && !group.HasJPG() {
			continue
		}
		assets = append(assets, *group)
	}

	sort.Slice(
		assets, func(i, j int) bool {
			if assets[i].Dir != assets[j].Dir {
				return assets[i].Dir < assets[j].Dir
			}
			return assets[i].BaseName < assets[j].BaseName
		},
	)

	for i := range assets {
		sort.Strings(assets[i].CompanionPaths)
	}

	return assets, nil
}

func (d *Discoverer) isRawExt(ext string) bool {
	_, ok := d.rawExtensions[strings.ToLower(ext)]
	return ok
}

// IsAllowedGPXTrack 判断文件名是否为受支持的 GPX 运动轨迹（仅限 hiking 与 walking，且后缀为 .gpx）
func IsAllowedGPXTrack(filename string) bool {
	if strings.HasPrefix(filename, ".") {
		return false
	}
	if !strings.EqualFold(filepath.Ext(filename), ".gpx") {
		return false
	}
	base := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	return strings.HasPrefix(base, "hiking") || strings.HasPrefix(base, "walking")
}

// ListGPXFiles 扫描指定目录下的全部有效 .gpx 轨迹文件（不递归子目录，仅保留 hiking 与 walking 轨迹）
func ListGPXFiles(gpxDir string) ([]string, error) {
	if _, err := os.Stat(gpxDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	entries, err := os.ReadDir(gpxDir)
	if err != nil {
		return nil, err
	}

	var gpxFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue // 严格不递归子目录
		}
		if IsAllowedGPXTrack(entry.Name()) {
			gpxFiles = append(gpxFiles, filepath.Join(gpxDir, entry.Name()))
		}
	}

	sort.Strings(gpxFiles)
	return gpxFiles, nil
}
