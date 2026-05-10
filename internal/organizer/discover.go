package organizer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PhotoAsset 代表一组具有相同基础名称（BaseName）的媒体资源及其配套文件
type PhotoAsset struct {
	BaseName       string
	Dir            string
	RawPath        string
	JPGPath        string
	XMPPath        string
	CompanionPaths []string
}

// AllFiles 返回该资产组中的所有文件路径
func (a *PhotoAsset) AllFiles() []string {
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

// DiscoverMediaGroups 扫描指定目录，根据基础名称分组并返回 PhotoAsset 列表。
// rawExts 用于识别 RAW 文件，不区分大小写。
func DiscoverMediaGroups(sourceDir string, rawExts []string) ([]PhotoAsset, error) {
	groups := map[string]*PhotoAsset{}

	err := filepath.WalkDir(
		sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Name()), "."))
			baseName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			dir := filepath.Dir(path)
			key := dir + "::" + strings.ToLower(baseName)

			group, ok := groups[key]
			if !ok {
				group = &PhotoAsset{
					BaseName: baseName,
					Dir:      dir,
				}
				groups[key] = group
			}

			switch {
			case isRawExt(ext, rawExts):
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

	var assets []PhotoAsset
	for _, group := range groups {
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

func isRawExt(ext string, rawExts []string) bool {
	for _, r := range rawExts {
		if strings.EqualFold(ext, r) {
			return true
		}
	}
	return false
}
