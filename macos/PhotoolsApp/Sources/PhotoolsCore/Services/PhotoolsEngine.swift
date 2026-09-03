import Foundation

// MARK: - 公共数据类型模型

public struct PhotoolsInitReport: Sendable {
    public let pluginID: String
    public let name: String
    public let stage: String
    public let message: String
    public let percent: Double
    public let status: String
    public let error: String?

    public init(pluginID: String, name: String, stage: String, message: String, percent: Double, status: String, error: String? = nil) {
        self.pluginID = pluginID
        self.name = name
        self.stage = stage
        self.message = message
        self.percent = percent
        self.status = status
        self.error = error
    }
}

public struct PhotoolsProgressEvent: Sendable {
    public let stage: String
    public let level: String
    public let message: String
    public let assetName: String
    public let currentIndex: Int
    public let totalItems: Int

    public init(stage: String, level: String, message: String, assetName: String, currentIndex: Int, totalItems: Int) {
        self.stage = stage
        self.level = level
        self.message = message
        self.assetName = assetName
        self.currentIndex = currentIndex
        self.totalItems = totalItems
    }
}

public struct PhotoolsTaskSummary: Sendable {
    public let successCount: Int
    public let skipCount: Int
    public let failCount: Int
    public let durationSeconds: Double

    public init(successCount: Int, skipCount: Int, failCount: Int, durationSeconds: Double) {
        self.successCount = successCount
        self.skipCount = skipCount
        self.failCount = failCount
        self.durationSeconds = durationSeconds
    }
}

// MARK: - C ABI 函数指针类型定义

private typealias FnInit = @convention(c) (
    @convention(c) (UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, Double, UnsafePointer<CChar>?, UnsafePointer<CChar>?) -> Void
) -> Void

private typealias FnRunPipeline = @convention(c) (
    UnsafePointer<CChar>,
    @convention(c) (UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, Int32, Int32) -> Void,
    @convention(c) (Int32, Int32, Int32, Double, UnsafePointer<CChar>?) -> Void
) -> Void

private typealias FnLookupCoordinates = @convention(c) (Double, Double, Double, Int32) -> UnsafePointer<CChar>?
private typealias FnListGeodataPacks = @convention(c) () -> UnsafePointer<CChar>?
private typealias FnInstallGeodata = @convention(c) (
    UnsafePointer<CChar>,
    @convention(c) (UnsafePointer<CChar>?) -> Void
) -> Int32
private typealias FnRemoveGeodata = @convention(c) (UnsafePointer<CChar>) -> Int32
private typealias FnCreateBackup = @convention(c) (UnsafePointer<CChar>, UnsafePointer<CChar>) -> Int32
private typealias FnRestoreBackup = @convention(c) (UnsafePointer<CChar>, UnsafePointer<CChar>, UnsafePointer<CChar>, Int32) -> Int32
private typealias FnCancelTask = @convention(c) () -> Void
private typealias FnInspectPhotoMetadata = @convention(c) (UnsafePointer<CChar>) -> UnsafePointer<CChar>?
private typealias FnWritePhotoGPSMetadata = @convention(c) (
    UnsafePointer<CChar>?,
    UnsafePointer<CChar>,
    Double,
    Double,
    Double,
    UnsafePointer<CChar>?,
    UnsafePointer<CChar>?
) -> Int32
private typealias FnSetLanguage = @convention(c) (UnsafePointer<CChar>?) -> Void
private typealias FnSavePreferences = @convention(c) (UnsafePointer<CChar>?) -> Int32
private typealias FnGetGlobalConfigJSON = @convention(c) () -> UnsafePointer<CChar>?
private typealias FnGetPluginMetasJSON = @convention(c) () -> UnsafePointer<CChar>?
private typealias FnGetGlobalOptionSpecsJSON = @convention(c) () -> UnsafePointer<CChar>?
private typealias FnShutdown = @convention(c) () -> Void
private typealias FnGetLastErrorMessage = @convention(c) () -> UnsafePointer<CChar>?
private typealias FnFreeString = @convention(c) (UnsafePointer<CChar>?) -> Void

// MARK: - Swift 回调保持器

