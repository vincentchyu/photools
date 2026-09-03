package domain

import (
	"slices"
	"testing"
)

func TestAssetGroup(t *testing.T) {
	ag := AssetGroup{
		BaseName:       "DSC_0001",
		Dir:            "/photos",
		RawPath:        "/photos/DSC_0001.NEF",
		JPGPath:        "/photos/DSC_0001.JPG",
		XMPPath:        "/photos/DSC_0001.xmp",
		CompanionPaths: []string{"/photos/DSC_0001.xmp", "/photos/DSC_0001.WAV"},
	}

	if !ag.HasRaw() {
		t.Errorf("expected HasRaw=true")
	}
	if !ag.HasJPG() {
		t.Errorf("expected HasJPG=true")
	}
	if !ag.IsPaired() {
		t.Errorf("expected IsPaired=true")
	}

	files := ag.AllFiles()
	if len(files) != 4 {
		t.Errorf("expected 4 all files, got %d", len(files))
	}

	if ag.DisplayName() != "DSC_0001.NEF" {
		t.Errorf("expected DisplayName=DSC_0001.NEF, got %s", ag.DisplayName())
	}

	sorted := ag.SortedCompanions()
	if len(sorted) != 2 || sorted[0] != "/photos/DSC_0001.WAV" {
		t.Errorf("SortedCompanions sorted order incorrect: %v", sorted)
	}

	if ag.PrimaryPath() != "/photos/DSC_0001.NEF" {
		t.Errorf("expected PrimaryPath to be RAW, got %s", ag.PrimaryPath())
	}

	// 纯 JPG 资产
	jpgOnly := AssetGroup{
		BaseName: "DSC_0002",
		Dir:      "/photos",
		JPGPath:  "/photos/DSC_0002.JPG",
	}
	if !jpgOnly.HasPrimary() || jpgOnly.PrimaryPath() != "/photos/DSC_0002.JPG" {
		t.Errorf("expected jpgOnly PrimaryPath to be JPG, got %s", jpgOnly.PrimaryPath())
	}
}

func TestAssetContext(t *testing.T) {
	ag := AssetGroup{BaseName: "IMG_100"}
	ctx := NewAssetContext(ag)

	ctx.UpdateMetadata(
		Metadata{
			DateTimeOriginal: "2026:08:23 12:00:00",
			GPSPosition:      "39.9 N, 116.4 E",
		},
	)
	if !ctx.HasGPS {
		t.Errorf("expected HasGPS=true after UpdateMetadata with GPSPosition")
	}
	if ctx.GetMetadata().DateTimeOriginal != "2026:08:23 12:00:00" {
		t.Errorf("metadata mismatch")
	}

	ctx.SetGPS(39.9, 116.4)
	if ctx.Latitude != 39.9 || ctx.Longitude != 116.4 {
		t.Errorf("SetGPS mismatch: %v, %v", ctx.Latitude, ctx.Longitude)
	}

	loc := &LocationInfo{
		Country:  "中国",
		Province: "北京",
		City:     "北京市",
	}
	ctx.SetLocation(loc)
	if ctx.Location == nil || ctx.Location.City != "北京市" {
		t.Errorf("SetLocation mismatch")
	}

	ctx.RecordModifiedFile("/path/to/file")
	if len(ctx.ModifiedFiles) != 1 {
		t.Errorf("RecordModifiedFile count mismatch")
	}
}

func TestLocationInfo_FormatSummary(t *testing.T) {
	var nilLoc *LocationInfo
	if nilLoc.FormatSummary() != "" {
		t.Errorf("expected nil location format to be empty")
	}

	loc := &LocationInfo{
		Country:  "中国",
		Province: "新疆维吾尔自治区",
		City:     "巴音郭楞蒙古自治州",
		District: "和静县",
	}
	sum := loc.FormatSummary()
	if sum != "中国 · 新疆维吾尔自治区 · 巴音郭楞蒙古自治州 · 和静县" {
		t.Errorf("FormatSummary mismatch: %s", sum)
	}

	sameCityLoc := &LocationInfo{
		Country:  "中国",
		Province: "北京市",
		City:     "北京市",
	}
	if sameCityLoc.FormatSummary() != "中国 · 北京市" {
		t.Errorf("Same province/city deduplication failed: %s", sameCityLoc.FormatSummary())
	}
}

