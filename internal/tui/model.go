package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/tasks/geotag"
	"github.com/vincentchyu/photo-processing/internal/tasks/organize"
)

type viewState int

const (
	stateMenu viewState = iota
	stateSettings
	stateConfig
	stateDryRun
	stateExecuting
	stateSummary
)

type menuAction int

const (
	actionGeotag menuAction = iota
	actionOrganize
	actionInspect
)

// Msg 定义
type (
	clearStatusMsg struct{}
	eventMsg       domain.ProgressEvent
	taskDoneMsg    struct {
		summary *domain.TaskSummary
		issues  []domain.Issue
		err     error
	}
	planDoneMsg struct {
		plan *domain.PlanResult
		err  error
	}
)

func clearStatusCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

type menuItem struct {
	title  string
	desc   string
	action menuAction
}

type Model struct {
	state      viewState
	width      int
	height     int
	menuIndex  int
	menuItems  []menuItem
	selectedOp menuAction

	// 工作目录与规范目录信息
	currentBaseDir string
	dirSpecs       []engine.DirMeaning
	statusMessage  string

	// 配置字段
	baseDirInput   textinput.Model
	geosyncInput   textinput.Model
	rawExtsInput   textinput.Model
	sourceDirInput textinput.Model
	targetDirInput textinput.Model
	configFocusIdx int

	// 任务与运行状态
	currentTask domain.Task
	planResult  *domain.PlanResult
	planIndex   int // Dry-run 列表滚动

	// 执行与监控状态
	spinner      spinner.Model
	progress     progress.Model
	viewport     viewport.Model
	logs         []string
	taskSummary  *domain.TaskSummary
	taskIssues   []domain.Issue
	taskErr      error
	executing    bool
	cancelFunc   context.CancelFunc
	currentStage domain.PipelineStage
	currentAsset string
	processedNum int
	totalNum     int
	issuesScroll int

	// 统计概览
	inboxAssetCount int
	gpxCount        int
	geotagCount     int
	organizeCount   int

	// 路径补全与快捷预设
	tabCandidates   []string
	tabCandidateIdx int
	quickPaths      []string

	// 异步事件流与结算管道
	eventChan      chan domain.ProgressEvent
	doneChan       chan taskDoneMsg
	archiveDetails []string
	summaryIndex   int
}

func InitialModel(defaultBaseDir string) Model {
	wd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	if defaultBaseDir == "" {
		if wd != "" {
			defaultBaseDir = wd
		} else {
			defaultBaseDir = filepath.Join(home, "Pictures", "GPS")
		}
	}
	defaultBaseDir, _ = filepath.Abs(defaultBaseDir)

	// 快捷预设路径列表
	quickPaths := []string{}
	if wd != "" {
		quickPaths = append(quickPaths, wd)
	}
	if home != "" {
		quickPaths = append(quickPaths, filepath.Join(home, "Pictures", "GPS"))
		quickPaths = append(quickPaths, filepath.Join(home, "Pictures"))
	}

	// 文本输入框初始化
	baseInput := textinput.New()
	baseInput.SetValue(defaultBaseDir)
	baseInput.Placeholder = "输入路径，按 [Tab] 自动补全"
	baseInput.Width = 60
	baseInput.Focus()

	geosyncInput := textinput.New()
	geosyncInput.SetValue("0")
	geosyncInput.Placeholder = "如 +00:00:05 或 -00:01:00"

	rawExtsInput := textinput.New()
	rawExtsInput.SetValue("nef,cr3,arw,dng,raf,rw2,orf")
	rawExtsInput.Placeholder = "逗号分隔扩展名"

	srcInput := textinput.New()
	srcInput.SetValue(filepath.Join(defaultBaseDir, "Inbox"))
	srcInput.Placeholder = "待整理源目录"

	tgtInput := textinput.New()
	tgtInput.SetValue(filepath.Join(defaultBaseDir, "Processed", "organize"))
	tgtInput.Placeholder = "归档目标根目录"

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	vp := viewport.New(80, 15)

	m := Model{
		state:          stateMenu,
		menuIndex:      0,
		selectedOp:     actionGeotag,
		currentBaseDir: defaultBaseDir,
		quickPaths:     quickPaths,
		menuItems: []menuItem{
			{
				title:  "[1] GPS 轨迹修正与归档 (Geotag & Archive)",
				desc:   "从 Inbox 读取 RAW+JPG，结合 GPX 轨迹写入经纬度并归档至 Processed/geotag/YYYY/MMDD/",
				action: actionGeotag,
			},
			{
				title:  "[2] 按拍摄日期归档整理 (Organize by Date)",
				desc:   "从 Inbox 扫描照片，提取 EXIF 拍摄日期，规范重命名并归档至 Processed/organize/YYYY/MMDD/",
				action: actionOrganize,
			},
			{
				title:  "[3] 资产预检与健康体检 (Dry-Run / Inspect)",
				desc:   "仅扫描 Inbox 与 GPX 目录，评估配对状态与时间覆盖，不产生实际写入与移动",
				action: actionInspect,
			},
		},
		baseDirInput:   baseInput,
		geosyncInput:   geosyncInput,
		rawExtsInput:   rawExtsInput,
		sourceDirInput: srcInput,
		targetDirInput: tgtInput,
		spinner:        s,
		progress:       p,
		viewport:       vp,
		currentStage:   domain.StageDiscover,
	}

	m.refreshWorkspace(defaultBaseDir)
	return m
}

