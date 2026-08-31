package common

import (
	"os"
	"path/filepath"
	"time"
)

// ==========================================
// 1. 核心能力插件标识与健康状态
// ==========================================

// CapabilityID 能力插件唯一标识
type CapabilityID string

const (
	CapGPXMatching    CapabilityID = "gpx_matching"    // 能力 1: GPX 轨迹匹配与 GPS 修正
	CapGPSInterpolate CapabilityID = "gps_interpolate" // 能力 1.5: GPS 智能邻近推断与时间插值
	CapReverseGeocode CapabilityID = "reverse_geocode" // 能力 2: 逆地理编码写入元数据
	CapDateArchive    CapabilityID = "date_archive"    // 能力 3: 按拍摄日期规范化归档
)

// String 返回能力标识的字符串
func (c CapabilityID) String() string {
	return string(c)
}

// PluginHealthStatus 插件健康自检状态
type PluginHealthStatus string

const (
	HealthReady    PluginHealthStatus = "ready"    // 正常就绪
	HealthDegraded PluginHealthStatus = "degraded" // 降级运行（如未安装离线数据包，降级到内置轻量库）
	HealthFailed   PluginHealthStatus = "failed"   // 初始化失败（如关键命令缺失）
)

// String 返回健康状态的字符串
func (s PluginHealthStatus) String() string {
	return string(s)
}

// ==========================================
// 2. 元数据分层模型与写入意图
// ==========================================

// MetadataWriteIntent 元数据写入意图分类（四层元数据分层模型）
type MetadataWriteIntent int

const (
	// IntentOriginalFact 第一层：原始摄影事实（相机曝光、快门、镜头、原始机身 GPS 等，只读保留）
	IntentOriginalFact MetadataWriteIntent = iota
	// IntentCorrectedFact 第二层：修正摄影事实（GPX 匹配 GPS、时间插值 GPS、时区与时间偏移修正）
	IntentCorrectedFact
	// IntentDerivedInfo 第三层：派生地理信息（离线逆地理地名、国家/省/市/区、Lightroom 分层关键词树、IPTC Extension）
	IntentDerivedInfo
	// IntentWorkflowData 第四层：工作流与主观数据（评分、色彩标签、AI 标签、标题描述）
	IntentWorkflowData
)

// SidecarPolicy 侧车写入策略
type SidecarPolicy string

const (
	// PolicySmart 智能分层模式（默认/推荐）：修正事实写入 RAW EXIF + JPG 内嵌 + XMP 同步；派生信息严格不修改 RAW，仅写入 .xmp 侧车 + JPG 内嵌
	PolicySmart SidecarPolicy = "smart"
	// PolicyReadOnly 向后兼容别名（等价于 PolicySmart）
	PolicyReadOnly SidecarPolicy = "read_only"
	// PolicySidecarOnly 纯 XMP 侧车模式：RAW 与 JPG 均不触碰原图，所有事实修正与派生信息严格写入 .xmp 侧车
	PolicySidecarOnly SidecarPolicy = "sidecar_only"
	// PolicyEmbedAndSidecar 原图与 XMP 双写同步：RAW 与 JPG 均内嵌修改全部元数据，同时维护 .xmp 侧车
	PolicyEmbedAndSidecar SidecarPolicy = "embed_and_sidecar"
	// PolicyEmbedOnly 纯原图内嵌模式：直接内嵌修改 RAW 与 JPG，不产生任何 .xmp 侧车
	PolicyEmbedOnly SidecarPolicy = "embed_only"
)

// String 返回侧车策略字符串
func (p SidecarPolicy) String() string {
	return string(p)
}

// IsValid 检查侧车策略枚举值是否合法
func (p SidecarPolicy) IsValid() bool {
	switch p {
	case PolicySmart, PolicyReadOnly, PolicySidecarOnly, PolicyEmbedAndSidecar, PolicyEmbedOnly:
		return true
	default:
		return false
	}
}

// ==========================================
// 3. GPS 来源、推算算法与地名来源
// ==========================================

// GPSSourceType 定义 GPS 坐标来源的公共枚举
type GPSSourceType string

const (
	// GPSSourceCamera 相机原生机身 GPS 记录 (例如机身内置 GPS 或 SnapBridge 蓝牙写入)
	GPSSourceCamera GPSSourceType = "camera"
	// GPSSourceGPX 通过 GPX 轨迹文件时间轴精准匹配写入/修正
	GPSSourceGPX GPSSourceType = "gpx"
	// GPSSourceInterpolated 通过时间权重与邻近锚点照片智能插值推算补全
	GPSSourceInterpolated GPSSourceType = "interpolated"
	// GPSSourceManual 人工手动指定的经纬度坐标
	GPSSourceManual GPSSourceType = "manual"
)

// String 返回枚举的字符串表示
func (s GPSSourceType) String() string {
	return string(s)
}

// IsValid 检查 GPS 来源枚举值是否合法
func (s GPSSourceType) IsValid() bool {
	switch s {
	case GPSSourceCamera, GPSSourceGPX, GPSSourceInterpolated, GPSSourceManual:
		return true
	default:
		return false
	}
}

// GPSMatchMethodType 定义 GPS 匹配与推算算法的公共枚举
type GPSMatchMethodType string

