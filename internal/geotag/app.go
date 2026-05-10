package geotag

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vincentchyu/photo-processing/internal/exiftool"
	"github.com/vincentchyu/photo-processing/internal/organizer"
)

type Config struct {
	BaseDir       string
	Geosync       string
	RawExtensions []string
	Workers       int
}

type App struct {
	cfg      resolvedConfig
	logger   *Logger
	runner   exiftool.CommandRunner
	lockPath string
}

type resolvedConfig struct {
	BaseDir       string
	InboxDir      string
	GPXDir        string
	ProcessedDir  string
	LogDir        string
	LogFile       string
	IssueFile     string
	Geosync       string
	RawExtensions map[string]struct{}
	Workers       int
}

type Logger struct {
	std  *log.Logger
	file io.WriteCloser
}

type Metadata = exiftool.Metadata

type runSummary struct {
	processed int
	success   int
	waiting   int
	failed    int
	skipped   int
	pending   int
}

type resultStatus string

const (
	statusSuccess resultStatus = "success"
	statusWaiting resultStatus = "waiting"
	statusFailed  resultStatus = "failed"
)

type issueKind string

const (
	issueKindMissingPair issueKind = "missing_pair"
	issueKindTrackGap    issueKind = "track_gap"
	issueKindFailure     issueKind = "failure"
)

type processingIssue struct {
	Kind               issueKind
	Reason             string
	Suggestion         string
	Asset              organizer.PhotoAsset
	PhotoTime          string
	PhotoOffset        string
	ReferencedGPXFiles []string
}

type assetResult struct {
	Status resultStatus
	Asset  organizer.PhotoAsset
	Issue  *processingIssue
}

func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Pictures", "GPS"), nil
}

