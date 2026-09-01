package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

func (m Model) viewDryRun() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiDryRunTitle")) + "\n\n")

	if m.planResult == nil {
		statusText := m.statusMessage
		if statusText == "" {
			statusText = i18n.T("tuiDryRunScanning")
		}
		b.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), lipgloss.NewStyle().Foreground(HighlightColor).Bold(true).Render(statusText)))
		if m.totalNum > 0 {
			pct := float64(m.processedNum) / float64(m.totalNum)
			if pct > 1.0 {
				pct = 1.0
			}
			b.WriteString(fmt.Sprintf("\n  %s  %s\n", m.progress.ViewAs(pct), lipgloss.NewStyle().Foreground(HighlightColor).Render(fmt.Sprintf("%d/%d (%s) (%.0f%%)", m.processedNum, m.totalNum, i18n.T("tuiMenuSetsCount", m.totalNum), pct*100))))
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiDryRunLoadingTip")) + "\n")
		return PanelStyle.Width(m.panelWidth()).Render(b.String())
	}

	statsLine := fmt.Sprintf(
		i18n.T("tuiDryRunStatsLine"),
		BadgeInfo.Render(fmt.Sprintf(" %d ", m.planResult.TotalAssets)),
		BadgeSuccess.Render(fmt.Sprintf(" %d ", m.planResult.ReadyCount)),
		BadgeWarning.Render(fmt.Sprintf(" %d ", m.planResult.PendingCount)),
		BadgeDanger.Render(fmt.Sprintf(" %d ", m.planResult.WarningsCount)),
	)
	b.WriteString(statsLine + "\n\n")

	if len(m.planResult.Items) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiDryRunNoPhotos")) + "\n\n")
	} else {
		maxShow := 8
		start, end := calculateWindow(len(m.planResult.Items), m.planIndex, maxShow)

		for i := start; i < end; i++ {
			item := m.planResult.Items[i]
			prefix := "  "
			if i == m.planIndex {
				prefix = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
			}

			statusBadge := BadgeSuccess.Render(i18n.T("tuiDryRunReadyBadge"))
			if item.Warning != "" {
				statusBadge = BadgeWarning.Render(" " + item.Warning + " ")
			}

			line := fmt.Sprintf(
				"%s%s  %s  ->  %s", prefix, statusBadge,
				lipgloss.NewStyle().Bold(true).Render(item.Asset.DisplayName()), item.Action,
			)
			b.WriteString(line + "\n")
		}

		if len(m.planResult.Items) > maxShow {
			scrollPrompt := fmt.Sprintf(i18n.T("tuiDryRunScrollPrompt"), m.planIndex+1, len(m.planResult.Items))
			b.WriteString(
				fmt.Sprintf(
					"\n%s\n", lipgloss.NewStyle().Foreground(MutedTextColor).Render(scrollPrompt),
				),
			)
		} else {
			b.WriteString("\n")
		}
	}

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(m.statusMessage) + "\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiDryRunShortcuts")))
	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}

func stageDisplayName(st domain.PipelineStage) string {
	switch st {
	case domain.StageInit:
		return i18n.T("stageInit")
	case domain.StageDiscover:
		return i18n.T("stageDiscover")
	case domain.StagePrecheck:
		return i18n.T("stagePrecheck")
	case domain.StageGeotag:
		return i18n.T("stageGeotag")
	case domain.StageInterpolate:
		return i18n.T("stageInterpolate")
	case domain.StageGeocode:
		return i18n.T("stageGeocode")
	case domain.StageSync:
		return i18n.T("stageSync")
	case domain.StageArchive:
		return i18n.T("stageArchive")
	case domain.StageBackup:
		return i18n.T("stageBackup")
	case domain.StageRestore:
		return i18n.T("stageRestore")
	case domain.StageSummary:
		return i18n.T("stageSummary")
	case domain.StageComplete:
		return i18n.T("stageComplete")
	default:
		return string(st)
	}
}