func (m *Model) refreshWorkspace(baseDir string) {
	abs, err := filepath.Abs(baseDir)
	if err == nil {
		baseDir = abs
	}
	m.currentBaseDir = baseDir
	m.baseDirInput.SetValue(baseDir)
	m.sourceDirInput.SetValue(filepath.Join(baseDir, "Inbox"))
	m.targetDirInput.SetValue(filepath.Join(baseDir, "Processed", "organize"))

	// 检查规范目录
	m.dirSpecs = engine.InspectStandardDirectories(baseDir)

	// 统计数据
	inboxDir := filepath.Join(baseDir, "Inbox")
	gpxDir := filepath.Join(baseDir, "GPX")
	geotagDir := filepath.Join(baseDir, "Processed", "geotag")
	organizeDir := filepath.Join(baseDir, "Processed", "organize")

	d := engine.NewDiscoverer([]string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"})
	if groups, err := d.Discover(inboxDir); err == nil {
		m.inboxAssetCount = len(groups)
	} else {
		m.inboxAssetCount = 0
	}

	if gpxFiles, err := geotag.ListGPXFiles(gpxDir); err == nil {
		m.gpxCount = len(gpxFiles)
	} else {
		m.gpxCount = 0
	}

	if groups, err := d.Discover(geotagDir); err == nil {
		m.geotagCount = len(groups)
	} else {
		m.geotagCount = 0
	}

	if groups, err := d.Discover(organizeDir); err == nil {
		m.organizeCount = len(groups)
	} else {
		m.organizeCount = 0
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
	)
}

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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m, tea.Quit
		}

		switch m.state {
		case stateMenu:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "r", "R":
				// 重新采集当前工作区上下文
				m.refreshWorkspace(m.currentBaseDir)
				m.statusMessage = "已重新采集并刷新当前工作区数据上下文！"
				return m, clearStatusCmd(3 * time.Second)
			case "s", "S":
				m.state = stateSettings
				m.baseDirInput.Focus()
			case "c", "C":
				// 一键初始化规范目录
				if _, err := engine.EnsureStandardDirectories(m.currentBaseDir); err == nil {
					m.statusMessage = "已成功初始化当前工作目录下的全部规范子目录！"
					m.refreshWorkspace(m.currentBaseDir)
				} else {
					m.statusMessage = fmt.Sprintf("初始化目录失败: %v", err)
				}
				return m, clearStatusCmd(3 * time.Second)
			case "up", "k":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "down", "j":
				if m.menuIndex < len(m.menuItems)-1 {
					m.menuIndex++
				}
			case "enter":
				m.selectedOp = m.menuItems[m.menuIndex].action
				if m.selectedOp == actionInspect {
					_, _ = engine.EnsureStandardDirectories(m.currentBaseDir)
					m.refreshWorkspace(m.currentBaseDir)
					m.state = stateDryRun
					return m, m.startPlanCmd()
				}
				m.state = stateConfig
				m.configFocusIdx = 0
				m.updateConfigFocus()
			}

		case stateSettings:
			switch msg.String() {
			case "esc":
				m.state = stateMenu
				m.tabCandidates = nil
			case "enter":
				newPath := strings.TrimSpace(m.baseDirInput.Value())
				if newPath != "" {
					newPath = engine.ExpandUserPath(newPath)
					_, _ = engine.EnsureStandardDirectories(newPath)
					m.refreshWorkspace(newPath)
					m.statusMessage = fmt.Sprintf("工作目录已更新为: %s", newPath)
				}
				m.state = stateMenu
				m.tabCandidates = nil
				return m, clearStatusCmd(3 * time.Second)
			case "tab":
				// 智能路径补全
				if len(m.tabCandidates) > 1 {
					// 循环切换下一个候选
					m.baseDirInput.SetValue(m.tabCandidates[m.tabCandidateIdx])
					m.baseDirInput.SetCursor(len(m.tabCandidates[m.tabCandidateIdx]))
					m.tabCandidateIdx = (m.tabCandidateIdx + 1) % len(m.tabCandidates)
				} else {
					current := m.baseDirInput.Value()
					completed, candidates := engine.CompleteDirectoryPath(current)
					m.baseDirInput.SetValue(completed)
					m.baseDirInput.SetCursor(len(completed))
					m.tabCandidates = candidates
					m.tabCandidateIdx = 0
				}
				return m, nil
			case "1":
				if len(m.quickPaths) >= 1 && m.baseDirInput.Value() == "" {
					m.baseDirInput.SetValue(m.quickPaths[0])
					m.baseDirInput.SetCursor(len(m.quickPaths[0]))
					m.tabCandidates = nil
					return m, nil
				}
			case "2":
				if len(m.quickPaths) >= 2 && m.baseDirInput.Value() == "" {
					m.baseDirInput.SetValue(m.quickPaths[1])
					m.baseDirInput.SetCursor(len(m.quickPaths[1]))
					m.tabCandidates = nil
					return m, nil
				}
			case "3":
				if len(m.quickPaths) >= 3 && m.baseDirInput.Value() == "" {
					m.baseDirInput.SetValue(m.quickPaths[2])
					m.baseDirInput.SetCursor(len(m.quickPaths[2]))
					m.tabCandidates = nil
					return m, nil
				}
			default:
				// 输入其他按键时重置补全列表
				m.tabCandidates = nil
			}
			m.baseDirInput, _ = m.baseDirInput.Update(msg)

		case stateConfig:
			switch msg.String() {
			case "esc", "b":
				m.state = stateMenu
			case "tab", "down":
				m.nextConfigFocus()
			case "shift+tab", "up":
				m.prevConfigFocus()
			case "enter":
				_, _ = engine.EnsureStandardDirectories(m.currentBaseDir)
				m.refreshWorkspace(m.currentBaseDir)
				m.state = stateDryRun
				return m, m.startPlanCmd()
			}
			m.handleConfigInput(msg)

		case stateDryRun:
			switch msg.String() {
			case "esc", "b":
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
		m.planIndex = 0
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("预检失败: %v", msg.err))
		}

	case eventMsg:
		evt := domain.ProgressEvent(msg)
		m.currentStage = evt.Stage
		if evt.Asset != nil {
			m.currentAsset = evt.Asset.DisplayName()
		}
		if evt.TotalItems > 0 {
			m.processedNum = evt.CurrentIndex
			m.totalNum = evt.TotalItems
		}
		if evt.Asset != nil && (evt.Stage == domain.StageArchive || evt.Issue != nil) {
			namePadded := padRunewidth(evt.Asset.DisplayName(), 34)
			detailLine := fmt.Sprintf("[%2d] %s  ➔  %s", len(m.archiveDetails)+1, namePadded, evt.Message)
			m.archiveDetails = append(m.archiveDetails, detailLine)
		}
		logLine := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), evt.Message)
		m.logs = append(m.logs, logLine)
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		m.viewport.GotoBottom()
		// 持续监听下一个事件
		if m.eventChan != nil {
			return m, listenForEvent(m.eventChan)
		}
		return m, nil

	case taskDoneMsg:
		m.executing = false
		m.taskSummary = msg.summary
		m.taskIssues = msg.issues
		m.taskErr = msg.err
		m.state = stateSummary
		m.issuesScroll = 0
		return m, nil

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case spinner.TickMsg:
		var cmdSpinner tea.Cmd
		m.spinner, cmdSpinner = m.spinner.Update(msg)
		cmds = append(cmds, cmdSpinner)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateConfigFocus() {
	m.geosyncInput.Blur()
	m.rawExtsInput.Blur()
	m.sourceDirInput.Blur()
	m.targetDirInput.Blur()

	if m.selectedOp == actionGeotag {
		switch m.configFocusIdx {
		case 0:
			m.geosyncInput.Focus()
		case 1:
			m.rawExtsInput.Focus()
		}
	} else {
		switch m.configFocusIdx {
		case 0:
			m.sourceDirInput.Focus()
		case 1:
			m.targetDirInput.Focus()
		case 2:
			m.rawExtsInput.Focus()
		}
	}
}

