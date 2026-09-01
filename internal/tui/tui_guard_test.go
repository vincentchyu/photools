package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

// CJK 统一表意文字与中文常见标点符号正则
var cjkRegex = regexp.MustCompile(`[\p{Han}\x{3000}-\x{303f}\x{ff01}-\x{ff5e}]`)

func TestTUIGuard_EnglishModeZeroCJK(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PHOTOOLS_PLUGINS_CONFIG", filepath.Join(tempDir, "test_plugins.json"))

	// 1. 切换至 en-US 英文模式
	i18n.SetLanguage(i18n.LangEnUS)
	defer i18n.SetLanguage(i18n.LangZhCN) // 测试完成后恢复

	m := InitialModel(tempDir)
	m.width = 120
	m.height = 40

	// 准备纯英文测试上下文数据，确保动态 Mock 数据不包含中文
	mockPlan := &domain.PlanResult{
		TotalAssets:   2,
		ReadyCount:    2,
		PendingCount:  0,
		WarningsCount: 0,
		Items: []domain.PlanItem{
			{
				Asset:   domain.AssetGroup{BaseName: "DSC_0001", RawPath: tempDir + "/DSC_0001.NEF"},
				Action:  "Match timestamps and move to Processed/2026/0515/",
				Warning: "",
			},
			{
				Asset:   domain.AssetGroup{BaseName: "DSC_0002", RawPath: tempDir + "/DSC_0002.NEF"},
				Action:  "Match timestamps and move to Processed/2026/0515/",
				Warning: "",
			},
		},
	}

	mockSummary := &domain.TaskSummary{
		TotalAssets: 2,
		Success:     2,
		Pending:     0,
		Failed:      0,
	}

	mockIssues := []domain.Issue{
		{
			Kind:       domain.IssueKindMissingPair,
			Reason:     "Missing companion JPG",
			Suggestion: "Please supplement JPG file with same basename",
			Asset:      domain.AssetGroup{BaseName: "DSC_0003"},
		},
	}

	m.planResult = mockPlan
	m.taskSummary = mockSummary
	m.taskIssues = mockIssues
	m.archiveDetails = []string{
		"Archived to 2026/0515/ (DSC_2026-05-15_0001)",
		"Archived to 2026/0515/ (DSC_2026-05-15_0002)",
	}
	m.logs = []string{
		"[10:00:00] Initializing pipeline...",
		"[10:00:01] Matched GPS coordinates for DSC_0001",
		"[10:00:02] Successfully archived to Processed/2026/0515/",
	}
	m.statusMessage = ""

	states := []struct {
		state viewState
		name  string
	}{
		{stateInitializing, "stateInitializing"},
		{stateMenu, "stateMenu"},
		{stateGlobalSettings, "stateGlobalSettings"},
		{statePluginSettings, "statePluginSettings"},
		{stateConfig, "stateConfig"},
		{stateDryRun, "stateDryRun"},
		{stateExecuting, "stateExecuting"},
		{stateSummary, "stateSummary"},
	}

	for _, s := range states {
		t.Run(s.name, func(t *testing.T) {
			m.state = s.state

			// 额外针对 stateInitializing 测试两种子场景 (未完成 / 已完成)
			if s.state == stateInitializing {
				m.initFinished = false
				view0 := m.View()
				assertZeroCJK(t, s.name+" (in progress)", view0)

				m.initFinished = true
				view1 := m.View()
				assertZeroCJK(t, s.name+" (finished)", view1)
				return
			}

			view := m.View()
			if strings.TrimSpace(view) == "" {
				t.Fatalf("State %s rendered empty view", s.name)
			}
			assertZeroCJK(t, s.name, view)
		})
	}
}

func assertZeroCJK(t *testing.T, stateName, viewContent string) {
	t.Helper()
	matches := cjkRegex.FindAllString(viewContent, -1)
	if len(matches) > 0 {
		lines := strings.Split(viewContent, "\n")
		var offendingLines []string
		for idx, line := range lines {
			if cjkRegex.MatchString(line) {
				offendingLines = append(offendingLines, fmt.Sprintf("  Line %02d: %s", idx+1, line))
			}
		}
		t.Errorf(
			"[%s] Found %d CJK characters in English mode output:\nMatched runes: %v\nOffending lines:\n%s",
			stateName, len(matches), matches, strings.Join(offendingLines, "\n"),
		)
	}
}
