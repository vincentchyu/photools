package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchiver_CalculateNormalizedName(t *testing.T) {
	archiver := NewArchiver()

	tests := []struct {
		name     string
		oldBase  string
		dateTime string
		template string
		expected string
	}{
		{
			name:     "标准相机编号 DSC_1010",
			oldBase:  "DSC_1010",
			dateTime: "2025:10:06 14:30:00",
			expected: "DSC_2025-10-06_1010",
		},
		{
			name:     "AdobeRGB 前缀 _DSC1010",
			oldBase:  "_DSC1010",
			dateTime: "2025:10:06 14:30:00",
			expected: "_DSC_2025-10-06_1010",
		},
		{
			name:     "佳能前缀 IMG_2020",
			oldBase:  "IMG_2020",
			dateTime: "2025:10:06 14:30:00",
			expected: "IMG_2025-10-06_2020",
		},
		{
			name:     "佳能 AdobeRGB 前缀 _MG_2020",
			oldBase:  "_MG_2020",
			dateTime: "2025:10:06 14:30:00",
			expected: "_MG_2025-10-06_2020",
		},
		{
			name:     "索尼机身前缀 ILCE_9988",
			oldBase:  "ILCE_9988",
			dateTime: "2025:10:06 14:30:00",
			expected: "ILCE_2025-10-06_9988",
		},
		{
			name:     "松下前缀 P1010001",
			oldBase:  "P1010001",
			dateTime: "2025:10:06 14:30:00",
			expected: "P_2025-10-06_1010001",
		},
		{
			name:     "带后缀编号 DSC_1011_edit",
			oldBase:  "DSC_1011_edit",
			dateTime: "2025:10:06 14:31:00",
			expected: "DSC_2025-10-06_1011_edit",
		},
		{
			name:     "非 DSC 前缀的纯数字 00123",
			oldBase:  "00123",
			dateTime: "2026:01:01 09:00:00",
			expected: "DSC_2026-01-01_00123",
		},
		{
			name:     "已经是规范格式 DSC_2025-10-06_1010",
			oldBase:  "DSC_2025-10-06_1010",
			dateTime: "2025:10:06 14:30:00",
			expected: "DSC_2025-10-06_1010",
		},
		{
			name:     "已经是规范格式（带其他前缀）IMG_2025-10-06_1010",
			oldBase:  "IMG_2025-10-06_1010",
			dateTime: "2025:10:06 14:30:00",
			expected: "IMG_2025-10-06_1010",
		},
		{
			name:     "无效日期时保留原名",
			oldBase:  "DSC_1010",
			dateTime: "invalid-date",
			expected: "DSC_1010",
		},
		{
			name:     "自定义命名模板 {YYYY}{MM}{DD}_{PREFIX}_{SEQ}{SUFFIX}",
			oldBase:  "IMG_8888_raw",
			dateTime: "2025:10:06 14:30:00",
			template: "{YYYY}{MM}{DD}_{PREFIX}_{SEQ}{SUFFIX}",
			expected: "20251006_IMG_8888_raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := archiver
			if tt.template != "" {
				a = NewArchiver(tt.template)
			}
			actual := a.CalculateNormalizedName(tt.oldBase, tt.dateTime)
			if actual != tt.expected {
				t.Errorf("CalculateNormalizedName(%q, %q) = %q; 期望 %q", tt.oldBase, tt.dateTime, actual, tt.expected)
			}
		})
	}
}

func TestArchiver_BuildArchiveDir(t *testing.T) {
	archiver := NewArchiver()
	base := "/tmp/Processed"

	dir, err := archiver.BuildArchiveDir(base, "2025:10:06 14:30:00")
	if err != nil {
		t.Fatalf("BuildArchiveDir 失败: %v", err)
	}

	expected := filepath.Join(base, "2025", "1006")
	if dir != expected {
		t.Errorf("BuildArchiveDir = %q; 期望 %q", dir, expected)
	}
}

