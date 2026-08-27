package geodata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedAdminZHLoaded(t *testing.T) {
	EnsureMappingsLoaded()

	admin1 := GetAllAdmin1ZH()
	if len(admin1) < 3800 {
		t.Fatalf("expected at least 3800 admin1 entries, got %d", len(admin1))
	}

	admin2 := GetAllAdmin2ZH()
	if len(admin2) < 47000 {
		t.Fatalf("expected at least 47000 admin2 entries, got %d", len(admin2))
	}

	// 检查所有条目均包含 code, name_ascii, name_zh
	for k, meta := range admin1 {
		if meta.Code == "" || meta.NameASCII == "" || meta.NameZH == "" {
			t.Errorf("admin1 entry %q has empty required field: %+v", k, meta)
			break
		}
	}

	for k, meta := range admin2 {
		if meta.Code == "" || meta.NameASCII == "" || meta.NameZH == "" {
			t.Errorf("admin2 entry %q has empty required field: %+v", k, meta)
			break
		}
	}
}

func TestChinaAdminConsistency(t *testing.T) {
	EnsureMappingsLoaded()

	// 1. 验证 Admin1 与 ChinaAdmin1Map 一致性
	a1Tests := []struct {
		code     string
		wantName string
	}{
		{"CN.13", "新疆维吾尔自治区"},
		{"CN.22", "北京市"},
		{"CN.01", "安徽省"},
		{"CN.02", "浙江省"},
		{"CN.30", "广东省"},
		{"CN.32", "四川省"},
		{"CN.33", "重庆市"},
		{"CN.14", "西藏自治区"},
	}

	for _, tt := range a1Tests {
		meta, ok := GetAdmin1ZH(tt.code)
		if !ok {
			t.Errorf("GetAdmin1ZH(%q) not found", tt.code)
			continue
		}
		if meta.NameZH != tt.wantName {
			t.Errorf("GetAdmin1ZH(%q) = %q, want %q", tt.code, meta.NameZH, tt.wantName)
		}
		// 校验短代码查找
		subCode := tt.code[3:]
		if got := GetAdmin1NameZH(subCode); got != tt.wantName {
			t.Errorf("GetAdmin1NameZH(%q) = %q, want %q", subCode, got, tt.wantName)
		}
	}

	// 2. 验证 Admin2 与 ChinaAdmin2Map 一致性
	a2Tests := []struct {
		code     string
		wantName string
	}{
		{"CN.13.6531", "喀什地区"},
		{"CN.06.6327", "玉树藏族自治州"},
		{"CN.02.3303", "温州市"},
		{"CN.30.4403", "深圳市"},
		{"CN.32.5103", "自贡市"},
	}

	for _, tt := range a2Tests {
		meta, ok := GetAdmin2ZH(tt.code)
		if !ok {
			t.Errorf("GetAdmin2ZH(%q) not found", tt.code)
			continue
		}
		if meta.NameZH != tt.wantName {
			t.Errorf("GetAdmin2ZH(%q) = %q, want %q", tt.code, meta.NameZH, tt.wantName)
		}
	}

	// 校验短代码查找
	if got := GetAdmin2NameZH("6531"); got != "喀什地区" {
		t.Errorf("GetAdmin2NameZH(\"6531\") = %q, want \"喀什地区\"", got)
	}
	if got := GetAdmin2NameZH("6327"); got != "玉树藏族自治州" {
		t.Errorf("GetAdmin2NameZH(\"6327\") = %q, want \"玉树藏族自治州\"", got)
	}
	if got := GetAdmin2NameZH("4403"); got != "深圳市" {
		t.Errorf("GetAdmin2NameZH(\"4403\") = %q, want \"深圳市\"", got)
	}
}

