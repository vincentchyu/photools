package main

/*
#include <stdlib.h>

// 回调函数类型定义
typedef void (*Photools_InitCallback)(const char* plugin_id, const char* name, const char* stage, const char* message, double percent, const char* status, const char* err_msg);
typedef void (*Photools_EventCallback)(const char* stage, const char* level, const char* message, const char* asset_name, int cur_index, int total_items);
typedef void (*Photools_DoneCallback)(int success_count, int skip_count, int fail_count, double duration_secs, const char* err_msg);
typedef void (*Photools_LogCallback)(const char* message);

// C 包装函数方便 Go 触发 C 函数指针
static inline void bridge_init_cb(Photools_InitCallback cb, const char* plugin_id, const char* name, const char* stage, const char* message, double percent, const char* status, const char* err_msg) {
    if (cb != NULL) {
        cb(plugin_id, name, stage, message, percent, status, err_msg);
    }
}

static inline void bridge_event_cb(Photools_EventCallback cb, const char* stage, const char* level, const char* message, const char* asset_name, int cur_index, int total_items) {
    if (cb != NULL) {
        cb(stage, level, message, asset_name, cur_index, total_items);
    }
}

static inline void bridge_done_cb(Photools_DoneCallback cb, int success, int skip, int fail, double dur, const char* err_msg) {
    if (cb != NULL) {
        cb(success, skip, fail, dur, err_msg);
    }
}

static inline void bridge_log_cb(Photools_LogCallback cb, const char* msg) {
    if (cb != NULL) {
        cb(msg);
    }
}
*/
import "C"
import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"github.com/vincentchyu/photo-processing/internal/capabilities/datearchive"
	"github.com/vincentchyu/photo-processing/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photo-processing/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photo-processing/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photo-processing/internal/config"
	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/engine"
	"github.com/vincentchyu/photo-processing/internal/exiftool"
	"github.com/vincentchyu/photo-processing/internal/geocoding"
	"github.com/vincentchyu/photo-processing/internal/geodata"
	"github.com/vincentchyu/photo-processing/internal/pipeline"
)

var (
	engineLock    sync.Mutex
	currentCancel context.CancelFunc
	initOnce      sync.Once
	isInitialized bool
)

// PipelineOptionsJSON 定义通过 C 接口传递的流水线配置结构
type PipelineOptionsJSON struct {
	BaseDir           string `json:"base_dir"`
	SourceDir         string `json:"source_dir"`
	GPXDir            string `json:"gpx_dir"`
	ProcessedDir      string `json:"processed_dir"`
	FlatMode          bool   `json:"flat_mode"`
	InPlace           bool   `json:"in_place"`
	Geosync           string `json:"geosync"`
	RawExtensions     string `json:"raw_extensions"`
	Workers           int    `json:"workers"`
	EnableGPXMatch    bool   `json:"enable_gpx_match"`
	EnableInterpolate bool   `json:"enable_interpolate"`
	InterpolateWindow string `json:"interpolate_window"`
	EnableGeocode     bool   `json:"enable_geocode"`
	AllowNoGPS        bool   `json:"allow_no_gps"`
	EnableArchive     bool   `json:"enable_archive"`
	TestBackup        bool   `json:"test_backup"`
	BackupDir         string `json:"backup_dir"`
}

