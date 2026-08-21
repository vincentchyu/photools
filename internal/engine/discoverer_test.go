package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverer_DiscoverAssets(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件:
	// DSC_1001.NEF, DSC_1001.JPG, DSC_1001.xmp (完整配对)
	// DSC_1002.CR3, DSC_1002.jpg, DSC_1002.WAV (完整配对 + companion)
	// DSC_1003.ARW (单 RAW)
	// DSC_1004.JPG (单 JPG)
	// sub/DSC_2001.DNG, sub/DSC_2001.JPG (子目录)

	files := []string{
		"DSC_1001.NEF",
		"DSC_1001.JPG",
		"DSC_1001.xmp",
		"DSC_1002.CR3",
		"DSC_1002.jpg",
		"DSC_1002.WAV",
		"DSC_1003.ARW",
		"DSC_1004.JPG",
		"sub/DSC_2001.DNG",
		"sub/DSC_2001.JPG",
	}

	for _, f := range files {
		fullPath := filepath.Join(tempDir, f)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	d := NewDiscoverer([]string{"nef", "cr3", "arw", "dng"})
	groups, err := d.Discover(tempDir)
	if err != nil {
		t.Fatalf("Discover 返回错误: %v", err)
	}

	if len(groups) != 5 {
		t.Fatalf("期望发现 5 个资产组，实际发现 %d 个", len(groups))
	}

	// 验证 DSC_1001 组
	g1 := groups[0]
	if g1.BaseName != "DSC_1001" || !g1.IsPaired() || g1.XMPPath == "" {
		t.Errorf("DSC_1001 配对异常: %+v", g1)
	}

	// 验证 DSC_1002 组
	g2 := groups[1]
	if g2.BaseName != "DSC_1002" || !g2.IsPaired() || len(g2.CompanionPaths) != 1 {
		t.Errorf("DSC_1002 配对异常: %+v", g2)
	}

	// 验证 DSC_1003 组 (单 RAW)
	g3 := groups[2]
	if g3.BaseName != "DSC_1003" || !g3.HasRaw() || g3.HasJPG() {
		t.Errorf("DSC_1003 单 RAW 识别异常: %+v", g3)
	}

	// 验证 DSC_1004 组 (单 JPG)
	g4 := groups[3]
	if g4.BaseName != "DSC_1004" || g4.HasRaw() || !g4.HasJPG() {
		t.Errorf("DSC_1004 单 JPG 识别异常: %+v", g4)
	}

	// 验证子目录中的 DSC_2001
	g5 := groups[4]
	if g5.BaseName != "DSC_2001" || !g5.IsPaired() {
		t.Errorf("子目录资产组识别异常: %+v", g5)
	}
}
