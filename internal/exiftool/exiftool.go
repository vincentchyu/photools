package exiftool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

type Metadata = domain.Metadata

type CommandRunner interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// ReadMetadata 读取基础 EXIF 元数据
func ReadMetadata(runner CommandRunner, path string) (Metadata, error) {
	output, err := runner.CombinedOutput(
		"exiftool", "-m", "-q", "-json", "-DateTimeOriginal", "-OffsetTimeOriginal", "-GPSPosition", "-GPSDateTime",
		path,
	)
	if len(output) > 0 {
		if meta, parseErr := ParseMetadataJSON(output); parseErr == nil {
			return meta, nil
		}
	}
	if err != nil {
		return Metadata{}, fmt.Errorf("exiftool 读取元数据失败: %w", err)
	}
	return ParseMetadataJSON(output)
}

// ReadBatchMetadataMap 批量读取多个照片文件的元数据，返回以规范化路径为 key 的映射
func ReadBatchMetadataMap(runner CommandRunner, paths []string) (map[string]Metadata, error) {
	return ReadBatchMetadataMapConcurrent(runner, paths, 8, nil)
}

// ReadBatchMetadataMapConcurrent 并发分批读取多个照片文件的元数据，支持多 Worker 并行与进度实时回调
func ReadBatchMetadataMapConcurrent(
	runner CommandRunner, paths []string, concurrency int, onProgress func(processed, total int),
) (map[string]Metadata, error) {
	if len(paths) == 0 {
		return map[string]Metadata{}, nil
	}

	if concurrency <= 0 {
		concurrency = 8
	}

	const batchSize = 250
	type batchChunk struct {
		chunk []string
	}

	var chunks []batchChunk
	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		chunks = append(chunks, batchChunk{chunk: paths[i:end]})
	}

	var mu sync.Mutex
	result := make(map[string]Metadata, len(paths))
	var processedCount atomic.Int64
	total := len(paths)

	var wg sync.WaitGroup
	chunkChan := make(chan batchChunk, len(chunks))
	for _, c := range chunks {
		chunkChan <- c
	}
	close(chunkChan)

	actualWorkers := concurrency
	if actualWorkers > len(chunks) {
		actualWorkers = len(chunks)
	}

	for w := 0; w < actualWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range chunkChan {
				args := []string{
					"-m", "-q", "-json", "-DateTimeOriginal", "-OffsetTimeOriginal", "-GPSPosition", "-GPSDateTime",
				}
				args = append(args, item.chunk...)

				output, _ := runner.CombinedOutput("exiftool", args...)
				if len(output) > 0 {
					records, parseErr := ParseBatchMetadataJSON(output)
					if parseErr == nil && len(records) > 0 {
						mu.Lock()
						for idx, rec := range records {
							filePath := rec.SourceFile
							if filePath == "" && idx < len(item.chunk) {
								filePath = item.chunk[idx]
							}
							if filePath != "" {
								cleanPath := filepath.Clean(filePath)
								result[cleanPath] = rec
								if abs, err := filepath.Abs(filePath); err == nil {
									result[abs] = rec
								}
							}
						}
						mu.Unlock()
					}
				}

				done := int(processedCount.Add(int64(len(item.chunk))))
				if onProgress != nil {
					if done > total {
						done = total
					}
					onProgress(done, total)
				}
			}
		}()
	}

	wg.Wait()
	return result, nil
}

// ParseBatchMetadataJSON 解析 exiftool -json 多文件输出（允许空数组返回空切片，且自动容错剥离 Warning 文本）
func ParseBatchMetadataJSON(output []byte) ([]Metadata, error) {
	cleanOutput := bytes.TrimSpace(output)
	if len(cleanOutput) == 0 {
		return nil, nil
	}

	// 自动容错剥离 ExifTool 输出前面的 Warning 或非 JSON 前缀
	// 需精准找到后面紧随 '{' 或 ']' 的有效 JSON 数组 '['
	jsonStart := -1
	for i := 0; i < len(cleanOutput); i++ {
		if cleanOutput[i] == '[' {
			rest := bytes.TrimSpace(cleanOutput[i+1:])
			if len(rest) > 0 && (rest[0] == '{' || rest[0] == ']') {
				jsonStart = i
				break
			}
		}
	}

	if jsonStart >= 0 {
		if end := bytes.LastIndexByte(cleanOutput, ']'); end > jsonStart {
			cleanOutput = cleanOutput[jsonStart : end+1]
		}
	}

	var records []Metadata
	if err := json.Unmarshal(cleanOutput, &records); err != nil {
		return nil, fmt.Errorf("解析 exiftool 批量输出失败: %w (原始输出: %s)", err, string(bytes.TrimSpace(output)))
	}
	return records, nil
}

