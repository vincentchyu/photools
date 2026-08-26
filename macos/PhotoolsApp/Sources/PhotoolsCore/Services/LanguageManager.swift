import Combine
import Foundation
import SwiftUI

/// 多语言本地化翻译键
public enum L10nKey: String, CaseIterable, Sendable {
    // App & Window
    case appTitle
    case windowTitle
    case preferencesTitle
    case done
    case cancel
    case close
    case refresh
    case refreshHelp
    case runPipeline
    case runPipelineHelp
    case cancelTask
    case interruptPipeline
    case interruptPipelineHelp
    case chooseDirectory
    case chooseDirHelp
    case chooseBaseDirPrompt
    case openSettings
    case openSettingsHelp

    // Sidebar Sections & Groups
    case groupWorkspace
    case groupRootDirectory
    case groupWorkflow
    case groupTools
    case sectionPipeline
    case sectionInbox
    case sectionProcessed
    case sectionGeodata
    case sectionGpx
    case sectionTestRestore
    case sectionGuide

    // Settings - Tabs
    case tabGeneral
    case tabPlugins

    // Settings - General
    case languageSetting
    case workspaceConfig
    case baseDirectory
    case flatMode
    case flatModeDesc
    case sourceDirectory
    case processedDirectory
    case performanceSection
    case rawExtensions
    case concurrencyWorkers
    case testBackupMode

    // Settings - Plugins
    case pluginDefaults
    case pluginGpxMatch
    case pluginInterpolate
    case pluginGeocode
    case pluginArchive
    case pluginParams
    case geosyncOffset
    case interpolateWindow
    case window15m
    case window30m
    case window1h
    case window2h
    case window4h
    case inPlaceRename
    case allowNoGps

    // Pipeline Dashboard
    case workbenchTitle
    case workbenchSubtitle
    case pipelineTitle
    case pipelineSubtitle
    case priority
    case phase
    case statusReady
    case statusLoading
    case statusProcessing
    case statusCompleted
    case statusSkipped
    case statusFailed
    case executionSummary
    case processedTotal
    case geotaggedCount
    case interpolatedCount
    case reverseGeocodedCount
    case archivedCount
    case pendingCount
    case duration
    case logOutput
    case metricGpx
    case metricInbox
    case metricRawPair
    case metricSingleFile
    case metricArchived
    case advancedOptions
    case advancedOptionsTitle
    case clickToExpand
    case clickToCollapse
    case rawExtensionsPrompt
    case concurrencyWorkersPrompt
    case geosyncOffsetPrompt
    case interpolateWindowPrompt
    case allowNoGpsDesc
    case capGpxTitle
    case capGpxDesc
    case capInterpolateTitle
    case capInterpolateDesc
    case capGeocodeTitle
    case capGeocodeDesc
    case capArchiveTitle
    case capArchiveDesc
    case stageDiscover
    case stageGeotag
    case stageInterpolate
    case stageGeocode
    case stageArchive
    case pipelineRunning
    case realtimeConsole
    case autoScroll
    case copyAllLogs
    case clearLogs
    case clearStatus
    case clearStatusHelp
    case taskCompletedTitle
    case taskErrorTitle
    case viewFullLog
    case pendingReport
    case waitingForTask

    // Asset List & Filter
    case filterAll
    case filterRawPair
    case filterRawOnly
    case filterJpgOnly
    case filterCompanionOnly
    case searchPlaceholder
    case emptyInbox
    case emptyFiltered
    case emptyGpxFiles
    case processedTotalFiles
    case openInFinder
    case openFile
    case showInFinder
    case openJpgPreview
    case openRawOriginal
    case moveToTrashDelete

    // Detail View & Inspector
    case noPhotoSelectedTitle
    case noPhotoSelectedSubtitle
    case processedArchiveInfo
    case processedArchiveCount
    case gpxMatchOverview
    case gpxFilesCount
    case gpxPipelineDesc
    case testRestoreMechTitle
    case testRestoreMechDesc
    case guideOverviewTitle
    case guideOverviewDesc
    case assetSummary
    case captureTime
    case primaryFile
    case companionFiles
    case cameraAndExposure
    case cameraModel
    case lensModel
    case exposureSettings
    case shutterSpeed
    case aperture
    case isoSensitivity
    case focalLength
    case locationAndGeocoding
    case gpsCoordinates
    case latitudeLabel
    case longitudeLabel
    case altitudeLabel
    case offlineGeocodeResult
    case instantLookup
    case noGpsData
    case copyCoordinates
    case previewGeocodeBtn
    case lookupInProgress
    case rawExifTagTree
    case searchTagsPlaceholder
    case pendingDiagnosticsSection
    case openLargeJpg
    case openOriginalImage
    case locateInFinderHelp
    case shootingParamsTitle
    case readingExif
    case noExifRead
    case gpsAndGeocodeTitle
    case geocodedWrittenBadge
    case hasGpsNotGeocodedBadge
    case noGpsBadge
    case coordinatesLabel
    case writtenLabel
    case previewPrefix
    case geocodeNotice
    case clickToPreviewGeocode
    case noGpsNotice
    case companionListTitle
    case primaryDecisionSource
    case openThisFile
    case pendingReasonTitle
    case allExifTagsTitle
    case filterTagsPlaceholder
    case noMatchingTags

