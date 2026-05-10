package organizer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverMediaGroups(t *testing.T) {
	root := t.TempDir()

	// 模拟文件系统中的文件，包含大写和小写后缀，以及多个文件组
	files := []string{
		// 第一组：标准 RAW + 配套文件
		"DSC_0989.NEF",
		"DSC_0989.acr",
		"DSC_0989.XMP",
		"DSC_0989.wav",

		// 第二组：只有 JPG 和 配套文件
		"DSC_1000.JPG",
		"DSC_1000.xmp",

		// 第三组：只有 RAW
		"DSC_1001.cr3",

		// 第四组：独立存在的配套文件（没有主图也会被成组）
		"DSC_2000.acr",
	}

	for _, f := range files {
		os.WriteFile(filepath.Join(root, f), []byte("test"), 0644)
	}

	groups, err := DiscoverMediaGroups(root, []string{"nef", "cr3"})
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}

	// 因为 DiscoverMediaGroups 会进行排序，所以顺序应该是: DSC_0989, DSC_1000, DSC_1001, DSC_2000

	// 第一组检查
	g1 := groups[0]
	if g1.BaseName != "DSC_0989" {
		t.Errorf("g1 BaseName: expected DSC_0989, got %s", g1.BaseName)
	}
	if filepath.Base(g1.RawPath) != "DSC_0989.NEF" {
		t.Errorf("g1 RawPath: expected DSC_0989.NEF, got %s", filepath.Base(g1.RawPath))
	}
	if len(g1.CompanionPaths) != 3 {
		t.Errorf("g1 CompanionPaths: expected 3, got %d", len(g1.CompanionPaths))
	}

	// 第二组检查
	g2 := groups[1]
	if g2.BaseName != "DSC_1000" {
		t.Errorf("g2 BaseName: expected DSC_1000, got %s", g2.BaseName)
	}
	if filepath.Base(g2.JPGPath) != "DSC_1000.JPG" {
		t.Errorf("g2 JPGPath: expected DSC_1000.JPG, got %s", filepath.Base(g2.JPGPath))
	}
	if len(g2.CompanionPaths) != 1 || filepath.Base(g2.XMPPath) != "DSC_1000.xmp" {
		t.Errorf("g2 XMPPath/Companion error")
	}

	// 第三组检查
	g3 := groups[2]
	if g3.BaseName != "DSC_1001" {
		t.Errorf("g3 BaseName: expected DSC_1001, got %s", g3.BaseName)
	}
	if filepath.Base(g3.RawPath) != "DSC_1001.cr3" {
		t.Errorf("g3 RawPath: expected DSC_1001.cr3, got %s", filepath.Base(g3.RawPath))
	}

	// 第四组检查
	g4 := groups[3]
	if g4.BaseName != "DSC_2000" {
		t.Errorf("g4 BaseName: expected DSC_2000, got %s", g4.BaseName)
	}
	if len(g4.CompanionPaths) != 1 || filepath.Base(g4.CompanionPaths[0]) != "DSC_2000.acr" {
		t.Errorf("g4 CompanionPaths error")
	}
}

func TestAllFiles(t *testing.T) {
	asset := PhotoAsset{
		BaseName:       "DSC_0001",
		RawPath:        "/tmp/DSC_0001.NEF",
		JPGPath:        "/tmp/DSC_0001.JPG",
		XMPPath:        "/tmp/DSC_0001.xmp",
		CompanionPaths: []string{"/tmp/DSC_0001.acr", "/tmp/DSC_0001.xmp"},
	}

	expected := []string{
		"/tmp/DSC_0001.NEF",
		"/tmp/DSC_0001.JPG",
		"/tmp/DSC_0001.acr",
		"/tmp/DSC_0001.xmp",
	}

	got := asset.AllFiles()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("AllFiles() returned %v, want %v", got, expected)
	}
}
