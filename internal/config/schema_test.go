package config

import (
	"testing"
	"time"

	"github.com/vincentchyu/photools/internal/domain"
)

func TestNewSessionConfig_Defaults(t *testing.T) {
	tempDir := t.TempDir()
	cfg := DefaultPluginsConfig()
	sc := NewSessionConfig(&cfg, tempDir)

	if sc.Global.BaseDir != tempDir {
		t.Errorf("expected BaseDir %s, got %s", tempDir, sc.Global.BaseDir)
	}

	if sc.Global.GPXDir != DefaultGPXDir() {
		t.Errorf("expected GPXDir %s, got %s", DefaultGPXDir(), sc.Global.GPXDir)
	}

	if sc.Global.FlatMode {
		t.Errorf("expected FlatMode default to false")
	}

	if len(sc.Global.RawExtensions) == 0 {
		t.Errorf("expected default RawExtensions not empty")
	}

	// 检查 P15 默认插值窗口
	win := sc.GetDurationOption(domain.CapGPSInterpolate, "window", 0)
	if win != 15*time.Minute {
		t.Errorf("expected P15 window 15m, got %v", win)
	}

	// 检查 P10 默认 geosync
	sync := sc.GetStringOption(domain.CapGPXMatching, "geosync", "")
	if sync != "0" {
		t.Errorf("expected P10 geosync 0, got %s", sync)
	}
}

func TestSessionConfig_SetAndApply(t *testing.T) {
	cfg := DefaultPluginsConfig()
	sc := NewSessionConfig(&cfg, "/tmp/photos")

	// 会话级修改 P15 窗口为 1h
	sc.SetPluginOption(domain.CapGPSInterpolate, "window", "1h")
	win := sc.GetDurationOption(domain.CapGPSInterpolate, "window", 0)
	if win != time.Hour {
		t.Errorf("expected updated window 1h, got %v", win)
	}

	// 会话级修改 P100 为 InPlace 原地模式
	sc.SetPluginOption(domain.CapDateArchive, "in_place", true)
	inPlace := sc.GetBoolOption(domain.CapDateArchive, "in_place", false)
	if !inPlace {
		t.Errorf("expected in_place true")
	}

	// 将会话同步到 PluginsConfig
	sc.ApplyToPluginsConfig(&cfg)
	metaP15 := cfg.FindPluginMeta(domain.CapGPSInterpolate)
	if metaP15.Options["window"] != "1h" {
		t.Errorf("expected synced window 1h in PluginsConfig, got %v", metaP15.Options["window"])
	}

	metaP100 := cfg.FindPluginMeta(domain.CapDateArchive)
	if metaP100.Options["in_place"] != true {
		t.Errorf("expected synced in_place true in PluginsConfig, got %v", metaP100.Options["in_place"])
	}
}

func TestOptionSpecs_Validation(t *testing.T) {
	globalSpecs := GlobalOptionSpecs()
	if len(globalSpecs) < 5 {
		t.Errorf("expected at least 5 global specs, got %d", len(globalSpecs))
	}

	p15Specs := PluginOptionSpecs(domain.CapGPSInterpolate)
	if len(p15Specs) == 0 {
		t.Fatalf("expected P15 specs not empty")
	}

	windowSpec := p15Specs[0]
	if windowSpec.Key != "window" {
		t.Errorf("expected window key, got %s", windowSpec.Key)
	}

	if err := windowSpec.Validate("1h"); err != nil {
		t.Errorf("expected '1h' to be valid: %v", err)
	}

	if err := windowSpec.Validate("invalid_duration"); err == nil {
		t.Errorf("expected invalid duration to fail validation")
	}
}
