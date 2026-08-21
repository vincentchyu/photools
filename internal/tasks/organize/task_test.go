package organize

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
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmdStr := name + " " + strings.Join(args, " ")
	for key, resp := range m.responses {
		if strings.Contains(cmdStr, key) {
			return resp, nil
		}
	}
	return []byte("[]"), nil
}

func TestOrganizeTask_Success(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "Source")
	targetDir := filepath.Join(tempDir, "Output")
	_ = os.MkdirAll(sourceDir, 0o755)

	rawPath := filepath.Join(sourceDir, "DSC_2001.NEF")
	jpgPath := filepath.Join(sourceDir, "DSC_2001.JPG")
	_ = os.WriteFile(rawPath, []byte("raw"), 0o644)
	_ = os.WriteFile(jpgPath, []byte("jpg"), 0o644)

	runner := &mockRunner{
		responses: map[string][]byte{
			"-DateTimeOriginal": []byte(`[{"DateTimeOriginal":"2025:10:06 14:00:00"}]`),
		},
	}

	task, err := NewTask(Config{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Runner:    runner,
	})
	if err != nil {
		t.Fatalf("NewTask 失败: %v", err)
	}

	eventCh := make(chan domain.ProgressEvent, 100)
	summary, issues, err := task.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute 失败: %v", err)
	}

	if summary.Success != 1 || len(issues) != 0 {
		t.Errorf("summary 异常: %+v, issues: %+v", summary, issues)
	}

	expectedRaw := filepath.Join(targetDir, "2025", "1006", "DSC_2025-10-06_2001.nef")
	if _, err := os.Stat(expectedRaw); err != nil {
		t.Errorf("归档文件未生成: %s, err: %v", expectedRaw, err)
	}
}
