package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vincentchyu/photo-processing/internal/exiftool"
	"github.com/vincentchyu/photo-processing/internal/geotag"
	"github.com/vincentchyu/photo-processing/internal/organizer"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "geotag":
		runGeotag()
	case "organize-by-date":
		runOrganizeByDate()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("用法: photools <命令> [选项]")
	fmt.Println("可用命令:")
	fmt.Println("  geotag            根据 GPX 轨迹为照片同步 GPS 信息")
	fmt.Println("  organize-by-date  根据照片拍摄日期整理并规范化重命名文件")
}

func runGeotag() {
	fs := flag.NewFlagSet("geotag", flag.ExitOnError)
	defaultBaseDir, _ := geotag.DefaultBaseDir()

	baseDir := fs.String("base-dir", defaultBaseDir, "基础目录，包含 Inbox/GPX/Processed/Logs")
	geosync := fs.String("geosync", "0", "传递给 exiftool 的 geosync 偏移值")
	rawExts := fs.String("raw-exts", "nef,cr3,arw,dng,raf,rw2,orf", "可识别的 RAW 扩展名，逗号分隔")
	workers := fs.Int("workers", runtime.NumCPU(), "并发处理的资产组数量")

	_ = fs.Parse(os.Args[2:])

	cfg := geotag.Config{
		BaseDir:       *baseDir,
		Geosync:       *geosync,
		RawExtensions: geotag.ParseExtensions(*rawExts),
		Workers:       *workers,
	}

	app, err := geotag.NewApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
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
		fmt.Fprintln(os.Stderr, "用法: photools organize-by-date -source-dir <目录> -target-dir <目录>")
		os.Exit(1)
	}

	fmt.Printf("正在从 %s 整理到 %s...\n", *sourceDir, *targetDir)

	err := runOrganizerLogic(*sourceDir, *targetDir, geotag.ParseExtensions(*rawExts))
	if err != nil {
		fmt.Fprintf(os.Stderr, "整理失败: %v\n", err)
		os.Exit(1)
	}
}

func runOrganizerLogic(sourceDir, targetDir string, rawExts []string) error {
	runner := exiftool.ExecRunner{}

	// 阶段一：扫描目录并将文件按照目录和 BaseName 进行分组
	assets, err := organizer.DiscoverMediaGroups(sourceDir, rawExts)
	if err != nil {
		return err
	}

	// 阶段二：处理每一组照片
	for _, asset := range assets {
		var meta exiftool.Metadata
		allFiles := asset.AllFiles()

		// 1. 在当前组中寻找可以读取拍摄日期的锚点文件 (优先 RAW 和 JPG)
		if asset.RawPath != "" {
			meta, _ = exiftool.ReadMetadata(runner, asset.RawPath)
		}
		if meta.DateTimeOriginal == "" && asset.JPGPath != "" {
			meta, _ = exiftool.ReadMetadata(runner, asset.JPGPath)
		}

		// 2. 如果 RAW/JPG 没有日期，尝试用同组的其他文件读取
		if meta.DateTimeOriginal == "" {
			for _, path := range asset.CompanionPaths {
				if m, err := exiftool.ReadMetadata(runner, path); err == nil && m.DateTimeOriginal != "" {
					meta = m
					break
				}
			}
		}

		if meta.DateTimeOriginal == "" {
			fmt.Printf("跳过组: %s,%s (无法提取拍摄日期)\n", asset.BaseName, strings.Join(asset.AllFiles(), ","))
			continue
		}

		// 3. 构建目标路径
		archiveDir, err := organizer.BuildArchiveDir(targetDir, meta.DateTimeOriginal)
		if err != nil {
			fmt.Printf("跳过组: %s (构建归档目录失败: %v)\n", asset.BaseName, err)
			continue
		}

		_ = os.MkdirAll(archiveDir, 0o755)
		newBase := organizer.CalculateNormalizedName(asset.BaseName, meta.DateTimeOriginal)

		// 4. 冲突检查：检查组内的所有文件，如果目标路径有冲突，则跳过整组
		conflict := false
		for _, path := range allFiles {
			ext := strings.ToLower(filepath.Ext(path))
			targetPath := filepath.Join(archiveDir, newBase+ext)
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("目标文件已存在，跳过该文件及同组: %s\n", targetPath)
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}

		// 5. 将同组所有文件一起移动/重命名
		fmt.Printf("移动组: %s -> %s/\n", asset.BaseName, filepath.Base(archiveDir))
		if err := organizer.MoveFilesWithRename(allFiles, archiveDir, newBase); err != nil {
			fmt.Printf("移动组 %s 失败: %v\n", asset.BaseName, err)
		}
	}

	return nil
}
