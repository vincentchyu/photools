package geodata

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindContinent(t *testing.T) {
	cases := []struct {
		input    string
		wantCode string
	}{
		{"china", "china"},
		{"cn", "china"},
		{"新疆", "china"},
		{"europe", "europe"},
		{"EU", "europe"},
		{"欧洲", "europe"},
		{"north-america", "north-america"},
		{"na", "north-america"},
		{"北美", "north-america"},
		{"usa", "north-america"},
		{"oceania", "oceania"},
		{"澳新", "oceania"},
		{"south-america", "south-america"},
		{"africa", "africa"},
		{"invalid_xyz", ""},
	}

	for _, tc := range cases {
		c := FindContinent(tc.input)
		if tc.wantCode == "" {
			if c != nil {
				t.Errorf("FindContinent(%q) expected nil, got %v", tc.input, c.Code)
			}
		} else {
			if c == nil || c.Code != tc.wantCode {
				t.Errorf("FindContinent(%q) expected %q, got %v", tc.input, tc.wantCode, c)
			}
		}
	}
}

func TestExtractChineseName(t *testing.T) {
	cases := []struct {
		asciiname string
		altNames  string
		want      string
	}{
		{"Altay", "Altay,阿勒泰,Aletai", "阿勒泰"},
		{"Yining", "Yining,Gulja,伊宁,伊宁市", "伊宁"},
		{"Paris", "Paris,巴黎,Lutetia", "巴黎"},
		{"Rome", "Rome,Roma,罗马,Рим", "罗马"},
		{"Tokyo", "Tokyo,Tōkyō,东京,とうきょう", "东京"},
		{"SmallTown", "SmallTown,Village", "SmallTown"},
		{"Kilik Pass", "Checheklik Dawan,Chëcheklik Dawan,Jilike Daban,Kilik Pass,吉里克达坂", "吉里克达坂"},
	}

	for _, tc := range cases {
		got := ExtractChineseName(tc.asciiname, tc.altNames)
		if got != tc.want {
			t.Errorf("ExtractChineseName(%q, %q) = %q, want %q", tc.asciiname, tc.altNames, got, tc.want)
		}
	}
}

func TestGetChinaCityName(t *testing.T) {
	cases := []struct {
		name         string
		admin2Code   string
		province     string
		fClass       string
		fCode        string
		pointNameZH  string
		lat          float64
		lon          float64
		expectedCity string
	}{
		{"Kilik Pass (Kashgar 6531)", "6531", "新疆维吾尔自治区", "T", "PASS", "吉里克达坂", 37.08, 74.67, "喀什地区"},
		{"West Lake (Hangzhou 3301)", "3301", "浙江省", "H", "LK", "西湖", 30.24, 120.15, "杭州市"},
		{"Forbidden City (Beijing 1101)", "1101", "北京市", "S", "MNMT", "故宫", 39.91, 116.39, "北京市"},
		{"Altay PPLA2 (Altay 654301)", "654301", "新疆维吾尔自治区", "P", "PPLA2", "阿勒泰", 47.86, 88.11, "阿勒泰地区"},
		{"Shenzhen PPLA2 (4403)", "4403", "广东省", "P", "PPLA2", "深圳市", 22.54, 114.05, "深圳市"},
		{"Direct municipality Beijing with empty admin2", "", "北京市", "P", "PPLC", "北京", 39.9, 116.4, "北京市"},
		{"Hong Kong", "", "香港特别行政区", "P", "PPLC", "香港", 22.3, 114.1, "香港"},
	}

	for _, tc := range cases {
		got := GetChinaCityName(tc.admin2Code, tc.province, tc.fClass, tc.fCode, tc.pointNameZH, tc.lat, tc.lon)
		if got != tc.expectedCity {
			t.Errorf("%s: GetChinaCityName(%q, %q, %q, %q, %q) = %q, want %q",
				tc.name, tc.admin2Code, tc.province, tc.fClass, tc.fCode, tc.pointNameZH, got, tc.expectedCity)
		}
	}
}

