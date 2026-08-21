package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompleteDirectoryPath(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "Photos"), 0o755)
	_ = os.MkdirAll(filepath.Join(tempDir, "Projects"), 0o755)
	_ = os.MkdirAll(filepath.Join(tempDir, "Documents"), 0o755)

	// 1. 测试唯一匹配
	completed, candidates := CompleteDirectoryPath(filepath.Join(tempDir, "Doc"))
	expected := filepath.Join(tempDir, "Documents") + string(filepath.Separator)
	if completed != expected {
		t.Errorf("唯一匹配失败: 期望 %q, 实际 %q", expected, completed)
	}
	if len(candidates) != 1 {
		t.Errorf("期望 1 个候选，实际 %d", len(candidates))
	}

	// 2. 测试多项前缀匹配 (P -> Photos, Projects)
	completed, candidates = CompleteDirectoryPath(filepath.Join(tempDir, "P"))
	if len(candidates) != 2 {
		t.Errorf("期望 2 个候选，实际 %d", len(candidates))
	}
	if !strings.HasPrefix(completed, filepath.Join(tempDir, "P")) {
		t.Errorf("多项匹配公共前缀异常: %s", completed)
	}

	// 3. 测试进入目录后列出所有子项
	completed, candidates = CompleteDirectoryPath(tempDir + string(filepath.Separator))
	if len(candidates) != 3 {
		t.Errorf("期望列出 3 个子目录，实际 %d", len(candidates))
	}
}
