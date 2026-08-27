package geodata

import (
	"bufio"
	"embed"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed data
var embeddedDataFS embed.FS

var (
	mappingInitOnce sync.Once
	mappingMu       sync.RWMutex

	// ChinaAdmin1Map FIPS 10-4 / ISO / GB 省级行政区划代码字典
	ChinaAdmin1Map = make(map[string]string)

	// ChinaAdmin2Map GB/T 2260 4位地级行政区划代码字典 (333+ 地级市/地区/自治州/盟)
	ChinaAdmin2Map = make(map[string]string)

	// Admin1ZHMap 全球一级行政区划中英字母名映射字典 (3865+ 省/州/大区/都道府县)
	Admin1ZHMap = make(map[string]AdminCodeMeta)

	// Admin2ZHMap 全球二级行政区划中英字母名映射字典 (47592+ 地级市/县/区/郡)
	Admin2ZHMap = make(map[string]AdminCodeMeta)

	// IsoCountryMap ISO 2位国家代码映射字典
	IsoCountryMap = make(map[string]CountryMeta)
)

// AdminCodeMeta 行政区划代码及中英字母名元数据
type AdminCodeMeta struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	NameASCII string `json:"name_ascii"`
	NameZH    string `json:"name_zh"`
	GeoNameID int64  `json:"geoname_id,omitempty"`
}

type admin1Data struct {
	FIPS map[string]string `json:"fips"`
	ISO  map[string]string `json:"iso"`
	GB   map[string]string `json:"gb"`
}

func init() {
	EnsureMappingsLoaded()
}

// EnsureMappingsLoaded 线程安全地装载内嵌与外部 GeoNames 映射源文件
func EnsureMappingsLoaded() {
	mappingInitOnce.Do(
		func() {
			loadEmbeddedMappings()
			home, err := os.UserHomeDir()
			if err == nil {
				geoDataDir := filepath.Join(home, ".config", "photools", "geodata")
				LoadMappingFilesFromDir(geoDataDir)
			}
		},
	)
}

// loadEmbeddedMappings 从内嵌的 JSON 文件中解析行政区划与国家映射字典
func loadEmbeddedMappings() {
	mappingMu.Lock()
	defer mappingMu.Unlock()

	// 1. 装载基础 Admin1 (省级)
	if data, err := embeddedDataFS.ReadFile("data/admin1_codes.json"); err == nil && len(data) > 0 {
		var a1 admin1Data
		if err := json.Unmarshal(data, &a1); err == nil {
			maps.Copy(ChinaAdmin1Map, a1.FIPS)
			for k, v := range a1.ISO {
				ChinaAdmin1Map[k] = v
				ChinaAdmin1Map["CN-"+k] = v
			}
			for k, v := range a1.GB {
				if _, exists := ChinaAdmin1Map[k]; !exists {
					ChinaAdmin1Map[k] = v
				}
			}
		}
	}

	// 2. 装载基础 Admin2 (地级市/地区/自治州/盟 333+)
	if data, err := embeddedDataFS.ReadFile("data/admin2_codes.json"); err == nil && len(data) > 0 {
		var a2 map[string]string
		if err := json.Unmarshal(data, &a2); err == nil {
			maps.Copy(ChinaAdmin2Map, a2)
		}
	}

	// 3. 装载 Country (国家元数据)
	if data, err := embeddedDataFS.ReadFile("data/country_codes.json"); err == nil && len(data) > 0 {
		var cm map[string]struct {
			Code      string `json:"code"`
			NameZH    string `json:"name_zh"`
			Continent string `json:"continent"`
		}
		if err := json.Unmarshal(data, &cm); err == nil {
			for k, v := range cm {
				IsoCountryMap[k] = CountryMeta{
					Code:      v.Code,
					NameZH:    v.NameZH,
					Continent: v.Continent,
				}
			}
		}
	}

	// 4. 装载全球 Admin1 一级行政区划中英字母字典 (admin1CodesASCII_zh.json)
	if data, err := embeddedDataFS.ReadFile("data/admin1CodesASCII_zh.json"); err == nil && len(data) > 0 {
		var a1zh map[string]AdminCodeMeta
		if err := json.Unmarshal(data, &a1zh); err == nil {
			for k, v := range a1zh {
				Admin1ZHMap[k] = v

				// 检查中国条目并同步/校验与原来 ChinaAdmin1Map 的一致性
				if strings.HasPrefix(k, "CN.") {
					parts := strings.Split(k, ".")
					if len(parts) >= 2 {
						subCode := parts[1]
						if orig, exists := ChinaAdmin1Map[subCode]; exists {
							// 原有中文名存在，保持原有权威中文名
							if orig != v.NameZH && orig != "" {
								v.NameZH = orig
								Admin1ZHMap[k] = v
							}
						} else {
							// 原有不存在则自动充实
							ChinaAdmin1Map[subCode] = v.NameZH
						}
					}
				}
			}
		}
	}

	// 5. 装载全球 Admin2 二级行政区划中英字母字典 (admin2Codes_zh.json)
	if data, err := embeddedDataFS.ReadFile("data/admin2Codes_zh.json"); err == nil && len(data) > 0 {
		var a2zh map[string]AdminCodeMeta
		if err := json.Unmarshal(data, &a2zh); err == nil {
			for k, v := range a2zh {
				Admin2ZHMap[k] = v

				// 检查中国条目并同步/校验与原来 ChinaAdmin2Map 的一致性
				if strings.HasPrefix(k, "CN.") {
					parts := strings.Split(k, ".")
					if len(parts) >= 3 {
						admin2Code := parts[2]
						if orig, exists := ChinaAdmin2Map[admin2Code]; exists {
							// 原有中文名存在，保持原有权威中文名
							if orig != v.NameZH && orig != "" {
								v.NameZH = orig
								Admin2ZHMap[k] = v
							}
						} else {
							// 原有不存在则自动充实
							ChinaAdmin2Map[admin2Code] = v.NameZH
						}
					}
				}
			}
		}
	}
}

