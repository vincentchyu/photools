package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
)

// Phase 代表由相同 Priority 插件组成的并发执行阶段
type Phase struct {
	Priority     int
	Capabilities []domain.Capability
}

// Config 封装流水线编排器初始化参数
type Config struct {
	Name          string
	Description   string
	SourceDir     string
	Capabilities  []domain.Capability
	RawExtensions []string
	Workers       int
	Runner        exiftool.CommandRunner
	LogDir        string
	IssueFile     string
	LockPath      string
	BackupDir     string
}

// Orchestrator 负责将多个独立能力插件按优先级分阶段调度执行（实现 domain.Task 接口）
type Orchestrator struct {
	name         string
	description  string
	sourceDir    string
	capabilities []domain.Capability
	phases       []Phase
	workers      int
	runner       exiftool.CommandRunner
	logDir       string
	issueFile    string
	lockPath     string
	backupDir    string

	discoverer *engine.Discoverer
	reporter   *engine.Reporter
}

// NewOrchestrator 创建流水线编排器实例，自动按插件 Priority 划分阶段
func NewOrchestrator(cfg Config) (*Orchestrator, error) {
	if cfg.SourceDir == "" {
		return nil, fmt.Errorf("必须指定流水线源目录 (SourceDir)")
	}

	sourceDir, err := filepath.Abs(cfg.SourceDir)
	if err != nil {
		return nil, err
	}

	if len(cfg.RawExtensions) == 0 {
		cfg.RawExtensions = []string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"}
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	runner := cfg.Runner
	if runner == nil {
		runner = exiftool.DefaultRunner()
	}

	// 1. 按 Priority 分组构建 Phase
	phaseMap := make(map[int][]domain.Capability)
	for _, c := range cfg.Capabilities {
		p := c.Priority()
		phaseMap[p] = append(phaseMap[p], c)
	}

	var priorities []int
	for p := range phaseMap {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	var phases []Phase
	for _, p := range priorities {
		phases = append(phases, Phase{
			Priority:     p,
			Capabilities: phaseMap[p],
		})
	}

	name := cfg.Name
	if name == "" {
		var phaseDescs []string
		for idx, ph := range phases {
			var capNames []string
			for _, c := range ph.Capabilities {
				capNames = append(capNames, c.Name())
			}
			phaseDescs = append(phaseDescs, fmt.Sprintf("阶段%d[P%d:%s]", idx+1, ph.Priority, strings.Join(capNames, "+")))
		}
		name = fmt.Sprintf("优先级分层流水线: %s", strings.Join(phaseDescs, " ➔ "))
	}

	desc := cfg.Description
	if desc == "" {
		desc = "按插件 Priority 从高到低分阶段串行流转，同阶段内安全并发处理"
	}

	return &Orchestrator{
		name:         name,
		description:  desc,
		sourceDir:    sourceDir,
		capabilities: cfg.Capabilities,
		phases:       phases,
		workers:      cfg.Workers,
		runner:       runner,
		logDir:       cfg.LogDir,
		issueFile:    cfg.IssueFile,
		lockPath:     cfg.LockPath,
		backupDir:    cfg.BackupDir,
		discoverer:   engine.NewDiscoverer(cfg.RawExtensions),
		reporter:     engine.NewReporter(),
	}, nil
}

func (o *Orchestrator) Name() string {
	return o.name
}

func (o *Orchestrator) Description() string {
	return o.description
}

// Phases 返回当前编排器划分的执行阶段列表
func (o *Orchestrator) Phases() []Phase {
	return o.phases
}

// Stages 动态计算当前流水线包含的有序阶段列表（供 TUI 进度条与状态看板渲染）
func (o *Orchestrator) Stages() []domain.PipelineStage {
	stages := []domain.PipelineStage{domain.StageDiscover, domain.StagePrecheck}
	seen := make(map[domain.PipelineStage]bool)
	seen[domain.StageDiscover] = true
	seen[domain.StagePrecheck] = true

	for _, ph := range o.phases {
		for _, capInst := range ph.Capabilities {
			st := capInst.RequiredStage()
			if !seen[st] {
				seen[st] = true
				stages = append(stages, st)
			}
		}
	}
	stages = append(stages, domain.StageComplete)
	return stages
}

// Plan 聚合执行各插件能力的 Dry-Run 预检评估
func (o *Orchestrator) Plan(ctx context.Context) (*domain.PlanResult, error) {
	return o.PlanWithProgress(ctx, nil)
}

// PlanWithProgress 聚合执行各插件能力的 Dry-Run 预检评估，并通过 eventCh 实时反馈扫描与元数据加载进度
func (o *Orchestrator) PlanWithProgress(ctx context.Context, eventCh chan<- domain.ProgressEvent) (*domain.PlanResult, error) {
	sendEvent := func(e domain.ProgressEvent) {
		if eventCh != nil {
			select {
			case eventCh <- e:
			default:
			}
		}
	}

	sendEvent(domain.ProgressEvent{
		Stage:   domain.StageDiscover,
		Level:   domain.LevelInfo,
		Message: fmt.Sprintf("正在扫描源目录: %s ...", o.sourceDir),
	})

	// 0. 执行各阶段能力插件环境自检与初始化
	for _, ph := range o.phases {
		for _, capInst := range ph.Capabilities {
			_ = capInst.Init(ctx, nil)
		}
	}

	allGroups, err := o.discoverer.Discover(o.sourceDir)
	if err != nil {
		return nil, fmt.Errorf("扫描目录失败 (%s): %w", o.sourceDir, err)
	}

	result := &domain.PlanResult{
		TotalAssets: len(allGroups),
	}

	if len(allGroups) == 0 {
		sendEvent(domain.ProgressEvent{
			Stage:   domain.StagePrecheck,
			Level:   domain.LevelInfo,
			Message: "源目录中未发现可处理的照片资产",
		})
		return result, nil
	}

	totalAssets := len(allGroups)
	sendEvent(domain.ProgressEvent{
		Stage:        domain.StagePrecheck,
		Level:        domain.LevelInfo,
		Message:      fmt.Sprintf("已发现 %d 组摄影资产，正在并发批量提取 EXIF 拍摄元数据...", totalAssets),
		TotalItems:   totalAssets,
		CurrentIndex: 0,
	})

	allAssetContexts := make([]*domain.AssetContext, len(allGroups))
	for i, group := range allGroups {
		allAssetContexts[i] = domain.NewAssetContext(group)
	}
	for _, actx := range allAssetContexts {
		actx.Batch = allAssetContexts
	}

	// 1. 批量极速并发预读元数据 (多 Worker 并发分批读取，彻底杜绝单线程逐个启动 ExifTool)
	var missingPaths []string
	pathMap := make(map[string]*domain.AssetContext, len(allAssetContexts)*4)
	baseMap := make(map[string]*domain.AssetContext, len(allAssetContexts)*2)

	for _, actx := range allAssetContexts {
		primary := actx.Asset.PrimaryPath()
		if primary != "" {
			cleanP := filepath.Clean(primary)
			pathMap[cleanP] = actx
			pathMap[strings.ToLower(cleanP)] = actx
			if abs, err := filepath.Abs(primary); err == nil {
				pathMap[abs] = actx
				pathMap[strings.ToLower(abs)] = actx
			}
			baseName := filepath.Base(primary)
			baseMap[baseName] = actx
			baseMap[strings.ToLower(baseName)] = actx

			missingPaths = append(missingPaths, primary)
		}
	}

	if len(missingPaths) > 0 {
		runner := o.runner
		if runner == nil {
			runner = exiftool.DefaultRunner()
		}
		metaMap, _ := exiftool.ReadBatchMetadataMapConcurrent(runner, missingPaths, o.workers, func(processed, total int) {
			pct := 0.0
			if total > 0 {
				pct = float64(processed) / float64(total) * 100
			}
			sendEvent(domain.ProgressEvent{
				Stage:        domain.StagePrecheck,
				Level:        domain.LevelInfo,
				Message:      fmt.Sprintf("正在并发批量装载 EXIF 元数据 (%d/%d 组 - %.0f%%)...", processed, total, pct),
				TotalItems:   total,
				CurrentIndex: processed,
			})
		})

		for p, meta := range metaMap {
			cleanP := filepath.Clean(p)
			actx, ok := pathMap[cleanP]
			if !ok {
				actx, ok = pathMap[strings.ToLower(cleanP)]
			}
			if !ok {
				actx, ok = baseMap[filepath.Base(p)]
			}
			if !ok {
				actx, ok = baseMap[strings.ToLower(filepath.Base(p))]
			}
			if ok && actx != nil {
				actx.UpdateMetadata(meta)
				if meta.GPSPosition != "" {
					if lat, lon, err := exiftool.ParseCoordinates(meta.GPSPosition); err == nil {
						actx.SetGPS(lat, lon)
					}
				}
			}
		}
	}

	// 强制将所有资产标记为已预装载元数据，彻底杜绝任何单文件在预检中再次尝试 exiftool 读盘！
	for _, actx := range allAssetContexts {
		actx.MetadataLoaded = true
	}

	sendEvent(domain.ProgressEvent{
		Stage:        domain.StagePrecheck,
		Level:        domain.LevelInfo,
		Message:      fmt.Sprintf("元数据装载完成，正在多协程并发评估各阶段动作 (共 %d 组)...", totalAssets),
		TotalItems:   totalAssets,
		CurrentIndex: 0,
	})

	// 2. 内存秒级多协程并发评估各插件 PlanPrecheck (高频平滑打点)
	var evalProcessed atomic.Int64
	var lastReportTime time.Time
	var reportMu sync.Mutex

	type evalResult struct {
		item  domain.PlanItem
		ready bool
		warn  bool
	}

	pool := engine.NewWorkerPool[*domain.AssetContext, evalResult](o.workers)
	evalResults := pool.Execute(ctx, allAssetContexts, func(c context.Context, actx *domain.AssetContext) evalResult {
		var actions []string
		var warnings []string
		hasExecutableStage := false
		hasBlockingWarning := false

		// 按 Phase 顺序预检
		for phaseIdx, ph := range o.phases {
			for _, capInst := range ph.Capabilities {
				plan := capInst.PlanPrecheck(c, actx)
				if plan.CanProcess {
					hasExecutableStage = true
				}
				if plan.ActionDesc != "" {
					actions = append(actions, fmt.Sprintf("阶段%d[P%d] %s: %s", phaseIdx+1, ph.Priority, capInst.Name(), plan.ActionDesc))
				}
				if plan.Warning != "" {
					hasBlockingWarning = true
					warnings = append(warnings, fmt.Sprintf("[%s] %s", capInst.Name(), plan.Warning))
				}
			}
		}

		item := domain.PlanItem{
			Asset:   actx.Asset,
			Action:  strings.Join(actions, " ➔ "),
			Warning: strings.Join(warnings, "；"),
		}

		isReady := !hasBlockingWarning && (hasExecutableStage || len(actions) > 0)
		hasWarn := len(warnings) > 0

		done := int(evalProcessed.Add(1))

		// 高频平滑汇报进度 (每 50 组或每 50ms 汇报一次，彻底消除停滞感)
		reportMu.Lock()
		now := time.Now()
		if done%50 == 0 || done == totalAssets || now.Sub(lastReportTime) > 50*time.Millisecond {
			lastReportTime = now
			sendEvent(domain.ProgressEvent{
				Stage:        domain.StagePrecheck,
				Level:        domain.LevelInfo,
				Message:      fmt.Sprintf("正在评估执行计划 (%d/%d 组 - %.0f%%)...", done, totalAssets, float64(done)/float64(totalAssets)*100),
				TotalItems:   totalAssets,
				CurrentIndex: done,
			})
		}
		reportMu.Unlock()

		return evalResult{
			item:  item,
			ready: isReady,
			warn:  hasWarn,
		}
	})

	for _, res := range evalResults {
		if res.ready {
			result.ReadyCount++
		} else {
			result.PendingCount++
			if res.warn {
				result.WarningsCount++
			}
		}
		result.Items = append(result.Items, res.item)
	}

	sendEvent(domain.ProgressEvent{
		Stage:        domain.StagePrecheck,
		Level:        domain.LevelSuccess,
		Message:      fmt.Sprintf("✅ 预检评估完成！就绪: %d 组，待补/跳过: %d 组，异常: %d 组", result.ReadyCount, result.PendingCount, result.WarningsCount),
		TotalItems:   totalAssets,
		CurrentIndex: totalAssets,
	})

	return result, nil
}

// Execute 按照 Phase 优先级顺序串行推进，每个 Phase 内部对全量资产并发执行，并将实时中文日志流写入 Logs/
func (o *Orchestrator) Execute(ctx context.Context, eventCh chan<- domain.ProgressEvent) (*domain.TaskSummary, []domain.Issue, error) {
	startTime := time.Now()

	// 初始化日志文件句柄 (同时写入时间戳日志与 photools_latest.log)
	var logFiles []*os.File
	var logFileMu sync.Mutex

	if o.logDir != "" {
		_ = os.MkdirAll(o.logDir, 0o755)
		tsLogName := fmt.Sprintf("photools_%s.log", startTime.Format("20060102_150405"))
		tsLogPath := filepath.Join(o.logDir, tsLogName)
		latestLogPath := filepath.Join(o.logDir, "photools_latest.log")

		if f1, err := os.OpenFile(tsLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
			logFiles = append(logFiles, f1)
		}
		if f2, err := os.OpenFile(latestLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
			logFiles = append(logFiles, f2)
		}

		// 写入执行头部信息
		headerText := fmt.Sprintf("================================================================================\n"+
			"📷 photools 流水线执行日志 (Pipeline Execution Log)\n"+
			"- 启动时间: %s\n"+
			"- 任务名称: %s\n"+
			"- 扫描源目录: %s\n"+
			"- 执行阶段数: %d 个阶段 (Worker 并发: %d)\n"+
			"================================================================================\n\n",
			startTime.Format("2006-01-02 15:04:05"), o.Name(), o.sourceDir, len(o.phases), o.workers)

		for _, f := range logFiles {
			_, _ = f.WriteString(headerText)
			_ = f.Sync()
		}
	}

	defer func() {
		logFileMu.Lock()
		defer logFileMu.Unlock()
		for _, f := range logFiles {
			_ = f.Close()
		}
	}()

	sendEvent := func(e domain.ProgressEvent) {
		if eventCh != nil {
			select {
			case eventCh <- e:
			default:
			}
		}

		// 实时中文日志流落盘
		if len(logFiles) > 0 && e.Message != "" {
			nowStr := time.Now().Format("15:04:05.000")
			lvlStr := "INFO"
			switch e.Level {
			case domain.LevelWarn:
				lvlStr = "WARN"
			case domain.LevelError:
				lvlStr = "ERROR"
			case domain.LevelSuccess:
				lvlStr = "SUCC"
			}
			stageStr := string(e.Stage)
			if stageStr == "" {
				stageStr = "执行阶段"
			}

			line := fmt.Sprintf("[%s] [%-5s] [%s] %s\n", nowStr, lvlStr, stageStr, e.Message)

			logFileMu.Lock()
			for _, f := range logFiles {
				_, _ = f.WriteString(line)
				_ = f.Sync()
			}
			logFileMu.Unlock()
		}
	}

	if o.lockPath != "" {
		if err := os.Mkdir(o.lockPath, 0o755); err != nil {
			if os.IsExist(err) {
				sendEvent(domain.NewInfoEvent(domain.StagePrecheck, "已有任务正在处理中，本次跳过。"))
				return &domain.TaskSummary{}, nil, nil
			}
			return nil, nil, fmt.Errorf("创建运行锁失败: %w", err)
		}
		defer os.Remove(o.lockPath)
	}

	// 0. 执行各阶段能力插件环境自检与初始化
	for _, ph := range o.phases {
		for _, capInst := range ph.Capabilities {
			_ = capInst.Init(ctx, func(report domain.PluginInitReport) {
				msg := fmt.Sprintf("[%s] %s: %s", capInst.Name(), report.Stage, report.Message)
				if report.Percent >= 0 && report.Percent < 1.0 {
					msg = fmt.Sprintf("[%s] %s (%.0f%%): %s", capInst.Name(), report.Stage, report.Percent*100, report.Message)
				}
				sendEvent(domain.ProgressEvent{
					Stage:   capInst.RequiredStage(),
					Level:   domain.LevelInfo,
					Message: msg,
				})
			})
		}
	}

	sendEvent(domain.NewInfoEvent(domain.StageDiscover, fmt.Sprintf("开始扫描目录 %s ...", o.sourceDir)))

	allGroups, err := o.discoverer.Discover(o.sourceDir)
	if err != nil {
		return nil, nil, err
	}

	summary := &domain.TaskSummary{
		TotalAssets: len(allGroups),
	}

	if len(allGroups) == 0 {
		sendEvent(domain.NewInfoEvent(domain.StageComplete, "目录中没有发现可处理的照片资产。"))
		return summary, nil, nil
	}

	// 若开启了测试备份模式，在正式处理前对待处理原始资产做全量快照
	if o.backupDir != "" && len(allGroups) > 0 {
		sendEvent(domain.NewInfoEvent(domain.StageDiscover, fmt.Sprintf("📦 [测试备份模式] 正在将 %d 组待处理原始照片快照备份至 %s ...", len(allGroups), o.backupDir)))
		copiedCount, err := engine.BackupAssetGroups(allGroups, o.sourceDir, o.backupDir)
		if err != nil {
			return nil, nil, fmt.Errorf("测试备份失败: %w", err)
		}
		sendEvent(domain.NewInfoEvent(domain.StageDiscover, fmt.Sprintf("✅ [测试备份模式] 已完成 %d 个原始文件快照备份 (%s)", copiedCount, o.backupDir)))
	}

	// 初始化共享上下文表（每个拍摄单元唯一对应一个 AssetContext）
	type assetState struct {
		actx    *domain.AssetContext
		failed  bool
		waiting bool
		issue   *domain.Issue
	}

	allAssetContexts := make([]*domain.AssetContext, len(allGroups))
	for i, g := range allGroups {
		allAssetContexts[i] = domain.NewAssetContext(g)
	}
	for _, actx := range allAssetContexts {
		actx.Batch = allAssetContexts
	}

	states := make([]*assetState, len(allGroups))
	for i, actx := range allAssetContexts {
		states[i] = &assetState{
			actx: actx,
		}
	}

	totalAssets := len(allGroups)

	// 分阶段串行推进各个 Phase
	for phaseIdx, ph := range o.phases {
		// 检查 context 是否已被取消
		if err := ctx.Err(); err != nil {
			sendEvent(domain.ProgressEvent{
				Stage:   domain.StageComplete,
				Level:   domain.LevelWarn,
				Message: "任务已被用户中断取消",
			})
			return summary, nil, err
		}

		var capNames []string
		for _, c := range ph.Capabilities {
			capNames = append(capNames, c.Name())
		}
		phaseTitle := fmt.Sprintf("【阶段 %d/%d · 优先级 %d】%s", phaseIdx+1, len(o.phases), ph.Priority, strings.Join(capNames, " & "))
		sendEvent(domain.NewInfoEvent(domain.StagePrecheck, fmt.Sprintf("🚦 进入 %s (共 %d 个资产并发处理)...", phaseTitle, totalAssets)))

		var phaseProcessed atomic.Int64
		var phaseMu sync.Mutex

		pool := engine.NewWorkerPool[*assetState, struct{}](o.workers)
		pool.Execute(ctx, states, func(c context.Context, st *assetState) struct{} {
			// 若前序阶段已失败或等待，输出透明跳过日志并记录级联阻断
			if st.failed || st.waiting {
				idx := int(phaseProcessed.Add(1))
				capName := ph.Capabilities[0].Name()
				var cause string
				phaseMu.Lock()
				if st.issue != nil {
					cause = st.issue.Reason
					st.issue.BlockedStages = append(st.issue.BlockedStages, capName)
					if st.issue.CurrentStatus == "" {
						st.issue.CurrentStatus = fmt.Sprintf("安全保留在源目录 (%s)", o.sourceDir)
					}
				}
				phaseMu.Unlock()

				sendEvent(domain.ProgressEvent{
					Stage:        ph.Capabilities[0].RequiredStage(),
					Level:        domain.LevelWarn,
					Message:      fmt.Sprintf("[%s] ⏭️ 前序阶段未通过，安全熔断跳过当前阶段：%s", capName, st.actx.Asset.DisplayName()),
					Asset:        &st.actx.Asset,
					CurrentIndex: idx,
					TotalItems:   totalAssets,
				})
				_ = cause
				return struct{}{}
			}

			// 在当前 Phase 内部执行该 Priority 包含的所有 Capabilities
			for _, capInst := range ph.Capabilities {
				// 1. 预检
				plan := capInst.PlanPrecheck(c, st.actx)
				if !plan.CanProcess {
					// 若 Warning 为空，说明属于良性跳过（如已有 GPS 坐标无需插值，或开启了无 GPS 降级允许跳过地名写入）
					if plan.Warning == "" {
						continue
					}

					idx := int(phaseProcessed.Add(1))
					issue := &domain.Issue{
						Kind:          domain.IssueKindMissingPair,
						Reason:        plan.Warning,
						Suggestion:    "请检查该资产的前置依赖条件后重试。",
						Asset:         st.actx.Asset,
						FailedStage:   capInst.Name(),
						CurrentStatus: fmt.Sprintf("未满足条件，保留在源目录 (%s)", o.sourceDir),
					}
					phaseMu.Lock()
					st.waiting = true
					st.issue = issue
					phaseMu.Unlock()

					sendEvent(domain.ProgressEvent{
						Stage:        capInst.RequiredStage(),
						Level:        domain.LevelWarn,
						Message:      fmt.Sprintf("[%s] 未满足执行条件，暂停后续阶段：%s (%s)", capInst.Name(), st.actx.Asset.DisplayName(), plan.Warning),
						Asset:        &st.actx.Asset,
						Issue:        issue,
						CurrentIndex: idx,
						TotalItems:   totalAssets,
					})
					return struct{}{}
				}

				// 2. 执行能力逻辑
				if err := capInst.ExecuteProcess(c, st.actx, sendEvent); err != nil {
					idx := int(phaseProcessed.Add(1))

					// 若当前是 GPX 匹配且属于时间未命中轨迹，但后续流水线启用了 GPS 智能插值能力，则平滑转交下游插值插件
					if capInst.ID() == domain.CapGPXMatching && o.hasCapability(domain.CapGPSInterpolate) &&
						(strings.Contains(err.Error(), "未检测到 GPSPosition") || strings.Contains(err.Error(), "未命中轨迹") || strings.Contains(err.Error(), "未落在轨迹")) {
						sendEvent(domain.ProgressEvent{
							Stage:        capInst.RequiredStage(),
							Level:        domain.LevelWarn,
							Message:      fmt.Sprintf("[%s] ⚠️ 拍摄时间未命中 GPX 轨迹，已流转至阶段 2 智能推算：%s", capInst.Name(), st.actx.Asset.DisplayName()),
							Asset:        &st.actx.Asset,
							CurrentIndex: idx,
							TotalItems:   totalAssets,
						})
						return struct{}{}
					}

					suggestion := "请检查底层工具、文件权限或补充必要输入。"
					if strings.Contains(err.Error(), "GPS") || strings.Contains(err.Error(), "轨迹") {
						suggestion = "拍摄时间未命中当前 GPX 轨迹。若需直接归档，可使用 photools pipeline --gpx=false 模式跳过 GPS 修正直接归档，或补全该时间段 GPX 轨迹。"
					} else if strings.Contains(err.Error(), "目标文件已存在") || strings.Contains(err.Error(), "冲突") {
						suggestion = "归档目标路径已存在同名文件，请检查是否已归档过或修改命名模板以避免重名。"
					}

					issue := &domain.Issue{
						Kind:          domain.IssueKindFailure,
						Reason:        fmt.Sprintf("[%s] 执行失败: %v", capInst.Name(), err),
						Suggestion:    suggestion,
						Asset:         st.actx.Asset,
						FailedStage:   capInst.Name(),
						CurrentStatus: fmt.Sprintf("处理中断，源文件安全保留在 (%s)", o.sourceDir),
					}
					phaseMu.Lock()
					st.failed = true
					st.issue = issue
					phaseMu.Unlock()

					sendEvent(domain.ProgressEvent{
						Stage:        capInst.RequiredStage(),
						Level:        domain.LevelError,
						Message:      fmt.Sprintf("[%s] 处理失败：%s (%v)", capInst.Name(), st.actx.Asset.DisplayName(), err),
						Asset:        &st.actx.Asset,
						Issue:        issue,
						CurrentIndex: idx,
						TotalItems:   totalAssets,
					})
					return struct{}{}
				}
			}

			idx := int(phaseProcessed.Add(1))
			sendEvent(domain.ProgressEvent{
				Stage:        ph.Capabilities[len(ph.Capabilities)-1].RequiredStage(),
				Level:        domain.LevelSuccess,
				Message:      fmt.Sprintf("[%s] 完成: %s", phaseTitle, st.actx.Asset.DisplayName()),
				Asset:        &st.actx.Asset,
				CurrentIndex: idx,
				TotalItems:   totalAssets,
			})
			return struct{}{}
		})

		sendEvent(domain.NewInfoEvent(domain.StagePrecheck, fmt.Sprintf("✅ %s 全部执行完毕，通过阶段同步屏障！", phaseTitle)))
	}

	// 最终汇总
	var issues []domain.Issue
	for _, st := range states {
		if st.failed {
			summary.Failed++
			if st.issue != nil {
				issues = append(issues, *st.issue)
			}
		} else if st.waiting {
			summary.Pending++
			if st.issue != nil {
				issues = append(issues, *st.issue)
			}
		} else {
			summary.Success++
		}
	}

	if o.issueFile != "" && len(issues) > 0 {
		_ = o.reporter.WriteMarkdownReport(o.issueFile, o.Name(), issues)
	}

	totalDur := time.Since(startTime)
	summaryMsg := fmt.Sprintf("全部 %d 个阶段流水线处理完成：共 %d 个资产，成功 %d，待补 %d，失败 %d (总耗时: %v)",
		len(o.phases), summary.TotalAssets, summary.Success, summary.Pending, summary.Failed, totalDur)

	sendEvent(domain.ProgressEvent{
		Stage:   domain.StageComplete,
		Level:   domain.LevelSuccess,
		Message: summaryMsg,
	})

	// 写入日志文件结算尾部
	if len(logFiles) > 0 {
		var footer strings.Builder
		footer.WriteString("\n================================================================================\n")
		footer.WriteString("📊 流水线执行结算概览 (Execution Summary)\n")
		footer.WriteString(fmt.Sprintf("- 结束时间: %s (总耗时: %v)\n", time.Now().Format("2006-01-02 15:04:05"), totalDur))
		footer.WriteString(fmt.Sprintf("- 资产总数: %d 组\n", summary.TotalAssets))
		footer.WriteString(fmt.Sprintf("- 成功完成: %d 组\n", summary.Success))
		footer.WriteString(fmt.Sprintf("- 待补保留: %d 组\n", summary.Pending))
		footer.WriteString(fmt.Sprintf("- 失败异常: %d 组\n", summary.Failed))

		if len(issues) > 0 {
			footer.WriteString("\n⚠️ 异常与待补资产清单:\n")
			for i, iss := range issues {
				footer.WriteString(fmt.Sprintf("  [%d] %s (阶段: %s) -> 原因: %s\n      建议: %s\n",
					i+1, iss.Asset.DisplayName(), iss.FailedStage, iss.Reason, iss.Suggestion))
			}
		} else {
			footer.WriteString("\n🎉 本次流水线全部资产处理成功，无任何异常！\n")
		}
		footer.WriteString("================================================================================\n")

		footerStr := footer.String()
		logFileMu.Lock()
		for _, f := range logFiles {
			_, _ = f.WriteString(footerStr)
			_ = f.Sync()
		}
		logFileMu.Unlock()
	}

	return summary, issues, nil
}

func (o *Orchestrator) hasCapability(id domain.CapabilityID) bool {
	for _, ph := range o.phases {
		for _, c := range ph.Capabilities {
			if c.ID() == id {
				return true
			}
		}
	}
	return false
}
