import Foundation

public enum WorkspaceScannerError: Error, LocalizedError {
    case missingBaseDirectory(String)

    public var errorDescription: String? {
        switch self {
        case .missingBaseDirectory(let path):
            return "基础工作目录不存在：\(path)"
        }
    }
}

public struct WorkspaceScanner: Sendable {
    public init() {}

    // 忽略的非摄影/文档/轨迹/系统文件扩展名（不作为照片资产或伴随文件处理）
    private static let ignoredDocExtensions: Set<String> = [
        "gpx", "md", "log", "txt", "json", "yaml", "yml",
        "csv", "pdf", "doc", "docx", "zip", "tar", "gz",
        "7z", "rar", "bak", "tmp", "ds_store", "toml", "sh", "bat"
    ]

    // 屏蔽的非照片/系统/备份/归档/日志目录名（不论是分层模式还是扁平模式，待处理照片扫描均完全屏蔽）
    public static func isIgnoredDirectoryName(_ name: String) -> Bool {
        let lower = name.lowercased()
        if lower.hasPrefix(".") {
            return true
        }
        if lower == "inbox_bak" || lower == "bak" || lower == "backup" || lower == "backups" ||
            lower.hasSuffix("_bak") || lower.hasSuffix("_backup") {
            return true
        }
        if lower == "processed" || lower == "gpx" || lower == "logs" || lower == "node_modules" || lower == "dist" {
            return true
        }
        return false
    }

    public func scan(
        baseDirectory: String,
        sourceDirectory: String = "",
        rawExtensions: [String] = ["nef", "cr3", "arw", "dng", "raf", "rw2", "orf"]
    ) throws -> WorkspaceSummary {
        let fm = FileManager.default
        var isDirectory: ObjCBool = false
        guard fm.fileExists(atPath: baseDirectory, isDirectory: &isDirectory), isDirectory.boolValue else {
            throw WorkspaceScannerError.missingBaseDirectory(baseDirectory)
        }

        let inboxDirectory = sourceDirectory.isEmpty ? (baseDirectory as NSString).appendingPathComponent("Inbox") : sourceDirectory
        let gpxDirectory = (baseDirectory as NSString).appendingPathComponent("GPX")
        let processedDirectory = (baseDirectory as NSString).appendingPathComponent("Processed")
        let logsDirectory = (baseDirectory as NSString).appendingPathComponent("Logs")
        let inboxBakDirectory = (baseDirectory as NSString).appendingPathComponent("Inbox_bak")
        let pendingReportPath = (logsDirectory as NSString).appendingPathComponent("inbox_pending_report_latest.md")
        
        // 查找最新的日志文件: 优先 photools_latest.log, 其次 geotag.log
        var logFilePath = (logsDirectory as NSString).appendingPathComponent("photools_latest.log")
        if !fm.fileExists(atPath: logFilePath) {
            let legacyLog = (logsDirectory as NSString).appendingPathComponent("geotag.log")
            if fm.fileExists(atPath: legacyLog) {
                logFilePath = legacyLog
            }
        }

        let assetGroups = scanAssetGroups(inboxDirectory: inboxDirectory, rawExtensions: rawExtensions)
        let gpxFiles = scanFiles(root: gpxDirectory, ignoreIgnoredDirectories: false) { $0.pathExtension.lowercased() == "gpx" }
        let processedFileCount = scanFiles(root: processedDirectory, ignoreIgnoredDirectories: false) { !$0.hasDirectoryPath }.count
        let backupFileCount = scanFiles(root: inboxBakDirectory, ignoreIgnoredDirectories: false) { !$0.hasDirectoryPath }.count
        let pendingReportText = (try? String(contentsOfFile: pendingReportPath, encoding: .utf8)) ?? ""

        return WorkspaceSummary(
            baseDirectory: baseDirectory,
            inboxDirectory: inboxDirectory,
            gpxDirectory: gpxDirectory,
            processedDirectory: processedDirectory,
            logsDirectory: logsDirectory,
            inboxBakDirectory: inboxBakDirectory,
            gpxFiles: gpxFiles,
            assetGroups: assetGroups,
            processedFileCount: processedFileCount,
            backupFileCount: backupFileCount,
            logFilePath: logFilePath,
            pendingReportPath: pendingReportPath,
            pendingReportExists: fm.fileExists(atPath: pendingReportPath),
            pendingReportText: pendingReportText
        )
    }

