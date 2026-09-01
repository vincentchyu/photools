package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/vincentchyu/photools/common"
	"github.com/vincentchyu/photools/internal/completion"
	"github.com/vincentchyu/photools/internal/config"
	"github.com/vincentchyu/photools/internal/domain"
	"github.com/vincentchyu/photools/internal/engine"
	"github.com/vincentchyu/photools/internal/exiftool"
	"github.com/vincentchyu/photools/internal/i18n"
	"github.com/vincentchyu/photools/internal/pipeline"
	"github.com/vincentchyu/photools/internal/tui"
	"github.com/vincentchyu/photools/pkg/geocoding"
	"github.com/vincentchyu/photools/pkg/geodata"
)

// Version 当前编译版本号（支持编译期 -ldflags "-X main.Version=vX.Y.Z" 注入）
var Version = common.CurrentVersion

func exitApp(code int) {
	exiftool.CloseDefaultPool()
	os.Exit(code)
}

func printVersion() {
	fmt.Printf("photools %s\n", Version)
}

func main() {
	// 确保退出时释放常驻 ExifTool 进程池
	defer exiftool.CloseDefaultPool()

	// 监听系统退出信号，确保 Ctrl+C 时也能清理子进程
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		exitApp(130)
	}()

	// 启动时自动同步并升级 ~/.config/photools/plugins.json 配置
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg)
	if sessionCfg.Global.Language != "" {
		i18n.SetLanguage(sessionCfg.Global.Language)
	}

	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "completion":
			runCompletion(os.Args[2:])
			os.Exit(0)
		case "version", "-v", "--version":
			printVersion()
			os.Exit(0)
		}
	}

	defaultBaseDir, _ := defaultBaseDir()

	if len(os.Args) < 2 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			if err := tui.Run(defaultBaseDir); err != nil {
				fmt.Fprintf(os.Stderr, "运行 TUI 失败: %v\n", err)
				exitApp(1)
			}
			return
		}
		printUsage()
		exitApp(0)
	}

	switch os.Args[1] {
	case "tui":
		if err := tui.Run(defaultBaseDir); err != nil {
			fmt.Fprintf(os.Stderr, "运行 TUI 失败: %v\n", err)
			exitApp(1)
		}
	case "language", "lang":
		runLanguage(os.Args[2:])
		exitApp(0)
	case "geotag":
		runGeotag(defaultBaseDir)
	case "geocode":
		runGeocode(defaultBaseDir)
	case "pipeline":
		runPipeline(defaultBaseDir)
	case "organize-by-date":
		runOrganizeByDate()
	case "restore-test", "restore-backup":
		runRestoreTest(defaultBaseDir)
	case "backup", "snapshot":
		runBackup(defaultBaseDir)
	case "inspect":
		runInspect(os.Args[2:])
	case "geodata":
		runGeoData(os.Args[2:])
	case "completion":
		runCompletion(os.Args[2:])
	case "version", "-v", "--version":
		printVersion()
		exitApp(0)
	case "-h", "--help", "help":
		printUsage()
	default:
		// 如果直接传入经纬度参数 (例如 ./photools 31.23 121.47)
		if lat, lon, alt, debug, err := parseCoordinatesWithDebug(os.Args[1:]); err == nil {
			runGeoDataTestCoordinates(lat, lon, alt, debug)
			return
		}

		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		printUsage()
		exitApp(1)
	}
}

func runLanguage(args []string) {
	if len(args) == 0 {
		fmt.Printf("当前语言 (Current Language): %s\n", i18n.GetLanguage())
		fmt.Println("支持的语言 (Supported): zh-CN, en-US")
		fmt.Println("用法 (Usage): photools language [zh-CN|en-US|zh|en]")
		return
	}

	target := i18n.NormalizeLanguage(args[0])
	i18n.SetLanguage(target)

	// 同步持久化写入 ~/.config/photools/plugins.json
	pluginsCfg, err := config.LoadPluginsConfig("")
	if err != nil || pluginsCfg == nil {
		def := config.DefaultPluginsConfig()
		pluginsCfg = &def
	}
	pluginsCfg.Global.Language = target
	if err := config.SavePluginsConfig("", pluginsCfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
	}

	if i18n.IsChinese() {
		fmt.Printf("✅ 界面与补全语言已成功切换为: %s\n", target)
	} else {
		fmt.Printf("✅ Interface & completion language switched to: %s\n", target)
	}
}

