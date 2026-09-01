package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

func (m Model) viewGlobalSettings() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiGlobalSettingsTitle")) + "\n\n")

	// 1. Language
	prefix0 := "  "
	if m.globalFocusIdx == 0 {
		prefix0 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	langBadge := BadgeSuccess.Render(i18n.T("currentLangBadge"))
	b.WriteString(i18n.T("tuiGlobalLangLabel", prefix0, StatusLabel.Render("[1/7]"), langBadge))

	// 2. GPXDir
	prefix1 := "  "
	if m.globalFocusIdx == 1 {
		prefix1 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiGlobalGPXDirLabel", prefix1, StatusLabel.Render("[2/7]")))
	b.WriteString(m.gpxDirInput.View() + "\n")
	if m.globalFocusIdx == 1 && len(m.tabCandidates) > 0 {
		var badges []string
		for i, c := range m.tabCandidates {
			if i < 4 {
				if i == m.tabCandidateIdx {
					badges = append(badges, BadgeSuccess.Render(" "+filepath.Base(c)+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+filepath.Base(c)+" "))
				}
			}
		}
		if len(m.tabCandidates) > 4 {
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiCandMore", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(i18n.T("tuiCandLabel")) + strings.Join(badges, " ") + "\n")
	}

	// 3. LogDir
	prefix2 := "  "
	if m.globalFocusIdx == 2 {
		prefix2 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiGlobalLogDirLabel", prefix2, StatusLabel.Render("[3/7]")))
	b.WriteString(m.logDirInput.View() + "\n")
	if m.globalFocusIdx == 2 && len(m.tabCandidates) > 0 {
		var badges []string
		for i, c := range m.tabCandidates {
			if i < 4 {
				if i == m.tabCandidateIdx {
					badges = append(badges, BadgeSuccess.Render(" "+filepath.Base(c)+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+filepath.Base(c)+" "))
				}
			}
		}
		if len(m.tabCandidates) > 4 {
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiCandMore", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(i18n.T("tuiCandLabel")) + strings.Join(badges, " ") + "\n")
	}

	// 4. Sidecar Policy
	prefix3 := "  "
	if m.globalFocusIdx == 3 {
		prefix3 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	var policyBadge string
	switch m.sidecarPolicy {
	case domain.PolicySmart, domain.PolicyReadOnly:
		policyBadge = BadgeSuccess.Render(i18n.T("policyBadgeSmart"))
	case domain.PolicySidecarOnly:
		policyBadge = BadgeInfo.Render(i18n.T("policyBadgeSidecarOnly"))
	case domain.PolicyEmbedAndSidecar:
		policyBadge = BadgeWarning.Render(i18n.T("policyBadgeEmbedAndSidecar"))
	case domain.PolicyEmbedOnly:
		policyBadge = lipgloss.NewStyle().Background(PrimaryColor).Foreground(lipgloss.Color("#FFFFFF")).Render(i18n.T("policyBadgeEmbedOnly"))
	default:
		policyBadge = BadgeSuccess.Render(i18n.T("policyBadgeSmart"))
	}
	b.WriteString(i18n.T("tuiGlobalPolicyLabel", prefix3, StatusLabel.Render("[4/7]"), policyBadge))

	// 5. Companion Extensions
	prefix4 := "  "
	if m.globalFocusIdx == 4 {
		prefix4 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiGlobalCompExtsLabel", prefix4, StatusLabel.Render("[5/7]")))
	b.WriteString(m.companionExtsInput.View() + "\n")

	// 6. Raw Extensions
	prefix5 := "  "
	if m.globalFocusIdx == 5 {
		prefix5 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiGlobalRawExtsLabel", prefix5, StatusLabel.Render("[6/7]")))
	b.WriteString(m.rawExtsInput.View() + "\n")

	// 7. Workers
	prefix6 := "  "
	if m.globalFocusIdx == 6 {
		prefix6 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiGlobalWorkersLabel", prefix6, StatusLabel.Render("[7/7]")))
	b.WriteString(m.workersInput.View() + "\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiGlobalShortcuts")))
	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}

func (m Model) viewPluginSettings() string {
	var b strings.Builder
	item := m.pluginItems[m.pluginIndex]

	b.WriteString(SubTitleStyle.Render(i18n.T("tuiPluginOptionsTitle", item.Title())) + "\n\n")
	b.WriteString(i18n.T("tuiPluginDescPrefix", item.Desc()))

	specs := item.cap.SupportedOptions()
	opts := m.sessionConfig.GetPluginOptions(item.id)

	if len(specs) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiPluginNoOptions")))
	} else {
		opt := specs[0]
		val, _ := opts[opt.Key].(string)
		if val == "" {
			val = fmt.Sprintf("%v", opt.DefaultValue)
		}

		if len(opt.Choices) > 0 {
			var badges []string
			for _, choice := range opt.Choices {
				if choice == val {
					badges = append(badges, BadgeSuccess.Render(" "+choice+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+choice+" "))
				}
			}
			b.WriteString(i18n.T("tuiPluginCycleLabel", opt.DisplayName()))
			b.WriteString(strings.Join(badges, " ") + "\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiPluginHelpPrefix", opt.DisplayDescription())))
		} else {
			b.WriteString(i18n.T("tuiPluginParamLabel", opt.DisplayName()))
			b.WriteString(m.pluginSettingInput.View() + "\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiPluginHelpPrefix", opt.DisplayDescription())))
		}
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiPluginShortcuts")))
	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render(i18n.T("tuiPreFlightTitle")) + "\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(i18n.T("tuiSessionCapTitle")) + "\n")

	var capSummaries []string
	if m.enableGPXMatch {
		sync := m.geosyncInput.Value()
		if sync == "" {
			sync = "0"
		}
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (%s)",
			BadgeSuccess.Render(i18n.T("tuiSessionPhasePrefix", 1)),
			lipgloss.NewStyle().Bold(true).Render(m.pluginItems[0].Title()),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(i18n.T("tuiSessionOffsetPrefix", sync))))
	}
	if m.enableInterpolate {
		win := m.sessionConfig.GetStringOption(domain.CapGPSInterpolate, "window", "15m")
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (%s)",
			BadgeSuccess.Render(i18n.T("tuiSessionPhasePrefix", 2)),
			lipgloss.NewStyle().Bold(true).Render(m.pluginItems[1].Title()),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(i18n.T("tuiSessionWindowPrefix", win))))
	}
	if m.enableGeocode {
		lang := m.sessionConfig.GetStringOption(domain.CapReverseGeocode, "language", "zh-CN")
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (%s)",
			BadgeSuccess.Render(i18n.T("tuiSessionPhasePrefix", 3)),
			lipgloss.NewStyle().Bold(true).Render(m.pluginItems[2].Title()),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(i18n.T("tuiSessionLangPrefix", lang))))
	}
	if m.enableArchive {
		inPlace := m.sessionConfig.GetBoolOption(domain.CapDateArchive, "in_place", false) || m.flatMode
		modeStr := i18n.T("tuiSessionArchiveHierarchy")
		if inPlace {
			modeStr = i18n.T("tuiSessionArchiveInPlace")
		}
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (%s)",
			BadgeSuccess.Render(i18n.T("tuiSessionPhasePrefix", 4)),
			lipgloss.NewStyle().Bold(true).Render(m.pluginItems[3].Title()),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(i18n.T("tuiSessionPolicyPrefix", modeStr))))
	}
	if len(capSummaries) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(DangerColor).Render(i18n.T("tuiSessionNoCaps")) + "\n")
	} else {
		b.WriteString(strings.Join(capSummaries, "\n") + "\n")
	}
	b.WriteString("\n")

	// 2. 会话可编辑参数与相册目录调整
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(i18n.T("tuiSessionSettingsTitle")) + "\n")

	// 2.1 Workspace BaseDir
	prefix0 := "  "
	if m.configFocusIdx == 0 {
		prefix0 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiSessionBaseDirLabel", prefix0, StatusLabel.Render("[1/7]")))
	b.WriteString(m.baseDirInput.View() + "\n")
	if m.configFocusIdx == 0 && len(m.tabCandidates) > 0 {
		var badges []string
		for i, c := range m.tabCandidates {
			if i < 4 {
				if i == m.tabCandidateIdx {
					badges = append(badges, BadgeSuccess.Render(" "+filepath.Base(c)+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+filepath.Base(c)+" "))
				}
			}
		}
		if len(m.tabCandidates) > 4 {
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiCandMore", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(i18n.T("tuiCandLabel")) + strings.Join(badges, " ") + "\n")
	}

	// 2.2 SourceDir
	prefix1 := "  "
	if m.configFocusIdx == 1 {
		prefix1 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiSessionSourceDirLabel", prefix1, StatusLabel.Render("[2/7]")))
	b.WriteString(m.sourceDirInput.View() + "\n")
	if m.configFocusIdx == 1 && len(m.tabCandidates) > 0 {
		var badges []string
		for i, c := range m.tabCandidates {
			if i < 4 {
				if i == m.tabCandidateIdx {
					badges = append(badges, BadgeSuccess.Render(" "+filepath.Base(c)+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+filepath.Base(c)+" "))
				}
			}
		}
		if len(m.tabCandidates) > 4 {
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiCandMore", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(i18n.T("tuiCandLabel")) + strings.Join(badges, " ") + "\n")
	}

	// 2.3 TargetDir
	prefix2 := "  "
	if m.configFocusIdx == 2 {
		prefix2 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiSessionTargetDirLabel", prefix2, StatusLabel.Render("[3/7]")))
	b.WriteString(m.targetDirInput.View() + "\n")
	if m.configFocusIdx == 2 && len(m.tabCandidates) > 0 {
		var badges []string
		for i, c := range m.tabCandidates {
			if i < 4 {
				if i == m.tabCandidateIdx {
					badges = append(badges, BadgeSuccess.Render(" "+filepath.Base(c)+" "))
				} else {
					badges = append(badges, BadgeInfo.Render(" "+filepath.Base(c)+" "))
				}
			}
		}
		if len(m.tabCandidates) > 4 {
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiCandMore", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render(i18n.T("tuiCandLabel")) + strings.Join(badges, " ") + "\n")
	}

	// 2.4 Geosync
	prefix3 := "  "
	if m.configFocusIdx == 3 {
		prefix3 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(i18n.T("tuiSessionGeosyncLabel", prefix3, StatusLabel.Render("[4/7]")))
	b.WriteString(m.geosyncInput.View() + "\n")

	// 2.5 Flat Mode
	prefix4 := "  "
	if m.configFocusIdx == 4 {
		prefix4 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	var flatBadge string
	if m.flatMode {
		flatBadge = BadgeSuccess.Render(i18n.T("flatBadgeFlat"))
	} else {
		flatBadge = BadgeInfo.Render(i18n.T("flatBadgeHierarchy"))
	}
	b.WriteString(i18n.T("tuiSessionFlatModeLabel", prefix4, StatusLabel.Render("[5/7]"), flatBadge))

	// 2.6 AllowNoGPS
	prefix5 := "  "
	if m.configFocusIdx == 5 {
		prefix5 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	var gpsBadge string
	if m.allowNoGPS {
		gpsBadge = BadgeSuccess.Render(i18n.T("gpsBadgeSoft"))
	} else {
		gpsBadge = BadgeWarning.Render(i18n.T("gpsBadgeStrict"))
	}
	b.WriteString(i18n.T("tuiSessionAllowNoGPSLabel", prefix5, StatusLabel.Render("[6/7]"), gpsBadge))

	// 2.7 TestBackup
	prefix6 := "  "
	if m.configFocusIdx == 6 {
		prefix6 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	var bakBadge string
	if m.testBackup {
		bakBadge = BadgeSuccess.Render(i18n.T("bakBadgeEnabled"))
	} else {
		bakBadge = lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("bakBadgeDisabled"))
	}
	b.WriteString(i18n.T("tuiSessionTestBackupLabel", prefix6, StatusLabel.Render("[7/7]"), bakBadge))

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(m.statusMessage) + "\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(i18n.T("tuiSessionShortcuts")))
	return PanelStyle.Width(m.panelWidth()).Render(b.String())
}
