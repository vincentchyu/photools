package domain

import (
	"context"
	"strings"
	"sync"

	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/i18n"
)

// CapabilityID 能力插件唯一标识 (指向 common.CapabilityID)
type CapabilityID = common.CapabilityID

const (
	CapGPXMatching    = common.CapGPXMatching
	CapGPSInterpolate = common.CapGPSInterpolate
	CapReverseGeocode = common.CapReverseGeocode
	CapDateArchive    = common.CapDateArchive
)

// PluginHealthStatus 插件健康自检状态 (指向 common.PluginHealthStatus)
type PluginHealthStatus = common.PluginHealthStatus

const (
	HealthReady    = common.HealthReady
	HealthDegraded = common.HealthDegraded
	HealthFailed   = common.HealthFailed
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

// MetadataWriteIntent 元数据写入意图分类（指向 common.MetadataWriteIntent）
type MetadataWriteIntent = common.MetadataWriteIntent

const (
	IntentOriginalFact  = common.IntentOriginalFact
	IntentCorrectedFact = common.IntentCorrectedFact
	IntentDerivedInfo   = common.IntentDerivedInfo
	IntentWorkflowData  = common.IntentWorkflowData
)

// GPSSourceType GPS 坐标来源枚举 (指向 common.GPSSourceType)
type GPSSourceType = common.GPSSourceType

const (
	GPSSourceCamera       = common.GPSSourceCamera
	GPSSourceGPX          = common.GPSSourceGPX
	GPSSourceInterpolated = common.GPSSourceInterpolated
	GPSSourceManual       = common.GPSSourceManual
)

// GPSMatchMethodType GPS 匹配与推算算法枚举 (指向 common.GPSMatchMethodType)
type GPSMatchMethodType = common.GPSMatchMethodType

const (
	GPSMatchMethodNativeCamera            = common.GPSMatchMethodNativeCamera
	GPSMatchMethodTimeProximity           = common.GPSMatchMethodTimeProximity
	GPSMatchMethodSphericalLinearInterpol = common.GPSMatchMethodSphericalLinearInterpol
	GPSMatchMethodNearestNeighborAnchor   = common.GPSMatchMethodNearestNeighborAnchor
)

// DefaultProcessorName 默认处理引擎名称与版本标识
var DefaultProcessorName = common.DefaultProcessorName

// GPSProvenance 记录 GPS 坐标的清洗与修正溯源指纹
type GPSProvenance struct {
	Source      GPSSourceType      `json:"source"`       // 来源: camera, gpx, interpolated, manual
	MatchMethod GPSMatchMethodType `json:"match_method"` // 推算方法: native_camera, time_proximity, spherical_linear_interpolation, nearest_neighbor_anchor
	Window      string             `json:"window"`       // 推算窗口: 15m, 30m, 1h
	Processor   string             `json:"processor"`    // 处理引擎: photools v1.2.0
	Timestamp   string             `json:"timestamp"`    // 处理时间戳 (ISO 8601)
}

// SidecarPolicy 侧车写入策略 (指向 common.SidecarPolicy)
type SidecarPolicy = common.SidecarPolicy

const (
	PolicySmart           = common.PolicySmart
	PolicyReadOnly        = common.PolicyReadOnly
	PolicySidecarOnly     = common.PolicySidecarOnly
	PolicyEmbedAndSidecar = common.PolicyEmbedAndSidecar
	PolicyEmbedOnly       = common.PolicyEmbedOnly
)

// NormalizePolicy 规整化策略输入（支持别名与向后兼容）
func NormalizePolicy(policy string) SidecarPolicy {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case string(PolicySidecarOnly), "sidecar":
		return PolicySidecarOnly
	case string(PolicyEmbedAndSidecar), "dual":
		return PolicyEmbedAndSidecar
	case string(PolicyEmbedOnly), "embed":
		return PolicyEmbedOnly
	case string(PolicySmart), string(PolicyReadOnly), "smart_read_only", "":
		return PolicySmart
	default:
		return PolicySmart
	}
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
	Provenance     *GPSProvenance
	Location       *LocationInfo
	TargetDir      string
	NewBaseName    string

	// 执行策略与修改文件记录
	SidecarPolicy SidecarPolicy // 侧车写入策略 (默认 PolicySmart)
	SidecarOnly   bool          // 保持向后兼容标志 (等价于 SidecarPolicy == PolicySidecarOnly)
	ModifiedFiles []string
	Skipped       bool
	SkipReason    string

	// 全批次共享上下文指针（用于多照片时间序列推断与插值）
	Batch []*AssetContext
}

