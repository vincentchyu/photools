package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

func TestTraceViewMenu(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateMenu
	m.width = 120
	m.height = 40
	m.pluginIndex = 0 // 聚焦在第 1 个插件，复现用户截图状态

	v := m.View()
	lines := strings.Split(v, "\n")
	for i, l := range lines {
		t.Logf("Line %02d: %s\n", i, l)
	}

	// 验证插件 1 的第一行是否以 "▶" 开头，而不是被空格顶到右边
	foundPlugin1 := false
	for _, l := range lines {
		if strings.Contains(l, "GPX 轨迹匹配") || strings.Contains(l, "GPX Track Matching") {
			foundPlugin1 = true
			if !strings.Contains(l, "▶") {
				t.Errorf("插件 1 第一行未正确包含焦点光标 ▶，实际行: %q", l)
			}
		}
	}
	if !foundPlugin1 {
		t.Errorf("未找到插件 1 渲染文本")
	}
}

func TestInitialModel(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	if m.state != stateInitializing {
		t.Errorf("初始状态期望 stateInitializing，实际: %d", m.state)
	}

	if m.currentBaseDir != tempDir {
		t.Errorf("初始 currentBaseDir 异常: %s", m.currentBaseDir)
	}

	if len(m.pluginItems) != 4 {
		t.Errorf("期望 4 个插件项，实际: %d", len(m.pluginItems))
	}

	if !m.enableGPXMatch || !m.enableGeocode || !m.enableArchive {
		t.Errorf("初始期望 3 项基础能力默认开启，实际: gpx=%v, geocode=%v, archive=%v",
			m.enableGPXMatch, m.enableGeocode, m.enableArchive)
	}

	if len(m.dirSpecs) != 5 {
		t.Errorf("期望 5 个规范目录定义，实际: %d", len(m.dirSpecs))
	}
}

func TestModel_InitializingState(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	// 1. 测试初始加载页渲染
	v0 := m.View()
	if !strings.Contains(v0, "自检与装载") && !strings.Contains(v0, "Self-Check") {
		t.Errorf("初始加载页渲染缺失标题: %s", v0)
	}

	// 2. 模拟收到插件汇报
	rep := domain.PluginInitReport{
		PluginID: domain.CapReverseGeocode,
		Name:     "逆地理编码",
		Stage:    "装载离线包",
		Message:  "正在解析 china.json (450k 点位)...",
		Percent:  0.5,
		Status:   domain.HealthReady,
	}
	newM, cmd := m.Update(rep)
	model := newM.(Model)
	if cmd == nil {
		t.Errorf("收到 PluginInitReport 后必须返回继续监听下一个 report 的 Cmd")
	}

	v1 := model.View()
	if !strings.Contains(v1, "china.json") {
		t.Errorf("加载页未渲染出流式进度汇报信息: %s", v1)
	}

	// 3. 模拟收到初始化完成消息
	newM, cmd = model.Update(initDoneMsg{})
	model = newM.(Model)
	if !model.initFinished {
		t.Errorf("收到 initDoneMsg 后 initFinished 必须为 true")
	}

	v2 := model.View()
	if !strings.Contains(v2, "自检就绪") && !strings.Contains(v2, "Ready") && !strings.Contains(v2, "ready") {
		t.Errorf("初始化完成后未渲染就绪信息: %s", v2)
	}

	// 4. 用户按 Enter 立即进入主菜单
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newM.(Model)
	if model.state != stateMenu {
		t.Errorf("按 Enter 期望进入 stateMenu，实际: %d", model.state)
	}
}

func TestModel_ViewRenderAllStates(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	states := []viewState{
		stateInitializing,
		stateMenu,
		stateGlobalSettings,
		statePluginSettings,
		stateConfig,
		stateDryRun,
		stateExecuting,
		stateSummary,
	}

	for _, st := range states {
		m.state = st
		view := m.View()
		if view == "" {
			t.Errorf("状态 %d 的 View() 渲染结果不应为空", st)
		}
	}
}