func defaultBaseDir() (string, error) {
	wd, err := os.Getwd()
	if err == nil && wd != "" {
		return wd, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Pictures", "GPS"), nil
}

func printUsage() {
	fmt.Printf("📷 photools %s - %s\n\n", Version, i18n.T("cliSubtitle"))
	fmt.Println(i18n.T("cliUsageTitle"))
	fmt.Printf("  %s\n\n", i18n.T("cliUsagePattern"))

	fmt.Println(i18n.T("cliCoreCommands"))
	fmt.Printf("  %-40s%s\n", "photools", i18n.T("cliCmdTui"))
	fmt.Printf("  %-40s%s\n", "photools tui", i18n.T("cliCmdTuiExplicit"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsagePipeline"), i18n.T("cliCmdPipeline"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeotag"), i18n.T("cliCmdGeotag"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeocode"), i18n.T("cliCmdGeocode"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageInspect"), i18n.T("cliCmdInspect"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeodata"), i18n.T("cliCmdGeodata"))
	fmt.Printf("  %-40s%s\n", "photools language [zh|en]", i18n.T("cliCmdLanguage"))
	fmt.Printf("  %-40s%s\n", "photools completion [shell]", i18n.T("cliCmdCompletion"))
	fmt.Printf("  %-40s%s\n\n", "photools version", i18n.T("cliCmdVersion"))

	fmt.Println(i18n.T("cliCommonOptions"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageBaseDir"), i18n.T("cliOptBaseDir"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageSourceDir"), i18n.T("cliOptSourceDir"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageGpxDir"), i18n.T("cliOptGpxDir"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageProcessedDir"), i18n.T("cliOptProcessedDir"))
	fmt.Printf("  %-40s%s\n", "-flat", i18n.T("cliOptFlat"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageSidecarPolicy"), i18n.T("cliOptSidecarPolicy"))
	fmt.Printf("  %-40s%s\n", "-sidecar-only", i18n.T("cliOptSidecarOnly"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageCompanionExts"), i18n.T("cliOptCompanionExts"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageRawExts"), i18n.T("cliOptRawExts"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageGeosync"), i18n.T("cliOptGeosync"))
	fmt.Printf("  %-40s%s\n", "-interpolate", i18n.T("cliOptInterpolate"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageInterpolateWindow"), i18n.T("cliOptInterpolateWindow"))
	fmt.Printf("  %-40s%s\n", "-allow-no-gps", i18n.T("cliOptAllowNoGps"))
	fmt.Printf("  %-40s%s\n", "-in-place", i18n.T("cliOptInPlace"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliOptUsageWorkers"), i18n.T("cliOptWorkers"))
	fmt.Printf("  %-40s%s\n", "-test, -backup", i18n.T("cliOptTestBackup"))
	fmt.Printf("  %-40s%s\n\n", i18n.T("cliOptUsageBackupDir"), i18n.T("cliOptBackupDir"))

	fmt.Println(i18n.T("cliGeodataCommands"))
	fmt.Printf("  %-40s%s\n", "photools geodata list", i18n.T("cliGeodataList"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeodataInstall"), i18n.T("cliGeodataInstall"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeodataRemove"), i18n.T("cliGeodataRemove"))
	fmt.Printf("  %-40s%s\n", "photools geodata info", i18n.T("cliGeodataInfo"))
	fmt.Printf("  %-40s%s\n", i18n.T("cliUsageGeodataTest"), i18n.T("cliGeodataTest"))
}

func runGeotag(defaultBaseDir string) {
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, defaultBaseDir)

	fs := flag.NewFlagSet("geotag", flag.ExitOnError)
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, i18n.T("cliOptBaseDir"))
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, i18n.T("cliOptSourceDir"))
	gpxDir := fs.String("gpx-dir", sessionCfg.Global.GPXDir, i18n.T("cliOptGpxDir"))
	processedDir := fs.String("processed-dir", sessionCfg.Global.TargetDir, i18n.T("cliOptProcessedDir"))
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, i18n.T("cliOptFlat"))
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, i18n.T("cliOptSidecarPolicy"))
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, i18n.T("cliOptSidecarOnly"))
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), i18n.T("cliOptCompanionExts"))
	inPlace := fs.Bool("in-place", sessionCfg.GetBoolOption(domain.CapDateArchive, "in_place", false), i18n.T("cliOptInPlace"))
	geosync := fs.String("geosync", sessionCfg.GetStringOption(domain.CapGPXMatching, "geosync", "0"), i18n.T("cliOptGeosync"))
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), i18n.T("cliOptRawExts"))
	workers := fs.Int("workers", sessionCfg.Global.Workers, i18n.T("cliOptWorkers"))
	enableGeocode := fs.Bool("geocode", true, i18n.T("cliOptEnableGeocode"))
	enableInterpolate := fs.Bool("interpolate", false, i18n.T("cliOptInterpolate"))
	interpolateWindow := fs.String("interpolate-window", sessionCfg.GetStringOption(domain.CapGPSInterpolate, "window", "15m"), i18n.T("cliOptInterpolateWindow"))
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, i18n.T("cliOptAllowNoGps"))
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, i18n.T("cliOptTestBackup"))
	isBackup := fs.Bool("backup", false, i18n.T("cliOptTestBackup"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, i18n.T("cliOptLogDir"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	win, _ := time.ParseDuration(*interpolateWindow)
	if win <= 0 {
		win = 15 * time.Minute
	}

	finalPolicy := *sidecarPolicy
	if *sidecarOnly {
		finalPolicy = string(domain.PolicySidecarOnly)
	}

	applySessionOverrides(sessionCfg, *baseDir, *sourceDir, *gpxDir, *processedDir, *logDir, *flatMode, finalPolicy, *allowNoGPS, *workers, *rawExts, *companionExts, map[domain.CapabilityID]map[string]any{
		domain.CapGPSInterpolate: {"window": *interpolateWindow},
		domain.CapGPXMatching:    {"geosync": *geosync},
		domain.CapDateArchive:    {"in_place": *inPlace},
	})

	task, err := pipeline.Build(pipeline.PipelineOptions{
		BaseDir:             *baseDir,
		SourceDir:           *sourceDir,
		GPXDir:              *gpxDir,
		ProcessedDir:        *processedDir,
		LogDir:              *logDir,
		FlatMode:            *flatMode,
		SidecarPolicy:       finalPolicy,
		SidecarOnly:         finalPolicy == string(domain.PolicySidecarOnly),
		CompanionExtensions: sessionCfg.Global.CompanionExtensions,
		InPlaceArchive:      *inPlace,
		Geosync:             *geosync,
		RawExtensions:       sessionCfg.Global.RawExtensions,
		Workers:             *workers,
		EnableGPXMatch:      true,
		EnableInterpolate:   *enableInterpolate,
		InterpolateWindow:   win,
		EnableGeocode:       *enableGeocode,
		AllowNoGPS:          *allowNoGPS,
		EnableArchive:       true,
		EnableBackup:        *isTest || *isBackup,
		BackupDir:           *backupDir,
		Session:             sessionCfg,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		exitApp(1)
	}

	runTaskWithEvents(task, *logDir)
}

func runGeocode(defaultBaseDir string) {
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, defaultBaseDir)

	fs := flag.NewFlagSet("geocode", flag.ExitOnError)
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, i18n.T("cliOptBaseDir"))
	dir := fs.String("dir", "", i18n.T("cliOptSourceDir"))
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, i18n.T("cliOptSourceDir"))
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, i18n.T("cliOptFlat"))
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, i18n.T("cliOptSidecarPolicy"))
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, i18n.T("cliOptSidecarOnly"))
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), i18n.T("cliOptCompanionExts"))
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), i18n.T("cliOptRawExts"))
	workers := fs.Int("workers", sessionCfg.Global.Workers, i18n.T("cliOptWorkers"))
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, i18n.T("cliOptAllowNoGps"))
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, i18n.T("cliOptTestBackup"))
	isBackup := fs.Bool("backup", false, i18n.T("cliOptTestBackup"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, i18n.T("cliOptLogDir"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	targetDir := *dir
	if targetDir == "" {
		targetDir = *sourceDir
	}

	finalPolicy := *sidecarPolicy
	if *sidecarOnly {
		finalPolicy = string(domain.PolicySidecarOnly)
	}

	sessionCfg.Global.BaseDir = *baseDir
	sessionCfg.Global.SourceDir = targetDir
	sessionCfg.Global.FlatMode = *flatMode
	sessionCfg.Global.SidecarPolicy = finalPolicy
	sessionCfg.Global.SidecarOnly = finalPolicy == string(domain.PolicySidecarOnly)
	sessionCfg.Global.CompanionExtensions = parseExtensions(*companionExts)
	sessionCfg.Global.AllowNoGPS = *allowNoGPS
	sessionCfg.Global.Workers = *workers
	sessionCfg.Global.RawExtensions = parseExtensions(*rawExts)
	if *logDir != "" {
		sessionCfg.Global.LogDir = *logDir
	}

	task, err := pipeline.Build(pipeline.PipelineOptions{
		BaseDir:             *baseDir,
		SourceDir:           targetDir,
		LogDir:              *logDir,
		FlatMode:            *flatMode,
		SidecarPolicy:       finalPolicy,
		SidecarOnly:         finalPolicy == string(domain.PolicySidecarOnly),
		CompanionExtensions: sessionCfg.Global.CompanionExtensions,
		EnableGeocode:       true,
		AllowNoGPS:          *allowNoGPS,
		RawExtensions:       sessionCfg.Global.RawExtensions,
		Workers:             *workers,
		EnableBackup:        *isTest || *isBackup,
		BackupDir:           *backupDir,
		Session:             sessionCfg,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		exitApp(1)
	}

	runTaskWithEvents(task, *logDir)
}

func runPipeline(defaultBaseDir string) {
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, defaultBaseDir)

	fs := flag.NewFlagSet("pipeline", flag.ExitOnError)
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, i18n.T("cliOptBaseDir"))
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, i18n.T("cliOptSourceDir"))
	gpxDir := fs.String("gpx-dir", sessionCfg.Global.GPXDir, i18n.T("cliOptGpxDir"))
	processedDir := fs.String("processed-dir", sessionCfg.Global.TargetDir, i18n.T("cliOptProcessedDir"))
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, i18n.T("cliOptFlat"))
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, i18n.T("cliOptSidecarPolicy"))
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, i18n.T("cliOptSidecarOnly"))
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), i18n.T("cliOptCompanionExts"))
	inPlace := fs.Bool("in-place", sessionCfg.GetBoolOption(domain.CapDateArchive, "in_place", false), i18n.T("cliOptInPlace"))
	geosync := fs.String("geosync", sessionCfg.GetStringOption(domain.CapGPXMatching, "geosync", "0"), i18n.T("cliOptGeosync"))
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), i18n.T("cliOptRawExts"))
	workers := fs.Int("workers", sessionCfg.Global.Workers, i18n.T("cliOptWorkers"))

	enableGPX := fs.Bool("gpx", true, i18n.T("cliOptEnableGpx"))
	enableInterpolate := fs.Bool("interpolate", false, i18n.T("cliOptInterpolate"))
	interpolateWindow := fs.String("interpolate-window", sessionCfg.GetStringOption(domain.CapGPSInterpolate, "window", "15m"), i18n.T("cliOptInterpolateWindow"))
	enableGeocode := fs.Bool("geocode", true, i18n.T("cliOptEnableGeocode"))
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, i18n.T("cliOptAllowNoGps"))
	enableArchive := fs.Bool("archive", true, i18n.T("cliOptEnableArchive"))
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, i18n.T("cliOptTestBackup"))
	isBackup := fs.Bool("backup", false, i18n.T("cliOptTestBackup"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, i18n.T("cliOptLogDir"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	win, _ := time.ParseDuration(*interpolateWindow)
	if win <= 0 {
		win = 15 * time.Minute
	}

	finalPolicy := *sidecarPolicy
	if *sidecarOnly {
		finalPolicy = string(domain.PolicySidecarOnly)
	}

	applySessionOverrides(sessionCfg, *baseDir, *sourceDir, *gpxDir, *processedDir, *logDir, *flatMode, finalPolicy, *allowNoGPS, *workers, *rawExts, *companionExts, map[domain.CapabilityID]map[string]any{
		domain.CapGPSInterpolate: {"window": *interpolateWindow},
		domain.CapGPXMatching:    {"geosync": *geosync},
		domain.CapDateArchive:    {"in_place": *inPlace},
	})

	task, err := pipeline.Build(pipeline.PipelineOptions{
		BaseDir:             *baseDir,
		SourceDir:           *sourceDir,
		GPXDir:              *gpxDir,
		ProcessedDir:        *processedDir,
		LogDir:              *logDir,
		FlatMode:            *flatMode,
		SidecarPolicy:       finalPolicy,
		SidecarOnly:         finalPolicy == string(domain.PolicySidecarOnly),
		CompanionExtensions: sessionCfg.Global.CompanionExtensions,
		InPlaceArchive:      *inPlace,
		Geosync:             *geosync,
		RawExtensions:       sessionCfg.Global.RawExtensions,
		Workers:             *workers,
		EnableGPXMatch:      *enableGPX,
		EnableInterpolate:   *enableInterpolate,
		InterpolateWindow:   win,
		EnableGeocode:       *enableGeocode,
		AllowNoGPS:          *allowNoGPS,
		EnableArchive:       *enableArchive,
		EnableBackup:        *isTest || *isBackup,
		BackupDir:           *backupDir,
		Session:             sessionCfg,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化流水线失败: %v\n", err)
		exitApp(1)
	}

	runTaskWithEvents(task, *logDir)
}

func runTaskWithEvents(task domain.Task, customLogDir ...string) {
	eventCh := make(chan domain.ProgressEvent, 100)
	go func() {
		for evt := range eventCh {
			if evt.Level == domain.LevelError {
				fmt.Fprintf(os.Stderr, "[%s] %s\n", evt.Stage, evt.Message)
			} else {
				fmt.Printf("[%s] %s\n", evt.Stage, evt.Message)
			}
		}
	}()

	summary, issues, err := task.Execute(context.Background(), eventCh)
	close(eventCh)

	if err != nil {
		fmt.Fprintf(os.Stderr, "执行任务出错: %v\n", err)
		exitApp(1)
	}

	targetLogDir := common.GetDefaultLogDir()
	if len(customLogDir) > 0 && customLogDir[0] != "" {
		targetLogDir = customLogDir[0]
	}
	if len(targetLogDir) >= 2 && targetLogDir[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			targetLogDir = filepath.Join(home, targetLogDir[2:])
		}
	}
	targetLogDir, _ = filepath.Abs(targetLogDir)

	fmt.Printf("📄 实时中文执行日志流已完整保存在: %s\n", filepath.Join(targetLogDir, common.LogFileNameLatest))

	if summary != nil && summary.Failed > 0 {
		fmt.Fprintf(os.Stderr, "完成时发现 %d 项失败，详情见报告文件 %s\n", len(issues), filepath.Join(targetLogDir, common.PendingReportFileNameLatest))
		exitApp(1)
	}
}

func runOrganizeByDate() {
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, "")

	fs := flag.NewFlagSet("organize-by-date", flag.ExitOnError)
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, i18n.T("cliOptBaseDir"))
	sourceDir := fs.String("source-dir", "", i18n.T("cliOptSourceDir"))
	targetDir := fs.String("target-dir", "", i18n.T("cliOptProcessedDir"))
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), i18n.T("cliOptRawExts"))
	workers := fs.Int("workers", sessionCfg.Global.Workers, i18n.T("cliOptWorkers"))
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, i18n.T("cliOptTestBackup"))
	isBackup := fs.Bool("backup", false, i18n.T("cliOptTestBackup"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, i18n.T("cliOptLogDir"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	if *sourceDir == "" || *targetDir == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -source-dir 和 -target-dir")
		fmt.Fprintln(os.Stderr, "用法: photools organize-by-date -source-dir <目录> -target-dir <目录> [-test]")
		exitApp(1)
	}

	sessionCfg.Global.BaseDir = *baseDir
	sessionCfg.Global.SourceDir = *sourceDir
	sessionCfg.Global.TargetDir = *targetDir
	sessionCfg.Global.Workers = *workers
	sessionCfg.Global.RawExtensions = parseExtensions(*rawExts)
	if *logDir != "" {
		sessionCfg.Global.LogDir = *logDir
	}

	task, err := pipeline.Build(pipeline.PipelineOptions{
		BaseDir:       *baseDir,
		SourceDir:     *sourceDir,
		ProcessedDir:  *targetDir,
		LogDir:        *logDir,
		RawExtensions: sessionCfg.Global.RawExtensions,
		Workers:       *workers,
		EnableArchive: true,
		EnableBackup:  *isTest || *isBackup,
		BackupDir:     *backupDir,
		Session:       sessionCfg,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		exitApp(1)
	}

	runTaskWithEvents(task, *logDir)
}

func runRestoreTest(defaultBaseDir string) {
	fs := flag.NewFlagSet("restore-test", flag.ExitOnError)
	baseDir := fs.String("base-dir", defaultBaseDir, i18n.T("cliOptBaseDir"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))
	targetDir := fs.String("target-dir", "", i18n.T("cliOptSourceDir"))
	cleanProcessed := fs.Bool("clean", false, i18n.T("cliOptCleanProcessed"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	bDir := *backupDir
	if bDir == "" {
		bDir = filepath.Join(*baseDir, "Inbox_bak")
	}
	tDir := *targetDir
	if tDir == "" {
		tDir = filepath.Join(*baseDir, "Inbox")
	}

	fmt.Printf("🔄 正在从备份目录 [%s] 还原原始测试照片至 [%s] ...\n", bDir, tDir)
	count, err := engine.RestoreBackup(bDir, tDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 还原失败: %v\n", err)
		exitApp(1)
	}

	if *cleanProcessed {
		pDir := filepath.Join(*baseDir, "Processed")
		fmt.Printf("🧹 正在清理归档目录 [%s] ...\n", pDir)
		_ = os.RemoveAll(pDir)
		_ = os.MkdirAll(pDir, 0o755)
		fmt.Println("✅ 已清理 Processed 归档目录")
	}

	fmt.Printf("🎉 成功还原 %d 个原始照片文件！现在可重新进行流水线测试。\n", count)
}

func runBackup(defaultBaseDir string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	baseDir := fs.String("base-dir", defaultBaseDir, i18n.T("cliOptBaseDir"))
	sourceDir := fs.String("source-dir", "", i18n.T("cliOptSourceDir"))
	backupDir := fs.String("backup-dir", "", i18n.T("cliOptBackupDir"))

	_ = fs.Parse(normalizeBoolFlags(fs, os.Args[2:]))

	sDir := *sourceDir
	if sDir == "" {
		sDir = filepath.Join(*baseDir, "Inbox")
	}
	bDir := *backupDir
	if bDir == "" {
		bDir = filepath.Join(*baseDir, "Inbox_bak")
	}

	fmt.Printf("📸 正在创建待处理照片快照备份 [%s -> %s] ...\n", sDir, bDir)
	count, err := engine.CreateBackup(sDir, bDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 备份失败: %v\n", err)
		exitApp(1)
	}

	fmt.Printf("🎉 快照备份成功！共备份 %d 个照片与伴随文件到 [%s]。\n", count, bDir)
}

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: photools completion [bash|zsh|fish|install]")
		return
	}

	switch args[0] {
	case "bash":
		completion.GenerateBash(os.Stdout)
	case "zsh":
		completion.GenerateZsh(os.Stdout)
	case "fish":
		completion.GenerateFish(os.Stdout)
	case "install":
		path, err := completion.InstallShellCompletion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "安装自动补全脚本失败: %v\n", err)
			exitApp(1)
		}
		fmt.Printf("✅ 自动补全脚本已成功安装至: %s\n", path)
		fmt.Println("💡 请在当前终端运行以下命令或重启终端生效:")
		if strings.Contains(path, "zsh") {
			fmt.Println("   source ~/.zshrc")
		} else if strings.Contains(path, "bash") {
			fmt.Println("   source ~/.bashrc")
		}
	default:
		fmt.Fprintf(os.Stderr, "未知的 Shell 类型: %s (支持 bash, zsh, fish, install)\n", args[0])
		exitApp(1)
	}
}