private typealias CInitCallback = @convention(c) (UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, Double, UnsafePointer<CChar>?, UnsafePointer<CChar>?) -> Void
private typealias CEventCallback = @convention(c) (UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, UnsafePointer<CChar>?, Int32, Int32) -> Void
private typealias CDoneCallback = @convention(c) (Int32, Int32, Int32, Double, UnsafePointer<CChar>?) -> Void
private typealias CLogCallback = @convention(c) (UnsafePointer<CChar>?) -> Void

private final class CallbackHolder: @unchecked Sendable {
    static let shared = CallbackHolder()
    var initCallback: ((PhotoolsInitReport) -> Void)?
    var eventCallback: ((PhotoolsProgressEvent) -> Void)?
    var doneCallback: ((PhotoolsTaskSummary, Error?) -> Void)?
    var logCallback: ((String) -> Void)?
}

public final class PhotoolsEngine: @unchecked Sendable {
    public static let shared = PhotoolsEngine()

    private var dylibHandle: UnsafeMutableRawPointer?
    private var fnInit: FnInit?
    private var fnRunPipeline: FnRunPipeline?
    private var fnLookupCoordinates: FnLookupCoordinates?
    private var fnListGeodataPacks: FnListGeodataPacks?
    private var fnInstallGeodata: FnInstallGeodata?
    private var fnRemoveGeodata: FnRemoveGeodata?
    private var fnCreateBackup: FnCreateBackup?
    private var fnRestoreBackup: FnRestoreBackup?
    private var fnCancelTask: FnCancelTask?
    private var fnInspectPhotoMetadata: FnInspectPhotoMetadata?
    private var fnWritePhotoGPSMetadata: FnWritePhotoGPSMetadata?
    private var fnSetLanguage: FnSetLanguage?
    private var fnSavePreferences: FnSavePreferences?
    private var fnGetGlobalConfigJSON: FnGetGlobalConfigJSON?
    private var fnGetPluginMetasJSON: FnGetPluginMetasJSON?
    private var fnGetGlobalOptionSpecsJSON: FnGetGlobalOptionSpecsJSON?
    private var fnShutdown: FnShutdown?
    private var fnGetLastErrorMessage: FnGetLastErrorMessage?
    private var fnFreeString: FnFreeString?

    public private(set) var isLoaded: Bool = false
    public private(set) var isInitialized: Bool = false
    private let lock = NSLock()

    public init(customDylibPath: String? = nil) {
        loadDylib(at: customDylibPath)
    }

    deinit {
        shutdown()
        if let handle = dylibHandle {
            dlclose(handle)
        }
    }

    private func loadDylib(at customPath: String?) {
        lock.lock()
        defer { lock.unlock() }

        var candidates: [String] = []
        if let customPath {
            candidates.append(customPath)
        }

        // 1. App Bundle Frameworks 目录
        if let frameworksPath = Bundle.main.privateFrameworksPath {
            candidates.append((frameworksPath as NSString).appendingPathComponent("libphotools.dylib"))
        }

        // 2. 当前运行目录或上级 dist 目录
        let repoRoot = RepositoryLocator().locateRepositoryRoot()
        candidates.append((repoRoot as NSString).appendingPathComponent("dist/libphotools.dylib"))
        candidates.append((repoRoot as NSString).appendingPathComponent("libphotools.dylib"))
        candidates.append("libphotools.dylib")

        for path in candidates {
            if let handle = dlopen(path, RTLD_NOW | RTLD_GLOBAL) {
                self.dylibHandle = handle
                bindSymbols(handle)
                self.isLoaded = true
                return
            }
        }
    }

