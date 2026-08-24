package datearchive

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
)

// Capability 封装按拍摄日期归档与规范重命名能力 (能力 3)
type Capability struct {
	runner       exiftool.CommandRunner
	archiver     *engine.Archiver
	processedDir string
	inPlace      bool
	priority     int
	initOnce     sync.Once
	initErr      error
	lastReport   domain.PluginInitReport
}

// Config 初始化配置
type Config struct {
	Runner         exiftool.CommandRunner
	Archiver       *engine.Archiver
	ProcessedDir   string
	NamingTemplate string
	InPlace        bool
	Priority       int
}

// NewCapability 实例化日期归档能力
func NewCapability(cfg Config) *Capability {
	runner := cfg.Runner
	if runner == nil {
		runner = exiftool.ExecRunner{}
	}
	archiver := cfg.Archiver
	if archiver == nil {
		archiver = engine.NewArchiver(cfg.NamingTemplate)
	}
	p := cfg.Priority
	if p <= 0 {
		p = 100
	}
	return &Capability{
		runner:       runner,
		archiver:     archiver,
		processedDir: cfg.ProcessedDir,
		inPlace:      cfg.InPlace,
		priority:     p,
	}
}

func (c *Capability) ID() domain.CapabilityID {
	return domain.CapDateArchive
}

func (c *Capability) Name() string {
	return "拍摄日期归档与规范重命名"
}

func (c *Capability) Description() string {
	return "根据 EXIF 拍摄日期重命名并在 Processed/YYYY/MMDD/ 目录下归档整组拍摄单元"
}

func (c *Capability) RequiredStage() domain.PipelineStage {
	return domain.StageArchive
}

func (c *Capability) Priority() int {
	return c.priority
}

// SupportedOptions 声明日期归档与重命名支持的可配置项
func (c *Capability) SupportedOptions() []domain.OptionSpec {
	return []domain.OptionSpec{
		{
			Key:          "in_place",
			Name:         "原地规范重命名模式 (In-Place)",
			Description:  "true 表示在源目录下就地重命名，不建立 Processed/YYYY/MMDD/ 分层目录",
			Type:         domain.OptionTypeBool,
			DefaultValue: false,
			Choices:      []string{"false", "true"},
		},
	}
}

// Configure 动态接收并注入配置
func (c *Capability) Configure(opts map[string]any) error {
	if opts == nil {
		return nil
	}
	if v, ok := opts["in_place"]; ok {
		switch val := v.(type) {
		case bool:
			c.inPlace = val
		case string:
			c.inPlace = val == "true"
		}
	}
	return nil
}

func (c *Capability) SetInPlace(inPlace bool) {
	c.inPlace = inPlace
}

// Init 执行日期归档插件规则与环境自检 (使用 sync.Once 保证只自检一次)
func (c *Capability) Init(ctx context.Context, report func(domain.PluginInitReport)) error {
	c.initOnce.Do(
		func() {
			c.lastReport = domain.PluginInitReport{
				PluginID: c.ID(),
				Name:     c.Name(),
				Stage:    "规则就绪",
				Message:  "拍摄日期归档引擎与规范化重命名模板已就绪",
				Percent:  1.0,
				Status:   domain.HealthReady,
			}
		},
	)

	if report != nil && c.lastReport.Name != "" {
		report(c.lastReport)
	}
	return c.initErr
}

// PlanPrecheck 预检拍摄日期与归档目标路径
func (c *Capability) PlanPrecheck(ctx context.Context, actx *domain.AssetContext) domain.CapabilityPlan {
	dateStr := actx.GetMetadata().DateTimeOriginal
	if dateStr == "" && !actx.IsMetadataLoaded() {
		// 尝试从主文件读取一次预检元数据
		anchor := actx.Asset.PrimaryPath()
		if anchor != "" {
			meta, err := exiftool.ReadMetadata(c.runner, anchor)
			if err == nil && meta.DateTimeOriginal != "" {
				dateStr = meta.DateTimeOriginal
				actx.UpdateMetadata(meta)
			}
		}
	}

	if dateStr == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "保留在原目录（缺少拍摄时间）",
			Warning:    "未检测到 DateTimeOriginal 拍摄日期",
		}
	}

	var targetDir string
	var actionDesc string
	if c.inPlace {
		// 原地重命名模式：目标目录直接为源文件所在目录
		sourceFile := actx.Asset.PrimaryPath()
		targetDir = filepath.Dir(sourceFile)
		newBase := c.archiver.CalculateNormalizedName(actx.Asset.BaseName, dateStr)
		actionDesc = fmt.Sprintf("原地规范重命名 -> %s", newBase)
	} else {
		var err error
		targetDir, err = c.archiver.BuildArchiveDir(c.processedDir, dateStr)
		if err != nil {
			return domain.CapabilityPlan{
				CanProcess: false,
				ActionDesc: "无法归档",
				Warning:    fmt.Sprintf("计算归档路径失败: %v", err),
			}
		}
		actionDesc = fmt.Sprintf("归档至 %s/", filepath.Base(targetDir))
	}

	return domain.CapabilityPlan{
		CanProcess: true,
		ActionDesc: actionDesc,
	}
}

// ExecuteProcess 执行文件安全重命名与移动归档
func (c *Capability) ExecuteProcess(
	ctx context.Context, actx *domain.AssetContext, sendEvent func(domain.ProgressEvent),
) error {
	dateStr := actx.GetMetadata().DateTimeOriginal
	if dateStr == "" {
		anchor := actx.Asset.PrimaryPath()
		if anchor != "" {
			meta, err := exiftool.ReadMetadata(c.runner, anchor)
			if err == nil && meta.DateTimeOriginal != "" {
				dateStr = meta.DateTimeOriginal
				actx.UpdateMetadata(meta)
			}
		}
	}

	if dateStr == "" {
		return fmt.Errorf("无法确定拍摄日期，拒绝归档以免发生目录错乱")
	}

	var targetDir string
	if c.inPlace {
		sourceFile := actx.Asset.PrimaryPath()
		targetDir = filepath.Dir(sourceFile)
	} else {
		var err error
		targetDir, err = c.archiver.BuildArchiveDir(c.processedDir, dateStr)
		if err != nil {
			return fmt.Errorf("计算归档目录失败: %w", err)
		}
	}

	newBase := c.archiver.CalculateNormalizedName(actx.Asset.BaseName, dateStr)
	if err := c.archiver.MoveFilesWithRename(actx.Asset.AllFiles(), targetDir, newBase); err != nil {
		return fmt.Errorf("移动与重命名归档文件失败: %w", err)
	}

	actx.TargetDir = targetDir
	actx.NewBaseName = newBase

	if sendEvent != nil {
		msg := fmt.Sprintf("已归档到 %s/ (%s)", filepath.Base(targetDir), newBase)
		if c.inPlace {
			msg = fmt.Sprintf("已原地规范重命名为: %s", newBase)
		}
		sendEvent(
			domain.ProgressEvent{
				Stage:   domain.StageArchive,
				Level:   domain.LevelSuccess,
				Message: msg,
				Asset:   &actx.Asset,
			},
		)
	}

	return nil
}
