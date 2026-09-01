package i18n

import (
	"sync"
	"testing"
)

func TestI18n_BasicTranslation(t *testing.T) {
	SetLanguage("zh-CN")
	if !IsChinese() {
		t.Errorf("期望 IsChinese 为 true")
	}
	if GetLanguage() != LangZhCN {
		t.Errorf("期望语言为 %s，实际: %s", LangZhCN, GetLanguage())
	}

	zhAppTitle := T("appTitle")
	if zhAppTitle != "photools 摄影资产处理工作台" {
		t.Errorf("中文 appTitle 翻译不符合预期: %s", zhAppTitle)
	}

	SetLanguage("en-US")
	if IsChinese() {
		t.Errorf("期望 IsChinese 为 false")
	}
	if GetLanguage() != LangEnUS {
		t.Errorf("期望语言为 %s，实际: %s", LangEnUS, GetLanguage())
	}

	enAppTitle := T("appTitle")
	if enAppTitle != "photools - Photo Processing Workbench" {
		t.Errorf("英文 appTitle 翻译不符合预期: %s", enAppTitle)
	}
}

func TestI18n_NormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"zh", LangZhCN},
		{"zh-CN", LangZhCN},
		{"zh_CN.UTF-8", LangZhCN},
		{"zh-Hans", LangZhCN},
		{"en", LangEnUS},
		{"en-US", LangEnUS},
		{"en_US.UTF-8", LangEnUS},
		{"ja", LangEnUS}, // 默认回退英文
		{"", LangEnUS},
	}

	for _, tt := range tests {
		got := NormalizeLanguage(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeLanguage(%q) = %q; 期望 %q", tt.input, got, tt.expected)
		}
	}
}

func TestI18n_FallbackAndMissingKey(t *testing.T) {
	SetLanguage("en-US")

	// 测试完全不存在的 key
	missing := T("non_existent_key_12345")
	if missing != "non_existent_key_12345" {
		t.Errorf("不存在的 key 期望返回自身，实际: %s", missing)
	}

	// 恢复中文
	SetLanguage("zh-CN")
}

func TestI18n_Concurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				SetLanguage("zh-CN")
			} else {
				SetLanguage("en-US")
			}
			_ = T("appTitle")
			_ = IsChinese()
			_ = GetLanguage()
		}(i)
	}
	wg.Wait()
	SetLanguage("zh-CN")
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) != 2 {
		t.Errorf("期望支持 2 种语言，实际: %d", len(langs))
	}
}
