package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

func (m Model) View() string {
	var content string

	switch m.state {
	case stateInitializing:
		content = m.viewInitializing()
	case stateMenu:
		content = m.viewMenu()
	case stateGlobalSettings:
		content = m.viewGlobalSettings()
	case statePluginSettings:
		content = m.viewPluginSettings()
	case stateConfig:
		content = m.viewConfig()
	case stateDryRun:
		content = m.viewDryRun()
	case stateExecuting:
		content = m.viewExecuting()
	case stateSummary:
		content = m.viewSummary()
	}

	return m.renderFrame(content)
}

func (m Model) renderFrame(body string) string {
	appTitle := fmt.Sprintf(" %s ", i18n.T("tuiTitle", common.CurrentVersion))
	header := TitleStyle.Render(appTitle)

	var footer string
	switch m.state {
	case stateInitializing:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpInitializing"))
	case stateMenu:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpMenu"))
	case stateGlobalSettings:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpGlobalSettings"))
	case statePluginSettings:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpPluginSettings"))
	case stateConfig:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpConfig"))
	case stateDryRun:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpDryRun"))
	case stateExecuting:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpExecuting"))
	case stateSummary:
		footer = HelpKeyStyle.Render(i18n.T("tuiHelpSummary"))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		"",
		HelpBar.Render(footer),
	)
}

func (m Model) panelWidth() int {
	if m.width > 20 {
		return min(96, max(50, m.width-6))
	}
	return 76
}

func (m Model) viewInitializing() string {
	var b strings.Builder

	b.WriteString(SubTitleStyle.Render(i18n.T("tuiSelfCheckTitle")) + "\n\n")

	cardWidth := m.panelWidth()

	readyCount := 0
	for _, item := range m.pluginItems {
		report, has := m.initReports[item.id]
		var statusBadge string
		var detailLine string
		var percent float64 = -1

		if !has {
			statusBadge = lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(0, 1).Render(i18n.T("tuiInitWaitingBadge"))
			detailLine = i18n.T("tuiInitWaitingDetail")
		} else {
			percent = report.Percent
			switch report.Status {
			case domain.HealthReady:
				if report.Percent >= 1.0 {
					statusBadge = BadgeSuccess.Render(i18n.T("tuiInitReadyBadge"))
					readyCount++
				} else {
					statusBadge = BadgeInfo.Render(fmt.Sprintf(i18n.T("tuiInitLoadingBadge"), m.spinner.View()))
				}
			case domain.HealthDegraded:
				statusBadge = BadgeWarning.Render(i18n.T("tuiInitDegradedBadge"))
				readyCount++
			case domain.HealthFailed:
				statusBadge = BadgeDanger.Render(i18n.T("tuiInitFailedBadge"))
			default:
				statusBadge = BadgeInfo.Render(fmt.Sprintf(i18n.T("tuiInitLoadingBadge"), m.spinner.View()))
			}

			stagePrefix := ""
			if report.Stage != "" {
				stagePrefix = fmt.Sprintf("[%s] ", report.Stage)
			}
			detailLine = stagePrefix + report.Message
		}

		cardText := fmt.Sprintf(
			"%s %s\n    └─ %s", statusBadge, lipgloss.NewStyle().Bold(true).Render(item.Title()), detailLine,
		)
		if percent >= 0 && percent < 1.0 {
			bar := m.progress.ViewAs(percent)
			cardText += fmt.Sprintf("\n    %s %.0f%%", bar, percent*100)
		} else if percent >= 1.0 {
			bar := m.progress.ViewAs(1.0)
			cardText += fmt.Sprintf("\n    %s 100%%", bar)
		}

		cardStyle := lipgloss.NewStyle().
			Foreground(TextColor).
			Padding(0, 1).
			Width(cardWidth)
		b.WriteString(cardStyle.Render(cardText) + "\n\n")
	}

	if m.initFinished {
		statusSummary := fmt.Sprintf(i18n.T("tuiInitSummaryDone"), readyCount, len(m.pluginItems))
		b.WriteString(lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(statusSummary) + "\n")
	} else {
		statusSummary := fmt.Sprintf(i18n.T("tuiInitSummaryRunning"), readyCount, len(m.pluginItems))
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(statusSummary) + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiInitSummaryTip")))
	}

	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}

