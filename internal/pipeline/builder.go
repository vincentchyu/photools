package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vincentchyu/photools/internal/capabilities/datearchive"
	"github.com/vincentchyu/photools/internal/capabilities/gpsinterpolate"
	"github.com/vincentchyu/photools/internal/capabilities/gpxmatch"
	"github.com/vincentchyu/photools/internal/capabilities/reversegeocode"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/engine"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/pkg/geocoding"
)

// PipelineOptions 封装从 CLI 或 TUI 传入的组合配置
type PipelineOptions struct {
	BaseDir        string
	SourceDir      string
	GPXDir         string
	ProcessedDir   string
	LogDir         string
	Geosync        string
	RawExtensions  []string
	Workers        int
	Runner         exiftool.CommandRunner
	Geocoder       *geocoding.ReverseGeocoder
	NamingTemplate string

	// 插件能力开关
	EnableGPXMatch    bool          // 能力 1: GPX 轨迹匹配与 GPS 修正
	EnableInterpolate bool          // 能力 1.5: GPS 智能邻近推断与时间插值
	InterpolateWindow time.Duration // 插值推算时间窗口 (默认 15m)
	EnableGeocode     bool          // 能力 2: 逆地理编码与地名元数据写入
	AllowNoGPS        bool          // 无 GPS 坐标时允许跳过地名写入直接归档 (软降级)
	EnableArchive     bool          // 能力 3: 按拍摄日期归档与规范重命名
	InPlaceArchive    bool          // 归档时原地规范重命名，不建立 Processed/YYYY/MMDD/ 子目录

	// 扁平/直接目录模式与测试备份
	FlatMode     bool                  // 忽略传统分层目录结构，在指定目录下直接扫描并原地处理/保存
	EnableBackup bool                  // 是否开启测试备份模式（处理前快照备份原始文件）
	BackupDir    string                // 自定义测试备份目录（默认 <BaseDir>/Inbox_bak）
	Session      *config.SessionConfig // 运行时会话配置
}

