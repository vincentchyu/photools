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
	BaseDir             string   `json:"base_dir,omitempty"`
	SourceDir           string   `json:"source_dir,omitempty"`
	TargetDir           string   `json:"target_dir,omitempty"`
	GPXDir              string   `json:"gpx_dir,omitempty"`              // GPX 轨迹目录 (默认 ~/.config/gpx)
	LogDir              string   `json:"log_dir,omitempty"`              // 全局日志与报告目录 (默认 ~/.logs/photools)
	FlatMode            bool     `json:"flat_mode,omitempty"`            // 忽略传统分层目录结构，在指定目录下直接扫描并原地保存/处理
	SidecarPolicy       string   `json:"sidecar_policy,omitempty"`       // 侧车写入策略: read_only, sidecar_only, embed_and_sidecar, embed_only
	SidecarOnly         bool     `json:"sidecar_only,omitempty"`         // 仅生成/修改 {file}.xmp 侧车文件 (向后兼容标志)
	CompanionExtensions []string `json:"companion_extensions,omitempty"` // 伴随文件扩展名白名单 (如 wav, acr, exf)
	RawExtensions       []string `json:"raw_extensions,omitempty"`       // RAW 格式白名单
	Workers             int      `json:"workers,omitempty"`              // 并发处理协程数
	AllowNoGPS          bool     `json:"allow_no_gps,omitempty"`         // 软降级容错：无 GPS 坐标时允许跳过地名写入直接归档
	TestBackup          bool     `json:"test_backup,omitempty"`          // 处理前是否快照备份源文件
	BackupDir           string   `json:"backup_dir,omitempty"`           // 自定义备份目录
	Language            string   `json:"language,omitempty"`             // 界面与补全语言: zh-CN, en-US
}

// SessionConfig 封装当前运行时会话的全部配置（全局配置 + 各插件独立配置）
type SessionConfig struct {
	Global  GlobalSettings                    `json:"global"`
	Plugins map[domain.CapabilityID]PluginOpt `json:"plugins"`
}

