package reversegeocode

import (
	"context"
	"fmt"
	"sync"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
	"github.com/vincentchyu/photo-processing/internal/geocoding"
)

// Config 封装逆地理编码插件配置
type Config struct {
	Runner     exiftool.CommandRunner
	Geocoder   *geocoding.ReverseGeocoder
	Priority   int
	AllowNoGPS bool
}

// Capability 负责为具备 GPS 坐标的照片反查地名并写入元数据
type Capability struct {
	runner     exiftool.CommandRunner
	geocoder   *geocoding.ReverseGeocoder
	priority   int
	allowNoGPS bool
	initOnce   sync.Once
	initErr    error
	lastReport domain.PluginInitReport
}

// NewCapability 创建逆地理编码插件实例
func NewCapability(cfg Config) *Capability {
	runner := cfg.Runner
	if runner == nil {
		runner = exiftool.ExecRunner{}
	}
	geocoder := cfg.Geocoder
	if geocoder == nil {
		geocoder = geocoding.GetDefault()
	}
	p := cfg.Priority
	if p <= 0 {
		p = 20
	}
	return &Capability{
		runner:     runner,
		geocoder:   geocoder,
		priority:   p,
		allowNoGPS: cfg.AllowNoGPS,
	}
}

func (c *Capability) ID() domain.CapabilityID {
	return domain.CapReverseGeocode
}

func (c *Capability) Name() string {
	return "逆地理编码与地名元数据写入"
}

func (c *Capability) Description() string {
	return "根据 GPS 经纬度坐标检索国家/省份/城市/区县/POI，写入照片 IPTC 与 XMP 地名标签"
}

func (c *Capability) RequiredStage() domain.PipelineStage {
	return domain.StageGeocode
}

func (c *Capability) Priority() int {
	return c.priority
}

// SupportedOptions 声明逆地理编码支持的可配置项
func (c *Capability) SupportedOptions() []domain.OptionSpec {
	return []domain.OptionSpec{
		{
			Key:          "language",
			Name:         "地名输出语言格式 (Language)",
			Description:  "写入照片元数据的地名语言（默认 zh-CN 规范中文地名）",
			Type:         domain.OptionTypeString,
			DefaultValue: "zh-CN",
			Choices:      []string{"zh-CN", "en"},
		},
	}
}

// Configure 动态接收并注入配置
func (c *Capability) Configure(opts map[string]any) error {
	return nil
}

// Init 执行插件自检并流式汇报就绪进度 (使用 sync.Once 防止重复加载)
func (c *Capability) Init(ctx context.Context, report func(domain.PluginInitReport)) error {
	if c.geocoder == nil {
		c.geocoder = geocoding.GetDefault()
	}

	c.initOnce.Do(func() {
		cb := func(stage string, percent float64, msg string, status domain.PluginHealthStatus, err error) {
			rep := domain.PluginInitReport{
				PluginID: c.ID(),
				Name:     c.Name(),
				Stage:    stage,
				Message:  msg,
				Percent:  percent,
				Status:   status,
				Err:      err,
			}
			c.lastReport = rep
			if report != nil {
				report(rep)
			}
		}

		c.initErr = c.geocoder.InitProgressive(ctx, cb)
	})

	if report != nil && c.lastReport.Name != "" {
		report(c.lastReport)
	}

	return c.initErr
}

