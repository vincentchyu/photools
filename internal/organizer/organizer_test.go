package organizer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateNormalizedName(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		date     string
		expected string
	}{
		{
			name:     "Standard DSC",
			old:      "DSC_1010",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_1010",
		},
		{
			name:     "With Suffix",
			old:      "DSC_2964_edit_ps",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_2964_edit_ps",
		},
		{
			name:     "Already Normalized",
			old:      "DSC_2026-01-01_1010",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_1010",
		},
		{
			name:     "Already Normalized with Suffix",
			old:      "DSC_2026-01-01_2964_edit_ps",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_2964_edit_ps",
		},
		{
			name:     "No Date",
			old:      "DSC_1010",
			date:     "",
			expected: "DSC_1010",
		},
		{
			name:     "Malformed Date",
			old:      "DSC_1010",
			date:     "2026-01-01",
			expected: "DSC_1010",
		},
		{
			name:     "Short Number",
			old:      "DSC_10",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_10",
		},
		{
			name:     "No DSC prefix",
			old:      "_1234",
			date:     "2026:01:01 12:00:00",
			expected: "DSC_2026-01-01_1234",
		},
		{
			name:     "Random name with digits",
			old:      "Trip_Photo_2025_001",
			date:     "2025:12:31 23:59:59",
			expected: "DSC_2025-12-31_2025_001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateNormalizedName(tt.old, tt.date)
			if got != tt.expected {
				t.Errorf("CalculateNormalizedName(%q, %q) = %q, want %q", tt.old, tt.date, got, tt.expected)
			}
		})
	}
}

func TestMoveFilesWithRename(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	_ = os.MkdirAll(sourceDir, 0o755)
	_ = os.MkdirAll(targetDir, 0o755)

	srcFile := filepath.Join(sourceDir, "test.JPG")
	_ = os.WriteFile(srcFile, []byte("test"), 0o644)

	err := MoveFilesWithRename([]string{srcFile}, targetDir, "new_name")
	if err != nil {
		t.Fatalf("MoveFilesWithRename failed: %v", err)
	}

	expectedTarget := filepath.Join(targetDir, "new_name.jpg")
	if _, err := os.Stat(expectedTarget); os.IsNotExist(err) {
		t.Fatalf("target file does not exist")
	}
}
