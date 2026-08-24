package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vincentchyu/photo-processing/internal/capabilities/datearchive"
	"github.com/vincentchyu/photo-processing/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photo-processing/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photo-processing/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photo-processing/internal/config"
	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
	"github.com/vincentchyu/photo-processing/internal/geocoding"
	"github.com/vincentchyu/photo-processing/internal/pipeline"
)

type viewState int

const (
	stateInitializing viewState = iota
	stateMenu
	stateGlobalSettings
	statePluginSettings
	stateConfig
	stateDryRun
	stateExecuting
	stateSummary
)

// Msg 定义
type (
	clearStatusMsg      struct{}
	initDoneMsg         struct{}
	transitionToMenuMsg struct{}
	eventMsg            domain.ProgressEvent
	taskDoneMsg         struct {
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
	return tea.Tick(
		d, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		},
	)
}

func transitionToMenuCmd(d time.Duration) tea.Cmd {
	return tea.Tick(
		d, func(time.Time) tea.Msg {
			return transitionToMenuMsg{}
		},
	)
}

type pluginItem struct {
	cap   domain.Capability
	id    domain.CapabilityID
	title string
	desc  string
}

type Model struct {
	state  viewState
	width  int
	height int

	// 插件化能力自检与异步初始化
	initReports  map[domain.CapabilityID]domain.PluginInitReport
	initChan     chan domain.PluginInitReport
	initDoneChan chan struct{}
	initFinished bool

	// 统一会话配置 (SessionConfig)
	sessionConfig *config.SessionConfig
	pluginsConfig *config.PluginsConfig

	// 插件化能力开关
	enableGPXMatch    bool // 能力 1: GPX 轨迹匹配与 GPS 修正
	enableInterpolate bool // 能力 1.5: GPS 智能邻近推断与时间插值
	enableGeocode     bool // 能力 2: 逆地理编码写入元数据
	enableArchive     bool // 能力 3: 按拍摄日期归档整理
	pluginIndex       int  // 当前焦点能力 0, 1, 2, 3
	pluginItems       []pluginItem

	// 工作目录与规范目录信息
	currentBaseDir string
	dirSpecs       []engine.DirMeaning
	statusMessage  string

	// 离线地理库状态
	geoStats geocoding.GeocoderLoadStats

	// 全局设置表单字段 (stateGlobalSettings)
	globalFocusIdx int
	baseDirInput   textinput.Model
	sourceDirInput textinput.Model
	rawExtsInput   textinput.Model
	workersInput   textinput.Model
	flatMode       bool
	allowNoGPS     bool
	testBackup     bool

	// 子插件专属设置表单字段 (statePluginSettings)
	pluginFocusIdx     int
	pluginSettingInput textinput.Model
	pluginChoiceIdx    int

	// 确认与执行参数字段 (stateConfig)
	configFocusIdx int
	geosyncInput   textinput.Model
	targetDirInput textinput.Model

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
	processedCount  int

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

func (m Model) hasGeoPacks() bool {
	return len(m.geoStats.Packs) > 0
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

	quickPaths := []string{}
	if wd != "" {
		quickPaths = append(quickPaths, wd)
	}
	if home != "" {
		quickPaths = append(quickPaths, filepath.Join(home, "Pictures", "GPS"))
		quickPaths = append(quickPaths, filepath.Join(home, "Pictures"))
	}

	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, defaultBaseDir)

	baseInput := textinput.New()
	baseInput.SetValue(defaultBaseDir)
	baseInput.Placeholder = "输入路径，按 [Tab] 自动补全"
	baseInput.Width = 60
	baseInput.Focus()

	srcInput := textinput.New()
	srcInput.SetValue(sessionCfg.Global.SourceDir)
	srcInput.Placeholder = "待处理照片源目录"
	srcInput.Width = 60

	rawExtsInput := textinput.New()
	rawExtsInput.SetValue(strings.Join(sessionCfg.Global.RawExtensions, ","))
	rawExtsInput.Placeholder = "逗号分隔扩展名"

	workersInput := textinput.New()
	workersInput.SetValue(strconv.Itoa(sessionCfg.Global.Workers))
	workersInput.Placeholder = "并发协程数"

	pluginSettingInput := textinput.New()
	pluginSettingInput.Width = 40

	geosyncInput := textinput.New()
	geosyncInput.SetValue(sessionCfg.GetStringOption(domain.CapGPXMatching, "geosync", "0"))
	geosyncInput.Placeholder = "如 +00:00:05 或 -00:01:00"

	tgtInput := textinput.New()
	tgtInput.SetValue(filepath.Join(defaultBaseDir, "Processed"))
	tgtInput.Placeholder = "归档目标目录"

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	p := progress.New(progress.WithDefaultGradient())
	vp := viewport.New(80, 15)

	initChan := make(chan domain.PluginInitReport, 100)
	initDoneChan := make(chan struct{}, 1)

	initialReports := map[domain.CapabilityID]domain.PluginInitReport{
		domain.CapGPXMatching: {
			PluginID: domain.CapGPXMatching,
			Name:     "GPX 轨迹匹配与 GPS 修正",
			Stage:    "环境自检",
			Message:  "准备自检 ExifTool 运行环境...",
			Percent:  0.0,
			Status:   domain.HealthReady,
		},
		domain.CapGPSInterpolate: {
			PluginID: domain.CapGPSInterpolate,
			Name:     "GPS 智能邻近推断与时间插值",
			Stage:    "环境自检",
			Message:  "准备加载 GPS 时间插值推算引擎...",
			Percent:  0.0,
			Status:   domain.HealthReady,
		},
		domain.CapReverseGeocode: {
			PluginID: domain.CapReverseGeocode,
			Name:     "逆地理编码与地名元数据写入",
			Stage:    "环境自检",
			Message:  "准备装载离线地理数据包与构建空间索引...",
			Percent:  0.0,
			Status:   domain.HealthReady,
		},
		domain.CapDateArchive: {
			PluginID: domain.CapDateArchive,
			Name:     "按拍摄日期归档与规范重命名",
			Stage:    "环境自检",
			Message:  "准备加载拍摄日期重命名与归档引擎...",
			Percent:  0.0,
			Status:   domain.HealthReady,
		},
	}

	m := Model{
		state:             stateInitializing,
		initReports:       initialReports,
		initChan:          initChan,
		initDoneChan:      initDoneChan,
		sessionConfig:     sessionCfg,
		pluginsConfig:     pluginsCfg,
		enableGPXMatch:    true,
		enableInterpolate: false,
		enableGeocode:     true,
		allowNoGPS:        false,
		enableArchive:     true,
		pluginIndex:       0,
		pluginItems: []pluginItem{
			{
				cap:   gpxmatch.NewCapability(gpxmatch.Config{}),
				id:    domain.CapGPXMatching,
				title: "GPX 轨迹匹配与 GPS 修正 (GPX Matching)",
				desc:  "从 GPX 目录读取轨迹，为 RAW 写入经纬度并同步到 JPG/XMP",
			},
			{
				cap:   gpsinterpolate.NewCapability(gpsinterpolate.Config{}),
				id:    domain.CapGPSInterpolate,
				title: "GPS 智能邻近推断与时间插值 (GPS Interpolation)",
				desc:  "根据同批次前后邻近照片时间权重，自动推算补全无轨迹照片 GPS 坐标",
			},
			{
				cap:   reversegeocode.NewCapability(reversegeocode.Config{}),
				id:    domain.CapReverseGeocode,
				title: "逆地理编码与地名元数据写入 (Reverse Geocode)",
				desc:  "根据 GPS 坐标检索国家/省/市/区/POI，写入 IPTC/XMP 地名元数据",
			},
			{
				cap:   datearchive.NewCapability(datearchive.Config{}),
				id:    domain.CapDateArchive,
				title: "按拍摄日期归档与规范重命名 (Date Archive)",
				desc:  "提取 EXIF 拍摄日期，规范重命名并安全归档至 Processed/YYYY/MMDD/",
			},
		},
		currentBaseDir:     defaultBaseDir,
		quickPaths:         quickPaths,
		baseDirInput:       baseInput,
		sourceDirInput:     srcInput,
		rawExtsInput:       rawExtsInput,
		workersInput:       workersInput,
		pluginSettingInput: pluginSettingInput,
		geosyncInput:       geosyncInput,
		targetDirInput:     tgtInput,
		spinner:            s,
		progress:           p,
		viewport:           vp,
		currentStage:       domain.StageDiscover,
	}

	m.refreshWorkspace(defaultBaseDir)
	return m
}

