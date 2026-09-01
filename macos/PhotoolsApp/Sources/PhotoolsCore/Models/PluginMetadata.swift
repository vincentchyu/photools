import Foundation

/// 插件元数据模型 (由 Go 端 Capability / FFI 动态下发权威数据)
public struct PluginMetadata: Codable, Identifiable, Sendable {
    public let id: String
    public let name: String
    public let desc: String
    public let nameKey: String?
    public let descKey: String?
    public let priority: Int
    public let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case desc
        case nameKey = "name_key"
        case descKey = "desc_key"
        case priority
        case enabled
    }

    public init(
        id: String,
        name: String,
        desc: String,
        nameKey: String? = nil,
        descKey: String? = nil,
        priority: Int,
        enabled: Bool
    ) {
        self.id = id
        self.name = name
        self.desc = desc
        self.nameKey = nameKey
        self.descKey = descKey
        self.priority = priority
        self.enabled = enabled
    }
}
