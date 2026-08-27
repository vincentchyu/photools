package gpxmatch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/exiftool"
)

// Config 封装 GPX 匹配能力初始化配置
type Config struct {
	Runner   exiftool.CommandRunner
	GPXFiles []string
	Geosync  string
	Priority int
}

// Capability 实现 domain.Capability 接口
type Capability struct {
	runner     exiftool.CommandRunner
	gpxFiles   []string
	geosync    string
	priority   int
	initOnce   sync.Once
	initErr    error
	lastReport domain.PluginInitReport
}

// NewCapability 实例化 GPX 匹配能力插件
func NewCapability(cfg Config) *Capability {
	p := cfg.Priority
	if p <= 0 {
		p = 10
	}
	runner := cfg.Runner
	if runner == nil {
		runner = exiftool.DefaultRunner()
	}
	return &Capability{
		runner:   runner,
		gpxFiles: cfg.GPXFiles,
		geosync:  cfg.Geosync,
		priority: p,
	}
}

func (c *Capability) ID() domain.CapabilityID {
	return domain.CapGPXMatching
}

func (c *Capability) Name() string {
	return "GPX 轨迹匹配与 GPS 修正"
}

func (c *Capability) Description() string {
	return "从 GPX 目录读取轨迹点时间轴，为 RAW 照片写入经纬度并同步到 JPG 与 XMP"
}

func (c *Capability) RequiredStage() domain.PipelineStage {
	return domain.StageGeotag
}

func (c *Capability) Priority() int {
	return c.priority
}

// SupportedOptions 声明 GPX 轨迹匹配支持的可配置项
func (c *Capability) SupportedOptions() []domain.OptionSpec {
	return []domain.OptionSpec{
		{
			Key:          "geosync",
			Name:         "时钟偏差偏移值 (Geosync)",
			Description:  "相机与 GPS 轨迹的时间偏差补偿值 (如 0, +00:00:05, -00:01:00)",
			Type:         domain.OptionTypeString,
			DefaultValue: "0",
			Choices:      []string{"0", "+00:00:05", "-00:01:00"},
		},
	}
}

// Configure 动态接收并注入配置
func (c *Capability) Configure(opts map[string]any) error {
	if opts == nil {
		return nil
	}
	if v, ok := opts["geosync"]; ok {
		if s, ok := v.(string); ok {
			c.geosync = s
		}
	}
	return nil
}

func (c *Capability) SetGPXFiles(files []string) {
	c.gpxFiles = files
}

func (c *Capability) SetGeosync(geosync string) {
	c.geosync = geosync
}

// Init 执行插件自检并流式汇报就绪进度 (使用 sync.Once 保证只自检一次)
func (c *Capability) Init(ctx context.Context, report func(domain.PluginInitReport)) error {
	c.initOnce.Do(func() {
		if report != nil {
			report(domain.PluginInitReport{
				PluginID: c.ID(),
				Name:     c.Name(),
				Stage:    "环境自检",
				Message:  "正在检查 ExifTool 运行环境与版本...",
				Percent:  0.3,
				Status:   domain.HealthReady,
			})
		}

		out, err := c.runner.CombinedOutput("exiftool", "-ver")
		if err != nil {
			c.initErr = err
			c.lastReport = domain.PluginInitReport{
				PluginID: c.ID(),
				Name:     c.Name(),
				Stage:    "自检失败",
				Message:  "未在系统 PATH 中找到可用 exiftool 命令行工具",
				Percent:  1.0,
				Status:   domain.HealthFailed,
				Err:      err,
			}
			return
		}

		ver := strings.TrimSpace(string(out))
		c.lastReport = domain.PluginInitReport{
			PluginID: c.ID(),
			Name:     c.Name(),
			Stage:    "自检完成",
			Message:  fmt.Sprintf("ExifTool 核心引擎就绪 (v%s)", ver),
			Percent:  1.0,
			Status:   domain.HealthReady,
		}
	})

	if report != nil && c.lastReport.Name != "" {
		report(c.lastReport)
	}
	return c.initErr
}

