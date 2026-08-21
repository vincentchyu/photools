package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

func TestReporter_GenerateReport(t *testing.T) {
	tempDir := t.TempDir()
	reportPath := filepath.Join(tempDir, "pending_report.md")

	issues := []domain.Issue{
		{
			Kind:       domain.IssueKindMissingPair,
			Reason:     "缺少同名 JPG 配对文件",
			Suggestion: "请补齐对应的 JPG 文件后重新运行",
			Asset: domain.AssetGroup{
				BaseName:       "DSC_1001",
				RawPath:        "/Inbox/DSC_1001.NEF",
				CompanionPaths: []string{"/Inbox/DSC_1001.xmp"},
			},
			PhotoTime:          "2025:10:06 14:00:00",
			PhotoOffset:        "+08:00",
			ReferencedGPXFiles: []string{"/GPX/track1.gpx"},
		},
	}

	reporter := NewReporter()
	err := reporter.WriteMarkdownReport(reportPath, "GPS 修正任务", issues)
	if err != nil {
		t.Fatalf("WriteMarkdownReport 失败: %v", err)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("读取报告文件失败: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "待处理资产清单") {
		t.Errorf("报告缺少标题: %s", content)
	}
	if !strings.Contains(content, "DSC_1001") {
		t.Errorf("报告缺少资产名: %s", content)
	}
	if !strings.Contains(content, "缺少同名 JPG 配对文件") {
		t.Errorf("报告缺少原因: %s", content)
	}
	if !strings.Contains(content, "track1.gpx") {
		t.Errorf("报告缺少 GPX 文件信息: %s", content)
	}
}