func TestCleanAndMergeLocationTags(t *testing.T) {
	// 场景 1: 初次写入，且包含摄影师原有自定义标签
	existing := ExistingTags{
		HierarchicalSubject: []string{"题材|人像|户外"},
		Subject:             []string{"人像", "大光圈"},
		Keywords:            []string{"人像", "大光圈"},
	}
	locBeijing := LocationInfo{
		Country:  "中国",
		Province: "北京市",
		City:     "东城区",
		District: "天安门/故宫",
	}

	res1 := CleanAndMergeLocationTags(existing, locBeijing)

	// 断言保留了自定义树，并追加了地理树
	if len(res1.HierarchicalSubject) != 2 {
		t.Fatalf(
			"HierarchicalSubject 长度期望为 2，实际为 %d: %v", len(res1.HierarchicalSubject), res1.HierarchicalSubject,
		)
	}
	if res1.HierarchicalSubject[0] != "题材|人像|户外" || res1.HierarchicalSubject[1] != "中国|北京市|东城区|天安门/故宫" {
		t.Errorf("HierarchicalSubject 内容不符合预期: %v", res1.HierarchicalSubject)
	}
	// 断言保留了自定义关键词，并合并了地理词（且去重）
	for _, expectedWord := range []string{"人像", "大光圈", "中国", "北京市", "东城区", "天安门/故宫"} {
		if !slices.Contains(res1.Subject, expectedWord) {
			t.Errorf("Subject 缺失预期词汇: %s", expectedWord)
		}
	}

	// 场景 2: 重复执行 10 次，验证绝对幂等性
	curr := existing
	for i := 0; i < 10; i++ {
		cleaned := CleanAndMergeLocationTags(curr, locBeijing)
		curr = ExistingTags{
			HierarchicalSubject: cleaned.HierarchicalSubject,
			Subject:             cleaned.Subject,
			Keywords:            cleaned.Keywords,
		}
	}
	if len(curr.HierarchicalSubject) != 2 {
		t.Errorf("幂等性测试失败：HierarchicalSubject 出现膨胀: %v", curr.HierarchicalSubject)
	}
	if len(curr.Subject) != 6 {
		t.Errorf("幂等性测试失败：Subject 出现膨胀，长度为 %d: %v", len(curr.Subject), curr.Subject)
	}

	// 场景 3: 机位变更（从北京市天安门 纠正为 山西省大同市城区），验证旧地名 100% 干净清洗剥离
	locDatong := LocationInfo{
		Country:  "中国",
		Province: "山西省",
		City:     "大同市",
		District: "城区",
	}
	resChanged := CleanAndMergeLocationTags(curr, locDatong)

	// 断言天安门树已被剔除，换成了大同树
	if slices.Contains(resChanged.HierarchicalSubject, "中国|北京市|东城区|天安门/故宫") {
		t.Errorf("机位变更后旧地理树未被清除: %v", resChanged.HierarchicalSubject)
	}
	if !slices.Contains(resChanged.HierarchicalSubject, "中国|山西省|大同市|城区") {
		t.Errorf("机位变更后缺少新地理树: %v", resChanged.HierarchicalSubject)
	}
	if !slices.Contains(resChanged.HierarchicalSubject, "题材|人像|户外") {
		t.Errorf("机位变更后摄影师自定义树被误删: %v", resChanged.HierarchicalSubject)
	}

	// 断言平铺关键词中天安门、东城区、北京市被清除，大同市、城区加入，人像保留
	for _, oldWord := range []string{"北京市", "东城区", "天安门/故宫"} {
		if slices.Contains(resChanged.Subject, oldWord) {
			t.Errorf("Subject 中旧地名残留: %s", oldWord)
		}
	}
	for _, newWord := range []string{"人像", "大光圈", "中国", "山西省", "大同市", "城区"} {
		if !slices.Contains(resChanged.Subject, newWord) {
			t.Errorf("Subject 中新词汇缺失: %s", newWord)
		}
	}
}
