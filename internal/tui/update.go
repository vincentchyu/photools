package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vincentchyu/photools/internal/capabilities/datearchive"
	"github.com/vincentchyu/photools/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photools/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photools/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/engine"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/internal/i18n"
	"github.com/vincentchyu/photools/internal/pipeline"
	"github.com/vincentchyu/photools/pkg/geocoding"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = max(20, msg.Width-6)
		m.viewport.Height = max(8, msg.Height-16)
		m.progress.Width = max(20, min(60, msg.Width-30))

	case domain.PluginInitReport:
		m.initReports[msg.PluginID] = msg
		cmds = append(cmds, listenForInitReport(m.initChan))

	case initDoneMsg:
		m.initFinished = true
		if geocoding.GetDefault() != nil {
			m.geoStats = geocoding.GetDefault().GetStats()
		}
		m.refreshWorkspace(m.currentBaseDir)
		cmds = append(cmds, transitionToMenuCmd(600*time.Millisecond))

	case transitionToMenuMsg:
		if m.state == stateInitializing {
			m.state = stateMenu
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m, tea.Quit
		}

		switch m.state {
		case stateInitializing:
			switch msg.String() {
			case "enter", " ", "esc":
				if m.initFinished {
					m.state = stateMenu
				}
			}
			return m, nil

		case stateMenu:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "1":
				m.enableGPXMatch = !m.enableGPXMatch
			case "2":
				m.enableInterpolate = !m.enableInterpolate
			case "3":
				m.enableGeocode = !m.enableGeocode
			case "4":
				m.enableArchive = !m.enableArchive
			case " ", "x", "X":
				switch m.pluginIndex {
				case 0:
					m.enableGPXMatch = !m.enableGPXMatch
				case 1:
					m.enableInterpolate = !m.enableInterpolate
				case 2:
					m.enableGeocode = !m.enableGeocode
				case 3:
					m.enableArchive = !m.enableArchive
				}
			case "a", "A":
				m.enableGPXMatch, m.enableInterpolate, m.enableGeocode, m.enableArchive = true, true, true, true
			case "c", "C":
				m.enableGPXMatch, m.enableInterpolate, m.enableGeocode, m.enableArchive = false, false, false, false
			case "r", "R":
				m.refreshWorkspace(m.currentBaseDir)
				m.statusMessage = i18n.T("tuiStatusRefreshed")
				return m, clearStatusCmd(3 * time.Second)
			case "l", "L":
				if i18n.IsChinese() {
					i18n.SetLanguage("en-US")
				} else {
					i18n.SetLanguage("zh-CN")
				}
				m.sessionConfig.Global.Language = i18n.GetLanguage()
				if m.pluginsConfig != nil {
					m.pluginsConfig.Global.Language = i18n.GetLanguage()
					_ = config.SavePluginsConfig("", m.pluginsConfig)
				}
				m.statusMessage = i18n.T("tuiStatusLangSwitched", i18n.GetLanguage())
				return m, clearStatusCmd(3 * time.Second)
			case "s", "S":
				// 打开全局设置面板并重新同步输入框内容
				m.openGlobalSettings()
			case "o", "O", "e", "E":
				// 打开当前光标选中的子插件专属设置面板
				m.openPluginSettings()
			case "up", "k":
				if m.pluginIndex > 0 {
					m.pluginIndex--
				}
			case "down", "j":
				if m.pluginIndex < len(m.pluginItems)-1 {
					m.pluginIndex++
				}
			case "w", "d":
				m.state = stateConfig
				m.configFocusIdx = 0
				m.updateConfigFocus()
			case "enter":
				if !m.enableGPXMatch && !m.enableInterpolate && !m.enableGeocode && !m.enableArchive {
					m.statusMessage = i18n.T("tuiStatusSelectAtLeastOne")
					return m, clearStatusCmd(3 * time.Second)
				}
				if m.enableGeocode && !m.hasGeoPacks() {
					m.statusMessage = i18n.T("tuiStatusGeodataNotInstalled")
				}
				m.state = stateConfig
				m.configFocusIdx = 0
				m.updateConfigFocus()
			}

		case stateGlobalSettings:
			switch msg.String() {
			case "esc":
				m.state = stateMenu
				m.tabCandidates = nil
			case "tab":
				if m.globalFocusIdx == 1 || m.globalFocusIdx == 2 {
					m.handlePathTab(m.globalFocusIdx)
				} else {
					m.nextGlobalFocus()
				}
			case "down":
				m.tabCandidates = nil
				m.nextGlobalFocus()
			case "shift+tab", "up":
				m.tabCandidates = nil
				m.prevGlobalFocus()
			case " ":
				switch m.globalFocusIdx {
				case 0:
					if i18n.IsChinese() {
						i18n.SetLanguage("en-US")
					} else {
						i18n.SetLanguage("zh-CN")
					}
					m.sessionConfig.Global.Language = i18n.GetLanguage()
					if m.pluginsConfig != nil {
						m.pluginsConfig.Global.Language = i18n.GetLanguage()
						_ = config.SavePluginsConfig("", m.pluginsConfig)
					}
				case 3:
					switch m.sidecarPolicy {
					case domain.PolicySmart, domain.PolicyReadOnly:
						m.sidecarPolicy = domain.PolicySidecarOnly
					case domain.PolicySidecarOnly:
						m.sidecarPolicy = domain.PolicyEmbedAndSidecar
					case domain.PolicyEmbedAndSidecar:
						m.sidecarPolicy = domain.PolicyEmbedOnly
					case domain.PolicyEmbedOnly:
						m.sidecarPolicy = domain.PolicySmart
					default:
						m.sidecarPolicy = domain.PolicySmart
					}
				default:
					m.handleGlobalInput(msg)
				}
			case "enter", "ctrl+s":
				m.saveGlobalSettings(true)
				m.state = stateMenu
				m.tabCandidates = nil
				m.statusMessage = i18n.T("tuiStatusGlobalSaved")
				return m, clearStatusCmd(4 * time.Second)
			default:
				m.tabCandidates = nil
				m.handleGlobalInput(msg)
			}

		case statePluginSettings:
			switch msg.String() {
			case "esc":
				m.state = stateMenu
			case "tab":
				m.cyclePluginChoice()
			case "enter":
				m.savePluginSettings(false)
				m.state = stateMenu
				m.statusMessage = i18n.T("tuiStatusPluginApplied", m.pluginItems[m.pluginIndex].Title())
				return m, clearStatusCmd(3 * time.Second)
			case "ctrl+s", "s":
				m.savePluginSettings(true)
				m.state = stateMenu
				m.statusMessage = i18n.T("tuiStatusPluginSaved", m.pluginItems[m.pluginIndex].Title())
				return m, clearStatusCmd(4 * time.Second)
			default:
				m.pluginSettingInput, _ = m.pluginSettingInput.Update(msg)
			}

		case stateConfig:
			switch msg.String() {
			case "esc":
				m.state = stateMenu
				m.tabCandidates = nil
			case "tab":
				if m.configFocusIdx == 0 || m.configFocusIdx == 1 || m.configFocusIdx == 2 {
					m.handlePathTabConfig()
				} else {
					m.nextConfigFocus()
				}
			case "down", "j":
				m.tabCandidates = nil
				m.nextConfigFocus()
			case "shift+tab", "up", "k":
				m.tabCandidates = nil
				m.prevConfigFocus()
			case " ":
				switch m.configFocusIdx {
				case 4:
					m.flatMode = !m.flatMode
					if m.flatMode {
						m.sourceDirInput.SetValue(m.baseDirInput.Value())
						m.targetDirInput.SetValue(m.baseDirInput.Value())
					} else {
						m.sourceDirInput.SetValue(filepath.Join(m.baseDirInput.Value(), "Inbox"))
						m.targetDirInput.SetValue(filepath.Join(m.baseDirInput.Value(), "Processed"))
					}
				case 5:
					m.allowNoGPS = !m.allowNoGPS
				case 6:
					m.testBackup = !m.testBackup
				}
			case "enter":
				if m.enableGPXMatch && !validateGeosync(m.geosyncInput.Value()) {
					m.statusMessage = i18n.T("tuiStatusInvalidGeosync")
					return m, clearStatusCmd(3 * time.Second)
				}
				// 同步会话中的目录
				newBase := strings.TrimSpace(m.baseDirInput.Value())
				if newBase != "" {
					m.currentBaseDir = engine.ExpandUserPath(newBase)
				}
				m.state = stateDryRun
				m.tabCandidates = nil
				return m, m.startPlanCmd()
			default:
				m.tabCandidates = nil
				m.handleConfigInput(msg)
			}

		case stateDryRun:
			switch msg.String() {
			case "esc":
				m.state = stateConfig
			case "up", "k":
				if m.planIndex > 0 {
					m.planIndex--
				}
			case "down", "j":
				if m.planResult != nil && m.planIndex < len(m.planResult.Items)-1 {
					m.planIndex++
				}
			case "enter":
				if m.planResult != nil && m.planResult.ReadyCount == 0 {
					m.statusMessage = i18n.T("tuiStatusNoReadyAssets")
					return m, clearStatusCmd(4 * time.Second)
				}
				m.state = stateExecuting
				m.logs = nil
				m.archiveDetails = nil
				m.summaryIndex = 0
				m.executing = true
				m.processedNum = 0
				m.totalNum = 0
				return m, m.startExecuteCmd()
			}

		case stateExecuting:
			switch msg.String() {
			case "esc", "q":
				if m.cancelFunc != nil {
					m.cancelFunc()
				}
				m.statusMessage = i18n.T("tuiStatusInterrupted")
				m.executing = false
				m.state = stateSummary
				return m, clearStatusCmd(3 * time.Second)
			}
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)

		case stateSummary:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc", "enter", "b":
				m.state = stateMenu
				m.refreshWorkspace(m.currentBaseDir)
			case "up", "k":
				if m.summaryIndex > 0 {
					m.summaryIndex--
				}
				if m.issuesScroll > 0 {
					m.issuesScroll--
				}
			case "down", "j":
				if m.summaryIndex < len(m.archiveDetails)-1 {
					m.summaryIndex++
				}
				if m.issuesScroll < len(m.taskIssues)-1 {
					m.issuesScroll++
				}
			}
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case planDoneMsg:
		m.planResult = msg.plan
		m.taskErr = msg.err
		m.planIndex = 0
		m.statusMessage = ""

	case eventMsg:
		evt := domain.ProgressEvent(msg)
		m.currentStage = evt.Stage
		if evt.Message != "" {
			m.statusMessage = evt.Message
		}
		if evt.Asset != nil {
			m.currentAsset = evt.Asset.DisplayName()
		}
		if evt.TotalItems > 0 {
			m.totalNum = evt.TotalItems
			m.processedNum = evt.CurrentIndex
		}
		if evt.Stage == domain.StageArchive && evt.Level != domain.LevelError {
			m.archiveDetails = append(m.archiveDetails, evt.Message)
		}

		logLine := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), evt.Message)
		switch evt.Level {
		case domain.LevelSuccess:
			logLine = lipgloss.NewStyle().Foreground(SuccessColor).Render(logLine)
		case domain.LevelWarn:
			logLine = lipgloss.NewStyle().Foreground(WarningColor).Render(logLine)
		case domain.LevelError:
			logLine = lipgloss.NewStyle().Foreground(DangerColor).Render(logLine)
		}
		m.logs = append(m.logs, logLine)
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		m.viewport.GotoBottom()

		if m.eventChan != nil {
			cmds = append(cmds, listenForEvent(m.eventChan))
		}

	case taskDoneMsg:
		m.executing = false
		m.state = stateSummary
		m.taskSummary = msg.summary
		m.taskIssues = msg.issues
		m.taskErr = msg.err
		m.summaryIndex = 0
		m.issuesScroll = 0
		m.refreshWorkspace(m.currentBaseDir)

	case clearStatusMsg:
		m.statusMessage = ""

	case spinner.TickMsg:
		var sCmd tea.Cmd
		m.spinner, sCmd = m.spinner.Update(msg)
		cmds = append(cmds, sCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateGlobalFocus() {
	m.gpxDirInput.Blur()
	m.logDirInput.Blur()
	m.companionExtsInput.Blur()
	m.rawExtsInput.Blur()
	m.workersInput.Blur()

	switch m.globalFocusIdx {
	case 1:
		m.gpxDirInput.Focus()
	case 2:
		m.logDirInput.Focus()
	case 4:
		m.companionExtsInput.Focus()
	case 5:
		m.rawExtsInput.Focus()
	case 6:
		m.workersInput.Focus()
	}
}

func (m *Model) nextGlobalFocus() {
	m.globalFocusIdx = (m.globalFocusIdx + 1) % 7
	m.updateGlobalFocus()
}

func (m *Model) prevGlobalFocus() {
	m.globalFocusIdx = (m.globalFocusIdx + 6) % 7
	m.updateGlobalFocus()
}

func (m *Model) handleGlobalInput(msg tea.Msg) {
	switch m.globalFocusIdx {
	case 1:
		m.gpxDirInput, _ = m.gpxDirInput.Update(msg)
	case 2:
		m.logDirInput, _ = m.logDirInput.Update(msg)
	case 4:
		m.companionExtsInput, _ = m.companionExtsInput.Update(msg)
	case 5:
		m.rawExtsInput, _ = m.rawExtsInput.Update(msg)
	case 6:
		m.workersInput, _ = m.workersInput.Update(msg)
	}
}

func (m *Model) handlePathTab(focusIdx int) {
	var targetInput *textinput.Model
	switch focusIdx {
	case 1:
		targetInput = &m.gpxDirInput
	case 2:
		targetInput = &m.logDirInput
	default:
		return
	}

	cur := targetInput.Value()
	completed, candidates := engine.CompleteDirectoryPath(cur)
	if len(candidates) == 0 {
		return
	}

	if len(candidates) == 1 {
		targetInput.SetValue(candidates[0])
		targetInput.SetCursor(len(candidates[0]))
		m.tabCandidates = nil
		return
	}

	// 多个候选: 若尚未补全到公共前缀，先补全到公共前缀
	if completed != cur && len(completed) > len(cur) {
		targetInput.SetValue(completed)
		targetInput.SetCursor(len(completed))
		m.tabCandidates = candidates
		m.tabCandidateIdx = 0
		return
	}

	// 循环切换候选列表
	if len(m.tabCandidates) > 0 {
		m.tabCandidateIdx = (m.tabCandidateIdx + 1) % len(m.tabCandidates)
		selected := m.tabCandidates[m.tabCandidateIdx]
		targetInput.SetValue(selected)
		targetInput.SetCursor(len(selected))
	} else {
		m.tabCandidates = candidates
		m.tabCandidateIdx = 0
		selected := m.tabCandidates[0]
		targetInput.SetValue(selected)
		targetInput.SetCursor(len(selected))
	}
}

func (m *Model) handlePathTabConfig() {
	var targetInput *textinput.Model
	if m.configFocusIdx == 0 {
		targetInput = &m.baseDirInput
	} else if m.configFocusIdx == 1 {
		targetInput = &m.sourceDirInput
	} else if m.configFocusIdx == 2 {
		targetInput = &m.targetDirInput
	} else {
		return
	}

	cur := targetInput.Value()
	completed, candidates := engine.CompleteDirectoryPath(cur)
	if len(candidates) == 0 {
		return
	}

	if len(candidates) == 1 {
		targetInput.SetValue(candidates[0])
		targetInput.SetCursor(len(candidates[0]))
		m.tabCandidates = nil
		if m.configFocusIdx == 0 {
			m.currentBaseDir = candidates[0]
			if m.flatMode {
				m.sourceDirInput.SetValue(candidates[0])
				m.targetDirInput.SetValue(candidates[0])
			} else {
				m.sourceDirInput.SetValue(filepath.Join(candidates[0], "Inbox"))
				m.targetDirInput.SetValue(filepath.Join(candidates[0], "Processed"))
			}
		}
		return
	}

	if completed != cur && len(completed) > len(cur) {
		targetInput.SetValue(completed)
		targetInput.SetCursor(len(completed))
		m.tabCandidates = candidates
		m.tabCandidateIdx = 0
		if m.configFocusIdx == 0 {
			m.currentBaseDir = completed
			if m.flatMode {
				m.sourceDirInput.SetValue(completed)
				m.targetDirInput.SetValue(completed)
			} else {
				m.sourceDirInput.SetValue(filepath.Join(completed, "Inbox"))
				m.targetDirInput.SetValue(filepath.Join(completed, "Processed"))
			}
		}
		return
	}

	if len(m.tabCandidates) > 0 {
		m.tabCandidateIdx = (m.tabCandidateIdx + 1) % len(m.tabCandidates)
		selected := m.tabCandidates[m.tabCandidateIdx]
		targetInput.SetValue(selected)
		targetInput.SetCursor(len(selected))
		if m.configFocusIdx == 0 {
			m.currentBaseDir = selected
			if m.flatMode {
				m.sourceDirInput.SetValue(selected)
				m.targetDirInput.SetValue(selected)
			} else {
				m.sourceDirInput.SetValue(filepath.Join(selected, "Inbox"))
				m.targetDirInput.SetValue(filepath.Join(selected, "Processed"))
			}
		}
	} else {
		m.tabCandidates = candidates
		m.tabCandidateIdx = 0
		selected := m.tabCandidates[0]
		targetInput.SetValue(selected)
		targetInput.SetCursor(len(selected))
		if m.configFocusIdx == 0 {
			m.currentBaseDir = selected
			if m.flatMode {
				m.sourceDirInput.SetValue(selected)
				m.targetDirInput.SetValue(selected)
			} else {
				m.sourceDirInput.SetValue(filepath.Join(selected, "Inbox"))
				m.targetDirInput.SetValue(filepath.Join(selected, "Processed"))
			}
		}
	}
}

func (m *Model) openGlobalSettings() {
	m.state = stateGlobalSettings
	m.globalFocusIdx = 0
	m.tabCandidates = nil
	if m.sessionConfig != nil {
		m.gpxDirInput.SetValue(m.sessionConfig.Global.GPXDir)
		m.logDirInput.SetValue(m.sessionConfig.Global.LogDir)
		m.sidecarPolicy = domain.NormalizePolicy(m.sessionConfig.Global.SidecarPolicy)
		m.companionExtsInput.SetValue(strings.Join(m.sessionConfig.Global.CompanionExtensions, ","))
		m.rawExtsInput.SetValue(strings.Join(m.sessionConfig.Global.RawExtensions, ","))
		m.workersInput.SetValue(strconv.Itoa(m.sessionConfig.Global.Workers))
	}
	m.updateGlobalFocus()
}

func (m *Model) saveGlobalSettings(persist bool) {
	newGPX := strings.TrimSpace(m.gpxDirInput.Value())
	if newGPX != "" {
		newGPX = engine.ExpandUserPath(newGPX)
	}
	m.sessionConfig.Global.GPXDir = newGPX

	newLog := strings.TrimSpace(m.logDirInput.Value())
	if newLog != "" {
		newLog = engine.ExpandUserPath(newLog)
	}
	m.sessionConfig.Global.LogDir = newLog

	m.sessionConfig.Global.Language = i18n.GetLanguage()
	m.sessionConfig.Global.SidecarPolicy = string(m.sidecarPolicy)
	m.sessionConfig.Global.RawExtensions = parseExts(m.rawExtsInput.Value())
	m.sessionConfig.Global.CompanionExtensions = parseExts(m.companionExtsInput.Value())

	if w, err := strconv.Atoi(strings.TrimSpace(m.workersInput.Value())); err == nil && w > 0 {
		m.sessionConfig.Global.Workers = w
	}

	// 同步主界面当前勾选的插件启用状态到 SessionConfig
	m.sessionConfig.SetPluginEnabled(domain.CapGPXMatching, m.enableGPXMatch)
	m.sessionConfig.SetPluginEnabled(domain.CapGPSInterpolate, m.enableInterpolate)
	m.sessionConfig.SetPluginEnabled(domain.CapReverseGeocode, m.enableGeocode)
	m.sessionConfig.SetPluginEnabled(domain.CapDateArchive, m.enableArchive)

	if m.pluginsConfig == nil {
		def := config.DefaultPluginsConfig()
		m.pluginsConfig = &def
	}
	m.sessionConfig.ApplyToPluginsConfig(m.pluginsConfig)

	m.refreshWorkspace(m.currentBaseDir)

	if persist {
		_ = config.SavePluginsConfig("", m.pluginsConfig)
	}
}

func (m *Model) openPluginSettings() {
	if m.pluginIndex < 0 || m.pluginIndex >= len(m.pluginItems) {
		return
	}
	item := m.pluginItems[m.pluginIndex]
	m.state = statePluginSettings
	m.pluginFocusIdx = 0

	if item.cap != nil {
		opts := item.cap.SupportedOptions()
		if len(opts) > 0 {
			opt := opts[0]
			defStr := fmt.Sprintf("%v", opt.DefaultValue)
			cur := m.sessionConfig.GetStringOption(item.id, opt.Key, defStr)
			m.pluginSettingInput.SetValue(cur)
			m.pluginSettingInput.Placeholder = opt.DisplayDescription()
		}
	}
	m.pluginSettingInput.Focus()
}

func (m *Model) cyclePluginChoice() {
	if m.pluginIndex < 0 || m.pluginIndex >= len(m.pluginItems) {
		return
	}
	item := m.pluginItems[m.pluginIndex]
	if item.cap == nil {
		return
	}
	opts := item.cap.SupportedOptions()
	if len(opts) == 0 || len(opts[0].Choices) == 0 {
		return
	}

	choices := opts[0].Choices
	curVal := strings.TrimSpace(m.pluginSettingInput.Value())
	nextIdx := 0
	for i, c := range choices {
		if c == curVal {
			nextIdx = (i + 1) % len(choices)
			break
		}
	}
	m.pluginSettingInput.SetValue(choices[nextIdx])
}

func (m *Model) savePluginSettings(persist bool) {
	if m.pluginIndex < 0 || m.pluginIndex >= len(m.pluginItems) {
		return
	}
	item := m.pluginItems[m.pluginIndex]
	val := strings.TrimSpace(m.pluginSettingInput.Value())

	if item.cap != nil {
		opts := item.cap.SupportedOptions()
		if len(opts) > 0 {
			opt := opts[0]
			if val == "" {
				val = fmt.Sprintf("%v", opt.DefaultValue)
			}
			if opt.Type == domain.OptionTypeBool {
				m.sessionConfig.SetPluginOption(item.id, opt.Key, val == "true")
			} else {
				m.sessionConfig.SetPluginOption(item.id, opt.Key, val)
			}
			_ = item.cap.Configure(m.sessionConfig.GetPluginOptions(item.id))
		}
	}

	if persist && m.pluginsConfig != nil {
		m.sessionConfig.ApplyToPluginsConfig(m.pluginsConfig)
		_ = config.SavePluginsConfig("", m.pluginsConfig)
	}
}

func (m *Model) updateConfigFocus() {
	m.baseDirInput.Blur()
	m.sourceDirInput.Blur()
	m.targetDirInput.Blur()
	m.geosyncInput.Blur()

	switch m.configFocusIdx {
	case 0:
		m.baseDirInput.Focus()
	case 1:
		m.sourceDirInput.Focus()
	case 2:
		m.targetDirInput.Focus()
	case 3:
		m.geosyncInput.Focus()
	case 4, 5, 6:
		// flatMode, allowNoGPS, testBackup 开关，无需文本框输入焦点
	}
}

func (m *Model) nextConfigFocus() {
	m.configFocusIdx = (m.configFocusIdx + 1) % 7
	m.updateConfigFocus()
}

func (m *Model) prevConfigFocus() {
	m.configFocusIdx = (m.configFocusIdx + 6) % 7
	m.updateConfigFocus()
}

func (m *Model) handleConfigInput(msg tea.Msg) {
	switch m.configFocusIdx {
	case 0:
		var cmd tea.Cmd
		m.baseDirInput, cmd = m.baseDirInput.Update(msg)
		_ = cmd
		newBase := strings.TrimSpace(m.baseDirInput.Value())
		if newBase != "" && newBase != m.currentBaseDir {
			m.currentBaseDir = engine.ExpandUserPath(newBase)
			if m.flatMode {
				m.sourceDirInput.SetValue(newBase)
				m.targetDirInput.SetValue(newBase)
			} else {
				m.sourceDirInput.SetValue(filepath.Join(newBase, "Inbox"))
				m.targetDirInput.SetValue(filepath.Join(newBase, "Processed"))
			}
		}
	case 1:
		m.sourceDirInput, _ = m.sourceDirInput.Update(msg)
	case 2:
		m.targetDirInput, _ = m.targetDirInput.Update(msg)
	case 3:
		m.geosyncInput, _ = m.geosyncInput.Update(msg)
	}
}

func (m *Model) buildCurrentTask() (domain.Task, error) {
	rawExts := parseExts(m.rawExtsInput.Value())
	win := m.sessionConfig.GetDurationOption(domain.CapGPSInterpolate, "window", 15*time.Minute)

	return pipeline.Build(
		pipeline.PipelineOptions{
			BaseDir:           m.currentBaseDir,
			SourceDir:         m.sourceDirInput.Value(),
			GPXDir:            m.sessionConfig.Global.GPXDir,
			ProcessedDir:      m.targetDirInput.Value(),
			Geosync:           m.geosyncInput.Value(),
			RawExtensions:     rawExts,
			Workers:           m.sessionConfig.Global.Workers,
			EnableGPXMatch:    m.enableGPXMatch,
			EnableInterpolate: m.enableInterpolate,
			InterpolateWindow: win,
			EnableGeocode:     m.enableGeocode,
			AllowNoGPS:        m.allowNoGPS,
			EnableArchive:     m.enableArchive,
			FlatMode:          m.flatMode,
			InPlaceArchive:    m.sessionConfig.GetBoolOption(domain.CapDateArchive, "in_place", false),
			Session:           m.sessionConfig,
		},
	)
}

func (m *Model) startPlanCmd() tea.Cmd {
	m.planResult = nil
	m.processedNum = 0
	m.totalNum = 0
	m.statusMessage = i18n.T("tuiStatusScanning")

	task, err := m.buildCurrentTask()
	if err != nil {
		return func() tea.Msg {
			return planDoneMsg{err: err}
		}
	}
	m.currentTask = task

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	eventCh := make(chan domain.ProgressEvent, 500)
	m.eventChan = eventCh
	planDoneCh := make(chan planDoneMsg, 1)

	go func() {
		var plan *domain.PlanResult
		var planErr error
		if pTask, ok := task.(interface {
			PlanWithProgress(ctx context.Context, eventCh chan<- domain.ProgressEvent) (*domain.PlanResult, error)
		}); ok {
			plan, planErr = pTask.PlanWithProgress(ctx, eventCh)
		} else {
			plan, planErr = task.Plan(ctx)
		}
		planDoneCh <- planDoneMsg{plan: plan, err: planErr}
		close(planDoneCh)
	}()

	return tea.Batch(
		listenForEvent(eventCh),
		listenForPlanDone(planDoneCh),
	)
}

func listenForPlanDone(ch <-chan planDoneMsg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) startExecuteCmd() tea.Cmd {
	task, err := m.buildCurrentTask()
	if err != nil {
		return func() tea.Msg {
			return taskDoneMsg{err: err}
		}
	}
	m.currentTask = task

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	eventCh := make(chan domain.ProgressEvent, 500)
	m.eventChan = eventCh

	doneCh := make(chan taskDoneMsg, 1)
	m.doneChan = doneCh

	go func() {
		summary, issues, execErr := task.Execute(ctx, eventCh)
		doneCh <- taskDoneMsg{
			summary: summary,
			issues:  issues,
			err:     execErr,
		}
		close(doneCh)
	}()

	return tea.Batch(
		listenForEvent(eventCh),
		listenForDone(doneCh),
	)
}

func listenForEvent(ch <-chan domain.ProgressEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		evt, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg(evt)
	}
}