func ParseExtensions(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(strings.TrimPrefix(part, ".")))
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func NewApp(cfg Config) (*App, error) {
	if len(cfg.RawExtensions) == 0 {
		cfg.RawExtensions = ParseExtensions("nef,cr3,arw,dng,raf,rw2,orf")
	}
	if cfg.Geosync == "" {
		cfg.Geosync = "0"
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	baseDir, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return nil, err
	}

	rcfg := resolvedConfig{
		BaseDir:      baseDir,
		InboxDir:     filepath.Join(baseDir, "Inbox"),
		GPXDir:       filepath.Join(baseDir, "GPX"),
		ProcessedDir: filepath.Join(baseDir, "Processed"),
		LogDir:       filepath.Join(baseDir, "Logs"),
		LogFile:      filepath.Join(baseDir, "Logs", "geotag.log"),
		IssueFile:    filepath.Join(baseDir, "Logs", "inbox_pending_report_latest.md"),
		Geosync:      cfg.Geosync,
		Workers:      cfg.Workers,
		RawExtensions: func() map[string]struct{} {
			out := make(map[string]struct{}, len(cfg.RawExtensions))
			for _, ext := range cfg.RawExtensions {
				out[strings.ToLower(ext)] = struct{}{}
			}
			return out
		}(),
	}

	if err := os.MkdirAll(rcfg.InboxDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(rcfg.GPXDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(rcfg.ProcessedDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(rcfg.LogDir, 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(rcfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:      rcfg,
		logger:   &Logger{std: log.New(io.MultiWriter(os.Stdout, file), "", 0), file: file},
		runner:   exiftool.ExecRunner{},
		lockPath: filepath.Join(baseDir, ".geotag.lock"),
	}, nil
}

func (a *App) Run() error {
	defer a.closeLogger()

	if err := a.acquireLock(); err != nil {
		if os.IsExist(err) {
			a.logger.Printf("已有任务正在处理中，本次跳过。")
			return nil
		}
		a.logger.Printf("处理失败：创建运行锁失败：%v", err)
		return err
	}
	defer a.releaseLock()

	a.logger.Printf("")
	a.logger.Printf("================ 开始处理 GPS 修正任务 ================")

	gpxFiles, err := a.listGPXFiles()
	if err != nil {
		a.logger.Printf("处理失败：读取 GPX 文件时出错：%v", err)
		return err
	}
	if len(gpxFiles) == 0 {
		a.logger.Printf("处理失败：未找到任何 GPX 轨迹文件。")
		return errors.New("no gpx files found")
	}

	a.logger.Printf("已加载 %d 个 GPX 轨迹文件。", len(gpxFiles))
	a.logger.Printf("本次并发处理数：%d", a.cfg.Workers)
	for _, gpx := range gpxFiles {
		a.logger.Printf("轨迹文件：%s", gpx)
	}

	assets, jpgOnly, err := a.discoverAssets()
	if err != nil {
		a.logger.Printf("处理失败：扫描 Inbox 时出错：%v", err)
		return err
	}

	summary := runSummary{}
	var issues []processingIssue
	for _, path := range jpgOnly {
		a.logger.Printf("发现未配对的 JPG，暂不处理：%s", filepath.Base(path))
		summary.skipped++
	}

	if len(assets) == 0 {
		a.logger.Printf("未发现可处理的 RAW 文件。")
		a.logger.Printf("================ 本次任务结束 ================")
		return nil
	}

	for _, result := range a.processAssets(assets, gpxFiles) {
		summary.processed++
		switch result.Status {
		case statusSuccess:
			summary.success++
		case statusWaiting:
			summary.waiting++
			summary.pending++
			if result.Issue != nil {
				issues = append(issues, *result.Issue)
			}
		case statusFailed:
			summary.failed++
			if result.Issue != nil {
				issues = append(issues, *result.Issue)
			}
		}
	}

	reportPath, reportErr := a.writePendingReport(issues)
	if reportErr != nil {
		a.logger.Printf("处理失败：写入待处理清单失败：%v", reportErr)
		return reportErr
	}
	if len(issues) > 0 {
		a.logger.Printf("已生成待处理清单：%s", reportPath)
	}

	a.logger.Printf("----------------------------------------")
	a.logger.Printf(
		"处理完成：共扫描 %d 个 RAW，成功 %d，待补资源 %d，失败 %d，跳过独立 JPG %d。", summary.processed, summary.success,
		summary.pending, summary.failed, summary.skipped,
	)
	a.logger.Printf("================ 本次任务结束 ================")

	if summary.failed > 0 {
		return fmt.Errorf("有 %d 个文件处理失败", summary.failed)
	}
	return nil
}

func (a *App) closeLogger() {
	if a.logger.file != nil {
		_ = a.logger.file.Close()
	}
}

func (a *App) acquireLock() error {
	return os.Mkdir(a.lockPath, 0o755)
}

func (a *App) releaseLock() {
	_ = os.Remove(a.lockPath)
}

func (a *App) listGPXFiles() ([]string, error) {
	entries := make([]string, 0)
	err := filepath.WalkDir(
		a.cfg.GPXDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".gpx") {
				entries = append(entries, path)
			}
			return nil
		},
	)
	return entries, err
}

func (a *App) discoverAssets() ([]organizer.PhotoAsset, []string, error) {
	rawExtsList := make([]string, 0, len(a.cfg.RawExtensions))
	for ext := range a.cfg.RawExtensions {
		rawExtsList = append(rawExtsList, ext)
	}

	allGroups, err := organizer.DiscoverMediaGroups(a.cfg.InboxDir, rawExtsList)
	if err != nil {
		return nil, nil, err
	}

	var assets []organizer.PhotoAsset
	var jpgOnly []string

	for _, group := range allGroups {
		if group.RawPath != "" {
			assets = append(assets, group)
			continue
		}
		if group.JPGPath != "" {
			jpgOnly = append(jpgOnly, group.JPGPath)
		}
	}

	return assets, jpgOnly, nil
}

func (a *App) processAssets(assets []organizer.PhotoAsset, gpxFiles []string) []assetResult {

	workerCount := a.cfg.Workers
	if workerCount > len(assets) {
		workerCount = len(assets)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	type job struct {
		index int
		asset organizer.PhotoAsset
	}
	type indexedResult struct {
		index  int
		result assetResult
	}

	jobs := make(chan job)
	results := make(chan indexedResult, len(assets))
	var wg sync.WaitGroup

	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				results <- indexedResult{
					index:  item.index,
					result: a.processAsset(item.asset, gpxFiles),
				}
			}
		}()
	}

	go func() {
		for i, asset := range assets {
			jobs <- job{index: i, asset: asset}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	ordered := make([]assetResult, len(assets))
	for result := range results {
		ordered[result.index] = result.result
	}
	return ordered
}

func (a *App) processAsset(asset organizer.PhotoAsset, gpxFiles []string) assetResult {

	a.logger.Printf("----------------------------------------")
	a.logger.PrintAssetf(asset, "开始处理：%s", filepath.Base(asset.RawPath))

	if asset.JPGPath == "" {
		a.logger.PrintAssetf(asset, "等待配对 JPG：%s", filepath.Base(asset.RawPath))
		return assetResult{
			Status: statusWaiting,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:       issueKindMissingPair,
				Reason:     "缺少同 basename 的 JPG，当前资产组继续保留在 Inbox。",
				Suggestion: "请补齐对应的 JPG 文件后重新运行。",
				Asset:      asset,
			},
		}
	}

	meta, err := a.readMetadata(asset.RawPath)
	if err != nil {
		a.logger.PrintAssetf(asset, "处理失败：读取 RAW 元数据失败：%v", err)
		return assetResult{
			Status: statusFailed,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:       issueKindFailure,
				Reason:     fmt.Sprintf("读取 RAW 元数据失败：%v", err),
				Suggestion: "请确认 RAW 文件可正常读取，然后重新运行。",
				Asset:      asset,
			},
		}
	}
	if meta.OffsetTimeOriginal != "" {
		a.logger.PrintAssetf(asset, "照片时区：%s", meta.OffsetTimeOriginal)
	} else {
		a.logger.PrintAssetf(asset, "照片时区：未读取到 OffsetTimeOriginal，将直接使用照片原始时间。")
	}

	changed, updatedMeta, err := a.geotagRaw(asset, meta, gpxFiles)
	if err != nil {
		issue := buildProcessingIssue(asset, meta, gpxFiles, err)
		a.logger.PrintAssetf(asset, "保留在 Inbox：%s", issue.Reason)
		return assetResult{
			Status: statusWaiting,
			Asset:  asset,
			Issue:  issue,
		}
	}

	if changed {
		a.logger.PrintAssetf(asset, "RAW 的 GPS 写入成功：%s @ %s", updatedMeta.GPSPosition, updatedMeta.GPSDateTime)
	} else {
		a.logger.PrintAssetf(asset, "RAW 的 GPS 坐标未变化：%s", updatedMeta.GPSPosition)
	}

	if err := a.syncJPG(asset, updatedMeta.GPSPosition); err != nil {
		a.logger.PrintAssetf(asset, "处理失败：%v", err)
		return assetResult{
			Status: statusFailed,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:               issueKindFailure,
				Reason:             fmt.Sprintf("同步 JPG 失败：%v", err),
				Suggestion:         "请检查 JPG 文件状态后重新运行。",
				Asset:              asset,
				PhotoTime:          updatedMeta.DateTimeOriginal,
				PhotoOffset:        updatedMeta.OffsetTimeOriginal,
				ReferencedGPXFiles: cloneStrings(gpxFiles),
			},
		}
	}
	a.logger.PrintAssetf(asset, "已同步 GPS 到 JPG：%s", filepath.Base(asset.JPGPath))

	if asset.XMPPath != "" {
		if err := a.syncXMP(asset, updatedMeta.GPSPosition); err != nil {
			a.logger.PrintAssetf(asset, "处理失败：%v", err)
			return assetResult{
				Status: statusFailed,
				Asset:  asset,
				Issue: &processingIssue{
					Kind:               issueKindFailure,
					Reason:             fmt.Sprintf("同步 XMP 失败：%v", err),
					Suggestion:         "请检查 XMP sidecar 内容后重新运行。",
					Asset:              asset,
					PhotoTime:          updatedMeta.DateTimeOriginal,
					PhotoOffset:        updatedMeta.OffsetTimeOriginal,
					ReferencedGPXFiles: cloneStrings(gpxFiles),
				},
			}
		}
		a.logger.PrintAssetf(asset, "已同步 GPS 到 XMP：%s", filepath.Base(asset.XMPPath))
	}

	targetDir, err := organizer.BuildArchiveDir(a.cfg.ProcessedDir, updatedMeta.DateTimeOriginal)
	if err != nil {
		a.logger.PrintAssetf(asset, "处理失败：无法确定归档目录：%v", err)
		return assetResult{
			Status: statusFailed,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:               issueKindFailure,
				Reason:             fmt.Sprintf("无法确定归档目录：%v", err),
				Suggestion:         "请检查 DateTimeOriginal 是否存在且格式正确。",
				Asset:              asset,
				PhotoTime:          updatedMeta.DateTimeOriginal,
				PhotoOffset:        updatedMeta.OffsetTimeOriginal,
				ReferencedGPXFiles: cloneStrings(gpxFiles),
			},
		}
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		a.logger.PrintAssetf(asset, "处理失败：创建归档目录失败：%v", err)
		return assetResult{
			Status: statusFailed,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:               issueKindFailure,
				Reason:             fmt.Sprintf("创建归档目录失败：%v", err),
				Suggestion:         "请检查 Processed 目录权限后重新运行。",
				Asset:              asset,
				PhotoTime:          updatedMeta.DateTimeOriginal,
				PhotoOffset:        updatedMeta.OffsetTimeOriginal,
				ReferencedGPXFiles: cloneStrings(gpxFiles),
			},
		}
	}
	newBaseName := organizer.CalculateNormalizedName(asset.BaseName, updatedMeta.DateTimeOriginal)
	if newBaseName != asset.BaseName {
		a.logger.PrintAssetf(asset, "将重命名为：%s", newBaseName)
	}

	if err := organizer.MoveFilesWithRename(asset.AllFiles(), targetDir, newBaseName); err != nil {
		a.logger.PrintAssetf(asset, "处理失败：归档文件失败：%v", err)
		return assetResult{
			Status: statusFailed,
			Asset:  asset,
			Issue: &processingIssue{
				Kind:               issueKindFailure,
				Reason:             fmt.Sprintf("归档文件失败：%v", err),
				Suggestion:         "请检查目标目录冲突或文件占用后重新运行。",
				Asset:              asset,
				PhotoTime:          updatedMeta.DateTimeOriginal,
				PhotoOffset:        updatedMeta.OffsetTimeOriginal,
				ReferencedGPXFiles: cloneStrings(gpxFiles),
			},
		}
	}
	a.logger.PrintAssetf(asset, "已归档到：%s", targetDir)

	return assetResult{
		Status: statusSuccess,
		Asset:  asset,
	}
}

