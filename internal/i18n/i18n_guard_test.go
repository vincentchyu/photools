package i18n_test

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/vincentchyu/photools/internal/capabilities/datearchive"
	"github.com/vincentchyu/photools/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photools/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photools/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/i18n"
)

// cjkRegex 匹配中文字符范围
var cjkRegex = regexp.MustCompile(`[\p{Han}]`)

// TestI18n_DictionarySymmetry 验证中英文 JSON 字典的键 100% 镜像对称，无漏译
func TestI18n_DictionarySymmetry(t *testing.T) {
	zhData, err := os.ReadFile("../../locales/zh-CN.json")
	if err != nil {
		t.Fatalf("无法读取 zh-CN.json: %v", err)
	}
	enData, err := os.ReadFile("../../locales/en-US.json")
	if err != nil {
		t.Fatalf("无法读取 en-US.json: %v", err)
	}

	var zhMap map[string]string
	var enMap map[string]string
	if err := json.Unmarshal(zhData, &zhMap); err != nil {
		t.Fatalf("解析 zh-CN.json 失败: %v", err)
	}
	if err := json.Unmarshal(enData, &enMap); err != nil {
		t.Fatalf("解析 en-US.json 失败: %v", err)
	}

	// 1. 检查 zh-CN 中的每个 key 是否都在 en-US 中存在且非空
	for k := range zhMap {
		enVal, exists := enMap[k]
		if !exists || enVal == "" {
			t.Errorf("en-US.json 缺失键或为空: %s", k)
		}
	}

	// 2. 检查 en-US 中的每个 key 是否都在 zh-CN 中存在且非空
	for k := range enMap {
		zhVal, exists := zhMap[k]
		if !exists || zhVal == "" {
			t.Errorf("zh-CN.json 缺失键或为空: %s", k)
		}
	}
}

// TestI18n_GlobalOptionSpecs_English 验证在英文模式下所有全局配置项名称与说明绝无硬编码中文
func TestI18n_GlobalOptionSpecs_English(t *testing.T) {
	i18n.SetLanguage(i18n.LangEnUS)
	defer i18n.SetLanguage(i18n.LangZhCN)

	specs := config.GlobalOptionSpecs()
	if len(specs) == 0 {
		t.Fatalf("GlobalOptionSpecs 返回为空")
	}

	for _, spec := range specs {
		name := spec.DisplayName()
		desc := spec.DisplayDescription()

		if cjkRegex.MatchString(name) {
			t.Errorf("全局配置项 [%s] DisplayName() 在 en-US 模式下包含中文字符: %s", spec.Key, name)
		}
		if cjkRegex.MatchString(desc) {
			t.Errorf("全局配置项 [%s] DisplayDescription() 在 en-US 模式下包含中文字符: %s", spec.Key, desc)
		}
	}
}

// TestI18n_PluginSupportedOptions_English 验证在英文模式下四大插件的 SupportedOptions 绝无硬编码中文
func TestI18n_PluginSupportedOptions_English(t *testing.T) {
	i18n.SetLanguage(i18n.LangEnUS)
	defer i18n.SetLanguage(i18n.LangZhCN)

	caps := []domain.Capability{
		gpxmatch.NewCapability(gpxmatch.Config{}),
		gpsinterpolate.NewCapability(gpsinterpolate.Config{}),
		reversegeocode.NewCapability(reversegeocode.Config{}),
		datearchive.NewCapability(datearchive.Config{}),
	}

	for _, capInst := range caps {
		opts := capInst.SupportedOptions()
		for _, opt := range opts {
			name := opt.DisplayName()
			desc := opt.DisplayDescription()

			if cjkRegex.MatchString(name) {
				t.Errorf("插件 [%s] 配置项 [%s] DisplayName() 在 en-US 模式下包含中文字符: %s", capInst.ID(), opt.Key, name)
			}
			if cjkRegex.MatchString(desc) {
				t.Errorf("插件 [%s] 配置项 [%s] DisplayDescription() 在 en-US 模式下包含中文字符: %s", capInst.ID(), opt.Key, desc)
			}
		}

		// 检查插件本身的 Name() 和 Description()
		if cjkRegex.MatchString(capInst.Name()) {
			t.Errorf("插件 [%s] Name() 在 en-US 模式下包含中文字符: %s", capInst.ID(), capInst.Name())
		}
		if cjkRegex.MatchString(capInst.Description()) {
			t.Errorf("插件 [%s] Description() 在 en-US 模式下包含中文字符: %s", capInst.ID(), capInst.Description())
		}
	}
}