    private func scanAssetGroups(inboxDirectory: String, rawExtensions: [String]) -> [PhotoAssetGroup] {
        let rawExts = Set(rawExtensions.map { $0.trimmingCharacters(in: CharacterSet(charactersIn: ".")).lowercased() })
        let files = scanFiles(root: inboxDirectory, ignoreIgnoredDirectories: true) { !$0.hasDirectoryPath }
        var groups: [String: MutableAssetGroup] = [:]

        for file in files {
            let url = URL(fileURLWithPath: file)
            let ext = url.pathExtension.lowercased()

            // 忽略非摄影文件、文档、轨迹文件（轨迹只属于 GPX 目录）
            if Self.ignoredDocExtensions.contains(ext) {
                continue
            }

            let baseName = url.deletingPathExtension().lastPathComponent
            let directory = url.deletingLastPathComponent().path
            let key = "\(directory)::\(baseName.lowercased())"
            var group = groups[key] ?? MutableAssetGroup(baseName: baseName, directory: directory)

            if rawExts.contains(ext) {
                group.rawPath = file
            } else if ext == "jpg" || ext == "jpeg" {
                group.jpgPath = file
            } else {
                if ext == "xmp" {
                    group.xmpPath = file
                }
                group.companionPaths.append(file)
            }

            groups[key] = group
        }

        // 核心规约：摄影资产组必须至少包含 RAW 或 JPG 主文件（与 Go 核心 Discoverer 100% 对齐）
        let fm = FileManager.default
        return groups.values
            .filter { $0.rawPath != nil || $0.jpgPath != nil }
            .map { group in
                let primary = group.rawPath ?? group.jpgPath ?? ""
                var modDate: Date? = nil
                if !primary.isEmpty {
                    if let attrs = try? fm.attributesOfItem(atPath: primary) {
                        modDate = attrs[.modificationDate] as? Date ?? attrs[.creationDate] as? Date
                    }
                }
                return PhotoAssetGroup(
                    id: "\(group.directory)::\(group.baseName.lowercased())",
                    baseName: group.baseName,
                    directory: group.directory,
                    rawPath: group.rawPath,
                    jpgPath: group.jpgPath,
                    xmpPath: group.xmpPath,
                    companionPaths: group.companionPaths.sorted(),
                    fileModificationDate: modDate
                )
            }
            .sorted {
                if let d0 = $0.fileModificationDate, let d1 = $1.fileModificationDate, d0 != d1 {
                    return d0 < d1
                }
                if $0.directory != $1.directory {
                    return $0.directory < $1.directory
                }
                return $0.baseName.localizedStandardCompare($1.baseName) == .orderedAscending
            }
    }

    private func scanFiles(root: String, ignoreIgnoredDirectories: Bool = false, shouldInclude: (URL) -> Bool) -> [String] {
        let rootURL = URL(fileURLWithPath: root).standardizedFileURL
        guard let enumerator = FileManager.default.enumerator(
            at: rootURL,
            includingPropertiesForKeys: [.isRegularFileKey, .isDirectoryKey],
            options: [.skipsHiddenFiles]
        ) else {
            return []
        }

        var files: [String] = []
        for case let url as URL in enumerator {
            let lastComponent = url.lastPathComponent
            let isDir = (try? url.resourceValues(forKeys: [.isDirectoryKey]))?.isDirectory ?? false

            if isDir {
                if ignoreIgnoredDirectories && Self.isIgnoredDirectoryName(lastComponent) {
                    enumerator.skipDescendants()
                    continue
                }
            } else {
                if ignoreIgnoredDirectories {
                    let relPath = url.standardizedFileURL.path.replacingOccurrences(of: rootURL.path, with: "")
                    let components = relPath.split(separator: "/")
                    if components.dropLast().contains(where: { Self.isIgnoredDirectoryName(String($0)) }) {
                        continue
                    }
                }
                if shouldInclude(url) {
                    files.append(url.path)
                }
            }
        }
        return files.sorted()
    }
}

private struct MutableAssetGroup {
    var baseName: String
    var directory: String
    var rawPath: String?
    var jpgPath: String?
    var xmpPath: String?
    var companionPaths: [String] = []
}
