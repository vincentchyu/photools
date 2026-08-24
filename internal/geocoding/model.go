package geocoding

import (
	"github.com/vincentchyu/photo-processing/internal/domain"
)

// LocationInfo 逆地理编码返回的地理位置详细信息 (类型别名映射到 domain.LocationInfo)
type LocationInfo = domain.LocationInfo

// GeoPoint 地理空间点位结构（完整映射 GeoNames 全量可用字段）
type GeoPoint struct {
	GeoNameID    int64   `json:"geoname_id,omitempty"`    // 0: geonameid 数据库主键 ID
	Name         string  `json:"name,omitempty"`          // 1: name 原始 UTF-8 名称
	NameASCII    string  `json:"name_ascii,omitempty"`    // 2: asciiname 纯 ASCII 名称
	NameZH       string  `json:"name_zh,omitempty"`       // 提取或匹配的中文名称
	Lat          float64 `json:"lat"`                     // 4: latitude 纬度 (WGS84)
	Lon          float64 `json:"lon"`                     // 5: longitude 经度 (WGS84)
	FeatureClass string  `json:"feature_class,omitempty"` // 6: feature class (A:行政区, P:城镇, T:山峰, H:水系, L:风景区, S:地标)
	FeatureCode  string  `json:"feature_code,omitempty"`  // 7: feature code (PPL, ADM1, MT, LK, PK 等)
	CountryCode  string  `json:"country_code"`            // 8: country code (ISO-3166 两字母)
	Country      string  `json:"country"`                 // 映射的中文国家名称 (如 中国, 日本)
	Admin1Code   string  `json:"admin1_code,omitempty"`   // 10: admin1 code (一级行政区代码)
	Admin2Code   string  `json:"admin2_code,omitempty"`   // 11: admin2 code (二级行政区代码)
	Province     string  `json:"province"`                // 省份/州
	City         string  `json:"city"`                    // 城市/地区/地标
	District     string  `json:"district,omitempty"`      // 区县/特定景区
	Population   int64   `json:"population,omitempty"`    // 14: population 人口数
	Elevation    int     `json:"elevation,omitempty"`     // 15: elevation 海拔 (米)
	DEM          int     `json:"dem,omitempty"`           // 16: dem 数字高程模型海拔 (米)
	Timezone     string  `json:"timezone,omitempty"`      // 17: timezone IANA 时区
	ModDate      string  `json:"mod_date,omitempty"`      // 18: modification date 记录最后修改日期 (YYYY-MM-DD)
	Source       string  `json:"source,omitempty"`        // 数据源标识 (如 geonames_china_ultra, geonames_asia)
}
