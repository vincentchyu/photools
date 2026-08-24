package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

func TestEnsureConfigFile_And_Load(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "plugins.json")

	// 1. 首次加载时文件不存在，自动创建默认配置
	cfg, err := LoadPluginsConfig(configPath)
	if err != nil {
		t.Fatalf("LoadPluginsConfig failed: %v", err)
	}

	if len(cfg.Plugins) != 4 {
		t.Fatalf("expected 4 default plugins, got %d", len(cfg.Plugins))
	}

	// 验证文件已被写入磁盘
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file %s to be created", configPath)
	}

	// 2. 检查默认优先级数值
	p1 := cfg.FindPluginMeta(domain.CapGPXMatching)
	if p1 == nil || p1.Priority != 10 {
		t.Errorf("expected CapGPXMatching Priority=10, got %v", p1)
	}

	p15 := cfg.FindPluginMeta(domain.CapGPSInterpolate)
	if p15 == nil || p15.Priority != 15 {
		t.Errorf("expected CapGPSInterpolate Priority=15, got %v", p15)
	}
	if win := p15.GetDurationOption("window", 0); win != 15*time.Minute {
		t.Errorf("expected CapGPSInterpolate default window=15m, got %v", win)
	}

	p2 := cfg.FindPluginMeta(domain.CapReverseGeocode)
	if p2 == nil || p2.Priority != 20 {
		t.Errorf("expected CapReverseGeocode Priority=20, got %v", p2)
	}

	p3 := cfg.FindPluginMeta(domain.CapDateArchive)
	if p3 == nil || p3.Priority != 100 {
		t.Errorf("expected CapDateArchive Priority=100, got %v", p3)
	}

	// 3. 修改配置并保存，验证持久化
	p1.Priority = 5
	p15.Options["window"] = "45m"
	if err := SavePluginsConfig(configPath, cfg); err != nil {
		t.Fatalf("SavePluginsConfig failed: %v", err)
	}

	reloaded, err := LoadPluginsConfig(configPath)
	if err != nil {
		t.Fatalf("reloading config failed: %v", err)
	}
	reloadedP1 := reloaded.FindPluginMeta(domain.CapGPXMatching)
	if reloadedP1.Priority != 5 {
		t.Errorf("expected updated priority 5, got %d", reloadedP1.Priority)
	}
	reloadedP15 := reloaded.FindPluginMeta(domain.CapGPSInterpolate)
	if win := reloadedP15.GetDurationOption("window", 0); win != 45*time.Minute {
		t.Errorf("expected reloaded window=45m, got %v", win)
	}
}

func TestAutoMigrateExistingConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "plugins.json")

	// 写入一个旧版本的只有 3 项的配置文件
	oldJSON := `{
  "plugins": [
    {
      "id": "gpx_matching",
      "name": "GPX 轨迹匹配与 GPS 修正",
      "priority": 10,
      "enabled": true,
      "description": "从 GPX 目录读取轨迹为 RAW 写入经纬度并同步到 JPG/XMP"
    },
    {
      "id": "reverse_geocode",
      "name": "逆地理编码与地名元数据写入",
      "priority": 20,
      "enabled": true,
      "description": "根据 GPS 坐标检索国家/省/市/区/POI，写入 IPTC/XMP 地名元数据"
    },
    {
      "id": "date_archive",
      "name": "按拍摄日期归档与规范重命名",
      "priority": 100,
      "enabled": true,
      "description": "提取 EXIF 拍摄日期，规范重命名并安全归档至 Processed/YYYY/MMDD/"
    }
  ]
}`
	_ = os.WriteFile(configPath, []byte(oldJSON), 0o644)

	// 调用 LoadPluginsConfig
	cfg, err := LoadPluginsConfig(configPath)
	if err != nil {
		t.Fatalf("LoadPluginsConfig failed: %v", err)
	}

	if len(cfg.Plugins) != 4 {
		t.Fatalf("expected 4 plugins after migration, got %d", len(cfg.Plugins))
	}

	// 验证磁盘上的文件是否被覆盖写入了新内容
	diskData, _ := os.ReadFile(configPath)
	if !strings.Contains(string(diskData), "gps_interpolate") {
		t.Errorf("expected file on disk to contain gps_interpolate, got:\n%s", string(diskData))
	}
}