    // Asset Types & Status
    case typeRawPairTitle
    case typeRawOnlyTitle
    case typeJpgOnlyTitle
    case typeCompanionOnlyTitle
    case statusReadyTitle
    case statusCompanionOnlyTitle
    case statusReadySuggestion
    case statusCompanionOnlySuggestion

    // Geodata Manager
    case geodataTitle
    case geodataSubtitle
    case globalGeodataTitle
    case globalGeodataDesc
    case geodataTerminalPolicyNotice
    case geodataTerminalPolicyDesc
    case continentListTitle
    case refreshStatus
    case copyInstallAllCmd
    case copyTerminalCmd
    case copiedToClipboard
    case checkingLocalGeodata
    case noGeodataPacks
    case uninstallCmdCopied
    case installCmdCopied
    case continentName
    case statusInstalled
    case statusNotInstalled
    case downloadInstall
    case removeGeodata
    case testLookup
    case testCoordinates
    case testResultTitle
    case queryTime
    case geoEngineTitle
    case geoEngineDesc
    case latShort
    case lonShort
    case altShort
    case testBtn
    case spatialAnalysisToggle
    case spatialAnalysisHelp
    case presetsLabel
    case topologicalResultTitle
    case countryRegion
    case stateProvince
    case cityPrefecture
    case districtCounty
    case scenicPoi
    case formattedIptcTag
    case nearestPointDist
    case timezoneLabel

    // Test & Restore
    case backupTitle
    case backupSubtitle
    case testRestoreConsoleTitle
    case testRestoreConsoleDesc
    case directoryStatusTitle
    case inboxSourcePhotos
    case inboxBakSnapshot
    case processedArchiveResult
    case backupStep1Title
    case backupStep1Desc
    case backupNowBtn
    case inboxEmptyWarning
    case restoreStep2Title
    case restoreStep2Desc
    case restoreNowBtn
    case noBackupAvailableWarning
    case createBackupBtn
    case restoreBackupBtn
    case cleanProcessedToggle
    case backupSuccessMsg
    case restoreSuccessMsg
    case snapshotStatusTitle
    case backupExists
    case backupNotExists
    case backupFileCount
}

/// 全局多语言本地化管理器
@MainActor
public final class LanguageManager: ObservableObject {
    public static let shared = LanguageManager()

    private let userDefaultsKey = "photools_app_language"

    @Published public var currentLanguage: AppLanguage {
        didSet {
            UserDefaults.standard.set(currentLanguage.rawValue, forKey: userDefaultsKey)
        }
    }

    public init() {
        if let saved = UserDefaults.standard.string(forKey: userDefaultsKey),
           let lang = AppLanguage(rawValue: saved) {
            self.currentLanguage = lang
        } else {
            self.currentLanguage = .system
        }
    }

    /// 设置语言
    public func setLanguage(_ lang: AppLanguage) {
        self.currentLanguage = lang
    }

    /// 翻译指定键
    public func text(_ key: L10nKey) -> String {
        let isZh = currentLanguage.isChinese
        return isZh ? chineseDictionary[key] ?? key.rawValue : englishDictionary[key] ?? key.rawValue
    }

