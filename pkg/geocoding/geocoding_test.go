package geocoding

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReverseGeocoder_Lookup(t *testing.T) {
	rg := NewReverseGeocoder()

	tests := []struct {
		name         string
		lat          float64
		lon          float64
		wantCountry  string
		wantProvince string
		wantCity     string
		maxDistKm    float64
	}{
		{
			name:         "北京故宫附近",
			lat:          39.9165,
			lon:          116.3971,
			wantCountry:  "中国",
			wantProvince: "北京市",
			wantCity:     "东城区",
			maxDistKm:    5.0,
		},
		{
			name:         "成都宽窄巷子",
			lat:          30.6695,
			lon:          104.0530,
			wantCountry:  "中国",
			wantProvince: "四川省",
			wantCity:     "成都市",
			maxDistKm:    5.0,
		},
		{
			name:         "九寨沟景区",
			lat:          33.2600,
			lon:          103.9200,
			wantCountry:  "中国",
			wantProvince: "四川省",
			wantCity:     "阿坝藏族羌族自治州",
			maxDistKm:    10.0,
		},
		{
			name:         "稻城亚丁三神山",
			lat:          29.0040,
			lon:          100.3020,
			wantCountry:  "中国",
			wantProvince: "四川省",
			wantCity:     "甘孜藏族自治州",
			maxDistKm:    10.0,
		},
		{
			name:         "新疆喀纳斯湖/禾木村",
			lat:          48.5800,
			lon:          87.0200,
			wantCountry:  "中国",
			wantProvince: "新疆维吾尔自治区",
			wantCity:     "阿勒泰地区",
			maxDistKm:    15.0,
		},
		{
			name:         "日本东京涩谷",
			lat:          35.6585,
			lon:          139.7013,
			wantCountry:  "日本",
			wantProvince: "东京都",
			wantCity:     "涩谷区",
			maxDistKm:    5.0,
		},
		{
			name:         "日本富士山顶附近",
			lat:          35.3610,
			lon:          138.7280,
			wantCountry:  "日本",
			wantProvince: "静冈县/山梨县",
			wantCity:     "富士吉田市",
			maxDistKm:    5.0,
		},
		{
			name:         "泰国清迈古城",
			lat:          18.7880,
			lon:          98.9850,
			wantCountry:  "泰国",
			wantProvince: "清迈府",
			wantCity:     "清迈市",
			maxDistKm:    5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := rg.Lookup(tt.lat, tt.lon)
			if loc == nil {
				t.Fatalf("Lookup(%f, %f) returned nil", tt.lat, tt.lon)
			}
			if loc.Country != tt.wantCountry {
				t.Errorf("Country: got %q, want %q", loc.Country, tt.wantCountry)
			}
			if loc.Province != tt.wantProvince {
				t.Errorf("Province: got %q, want %q", loc.Province, tt.wantProvince)
			}
			if loc.City != tt.wantCity {
				t.Errorf("City: got %q, want %q", loc.City, tt.wantCity)
			}
			if loc.DistanceKm > tt.maxDistKm {
				t.Errorf("DistanceKm: got %f, want <= %f", loc.DistanceKm, tt.maxDistKm)
			}
		})
	}
}

func TestReverseGeocoder_CustomPlacesAndGeoPacks(t *testing.T) {
	tempDir := t.TempDir()

	// 1. 测试自定义地标加载
	customJSON := `[
		{
			"lat": 30.1234,
			"lon": 104.5678,
			"country": "中国",
			"country_code": "CN",
			"province": "四川省",
			"city": "成都市",
			"district": "私密星空露营地",
			"source": "my_secret_spot"
		}
	]`
	customFile := filepath.Join(tempDir, "places.json")
	if err := os.WriteFile(customFile, []byte(customJSON), 0644); err != nil {
		t.Fatal(err)
	}

	rg := NewReverseGeocoder()
	initialSize := rg.Size()

	if err := rg.LoadCustomPlaces(customFile); err != nil {
		t.Fatalf("LoadCustomPlaces failed: %v", err)
	}

	if rg.Size() != initialSize+1 {
		t.Fatalf("expected size %d, got %d", initialSize+1, rg.Size())
	}

	loc := rg.Lookup(30.1234, 104.5678)
	if loc == nil || loc.District != "私密星空露营地" {
		t.Errorf("expected District 私密星空露营地, got %+v", loc)
	}

	// 2. 测试外挂大洲数据包目录扫描加载
	packDir := filepath.Join(tempDir, "geodata")
	if err := os.MkdirAll(packDir, 0755); err != nil {
		t.Fatal(err)
	}

	packJSON := `[
		{
			"lat": 48.8584,
			"lon": 2.2945,
			"country": "法国",
			"country_code": "FR",
			"province": "法兰西岛",
			"city": "巴黎",
			"district": "埃菲尔铁塔"
		}
	]`
	packFile := filepath.Join(packDir, "europe.json")
	if err := os.WriteFile(packFile, []byte(packJSON), 0644); err != nil {
		t.Fatal(err)
	}

	if err := rg.LoadGeoPackDirectory(packDir); err != nil {
		t.Fatalf("LoadGeoPackDirectory failed: %v", err)
	}

	stats := rg.GetStats()
	if len(stats.Packs) != 1 || stats.Packs[0].Name != "europe.json" {
		t.Errorf("expected 1 pack named europe.json, got %+v", stats.Packs)
	}

	frLoc := rg.Lookup(48.8584, 2.2945)
	if frLoc == nil || frLoc.Country != "法国" || frLoc.City != "巴黎" {
		t.Errorf("expected 法国/巴黎, got %+v", frLoc)
	}
}
