package locales

import "embed"

// FS 静态内嵌根目录所有语言 JSON 字典文件 (单一事实源)
//
//go:embed *.json
var FS embed.FS
