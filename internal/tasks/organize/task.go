package organize

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
)

// Config 归档整理任务配置
type Config struct {
	SourceDir     string
	TargetDir     string
	RawExtensions []string
	Runner        exiftool.CommandRunner
}

// Task 按拍摄日期整理并规范化重命名的任务实现
type Task struct {
	sourceDir     string
	targetDir     string
	rawExtensions []string
	runner        exiftool.CommandRunner

	discoverer *engine.Discoverer
	archiver   *engine.Archiver
	reporter   *engine.Reporter
}

// NewTask 实例化 OrganizeTask
func NewTask(cfg Config) (*Task, error) {
	if cfg.SourceDir == "" || cfg.TargetDir == "" {
		return nil, errors.New("源目录和目标目录不能为空")
	}

	src, err := filepath.Abs(cfg.SourceDir)
	if err != nil {
		return nil, err
	}
	tgt, err := filepath.Abs(cfg.TargetDir)
	if err != nil {
		return nil, err
	}

	if len(cfg.RawExtensions) == 0 {
		cfg.RawExtensions = []string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"}
	}
	if cfg.Runner == nil {
		cfg.Runner = exiftool.ExecRunner{}
	}

	return &Task{
		sourceDir:     src,
		targetDir:     tgt,
		rawExtensions: cfg.RawExtensions,
		runner:        cfg.Runner,
		discoverer:    engine.NewDiscoverer(cfg.RawExtensions),
		archiver:      engine.NewArchiver(),
		reporter:      engine.NewReporter(),
	}, nil
}

func (t *Task) Name() string {
	return "按拍摄日期归档整理 (Organize by Date)"
}

func (t *Task) Description() string {
	return "根据照片拍摄日期，将同组照片及附属文件规范化重命名并归档到目标目录 (YYYY/MMDD/)"
}

func (t *Task) Stages() []domain.PipelineStage {
	return []domain.PipelineStage{
		domain.StageDiscover,
		domain.StagePrecheck,
		domain.StageArchive,
		domain.StageComplete,
	}
}

func (t *Task) findDateAnchor(asset domain.AssetGroup) (domain.Metadata, error) {
	var meta domain.Metadata
	var err error

	if asset.RawPath != "" {
		if meta, err = exiftool.ReadMetadata(t.runner, asset.RawPath); err == nil && meta.DateTimeOriginal != "" {
			return meta, nil
		}
	}
	if asset.JPGPath != "" {
		if meta, err = exiftool.ReadMetadata(t.runner, asset.JPGPath); err == nil && meta.DateTimeOriginal != "" {
			return meta, nil
		}
	}
	for _, path := range asset.CompanionPaths {
		if m, err := exiftool.ReadMetadata(t.runner, path); err == nil && m.DateTimeOriginal != "" {
			return m, nil
		}
	}

	return domain.Metadata{}, errors.New("未能从资产组中读取到 DateTimeOriginal 拍摄日期")
}