// ParseMetadataJSON 解析 exiftool -json 输出
func ParseMetadataJSON(output []byte) (Metadata, error) {
	records, err := ParseBatchMetadataJSON(output)
	if err != nil {
		return Metadata{}, err
	}
	if len(records) == 0 {
		return Metadata{}, errors.New("exiftool 未返回任何元数据")
	}
	return records[0], nil
}

// ClassifyFailure 分析 exiftool 执行失败原因并输出用户友好的提示
func ClassifyFailure(output []byte, err error) string {
	text := strings.ToLower(string(bytes.TrimSpace(output)))
	switch {
	case strings.Contains(text, "no track points found"), strings.Contains(text, "gps track is empty"):
		return "照片时间未落在轨迹范围内，未写入有效 GPS 信息。"
	case strings.Contains(text, "empty track file"):
		return "GPX 轨迹文件为空或无法读取。"
	case strings.Contains(text, "file not found"):
		return "底层工具未找到待处理文件。"
	case err != nil:
		return fmt.Sprintf("调用 exiftool 失败：%v", err)
	default:
		return "未写入有效 GPS 信息。"
	}
}

// ==========================================
// 1. GPX 轨迹匹配与 GPS 经纬度操作 (Cap 1)
// ==========================================

// WriteGeotag 调用 exiftool 匹配 GPX 轨迹写入照片 GPS 经纬度
func WriteGeotag(runner CommandRunner, rawPath string, gpxFiles []string, geosync string) ([]byte, error) {
	args := []string{
		"-overwrite_original",
		"-P",
		"-api", "GeoMaxIntSecs=1800",
		"-api", "GeoMaxExtSecs=1800",
		fmt.Sprintf("-geosync=%s", geosync),
	}

	for _, gpx := range gpxFiles {
		args = append(args, "-geotag", gpx)
	}

	args = append(args, rawPath)
	return runner.CombinedOutput("exiftool", args...)
}

// WriteCoordinates 直接为照片文件写入指定的 GPS 经纬度与海拔（用于智能插值与手动标记）
func WriteCoordinates(runner CommandRunner, filePath string, lat, lon, alt float64) error {
	latRef := "N"
	absLat := lat
	if lat < 0 {
		latRef = "S"
		absLat = -lat
	}

	lonRef := "E"
	absLon := lon
	if lon < 0 {
		lonRef = "W"
		absLon = -lon
	}

	args := []string{
		"-overwrite_original",
		"-P",
		fmt.Sprintf("-GPSLatitude=%.6f", absLat),
		fmt.Sprintf("-GPSLatitudeRef=%s", latRef),
		fmt.Sprintf("-GPSLongitude=%.6f", absLon),
		fmt.Sprintf("-GPSLongitudeRef=%s", lonRef),
	}

	if alt != 0 {
		altRef := "0" // 0 = 海拔高于海平面 (Above sea level)
		absAlt := alt
		if alt < 0 {
			altRef = "1" // 1 = 海拔低于海平面 (Below sea level)
			absAlt = -alt
		}
		args = append(
			args,
			fmt.Sprintf("-GPSAltitude=%.2f", absAlt),
			fmt.Sprintf("-GPSAltitudeRef=%s", altRef),
		)
	}

	args = append(args, filePath)
	_, err := runner.CombinedOutput("exiftool", args...)
	if err != nil {
		return fmt.Errorf("写入 GPS 经纬度坐标失败: %w", err)
	}
	return nil
}

// SyncGPSToJPG 将 RAW 中的 GPS 经纬度信息同步到伴随的 JPG 文件（仅同步 GPS，不侵入其他元数据）
func SyncGPSToJPG(runner CommandRunner, sourceRaw, targetJPG string) error {
	_, err := runner.CombinedOutput(
		"exiftool", "-overwrite_original", "-P", "-TagsFromFile", sourceRaw,
		"-GPS:all",
		targetJPG,
	)
	if err != nil {
		return fmt.Errorf("同步 GPS 经纬度到 JPG 失败：%w", err)
	}
	return nil
}

