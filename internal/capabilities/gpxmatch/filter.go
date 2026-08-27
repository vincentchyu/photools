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

// FilterGPXFilesByDate 根据照片拍摄时间从全部 GPX 列表中筛选出最相关的候选文件：
// 1. 梯队 1（日邻近 ±1 天）：优先匹配当天及前后各 1 天（±24h 容差）的 GPX 轨迹；
// 2. 梯队 2（同月份扩展）：若 ±1 天未命中，则平滑扩展匹配当月（同一 YYYY-MM，支持多日合并轨迹）的 GPX 轨迹；
// 3. 梯队 3（安全兜底）：无日期标识的文件持续保留，若前序均未命中则安全降级使用全量 GPX 列表。
func FilterGPXFilesByDate(gpxFiles []string, photoDateTimeStr string) []string {
	if len(gpxFiles) <= 1 {
		return gpxFiles
	}

	photoDate, ok := ParsePhotoDate(photoDateTimeStr)
	if !ok {
		return gpxFiles
	}

	prevDay := photoDate.AddDate(0, 0, -1).Format("2006-01-02")
	currDay := photoDate.Format("2006-01-02")
	nextDay := photoDate.AddDate(0, 0, 1).Format("2006-01-02")
	targetMonth := photoDate.Format("2006-01") // YYYY-MM

	var dayMatches []string
	var monthMatches []string
	var noDateFiles []string

	for _, file := range gpxFiles {
		fileDate := ExtractDateFromFilename(file)
		if fileDate == "" {
			noDateFiles = append(noDateFiles, file)
			continue
		}

		// 梯队 1: ±1 天
		if fileDate == prevDay || fileDate == currDay || fileDate == nextDay {
			dayMatches = append(dayMatches, file)
		}

		// 梯队 2: 同月份 (例如 2025-10)
		if strings.HasPrefix(fileDate, targetMonth) {
			monthMatches = append(monthMatches, file)
		}
	}

	// 1. 优先使用日邻近结果 (加上无日期兜底)
	if len(dayMatches) > 0 {
		return append(dayMatches, noDateFiles...)
	}

	// 2. 降级使用同月份结果 (加上无日期兜底)
	if len(monthMatches) > 0 {
		return append(monthMatches, noDateFiles...)
	}

	// 3. 若只存在无日期文件且有过滤效果，返回无日期文件
	if len(noDateFiles) > 0 && len(noDateFiles) < len(gpxFiles) {
		return noDateFiles
	}

	// 4. 全量兜底
	return gpxFiles
}