func TestGlobalAdminTranslations(t *testing.T) {
	EnsureMappingsLoaded()

	cases1 := []struct {
		code     string
		wantZH   string
		wantOrig string
	}{
		{"US.CA", "加利福尼亚州", "California"},
		{"US.NY", "纽约州", "New York"},
		{"JP.40", "东京都", "Tokyo"},
		{"JP.22", "京都府", "Kyoto"},
		{"FR.11", "法兰西岛大区", "Île-de-France"},
		{"DE.02", "巴伐利亚州", "Bavaria"},
		{"GB.ENG", "英格兰", "England"},
		{"IT.09", "伦巴第大区", "Lombardy"},
		{"BR.27", "圣保罗州", "São Paulo"},
	}

	for _, tc := range cases1 {
		meta, ok := GetAdmin1ZH(tc.code)
		if !ok {
			t.Errorf("GetAdmin1ZH(%q) not found", tc.code)
			continue
		}
		if meta.NameZH != tc.wantZH {
			t.Errorf("GetAdmin1ZH(%q) NameZH = %q, want %q", tc.code, meta.NameZH, tc.wantZH)
		}
		if meta.Name != tc.wantOrig {
			t.Errorf("GetAdmin1ZH(%q) Name = %q, want %q", tc.code, meta.Name, tc.wantOrig)
		}
		if got := GetAdmin1NameZH(tc.code); got != tc.wantZH {
			t.Errorf("GetAdmin1NameZH(%q) = %q, want %q", tc.code, got, tc.wantZH)
		}
	}

	cases2 := []struct {
		code   string
		wantZH string
	}{
		{"US.CA.075", "旧金山市县"},
		{"US.CA.037", "洛杉矶县"},
		{"US.NY.061", "纽约县"},
		{"AE.01.101", "阿布扎比市"},
	}

	for _, tc := range cases2 {
		meta, ok := GetAdmin2ZH(tc.code)
		if !ok {
			t.Errorf("GetAdmin2ZH(%q) not found", tc.code)
			continue
		}
		if meta.NameZH != tc.wantZH {
			t.Errorf("GetAdmin2ZH(%q) NameZH = %q, want %q", tc.code, meta.NameZH, tc.wantZH)
		}
		if got := GetAdmin2NameZH(tc.code); got != tc.wantZH {
			t.Errorf("GetAdmin2NameZH(%q) = %q, want %q", tc.code, got, tc.wantZH)
		}
	}

	// 边界测试：不存在的代码回退
	if got := GetAdmin1NameZH("NON.EXISTENT"); got != "NON.EXISTENT" {
		t.Errorf("expected fallback to code, got %q", got)
	}
	if got := GetAdmin2NameZH("NON.EXISTENT.CODE"); got != "NON.EXISTENT.CODE" {
		t.Errorf("expected fallback to code, got %q", got)
	}
}

func TestLoadMappingFilesFromDir_WithZHLookup(t *testing.T) {
	tempDir := t.TempDir()

	// 1. admin2 文本行包含已知字典的 US.CA.037
	admin2Content := "US.CA.037\tLos Angeles County\tLos Angeles County\t5368381\n" +
		"CN.13.9876\tUnknown County\tUnknown County\t987654\n"
	if err := os.WriteFile(filepath.Join(tempDir, "admin2Codes.txt"), []byte(admin2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. admin1 文本行包含已知字典的 US.CA
	admin1Content := "US.CA\tCalifornia\tCalifornia\t5332921\n"
	if err := os.WriteFile(filepath.Join(tempDir, "admin1CodesASCII.txt"), []byte(admin1Content), 0644); err != nil {
		t.Fatal(err)
	}

	LoadMappingFilesFromDir(tempDir)

	// 校验从字典中查找到中文
	if meta, ok := Admin2ZHMap["US.CA.037"]; !ok || meta.NameZH != "洛杉矶县" {
		t.Errorf("expected Admin2ZHMap to have '洛杉矶县', got %+v", meta)
	}
	if meta, ok := Admin1ZHMap["US.CA"]; !ok || meta.NameZH != "加利福尼亚州" {
		t.Errorf("expected Admin1ZHMap to have '加利福尼亚州', got %+v", meta)
	}
}

func TestFallbackWithoutChineseDictionaries(t *testing.T) {
	// 验证在没有该国中文时，GetCountryMeta 优雅降级返回原大写代码
	meta := GetCountryMeta("ZZ")
	if meta.NameZH != "ZZ" || meta.Code != "ZZ" {
		t.Errorf("expected GetCountryMeta fallback to 'ZZ', got %+v", meta)
	}

	// 验证未收录的省份代码回退到源代码或英文
	prov := GetChinaProvinceName("8888", 0, 0)
	if prov != "8888" {
		t.Errorf("expected GetChinaProvinceName fallback to '8888', got %q", prov)
	}
}
