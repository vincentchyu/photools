package geotag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
)

// Config 定义 GPS 修正任务的运行配置
type Config struct {
	BaseDir       string
	ProcessedDir  string
	Geosync       string
	RawExtensions []string
	Workers       int
	Runner        exiftool.CommandRunner
}

// Task 实现 domain.Task 接口
type Task struct {
	baseDir       string
	inboxDir      string
	gpxDir        string
	processedDir  string
	logDir        string
	logFile       string
	issueFile     string
	lockPath      string
	geosync       string
	rawExtensions []string
	workers       int
	runner        exiftool.CommandRunner

	discoverer *engine.Discoverer
	archiver   *engine.Archiver
	reporter   *engine.Reporter
}

// NewTask 实例化 GeotagTask
func NewTask(cfg Config) (*Task, error) {
	if cfg.BaseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.BaseDir = filepath.Join(home, "Pictures", "GPS")
	}

	baseDir, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return nil, err
	}

	processedDir := cfg.ProcessedDir
	if processedDir == "" {
		processedDir = filepath.Join(baseDir, "Processed", "geotag")
	} else {
		if abs, err := filepath.Abs(processedDir); err == nil {
			processedDir = abs
		}
	}

	if len(cfg.RawExtensions) == 0 {
		cfg.RawExtensions = []string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"}
	}
	if cfg.Geosync == "" {
		cfg.Geosync = "0"
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.Runner == nil {
		cfg.Runner = exiftool.ExecRunner{}
	}

	return &Task{
		baseDir:       baseDir,
		inboxDir:      filepath.Join(baseDir, "Inbox"),
		gpxDir:        filepath.Join(baseDir, "GPX"),
		processedDir:  processedDir,
		logDir:        filepath.Join(baseDir, "Logs"),
		logFile:       filepath.Join(baseDir, "Logs", "geotag.log"),
		issueFile:     filepath.Join(baseDir, "Logs", "inbox_pending_report_latest.md"),
		lockPath:      filepath.Join(baseDir, ".geotag.lock"),
		geosync:       cfg.Geosync,
		rawExtensions: cfg.RawExtensions,
		workers:       cfg.Workers,
		runner:        cfg.Runner,
		discoverer:    engine.NewDiscoverer(cfg.RawExtensions),
		archiver:      engine.NewArchiver(),
		reporter:      engine.NewReporter(),
	}, nil
}

func (t *Task) Name() string {
	return "GPS 轨迹匹配与归档 (Geotag)"
}

func (t *Task) Description() string {
	return "根据 GPX 轨迹为 Inbox 中的 RAW 照片批量修正写入 GPS，同步 JPG/XMP 并按拍摄日期归档"
}

func (t *Task) Stages() []domain.PipelineStage {
	return []domain.PipelineStage{
		domain.StageDiscover,
		domain.StagePrecheck,
		domain.StageGeotag,
		domain.StageSync,
		domain.StageArchive,
	}
}

