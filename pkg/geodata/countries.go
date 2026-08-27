package geodata

import (
	"math"
	"strings"
	"unicode"
)

// CountryMeta 国家元数据定义
type CountryMeta struct {
	Code      string `json:"code"`      // ISO 2位国家代码 (如 CN, JP, US)
	NameZH    string `json:"name_zh"`   // 中文规范国家名 (如 中国, 日本, 美国)
	Continent string `json:"continent"` // 所属大洲代码 (china, asia, europe, north-america, oceania, south-america, africa)
}

// GeoBBox 地理矩形边界 (MinLat, MaxLat, MinLon, MaxLon)
type GeoBBox struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// GeoPoint2D 地理坐标点
type GeoPoint2D struct {
	Lat float64
	Lon float64
}

// ChinaProvinceBBox 中国 34 省级行政区划地理边界包围盒 (以 FIPS/ISO/GB 省份代码为键)
var ChinaProvinceBBox = map[string]GeoBBox{
	"22": {MinLat: 39.4, MaxLat: 41.1, MinLon: 115.4, MaxLon: 117.6}, // 北京
	"28": {MinLat: 38.5, MaxLat: 40.3, MinLon: 116.7, MaxLon: 118.1}, // 天津
	"10": {MinLat: 36.0, MaxLat: 42.7, MinLon: 113.4, MaxLon: 119.9}, // 河北
	"24": {MinLat: 34.5, MaxLat: 40.8, MinLon: 110.2, MaxLon: 114.6}, // 山西
	"20": {MinLat: 37.4, MaxLat: 53.4, MinLon: 97.1, MaxLon: 126.1},  // 内蒙古
	"19": {MinLat: 38.7, MaxLat: 43.5, MinLon: 118.8, MaxLon: 125.8}, // 辽宁
	"05": {MinLat: 40.8, MaxLat: 46.3, MinLon: 121.6, MaxLon: 131.4}, // 吉林
	"08": {MinLat: 43.4, MaxLat: 53.6, MinLon: 121.1, MaxLon: 135.1}, // 黑龙江
	"23": {MinLat: 30.6, MaxLat: 31.9, MinLon: 120.8, MaxLon: 122.3}, // 上海
	"04": {MinLat: 30.7, MaxLat: 35.2, MinLon: 116.3, MaxLon: 122.0}, // 江苏
	"02": {MinLat: 27.0, MaxLat: 31.2, MinLon: 118.0, MaxLon: 123.0}, // 浙江
	"01": {MinLat: 29.4, MaxLat: 34.7, MinLon: 114.8, MaxLon: 119.7}, // 安徽
	"07": {MinLat: 23.5, MaxLat: 28.4, MinLon: 115.8, MaxLon: 120.8}, // 福建
	"03": {MinLat: 24.4, MaxLat: 30.1, MinLon: 113.5, MaxLon: 118.5}, // 江西
	"25": {MinLat: 34.3, MaxLat: 38.4, MinLon: 114.8, MaxLon: 122.8}, // 山东
	"09": {MinLat: 31.3, MaxLat: 36.4, MinLon: 110.3, MaxLon: 116.7}, // 河南
	"12": {MinLat: 29.0, MaxLat: 33.3, MinLon: 108.3, MaxLon: 116.2}, // 湖北
	"11": {MinLat: 24.6, MaxLat: 30.2, MinLon: 108.7, MaxLon: 114.3}, // 湖南
	"30": {MinLat: 20.2, MaxLat: 25.6, MinLon: 109.6, MaxLon: 117.4}, // 广东
	"16": {MinLat: 20.9, MaxLat: 26.4, MinLon: 104.4, MaxLon: 112.1}, // 广西
	"31": {MinLat: 3.5, MaxLat: 20.3, MinLon: 108.5, MaxLon: 117.5},  // 海南
	"33": {MinLat: 28.1, MaxLat: 32.3, MinLon: 105.2, MaxLon: 110.3}, // 重庆
	"32": {MinLat: 26.0, MaxLat: 34.4, MinLon: 97.3, MaxLon: 108.6},  // 四川
	"18": {MinLat: 24.6, MaxLat: 29.3, MinLon: 103.5, MaxLon: 109.6}, // 贵州
	"29": {MinLat: 21.1, MaxLat: 29.3, MinLon: 97.5, MaxLon: 106.2},  // 云南
	"14": {MinLat: 26.8, MaxLat: 36.6, MinLon: 78.4, MaxLon: 99.2},   // 西藏
	"26": {MinLat: 31.7, MaxLat: 39.6, MinLon: 105.4, MaxLon: 111.3}, // 陕西
	"15": {MinLat: 32.5, MaxLat: 42.8, MinLon: 92.2, MaxLon: 108.8},  // 甘肃
	"06": {MinLat: 31.6, MaxLat: 39.4, MinLon: 89.4, MaxLon: 103.1},  // 青海
	"21": {MinLat: 35.2, MaxLat: 39.4, MinLon: 104.2, MaxLon: 107.7}, // 宁夏
	"13": {MinLat: 34.3, MaxLat: 49.2, MinLon: 73.5, MaxLon: 96.4},   // 新疆
	"TW": {MinLat: 21.8, MaxLat: 25.4, MinLon: 119.3, MaxLon: 122.1}, // 台湾
	"HK": {MinLat: 22.1, MaxLat: 22.6, MinLon: 113.8, MaxLon: 114.4}, // 香港
	"MO": {MinLat: 22.1, MaxLat: 22.3, MinLon: 113.5, MaxLon: 113.6}, // 澳门
}