// PluginOpt 封装单个插件在会话中的专属配置项集合
type PluginOpt struct {
	ID       domain.CapabilityID `json:"id"`
	Priority int                 `json:"priority"`
	Enabled  bool                `json:"enabled"`
	Options  map[string]any      `json:"options"`
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
		Language:            "en-US",
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
	} else {
		if cfg.Global.Language != "" {
			sc.Global.Language = cfg.Global.Language
		}
		if cfg.Global.BaseDir != "" && len(baseDir) == 0 {
			sc.Global.BaseDir = cfg.Global.BaseDir
		}
		if cfg.Global.SourceDir != "" {
			sc.Global.SourceDir = cfg.Global.SourceDir
		}
		if cfg.Global.SidecarPolicy != "" {
			sc.Global.SidecarPolicy = cfg.Global.SidecarPolicy
		}
		if len(cfg.Global.CompanionExtensions) > 0 {
			sc.Global.CompanionExtensions = cfg.Global.CompanionExtensions
		}
		if len(cfg.Global.RawExtensions) > 0 {
			sc.Global.RawExtensions = cfg.Global.RawExtensions
		}
		if cfg.Global.Workers > 0 {
			sc.Global.Workers = cfg.Global.Workers
		}
		sc.Global.FlatMode = cfg.Global.FlatMode
		sc.Global.AllowNoGPS = cfg.Global.AllowNoGPS
		sc.Global.TestBackup = cfg.Global.TestBackup
	}

	for _, p := range cfg.Plugins {
		sc.Plugins[p.ID] = PluginOpt{
			ID:       p.ID,
			Priority: p.Priority,
			Enabled:  p.Enabled,
			Options:  maps.Clone(p.Options),
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
func (s *SessionConfig) GetPluginOptions(pluginID domain.CapabilityID) map[string]any {
	if p, ok := s.Plugins[pluginID]; ok && p.Options != nil {
		return p.Options
	}
	return nil
}

// SetPluginOption 设置指定插件的配置项（当前会话生效）
func (s *SessionConfig) SetPluginOption(pluginID domain.CapabilityID, key string, value any) {
	p, ok := s.Plugins[pluginID]
	if !ok {
		p = PluginOpt{
			ID:      pluginID,
			Enabled: true,
			Options: make(map[string]any),
		}
	}
	if p.Options == nil {
		p.Options = make(map[string]any)
	}
	p.Options[key] = value
	s.Plugins[pluginID] = p
}

// SetPluginEnabled 设置指定插件的启用状态
func (s *SessionConfig) SetPluginEnabled(pluginID domain.CapabilityID, enabled bool) {
	p, ok := s.Plugins[pluginID]
	if !ok {
		p = PluginOpt{
			ID:      pluginID,
			Options: make(map[string]any),
		}
	}
	p.Enabled = enabled
	s.Plugins[pluginID] = p
}

// ApplyToPluginsConfig 将当前会话中的插件优先级与选项同步回持久化配置结构
func (s *SessionConfig) ApplyToPluginsConfig(cfg *PluginsConfig) {
	if cfg == nil {
		return
	}
	// 仅将真正的全局持久化项回写到持久化配置中，绝不将临时会话项 (BaseDir/SourceDir/FlatMode/BackupDir 等) 污染写盘
	cfg.Global.Language = s.Global.Language
	cfg.Global.GPXDir = s.Global.GPXDir
	cfg.Global.LogDir = s.Global.LogDir
	cfg.Global.SidecarPolicy = s.Global.SidecarPolicy
	cfg.Global.CompanionExtensions = s.Global.CompanionExtensions
	cfg.Global.RawExtensions = s.Global.RawExtensions
	cfg.Global.Workers = s.Global.Workers

	for i := range cfg.Plugins {
		p := &cfg.Plugins[i]
		if opt, ok := s.Plugins[p.ID]; ok {
			p.Priority = opt.Priority
			p.Enabled = opt.Enabled
			if p.Options == nil {
				p.Options = make(map[string]any)
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
			NameKey:      "optBaseDirName",
			DescKey:      "optBaseDirDesc",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "~/Pictures/GPS",
		},
		{
			Key:          "gpx_dir",
			NameKey:      "optGpxDirName",
			DescKey:      "optGpxDirDesc",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "~/.config/gpx",
		},
		{
			Key:          "source_dir",
			NameKey:      "optSourceDirName",
			DescKey:      "optSourceDirDesc",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "<BaseDir>/Inbox",
		},
		{
			Key:          "flat_mode",
			NameKey:      "optFlatModeName",
			DescKey:      "optFlatModeDesc",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:     "sidecar_policy",
			NameKey: "optSidecarPolicyName",
			DescKey: "optSidecarPolicyDesc",
			Type:    TypeChoice,
			Choices: []string{
				string(domain.PolicySmart), string(domain.PolicySidecarOnly), string(domain.PolicyEmbedAndSidecar),
				string(domain.PolicyEmbedOnly),
			},
			Scope:        ScopeGlobal,
			DefaultValue: string(domain.PolicySmart),
		},
		{
			Key:          "sidecar_only",
			NameKey:      "optSidecarOnlyName",
			DescKey:      "optSidecarOnlyDesc",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "companion_extensions",
			NameKey:      "optCompanionExtsName",
			DescKey:      "optCompanionExtsDesc",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "wav,acr,exf",
		},
		{
			Key:          "raw_extensions",
			NameKey:      "optRawExtsName",
			DescKey:      "optRawExtsDesc",
			Type:         TypeString,
			Scope:        ScopeGlobal,
			DefaultValue: "nef,cr3,arw,dng,raf,rw2,orf",
		},
		{
			Key:          "workers",
			NameKey:      "optWorkersName",
			DescKey:      "optWorkersDesc",
			Type:         TypeInt,
			Scope:        ScopeGlobal,
			DefaultValue: runtime.NumCPU(),
		},
		{
			Key:          "allow_no_gps",
			NameKey:      "optAllowNoGpsName",
			DescKey:      "optAllowNoGpsDesc",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "test_backup",
			NameKey:      "optTestBackupName",
			DescKey:      "optTestBackupDesc",
			Type:         TypeBool,
			Scope:        ScopeGlobal,
			DefaultValue: false,
		},
		{
			Key:          "language",
			NameKey:      "optLanguageName",
			DescKey:      "optLanguageDesc",
			Type:         TypeChoice,
			Choices:      []string{"zh-CN", "en-US"},
			Scope:        ScopeGlobal,
			DefaultValue: "zh-CN",
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
				NameKey:      "optGeosyncName",
				DescKey:      "optGeosyncDesc",
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
				NameKey:      "optWindowName",
				DescKey:      "optWindowDesc",
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
				NameKey:      "optGeocodeLangName",
				DescKey:      "optGeocodeLangDesc",
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
				NameKey:      "optInPlaceName",
				DescKey:      "optInPlaceDesc",
				Type:         TypeBool,
				Scope:        ScopePlugin,
				PluginID:     domain.CapDateArchive,
				DefaultValue: false,
			},
			{
				Key:          "naming_template",
				NameKey:      "optNamingTemplateName",
				DescKey:      "optNamingTemplateDesc",
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
