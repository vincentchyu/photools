import Foundation
import PhotoolsCore
import SwiftUI

public enum WorkspaceSection: Hashable, CaseIterable, Identifiable, Sendable {
    case pipeline
    case inbox
    case processed
    case geodata
    case gpx
    case testRestore
    case guide

    public var id: Self { self }

    @MainActor
    public var title: String {
        switch self {
        case .pipeline:
            return LanguageManager.shared.text(.sectionPipeline)
        case .inbox:
            return LanguageManager.shared.text(.sectionInbox)
        case .processed:
            return LanguageManager.shared.text(.sectionProcessed)
        case .geodata:
            return LanguageManager.shared.text(.sectionGeodata)
        case .gpx:
            return LanguageManager.shared.text(.sectionGpx)
        case .testRestore:
            return LanguageManager.shared.text(.sectionTestRestore)
        case .guide:
            return LanguageManager.shared.text(.sectionGuide)
        }
    }

    public var systemImage: String {
        switch self {
        case .pipeline:
            return "bolt.horizontal.circle.fill"
        case .inbox:
            return "tray.full.fill"
        case .processed:
            return "archivebox.fill"
        case .geodata:
            return "globe.asia.australia.fill"
        case .gpx:
            return "map.fill"
        case .testRestore:
            return "arrow.counterclockwise.circle.fill"
        case .guide:
            return "questionmark.circle.fill"
        }
    }
}

private actor StringCollector {
    var text: String = ""
    func append(_ str: String) { text.append(str) }
    func getText() -> String { text }
}

@MainActor
public final class WorkspaceStore: ObservableObject {
    // 工作目录与基础设置
    @Published public var baseDirectory: String {
        didSet {
            if baseDirectory != oldValue {
                if flatMode {
                    sourceDirectory = baseDirectory
                    processedDirectory = baseDirectory
                } else {
                    let oldInbox = (oldValue as NSString).appendingPathComponent("Inbox")
                    let oldProcessed = (oldValue as NSString).appendingPathComponent("Processed")
                    if sourceDirectory.isEmpty || sourceDirectory == oldValue || sourceDirectory == oldInbox || sourceDirectory.hasSuffix("/Inbox") || !sourceDirectory.hasPrefix(baseDirectory) {
                        sourceDirectory = (baseDirectory as NSString).appendingPathComponent("Inbox")
                    }
                    if processedDirectory.isEmpty || processedDirectory == oldValue || processedDirectory == oldProcessed || processedDirectory.hasSuffix("/Processed") || !processedDirectory.hasPrefix(baseDirectory) {
                        processedDirectory = (baseDirectory as NSString).appendingPathComponent("Processed")
                    }
                }
                refresh()
            }
        }
    }
    @Published public var sourceDirectory: String
    @Published public var processedDirectory: String
    @Published public var flatMode: Bool {
        didSet {
            if flatMode != oldValue {
                if flatMode {
                    if sourceDirectory == (baseDirectory as NSString).appendingPathComponent("Inbox") || !sourceDirectory.hasPrefix(baseDirectory) {
                        sourceDirectory = baseDirectory
                    }
                    if processedDirectory == (baseDirectory as NSString).appendingPathComponent("Processed") || !processedDirectory.hasPrefix(baseDirectory) {
                        processedDirectory = baseDirectory
                    }
                } else {
                    if sourceDirectory == baseDirectory {
                        sourceDirectory = (baseDirectory as NSString).appendingPathComponent("Inbox")
                    }
                    if processedDirectory == baseDirectory {
                        processedDirectory = (baseDirectory as NSString).appendingPathComponent("Processed")
                    }
                }
                refresh()
            }
        }
    }
    @Published public var inPlace: Bool
    @Published public var gpxDirectory: String
    @Published public var rawExtensions: String
    @Published public var workers: Int

    // 四大能力插件开关与参数
    @Published public var enableGPXMatch: Bool
    @Published public var geosync: String
    @Published public var enableInterpolate: Bool
    @Published public var interpolateWindow: String
    @Published public var enableGeocode: Bool
    @Published public var allowNoGPS: Bool
    @Published public var enableArchive: Bool
    @Published public var testBackup: Bool

