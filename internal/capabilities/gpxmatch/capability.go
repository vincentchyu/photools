package gpxmatch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/internal/i18n"
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
	return i18n.T("capGpxName")
}

func (c *Capability) Description() string {
	return i18n.T("capGpxDesc")
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
			NameKey:      "optGeosyncName",
			DescKey:      "optGeosyncDesc",
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
				Stage:    i18n.T("stageInit"),
				Message:  i18n.T("logGpxSelfCheckChecking"),
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
				Stage:    i18n.T("stageInit"),
				Message:  i18n.T("logGpxSelfCheckFailed"),
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
			Stage:    i18n.T("statusReady"),
			Message:  fmt.Sprintf(i18n.T("logGpxSelfCheckReady"), ver),
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
			ActionDesc: i18n.T("actionInterpolateSkipNoPrimary"),
		}
	}

	meta := actx.GetMetadata()
	if (meta.DateTimeOriginal == "" || meta.GPSPosition == "") && !actx.IsMetadataLoaded() {
		m, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil {
			return domain.CapabilityPlan{
				CanProcess: false,
				ActionDesc: i18n.T("tuiMenuHealthFailed"),
				Warning:    fmt.Sprintf("无法读取主文件元数据: %v", err),
			}
		}
		meta = m
		actx.UpdateMetadata(meta)
	}

	// 1. 检查是否有拍摄时间
	if meta.DateTimeOriginal == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionGpxMissingTime"),
			Warning:    "主文件缺少 DateTimeOriginal 拍摄时间元数据",
		}
	}

	// 2. 检查是否有与拍摄日期相匹配的 GPX 轨迹文件
	candidateGPX := FilterGPXFilesByDate(c.gpxFiles, meta.DateTimeOriginal)
	if len(candidateGPX) == 0 {
		if meta.GPSPosition != "" || meta.HasGPS() || actx.HasGPS {
			return domain.CapabilityPlan{
				CanProcess: false,
				ActionDesc: i18n.T("actionGpxKeepExisting"),
				Warning:    "",
			}
		}
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionGpxWaiting"),
			Warning:    "未找到该拍摄日期的 GPX 轨迹文件，无法进行时间轴匹配",
		}
	}

	var gpxNames []string
	for _, g := range candidateGPX {
		gpxNames = append(gpxNames, filepath.Base(g))
	}
	if meta.GPSPosition != "" || meta.HasGPS() || actx.HasGPS {
		return domain.CapabilityPlan{
			CanProcess: true,
			ActionDesc: fmt.Sprintf(i18n.T("actionGpxVerify"), strings.Join(gpxNames, ",")),
		}
	}
	return domain.CapabilityPlan{
		CanProcess: true,
		ActionDesc: fmt.Sprintf(i18n.T("actionGpxMatch"), strings.Join(gpxNames, ",")),
	}
}

func isGPSPositionChanged(oldPos, newPos string) bool {
	if oldPos == "" && newPos != "" {
		return true
	}
	if oldPos != "" && newPos == "" {
		return true
	}
	if oldPos == "" && newPos == "" {
		return false
	}
	oldLat, oldLon, err1 := exiftool.ParseCoordinates(oldPos)
	newLat, newLon, err2 := exiftool.ParseCoordinates(newPos)
	if err1 != nil || err2 != nil {
		return oldPos != newPos
	}
	dLat := oldLat - newLat
	if dLat < 0 {
		dLat = -dLat
	}
	dLon := oldLon - newLon
	if dLon < 0 {
		dLon = -dLon
	}
	// 经纬度差大于 0.00001 度（约 1 米）视为发生有效修正
	return dLat > 0.00001 || dLon > 0.00001
}

