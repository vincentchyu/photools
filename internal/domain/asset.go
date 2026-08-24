package domain

import (
	"path/filepath"
	"sort"
)

// AssetGroup 代表同 basename 的一组媒体资产及其附属文件（拍摄单元）
type AssetGroup struct {
	BaseName       string   `json:"base_name"`
	Dir            string   `json:"dir"`
	RawPath        string   `json:"raw_path,omitempty"`
	JPGPath        string   `json:"jpg_path,omitempty"`
	XMPPath        string   `json:"xmp_path,omitempty"`
	CompanionPaths []string `json:"companion_paths,omitempty"`
}

// AllFiles 返回该资产组中的所有物理文件路径（去重保序）
func (a AssetGroup) AllFiles() []string {
	seen := make(map[string]bool)
	files := make([]string, 0, 3+len(a.CompanionPaths))
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	add(a.RawPath)
	add(a.JPGPath)
	add(a.XMPPath)
	for _, cp := range a.CompanionPaths {
		add(cp)
	}
	return files
}

// HasRaw 是否包含 RAW 格式主文件
func (a AssetGroup) HasRaw() bool {
	return a.RawPath != ""
}

// HasJPG 是否包含 JPG 预览/成片
func (a AssetGroup) HasJPG() bool {
	return a.JPGPath != ""
}

// HasXMP 是否包含 XMP 侧车文件
func (a AssetGroup) HasXMP() bool {
	return a.XMPPath != ""
}

// IsPaired 是否具备 RAW + JPG 的最小核心配对
func (a AssetGroup) IsPaired() bool {
	return a.HasRaw() && a.HasJPG()
}

// SortedCompanions 返回排序后的附加文件列表
func (a AssetGroup) SortedCompanions() []string {
	cp := make([]string, len(a.CompanionPaths))
	copy(cp, a.CompanionPaths)
	sort.Strings(cp)
	return cp
}

// HasPrimary 是否包含可决策的主媒体文件 (RAW 或 JPG)
func (a AssetGroup) HasPrimary() bool {
	return a.PrimaryPath() != ""
}

// PrimaryPath 返回主媒体文件路径（RAW 优先，若无 RAW 则以 JPG 为主文件）
func (a AssetGroup) PrimaryPath() string {
	if a.RawPath != "" {
		return a.RawPath
	}
	if a.JPGPath != "" {
		return a.JPGPath
	}
	if a.XMPPath != "" {
		return a.XMPPath
	}
	if len(a.CompanionPaths) > 0 {
		return a.CompanionPaths[0]
	}
	return ""
}

// DisplayName 返回用于展示的友好名称
func (a AssetGroup) DisplayName() string {
	if a.RawPath != "" {
		return filepath.Base(a.RawPath)
	}
	if a.JPGPath != "" {
		return filepath.Base(a.JPGPath)
	}
	return a.BaseName
}
