package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/domain"
)

// OptionScope 标识设置项的作用范围
type OptionScope = domain.OptionScope

const (
	ScopeGlobal = domain.ScopeGlobal // 全局运行与环境配置
	ScopePlugin = domain.ScopePlugin // 插件专属算法/行为配置
)

// OptionType 标识设置项的数据类型 (与 domain 统一)
type OptionType = domain.OptionType

const (
	TypeString   = domain.OptionTypeString
	TypeDuration = domain.OptionTypeDuration
	TypeBool     = domain.OptionTypeBool
	TypeInt      = domain.OptionTypeInt
	TypeChoice   = domain.OptionTypeChoice
)

// OptionSpec 描述一个配置项的元数据、校验规则与默认值 (与 domain 统一)
type OptionSpec = domain.OptionSpec

// GlobalSettings 封装整条流水线的全局环境与调度配置
type GlobalSettings struct {
	BaseDir             string   `json:"base_dir"`
	SourceDir           string   `json:"source_dir"`
	TargetDir           string   `json:"target_dir"`
	GPXDir              string   `json:"gpx_dir"`              // GPX 轨迹目录 (默认 ~/.config/gpx)
	LogDir              string   `json:"log_dir"`              // 全局日志与报告目录 (默认 ~/.logs/photools)
	FlatMode            bool     `json:"flat_mode"`            // 忽略传统分层目录结构，在指定目录下直接扫描并原地保存/处理
	SidecarPolicy       string   `json:"sidecar_policy"`       // 侧车写入策略: read_only, sidecar_only, embed_and_sidecar, embed_only
	SidecarOnly         bool     `json:"sidecar_only"`         // 仅生成/修改 {file}.xmp 侧车文件 (向后兼容标志)
	CompanionExtensions []string `json:"companion_extensions"` // 伴随文件扩展名白名单 (如 wav, acr, exf)
	RawExtensions       []string `json:"raw_extensions"`       // RAW 格式白名单
	Workers             int      `json:"workers"`              // 并发处理协程数
	AllowNoGPS          bool     `json:"allow_no_gps"`         // 软降级容错：无 GPS 坐标时允许跳过地名写入直接归档
	TestBackup          bool     `json:"test_backup"`          // 处理前是否快照备份源文件
	BackupDir           string   `json:"backup_dir"`           // 自定义备份目录
}

// SessionConfig 封装当前运行时会话的全部配置（全局配置 + 各插件独立配置）
type SessionConfig struct {
	Global  GlobalSettings                    `json:"global"`
	Plugins map[domain.CapabilityID]PluginOpt `json:"plugins"`
}

// PluginOpt 封装单个插件在会话中的专属配置项集合
type PluginOpt struct {
	ID       domain.CapabilityID    `json:"id"`
	Priority int                    `json:"priority"`
	Enabled  bool                   `json:"enabled"`
	Options  map[string]interface{} `json:"options"`
}

// DefaultGPXDir 返回内置默认的全局 GPX 轨迹目录 (~/.config/gpx)
func DefaultGPXDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".config", "gpx")
	}
	return filepath.Join(home, ".config", "gpx")
}

// DefaultLogDir 返回内置默认的全局日志目录 (~/.logs/photools)
func DefaultLogDir() string {
	return common.GetDefaultLogDir()
}

// DefaultGlobalSettings 返回内置默认的全局设置
func DefaultGlobalSettings(baseDir ...string) GlobalSettings {
	bDir := ""
	if len(baseDir) > 0 && baseDir[0] != "" {
		bDir = baseDir[0]
	} else {
		if wd, err := os.Getwd(); err == nil && wd != "" {
			bDir = wd
		} else {
			home, _ := os.UserHomeDir()
			bDir = filepath.Join(home, "Pictures", "GPS")
		}
	}
	bDir, _ = filepath.Abs(bDir)

	return GlobalSettings{
		BaseDir:             bDir,
		SourceDir:           filepath.Join(bDir, "Inbox"),
		TargetDir:           filepath.Join(bDir, "Processed"),
		GPXDir:              DefaultGPXDir(),
		LogDir:              DefaultLogDir(),
		FlatMode:            false,
		SidecarPolicy:       common.PolicySmart.String(),
		SidecarOnly:         false,
		CompanionExtensions: common.DefaultCompanionExtensions,
		RawExtensions:       common.DefaultRawExtensions,
		Workers:             runtime.NumCPU(),
		AllowNoGPS:          false,
		TestBackup:          false,
		BackupDir:           filepath.Join(bDir, "Inbox_bak"),
	}
}

