package domain

import (
	"context"
	"sync"
)

// CapabilityID 能力插件唯一标识
type CapabilityID string

const (
	CapGPXMatching    CapabilityID = "gpx_matching"    // 能力 1: GPX 轨迹匹配与 GPS 修正
	CapGPSInterpolate CapabilityID = "gps_interpolate" // 能力 1.5: GPS 智能邻近推断与时间插值
	CapReverseGeocode CapabilityID = "reverse_geocode" // 能力 2: 逆地理编码写入元数据
	CapDateArchive    CapabilityID = "date_archive"    // 能力 3: 按拍摄日期规范化归档
)

// PluginHealthStatus 插件健康自检状态
type PluginHealthStatus string

const (
	HealthReady    PluginHealthStatus = "ready"    // 正常就绪
	HealthDegraded PluginHealthStatus = "degraded" // 降级运行（如未安装离线数据包，降级到内置轻量库）
	HealthFailed   PluginHealthStatus = "failed"   // 初始化失败（如关键命令缺失）
)

// PluginInitReport 插件初始化过程进度汇报
type PluginInitReport struct {
	PluginID CapabilityID       `json:"plugin_id"`
	Name     string             `json:"name"`
	Stage    string             `json:"stage"`   // 当前步骤说明（如 "扫描离线数据包", "构建 KD-Tree"）
	Message  string             `json:"message"` // 详细提示信息
	Percent  float64            // 0.0 ~ 1.0 (-1 表示不确定进度)
	Status   PluginHealthStatus // 当前健康状态
	Err      error              // 错误信息（若有）
}

// AssetContext 单个拍摄单元在流水线各阶段流转时的共享上下文
type AssetContext struct {
	mu sync.RWMutex

	// 基础资产组
	Asset AssetGroup

	// 缓存的元数据与解析结果（避免多次重复调用 exiftool 读盘）
	Metadata       Metadata
	MetadataLoaded bool
	HasGPS         bool
	Latitude       float64
	Longitude      float64
	Altitude       float64
	Location       *LocationInfo
	TargetDir      string
	NewBaseName    string

	// 执行标记与修改文件记录
	ModifiedFiles []string
	Skipped       bool
	SkipReason    string

	// 全批次共享上下文指针（用于多照片时间序列推断与插值）
	Batch []*AssetContext
}

// NewAssetContext 初始化资产上下文
func NewAssetContext(asset AssetGroup) *AssetContext {
	return &AssetContext{
		Asset: asset,
	}
}

// GetMetadata 安全读取元数据
func (c *AssetContext) GetMetadata() Metadata {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Metadata
}

// IsMetadataLoaded 判断元数据是否已经经过预装载/读取
func (c *AssetContext) IsMetadataLoaded() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MetadataLoaded
}

// UpdateMetadata 安全更新元数据与 GPS 状态
func (c *AssetContext) UpdateMetadata(meta Metadata) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metadata = meta
	c.MetadataLoaded = true
	if meta.GPSPosition != "" {
		c.HasGPS = true
	}
}

// SetGPS 设置经纬度
func (c *AssetContext) SetGPS(lat, lon float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Latitude = lat
	c.Longitude = lon
	c.HasGPS = true
}

// SetLocation 设置逆地理位置信息
func (c *AssetContext) SetLocation(loc *LocationInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Location = loc
}

// RecordModifiedFile 记录被修改的文件
func (c *AssetContext) RecordModifiedFile(file string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ModifiedFiles = append(c.ModifiedFiles, file)
}

// OptionScope 标识设置项的作用范围
type OptionScope string

const (
	ScopeGlobal OptionScope = "global" // 全局运行与环境配置
	ScopePlugin OptionScope = "plugin" // 插件专属算法/行为配置
)

// OptionType 配置项数据类型
type OptionType string

const (
	OptionTypeString   OptionType = "string"
	OptionTypeDuration OptionType = "duration"
	OptionTypeBool     OptionType = "bool"
	OptionTypeInt      OptionType = "int"
	OptionTypeChoice   OptionType = "choice"
)

// OptionSpec 插件可配置项的自描述契约规范
type OptionSpec struct {
	Key          string                 `json:"key"`         // 选项标识，如 "window", "geosync", "in_place"
	Name         string                 `json:"name"`        // 中文名称，如 "推算最大时间窗口"
	Description  string                 `json:"description"` // 详细功能说明
	Type         OptionType             `json:"type"`        // 数据类型
	Scope        OptionScope            `json:"scope,omitempty"`
	PluginID     CapabilityID           `json:"plugin_id,omitempty"`
	DefaultValue any                    `json:"default_value"`     // 默认值
	Choices      []string               `json:"choices,omitempty"` // 预设快速切换项，如 ["15m", "30m", "1h", "2h"]
	Validate     func(val string) error `json:"-"`
}

// CapabilityPlan 预检阶段评估结果
type CapabilityPlan struct {
	CanProcess bool   // 是否满足前置条件
	ActionDesc string // 预定动作描述
	Warning    string // 预警信息
}

// Capability 核心能力插件接口规范 (支持内聚自描述与动态配置注入)
type Capability interface {
	// ID 返回能力唯一标识
	ID() CapabilityID
	// Name 返回能力中文名称
	Name() string
	// Description 返回能力详细描述
	Description() string
	// RequiredStage 返回该能力对应的流水线阶段标识
	RequiredStage() PipelineStage
	// Priority 返回能力执行优先级（数值越小越优先执行；同优先级并行/同阶段执行，不同优先级分阶段串行执行）
	Priority() int

	// SupportedOptions 🌟 插件自声明支持的可配置项清单 (自描述 Schema)
	SupportedOptions() []OptionSpec
	// Configure 🌟 动态注入用户或会话级别的配置选项键值对
	Configure(options map[string]any) error

	// Init 执行能力自检与渐进式初始化，通过 report 回调向外部汇报进度与阶段
	Init(ctx context.Context, report func(PluginInitReport)) error

	// PlanPrecheck 预检单个资产，评估是否可处理
	PlanPrecheck(ctx context.Context, actx *AssetContext) CapabilityPlan
	// ExecuteProcess 执行具体业务逻辑（如写入 Exif、查询地名、移动归档）
	ExecuteProcess(ctx context.Context, actx *AssetContext, sendEvent func(ProgressEvent)) error
}
