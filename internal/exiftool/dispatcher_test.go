package exiftool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
)

func TestFindAssetGroupForPath(t *testing.T) {
	tmpDir := t.TempDir()

	raw := filepath.Join(tmpDir, "DSC_0001.NEF")
	jpg := filepath.Join(tmpDir, "DSC_0001.JPG")
	xmp := filepath.Join(tmpDir, "DSC_0001.NEF.xmp")

	_ = os.WriteFile(raw, []byte("raw data"), 0644)
	_ = os.WriteFile(jpg, []byte("jpg data"), 0644)
	_ = os.WriteFile(xmp, []byte("xmp data"), 0644)

	t.Run(
		"从 RAW 路径发现完整资产组", func(t *testing.T) {
			group := FindAssetGroupForPath(raw)
			if group.BaseName != "DSC_0001" {
				t.Errorf("expected BaseName DSC_0001, got %s", group.BaseName)
			}
			if group.RawPath != raw {
				t.Errorf("expected RawPath %s, got %s", raw, group.RawPath)
			}
			if group.JPGPath != jpg {
				t.Errorf("expected JPGPath %s, got %s", jpg, group.JPGPath)
			}
			if group.XMPPath != xmp {
				t.Errorf("expected XMPPath %s, got %s", xmp, group.XMPPath)
			}
		},
	)

	t.Run(
		"从 JPG 路径发现完整资产组", func(t *testing.T) {
			group := FindAssetGroupForPath(jpg)
			if group.RawPath != raw || group.JPGPath != jpg {
				t.Errorf("failed to pair from jpg: %+v", group)
			}
		},
	)

	t.Run(
		"从 XMP 路径发现完整资产组", func(t *testing.T) {
			group := FindAssetGroupForPath(xmp)
			if group.RawPath != raw || group.JPGPath != jpg {
				t.Errorf("failed to pair from xmp: %+v", group)
			}
		},
	)
}

func TestWriteGPS_Dispatcher(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("31.230000 N, 121.470000 E")}
	tmpDir := t.TempDir()

	raw := filepath.Join(tmpDir, "DSC_0002.NEF")
	jpg := filepath.Join(tmpDir, "DSC_0002.JPG")
	_ = os.WriteFile(raw, []byte("raw"), 0644)
	_ = os.WriteFile(jpg, []byte("jpg"), 0644)

	asset := domain.AssetGroup{
		BaseName: "DSC_0002",
		Dir:      tmpDir,
		RawPath:  raw,
		JPGPath:  jpg,
	}

	payload := GPSWritePayload{
		Latitude:  31.23,
		Longitude: 121.47,
		Altitude:  10.0,
		Provenance: domain.GPSProvenance{
			Source:      "manual_copied",
			MatchMethod: "clipboard_paste",
		},
	}

	t.Run(
		"PolicySmart 智能模式写入 RAW+JPG+XMP", func(t *testing.T) {
			mod, err := WriteGPS(runner, asset, payload, domain.PolicySmart)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// 应该修改 RAW, JPG 以及生成/修改 XMP 侧车
			if len(mod) != 3 {
				t.Fatalf("expected 3 modified files in smart mode, got %d: %v", len(mod), mod)
			}
		},
	)

	t.Run(
		"PolicySidecarOnly 纯侧车模式仅写入 XMP", func(t *testing.T) {
			mod, err := WriteGPS(runner, asset, payload, domain.PolicySidecarOnly)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mod) != 1 || filepath.Ext(mod[0]) != ".xmp" {
				t.Fatalf("expected 1 xmp modified file, got %v", mod)
			}
		},
	)

	t.Run(
		"PolicyEmbedOnly 纯内嵌模式仅写 RAW+JPG", func(t *testing.T) {
			mod, err := WriteGPS(runner, asset, payload, domain.PolicyEmbedOnly)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mod) != 2 {
				t.Fatalf("expected 2 modified files (RAW+JPG), got %d: %v", len(mod), mod)
			}
		},
	)

	t.Run(
		"二次校验失败时阻断报错", func(t *testing.T) {
			emptyRunner := &mockCommandRunner{output: []byte("")}
			_, err := WriteGPS(emptyRunner, asset, payload, domain.PolicySmart)
			if err == nil {
				t.Fatal("expected error on secondary verification failure, got nil")
			}
		},
	)
}

func TestWriteLocation_Dispatcher(t *testing.T) {
	runner := &mockCommandRunner{output: []byte("Shanghai, China")}
	tmpDir := t.TempDir()

	raw := filepath.Join(tmpDir, "DSC_0003.NEF")
	jpg := filepath.Join(tmpDir, "DSC_0003.JPG")
	_ = os.WriteFile(raw, []byte("raw"), 0644)
	_ = os.WriteFile(jpg, []byte("jpg"), 0644)

	asset := domain.AssetGroup{
		BaseName: "DSC_0003",
		Dir:      tmpDir,
		RawPath:  raw,
		JPGPath:  jpg,
	}

	loc := domain.LocationInfo{
		Country:  "中国",
		Province: "上海市",
		City:     "上海市",
		District: "黄浦区",
	}

	t.Run(
		"PolicySmart 下 RAW 保持只读仅写 XMP，JPG 内嵌写入", func(t *testing.T) {
			mod, err := WriteLocation(runner, asset, loc, domain.PolicySmart)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mod) != 2 {
				t.Fatalf("expected 2 files (XMP + JPG), got %d: %v", len(mod), mod)
			}
			for _, f := range mod {
				if f == raw {
					t.Fatalf("PolicySmart 严禁为了地名修改 RAW 原图: %s", f)
				}
			}
		},
	)

	t.Run(
		"PolicySidecarOnly 下仅写 XMP", func(t *testing.T) {
			mod, err := WriteLocation(runner, asset, loc, domain.PolicySidecarOnly)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, f := range mod {
				if filepath.Ext(f) != ".xmp" {
					t.Fatalf("PolicySidecarOnly 下只能修改 XMP，但修改了: %s", f)
				}
			}
		},
	)
}