func (m *Model) refreshWorkspace(baseDir string) {
	tabs, err := filepath.Abs(baseDir)
	if err == nil {
		baseDir = tabs
	}
	m.currentBaseDir = baseDir
	m.baseDirInput.SetValue(baseDir)

	if m.sessionConfig == nil {
		m.sessionConfig = config.NewSessionConfig(m.pluginsConfig, baseDir)
	}
	m.sessionConfig.Global.BaseDir = baseDir

	if m.flatMode {
		m.sourceDirInput.SetValue(baseDir)
		m.targetDirInput.SetValue(baseDir)
		m.sessionConfig.Global.SourceDir = baseDir
		m.sessionConfig.Global.TargetDir = baseDir
	} else {
		m.sourceDirInput.SetValue(filepath.Join(baseDir, "Inbox"))
		m.targetDirInput.SetValue(filepath.Join(baseDir, "Processed"))
		m.sessionConfig.Global.SourceDir = filepath.Join(baseDir, "Inbox")
		m.sessionConfig.Global.TargetDir = filepath.Join(baseDir, "Processed")
	}

	// 重新加载外部插件优先级配置
	if cfg, err := config.LoadPluginsConfig(""); err == nil {
		m.pluginsConfig = cfg
	}

	// 检查规范目录
	m.dirSpecs = engine.InspectStandardDirectories(baseDir)

	// 刷新离线地理库状态
	if geocoding.GetDefault() != nil {
		m.geoStats = geocoding.GetDefault().GetStats()
	}

	// 统计数据
	srcDir := m.sourceDirInput.Value()
	gpxDir := filepath.Join(baseDir, "GPX")
	processedDir := m.targetDirInput.Value()

	rawExts := parseExts(m.rawExtsInput.Value())
	if len(rawExts) == 0 {
		rawExts = []string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"}
	}
	d := engine.NewDiscoverer(rawExts)
	if groups, err := d.Discover(srcDir); err == nil {
		m.inboxAssetCount = len(groups)
	} else {
		m.inboxAssetCount = 0
	}

	if gpxFiles, err := engine.ListGPXFiles(gpxDir); err == nil {
		m.gpxCount = len(gpxFiles)
	} else {
		m.gpxCount = 0
	}

	if groups, err := d.Discover(processedDir); err == nil {
		m.processedCount = len(groups)
	} else {
		m.processedCount = 0
	}
}

