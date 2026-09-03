package gpsinterpolate

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/internal/i18n"
)

// Config 封装 GPS 智能插值插件配置
type Config struct {
	Runner     exiftool.CommandRunner
	MaxTimeGap time.Duration // 最大允许插值/推断的时间差窗口（默认 15m）
	Priority   int
	AllowNoGPS bool // 是否允许无 GPS 照片安全跳过插值而不报错
}

// Capability 负责对未命中 GPX 轨迹的照片通过同批次前后邻近照片进行时间权重空间插值
type Capability struct {
	runner      exiftool.CommandRunner
	maxTimeGap  time.Duration
	priority    int
	allowNoGPS  bool
	initOnce    sync.Once
	initErr     error
	lastReport  domain.PluginInitReport
	indexMu     sync.Mutex
	anchorIndex *AnchorIndex
}

// NewCapability 创建 GPS 插值插件实例
func NewCapability(cfg Config) *Capability {
	runner := cfg.Runner
	if runner == nil {
		runner = exiftool.DefaultRunner()
	}
	p := cfg.Priority
	if p <= 0 {
		p = 15
	}
	maxGap := cfg.MaxTimeGap
	if maxGap <= 0 {
		maxGap = 15 * time.Minute
	}
	return &Capability{
		runner:     runner,
		maxTimeGap: maxGap,
		priority:   p,
		allowNoGPS: cfg.AllowNoGPS,
	}
}

func (c *Capability) ID() domain.CapabilityID {
	return domain.CapGPSInterpolate
}

func (c *Capability) Name() string {
	return i18n.T("capInterpolateName")
}

func (c *Capability) Description() string {
	return i18n.T("capInterpolateDesc")
}

func (c *Capability) RequiredStage() domain.PipelineStage {
	return domain.StageGeotag
}

func (c *Capability) Priority() int {
	return c.priority
}

// SupportedOptions 声明 GPS 智能插值推断支持的可配置项
func (c *Capability) SupportedOptions() []domain.OptionSpec {
	return []domain.OptionSpec{
		{
			Key:          "window",
			NameKey:      "optWindowName",
			DescKey:      "optWindowDesc",
			Type:         domain.OptionTypeDuration,
			DefaultValue: "15m",
			Choices:      []string{"15m", "30m", "1h", "2h", "4h"},
		},
	}
}

// Configure 动态接收并注入配置
func (c *Capability) Configure(opts map[string]any) error {
	if opts == nil {
		return nil
	}
	if v, ok := opts["window"]; ok {
		switch val := v.(type) {
		case string:
			if d, err := time.ParseDuration(val); err == nil && d > 0 {
				c.maxTimeGap = d
			}
		case time.Duration:
			if val > 0 {
				c.maxTimeGap = val
			}
		}
	}
	return nil
}

func (c *Capability) SetMaxTimeGap(d time.Duration) {
	if d > 0 {
		c.maxTimeGap = d
	}
}

