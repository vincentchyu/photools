package organizer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// 匹配数字部分
	nameRegex = regexp.MustCompile(`(?:DSC_)?(\d+)(.*)`)
	// 匹配已经规范化过的格式 DSC_YYYY-MM-DD_NUMBER...
	normalizedRegex = regexp.MustCompile(`^DSC_\d{4}-\d{2}-\d{2}_\d+.*`)
)

func CalculateNormalizedName(oldBaseName, dateTimeOriginal string) string {
	if dateTimeOriginal == "" {
		return oldBaseName
	}

	if normalizedRegex.MatchString(oldBaseName) {
		return oldBaseName
	}

	t, err := time.Parse("2006:01:02 15:04:05", dateTimeOriginal)
	if err != nil {
		return oldBaseName
	}

	dateStr := t.Format("2006-01-02")
	matches := nameRegex.FindStringSubmatch(oldBaseName)
	if len(matches) < 3 {
		fallbackRegex := regexp.MustCompile(`(\d+)(.*)`)
		matches = fallbackRegex.FindStringSubmatch(oldBaseName)
		if len(matches) < 3 {
			return oldBaseName
		}
	}

	number := matches[1]
	suffix := matches[2]

	return fmt.Sprintf("DSC_%s_%s%s", dateStr, number, suffix)
}

func MoveFilesWithRename(files []string, targetDir, newBaseName string) error {
	for _, source := range files {
		ext := strings.ToLower(filepath.Ext(source))
		target := filepath.Join(targetDir, newBaseName+ext)
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("目标文件已存在：%s", target)
		}
		if err := os.Rename(source, target); err != nil {
			return err
		}
	}
	return nil
}

func BuildArchiveDir(baseDir, dateTimeOriginal string) (string, error) {
	if strings.TrimSpace(dateTimeOriginal) == "" {
		return "", fmt.Errorf("缺少拍摄日期，无法确定归档目录")
	}
	t, err := time.Parse("2006:01:02 15:04:05", dateTimeOriginal)
	if err != nil {
		return "", fmt.Errorf("解析日期失败: %w", err)
	}
	return filepath.Join(baseDir, t.Format("2006"), t.Format("0102")), nil
}