func TestModel_PluginToggleKeys(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateMenu

	// 1. 按 '1' 切换能力 1
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model := newM.(Model)
	if model.enableGPXMatch != false {
		t.Errorf("按 1 期望关闭能力 1，实际开启")
	}

	// 2. 按 '2' 切换能力 1.5 (插值)
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = newM.(Model)
	if model.enableInterpolate != true {
		t.Errorf("按 2 期望开启能力 1.5，实际关闭")
	}

	// 3. 按 '3' 切换能力 2
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	model = newM.(Model)
	if model.enableGeocode != false {
		t.Errorf("按 3 期望关闭能力 2，实际开启")
	}

	// 4. 按 '4' 切换能力 3
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	model = newM.(Model)
	if model.enableArchive != false {
		t.Errorf("按 4 期望关闭能力 3，实际开启")
	}

	// 关闭能力 1.5，使其全清空
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = newM.(Model)

	// 全清空时按 Enter 应该被拦截并显示提示
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newM.(Model)
	if model.state != stateMenu {
		t.Errorf("全部未勾选时按 Enter 不应进入配置，实际 state: %d", model.state)
	}
	if !strings.Contains(model.statusMessage, "至少勾选") && !strings.Contains(model.statusMessage, "at least one") {
		t.Errorf("未给出至少勾选提示: %s", model.statusMessage)
	}

	// 5. 按 'a' 全选
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = newM.(Model)
	if !model.enableGPXMatch || !model.enableInterpolate || !model.enableGeocode || !model.enableArchive {
		t.Errorf("按 a 全选失败: gpx=%v, interpolate=%v, geocode=%v, archive=%v",
			model.enableGPXMatch, model.enableInterpolate, model.enableGeocode, model.enableArchive)
	}

	// 6. 按 'c' 清空
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model = newM.(Model)
	if model.enableGPXMatch || model.enableGeocode || model.enableArchive {
		t.Errorf("按 c 清空失败: gpx=%v, geocode=%v, archive=%v",
			model.enableGPXMatch, model.enableGeocode, model.enableArchive)
	}

	// 7. 使用空格切换聚焦项
	model.pluginIndex = 0
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = newM.(Model)
	if !model.enableGPXMatch {
		t.Errorf("按空格期望开启能力 1，实际未开启")
	}
}