    // MARK: - 中文字典 (Simplified Chinese)
    private let chineseDictionary: [L10nKey: String] = [
        .appTitle: "photools 摄影资产处理工作台",
        .windowTitle: "photools 工作台",
        .preferencesTitle: "偏好设置",
        .done: "完成",
        .cancel: "取消",
        .close: "关闭",
        .refresh: "刷新",
        .refreshHelp: "重新扫描工作目录 (⌘R)",
        .runPipeline: "执行流水线",
        .runPipelineHelp: "执行 photools 自动化流水线 (⌘↩)",
        .cancelTask: "终止任务",
        .interruptPipeline: "中断流水线",
        .interruptPipelineHelp: "中断正在执行的任务 (⌘.)",
        .chooseDirectory: "选择目录...",
        .chooseDirHelp: "选择 GPS 基础工作目录 (⌘O)",
        .chooseBaseDirPrompt: "选择工作目录",
        .openSettings: "设置",
        .openSettingsHelp: "打开偏好设置 (⌘,)",

        // Sidebar
        .groupWorkspace: "工作区",
        .groupRootDirectory: "根目录",
        .groupWorkflow: "工作流",
        .groupTools: "工具与资源",
        .sectionPipeline: "处理工作台",
        .sectionInbox: "待处理照片",
        .sectionProcessed: "归档目录",
        .sectionGeodata: "离线地理库",
        .sectionGpx: "GPX 轨迹",
        .sectionTestRestore: "测试快照还原",
        .sectionGuide: "使用指南",

        // Settings Tabs
        .tabGeneral: "通用与路径",
        .tabPlugins: "插件与策略",

        // Settings General
        .languageSetting: "界面语言 (Language)",
        .workspaceConfig: "工作区目录配置",
        .baseDirectory: "基础工作根目录 (Base Directory)",
        .flatMode: "扁平/直接目录模式 (在指定目录下直接扫描并保存)",
        .flatModeDesc: "忽略 Inbox/Processed 分层，直接在源目录下原地识别与规范化处理",
        .sourceDirectory: "待处理照片源目录 (默认 Inbox)",
        .processedDirectory: "规范归档目录 (默认 Processed)",
        .performanceSection: "性能与格式策略",
        .rawExtensions: "识别的 RAW 扩展名列表 (逗号分隔)",
        .concurrencyWorkers: "并发工作协程数 (Workers)",
        .testBackupMode: "测试快照备份 (执行前自动备份至 Inbox_bak)",

        // Settings Plugins
        .pluginDefaults: "核心能力插件默认启用状态",
        .pluginGpxMatch: "GPX 轨迹精准匹配与 GPS 写入",
        .pluginInterpolate: "GPS 智能邻近推断与时间插值",
        .pluginGeocode: "离线高精逆地理编码与地名写入",
        .pluginArchive: "拍摄日期规范重命名与安全归档",
        .pluginParams: "能力插件细项参数调优",
        .geosyncOffset: "相机与 GPS 时钟偏差补偿 (geosync，如 +00:00:05)",
        .interpolateWindow: "GPS 智能插值推算最大时间窗口",
        .window15m: "15 分钟 (默认推荐)",
        .window30m: "30 分钟",
        .window1h: "1 小时",
        .window2h: "2 小时",
        .window4h: "4 小时",
        .inPlaceRename: "原地规范重命名 (不创建 YYYY/MMDD/ 子目录)",
        .allowNoGps: "容错软降级策略 (无 GPS 照片跳过地名写入，安全继续归档)",

        // Dashboard
        .workbenchTitle: "摄影处理工作台",
        .workbenchSubtitle: "一键全自动轨迹匹配、智能推算、中文地名标记与规范归档",
        .pipelineTitle: "多阶段屏障摄影自动化流水线",
        .pipelineSubtitle: "支持 GPX 轨迹对齐、GPS 时间插值推算、离线 3D KD-Tree 逆地理地名写入与拍摄日原子归档",
        .priority: "优先级",
        .phase: "阶段",
        .statusReady: "就绪",
        .statusLoading: "装载中",
        .statusProcessing: "执行中",
        .statusCompleted: "已完成",
        .statusSkipped: "已跳过",
        .statusFailed: "失败",
        .executionSummary: "执行结果与统计汇总",
        .processedTotal: "总处理照片组",
        .geotaggedCount: "GPX 轨迹匹配",
        .interpolatedCount: "GPS 智能插值",
        .reverseGeocodedCount: "逆地理地名写入",
        .archivedCount: "安全归档",
        .pendingCount: "待处理异常",
        .duration: "耗时",
        .logOutput: "实时流式执行日志",
        .metricGpx: "GPX 轨迹",
        .metricInbox: "待处理资产",
        .metricRawPair: "RAW+JPG",
        .metricSingleFile: "独立单文件",
        .metricArchived: "已归档照片",
        .advancedOptions: "高级选项与策略",
        .advancedOptionsTitle: "高级选项与并发设置",
        .clickToExpand: "点击展开",
        .clickToCollapse: "点击收起",
        .rawExtensionsPrompt: "RAW 文件扩展名列表:",
        .concurrencyWorkersPrompt: "并发工作协程数 (workers):",
        .geosyncOffsetPrompt: "时间偏移补偿 (geosync):",
        .interpolateWindowPrompt: "推算最大时间窗口:",
        .allowNoGpsDesc: "无 GPS 照片跳过地名写入，安全继续归档",
        .capGpxTitle: "GPX 轨迹精准匹配与 GPS 写入",
        .capGpxDesc: "从 GPX 目录读取轨迹，依据拍摄时间为 RAW 写入经纬度并同步至 JPG/XMP。",
        .capInterpolateTitle: "GPS 智能邻近推断与时间插值",
        .capInterpolateDesc: "根据同批次前后邻近照片时间与机位权重，自动推算补全无轨迹照片的位置。",
        .capGeocodeTitle: "离线高精逆地理编码与地名写入",
        .capGeocodeDesc: "基于内置离线地理数据库，毫秒级反查国家、省份、城市、区县与名胜 POI 中文元数据。",
        .capArchiveTitle: "拍摄日期规范重命名与安全归档",
        .capArchiveDesc: "提取原始 EXIF 拍摄日期，规范重命名并安全移动归档至 Processed/YYYY/MMDD/ 目录。",
        .stageDiscover: "扫描",
        .stageGeotag: "轨迹匹配",
        .stageInterpolate: "GPS推算",
        .stageGeocode: "中文地名",
        .stageArchive: "日期归档",
        .pipelineRunning: "流水线执行中...",
        .realtimeConsole: "实时执行控制台",
        .autoScroll: "自动滚屏",
        .copyAllLogs: "复制全部日志",
        .clearLogs: "清空当前日志",
        .clearStatus: "清除状态",
        .clearStatusHelp: "清除当前任务状态并重置看板",
        .taskCompletedTitle: "本次流水线执行完成",
        .taskErrorTitle: "执行遇到错误",
        .viewFullLog: "查看完整日志",
        .pendingReport: "待补报告",
        .waitingForTask: "等待任务启动...\n",

        // Asset List & Filter
        .filterAll: "全部",
        .filterRawPair: "RAW+JPG",
        .filterRawOnly: "单 RAW",
        .filterJpgOnly: "单 JPG",
        .filterCompanionOnly: "仅伴随",
        .searchPlaceholder: "搜索照片 (文件名、机型、地名)...",
        .emptyInbox: "Inbox 待处理目录为空",
        .emptyFiltered: "没有符合筛选条件的照片",
        .emptyGpxFiles: "未找到 GPX 轨迹文件",
        .processedTotalFiles: "已归档照片总数",
        .openInFinder: "在访达 (Finder) 中打开",
        .openFile: "打开文件",
        .showInFinder: "在访达中显示",
        .openJpgPreview: "打开 JPG 预览",
        .openRawOriginal: "打开 RAW 原片",
        .moveToTrashDelete: "移到废纸篓 / 删除照片组",

        // Detail View & Inspector
        .noPhotoSelectedTitle: "未选择照片",
        .noPhotoSelectedSubtitle: "在中间列表点击任意照片，即可在此查看完整的 ExifTool 拍摄参数与元数据。",
        .processedArchiveInfo: "归档目录信息",
        .processedArchiveCount: "已成功归档照片数",
        .gpxMatchOverview: "GPX 轨迹匹配概览",
        .gpxFilesCount: "已发现轨迹文件数",
        .gpxPipelineDesc: "流水线将自动提取 GPX 目录下的全部轨迹，并按拍摄时间为照片进行微秒级精准经纬度匹配。",
        .testRestoreMechTitle: "快照与还原机制",
        .testRestoreMechDesc: "在开启测试备份模式执行流水线时，系统会在处理前将 Inbox 中的全部原始照片原样备份至 Inbox_bak，以便随时进行一键恢复测试。",
        .guideOverviewTitle: "文档与设计指引",
        .guideOverviewDesc: "请在中间内容区查阅完整的中英文架构与用户指南。",
        .assetSummary: "资产单元摘要",
        .captureTime: "拍摄时间",
        .primaryFile: "主决策文件",
        .companionFiles: "伴随文件",
        .cameraAndExposure: "相机与曝光参数",
        .cameraModel: "相机机型",
        .lensModel: "镜头型号",
        .exposureSettings: "曝光四要素",
        .shutterSpeed: "快门速度",
        .aperture: "光圈大小",
        .isoSensitivity: "感光度",
        .focalLength: "焦距",
        .locationAndGeocoding: "地理位置与逆地理地名",
        .gpsCoordinates: "GPS 经纬度坐标",
        .latitudeLabel: "纬度",
        .longitudeLabel: "经度",
        .altitudeLabel: "海拔",
        .offlineGeocodeResult: "离线逆地理地名解析",
        .instantLookup: "即时反查预览",
        .noGpsData: "无 GPS 坐标数据",
        .copyCoordinates: "复制经纬度",
        .previewGeocodeBtn: "即时反查地名",
        .lookupInProgress: "正在反查...",
        .rawExifTagTree: "原始 ExifTool 标签树",
        .searchTagsPlaceholder: "搜索 EXIF / IPTC / XMP 标签...",
        .pendingDiagnosticsSection: "待处理与异常原因诊断",
        .openLargeJpg: "打开大图 (JPG)",
        .openOriginalImage: "打开原图",
        .locateInFinderHelp: "在访达中定位高亮此照片",
        .shootingParamsTitle: "拍摄参数",
        .readingExif: "正在读取拍摄参数...",
        .noExifRead: "未读取到拍摄参数",
        .gpsAndGeocodeTitle: "GPS 坐标与中文地名",
        .geocodedWrittenBadge: "已写入地名",
        .hasGpsNotGeocodedBadge: "已有 GPS · 未写入地名",
        .noGpsBadge: "无 GPS 坐标",
        .coordinatesLabel: "坐标:",
        .writtenLabel: "已写入:",
        .previewPrefix: "反查预览:",
        .geocodeNotice: "提示：执行流水线时将自动把此规范中文地名写入 IPTC/XMP。",
        .clickToPreviewGeocode: "点击即时预览逆地理中文地名",
        .noGpsNotice: "该照片尚未写入 GPS 经纬度。执行流水线时将根据 GPX 轨迹或同批次邻近机位智能推算并写入地名。",
        .companionListTitle: "配套文件清单",
        .primaryDecisionSource: "主决策源",
        .openThisFile: "打开此文件",
        .pendingReasonTitle: "待处理原因提示",
        .allExifTagsTitle: "全部 ExifTool 标签",
        .filterTagsPlaceholder: "过滤标签名或值...",
        .noMatchingTags: "未匹配到标签",

        // Asset Types & Status
        .typeRawPairTitle: "RAW + JPG 配套",
        .typeRawOnlyTitle: "单 RAW 主文件",
        .typeJpgOnlyTitle: "单 JPG 主文件",
        .typeCompanionOnlyTitle: "仅伴随文件",
        .statusReadyTitle: "就绪可处理",
        .statusCompanionOnlyTitle: "仅伴随文件",
        .statusReadySuggestion: "具备完整主文件，可正常执行 GPX 匹配、GPS 插值推算、逆地理编码与规范归档。",
        .statusCompanionOnlySuggestion: "缺少 RAW/JPG 主文件，暂不进入流水线处理流程。",

        // Geodata
        .geodataTitle: "全球离线高精逆地理数据包",
        .geodataSubtitle: "基于 GeoNames 与 3D KD-Tree 空间索引，毫秒级解析五级规范中文地名",
        .globalGeodataTitle: "全球离线地理数据包",
        .globalGeodataDesc: "内置中国精细地名库及各大洲离线点位索引，断网环境下也能精准写入中文规范地名。",
        .geodataTerminalPolicyNotice: "终端操作指引",
        .geodataTerminalPolicyDesc: "为确保离线地理数据库的准确性与数据完整性校验，数据包的下载与构建需由使用者在终端（CLI）中显式操作。",
        .continentListTitle: "各大洲数据包状态",
        .refreshStatus: "刷新状态",
        .copyInstallAllCmd: "复制全量安装命令",
        .copyTerminalCmd: "复制命令",
        .copiedToClipboard: "已复制到剪贴板",
        .checkingLocalGeodata: "正在查询本地离线地理数据包状态...",
        .noGeodataPacks: "暂无数据包信息，请点击右上角刷新",
        .uninstallCmdCopied: "复制卸载命令",
        .installCmdCopied: "复制安装命令",
        .continentName: "大洲 / 区域",
        .statusInstalled: "已安装",
        .statusNotInstalled: "未安装",
        .downloadInstall: "终端安装",
        .removeGeodata: "终端卸载",
        .testLookup: "经纬度即时反查测试",
        .testCoordinates: "测试坐标",
        .testResultTitle: "反查地名结果",
        .queryTime: "查询耗时",
        .geoEngineTitle: "离线地理反查引擎",
        .geoEngineDesc: "内置基于 KD-Tree 空间索引的全球离线地名库，支持毫秒级推算 GPS 坐标所在的省份、城市、区县与 POI 名胜景点。",
        .latShort: "纬度",
        .lonShort: "经度",
        .altShort: "海拔",
        .testBtn: "测试",
        .spatialAnalysisToggle: "空间分析",
        .spatialAnalysisHelp: "开启后实时输出 3D 笛卡尔坐标投影、KD-Tree 剪枝遍历与 Top-K 最近邻拓扑分析过程",
        .presetsLabel: "地标预设:",
        .topologicalResultTitle: "最近邻拓扑反查结果",
        .countryRegion: "国家 / 地区",
        .stateProvince: "一级行政区 (省/州)",
        .cityPrefecture: "二级行政区 (地级市)",
        .districtCounty: "三级行政区 (区/县)",
        .scenicPoi: "名胜景点 / POI",
        .formattedIptcTag: "IPTC/XMP 规范中文标签",
        .nearestPointDist: "最近邻点位",
        .timezoneLabel: "时区",

        // Test & Restore
        .backupTitle: "测试快照与环境还原",
        .backupSubtitle: "一键全量快照备份 Inbox 待处理照片，测试完毕后秒级复原",
        .testRestoreConsoleTitle: "测试快照与一键备份/还原控制台",
        .testRestoreConsoleDesc: "支持在批量流水线处理测试前手动快照备份 Inbox 原始文件至 Inbox_bak，并可在测试完成后一键还原测试环境，安全无忧。",
        .directoryStatusTitle: "目录状态",
        .inboxSourcePhotos: "待处理源照片 (Inbox)",
        .inboxBakSnapshot: "Inbox_bak 快照备份",
        .processedArchiveResult: "Processed 归档结果",
        .backupStep1Title: "1. 一键创建快照备份",
        .backupStep1Desc: "将当前待处理照片（Inbox）全量快照备份至 Inbox_bak 目录。在执行任何修改或测试前，可随时备份。",
        .backupNowBtn: "立即快照备份当前照片 (Inbox → Inbox_bak)",
        .inboxEmptyWarning: "⚠️ 当前待处理目录为空",
        .restoreStep2Title: "2. 一键从快照还原环境",
        .restoreStep2Desc: "从 Inbox_bak 备份目录完整还原原始测试照片至 Inbox。可选选择是否同时清空 Processed 目录下的生成结果。",
        .restoreNowBtn: "立即从快照还原 (Inbox_bak → Inbox)",
        .noBackupAvailableWarning: "⚠️ 尚未发现 Inbox_bak 快照备份",
        .createBackupBtn: "立即创建全量快照备份",
        .restoreBackupBtn: "从快照还原至 Inbox",
        .cleanProcessedToggle: "还原时同时清空 Processed 目录",
        .backupSuccessMsg: "快照备份创建成功",
        .restoreSuccessMsg: "环境已成功从快照还原",
        .snapshotStatusTitle: "快照备份状态",
        .backupExists: "已存在备份",
        .backupNotExists: "无快照备份",
        .backupFileCount: "备份照片数"
    ]