// NewSessionConfig 根据持久化配置和默认值初始化会话配置
func NewSessionConfig(cfg *PluginsConfig, baseDir ...string) *SessionConfig {
	sc := &SessionConfig{
		Global:  DefaultGlobalSettings(baseDir...),
		Plugins: make(map[domain.CapabilityID]PluginOpt),
	}

	if cfg == nil {
		def := DefaultPluginsConfig()
		cfg = &def
	}

	for _, p := range cfg.Plugins {
		optCopy := make(map[string]interface{})
		maps.Copy(optCopy, p.Options)
		sc.Plugins[p.ID] = PluginOpt{
			ID:       p.ID,
			Priority: p.Priority,
			Enabled:  p.Enabled,
			Options:  optCopy,
		}
	}

	return sc
}

// GetStringOption 获取插件字符串配置
func (s *SessionConfig) GetStringOption(pluginID domain.CapabilityID, key, defaultValue string) string {
	if p, ok := s.Plugins[pluginID]; ok && p.Options != nil {
		if v, ok := p.Options[key]; ok {
			if str, ok := v.(string); ok && str != "" {
				return str
			}
		}
	}
	return defaultValue
}

// GetDurationOption 获取插件时长配置 (如 "15m", "1h")
func (s *SessionConfig) GetDurationOption(
	pluginID domain.CapabilityID, key string, defaultValue time.Duration,
) time.Duration {
	str := s.GetStringOption(pluginID, key, "")
	if str == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return defaultValue
	}
	return d
}