// Init 执行 ExifTool 自检 (使用 sync.Once 保证只自检一次)
func (c *Capability) Init(ctx context.Context, report func(domain.PluginInitReport)) error {
	c.initOnce.Do(
		func() {
			if report != nil {
				report(
					domain.PluginInitReport{
						PluginID: c.ID(),
						Name:     c.Name(),
						Stage:    i18n.T("stageInit"),
						Message:  i18n.T("logInterpolateSelfCheckChecking"),
						Percent:  0.5,
						Status:   domain.HealthReady,
					},
				)
			}

			out, err := c.runner.CombinedOutput("exiftool", "-ver")
			if err != nil {
				c.initErr = err
				c.lastReport = domain.PluginInitReport{
					PluginID: c.ID(),
					Name:     c.Name(),
					Stage:    i18n.T("stageInit"),
					Message:  fmt.Sprintf("ExifTool: %v", err),
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
				Message:  fmt.Sprintf(i18n.T("logInterpolateSelfCheckReady"), c.maxTimeGap, ver),
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

// PlanPrecheck 预检资产是否需要且满足插值推算条件
func (c *Capability) PlanPrecheck(ctx context.Context, actx *domain.AssetContext) domain.CapabilityPlan {
	primary := actx.Asset.PrimaryPath()
	if primary == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionInterpolateSkipNoPrimary"),
		}
	}

	meta := actx.GetMetadata()
	if meta.DateTimeOriginal == "" && !actx.IsMetadataLoaded() {
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

	if meta.DateTimeOriginal == "" {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionInterpolateSkipMissingTime"),
			Warning:    "未检测到 DateTimeOriginal 拍摄时间",
		}
	}

	// 若已有有效 GPS 坐标，则无需重复插值推算
	if meta.HasGPS() || actx.HasGPS {
		return domain.CapabilityPlan{
			CanProcess: false,
			ActionDesc: i18n.T("actionInterpolateSkipHasGPS"),
		}
	}

	return domain.CapabilityPlan{
		CanProcess: true,
		ActionDesc: fmt.Sprintf(i18n.T("actionInterpolateEstimate"), c.maxTimeGap),
	}
}

// GPSAnchor 封装具备 GPS 坐标的基准锚点
type GPSAnchor struct {
	Time     time.Time
	DateKey  string
	Lat      float64
	Lon      float64
	Alt      float64
	BaseName string
}

// AnchorIndex 维护按拍摄时间有序排序的 GPS 空间时间索引
type AnchorIndex struct {
	mu           sync.RWMutex
	allAnchors   []GPSAnchor            // 全批次按时间升序排列
	dailyAnchors map[string][]GPSAnchor // 按 YYYY-MM-DD 日期分桶的有序切片
}

// buildAnchorIndex 从批次中一次性构建日期分桶时间索引 (极速批量提取，秒级建树)
func (c *Capability) buildAnchorIndex(batch []*domain.AssetContext, sendEvent func(domain.ProgressEvent)) *AnchorIndex {
	idx := &AnchorIndex{
		dailyAnchors: make(map[string][]GPSAnchor),
	}

	if len(batch) == 0 {
		return idx
	}

	if sendEvent != nil && len(batch) > 100 {
		sendEvent(
			domain.ProgressEvent{
				Stage:   c.RequiredStage(),
				Level:   domain.LevelInfo,
				Message: fmt.Sprintf(i18n.T("logInterpolateBuildingIndex"), len(batch)),
			},
		)
	}

	// 1. 收集尚未读取元数据的主文件路径
	var missingPaths []string
	pathMap := make(map[string]*domain.AssetContext)
	for _, actx := range batch {
		if actx == nil {
			continue
		}
		primary := actx.Asset.PrimaryPath()
		if primary == "" {
			continue
		}
		cleanPath := filepath.Clean(primary)
		pathMap[cleanPath] = actx
		if actx.GetMetadata().DateTimeOriginal == "" {
			missingPaths = append(missingPaths, primary)
		}
	}

	// 2. 批量一次性读取元数据并回填至 AssetContext
	if len(missingPaths) > 0 {
		metaMap, err := exiftool.ReadBatchMetadataMap(c.runner, missingPaths)
		if err == nil {
			for cleanP, meta := range metaMap {
				if actx, ok := pathMap[cleanP]; ok {
					actx.UpdateMetadata(meta)
				}
			}
		}
	}

	// 3. 构建锚点列表
	for _, actx := range batch {
		if actx == nil {
			continue
		}

		lat, lon, alt, ok := extractGPSFromContext(c.runner, actx)
		if !ok {
			continue
		}

		meta := actx.GetMetadata()
		if meta.DateTimeOriginal == "" {
			continue
		}

		t, err := parseExifTime(meta.DateTimeOriginal)
		if err != nil {
			continue
		}

		dateKey := t.Format("2006-01-02")
		anchor := GPSAnchor{
			Time:     t,
			DateKey:  dateKey,
			Lat:      lat,
			Lon:      lon,
			Alt:      alt,
			BaseName: actx.Asset.BaseName,
		}

		idx.allAnchors = append(idx.allAnchors, anchor)
	}

	// 按拍摄时间升序排序
	slices.SortFunc(
		idx.allAnchors, func(a, b GPSAnchor) int {
			return a.Time.Compare(b.Time)
		},
	)

	// 构建按日期分桶的有序列表
	for _, a := range idx.allAnchors {
		idx.dailyAnchors[a.DateKey] = append(idx.dailyAnchors[a.DateKey], a)
	}

	return idx
}

// findNearestAnchors 利用二分查找在 O(log K) 时间内精准定位前后时间最近的锚点
func (idx *AnchorIndex) findNearestAnchors(targetTime time.Time, maxGap time.Duration) (before, after *GPSAnchor) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	dateKey := targetTime.Format("2006-01-02")
	candidates := idx.dailyAnchors[dateKey]
	// 如果当天无任何带 GPS 的锚点，退回全量时间轴查找（跨天跨午夜容错）
	if len(candidates) == 0 {
		candidates = idx.allAnchors
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// 二分查找第一个 Time >= targetTime 的位置
	pos := sort.Search(
		len(candidates), func(i int) bool {
			return !candidates[i].Time.Before(targetTime)
		},
	)

	// 1. 检查前置锚点 (pos-1)
	if pos > 0 {
		cand := candidates[pos-1]
		if targetTime.Sub(cand.Time) <= maxGap {
			before = &cand
		}
	}

	// 2. 检查后置锚点 (pos)
	if pos < len(candidates) {
		cand := candidates[pos]
		if cand.Time.Equal(targetTime) {
			// 精确时间匹配
			before = &cand
			after = &cand
			return before, after
		}
		if cand.Time.Sub(targetTime) <= maxGap {
			after = &cand
		}
	}

	return before, after
}

// addAnchor 将新推算成功的 GPS 锚点动态合入索引，供后续同批次照片继承
func (idx *AnchorIndex) addAnchor(anchor GPSAnchor) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 插入全量有序切片
	pos := sort.Search(
		len(idx.allAnchors), func(i int) bool {
			return !idx.allAnchors[i].Time.Before(anchor.Time)
		},
	)
	idx.allAnchors = append(idx.allAnchors, GPSAnchor{})
	copy(idx.allAnchors[pos+1:], idx.allAnchors[pos:])
	idx.allAnchors[pos] = anchor

	// 插入日期分桶切片
	daily := idx.dailyAnchors[anchor.DateKey]
	dPos := sort.Search(
		len(daily), func(i int) bool {
			return !daily[i].Time.Before(anchor.Time)
		},
	)
	daily = append(daily, GPSAnchor{})
	copy(daily[dPos+1:], daily[dPos:])
	daily[dPos] = anchor
	idx.dailyAnchors[anchor.DateKey] = daily
}

// ExecuteProcess 根据同批次前后邻近照片时间差做双向或单向插值推算
func (c *Capability) ExecuteProcess(
	ctx context.Context, actx *domain.AssetContext, sendEvent func(domain.ProgressEvent),
) error {
	primary := actx.Asset.PrimaryPath()
	if primary == "" {
		return nil
	}

	meta := actx.GetMetadata()
	if meta.DateTimeOriginal == "" {
		m, err := exiftool.ReadMetadata(c.runner, primary)
		if err != nil {
			return fmt.Errorf("读取主文件元数据失败: %w", err)
		}
		meta = m
		actx.UpdateMetadata(meta)
	}

	targetTime, err := parseExifTime(meta.DateTimeOriginal)
	if err != nil {
		return fmt.Errorf("解析照片拍摄时间失败 [%s]: %w", meta.DateTimeOriginal, err)
	}

	// 1. 获取或构建当前批次的快速时间索引
	var anchorIdx *AnchorIndex
	c.indexMu.Lock()
	if c.anchorIndex == nil {
		c.anchorIndex = c.buildAnchorIndex(actx.Batch, sendEvent)
	}
	anchorIdx = c.anchorIndex
	c.indexMu.Unlock()

	// 2. 在 O(log K) 时间内二分查找前后最邻近锚点
	beforeAnchor, afterAnchor := anchorIdx.findNearestAnchors(targetTime, c.maxTimeGap)

	var targetLat, targetLon, targetAlt float64
	var inferMethod string

	switch {
	case beforeAnchor != nil && afterAnchor != nil:
		if beforeAnchor == afterAnchor || beforeAnchor.Time.Equal(afterAnchor.Time) {
			targetLat = beforeAnchor.Lat
			targetLon = beforeAnchor.Lon
			targetAlt = beforeAnchor.Alt
			inferMethod = fmt.Sprintf("同时刻精确匹配 (%s)", beforeAnchor.BaseName)
		} else {
			// 双向时间权重线性插值
			totalDuration := afterAnchor.Time.Sub(beforeAnchor.Time).Seconds()
			if totalDuration <= 0 {
				targetLat = beforeAnchor.Lat
				targetLon = beforeAnchor.Lon
				targetAlt = beforeAnchor.Alt
			} else {
				weight := targetTime.Sub(beforeAnchor.Time).Seconds() / totalDuration
				targetLat = beforeAnchor.Lat + weight*(afterAnchor.Lat-beforeAnchor.Lat)
				targetLon = beforeAnchor.Lon + weight*(afterAnchor.Lon-beforeAnchor.Lon)
				targetAlt = beforeAnchor.Alt + weight*(afterAnchor.Alt-beforeAnchor.Alt)
			}
			inferMethod = fmt.Sprintf(
				"双向时间插值 (前置: %s [%.0fs 前], 后置: %s [%.0fs 后])",
				beforeAnchor.BaseName, targetTime.Sub(beforeAnchor.Time).Seconds(),
				afterAnchor.BaseName, afterAnchor.Time.Sub(targetTime).Seconds(),
			)
		}

	case beforeAnchor != nil:
		// 仅有前置锚点 (同机位/就近继承)
		targetLat = beforeAnchor.Lat
		targetLon = beforeAnchor.Lon
		targetAlt = beforeAnchor.Alt
		inferMethod = fmt.Sprintf(
			"前置近邻推断 (参考: %s [%.0fs 前])",
			beforeAnchor.BaseName, targetTime.Sub(beforeAnchor.Time).Seconds(),
		)

	case afterAnchor != nil:
		// 仅有后置锚点 (同机位/就近继承)
		targetLat = afterAnchor.Lat
		targetLon = afterAnchor.Lon
		targetAlt = afterAnchor.Alt
		inferMethod = fmt.Sprintf(
			"后置近邻推断 (参考: %s [%.0fs 后])",
			afterAnchor.BaseName, afterAnchor.Time.Sub(targetTime).Seconds(),
		)

	default:
		if c.allowNoGPS {
			if sendEvent != nil {
				sendEvent(
					domain.ProgressEvent{
						Stage: c.RequiredStage(),
						Level: domain.LevelWarn,
						Message: fmt.Sprintf(
							i18n.T("logInterpolateSkipNoAnchor"), c.maxTimeGap,
							actx.Asset.DisplayName(),
						),
						Asset: &actx.Asset,
					},
				)
			}
			return nil
		}
		return fmt.Errorf("在时间窗口 %s 内未找到可参考的 GPS 锚点照片", c.maxTimeGap)
	}

	_ = inferMethod
	matchMethod := domain.GPSMatchMethodSphericalLinearInterpol
	if beforeAnchor == nil || afterAnchor == nil {
		matchMethod = domain.GPSMatchMethodNearestNeighborAnchor
	}

	// 3. 构造 GPS 插值溯源指纹
	prov := domain.GPSProvenance{
		Source:      domain.GPSSourceInterpolated,
		MatchMethod: matchMethod,
		Window:      c.maxTimeGap.String(),
		Processor:   domain.DefaultProcessorName,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// 4. 写入 GPS 经纬度（委托给全局统一调度引擎 WriteGPS，严格遵守第二层修正事实规约与四档 SidecarPolicy）
	modifiedFiles, err := exiftool.WriteGPS(
		c.runner, actx.Asset, exiftool.GPSWritePayload{
			Latitude:   targetLat,
			Longitude:  targetLon,
			Altitude:   targetAlt,
			Provenance: prov,
		}, actx.GetPolicy(),
	)
	if err != nil {
		return fmt.Errorf("写入插值 GPS 坐标失败: %w", err)
	}
	for _, f := range modifiedFiles {
		actx.RecordModifiedFile(f)
	}

	// 5. 更新上下文状态并合入索引
	actx.SetGPSWithProvenance(targetLat, targetLon, targetAlt, prov)
	newPos := fmt.Sprintf("%.6f, %.6f", targetLat, targetLon)
	meta.GPSPosition = newPos
	actx.UpdateMetadata(meta)

	// 动态补充新锚点
	anchorIdx.addAnchor(
		GPSAnchor{
			Time:     targetTime,
			DateKey:  targetTime.Format("2006-01-02"),
			Lat:      targetLat,
			Lon:      targetLon,
			Alt:      targetAlt,
			BaseName: actx.Asset.BaseName,
		},
	)

	if sendEvent != nil {
		sendEvent(
			domain.ProgressEvent{
				Stage: c.RequiredStage(),
				Level: domain.LevelInfo,
				Message: fmt.Sprintf(
					i18n.T("logInterpolateSuccess"), actx.Asset.DisplayName(), inferMethod,
					targetLat, targetLon,
				),
				Asset: &actx.Asset,
			},
		)
	}

	return nil
}

func extractGPSFromContext(runner exiftool.CommandRunner, ctx *domain.AssetContext) (lat, lon, alt float64, ok bool) {
	if ctx.HasGPS && (ctx.Latitude != 0 || ctx.Longitude != 0) {
		return ctx.Latitude, ctx.Longitude, ctx.Altitude, true
	}

	meta := ctx.GetMetadata()
	if meta.GPSPosition != "" {
		parsedLat, parsedLon, err := exiftool.ParseCoordinates(meta.GPSPosition)
		if err == nil {
			ctx.SetGPS(parsedLat, parsedLon)
			return parsedLat, parsedLon, ctx.Altitude, true
		}
	}

	// 若尚未读取，从主文件 (RAW 或 JPG) 中读取一次并缓存
	primary := ctx.Asset.PrimaryPath()
	if primary != "" {
		m, err := exiftool.ReadMetadata(runner, primary)
		if err == nil {
			ctx.UpdateMetadata(m)
			if m.GPSPosition != "" {
				parsedLat, parsedLon, err := exiftool.ParseCoordinates(m.GPSPosition)
				if err == nil {
					ctx.SetGPS(parsedLat, parsedLon)
					return parsedLat, parsedLon, ctx.Altitude, true
				}
			}
		}
	}

	return 0, 0, 0, false
}

func parseExifTime(timeStr string) (time.Time, error) {
	clean := strings.TrimSpace(timeStr)
	layouts := []string{
		"2006:01:02 15:04:05-07:00",
		"2006:01:02 15:04:05+07:00",
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, clean); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法识别的时间格式: %s", timeStr)
}