    // MARK: - 英文字典 (English)
    private let englishDictionary: [L10nKey: String] = [
        .appTitle: "photools - Photo Processing Workbench",
        .windowTitle: "photools Workbench",
        .preferencesTitle: "Preferences",
        .done: "Done",
        .cancel: "Cancel",
        .close: "Close",
        .refresh: "Refresh",
        .refreshHelp: "Rescan Workspace (⌘R)",
        .runPipeline: "Run Pipeline",
        .runPipelineHelp: "Execute photools Pipeline (⌘↩)",
        .cancelTask: "Cancel Task",
        .interruptPipeline: "Interrupt Pipeline",
        .interruptPipelineHelp: "Interrupt Running Task (⌘.)",
        .chooseDirectory: "Choose Directory...",
        .chooseDirHelp: "Select Base Workspace Directory (⌘O)",
        .chooseBaseDirPrompt: "Select Workspace Directory",
        .openSettings: "Settings",
        .openSettingsHelp: "Open Preferences (⌘,)",

        // Sidebar
        .groupWorkspace: "Workspace",
        .groupRootDirectory: "Root Directory",
        .groupWorkflow: "Workflow",
        .groupTools: "Tools & Resources",
        .sectionPipeline: "Pipeline Dashboard",
        .sectionInbox: "Inbox Photos",
        .sectionProcessed: "Processed Archive",
        .sectionGeodata: "Offline Geodata",
        .sectionGpx: "GPX Tracks",
        .sectionTestRestore: "Test & Restore",
        .sectionGuide: "User Guides",

        // Settings Tabs
        .tabGeneral: "General & Paths",
        .tabPlugins: "Plugins & Policies",

        // Settings General
        .languageSetting: "Language (界面语言)",
        .workspaceConfig: "Workspace Directories",
        .baseDirectory: "Base Workspace Directory",
        .flatMode: "Flat / In-Place Mode (Scan & Process in Place)",
        .flatModeDesc: "Ignore Inbox/Processed hierarchy; scan and normalize in source directory directly",
        .sourceDirectory: "Source Directory (Default Inbox)",
        .processedDirectory: "Processed Directory (Default Processed)",
        .performanceSection: "Performance & Formats",
        .rawExtensions: "Recognized RAW Extensions (Comma Separated)",
        .concurrencyWorkers: "Worker Concurrency",
        .testBackupMode: "Test Snapshot Backup (Auto-backup to Inbox_bak)",

        // Settings Plugins
        .pluginDefaults: "Capability Plugin Defaults",
        .pluginGpxMatch: "GPX Track Matching & GPS Tagging",
        .pluginInterpolate: "GPS Intelligent Time-Weighted Interpolation",
        .pluginGeocode: "Offline 3D KD-Tree Reverse Geocoding",
        .pluginArchive: "Date-Based Normalization & Archive",
        .pluginParams: "Plugin Parameter Tuning",
        .geosyncOffset: "geosync Time Offset (e.g. 0, +00:00:05)",
        .interpolateWindow: "Interpolation Time Window",
        .window15m: "15 Minutes (Default)",
        .window30m: "30 Minutes",
        .window1h: "1 Hour",
        .window2h: "2 Hours",
        .window4h: "4 Hours",
        .inPlaceRename: "In-Place Rename (No YYYY/MMDD/ Subdirectories)",
        .allowNoGps: "Fault-Tolerant Soft-Degradation (Safe Archive for Non-GPS)",

        // Dashboard
        .workbenchTitle: "Photo Processing Workbench",
        .workbenchSubtitle: "Automated track matching, smart interpolation, offline geocoding & date archive",
        .pipelineTitle: "Multi-Phase Barrier Photo Pipeline",
        .pipelineSubtitle: "GPX alignment, GPS time interpolation, offline 3D KD-Tree geocoding & date-based atomic archiving",
        .priority: "Priority",
        .phase: "Phase",
        .statusReady: "Ready",
        .statusLoading: "Loading",
        .statusProcessing: "Running",
        .statusCompleted: "Completed",
        .statusSkipped: "Skipped",
        .statusFailed: "Failed",
        .executionSummary: "Execution Summary & Audit",
        .processedTotal: "Total Processed Groups",
        .geotaggedCount: "GPX Track Matches",
        .interpolatedCount: "GPS Interpolations",
        .reverseGeocodedCount: "Geocoded Tags",
        .archivedCount: "Safely Archived",
        .pendingCount: "Pending / Outliers",
        .duration: "Duration",
        .logOutput: "Real-Time Streaming Log",
        .metricGpx: "GPX Tracks",
        .metricInbox: "Inbox Assets",
        .metricRawPair: "RAW+JPG Pairs",
        .metricSingleFile: "Single Files",
        .metricArchived: "Archived Photos",
        .advancedOptions: "Advanced Options & Policies",
        .advancedOptionsTitle: "Advanced Options & Concurrency",
        .clickToExpand: "Click to Expand",
        .clickToCollapse: "Click to Collapse",
        .rawExtensionsPrompt: "RAW Extensions:",
        .concurrencyWorkersPrompt: "Worker Concurrency:",
        .geosyncOffsetPrompt: "Time Offset (geosync):",
        .interpolateWindowPrompt: "Max Interpolation Window:",
        .allowNoGpsDesc: "Skip geocoding for non-GPS photos and safely archive",
        .capGpxTitle: "GPX Track Matching & GPS Tagging",
        .capGpxDesc: "Read GPX tracks and write coordinates to RAW and companion files based on capture time.",
        .capInterpolateTitle: "GPS Intelligent Time-Weighted Interpolation",
        .capInterpolateDesc: "Intelligently interpolate missing GPS coordinates based on adjacent photo timestamps.",
        .capGeocodeTitle: "Offline 3D KD-Tree Reverse Geocoding",
        .capGeocodeDesc: "Sub-millisecond reverse geocoding into Country, State, City, District & POI metadata.",
        .capArchiveTitle: "Date-Based Normalization & Safe Archive",
        .capArchiveDesc: "Extract capture dates, safely rename and atomically move files to Processed/YYYY/MMDD/.",
        .stageDiscover: "Scan",
        .stageGeotag: "Track Match",
        .stageInterpolate: "GPS Interpolate",
        .stageGeocode: "Geocode",
        .stageArchive: "Date Archive",
        .pipelineRunning: "Pipeline Running...",
        .realtimeConsole: "Real-Time Execution Console",
        .autoScroll: "Auto-Scroll",
        .copyAllLogs: "Copy All Logs",
        .clearLogs: "Clear Current Log",
        .clearStatus: "Clear Status",
        .clearStatusHelp: "Reset current task status and pipeline board",
        .taskCompletedTitle: "Pipeline Completed Successfully",
        .taskErrorTitle: "Pipeline Execution Error",
        .viewFullLog: "View Full Log",
        .pendingReport: "Pending Report",
        .waitingForTask: "Waiting for pipeline task to start...\n",

        // Asset List & Filter
        .filterAll: "All",
        .filterRawPair: "RAW+JPG",
        .filterRawOnly: "RAW Only",
        .filterJpgOnly: "JPG Only",
        .filterCompanionOnly: "Companion",
        .searchPlaceholder: "Search photos (name, camera, place)...",
        .emptyInbox: "Inbox Directory is Empty",
        .emptyFiltered: "No photos matching current filter",
        .emptyGpxFiles: "No GPX Track Files Found",
        .processedTotalFiles: "Total Archived Photos",
        .openInFinder: "Open in Finder",
        .openFile: "Open File",
        .showInFinder: "Show in Finder",
        .openJpgPreview: "Open JPG Preview",
        .openRawOriginal: "Open RAW Original",
        .moveToTrashDelete: "Move to Trash / Delete Asset",

        // Detail View & Inspector
        .noPhotoSelectedTitle: "No Photo Selected",
        .noPhotoSelectedSubtitle: "Select any photo from the list to inspect full ExifTool parameters and metadata.",
        .processedArchiveInfo: "Archive Directory Information",
        .processedArchiveCount: "Archived Photo Groups",
        .gpxMatchOverview: "GPX Track Matching Overview",
        .gpxFilesCount: "Discovered GPX Tracks",
        .gpxPipelineDesc: "The pipeline automatically scans the GPX directory and performs microsecond-level GPS geotagging based on capture timestamps.",
        .testRestoreMechTitle: "Snapshot & Restore Safeguard",
        .testRestoreMechDesc: "When Test Backup mode is enabled, all raw assets in Inbox are backed up to Inbox_bak before execution for 1-click restore.",
        .guideOverviewTitle: "Documentation & Architecture Guides",
        .guideOverviewDesc: "Read full English and Chinese architecture documentation and user manual.",
        .assetSummary: "Asset Summary",
        .captureTime: "Capture Time",
        .primaryFile: "Primary File",
        .companionFiles: "Companion Files",
        .cameraAndExposure: "Camera & Exposure",
        .cameraModel: "Camera Model",
        .lensModel: "Lens Model",
        .exposureSettings: "Exposure 4-Elements",
        .shutterSpeed: "Shutter Speed",
        .aperture: "Aperture",
        .isoSensitivity: "ISO",
        .focalLength: "Focal Length",
        .locationAndGeocoding: "Geotag & Reverse Geocoding",
        .gpsCoordinates: "GPS Coordinates",
        .latitudeLabel: "Latitude",
        .longitudeLabel: "Longitude",
        .altitudeLabel: "Altitude",
        .offlineGeocodeResult: "Offline Geocoding Resolution",
        .instantLookup: "Instant Lookup Preview",
        .noGpsData: "No GPS Coordinates",
        .copyCoordinates: "Copy Coordinates",
        .previewGeocodeBtn: "Instant Geocode Lookup",
        .lookupInProgress: "Looking up...",
        .rawExifTagTree: "Raw ExifTool Tag Tree",
        .searchTagsPlaceholder: "Search EXIF / IPTC / XMP tags...",
        .pendingDiagnosticsSection: "Pending Diagnostics & Outliers",
        .openLargeJpg: "Open Large Image (JPG)",
        .openOriginalImage: "Open Original Image",
        .locateInFinderHelp: "Locate and highlight in Finder",
        .shootingParamsTitle: "Shooting Parameters",
        .readingExif: "Reading shooting parameters...",
        .noExifRead: "No shooting metadata found",
        .gpsAndGeocodeTitle: "GPS Coordinates & Geocoded Place",
        .geocodedWrittenBadge: "Geotagged & Named",
        .hasGpsNotGeocodedBadge: "Has GPS · Unnamed",
        .noGpsBadge: "No GPS Data",
        .coordinatesLabel: "Coordinates:",
        .writtenLabel: "Written Tag:",
        .previewPrefix: "Preview:",
        .geocodeNotice: "Notice: Running pipeline will write standard Chinese place names into IPTC/XMP tags.",
        .clickToPreviewGeocode: "Click to preview offline reverse geocoding result",
        .noGpsNotice: "This asset does not have GPS coordinates. Running pipeline will interpolate or geotag from GPX.",
        .companionListTitle: "Companion Files List",
        .primaryDecisionSource: "Primary Decision Source",
        .openThisFile: "Open File",
        .pendingReasonTitle: "Pending Reason Diagnostic",
        .allExifTagsTitle: "All ExifTool Tags",
        .filterTagsPlaceholder: "Filter tag names or values...",
        .noMatchingTags: "No matching tags",

        // Asset Types & Status
        .typeRawPairTitle: "RAW + JPG Pair",
        .typeRawOnlyTitle: "Single RAW File",
        .typeJpgOnlyTitle: "Single JPG File",
        .typeCompanionOnlyTitle: "Companion Only",
        .statusReadyTitle: "Ready to Process",
        .statusCompanionOnlyTitle: "Companion Only",
        .statusReadySuggestion: "Complete primary files available for GPX geotagging, interpolation, reverse geocoding and archiving.",
        .statusCompanionOnlySuggestion: "Missing primary RAW/JPG file; temporarily excluded from pipeline execution.",

        // Geodata
        .geodataTitle: "Offline Global High-Precision Geodata Packs",
        .geodataSubtitle: "Powered by GeoNames & 3D KD-Tree spatial indexing for sub-millisecond 5-level geocoding",
        .globalGeodataTitle: "Global Offline Geodata Packs",
        .globalGeodataDesc: "Built-in China fine-grained places and continental offline indexes for precision geocoding completely offline.",
        .geodataTerminalPolicyNotice: "Terminal Operation Policy",
        .geodataTerminalPolicyDesc: "To ensure database integrity and offline dataset validation, pack downloads must be triggered via CLI terminal.",
        .continentListTitle: "Continental Packs Status",
        .refreshStatus: "Refresh Status",
        .copyInstallAllCmd: "Copy Install All Command",
        .copyTerminalCmd: "Copy Command",
        .copiedToClipboard: "Copied to Clipboard",
        .checkingLocalGeodata: "Querying local geodata pack status...",
        .noGeodataPacks: "No geodata packs found. Click Refresh above.",
        .uninstallCmdCopied: "Uninstall Command Copied",
        .installCmdCopied: "Install Command Copied",
        .continentName: "Continent / Region",
        .statusInstalled: "Installed",
        .statusNotInstalled: "Not Installed",
        .downloadInstall: "Terminal Install",
        .removeGeodata: "Terminal Remove",
        .testLookup: "Instant Geocoordinate Lookup Test",
        .testCoordinates: "Test Coordinates",
        .testResultTitle: "Reverse Geocode Result",
        .queryTime: "Query Duration",
        .geoEngineTitle: "Offline Geocoding Engine",
        .geoEngineDesc: "Built-in global KD-Tree spatial index for sub-millisecond country, province, city, district and POI resolution.",
        .latShort: "Lat",
        .lonShort: "Lon",
        .altShort: "Alt",
        .testBtn: "Test",
        .spatialAnalysisToggle: "Spatial Analysis",
        .spatialAnalysisHelp: "Enable live 3D Cartesian coordinates, KD-Tree branch pruning and Top-K nearest neighbor topological debug logs",
        .presetsLabel: "Landmark Presets:",
        .topologicalResultTitle: "Topological Nearest Neighbor Results",
        .countryRegion: "Country / Region",
        .stateProvince: "State / Province (Admin 1)",
        .cityPrefecture: "City / Prefecture (Admin 2)",
        .districtCounty: "District / County (Admin 3)",
        .scenicPoi: "Scenic POI / Landmark",
        .formattedIptcTag: "IPTC / XMP Formatted Place Tag",
        .nearestPointDist: "Nearest Point Distance",
        .timezoneLabel: "Timezone",

        // Test & Restore
        .backupTitle: "Test Snapshot & Environment Restore",
        .backupSubtitle: "1-click snapshot backup of Inbox photos with instant restoration",
        .testRestoreConsoleTitle: "Test Snapshot & 1-Click Backup / Restore Console",
        .testRestoreConsoleDesc: "Take a full snapshot of Inbox before running pipeline tests and restore raw files anytime.",
        .directoryStatusTitle: "Directory Status",
        .inboxSourcePhotos: "Inbox Source Photos",
        .inboxBakSnapshot: "Inbox_bak Snapshot",
        .processedArchiveResult: "Processed Archive Result",
        .backupStep1Title: "1. Create Full Snapshot",
        .backupStep1Desc: "Take an atomic snapshot of current Inbox photos into Inbox_bak before running tests.",
        .backupNowBtn: "Snapshot Backup Now (Inbox → Inbox_bak)",
        .inboxEmptyWarning: "⚠️ Inbox source directory is empty",
        .restoreStep2Title: "2. Restore Environment from Snapshot",
        .restoreStep2Desc: "Restore original test photos from Inbox_bak back to Inbox. Optionally clean Processed folder.",
        .restoreNowBtn: "Restore from Snapshot Now (Inbox_bak → Inbox)",
        .noBackupAvailableWarning: "⚠️ No Inbox_bak snapshot found",
        .createBackupBtn: "Create Snapshot Backup Now",
        .restoreBackupBtn: "Restore from Snapshot to Inbox",
        .cleanProcessedToggle: "Clean Processed directory upon restore",
        .backupSuccessMsg: "Snapshot backup created successfully",
        .restoreSuccessMsg: "Environment restored from snapshot successfully",
        .snapshotStatusTitle: "Snapshot Status",
        .backupExists: "Snapshot Exists",
        .backupNotExists: "No Snapshot Backup",
        .backupFileCount: "Backed Up Photos"
    ]
}