// LoadMappingFilesFromDir 从指定目录装载 GeoNames 原始映射源文件 (admin2Codes.txt, admin1CodesASCII.txt) 与自定义 json
func LoadMappingFilesFromDir(dir string) {
	if dir == "" {
		return
	}

	// 1. 优先检查并装载该目录下的 admin1CodesASCII_zh.json 与 admin2Codes_zh.json
	userAdmin1ZH := filepath.Join(dir, "admin1CodesASCII_zh.json")
	if data, err := os.ReadFile(userAdmin1ZH); err == nil {
		var a1zh map[string]AdminCodeMeta
		if err := json.Unmarshal(data, &a1zh); err == nil {
			mappingMu.Lock()
			for k, v := range a1zh {
				Admin1ZHMap[k] = v
				if strings.HasPrefix(k, "CN.") {
					parts := strings.Split(k, ".")
					if len(parts) >= 2 && v.NameZH != "" {
						ChinaAdmin1Map[parts[1]] = v.NameZH
					}
				}
			}
			mappingMu.Unlock()
		}
	}

	userAdmin2ZH := filepath.Join(dir, "admin2Codes_zh.json")
	if data, err := os.ReadFile(userAdmin2ZH); err == nil {
		var a2zh map[string]AdminCodeMeta
		if err := json.Unmarshal(data, &a2zh); err == nil {
			mappingMu.Lock()
			for k, v := range a2zh {
				Admin2ZHMap[k] = v
				if strings.HasPrefix(k, "CN.") {
					parts := strings.Split(k, ".")
					if len(parts) >= 3 && v.NameZH != "" {
						ChinaAdmin2Map[parts[2]] = v.NameZH
					}
				}
			}
			mappingMu.Unlock()
		}
	}

	// 2. 检查 mappings/ 目录下的自定义覆写 json
	userMapDir := filepath.Join(dir, "mappings")
	userAdmin2 := filepath.Join(userMapDir, "admin2_codes.json")
	if data, err := os.ReadFile(userAdmin2); err == nil {
		var userA2 map[string]string
		if err := json.Unmarshal(data, &userA2); err == nil {
			mappingMu.Lock()
			for k, v := range userA2 {
				ChinaAdmin2Map[k] = v
			}
			mappingMu.Unlock()
		}
	}

	// 3. 装载官方下载的 admin2Codes.txt
	admin2Txt := filepath.Join(dir, "admin2Codes.txt")
	if file, err := os.Open(admin2Txt); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		mappingMu.Lock()
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cols := strings.Split(line, "\t")
			if len(cols) >= 2 {
				codeKey := cols[0] // e.g. "CN.13.6531"
				name := cols[1]
				asciiName := name
				if len(cols) >= 3 {
					asciiName = cols[2]
				}

				// 到 Admin2ZHMap 字典中查找中文是否存在
				nameZH := name
				if meta, ok := Admin2ZHMap[codeKey]; ok && meta.NameZH != "" {
					nameZH = meta.NameZH
				} else {
					Admin2ZHMap[codeKey] = AdminCodeMeta{
						Code:      codeKey,
						Name:      name,
						NameASCII: asciiName,
						NameZH:    name,
					}
				}

				parts := strings.Split(codeKey, ".")
				if len(parts) >= 3 && parts[0] == "CN" {
					admin2Code := parts[2]
					// 如果已内置中文译名则优先保留中文，否则使用源文件/字典译名
					if _, ok := ChinaAdmin2Map[admin2Code]; !ok {
						ChinaAdmin2Map[admin2Code] = nameZH
					}
				}
			}
		}
		mappingMu.Unlock()
	}

	// 4. 装载官方下载的 admin1CodesASCII.txt
	admin1Txt := filepath.Join(dir, "admin1CodesASCII.txt")
	if file, err := os.Open(admin1Txt); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		mappingMu.Lock()
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cols := strings.Split(line, "\t")
			if len(cols) >= 2 {
				codeKey := cols[0] // e.g. "CN.13"
				name := cols[1]
				asciiName := name
				if len(cols) >= 3 {
					asciiName = cols[2]
				}

				// 到 Admin1ZHMap 字典中查找中文是否存在
				nameZH := name
				if meta, ok := Admin1ZHMap[codeKey]; ok && meta.NameZH != "" {
					nameZH = meta.NameZH
				} else {
					Admin1ZHMap[codeKey] = AdminCodeMeta{
						Code:      codeKey,
						Name:      name,
						NameASCII: asciiName,
						NameZH:    name,
					}
				}

				parts := strings.Split(codeKey, ".")
				if len(parts) >= 2 && parts[0] == "CN" {
					admin1Code := parts[1]
					if _, ok := ChinaAdmin1Map[admin1Code]; !ok {
						ChinaAdmin1Map[admin1Code] = nameZH
					}
				}
			}
		}
		mappingMu.Unlock()
	}
}

