package engine

import (
	"os"
	"path/filepath"
)

// DirMeaning 定义规范目录的说明与状态
type DirMeaning struct {
	Name       string `json:"name"`
	RelPath    string `json:"rel_path"`
	FullPath   string `json:"full_path"`
	Icon       string `json:"icon"`
	Usage      string `json:"usage"`
	ForMode    string `json:"for_mode"` // "通用", "geotag", "organize"
	Exists     bool   `json:"exists"`
	ItemsCount int    `json:"items_count"`
}

// GetStandardDirectorySpecs 返回规范目录的定义列表
func GetStandardDirectorySpecs(baseDir string) []DirMeaning {
	return []DirMeaning{
		{
			Name:     "Inbox",
			RelPath:  "Inbox",
			FullPath: filepath.Join(baseDir, "Inbox"),
			Icon:     "📥",
			Usage:    "待处理的照片源（RAW + JPG 配对及附属文件）",
			ForMode:  "通用（GPS清洗 / 日期归档）",
		},
		{
			Name:     "GPX",
			RelPath:  "GPX",
			FullPath: filepath.Join(baseDir, "GPX"),
			Icon:     "🗺️",
			Usage:    "移动设备或手表导出的 GPX 轨迹文件",
			ForMode:  "GPS 轨迹匹配",
		},
		{
			Name:     "Processed/geotag",
			RelPath:  filepath.Join("Processed", "geotag"),
			FullPath: filepath.Join(baseDir, "Processed", "geotag"),
			Icon:     "📍",
			Usage:    "GPS 修正后按拍摄日期规范归档的目标位置 (YYYY/MMDD/)",
			ForMode:  "GPS 轨迹匹配",
		},
		{
			Name:     "Processed/organize",
			RelPath:  filepath.Join("Processed", "organize"),
			FullPath: filepath.Join(baseDir, "Processed", "organize"),
			Icon:     "📁",
			Usage:    "仅按拍摄日期重命名整理后的归档位置 (YYYY/MMDD/)",
			ForMode:  "按拍摄日期归档",
		},
		{
			Name:     "Logs",
			RelPath:  "Logs",
			FullPath: filepath.Join(baseDir, "Logs"),
			Icon:     "📋",
			Usage:    "处理日志与待处理资产清单 (Markdown 报告)",
			ForMode:  "通用系统记录",
		},
	}
}

// InspectStandardDirectories 检查当前工作目录下的规范目录状态
func InspectStandardDirectories(baseDir string) []DirMeaning {
	specs := GetStandardDirectorySpecs(baseDir)
	for i := range specs {
		info, err := os.Stat(specs[i].FullPath)
		if err == nil && info.IsDir() {
			specs[i].Exists = true
			if entries, err := os.ReadDir(specs[i].FullPath); err == nil {
				specs[i].ItemsCount = len(entries)
			}
		} else {
			specs[i].Exists = false
			specs[i].ItemsCount = 0
		}
	}
	return specs
}

// EnsureStandardDirectories 检查并自动创建规范目录
func EnsureStandardDirectories(baseDir string) ([]DirMeaning, error) {
	specs := InspectStandardDirectories(baseDir)
	for i := range specs {
		if !specs[i].Exists {
			if err := os.MkdirAll(specs[i].FullPath, 0o755); err != nil {
				return nil, err
			}
			specs[i].Exists = true
		}
	}
	return specs, nil
}
