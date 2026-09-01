import Foundation

/// 对应 ~/.config/photools/plugins.json 中的 global 全局偏好持久化结构
public struct DiskGlobalSettings: Codable, Sendable {
    public var gpxDir: String?
    public var logDir: String?
    public var sidecarPolicy: String?
    public var companionExtensions: [String]?
    public var rawExtensions: [String]?
    public var workers: Int?
    public var language: String?

    enum CodingKeys: String, CodingKey {
        case gpxDir = "gpx_dir"
        case logDir = "log_dir"
        case sidecarPolicy = "sidecar_policy"
        case companionExtensions = "companion_extensions"
        case rawExtensions = "raw_extensions"
        case workers = "workers"
        case language = "language"
    }

    public init(
        gpxDir: String? = nil,
        logDir: String? = nil,
        sidecarPolicy: String? = nil,
        companionExtensions: [String]? = nil,
        rawExtensions: [String]? = nil,
        workers: Int? = nil,
        language: String? = nil
    ) {
        self.gpxDir = gpxDir
        self.logDir = logDir
        self.sidecarPolicy = sidecarPolicy
        self.companionExtensions = companionExtensions
        self.rawExtensions = rawExtensions
        self.workers = workers
        self.language = language
    }
}

/// 对应 ~/.config/photools/plugins.json 顶级结构
public struct DiskPluginsConfig: Codable, Sendable {
    public var global: DiskGlobalSettings?

    public init(global: DiskGlobalSettings? = nil) {
        self.global = global
    }
}

/// 全局磁盘配置加载器 (单一事实源 Single Source of Truth)
public enum DiskConfigLoader {
    public static var configURL: URL {
        let home = FileManager.default.homeDirectoryForCurrentUser
        return home.appendingPathComponent(".config/photools/plugins.json")
    }

    /// 从指定或默认路径加载磁盘上的全局偏好
    public static func load(from customURL: URL? = nil) -> DiskGlobalSettings? {
        let url = customURL ?? configURL
        guard FileManager.default.fileExists(atPath: url.path) else {
            return nil
        }
        guard let data = try? Data(contentsOf: url),
              let cfg = try? JSONDecoder().decode(DiskPluginsConfig.self, from: data) else {
            return nil
        }
        return cfg.global
    }
}
