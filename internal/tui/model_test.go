package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vincentchyu/photo-processing/internal/domain"
)

func TestInitialModel(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	if m.state != stateMenu {
		t.Errorf("初始状态期望 stateMenu，实际: %d", m.state)
	}

	if m.currentBaseDir != tempDir {
		t.Errorf("初始 currentBaseDir 异常: %s", m.currentBaseDir)
	}

	if len(m.menuItems) != 3 {
		t.Errorf("期望 3 个主菜单项，实际: %d", len(m.menuItems))
	}

	if len(m.dirSpecs) != 5 {
		t.Errorf("期望 5 个规范目录定义，实际: %d", len(m.dirSpecs))
	}
}

func TestModel_ViewRenderAllStates(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	// 1. 测试 viewMenu 渲染，确保绝不 panic
	view := m.View()
	if view == "" {
		t.Errorf("viewMenu 返回空内容")
	}

	// 2. 测试 viewSettings 渲染
	m.state = stateSettings
	if m.View() == "" {
		t.Errorf("viewSettings 返回空内容")
	}

	// 3. 测试 viewConfig 渲染
	m.state = stateConfig
	if m.View() == "" {
		t.Errorf("viewConfig 返回空内容")
	}

	// 4. 测试 viewDryRun 渲染
	m.state = stateDryRun
	if m.View() == "" {
		t.Errorf("viewDryRun 返回空内容")
	}

	// 5. 测试 viewExecuting 渲染
	m.state = stateExecuting
	if m.View() == "" {
		t.Errorf("viewExecuting 返回空内容")
	}

	// 6. 测试 viewSummary 渲染
	m.state = stateSummary
	if m.View() == "" {
		t.Errorf("viewSummary 返回空内容")
	}
}

func TestModel_SettingsState(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	// 按 's' 进入设置工作目录模式
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := newM.(Model)

	if model.state != stateSettings {
		t.Errorf("按 s 后期望进入 stateSettings，实际: %d", model.state)
	}

	// 模拟按 Esc 取消返回
	newM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = newM.(Model)

	if model.state != stateMenu {
		t.Errorf("按 Esc 后期望返回 stateMenu，实际: %d", model.state)
	}
}

func TestModel_TabCompletion(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tempDir, "Photos"), 0o755)

	m := InitialModel(tempDir)
	m.state = stateSettings
	m.baseDirInput.SetValue(filepath.Join(tempDir, "Ph"))

	// 模拟按 Tab 触发自动补全
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := newM.(Model)

	expected := filepath.Join(tempDir, "Photos") + string(filepath.Separator)
	if !strings.HasPrefix(model.baseDirInput.Value(), filepath.Join(tempDir, "Photos")) {
		t.Errorf("Tab 补全失败: 期望以 %s 开头，实际: %s", expected, model.baseDirInput.Value())
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

	// 验证结算视图包含成果明细
	view := model.View()
	if !strings.Contains(view, "处理成果与归档去向明细") {
		t.Errorf("viewSummary 应该包含处理成果与归档去向明细")
	}
}

func TestModel_EnsureDirectories(t *testing.T) {
	tempDir := t.TempDir()
	m := InitialModel(tempDir)

	// 模拟按 'c' 初始化规范目录
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model := newM.(Model)

	// 检查 Processed/geotag 和 Processed/organize 目录是否存在
	geotagDir := filepath.Join(tempDir, "Processed", "geotag")
	if _, err := os.Stat(geotagDir); err != nil {
		t.Errorf("Processed/geotag 未被创建: %v", err)
	}
	organizeDir := filepath.Join(tempDir, "Processed", "organize")
	if _, err := os.Stat(organizeDir); err != nil {
		t.Errorf("Processed/organize 未被创建: %v", err)
	}

	for _, spec := range model.dirSpecs {
		if !spec.Exists {
			t.Errorf("目录 %s 应该存在", spec.RelPath)
		}
	}
}

func TestModel_RefreshKey(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "Inbox")
	_ = os.MkdirAll(inboxDir, 0o755)

	m := InitialModel(tempDir)
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
	if !strings.Contains(model.statusMessage, "刷新") {
		t.Errorf("按 r 刷新后期望有状态提示，实际: %s", model.statusMessage)
	}

	// 模拟定时器到期收到 clearStatusMsg
	clearedM, _ := model.Update(clearStatusMsg{})
	clearedModel := clearedM.(Model)
	if clearedModel.statusMessage != "" {
		t.Errorf("收到 clearStatusMsg 后期望 statusMessage 为空，实际: %s", clearedModel.statusMessage)
	}
}

func TestParseExts(t *testing.T) {
	exts := parseExts(".nef, CR3, .arw , dng ")
	if len(exts) != 4 {
		t.Fatalf("parseExts 长度异常: %v", exts)
	}
	if exts[0] != "nef" || exts[1] != "cr3" || exts[2] != "arw" || exts[3] != "dng" {
		t.Errorf("parseExts 内容异常: %v", exts)
	}
}
