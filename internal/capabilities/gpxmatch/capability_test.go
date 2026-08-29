package gpxmatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
)

type mockRunner struct {
	// outputs: 关键字 -> 每次调用依次返回的输出列表（按调用次序轮转）
	outputs map[string][][]byte
	errs    map[string]error
	calls   map[string]int
}

func newMockRunner(outputs map[string][][]byte, errs map[string]error) *mockRunner {
	return &mockRunner{
		outputs: outputs,
		errs:    errs,
		calls:   make(map[string]int),
	}
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmdStr := strings.Join(args, " ")
	for k, respList := range m.outputs {
		if strings.Contains(cmdStr, k) {
			idx := m.calls[k]
			m.calls[k]++
			if idx < len(respList) {
				return respList[idx], m.errs[k]
			}
			// 超出列表范围时返回最后一个
			return respList[len(respList)-1], m.errs[k]
		}
	}
	return []byte("ok"), nil
}

func TestGPXMatchingCapability_PlanPrecheck(t *testing.T) {
	runner := newMockRunner(map[string][][]byte{
		"-json": {[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00"}]`)},
	}, nil)
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

// TestGPXMatchingCapability_ExecuteProcess_Success 模拟照片原本无 GPS，通过 GPX 轨迹新写入 GPS 坐标
func TestGPXMatchingCapability_ExecuteProcess_Success(t *testing.T) {
	// 第1次调用 -json: 初始读取，无 GPS
	// 第2次调用 -json: geotag 后二次校验，已有新 GPS
	runner := newMockRunner(map[string][][]byte{
		"-json": {
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}, nil)

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
	actx.SidecarPolicy = domain.PolicyEmbedAndSidecar

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

// TestGPXMatchingCapability_PolicySmart 模拟照片原本无 GPS，通过 GPX 写入后 PolicySmart 模式处理
func TestGPXMatchingCapability_PolicySmart(t *testing.T) {
	// 第1次 -json: 初始读取无 GPS；第2次 -json: geotag 后有 GPS
	runner := newMockRunner(map[string][][]byte{
		"-json": {
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"39.9042 N, 116.3917 E"}]`),
		},
	}, nil)

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_001",
		RawPath:  "/tmp/DSC_001.NEF",
		JPGPath:  "/tmp/DSC_001.JPG",
	})
	actx.SidecarPolicy = domain.PolicySmart

	err := capInst.ExecuteProcess(context.Background(), actx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !actx.HasGPS {
		t.Errorf("expected actx.HasGPS=true")
	}
	// PolicySmart (智能分层模式): RAW 写入 EXIF + RAW.XMP 侧车注入指纹 + JPG 写入 EXIF = 3 个修改记录
	if len(actx.ModifiedFiles) != 3 {
		t.Errorf("expected 3 modified files in PolicySmart (RAW + RAW.XMP + JPG), got %d: %v", len(actx.ModifiedFiles), actx.ModifiedFiles)
	}
	if actx.Provenance == nil || actx.Provenance.Source != "gpx" {
		t.Errorf("expected Provenance.Source='gpx', got %+v", actx.Provenance)
	}
}

func TestGPXMatchingCapability_SidecarOnly(t *testing.T) {
	// 第1次 -json: 初始读取无 GPS；第2次 -json: geotag 后有 GPS（读 XMP 侧车）
	runner := newMockRunner(map[string][][]byte{
		"-json": {
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"39.9042 N, 116.3917 E"}]`),
		},
	}, nil)

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_001",
		RawPath:  "/tmp/DSC_001.NEF",
		JPGPath:  "/tmp/DSC_001.JPG",
	})
	actx.SidecarOnly = true

	err := capInst.ExecuteProcess(context.Background(), actx, nil)
	if err != nil {
		t.Fatalf("unexpected error in SidecarOnly: %v", err)
	}

	if !actx.HasGPS {
		t.Errorf("expected actx.HasGPS=true")
	}
	// 在 SidecarOnly 模式下，RAW 与 JPG 原图不应被列入修改列表，而是修改对应的 XMP 侧车文件
	for _, mod := range actx.ModifiedFiles {
		if !strings.HasSuffix(mod, ".xmp") {
			t.Errorf("SidecarOnly 模式下不应修改非 XMP 文件: %s", mod)
		}
	}
}

func TestGPXMatchingCapability_ExecuteProcess_ValidationFailure(t *testing.T) {
	runner := newMockRunner(map[string][][]byte{
		"-json": {[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`)},
	}, map[string]error{
		"-geotag": errors.New("warning: no track points found"),
	})

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

// TestGPXMatchingCapability_ExecuteProcess_SingleRawOnly 模拟单 RAW 照片无 GPS，通过 GPX 写入
func TestGPXMatchingCapability_ExecuteProcess_SingleRawOnly(t *testing.T) {
	// 第1次 -json: 无 GPS；第2次 -json: geotag 后有 GPS
	runner := newMockRunner(map[string][][]byte{
		"-json": {
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}, nil)

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	// 只有 RAW，没有 JPG 和 XMP (默认 PolicySmart: RAW EXIF + 生成 RAW.xmp 侧车并记录指纹)
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
	// Smart 模式下单 RAW 资产: 修改 RAW 本身 + 生成/更新配套 RAW.xmp 侧车注入指纹 = 2 个文件
	if len(actx.ModifiedFiles) != 2 {
		t.Errorf("expected 2 modified files (RAW + XMP sidecar), got %d: %v", len(actx.ModifiedFiles), actx.ModifiedFiles)
	}
	if actx.Provenance == nil || actx.Provenance.Source != "gpx" {
		t.Errorf("expected Provenance.Source='gpx', got %+v", actx.Provenance)
	}
}

// TestGPXMatchingCapability_ExecuteProcess_SingleJPGOnly 模拟单 JPG 照片无 GPS，通过 GPX 写入
func TestGPXMatchingCapability_ExecuteProcess_SingleJPGOnly(t *testing.T) {
	// 第1次 -json: 无 GPS；第2次 -json: geotag 后有 GPS
	runner := newMockRunner(map[string][][]byte{
		"-json": {
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":""}]`),
			[]byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`),
		},
	}, nil)

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

// TestGPXMatchingCapability_NoDriftCorrection 模拟照片本身有 GPS 且与 GPX 轨迹完全一致，不应修改任何文件
func TestGPXMatchingCapability_NoDriftCorrection(t *testing.T) {
	// 初次读和二次读均返回相同坐标，模拟 ExifTool geotag 未改变坐标
	sameGPS := `[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`
	runner := newMockRunner(map[string][][]byte{
		"-json": {[]byte(sameGPS), []byte(sameGPS)},
	}, nil)

	capInst := NewCapability(Config{
		Runner:   runner,
		GPXFiles: []string{"/path/to/track.gpx"},
		Geosync:  "0",
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_999",
		RawPath:  "/tmp/DSC_999.NEF",
	})

	var events []domain.ProgressEvent
	err := capInst.ExecuteProcess(context.Background(), actx, func(e domain.ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	// 坐标未变化，不应修改任何文件
	if len(actx.ModifiedFiles) != 0 {
		t.Errorf("坐标未变化时不应修改任何文件，got %v", actx.ModifiedFiles)
	}
	// 应有一条 Info 级别的"无需校准"事件
	hasSkipEvent := false
	for _, e := range events {
		if e.Level == domain.LevelInfo && strings.Contains(e.Message, "无需校准") {
			hasSkipEvent = true
		}
	}
	if !hasSkipEvent {
		t.Errorf("坐标未变化时应输出[无需校准]事件，got events: %v", events)
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