func (m Model) viewExecuting() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiExecutingTitle")) + "\n\n")

	// 动态从当前 Task 获取 Stages 阶段列表
	var stages []domain.PipelineStage
	if m.currentTask != nil {
		stages = m.currentTask.Stages()
	} else {
		stages = []domain.PipelineStage{domain.StageDiscover, domain.StagePrecheck, domain.StageComplete}
	}

	var stageBadges []string
	for _, st := range stages {
		disp := stageDisplayName(st)
		if st == m.currentStage {
			stageBadges = append(
				stageBadges,
				lipgloss.NewStyle().Background(PrimaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Padding(
					0, 1,
				).Render(disp),
			)
		} else {
			stageBadges = append(
				stageBadges,
				lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(
					0, 1,
				).Render(disp),
			)
		}
	}
	b.WriteString(strings.Join(stageBadges, " ➔ ") + "\n\n")

	percent := 0.0
	if m.totalNum > 0 {
		percent = float64(m.processedNum) / float64(m.totalNum)
	}
	progressText := fmt.Sprintf(
		"%s %s: %d/%d (%.1f%%)", m.spinner.View(), i18n.T("tuiExecProgressLabel"), m.processedNum, m.totalNum, percent*100,
	)
	b.WriteString(progressText + "\n")
	b.WriteString(m.progress.ViewAs(percent) + "\n\n")

	if m.currentAsset != "" {
		b.WriteString(
			fmt.Sprintf(
				"%s%s\n\n", i18n.T("tuiExecProcessingLabel"), lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(m.currentAsset),
			),
		)
	}

	b.WriteString(i18n.T("tuiExecLiveLogTitle"))
	b.WriteString(m.viewport.View() + "\n\n")

	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiSummaryTitle")) + "\n\n")

	if m.taskErr != nil {
		b.WriteString(
			lipgloss.NewStyle().Foreground(DangerColor).Bold(true).Render(
				fmt.Sprintf(
					"%s: %v", i18n.T("tuiSummaryErrorLabel"), m.taskErr,
				),
			) + "\n\n",
		)
	}

	if m.taskSummary != nil {
		b.WriteString(
			fmt.Sprintf(
				i18n.T("tuiSummaryStatsLine"),
				BadgeInfo.Render(fmt.Sprintf(" %d ", m.taskSummary.TotalAssets)),
				BadgeSuccess.Render(fmt.Sprintf(" %d ", m.taskSummary.Success)),
				BadgeWarning.Render(fmt.Sprintf(" %d ", m.taskSummary.Pending)),
				BadgeDanger.Render(fmt.Sprintf(" %d ", m.taskSummary.Failed)),
			),
		)
	}

	if len(m.archiveDetails) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Render(i18n.T("tuiSummaryArchivedTitle")) + "\n")
		maxShow := 5
		start, end := calculateWindow(len(m.archiveDetails), m.summaryIndex, maxShow)
		for i := start; i < end; i++ {
			b.WriteString(fmt.Sprintf("  • %s\n", m.archiveDetails[i]))
		}
		b.WriteString("\n")
	}

	if len(m.taskIssues) > 0 {
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Foreground(WarningColor).Render(
				fmt.Sprintf(
					i18n.T("tuiSummaryIssuesTitle"),
					len(m.taskIssues),
				),
			) + "\n",
		)
		maxShow := 5
		start, end := calculateWindow(len(m.taskIssues), m.issuesScroll, maxShow)
		for i := start; i < end; i++ {
			issue := m.taskIssues[i]
			b.WriteString(
				fmt.Sprintf(
					i18n.T("tuiSummaryIssueItem"),
					i+1, issue.Asset.DisplayName(), issue.Reason, issue.Suggestion,
				),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(HighlightColor).Render(i18n.T("tuiSummaryLogLocation")) + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiSummaryShortcuts")))
	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}
