import Combine
import Foundation
import PhotoolsCore
import QuickLookUI
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
    @Published public var sidecarPolicy: String {
        didSet {
            if sidecarPolicy != oldValue {
                persistPreferences()
            }
        }
    }
    @Published public var companionExtensions: String
    @Published public var gpxDirectory: String
    @Published public var logDirectory: String
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

    // 插件权威元数据 (由 Go Capability/FFI 动态下发)
    @Published public private(set) var pluginMetas: [String: PluginMetadata] = [:]
    private var cancellables = Set<AnyCancellable>()

    // 运行状态与日志
    @Published public private(set) var summary: WorkspaceSummary?
    @Published public private(set) var runState: RunState = .idle
    @Published public private(set) var currentStageIndex: Int = -1
    @Published public private(set) var liveLog: String = ""
    @Published public var selectedSection: WorkspaceSection = .pipeline
    // QuickLook 快速预览状态 (按空格键唤起/关闭，切换选中照片时自动联动刷新)
    @Published public var quickLookURL: URL?

    @Published public var selectedAssetID: PhotoAssetGroup.ID? {
        didSet {
            loadExifForSelectedAsset()
            if quickLookURL != nil {
                if let asset = selectedAsset {
                    if let jpg = asset.jpgPath {
                        quickLookURL = URL(fileURLWithPath: jpg)
                    } else if let primary = asset.primaryPath {
                        quickLookURL = URL(fileURLWithPath: primary)
                    }
                } else {
                    quickLookURL = nil
                }
                if QLPreviewPanel.sharedPreviewPanelExists() && QLPreviewPanel.shared().isVisible {
                    QLPreviewPanel.shared().reloadData()
                }
            }
        }
    }

    /// 空格键触发/切换 QuickLook 快速预览
    public func toggleQuickLookForSelectedAsset() {
        guard let asset = selectedAsset else { return }
        if quickLookURL != nil {
            quickLookURL = nil
        } else {
            if let jpg = asset.jpgPath {
                quickLookURL = URL(fileURLWithPath: jpg)
            } else if let primary = asset.primaryPath {
                quickLookURL = URL(fileURLWithPath: primary)
            }
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

    // GPS 快速标记与剪贴板 (⌘G / ⌥⌘G / ⌥G)
    @Published public private(set) var copiedGPSMetadata: CopiedGPSMetadata?
    @Published public var assetGPSMap: [PhotoAssetGroup.ID: Bool] = [:]
    @Published public var hudMessage: String?
    @Published public var showingCopiedGPSInspector: Bool = false
    private var hudDismissTask: Task<Void, Never>?
    private var gpsScanTask: Task<Void, Never>?

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

    // 已归档照片目录层级下探与平铺状态
    @Published public var processedCurrentSubdir: String = ""
    @Published public var processedIsFlatRecursive: Bool {
        didSet {
            UserDefaults.standard.set(processedIsFlatRecursive, forKey: "processedIsFlatRecursive")
        }
    }

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

        // 1. 最高优先级：从系统磁盘 ~/.config/photools/plugins.json 加载全局 7 项持久偏好
        let diskGlobal = DiskConfigLoader.load()
        if let diskLang = diskGlobal?.language, !diskLang.isEmpty {
            let lang = AppLanguage.fromConfigString(diskLang)
            LanguageManager.shared.setLanguage(lang)
        }

        let defaultGPX = WorkspaceScanner.defaultGPXDirectory
        self.gpxDirectory = diskGlobal?.gpxDir ?? UserDefaults.standard.string(forKey: "gpxDirectory") ?? defaultGPX
        let defaultLog = ("~/.logs/photools" as NSString).expandingTildeInPath
        self.logDirectory = diskGlobal?.logDir ?? UserDefaults.standard.string(forKey: "logDirectory") ?? defaultLog

        let savedPolicy = diskGlobal?.sidecarPolicy ?? UserDefaults.standard.string(forKey: "sidecarPolicy") ?? "smart"
        self.sidecarPolicy = (savedPolicy == "read_only") ? "smart" : savedPolicy

        if let comp = diskGlobal?.companionExtensions, !comp.isEmpty {
            self.companionExtensions = comp.joined(separator: ",")
        } else {
            self.companionExtensions = UserDefaults.standard.string(forKey: "companionExtensions") ?? "wav,acr,exf"
        }

        if let raw = diskGlobal?.rawExtensions, !raw.isEmpty {
            self.rawExtensions = raw.joined(separator: ",")
        } else {
            self.rawExtensions = UserDefaults.standard.string(forKey: "rawExtensions") ?? "nef,cr3,arw,dng,raf,rw2,orf"
        }

        if let w = diskGlobal?.workers, w > 0 {
            self.workers = w
        } else {
            let savedWorkers = UserDefaults.standard.integer(forKey: "workers")
            self.workers = savedWorkers > 0 ? savedWorkers : ProcessInfo.processInfo.processorCount
        }

        self.inPlace = UserDefaults.standard.bool(forKey: "inPlace")

        // 插件默认设置
        self.enableGPXMatch = UserDefaults.standard.object(forKey: "enableGPXMatch") != nil ? UserDefaults.standard.bool(forKey: "enableGPXMatch") : true
        self.geosync = UserDefaults.standard.string(forKey: "geosync") ?? "0"
        self.enableInterpolate = UserDefaults.standard.bool(forKey: "enableInterpolate")
        self.interpolateWindow = UserDefaults.standard.string(forKey: "interpolateWindow") ?? "15m"
        self.enableGeocode = UserDefaults.standard.object(forKey: "enableGeocode") != nil ? UserDefaults.standard.bool(forKey: "enableGeocode") : true
        self.allowNoGPS = UserDefaults.standard.bool(forKey: "allowNoGPS")
        self.enableArchive = UserDefaults.standard.object(forKey: "enableArchive") != nil ? UserDefaults.standard.bool(forKey: "enableArchive") : true
        self.testBackup = UserDefaults.standard.bool(forKey: "testBackup")
        self.processedIsFlatRecursive = UserDefaults.standard.bool(forKey: "processedIsFlatRecursive")

        // 启动瞬间执行进程内自检与插件预热
        warmupInProcessEngine()

        // 确保引擎语言与当前 LanguageManager 同步后再拉取权威插件元数据
        let currentLang = LanguageManager.shared.currentLanguage
        engine.setLanguage(currentLang.toConfigString)
        reloadPluginMetas()

        // 监听多语言切换事件，实时热刷新插件权威文案
        LanguageManager.shared.$currentLanguage
            .receive(on: RunLoop.main)
            .sink { [weak self] lang in
                guard let self else { return }
                self.engine.setLanguage(lang.toConfigString)
                self.reloadPluginMetas()
            }
            .store(in: &cancellables)
    }

    /// 重新从 Go 引擎拉取最新语言的插件元数据
    public func reloadPluginMetas() {
        let isZh = LanguageManager.shared.currentLanguage.isChinese
        engine.setLanguage(isZh ? "zh-CN" : "en-US")
        let metas = engine.getPluginMetas()
        if !metas.isEmpty {
            var dict: [String: PluginMetadata] = [:]
            for m in metas {
                dict[m.id] = m
            }
            self.pluginMetas = dict
        }
    }

    /// 获取插件权威显示名称（优先 Go Capability，回退本地多语言）
    public func pluginName(id: String, fallback: String) -> String {
        pluginMetas[id]?.name ?? fallback
    }

    /// 获取插件权威详细描述（优先 Go Capability，回退本地多语言）
    public func pluginDesc(id: String, fallback: String) -> String {
        pluginMetas[id]?.desc ?? fallback
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
            if selectedSection == .processed {
                return processedCurrentAssets.first
            }
            return summary?.assetGroups.first
        }
        if let inboxMatch = summary?.assetGroups.first(where: { $0.id == selectedAssetID }) {
            return inboxMatch
        }
        return summary?.processedAssetGroups.first(where: { $0.id == selectedAssetID })
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
                self.assetGPSMap[asset.id] = meta.hasGPS
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
            appendLog(LanguageManager.shared.text(.logConsoleCliModeNotice))
            return
        }

        engine.setLanguage(LanguageManager.shared.currentLanguage.toConfigString)
        appendLog(LanguageManager.shared.text(.logConsoleWarmupStart))
        let eng = self.engine
        Task.detached(priority: .userInitiated) {
            eng.initializeEngine { [weak self] rep in
                guard let self else { return }
                Task { @MainActor in
                    if rep.status == "ready" {
                        self.appendLog("  • [\(rep.stage)] \(rep.name): \(rep.message)")
                    } else if rep.status == "warning" {
                        self.appendLog("  • [\(rep.stage)] \(rep.name): \(rep.message)")
                    }
                }
            }
            Task { @MainActor [weak self] in
                self?.appendLog(LanguageManager.shared.text(.logConsoleWarmupReady))
                self?.loadGeodataList()
            }
        }
    }

    public func persistPreferences() {
        UserDefaults.standard.set(baseDirectory, forKey: "baseDirectory")
        UserDefaults.standard.set(sourceDirectory, forKey: "sourceDirectory")
        UserDefaults.standard.set(gpxDirectory, forKey: "gpxDirectory")
        UserDefaults.standard.set(logDirectory, forKey: "logDirectory")
        UserDefaults.standard.set(processedDirectory, forKey: "processedDirectory")
        UserDefaults.standard.set(flatMode, forKey: "flatMode")
        UserDefaults.standard.set(inPlace, forKey: "inPlace")
        UserDefaults.standard.set(sidecarPolicy, forKey: "sidecarPolicy")
        UserDefaults.standard.set(companionExtensions, forKey: "companionExtensions")
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

        // 同步持久化写入 ~/.config/photools/plugins.json，与 CLI / TUI 保持 100% 共享
        let payload: [String: Any] = [
            "base_dir": baseDirectory,
            "source_dir": sourceDirectory,
            "gpx_dir": gpxDirectory,
            "log_dir": logDirectory,
            "language": LanguageManager.shared.currentLanguage.toConfigString,
            "processed_dir": processedDirectory,
            "flat_mode": flatMode,
            "sidecar_policy": sidecarPolicy,
            "sidecar_only": (sidecarPolicy == "sidecar_only"),
            "companion_extensions": companionExtensions,
            "raw_extensions": rawExtensions,
            "in_place": inPlace,
            "workers": workers,
            "enable_gpx_match": enableGPXMatch,
            "geosync": geosync,
            "enable_interpolate": enableInterpolate,
            "interpolate_window": interpolateWindow,
            "enable_geocode": enableGeocode,
            "allow_no_gps": allowNoGPS,
            "enable_archive": enableArchive,
            "test_backup": testBackup
        ]
        if let data = try? JSONSerialization.data(withJSONObject: payload),
           let jsonStr = String(data: data, encoding: .utf8) {
            engine.savePreferences(optionsJSON: jsonStr)
        }
    }

    public func refresh() {
        persistPreferences()
        do {
            let summary = try scanner.scan(
                baseDirectory: baseDirectory,
                sourceDirectory: effectiveSourceDirectory,
                gpxDirectory: effectiveGPXDirectory,
                logDirectory: logDirectory,
                rawExtensions: parsedRawExtensions
            )
            self.summary = summary
            if selectedAssetID == nil {
                selectedAssetID = summary.assetGroups.first?.id
            } else {
                loadExifForSelectedAsset()
            }
            scanAssetGPSStatuses()
        } catch {
            appendLog("⚠️ 扫描工作区失败：\(error.localizedDescription)")
            summary = nil
        }
    }

    public func scanAssetGPSStatuses() {
        guard let assets = summary?.assetGroups, !assets.isEmpty else { return }
        gpsScanTask?.cancel()
        gpsScanTask = Task { [weak self] in
            guard let self else { return }
            for asset in assets {
                if Task.isCancelled { break }
                if self.assetGPSMap[asset.id] != nil { continue }
                guard let path = asset.primaryPath else { continue }
                if let meta = try? await self.exifReader.readMetadata(for: path) {
                    if !Task.isCancelled {
                        self.assetGPSMap[asset.id] = meta.hasGPS
                    }
                }
            }
        }
    }

    // 全局 HUD 浮层提示与自动淡出
    public func showHUD(_ message: String) {
        hudDismissTask?.cancel()
        self.hudMessage = message
        hudDismissTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 2_500_000_000)
            guard !Task.isCancelled else { return }
            self?.hudMessage = nil
        }
    }

    // 快捷键 ⌘G: 快速拷贝当前选中照片的全部 GPS 元数据
    public func copySelectedAssetGPS() {
        guard let asset = selectedAsset else {
            showHUD(LanguageManager.shared.text(.noPhotoSelectedTitle))
            return
        }

        Task { [weak self] in
            guard let self else { return }
            let meta: ExifMetadata?
            if let cached = self.selectedAssetExif, cached.filePath == asset.primaryPath {
                meta = cached
            } else if let path = asset.primaryPath {
                meta = try? await self.exifReader.readMetadata(for: path)
            } else {
                meta = nil
            }

            guard let meta, meta.hasGPS, let lat = meta.latitude, let lon = meta.longitude else {
                self.showHUD(LanguageManager.shared.text(.noGpsToCopy))
                return
            }

            // 提取源照片上所有包含 GPS 的原始键值对
            var extractedGPSTags: [String: String] = [:]
            for item in meta.rawTags {
                if item.tag.localizedCaseInsensitiveContains("GPS") || item.group.localizedCaseInsensitiveContains("GPS") {
                    extractedGPSTags[item.tag] = item.value
                }
            }
            if extractedGPSTags["GPSLatitude"] == nil {
                extractedGPSTags["GPSLatitude"] = String(format: "%.6f°", abs(lat)) + (lat >= 0 ? " N" : " S")
            }
            if extractedGPSTags["GPSLongitude"] == nil {
                extractedGPSTags["GPSLongitude"] = String(format: "%.6f°", abs(lon)) + (lon >= 0 ? " E" : " W")
            }
            if extractedGPSTags["GPSLatitudeRef"] == nil {
                extractedGPSTags["GPSLatitudeRef"] = lat >= 0 ? "North" : "South"
            }
            if extractedGPSTags["GPSLongitudeRef"] == nil {
                extractedGPSTags["GPSLongitudeRef"] = lon >= 0 ? "East" : "West"
            }
            if extractedGPSTags["GPSPosition"] == nil, let pos = meta.gpsPosition {
                extractedGPSTags["GPSPosition"] = pos
            }
            if extractedGPSTags["GPSAltitude"] == nil, let alt = meta.altitude {
                extractedGPSTags["GPSAltitude"] = String(format: "%.1f m", alt)
            }
            if extractedGPSTags["GPSVersionID"] == nil {
                extractedGPSTags["GPSVersionID"] = "2.3.0.0"
            }
            if extractedGPSTags["GPSMapDatum"] == nil {
                extractedGPSTags["GPSMapDatum"] = "WGS-84"
            }

            let copied = CopiedGPSMetadata(
                latitude: lat,
                longitude: lon,
                altitude: meta.altitude,
                sourceAssetBaseName: asset.baseName,
                sourceFilePath: asset.primaryPath ?? "",
                captureDate: meta.dateTimeOriginal,
                gpsSource: meta.gpsSource,
                gpsMatchMethod: meta.gpsMatchMethod,
                locationSummary: meta.locationSummary,
                country: meta.country,
                province: meta.province,
                city: meta.city,
                district: meta.district,
                rawGPSTags: extractedGPSTags
            )

            self.copiedGPSMetadata = copied

            // 写入系统剪贴板 (纯文本格式)
            let pasteboard = NSPasteboard.general
            pasteboard.clearContents()
            pasteboard.setString(copied.plainTextSummary, forType: .string)

            self.showHUD("\(LanguageManager.shared.text(.gpsCopiedSuccess)) (\(copied.formattedDecimal)) [\(extractedGPSTags.count) 项]")
        }
    }

    // 快捷键 ⌥⌘G: 把已拷贝的 GPS 写入到目标照片及其配套文件
    public func pasteGPSToSelectedAsset() {
        guard let copied = copiedGPSMetadata else {
            showHUD(LanguageManager.shared.text(.noCopiedGps))
            return
        }

        guard let asset = selectedAsset, let primaryPath = asset.primaryPath else {
            showHUD(LanguageManager.shared.text(.noPhotoSelectedTitle))
            return
        }

        Task { [weak self] in
            guard let self else { return }
            let provDict: [String: String] = [
                "source": "manual_copied",
                "processor": "photools-desktop",
                "match_method": "clipboard_paste",
                "timestamp": ISO8601DateFormatter().string(from: Date())
            ]
            let provJSON = (try? JSONSerialization.data(withJSONObject: provDict)).flatMap { String(data: $0, encoding: .utf8) }

            let ok = self.engine.writePhotoGPSMetadata(
                sourcePath: copied.sourceFilePath,
                targetPath: primaryPath,
                latitude: copied.latitude,
                longitude: copied.longitude,
                altitude: copied.altitude ?? 0.0,
                provenanceJSON: provJSON,
                sidecarPolicy: self.sidecarPolicy
            )

            if ok {
                self.assetGPSMap[asset.id] = true
                self.loadExifForSelectedAsset()
                let policyTag: String
                switch self.sidecarPolicy {
                case "smart": policyTag = "智能模式 (RAW+JPG+XMP)"
                case "sidecar_only": policyTag = "纯侧车模式 (仅XMP)"
                case "embed_only": policyTag = "纯内嵌模式 (RAW+JPG)"
                default: policyTag = "双写模式 (原图+XMP)"
                }
                self.appendLog("✅ [\(policyTag)] 成功将 GPS 写入资产: \(asset.baseName)")
                self.showHUD("✅ 已按[\(policyTag)]写入: \(asset.baseName)")
            } else {
                let reason = self.engine.getLastErrorMessage() ?? "写入校验未通过"
                self.appendLog("❌ 写入 GPS 失败 [\(asset.baseName)]: \(reason)")
                self.showHUD("⚠️ 写入失败: \(reason)")
            }
        }
    }

    // 快捷键 ⌥G: 打开已拷贝 GPS 渲染预览窗口
    public func showCopiedGPSInspector() {
        self.showingCopiedGPSInspector = true
    }

    public func clearCopiedGPS() {
        self.copiedGPSMetadata = nil
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
            sidecarPolicy: sidecarPolicy,
            companionExtensions: companionExtensions,
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
            backupDirectory: backupDir,
            language: LanguageManager.shared.currentLanguage.toConfigString
        )

        if engine.isLoaded {
            appendLog(LanguageManager.shared.text(.logConsoleStarting))
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
                    let durStr = String(format: "%.2f", summary.durationSeconds)
                    let summaryFmt = LanguageManager.shared.text(.logConsoleFinishedSummary)
                    appendLog(String(format: summaryFmt, summary.successCount, summary.skipCount, summary.failCount, durStr))
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ \(error.localizedDescription)")
                    refresh()
                }
            }
        } else {
            // CLI 子进程降级模式
            let command = PhotoolsCommand.pipeline(executablePath: photoolsExecutablePath, options: options)
            appendLog(LanguageManager.shared.text(.logConsoleCliStarting))
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
                    appendLog("✅ " + LanguageManager.shared.text(.taskCompletedTitle))
                    refresh()
                } catch {
                    runState = .failed(error.localizedDescription)
                    appendLog("❌ \(error.localizedDescription)")
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
        runState = .failed(LanguageManager.shared.text(.logConsoleInterrupted))
        appendLog(LanguageManager.shared.text(.logConsoleInterrupted))
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
                sidecarPolicy: sidecarPolicy,
                companionExtensions: companionExtensions,
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

        if engine.isLoaded {
            let packs = engine.listGeodataPacks()
            if !packs.isEmpty {
                self.continentPacks = packs
                self.isGeodataLoading = false
                return
            }
        }

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

    // MARK: - Sidecar Policy Helper 属性 (供状态栏、工具栏与仪表盘复用)
    public var sidecarPolicyTitle: String {
        switch sidecarPolicy {
        case "sidecar_only":
            return LanguageManager.shared.text(.policySidecarOnly)
        case "embed_and_sidecar":
            return LanguageManager.shared.text(.policyEmbedAndSidecar)
        case "embed_only":
            return LanguageManager.shared.text(.policyEmbedOnly)
        default:
            return LanguageManager.shared.text(.policySmart)
        }
    }

    public var sidecarPolicyDescription: String {
        switch sidecarPolicy {
        case "sidecar_only":
            return LanguageManager.shared.text(.policySidecarOnlyDesc)
        case "embed_and_sidecar":
            return LanguageManager.shared.text(.policyEmbedAndSidecarDesc)
        case "embed_only":
            return LanguageManager.shared.text(.policyEmbedOnlyDesc)
        default:
            return LanguageManager.shared.text(.policySmartDesc)
        }
    }

    public var sidecarPolicyIcon: String {
        switch sidecarPolicy {
        case "sidecar_only":
            return "shield.lefthalf.filled"
        case "embed_and_sidecar":
            return "doc.on.doc.fill"
        case "embed_only":
            return "square.and.pencil"
        default:
            return "sparkles"
        }
    }

    public var sidecarPolicyColor: Color {
        switch sidecarPolicy {
        case "sidecar_only":
            return .blue
        case "embed_and_sidecar":
            return .purple
        case "embed_only":
            return .orange
        default:
            return .green
        }
    }

    // MARK: - 已归档照片资产与目录层级导航
    public var processedAllAssets: [PhotoAssetGroup] {
        summary?.processedAssetGroups ?? []
    }

    /// 面包屑导航各层级路径段
    public var processedBreadcrumbs: [String] {
        processedCurrentSubdir
            .split(separator: "/")
            .map(String.init)
            .filter { !$0.isEmpty }
    }

    /// 当前层级下的直接子文件夹列表（含下属照片数聚合，按名称倒序排列）
    public var processedDrillDownFolders: [ProcessedFolderItem] {
        guard !processedIsFlatRecursive else { return [] }
        let all = processedAllAssets
        guard let procDir = summary?.processedDirectory, !procDir.isEmpty else { return [] }

        var folderCounts: [String: Int] = [:]
        let targetPrefix = processedCurrentSubdir.isEmpty ? "" : processedCurrentSubdir + "/"

        for asset in all {
            let dir = asset.directory
            guard dir.hasPrefix(procDir) else { continue }
            var rel = String(dir.dropFirst(procDir.count))
            if rel.hasPrefix("/") {
                rel.removeFirst()
            }

            if targetPrefix.isEmpty {
                let parts = rel.split(separator: "/")
                if let first = parts.first {
                    folderCounts[String(first), default: 0] += 1
                }
            } else if rel.hasPrefix(targetPrefix) {
                let subRel = String(rel.dropFirst(targetPrefix.count))
                let parts = subRel.split(separator: "/")
                if let first = parts.first {
                    folderCounts[String(first), default: 0] += 1
                }
            }
        }

        return folderCounts.map { name, count in
            let path = targetPrefix.isEmpty ? name : "\(targetPrefix)\(name)"
            return ProcessedFolderItem(name: name, relativePath: path, photoCount: count)
        }
        .sorted { $0.name.localizedStandardCompare($1.name) == .orderedDescending }
    }

    /// 当前应该展示的照片组（平铺时返回全部，层级下探时仅返回当前目录直接包含的照片）
    public var processedCurrentAssets: [PhotoAssetGroup] {
        let all = processedAllAssets
        if processedIsFlatRecursive {
            return all
        }
        guard let procDir = summary?.processedDirectory, !procDir.isEmpty else { return [] }

        return all.filter { asset in
            let dir = asset.directory
            guard dir.hasPrefix(procDir) else { return false }
            var rel = String(dir.dropFirst(procDir.count))
            if rel.hasPrefix("/") {
                rel.removeFirst()
            }
            return rel == processedCurrentSubdir
        }
    }

    /// 进入子文件夹
    public func enterProcessedFolder(_ folderRelativePath: String) {
        processedCurrentSubdir = folderRelativePath
        selectedAssetID = processedCurrentAssets.first?.id
    }

    /// 面包屑导航跳转
    public func navigateToProcessedBreadcrumb(at index: Int) {
        let crumbs = processedBreadcrumbs
        if index < 0 || index >= crumbs.count {
            processedCurrentSubdir = ""
        } else {
            let selected = crumbs[0...index]
            processedCurrentSubdir = selected.joined(separator: "/")
        }
        selectedAssetID = processedCurrentAssets.first?.id
    }

    /// 重置为 Processed 根目录
    public func resetProcessedNavigation() {
        processedCurrentSubdir = ""
        selectedAssetID = processedCurrentAssets.first?.id
    }
}

public struct ProcessedFolderItem: Identifiable, Hashable, Sendable {
    public var id: String { relativePath }
    public let name: String
    public let relativePath: String
    public let photoCount: Int

    public init(name: String, relativePath: String, photoCount: Int) {
        self.name = name
        self.relativePath = relativePath
        self.photoCount = photoCount
    }
}