func (m Model) startPluginsInitCmd() tea.Cmd {
	ch := m.initChan
	doneCh := m.initDoneChan

	caps := []domain.Capability{
		gpxmatch.NewCapability(gpxmatch.Config{Runner: exiftool.ExecRunner{}}),
		gpsinterpolate.NewCapability(gpsinterpolate.Config{Runner: exiftool.ExecRunner{}}),
		reversegeocode.NewCapability(reversegeocode.Config{Runner: exiftool.ExecRunner{}}),
		datearchive.NewCapability(datearchive.Config{Runner: exiftool.ExecRunner{}}),
	}

	return func() tea.Msg {
		go func() {
			var wg sync.WaitGroup
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			for _, capItem := range caps {
				wg.Add(1)
				go func(c domain.Capability) {
					defer wg.Done()
					_ = c.Init(
						ctx, func(report domain.PluginInitReport) {
							if ch != nil {
								ch <- report
							}
						},
					)
				}(capItem)
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
		m.startPluginsInitCmd(),
		listenForInitReport(m.initChan),
		listenForInitDone(m.initDoneChan),
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
				m.statusMessage = "已重新加载插件配置并刷新工作区上下文！"
				return m, clearStatusCmd(3 * time.Second)
			case "s", "S":
				// 打开全局设置面板
				m.state = stateGlobalSettings
				m.globalFocusIdx = 0
				m.updateGlobalFocus()
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
			case "enter":
				if !m.enableGPXMatch && !m.enableInterpolate && !m.enableGeocode && !m.enableArchive {
					m.statusMessage = "⚠️ 请至少勾选启用一项能力插件！"
					return m, clearStatusCmd(3 * time.Second)
				}
				if m.enableGeocode && !m.hasGeoPacks() {
					m.statusMessage = "⚠️ 提示: 尚未安装离线地理数据包！建议在终端运行 'photools geodata install all' 安装数据库。"
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
				if m.globalFocusIdx == 0 || m.globalFocusIdx == 2 {
					m.handlePathTab(m.globalFocusIdx)
				} else {
					m.nextGlobalFocus()
				}
			case "down", "j":
				m.tabCandidates = nil
				m.nextGlobalFocus()
			case "shift+tab", "up", "k":
				m.tabCandidates = nil
				m.prevGlobalFocus()
			case " ":
				switch m.globalFocusIdx {
				case 1:
					m.flatMode = !m.flatMode
					if m.flatMode {
						m.sourceDirInput.SetValue(m.baseDirInput.Value())
					} else {
						m.sourceDirInput.SetValue(filepath.Join(m.baseDirInput.Value(), "Inbox"))
					}
				case 3:
					m.allowNoGPS = !m.allowNoGPS
				case 5:
					m.testBackup = !m.testBackup
				}
			case "enter":
				m.saveGlobalSettings(false)
				m.state = stateMenu
				m.tabCandidates = nil
				m.statusMessage = "✅ 全局设置已成功应用于当前会话！"
				return m, clearStatusCmd(3 * time.Second)
			case "ctrl+s":
				m.saveGlobalSettings(true)
				m.state = stateMenu
				m.tabCandidates = nil
				m.statusMessage = "💾 全局设置已保存至 ~/.config/photools/plugins.json 并应用于当前会话！"
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
				m.statusMessage = fmt.Sprintf("✅ 插件 [%s] 配置已更新至当前会话！", m.pluginItems[m.pluginIndex].title)
				return m, clearStatusCmd(3 * time.Second)
			case "ctrl+s":
				m.savePluginSettings(true)
				m.state = stateMenu
				m.statusMessage = fmt.Sprintf("💾 插件 [%s] 配置已持久化保存并更新！", m.pluginItems[m.pluginIndex].title)
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
				if m.configFocusIdx == 0 {
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
			case "enter":
				if m.enableGPXMatch && !validateGeosync(m.geosyncInput.Value()) {
					m.statusMessage = "❌ geosync 偏移格式错误 (支持: 0, 5, -5, +00:00:05)"
					return m, clearStatusCmd(3 * time.Second)
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
					m.statusMessage = "⚠️ 当前无就绪可执行的资产，请检查待处理项后按 Esc 返回"
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
				m.statusMessage = "已中断当前任务"
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
	m.baseDirInput.Blur()
	m.sourceDirInput.Blur()
	m.rawExtsInput.Blur()
	m.workersInput.Blur()

	switch m.globalFocusIdx {
	case 0:
		m.baseDirInput.Focus()
	case 2:
		m.sourceDirInput.Focus()
	case 4:
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
	case 0:
		m.baseDirInput, _ = m.baseDirInput.Update(msg)
	case 2:
		m.sourceDirInput, _ = m.sourceDirInput.Update(msg)
	case 4:
		m.rawExtsInput, _ = m.rawExtsInput.Update(msg)
	case 6:
		m.workersInput, _ = m.workersInput.Update(msg)
	}
}

func (m *Model) handlePathTab(focusIdx int) {
	var targetInput *textinput.Model
	switch focusIdx {
	case 0:
		targetInput = &m.baseDirInput
	case 2:
		targetInput = &m.sourceDirInput
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
	targetInput := &m.sourceDirInput
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

	if completed != cur && len(completed) > len(cur) {
		targetInput.SetValue(completed)
		targetInput.SetCursor(len(completed))
		m.tabCandidates = candidates
		m.tabCandidateIdx = 0
		return
	}

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

func (m *Model) saveGlobalSettings(persist bool) {
	newBase := strings.TrimSpace(m.baseDirInput.Value())
	if newBase != "" {
		newBase = engine.ExpandUserPath(newBase)
		m.currentBaseDir = newBase
		m.sessionConfig.Global.BaseDir = newBase
	}

	newSrc := strings.TrimSpace(m.sourceDirInput.Value())
	if newSrc != "" {
		newSrc = engine.ExpandUserPath(newSrc)
		m.sessionConfig.Global.SourceDir = newSrc
	}

	m.sessionConfig.Global.FlatMode = m.flatMode
	m.sessionConfig.Global.AllowNoGPS = m.allowNoGPS
	m.sessionConfig.Global.TestBackup = m.testBackup

	rawExts := parseExts(m.rawExtsInput.Value())
	if len(rawExts) > 0 {
		m.sessionConfig.Global.RawExtensions = rawExts
	}

	if w, err := strconv.Atoi(strings.TrimSpace(m.workersInput.Value())); err == nil && w > 0 {
		m.sessionConfig.Global.Workers = w
	}

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
			m.pluginSettingInput.Placeholder = opt.Description
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
	m.sourceDirInput.Blur()
	m.geosyncInput.Blur()
	m.rawExtsInput.Blur()

	switch m.configFocusIdx {
	case 0:
		m.sourceDirInput.Focus()
	case 1:
		m.geosyncInput.Focus()
	case 2:
		m.rawExtsInput.Focus()
	}
}

func (m *Model) nextConfigFocus() {
	m.configFocusIdx = (m.configFocusIdx + 1) % 3
	m.updateConfigFocus()
}

func (m *Model) prevConfigFocus() {
	m.configFocusIdx = (m.configFocusIdx + 2) % 3
	m.updateConfigFocus()
}

func (m *Model) handleConfigInput(msg tea.Msg) {
	switch m.configFocusIdx {
	case 0:
		m.sourceDirInput, _ = m.sourceDirInput.Update(msg)
	case 1:
		m.geosyncInput, _ = m.geosyncInput.Update(msg)
	case 2:
		m.rawExtsInput, _ = m.rawExtsInput.Update(msg)
	}
}

func (m *Model) buildCurrentTask() (domain.Task, error) {
	rawExts := parseExts(m.rawExtsInput.Value())
	win := m.sessionConfig.GetDurationOption(domain.CapGPSInterpolate, "window", 15*time.Minute)

	return pipeline.Build(
		pipeline.PipelineOptions{
			BaseDir:           m.currentBaseDir,
			SourceDir:         m.sourceDirInput.Value(),
			GPXDir:            filepath.Join(m.currentBaseDir, "GPX"),
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
	m.statusMessage = "正在扫描资产目录并准备预检..."

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

var geosyncPattern = regexp.MustCompile(`^(?:0|[+-]?\d+|[+-]?\d{1,2}:\d{2}:\d{2})$`)

func validateGeosync(s string) bool {
	clean := strings.TrimSpace(s)
	if clean == "" || clean == "0" {
		return true
	}
	return geosyncPattern.MatchString(clean)
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

func calculateWindow(total, current, windowSize int) (int, int) {
	if total <= windowSize {
		return 0, total
	}
	start := current - windowSize/2
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > total {
		end = total
		start = end - windowSize
	}
	return start, end
}

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
	header := TitleStyle.Render(" 📷 PhotoTools - 摄影工作流自动化处理工作台 v2.0 ")

	var footer string
	switch m.state {
	case stateInitializing:
		footer = HelpKeyStyle.Render(" [Ctrl+C] 退出程序 ")
	case stateMenu:
		footer = HelpKeyStyle.Render(" [1/2/3/4/空格] 切换勾选  [o] 插件设置  [s] 全局设置  [a] 全选  [c] 清空  [Enter] 预检执行  [r] 刷新  [q] 退出 ")
	case stateGlobalSettings:
		footer = HelpKeyStyle.Render(" [Tab/↑/↓] 切换输入项  [空格] 切换开关  [Enter] 应用会话  [Ctrl+S] 永久保存  [Esc] 取消 ")
	case statePluginSettings:
		footer = HelpKeyStyle.Render(" [Tab] 切换预设时长  [Enter] 应用会话  [Ctrl+S] 永久保存  [Esc] 取消 ")
	case stateConfig:
		footer = HelpKeyStyle.Render(" [Tab] 切换输入焦点  [Enter] 预检(Dry-Run)  [Esc] 返回 ")
	case stateDryRun:
		footer = HelpKeyStyle.Render(" [↑/↓] 浏览资产  [Enter] 确认执行  [Esc] 返回配置 ")
	case stateExecuting:
		footer = HelpKeyStyle.Render(" [Esc] 终止并结算  [Ctrl+C] 退出程序  [↑/↓] 滚动查看实时日志 ")
	case stateSummary:
		footer = HelpKeyStyle.Render(" [↑/↓] 浏览明细与异常  [Enter/Esc] 返回主菜单  [q] 退出 ")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		"",
		HelpBar.Render(footer),
	)
}

func (m Model) viewInitializing() string {
	var b strings.Builder

	b.WriteString(SubTitleStyle.Render("⚡ 流水线能力插件自检与装载 (Capabilities Self-Check & Loading)...") + "\n\n")

	cardWidth := 74
	if m.width > 20 {
		cardWidth = min(80, max(50, m.width-6))
	}

	readyCount := 0
	for _, item := range m.pluginItems {
		report, has := m.initReports[item.id]
		var statusBadge string
		var detailLine string
		var percent float64 = -1

		if !has {
			statusBadge = lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(
				0, 1,
			).Render(" [⏳] 等待中 ")
			detailLine = "等待调度初始化..."
		} else {
			percent = report.Percent
			switch report.Status {
			case domain.HealthReady:
				if report.Percent >= 1.0 {
					statusBadge = BadgeSuccess.Render(" [✔] 就绪 ")
					readyCount++
				} else {
					statusBadge = BadgeInfo.Render(fmt.Sprintf(" %s 装载中 ", m.spinner.View()))
				}
			case domain.HealthDegraded:
				statusBadge = BadgeWarning.Render(" [⚠️] 降级 ")
				readyCount++
			case domain.HealthFailed:
				statusBadge = BadgeDanger.Render(" [❌] 失败 ")
			default:
				statusBadge = BadgeInfo.Render(fmt.Sprintf(" %s 装载中 ", m.spinner.View()))
			}

			stagePrefix := ""
			if report.Stage != "" {
				stagePrefix = fmt.Sprintf("[%s] ", report.Stage)
			}
			detailLine = stagePrefix + report.Message
		}

		cardText := fmt.Sprintf(
			"%s %s\n    └─ %s", statusBadge, lipgloss.NewStyle().Bold(true).Render(item.title), detailLine,
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
		statusSummary := fmt.Sprintf(
			"🎉 全部 %d / %d 个能力插件自检就绪！正在进入主工作台 (或按 [Enter] 立即进入)...", readyCount,
			len(m.pluginItems),
		)
		b.WriteString(lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(statusSummary) + "\n")
	} else {
		statusSummary := fmt.Sprintf("已就绪: %d / %d 个能力插件", readyCount, len(m.pluginItems))
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(statusSummary) + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("⚙️ 系统正在并发自检环境与流式装载本地离线地理数据包，请稍候..."))
	}

	return PanelStyle.Render(b.String())
}

func (m Model) viewMenu() string {
	var b strings.Builder

	// 1. 顶部工作区状态
	flatBadge := ""
	if m.flatMode {
		flatBadge = " " + BadgeWarning.Render(" [⚡ 扁平原地模式] ")
	}
	b.WriteString(StatusLabel.Render("当前工作区: ") + StatusPath.Render(m.currentBaseDir) + flatBadge + "\n")
	statusBadges := fmt.Sprintf(
		"源目录: %s (%s)  |  GPX: %s  |  已归档: %s",
		m.sourceDirInput.Value(),
		BadgeInfo.Render(fmt.Sprintf(" %d 组 ", m.inboxAssetCount)),
		BadgeSuccess.Render(fmt.Sprintf(" %d 个 ", m.gpxCount)),
		BadgeSuccess.Render(fmt.Sprintf(" %d 组 ", m.processedCount)),
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
	b.WriteString(SubTitleStyle.Render("🧩 摄影处理流水线能力插件 (按 [1/2/3/4/空格] 勾选，按 [o] 调出当前插件专属设置，[s] 全局设置)：") + "\n\n")

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
				paramBadge = " " + BadgeParam.Render(fmt.Sprintf("时钟偏移:%s", sync))
			}
		case 1:
			checked = m.enableInterpolate
			priority = p15
			win := m.sessionConfig.GetStringOption(domain.CapGPSInterpolate, "window", "15m")
			paramBadge = " " + BadgeParam.Render(fmt.Sprintf("推算窗口:%s", win))
		case 2:
			checked = m.enableGeocode
			priority = p2
		case 3:
			checked = m.enableArchive
			priority = p3
			inPlace := m.sessionConfig.GetBoolOption(domain.CapDateArchive, "in_place", false) || m.flatMode
			if inPlace {
				paramBadge = " " + BadgeParam.Render("原地重命名")
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
			priorityBadge = BadgeInfo.Render(fmt.Sprintf(" P%d · 阶段 %d ", priority, phaseIdx))
		} else {
			priorityBadge = lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(
				0, 1,
			).Render(fmt.Sprintf(" P%d (未激活) ", priority))
		}

		// 自检信息与健康状态行
		var initStatusLine string
		if rep, has := m.initReports[item.id]; has && rep.Message != "" {
			var hBadge string
			switch rep.Status {
			case domain.HealthReady:
				hBadge = BadgeSuccess.Render(" ✔ 正常 ")
			case domain.HealthDegraded:
				hBadge = BadgeWarning.Render(" ⚠️ 降级 ")
			case domain.HealthFailed:
				hBadge = BadgeDanger.Render(" ❌ 异常 ")
			default:
				hBadge = BadgeInfo.Render(" 就绪 ")
			}
			initStatusLine = fmt.Sprintf("├─ 环境自检: %s %s", hBadge, rep.Message)
		}

		isFocused := (i == m.pluginIndex)
		var cardText string
		if initStatusLine != "" {
			if isFocused {
				cardText = fmt.Sprintf(
					"▶ %s %s %s%s\n    %s\n    └─ 功能说明: %s", checkMark, priorityBadge, item.title, paramBadge, initStatusLine,
					item.desc,
				)
			} else {
				cardText = fmt.Sprintf(
					"  %s %s %s%s\n    %s\n    └─ 功能说明: %s", checkMark, priorityBadge, item.title, paramBadge, initStatusLine,
					item.desc,
				)
			}
		} else {
			if isFocused {
				cardText = fmt.Sprintf("▶ %s %s %s%s\n    └─ %s", checkMark, priorityBadge, item.title, paramBadge, item.desc)
			} else {
				cardText = fmt.Sprintf("  %s %s %s%s\n    └─ %s", checkMark, priorityBadge, item.title, paramBadge, item.desc)
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
			fmt.Sprintf(
				"⚙️ 会话配置已载入: %s (按 [o] 调整当前插件专属参数，按 [s] 进入全局设置)", config.GetConfigPath(),
			),
		) + "\n",
	)

	return b.String()
}

func (m Model) viewGlobalSettings() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("⚙️ 全局环境与调度设置 (Global Settings)") + "\n\n")

	// 1. BaseDir
	prefix0 := "  "
	if m.globalFocusIdx == 0 {
		prefix0 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s 工作区根目录 (BaseDir / 按 [Tab] 自动补全路径)：\n", prefix0, StatusLabel.Render("[1/7]")))
	b.WriteString(m.baseDirInput.View() + "\n")
	if m.globalFocusIdx == 0 && len(m.tabCandidates) > 0 {
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
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(fmt.Sprintf("等 %d 个候选...", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render("💡 候选路径 (按 [Tab] 循环切换): ") + strings.Join(badges, " ") + "\n")
	}
	b.WriteString("\n")

	// 2. Flat Mode
	prefix1 := "  "
	if m.globalFocusIdx == 1 {
		prefix1 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	flatCheck := "[ ]"
	if m.flatMode {
		flatCheck = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("[✔]")
	}
	b.WriteString(fmt.Sprintf("%s%s %s 扁平原地模式 (Flat Mode / 忽略 Inbox/Processed 结构，直接扫描并就地处理保存)\n\n",
		prefix1, StatusLabel.Render("[2/7]"), flatCheck))

	// 3. SourceDir
	prefix2 := "  "
	if m.globalFocusIdx == 2 {
		prefix2 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s 扫描源目录 (SourceDir / 按 [Tab] 自动补全路径)：\n", prefix2, StatusLabel.Render("[3/7]")))
	b.WriteString(m.sourceDirInput.View() + "\n")
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
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(fmt.Sprintf("等 %d 个候选...", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render("💡 候选路径 (按 [Tab] 循环切换): ") + strings.Join(badges, " ") + "\n")
	}
	b.WriteString("\n")

	// 4. AllowNoGPS
	prefix3 := "  "
	if m.globalFocusIdx == 3 {
		prefix3 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	gpsCheck := "[ ]"
	if m.allowNoGPS {
		gpsCheck = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("[✔]")
	}
	b.WriteString(fmt.Sprintf("%s%s %s 无 GPS 软降级容错 (允许无 GPS 照片跳过地名写入直接归档)\n\n",
		prefix3, StatusLabel.Render("[4/7]"), gpsCheck))

	// 5. RawExts
	prefix4 := "  "
	if m.globalFocusIdx == 4 {
		prefix4 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s RAW 格式识别白名单：\n", prefix4, StatusLabel.Render("[5/7]")))
	b.WriteString(m.rawExtsInput.View() + "\n\n")

	// 6. TestBackup
	prefix5 := "  "
	if m.globalFocusIdx == 5 {
		prefix5 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	bakCheck := "[ ]"
	if m.testBackup {
		bakCheck = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render("[✔]")
	}
	b.WriteString(fmt.Sprintf("%s%s %s 测试安全快照备份 (执行前全量备份至 Inbox_bak)\n\n",
		prefix5, StatusLabel.Render("[6/7]"), bakCheck))

	// 7. Workers
	prefix6 := "  "
	if m.globalFocusIdx == 6 {
		prefix6 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s 并发处理 Worker 协程数：\n", prefix6, StatusLabel.Render("[7/7]")))
	b.WriteString(m.workersInput.View() + "\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("快捷操作：按 [Tab] 补全路径/切换，按 [↑/↓] 切换输入项，按 [空格] 切换开关，按 [Enter] 应用会话，按 [Ctrl+S] 永久保存，按 [Esc] 取消返回"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewPluginSettings() string {
	var b strings.Builder
	item := m.pluginItems[m.pluginIndex]

	b.WriteString(SubTitleStyle.Render(fmt.Sprintf("⚙️ 子插件专属参数设置: %s", item.title)) + "\n\n")
	b.WriteString(fmt.Sprintf("插件说明: %s\n\n", item.desc))

	if item.cap != nil {
		opts := item.cap.SupportedOptions()
		if len(opts) > 0 {
			opt := opts[0]
			b.WriteString(StatusLabel.Render(fmt.Sprintf("📍 %s (%s)：", opt.Name, opt.Key)) + "\n")
			b.WriteString(m.pluginSettingInput.View() + "\n\n")

			if len(opt.Choices) > 0 {
				b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("💡 快捷预设选项 (按 [Tab] 循环切换)：") + "\n")
				var badges []string
				for _, c := range opt.Choices {
					if c == m.pluginSettingInput.Value() {
						badges = append(badges, BadgeSuccess.Render(" "+c+" "))
					} else {
						badges = append(badges, BadgeInfo.Render(" "+c+" "))
					}
				}
				b.WriteString(strings.Join(badges, "  ") + "\n\n")
			}
			b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render(fmt.Sprintf("说明：%s\n\n", opt.Description)))
		}
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("快捷操作：按 [Enter] 应用于当前会话，按 [Ctrl+S] 永久保存至配置文件，按 [Esc] 取消返回"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("⚙️ 摄影处理流水线执行确认 (Pipeline Pre-Flight Check)") + "\n\n")

	// 1. 激活的能力插件与阶段链路看板
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("🧩 1. 本次执行的能力插件与阶段链路：") + "\n")
	var capSummaries []string
	if m.enableGPXMatch {
		sync := m.geosyncInput.Value()
		if sync == "" {
			sync = "0"
		}
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (时钟偏移: %s)",
			BadgeSuccess.Render("阶段 1"), lipgloss.NewStyle().Bold(true).Render("GPX 轨迹匹配"),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(sync)))
	}
	if m.enableInterpolate {
		win := m.sessionConfig.GetStringOption(domain.CapGPSInterpolate, "window", "15m")
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (推算窗口: %s)",
			BadgeSuccess.Render("阶段 2"), lipgloss.NewStyle().Bold(true).Render("GPS 智能时间插值"),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(win)))
	}
	if m.enableGeocode {
		lang := m.sessionConfig.GetStringOption(domain.CapReverseGeocode, "language", "zh-CN")
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (地名语言: %s)",
			BadgeSuccess.Render("阶段 3"), lipgloss.NewStyle().Bold(true).Render("逆地理编码与地名写入"),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(lang)))
	}
	if m.enableArchive {
		inPlace := m.sessionConfig.GetBoolOption(domain.CapDateArchive, "in_place", false) || m.flatMode
		modeStr := "按 Processed/YYYY/MMDD/ 归档"
		if inPlace {
			modeStr = "原地规范重命名 (不移入子目录)"
		}
		capSummaries = append(capSummaries, fmt.Sprintf("  • %s %s (策略: %s)",
			BadgeSuccess.Render("阶段 4"), lipgloss.NewStyle().Bold(true).Render("拍摄日期归档与规范重命名"),
			lipgloss.NewStyle().Foreground(HighlightColor).Render(modeStr)))
	}
	if len(capSummaries) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(DangerColor).Render("  ⚠️ 未勾选任何能力插件！") + "\n")
	} else {
		b.WriteString(strings.Join(capSummaries, "\n") + "\n")
	}
	b.WriteString("\n")

	// 2. 全局环境与安全调度策略
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("🛡️ 2. 全局运行环境与调度策略：") + "\n")
	flatStr := "标准分层模式 (Inbox ➔ Processed)"
	if m.flatMode {
		flatStr = "⚡ 扁平原地模式 (指定目录下直接就地处理与保存)"
	}
	allowGPSStr := "无 GPS 照片在逆地理阶段良性跳过，安全进入拍摄日期归档"
	if !m.allowNoGPS {
		allowGPSStr = "严格模式 (无 GPS 照片将保留在源目录并在报告中提示)"
	}
	bakStr := "关闭 (直接操作源文件)"
	if m.testBackup {
		bakStr = "✅ 开启 (处理前自动全量快照备份至 Inbox_bak)"
	}

	b.WriteString(fmt.Sprintf("  • 工作区根目录: %s\n", lipgloss.NewStyle().Foreground(TextColor).Render(m.currentBaseDir)))
	b.WriteString(fmt.Sprintf("  • 运行目录模式: %s\n", lipgloss.NewStyle().Foreground(HighlightColor).Render(flatStr)))
	b.WriteString(fmt.Sprintf("  • 并发处理协程: %s 个 Worker 协程并发\n", lipgloss.NewStyle().Foreground(HighlightColor).Render(strconv.Itoa(m.sessionConfig.Global.Workers))))
	b.WriteString(fmt.Sprintf("  • 无GPS容错策略: %s\n", lipgloss.NewStyle().Foreground(TextColor).Render(allowGPSStr)))
	b.WriteString(fmt.Sprintf("  • 安全快照备份: %s\n", lipgloss.NewStyle().Foreground(TextColor).Render(bakStr)))
	b.WriteString("\n")

	// 3. 核心可编辑参数快速微调
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render("✏️ 3. 核心参数快速调整 (按 [↑/↓] 移动光标可直接修改)：") + "\n\n")

	prefix0 := "  "
	if m.configFocusIdx == 0 {
		prefix0 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s 扫描源目录 (SourceDir / 按 [Tab] 自动补全路径)：\n", prefix0, StatusLabel.Render("[1/3]")))
	b.WriteString(m.sourceDirInput.View() + "\n")
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
			badges = append(badges, lipgloss.NewStyle().Foreground(MutedTextColor).Render(fmt.Sprintf("等 %d 个候选...", len(m.tabCandidates))))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(PrimaryColor).Render("💡 候选路径 (按 [Tab] 循环切换): ") + strings.Join(badges, " ") + "\n")
	}
	b.WriteString("\n")

	prefix1 := "  "
	if m.configFocusIdx == 1 {
		prefix1 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s 时间偏差补偿 (-geosync)：\n", prefix1, StatusLabel.Render("[2/3]")))
	b.WriteString(m.geosyncInput.View() + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("说明：格式如 0 (无偏移), 15 (快15秒), -00:01:30 (相机慢1分30秒)\n\n"))

	prefix2 := "  "
	if m.configFocusIdx == 2 {
		prefix2 = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
	}
	b.WriteString(fmt.Sprintf("%s%s RAW 格式识别白名单：\n", prefix2, StatusLabel.Render("[3/3]")))
	b.WriteString(m.rawExtsInput.View() + "\n\n")

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(m.statusMessage) + "\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("操作指引：按 [Tab] 补全路径/切换，按 [↑/↓] 切换输入焦点，按 [Enter] 进入任务预检(Dry-Run)，按 [Esc] 返回能力勾选"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewDryRun() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("📋 任务预检与执行计划清单 (Dry-Run Plan)") + "\n\n")

	if m.planResult == nil {
		statusText := m.statusMessage
		if statusText == "" {
			statusText = "正在扫描源目录并准备预检..."
		}
		b.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), lipgloss.NewStyle().Foreground(HighlightColor).Bold(true).Render(statusText)))
		if m.totalNum > 0 {
			pct := float64(m.processedNum) / float64(m.totalNum)
			if pct > 1.0 {
				pct = 1.0
			}
			b.WriteString(fmt.Sprintf("\n  %s  %s\n", m.progress.ViewAs(pct), lipgloss.NewStyle().Foreground(HighlightColor).Render(fmt.Sprintf("%d/%d 组 (%.0f%%)", m.processedNum, m.totalNum, pct*100))))
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(MutedTextColor).Render("⚡ 正在多协程并发极速装载元数据，按 [Esc] 可随时取消返回配置") + "\n")
		return PanelStyle.Render(b.String())
	}

	statsLine := fmt.Sprintf(
		"扫描总资产: %s | 待处理就绪: %s | 待补/跳过: %s | 异常警报: %s",
		BadgeInfo.Render(fmt.Sprintf(" %d ", m.planResult.TotalAssets)),
		BadgeSuccess.Render(fmt.Sprintf(" %d ", m.planResult.ReadyCount)),
		BadgeWarning.Render(fmt.Sprintf(" %d ", m.planResult.PendingCount)),
		BadgeDanger.Render(fmt.Sprintf(" %d ", m.planResult.WarningsCount)),
	)
	b.WriteString(statsLine + "\n\n")

	if len(m.planResult.Items) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("（Inbox 目录中当前没有发现任何待处理照片）") + "\n\n")
	} else {
		maxShow := 8
		start, end := calculateWindow(len(m.planResult.Items), m.planIndex, maxShow)

		for i := start; i < end; i++ {
			item := m.planResult.Items[i]
			prefix := "  "
			if i == m.planIndex {
				prefix = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("▶ ")
			}

			statusBadge := BadgeSuccess.Render(" 就绪 ")
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
			b.WriteString(
				fmt.Sprintf(
					"\n%s\n", lipgloss.NewStyle().Foreground(MutedTextColor).Render(
						fmt.Sprintf(
							"... 当前第 %d/%d 项，按 [↑/↓] 上下滚动查看完整清单", m.planIndex+1, len(m.planResult.Items),
						),
					),
				),
			)
		} else {
			b.WriteString("\n")
		}
	}

	if m.statusMessage != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(m.statusMessage) + "\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("操作指引：按 [↑/↓] 浏览资产，按 [Enter] 确认并正式执行流水线，按 [Esc] 返回配置"))
	return PanelStyle.Render(b.String())
}

func (m Model) viewExecuting() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("⚡ 摄影工作流任务执行中") + "\n\n")

	// 动态从当前 Task 获取 Stages 阶段列表
	var stages []domain.PipelineStage
	if m.currentTask != nil {
		stages = m.currentTask.Stages()
	} else {
		stages = []domain.PipelineStage{domain.StageDiscover, domain.StagePrecheck, domain.StageComplete}
	}

	var stageBadges []string
	for _, st := range stages {
		if st == m.currentStage {
			stageBadges = append(
				stageBadges,
				lipgloss.NewStyle().Background(PrimaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Padding(
					0, 1,
				).Render(string(st)),
			)
		} else {
			stageBadges = append(
				stageBadges,
				lipgloss.NewStyle().Background(lipgloss.Color("#334155")).Foreground(MutedTextColor).Padding(
					0, 1,
				).Render(string(st)),
			)
		}
	}
	b.WriteString(strings.Join(stageBadges, " ➔ ") + "\n\n")

	percent := 0.0
	if m.totalNum > 0 {
		percent = float64(m.processedNum) / float64(m.totalNum)
	}
	progressText := fmt.Sprintf(
		"%s 处理进度: %d/%d (%.1f%%)", m.spinner.View(), m.processedNum, m.totalNum, percent*100,
	)
	b.WriteString(progressText + "\n")
	b.WriteString(m.progress.ViewAs(percent) + "\n\n")

	if m.currentAsset != "" {
		b.WriteString(
			fmt.Sprintf(
				"当前正在处理: %s\n\n", lipgloss.NewStyle().Bold(true).Foreground(PrimaryColor).Render(m.currentAsset),
			),
		)
	}

	b.WriteString("执行实时中文日志流：\n")
	b.WriteString(m.viewport.View() + "\n\n")

	return PanelStyle.Render(b.String())
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(SubTitleStyle.Render("🎉 任务执行结算概览") + "\n\n")

	if m.taskErr != nil {
		b.WriteString(
			lipgloss.NewStyle().Foreground(DangerColor).Bold(true).Render(
				fmt.Sprintf(
					"❌ 执行出错: %v", m.taskErr,
				),
			) + "\n\n",
		)
	}

	if m.taskSummary != nil {
		b.WriteString(
			fmt.Sprintf(
				"资产总数: %s | 成功完成: %s | 待补保留: %s | 失败项: %s\n\n",
				BadgeInfo.Render(fmt.Sprintf(" %d ", m.taskSummary.TotalAssets)),
				BadgeSuccess.Render(fmt.Sprintf(" %d ", m.taskSummary.Success)),
				BadgeWarning.Render(fmt.Sprintf(" %d ", m.taskSummary.Pending)),
				BadgeDanger.Render(fmt.Sprintf(" %d ", m.taskSummary.Failed)),
			),
		)
	}

	if len(m.archiveDetails) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(SuccessColor).Render("📦 本次成功归档明细 (最近):") + "\n")
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
					"⚠️ 发现 %d 项待处理/异常资产 (已生成详细报告 Logs/inbox_pending_report_latest.md)：",
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
					"  [%d] %s - 原因: %s (建议: %s)\n", i+1, issue.Asset.DisplayName(), issue.Reason, issue.Suggestion,
				),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(HighlightColor).Render("📄 实时中文执行日志流已完整保存在: Logs/photools_latest.log") + "\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(MutedTextColor).Render("操作指引：按 [Enter] 或 [Esc] 返回主菜单，按 [q] 退出程序"))
	return PanelStyle.Render(b.String())
}
