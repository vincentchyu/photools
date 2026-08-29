package exiftool

import (
	"os"
	"strings"
	"testing"
)

func TestEnsureConfigFile(t *testing.T) {
	path := EnsureConfigFile()
	if path == "" {
		t.Fatalf("expected non-empty config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "XMP-photools") {
		t.Errorf("expected config to contain 'XMP-photools', got:\n%s", content)
	}
	if !strings.Contains(content, "GPSSource") {
		t.Errorf("expected config to contain 'GPSSource'")
	}
}