func parseExtensions(s string) []string {
	// 支持逗号与空格切分
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	var res []string
	for _, p := range fields {
		c := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(p)), ".")
		if c != "" {
			res = append(res, c)
		}
	}
	return res
}

func applySessionOverrides(
	sessionCfg *config.SessionConfig,
	baseDir, srcDir, gpxDir, procDir, logDir string,
	flat bool,
	sidecarPolicy string,
	allowNoGPS bool,
	workers int,
	rawExts string,
	companionExts string,
	pluginOpts map[domain.CapabilityID]map[string]interface{},
) {
	sessionCfg.Global.BaseDir = baseDir
	sessionCfg.Global.SourceDir = srcDir
	if gpxDir != "" {
		sessionCfg.Global.GPXDir = gpxDir
	}
	sessionCfg.Global.TargetDir = procDir
	if logDir != "" {
		sessionCfg.Global.LogDir = logDir
	}
	sessionCfg.Global.FlatMode = flat
	sessionCfg.Global.SidecarPolicy = sidecarPolicy
	sessionCfg.Global.SidecarOnly = sidecarPolicy == string(domain.PolicySidecarOnly)
	sessionCfg.Global.AllowNoGPS = allowNoGPS
	sessionCfg.Global.Workers = workers
	sessionCfg.Global.RawExtensions = parseExtensions(rawExts)
	if companionExts != "" {
		sessionCfg.Global.CompanionExtensions = parseExtensions(companionExts)
	}
	for pID, opts := range pluginOpts {
		for k, v := range opts {
			sessionCfg.SetPluginOption(pID, k, v)
		}
	}
}

