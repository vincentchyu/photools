package main

import (
	"flag"
	"reflect"
	"testing"

	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

func TestParseExtensions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"nef,jpg,xmp", []string{"nef", "jpg", "xmp"}},
		{" .nef; .CR3  .dng,wav ", []string{"nef", "cr3", "dng", "wav"}},
		{"", nil},
	}

	for _, tt := range tests {
		got := parseExtensions(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("parseExtensions(%q) = %v; expected %v", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeBoolFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = fs.Bool("flat", false, "flat mode")
	_ = fs.String("source-dir", "", "source dir")

	args := []string{"-flat", "true", "-source-dir", "/tmp/photos", "--flat", "false"}
	normalized := normalizeBoolFlags(fs, args)

	expected := []string{"-flat=true", "-source-dir", "/tmp/photos", "-flat=false"}
	if !reflect.DeepEqual(normalized, expected) {
		t.Errorf("normalizeBoolFlags(%v) = %v; expected %v", args, normalized, expected)
	}
}

func TestParseCoordinatesWithDebug(t *testing.T) {
	lat, lon, alt, debug, err := parseCoordinatesWithDebug([]string{"31.23", "121.47", "10", "--debug"})
	if err != nil {
		t.Fatalf("parseCoordinatesWithDebug failed: %v", err)
	}
	if lat != 31.23 || lon != 121.47 || alt != 10 || !debug {
		t.Errorf("parseCoordinatesWithDebug unexpected result: lat=%f, lon=%f, alt=%f, debug=%v", lat, lon, alt, debug)
	}

	_, _, _, _, err = parseCoordinatesWithDebug([]string{"invalid", "coord"})
	if err == nil {
		t.Errorf("expected error for invalid coordinates")
	}
}

func TestApplySessionOverrides(t *testing.T) {
	cfg := config.NewSessionConfig(nil)
	applySessionOverrides(cfg, "/base", "/base/Inbox", "/base/gpx", "/base/Processed", "/base/logs", true, string(domain.PolicySidecarOnly), true, 4, "nef,cr3", "wav,acr", map[domain.CapabilityID]map[string]any{
		domain.CapGPSInterpolate: {"window": "30m"},
	})

	if cfg.Global.BaseDir != "/base" {
		t.Errorf("BaseDir = %s; expected /base", cfg.Global.BaseDir)
	}
	if cfg.Global.SourceDir != "/base/Inbox" {
		t.Errorf("SourceDir = %s; expected /base/Inbox", cfg.Global.SourceDir)
	}
	if !cfg.Global.FlatMode {
		t.Errorf("FlatMode expected true")
	}
	if cfg.Global.SidecarPolicy != string(domain.PolicySidecarOnly) {
		t.Errorf("SidecarPolicy = %s; expected %s", cfg.Global.SidecarPolicy, domain.PolicySidecarOnly)
	}
	if !cfg.Global.AllowNoGPS {
		t.Errorf("AllowNoGPS expected true")
	}
	if cfg.Global.Workers != 4 {
		t.Errorf("Workers = %d; expected 4", cfg.Global.Workers)
	}
	if cfg.GetStringOption(domain.CapGPSInterpolate, "window", "") != "30m" {
		t.Errorf("Interpolate window option expected 30m")
	}
}

func TestI18n_FlagUsageResolution(t *testing.T) {
	i18n.SetLanguage(i18n.LangZhCN)
	zhOpt := i18n.T("cliOptBaseDir")
	if zhOpt != "工作根目录 (默认当前目录或 ~/Pictures/GPS)" {
		t.Errorf("zh-CN cliOptBaseDir unexpected: %s", zhOpt)
	}

	i18n.SetLanguage(i18n.LangEnUS)
	enOpt := i18n.T("cliOptBaseDir")
	if enOpt != "Base workspace root directory (Default ~/Pictures/GPS)" {
		t.Errorf("en-US cliOptBaseDir unexpected: %s", enOpt)
	}

	// 恢复中文
	i18n.SetLanguage(i18n.LangZhCN)
}

func TestPrintUsage(t *testing.T) {
	// 验证 printUsage 和 printGeoDataHelp 执行不 panic
	printUsage()
	printGeoDataHelp()
	printVersion()
	_ = common.CurrentVersion
}