    // 运行状态与日志
    @Published public private(set) var summary: WorkspaceSummary?
    @Published public private(set) var runState: RunState = .idle
    @Published public private(set) var currentStageIndex: Int = -1
    @Published public private(set) var liveLog: String = ""
    @Published public var selectedSection: WorkspaceSection = .pipeline
    @Published public var selectedAssetID: PhotoAssetGroup.ID? {
        didSet {
            loadExifForSelectedAsset()
        }
    }

    // 高性能 500 行环形日志缓冲区 (丢弃超额历史日志，彻底杜绝内存与渲染卡顿)
    private var logLines: [String] = []
    public static let maxLiveLogLines = 500

    private static let logTimeFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss.SSS"
        return formatter
    }()

    // ExifTool 异步元数据检查器
    @Published public private(set) var selectedAssetExif: ExifMetadata?
    @Published public private(set) var isExifLoading: Bool = false
    private var exifLoadTask: Task<Void, Never>?

    // 离线地理数据包管理与反查测试
    @Published public private(set) var continentPacks: [GeodataContinentPack] = []
    @Published public private(set) var isGeodataLoading: Bool = false
    @Published public var testLatitude: String = "31.2304"
    @Published public var testLongitude: String = "121.4737"
    @Published public var testAltitude: String = "10"
    @Published public var isGeodataDebugMode: Bool = true
    @Published public private(set) var testLookupResult: GeodataLookupResult?
    @Published public private(set) var geodataLog: String = ""
    @Published public private(set) var isGeodataTesting: Bool = false

    // 使用指南与设计文档
    @Published public var selectedGuideDoc: GuideDocItem = GuideDocItem.allDocs.first!

    private let scanner: WorkspaceScanner
    private let engine: PhotoolsEngine
    private let processClient: PhotoolsProcessClient
    private let repositoryLocator: RepositoryLocator
    private let pendingReportParser: PendingReportParser
    private let exifReader: ExifMetadataReader

    public init(
        scanner: WorkspaceScanner = WorkspaceScanner(),
        engine: PhotoolsEngine = .shared,
        processClient: PhotoolsProcessClient = PhotoolsProcessClient(),
        repositoryLocator: RepositoryLocator = RepositoryLocator(),
        pendingReportParser: PendingReportParser = PendingReportParser(),
        exifReader: ExifMetadataReader = ExifMetadataReader()
    ) {
        self.scanner = scanner
        self.engine = engine
        self.processClient = processClient
        self.repositoryLocator = repositoryLocator
        self.pendingReportParser = pendingReportParser
        self.exifReader = exifReader

        let defaultBase = Self.defaultBaseDirectory
        let savedBase = UserDefaults.standard.string(forKey: "baseDirectory") ?? defaultBase
        self.baseDirectory = savedBase
        self.flatMode = UserDefaults.standard.bool(forKey: "flatMode")

        let defaultInbox = (savedBase as NSString).appendingPathComponent("Inbox")
        let defaultProcessed = (savedBase as NSString).appendingPathComponent("Processed")

        let savedSource = UserDefaults.standard.string(forKey: "sourceDirectory")
        let savedProcessed = UserDefaults.standard.string(forKey: "processedDirectory")

        if let savedSource, !savedSource.isEmpty {
            if savedSource.hasSuffix("/Inbox") || savedSource == (defaultBase as NSString).appendingPathComponent("Inbox") || !savedSource.hasPrefix(savedBase) {
                self.sourceDirectory = defaultInbox
            } else {
                self.sourceDirectory = savedSource
            }
        } else {
            self.sourceDirectory = defaultInbox
        }

        if let savedProcessed, !savedProcessed.isEmpty {
            if savedProcessed.hasSuffix("/Processed") || savedProcessed == (defaultBase as NSString).appendingPathComponent("Processed") || !savedProcessed.hasPrefix(savedBase) {
                self.processedDirectory = defaultProcessed
            } else {
                self.processedDirectory = savedProcessed
            }
        } else {
            self.processedDirectory = defaultProcessed
        }

        let defaultGPX = WorkspaceScanner.defaultGPXDirectory
        self.gpxDirectory = UserDefaults.standard.string(forKey: "gpxDirectory") ?? defaultGPX

        self.inPlace = UserDefaults.standard.bool(forKey: "inPlace")
        self.rawExtensions = UserDefaults.standard.string(forKey: "rawExtensions") ?? "nef,cr3,arw,dng,raf,rw2,orf"
        
        let savedWorkers = UserDefaults.standard.integer(forKey: "workers")
        self.workers = savedWorkers > 0 ? savedWorkers : ProcessInfo.processInfo.processorCount

        // 插件默认设置
        self.enableGPXMatch = UserDefaults.standard.object(forKey: "enableGPXMatch") != nil ? UserDefaults.standard.bool(forKey: "enableGPXMatch") : true
        self.geosync = UserDefaults.standard.string(forKey: "geosync") ?? "0"
        self.enableInterpolate = UserDefaults.standard.bool(forKey: "enableInterpolate")
        self.interpolateWindow = UserDefaults.standard.string(forKey: "interpolateWindow") ?? "15m"
        self.enableGeocode = UserDefaults.standard.object(forKey: "enableGeocode") != nil ? UserDefaults.standard.bool(forKey: "enableGeocode") : true
        self.allowNoGPS = UserDefaults.standard.bool(forKey: "allowNoGPS")
        self.enableArchive = UserDefaults.standard.object(forKey: "enableArchive") != nil ? UserDefaults.standard.bool(forKey: "enableArchive") : true
        self.testBackup = UserDefaults.standard.bool(forKey: "testBackup")

        // 启动瞬间执行进程内自检与插件预热
        warmupInProcessEngine()
    }

    public static var defaultBaseDirectory: String {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Pictures")
            .appendingPathComponent("GPS")
            .path
    }

    public var effectiveSourceDirectory: String {
        if flatMode {
            return baseDirectory
        }
        if sourceDirectory.isEmpty || sourceDirectory == baseDirectory || sourceDirectory.hasSuffix("/Inbox") || !sourceDirectory.hasPrefix(baseDirectory) {
            return (baseDirectory as NSString).appendingPathComponent("Inbox")
        }
        return sourceDirectory
    }

    public var effectiveProcessedDirectory: String {
        if flatMode {
            return baseDirectory
        }
        if processedDirectory.isEmpty || processedDirectory == baseDirectory || processedDirectory.hasSuffix("/Processed") || !processedDirectory.hasPrefix(baseDirectory) {
            return (baseDirectory as NSString).appendingPathComponent("Processed")
        }
        return processedDirectory
    }

    public var effectiveGPXDirectory: String {
        let trimmed = gpxDirectory.trimmingCharacters(in: .whitespaces)
        if trimmed.isEmpty {
            return WorkspaceScanner.defaultGPXDirectory
        }
        return (trimmed as NSString).expandingTildeInPath
    }

    public var selectedAsset: PhotoAssetGroup? {
        guard let selectedAssetID else {
            return summary?.assetGroups.first
        }
        return summary?.assetGroups.first { $0.id == selectedAssetID }
    }

    public var photoolsExecutablePath: String {
        let repoRoot = repositoryLocator.locateRepositoryRoot()
        return repositoryLocator.photoolsExecutablePath(repoRoot: repoRoot)
    }

    public func loadExifForSelectedAsset() {
        guard let asset = selectedAsset, let path = asset.primaryPath else {
            selectedAssetExif = nil
            return
        }
        exifLoadTask?.cancel()
        isExifLoading = true
        exifLoadTask = Task { [weak self] in
            guard let self else { return }
            do {
                let meta = try await self.exifReader.readMetadata(for: path)
                guard !Task.isCancelled else { return }
                self.selectedAssetExif = meta
                self.isExifLoading = false
            } catch {
                guard !Task.isCancelled else { return }
                self.isExifLoading = false
            }
        }
    }

    // 启动瞬间在后台并发完成四大插件的初始化并常驻内存
    private func warmupInProcessEngine() {
        guard engine.isLoaded else {
            appendLog("ℹ️ 运行于 CLI 子进程模式 (未检测到 libphotools.dylib)")
            return
        }

        appendLog("⚡ [photools Engine] 正在进程内异步预热四大核心插件与离线 3D KD-Tree 索引...")
        let eng = self.engine
        Task.detached(priority: .userInitiated) {
            eng.initializeEngine { [weak self] rep in
                guard let self else { return }
                Task { @MainActor in
                    if rep.status == "ready" {
                        self.appendLog("  • [就绪] \(rep.name): \(rep.message)")
                    } else if rep.status == "warning" {
                        self.appendLog("  • [提示] \(rep.name): \(rep.message)")
                    }
                }
            }
            Task { @MainActor [weak self] in
                self?.appendLog("🚀 [photools Engine] 插件全生命周期常驻内存就绪！\n")
                self?.loadGeodataList()
            }
        }
    }

    public func persistPreferences() {
        UserDefaults.standard.set(baseDirectory, forKey: "baseDirectory")
        UserDefaults.standard.set(sourceDirectory, forKey: "sourceDirectory")
        UserDefaults.standard.set(gpxDirectory, forKey: "gpxDirectory")
        UserDefaults.standard.set(processedDirectory, forKey: "processedDirectory")
        UserDefaults.standard.set(flatMode, forKey: "flatMode")
        UserDefaults.standard.set(inPlace, forKey: "inPlace")
        UserDefaults.standard.set(rawExtensions, forKey: "rawExtensions")
        UserDefaults.standard.set(workers, forKey: "workers")

        UserDefaults.standard.set(enableGPXMatch, forKey: "enableGPXMatch")
        UserDefaults.standard.set(geosync, forKey: "geosync")
        UserDefaults.standard.set(enableInterpolate, forKey: "enableInterpolate")
        UserDefaults.standard.set(interpolateWindow, forKey: "interpolateWindow")
        UserDefaults.standard.set(enableGeocode, forKey: "enableGeocode")
        UserDefaults.standard.set(allowNoGPS, forKey: "allowNoGPS")
        UserDefaults.standard.set(enableArchive, forKey: "enableArchive")
        UserDefaults.standard.set(testBackup, forKey: "testBackup")
    }

    public func refresh() {
        persistPreferences()
        do {
            let summary = try scanner.scan(
                baseDirectory: baseDirectory,
                sourceDirectory: effectiveSourceDirectory,
                gpxDirectory: effectiveGPXDirectory,
                rawExtensions: parsedRawExtensions
            )
            self.summary = summary
            if selectedAssetID == nil {
                selectedAssetID = summary.assetGroups.first?.id
            } else {
                loadExifForSelectedAsset()
            }
        } catch {
            appendLog("⚠️ 扫描工作区失败：\(error.localizedDescription)")
            summary = nil
        }
    }

    public func setBaseDirectoryAndResetPaths(_ path: String) {
        self.baseDirectory = path
        if flatMode {
            self.sourceDirectory = path
            self.processedDirectory = path
        } else {
            self.sourceDirectory = (path as NSString).appendingPathComponent("Inbox")
            self.processedDirectory = (path as NSString).appendingPathComponent("Processed")
        }
        refresh()
    }

    // 高性能追加日志 (自动增加毫秒级时间戳 [HH:mm:ss.SSS]，自动限制最多保留最近 500 行，杜绝内存与渲染卡顿)
    public func appendLog(_ text: String, withTimestamp: Bool = true) {
        let lines = text.components(separatedBy: "\n")
        let now = Self.logTimeFormatter.string(from: Date())
        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if !trimmed.isEmpty {
                if withTimestamp && !hasTimestampPrefix(trimmed) {
                    logLines.append("[\(now)] \(trimmed)")
                } else {
                    logLines.append(trimmed)
                }
            }
        }
        if logLines.count > Self.maxLiveLogLines {
            logLines.removeFirst(logLines.count - Self.maxLiveLogLines)
        }
        self.liveLog = logLines.joined(separator: "\n")
    }

    private func hasTimestampPrefix(_ text: String) -> Bool {
        guard text.count >= 13, text.hasPrefix("[") else { return false }
        let chars = Array(text.prefix(13))
        // 格式: [12:34:56.789]
        return chars[0] == "[" && chars[3] == ":" && chars[6] == ":" && chars[9] == "." && chars[12] == "]"
    }

    public func clearLiveLog() {
        logLines.removeAll()
        liveLog = ""
    }

    // 清除/重置当前任务状态与执行看板
    public func resetTaskStatus() {
        runState = .idle
        currentStageIndex = -1
    }

    public func updateStageProgress(stageName: String, message: String) {
        // 过滤环境自检与规则就绪等非运行时调度事件，防止启动预检时误推阶段
        if message.contains("自检") || message.contains("规则就绪") || message.contains("引擎就绪") || message.contains("就绪:") {
            return
        }

        let combined = (stageName + " " + message).lowercased()

        // 阶段 4: 拍摄日期归档与规范重命名
        if combined.contains("阶段 4") || combined.contains("拍摄日期归档") || (stageName == "归档重命名" && !combined.contains("就绪")) {
            if currentStageIndex < 4 { currentStageIndex = 4 }
        }
        // 阶段 3: 逆地理编码与地名写入
        else if combined.contains("阶段 3") || combined.contains("逆地理编码") || (stageName == "逆地理编码" && !combined.contains("就绪")) {
            if currentStageIndex < 3 { currentStageIndex = 3 }
        }
        // 阶段 2: GPS 智能邻近推断与时间插值
        else if combined.contains("阶段 2") || combined.contains("gps 智能邻近推断") || combined.contains("时间插值") || (combined.contains("推算") && !combined.contains("就绪")) {
            if currentStageIndex < 2 { currentStageIndex = 2 }
        }
        // 阶段 1: GPX 轨迹匹配与 GPS 修正
        else if combined.contains("阶段 1") || combined.contains("gpx 轨迹匹配") || (combined.contains("轨迹匹配") && !combined.contains("就绪")) {
            if currentStageIndex < 1 { currentStageIndex = 1 }
        }
        // 阶段 0: 资产扫描与预检
        else if combined.contains("扫描") || combined.contains("测试备份模式") || combined.contains("exif 元数据") || combined.contains("评估执行计划") || stageName == "扫描资产" || stageName == "预检校验" {
            if currentStageIndex < 0 { currentStageIndex = 0 }
        }
    }

    // 执行流水线 (优先使用进程内 FFI 引擎)
    public func runPipeline() {
        guard !runState.isRunning else { return }
        guard enableGPXMatch || enableInterpolate || enableGeocode || enableArchive else {
            appendLog("⚠️ 请至少启用一项能力插件！")
            return
        }

        persistPreferences()
        runState = .running
        currentStageIndex = 0
        clearLiveLog() // 启动前清空旧日志

        let gpxDir = effectiveGPXDirectory
        let backupDir = testBackup ? (baseDirectory as NSString).appendingPathComponent("Inbox_bak") : ""

        let options = PipelineRunOptions(
            baseDirectory: baseDirectory,
            sourceDirectory: effectiveSourceDirectory,
            gpxDirectory: gpxDir,
            processedDirectory: effectiveProcessedDirectory,
            flatMode: flatMode,
            inPlace: inPlace,
            geosync: geosync,
            rawExtensions: rawExtensions,
            workers: max(1, workers),
            enableGPX: enableGPXMatch,
            enableInterpolate: enableInterpolate,
            interpolateWindow: interpolateWindow,
            enableGeocode: enableGeocode,
            allowNoGPS: allowNoGPS,
            enableArchive: enableArchive,
            testBackup: testBackup,
            backupDirectory: backupDir
        )

        if engine.isLoaded {
            appendLog("🚀 开始执行自动化处理流水线 (In-Process Engine)...")
            Task {
                do {
                    let summary = try await engine.runPipeline(options: options) { [weak self] evt in
                        Task { @MainActor in
                            self?.appendLog("[\(evt.stage)] \(evt.message)")
                            self?.updateStageProgress(stageName: evt.stage, message: evt.message)
                        }
                    }
                    runState = .succeeded
                    currentStageIndex = 5 // 全部 5 个节点圆满完成
                    appendLog("🎉 流水线执行完毕！成功: \(summary.successCount), 跳过: \(summary.skipCount), 失败: \(summary.failCount), 耗时: \(String(format: "%.2f", summary.durationSeconds))s")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 流水线执行失败：\(error.localizedDescription)")
                    refresh()
                }
            }
        } else {
            // CLI 子进程降级模式
            let command = PhotoolsCommand.pipeline(executablePath: photoolsExecutablePath, options: options)
            appendLog("🚀 开始执行外部 CLI 流水线...")
            appendLog("$ \(command.executablePath) \(command.arguments.joined(separator: " "))")

            Task {
                do {
                    try await processClient.run(command: command) { [weak self] text in
                        Task { @MainActor in
                            self?.appendLog(text)
                            self?.updateStageProgress(stageName: "", message: text)
                        }
                    }
                    runState = .succeeded
                    currentStageIndex = 5 // 全部 5 个节点圆满完成
                    appendLog("✅ 流水线任务执行完成！")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 执行异常：\(error.localizedDescription)")
                    refresh()
                }
            }
        }
    }

    public func cancelCurrentTask() {
        guard runState.isRunning else { return }
        if engine.isLoaded {
            engine.cancelTask()
        } else {
            processClient.cancel()
        }
        runState = .failed("任务已被用户手动中断")
        appendLog("🛑 任务已被手动中断。")
    }

    public func createBackup() {
        guard !runState.isRunning else { return }
        runState = .running
        let srcDir = effectiveSourceDirectory
        let bakDir = (baseDirectory as NSString).appendingPathComponent("Inbox_bak")
        appendLog("📸 开始创建原始照片全量快照备份 [\(srcDir) -> \(bakDir)]...")

        if engine.isLoaded {
            Task {
                do {
                    let count = try await engine.createBackup(sourceDir: srcDir, backupDir: bakDir)
                    runState = .succeeded
                    appendLog("✅ 快照备份成功！共备份 \(count) 个文件。")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 快照备份失败：\(error.localizedDescription)")
                    refresh()
                }
            }
        } else {
            let gpxDir = effectiveGPXDirectory
            let options = PipelineRunOptions(
                baseDirectory: baseDirectory,
                sourceDirectory: srcDir,
                gpxDirectory: gpxDir,
                processedDirectory: effectiveProcessedDirectory,
                flatMode: flatMode,
                inPlace: inPlace,
                geosync: geosync,
                rawExtensions: rawExtensions,
                workers: max(1, workers),
                enableGPX: false,
                enableInterpolate: false,
                interpolateWindow: interpolateWindow,
                enableGeocode: false,
                allowNoGPS: true,
                enableArchive: false,
                testBackup: true,
                backupDirectory: bakDir
            )
            let cmd = PhotoolsCommand.pipeline(executablePath: photoolsExecutablePath, options: options)
            Task {
                do {
                    try await processClient.run(command: cmd) { [weak self] text in
                        Task { @MainActor in
                            self?.appendLog(text)
                        }
                    }
                    runState = .succeeded
                    appendLog("✅ 快照备份成功！")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 快照备份失败：\(error.localizedDescription)")
                    refresh()
                }
            }
        }
    }

    public func restoreBackup() {
        restoreTest(cleanProcessed: true)
    }

    public func restoreTest(cleanProcessed: Bool = true) {
        guard !runState.isRunning else { return }
        runState = .running
        let srcDir = effectiveSourceDirectory
        let bakDir = (baseDirectory as NSString).appendingPathComponent("Inbox_bak")
        appendLog("🔄 开始从快照还原原始照片 [\(bakDir) -> \(srcDir)]...")

        if engine.isLoaded {
            Task {
                do {
                    let count = try await engine.restoreBackup(
                        baseDirectory: baseDirectory,
                        backupDir: bakDir,
                        targetDir: srcDir,
                        cleanProcessed: cleanProcessed
                    )
                    runState = .succeeded
                    appendLog("✅ 快照还原成功！已恢复 \(count) 个文件至原始状态。")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 快照还原失败：\(error.localizedDescription)")
                    refresh()
                }
            }
        } else {
            let cmd = PhotoolsCommand.restoreTest(
                executablePath: photoolsExecutablePath,
                baseDirectory: baseDirectory,
                backupDir: bakDir,
                targetDir: srcDir,
                cleanProcessed: cleanProcessed
            )
            Task {
                do {
                    try await processClient.run(command: cmd) { [weak self] text in
                        Task { @MainActor in
                            self?.appendLog(text)
                        }
                    }
                    runState = .succeeded
                    appendLog("✅ 快照还原成功！")
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 快照还原失败：\(error.localizedDescription)")
                    refresh()
                }
            }
        }
    }

    public func loadGeodataList() {
        guard !isGeodataLoading else { return }
        isGeodataLoading = true

        let client = processClient
        let cmd = PhotoolsCommand.geodataList(executablePath: photoolsExecutablePath)
        let collector = StringCollector()

        Task {
            do {
                try await client.run(command: cmd) { text in
                    Task { await collector.append(text) }
                }
                let out = await collector.getText()
                let parsed = GeodataParser.parseListOutput(out)
                self.continentPacks = parsed
                self.isGeodataLoading = false
            } catch {
                self.isGeodataLoading = false
            }
        }
    }

    public func installGeodata(target: String) {
        guard !runState.isRunning else { return }
        runState = .running
        appendLog("📦 开始安装离线地理数据包 [\(target)]...")

        if engine.isLoaded {
            Task {
                do {
                    try await engine.installGeodata(target: target) { [weak self] line in
                        Task { @MainActor in
                            self?.appendLog(line)
                        }
                    }
                    runState = .succeeded
                    appendLog("✅ 离线地理数据包 [\(target)] 安装成功！")
                    loadGeodataList()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 安装失败：\(error.localizedDescription)")
                }
            }
        } else {
            let cmd = PhotoolsCommand.geodataInstall(executablePath: photoolsExecutablePath, target: target)
            Task {
                do {
                    try await processClient.run(command: cmd) { [weak self] text in
                        Task { @MainActor in
                            self?.appendLog(text)
                        }
                    }
                    runState = .succeeded
                    appendLog("✅ 离线地理数据包 [\(target)] 安装完成！")
                    loadGeodataList()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ 安装失败：\(error.localizedDescription)")
                }
            }
        }
    }

    public func removeGeodata(target: String) {
        guard !runState.isRunning else { return }
        appendLog("🗑️ 开始移除离线地理数据包 [\(target)]...")

        if engine.isLoaded {
            do {
                try engine.removeGeodata(target: target)
                appendLog("✅ 离线地理数据包 [\(target)] 已移除。")
                loadGeodataList()
            } catch {
                appendLog("❌ 移除失败：\(error.localizedDescription)")
            }
        } else {
            let cmd = PhotoolsCommand.geodataRemove(executablePath: photoolsExecutablePath, target: target)
            Task {
                do {
                    try await processClient.run(command: cmd) { [weak self] text in
                        Task { @MainActor in
                            self?.appendLog(text)
                        }
                    }
                    appendLog("✅ 离线地理数据包 [\(target)] 已移除。")
                    loadGeodataList()
                } catch {
                    appendLog("❌ 移除失败：\(error.localizedDescription)")
                }
            }
        }
    }

    public func appendGeodataLog(_ text: String) {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss.SSS"
        let timestamp = formatter.string(from: Date())
        let formatted = "[\(timestamp)] \(text)"
        if geodataLog.isEmpty {
            geodataLog = formatted
        } else {
            geodataLog += "\n" + formatted
        }
    }

    public func clearGeodataLog() {
        geodataLog = ""
    }

    public func testGeodataLookup() {
        guard let lat = Double(testLatitude), let lon = Double(testLongitude) else {
            testLookupResult = GeodataLookupResult(formattedSummary: "经纬度格式无效")
            appendGeodataLog("[ERROR] [逆地理] 经纬度格式无效: 纬度=\(testLatitude), 经度=\(testLongitude)")
            return
        }
        let alt = Double(testAltitude) ?? 0

        isGeodataTesting = true
        clearGeodataLog()
        appendGeodataLog("[INFO] [逆地理] 发起离线地理反查测试: 坐标 (纬度: \(String(format: "%.6f", lat)), 经度: \(String(format: "%.6f", lon)), 海拔: \(alt)m)")

        let startTime = CFAbsoluteTimeGetCurrent()

        if isGeodataDebugMode {
            let radLat = lat * .pi / 180.0
            let radLon = lon * .pi / 180.0
            let earthRadius = 6371.0
            let x = earthRadius * cos(radLat) * cos(radLon)
            let y = earthRadius * cos(radLat) * sin(radLon)
            let z = earthRadius * sin(radLat)
            appendGeodataLog("[INFO] [空间分析] 3D 笛卡尔坐标转换: (X: \(String(format: "%.2f", x)), Y: \(String(format: "%.2f", y)), Z: \(String(format: "%.2f", z)))")
            appendGeodataLog("[INFO] [空间分析] 启动 3D KD-Tree 拓扑空间索引剪枝检索 (维度 k=3)...")
        }

        if engine.isLoaded {
            appendGeodataLog("[INFO] [阶段 3] 采用 FFI 进程内 C-Shared 极速直通引擎检索...")
            let res = engine.lookupCoordinates(latitude: lat, longitude: lon, altitude: alt, debug: isGeodataDebugMode)
            let elapsed = (CFAbsoluteTimeGetCurrent() - startTime) * 1000.0
            self.testLookupResult = res
            self.isGeodataTesting = false

            if let debugTxt = res?.debugText, !debugTxt.isEmpty {
                // 直接展示完整的 3D 空间索引诊断、Top-5 拓扑候选点与 GeoPoint 结构
                self.geodataLog = debugTxt
            } else {
                if let res = res, !res.formattedSummary.isEmpty {
                    appendGeodataLog("[SUCCESS] [阶段 3] 命中中文地名: \(res.formattedSummary) (耗时: \(String(format: "%.2f", elapsed))ms, 数据源: \(res.source.isEmpty ? "Geonames" : res.source))")
                } else {
                    appendGeodataLog("[WARN] [阶段 3] 未在离线地理库中匹配到有效地点 (耗时: \(String(format: "%.2f", elapsed))ms)")
                }
            }
        } else {
            appendGeodataLog("[INFO] [阶段 3] 启动 photools CLI 异步子进程检索 (参数: --debug=\(isGeodataDebugMode))...")
            let client = processClient
            let cmd = PhotoolsCommand.geodataTest(
                executablePath: photoolsExecutablePath,
                latitude: lat,
                longitude: lon,
                altitude: alt,
                debug: isGeodataDebugMode
            )
            let collector = StringCollector()
            Task {
                do {
                    try await client.run(command: cmd) { [weak self] text in
                        Task { @MainActor [weak self] in
                            self?.appendGeodataLog(text)
                        }
                        Task { await collector.append(text) }
                    }
                    let out = await collector.getText()
                    let elapsed = (CFAbsoluteTimeGetCurrent() - startTime) * 1000.0
                    let parsed = GeodataParser.parseLookupOutput(out)
                    Task { @MainActor in
                        self.testLookupResult = parsed
                        self.isGeodataTesting = false
                        if let res = parsed, !res.formattedSummary.isEmpty {
                            self.appendGeodataLog("[SUCCESS] [阶段 3] 匹配完成: \(res.formattedSummary) (耗时: \(String(format: "%.2f", elapsed))ms)")
                        } else {
                            self.appendGeodataLog("[WARN] [阶段 3] 未匹配到有效地点 (耗时: \(String(format: "%.2f", elapsed))ms)")
                        }
                    }
                } catch {
                    let elapsed = (CFAbsoluteTimeGetCurrent() - startTime) * 1000.0
                    Task { @MainActor in
                        self.testLookupResult = GeodataLookupResult(formattedSummary: "反查失败: \(error.localizedDescription)")
                        self.isGeodataTesting = false
                        self.appendGeodataLog("[ERROR] [阶段 3] 反查失败: \(error.localizedDescription) (耗时: \(String(format: "%.2f", elapsed))ms)")
                    }
                }
            }
        }
    }

    public func performTestLookup() {
        testGeodataLookup()
    }

    public func pendingReportSection(for asset: PhotoAssetGroup) -> String? {
        guard let reportPath = summary?.pendingReportPath else { return nil }
        guard let content = try? String(contentsOfFile: reportPath, encoding: .utf8) else { return nil }
        return pendingReportParser.section(for: asset.baseName, in: content)
    }

    private var parsedRawExtensions: [String] {
        rawExtensions
            .components(separatedBy: ",")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() }
            .filter { !$0.isEmpty }
    }
}
