package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

// PluginMeta 记录单个插件在配置文件中的元数据与优先级设定
type PluginMeta struct {
	ID       domain.CapabilityID    `json:"id"`
	NameKey  string                 `json:"-"`        // 内存中 i18n 字典键 (不持久化到 JSON)
	DescKey  string                 `json:"-"`        // 内存中 i18n 字典键 (不持久化到 JSON)
	Priority int                    `json:"priority"` // 数值越小越优先；同数值归入同一 Phase 并行执行
	Enabled  bool                   `json:"enabled"`
	Options  map[string]interface{} `json:"options,omitempty"` // 插件专属自定义扩展参数 (如 {"window": "15m"})
}

// Title 根据当前语言动态返回插件标题 (100% 字典驱动)
func (p PluginMeta) Title() string {
	if p.NameKey != "" {
		return i18n.T(p.NameKey)
	}
	return string(p.ID)
}

// Desc 根据当前语言动态返回插件说明 (100% 字典驱动)
func (p PluginMeta) Desc() string {
	if p.DescKey != "" {
		return i18n.T(p.DescKey)
	}
	return ""
}

// GetStringOption 获取字符串配置项
func (p *PluginMeta) GetStringOption(key, defaultValue string) string {
	if p == nil || p.Options == nil {
		return defaultValue
	}
	if v, ok := p.Options[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultValue
}

// GetDurationOption 获取时间间隔配置项 (如 "15m", "1h")
func (p *PluginMeta) GetDurationOption(key string, defaultValue time.Duration) time.Duration {
	str := p.GetStringOption(key, "")
	if str == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return defaultValue
	}
	return d
}