// SyncGPSToXMP 将 RAW 中的 GPS 经纬度坐标规范化同步到伴随的 XMP 侧车文件
func SyncGPSToXMP(runner CommandRunner, sourceRaw, targetXMP string) error {
	args := []string{
		"-overwrite_original",
		"-P",
		"-TagsFromFile", sourceRaw,
		"-XMP-exif:GPSVersionID<GPSVersionID",
		"-XMP-exif:GPSLatitude<GPSLatitude",
		"-XMP-exif:GPSLongitude<GPSLongitude",
		"-XMP-exif:GPSAltitudeRef<GPSAltitudeRef",
		"-XMP-exif:GPSAltitude<GPSAltitude",
		"-XMP-exif:GPSDateTime<GPSDateTime",
		"-XMP-exif:GPSSatellites<GPSSatellites",
		"-XMP-exif:GPSMapDatum<GPSMapDatum",
		targetXMP,
	}
	_, err := runner.CombinedOutput("exiftool", args...)
	if err != nil {
		return fmt.Errorf("同步 GPS 经纬度到 XMP 失败：%w", err)
	}
	return nil
}

// ==========================================
// 2. 逆地理编码地名元数据写入与同步 (Cap 2)
// ==========================================

// WriteLocation 写入逆地理编码地名标签到照片文件或侧车文件
func WriteLocation(runner CommandRunner, filePath string, loc domain.LocationInfo) error {
	if loc.City == "" && loc.Province == "" && loc.Country == "" && loc.District == "" {
		return nil
	}

	args := []string{
		"-overwrite_original",
		"-P",
		"-charset", "UTF8",
		"-codedcharacterset=utf8",
	}

	if loc.Country != "" {
		args = append(
			args,
			fmt.Sprintf("-XMP-photoshop:Country=%s", loc.Country),
			fmt.Sprintf("-IPTC:Country-PrimaryLocationName=%s", loc.Country),
		)
	}
	if loc.CountryCode != "" {
		args = append(
			args,
			fmt.Sprintf("-XMP-iptcCore:CountryCode=%s", loc.CountryCode),
			fmt.Sprintf("-IPTC:Country-PrimaryLocationCode=%s", loc.CountryCode),
		)
	}
	if loc.Province != "" {
		args = append(
			args,
			fmt.Sprintf("-XMP-photoshop:State=%s", loc.Province),
			fmt.Sprintf("-IPTC:Province-State=%s", loc.Province),
		)
	}
	if loc.City != "" {
		args = append(
			args,
			fmt.Sprintf("-XMP-photoshop:City=%s", loc.City),
			fmt.Sprintf("-IPTC:City=%s", loc.City),
		)
	}
	if loc.District != "" {
		args = append(
			args,
			fmt.Sprintf("-XMP-iptcCore:Location=%s", loc.District),
			fmt.Sprintf("-IPTC:Sub-location=%s", loc.District),
			fmt.Sprintf("-XMP-iptcExt:LocationCreatedSublocation=%s", loc.District),
			fmt.Sprintf("-XMP-iptcExt:LocationShownSublocation=%s", loc.District),
		)
	}

	args = append(args, filePath)
	_, err := runner.CombinedOutput("exiftool", args...)
	if err != nil {
		return fmt.Errorf("写入地理位置信息失败: %w", err)
	}
	return nil
}

// SyncLocationToJPG 将 RAW 中的地名元数据（IPTC/XMP-photoshop）同步到 JPG 文件
func SyncLocationToJPG(runner CommandRunner, sourceRaw, targetJPG string) error {
	_, err := runner.CombinedOutput(
		"exiftool", "-overwrite_original", "-P",
		"-charset", "UTF8",
		"-codedcharacterset=utf8",
		"-TagsFromFile", sourceRaw,
		"-IPTC:all",
		"-XMP-photoshop:all",
		"-XMP-iptcCore:all",
		"-XMP-iptcExt:all",
		targetJPG,
	)
	if err != nil {
		return fmt.Errorf("同步地名元数据到 JPG 失败：%w", err)
	}
	return nil
}