// TestI18n_PluginMetas_English 验证在英文模式下 defaultMetas 的 Title() 和 Desc() 绝无硬编码中文
func TestI18n_PluginMetas_English(t *testing.T) {
	i18n.SetLanguage(i18n.LangEnUS)
	defer i18n.SetLanguage(i18n.LangZhCN)

	cfg := config.DefaultPluginsConfig()
	for _, meta := range cfg.Plugins {
		title := meta.Title()
		desc := meta.Desc()

		if cjkRegex.MatchString(title) {
			t.Errorf("插件元数据 [%s] Title() 在 en-US 模式下包含中文字符: %s", meta.ID, title)
		}
		if cjkRegex.MatchString(desc) {
			t.Errorf("插件元数据 [%s] Desc() 在 en-US 模式下包含中文字符: %s", meta.ID, desc)
		}
	}
}

// TestI18n_OptionSpecKeysExist 验证所有配置项的 NameKey 和 DescKey 在字典中均存在
func TestI18n_OptionSpecKeysExist(t *testing.T) {
	zhData, err := os.ReadFile("../../locales/zh-CN.json")
	if err != nil {
		t.Fatalf("无法读取 zh-CN.json: %v", err)
	}
	var zhMap map[string]string
	if err := json.Unmarshal(zhData, &zhMap); err != nil {
		t.Fatalf("解析 zh-CN.json 失败: %v", err)
	}

	checkSpec := func(context string, spec domain.OptionSpec) {
		if spec.NameKey != "" {
			if _, exists := zhMap[spec.NameKey]; !exists {
				t.Errorf("[%s] 选项 [%s] 的 NameKey=%s 在字典中不存在", context, spec.Key, spec.NameKey)
			}
		}
		if spec.DescKey != "" {
			if _, exists := zhMap[spec.DescKey]; !exists {
				t.Errorf("[%s] 选项 [%s] 的 DescKey=%s 在字典中不存在", context, spec.Key, spec.DescKey)
			}
		}
	}

	// 1. 全局配置项
	for _, spec := range config.GlobalOptionSpecs() {
		checkSpec("Global", spec)
	}

	// 2. 四大插件的 SupportedOptions
	caps := []domain.Capability{
		gpxmatch.NewCapability(gpxmatch.Config{}),
		gpsinterpolate.NewCapability(gpsinterpolate.Config{}),
		reversegeocode.NewCapability(reversegeocode.Config{}),
		datearchive.NewCapability(datearchive.Config{}),
	}
	for _, capInst := range caps {
		for _, spec := range capInst.SupportedOptions() {
			checkSpec(string(capInst.ID()), spec)
		}
		for _, spec := range config.PluginOptionSpecs(capInst.ID()) {
			checkSpec(string(capInst.ID())+"_schema", spec)
		}
	}

	// 3. 插件元数据 NameKey / DescKey
	cfg := config.DefaultPluginsConfig()
	for _, meta := range cfg.Plugins {
		if meta.NameKey != "" {
			if _, exists := zhMap[meta.NameKey]; !exists {
				t.Errorf("插件元数据 [%s] 的 NameKey=%s 在字典中不存在", meta.ID, meta.NameKey)
			}
		}
		if meta.DescKey != "" {
			if _, exists := zhMap[meta.DescKey]; !exists {
				t.Errorf("插件元数据 [%s] 的 DescKey=%s 在字典中不存在", meta.ID, meta.DescKey)
			}
		}
	}
}

