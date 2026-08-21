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

// AllFiles 返回该资产组中的所有物理文件路径
func (a AssetGroup) AllFiles() []string {
	files := make([]string, 0, 2+len(a.CompanionPaths))
	if a.RawPath != "" {
		files = append(files, a.RawPath)
	}
	if a.JPGPath != "" {
		files = append(files, a.JPGPath)
	}
	files = append(files, a.CompanionPaths...)
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