// Plan 预检
func (t *Task) Plan(ctx context.Context) (*domain.PlanResult, error) {
	assets, err := t.discoverer.Discover(t.sourceDir)
	if err != nil {
		return nil, fmt.Errorf("扫描源目录失败: %w", err)
	}

	result := &domain.PlanResult{
		TotalAssets: len(assets),
	}

	for _, asset := range assets {
		item := domain.PlanItem{
			Asset: asset,
		}

		meta, err := t.findDateAnchor(asset)
		if err != nil {
			item.Action = "跳过（无法提取拍摄日期）"
			item.Warning = "缺少拍摄日期元数据"
			result.WarningsCount++
			result.Items = append(result.Items, item)
			continue
		}

		item.EstimatedTime = meta.DateTimeOriginal
		archiveDir, err := t.archiver.BuildArchiveDir(t.targetDir, meta.DateTimeOriginal)
		if err != nil {
			item.Action = "跳过（解析拍摄日期失败）"
			item.Warning = err.Error()
			result.WarningsCount++
			result.Items = append(result.Items, item)
			continue
		}

		newBase := t.archiver.CalculateNormalizedName(asset.BaseName, meta.DateTimeOriginal)
		item.TargetPath = filepath.Join(archiveDir, newBase)

		if conflict, targetFile := t.archiver.CheckConflict(asset.AllFiles(), archiveDir, newBase); conflict {
			item.Action = "冲突预警（目标文件已存在）"
			item.Warning = fmt.Sprintf("文件已存在: %s", filepath.Base(targetFile))
			result.WarningsCount++
			result.Items = append(result.Items, item)
			continue
		}

		item.Action = fmt.Sprintf("重命名并归档至 %s/", filepath.Base(archiveDir))
		result.ReadyCount++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// Execute 执行归档
func (t *Task) Execute(ctx context.Context, eventCh chan<- domain.ProgressEvent) (*domain.TaskSummary, []domain.Issue, error) {
	sendEvent := func(e domain.ProgressEvent) {
		if eventCh != nil {
			select {
			case eventCh <- e:
			default:
			}
		}
	}

	sendEvent(domain.NewInfoEvent(domain.StageDiscover, fmt.Sprintf("开始扫描源目录: %s", t.sourceDir)))

	assets, err := t.discoverer.Discover(t.sourceDir)
	if err != nil {
		return nil, nil, err
	}

	summary := &domain.TaskSummary{
		TotalAssets: len(assets),
	}

	if len(assets) == 0 {
		sendEvent(domain.NewInfoEvent(domain.StageComplete, "源目录未找到任何照片文件。"))
		return summary, nil, nil
	}

	var issues []domain.Issue

	for i, asset := range assets {
		select {
		case <-ctx.Done():
			return summary, issues, ctx.Err()
		default:
		}

		sendEvent(domain.NewAssetProgressEvent(
			domain.StagePrecheck,
			domain.LevelInfo,
			fmt.Sprintf("正在处理: %s", asset.DisplayName()),
			&asset,
			i+1,
			len(assets),
		))

		meta, err := t.findDateAnchor(asset)
		if err != nil {
			issue := domain.Issue{
				Kind:       domain.IssueKindMissingDate,
				Reason:     "无法提取拍摄日期 (DateTimeOriginal)",
				Suggestion: "请检查该文件 EXIF 元数据是否完整。",
				Asset:      asset,
			}
			issues = append(issues, issue)
			summary.Pending++
			sendEvent(domain.ProgressEvent{
				Stage:   domain.StagePrecheck,
				Level:   domain.LevelWarn,
				Message: fmt.Sprintf("跳过组 %s: 无法提取拍摄日期", asset.DisplayName()),
				Asset:   &asset,
				Issue:   &issue,
			})
			continue
		}

		archiveDir, err := t.archiver.BuildArchiveDir(t.targetDir, meta.DateTimeOriginal)
		if err != nil {
			issue := domain.Issue{
				Kind:       domain.IssueKindFailure,
				Reason:     fmt.Sprintf("构建归档路径失败: %v", err),
				Suggestion: "请检查拍摄日期格式。",
				Asset:      asset,
				PhotoTime:  meta.DateTimeOriginal,
			}
			issues = append(issues, issue)
			summary.Failed++
			continue
		}

		newBase := t.archiver.CalculateNormalizedName(asset.BaseName, meta.DateTimeOriginal)

		if conflict, target := t.archiver.CheckConflict(asset.AllFiles(), archiveDir, newBase); conflict {
			issue := domain.Issue{
				Kind:       domain.IssueKindConflict,
				Reason:     fmt.Sprintf("目标文件已存在：%s", target),
				Suggestion: "请确认是否重复归档或重命名冲突。",
				Asset:      asset,
				PhotoTime:  meta.DateTimeOriginal,
			}
			issues = append(issues, issue)
			summary.Pending++
			sendEvent(domain.ProgressEvent{
				Stage:        domain.StageArchive,
				Level:        domain.LevelWarn,
				Message:      fmt.Sprintf("目标冲突跳过: %s", target),
				Asset:        &asset,
				Issue:        &issue,
				CurrentIndex: i + 1,
				TotalItems:   len(assets),
			})
			continue
		}

		if err := t.archiver.MoveFilesWithRename(asset.AllFiles(), archiveDir, newBase); err != nil {
			issue := domain.Issue{
				Kind:       domain.IssueKindFailure,
				Reason:     fmt.Sprintf("文件移动失败: %v", err),
				Suggestion: "请检查目标目录权限与磁盘空间。",
				Asset:      asset,
				PhotoTime:  meta.DateTimeOriginal,
			}
			issues = append(issues, issue)
			summary.Failed++
			sendEvent(domain.ProgressEvent{
				Stage:        domain.StageArchive,
				Level:        domain.LevelError,
				Message:      fmt.Sprintf("移动失败: %v", err),
				Asset:        &asset,
				Issue:        &issue,
				CurrentIndex: i + 1,
				TotalItems:   len(assets),
			})
			continue
		}

		summary.Success++
		sendEvent(domain.ProgressEvent{
			Stage:        domain.StageArchive,
			Level:        domain.LevelSuccess,
			Message:      fmt.Sprintf("已成功归档到 %s/ (%s)", filepath.Base(archiveDir), newBase),
			Asset:        &asset,
			CurrentIndex: i + 1,
			TotalItems:   len(assets),
		})
	}

	reportPath := filepath.Join(t.targetDir, "_organize_logs", "organize_pending_report_latest.md")
	_ = t.reporter.WriteMarkdownReport(reportPath, t.Name(), issues)

	sendEvent(domain.ProgressEvent{
		Stage:   domain.StageComplete,
		Level:   domain.LevelSuccess,
		Message: fmt.Sprintf("整理完成：共处理 %d 组，成功 %d，待补/冲突 %d，失败 %d", summary.TotalAssets, summary.Success, summary.Pending, summary.Failed),
	})

	return summary, issues, nil
}