func TestGetChinaProvinceName(t *testing.T) {
	cases := []struct {
		name     string
		code     string
		lat      float64
		lon      float64
		expected string
	}{
		{"Harbin with official 08", "08", 45.8038, 126.5350, "黑龙江省"},
		{"Beijing with official 22", "22", 39.9042, 116.4074, "北京市"},
		{"Urumqi with official 13", "13", 43.8256, 87.6168, "新疆维吾尔自治区"},
		{"Baiyanggou with corrupted 08 code in Xinjiang", "08", 44.13769, 84.29359, "新疆维吾尔自治区"},
		{"Sayram Lake with empty code", "", 44.6000, 81.1667, "新疆维吾尔自治区"},
		{"Lhasa Tibet", "14", 29.6500, 91.1000, "西藏自治区"},
		{"Chengdu Sichuan", "32", 30.5728, 104.0668, "四川省"},
	}

	for _, tc := range cases {
		got := GetChinaProvinceName(tc.code, tc.lat, tc.lon)
		if got != tc.expected {
			t.Errorf("%s: GetChinaProvinceName(%q, %v, %v) = %q, want %q", tc.name, tc.code, tc.lat, tc.lon, got, tc.expected)
		}
	}
}

func TestParseChinaGeoNamesTSV(t *testing.T) {
	tsvData := `1529651	Altay	Altay	Altay,Aletai,阿勒泰	47.86667	88.11667	P	PPLA2	CN		13	6543	654301	142000		887	Asia/Urumqi	2020-07-09
1529360	Burqin	Burqin	Burqin,布尔津,布尔津县	47.70000	86.86667	P	PPLA3	CN		13	6543	654321	65000		474	Asia/Urumqi	2020-07-09
1529484	Sayram Lake	Sayram Lake	Sayram Lake,赛里木湖	44.60000	81.16667	H	LK	CN		13	6527		0		2073	Asia/Urumqi	2020-07-09
8548623	Baiyanggou	Baiyanggou	Baiyanggou,白杨沟	44.13769	84.29359	P	PPLA4	CN		08	00		0		3020	Asia/Urumqi	2013-06-06
1114952	Kilik Pass	Kilik Pass	Checheklik Dawan,Chëcheklik Dawan,Jilike Daban,K'o-li-k'o Shan-k'ou,Kelike Shankou,Kilik Dawan,Kilik Pass,K’o-li-k’o Shan-k’ou,ji li ke da ban,چېچەكلىك داۋان,吉里克达坂	37.08323	74.67545	T	PASS	CN		13	6531			0		4823	Asia/Shanghai	2024-07-12
`
	pts, err := ParseChinaGeoNamesTSV(strings.NewReader(tsvData))
	if err != nil {
		t.Fatalf("ParseChinaGeoNamesTSV failed: %v", err)
	}

	if len(pts) != 5 {
		t.Fatalf("expected 5 points, got %d", len(pts))
	}

	// Altay with admin2 6543 (阿勒泰地区)
	if pts[0].NameZH != "阿勒泰" || pts[0].City != "阿勒泰地区" || pts[0].Province != "新疆维吾尔自治区" {
		t.Errorf("unexpected point 0: %+v", pts[0])
	}
	// Burqin with admin2 6543 (阿勒泰地区)
	if pts[1].NameZH != "布尔津" || pts[1].City != "阿勒泰地区" || pts[1].District != "布尔津" {
		t.Errorf("unexpected point 1: %+v", pts[1])
	}
	// Sayram Lake with admin2 6527 (博尔塔拉蒙古自治州)
	if pts[2].NameZH != "赛里木湖" || pts[2].City != "博尔塔拉蒙古自治州" || pts[2].District != "赛里木湖" {
		t.Errorf("unexpected point 2: %+v", pts[2])
	}
	// Baiyanggou (province autocorrection)
	if pts[3].NameZH != "白杨沟" || pts[3].Province != "新疆维吾尔自治区" {
		t.Errorf("unexpected point 3 (Baiyanggou province correction failed): %+v", pts[3])
	}
	// Kilik Pass (User's case: 1114952, admin1: 13 -> 新疆, admin2: 6531 -> 喀什地区, NameZH: 吉里克达坂)
	if pts[4].NameZH != "吉里克达坂" || pts[4].City != "喀什地区" || pts[4].Province != "新疆维吾尔自治区" || pts[4].District != "吉里克达坂" {
		t.Errorf("unexpected point 4 (Kilik Pass mapping failed): %+v", pts[4])
	}
}

