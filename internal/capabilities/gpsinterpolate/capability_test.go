package gpsinterpolate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vincentchyu/photools/internal/domain"
)

type mockRunner struct {
	recordedArgs [][]string
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	m.recordedArgs = append(m.recordedArgs, args)
	for _, a := range args {
		if a == "-ver" {
			return []byte("13.55"), nil
		}
	}
	return []byte("1 image files updated"), nil
}

func TestGPSInterpolate_BidirectionalInterpolation(t *testing.T) {
	tempDir := t.TempDir()
	rawPath1 := filepath.Join(tempDir, "DSC_0001.NEF")
	rawPath2 := filepath.Join(tempDir, "DSC_0002.NEF")
	rawPath3 := filepath.Join(tempDir, "DSC_0003.NEF")

	_ = os.WriteFile(rawPath1, []byte("raw1"), 0o644)
	_ = os.WriteFile(rawPath2, []byte("raw2"), 0o644)
	_ = os.WriteFile(rawPath3, []byte("raw3"), 0o644)

	// 照片 1: 10:00:00, (40.0, 100.0)
	actx1 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0001", RawPath: rawPath1})
	actx1.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "40.0 N, 100.0 E"})
	actx1.SetGPS(40.0, 100.0)

	// 照片 2 (目标待推断): 10:06:00, 无 GPS
	actx2 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0002", RawPath: rawPath2})
	actx2.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:06:00"})

	// 照片 3: 10:10:00, (40.1, 100.2)
	actx3 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0003", RawPath: rawPath3})
	actx3.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:10:00", GPSPosition: "40.1 N, 100.2 E"})
	actx3.SetGPS(40.1, 100.2)

	batch := []*domain.AssetContext{actx1, actx2, actx3}
	actx2.Batch = batch

	runner := &mockRunner{}
	capInst := NewCapability(Config{
		Runner:     runner,
		MaxTimeGap: 15 * time.Minute,
		Priority:   15,
	})

	// 1. 测试 Init
	err := capInst.Init(context.Background(), nil)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// 2. 测试 PlanPrecheck
	plan := capInst.PlanPrecheck(context.Background(), actx2)
	if !plan.CanProcess {
		t.Fatalf("expected CanProcess=true, got false")
	}

	// 3. 测试 ExecuteProcess
	var events []domain.ProgressEvent
	err = capInst.ExecuteProcess(context.Background(), actx2, func(e domain.ProgressEvent) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ExecuteProcess failed: %v", err)
	}

	// 验证插值计算结果: 10:06 在 10:00 和 10:10 之间，权重比为 6/10 = 0.6
	// Lat = 40.0 + 0.6 * (40.1 - 40.0) = 40.06
	// Lon = 100.0 + 0.6 * (100.2 - 100.0) = 100.12
	expectedLat := 40.06
	expectedLon := 100.12

	if !actx2.HasGPS {
		t.Errorf("actx2.HasGPS should be true")
	}
	if actx2.Latitude < expectedLat-0.0001 || actx2.Latitude > expectedLat+0.0001 {
		t.Errorf("expected Latitude=%.4f, got %.4f", expectedLat, actx2.Latitude)
	}
	if actx2.Longitude < expectedLon-0.0001 || actx2.Longitude > expectedLon+0.0001 {
		t.Errorf("expected Longitude=%.4f, got %.4f", expectedLon, actx2.Longitude)
	}
	if actx2.Provenance == nil || actx2.Provenance.Source != "interpolated" {
		t.Errorf("expected Provenance.Source='interpolated', got %+v", actx2.Provenance)
	}
}

func TestGPSInterpolate_SingleNeighborInheritance(t *testing.T) {
	tempDir := t.TempDir()
	rawPath1 := filepath.Join(tempDir, "DSC_0001.NEF")
	rawPath2 := filepath.Join(tempDir, "DSC_0002.NEF")

	_ = os.WriteFile(rawPath1, []byte("raw1"), 0o644)
	_ = os.WriteFile(rawPath2, []byte("raw2"), 0o644)

	// 照片 1: 10:00:00, (43.0, 82.0)
	actx1 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0001", RawPath: rawPath1})
	actx1.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "43.0 N, 82.0 E"})
	actx1.SetGPS(43.0, 82.0)

	// 照片 2 (目标): 10:05:00, 无 GPS
	actx2 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0002", RawPath: rawPath2})
	actx2.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:05:00"})

	actx2.Batch = []*domain.AssetContext{actx1, actx2}

	runner := &mockRunner{}
	capInst := NewCapability(Config{
		Runner:     runner,
		MaxTimeGap: 10 * time.Minute,
	})

	err := capInst.ExecuteProcess(context.Background(), actx2, nil)
	if err != nil {
		t.Fatalf("ExecuteProcess failed: %v", err)
	}

	if actx2.Latitude != 43.0 || actx2.Longitude != 82.0 {
		t.Errorf("expected inherited coords (43.0, 82.0), got (%.4f, %.4f)", actx2.Latitude, actx2.Longitude)
	}
}

