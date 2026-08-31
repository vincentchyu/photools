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

	// 启动时自动同步并升级 ~/.config/photools/plugins.json 配置
	_, _ = config.LoadPluginsConfig("")

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
	fmt.Printf("📷 photools %s - 摄影师专业 GPS 轨迹匹配、地理逆编码与照片结构化归档工具箱\n", Version)
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  photools                                在交互终端中直接启动可视化 TUI 插件工作台")
	fmt.Println("  photools tui                            显式启动 TUI 工作台")
	fmt.Println("  photools geotag [选项]                  根据 GPX 轨迹为照片批量修正写入 GPS 并归档")
	fmt.Println("  photools geocode [选项]                 [独立能力] 为已有 GPS 坐标的照片批量写入离线中文地名元数据")
	fmt.Println("  photools pipeline [选项]                [复合流水线] 自由勾选组合 [GPX修正/逆地理地名/拍摄日归档]")
	fmt.Println("  photools organize-by-date [选项]         [独立能力] 根据照片拍摄日期整理并规范化重命名文件")
	fmt.Println("  photools restore-test [选项]            [测试辅助] 从 Inbox_bak 备份目录一键还原原始照片至 Inbox")
	fmt.Println("  photools geodata [操作]                 管理各大洲精细离线逆地理编码数据包 (list/install/remove/info/test)")
	fmt.Println("  photools completion [shell]             生成或安装 Shell 自动补全脚本 (zsh/bash/fish/install)")
	fmt.Println("  photools version                        显示当前版本号 (或 -v, --version)")
	fmt.Println()
	fmt.Println("常用选项 (适用 geotag/geocode/pipeline):")
	fmt.Println("  -flat                                   [扁平原地模式] 忽略 Inbox/Processed 分层，直接扫描指定目录并原地保存/处理")
	fmt.Println("  -in-place                               [原地重命名] 规范化重命名但不创建 Processed/YYYY/MMDD/ 子目录")
	fmt.Println("  -interpolate                            [智能推断] 启用 GPS 邻近前后照片时间权重插值与推算")
	fmt.Println("  -interpolate-window <时长>              智能推算最大时间窗口 (默认 15m，如 30m, 1h)")
	fmt.Println("  -allow-no-gps                           [容错降级] 无 GPS 照片允许跳过地名写入，安全直接进行日期归档")
	fmt.Println("  -test, -backup                          [测试备份模式] 在处理前自动将源目录待处理照片全量快照备份至 Inbox_bak")
	fmt.Println("  -backup-dir <目录>                      自定义测试快照备份的存放目录")
	fmt.Println("  -base-dir <目录>                        工作根目录 (默认当前目录或 ~/Pictures/GPS)")
	fmt.Println()
	fmt.Println("geodata 常用命令:")
	fmt.Println("  photools geodata list                   列出所有支持的大洲数据包及本地安装状态")
	fmt.Println("  photools geodata install <大洲>         下载并安装指定大洲数据包 (如 china, asia, europe, north-america, oceania, all)")
	fmt.Println("  photools geodata remove <大洲>          卸载并移除指定大洲数据包")
	fmt.Println("  photools geodata info                   查看当前全局已加载的地理点位统计")
	fmt.Println("  photools geodata test <纬度> <经度> [海拔] [--debug] 测试并验证指定经纬度的离线逆地理编码匹配结果")
}

