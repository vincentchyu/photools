package reversegeocode

import (
	"context"
	"fmt"
	"sync"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/internal/i18n"
	"github.com/vincentchyu/photools/pkg/geocoding"
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
		runner = exiftool.DefaultRunner()
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
	return i18n.T("capGeocodeName")
}

func (c *Capability) Description() string {
	return i18n.T("capGeocodeDesc")
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
			NameKey:      "optGeocodeLangName",
			DescKey:      "optGeocodeLangDesc",
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
		cb := func(stage string, percent float64, msg string, status geocoding.HealthStatus, err error) {
			rep := domain.PluginInitReport{
				PluginID: c.ID(),
				Name:     c.Name(),
				Stage:    stage,
				Message:  msg,
				Percent:  percent,
				Status:   domain.PluginHealthStatus(status),
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
				ActionDesc: fmt.Sprintf(i18n.T("actionGeocodeWrite"), loc.FormatSummary()),
			}
		}
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionGeocodeMiss"),
			Warning:    fmt.Sprintf("坐标 (%.4f, %.4f) 未在离线地理库中命中有效地点", lat, lon),
		}
	}

	if c.allowNoGPS {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionGeocodeSkipNoGPS"),
			Warning:    "",
		}
	}

	return domain.CapabilityPlan{
		CanProcess: false,
		ActionDesc: i18n.T("actionGeocodeWaiting"),
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

	// 1. 写入地名元数据（根据 SidecarPolicy 四档策略调度写入目标）
	targetMain := actx.Asset.PrimaryPath()
	if targetMain != "" {
		modifiedFiles, err := exiftool.WriteLocation(c.runner, actx.Asset, *loc, actx.GetPolicy())
		if err != nil {
			return err
		}
		for _, f := range modifiedFiles {
			actx.RecordModifiedFile(f)
		}
	}

	if sendEvent != nil {
		sendEvent(domain.ProgressEvent{
			Stage:   domain.StageGeocode,
			Level:   domain.LevelSuccess,
			Message: fmt.Sprintf(i18n.T("logGeocodeTaggedSuccess"), actx.Asset.DisplayName(), loc.FormatSummary()),
			Asset:   &actx.Asset,
		})
	}

	return nil
}