// SyncLocationToXMP 将 RAW 中的地名元数据映射写入到伴随的 XMP 侧车文件
func SyncLocationToXMP(runner CommandRunner, sourceRaw, targetXMP string) error {
	args := []string{
		"-overwrite_original",
		"-P",
		"-charset", "UTF8",
		"-TagsFromFile", sourceRaw,
		"-XMP-photoshop:Country<XMP-photoshop:Country",
		"-XMP-photoshop:State<XMP-photoshop:State",
		"-XMP-photoshop:City<XMP-photoshop:City",
		"-XMP-iptcCore:CountryCode<XMP-iptcCore:CountryCode",
		"-XMP-iptcCore:Location<XMP-iptcCore:Location",
		"-XMP-iptcExt:LocationCreatedSublocation<XMP-iptcExt:LocationCreatedSublocation",
		"-XMP-iptcExt:LocationShownSublocation<XMP-iptcExt:LocationShownSublocation",
		targetXMP,
	}
	_, err := runner.CombinedOutput("exiftool", args...)
	if err != nil {
		return fmt.Errorf("同步地名元数据到 XMP 失败：%w", err)
	}
	return nil
}

// ==========================================
// 3. 坐标解析与格式转换工具
// ==========================================

var (
	// 匹配 "39 deg 54' 15.00\" N, 116 deg 23' 30.00\" E" 或 "39 deg 54' 15.00\" N 116 deg 23' 30.00\" E"
	dmsRegex = regexp.MustCompile(`(\d+)\s*deg\s*(\d+)'\s*([\d.]+)"?\s*([NSEWnsew])`)
	// 匹配 "39.9042 N, 116.3917 E"
	decHemiRegex = regexp.MustCompile(`([\d.]+)\s*([NSEWnsew])`)
)

// ParseCoordinates 解析 GPSPosition 字符串为浮点数经纬度 (lat, lon)
func ParseCoordinates(posStr string) (lat, lon float64, err error) {
	clean := strings.TrimSpace(posStr)
	if clean == "" {
		return 0, 0, errors.New("GPSPosition 字符串为空")
	}

	// 1. 尝试 DMS 匹配
	dmsMatches := dmsRegex.FindAllStringSubmatch(clean, -1)
	if len(dmsMatches) >= 2 {
		lat, err1 := dmsToDecimal(dmsMatches[0][1], dmsMatches[0][2], dmsMatches[0][3], dmsMatches[0][4])
		lon, err2 := dmsToDecimal(dmsMatches[1][1], dmsMatches[1][2], dmsMatches[1][3], dmsMatches[1][4])
		if err1 == nil && err2 == nil {
			return lat, lon, nil
		}
	}

	// 2. 尝试纯十进制+方向匹配 (如 "39.9042 N, 116.3917 E")
	decMatches := decHemiRegex.FindAllStringSubmatch(clean, -1)
	if len(decMatches) >= 2 {
		lat, err1 := decHemiToDecimal(decMatches[0][1], decMatches[0][2])
		lon, err2 := decHemiToDecimal(decMatches[1][1], decMatches[1][2])
		if err1 == nil && err2 == nil {
			return lat, lon, nil
		}
	}

	// 3. 尝试逗号或空格分隔的纯浮点数 (如 "39.9042, 116.3917")
	parts := strings.FieldsFunc(
		clean, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		},
	)
	var cleanParts []string
	for _, p := range parts {
		if p != "" {
			cleanParts = append(cleanParts, p)
		}
	}
	if len(cleanParts) >= 2 {
		latVal, err1 := strconv.ParseFloat(cleanParts[0], 64)
		lonVal, err2 := strconv.ParseFloat(cleanParts[1], 64)
		if err1 == nil && err2 == nil {
			return latVal, lonVal, nil
		}
	}

	return 0, 0, fmt.Errorf("无法识别的 GPSPosition 格式: %s", posStr)
}

func dmsToDecimal(degStr, minStr, secStr, hemi string) (float64, error) {
	deg, err1 := strconv.ParseFloat(degStr, 64)
	min, err2 := strconv.ParseFloat(minStr, 64)
	sec, err3 := strconv.ParseFloat(secStr, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, errors.New("无效的 DMS 数值")
	}

	dec := deg + (min / 60.0) + (sec / 3600.0)
	hemi = strings.ToUpper(hemi)
	if hemi == "S" || hemi == "W" {
		dec = -dec
	}
	return dec, nil
}

func decHemiToDecimal(valStr, hemi string) (float64, error) {
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, err
	}
	hemi = strings.ToUpper(hemi)
	if hemi == "S" || hemi == "W" {
		val = -val
	}
	return val, nil
}

// ==========================================
// 4. 完整照片元数据深度检查 (用于 UI 检查器与调试)
// ==========================================