func (a *App) readMetadata(path string) (Metadata, error) {
	return exiftool.ReadMetadata(a.runner, path)
}

func (a *App) geotagRaw(asset organizer.PhotoAsset, before Metadata, gpxFiles []string) (bool, Metadata, error) {
	output, err := exiftool.WriteGeotag(a.runner, asset.RawPath, gpxFiles, a.cfg.Geosync)

	after, readErr := a.readMetadata(asset.RawPath)
	if readErr != nil {
		return false, Metadata{}, fmt.Errorf("GPS 写入后重新读取 RAW 元数据失败: %w", readErr)
	}
	if after.GPSPosition == "" {
		reason := exiftool.ClassifyFailure(output, err)
		return false, Metadata{}, fmt.Errorf("%s", reason)
	}
	if err != nil {
		reason := exiftool.ClassifyFailure(output, err)
		return false, Metadata{}, fmt.Errorf("%s", reason)
	}

	return before.GPSPosition != after.GPSPosition, after, nil
}

func buildProcessingIssue(asset organizer.PhotoAsset, meta Metadata, gpxFiles []string, err error) *processingIssue {

	issue := &processingIssue{
		Kind:               issueKindFailure,
		Reason:             err.Error(),
		Suggestion:         "请根据失败原因检查相关资源后重新运行。",
		Asset:              asset,
		PhotoTime:          meta.DateTimeOriginal,
		PhotoOffset:        meta.OffsetTimeOriginal,
		ReferencedGPXFiles: cloneStrings(gpxFiles),
	}
	if strings.Contains(err.Error(), "轨迹范围内") {
		issue.Kind = issueKindTrackGap
		issue.Suggestion = "请补充覆盖该拍摄时间段的 GPX 轨迹文件，或检查照片时间和时区是否正确。"
	}
	return issue
}