// TestI18n_StageDisplayName_English 验证在英文模式下 PipelineStage 的 DisplayName 绝无中文字符
func TestI18n_StageDisplayName_English(t *testing.T) {
	i18n.SetLanguage(i18n.LangEnUS)
	defer i18n.SetLanguage(i18n.LangZhCN)

	stages := []domain.PipelineStage{
		domain.StageInit,
		domain.StageDiscover,
		domain.StagePrecheck,
		domain.StageGeotag,
		domain.StageInterpolate,
		domain.StageGeocode,
		domain.StageSync,
		domain.StageArchive,
		domain.StageBackup,
		domain.StageRestore,
		domain.StageSummary,
		domain.StageComplete,
	}

	for _, st := range stages {
		displayName := domain.StageDisplayName(st)
		if cjkRegex.MatchString(displayName) {
			t.Errorf("阶段 [%s] DisplayName() 在 en-US 模式下包含中文字符: %s", st, displayName)
		}
	}
}

// TestI18n_ChineseMode 验证在中文模式下全部能正常返回中文
func TestI18n_ChineseMode(t *testing.T) {
	i18n.SetLanguage(i18n.LangZhCN)

	specs := config.GlobalOptionSpecs()
	for _, spec := range specs {
		name := spec.DisplayName()
		if !cjkRegex.MatchString(name) {
			t.Errorf("全局配置项 [%s] 在 zh-CN 模式下未包含中文: %s", spec.Key, name)
		}
	}

	cfg := config.DefaultPluginsConfig()
	for _, meta := range cfg.Plugins {
		title := meta.Title()
		if !cjkRegex.MatchString(title) {
			t.Errorf("插件元数据 [%s] 在 zh-CN 模式下未包含中文: %s", meta.ID, title)
		}
	}

	caps := []domain.Capability{
		gpxmatch.NewCapability(gpxmatch.Config{}),
		gpsinterpolate.NewCapability(gpsinterpolate.Config{}),
		reversegeocode.NewCapability(reversegeocode.Config{}),
		datearchive.NewCapability(datearchive.Config{}),
	}
	for _, capInst := range caps {
		if !cjkRegex.MatchString(capInst.Name()) {
			t.Errorf("插件 [%s] Name() 在 zh-CN 模式下未包含中文: %s", capInst.ID(), capInst.Name())
		}
		if !cjkRegex.MatchString(capInst.Description()) {
			t.Errorf("插件 [%s] Description() 在 zh-CN 模式下未包含中文: %s", capInst.ID(), capInst.Description())
		}
		for _, opt := range capInst.SupportedOptions() {
			if !cjkRegex.MatchString(opt.DisplayName()) {
				t.Errorf("插件 [%s] 选项 [%s] DisplayName() 在 zh-CN 模式下未包含中文: %s", capInst.ID(), opt.Key, opt.DisplayName())
			}
		}
	}
}

// TestI18n_AnalyzeDuplicateValuesAndUnusedKeys 分析字典中的重复文案项目
func TestI18n_AnalyzeDuplicateValuesAndUnusedKeys(t *testing.T) {
	zhData, err := os.ReadFile("../../locales/zh-CN.json")
	if err != nil {
		t.Fatalf("无法读取 zh-CN.json: %v", err)
	}
	var zhMap map[string]string
	if err := json.Unmarshal(zhData, &zhMap); err != nil {
		t.Fatalf("解析 zh-CN.json 失败: %v", err)
	}

	// 查找完全相同文案的键分组
	valToKeys := make(map[string][]string)
	for k, v := range zhMap {
		valToKeys[v] = append(valToKeys[v], k)
	}

	for v, keys := range valToKeys {
		if len(keys) > 1 {
			t.Logf("发现文案重合项目 (%d 个键具有相同中文): %v => %q", len(keys), keys, v)
		}
	}
}
