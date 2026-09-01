package tui

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vincentchyu/photools/internal/capabilities/datearchive"
	"github.com/vincentchyu/photools/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photools/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photools/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/engine"
	"github.com/vincentchyu/photools/internal/i18n"
	"github.com/vincentchyu/photools/pkg/geocoding"
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
	cap domain.Capability
	id  domain.CapabilityID
}

func (p pluginItem) Title() string {
	if p.cap != nil {
		return p.cap.Name()
	}
	return string(p.id)
}

func (p pluginItem) Desc() string {
	if p.cap != nil {
		return p.cap.Description()
	}
	return ""
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

	// 全局持久设置表单字段 (stateGlobalSettings - 持久化至 plugins.json)
	globalFocusIdx     int
	gpxDirInput        textinput.Model
	logDirInput        textinput.Model
	sidecarPolicy      domain.SidecarPolicy
	companionExtsInput textinput.Model
	rawExtsInput       textinput.Model
	workersInput       textinput.Model

	// 子插件专属设置表单字段 (statePluginSettings)
	pluginFocusIdx     int
	pluginSettingInput textinput.Model
	pluginChoiceIdx    int

	// 会话执行与预检确认字段 (stateConfig - 当前批次临时生效，退出即销毁)
	configFocusIdx int
	baseDirInput   textinput.Model
	sourceDirInput textinput.Model
	targetDirInput textinput.Model
	geosyncInput   textinput.Model
	flatMode       bool
	allowNoGPS     bool
	testBackup     bool

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

func buildInitialReports(items []pluginItem) map[domain.CapabilityID]domain.PluginInitReport {
	reports := make(map[domain.CapabilityID]domain.PluginInitReport, len(items))
	for _, it := range items {
		reports[it.id] = domain.PluginInitReport{
			PluginID: it.id,
			Name:     it.Title(),
			Stage:    i18n.T("stagePrecheck"),
			Message:  i18n.T("tuiInitPreparingExifTool"),
			Percent:  0.0,
			Status:   domain.HealthReady,
		}
	}
	return reports
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
	if sessionCfg.Global.Language != "" {
		i18n.SetLanguage(sessionCfg.Global.Language)
	}

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

	gpxInput := textinput.New()
	gpxInput.SetValue(sessionCfg.Global.GPXDir)
	gpxInput.Placeholder = "GPX 轨迹库目录 (默认 ~/.config/gpx)"
	gpxInput.Width = 60

	logInput := textinput.New()
	logInput.SetValue(sessionCfg.Global.LogDir)
	logInput.Placeholder = "全局日志与报告目录 (默认 ~/.logs/photools)"
	logInput.Width = 60

	compExtsInput := textinput.New()
	compExtsInput.SetValue(strings.Join(sessionCfg.Global.CompanionExtensions, ","))
	compExtsInput.Placeholder = "逗号/空格分隔伴随扩展名 (如 wav, acr, exf)"

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

	pluginItems := []pluginItem{
		{
			cap: gpxmatch.NewCapability(gpxmatch.Config{}),
			id:  domain.CapGPXMatching,
		},
		{
			cap: gpsinterpolate.NewCapability(gpsinterpolate.Config{}),
			id:  domain.CapGPSInterpolate,
		},
		{
			cap: reversegeocode.NewCapability(reversegeocode.Config{}),
			id:  domain.CapReverseGeocode,
		},
		{
			cap: datearchive.NewCapability(datearchive.Config{}),
			id:  domain.CapDateArchive,
		},
	}

	initialReports := buildInitialReports(pluginItems)

	m := Model{
		state:              stateInitializing,
		initReports:        initialReports,
		initChan:           initChan,
		initDoneChan:       initDoneChan,
		sessionConfig:      sessionCfg,
		pluginsConfig:      pluginsCfg,
		enableGPXMatch:     true,
		enableInterpolate:  false,
		enableGeocode:      true,
		allowNoGPS:         sessionCfg.Global.AllowNoGPS,
		sidecarPolicy:      domain.NormalizePolicy(sessionCfg.Global.SidecarPolicy),
		flatMode:           sessionCfg.Global.FlatMode,
		testBackup:         sessionCfg.Global.TestBackup,
		enableArchive:      true,
		pluginIndex:        0,
		pluginItems:        pluginItems,
		currentBaseDir:     defaultBaseDir,
		quickPaths:         quickPaths,
		gpxDirInput:        gpxInput,
		logDirInput:        logInput,
		baseDirInput:       baseInput,
		sourceDirInput:     srcInput,
		rawExtsInput:       rawExtsInput,
		companionExtsInput: compExtsInput,
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

	// 检查规范目录
	gpxDir := m.sessionConfig.Global.GPXDir
	if gpxDir == "" {
		gpxDir = config.DefaultGPXDir()
	}
	resolvedGPXDir := gpxDir
	if len(resolvedGPXDir) >= 2 && resolvedGPXDir[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			resolvedGPXDir = filepath.Join(home, resolvedGPXDir[2:])
		}
	}
	m.dirSpecs = engine.InspectStandardDirectories(baseDir, resolvedGPXDir)

	// 刷新离线地理库状态
	if geocoding.GetDefault() != nil {
		m.geoStats = geocoding.GetDefault().GetStats()
	}

	// 统计数据
	srcDir := m.sourceDirInput.Value()
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

	if gpxFiles, err := engine.ListGPXFiles(resolvedGPXDir); err == nil {
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
		m.startPluginsInitCmd(),
		listenForInitReport(m.initChan),
		listenForInitDone(m.initDoneChan),
	)
}