// GetBoolOption 获取布尔配置项
func (p *PluginMeta) GetBoolOption(key string, defaultValue bool) bool {
	if p == nil || p.Options == nil {
		return defaultValue
	}
	if v, ok := p.Options[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// PluginsConfig 插件全局持久化配置结构
type PluginsConfig struct {
	Global  GlobalSettings `json:"global,omitempty"`
	Plugins []PluginMeta   `json:"plugins"`
}

var (
	defaultMetas = []PluginMeta{
		{
			ID:       domain.CapGPXMatching,
			NameKey:  "capGpxName",
			DescKey:  "capGpxDesc",
			Priority: 10,
			Enabled:  true,
			Options: map[string]interface{}{
				"geosync": "0",
			},
		},
		{
			ID:       domain.CapGPSInterpolate,
			NameKey:  "capInterpolateName",
			DescKey:  "capInterpolateDesc",
			Priority: 15,
			Enabled:  true,
			Options: map[string]interface{}{
				"window": "15m",
			},
		},
		{
			ID:       domain.CapReverseGeocode,
			NameKey:  "capGeocodeName",
			DescKey:  "capGeocodeDesc",
			Priority: 20,
			Enabled:  true,
		},
		{
			ID:       domain.CapDateArchive,
			NameKey:  "capArchiveName",
			DescKey:  "capArchiveDesc",
			Priority: 100,
			Enabled:  true,
			Options: map[string]interface{}{
				"in_place": false,
			},
		},
	}
	cfgMutex sync.RWMutex
)

// DefaultPluginsConfig 返回内置默认的插件配置实例副本
func DefaultPluginsConfig() PluginsConfig {
	list := make([]PluginMeta, len(defaultMetas))
	copy(list, defaultMetas)
	return PluginsConfig{
		Global: GlobalSettings{
			Language:            "en-US",
			GPXDir:              DefaultGPXDir(),
			LogDir:              DefaultLogDir(),
			SidecarPolicy:       common.PolicySmart.String(),
			CompanionExtensions: common.DefaultCompanionExtensions,
			RawExtensions:       common.DefaultRawExtensions,
			Workers:             runtime.NumCPU(),
		},
		Plugins: list,
	}
}

// GetConfigPath 返回插件配置文件的全局默认存储路径 ~/.config/photools/plugins.json
func GetConfigPath() string {
	if custom := os.Getenv("PHOTOOLS_PLUGINS_CONFIG"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "plugins.json"
	}
	return filepath.Join(home, ".config", "photools", "plugins.json")
}

// EnsureConfigFile 检查指定路径的配置文件，若不存在则自动创建并写入默认模板
func EnsureConfigFile(path string) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	if _, err := os.Stat(path); err == nil {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败 (%s): %w", dir, err)
	}

	cfg := DefaultPluginsConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// LoadPluginsConfig 加载插件配置（若文件不存在自动初始化创建）
func LoadPluginsConfig(path string) (*PluginsConfig, error) {
	if path == "" {
		path = GetConfigPath()
	}

	if err := EnsureConfigFile(path); err != nil {
		cfg := DefaultPluginsConfig()
		return &cfg, nil
	}

	cfgMutex.RLock()
	data, err := os.ReadFile(path)
	cfgMutex.RUnlock()
	if err != nil {
		cfg := DefaultPluginsConfig()
		return &cfg, nil
	}

	var loaded PluginsConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		cfg := DefaultPluginsConfig()
		return &cfg, fmt.Errorf("解析 plugins.json 失败: %w", err)
	}

	// 补全缺失的默认插件定义（防止用户升级后缺少新插件）
	existingMap := make(map[domain.CapabilityID]bool)
	for _, p := range loaded.Plugins {
		existingMap[p.ID] = true
	}

	modified := false
	for _, def := range defaultMetas {
		if !existingMap[def.ID] {
			loaded.Plugins = append(loaded.Plugins, def)
			modified = true
		}
	}

	// 检查原始配置中是否存在旧版本的冗余文案字段，若存在则触发自愈清洗
	rawStr := string(data)
	if strings.Contains(rawStr, "\"name\":") || strings.Contains(rawStr, "\"description\":") ||
		strings.Contains(rawStr, "\"name_key\":") || strings.Contains(rawStr, "\"desc_key\":") {
		modified = true
	}

	// 为所有插件绑定内存中的 NameKey 与 DescKey
	for i := range loaded.Plugins {
		p := &loaded.Plugins[i]
		for _, def := range defaultMetas {
			if def.ID == p.ID {
				p.NameKey = def.NameKey
				p.DescKey = def.DescKey
				break
			}
		}
	}

	// 补全已有插件中缺失的默认 Options
	for i := range loaded.Plugins {
		p := &loaded.Plugins[i]
		for _, def := range defaultMetas {
			if def.ID == p.ID && len(def.Options) > 0 {
				if p.Options == nil {
					p.Options = make(map[string]interface{})
				}
				for optK, optV := range def.Options {
					if _, exists := p.Options[optK]; !exists {
						p.Options[optK] = optV
						modified = true
					}
				}
			}
		}
	}

	// 按 Priority 升序排序
	sort.Slice(loaded.Plugins, func(i, j int) bool {
		return loaded.Plugins[i].Priority < loaded.Plugins[j].Priority
	})

	if modified {
		if err := SavePluginsConfig(path, &loaded); err != nil {
			return &loaded, fmt.Errorf("保存升级配置失败 (%s): %w", path, err)
		}
	}

	return &loaded, nil
}

// SavePluginsConfig 保存插件配置到指定文件
func SavePluginsConfig(path string, cfg *PluginsConfig) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	if path == "" {
		path = GetConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 复制一份副本并精简 GlobalSettings 为纯净持久化字段，彻底杜绝会话临时工作路径被序列化落盘
	cleanCfg := PluginsConfig{
		Global: GlobalSettings{
			Language:            cfg.Global.Language,
			GPXDir:              cfg.Global.GPXDir,
			LogDir:              cfg.Global.LogDir,
			SidecarPolicy:       cfg.Global.SidecarPolicy,
			CompanionExtensions: cfg.Global.CompanionExtensions,
			RawExtensions:       cfg.Global.RawExtensions,
			Workers:             cfg.Global.Workers,
		},
		Plugins: cfg.Plugins,
	}

	data, err := json.MarshalIndent(cleanCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// FindPluginMeta 根据 ID 查找指定插件的配置
func (c *PluginsConfig) FindPluginMeta(id domain.CapabilityID) *PluginMeta {
	for i := range c.Plugins {
		if c.Plugins[i].ID == id {
			return &c.Plugins[i]
		}
	}
	return nil
}
