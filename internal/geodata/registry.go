package geodata

import (
	"strings"

	"github.com/vincentchyu/photo-processing/internal/geocoding"
)

// ContinentMeta 大洲与国家数据包元信息
type ContinentMeta struct {
	Code         string               `json:"code"`          // 唯一标识 (如 china, asia, europe, north-america)
	NameZH       string               `json:"name_zh"`       // 中文名 (如 中国全境高精, 亚洲, 欧洲)
	Aliases      []string             `json:"aliases"`       // 匹配别名 (如 cn, china, 亚洲)
	Description  string               `json:"description"`   // 说明
	ApproxPoints int                  `json:"approx_points"` // 大致点位数
	ApproxSize   string               `json:"approx_size"`   // 大致大小
	DownloadURLs []string             `json:"download_urls"` // 官方/镜像下载地址列表
	BasePoints   []geocoding.GeoPoint `json:"-"`             // 离线基础精选包 (网络不可达时自动就地回退)
}

// AvailableContinents 所有支持的大洲与区域数据包清单
var AvailableContinents = []ContinentMeta{
	{
		Code:         "china",
		NameZH:       "中国全境高精 (China Ultra)",
		Aliases:      []string{"cn", "china", "中国", "新疆", "西藏", "川西", "阿勒泰", "伊犁"},
		Description:  "覆盖中国34省市区县全量行政区划、阿勒泰/伊犁/喀纳斯/赛里木湖等深度摄影自然地标",
		ApproxPoints: 32000,
		ApproxSize:   "~5.2 MB",
		DownloadURLs: []string{
			GeoNamesChinaURL,
		},
	},
	{
		Code:         "asia",
		NameZH:       "亚洲扩展 (Asia)",
		Aliases:      []string{"as", "asia", "亚洲"},
		Description:  "覆盖中国各省市区县深度地名、日韩、东南亚、南亚及中亚全量城镇扩展包",
		ApproxPoints: 10700,
		ApproxSize:   "~1.8 MB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
	},
	{
		Code:         "europe",
		NameZH:       "欧洲 (Europe)",
		Aliases:      []string{"eu", "europe", "欧洲", "欧陆"},
		Description:  "覆盖英国、法国、德国、意大利、瑞士阿尔卑斯、冰岛极光区、西班牙等欧洲全境",
		ApproxPoints: 7900,
		ApproxSize:   "~1.4 MB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
		BasePoints: curatedEuropePoints,
	},
	{
		Code:         "north-america",
		NameZH:       "北美洲 (North America)",
		Aliases:      []string{"na", "north-america", "northamerica", "北美", "北美洲", "usa", "canada"},
		Description:  "覆盖美国 50 州、黄石/大峡谷/优胜美地等国家公园、加拿大班夫/贾斯珀等",
		ApproxPoints: 4700,
		ApproxSize:   "~890 KB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
		BasePoints: curatedNorthAmericaPoints,
	},
	{
		Code:         "oceania",
		NameZH:       "大洋洲 (Oceania & NZ)",
		Aliases:      []string{"oc", "oceania", "大洋洲", "澳新", "australia", "new-zealand"},
		Description:  "覆盖澳大利亚大洋路/大堡礁/悉尼/墨尔本、新西兰南岛皇后镇/米尔福德峡湾等",
		ApproxPoints: 400,
		ApproxSize:   "~75 KB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
		BasePoints: curatedOceaniaPoints,
	},
	{
		Code:         "south-america",
		NameZH:       "南美洲 (South America)",
		Aliases:      []string{"sa", "south-america", "southamerica", "南美", "南美洲"},
		Description:  "覆盖秘鲁马丘比丘/库斯科、玻利维亚乌尤尼天空之镜、巴塔哥尼亚/阿根廷/智利等",
		ApproxPoints: 3700,
		ApproxSize:   "~700 KB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
		BasePoints: curatedSouthAmericaPoints,
	},
	{
		Code:         "africa",
		NameZH:       "非洲 (Africa)",
		Aliases:      []string{"af", "africa", "非洲"},
		Description:  "覆盖埃及金字塔/卢克索、肯尼亚马赛马拉动物大迁徙、坦桑尼亚塞伦盖蒂、南非开普敦等",
		ApproxPoints: 2000,
		ApproxSize:   "~360 KB",
		DownloadURLs: []string{
			GeoNamesCities15000URL,
		},
		BasePoints: curatedAfricaPoints,
	},
}

// FindContinent 根据名称或别名查找大洲/国家元信息
func FindContinent(nameOrAlias string) *ContinentMeta {
	cleaned := strings.ToLower(strings.TrimSpace(nameOrAlias))
	if cleaned == "" {
		return nil
	}

	for i := range AvailableContinents {
		meta := &AvailableContinents[i]
		if strings.EqualFold(meta.Code, cleaned) {
			return meta
		}
		for _, alias := range meta.Aliases {
			if strings.EqualFold(alias, cleaned) {
				return meta
			}
		}
	}
	return nil
}
