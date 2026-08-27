package reversegeocode

import (
	"context"
	"strings"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/pkg/geocoding"
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

func TestReverseGeocodeCapability_PlanPrecheck(t *testing.T) {
	capInst := NewCapability(Config{})

	// 1. 无 GPS 资产
	actxNoGPS := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_001"})
	plan1 := capInst.PlanPrecheck(context.Background(), actxNoGPS)
	if plan1.CanProcess {
		t.Errorf("expected CanProcess=false when no GPS, got true")
	}

	// 2. 具备 GPS 资产
	actxWithGPS := domain.NewAssetContext(domain.AssetGroup{BaseName: "DSC_002"})
	actxWithGPS.SetGPS(31.2304, 121.4737)
	plan2 := capInst.PlanPrecheck(context.Background(), actxWithGPS)
	if !plan2.CanProcess {
		t.Errorf("expected CanProcess=true when GPS exists, got false: %v", plan2.Warning)
	}
}

func TestReverseGeocodeCapability_ExecuteProcess_Success(t *testing.T) {
	runner := &mockRunner{}
	geocoder := geocoding.NewReverseGeocoder()

	capInst := NewCapability(Config{
		Runner:   runner,
		Geocoder: geocoder,
	})

	actx := domain.NewAssetContext(domain.AssetGroup{
		BaseName: "DSC_001",
		RawPath:  "/tmp/DSC_001.NEF",
		JPGPath:  "/tmp/DSC_001.JPG",
		XMPPath:  "/tmp/DSC_001.xmp",
	})
	// 上海人民广场经纬度
	actx.SetGPS(31.2304, 121.4737)

	var events []domain.ProgressEvent
	err := capInst.ExecuteProcess(context.Background(), actx, func(e domain.ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actx.Location == nil {
		t.Fatalf("expected location to be populated, got nil")
	}
	if actx.Location.City == "" {
		t.Errorf("expected city to be populated, got empty")
	}
	if len(actx.ModifiedFiles) != 3 {
		t.Errorf("expected 3 modified files (RAW, JPG, XMP), got %d", len(actx.ModifiedFiles))
	}
	if len(events) == 0 {
		t.Errorf("expected progress event")
	}
}

func TestReverseGeocodeCapability_Init(t *testing.T) {
	capInst := NewCapability(Config{})
	var reports []domain.PluginInitReport
	err := capInst.Init(context.Background(), func(rep domain.PluginInitReport) {
		reports = append(reports, rep)
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if len(reports) == 0 {
		t.Fatalf("expected init reports, got 0")
	}
	last := reports[len(reports)-1]
	if last.Status != domain.HealthReady {
		t.Errorf("expected HealthReady, got %v", last.Status)
	}
}