// GetAdmin1ZH 根据代码获取一级行政区划的完整元数据 (code, name, name_ascii, name_zh)
func GetAdmin1ZH(code string) (AdminCodeMeta, bool) {
	EnsureMappingsLoaded()
	c := strings.TrimSpace(code)
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	if meta, ok := Admin1ZHMap[c]; ok {
		return meta, true
	}
	if !strings.Contains(c, ".") {
		if meta, ok := Admin1ZHMap["CN."+c]; ok {
			return meta, true
		}
	}
	return AdminCodeMeta{}, false
}

// GetAdmin2ZH 根据代码获取二级行政区划的完整元数据 (code, name, name_ascii, name_zh)
func GetAdmin2ZH(code string) (AdminCodeMeta, bool) {
	EnsureMappingsLoaded()
	c := strings.TrimSpace(code)
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	if meta, ok := Admin2ZHMap[c]; ok {
		return meta, true
	}
	// 支持 4 位地级市代码检索 (如 "6531" -> "CN.13.6531")
	if !strings.Contains(c, ".") {
		for k, meta := range Admin2ZHMap {
			if strings.HasSuffix(k, "."+c) {
				return meta, true
			}
		}
		if name, ok := ChinaAdmin2Map[c]; ok && name != "" {
			return AdminCodeMeta{
				Code:   c,
				Name:   name,
				NameZH: name,
			}, true
		}
	}
	return AdminCodeMeta{}, false
}

// GetAdmin1NameZH 获取一级行政区划的中文名称，若不存在返回默认值或原始代码
func GetAdmin1NameZH(code string) string {
	EnsureMappingsLoaded()
	c := strings.TrimSpace(code)
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	if meta, ok := Admin1ZHMap[c]; ok && meta.NameZH != "" {
		return meta.NameZH
	}
	if name, ok := ChinaAdmin1Map[c]; ok && name != "" {
		return name
	}
	if !strings.Contains(c, ".") {
		if meta, ok := Admin1ZHMap["CN."+c]; ok && meta.NameZH != "" {
			return meta.NameZH
		}
	}
	return c
}

// GetAdmin2NameZH 获取二级行政区划的中文名称，若不存在返回默认值或原始代码
func GetAdmin2NameZH(code string) string {
	EnsureMappingsLoaded()
	c := strings.TrimSpace(code)
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	if meta, ok := Admin2ZHMap[c]; ok && meta.NameZH != "" {
		return meta.NameZH
	}
	if name, ok := ChinaAdmin2Map[c]; ok && name != "" {
		return name
	}
	if !strings.Contains(c, ".") {
		for k, meta := range Admin2ZHMap {
			if strings.HasSuffix(k, "."+c) && meta.NameZH != "" {
				return meta.NameZH
			}
		}
	}
	return c
}

// GetAllAdmin1ZH 获取当前加载的所有一级行政区划元数据字典快照
func GetAllAdmin1ZH() map[string]AdminCodeMeta {
	EnsureMappingsLoaded()
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	res := make(map[string]AdminCodeMeta, len(Admin1ZHMap))
	maps.Copy(res, Admin1ZHMap)
	return res
}

// GetAllAdmin2ZH 获取当前加载的所有二级行政区划元数据字典快照
func GetAllAdmin2ZH() map[string]AdminCodeMeta {
	EnsureMappingsLoaded()
	mappingMu.RLock()
	defer mappingMu.RUnlock()

	res := make(map[string]AdminCodeMeta, len(Admin2ZHMap))
	maps.Copy(res, Admin2ZHMap)
	return res
}

// GetCountryNameZH 获取 ISO 国家两字母代码对应的标准中文名称 (如 "CN" -> "中国", "US" -> "美国")
func GetCountryNameZH(code string) string {
	meta := GetCountryMeta(code)
	if meta.NameZH != "" {
		return meta.NameZH
	}
	return code
}