type DetailedPhotoMetadata struct {
	FilePath         string        `json:"file_path"`
	FileName         string        `json:"file_name"`
	FileSize         string        `json:"file_size"`
	FileModifyDate   string        `json:"file_modify_date"`
	CameraMake       string        `json:"camera_make,omitempty"`
	CameraModel      string        `json:"camera_model,omitempty"`
	LensModel        string        `json:"lens_model,omitempty"`
	DateTimeOriginal string        `json:"date_time_original,omitempty"`
	ExposureTime     string        `json:"exposure_time,omitempty"`
	FNumber          string        `json:"f_number,omitempty"`
	ISO              string        `json:"iso,omitempty"`
	FocalLength      string        `json:"focal_length,omitempty"`
	ExposureProgram  string        `json:"exposure_program,omitempty"`
	Latitude         *float64      `json:"latitude,omitempty"`
	Longitude        *float64      `json:"longitude,omitempty"`
	Altitude         *float64      `json:"altitude,omitempty"`
	GPSPosition      string        `json:"gps_position,omitempty"`
	Country          string        `json:"country,omitempty"`
	Province         string        `json:"province,omitempty"`
	City             string        `json:"city,omitempty"`
	District         string        `json:"district,omitempty"`
	Title            string        `json:"title,omitempty"`
	Description      string        `json:"description,omitempty"`
	RawTags          []ExifTagItem `json:"raw_tags,omitempty"`
}

