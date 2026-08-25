package exiftool

import (
	"os"
	"path/filepath"
	"testing"
)

func testIsExecutable(path string) bool {
	return isExecutable(path)
}

func TestLocateExifToolEnv(t *testing.T) {
	ResetExifToolPath()
	defer ResetExifToolPath()

	tmpDir := t.TempDir()
	fakeTool := filepath.Join(tmpDir, "exiftool")
	if err := os.WriteFile(fakeTool, []byte("#!/bin/sh\necho 12.0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PHOTOOLS_EXIFTOOL", fakeTool)
	got := LocateExifTool()
	if got != fakeTool {
		t.Errorf("期望定位到环境变量指定的 %s, 实际得到 %s", fakeTool, got)
	}
}

func TestIsExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	nonExec := filepath.Join(tmpDir, "file.txt")
	_ = os.WriteFile(nonExec, []byte("test"), 0644)

	if isExecutable(nonExec) {
		t.Errorf("644 权限文件不应判定为可执行")
	}

	execFile := filepath.Join(tmpDir, "run.sh")
	_ = os.WriteFile(execFile, []byte("echo 1"), 0755)

	if !isExecutable(execFile) {
		t.Errorf("755 权限文件应判定为可执行")
	}
}
