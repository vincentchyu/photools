package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/vincentchyu/photools/internal/domain"
)

// PluginMeta 记录单个插件在配置文件中的元数据与优先级设定
type PluginMeta struct {
	ID          domain.CapabilityID    `json:"id"`
	Name        string                 `json:"name"`
	Priority    int                    `json:"priority"` // 数值越小越优先；同数值归入同一 Phase 并行执行
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
	Options     map[string]interface{} `json:"options,omitempty"` // 插件专属自定义扩展参数 (如 {"window": "15m"})
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
	Plugins []PluginMeta `json:"plugins"`
}

var (
	defaultMetas = []PluginMeta{
		{
			ID:          domain.CapGPXMatching,
			Name:        "GPX 轨迹匹配与 GPS 修正",
			Priority:    10,
			Enabled:     true,
			Description: "从 GPX 目录读取轨迹为 RAW 写入经纬度并同步到 JPG/XMP",
			Options: map[string]interface{}{
				"geosync": "0",
			},
		},
		{
			ID:          domain.CapGPSInterpolate,
			Name:        "GPS 智能邻近推断与时间插值",
			Priority:    15,
			Enabled:     true,
			Description: "根据同批次前后邻近照片时间权重，自动推算补全无轨迹照片 GPS 坐标",
			Options: map[string]interface{}{
				"window": "15m",
			},
		},
		{
			ID:          domain.CapReverseGeocode,
			Name:        "逆地理编码与地名元数据写入",
			Priority:    20,
			Enabled:     true,
			Description: "根据 GPS 坐标检索国家/省/市/区/POI，写入 IPTC/XMP 地名元数据",
		},
		{
			ID:          domain.CapDateArchive,
			Name:        "按拍摄日期归档与规范重命名",
			Priority:    100,
			Enabled:     true,
			Description: "提取 EXIF 拍摄日期，规范重命名并安全归档至 Processed/YYYY/MMDD/",
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
	return PluginsConfig{Plugins: list}
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

	data, err := json.MarshalIndent(cfg, "", "  ")
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