type ExifTagItem struct {
	Group string `json:"group"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// InspectPhotoMetadata 深度检查单张照片的全部 EXIF / IPTC / XMP 元数据
func InspectPhotoMetadata(runner CommandRunner, path string) (*DetailedPhotoMetadata, error) {
	output, err := runner.CombinedOutput(
		"exiftool", "-m", "-q", "-j", "-G1", "-a", "-s", "-c", "%.6f", path,
	)
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("exiftool 检查元数据失败: %w", err)
	}

	cleanOutput := bytes.TrimSpace(output)
	if len(cleanOutput) == 0 {
		return nil, errors.New("exiftool 未返回任何元数据")
	}

	// 容错剥离前缀
	jsonStart := bytes.IndexByte(cleanOutput, '[')
	if jsonStart >= 0 {
		if end := bytes.LastIndexByte(cleanOutput, ']'); end > jsonStart {
			cleanOutput = cleanOutput[jsonStart : end+1]
		}
	}

	var records []map[string]any
	if err := json.Unmarshal(cleanOutput, &records); err != nil || len(records) == 0 {
		return nil, fmt.Errorf("解析 exiftool 检查输出失败: %w (原始: %s)", err, string(cleanOutput))
	}

	dict := records[0]

	strVal := func(keys ...string) string {
		for _, k := range keys {
			for dk, dv := range dict {
				tagPart := dk
				if idx := strings.Index(dk, ":"); idx >= 0 {
					tagPart = dk[idx+1:]
				}
				if strings.EqualFold(dk, k) || strings.EqualFold(tagPart, k) {
					s := fmt.Sprintf("%v", dv)
					s = strings.TrimSpace(s)
					if s != "" && s != "<nil>" && s != "0" && s != "undef" && s != "-" {
						return s
					}
				}
			}
		}
		return ""
	}

	floatVal := func(keys ...string) *float64 {
		for _, k := range keys {
			for dk, dv := range dict {
				tagPart := dk
				if idx := strings.Index(dk, ":"); idx >= 0 {
					tagPart = dk[idx+1:]
				}
				if strings.EqualFold(dk, k) || strings.EqualFold(tagPart, k) {
					switch v := dv.(type) {
					case float64:
						return &v
					case float32:
						fv := float64(v)
						return &fv
					case int:
						fv := float64(v)
						return &fv
					case int64:
						fv := float64(v)
						return &fv
					case string:
						s := strings.TrimSpace(v)
						s = strings.TrimSuffix(s, " Above Sea Level")
						s = strings.TrimSuffix(s, " Below Sea Level")
						s = strings.TrimSuffix(s, " m")
						s = strings.TrimSpace(s)
						isNeg := strings.HasSuffix(strings.ToUpper(s), "S") || strings.HasSuffix(strings.ToUpper(s), "W")
						s = strings.TrimSuffix(s, " N")
						s = strings.TrimSuffix(s, " S")
						s = strings.TrimSuffix(s, " E")
						s = strings.TrimSuffix(s, " W")
						s = strings.TrimSpace(s)
						if fv, err := strconv.ParseFloat(s, 64); err == nil {
							if isNeg && fv > 0 {
								fv = -fv
							}
							return &fv
						}
					}
				}
			}
		}
		return nil
	}

	makeStr := strVal("EXIF:Make", "Make", "QuickTime:Make")
	modelStr := strVal("EXIF:Model", "Model", "QuickTime:Model")
	lensStr := strVal("EXIF:LensModel", "XMP:LensModel", "MakerNotes:Lens", "MakerNotes:LensType", "LensModel", "Composite:LensSpec", "Composite:LensID", "Lens")
	dateStr := strVal("EXIF:DateTimeOriginal", "XMP:DateTimeOriginal", "QuickTime:CreateDate", "DateTimeOriginal")
	expStr := strVal("EXIF:ExposureTime", "Composite:ShutterSpeed", "ExposureTime", "ShutterSpeed")
	fnStr := strVal("EXIF:FNumber", "Composite:Aperture", "FNumber", "ApertureValue", "Aperture")
	isoStr := strVal("EXIF:ISO", "EXIF:ISOSpeedRatings", "MakerNotes:ISO", "ISO")
	focalStr := strVal("EXIF:FocalLength", "Composite:FocalLength35efl", "EXIF:FocalLengthIn35mmFormat", "FocalLength")
	progStr := strVal("EXIF:ExposureProgram", "ExposureProgram")

	latVal := floatVal("GPS:GPSLatitude", "Composite:GPSLatitude", "GPSLatitude")
	lonVal := floatVal("GPS:GPSLongitude", "Composite:GPSLongitude", "GPSLongitude")
	altVal := floatVal("GPS:GPSAltitude", "Composite:GPSAltitude", "GPSAltitude")
	posStr := strVal("Composite:GPSPosition", "GPS:GPSPosition", "GPSPosition")

	// 如果经纬度解析为 nil 但有 GPSPosition 字符串，尝试反解
	if (latVal == nil || lonVal == nil) && posStr != "" {
		if lat, lon, err := ParseCoordinates(posStr); err == nil {
			latVal = &lat
			lonVal = &lon
		}
	}

	countryStr := strVal("XMP:Country", "IPTC:Country-PrimaryLocationName", "Country")
	provStr := strVal("XMP:State", "IPTC:Province-State", "State", "Province")
	cityStr := strVal("XMP:City", "IPTC:City", "City")
	distStr := strVal("XMP:Location", "IPTC:Sub-location", "Sublocation", "District")
	titleStr := strVal("XMP:Title", "IPTC:ObjectName", "Title")
	descStr := strVal("XMP:Description", "IPTC:Caption-Abstract", "Description")

	fileSizeStr := strVal("File:FileSize", "FileSize")
	modDateStr := strVal("File:FileModifyDate", "FileModifyDate")

	var rawTags []ExifTagItem
	for k, v := range dict {
		if k == "SourceFile" {
			continue
		}
		parts := strings.SplitN(k, ":", 2)
		group := "General"
		tag := k
		if len(parts) == 2 {
			group = parts[0]
			tag = parts[1]
		}
		rawTags = append(rawTags, ExifTagItem{
			Group: group,
			Tag:   tag,
			Value: fmt.Sprintf("%v", v),
		})
	}

	sort.Slice(rawTags, func(i, j int) bool {
		if rawTags[i].Group == rawTags[j].Group {
			return rawTags[i].Tag < rawTags[j].Tag
		}
		return rawTags[i].Group < rawTags[j].Group
	})

	return &DetailedPhotoMetadata{
		FilePath:         path,
		FileName:         filepath.Base(path),
		FileSize:         fileSizeStr,
		FileModifyDate:   modDateStr,
		CameraMake:       makeStr,
		CameraModel:      modelStr,
		LensModel:        lensStr,
		DateTimeOriginal: dateStr,
		ExposureTime:     expStr,
		FNumber:          fnStr,
		ISO:              isoStr,
		FocalLength:      focalStr,
		ExposureProgram:  progStr,
		Latitude:         latVal,
		Longitude:        lonVal,
		Altitude:         altVal,
		GPSPosition:      posStr,
		Country:          countryStr,
		Province:         provStr,
		City:             cityStr,
		District:         distStr,
		Title:            titleStr,
		Description:      descStr,
		RawTags:          rawTags,
	}, nil
}