// GetPolicy 返回当前生效的侧车策略（自动兼容 SidecarOnly 标志）
func (c *AssetContext) GetPolicy() SidecarPolicy {
	if c.SidecarOnly {
		return PolicySidecarOnly
	}
	if c.SidecarPolicy != "" {
		return NormalizePolicy(string(c.SidecarPolicy))
	}
	return PolicySmart
}

// ShouldEmbed 基于元数据意图判断是否应当直接修改内嵌元数据
func (c *AssetContext) ShouldEmbed(intent MetadataWriteIntent, isRaw bool) bool {
	policy := c.GetPolicy()
	switch policy {
	case PolicySidecarOnly:
		return false
	case PolicyEmbedAndSidecar, PolicyEmbedOnly:
		return true
	case PolicySmart, PolicyReadOnly:
		if !isRaw {
			// 非 RAW (如 JPG 交付格式) 始终直接内嵌全部元数据
			return true
		}
		// 对 RAW 主文件：仅允许修正事实 (如 GPS 经纬度/时间偏移) 写入 EXIF 头部，派生信息与工作流绝不触碰 RAW
		return intent == IntentCorrectedFact
	default:
		return !isRaw || intent == IntentCorrectedFact
	}
}

// ShouldWriteSidecar 基于元数据意图判断是否应当生成/更新独立 XMP 侧车文件
func (c *AssetContext) ShouldWriteSidecar(intent MetadataWriteIntent, isRaw bool) bool {
	policy := c.GetPolicy()
	switch policy {
	case PolicySidecarOnly, PolicyEmbedAndSidecar:
		return true
	case PolicyEmbedOnly:
		return false
	case PolicySmart, PolicyReadOnly:
		// Smart 模式下：RAW 资产的所有修正与派生元数据均同步写入 XMP 侧车
		return isRaw
	default:
		return isRaw
	}
}

// ShouldWriteSidecarFor 判断对于指定文件是否应该生成/更新独立 XMP 侧车文件 (向后兼容)
func (c *AssetContext) ShouldWriteSidecarFor(isRaw bool) bool {
	return c.ShouldWriteSidecar(IntentCorrectedFact, isRaw)
}

// ShouldEmbedMetadataFor 判断对于指定文件是否应该直接修改其内嵌 EXIF/IPTC (向后兼容)
func (c *AssetContext) ShouldEmbedMetadataFor(isRaw bool) bool {
	return c.ShouldEmbed(IntentCorrectedFact, isRaw)
}

// NewAssetContext 初始化资产上下文
func NewAssetContext(asset AssetGroup) *AssetContext {
	return &AssetContext{
		Asset:         asset,
		SidecarPolicy: PolicyReadOnly,
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

// SetGPSWithProvenance 设置经纬度及溯源指纹信息
func (c *AssetContext) SetGPSWithProvenance(lat, lon, alt float64, prov GPSProvenance) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Latitude = lat
	c.Longitude = lon
	c.Altitude = alt
	c.HasGPS = true
	c.Provenance = &prov
}

// GetProvenance 安全读取溯源指纹
func (c *AssetContext) GetProvenance() *GPSProvenance {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Provenance
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
	Key          string                 `json:"key"`                // 选项标识，如 "window", "geosync", "in_place"
	NameKey      string                 `json:"name_key,omitempty"` // i18n 字典键
	DescKey      string                 `json:"desc_key,omitempty"` // i18n 字典键
	Type         OptionType             `json:"type"`               // 数据类型
	Scope        OptionScope            `json:"scope,omitempty"`
	PluginID     CapabilityID           `json:"plugin_id,omitempty"`
	DefaultValue any                    `json:"default_value"`     // 默认值
	Choices      []string               `json:"choices,omitempty"` // 预设快速切换项，如 ["15m", "30m", "1h", "2h"]
	Validate     func(val string) error `json:"-"`
}

// DisplayName 返回根据当前语言动态匹配的名称 (100% 字典驱动)
func (o OptionSpec) DisplayName() string {
	if o.NameKey != "" {
		return i18n.T(o.NameKey)
	}
	return o.Key
}

// DisplayDescription 返回根据当前语言动态匹配的详细说明 (100% 字典驱动)
func (o OptionSpec) DisplayDescription() string {
	if o.DescKey != "" {
		return i18n.T(o.DescKey)
	}
	return ""
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