//export Photools_Init
func Photools_Init(cb C.Photools_InitCallback) {
	initOnce.Do(func() {
		// 启动并发初始化四大能力插件与离线地理空间索引
		runner := exiftool.DefaultRunner()
		caps := []domain.Capability{
			gpxmatch.NewCapability(gpxmatch.Config{Runner: runner}),
			gpsinterpolate.NewCapability(gpsinterpolate.Config{Runner: runner}),
			reversegeocode.NewCapability(reversegeocode.Config{Runner: runner}),
			datearchive.NewCapability(datearchive.Config{Runner: runner}),
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, capItem := range caps {
			wg.Add(1)
			go func(c domain.Capability) {
				defer wg.Done()
				_ = c.Init(ctx, func(rep domain.PluginInitReport) {
					if cb != nil {
						cPluginID := C.CString(string(rep.PluginID))
						cName := C.CString(rep.Name)
						cStage := C.CString(rep.Stage)
						cMsg := C.CString(rep.Message)
						cStatus := C.CString(string(rep.Status))
						var cErr *C.char
						if rep.Err != nil {
							cErr = C.CString(rep.Err.Error())
						} else {
							cErr = C.CString("")
						}

						C.bridge_init_cb(cb, cPluginID, cName, cStage, cMsg, C.double(rep.Percent), cStatus, cErr)

						C.free(unsafe.Pointer(cPluginID))
						C.free(unsafe.Pointer(cName))
						C.free(unsafe.Pointer(cStage))
						C.free(unsafe.Pointer(cMsg))
						C.free(unsafe.Pointer(cStatus))
						C.free(unsafe.Pointer(cErr))
					}
				})
			}(capItem)
		}

		wg.Wait()
		isInitialized = true
	})
}

//export Photools_RunPipeline
func Photools_RunPipeline(optionsJSON *C.char, eventCB C.Photools_EventCallback, doneCB C.Photools_DoneCallback) {
	go func() {
		engineLock.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		currentCancel = cancel
		engineLock.Unlock()

		defer func() {
			engineLock.Lock()
			currentCancel = nil
			engineLock.Unlock()
		}()

		rawJSON := C.GoString(optionsJSON)
		var opts PipelineOptionsJSON
		if err := json.Unmarshal([]byte(rawJSON), &opts); err != nil {
			if doneCB != nil {
				cErr := C.CString("解析配置 JSON 失败: " + err.Error())
				C.bridge_done_cb(doneCB, 0, 0, 0, 0, cErr)
				C.free(unsafe.Pointer(cErr))
			}
			return
		}

		win, _ := time.ParseDuration(opts.InterpolateWindow)
		if win <= 0 {
			win = 15 * time.Minute
		}

		pluginsCfg, _ := config.LoadPluginsConfig("")
		sessionCfg := config.NewSessionConfig(pluginsCfg, opts.BaseDir)

		task, err := pipeline.Build(pipeline.PipelineOptions{
			BaseDir:           opts.BaseDir,
			SourceDir:         opts.SourceDir,
			GPXDir:            opts.GPXDir,
			ProcessedDir:      opts.ProcessedDir,
			FlatMode:          opts.FlatMode,
			InPlaceArchive:    opts.InPlace,
			Geosync:           opts.Geosync,
			RawExtensions:     sessionCfg.Global.RawExtensions,
			Workers:           opts.Workers,
			EnableGPXMatch:    opts.EnableGPXMatch,
			EnableInterpolate: opts.EnableInterpolate,
			InterpolateWindow: win,
			EnableGeocode:     opts.EnableGeocode,
			AllowNoGPS:        opts.AllowNoGPS,
			EnableArchive:     opts.EnableArchive,
			EnableBackup:      opts.TestBackup,
			BackupDir:         opts.BackupDir,
			Session:           sessionCfg,
		})

		if err != nil {
			if doneCB != nil {
				cErr := C.CString("初始化流水线失败: " + err.Error())
				C.bridge_done_cb(doneCB, 0, 0, 0, 0, cErr)
				C.free(unsafe.Pointer(cErr))
			}
			return
		}

		eventCh := make(chan domain.ProgressEvent, 100)
		go func() {
			for evt := range eventCh {
				if eventCB != nil {
					cStage := C.CString(string(evt.Stage))
					cLevel := C.CString(string(evt.Level))
					cMsg := C.CString(evt.Message)
					var cAsset *C.char
					if evt.Asset != nil {
						cAsset = C.CString(evt.Asset.DisplayName())
					} else {
						cAsset = C.CString("")
					}

					C.bridge_event_cb(eventCB, cStage, cLevel, cMsg, cAsset, C.int(evt.CurrentIndex), C.int(evt.TotalItems))

					C.free(unsafe.Pointer(cStage))
					C.free(unsafe.Pointer(cLevel))
					C.free(unsafe.Pointer(cMsg))
					C.free(unsafe.Pointer(cAsset))
				}
			}
		}()

		start := time.Now()
		summary, _, execErr := task.Execute(ctx, eventCh)
		close(eventCh)
		dur := time.Since(start).Seconds()

		if doneCB != nil {
			var errMsg string
			if execErr != nil {
				errMsg = execErr.Error()
			}
			cErr := C.CString(errMsg)
			successCount := 0
			skipCount := 0
			failCount := 0
			if summary != nil {
				successCount = summary.Success
				skipCount = summary.Skipped
				failCount = summary.Failed
			}
			C.bridge_done_cb(doneCB, C.int(successCount), C.int(skipCount), C.int(failCount), C.double(dur), cErr)
			C.free(unsafe.Pointer(cErr))
		}
	}()
}

//export Photools_LookupCoordinates
func Photools_LookupCoordinates(lat, lon, alt C.double, debug C.int) *C.char {
	rg := geocoding.GetDefault()
	if !rg.IsInitialized() {
		_ = rg.InitProgressive(context.Background(), nil)
	}

	if debug != 0 {
		qStart := time.Now()
		loc, bestPt, distKm, debugStats, loadStats := rg.LookupDetailedWithDebug(float64(lat), float64(lon), 5)
		queryDur := time.Since(qStart)
		if loc == nil {
			return C.CString("{}")
		}

		debugReport := geocoding.FormatDebugReport(float64(lat), float64(lon), float64(alt), loc, bestPt, distKm, debugStats, loadStats, queryDur)

		var candidates []map[string]any
		for i, c := range debugStats.TopCandidates {
			pt := c.Point
			name := pt.NameZH
			if name == "" {
				name = pt.Name
			}
			if name == "" {
				if pt.District != "" {
					name = pt.District
				} else if pt.City != "" {
					name = pt.City
				} else {
					name = pt.Province
				}
			}
			featureDesc := geocoding.FormatFeatureCodeZH(pt.FeatureClass, pt.FeatureCode)
			hierarchy := geocoding.FormatPointLocation(&pt)

			ele := pt.DEM
			if ele == 0 {
				ele = pt.Elevation
			}

			candidates = append(candidates, map[string]any{
				"rank":               i + 1,
				"name":               pt.Name,
				"name_zh":            name,
				"lat":                pt.Lat,
				"lon":                pt.Lon,
				"distance_km":        c.DistanceKm,
				"elevation":          ele,
				"geoname_id":         pt.GeoNameID,
				"source":             pt.Source,
				"feature_desc":       featureDesc,
				"location_hierarchy": hierarchy,
			})
		}

		res := map[string]any{
			"country":           loc.Country,
			"country_code":      loc.CountryCode,
			"province":          loc.Province,
			"city":              loc.City,
			"district":          loc.District,
			"timezone":          loc.Timezone,
			"elevation":         loc.Elevation,
			"distance_km":       distKm,
			"source":            loc.Source,
			"formatted_summary": loc.FormatSummary(),
			"debug_text":        debugReport,
			"candidates":        candidates,
			"debug_info": map[string]any{
				"total_points":    loadStats.TotalPoints,
				"builtin_points":  loadStats.BuiltinPoints,
				"custom_points":   loadStats.CustomPoints,
				"packs_count":     len(loadStats.Packs),
				"tree_build_ms":   float64(loadStats.TreeBuildTime.Microseconds()) / 1000.0,
				"visited_nodes":   debugStats.VisitedNodes,
				"prune_rate":      debugStats.PruneRate,
				"candidates":      candidates,
				"best_point_name": bestPt.NameZH,
			},
		}

		data, err := json.Marshal(res)
		if err != nil {
			return C.CString("{}")
		}
		return C.CString(string(data))
	}

	loc, _, distKm := rg.LookupDetailed(float64(lat), float64(lon))
	if loc == nil {
		return C.CString("{}")
	}

	res := map[string]any{
		"country":           loc.Country,
		"country_code":      loc.CountryCode,
		"province":          loc.Province,
		"city":              loc.City,
		"district":          loc.District,
		"timezone":          loc.Timezone,
		"elevation":         loc.Elevation,
		"distance_km":       distKm,
		"source":            loc.Source,
		"formatted_summary": loc.FormatSummary(),
	}

	data, err := json.Marshal(res)
	if err != nil {
		return C.CString("{}")
	}

	return C.CString(string(data))
}

//export Photools_ListGeodataPacks
func Photools_ListGeodataPacks() *C.char {
	mgr, err := geodata.NewManager()
	if err != nil {
		return C.CString("[]")
	}

	list := mgr.ListContinents()
	data, err := json.Marshal(list)
	if err != nil {
		return C.CString("[]")
	}

	return C.CString(string(data))
}

//export Photools_InstallGeodata
func Photools_InstallGeodata(target *C.char, logCB C.Photools_LogCallback) C.int {
	targetStr := C.GoString(target)
	mgr, err := geodata.NewManager()
	if err != nil {
		return -1
	}

	logFn := func(msg string) {
		if logCB != nil {
			cMsg := C.CString(msg)
			C.bridge_log_cb(logCB, cMsg)
			C.free(unsafe.Pointer(cMsg))
		}
	}

	if err := mgr.Install(context.Background(), targetStr, logFn); err != nil {
		return -1
	}
	geocoding.ResetDefault()
	_ = geocoding.GetDefault().InitProgressive(context.Background(), nil)
	return 0
}

//export Photools_RemoveGeodata
func Photools_RemoveGeodata(target *C.char) C.int {
	targetStr := C.GoString(target)
	mgr, err := geodata.NewManager()
	if err != nil {
		return -1
	}

	if err := mgr.Remove(targetStr); err != nil {
		return -1
	}
	geocoding.ResetDefault()
	return 0
}

//export Photools_CreateBackup
func Photools_CreateBackup(sourceDir, backupDir *C.char) C.int {
	src := C.GoString(sourceDir)
	bak := C.GoString(backupDir)

	count, err := engine.CreateBackup(src, bak)
	if err != nil {
		return -1
	}
	return C.int(count)
}

//export Photools_RestoreBackup
func Photools_RestoreBackup(baseDir, backupDir, targetDir *C.char, cleanProcessed C.int) C.int {
	bBase := C.GoString(baseDir)
	bBak := C.GoString(backupDir)
	bTgt := C.GoString(targetDir)

	if bBak == "" {
		bBak = filepath.Join(bBase, "Inbox_bak")
	}
	if bTgt == "" {
		bTgt = filepath.Join(bBase, "Inbox")
	}

	count, err := engine.RestoreBackup(bBak, bTgt)
	if err != nil {
		return -1
	}

	if cleanProcessed != 0 {
		pDir := filepath.Join(bBase, "Processed")
		_ = os.RemoveAll(pDir)
		_ = os.MkdirAll(pDir, 0755)
	}

	return C.int(count)
}

//export Photools_CancelTask
func Photools_CancelTask() {
	engineLock.Lock()
	defer engineLock.Unlock()
	if currentCancel != nil {
		currentCancel()
	}
}

//export Photools_InspectPhotoMetadata
func Photools_InspectPhotoMetadata(filePath *C.char) *C.char {
	path := C.GoString(filePath)
	if path == "" {
		return C.CString("{}")
	}

	meta, err := exiftool.InspectPhotoMetadata(exiftool.DefaultRunner(), path)
	if err != nil {
		return C.CString("{}")
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return C.CString("{}")
	}

	return C.CString(string(data))
}

//export Photools_Shutdown
func Photools_Shutdown() {
	Photools_CancelTask()
	exiftool.CloseDefaultPool()
}

//export Photools_FreeString
func Photools_FreeString(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

func main() {}
