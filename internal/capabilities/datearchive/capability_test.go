package datearchive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
)

type mockRunner struct {
	outputs map[string][]byte
	errs    map[string]error
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmdStr := strings.Join(args, " ")
	for k, v := range m.outputs {
		if strings.Contains(cmdStr, k) {
			return v, m.errs[k]
		}
	}
	return []byte("ok"), nil
}

func TestDateArchiveCapability_PlanPrecheck(t *testing.T) {
	capInst := NewCapability(Config{
		ProcessedDir: "/tmp/processed",
	})

	// 1. 无拍摄日期
	actxNoDate := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_001"})
	plan1 := capInst.PlanPrecheck(context.Background(), actxNoDate)
	if plan1.CanProcess {
		t.Errorf("expected CanProcess=false when no date, got true")
	}

	// 2. 具备拍摄日期
	actxWithDate := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_002"})
	actxWithDate.UpdateMetadata(domain.Metadata{
		DateTimeOriginal: "2025:10:06 14:30:00",
	})
	plan2 := capInst.PlanPrecheck(context.Background(), actxWithDate)
	if !plan2.CanProcess {
		t.Errorf("expected CanProcess=true when date exists, got false: %v", plan2.Warning)
	}
}

func TestDateArchiveCapability_ExecuteProcess_Success(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "Inbox")
	targetDir := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(sourceDir, 0o755)

	rawPath := filepath.Join(sourceDir, "DSC_1010.NEF")
	jpgPath := filepath.Join(sourceDir, "DSC_1010.JPG")
	xmpPath := filepath.Join(sourceDir, "DSC_1010.xmp")
	_ = os.WriteFile(rawPath, []byte("raw"), 0o644)
	_ = os.WriteFile(jpgPath, []byte("jpg"), 0o644)
	_ = os.WriteFile(xmpPath, []byte("xmp"), 0o644)

	capInst := NewCapability(Config{
		Archiver:     engine.NewArchiver(),
		ProcessedDir: targetDir,
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName:       "DSC_1010",
		Dir:            sourceDir,
		RawPath:        rawPath,
		JPGPath:        jpgPath,
		XMPPath:        xmpPath,
		CompanionPaths: []string{xmpPath},
	})
	actx.UpdateMetadata(domain.Metadata{
		DateTimeOriginal: "2025:10:06 14:30:00",
	})

	var events []domain.ProgressEvent
	err := capInst.ExecuteProcess(context.Background(), actx, func(e domain.ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFolder := filepath.Join(targetDir, "2025", "1006")
	if actx.TargetDir != expectedFolder {
		t.Errorf("expected TargetDir=%s, got %s", expectedFolder, actx.TargetDir)
	}

	// 确认物理文件已被移动
	expectedNewRaw := filepath.Join(expectedFolder, "DSC_2025-10-06_1010.NEF")
	if _, err := os.Stat(expectedNewRaw); os.IsNotExist(err) {
		t.Errorf("expected archived file %s to exist, but not found", expectedNewRaw)
	}
}

func TestDateArchiveCapability_SupportedOptions_And_Configure(t *testing.T) {
	capInst := NewCapability(Config{})
	opts := capInst.SupportedOptions()
	if len(opts) == 0 {
		t.Fatalf("SupportedOptions should not be empty")
	}

	opt := opts[0]
	if opt.Key != "in_place" {
		t.Errorf("expected option key 'in_place', got %s", opt.Key)
	}

	// 测试 Configure bool
	_ = capInst.Configure(map[string]any{
		"in_place": true,
	})
	if !capInst.inPlace {
		t.Errorf("expected inPlace=true, got false")
	}

	// 测试 Configure string
	_ = capInst.Configure(map[string]any{
		"in_place": "false",
	})
	if capInst.inPlace {
		t.Errorf("expected inPlace=false, got true")
	}
}
