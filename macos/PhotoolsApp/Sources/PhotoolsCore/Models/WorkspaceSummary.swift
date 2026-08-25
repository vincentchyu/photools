import Foundation

public struct WorkspaceSummary: Equatable, Sendable {
    public let baseDirectory: String
    public let inboxDirectory: String
    public let gpxDirectory: String
    public let processedDirectory: String
    public let logsDirectory: String
    public let inboxBakDirectory: String
    public let gpxFiles: [String]
    public let assetGroups: [PhotoAssetGroup]
    public let processedFileCount: Int
    public let backupFileCount: Int
    public let logFilePath: String
    public let pendingReportPath: String
    public let pendingReportExists: Bool
    public let pendingReportText: String

    public init(
        baseDirectory: String,
        inboxDirectory: String,
        gpxDirectory: String,
        processedDirectory: String,
        logsDirectory: String,
        inboxBakDirectory: String = "",
        gpxFiles: [String],
        assetGroups: [PhotoAssetGroup],
        processedFileCount: Int,
        backupFileCount: Int = 0,
        logFilePath: String,
        pendingReportPath: String,
        pendingReportExists: Bool,
        pendingReportText: String
    ) {
        self.baseDirectory = baseDirectory
        self.inboxDirectory = inboxDirectory
        self.gpxDirectory = gpxDirectory
        self.processedDirectory = processedDirectory
        self.logsDirectory = logsDirectory
        self.inboxBakDirectory = inboxBakDirectory.isEmpty ? (baseDirectory as NSString).appendingPathComponent("Inbox_bak") : inboxBakDirectory
        self.gpxFiles = gpxFiles
        self.assetGroups = assetGroups
        self.processedFileCount = processedFileCount
        self.backupFileCount = backupFileCount
        self.logFilePath = logFilePath
        self.pendingReportPath = pendingReportPath
        self.pendingReportExists = pendingReportExists
        self.pendingReportText = pendingReportText
    }

    public var readyCount: Int {
        assetGroups.filter { $0.status == .ready }.count
    }

    public var rawPairCount: Int {
        assetGroups.filter { $0.primaryType == .rawPair }.count
    }

    public var rawOnlyCount: Int {
        assetGroups.filter { $0.primaryType == .rawOnly }.count
    }

    public var jpgOnlyCount: Int {
        assetGroups.filter { $0.primaryType == .jpgOnly }.count
    }

    public var companionOnlyCount: Int {
        assetGroups.filter { $0.primaryType == .companionOnly }.count
    }

    public var hasGPX: Bool {
        !gpxFiles.isEmpty
    }

    public var hasBackup: Bool {
        backupFileCount > 0
    }
}
