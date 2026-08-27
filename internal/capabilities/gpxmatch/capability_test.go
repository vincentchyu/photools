package gpxmatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
)

type mockRunner struct {
	outputs map[string][]byte
	errs    map[string]error
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmdStr := strings.Join(args, " ")
	for k, v := range m.outputs {
		if strings.Contains(cmdStr, k) {
			return v, m.errs[k]
		}
	}
	return []byte("ok"), nil
}

func TestGPXMatchingCapability_PlanPrecheck(t *testing.T) {
	runner := &mockRunner{
		outputs: map[string][]byte{
			"-json": []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00"}]`),
		},
	}
	capWithGPX := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
	})
	capWithoutGPX := NewCapability(Config{
		Runner:   runner,
		GPXFiles: nil,
	})

	// Case 1: 无主文件
	plan1 := capWithGPX.PlanPrecheck(context.Background(), &domain.AssetContext{
		Asset: domain.AssetGroup{BaseName: "DSC_001"},
	})
	if plan1.CanProcess {
		t.Errorf("expected CanProcess=false for no primary file, got true")
	}

	// Case 2: 缺少 GPX
	plan2 := capWithoutGPX.PlanPrecheck(context.Background(), &domain.AssetContext{
		Asset: domain.AssetGroup{BaseName: "DSC_001", RawPath: "DSC_001.NEF", JPGPath: "DSC_001.JPG"},
	})
	if plan2.CanProcess {
		t.Errorf("expected CanProcess=false for gpx missing, got true")
	}

	// Case 3: 正常成对就绪 (RAW + JPG)
	plan3 := capWithGPX.PlanPrecheck(context.Background(), &domain.AssetContext{
		Asset: domain.AssetGroup{BaseName: "DSC_001", RawPath: "DSC_001.NEF", JPGPath: "DSC_001.JPG"},
	})
	if !plan3.CanProcess {
		t.Errorf("expected CanProcess=true, got false: %v", plan3.Warning)
	}

	// Case 4: 单 JPG 主文件正常就绪
	plan4 := capWithGPX.PlanPrecheck(context.Background(), &domain.AssetContext{
		Asset: domain.AssetGroup{BaseName: "DSC_002", JPGPath: "DSC_002.JPG"},
	})
	if !plan4.CanProcess {
		t.Errorf("expected CanProcess=true for single JPG primary, got false: %v", plan4.Warning)
	}
}

func TestGPXMatchingCapability_ExecuteProcess_Success(t *testing.T) {
	runner := &mockRunner{
		outputs: map[string][]byte{
			"-json": []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_001",
		RawPath:  "/tmp/DSC_001.NEF",
		JPGPath:  "/tmp/DSC_001.JPG",
		XMPPath:  "/tmp/DSC_001.xmp",
	})

	var events []domain.ProgressEvent
	err := capInst.ExecuteProcess(context.Background(), actx, func(e domain.ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !actx.HasGPS {
		t.Errorf("expected actx.HasGPS=true")
	}
	if actx.Latitude == 0 || actx.Longitude == 0 {
		t.Errorf("expected coordinates parsed, got lat=%v lon=%v", actx.Latitude, actx.Longitude)
	}
	if len(actx.ModifiedFiles) != 3 {
		t.Errorf("expected 3 modified files (RAW, JPG, XMP), got %d", len(actx.ModifiedFiles))
	}
	if len(events) == 0 {
		t.Errorf("expected at least one success event")
	}
}

func TestGPXMatchingCapability_ExecuteProcess_ValidationFailure(t *testing.T) {
	runner := &mockRunner{
		outputs: map[string][]byte{
			"-json": []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
		},
		errs: map[string]error{
			"-geotag": errors.New("warning: no track points found"),
		},
	}

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_001",
		RawPath:  "/tmp/DSC_001.NEF",
		JPGPath:  "/tmp/DSC_001.JPG",
	})

	err := capInst.ExecuteProcess(context.Background(), actx, nil)
	if err == nil {
		t.Errorf("expected error for missing GPSPosition after geotag, got nil")
	}
}

func TestGPXMatchingCapability_ExecuteProcess_SingleRawOnly(t *testing.T) {
	runner := &mockRunner{
		outputs: map[string][]byte{
			"-json": []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	// 只有 RAW，没有 JPG 和 XMP
	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_8465",
		RawPath:  "/tmp/DSC_8465.NEF",
	})

	err := capInst.ExecuteProcess(context.Background(), actx, nil)
	if err != nil {
		t.Fatalf("单 RAW 资产不应报错: %v", err)
	}

	if !actx.HasGPS {
		t.Errorf("expected actx.HasGPS=true")
	}
	if len(actx.ModifiedFiles) != 1 {
		t.Errorf("expected 1 modified file (RAW only), got %d", len(actx.ModifiedFiles))
	}
}

func TestGPXMatchingCapability_ExecuteProcess_SingleJPGOnly(t *testing.T) {
	runner := &mockRunner{
		outputs: map[string][]byte{
			"-json": []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	// 只有 JPG，没有 RAW
	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "IMG_8888",
		JPGPath:  "/tmp/IMG_8888.JPG",
	})

	err := capInst.ExecuteProcess(context.Background(), actx, nil)
	if err != nil {
		t.Fatalf("单 JPG 资产不应报错: %v", err)
	}

	if !actx.HasGPS {
		t.Errorf("expected actx.HasGPS=true")
	}
	if len(actx.ModifiedFiles) != 1 || actx.ModifiedFiles[0] != "/tmp/IMG_8888.JPG" {
		t.Errorf("expected 1 modified file (JPG only), got %v", actx.ModifiedFiles)
	}
}

func TestGPXMatchingCapability_SupportedOptions_And_Configure(t *testing.T) {
	capInst := NewCapability(Config{})
	opts := capInst.SupportedOptions()
	if len(opts) == 0 {
		t.Fatalf("SupportedOptions should not be empty")
	}

	opt := opts[0]
	if opt.Key != "geosync" {
		t.Errorf("expected option key 'geosync', got %s", opt.Key)
	}

	// 测试 Configure
	_ = capInst.Configure(map[string]any{
		"geosync": "+00:00:15",
	})
	if capInst.geosync != "+00:00:15" {
		t.Errorf("expected geosync=+00:00:15, got %s", capInst.geosync)
	}
}