// Build 根据启用的插件开关动态构建流水线编排器 Task
func Build(opts PipelineOptions) (domain.Task, error) {
	if !opts.EnableGPXMatch && !opts.EnableInterpolate && !opts.EnableGeocode && !opts.EnableArchive {
		return nil, fmt.Errorf("必须至少启用一项能力插件")
	}

	baseDir := opts.BaseDir
	if baseDir == "" {
		if wd, err := os.Getwd(); err == nil && wd != "" {
			baseDir = wd
		} else {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, "Pictures", "GPS")
		}
	}
	baseDir, _ = filepath.Abs(baseDir)

	sourceDir := opts.SourceDir
	if sourceDir == "" {
		if opts.FlatMode {
			sourceDir = baseDir
		} else {
			sourceDir = filepath.Join(baseDir, "Inbox")
		}
	}
	sourceDir, _ = filepath.Abs(sourceDir)

	gpxDir := opts.GPXDir
	if gpxDir == "" {
		if opts.Session != nil && opts.Session.Global.GPXDir != "" {
			gpxDir = opts.Session.Global.GPXDir
		} else {
			gpxDir = config.DefaultGPXDir()
		}
	}
	if len(gpxDir) >= 2 && gpxDir[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			gpxDir = filepath.Join(home, gpxDir[2:])
		}
	}
	gpxDir, _ = filepath.Abs(gpxDir)

	processedDir := opts.ProcessedDir
	if processedDir == "" {
		if opts.FlatMode {
			processedDir = sourceDir
		} else {
			processedDir = filepath.Join(baseDir, "Processed")
		}
	}
	processedDir, _ = filepath.Abs(processedDir)

	logDir := opts.LogDir
	if logDir == "" {
		logDir = filepath.Join(baseDir, "Logs")
	}
	logDir, _ = filepath.Abs(logDir)
	_ = os.MkdirAll(logDir, 0o755)

	issueFile := filepath.Join(logDir, "inbox_pending_report_latest.md")
	lockPath := filepath.Join(logDir, "pipeline.lock")

	runner := opts.Runner
	if runner == nil {
		runner = exiftool.DefaultRunner()
	}

	rawExts := opts.RawExtensions
	if len(rawExts) == 0 {
		rawExts = []string{"nef", "cr3", "arw", "dng", "raf", "rw2", "orf"}
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// 读取外部 plugins.json 配置或传入的 SessionConfig 设定的 Priority 与 Options
	cfg, _ := config.LoadPluginsConfig("")
	session := opts.Session
	if session == nil {
		session = config.NewSessionConfig(cfg, baseDir)
	}

	var activeCaps []domain.Capability

	// 1. 能力 1: GPX 轨迹匹配
	if opts.EnableGPXMatch {
		gpxFiles, _ := engine.ListGPXFiles(gpxDir)
		p := 10
		if meta := cfg.FindPluginMeta(domain.CapGPXMatching); meta != nil && meta.Priority > 0 {
			p = meta.Priority
		}
		if opt, ok := session.Plugins[domain.CapGPXMatching]; ok && opt.Priority > 0 {
			p = opt.Priority
		}
		cap1 := gpxmatch.NewCapability(gpxmatch.Config{
			Runner:   runner,
			GPXFiles: gpxFiles,
			Priority: p,
		})
		_ = cap1.Configure(session.GetPluginOptions(domain.CapGPXMatching))
		if opts.Geosync != "" && opts.Geosync != "0" {
			cap1.SetGeosync(opts.Geosync)
		}
		activeCaps = append(activeCaps, cap1)
	}

	// 1.5. 能力 1.5: GPS 智能邻近推断与时间插值
	if opts.EnableInterpolate {
		p := 15
		if meta := cfg.FindPluginMeta(domain.CapGPSInterpolate); meta != nil && meta.Priority > 0 {
			p = meta.Priority
		}
		if opt, ok := session.Plugins[domain.CapGPSInterpolate]; ok && opt.Priority > 0 {
			p = opt.Priority
		}
		cap15 := gpsinterpolate.NewCapability(gpsinterpolate.Config{
			Runner:     runner,
			Priority:   p,
			AllowNoGPS: opts.AllowNoGPS,
		})
		_ = cap15.Configure(session.GetPluginOptions(domain.CapGPSInterpolate))
		if opts.InterpolateWindow > 0 {
			cap15.SetMaxTimeGap(opts.InterpolateWindow)
		}
		activeCaps = append(activeCaps, cap15)
	}

	// 2. 能力 2: 逆地理编码地名写入
	if opts.EnableGeocode {
		p := 20
		if meta := cfg.FindPluginMeta(domain.CapReverseGeocode); meta != nil && meta.Priority > 0 {
			p = meta.Priority
		}
		if opt, ok := session.Plugins[domain.CapReverseGeocode]; ok && opt.Priority > 0 {
			p = opt.Priority
		}
		geocoder := opts.Geocoder
		if geocoder == nil {
			geocoder = geocoding.GetDefault()
		}
		cap2 := reversegeocode.NewCapability(reversegeocode.Config{
			Runner:     runner,
			Geocoder:   geocoder,
			Priority:   p,
			AllowNoGPS: opts.AllowNoGPS,
		})
		_ = cap2.Configure(session.GetPluginOptions(domain.CapReverseGeocode))
		activeCaps = append(activeCaps, cap2)
	}

	// 3. 能力 3: 按拍摄日期归档
	if opts.EnableArchive {
		p := 100
		if meta := cfg.FindPluginMeta(domain.CapDateArchive); meta != nil && meta.Priority > 0 {
			p = meta.Priority
		}
		if opt, ok := session.Plugins[domain.CapDateArchive]; ok && opt.Priority > 0 {
			p = opt.Priority
		}
		inPlace := opts.InPlaceArchive || (opts.FlatMode && opts.SourceDir == opts.ProcessedDir)
		cap3 := datearchive.NewCapability(datearchive.Config{
			Runner:         runner,
			Archiver:       engine.NewArchiver(opts.NamingTemplate),
			ProcessedDir:   processedDir,
			NamingTemplate: opts.NamingTemplate,
			InPlace:        inPlace,
			Priority:       p,
		})
		_ = cap3.Configure(session.GetPluginOptions(domain.CapDateArchive))
		if inPlace {
			cap3.SetInPlace(true)
		}
		activeCaps = append(activeCaps, cap3)
	}

	// 仅当明确启用了 EnableBackup 时才设置 backupDir
	backupDir := ""
	if opts.EnableBackup {
		backupDir = opts.BackupDir
		if backupDir == "" {
			backupDir = filepath.Join(baseDir, "Inbox_bak")
		}
		backupDir, _ = filepath.Abs(backupDir)
	}

	return NewOrchestrator(Config{
		SourceDir:     sourceDir,
		Capabilities:  activeCaps,
		RawExtensions: rawExts,
		Workers:       workers,
		Runner:        runner,
		LogDir:        logDir,
		IssueFile:     issueFile,
		LockPath:      lockPath,
		BackupDir:     backupDir,
	})
}
