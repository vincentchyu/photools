package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

// Discoverer 负责扫描指定目录并聚合 AssetGroup
type Discoverer struct {
	rawExtensions map[string]struct{}
}

// NewDiscoverer 创建 Discoverer 实例
func NewDiscoverer(rawExts []string) *Discoverer {
	rawMap := make(map[string]struct{}, len(rawExts))
	for _, ext := range rawExts {
		clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
		if clean != "" {
			rawMap[clean] = struct{}{}
		}
	}
	return &Discoverer{rawExtensions: rawMap}
}

// 忽略的非摄影/文档/日志等文件扩展名（不作为伴随文件与独立资产处理）
var ignoredDocExtensions = map[string]struct{}{
	"md": {}, "log": {}, "txt": {}, "json": {}, "yaml": {}, "yml": {},
	"csv": {}, "pdf": {}, "doc": {}, "docx": {}, "zip": {}, "tar": {},
	"gz": {}, "7z": {}, "rar": {}, "bak": {}, "tmp": {}, "ds_store": {},
	"toml": {}, "sh": {}, "bat": {},
}

func isIgnoredDocExt(ext string) bool {
	_, ok := ignoredDocExtensions[strings.ToLower(ext)]
	return ok
}

// Discover 递归扫描 sourceDir 并返回按目录和 BaseName 排序的 AssetGroup
func (d *Discoverer) Discover(sourceDir string) ([]domain.AssetGroup, error) {
	groups := map[string]*domain.AssetGroup{}

	err := filepath.WalkDir(
		sourceDir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// 忽略隐藏目录（以 . 开头）
				if strings.HasPrefix(entry.Name(), ".") && entry.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}

			// 忽略隐藏文件与系统文件
			if strings.HasPrefix(entry.Name(), ".") {
				return nil
			}

			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(entry.Name()), "."))
			if isIgnoredDocExt(ext) {
				return nil
			}

			baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			dir := filepath.Dir(path)
			key := dir + "::" + strings.ToLower(baseName)

			group, ok := groups[key]
			if !ok {
				group = &domain.AssetGroup{
					BaseName: baseName,
					Dir:      dir,
				}
				groups[key] = group
			}

			switch {
			case d.isRawExt(ext):
				group.RawPath = path
			case ext == "jpg" || ext == "jpeg":
				group.JPGPath = path
			default:
				if strings.EqualFold(ext, "xmp") {
					group.XMPPath = path
				}
				group.CompanionPaths = append(group.CompanionPaths, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	var assets []domain.AssetGroup
	for _, group := range groups {
		// 摄影资产必须至少包含 RAW 或 JPG 主文件
		if !group.HasRaw() && !group.HasJPG() {
			continue
		}
		assets = append(assets, *group)
	}

	sort.Slice(
		assets, func(i, j int) bool {
			if assets[i].Dir != assets[j].Dir {
				return assets[i].Dir < assets[j].Dir
			}
			return assets[i].BaseName < assets[j].BaseName
		},
	)

	for i := range assets {
		sort.Strings(assets[i].CompanionPaths)
	}

	return assets, nil
}

func (d *Discoverer) isRawExt(ext string) bool {
	_, ok := d.rawExtensions[strings.ToLower(ext)]
	return ok
}

// ListGPXFiles 扫描指定目录下的全部 .gpx 轨迹文件
func ListGPXFiles(gpxDir string) ([]string, error) {
	var entries []string
	err := filepath.WalkDir(
		gpxDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".gpx") {
				entries = append(entries, path)
			}
			return nil
		},
	)
	return entries, err
}
