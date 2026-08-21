package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExpandUserPath 展开 ~ 为用户真实 Home 目录
func ExpandUserPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// CompleteDirectoryPath 根据输入前缀自动补全文件夹路径，并返回补全后的字符串及候选列表
func CompleteDirectoryPath(input string) (string, []string) {
	if input == "" {
		input = "." + string(filepath.Separator)
	}

	input = ExpandUserPath(input)

	var dir string
	var base string

	if strings.HasSuffix(input, "/") || strings.HasSuffix(input, "\\") {
		dir = input
		base = ""
	} else {
		dir = filepath.Dir(input)
		base = filepath.Base(input)
		if dir == "." && !strings.HasPrefix(input, ".") {
			dir = "."
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return input, nil
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// 忽略隐藏文件夹
		if strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(base, ".") {
			continue
		}

		if base == "" || strings.HasPrefix(strings.ToLower(e.Name()), strings.ToLower(base)) {
			full := filepath.Join(dir, e.Name()) + string(filepath.Separator)
			matches = append(matches, full)
		}
	}

	sort.Strings(matches)

	if len(matches) == 0 {
		return input, nil
	}

	if len(matches) == 1 {
		return matches[0], matches
	}

	// 多个候选，计算公共前缀
	common := longestCommonPrefix(matches)
	if common == "" || len(common) < len(input) {
		return input, matches
	}

	return common, matches
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