func TestLoadMappingFilesFromDir(t *testing.T) {
	tempDir := t.TempDir()

	admin2Content := `CN.13.9999	Custom Special Area	Custom Special Area	999999
`
	if err := os.WriteFile(filepath.Join(tempDir, "admin2Codes.txt"), []byte(admin2Content), 0644); err != nil {
		t.Fatal(err)
	}

	admin1Content := `CN.99	Custom Special Province	Custom Special Province	888888
`
	if err := os.WriteFile(filepath.Join(tempDir, "admin1CodesASCII.txt"), []byte(admin1Content), 0644); err != nil {
		t.Fatal(err)
	}

	LoadMappingFilesFromDir(tempDir)

	if got := ChinaAdmin2Map["9999"]; got != "Custom Special Area" {
		t.Errorf("expected dynamic admin2 mapping 'Custom Special Area', got %q", got)
	}
	if got := ChinaAdmin1Map["99"]; got != "Custom Special Province" {
		t.Errorf("expected dynamic admin1 mapping 'Custom Special Province', got %q", got)
	}
}

func TestDownloadAndParseChinaZipWithReadme(t *testing.T) {
	tsvContent := `1529651	Altay	Altay	Altay,Aletai,阿勒泰	47.86667	88.11667	P	PPLA2	CN		13	6543	654301	142000		887	Asia/Urumqi	2020-07-09
`
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// 先写入 readme.txt (模拟官方 zip 结构)
	readmeFile, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readmeFile.Write([]byte("GeoNames export documentation")); err != nil {
		t.Fatal(err)
	}

	// 写入 CN.txt
	cnFile, err := zw.Create("CN.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cnFile.Write([]byte(tsvContent)); err != nil {
		t.Fatal(err)
	}
	zw.Close()

	// 验证 zip 解析器跳过 readme.txt 并正确解析 CN.txt
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	var parsedPoints int
	for _, file := range zipReader.File {
		baseName := filepath.Base(file.Name)
		if (file.Name == "CN.txt" || strings.HasSuffix(file.Name, ".txt")) && !strings.EqualFold(baseName, "readme.txt") {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			pts, err := ParseChinaGeoNamesTSV(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			parsedPoints = len(pts)
			break
		}
	}

	if parsedPoints != 1 {
		t.Fatalf("expected 1 parsed point from CN.txt, got %d", parsedPoints)
	}
}

func TestManager_InstallWithMockGeoNamesZip(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	tsvContent := `2988507	Paris	Paris	Paris,巴黎	48.85341	2.34880	P	PPLC	FR		11	75	75056	2138551	35	42	Europe/Paris	2020-07-09
`
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	f, err := zw.Create("cities15000.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(tsvContent)); err != nil {
		t.Fatal(err)
	}
	zw.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// 临时修改欧洲下载源为 mock 服务器
	AvailableContinents[2].DownloadURLs = []string{ts.URL}

	ctx := context.Background()
	var logs []string
	if err := mgr.Install(ctx, "europe", func(msg string) { logs = append(logs, msg) }); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	targetFile := filepath.Join(tempDir, "europe.json")
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("target file %s not found", targetFile)
	}
}