func parseCoordinatesWithDebug(args []string) (float64, float64, float64, bool, error) {
	if len(args) < 2 {
		return 0, 0, 0, false, fmt.Errorf("参数数量不足")
	}

	var cleanArgs []string
	debug := false
	for _, a := range args {
		if a == "--debug" || a == "-d" {
			debug = true
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}

	if len(cleanArgs) < 2 {
		return 0, 0, 0, false, fmt.Errorf("缺少经纬度")
	}

	lat, err1 := strconv.ParseFloat(cleanArgs[0], 64)
	lon, err2 := strconv.ParseFloat(cleanArgs[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, 0, false, fmt.Errorf("经纬度格式错误")
	}

	alt := 0.0
	if len(cleanArgs) >= 3 {
		if a, err := strconv.ParseFloat(cleanArgs[2], 64); err == nil {
			alt = a
		}
	}

	return lat, lon, alt, debug, nil
}

func runGeoData(args []string) {
	mgr, err := geodata.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据包管理器失败: %v\n", err)
		exitApp(1)
	}

	if len(args) == 0 {
		printGeoDataHelp()
		return
	}

	switch args[0] {
	case "list", "ls":
		list := mgr.ListContinents()
		fmt.Println("🗺️  【全球各洲离线逆地理编码数据包列表】")
		for _, item := range list {
			status := "❌ 未安装"
			if item.Installed {
				status = fmt.Sprintf("✅ 已安装 (%d 点位, %.1f MB)", item.Points, float64(item.FileSize)/(1024*1024))
			}
			fmt.Printf("  • %-15s [%s]: %s\n    └─ %s\n", item.Meta.Code, item.Meta.NameZH, status, item.Meta.Description)
		}
	case "install", "get", "download":
		if len(args) < 2 {
			fmt.Println("❌ 请指定要安装的数据包标识 (如 china, asia, europe, north-america, oceania, all)")
			exitApp(1)
		}
		logFn := func(msg string) {
			fmt.Println(msg)
		}
		if err := mgr.Install(context.Background(), args[1], logFn); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 安装失败: %v\n", err)
			exitApp(1)
		}
		geocoding.ResetDefault()
	case "remove", "rm", "uninstall":
		if len(args) < 2 {
			fmt.Println("❌ 请指定要移除的数据包标识 (如 china, asia 等)")
			exitApp(1)
		}
		if err := mgr.Remove(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 移除失败: %v\n", err)
			exitApp(1)
		}
		geocoding.ResetDefault()
		fmt.Printf("✅ 已成功移除数据包 [%s]\n", args[1])
	case "info", "status":
		rg := geocoding.GetDefault()
		fmt.Println("🔍 正在扫描并装载本地离线地理库索引...")
		_ = rg.InitProgressive(context.Background(), func(stage string, percent float64, msg string, status geocoding.HealthStatus, err error) {
			if percent >= 0 {
				fmt.Printf("  • [%s] %s (%.0f%%)\n", stage, msg, percent*100)
			} else {
				fmt.Printf("  • [%s] %s\n", stage, msg)
			}
		})
		stats := rg.GetStats()
		fmt.Printf("\n📊 【当前离线地理空间索引库统计】\n")
		fmt.Printf("  • 离线库总点数:   %d 个 (内置: %d, 用户自定义: %d)\n",
			stats.TotalPoints, stats.BuiltinPoints, stats.CustomPoints)
		fmt.Printf("  • 已载入大洲包:   %d 个\n", len(stats.Packs))
		for _, pk := range stats.Packs {
			fmt.Printf("    - %-20s: %6d 点位 (%.1f MB, 加载 %.2fms)\n",
				pk.Name, pk.Points, float64(pk.SizeBytes)/(1024*1024), float64(pk.LoadTime.Microseconds())/1000.0)
		}
		fmt.Printf("  • KD-Tree建树耗时: %.2f ms\n", float64(stats.TreeBuildTime.Microseconds())/1000.0)
		fmt.Printf("  • 全量初始化耗时:  %.2f ms\n", float64(stats.TotalInitTime.Microseconds())/1000.0)
	case "test":
		if len(args) < 3 {
			fmt.Println("❌ 用法: photools geodata test <纬度> <经度> [海拔高度米] [--debug]")
			exitApp(1)
		}
		lat, lon, alt, debug, err := parseCoordinatesWithDebug(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 经纬度解析失败: %v\n", err)
			exitApp(1)
		}
		runGeoDataTestCoordinates(lat, lon, alt, debug)
	default:
		fmt.Fprintf(os.Stderr, "未知的 geodata 子命令: %s\n", args[0])
		printGeoDataHelp()
		exitApp(1)
	}
}

