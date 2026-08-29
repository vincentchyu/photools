package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureStandardDirectories(t *testing.T) {
	tempDir := t.TempDir()
	customGPX := filepath.Join(tempDir, "custom_gpx")

	statuses, err := EnsureStandardDirectories(tempDir, customGPX)
	if err != nil {
		t.Fatalf("EnsureStandardDirectories 失败: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(tempDir, "Inbox"),
		customGPX,
		filepath.Join(tempDir, "Processed", "geotag"),
		filepath.Join(tempDir, "Processed", "organize"),
	}

	for _, d := range expectedDirs {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			t.Errorf("期望目录已创建且为文件夹: %s, err: %v", d, err)
		}
	}

	if len(statuses) != 5 {
		t.Errorf("期望返回 5 个规范目录状态，实际返回 %d", len(statuses))
	}
}
