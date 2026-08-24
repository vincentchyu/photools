package domain

import (
	"strings"
)

// LocationInfo 逆地理编码返回的地理位置详细信息
type LocationInfo struct {
	Country      string  `json:"country"`                 // 国家 (如: 中国 / 日本 / 法国)
	CountryCode  string  `json:"country_code"`            // 国家代码 (如: CN / JP / FR)
	Province     string  `json:"province"`                // 省份/州 (如: 新疆维吾尔自治区 / 东京都)
	City         string  `json:"city"`                    // 城市/地区 (如: 柯尔干布拉克村 / 涉谷区)
	District     string  `json:"district,omitempty"`      // 区县/特定景区 (如: 九寨沟景区 / 宽窄巷子)
	Timezone     string  `json:"timezone,omitempty"`      // IANA 时区 (如: Asia/Shanghai, Asia/Urumqi)
	Elevation    int     `json:"elevation,omitempty"`     // 海拔高度 (米)
	FeatureCode  string  `json:"feature_code,omitempty"`  // 地理特征代码 (如 PPL, ADM1, MT, LK 等)
	FeatureClass string  `json:"feature_class,omitempty"` // 地理特征分类 (A, P, T, H, L, S)
	GeoNameID    int64   `json:"geoname_id,omitempty"`    // GeoNames 数据库唯一主键 ID
	DistanceKm   float64 `json:"distance_km"`             // 距离参考点的物理大圆距离 (km)
	Source       string  `json:"source"`                  // 数据源标识 (如: "geonames_china_ultra", "builtin_asia")
}

// FormatSummary 格式化为便于展示的地名简述
func (l *LocationInfo) FormatSummary() string {
	if l == nil {
		return ""
	}
	parts := []string{}
	if l.Country != "" {
		parts = append(parts, l.Country)
	}
	if l.Province != "" && l.Province != l.City {
		parts = append(parts, l.Province)
	}
	if l.City != "" {
		parts = append(parts, l.City)
	}
	if l.District != "" && l.District != l.City {
		parts = append(parts, l.District)
	}
	return strings.Join(parts, " · ")
}
