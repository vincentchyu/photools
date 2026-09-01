package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vincentchyu/photools/internal/capabilities/datearchive"
	"github.com/vincentchyu/photools/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photools/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/engine"
	"github.com/vincentchyu/photools/pkg/geocoding"
)

type mockRunner struct {
	metadataMap map[string]domain.Metadata
}

func (m *mockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	for _, a := range args {
		if a == "-ver" {
			return []byte("13.55"), nil
		}
	}
	if len(args) > 0 && m != nil && m.metadataMap != nil {
		var batchMetas []domain.Metadata
		for _, arg := range args {
			if meta, ok := m.metadataMap[arg]; ok {
				meta.SourceFile = arg
				batchMetas = append(batchMetas, meta)
			}
		}
		if len(batchMetas) > 0 {
			data, _ := json.Marshal(batchMetas)
			return data, nil
		}
		target := args[len(args)-1]
		if meta, ok := m.metadataMap[target]; ok {
			meta.SourceFile = target
			data, _ := json.Marshal([]domain.Metadata{meta})
			return data, nil
		}
	}
	return []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00","GPSPosition":"31.2304 N, 121.4737 E"}]`), nil
}

func TestPipelineOrchestrator_PhasingAndPriorityBuckets(t *testing.T) {
	cap1 := gpxmatch.NewCapability(gpxmatch.Config{Priority: 10})
	cap2 := reversegeocode.NewCapability(reversegeocode.Config{Priority: 20})
	cap3 := datearchive.NewCapability(datearchive.Config{Priority: 100})

	// 传入乱序的能力列表，验证自动按 Priority 升序分 Phase
	orch, err := NewOrchestrator(Config{
		SourceDir:    "/tmp/inbox",
		Capabilities: []domain.Capability{cap3, cap1, cap2},
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	phases := orch.Phases()
	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}
	if phases[0].Priority != 10 || phases[1].Priority != 20 || phases[2].Priority != 100 {
		t.Errorf("Phase 优先级排序异常: [%d, %d, %d]", phases[0].Priority, phases[1].Priority, phases[2].Priority)
	}

	stages := orch.Stages()
	if len(stages) != 6 { // Discover, Precheck, Geotag, Geocode, Archive, Complete
		t.Errorf("expected 6 stages, got %d: %v", len(stages), stages)
	}
}

func TestPipelineOrchestrator_FullPipeline_Execution(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "Inbox")
	processedDir := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(inboxDir, 0o755)

	rawPath := filepath.Join(inboxDir, "DSC_1010.NEF")
	jpgPath := filepath.Join(inboxDir, "DSC_1010.JPG")
	_ = os.WriteFile(rawPath, []byte("raw content"), 0o644)
	_ = os.WriteFile(jpgPath, []byte("jpg content"), 0o644)

	runner := &mockRunner{}
	geocoder := geocoding.NewReverseGeocoder()

	cap1 := gpxmatch.NewCapability(gpxmatch.Config{
		Runner:   runner,
		GPXFiles: []string{"/tmp/track.gpx"},
		Priority: 10,
	})
	cap2 := reversegeocode.NewCapability(reversegeocode.Config{
		Runner:   runner,
		Geocoder: geocoder,
		Priority: 20,
	})
	cap3 := datearchive.NewCapability(datearchive.Config{
		Runner:       runner,
		Archiver:     engine.NewArchiver(),
		ProcessedDir: processedDir,
		Priority:     100,
	})

	orch, err := NewOrchestrator(Config{
		SourceDir:    inboxDir,
		Capabilities: []domain.Capability{cap1, cap2, cap3},
		Workers:      2,
		Runner:       runner,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// 1. 测试 Plan
	plan, err := orch.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if plan.TotalAssets != 1 || plan.ReadyCount != 1 {
		t.Errorf("expected 1 ready asset, got total=%d ready=%d", plan.TotalAssets, plan.ReadyCount)
	}

	// 2. 测试 Execute
	eventCh := make(chan domain.ProgressEvent, 50)
	summary, issues, err := orch.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %v", len(issues), issues)
	}
	if summary.Success != 1 {
		t.Errorf("expected 1 success, got %d", summary.Success)
	}

	// 检查归档目录生成
	archivedFile := filepath.Join(processedDir, "2025", "1006", "DSC_2025-10-06_1010.NEF")
	if _, err := os.Stat(archivedFile); os.IsNotExist(err) {
		t.Errorf("expected archived file %s to exist", archivedFile)
	}
}

type errMockRunner struct{}

func (errMockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	for _, arg := range args {
		if arg == "-json" {
			return []byte(`[{"DateTimeOriginal":"2025:10:06 12:00:00","OffsetTimeOriginal":"+08:00"}]`), nil
		}
	}
	return nil, os.ErrNotExist
}

func TestPipelineOrchestrator_CascadeFailure_Phase1Error_SafelyRetainsFileAndReports(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "Inbox")
	processedDir := filepath.Join(tempDir, "Processed")
	reportFile := filepath.Join(tempDir, "report.md")
	_ = os.MkdirAll(inboxDir, 0o755)

	rawPath := filepath.Join(inboxDir, "DSC_2000.NEF")
	jpgPath := filepath.Join(inboxDir, "DSC_2000.JPG")
	_ = os.WriteFile(rawPath, []byte("raw content"), 0o644)
	_ = os.WriteFile(jpgPath, []byte("jpg content"), 0o644)

	runner := errMockRunner{} // 会在 GPX 阶段报错
	geocoder := geocoding.NewReverseGeocoder()

	cap1 := gpxmatch.NewCapability(gpxmatch.Config{
		Runner:   runner,
		GPXFiles: []string{"/tmp/track.gpx"},
		Priority: 10,
	})
	cap2 := reversegeocode.NewCapability(reversegeocode.Config{
		Runner:   runner,
		Geocoder: geocoder,
		Priority: 20,
	})
	cap3 := datearchive.NewCapability(datearchive.Config{
		Runner:       runner,
		Archiver:     engine.NewArchiver(),
		ProcessedDir: processedDir,
		Priority:     100,
	})

	orch, err := NewOrchestrator(Config{
		SourceDir:    inboxDir,
		Capabilities: []domain.Capability{cap1, cap2, cap3},
		Workers:      1,
		Runner:       runner,
		IssueFile:    reportFile,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	eventCh := make(chan domain.ProgressEvent, 50)
	var events []domain.ProgressEvent
	go func() {
		for e := range eventCh {
			events = append(events, e)
		}
	}()

	summary, issues, err := orch.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if summary.Failed != 1 {
		t.Errorf("expected 1 failed asset, got %d", summary.Failed)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	iss := issues[0]
	if iss.FailedStage != cap1.Name() {
		t.Errorf("expected FailedStage=%s, got %s", cap1.Name(), iss.FailedStage)
	}
	if len(iss.BlockedStages) == 0 {
		t.Errorf("expected BlockedStages to contain skipped stages, got empty")
	}

	// 验证源文件安全保留在 Inbox，没有被阶段 3 误移动
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Errorf("raw file should remain in Inbox when phase 1 fails")
	}

	// 验证生成了 Markdown 报告，且包含阻断阶段与文件状态
	reportContent, err := os.ReadFile(reportFile)
	if err != nil {
		t.Fatalf("failed to read generated report file: %v", err)
	}
	reportStr := string(reportContent)
	if !os.IsNotExist(err) && (len(reportStr) == 0 || !containsSubstring(reportStr, "中断阶段") || !containsSubstring(reportStr, "受影响跳过阶段")) {
		t.Errorf("report markdown missing stage diagnostic details:\n%s", reportStr)
	}
}

func TestPipelineOrchestrator_GPSInterpolate_AutoHealsMissingTrack(t *testing.T) {
	tempDir := t.TempDir()
	inbox := filepath.Join(tempDir, "Inbox")
	processed := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(inbox, 0o755)
	_ = os.MkdirAll(processed, 0o755)

	p1 := filepath.Join(inbox, "DSC_0001.NEF")
	p2 := filepath.Join(inbox, "DSC_0002.NEF")
	p3 := filepath.Join(inbox, "DSC_0003.NEF")

	_ = os.WriteFile(p1, []byte("p1"), 0o644)
	_ = os.WriteFile(p2, []byte("p2"), 0o644)
	_ = os.WriteFile(p3, []byte("p3"), 0o644)

	runner := &mockRunner{
		metadataMap: map[string]domain.Metadata{
			p1: {DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "40.0 N, 100.0 E"},
			p2: {DateTimeOriginal: "2026:08:24 10:05:00"}, // 无 GPS
			p3: {DateTimeOriginal: "2026:08:24 10:10:00", GPSPosition: "40.1 N, 100.2 E"},
		},
	}

	task, err := Build(PipelineOptions{
		BaseDir:           tempDir,
		SourceDir:         inbox,
		ProcessedDir:      processed,
		LogDir:            filepath.Join(tempDir, "logs"),
		Runner:            runner,
		EnableInterpolate: true,
		InterpolateWindow: 15 * time.Minute,
		EnableArchive:     true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	summary, issues, err := task.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if summary.Success != 3 {
		for i, iss := range issues {
			t.Logf("issue [%d]: %s: %s (status: %s)", i, iss.Asset.DisplayName(), iss.Reason, iss.CurrentStatus)
		}
		t.Errorf("expected all 3 assets processed successfully, got %d (issues: %d)", summary.Success, len(issues))
	}
}

func TestPipelineOrchestrator_AllowNoGPS_SoftDegradation(t *testing.T) {
	tempDir := t.TempDir()
	inbox := filepath.Join(tempDir, "Inbox")
	processed := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(inbox, 0o755)
	_ = os.MkdirAll(processed, 0o755)

	p1 := filepath.Join(inbox, "DSC_0001.NEF")
	_ = os.WriteFile(p1, []byte("p1"), 0o644)

	runner := &mockRunner{
		metadataMap: map[string]domain.Metadata{
			p1: {DateTimeOriginal: "2026:08:24 10:00:00"}, // 无 GPS
		},
	}

	task, err := Build(PipelineOptions{
		BaseDir:       tempDir,
		SourceDir:     inbox,
		ProcessedDir:  processed,
		LogDir:        filepath.Join(tempDir, "logs"),
		Runner:        runner,
		EnableGeocode: true,
		AllowNoGPS:    true, // 开启软降级
		EnableArchive: true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	summary, issues, err := task.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if summary.Success != 1 || len(issues) != 0 {
		t.Errorf("expected 1 success and 0 issues with AllowNoGPS=true, got success=%d, issues=%d", summary.Success, len(issues))
	}
}

func TestPipelineOrchestrator_Plan_BenignSkip_StillReadyForDownstream(t *testing.T) {
	tempDir := t.TempDir()
	inbox := filepath.Join(tempDir, "Inbox")
	_ = os.MkdirAll(inbox, 0o755)

	p1 := filepath.Join(inbox, "DSC_0001.NEF")
	_ = os.WriteFile(p1, []byte("p1"), 0o644)

	runner := &mockRunner{
		metadataMap: map[string]domain.Metadata{
			p1: {DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "39.9042 N, 116.3917 E"}, // 已有 GPS
		},
	}

	task, err := Build(PipelineOptions{
		BaseDir:           tempDir,
		SourceDir:         inbox,
		Runner:            runner,
		EnableInterpolate: true, // P15 应该跳过（已有 GPS）
		EnableGeocode:     true, // P20 应该就绪（写入地名）
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	plan, err := task.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan.TotalAssets != 1 || plan.ReadyCount != 1 {
		t.Errorf("expected TotalAssets=1 ReadyCount=1, got total=%d ready=%d (pending=%d warnings=%d)",
			plan.TotalAssets, plan.ReadyCount, plan.PendingCount, plan.WarningsCount)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || filepath.Base(s) != "" && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPipelineOrchestrator_RealtimeLogFileStreaming(t *testing.T) {
	tempDir := t.TempDir()
	inboxDir := filepath.Join(tempDir, "Inbox")
	logDir := filepath.Join(tempDir, "Logs")
	processedDir := filepath.Join(tempDir, "Processed")
	_ = os.MkdirAll(inboxDir, 0o755)
	_ = os.MkdirAll(logDir, 0o755)

	p1 := filepath.Join(inboxDir, "DSC_8888.NEF")
	_ = os.WriteFile(p1, []byte("raw content"), 0o644)

	runner := &mockRunner{
		metadataMap: map[string]domain.Metadata{
			p1: {DateTimeOriginal: "2026:08:24 10:00:00", GPSPosition: "39.9042 N, 116.3917 E"},
		},
	}

	task, err := Build(PipelineOptions{
		BaseDir:       tempDir,
		SourceDir:     inboxDir,
		LogDir:        logDir,
		ProcessedDir:  processedDir,
		Runner:        runner,
		EnableGeocode: true,
		EnableArchive: true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	eventCh := make(chan domain.ProgressEvent, 50)
	summary, issues, err := task.Execute(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if summary.Success != 1 || len(issues) != 0 {
		t.Fatalf("expected 1 success, got %d", summary.Success)
	}

	// 验证 Logs 目录下生成了 photools_latest.log
	latestLogFile := filepath.Join(logDir, "photools_latest.log")
	content, err := os.ReadFile(latestLogFile)
	if err != nil {
		t.Fatalf("failed to read latest log file: %v", err)
	}

	logStr := string(content)
	if !contains(logStr, "photools 流水线执行日志") {
		t.Errorf("expected log header in photools_latest.log, got:\n%s", logStr)
	}
	if !contains(logStr, "DSC_8888") {
		t.Errorf("expected asset progress log in photools_latest.log, got:\n%s", logStr)
	}
	if !contains(logStr, "流水线执行结算概览") {
		t.Errorf("expected summary footer in photools_latest.log, got:\n%s", logStr)
	}

	// 验证同时生成了时间戳格式的日志文件
	matches, _ := filepath.Glob(filepath.Join(logDir, "photools_*.log"))
	if len(matches) < 2 { // 至少应有 photools_latest.log 和 photools_YYYYMMDD_HHMMSS.log
		t.Errorf("expected timestamped log file in %s, found: %v", logDir, matches)
	}
}