const (
	// GPSMatchMethodNativeCamera 原生机身传感器记录
	GPSMatchMethodNativeCamera GPSMatchMethodType = "native_camera"
	// GPSMatchMethodTimeProximity 基于 GPX 时间轴的最邻近航点插值匹配
	GPSMatchMethodTimeProximity GPSMatchMethodType = "time_proximity"
	// GPSMatchMethodSphericalLinearInterpol 双向前后锚点球面大圆时间权重线性插值
	GPSMatchMethodSphericalLinearInterpol GPSMatchMethodType = "spherical_linear_interpolation"
	// GPSMatchMethodNearestNeighborAnchor 单向同机位前后最近邻近锚点继承
	GPSMatchMethodNearestNeighborAnchor GPSMatchMethodType = "nearest_neighbor_anchor"
)

// String 返回枚举的字符串表示
func (m GPSMatchMethodType) String() string {
	return string(m)
}

// IsValid 检查 GPS 匹配算法枚举值是否合法
func (m GPSMatchMethodType) IsValid() bool {
	switch m {
	case GPSMatchMethodNativeCamera, GPSMatchMethodTimeProximity,
		GPSMatchMethodSphericalLinearInterpol, GPSMatchMethodNearestNeighborAnchor:
		return true
	default:
		return false
	}
}

// GeocodeSourceType 逆地理地名元数据来源枚举
type GeocodeSourceType string

const (
	// GeocodeSourceEmbedded 来自原图内嵌 IPTC/EXIF
	GeocodeSourceEmbedded GeocodeSourceType = "embedded"
	// GeocodeSourceXMP 来自伴随的 XMP 侧车文件
	GeocodeSourceXMP GeocodeSourceType = "xmp"
)

// String 返回地名来源字符串表示
func (g GeocodeSourceType) String() string {
	return string(g)
}

// ==========================================
// 4. 事件级别与流水线执行阶段
// ==========================================

// EventLevel 定义事件严重级别
type EventLevel string

const (
	LevelInfo    EventLevel = "info"
	LevelSuccess EventLevel = "success"
	LevelWarn    EventLevel = "warn"
	LevelError   EventLevel = "error"
)

// String 返回事件级别字符串
func (l EventLevel) String() string {
	return string(l)
}

// PipelineStage 定义流水线所处的阶段标识
type PipelineStage string

const (
	StageInit        PipelineStage = "环境自检"
	StageDiscover    PipelineStage = "扫描资产"
	StagePrecheck    PipelineStage = "预检校验"
	StageGeotag      PipelineStage = "写入GPS"
	StageInterpolate PipelineStage = "智能推算"
	StageGeocode     PipelineStage = "逆地理编码"
	StageSync        PipelineStage = "同步附属文件"
	StageArchive     PipelineStage = "归档重命名"
	StageBackup      PipelineStage = "快照备份"
	StageRestore     PipelineStage = "快照还原"
	StageSummary     PipelineStage = "流水线汇总"
	StageComplete    PipelineStage = "任务完成"
)

// String 返回阶段名称字符串
func (s PipelineStage) String() string {
	return string(s)
}

// ==========================================
// 5. 问题与异常分类
// ==========================================

// IssueKind 定义待处理项或失败分类
type IssueKind string

const (
	IssueKindMissingPair IssueKind = "missing_pair" // 缺少同名配对 JPG
	IssueKindTrackGap    IssueKind = "track_gap"    // 拍摄时间未在 GPX 轨迹范围内
	IssueKindFailure     IssueKind = "failure"      // 元数据解析或写入失败
	IssueKindMissingDate IssueKind = "missing_date" // 无法读取拍摄日期
	IssueKindConflict    IssueKind = "conflict"     // 归档目标目录已存在同名冲突文件
)

// String 返回问题类型字符串
func (k IssueKind) String() string {
	return string(k)
}

// ==========================================
// 6. 全局系统常量 (严禁魔法值)
// ==========================================

const (
	// CurrentVersion 当前发布版本号
	CurrentVersion = "v0.0.2"
	// DefaultProcessorName photools 全局处理引擎名称与版本标识
	DefaultProcessorName = "photools " + CurrentVersion
	// DefaultWorkers 默认并发协程数
	DefaultWorkers = 8
	// DefaultGeosync 默认时间同步修正偏移
	DefaultGeosync = "0"
	// DefaultInterpolateWindowStr 默认时间推算窗口字符串
	DefaultInterpolateWindowStr = "15m"
	// DefaultInterpolateWindow 默认时间推算窗口 Duration
	DefaultInterpolateWindow = 15 * time.Minute
	// DefaultLogDirSubPath 默认全局日志子路径
	DefaultLogDirSubPath = ".logs/photools"
	// LogFileNameLatest 最新实时日志文件名
	LogFileNameLatest = "photools_latest.log"
	// PendingReportFileNameLatest 最新待补清单报告文件名
	PendingReportFileNameLatest = "inbox_pending_report_latest.md"
)

// GetDefaultLogDir 获取默认的全局日志目录 (~/.logs/photools)
func GetDefaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".logs", "photools")
	}
	return filepath.Join(home, ".logs", "photools")
}

// DefaultCompanionExtensions 默认伴随文件扩展名列表
var DefaultCompanionExtensions = []string{"wav", "acr", "exf", "xmp"}

// DefaultRawExtensions 默认支持的相机 RAW 文件扩展名列表
var DefaultRawExtensions = []string{"nef", "nrw", "cr2", "cr3", "arw", "dng", "raf", "orf", "rw2", "pef"}