func TestArchiver_MoveAssetWithRename(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "Inbox")
	targetDir := filepath.Join(tempDir, "Processed", "2025", "1006")

	_ = os.MkdirAll(sourceDir, 0o755)
	_ = os.MkdirAll(targetDir, 0o755)

	raw := filepath.Join(sourceDir, "DSC_1010.NEF")
	jpg := filepath.Join(sourceDir, "DSC_1010.JPG")
	xmp := filepath.Join(sourceDir, "DSC_1010.xmp")

	_ = os.WriteFile(raw, []byte("raw"), 0o644)
	_ = os.WriteFile(jpg, []byte("jpg"), 0o644)
	_ = os.WriteFile(xmp, []byte("xmp"), 0o644)

	archiver := NewArchiver()
	err := archiver.MoveFilesWithRename([]string{raw, jpg, xmp}, targetDir, "DSC_2025-10-06_1010")
	if err != nil {
		t.Fatalf("MoveFilesWithRename 失败: %v", err)
	}

	// 验证源文件已移动，目标文件存在
	if _, err := os.Stat(raw); !os.IsNotExist(err) {
		t.Errorf("源 RAW 文件未被移动")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2025-10-06_1010.nef")); err != nil {
		t.Errorf("目标 RAW 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2025-10-06_1010.jpg")); err != nil {
		t.Errorf("目标 JPG 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2025-10-06_1010.xmp")); err != nil {
		t.Errorf("目标 XMP 文件不存在: %v", err)
	}
}

func TestArchiver_MoveAssetWithCompoundXMP(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "Inbox")
	targetDir := filepath.Join(tempDir, "Processed", "2026", "0101")

	_ = os.MkdirAll(sourceDir, 0o755)
	_ = os.MkdirAll(targetDir, 0o755)

	raw := filepath.Join(sourceDir, "DSC_2948.nef")
	jpg := filepath.Join(sourceDir, "DSC_2948.JPG")
	rawXmp := filepath.Join(sourceDir, "DSC_2948.nef.xmp")
	jpgXmp := filepath.Join(sourceDir, "DSC_2948.JPG.xmp")

	_ = os.WriteFile(raw, []byte("raw"), 0o644)
	_ = os.WriteFile(jpg, []byte("jpg"), 0o644)
	_ = os.WriteFile(rawXmp, []byte("rawXmp"), 0o644)
	_ = os.WriteFile(jpgXmp, []byte("jpgXmp"), 0o644)

	archiver := NewArchiver()
	err := archiver.MoveFilesWithRename([]string{raw, jpg, rawXmp, jpgXmp}, targetDir, "DSC_2026-01-01_2948")
	if err != nil {
		t.Fatalf("MoveFilesWithRename 失败: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2026-01-01_2948.nef")); err != nil {
		t.Errorf("目标 RAW 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2026-01-01_2948.jpg")); err != nil {
		t.Errorf("目标 JPG 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2026-01-01_2948.nef.xmp")); err != nil {
		t.Errorf("目标 RAW XMP 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "DSC_2026-01-01_2948.jpg.xmp")); err != nil {
		t.Errorf("目标 JPG XMP 文件不存在: %v", err)
	}
}

// TestArchiver_InPlaceRename_CaseInsensitiveNoSelfConflict 验证在同目录下大写扩展名（如 .NEF / .JPG）原地重命名为小写规范名时不会误报冲突
func TestArchiver_InPlaceRename_CaseInsensitiveNoSelfConflict(t *testing.T) {
	tempDir := t.TempDir()

	raw := filepath.Join(tempDir, "DSC_2025-10-09_438.NEF")
	jpg := filepath.Join(tempDir, "DSC_2025-10-09_438.JPG")
	xmp := filepath.Join(tempDir, "DSC_2025-10-09_438.NEF.xmp")

	_ = os.WriteFile(raw, []byte("raw_content"), 0o644)
	_ = os.WriteFile(jpg, []byte("jpg_content"), 0o644)
	_ = os.WriteFile(xmp, []byte("xmp_content"), 0o644)

	archiver := NewArchiver()

	// 1. 检查冲突应该返回 false
	conflict, conflictFile := archiver.CheckConflict([]string{raw, jpg, xmp}, tempDir, "DSC_2025-10-09_438")
	if conflict {
		t.Fatalf("原地重命名不应被误判为冲突，冲突文件: %s", conflictFile)
	}

	// 2. 执行原地重命名应成功
	err := archiver.MoveFilesWithRename([]string{raw, jpg, xmp}, tempDir, "DSC_2025-10-09_438")
	if err != nil {
		t.Fatalf("原地重命名执行失败: %v", err)
	}

	// 3. 验证规范化小写文件存在
	if _, err := os.Stat(filepath.Join(tempDir, "DSC_2025-10-09_438.nef")); err != nil {
		t.Errorf("原地规范化后 RAW 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "DSC_2025-10-09_438.jpg")); err != nil {
		t.Errorf("原地规范化后 JPG 文件不存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "DSC_2025-10-09_438.nef.xmp")); err != nil {
		t.Errorf("原地规范化后 XMP 文件不存在: %v", err)
	}
}
