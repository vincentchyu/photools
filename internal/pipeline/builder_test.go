package pipeline

import (
	"path/filepath"
	"testing"
)

func TestBuild_Validation(t *testing.T) {
	// 没有启用任何插件时报错
	_, err := Build(PipelineOptions{})
	if err == nil {
		t.Fatalf("expected error when no capabilities enabled")
	}
}

func TestBuild_Success(t *testing.T) {
	tempDir := t.TempDir()
	opts := PipelineOptions{
		BaseDir:        tempDir,
		SourceDir:      filepath.Join(tempDir, "Inbox"),
		GPXDir:         filepath.Join(tempDir, "GPX"),
		ProcessedDir:   filepath.Join(tempDir, "Processed"),
		LogDir:         filepath.Join(tempDir, "Logs"),
		EnableGPXMatch: true,
		EnableGeocode:  true,
		EnableArchive:  true,
		Workers:        4,
	}

	task, err := Build(opts)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if task == nil {
		t.Fatalf("expected task instance, got nil")
	}

	orch, ok := task.(*Orchestrator)
	if !ok {
		t.Fatalf("expected *Orchestrator instance")
	}

	if len(orch.capabilities) != 3 {
		t.Errorf("expected 3 capabilities, got %d", len(orch.capabilities))
	}
}