func listenForDone(ch <-chan taskDoneMsg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m Model) startPluginsInitCmd() tea.Cmd {
	ch := m.initChan
	doneCh := m.initDoneChan

	runner := exiftool.DefaultRunner()
	caps := []domain.Capability{
		gpxmatch.NewCapability(gpxmatch.Config{Runner: runner}),
		gpsinterpolate.NewCapability(gpsinterpolate.Config{Runner: runner}),
		reversegeocode.NewCapability(reversegeocode.Config{Runner: runner}),
		datearchive.NewCapability(datearchive.Config{Runner: runner}),
	}

	return func() tea.Msg {
		go func() {
			var wg sync.WaitGroup
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			for _, capItem := range caps {
				wg.Go(
					func() {
						_ = capItem.Init(
							ctx, func(report domain.PluginInitReport) {
								if ch != nil {
									ch <- report
								}
							},
						)
					},
				)
			}

			wg.Wait()
			if doneCh != nil {
				doneCh <- struct{}{}
			}
		}()
		return nil
	}
}

func listenForInitReport(ch <-chan domain.PluginInitReport) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		rep, ok := <-ch
		if !ok {
			return nil
		}
		return rep
	}
}

func listenForInitDone(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		_, ok := <-ch
		if !ok {
			return nil
		}
		return initDoneMsg{}
	}
}