// PlanPrecheck 对单组资产进行只读预检评估
func (c *Capability) PlanPrecheck(ctx context.Context, actx *domain.AssetContext) domain.CapabilityPlan {
	primary := actx.Asset.PrimaryPath()
	if primary == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "跳过（无主文件）",
		}
	}

	if len(c.gpxFiles) == 0 {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "等待 GPX 轨迹文件",
			Warning:    "未找到任何 GPX 轨迹文件，无法进行时间轴匹配",
		}
	}

	meta := actx.GetMetadata()
	if meta.DateTimeOriginal == "" && !actx.IsMetadataLoaded() {
		m, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil {
			return domain.CapabilityPlan{
				CanProcess: false,
				ActionDesc: "读取失败",
				Warning:    fmt.Sprintf("无法读取主文件元数据: %v", err),
			}
		}
		meta = m
		actx.UpdateMetadata(meta)
	}

	if meta.DateTimeOriginal == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "缺少拍摄时间",
			Warning:    "主文件缺少 DateTimeOriginal 拍摄时间元数据",
		}
	}

	if meta.GPSPosition != "" {
		return domain.CapabilityPlan{
			CanProcess: true,
			ActionDesc: fmt.Sprintf("校准已有 GPS (%s)", meta.GPSPosition),
		}
	}

	candidateGPX := FilterGPXFilesByDate(c.gpxFiles, meta.DateTimeOriginal)
	var gpxNames []string
	for _, g := range candidateGPX {
		gpxNames = append(gpxNames, filepath.Base(g))
	}
	return domain.CapabilityPlan{
		CanProcess: true,
		ActionDesc: fmt.Sprintf("匹配轨迹 (%s)", strings.Join(gpxNames, ",")),
	}
}

// ExecuteProcess 正式执行 GPX 匹配与 GPS 写入/同步
func (c *Capability) ExecuteProcess(ctx context.Context, actx *domain.AssetContext, sendEvent func(domain.ProgressEvent)) error {
	primary := actx.Asset.PrimaryPath()
	if primary == "" {
		return nil
	}

	// 1. 读取或复用拍摄时间
	primaryMeta := actx.GetMetadata()
	if primaryMeta.DateTimeOriginal == "" {
		meta, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil {
			return fmt.Errorf("读取主文件元数据失败: %w", err)
		}
		primaryMeta = meta
		actx.UpdateMetadata(meta)
	}

	// 2. 按拍摄日期智能筛选相关 GPX 轨迹（±1 天容差）
	targetGPX := FilterGPXFilesByDate(c.gpxFiles, primaryMeta.DateTimeOriginal)

	// 3. 写入主文件 GPS (RAW 或 JPG)
	output, err := exiftool.WriteGeotag(c.runner, primary, targetGPX, c.geosync)
	if err != nil {
		return fmt.Errorf("主文件写入 GPS 失败: %s", exiftool.ClassifyFailure(output, err))
	}

	// 3. 二次校验主文件 GPSPosition
	updatedMeta, err := exiftool.ReadMetadata(c.runner, primary)
	if err != nil || updatedMeta.GPSPosition == "" {
		return fmt.Errorf("二次校验未检测到 GPSPosition，可能时间未命中轨迹: %s", exiftool.ClassifyFailure(output, err))
	}

	actx.UpdateMetadata(updatedMeta)
	if lat, lon, err := exiftool.ParseCoordinates(updatedMeta.GPSPosition); err == nil {
		actx.SetGPS(lat, lon)
	}
	actx.RecordModifiedFile(primary)

	// 4. 若主文件是 RAW 且存在配对的同名 JPG，同步 GPS 经纬度到 JPG
	if actx.Asset.HasRaw() && actx.Asset.HasJPG() {
		if err := exiftool.SyncGPSToJPG(c.runner, actx.Asset.RawPath, actx.Asset.JPGPath); err != nil {
			return fmt.Errorf("同步 GPS 到 JPG 失败: %w", err)
		}
		actx.RecordModifiedFile(actx.Asset.JPGPath)
	}

	// 5. 若存在 XMP，同步 GPS 经纬度到 XMP
	if actx.Asset.HasXMP() {
		sourceForXMP := primary
		if err := exiftool.SyncGPSToXMP(c.runner, sourceForXMP, actx.Asset.XMPPath); err != nil {
			return fmt.Errorf("同步 GPS 到 XMP 失败: %w", err)
		}
		actx.RecordModifiedFile(actx.Asset.XMPPath)
	}

	if sendEvent != nil {
		var gpxNames []string
		for _, g := range targetGPX {
			gpxNames = append(gpxNames, filepath.Base(g))
		}
		trackInfo := strings.Join(gpxNames, ", ")
		sendEvent(domain.ProgressEvent{
			Stage:   domain.StageGeotag,
			Level:   domain.LevelSuccess,
			Message: fmt.Sprintf("GPS 写入并同步成功：%s (%s) [匹配轨迹: %s]", actx.Asset.DisplayName(), updatedMeta.GPSPosition, trackInfo),
			Asset:   &actx.Asset,
		})
	}

	return nil
}
