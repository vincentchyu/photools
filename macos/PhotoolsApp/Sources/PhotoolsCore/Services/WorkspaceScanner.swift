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

    public static var defaultGPXDirectory: String {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        return (home as NSString).appendingPathComponent(".config/gpx")
    }

    // 判定是否为受支持的 GPX 运动轨迹（仅限 hiking 与 walking，且后缀为 .gpx，非隐藏文件）
    public static func isAllowedGPXTrack(filename: String) -> Bool {
        if filename.hasPrefix(".") {
            return false
        }
        let url = URL(fileURLWithPath: filename)
        guard url.pathExtension.lowercased() == "gpx" else {
            return false
        }
        let baseName = url.deletingPathExtension().lastPathComponent.lowercased()
        return baseName.hasPrefix("hiking") || baseName.hasPrefix("walking")
    }

    private func scanGPXFiles(gpxDirectory: String) -> [String] {
        let fm = FileManager.default
        var isDir: ObjCBool = false
        guard fm.fileExists(atPath: gpxDirectory, isDirectory: &isDir), isDir.boolValue else {
            return []
        }
        guard let items = try? fm.contentsOfDirectory(atPath: gpxDirectory) else {
            return []
        }

        var results: [String] = []
        for item in items {
            let fullPath = (gpxDirectory as NSString).appendingPathComponent(item)
            var itemIsDir: ObjCBool = false
            if fm.fileExists(atPath: fullPath, isDirectory: &itemIsDir), !itemIsDir.boolValue {
                if Self.isAllowedGPXTrack(filename: item) {
                    results.append(fullPath)
                }
            }
        }
        return results.sorted()
    }

    public func scan(
        baseDirectory: String,
        sourceDirectory: String = "",
        gpxDirectory: String = "",
        rawExtensions: [String] = ["nef", "cr3", "arw", "dng", "raf", "rw2", "orf"]
    ) throws -> WorkspaceSummary {
        let fm = FileManager.default
        var isDirectory: ObjCBool = false
        guard fm.fileExists(atPath: baseDirectory, isDirectory: &isDirectory), isDirectory.boolValue else {
            throw WorkspaceScannerError.missingBaseDirectory(baseDirectory)
        }

        let inboxDirectory = sourceDirectory.isEmpty ? (baseDirectory as NSString).appendingPathComponent("Inbox") : sourceDirectory
        let effectiveGPXDir: String
        if gpxDirectory.trimmingCharacters(in: .whitespaces).isEmpty {
            effectiveGPXDir = Self.defaultGPXDirectory
        } else {
            effectiveGPXDir = (gpxDirectory as NSString).expandingTildeInPath
        }
        let processedDirectory = (baseDirectory as NSString).appendingPathComponent("Processed")
        let defaultLogDir = ("~/.logs/photools" as NSString).expandingTildeInPath
        var logsDirectory = defaultLogDir
        var logFilePath = (logsDirectory as NSString).appendingPathComponent("photools_latest.log")
        var pendingReportPath = (logsDirectory as NSString).appendingPathComponent("inbox_pending_report_latest.md")
        
        // 查找最新的日志文件: 优先全局 ~/.logs/photools/photools_latest.log，若无则优雅兼容工作区本地 Logs/
        if !fm.fileExists(atPath: logFilePath) {
            let localLogsDir = (baseDirectory as NSString).appendingPathComponent("Logs")
            let localLog = (localLogsDir as NSString).appendingPathComponent("photools_latest.log")
            if fm.fileExists(atPath: localLog) {
                logsDirectory = localLogsDir
                logFilePath = localLog
                pendingReportPath = (localLogsDir as NSString).appendingPathComponent("inbox_pending_report_latest.md")
            } else {
                let legacyLog = (localLogsDir as NSString).appendingPathComponent("geotag.log")
                if fm.fileExists(atPath: legacyLog) {
                    logsDirectory = localLogsDir
                    logFilePath = legacyLog
                    pendingReportPath = (localLogsDir as NSString).appendingPathComponent("inbox_pending_report_latest.md")
                }
            }
        }
        let inboxBakDirectory = (baseDirectory as NSString).appendingPathComponent("Inbox_bak")

        let assetGroups = scanAssetGroups(inboxDirectory: inboxDirectory, rawExtensions: rawExtensions)
        let gpxFiles = scanGPXFiles(gpxDirectory: effectiveGPXDir)
        let processedFileCount = scanFiles(root: processedDirectory, ignoreIgnoredDirectories: false) { !$0.hasDirectoryPath }.count
        let backupFileCount = scanFiles(root: inboxBakDirectory, ignoreIgnoredDirectories: false) { !$0.hasDirectoryPath }.count
        let pendingReportText = (try? String(contentsOfFile: pendingReportPath, encoding: .utf8)) ?? ""

        return WorkspaceSummary(
            baseDirectory: baseDirectory,
            inboxDirectory: inboxDirectory,
            gpxDirectory: effectiveGPXDir,
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
