package domain

import (
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

	ctx.UpdateMetadata(Metadata{
		DateTimeOriginal: "2026:08:23 12:00:00",
		GPSPosition:      "39.9 N, 116.4 E",
	})
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
