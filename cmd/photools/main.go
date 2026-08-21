package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"

	"github.com/vincentchyu/photo-processing/internal/domain"
	"github.com/vincentchyu/photo-processing/internal/tasks/geotag"
	"github.com/vincentchyu/photo-processing/internal/tasks/organize"
	"github.com/vincentchyu/photo-processing/internal/tui"
)

func main() {
	defaultBaseDir, _ := defaultBaseDir()

	// 如果没有传参数，或者显式传入 tui
	if len(os.Args) < 2 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			if err := tui.Run(defaultBaseDir); err != nil {
				fmt.Fprintf(os.Stderr, "运行 TUI 失败: %v\n", err)
				os.Exit(1)
			}
			return
		}
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "tui":
		if err := tui.Run(defaultBaseDir); err != nil {
			fmt.Fprintf(os.Stderr, "运行 TUI 失败: %v\n", err)
			os.Exit(1)
		}
	case "geotag":
		runGeotag(defaultBaseDir)
	case "organize-by-date":
		runOrganizeByDate()
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
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
	fmt.Println("📷 PhotoTools - 摄影工作流自动化处理工具集")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  photools                        在交互终端中直接启动可视化 TUI 工作台")
	fmt.Println("  photools tui                    显式启动 TUI 工作台")
	fmt.Println("  photools geotag [选项]          根据 GPX 轨迹为照片批量修正写入 GPS 并归档")
	fmt.Println("  photools organize-by-date [选项] 根据照片拍摄日期整理并规范化重命名文件")
	fmt.Println()
	fmt.Println("geotag 常用选项:")
	fmt.Println("  -base-dir  基础工作目录（默认: $HOME/Pictures/GPS，包含 Inbox/GPX/Processed/Logs）")
	fmt.Println("  -geosync   时间偏移（如 +00:00:05 或 -00:01:00，默认: 0）")
	fmt.Println("  -raw-exts  可识别的 RAW 扩展名（默认: nef,cr3,arw,dng,raf,rw2,orf）")
	fmt.Println("  -workers   并发处理的资产组数量（默认: CPU 核心数）")
	fmt.Println()
	fmt.Println("organize-by-date 常用选项:")
	fmt.Println("  -source-dir 需要整理的源目录")
	fmt.Println("  -target-dir 归档目标根目录")
	fmt.Println("  -raw-exts   可识别的 RAW 扩展名")
}

func runGeotag(defaultBaseDir string) {
	fs := flag.NewFlagSet("geotag", flag.ExitOnError)
	baseDir := fs.String("base-dir", defaultBaseDir, "基础目录，包含 Inbox/GPX/Processed/Logs")
	geosync := fs.String("geosync", "0", "传递给 exiftool 的 geosync 偏移值")
	rawExts := fs.String("raw-exts", "nef,cr3,arw,dng,raf,rw2,orf", "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", runtime.NumCPU(), "并发处理的资产组数量")

	_ = fs.Parse(os.Args[2:])

	task, err := geotag.NewTask(geotag.Config{
		BaseDir:       *baseDir,
		Geosync:       *geosync,
		RawExtensions: parseExtensions(*rawExts),
		Workers:       *workers,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

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
		os.Exit(1)
	}

	if summary != nil && summary.Failed > 0 {
		fmt.Fprintf(os.Stderr, "完成时发现 %d 项失败，详情见 Logs/inbox_pending_report_latest.md\n", len(issues))
		os.Exit(1)
	}
}

func runOrganizeByDate() {
	fs := flag.NewFlagSet("organize-by-date", flag.ExitOnError)
	sourceDir := fs.String("source-dir", "", "需要整理的源目录")
	targetDir := fs.String("target-dir", "", "归档目标根目录")
	rawExts := fs.String("raw-exts", "nef,cr3,arw,dng,raf,rw2,orf", "可识别的 RAW 扩展名，逗号分隔")

	_ = fs.Parse(os.Args[2:])

	if *sourceDir == "" || *targetDir == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -source-dir 和 -target-dir")
		fmt.Fprintln(os.Stderr, "用法: photools organize-by-date -source-dir <目录> -target-dir <目录>")
		os.Exit(1)
	}

	task, err := organize.NewTask(organize.Config{
		SourceDir:     *sourceDir,
		TargetDir:     *targetDir,
		RawExtensions: parseExtensions(*rawExts),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

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

	_, _, err = task.Execute(context.Background(), eventCh)
	close(eventCh)

	if err != nil {
		fmt.Fprintf(os.Stderr, "整理失败: %v\n", err)
		os.Exit(1)
	}
}

func parseExtensions(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(part, ".")))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; !ok {
			seen[clean] = struct{}{}
			out = append(out, clean)
		}
	}
	return out
}
