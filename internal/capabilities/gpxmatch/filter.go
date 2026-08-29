package gpxmatch

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// 匹配文件名中的日期格式 (例如 20250322, 2025-03-22, 2025_03_22)
	datePattern = regexp.MustCompile(`(?i)(?:^|[-_])(20\d{2})[-_]?(\d{2})[-_]?(\d{2})(?:[.-_]|$)`)
)

// ExtractDateFromFilename 从 GPX 文件名中提取拍摄日期 (YYYY-MM-DD)，若无法识别则返回空字符串
func ExtractDateFromFilename(filename string) string {
	base := filepath.Base(filename)
	matches := datePattern.FindStringSubmatch(base)
	if len(matches) == 4 {
		year, month, day := matches[1], matches[2], matches[3]
		if t, err := time.Parse("2006-01-02", year+"-"+month+"-"+day); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// ParsePhotoDate 解析照片的拍摄时间字符串（支持 "2025:03:22 14:30:00", "2025-03-22T14:30:00" 等格式）
func ParsePhotoDate(dateTimeStr string) (time.Time, bool) {
	clean := strings.TrimSpace(dateTimeStr)
	if len(clean) < 10 {
		return time.Time{}, false
	}
	prefix := clean[:10]
	prefix = strings.ReplaceAll(prefix, ":", "-")
	prefix = strings.ReplaceAll(prefix, "/", "-")
	t, err := time.Parse("2006-01-02", prefix)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// FilterGPXFilesByDate 根据照片拍摄时间从全部 GPX 列表中严格筛选候选文件：
// 1. 严格日邻近（当天及前后各 1 天 ±24h 容差跨午夜）：匹配拍摄当天的专属轨迹；
// 2. 同月降级（日邻近无命中时，在同年同月内降级匹配，适用于多日合并轨迹的场景）；
// 3. 通用无日期标识文件（如 track.gpx）：作为可能包含多日轨迹的候选文件保留；
// 4. 日邻近和同月均无匹配时返回 nil，杜绝跨月将异地轨迹误传给 ExifTool。
func FilterGPXFilesByDate(gpxFiles []string, photoDateTimeStr string) []string {
	if len(gpxFiles) == 0 {
		return nil
	}

	photoDate, ok := ParsePhotoDate(photoDateTimeStr)
	if !ok {
		// 无法识别照片日期时，保留全量候选
		return gpxFiles
	}

	prevDay := photoDate.AddDate(0, 0, -1).Format("2006-01-02")
	currDay := photoDate.Format("2006-01-02")
	nextDay := photoDate.AddDate(0, 0, 1).Format("2006-01-02")
	sameMonth := photoDate.Format("2006-01") // 同年同月前缀，用于降级匹配

	var dayMatches []string
	var monthMatches []string
	var noDateFiles []string

	for _, file := range gpxFiles {
		fileDate := ExtractDateFromFilename(file)
		if fileDate == "" {
			noDateFiles = append(noDateFiles, file)
			continue
		}

		// 严格日邻近匹配 (当天及跨午夜 ±1 天)
		if fileDate == prevDay || fileDate == currDay || fileDate == nextDay {
			dayMatches = append(dayMatches, file)
		} else if strings.HasPrefix(fileDate, sameMonth) {
			// 同年同月降级（适用于合并轨迹文件，相差超 1 天但同月）
			monthMatches = append(monthMatches, file)
		}
	}

	// 1. 优先使用日邻近结果（含无日期通用轨迹）
	if len(dayMatches) > 0 {
		return append(dayMatches, noDateFiles...)
	}

	// 2. 日邻近无命中时，降级使用同年同月轨迹（含无日期通用轨迹）
	if len(monthMatches) > 0 {
		return append(monthMatches, noDateFiles...)
	}

	// 3. 若只有无日期通用轨迹文件，作为候选返回
	if len(noDateFiles) > 0 {
		return noDateFiles
	}

	// 4. 当天无任何匹配 GPX 轨迹
	return nil
}