func TestModel_GlobalSettingsState(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "plugins.json")
	t.Setenv("PHOTOOLS_PLUGINS_CONFIG", cfgPath)

	m := InitialModel(tempDir)
	m.state = stateMenu

	// 按 's' 进入全局设置模式
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := newM.(Model)

	if model.state != stateGlobalSettings {
		t.Fatalf("按 s 后期望进入 stateGlobalSettings，实际: %d", model.state)
	}

	// 1. 焦点在 Language (index 0) 并按空格切换语言
	model.globalFocusIdx = 0
	i18n.SetLanguage("zh-CN")
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = newM.(Model)

	// 2. 修改 GPXDir
	model.gpxDirInput.SetValue(filepath.Join(tempDir, "custom_gpx"))
	// 3. 修改 LogDir
	model.logDirInput.SetValue(filepath.Join(tempDir, "custom_logs"))
	// 4. 修改 SidecarPolicy 为 sidecar_only
	model.sidecarPolicy = domain.PolicySidecarOnly
	// 5. 修改 CompanionExtensions (空格与逗号混合)
	model.companionExtsInput.SetValue("wav acr exf custom_ext")
	// 6. 修改 RawExtensions
	model.rawExtsInput.SetValue("nef,cr3,dng,myraw")
	// 7. 修改 Workers
	model.workersInput.SetValue("16")

	// 按 Enter 保存并返回
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newM.(Model)

	if model.state != stateMenu {
		t.Fatalf("按 Enter 期望返回 stateMenu，实际: %d", model.state)
	}

	diskData, readErr := os.ReadFile(cfgPath)
	t.Logf("磁盘真实文件内容 (%s, err=%v):\n%s", cfgPath, readErr, string(diskData))

	// 验证内存会话
	if model.sessionConfig.Global.Language != "en-US" {
		t.Errorf("SessionConfig.Global.Language 未更新为 en-US")
	}
	if model.sessionConfig.Global.SidecarPolicy != "sidecar_only" {
		t.Errorf("SessionConfig.Global.SidecarPolicy 期望 sidecar_only，实际: %s", model.sessionConfig.Global.SidecarPolicy)
	}
	if model.sessionConfig.Global.Workers != 16 {
		t.Errorf("SessionConfig.Global.Workers 期望 16，实际: %d", model.sessionConfig.Global.Workers)
	}

	// 验证磁盘 plugins.json 文件
	loaded, err := config.LoadPluginsConfig(cfgPath)
	if err != nil {
		t.Fatalf("读取落盘 plugins.json 失败: %v", err)
	}
	if loaded.Global.Language != "en-US" {
		t.Errorf("磁盘 plugins.json Language 期望 en-US，实际: %s", loaded.Global.Language)
	}
	if loaded.Global.SidecarPolicy != "sidecar_only" {
		t.Errorf("磁盘 plugins.json SidecarPolicy 期望 sidecar_only，实际: %s", loaded.Global.SidecarPolicy)
	}
	if loaded.Global.Workers != 16 {
		t.Errorf("磁盘 plugins.json Workers 期望 16，实际: %d", loaded.Global.Workers)
	}
	if len(loaded.Global.CompanionExtensions) != 4 || loaded.Global.CompanionExtensions[3] != "custom_ext" {
		t.Errorf("磁盘 plugins.json CompanionExtensions 期望包含 custom_ext，实际: %v", loaded.Global.CompanionExtensions)
	}
	if len(loaded.Global.RawExtensions) != 4 || loaded.Global.RawExtensions[3] != "myraw" {
		t.Errorf("磁盘 plugins.json RawExtensions 期望包含 myraw，实际: %v", loaded.Global.RawExtensions)
	}
}

func TestModel_LanguageToggle_KeyL(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "plugins.json")
	t.Setenv("PHOTOOLS_PLUGINS_CONFIG", cfgPath)

	m := InitialModel(tempDir)
	m.state = stateMenu

	i18n.SetLanguage("zh-CN")
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model := newM.(Model)

	if i18n.GetLanguage() != "en-US" {
		t.Fatalf("按 l 键期望切换为 en-US，实际: %s", i18n.GetLanguage())
	}
	if model.sessionConfig.Global.Language != "en-US" {
		t.Errorf("sessionConfig 未同步更新为 en-US")
	}

	loaded, err := config.LoadPluginsConfig(cfgPath)
	if err != nil || loaded.Global.Language != "en-US" {
		t.Errorf("plugins.json 未即时持久化写入 en-US, loaded: %+v, err: %v", loaded, err)
	}

	// 再次按 l 切换回 zh-CN
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model = newM.(Model)

	if i18n.GetLanguage() != "zh-CN" {
		t.Fatalf("再次按 l 键期望切换回 zh-CN，实际: %s", i18n.GetLanguage())
	}
}