func TestGPSInterpolate_SidecarOnly(t *testing.T) {
	tempDir := t.TempDir()
	rawPath1 := filepath.Join(tempDir, "DSC_0001.NEF")
	rawPath2 := filepath.Join(tempDir, "DSC_0002.NEF")

	_ = os.WriteFile(rawPath1, []byte("raw1"), 0o644)
	_ = os.WriteFile(rawPath2, []byte("raw2"), 0o644)

	actx1 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0001", RawPath: rawPath1})
	actx1.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "43.0 N, 82.0 E"})
	actx1.SetGPS(43.0, 82.0)

	actx2 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0002", RawPath: rawPath2})
	actx2.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:05:00"})
	actx2.SidecarOnly = true

	actx2.Batch = []*domain.AssetContext{actx1, actx2}

	runner := &mockRunner{}
	capInst := NewCapability(Config{
		Runner:     runner,
		MaxTimeGap: 10 * time.Minute,
	})

	err := capInst.ExecuteProcess(context.Background(), actx2, nil)
	if err != nil {
		t.Fatalf("ExecuteProcess in SidecarOnly failed: %v", err)
	}

	for _, mod := range actx2.ModifiedFiles {
		if !strings.HasSuffix(mod, ".xmp") {
			t.Errorf("SidecarOnly 模式下不应修改非 XMP 文件: %s", mod)
		}
	}
}

func TestGPSInterpolate_ExceedWindow_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	rawPath1 := filepath.Join(tempDir, "DSC_0001.NEF")
	rawPath2 := filepath.Join(tempDir, "DSC_0002.NEF")

	_ = os.WriteFile(rawPath1, []byte("raw1"), 0o644)
	_ = os.WriteFile(rawPath2, []byte("raw2"), 0o644)

	// 照片 1: 10:00:00
	actx1 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0001", RawPath: rawPath1})
	actx1.SetGPS(43.0, 82.0)
	actx1.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "43.0 N, 82.0 E"})

	// 照片 2: 11:30:00 (相隔 90 分钟，超出 15 分钟窗口)
	actx2 := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_0002", RawPath: rawPath2})
	actx2.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 11:30:00"})

	actx2.Batch = []*domain.AssetContext{actx1, actx2}

	runner := &mockRunner{}
	capInst := NewCapability(Config{
		Runner:     runner,
		MaxTimeGap: 15 * time.Minute,
	})

	err := capInst.ExecuteProcess(context.Background(), actx2, nil)
	if err == nil {
		t.Errorf("expected error due to window exceeded, got nil")
	}
}

func TestGPSInterpolate_SingleJPG_Interpolation(t *testing.T) {
	tempDir := t.TempDir()
	jpgPath1 := filepath.Join(tempDir, "IMG_0001.JPG")
	jpgPath2 := filepath.Join(tempDir, "IMG_0002.JPG")

	_ = os.WriteFile(jpgPath1, []byte("jpg1"), 0o644)
	_ = os.WriteFile(jpgPath2, []byte("jpg2"), 0o644)

	// 照片 1 (JPG): 10:00:00, (39.9, 116.3)
	actx1 := domain.NewAssetContext(domain.AssetGroup{BaseName: "IMG_0001", JPGPath: jpgPath1})
	actx1.SetGPS(39.9, 116.3)
	actx1.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "39.9 N, 116.3 E"})

	// 照片 2 (单 JPG，待插值): 10:05:00
	actx2 := domain.NewAssetContext(domain.AssetGroup{BaseName: "IMG_0002", JPGPath: jpgPath2})
	actx2.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2026:08:24 10:05:00"})

	actx2.Batch = []*domain.AssetContext{actx1, actx2}

	runner := &mockRunner{}
	capInst := NewCapability(Config{
		Runner:     runner,
		MaxTimeGap: 15 * time.Minute,
	})

	// 预检
	plan := capInst.PlanPrecheck(context.Background(), actx2)
	if !plan.CanProcess {
		t.Fatalf("单 JPG 资产在 PlanPrecheck 中应可处理，实际: %v", plan.ActionDesc)
	}

	// 执行
	err := capInst.ExecuteProcess(context.Background(), actx2, nil)
	if err != nil {
		t.Fatalf("单 JPG 资产插值执行失败: %v", err)
	}

	if !actx2.HasGPS || actx2.Latitude != 39.9 || actx2.Longitude != 116.3 {
		t.Errorf("单 JPG 资产未正确继承 GPS 坐标: lat=%v lon=%v", actx2.Latitude, actx2.Longitude)
	}
	if len(actx2.ModifiedFiles) != 1 || actx2.ModifiedFiles[0] != jpgPath2 {
		t.Errorf("expected 1 modified file (JPG only), got %v", actx2.ModifiedFiles)
	}
}

