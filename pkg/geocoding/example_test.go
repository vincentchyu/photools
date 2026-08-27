package geocoding_test

import (
	"fmt"

	"github.com/vincentchyu/photools/pkg/geocoding"
)

func ExampleReverseGeocoder_Lookup() {
	// 初始化基础逆地理编码器（内置点位）
	geocoder := geocoding.NewReverseGeocoder()

	// 查询北京故宫经纬度
	loc := geocoder.Lookup(39.9165, 116.3971)
	if loc != nil {
		fmt.Printf("国家: %s\n", loc.Country)
		fmt.Printf("省份: %s\n", loc.Province)
		fmt.Printf("城市: %s\n", loc.City)
	}

	// Output:
	// 国家: 中国
	// 省份: 北京市
	// 城市: 东城区
}

func ExampleLocationInfo_FormatSummary() {
	loc := &geocoding.LocationInfo{
		Country:  "中国",
		Province: "四川省",
		City:     "成都市",
		District: "青羊区",
	}

	fmt.Println(loc.FormatSummary())

	// Output:
	// 中国 · 四川省 · 成都市 · 青羊区
}
