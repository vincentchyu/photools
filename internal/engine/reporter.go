package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

// Reporter 负责生成用户可读的结构化中文报告
type Reporter struct{}

// NewReporter 创建 Reporter 实例
func NewReporter() *Reporter {
	return &Reporter{}
}

// WriteMarkdownReport 将待处理问题生成为 Markdown 报告并写入目标路径
func (r *Reporter) WriteMarkdownReport(filePath, taskName string, issues []domain.Issue) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("创建报告所在目录失败: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s - 待处理资产清单\n\n", taskName))
	buf.WriteString(fmt.Sprintf("- **生成时间**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("- **待处理资产数**: %d\n\n", len(issues)))

	if len(issues) == 0 {
		buf.WriteString("🎉 本次运行没有发现任何待处理或异常资产，全部处理完毕！\n")
		return os.WriteFile(filePath, buf.Bytes(), 0o644)
	}

	buf.WriteString("| 序号 | 资产名称 | 拍摄时间 (时区) | 待补/失败原因 | 建议解决动作 |\n")
	buf.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for i, issue := range issues {
		timeStr := issue.PhotoTime
		if issue.PhotoOffset != "" {
			timeStr += fmt.Sprintf(" (%s)", issue.PhotoOffset)
		}
		if timeStr == "" {
			timeStr = "未知"
		}

		name := issue.Asset.DisplayName()
		buf.WriteString(fmt.Sprintf("| %d | `%s` | %s | %s | %s |\n", i+1, name, timeStr, issue.Reason, issue.Suggestion))
	}

	buf.WriteString("\n## 详细清单\n\n")
	for i, issue := range issues {
		buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, issue.Asset.DisplayName()))
		buf.WriteString(fmt.Sprintf("- **BaseName**: `%s`\n", issue.Asset.BaseName))
		if issue.Asset.RawPath != "" {
			buf.WriteString(fmt.Sprintf("- **RAW 文件**: `%s`\n", filepath.Base(issue.Asset.RawPath)))
		}
		if issue.Asset.JPGPath != "" {
			buf.WriteString(fmt.Sprintf("- **JPG 文件**: `%s`\n", filepath.Base(issue.Asset.JPGPath)))
		}
		if len(issue.Asset.CompanionPaths) > 0 {
			var compNames []string
			for _, cp := range issue.Asset.CompanionPaths {
				compNames = append(compNames, filepath.Base(cp))
			}
			buf.WriteString(fmt.Sprintf("- **附属文件**: %s\n", strings.Join(compNames, ", ")))
		}
		buf.WriteString(fmt.Sprintf("- **原因说明**: %s\n", issue.Reason))
		buf.WriteString(fmt.Sprintf("- **建议动作**: %s\n", issue.Suggestion))

		if len(issue.ReferencedGPXFiles) > 0 {
			var gpxNames []string
			for _, g := range issue.ReferencedGPXFiles {
				gpxNames = append(gpxNames, filepath.Base(g))
			}
			buf.WriteString(fmt.Sprintf("- **参与匹配的 GPX**: %s\n", strings.Join(gpxNames, ", ")))
		}
		buf.WriteString("\n")
	}

	return os.WriteFile(filePath, buf.Bytes(), 0o644)
}
