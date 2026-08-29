package engine

import (
	"os"
	"path/filepath"
	"strings"
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
		"inbox_pending_report_latest.md",
		"photools_latest.log",
		"notes.txt",
		"data.json",
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

func TestDiscoverer_CompoundXMP(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"DSC_2948.nef",
		"DSC_2948.JPG",
		"DSC_2948.nef.xmp",
		"DSC_2948.JPG.xmp",
	}

	for _, f := range files {
		fullPath := filepath.Join(tempDir, f)
		_ = os.WriteFile(fullPath, []byte("data"), 0o644)
	}

	d := NewDiscoverer([]string{"nef"})
	groups, err := d.Discover(tempDir)
	if err != nil {
		t.Fatalf("Discover 失败: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("期望聚合为 1 个资产组，实际发现 %d 个: %+v", len(groups), groups)
	}

	g := groups[0]
	if g.BaseName != "DSC_2948" {
		t.Errorf("期望 BaseName 为 DSC_2948，实际为 %s", g.BaseName)
	}
	if g.RawPath == "" || g.JPGPath == "" {
		t.Errorf("期望同时识别 RAW 与 JPG: %+v", g)
	}
	if !strings.HasSuffix(g.XMPPath, ".nef.xmp") {
		t.Errorf("期望 XMPPath 指向 .nef.xmp，实际为 %s", g.XMPPath)
	}
	if len(g.CompanionPaths) != 2 {
		t.Errorf("期望 CompanionPaths 包含两个 xmp 文件，实际为 %d: %+v", len(g.CompanionPaths), g.CompanionPaths)
	}
}

func TestDiscoverer_ShieldsBackupAndSystemDirs(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"Inbox/valid.NEF",
		"Inbox/valid.JPG",
		"Inbox_bak/backup.NEF",
		"Inbox_bak/backup.JPG",
		"Inbox/sub_bak/bak.NEF",
		"Processed/2026/0825/archived.NEF",
		"GPX/track.gpx",
		"Logs/app.log",
		".hidden/hidden.NEF",
	}

	for _, f := range files {
		fullPath := filepath.Join(tempDir, f)
		_ = os.MkdirAll(filepath.Dir(fullPath), 0o755)
		_ = os.WriteFile(fullPath, []byte("test"), 0o644)
	}

	d := NewDiscoverer([]string{"nef"})
	groups, err := d.Discover(tempDir)
	if err != nil {
		t.Fatalf("Discover 返回错误: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("期望只发现 1 个资产组，实际发现 %d 个: %+v", len(groups), groups)
	}

	if groups[0].BaseName != "valid" {
		t.Errorf("期望发现 valid，实际发现: %s", groups[0].BaseName)
	}
}

func TestDiscoverer_NonExistentDir(t *testing.T) {
	d := NewDiscoverer([]string{"nef"})
	_, err := d.Discover("/non/existent/path/for/photools/test")
	if err == nil {
		t.Fatal("期望 Discover 对不存在的目录返回错误，但实际返回 nil")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("期望返回包含友好提示的错误，实际返回: %v", err)
	}
}

func TestIsAllowedGPXTrack(t *testing.T) {
	cases := []struct {
		filename string
		expected bool
	}{
		{"hiking-中国-广东-白云山-20250322.gpx", true},
		{"walking-中国-广东-海珠湖-20250501.gpx", true},
		{"hiking.gpx", true},
		{"walking.GPX", true},
		{"driving-中国-新疆-孟克特古道-20260613.gpx", false},
		{"driving-中国-新疆-伊犁到乌鲁木齐-20260613.gpx", false},
		{"hiking-中国-广东-白云山-20250322.gpx.bak", false},
		{".hiking-hidden.gpx", false},
		{"cycling-route.gpx", false},
		{"notes.txt", false},
	}

	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			got := IsAllowedGPXTrack(tc.filename)
			if got != tc.expected {
				t.Errorf("IsAllowedGPXTrack(%q) = %v, expected %v", tc.filename, got, tc.expected)
			}
		})
	}
}

func TestListGPXFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 允许的
	_ = os.WriteFile(filepath.Join(tempDir, "hiking-day1.gpx"), []byte("gpx"), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, "walking-day2.GPX"), []byte("gpx"), 0o644)
	// 排除的 (driving / bak / 其它扩展名 / 隐藏文件)
	_ = os.WriteFile(filepath.Join(tempDir, "driving-xinjiang.gpx"), []byte("gpx"), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, "hiking-day1.gpx.bak"), []byte("bak"), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, ".hiking-hidden.gpx"), []byte("gpx"), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("txt"), 0o644)

	// 子文件夹内的文件（必须不被递归扫描）
	subDir := filepath.Join(tempDir, "sub_tracks")
	_ = os.MkdirAll(subDir, 0o755)
	_ = os.WriteFile(filepath.Join(subDir, "hiking-in-sub.gpx"), []byte("gpx"), 0o644)

	gpxs, err := ListGPXFiles(tempDir)
	if err != nil {
		t.Fatalf("ListGPXFiles failed: %v", err)
	}
	if len(gpxs) != 2 {
		t.Fatalf("expected exactly 2 gpx files, got %d: %v", len(gpxs), gpxs)
	}

	expectedNames := []string{"hiking-day1.gpx", "walking-day2.GPX"}
	for i, g := range gpxs {
		if filepath.Base(g) != expectedNames[i] {
			t.Errorf("expected [%d] %s, got %s", i, expectedNames[i], filepath.Base(g))
		}
	}
}
