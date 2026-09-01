package i18n

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/vincentchyu/photools/locales"
)

const (
	LangZhCN = "zh-CN"
	LangEnUS = "en-US"
)

var (
	mu           sync.RWMutex
	currentLang  = LangZhCN
	translations = make(map[string]map[string]string)
	supported    = []string{LangZhCN, LangEnUS}
)

func init() {
	// 从全局唯一 locales.FS 加载所有内嵌的语言 JSON
	for _, lang := range supported {
		data, err := locales.FS.ReadFile(lang + ".json")
		if err == nil {
			var dict map[string]string
			if json.Unmarshal(data, &dict) == nil {
				translations[lang] = dict
			}
		}
	}
}

// NormalizeLanguage 规范化输入的语言标识
func NormalizeLanguage(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if strings.HasPrefix(lower, "zh") {
		return LangZhCN
	}
	if strings.HasPrefix(lower, "en") {
		return LangEnUS
	}
	return LangEnUS
}

// SetLanguage 设置当前运行时语言
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = NormalizeLanguage(lang)
}

// GetLanguage 获取当前语言标识 (zh-CN 或 en-US)
func GetLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// IsChinese 判断当前是否为中文模式
func IsChinese() bool {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang == LangZhCN
}

// SupportedLanguages 返回所有支持的语言列表
func SupportedLanguages() []string {
	return supported
}

// T 翻译指定的键，支持可选的格式化参数
func T(key string, args ...any) string {
	mu.RLock()
	lang := currentLang
	dict, ok := translations[lang]
	mu.RUnlock()

	var text string
	if ok {
		text = dict[key]
	}

	// 降级回退到 zh-CN
	if text == "" && lang != LangZhCN {
		if zhDict, hasZh := translations[LangZhCN]; hasZh {
			text = zhDict[key]
		}
	}

	// 依然找不到则直接返回 key 本身
	if text == "" {
		text = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}