func (m *Model) nextConfigFocus() {
	limit := 2
	if m.selectedOp == actionOrganize {
		limit = 3
	}
	m.configFocusIdx = (m.configFocusIdx + 1) % limit
	m.updateConfigFocus()
}

func (m *Model) prevConfigFocus() {
	limit := 2
	if m.selectedOp == actionOrganize {
		limit = 3
	}
	m.configFocusIdx = (m.configFocusIdx + limit - 1) % limit
	m.updateConfigFocus()
}

func (m *Model) handleConfigInput(msg tea.Msg) {
	if m.selectedOp == actionGeotag {
		switch m.configFocusIdx {
		case 0:
			m.geosyncInput, _ = m.geosyncInput.Update(msg)
		case 1:
			m.rawExtsInput, _ = m.rawExtsInput.Update(msg)
		}
	} else {
		switch m.configFocusIdx {
		case 0:
			m.sourceDirInput, _ = m.sourceDirInput.Update(msg)
		case 1:
			m.targetDirInput, _ = m.targetDirInput.Update(msg)
		case 2:
			m.rawExtsInput, _ = m.rawExtsInput.Update(msg)
		}
	}
}

func (m *Model) buildCurrentTask() (domain.Task, error) {
	rawExts := parseExts(m.rawExtsInput.Value())
	if m.selectedOp == actionGeotag || m.selectedOp == actionInspect {
		return geotag.NewTask(geotag.Config{
			BaseDir:       m.currentBaseDir,
			ProcessedDir:  filepath.Join(m.currentBaseDir, "Processed", "geotag"),
			Geosync:       m.geosyncInput.Value(),
			RawExtensions: rawExts,
			Workers:       runtime.NumCPU(),
		})
	}
	return organize.NewTask(organize.Config{
		SourceDir:     m.sourceDirInput.Value(),
		TargetDir:     m.targetDirInput.Value(),
		RawExtensions: rawExts,
	})
}

