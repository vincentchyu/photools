package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// 匹配已经规范化过的格式：包含 YYYY-MM-DD_数字...
	normalizedDateRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2}_\d+`)

	// 匹配常见相机构造：[前缀][数字编号][可选后缀]
	// 例如：
	// DSC_1010 -> prefix="DSC_", number="1010", suffix=""
	// _DSC1010 -> prefix="_DSC", number="1010", suffix=""
	// IMG_1010 -> prefix="IMG_", number="1010", suffix=""
	// _MG_1010 -> prefix="_MG_", number="1010", suffix=""
	// ILCE_1010 -> prefix="ILCE_", number="1010", suffix=""
	// P1010001 -> prefix="P", number="1010001", suffix=""
	// 00123    -> prefix="", number="00123", suffix=""
	// DSC_1011_edit -> prefix="DSC_", number="1011", suffix="_edit"
	cameraBaseRegex = regexp.MustCompile(`^([A-Za-z_-]*?)(\d+)(.*)$`)
)

// Archiver 负责照片的规范化命名、目录构建和安全归档移动
type Archiver struct {
	// 可选自定义命名模板，例如 "{PREFIX}_{YYYY-MM-DD}_{SEQ}{SUFFIX}"
	Template string
}

// NewArchiver 创建 Archiver 实例
func NewArchiver(template ...string) *Archiver {
	t := ""
	if len(template) > 0 {
		t = template[0]
	}
	return &Archiver{
		Template: t,
	}
}

// ParseNameComponents 解析旧文件名的前缀、序号与后缀
func (a *Archiver) ParseNameComponents(oldBaseName string) (prefix, seq, suffix string) {
	matches := cameraBaseRegex.FindStringSubmatch(oldBaseName)
	if len(matches) == 4 {
		return matches[1], matches[2], matches[3]
	}
	return "DSC_", oldBaseName, ""
}

// CalculateNormalizedName 根据拍摄日期和前缀计算规范化后的 BaseName
func (a *Archiver) CalculateNormalizedName(oldBaseName, dateTimeOriginal string) string {
	if dateTimeOriginal == "" {
		return oldBaseName
	}

	// 如果已经包含日期归档标记，则无需重复规范化
	if normalizedDateRegex.MatchString(oldBaseName) {
		return oldBaseName
	}

	t, err := time.Parse("2006:01:02 15:04:05", dateTimeOriginal)
	if err != nil {
		return oldBaseName
	}

	dateStr := t.Format("2006-01-02")
	prefix, number, suffix := a.ParseNameComponents(oldBaseName)

	// 如果前缀为空，默认赋予 "DSC"
	cleanPrefix := prefix
	if cleanPrefix == "" {
		cleanPrefix = "DSC"
	}
	// 去除末尾多余下划线，便于统一拼装
	cleanPrefix = strings.TrimSuffix(cleanPrefix, "_")

	if a.Template != "" {
		res := a.Template
		res = strings.ReplaceAll(res, "{PREFIX}", cleanPrefix)
		res = strings.ReplaceAll(res, "{YYYY-MM-DD}", dateStr)
		res = strings.ReplaceAll(res, "{YYYY}", t.Format("2006"))
		res = strings.ReplaceAll(res, "{MM}", t.Format("01"))
		res = strings.ReplaceAll(res, "{DD}", t.Format("02"))
		res = strings.ReplaceAll(res, "{SEQ}", number)
		res = strings.ReplaceAll(res, "{SUFFIX}", suffix)
		return res
	}

	return fmt.Sprintf("%s_%s_%s%s", cleanPrefix, dateStr, number, suffix)
}

// BuildArchiveDir 构建按拍摄日期组织的归档目录路径 (baseDir/YYYY/MMDD)
func (a *Archiver) BuildArchiveDir(baseDir, dateTimeOriginal string) (string, error) {
	if strings.TrimSpace(dateTimeOriginal) == "" {
		return "", fmt.Errorf("缺少拍摄日期，无法确定归档目录")
	}
	t, err := time.Parse("2006:01:02 15:04:05", dateTimeOriginal)
	if err != nil {
		return "", fmt.Errorf("解析日期失败: %w", err)
	}
	return filepath.Join(baseDir, t.Format("2006"), t.Format("0102")), nil
}

// ExtractFileSuffix 提取文件的规范化后缀（支持 .nef.xmp / .jpg.xmp 等复合侧车后缀）
func (a *Archiver) ExtractFileSuffix(sourcePath string) string {
	lower := strings.ToLower(sourcePath)
	if strings.HasSuffix(lower, ".xmp") {
		trimmed := sourcePath[:len(sourcePath)-4]
		subExt := strings.ToLower(filepath.Ext(trimmed))
		if subExt != "" && len(subExt) <= 6 { // 识别形如 .nef, .jpg, .cr3, .arw
			return subExt + ".xmp"
		}
		return ".xmp"
	}
	return strings.ToLower(filepath.Ext(sourcePath))
}

// CheckConflict 检查目标目录是否存在重名冲突
func (a *Archiver) CheckConflict(files []string, targetDir, newBaseName string) (bool, string) {
	for _, source := range files {
		suffix := a.ExtractFileSuffix(source)
		target := filepath.Join(targetDir, newBaseName+suffix)

		absSrc, errSrc := filepath.Abs(source)
		absTgt, errTgt := filepath.Abs(target)
		if errSrc == nil && errTgt == nil {
			// 1. 如果路径在忽略大小写后完全一致（同目录同文件名，或仅有大小写扩展名差异如 .NEF -> .nef），不判定为外部冲突
			if strings.EqualFold(absSrc, absTgt) {
				continue
			}
		}

		// 2. 如果目标文件存在，进一步通过 os.SameFile 确认是否为源文件本身
		if targetFi, err := os.Stat(target); err == nil {
			if sourceFi, errSrcStat := os.Stat(source); errSrcStat == nil {
				if os.SameFile(sourceFi, targetFi) {
					continue
				}
			}
			return true, target
		}
	}
	return false, ""
}

// MoveFilesWithRename 将一组文件原子重命名并移动到目标目录（支持跨目录移动或同目录原地重命名）
func (a *Archiver) MoveFilesWithRename(files []string, targetDir, newBaseName string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("创建归档目录失败: %w", err)
	}

	if conflict, target := a.CheckConflict(files, targetDir, newBaseName); conflict {
		return fmt.Errorf("目标文件已存在：%s", target)
	}

	for _, source := range files {
		suffix := a.ExtractFileSuffix(source)
		target := filepath.Join(targetDir, newBaseName+suffix)

		absSrc, errSrc := filepath.Abs(source)
		absTgt, errTgt := filepath.Abs(target)
		if errSrc == nil && errTgt == nil {
			if absSrc == absTgt {
				// 源文件与目标文件完全一致，无需任何移动或重命名
				continue
			}
			// 在 macOS 大小写不敏感文件系统上，如果同路径仅大小写差异（如 DSC_001.NEF -> DSC_001.nef），
			// 通过中间临时文件名安全跨越系统对同一大小写别名的覆盖限制
			if strings.EqualFold(absSrc, absTgt) {
				tmpTarget := target + fmt.Sprintf(".tmp_%d", time.Now().UnixNano())
				if err := os.Rename(source, tmpTarget); err != nil {
					return err
				}
				if err := os.Rename(tmpTarget, target); err != nil {
					return err
				}
				continue
			}
		}

		if err := os.Rename(source, target); err != nil {
			return err
		}
	}
	return nil
}