    private func bindSymbols(_ handle: UnsafeMutableRawPointer) {
        if let sym = dlsym(handle, "Photools_Init") {
            fnInit = unsafeBitCast(sym, to: FnInit.self)
        }
        if let sym = dlsym(handle, "Photools_RunPipeline") {
            fnRunPipeline = unsafeBitCast(sym, to: FnRunPipeline.self)
        }
        if let sym = dlsym(handle, "Photools_LookupCoordinates") {
            fnLookupCoordinates = unsafeBitCast(sym, to: FnLookupCoordinates.self)
        }
        if let sym = dlsym(handle, "Photools_ListGeodataPacks") {
            fnListGeodataPacks = unsafeBitCast(sym, to: FnListGeodataPacks.self)
        }
        if let sym = dlsym(handle, "Photools_InstallGeodata") {
            fnInstallGeodata = unsafeBitCast(sym, to: FnInstallGeodata.self)
        }
        if let sym = dlsym(handle, "Photools_RemoveGeodata") {
            fnRemoveGeodata = unsafeBitCast(sym, to: FnRemoveGeodata.self)
        }
        if let sym = dlsym(handle, "Photools_CreateBackup") {
            fnCreateBackup = unsafeBitCast(sym, to: FnCreateBackup.self)
        }
        if let sym = dlsym(handle, "Photools_RestoreBackup") {
            fnRestoreBackup = unsafeBitCast(sym, to: FnRestoreBackup.self)
        }
        if let sym = dlsym(handle, "Photools_CancelTask") {
            fnCancelTask = unsafeBitCast(sym, to: FnCancelTask.self)
        }
        if let sym = dlsym(handle, "Photools_InspectPhotoMetadata") {
            fnInspectPhotoMetadata = unsafeBitCast(sym, to: FnInspectPhotoMetadata.self)
        }
        if let sym = dlsym(handle, "Photools_WritePhotoGPSMetadata") {
            fnWritePhotoGPSMetadata = unsafeBitCast(sym, to: FnWritePhotoGPSMetadata.self)
        }
        if let sym = dlsym(handle, "Photools_SetLanguage") {
            fnSetLanguage = unsafeBitCast(sym, to: FnSetLanguage.self)
        }
        if let sym = dlsym(handle, "Photools_SavePreferences") {
            fnSavePreferences = unsafeBitCast(sym, to: FnSavePreferences.self)
        }
        if let sym = dlsym(handle, "Photools_GetGlobalConfigJSON") {
            fnGetGlobalConfigJSON = unsafeBitCast(sym, to: FnGetGlobalConfigJSON.self)
        }
        if let sym = dlsym(handle, "Photools_GetPluginMetasJSON") {
            fnGetPluginMetasJSON = unsafeBitCast(sym, to: FnGetPluginMetasJSON.self)
        }
        if let sym = dlsym(handle, "Photools_GetGlobalOptionSpecsJSON") {
            fnGetGlobalOptionSpecsJSON = unsafeBitCast(sym, to: FnGetGlobalOptionSpecsJSON.self)
        }
        if let sym = dlsym(handle, "Photools_Shutdown") {
            fnShutdown = unsafeBitCast(sym, to: FnShutdown.self)
        }
        if let sym = dlsym(handle, "Photools_GetLastErrorMessage") {
            fnGetLastErrorMessage = unsafeBitCast(sym, to: FnGetLastErrorMessage.self)
        }
        if let sym = dlsym(handle, "Photools_FreeString") {
            fnFreeString = unsafeBitCast(sym, to: FnFreeString.self)
        }
    }

    // 1. 初始化引擎
    public func initialize(onProgress: @escaping @Sendable (PhotoolsInitReport) -> Void) {
        guard let fnInit else { return }
        CallbackHolder.shared.initCallback = onProgress

        let cCallback: CInitCallback = { pluginID, name, stage, msg, pct, status, errMsg in
            let pID = pluginID.map { String(cString: $0) } ?? ""
            let n = name.map { String(cString: $0) } ?? ""
            let s = stage.map { String(cString: $0) } ?? ""
            let m = msg.map { String(cString: $0) } ?? ""
            let st = status.map { String(cString: $0) } ?? "ready"
            let err = (errMsg != nil && String(cString: errMsg!).isEmpty == false) ? String(cString: errMsg!) : nil

            let rep = PhotoolsInitReport(
                pluginID: pID,
                name: n,
                stage: s,
                message: m,
                percent: pct,
                status: st,
                error: err
            )
            CallbackHolder.shared.initCallback?(rep)
        }

        fnInit(cCallback)
        isInitialized = true
    }

    public func initializeEngine(onProgress: @escaping @Sendable (PhotoolsInitReport) -> Void) {
        initialize(onProgress: onProgress)
    }

    // 2. 毫秒级内存经纬度高精反查
    public func lookupCoordinates(latitude: Double, longitude: Double, altitude: Double = 0.0, debug: Bool = false) -> GeodataLookupResult? {
        guard let fnLookupCoordinates else { return nil }

        guard let cStr = fnLookupCoordinates(latitude, longitude, altitude, debug ? 1 : 0) else {
            return nil
        }
        defer { fnFreeString?(cStr) }

        let jsonString = String(cString: cStr)
        guard let data = jsonString.data(using: .utf8),
              let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }

        let eleInt: Int = {
            if let d = dict["elevation"] as? Double { return Int(d) }
            if let i = dict["elevation"] as? Int { return i }
            return 0
        }()
        let dist = dict["distance_km"] as? Double ?? 0.0
        let debugTxt = dict["debug_text"] as? String ?? ""

        var candidateList: [GeodataCandidatePoint] = []
        if let rawCandidates = dict["candidates"] as? [[String: Any]] {
            for item in rawCandidates {
                let rank = item["rank"] as? Int ?? 1
                let name = item["name"] as? String ?? ""
                let nameZH = item["name_zh"] as? String ?? name
                let feature = item["feature_desc"] as? String ?? ""
                let hierarchy = item["location_hierarchy"] as? String ?? ""
                let dKm = item["distance_km"] as? Double ?? 0.0
                let cLat = item["lat"] as? Double ?? 0.0
                let cLon = item["lon"] as? Double ?? 0.0
                let cEle = item["elevation"] as? Int ?? 0
                let gID = item["geoname_id"] as? Int ?? 0
                let src = item["source"] as? String ?? ""

                candidateList.append(
                    GeodataCandidatePoint(
                        rank: rank,
                        name: name,
                        nameZH: nameZH,
                        featureDesc: feature,
                        locationHierarchy: hierarchy,
                        distanceKm: dKm,
                        lat: cLat,
                        lon: cLon,
                        elevation: cEle,
                        geonameID: gID,
                        source: src
                    )
                )
            }
        }