// ChinaProvinceCenters 各省中心坐标 (以 FIPS/ISO/GB 省份代码为键)
var ChinaProvinceCenters = map[string]GeoPoint2D{
	"22": {Lat: 39.9042, Lon: 116.4074},
	"28": {Lat: 39.0842, Lon: 117.2009},
	"10": {Lat: 38.0428, Lon: 114.5149},
	"24": {Lat: 37.8706, Lon: 112.5489},
	"20": {Lat: 40.8427, Lon: 111.7492},
	"19": {Lat: 41.8057, Lon: 123.4315},
	"05": {Lat: 43.8171, Lon: 125.3235},
	"08": {Lat: 45.8038, Lon: 126.5350},
	"23": {Lat: 31.2304, Lon: 121.4737},
	"04": {Lat: 32.0603, Lon: 118.7969},
	"02": {Lat: 30.2741, Lon: 120.1551},
	"01": {Lat: 31.8206, Lon: 117.2272},
	"07": {Lat: 26.0745, Lon: 119.2965},
	"03": {Lat: 28.6820, Lon: 115.8579},
	"25": {Lat: 36.6512, Lon: 117.1201},
	"09": {Lat: 34.7466, Lon: 113.6253},
	"12": {Lat: 30.5928, Lon: 114.3055},
	"11": {Lat: 28.2282, Lon: 112.9388},
	"30": {Lat: 23.1291, Lon: 113.2644},
	"16": {Lat: 22.8170, Lon: 108.3665},
	"31": {Lat: 20.0440, Lon: 110.1999},
	"33": {Lat: 29.5630, Lon: 106.5516},
	"32": {Lat: 30.5728, Lon: 104.0668},
	"18": {Lat: 26.6470, Lon: 106.6302},
	"29": {Lat: 25.0406, Lon: 102.7123},
	"14": {Lat: 29.6500, Lon: 91.1000},
	"26": {Lat: 34.3416, Lon: 108.9398},
	"15": {Lat: 36.0611, Lon: 103.8343},
	"06": {Lat: 36.6171, Lon: 101.7782},
	"21": {Lat: 38.4872, Lon: 106.2309},
	"13": {Lat: 43.8256, Lon: 87.6168},
	"TW": {Lat: 25.0330, Lon: 121.5654},
	"HK": {Lat: 22.3193, Lon: 114.1694},
	"MO": {Lat: 22.1987, Lon: 113.5439},
}

// GetCountryMeta 获取国家元信息
func GetCountryMeta(countryCode string) CountryMeta {
	EnsureMappingsLoaded()
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	mappingMu.RLock()
	meta, ok := IsoCountryMap[code]
	mappingMu.RUnlock()
	if ok {
		return meta
	}
	return CountryMeta{
		Code:      code,
		NameZH:    code,
		Continent: "other",
	}
}