func printGeoDataHelp() {
	fmt.Println("用法: photools geodata <子命令> [参数]")
	fmt.Println("子命令:")
	fmt.Println("  list                    列出所有支持的大洲数据包及本地安装状态")
	fmt.Println("  install <大洲|all>       下载并安装指定大洲或全部离线数据包")
	fmt.Println("  remove <大洲>            卸载并移除指定大洲离线数据包")
	fmt.Println("  info                    查看当前已载入的地理空间索引状态")
	fmt.Println("  test <纬度> <经度>       测试经纬度逆地理编码匹配")
}

func runGeoDataTestCoordinates(lat, lon, alt float64, debug bool) {
	rg := geocoding.GetDefault()
	if !rg.IsInitialized() {
		fmt.Println("⚙️  检测到首次冷启动，正在装载离线高精地理空间索引...")
		_ = rg.InitProgressive(context.Background(), func(stage string, percent float64, msg string, status geocoding.HealthStatus, err error) {
			if percent >= 0 {
				fmt.Printf("  • [%s] %s (%.0f%%)\n", stage, msg, percent*100)
			} else {
				fmt.Printf("  • [%s] %s\n", stage, msg)
			}
		})
		fmt.Println()
	}

	qStart := time.Now()
	loc, bestPt, distKm, debugStats, loadStats := rg.LookupDetailedWithDebug(lat, lon, 5)
	queryDur := time.Since(qStart)

	if !debug {
		if loc == nil {
			fmt.Println("❌ 未在离线地理库中匹配到有效地点")
			return
		}
		fmt.Printf("🔍 正在检索经纬度坐标: (%.6f, %.6f)", lat, lon)
		if alt != 0 {
			fmt.Printf(" [海拔: %.1fm]", alt)
		}
		fmt.Println("\n\n📍 【逆地理编码匹配结果】")
		fmt.Printf("  • 国家:       %s (%s)\n", loc.Country, loc.CountryCode)
		if loc.Province != "" {
			fmt.Printf("  • 省份/州:    %s\n", loc.Province)
		}
		if loc.City != "" {
			fmt.Printf("  • 城市/地区:  %s\n", loc.City)
		}
		if loc.District != "" {
			fmt.Printf("  • 区县/POI:   %s\n", loc.District)
		}
		if loc.Timezone != "" {
			fmt.Printf("  • IANA时区:   %s\n", loc.Timezone)
		}
		if loc.Elevation != 0 {
			fmt.Printf("  • 地形海拔:   %d 米\n", loc.Elevation)
		}
		fmt.Printf("  • 物理距离:   %.2f km\n", distKm)
		fmt.Printf("  • 数据源:     %s\n", loc.Source)
		fmt.Printf("  • 规范全称:   %s\n", loc.FormatSummary())
		return
	}

	report := geocoding.FormatDebugReport(lat, lon, alt, loc, bestPt, distKm, debugStats, loadStats, queryDur)
	fmt.Print(report)
}

// normalizeBoolFlags 预处理命令行参数，自动将 "-flag true" 或 "--flag false" 合并为 "-flag=true"，
// 彻底解决 Go 标准库 flag 针对布尔参数后跟随空格值会提前截断后续所有参数的陷阱
func normalizeBoolFlags(fs *flag.FlagSet, args []string) []string {
	var normalized []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
			name := strings.TrimLeft(arg, "-")
			if f := fs.Lookup(name); f != nil {
				if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
					if i+1 < len(args) {
						next := strings.ToLower(args[i+1])
						if next == "true" || next == "false" || next == "1" || next == "0" || next == "t" || next == "f" {
							normalized = append(normalized, fmt.Sprintf("-%s=%s", name, args[i+1]))
							i++
							continue
						}
					}
				}
			}
		}
		normalized = append(normalized, arg)
	}
	return normalized
}

func runInspect(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: photools inspect <照片路径>")
		exitApp(1)
	}
	filePath := args[len(args)-1]
	meta, err := exiftool.InspectPhotoMetadata(exiftool.DefaultRunner(), filePath)
	defer exiftool.CloseDefaultPool()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取照片元数据失败: %v\n", err)
		exitApp(1)
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	fmt.Println(string(data))
}