func TestGPSInterpolate_SupportedOptions_And_Configure(t *testing.T) {
	capInst := NewCapability(Config{})
	opts := capInst.SupportedOptions()
	if len(opts) == 0 {
		t.Fatalf("SupportedOptions should not be empty")
	}

	opt := opts[0]
	if opt.Key != "window" {
		t.Errorf("expected option key 'window', got %s", opt.Key)
	}
	if len(opt.Choices) == 0 {
		t.Errorf("expected choices for window, got empty")
	}

	// 测试 Configure
	_ = capInst.Configure(map[string]any{
		"window": "1h",
	})
	if capInst.maxTimeGap != time.Hour {
		t.Errorf("expected maxTimeGap=1h, got %v", capInst.maxTimeGap)
	}

	_ = capInst.Configure(map[string]any{
		"window": 30 * time.Minute,
	})
	if capInst.maxTimeGap != 30*time.Minute {
		t.Errorf("expected maxTimeGap=30m, got %v", capInst.maxTimeGap)
	}
}

func TestGPSInterpolate_LargeBatchDateIndexedBinarySearch(t *testing.T) {
	tempDir := t.TempDir()
	var batch []*domain.AssetContext

	// 构造 2025-10-05 和 2025-10-06 两天共 100 张照片
	// 其中 2025-10-05 只有 10:00 和 11:00 有 GPS，中间的照片待插值
	for i := 0; i < 50; i++ {
		p := filepath.Join(tempDir, fmt.Sprintf("DSC_1005_%02d.NEF", i))
		_ = os.WriteFile(p, []byte("raw"), 0o644)
		actx := domain.NewAssetContext(domain.AssetGroup{BaseName: fmt.Sprintf("DSC_1005_%02d", i), RawPath: p})
		tStr := fmt.Sprintf("2025:10:05 10:%02d:00", i)
		if i == 0 {
			actx.SetGPS(31.23, 121.47)
			actx.UpdateMetadata(domain.Metadata{DateTimeOriginal: tStr, GPSPosition: "31.23 N, 121.47 E"})
		} else if i == 49 {
			actx.SetGPS(31.25, 121.49)
			actx.UpdateMetadata(domain.Metadata{DateTimeOriginal: tStr, GPSPosition: "31.25 N, 121.49 E"})
		} else {
			actx.UpdateMetadata(domain.Metadata{DateTimeOriginal: tStr})
		}
		batch = append(batch, actx)
	}

	for _, actx := range batch {
		actx.Batch = batch
	}

	capInst := NewCapability(Config{
		Runner:     &mockRunner{},
		MaxTimeGap: 1 * time.Hour,
	})

	// 测试中间第 25 张照片 (10:25:00) 的插值
	targetActx := batch[25]
	err := capInst.ExecuteProcess(context.Background(), targetActx, nil)
	if err != nil {
		t.Fatalf("二分查找插值执行失败: %v", err)
	}

	if !targetActx.HasGPS || targetActx.Latitude == 0 {
		t.Errorf("expected targetActx to have interpolated GPS coords")
	}
}