func (m *Model) startPlanCmd() tea.Cmd {
	return func() tea.Msg {
		task, err := m.buildCurrentTask()
		if err != nil {
			return planDoneMsg{err: err}
		}
		m.currentTask = task
		plan, err := task.Plan(context.Background())
		return planDoneMsg{plan: plan, err: err}
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

func parseExts(s string) []string {
	parts := strings.Split(s, ",")
	var res []string
	for _, p := range parts {
		c := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(p)), ".")
		if c != "" {
			res = append(res, c)
		}
	}
	return res
}

func (m Model) View() string {
	var content string

	switch m.state {
	case stateMenu:
		content = m.viewMenu()
	case stateSettings:
		content = m.viewSettings()
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
	header := TitleStyle.Render(" 📷 PhotoTools - 摄影工作流自动化处理工作台 v2.0 ")

	var footer string
	switch m.state {
	case stateMenu:
		footer = HelpKeyStyle.Render(" [↑/↓] 选择模式  [Enter] 进入  [s] 切换目录  [r] 刷新数据  [c] 初始化目录  [q] 退出 ")
	case stateSettings:
		footer = HelpKeyStyle.Render(" [Enter] 保存并初始化目录  [Esc] 取消返回 ")
	case stateConfig:
		footer = HelpKeyStyle.Render(" [Tab] 切换输入  [Enter] 预检(Dry-Run)  [Esc] 返回 ")
	case stateDryRun:
		footer = HelpKeyStyle.Render(" [↑/↓] 浏览资产  [Enter] 确认执行  [Esc] 返回配置 ")
	case stateExecuting:
		footer = HelpKeyStyle.Render(" [Ctrl+C] 终止任务  [↑/↓] 滚动查看实时日志 ")
	case stateSummary:
		footer = HelpKeyStyle.Render(" [↑/↓] 查看待处理详情  [Enter/Esc] 返回主菜单  [q] 退出 ")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		"",
		HelpBar.Render(footer),
	)
}

func (m Model) viewMenu() string {
	var b strings.Builder

	// 1. 顶部工作区状态
	b.WriteString(StatusLabel.Render("当前工作区: ") + StatusPath.Render(m.currentBaseDir) + "\n")
	statusBadges := fmt.Sprintf(
		"Inbox: %s  |  GPX: %s  |  Geo已归档: %s  |  日期已归档: %s",
		BadgeInfo.Render(fmt.Sprintf(" %d 组 ", m.inboxAssetCount)),
		BadgeSuccess.Render(fmt.Sprintf(" %d 个 ", m.gpxCount)),
		BadgeSuccess.Render(fmt.Sprintf(" %d 组 ", m.geotagCount)),
		BadgeWarning.Render(fmt.Sprintf(" %d 组 ", m.organizeCount)),
	)
	b.WriteString(statusBadges + "\n")

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(m.statusMessage) + "\n\n")
	} else {
		b.WriteString("\n")
	}

	// 2. 规范目录结构与职责说明（使用紧凑 Table，确保 80 列终端绝不折行）
	b.WriteString(SubTitleStyle.Render("规范目录结构与职责说明：\n"))

	tableWidth := 74
	if m.width > 20 {
		tableWidth = min(80, max(50, m.width-6))
	}

	var rows [][]string
	for _, spec := range m.dirSpecs {
		statusText := "已就绪"
		if !spec.Exists {
			statusText = "未创建(按c建)"
		}

		// 精炼简述，避免超宽折行
		usage := spec.Usage
		switch spec.RelPath {
		case "Inbox":
			usage = "待处理照片源 (RAW+JPG 配对及附属)"
		case "GPX":
			usage = "移动设备导出的 GPX 轨迹文件"
		case filepath.Join("Processed", "geotag"):
			usage = "GPS 修正后按日期归档 (YYYY/MMDD/)"
		case filepath.Join("Processed", "organize"):
			usage = "按拍摄日期整理归档 (YYYY/MMDD/)"
		case "Logs":
			usage = "中文日志与待处理报告 (Markdown)"
		}

		rows = append(rows, []string{
			spec.RelPath,
			usage,
			statusText,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(CardBorderColor)).
		Headers("规范目录", "职责与用途说明", "状态").
		Rows(rows...).
		Width(tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow || row < 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Padding(0, 1)
			}
			if col == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(TextColor).Padding(0, 1)
			}
			if col == 2 && row >= 0 && row < len(rows) {
				if strings.Contains(rows[row][2], "未创建") {
					return lipgloss.NewStyle().Bold(true).Foreground(WarningColor).Padding(0, 1)
				}
				return lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Padding(0, 1)
			}
			return lipgloss.NewStyle().Foreground(MutedTextColor).Padding(0, 1)
		})

	b.WriteString(t.Render() + "\n\n")

	// 3. 模式选择列表
	b.WriteString(SubTitleStyle.Render("请选择处理模式：\n\n"))

	cardWidth := 74
	if m.width > 20 {
		cardWidth = min(80, max(50, m.width-6))
	}

	for i, item := range m.menuItems {
		if i == m.menuIndex {
			cardText := fmt.Sprintf("▶ %s\n  %s", item.title, item.desc)
			cardStyle := lipgloss.NewStyle().
				Background(ActiveBgColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1).
				Width(cardWidth).
				Bold(true)
			b.WriteString(cardStyle.Render(cardText) + "\n\n")
		} else {
			cardText := fmt.Sprintf("  %s\n  ↳ %s", item.title, item.desc)
			cardStyle := lipgloss.NewStyle().
				Foreground(TextColor).
				Padding(0, 1).
				Width(cardWidth)
			b.WriteString(cardStyle.Render(cardText) + "\n\n")
		}
	}

	return b.String()
}