func TestModel_PluginSettingsState_P15Window(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateMenu

	// 显式重置为 15m 确保测试环境隔离不受外部 plugins.json 影响
	m.sessionConfig.SetPluginOption(domain.CapGPSInterpolate, "window", "15m")

	// 光标移动到 P15 插值插件 (index 1)
	m.pluginIndex = 1

	// 按 'o' 进入该插件专属设置
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model := newM.(Model)

	if model.state != statePluginSettings {
		t.Errorf("按 o 后期望进入 statePluginSettings，实际: %d", model.state)
	}

	// 模拟按 Tab 循环切换预设 (15m -> 30m -> 1h)
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = newM.(Model)
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = newM.(Model)

	if model.pluginSettingInput.Value() != "1h" {
		t.Errorf("循环切换后期望时间窗口为 1h，实际: %s", model.pluginSettingInput.Value())
	}

	// 按 Enter 保存会话设置
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newM.(Model)

	if model.state != stateMenu {
		t.Errorf("按 Enter 期望返回 stateMenu，实际: %d", model.state)
	}

	// 验证 sessionConfig 中已生效
	dur := model.sessionConfig.GetDurationOption(domain.CapGPSInterpolate, "window", 0)
	if dur != time.Hour {
		t.Errorf("sessionConfig 中 P15 窗口期望 1h，实际: %v", dur)
	}

	// 验证主菜单渲染出 [推算窗口:1h] 或 [window:1h] 徽章
	view := model.View()
	if !strings.Contains(view, "推算窗口:1h") && !strings.Contains(view, "window:1h") {
		t.Errorf("主菜单未正确渲染推算窗口徽章: %s", view)
	}
}

func TestModel_TabCompletion(t *testing.T) {
	tempDir := t.TempDir()
	photosDir := filepath.Join(tempDir, "Photos")
	_ = os.MkdirAll(photosDir, 0o755)

	m := InitialModel(tempDir)
	m.state = stateGlobalSettings
	m.globalFocusIdx = 1 // GPXDir (第 2 项，支持 Tab 补全)
	m.gpxDirInput.SetValue(filepath.Join(tempDir, "Ph"))

	// 模拟按 Tab 触发路径自动补全
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := newM.(Model)

	expectedCompleted := photosDir + string(filepath.Separator)
	if model.gpxDirInput.Value() != expectedCompleted {
		t.Errorf("按 Tab 期望路径自动补全为 %q，实际: %q", expectedCompleted, model.gpxDirInput.Value())
	}
}

func TestModel_GlobalSettings_SidecarOnly(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateGlobalSettings
	m.globalFocusIdx = 3 // 聚焦在 SidecarPolicy 选项 (第 4 项)
	m.sidecarPolicy = domain.PolicySmart

	// 按空格键切换策略为 sidecar_only
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model := newM.(Model)

	if model.sidecarPolicy != domain.PolicySidecarOnly {
		t.Errorf("按空格键后期望 sidecarPolicy 为 sidecar_only，实际为 %v", model.sidecarPolicy)
	}

	// 按 Enter 保存生效
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newM.(Model)

	if model.sessionConfig.Global.SidecarPolicy != string(domain.PolicySidecarOnly) {
		t.Errorf("按 Enter 后期望 sessionConfig.Global.SidecarPolicy 为 sidecar_only，实际为 %v", model.sessionConfig.Global.SidecarPolicy)
	}
	if model.state != stateMenu {
		t.Errorf("保存后期望返回 stateMenu")
	}
}

func TestModel_EventStreamContinuity(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateExecuting
	m.eventChan = make(chan domain.ProgressEvent, 10)

	// 1. 模拟收到第 1 个进度事件与归档事件
	evt1 := domain.ProgressEvent{
		Stage:        domain.StageArchive,
		Message:      "已归档到 2026/0515/ (DSC_2026-05-15_5537)",
		Asset:        &domain.AssetGroup{BaseName: "DSC_5537", RawPath: "/path/DSC_5537.NEF"},
		CurrentIndex: 1,
		TotalItems:   5,
	}
	newM, cmd := m.Update(eventMsg(evt1))
	model := newM.(Model)

	if len(model.archiveDetails) != 1 {
		t.Errorf("期望 archiveDetails 记录 1 条归档明细，实际: %d", len(model.archiveDetails))
	}
	if model.processedNum != 1 || model.totalNum != 5 {
		t.Errorf("进度未更新: %d/%d", model.processedNum, model.totalNum)
	}
	if model.currentStage != domain.StageArchive {
		t.Errorf("阶段未更新: %s", model.currentStage)
	}
	if cmd == nil {
		t.Errorf("收到 eventMsg 后必须返回监听下一个事件的 cmd，不能为 nil")
	}

	// 2. 模拟收到任务完成消息
	doneMsg := taskDoneMsg{
		summary: &domain.TaskSummary{
			TotalAssets: 5,
			Success:     5,
		},
	}
	newM, _ = model.Update(doneMsg)
	model = newM.(Model)

	if model.state != stateSummary {
		t.Errorf("任务完成后期望跳转 stateSummary，实际: %d", model.state)
	}
	if model.taskSummary == nil || model.taskSummary.Success != 5 {
		t.Errorf("TaskSummary 未正确设置")
	}
}