func TestGPSInterpolate_AllowNoGPS_GracefulSkip(t *testing.T) {
	tempDir := t.TempDir()
	rawPath := filepath.Join(tempDir, "DSC_9999.NEF")
	_ = os.WriteFile(rawPath, []byte("raw"), 0o644)

	actx := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_9999", RawPath: rawPath})
	actx.UpdateMetadata(domain.Metadata{DateTimeOriginal: "2025:12:24 12:00:00"})
	actx.Batch = []*domain.AssetContext{actx} // 孤立点，无任何前后锚点

	capInst := NewCapability(Config{
		Runner:     &mockRunner{},
		MaxTimeGap: 15 * time.Minute,
		AllowNoGPS: true, // 开启容错降级
	})

	var events []domain.ProgressEvent
	err := capInst.ExecuteProcess(context.Background(), actx, func(e domain.ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("开启 AllowNoGPS 后应平滑跳过而不报错，实际报错: %v", err)
	}
	if len(events) == 0 || events[0].Level != domain.LevelWarn {
		t.Errorf("expected warning event for graceful skip")
	}
}

// BenchmarkAnchorIndex_Build 测试构建 1000 个资产锚点索引的性能
func BenchmarkAnchorIndex_Build(b *testing.B) {
	var batch []*domain.AssetContext
	baseTime := time.Date(2025, 10, 5, 8, 0, 0, 0, time.Local)

	for i := 0; i < 1000; i++ {
		t := baseTime.Add(time.Duration(i) * 10 * time.Second)
		actx := domain.NewAssetContext(domain.AssetGroup{BaseName: fmt.Sprintf("DSC_%04d", i), RawPath: fmt.Sprintf("/tmp/DSC_%04d.NEF", i)})
		actx.UpdateMetadata(domain.Metadata{
			DateTimeOriginal: t.Format("2006:01:02 15:04:05"),
			GPSPosition:      "31.2304 N, 121.4737 E",
		})
		actx.SetGPS(31.2304, 121.4737)
		batch = append(batch, actx)
	}

	capInst := NewCapability(Config{Runner: &mockRunner{}})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = capInst.buildAnchorIndex(batch, nil)
	}
}

// BenchmarkAnchorIndex_FindNearestAnchors 测试在 10000 个点位的索引中二分查找前后锚点的性能与吞吐量
func BenchmarkAnchorIndex_FindNearestAnchors(b *testing.B) {
	idx := &AnchorIndex{
		dailyAnchors: make(map[string][]GPSAnchor),
	}

	baseTime := time.Date(2025, 10, 5, 8, 0, 0, 0, time.Local)
	// 构造 10,000 个跨 10 天的锚点
	for i := 0; i < 10000; i++ {
		t := baseTime.Add(time.Duration(i) * 30 * time.Second)
		dKey := t.Format("2006-01-02")
		anchor := GPSAnchor{
			Time:     t,
			DateKey:  dKey,
			Lat:      31.23 + float64(i)*0.0001,
			Lon:      121.47 + float64(i)*0.0001,
			BaseName: fmt.Sprintf("DSC_%05d", i),
		}
		idx.allAnchors = append(idx.allAnchors, anchor)
		idx.dailyAnchors[dKey] = append(idx.dailyAnchors[dKey], anchor)
	}

	queryTime := baseTime.Add(2500 * 30 * time.Second).Add(12 * time.Second) // 位于中间某个点
	maxGap := 15 * time.Minute

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = idx.findNearestAnchors(queryTime, maxGap)
	}
}

// BenchmarkGPSInterpolate_ExecuteProcess 测试单张照片完整插值计算的性能与吞吐量
func BenchmarkGPSInterpolate_ExecuteProcess(b *testing.B) {
	var batch []*domain.AssetContext
	baseTime := time.Date(2025, 10, 5, 8, 0, 0, 0, time.Local)

	for i := 0; i < 200; i++ {
		t := baseTime.Add(time.Duration(i) * 15 * time.Second)
		actx := domain.NewAssetContext(domain.AssetGroup{BaseName: fmt.Sprintf("DSC_%04d", i), RawPath: fmt.Sprintf("/tmp/DSC_%04d.NEF", i)})
		if i == 0 || i == 199 {
			actx.SetGPS(31.23, 121.47)
			actx.UpdateMetadata(domain.Metadata{
				DateTimeOriginal: t.Format("2006:01:02 15:04:05"),
				GPSPosition:      "31.23 N, 121.47 E",
			})
		} else {
			actx.UpdateMetadata(domain.Metadata{
				DateTimeOriginal: t.Format("2006:01:02 15:04:05"),
			})
		}
		batch = append(batch, actx)
	}
	for _, actx := range batch {
		actx.Batch = batch
	}

	capInst := NewCapability(Config{
		Runner:     &mockRunner{},
		MaxTimeGap: 1 * time.Hour,
	})

	targetActx := batch[100]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = capInst.ExecuteProcess(context.Background(), targetActx, nil)
	}
}