func (m Model) viewSettings() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("⚙️ 设置工作目录 (Root BaseDir)\n\n"))
	b.WriteString("路径输入（支持 ~ 展开与 [Tab] 自动补全）：\n\n")
	b.WriteString(m.baseDirInput.View() + "\n\n")

	if len(m.tabCandidates) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("📁 候选文件夹（按 [Tab] 循环填入）：\n"))
		var candBadges []string
		for i, c := range m.tabCandidates {
			if i >= 6 {
				candBadges = append(candBadges, fmt.Sprintf("...等共 %d 个", len(m.tabCandidates)))
				break
			}
			base := filepath.Base(strings.TrimSuffix(c, string(filepath.Separator))) + "/"
			candBadges = append(candBadges, BadgeInfo.Render(" "+base+" "))
		}
		b.WriteString(strings.Join(candBadges, "  ") + "\n\n")
	}

	if len(m.quickPaths) > 0 {
		b.WriteString("💡 常用预设路径参考：\n")
		for _, p := range m.quickPaths {
			b.WriteString(fmt.Sprintf("  • %s\n", p))
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("操作指引：按 [Tab] 补全/切换候选，按 [Enter] 确认切换并自动建目录，按 [Esc] 取消返回"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("任务参数微调\n\n"))

	if m.selectedOp == actionGeotag {
		b.WriteString(fmt.Sprintf("工作目录: %s\n(输入源: Inbox/  轨迹: GPX/  输出: Processed/geotag/)\n\n", m.currentBaseDir))

		b.WriteString("时钟偏移 -geosync (相机时间偏差补偿，如 +00:00:05 或 -00:01:00):\n")
		b.WriteString(m.geosyncInput.View() + "\n\n")

		b.WriteString("RAW 扩展名列表:\n")
		b.WriteString(m.rawExtsInput.View() + "\n\n")
	} else {
		b.WriteString("源待整理目录 (Source Dir):\n")
		b.WriteString(m.sourceDirInput.View() + "\n\n")

		b.WriteString("目标归档根目录 (Target Dir):\n")
		b.WriteString(m.targetDirInput.View() + "\n\n")

		b.WriteString("RAW 扩展名列表:\n")
		b.WriteString(m.rawExtsInput.View() + "\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(TextColor).Render("按 [Enter] 立即进入 Dry-Run 预检"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewDryRun() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("资产预检与健康体检 (Dry-Run)\n\n"))

	if m.planResult == nil {
		b.WriteString(m.spinner.View() + " 正在扫描与预检资产...")
		return PanelStyle.Render(b.String())
	}

	summaryText := fmt.Sprintf(
		"扫描总计: %d 组   |   ✅ 就绪可执行: %s   |   ⚠️ 待补/跳过: %s",
		m.planResult.TotalAssets,
		BadgeSuccess.Render(fmt.Sprintf(" %d 组 ", m.planResult.ReadyCount)),
		BadgeWarning.Render(fmt.Sprintf(" %d 组 ", m.planResult.PendingCount+m.planResult.WarningsCount)),
	)
	b.WriteString(summaryText + "\n\n")

	total := len(m.planResult.Items)
	if total == 0 {
		b.WriteString("（未在当前工作目录的 Inbox 中发现任何媒体资产）\n")
	} else {
		b.WriteString(fmt.Sprintf("预检明细清单 (第 %d/%d 项，按 [↑/↓] 浏览)：\n", m.planIndex+1, total))

		// 固定高度滑动窗口：始终保持展示 pageSize 行，杜绝滚动时边框高度跳变闪烁
		pageSize := 8
		start := 0
		if total > pageSize {
			start = m.planIndex - pageSize/2
			if start < 0 {
				start = 0
			}
			if start+pageSize > total {
				start = total - pageSize
			}
		}
		end := min(total, start+pageSize)

		for i := start; i < end; i++ {
			item := m.planResult.Items[i]
			namePadded := padRunewidth(item.Asset.DisplayName(), 36)
			prefix := "  "
			if i == m.planIndex {
				prefix = "▶ "
			}

			line := fmt.Sprintf("%s[%2d] %s  ➔  %s", prefix, i+1, namePadded, item.Action)
			if item.Warning != "" {
				line += "  " + BadgeWarning.Render(item.Warning)
			}

			if i == m.planIndex {
				cardStyle := lipgloss.NewStyle().
					Background(ActiveBgColor).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Padding(0, 1)
				b.WriteString(cardStyle.Render(line) + "\n")
			} else {
				lineStyle := lipgloss.NewStyle().
					Foreground(TextColor).
					Padding(0, 1)
				b.WriteString(lineStyle.Render(line) + "\n")
			}
		}
	}

	return PanelStyle.Render(b.String())
}

func padRunewidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m Model) viewExecuting() string {
	var b strings.Builder

	taskName := "任务执行中"
	if m.currentTask != nil {
		taskName = m.currentTask.Name()
	}
	b.WriteString(SubTitleStyle.Render(taskName + " - 流水线执行中\n\n"))

	var stages []domain.PipelineStage
	if m.currentTask != nil {
		stages = m.currentTask.Stages()
	} else if m.selectedOp == actionOrganize {
		stages = []domain.PipelineStage{
			domain.StageDiscover,
			domain.StagePrecheck,
			domain.StageArchive,
		}
	} else {
		stages = []domain.PipelineStage{
			domain.StageDiscover,
			domain.StagePrecheck,
			domain.StageGeotag,
			domain.StageSync,
			domain.StageArchive,
		}
	}

	var stageViews []string
	for _, s := range stages {
		if s == m.currentStage {
			stageViews = append(stageViews, BadgeInfo.Render(string(s)))
		} else {
			stageViews = append(stageViews, lipgloss.NewStyle().Foreground(TextColor).Render(string(s)))
		}
	}
	b.WriteString(strings.Join(stageViews, "  ➜  ") + "\n\n")

	percent := 0.0
	if m.totalNum > 0 {
		percent = float64(m.processedNum) / float64(m.totalNum)
	}
	progressView := m.progress.ViewAs(percent)
	b.WriteString(fmt.Sprintf("%s %s  (%d/%d)\n", m.spinner.View(), progressView, m.processedNum, m.totalNum))
	if m.currentAsset != "" {
		b.WriteString(fmt.Sprintf("当前正在处理: %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.currentAsset)))
	}

	b.WriteString("实时执行日志:\n")
	b.WriteString(PanelStyle.Render(m.viewport.View()))

	return b.String()
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("🎉 任务执行结算与成果报告\n\n"))

	// 1. 顶部统计徽章
	if m.taskSummary != nil {
		stats := fmt.Sprintf(
			"扫描总计: %d 组  |  %s  |  %s  |  %s  |  %s",
			m.taskSummary.TotalAssets,
			BadgeSuccess.Render(fmt.Sprintf(" 成功归档: %d 组 ", m.taskSummary.Success)),
			BadgeWarning.Render(fmt.Sprintf(" 待补/跳过: %d 组 ", m.taskSummary.Pending)),
			BadgeDanger.Render(fmt.Sprintf(" 失败: %d 组 ", m.taskSummary.Failed)),
			BadgeInfo.Render(fmt.Sprintf(" 独立JPG跳过: %d ", m.taskSummary.Skipped)),
		)
		b.WriteString(stats + "\n\n")
	}

	// 2. 成果与归档去向明细（保留刚才处理的每一项，支持滚动）
	if len(m.archiveDetails) > 0 {
		totalDetails := len(m.archiveDetails)
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(fmt.Sprintf("📁 处理成果与归档去向明细 (第 %d/%d 项，按 [↑/↓] 浏览)：\n", m.summaryIndex+1, totalDetails)))

		pageSize := 6
		start := 0
		if totalDetails > pageSize {
			start = m.summaryIndex - pageSize/2
			if start < 0 {
				start = 0
			}
			if start+pageSize > totalDetails {
				start = totalDetails - pageSize
			}
		}
		end := min(totalDetails, start+pageSize)

		for i := start; i < end; i++ {
			line := m.archiveDetails[i]
			if i == m.summaryIndex {
				b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("▶ "+line) + "\n")
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(TextColor).Render("  "+line) + "\n")
			}
		}
		b.WriteString("\n")
	}

	// 3. 待处理异常清单（如有）
	if len(m.taskIssues) > 0 {
		totalIssues := len(m.taskIssues)
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(WarningColor).Render(fmt.Sprintf("⚠️ 待处理异常清单 (%d 项未归档，安全留存源目录)：\n", totalIssues)))

		for i, issue := range m.taskIssues {
			if i >= 3 {
				b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(fmt.Sprintf("  ...等共 %d 项，详情请查看 Markdown 报告\n", totalIssues)))
				break
			}
			b.WriteString(fmt.Sprintf("  • [%s] %s: %s (建议: %s)\n", issue.Kind, issue.Asset.DisplayName(), issue.Reason, issue.Suggestion))
		}
		b.WriteString("\n")
	}

	// 4. 详细执行日志视窗
	b.WriteString("📋 详细执行日志 (按 [↑/↓] 联动滚动)：\n")
	b.WriteString(PanelStyle.Render(m.viewport.View()))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