// PlanPrecheck 对单组资产进行只读预检评估
func (c *Capability) PlanPrecheck(ctx context.Context, actx *domain.AssetContext) domain.CapabilityPlan {
	lat, lon := actx.Latitude, actx.Longitude
	if lat == 0 && lon == 0 {
		meta := actx.GetMetadata()
		if meta.GPSPosition != "" {
			lat, lon, _ = exiftool.ParseCoordinates(meta.GPSPosition)
		} else if !actx.IsMetadataLoaded() {
			anchorPath := actx.Asset.PrimaryPath()
			if anchorPath != "" {
				m, err := exiftool.ReadMetadata(c.runner, anchorPath)
				if err == nil && m.GPSPosition != "" {
					meta = m
					actx.UpdateMetadata(meta)
					lat, lon, _ = exiftool.ParseCoordinates(meta.GPSPosition)
				}
			}
		}
	}

	if lat != 0 || lon != 0 {
		actx.SetGPS(lat, lon)
		loc := c.geocoder.Lookup(lat, lon)
		if loc != nil {
			actx.SetLocation(loc)
			return domain.CapabilityPlan{
				CanProcess: true,
				ActionDesc: fmt.Sprintf("写入地名 (%s)", loc.FormatSummary()),
			}
		}
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "未命中离线库",
			Warning:    fmt.Sprintf("坐标 (%.4f, %.4f) 未在离线地理库中命中有效地点", lat, lon),
		}
	}

	if c.allowNoGPS {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: "跳过地名（无 GPS 坐标，允许降级归档）",
			Warning:    "",
		}
	}

	return domain.CapabilityPlan{
		CanProcess: false,
		ActionDesc: "等待 GPS 坐标",
		Warning:    "当前资产尚无 GPS 坐标",
	}
}

// ExecuteProcess 正式执行逆地理编码与 IPTC/XMP 地名元数据写入
func (c *Capability) ExecuteProcess(ctx context.Context, actx *domain.AssetContext, sendEvent func(domain.ProgressEvent)) error {
	lat, lon := actx.Latitude, actx.Longitude
	if lat == 0 && lon == 0 {
		anchor := actx.Asset.PrimaryPath()
		if anchor != "" {
			meta, err := exiftool.ReadMetadata(c.runner, anchor)
			if err == nil && meta.GPSPosition != "" {
				actx.UpdateMetadata(meta)
				lat, lon, _ = exiftool.ParseCoordinates(meta.GPSPosition)
			}
		}
	}

	if lat == 0 && lon == 0 {
		return nil
	}

	actx.SetGPS(lat, lon)

	loc := c.geocoder.Lookup(lat, lon)
	if loc == nil {
		return nil
	}
	actx.SetLocation(loc)

	// 1. 写入主文件（RAW 优先，若无 RAW 则写入 JPG）
	targetMain := actx.Asset.PrimaryPath()
	if targetMain != "" {
		if err := exiftool.WriteLocation(c.runner, targetMain, *loc); err != nil {
			return fmt.Errorf("写入地名元数据失败 (%s): %w", targetMain, err)
		}
		actx.RecordModifiedFile(targetMain)
	}

	// 2. 同步地名元数据到配对的 JPG (若主文件为 RAW)
	if actx.Asset.HasRaw() && actx.Asset.HasJPG() {
		if err := exiftool.SyncLocationToJPG(c.runner, actx.Asset.RawPath, actx.Asset.JPGPath); err != nil {
			return fmt.Errorf("同步地名元数据到 JPG 失败: %w", err)
		}
		actx.RecordModifiedFile(actx.Asset.JPGPath)
	}

	// 3. 若存在 XMP，同步地名元数据到 XMP
	if actx.Asset.XMPPath != "" && targetMain != "" {
		if err := exiftool.SyncLocationToXMP(c.runner, targetMain, actx.Asset.XMPPath); err != nil {
			return fmt.Errorf("同步地名元数据到 XMP 失败: %w", err)
		}
		actx.RecordModifiedFile(actx.Asset.XMPPath)
	}

	if sendEvent != nil {
		sendEvent(domain.ProgressEvent{
			Stage:   domain.StageGeocode,
			Level:   domain.LevelSuccess,
			Message: fmt.Sprintf("已写入地名元数据：%s (%s)", actx.Asset.DisplayName(), loc.FormatSummary()),
			Asset:   &actx.Asset,
		})
	}

	return nil
}