func TestModel_RefreshKey(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "Inbox")
	_ = os.MkdirAll(inboxDir, 0o755)

	m := InitialModel(tempDir)
	m.state = stateMenu
	if m.inboxAssetCount != 0 {
		t.Fatalf("初始资产数应为 0")
	}

	// 模拟外部往 Inbox 写入新文件
	_ = os.WriteFile(filepath.Join(inboxDir, "DSC_0001.NEF"), []byte("dummy raw"), 0o644)
	_ = os.WriteFile(filepath.Join(inboxDir, "DSC_0001.JPG"), []byte("dummy jpg"), 0o644)

	// 模拟按 'r' 刷新工作区
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model := newM.(Model)

	if model.inboxAssetCount != 1 {
		t.Errorf("按 r 刷新后期望 inboxAssetCount 为 1，实际: %d", model.inboxAssetCount)
	}
	if !strings.Contains(model.statusMessage, "刷新") && !strings.Contains(model.statusMessage, "refresh") && !strings.Contains(model.statusMessage, "Refresh") {
		t.Errorf("按 r 刷新后期望有状态提示，实际: %s", model.statusMessage)
	}
}

func TestModel_SummaryIssuesScroll(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateSummary

	// 构造 8 个 issue 项
	var issues []domain.Issue
	for i := 1; i <= 8; i++ {
		issues = append(issues, domain.Issue{
			Kind:       domain.IssueKindMissingPair,
			Reason:     "缺少配对 JPG",
			Suggestion: "请补充同名 JPG",
			Asset:      domain.AssetGroup{BaseName: "DSC_000" + string(rune('0'+i))},
		})
	}
	m.taskIssues = issues
	m.taskSummary = &domain.TaskSummary{TotalAssets: 8, Pending: 8}

	// 验证初始渲染包含了第 1 项
	v1 := m.View()
	if !strings.Contains(v1, "DSC_0001") {
		t.Errorf("初始结算视图未渲染出第 1 项 issue")
	}

	// 向下滚动 5 次
	for i := 0; i < 5; i++ {
		newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newM.(Model)
	}

	if m.issuesScroll != 5 {
		t.Errorf("期望 issuesScroll 为 5，实际: %d", m.issuesScroll)
	}

	// 验证滚动后视图中能渲染出第 6 项
	v2 := m.View()
	if !strings.Contains(v2, "DSC_0006") {
		t.Errorf("滚动后结算视图应该渲染出第 6 项 DSC_0006")
	}
}

func TestModel_ExecutingSoftCancel(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateExecuting

	cancelled := false
	m.cancelFunc = func() {
		cancelled = true
	}

	// 模拟按 Esc 软中断
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)

	if !cancelled {
		t.Errorf("按 Esc 期望触发 cancelFunc 取消在途任务")
	}
	if !strings.Contains(model.statusMessage, "已中断") && !strings.Contains(model.statusMessage, "取消") && !strings.Contains(model.statusMessage, "interrupted") && !strings.Contains(model.statusMessage, "Interrupted") {
		t.Errorf("取消任务后期望有状态提示，实际: %s", model.statusMessage)
	}
}