// ExecuteProcess 正式执行 GPX 匹配与 GPS 写入/同步
func (c *Capability) ExecuteProcess(ctx context.Context, actx *domain.AssetContext, sendEvent func(domain.ProgressEvent)) error {
	primary := actx.Asset.PrimaryPath()
	if primary == "" {
		return nil
	}

	// 1. 读取或复用拍摄时间与原始 GPS 坐标
	primaryMeta := actx.GetMetadata()
	if primaryMeta.DateTimeOriginal == "" || (primaryMeta.GPSPosition == "" && !actx.IsMetadataLoaded()) {
		meta, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil {
			return fmt.Errorf("读取主文件元数据失败: %w", err)
		}
		primaryMeta = meta
		actx.UpdateMetadata(meta)
	}

	origGPSPos := primaryMeta.GPSPosition

	// 2. 按拍摄日期智能筛选相关 GPX 轨迹（±1 天容差）
	targetGPX := FilterGPXFilesByDate(c.gpxFiles, primaryMeta.DateTimeOriginal)
	if len(targetGPX) == 0 {
		if origGPSPos != "" {
			if sendEvent != nil {
				sendEvent(domain.ProgressEvent{
					Stage:   domain.StageGeotag,
					Level:   domain.LevelInfo,
					Message: fmt.Sprintf(i18n.T("logGpxKeepCameraGps"), actx.Asset.DisplayName(), origGPSPos),
					Asset:   &actx.Asset,
				})
			}
			return nil
		}
		return fmt.Errorf("未找到拍摄日期匹配的 GPX 轨迹")
	}

	// 3. 构造 GPX 匹配溯源指纹
	prov := domain.GPSProvenance{
		Source:      domain.GPSSourceGPX,
		MatchMethod: domain.GPSMatchMethodTimeProximity,
		Processor:   domain.DefaultProcessorName,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	var trackNames []string
	for _, g := range targetGPX {
		trackNames = append(trackNames, filepath.Base(g))
	}
	trackInfo := strings.Join(trackNames, ", ")

	// 4. 写入 GPS 坐标（根据 SidecarPolicy 四档策略调度写入目标）
	switch actx.GetPolicy() {
	case domain.PolicySidecarOnly:
		targetXMP := actx.Asset.SidecarPathFor(primary)
		if actx.Asset.HasXMP() {
			targetXMP = actx.Asset.XMPPath
		}
		output, err := exiftool.WriteGeotagToXMP(c.runner, primary, targetXMP, targetGPX, c.geosync)
		if err != nil {
			return fmt.Errorf("生成/写入 XMP 侧车文件 GPS 失败: %s", exiftool.ClassifyFailure(output, err))
		}

		updatedMeta, err := exiftool.ReadMetadata(c.runner, targetXMP)
		if err != nil || updatedMeta.GPSPosition == "" {
			return fmt.Errorf("二次校验 XMP 未检测到 GPSPosition，可能时间未命中轨迹: %s", exiftool.ClassifyFailure(output, err))
		}

		coordsChanged := origGPSPos == "" || isGPSPositionChanged(origGPSPos, updatedMeta.GPSPosition)
		if !coordsChanged && origGPSPos != "" {
			if sendEvent != nil {
				sendEvent(domain.ProgressEvent{
					Stage:   domain.StageGeotag,
					Level:   domain.LevelInfo,
					Message: fmt.Sprintf(i18n.T("logGpxDriftVerifiedOk"), actx.Asset.DisplayName(), origGPSPos),
					Asset:   &actx.Asset,
				})
			}
			return nil
		}

		actx.UpdateMetadata(updatedMeta)
		if lat, lon, err := exiftool.ParseCoordinates(updatedMeta.GPSPosition); err == nil {
			actx.SetGPSWithProvenance(lat, lon, 0, prov)
			_ = exiftool.WriteCoordinatesToXMPWithProvenance(c.runner, targetXMP, lat, lon, 0, prov)
		}
		actx.RecordModifiedFile(targetXMP)

		if actx.Asset.HasRaw() && actx.Asset.HasJPG() {
			jpgXMP := actx.Asset.SidecarPathFor(actx.Asset.JPGPath)
			if jpgXMP != targetXMP {
				_, _ = exiftool.WriteGeotagToXMP(c.runner, actx.Asset.JPGPath, jpgXMP, targetGPX, c.geosync)
				actx.RecordModifiedFile(jpgXMP)
			}
		}

	case domain.PolicySmart, domain.PolicyReadOnly:
		if actx.Asset.HasRaw() {
			output, err := exiftool.WriteGeotag(c.runner, actx.Asset.RawPath, targetGPX, c.geosync)
			if err != nil {
				return fmt.Errorf("RAW 写入 GPS 失败: %s", exiftool.ClassifyFailure(output, err))
			}
			updatedMeta, err := exiftool.ReadMetadata(c.runner, actx.Asset.RawPath)
			if err != nil || updatedMeta.GPSPosition == "" {
				return fmt.Errorf("二次校验 RAW 未检测到 GPSPosition: %s", exiftool.ClassifyFailure(output, err))
			}

			coordsChanged := origGPSPos == "" || isGPSPositionChanged(origGPSPos, updatedMeta.GPSPosition)
			if !coordsChanged && origGPSPos != "" {
				if sendEvent != nil {
					sendEvent(domain.ProgressEvent{
						Stage:   domain.StageGeotag,
						Level:   domain.LevelInfo,
						Message: fmt.Sprintf(i18n.T("logGpxDriftVerifiedOk"), actx.Asset.DisplayName(), origGPSPos),
						Asset:   &actx.Asset,
					})
				}
				return nil
			}

			actx.UpdateMetadata(updatedMeta)
			if lat, lon, err := exiftool.ParseCoordinates(updatedMeta.GPSPosition); err == nil {
				actx.SetGPSWithProvenance(lat, lon, 0, prov)
			}
			actx.RecordModifiedFile(actx.Asset.RawPath)

			targetXMP := actx.Asset.SidecarPathFor(actx.Asset.RawPath)
			if actx.Asset.HasXMP() {
				targetXMP = actx.Asset.XMPPath
			}
			if err := exiftool.SyncGPSToXMPWithProvenance(c.runner, actx.Asset.RawPath, targetXMP, prov); err == nil {
				actx.RecordModifiedFile(targetXMP)
			}

			if actx.Asset.HasJPG() {
				if err := exiftool.SyncGPSToJPG(c.runner, actx.Asset.RawPath, actx.Asset.JPGPath); err == nil {
					actx.RecordModifiedFile(actx.Asset.JPGPath)
				}
			}
		} else {
			output, err := exiftool.WriteGeotag(c.runner, primary, targetGPX, c.geosync)
			if err != nil {
				return fmt.Errorf("JPG 写入 GPS 失败: %s", exiftool.ClassifyFailure(output, err))
			}
			updatedMeta, err := exiftool.ReadMetadata(c.runner, primary)
			if err != nil || updatedMeta.GPSPosition == "" {
				return fmt.Errorf("二次校验 JPG 未检测到 GPSPosition: %s", exiftool.ClassifyFailure(output, err))
			}

			coordsChanged := origGPSPos == "" || isGPSPositionChanged(origGPSPos, updatedMeta.GPSPosition)
			if !coordsChanged && origGPSPos != "" {
				if sendEvent != nil {
					sendEvent(domain.ProgressEvent{
						Stage:   domain.StageGeotag,
						Level:   domain.LevelInfo,
						Message: fmt.Sprintf(i18n.T("logGpxDriftVerifiedOk"), actx.Asset.DisplayName(), origGPSPos),
						Asset:   &actx.Asset,
					})
				}
				return nil
			}

			actx.UpdateMetadata(updatedMeta)
			if lat, lon, err := exiftool.ParseCoordinates(updatedMeta.GPSPosition); err == nil {
				actx.SetGPSWithProvenance(lat, lon, 0, prov)
			}
			actx.RecordModifiedFile(primary)

			if actx.Asset.HasXMP() {
				_ = exiftool.SyncGPSToXMPWithProvenance(c.runner, primary, actx.Asset.XMPPath, prov)
				actx.RecordModifiedFile(actx.Asset.XMPPath)
			}
		}

	case domain.PolicyEmbedAndSidecar, domain.PolicyEmbedOnly:
		output, err := exiftool.WriteGeotag(c.runner, primary, targetGPX, c.geosync)
		if err != nil {
			return fmt.Errorf("主文件写入 GPS 失败: %s", exiftool.ClassifyFailure(output, err))
		}

		updatedMeta, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil || updatedMeta.GPSPosition == "" {
			return fmt.Errorf("二次校验未检测到 GPSPosition，可能时间未命中轨迹: %s", exiftool.ClassifyFailure(output, err))
		}

		coordsChanged := origGPSPos == "" || isGPSPositionChanged(origGPSPos, updatedMeta.GPSPosition)
		if !coordsChanged && origGPSPos != "" {
			if sendEvent != nil {
				sendEvent(domain.ProgressEvent{
					Stage:   domain.StageGeotag,
					Level:   domain.LevelInfo,
					Message: fmt.Sprintf(i18n.T("logGpxDriftVerifiedOk"), actx.Asset.DisplayName(), origGPSPos),
					Asset:   &actx.Asset,
				})
			}
			return nil
		}

		actx.UpdateMetadata(updatedMeta)
		if lat, lon, err := exiftool.ParseCoordinates(updatedMeta.GPSPosition); err == nil {
			actx.SetGPSWithProvenance(lat, lon, 0, prov)
		}
		actx.RecordModifiedFile(primary)

		if actx.Asset.HasRaw() && actx.Asset.HasJPG() {
			if err := exiftool.SyncGPSToJPG(c.runner, actx.Asset.RawPath, actx.Asset.JPGPath); err != nil {
				return fmt.Errorf("同步 GPS 到 JPG 失败: %w", err)
			}
			actx.RecordModifiedFile(actx.Asset.JPGPath)
		}

		if actx.GetPolicy() == domain.PolicyEmbedAndSidecar || actx.Asset.HasXMP() {
			targetXMP := actx.Asset.XMPPath
			if targetXMP == "" {
				targetXMP = actx.Asset.SidecarPathFor(primary)
			}
			if err := exiftool.SyncGPSToXMPWithProvenance(c.runner, primary, targetXMP, prov); err != nil {
				return fmt.Errorf("同步 GPS 到 XMP 失败: %w", err)
			}
			actx.RecordModifiedFile(targetXMP)
		}
	}

	if sendEvent != nil {
		if origGPSPos != "" {
			sendEvent(domain.ProgressEvent{
				Stage:   domain.StageGeotag,
				Level:   domain.LevelSuccess,
				Message: fmt.Sprintf(i18n.T("logGpxDriftCalibratedSuccess"), actx.Asset.DisplayName(), origGPSPos, actx.GetMetadata().GPSPosition, trackInfo),
				Asset:   &actx.Asset,
			})
		} else {
			sendEvent(domain.ProgressEvent{
				Stage:   domain.StageGeotag,
				Level:   domain.LevelSuccess,
				Message: fmt.Sprintf(i18n.T("logGpxTaggedSuccess"), actx.Asset.DisplayName(), actx.GetMetadata().GPSPosition, trackInfo),
				Asset:   &actx.Asset,
			})
		}
	}

	return nil
}