func (m Model) viewMenu() string {
	var b strings.Builder

	// 1. 顶部工作区状态
	flatBadge := ""
	if m.flatMode {
		flatBadge = " " + BadgeWarning.Render(i18n.T("tuiMenuFlatBadge"))
	}
	b.WriteString(StatusLabel.Render(i18n.T("tuiMenuWorkspaceLabel")) + StatusPath.Render(m.currentBaseDir) + flatBadge + "\n")

	statusBadges := fmt.Sprintf(
		i18n.T("tuiMenuStatusBadges"),
		m.sourceDirInput.Value(),
		BadgeInfo.Render(fmt.Sprintf(i18n.T("tuiMenuSetsCount"), m.inboxAssetCount)),
		BadgeSuccess.Render(fmt.Sprintf(i18n.T("tuiMenuFilesCount"), m.gpxCount)),
		BadgeSuccess.Render(fmt.Sprintf(i18n.T("tuiMenuSetsCount"), m.processedCount)),
	)
	b.WriteString(statusBadges + "\n")

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(m.statusMessage) + "\n\n")
	} else {
		b.WriteString("\n")
	}

	// 计算当前已选插件的 Phase 编号映射
	var activePriorities []int
	priMap := make(map[int]int)
	getPriority := func(id domain.CapabilityID, def int) int {
		if m.sessionConfig != nil {
			if opt, ok := m.sessionConfig.Plugins[id]; ok && opt.Priority > 0 {
				return opt.Priority
			}
		}
		return def
	}

	p1 := getPriority(domain.CapGPXMatching, 10)
	p15 := getPriority(domain.CapGPSInterpolate, 15)
	p2 := getPriority(domain.CapReverseGeocode, 20)
	p3 := getPriority(domain.CapDateArchive, 100)

	priList := []struct {
		p       int
		enabled bool
	}{
		{p1, m.enableGPXMatch},
		{p15, m.enableInterpolate},
		{p2, m.enableGeocode},
		{p3, m.enableArchive},
	}

	for _, item := range priList {
		if item.enabled {
			found := false
			for _, ap := range activePriorities {
				if ap == item.p {
					found = true
					break
				}
			}
			if !found {
				activePriorities = append(activePriorities, item.p)
			}
		}
	}
	sort.Ints(activePriorities)
	for idx, ap := range activePriorities {
		priMap[ap] = idx + 1
	}

	// 2. 插件能力勾选清单
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiMenuPipelineSection")) + "\n\n")

	cardWidth := 74
	if m.width > 20 {
		cardWidth = min(80, max(50, m.width-6))
	}

	for i, item := range m.pluginItems {
		var checked bool
		var priority int
		var paramBadge string

		switch i {
		case 0:
			checked = m.enableGPXMatch
			priority = p1
			sync := m.sessionConfig.GetStringOption(domain.CapGPXMatching, "geosync", "0")
			if sync != "" && sync != "0" {
				paramBadge = " " + BadgeParam.Render(fmt.Sprintf(i18n.T("tuiMenuGeosyncBadge"), sync))
			}
		case 1:
			checked = m.enableInterpolate
			priority = p15
			win := m.sessionConfig.GetStringOption(domain.CapGPSInterpolate, "window", "15m")
			paramBadge = " " + BadgeParam.Render(fmt.Sprintf(i18n.T("tuiMenuWindowBadge"), win))
		case 2:
			checked = m.enableGeocode
			priority = p2
		case 3:
			checked = m.enableArchive
			priority = p3
			inPlace := m.sessionConfig.GetBoolOption(domain.CapDateArchive, "in_place", false) || m.flatMode
			if inPlace {
				paramBadge = " " + BadgeParam.Render(i18n.T("tuiMenuInPlaceBadge"))
			}
		}

		checkMark := "[ ]"
		if checked {
			checkMark = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("[✔]")
		}

		// 优先级徽章
		var priorityBadge string
		if checked {
			phaseIdx := priMap[priority]
			priorityBadge = BadgeInfo.Render(fmt.Sprintf(i18n.T("tuiMenuPhaseBadge"), priority, phaseIdx))
		} else {
			priorityBadge = lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(0, 1).Render(fmt.Sprintf(i18n.T("tuiMenuInactiveBadge"), priority))
		}

		// 自检信息与健康状态行
		var initStatusLine string
		if rep, has := m.initReports[item.id]; has && rep.Message != "" {
			var hBadge string
			switch rep.Status {
			case domain.HealthReady:
				hBadge = BadgeSuccess.Render(i18n.T("tuiMenuHealthOK"))
			case domain.HealthDegraded:
				hBadge = BadgeWarning.Render(i18n.T("tuiMenuHealthDegraded"))
			case domain.HealthFailed:
				hBadge = BadgeDanger.Render(i18n.T("tuiMenuHealthFailed"))
			default:
				hBadge = BadgeInfo.Render(i18n.T("tuiMenuHealthReady"))
			}
			initStatusLine = fmt.Sprintf("%s %s %s", i18n.T("tuiMenuCheckPrefix"), hBadge, rep.Message)
		}

		descPrefix := i18n.T("tuiMenuDescPrefix")

		isFocused := (i == m.pluginIndex)
		var cardText string
		if initStatusLine != "" {
			if isFocused {
				cardText = fmt.Sprintf(
					"▶ %s %s %s%s\n    %s\n    └─ %s %s", checkMark, priorityBadge, item.Title(), paramBadge, initStatusLine,
					descPrefix, item.Desc(),
				)
			} else {
				cardText = fmt.Sprintf(
					"  %s %s %s%s\n    %s\n    └─ %s %s", checkMark, priorityBadge, item.Title(), paramBadge, initStatusLine,
					descPrefix, item.Desc(),
				)
			}
		} else {
			if isFocused {
				cardText = fmt.Sprintf("▶ %s %s %s%s\n    └─ %s %s", checkMark, priorityBadge, item.Title(), paramBadge, descPrefix, item.Desc())
			} else {
				cardText = fmt.Sprintf("  %s %s %s%s\n    └─ %s %s", checkMark, priorityBadge, item.Title(), paramBadge, descPrefix, item.Desc())
			}
		}

		if isFocused {
			cardStyle := lipgloss.NewStyle().
				Background(ActiveBgColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1).
				Width(cardWidth).
				Bold(true)
			b.WriteString(cardStyle.Render(cardText) + "\n\n")
		} else {
			cardStyle := lipgloss.NewStyle().
				Foreground(TextColor).
				Padding(0, 1).
				Width(cardWidth)
			b.WriteString(cardStyle.Render(cardText) + "\n\n")
		}
	}

	b.WriteString(
		lipgloss.NewStyle().Foreground(MutedTextColor).Render(
			i18n.T("tuiMenuConfigFooter", config.GetConfigPath()),
		) + "\n",
	)

	return b.String()
}