        return GeodataLookupResult(
            country: dict["country"] as? String ?? "",
            countryCode: dict["country_code"] as? String ?? "",
            province: dict["province"] as? String ?? "",
            city: dict["city"] as? String ?? "",
            district: dict["district"] as? String ?? "",
            timezone: dict["timezone"] as? String ?? "",
            elevation: eleInt,
            distanceKm: dist,
            source: dict["source"] as? String ?? "",
            formattedSummary: dict["formatted_summary"] as? String ?? "",
            debugText: debugTxt,
            candidates: candidateList
        )
    }

    // 3. 读取地理数据包列表
    public func listGeodataPacks() -> [GeodataContinentPack] {
        guard let fnListGeodataPacks else { return [] }

        guard let cStr = fnListGeodataPacks() else { return [] }
        defer { fnFreeString?(cStr) }

        let jsonString = String(cString: cStr)
        return GeodataParser.parseListJSON(jsonString)
    }

    // 4. 执行流水线
    public func runPipeline(
        options: PipelineRunOptions,
        onEvent: @escaping @Sendable (PhotoolsProgressEvent) -> Void
    ) async throws -> PhotoolsTaskSummary {
        guard let fnRunPipeline else {
            throw NSError(domain: "PhotoolsEngine", code: -1, userInfo: [NSLocalizedDescriptionKey: "动态库未加载或缺少 Photools_RunPipeline 导出符号"])
        }

        let jsonDict: [String: Any] = [
            "base_dir": options.baseDirectory,
            "source_dir": options.sourceDirectory,
            "gpx_dir": options.gpxDirectory,
            "processed_dir": options.processedDirectory,
            "flat_mode": options.flatMode,
            "sidecar_policy": options.sidecarPolicy,
            "sidecar_only": options.sidecarPolicy == "sidecar_only",
            "companion_extensions": options.companionExtensions,
            "in_place": options.inPlace,
            "geosync": options.geosync,
            "raw_extensions": options.rawExtensions,
            "workers": options.workers,
            "enable_gpx_match": options.enableGPX,
            "enable_interpolate": options.enableInterpolate,
            "interpolate_window": options.interpolateWindow,
            "enable_geocode": options.enableGeocode,
            "allow_no_gps": options.allowNoGPS,
            "enable_archive": options.enableArchive,
            "test_backup": options.testBackup,
            "backup_dir": options.testBackup ? options.backupDirectory : "",
            "language": options.language
        ]

        guard let jsonData = try? JSONSerialization.data(withJSONObject: jsonDict),
              let jsonStr = String(data: jsonData, encoding: .utf8) else {
            throw NSError(domain: "PhotoolsEngine", code: -2, userInfo: [NSLocalizedDescriptionKey: "配置参数序列化失败"])
        }

        CallbackHolder.shared.eventCallback = onEvent

        return try await withCheckedThrowingContinuation { continuation in
            CallbackHolder.shared.doneCallback = { summary, err in
                if let err {
                    continuation.resume(throwing: err)
                } else {
                    continuation.resume(returning: summary)
                }
            }

            let cEventCB: CEventCallback = { stage, level, msg, asset, cur, total in
                let evt = PhotoolsProgressEvent(
                    stage: stage.map { String(cString: $0) } ?? "",
                    level: level.map { String(cString: $0) } ?? "info",
                    message: msg.map { String(cString: $0) } ?? "",
                    assetName: asset.map { String(cString: $0) } ?? "",
                    currentIndex: Int(cur),
                    totalItems: Int(total)
                )
                CallbackHolder.shared.eventCallback?(evt)
            }

            let cDoneCB: CDoneCallback = { succ, skip, fail, dur, errMsg in
                let summary = PhotoolsTaskSummary(
                    successCount: Int(succ),
                    skipCount: Int(skip),
                    failCount: Int(fail),
                    durationSeconds: dur
                )
                if let errMsg, !String(cString: errMsg).isEmpty {
                    let err = NSError(domain: "PhotoolsEngine", code: -3, userInfo: [NSLocalizedDescriptionKey: String(cString: errMsg)])
                    CallbackHolder.shared.doneCallback?(summary, err)
                } else {
                    CallbackHolder.shared.doneCallback?(summary, nil)
                }
            }

            jsonStr.withCString { cJson in
                fnRunPipeline(cJson, cEventCB, cDoneCB)
            }
        }
    }

    // 5. 安装地理数据包
    public func installGeodata(target: String, onLog: @escaping @Sendable (String) -> Void) async throws {
        guard let fnInstallGeodata else {
            throw NSError(domain: "PhotoolsEngine", code: -1, userInfo: [NSLocalizedDescriptionKey: "动态库未加载"])
        }

        CallbackHolder.shared.logCallback = onLog
        let cLogCB: CLogCallback = { msg in
            let text = msg.map { String(cString: $0) } ?? ""
            CallbackHolder.shared.logCallback?(text)
        }

        return try await withCheckedThrowingContinuation { continuation in
            let res = target.withCString { cTarget in
                fnInstallGeodata(cTarget, cLogCB)
            }
            if res == 0 {
                continuation.resume()
            } else {
                continuation.resume(throwing: NSError(domain: "PhotoolsEngine", code: Int(res), userInfo: [NSLocalizedDescriptionKey: "安装失败"]))
            }
        }
    }

    // 6. 卸载地理数据包
    public func removeGeodata(target: String) throws {
        guard let fnRemoveGeodata else {
            throw NSError(domain: "PhotoolsEngine", code: -1, userInfo: [NSLocalizedDescriptionKey: "动态库未加载"])
        }
        let res = target.withCString { cTarget in
            fnRemoveGeodata(cTarget)
        }
        if res != 0 {
            throw NSError(domain: "PhotoolsEngine", code: Int(res), userInfo: [NSLocalizedDescriptionKey: "卸载失败"])
        }
    }

    // 7. 一键快照备份
    public func createBackup(sourceDir: String, backupDir: String) async throws -> Int {
        guard let fnCreateBackup else {
            throw NSError(domain: "PhotoolsEngine", code: -1, userInfo: [NSLocalizedDescriptionKey: "动态库未加载"])
        }
        let res = sourceDir.withCString { cSrc in
            backupDir.withCString { cBak in
                fnCreateBackup(cSrc, cBak)
            }
        }
        if res < 0 {
            throw NSError(domain: "PhotoolsEngine", code: Int(res), userInfo: [NSLocalizedDescriptionKey: "快照备份失败"])
        }
        return Int(res)
    }

    // 8. 一键快照还原 (正确对接 4 个 C ABI 参数: baseDir, backupDir, targetDir, cleanProcessed)
    public func restoreBackup(
        baseDirectory: String,
        backupDir: String = "",
        targetDir: String = "",
        cleanProcessed: Bool = true
    ) async throws -> Int {
        guard let fnRestoreBackup else {
            throw NSError(domain: "PhotoolsEngine", code: -1, userInfo: [NSLocalizedDescriptionKey: "动态库未加载"])
        }
        let res = baseDirectory.withCString { cBase in
            backupDir.withCString { cBak in
                targetDir.withCString { cTgt in
                    fnRestoreBackup(cBase, cBak, cTgt, cleanProcessed ? 1 : 0)
                }
            }
        }
        if res < 0 {
            throw NSError(domain: "PhotoolsEngine", code: Int(res), userInfo: [NSLocalizedDescriptionKey: "快照还原失败"])
        }
        return Int(res)
    }

    // 9. 中断任务
    public func cancelTask() {
        fnCancelTask?()
    }

    // 10. 读取照片元数据
    public func inspectPhotoMetadata(filePath: String) -> ExifMetadata? {
        guard let fnInspectPhotoMetadata else { return nil }

        guard let cStr = filePath.withCString({ fnInspectPhotoMetadata($0) }) else {
            return nil
        }
        defer { fnFreeString?(cStr) }

        let jsonString = String(cString: cStr)
        guard let data = jsonString.data(using: .utf8),
              let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        return ExifMetadata.parse(from: dict, fallbackPath: filePath)
    }

    // 11. 设置当前运行时语言
    public func setLanguage(_ lang: String) {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnSetLanguage else { return }
        lang.withCString { fn($0) }
    }

    // 12. 获取 Go 端权威插件元数据与多语言职能表述 JSON
    public func getPluginMetasJSON() -> String? {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnGetPluginMetasJSON, let ptr = fn() else { return nil }
        defer { fnFreeString?(ptr) }
        return String(cString: ptr)
    }

    /// 解析并返回强类型的插件元数据列表
    public func getPluginMetas() -> [PluginMetadata] {
        guard let jsonStr = getPluginMetasJSON(),
              let data = jsonStr.data(using: .utf8),
              let list = try? JSONDecoder().decode([PluginMetadata].self, from: data) else {
            return []
        }
        return list
    }

    // 13. 保存偏好设置至 ~/.config/photools/plugins.json
    @discardableResult
    public func savePreferences(optionsJSON: String) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnSavePreferences else { return false }
        return optionsJSON.withCString { ptr in
            fn(ptr) == 0
        }
    }

    // 14. 获取 Go 端全局配置 JSON
    public func getGlobalConfigJSON() -> String? {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnGetGlobalConfigJSON, let ptr = fn() else { return nil }
        defer { fnFreeString?(ptr) }
        return String(cString: ptr)
    }

    // 15. 获取 Go 端权威全局选项配置 JSON
    public func getGlobalOptionSpecsJSON() -> String? {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnGetGlobalOptionSpecsJSON, let ptr = fn() else { return nil }
        defer { fnFreeString?(ptr) }
        return String(cString: ptr)
    }

    // 16. 直接向照片及伴随文件写入/克隆全量 GPS 坐标与溯源指纹
    public func writePhotoGPSMetadata(
        sourcePath: String? = nil,
        targetPath: String,
        latitude: Double,
        longitude: Double,
        altitude: Double = 0.0,
        provenanceJSON: String? = nil,
        sidecarPolicy: String = "smart"
    ) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnWritePhotoGPSMetadata else { return false }

        let invokeWithPointers: (UnsafePointer<CChar>?, UnsafePointer<CChar>) -> Bool = { cSrc, cTgt in
            let res: Int32
            if let prov = provenanceJSON {
                res = prov.withCString { cProv in
                    sidecarPolicy.withCString { cPolicy in
                        fn(cSrc, cTgt, latitude, longitude, altitude, cProv, cPolicy)
                    }
                }
            } else {
                res = sidecarPolicy.withCString { cPolicy in
                    fn(cSrc, cTgt, latitude, longitude, altitude, nil, cPolicy)
                }
            }
            return res == 0
        }

        return targetPath.withCString { cTgt in
            if let src = sourcePath {
                return src.withCString { cSrc in
                    invokeWithPointers(cSrc, cTgt)
                }
            } else {
                return invokeWithPointers(nil, cTgt)
            }
        }
    }

    // 17. 获取 Go 端最新发生的底层错误详细描述
    public func getLastErrorMessage() -> String? {
        lock.lock()
        defer { lock.unlock() }
        guard let fn = fnGetLastErrorMessage, let ptr = fn() else { return nil }
        defer { fnFreeString?(ptr) }
        return String(cString: ptr)
    }

    // 18. 安全退出并回收全部常驻子进程资源
    public func shutdown() {
        lock.lock()
        defer { lock.unlock() }
        fnShutdown?()
    }
}