// GetChinaProvinceName 将 GeoNames 的 admin1 代码转换为规范省份名，并结合经纬度空间边界进行几何校验与自动纠错
func GetChinaProvinceName(admin1Code string, lat, lon float64) string {
	EnsureMappingsLoaded()
	code := strings.TrimSpace(admin1Code)

	cleanCode := code
	if after, ok := strings.CutPrefix(cleanCode, "CN."); ok {
		cleanCode = after
	} else if len(cleanCode) == 1 && cleanCode >= "0" && cleanCode <= "9" {
		cleanCode = "0" + cleanCode
	}

	resolveName := func(c string) string {
		mappingMu.RLock()
		defer mappingMu.RUnlock()
		if name, ok := ChinaAdmin1Map[c]; ok && name != "" {
			return name
		}
		if name, ok := ChinaAdmin1Map["CN."+c]; ok && name != "" {
			return name
		}
		if meta, ok := Admin1ZHMap["CN."+c]; ok && meta.NameZH != "" {
			return meta.NameZH
		}
		return c
	}

	// 1. 如果经纬度有效，进行空间校验与纠错
	if lat != 0 || lon != 0 {
		// 如果 cleanCode 与经纬度范围完全匹配，直接返回
		if bbox, ok := ChinaProvinceBBox[cleanCode]; ok {
			if lat >= bbox.MinLat && lat <= bbox.MaxLat && lon >= bbox.MinLon && lon <= bbox.MaxLon {
				return resolveName(cleanCode)
			}
		}

		// 2. cleanCode 为空或空间越界（如 GeoNames 脏数据导致 Xinjiang 错标为 08），基于经纬度空间边界重新定位
		var candidates []string
		for provCode, bbox := range ChinaProvinceBBox {
			if lat >= bbox.MinLat && lat <= bbox.MaxLat && lon >= bbox.MinLon && lon <= bbox.MaxLon {
				candidates = append(candidates, provCode)
			}
		}

		if len(candidates) == 1 {
			return resolveName(candidates[0])
		}
		if len(candidates) > 1 {
			bestCode := candidates[0]
			bestDist := 1e9
			for _, p := range candidates {
				if center, ok := ChinaProvinceCenters[p]; ok {
					d := math.Hypot(lat-center.Lat, lon-center.Lon)
					if d < bestDist {
						bestDist = d
						bestCode = p
					}
				}
			}
			return resolveName(bestCode)
		}

		// 3. 不在任何矩形框内，匹配距离最近的省份中心
		bestCode := "13"
		bestDist := 1e9
		for provCode, center := range ChinaProvinceCenters {
			d := math.Hypot(lat-center.Lat, lon-center.Lon)
			if d < bestDist {
				bestDist = d
				bestCode = provCode
			}
		}
		return resolveName(bestCode)
	}

	return resolveName(cleanCode)
}

// GetChinaCityName 根据 GeoNames admin2 代码解析规范中文地级市/地区/自治州/盟名称
func GetChinaCityName(admin2Code, province, featureClass, featureCode, pointNameZH string, lat, lon float64) string {
	EnsureMappingsLoaded()
	code := strings.TrimSpace(admin2Code)
	if code != "" && code != "00" {
		mappingMu.RLock()
		name, ok := ChinaAdmin2Map[code]
		if !ok && len(code) >= 4 {
			name, ok = ChinaAdmin2Map[code[:4]]
		}
		mappingMu.RUnlock()

		if ok && name != "" {
			return name
		}
	}

	// 直辖市/特别行政区兜底
	switch province {
	case "北京市", "上海市", "天津市", "重庆市":
		return province
	case "香港特别行政区":
		return "香港"
	case "澳门特别行政区":
		return "澳门"
	}

	// 若当前点位本身就是城市/行政中心类点位 (PPLC/PPLA/PPLA2/ADM2 等)
	if featureClass == "P" || featureClass == "A" {
		switch featureCode {
		case "PPLC", "PPLA", "PPLA2", "PPLA3", "PPLA4", "PPL", "ADM2":
			if pointNameZH != "" && pointNameZH != province {
				return pointNameZH
			}
		}
	}

	return ""
}

// ExtractChineseName 从 GeoNames 逗号分隔的别名中提取中文地名
func ExtractChineseName(asciiname, altNames string) string {
	// 如果 asciiname 本身就是中文
	if isAllChinese(asciiname) {
		return asciiname
	}

	if altNames == "" {
		return asciiname
	}

	parts := strings.Split(altNames, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if isAllChinese(trimmed) && len([]rune(trimmed)) >= 2 {
			return trimmed
		}
	}

	return asciiname
}

// isAllChinese 判断字符串是否主要由中文字符组成
func isAllChinese(s string) bool {
	if len(s) == 0 {
		return false
	}
	chineseCount := 0
	totalCount := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
		totalCount++
	}
	return totalCount > 0 && float64(chineseCount)/float64(totalCount) >= 0.8
}