// Plan 预检
func (t *Task) Plan(ctx context.Context) (*domain.PlanResult, error) {
	gpxFiles, err := ListGPXFiles(t.gpxDir)
	if err != nil {
		return nil, fmt.Errorf("读取 GPX 目录失败: %w", err)
	}

	allGroups, err := t.discoverer.Discover(t.inboxDir)
	if err != nil {
		return nil, fmt.Errorf("扫描 Inbox 失败: %w", err)
	}

	result := &domain.PlanResult{
		TotalAssets: len(allGroups),
	}

	for _, group := range allGroups {
		item := domain.PlanItem{
			Asset: group,
		}

		if !group.HasRaw() {
			item.Action = "跳过（独立 JPG，缺少主 RAW）"
			item.Warning = "缺少 RAW 权威源"
			result.WarningsCount++
			result.Items = append(result.Items, item)
			continue
		}

		if !group.HasJPG() {
			item.Action = "保留在 Inbox（缺少配对 JPG）"
			item.Warning = "等待配对 JPG"
			result.PendingCount++
			result.Items = append(result.Items, item)
			continue
		}

		if len(gpxFiles) == 0 {
			item.Action = "无法处理（未找到 GPX 轨迹）"
			item.Warning = "缺少 GPX"
			result.PendingCount++
			result.Items = append(result.Items, item)
			continue
		}

		item.Action = "写入 GPS 并在成功后按拍摄日期归档"
		item.WillWriteGPS = true
		result.ReadyCount++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// Execute 真实执行
func (t *Task) Execute(ctx context.Context, eventCh chan<- domain.ProgressEvent) (*domain.TaskSummary, []domain.Issue, error) {
	sendEvent := func(e domain.ProgressEvent) {
		if eventCh != nil {
			select {
			case eventCh <- e:
			default:
			}
		}
	}

	if err := os.MkdirAll(t.inboxDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(t.gpxDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(t.processedDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(t.logDir, 0o755); err != nil {
		return nil, nil, err
	}

	if err := os.Mkdir(t.lockPath, 0o755); err != nil {
		if os.IsExist(err) {
			sendEvent(domain.NewInfoEvent(domain.StagePrecheck, "已有任务正在处理中，本次跳过。"))
			return &domain.TaskSummary{}, nil, nil
		}
		return nil, nil, fmt.Errorf("创建运行锁失败: %w", err)
	}
	defer os.Remove(t.lockPath)

	sendEvent(domain.NewInfoEvent(domain.StageDiscover, "开始扫描 GPX 轨迹与 Inbox 待处理照片..."))

	gpxFiles, err := ListGPXFiles(t.gpxDir)
	if err != nil {
		return nil, nil, err
	}
	if len(gpxFiles) == 0 {
		sendEvent(domain.ProgressEvent{
			Stage:   domain.StagePrecheck,
			Level:   domain.LevelWarn,
			Message: "未在 GPX 目录中找到任何轨迹文件。",
		})
		return nil, nil, errors.New("no gpx files found")
	}

	allGroups, err := t.discoverer.Discover(t.inboxDir)
	if err != nil {
		return nil, nil, err
	}

	var rawAssets []domain.AssetGroup
	var jpgOnlyCount int

	for _, group := range allGroups {
		if group.HasRaw() {
			rawAssets = append(rawAssets, group)
		} else if group.HasJPG() {
			jpgOnlyCount++
			sendEvent(domain.ProgressEvent{
				Stage:   domain.StageDiscover,
				Level:   domain.LevelWarn,
				Message: fmt.Sprintf("发现未配对的独立 JPG，跳过：%s", group.DisplayName()),
				Asset:   &group,
			})
		}
	}

	summary := &domain.TaskSummary{
		TotalAssets: len(rawAssets),
		Skipped:     jpgOnlyCount,
	}

	if len(rawAssets) == 0 {
		sendEvent(domain.NewInfoEvent(domain.StageComplete, "Inbox 中没有发现可处理的 RAW 照片。"))
		return summary, nil, nil
	}

	type assetResult struct {
		status  string // "success", "waiting", "failed"
		issue   *domain.Issue
		message string
	}

	var processedCount atomic.Int64
	totalAssets := len(rawAssets)

	pool := engine.NewWorkerPool[domain.AssetGroup, assetResult](t.workers)

	results := pool.Execute(ctx, rawAssets, func(c context.Context, asset domain.AssetGroup) assetResult {
		return t.processSingleAsset(c, asset, gpxFiles, sendEvent, &processedCount, totalAssets)
	})

	var issues []domain.Issue
	for _, res := range results {
		switch res.status {
		case "success":
			summary.Success++
		case "waiting":
			summary.Pending++
			if res.issue != nil {
				issues = append(issues, *res.issue)
			}
		case "failed":
			summary.Failed++
			if res.issue != nil {
				issues = append(issues, *res.issue)
			}
		}
	}

	_ = t.reporter.WriteMarkdownReport(t.issueFile, t.Name(), issues)

	sendEvent(domain.ProgressEvent{
		Stage:   domain.StageComplete,
		Level:   domain.LevelSuccess,
		Message: fmt.Sprintf("处理完成：共 %d 个 RAW，成功 %d，待补 %d，失败 %d，跳过独立 JPG %d", summary.TotalAssets, summary.Success, summary.Pending, summary.Failed, summary.Skipped),
	})

	return summary, issues, nil
}

func (t *Task) processSingleAsset(
	ctx context.Context,
	asset domain.AssetGroup,
	gpxFiles []string,
	sendEvent func(domain.ProgressEvent),
	processedCount *atomic.Int64,
	totalAssets int,
) struct {
	status  string
	issue   *domain.Issue
	message string
} {
	if !asset.HasJPG() {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:       domain.IssueKindMissingPair,
			Reason:     "缺少同 basename 的 JPG，当前资产组继续保留在 Inbox。",
			Suggestion: "请补齐对应的 JPG 文件后重新运行。",
			Asset:      asset,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StagePrecheck,
			Level:        domain.LevelWarn,
			Message:      fmt.Sprintf("等待配对 JPG：%s", asset.DisplayName()),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "waiting", issue: issue}
	}

	// 1. 读取 RAW 元数据
	rawMeta, err := exiftool.ReadMetadata(t.runner, asset.RawPath)
	if err != nil {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:       domain.IssueKindFailure,
			Reason:     fmt.Sprintf("读取 RAW 元数据失败: %v", err),
			Suggestion: "请确认 RAW 文件可正常读取，然后重新运行。",
			Asset:      asset,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageGeotag,
			Level:        domain.LevelError,
			Message:      fmt.Sprintf("读取 RAW 元数据失败: %v", err),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "failed", issue: issue}
	}

	// 2. 写入 RAW GPS
	output, err := exiftool.WriteGeotag(t.runner, asset.RawPath, gpxFiles, t.geosync)
	if err != nil {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:               domain.IssueKindTrackGap,
			Reason:             exiftool.ClassifyFailure(output, err),
			Suggestion:         "请检查 GPX 轨迹时间范围是否覆盖照片拍摄时间，或确认是否需要调整 -geosync 偏移值。",
			Asset:              asset,
			PhotoTime:          rawMeta.DateTimeOriginal,
			PhotoOffset:        rawMeta.OffsetTimeOriginal,
			ReferencedGPXFiles: gpxFiles,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageGeotag,
			Level:        domain.LevelWarn,
			Message:      fmt.Sprintf("RAW 写入 GPS 失败，保留在 Inbox：%s", issue.Reason),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "waiting", issue: issue}
	}

	// 3. 二次校验 RAW GPS
	updatedMeta, err := exiftool.ReadMetadata(t.runner, asset.RawPath)
	if err != nil || updatedMeta.GPSPosition == "" {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:               domain.IssueKindTrackGap,
			Reason:             exiftool.ClassifyFailure(output, err),
			Suggestion:         "请检查 GPX 轨迹时间范围是否覆盖照片拍摄时间，或调整时间偏移值。",
			Asset:              asset,
			PhotoTime:          rawMeta.DateTimeOriginal,
			PhotoOffset:        rawMeta.OffsetTimeOriginal,
			ReferencedGPXFiles: gpxFiles,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageGeotag,
			Level:        domain.LevelWarn,
			Message:      fmt.Sprintf("二次校验未检测到 GPSPosition，保留在 Inbox：%s", asset.DisplayName()),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "waiting", issue: issue}
	}

	sendEvent(domain.ProgressEvent{
		Stage:   domain.StageGeotag,
		Level:   domain.LevelSuccess,
		Message: fmt.Sprintf("RAW GPS 写入成功：%s (%s)", asset.DisplayName(), updatedMeta.GPSPosition),
		Asset:   &asset,
	})

	// 4. 同步 GPS 到同名 JPG
	if err := exiftool.SyncGPS(t.runner, asset.RawPath, asset.JPGPath); err != nil {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:               domain.IssueKindFailure,
			Reason:             fmt.Sprintf("同步 GPS 到 JPG 失败: %v", err),
			Suggestion:         "请检查 JPG 文件状态后重新运行。",
			Asset:              asset,
			PhotoTime:          updatedMeta.DateTimeOriginal,
			PhotoOffset:        updatedMeta.OffsetTimeOriginal,
			ReferencedGPXFiles: gpxFiles,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageSync,
			Level:        domain.LevelError,
			Message:      fmt.Sprintf("同步 GPS 到 JPG 失败：%s", asset.DisplayName()),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "failed", issue: issue}
	}

	// 5. 若存在 XMP，同步 GPS 到 XMP
	if asset.XMPPath != "" {
		if err := exiftool.SyncXMPGPS(t.runner, asset.RawPath, asset.XMPPath); err != nil {
			idx := int(processedCount.Add(1))
			issue := &domain.Issue{
				Kind:               domain.IssueKindFailure,
				Reason:             fmt.Sprintf("同步 GPS 到 XMP 失败: %v", err),
				Suggestion:         "请检查 XMP sidecar 内容后重新运行。",
				Asset:              asset,
				PhotoTime:          updatedMeta.DateTimeOriginal,
				PhotoOffset:        updatedMeta.OffsetTimeOriginal,
				ReferencedGPXFiles: gpxFiles,
			}
			sendEvent(domain.ProgressEvent{
				Stage:        domain.StageSync,
				Level:        domain.LevelError,
				Message:      fmt.Sprintf("同步 GPS 到 XMP 失败：%s", asset.DisplayName()),
				Asset:        &asset,
				Issue:        issue,
				CurrentIndex: idx,
				TotalItems:   totalAssets,
			})
			return struct {
				status  string
				issue   *domain.Issue
				message string
			}{status: "failed", issue: issue}
		}
	}

	// 6. 归档与规范化重命名
	targetDir, err := t.archiver.BuildArchiveDir(t.processedDir, updatedMeta.DateTimeOriginal)
	if err != nil {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:       domain.IssueKindFailure,
			Reason:     fmt.Sprintf("无法确定归档目录: %v", err),
			Suggestion: "请检查拍摄日期元数据。",
			Asset:      asset,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageArchive,
			Level:        domain.LevelError,
			Message:      fmt.Sprintf("无法确定归档目录: %v", err),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "failed", issue: issue}
	}

	newBase := t.archiver.CalculateNormalizedName(asset.BaseName, updatedMeta.DateTimeOriginal)
	if err := t.archiver.MoveFilesWithRename(asset.AllFiles(), targetDir, newBase); err != nil {
		idx := int(processedCount.Add(1))
		issue := &domain.Issue{
			Kind:       domain.IssueKindConflict,
			Reason:     fmt.Sprintf("移动/归档失败: %v", err),
			Suggestion: "请检查目标目录是否存在同名冲突文件或权限问题。",
			Asset:      asset,
		}
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageArchive,
			Level:        domain.LevelError,
			Message:      fmt.Sprintf("移动/归档失败: %v", err),
			Asset:        &asset,
			Issue:        issue,
			CurrentIndex: idx,
			TotalItems:   totalAssets,
		})
		return struct {
			status  string
			issue   *domain.Issue
			message string
		}{status: "failed", issue: issue}
	}

	idx := int(processedCount.Add(1))
	sendEvent(domain.ProgressEvent{
		Stage:        domain.StageArchive,
		Level:        domain.LevelSuccess,
		Message:      fmt.Sprintf("已归档到 %s/ (%s)", filepath.Base(targetDir), newBase),
		Asset:        &asset,
		CurrentIndex: idx,
		TotalItems:   totalAssets,
	})

	return struct {
		status  string
		issue   *domain.Issue
		message string
	}{status: "success"}
}