func runGeotag(defaultBaseDir string) {
	pluginsCfg, _ := config.LoadPluginsConfig("")
	sessionCfg := config.NewSessionConfig(pluginsCfg, defaultBaseDir)

	fs := flag.NewFlagSet("geotag", flag.ExitOnError)
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, "基础目录，包含 Inbox/Processed/Logs")
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, "待处理照片源目录 (默认 <base-dir>/Inbox)")
	gpxDir := fs.String("gpx-dir", sessionCfg.Global.GPXDir, "GPX 轨迹目录 (默认 ~/.config/gpx)")
	processedDir := fs.String("processed-dir", sessionCfg.Global.TargetDir, "归档目标根目录 (默认 <base-dir>/Processed)")
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, "扁平原地模式 (直接扫描并就地处理/保存)")
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, "侧车写入策略: read_only(默认/推荐: 只读RAW写XMP，JPG写内嵌), sidecar_only(纯XMP侧车), embed_and_sidecar(双写同步), embed_only(纯原图内嵌)")
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, "仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)")
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), "伴随文件扩展名白名单 (如 wav, acr, exf，逗号或空格分隔)")
	inPlace := fs.Bool("in-place", sessionCfg.GetBoolOption(domain.CapDateArchive, "in_place", false), "原地重命名归档，不建立 YYYY/MMDD 子目录")
	geosync := fs.String("geosync", sessionCfg.GetStringOption(domain.CapGPXMatching, "geosync", "0"), "传递给 exiftool 的 geosync 偏移值")
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", sessionCfg.Global.Workers, "并发处理的资产组数量")
	enableGeocode := fs.Bool("geocode", true, "是否同时写入逆地理地名元数据")
	enableInterpolate := fs.Bool("interpolate", false, "启用能力 1.5: 根据前后照片时间推算补全 GPS")
	interpolateWindow := fs.String("interpolate-window", sessionCfg.GetStringOption(domain.CapGPSInterpolate, "window", "15m"), "智能推算最大时间窗口 (如 15m, 30m, 1h)")
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, "无 GPS 坐标时允许跳过地名写入直接归档 (软降级)")
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, "开启测试备份模式 (处理前自动备份 Inbox 到 Inbox_bak)")
	isBackup := fs.Bool("backup", false, "同 -test，处理前备份原始文件")
	backupDir := fs.String("backup-dir", "", "自定义测试备份目录")
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, "日志与待补报告目录 (默认 ~/.logs/photools)")

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
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, "基础工作目录")
	dir := fs.String("dir", "", "待处理照片目录（默认 <base-dir>/Inbox）")
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, "同 -dir")
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, "扁平模式 (在当前目录下直接处理)")
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, "侧车写入策略: smart(默认/推荐: 智能分层模式), sidecar_only(纯XMP侧车), embed_and_sidecar(双写同步), embed_only(纯原图内嵌)")
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, "仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)")
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), "伴随文件扩展名白名单 (如 wav, acr, exf)")
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", sessionCfg.Global.Workers, "并发处理数量")
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, "无 GPS 坐标时允许跳过地名写入直接归档")
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, "开启测试备份模式 (处理前自动备份到 Inbox_bak)")
	isBackup := fs.Bool("backup", false, "同 -test，处理前备份原始文件")
	backupDir := fs.String("backup-dir", "", "自定义测试备份目录")
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, "日志与待补报告目录 (默认 ~/.logs/photools)")

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
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, "基础工作目录")
	sourceDir := fs.String("source-dir", sessionCfg.Global.SourceDir, "待处理源目录（默认 <base-dir>/Inbox）")
	gpxDir := fs.String("gpx-dir", sessionCfg.Global.GPXDir, "GPX 轨迹目录（默认 ~/.config/gpx）")
	processedDir := fs.String("processed-dir", sessionCfg.Global.TargetDir, "归档目标根目录（默认 <base-dir>/Processed）")
	flatMode := fs.Bool("flat", sessionCfg.Global.FlatMode, "扁平原地模式 (忽略 Inbox/Processed 分层，直接扫描并就地处理/保存)")
	sidecarPolicy := fs.String("sidecar-policy", sessionCfg.Global.SidecarPolicy, "侧车写入策略: smart(默认/推荐: 智能分层模式), sidecar_only(纯XMP侧车), embed_and_sidecar(双写同步), embed_only(纯原图内嵌)")
	sidecarOnly := fs.Bool("sidecar-only", sessionCfg.Global.SidecarOnly, "仅生成/修改 {file}.xmp 侧车文件 (等价于 -sidecar-policy=sidecar_only)")
	companionExts := fs.String("companion-exts", strings.Join(sessionCfg.Global.CompanionExtensions, ","), "伴随文件扩展名白名单 (如 wav, acr, exf，逗号或空格分隔)")
	inPlace := fs.Bool("in-place", sessionCfg.GetBoolOption(domain.CapDateArchive, "in_place", false), "原地规范重命名，不建立 YYYY/MMDD 子目录")
	geosync := fs.String("geosync", sessionCfg.GetStringOption(domain.CapGPXMatching, "geosync", "0"), "传递给 exiftool 的 geosync 偏移值")
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", sessionCfg.Global.Workers, "并发处理的资产组数量")

	enableGPX := fs.Bool("gpx", true, "启用能力 1: GPX 轨迹匹配与 GPS 修正")
	enableInterpolate := fs.Bool("interpolate", false, "启用能力 1.5: 根据前后照片时间推算补全 GPS")
	interpolateWindow := fs.String("interpolate-window", sessionCfg.GetStringOption(domain.CapGPSInterpolate, "window", "15m"), "智能推算最大时间窗口 (如 15m, 30m, 1h)")
	enableGeocode := fs.Bool("geocode", true, "启用能力 2: 逆地理编码写入元数据")
	allowNoGPS := fs.Bool("allow-no-gps", sessionCfg.Global.AllowNoGPS, "无 GPS 坐标时允许跳过地名写入直接归档 (软降级)")
	enableArchive := fs.Bool("archive", true, "启用能力 3: 拍摄日期归档与规范重命名")
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, "开启测试备份模式 (处理前自动备份 Inbox 到 Inbox_bak)")
	isBackup := fs.Bool("backup", false, "同 -test，处理前备份原始文件")
	backupDir := fs.String("backup-dir", "", "自定义测试备份目录")
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, "日志与待补报告目录 (默认 ~/.logs/photools)")

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
	baseDir := fs.String("base-dir", sessionCfg.Global.BaseDir, "基础工作目录 (可选)")
	sourceDir := fs.String("source-dir", "", "需要整理的源目录")
	targetDir := fs.String("target-dir", "", "归档目标根目录")
	rawExts := fs.String("raw-exts", strings.Join(sessionCfg.Global.RawExtensions, ","), "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", sessionCfg.Global.Workers, "并发处理数量")
	isTest := fs.Bool("test", sessionCfg.Global.TestBackup, "开启测试备份模式 (处理前自动备份)")
	isBackup := fs.Bool("backup", false, "同 -test，处理前备份原始文件")
	backupDir := fs.String("backup-dir", "", "自定义测试备份目录")
	logDir := fs.String("log-dir", sessionCfg.Global.LogDir, "日志与待补报告目录 (默认 ~/.logs/photools)")

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
	baseDir := fs.String("base-dir", defaultBaseDir, "工作根目录 (包含 Inbox/Inbox_bak/Processed)")
	backupDir := fs.String("backup-dir", "", "备份源目录 (默认 <base-dir>/Inbox_bak)")
	targetDir := fs.String("target-dir", "", "还原目标目录 (默认 <base-dir>/Inbox)")
	cleanProcessed := fs.Bool("clean", false, "是否同时清理 Processed 目录下的测试归档文件")

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
	baseDir := fs.String("base-dir", defaultBaseDir, "工作根目录 (包含 Inbox/Inbox_bak)")
	sourceDir := fs.String("source-dir", "", "待备份源目录 (默认 <base-dir>/Inbox)")
	backupDir := fs.String("backup-dir", "", "快照目标目录 (默认 <base-dir>/Inbox_bak)")

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
