package completion

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCompletions_AllCommandsAndFlagsIncluded(t *testing.T) {
	requiredCommands := []string{
		"tui",
		"geotag",
		"geocode",
		"pipeline",
		"organize-by-date",
		"geodata",
		"completion",
		"version",
		"help",
	}

	requiredFlags := []string{
		"base-dir",
		"source-dir",
		"flat",
		"in-place",
		"interpolate",
		"interpolate-window",
		"allow-no-gps",
		"geosync",
		"raw-exts",
		"workers",
		"geocode",
		"test",
	}

	// 1. 测试 Zsh
	var zshBuf bytes.Buffer
	GenerateZsh(&zshBuf)
	zshOut := zshBuf.String()
	for _, cmd := range requiredCommands {
		if !strings.Contains(zshOut, cmd) {
			t.Errorf("GenerateZsh 缺少核心子命令: %s", cmd)
		}
	}
	for _, flg := range requiredFlags {
		if !strings.Contains(zshOut, flg) {
			t.Errorf("GenerateZsh 缺少关键参数: %s", flg)
		}
	}

	// 2. 测试 Bash
	var bashBuf bytes.Buffer
	GenerateBash(&bashBuf)
	bashOut := bashBuf.String()
	for _, cmd := range requiredCommands {
		if !strings.Contains(bashOut, cmd) {
			t.Errorf("GenerateBash 缺少核心子命令: %s", cmd)
		}
	}
	for _, flg := range requiredFlags {
		if !strings.Contains(bashOut, flg) {
			t.Errorf("GenerateBash 缺少关键参数: %s", flg)
		}
	}

	// 3. 测试 Fish
	var fishBuf bytes.Buffer
	GenerateFish(&fishBuf)
	fishOut := fishBuf.String()
	for _, cmd := range requiredCommands {
		if !strings.Contains(fishOut, cmd) {
			t.Errorf("GenerateFish 缺少核心子命令: %s", cmd)
		}
	}
	for _, flg := range requiredFlags {
		if !strings.Contains(fishOut, flg) {
			t.Errorf("GenerateFish 缺少关键参数: %s", flg)
		}
	}
}

func TestInstallShellCompletion_Zsh_And_Bash_And_Idempotent(t *testing.T) {
	// 1. 测试 Zsh 环境安装
	tempHomeZsh := t.TempDir()
	t.Setenv("HOME", tempHomeZsh)
	t.Setenv("SHELL", "/bin/zsh")

	msgZsh, err := InstallShellCompletion()
	if err != nil {
		t.Fatalf("InstallShellCompletion (zsh) 失败: %v", err)
	}
	if !strings.Contains(msgZsh, "已成功") || !strings.Contains(msgZsh, "zsh") {
		t.Errorf("zsh 安装提示信息不符合预期: %s", msgZsh)
	}

	zshrcPath := filepath.Join(tempHomeZsh, ".zshrc")
	contentZsh, err := os.ReadFile(zshrcPath)
	if err != nil {
		t.Fatalf("无法读取 .zshrc: %v", err)
	}
	if !strings.Contains(string(contentZsh), "photools.zsh") {
		t.Errorf(".zshrc 未包含 photools.zsh source 行: %s", string(contentZsh))
	}

	// 2. 测试幂等性 (再次运行不重复追加)
	_, _ = InstallShellCompletion()
	contentZshAgain, _ := os.ReadFile(zshrcPath)
	if strings.Count(string(contentZshAgain), "# photools Shell Auto-Completion") != 1 {
		t.Errorf("重复运行 InstallShellCompletion 导致重复追加配置")
	}

	// 3. 测试 Bash 环境安装
	tempHomeBash := t.TempDir()
	t.Setenv("HOME", tempHomeBash)
	t.Setenv("SHELL", "/bin/bash")

	msgBash, err := InstallShellCompletion()
	if err != nil {
		t.Fatalf("InstallShellCompletion (bash) 失败: %v", err)
	}
	if !strings.Contains(msgBash, "已成功") || !strings.Contains(msgBash, "bash") {
		t.Errorf("bash 安装提示信息不符合预期: %s", msgBash)
	}

	bashrcPath := filepath.Join(tempHomeBash, ".bashrc")
	contentBash, err := os.ReadFile(bashrcPath)
	if err != nil {
		t.Fatalf("无法读取 .bashrc: %v", err)
	}
	if !strings.Contains(string(contentBash), "photools.bash") {
		t.Errorf(".bashrc 未包含 photools.bash source 行: %s", string(contentBash))
	}
}