// GetBoolOption 获取插件布尔配置
func (s *SessionConfig) GetBoolOption(pluginID domain.CapabilityID, key string, defaultValue bool) bool {
	if p, ok := s.Plugins[pluginID]; ok && p.Options != nil {
		if v, ok := p.Options[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return defaultValue
}

// GetPluginOptions 获取指定插件的全部配置选项字典
func (s *SessionConfig) GetPluginOptions(pluginID domain.CapabilityID) map[string]interface{} {
	if p, ok := s.Plugins[pluginID]; ok && p.Options != nil {
		return p.Options
	}
	return nil
}

// SetPluginOption 设置指定插件的配置项（当前会话生效）
func (s *SessionConfig) SetPluginOption(pluginID domain.CapabilityID, key string, value interface{}) {
	p, ok := s.Plugins[pluginID]
	if !ok {
		p = PluginOpt{
			ID:      pluginID,
			Enabled: true,
			Options: make(map[string]interface{}),
		}
	}
	if p.Options == nil {
		p.Options = make(map[string]interface{})
	}
	p.Options[key] = value
	s.Plugins[pluginID] = p
}

// ApplyToPluginsConfig 将当前会话中的插件优先级与选项同步回持久化配置结构
func (s *SessionConfig) ApplyToPluginsConfig(cfg *PluginsConfig) {
	if cfg == nil {
		return
	}
	for i := range cfg.Plugins {
		p := &cfg.Plugins[i]
		if opt, ok := s.Plugins[p.ID]; ok {
			p.Priority = opt.Priority
			p.Enabled = opt.Enabled
			if p.Options == nil {
				p.Options = make(map[string]interface{})
			}
			for k, v := range opt.Options {
				p.Options[k] = v
			}
		}
	}
}

// GlobalOptionSpecs 返回所有已注册的全局配置项描述
func GlobalOptionSpecs() []OptionSpec {
	return []OptionSpec{
		{
			Key:          "base_dir",
			Name:         "工作区根目录 (BaseDir)",
			Description:  "基础工作目录，包含 Inbox/Processed/Logs 等子目录",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "~/Pictures/GPS",
		},
		{
			Key:          "gpx_dir",
			Name:         "GPX 轨迹目录 (GPXDir)",
			Description:  "GPX 轨迹文件所在目录（默认 ~/.config/gpx）",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "~/.config/gpx",
		},
		{
			Key:          "source_dir",
			Name:         "扫描源目录 (SourceDir)",
			Description:  "照片扫描源目录（默认 <BaseDir>/Inbox 或 Flat 模式下的指定目录）",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "<BaseDir>/Inbox",
		},
		{
			Key:          "flat_mode",
			Name:         "扁平原地模式 (Flat Mode)",
			Description:  "忽略传统 Inbox/Processed 分层结构，直接扫描指定目录所有照片并原地保存/处理",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "sidecar_policy",
			Name:         "元数据与侧车写入策略 (Sidecar Policy)",
			Description:  "控制元数据写入目标：smart(智能分层模式：GPS修正写RAW/JPG/XMP，地名写XMP/JPG，默认推荐)、sidecar_only(纯XMP侧车)、embed_and_sidecar(双写同步)、embed_only(纯原图内嵌)",
			Type:         TypeChoice,
			Choices:      []string{string(domain.PolicySmart), string(domain.PolicySidecarOnly), string(domain.PolicyEmbedAndSidecar), string(domain.PolicyEmbedOnly)},
			Scope:        ScopeGlobal,
			DefaultValue: string(domain.PolicySmart),
		},
		{
			Key:          "sidecar_only",
			Name:         "仅生成 XMP 侧车文件 (Sidecar Only / 兼容选项)",
			Description:  "元数据变更仅写入 {file}.xmp 侧车文件，严格保护不修改原始 RAW/JPG 文件（等价于 sidecar_policy=sidecar_only）",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "companion_extensions",
			Name:         "伴随文件扩展名白名单",
			Description:  "与主照片关联的伴随文件扩展名（如 wav, acr, exf，归档时整体原子移动）",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "wav,acr,exf",
		},
		{
			Key:          "raw_extensions",
			Name:         "RAW 格式白名单",
			Description:  "可识别为主决策权威源的 RAW 格式文件扩展名（逗号分隔）",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "nef,cr3,arw,dng,raf,rw2,orf",
		},
		{
			Key:          "workers",
			Name:         "并发处理协程数",
			Description:  "并发处理拍摄单元的 Worker 数量（默认 CPU 核心数）",
			Type:         TypeInt,
			Scope:        ScopeGlobal,
			DefaultValue: runtime.NumCPU(),
		},
		{
			Key:          "allow_no_gps",
			Name:         "无 GPS 软降级容错",
			Description:  "彻底无 GPS 坐标的照片在逆地理阶段良性跳过，安全进入拍摄日期归档",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "test_backup",
			Name:         "测试安全快照备份",
			Description:  "在流水线执行前，自动将源目录照片全量备份至 Inbox_bak 目录",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
	}
}

// PluginOptionSpecs 返回指定插件的所有专属配置项描述
func PluginOptionSpecs(id domain.CapabilityID) []OptionSpec {
	switch id {
	case domain.CapGPXMatching:
		return []OptionSpec{
			{
				Key:          "geosync",
				Name:         "时钟偏差偏移 (Geosync)",
				Description:  "相机时钟与 GPS 轨迹的时间偏差补偿（如 0, +00:00:05, -00:01:00）",
				Type:         TypeString,
				Scope:        ScopePlugin,
				PluginID:     domain.CapGPXMatching,
				DefaultValue: "0",
			},
		}
	case domain.CapGPSInterpolate:
		return []OptionSpec{
			{
				Key:          "window",
				Name:         "智能推算时间窗口 (Interpolate Window)",
				Description:  "根据前后邻近照片时间权重推算 GPS 的最大允许时间间隔 (如 15m, 30m, 1h, 2h)",
				Type:         TypeChoice,
				Scope:        ScopePlugin,
				PluginID:     domain.CapGPSInterpolate,
				DefaultValue: "15m",
				Choices:      []string{"15m", "30m", "1h", "2h", "4h"},
				Validate: func(val string) error {
					d, err := time.ParseDuration(strings.TrimSpace(val))
					if err != nil || d <= 0 {
						return fmt.Errorf("非法时间格式，请输入如 15m, 30m, 1h: %w", err)
					}
					return nil
				},
			},
		}
	case domain.CapReverseGeocode:
		return []OptionSpec{
			{
				Key:          "language",
				Name:         "地名元数据语言",
				Description:  "写入照片元数据的中文/本地地名语言格式 (默认 zh-CN)",
				Type:         TypeString,
				Scope:        ScopePlugin,
				PluginID:     domain.CapReverseGeocode,
				DefaultValue: "zh-CN",
			},
		}
	case domain.CapDateArchive:
		return []OptionSpec{
			{
				Key:          "in_place",
				Name:         "原地重命名模式 (In-Place)",
				Description:  "在源目录内直接规范化重命名，不建立 Processed/YYYY/MMDD/ 子目录",
				Type:         TypeBool,
				Scope:        ScopePlugin,
				PluginID:     domain.CapDateArchive,
				DefaultValue: false,
			},
			{
				Key:          "naming_template",
				Name:         "规范化重命名模板",
				Description:  "照片重命名模板（如 {PREFIX}_{YYYY-MM-DD}_{SEQ}{SUFFIX}）",
				Type:         TypeString,
				Scope:        ScopePlugin,
				PluginID:     domain.CapDateArchive,
				DefaultValue: "{PREFIX}_{YYYY-MM-DD}_{SEQ}{SUFFIX}",
			},
		}
	default:
		return nil
	}
}