func (a *App) syncJPG(asset organizer.PhotoAsset, expectedGPS string) error {
	if err := exiftool.SyncGPS(a.runner, asset.RawPath, asset.JPGPath); err != nil {
		return err
	}
	meta, err := a.readMetadata(asset.JPGPath)
	if err != nil {
		return fmt.Errorf("同步后读取 JPG 元数据失败：%w", err)
	}
	if meta.GPSPosition == "" {
		return errors.New("同步 JPG 后未读取到有效 GPS 信息")
	}
	if expectedGPS != "" && meta.GPSPosition != expectedGPS {
		return fmt.Errorf("同步 JPG 后坐标不一致：RAW=%s，JPG=%s", expectedGPS, meta.GPSPosition)
	}
	return nil
}

func (a *App) syncXMP(asset organizer.PhotoAsset, expectedGPS string) error {
	if err := exiftool.SyncXMPGPS(a.runner, asset.RawPath, asset.XMPPath); err != nil {
		return err
	}
	meta, err := a.readMetadata(asset.XMPPath)
	if err != nil {
		return fmt.Errorf("同步后读取 XMP 元数据失败：%w", err)
	}
	if meta.GPSPosition == "" {
		return errors.New("同步 XMP 后未读取到有效 GPS 信息")
	}
	if expectedGPS != "" && meta.GPSPosition != expectedGPS {
		return fmt.Errorf("同步 XMP 后坐标不一致：RAW=%s，XMP=%s", expectedGPS, meta.GPSPosition)
	}
	return nil
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func (a *App) writePendingReport(issues []processingIssue) (string, error) {
	var builder strings.Builder
	builder.WriteString("# Inbox 待处理清单\n\n")
	builder.WriteString(fmt.Sprintf("生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	if len(issues) == 0 {
		builder.WriteString("本次运行没有待补资源或待排查问题。\n")
		return a.cfg.IssueFile, os.WriteFile(a.cfg.IssueFile, []byte(builder.String()), 0o644)
	}

	for index, issue := range issues {
		builder.WriteString(fmt.Sprintf("## %d. %s\n\n", index+1, issue.Asset.BaseName))
		builder.WriteString(fmt.Sprintf("- 原因：%s\n", issue.Reason))
		builder.WriteString(fmt.Sprintf("- 建议：%s\n", issue.Suggestion))
		if issue.PhotoTime != "" {
			builder.WriteString(fmt.Sprintf("- 拍摄时间：%s\n", issue.PhotoTime))
		}
		if issue.PhotoOffset != "" {
			builder.WriteString(fmt.Sprintf("- 时区偏移：%s\n", issue.PhotoOffset))
		}
		builder.WriteString("- 涉及文件：\n")
		for _, path := range issue.Asset.AllFiles() {
			builder.WriteString(fmt.Sprintf("  - %s\n", path))
		}
		if len(issue.ReferencedGPXFiles) > 0 {
			builder.WriteString("- 本次参与匹配的 GPX：\n")
			for _, gpx := range issue.ReferencedGPXFiles {
				builder.WriteString(fmt.Sprintf("  - %s\n", gpx))
			}
		}
		builder.WriteString("\n")
	}

	return a.cfg.IssueFile, os.WriteFile(a.cfg.IssueFile, []byte(builder.String()), 0o644)
}

func (l *Logger) Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.std.Printf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func (l *Logger) PrintAssetf(asset organizer.PhotoAsset, format string, args ...any) {
	name := asset.BaseName
	if name == "" {
		name = filepath.Base(asset.RawPath)
	}
	l.Printf("[%s] %s", name, fmt.Sprintf(format, args...))
}
