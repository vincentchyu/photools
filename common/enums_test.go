package common

import (
	"testing"
)

func TestEnums_StringAndValid(t *testing.T) {
	// 1. GPSSourceType
	if !GPSSourceCamera.IsValid() || GPSSourceCamera.String() != "camera" {
		t.Errorf("GPSSourceCamera invalid")
	}
	if !GPSSourceGPX.IsValid() || GPSSourceGPX.String() != "gpx" {
		t.Errorf("GPSSourceGPX invalid")
	}
	if !GPSSourceInterpolated.IsValid() || GPSSourceInterpolated.String() != "interpolated" {
		t.Errorf("GPSSourceInterpolated invalid")
	}
	if !GPSSourceManual.IsValid() || GPSSourceManual.String() != "manual" {
		t.Errorf("GPSSourceManual invalid")
	}
	if GPSSourceType("unknown").IsValid() {
		t.Errorf("expected false for unknown GPSSourceType")
	}

	// 2. GPSMatchMethodType
	if !GPSMatchMethodNativeCamera.IsValid() || GPSMatchMethodNativeCamera.String() != "native_camera" {
		t.Errorf("GPSMatchMethodNativeCamera invalid")
	}
	if !GPSMatchMethodTimeProximity.IsValid() || GPSMatchMethodTimeProximity.String() != "time_proximity" {
		t.Errorf("GPSMatchMethodTimeProximity invalid")
	}
	if !GPSMatchMethodSphericalLinearInterpol.IsValid() || GPSMatchMethodSphericalLinearInterpol.String() != "spherical_linear_interpolation" {
		t.Errorf("GPSMatchMethodSphericalLinearInterpol invalid")
	}
	if !GPSMatchMethodNearestNeighborAnchor.IsValid() || GPSMatchMethodNearestNeighborAnchor.String() != "nearest_neighbor_anchor" {
		t.Errorf("GPSMatchMethodNearestNeighborAnchor invalid")
	}
	if GPSMatchMethodType("invalid").IsValid() {
		t.Errorf("expected false for invalid GPSMatchMethodType")
	}

	// 3. SidecarPolicy
	if !PolicySmart.IsValid() || PolicySmart.String() != "smart" {
		t.Errorf("PolicySmart invalid")
	}
	if !PolicySidecarOnly.IsValid() || PolicySidecarOnly.String() != "sidecar_only" {
		t.Errorf("PolicySidecarOnly invalid")
	}
	if SidecarPolicy("invalid").IsValid() {
		t.Errorf("expected false for invalid SidecarPolicy")
	}

	// 4. CapabilityID
	if CapGPXMatching.String() != "gpx_matching" || CapGPSInterpolate.String() != "gps_interpolate" ||
		CapReverseGeocode.String() != "reverse_geocode" || CapDateArchive.String() != "date_archive" {
		t.Errorf("CapabilityID string mismatched")
	}

	// 5. PluginHealthStatus
	if HealthReady.String() != "ready" || HealthDegraded.String() != "degraded" || HealthFailed.String() != "failed" {
		t.Errorf("PluginHealthStatus string mismatched")
	}

	// 6. EventLevel & PipelineStage
	if LevelInfo.String() != "info" || StageGeotag.String() != "写入GPS" {
		t.Errorf("Level or Stage string mismatched")
	}

	// 7. GeocodeSourceType
	if GeocodeSourceEmbedded.String() != "embedded" || GeocodeSourceXMP.String() != "xmp" {
		t.Errorf("GeocodeSourceType string mismatched")
	}

	// 8. IssueKind
	if IssueKindFailure.String() != "failure" {
		t.Errorf("IssueKind string mismatched")
	}

	// 9. GetDefaultLogDir
	logDir := GetDefaultLogDir()
	if logDir == "" {
		t.Errorf("GetDefaultLogDir returned empty string")
	}
}