func TestModel_ConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateConfig
	m.enableGPXMatch = true
	m.geosyncInput.SetValue("invalid_geosync_offset")

	// 模拟按 Enter 提交配置
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if model.state == stateDryRun {
		t.Fatalf("输入非法 geosync 时不应进入 stateDryRun")
	}
	if !strings.Contains(model.statusMessage, "geosync") && !strings.Contains(model.statusMessage, "Invalid") {
		t.Errorf("非法 geosync 应该报错提示格式，实际: %s", model.statusMessage)
	}
}

func TestValidateGeosync(t *testing.T) {
	validCases := []string{
		"0", "", "   ", "+00:00:05", "-00:01:00", "01:23:45",
		"+5", "-10", "123", "+00:00:00",
	}
	for _, c := range validCases {
		if !validateGeosync(c) {
			t.Errorf("期望 %q 校验通过，实际失败", c)
		}
	}

	invalidCases := []string{
		"invalid", "abc", "+1:2", "00:00", "+00:00:00:00", "++5", "--10",
	}
	for _, c := range invalidCases {
		if validateGeosync(c) {
			t.Errorf("期望 %q 校验失败，实际通过", c)
		}
	}
}

func TestCalculateWindow(t *testing.T) {
	s, e := calculateWindow(3, 1, 5)
	if s != 0 || e != 3 {
		t.Errorf("calculateWindow(3, 1, 5) 期望 (0, 3)，实际 (%d, %d)", s, e)
	}

	s, e = calculateWindow(20, 1, 6)
	if s != 0 || e != 6 {
		t.Errorf("calculateWindow(20, 1, 6) 期望 (0, 6)，实际 (%d, %d)", s, e)
	}

	s, e = calculateWindow(20, 10, 6)
	if s != 7 || e != 13 {
		t.Errorf("calculateWindow(20, 10, 6) 期望 (7, 13)，实际 (%d, %d)", s, e)
	}

	s, e = calculateWindow(20, 19, 6)
	if s != 14 || e != 20 {
		t.Errorf("calculateWindow(20, 19, 6) 期望 (14, 20)，实际 (%d, %d)", s, e)
	}
}

func TestModel_SessionSettings_DirectoryAndFlatMode(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)
	m.state = stateMenu

	// 1. 按 'w' 快捷键进入会话与目录设置
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	model := newM.(Model)
	if model.state != stateConfig {
		t.Errorf("按 w 期望进入 stateConfig，实际 state: %d", model.state)
	}

	// 2. 焦点在 BaseDir (idx 0)，模拟输入新目录
	newAlbumDir := filepath.Join(tempDir, "Album2026")
	model.baseDirInput.SetValue(newAlbumDir)
	model.handleConfigInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}})
	if model.currentBaseDir != newAlbumDir {
		t.Errorf("currentBaseDir 期望同步更新为 %s，实际: %s", newAlbumDir, model.currentBaseDir)
	}

	// 3. 轮转到模式开关项 (idx 4)，按空格切换扁平原地模式
	model.configFocusIdx = 4
	initFlat := model.flatMode
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = newM.(Model)
	if model.flatMode == initFlat {
		t.Errorf("按空格期望切换 flatMode 状态")
	}

	// 4. 轮转到无 GPS 软降级项 (idx 5) 与快照备份项 (idx 6)，验证空格切换
	model.configFocusIdx = 5
	initGPS := model.allowNoGPS
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = newM.(Model)
	if model.allowNoGPS == initGPS {
		t.Errorf("按空格期望切换 allowNoGPS 状态")
	}

	model.configFocusIdx = 6
	initBak := model.testBackup
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = newM.(Model)
	if model.testBackup == initBak {
		t.Errorf("按空格期望切换 testBackup 状态")
	}

	// 5. 按 Esc 取消并返回主菜单
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = newM.(Model)
	if model.state != stateMenu {
		t.Errorf("按 Esc 期望返回 stateMenu，实际: %d", model.state)
	}
}
