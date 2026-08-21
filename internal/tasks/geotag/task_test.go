package geotag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

type mockRunner struct {
	responses map[string][]byte
	calls     [][]string
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmdStr := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, append([]string{name}, args...))

	for key, resp := range m.responses {
		if strings.Contains(cmdStr, key) {
			return resp, nil
		}
	}
	return []byte("[]"), nil
}

func TestGeotagTask_MissingJPG_KeptInInbox(t *testing.T) {
	tempDir := t.TempDir()
	inbox := filepath.Join(tempDir, "Inbox")
	gpxDir := filepath.Join(tempDir, "GPX")
	_ = os.MkdirAll(inbox, 0o755)
	_ = os.MkdirAll(gpxDir, 0o755)

	// 创建单 RAW 文件（无 JPG）
	rawPath := filepath.Join(inbox, "DSC_1001.NEF")
	_ = os.WriteFile(rawPath, []byte("raw"), 0o644)
	gpxPath := filepath.Join(gpxDir, "track.gpx")
	_ = os.WriteFile(gpxPath, []byte("<gpx></gpx>"), 0o644)

	task, err := NewTask(Config{
		BaseDir: tempDir,
		Runner:  &mockRunner{},
	})
	if err != nil {
		t.Fatalf("NewTask 失败: %v", err)
	}

	eventCh := make(chan domain.ProgressEvent, 100)
	summary, issues, err := task.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute 期望无致命错误，但返回: %v", err)
	}

	if summary.Pending != 1 || summary.Success != 0 {
		t.Errorf("summary 异常: %+v", summary)
	}

	if len(issues) != 1 || issues[0].Kind != domain.IssueKindMissingPair {
		t.Errorf("期望记录缺少配对异常，实际: %+v", issues)
	}

	// 确认文件仍留在 Inbox
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Errorf("单 RAW 文件被误移除了 Inbox")
	}
}

func TestGeotagTask_SuccessFlow(t *testing.T) {
	tempDir := t.TempDir()
	inbox := filepath.Join(tempDir, "Inbox")
	gpxDir := filepath.Join(tempDir, "GPX")
	processed := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(inbox, 0o755)
	_ = os.MkdirAll(gpxDir, 0o755)

	rawPath := filepath.Join(inbox, "DSC_1002.NEF")
	jpgPath := filepath.Join(inbox, "DSC_1002.JPG")
	xmpPath := filepath.Join(inbox, "DSC_1002.xmp")
	_ = os.WriteFile(rawPath, []byte("raw"), 0o644)
	_ = os.WriteFile(jpgPath, []byte("jpg"), 0o644)
	_ = os.WriteFile(xmpPath, []byte("xmp"), 0o644)

	gpxPath := filepath.Join(gpxDir, "track.gpx")
	_ = os.WriteFile(gpxPath, []byte("<gpx></gpx>"), 0o644)

	runner := &mockRunner{
		responses: map[string][]byte{
			"-DateTimeOriginal": []byte(`[{"DateTimeOriginal":"2025:10:06 14:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"23.123 113.123","GPSDateTime":"2025:10:06 06:00:00Z"}]`),
		},
	}

	task, err := NewTask(Config{
		BaseDir: tempDir,
		Runner:  runner,
	})
	if err != nil {
		t.Fatalf("NewTask 失败: %v", err)
	}

	eventCh := make(chan domain.ProgressEvent, 100)
	summary, issues, err := task.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute 失败: %v", err)
	}

	if summary.Success != 1 || summary.Failed != 0 || len(issues) != 0 {
		t.Errorf("期望成功 1 组，实际: %+v, issues: %+v", summary, issues)
	}

	// 确认已归档到 Processed/geotag/2025/1006/
	targetRAW := filepath.Join(processed, "geotag", "2025", "1006", "DSC_2025-10-06_1002.nef")
	if _, err := os.Stat(targetRAW); err != nil {
		t.Errorf("归档文件未生成: %s, err: %v", targetRAW, err)
	}
}
