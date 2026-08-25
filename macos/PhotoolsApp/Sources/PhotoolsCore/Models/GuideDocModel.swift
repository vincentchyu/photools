import Foundation

public struct GuideDocItem: Identifiable, Equatable, Hashable, Sendable {
    public let id: String
    public let title: String
    public let subtitle: String
    public let icon: String
    public let category: String
    public let fileName: String?
    public let badge: String?

    public init(
        id: String,
        title: String,
        subtitle: String,
        icon: String,
        category: String,
        fileName: String? = nil,
        badge: String? = nil
    ) {
        self.id = id
        self.title = title
        self.subtitle = subtitle
        self.icon = icon
        self.category = category
        self.fileName = fileName
        self.badge = badge
    }

    public static let allDocs: [GuideDocItem] = [
        GuideDocItem(
            id: "quick-guide",
            title: "快速上手指引",
            subtitle: "五大核心工作流、四阶段流水线矩阵与高级就地模式",
            icon: "sparkles",
            category: "使用指南",
            badge: "入门必读"
        ),
        GuideDocItem(
            id: "macos-design",
            title: "macOS 原生架构与 FFI",
            subtitle: "C-Shared FFI 极速直通、三栏 TEA 架构与内存安全契约",
            icon: "macwindow",
            category: "核心技术设计",
            fileName: "MACOS_CLIENT_TECHNICAL_DESIGN.md",
            badge: "macOS 13+"
        ),
        GuideDocItem(
            id: "system-arch",
            title: "系统总体架构与规约",
            subtitle: "主文件优先模型、分阶段屏障调度器与插件生命周期",
            icon: "point.3.filled.connected.trianglepath.dotted",
            category: "核心技术设计",
            fileName: "ARCHITECTURE_AND_DESIGN.md",
            badge: "架构基石"
        ),
        GuideDocItem(
            id: "geocoding-design",
            title: "离线逆地理与 KD-Tree",
            subtitle: "全球 94 万+ 离线点位、3D 空间索引与高精拓扑反查",
            icon: "globe.asia.australia.fill",
            category: "核心技术设计",
            fileName: "GEONAMES_AND_GEOCODING_DESIGN.md",
            badge: "KD-Tree"
        ),
        GuideDocItem(
            id: "exiftool-safety",
            title: "ExifTool 并发与安全 I/O",
            subtitle: "Stay-Open 常驻进程池、双写二次校验与元数据防污染",
            icon: "camera.badge.ellipsis",
            category: "核心技术设计",
            fileName: "EXIFTOOL_IO_AND_SAFETY_DESIGN.md",
            badge: "进程池"
        ),
        GuideDocItem(
            id: "config-settings",
            title: "配置持久化与插件选项",
            subtitle: "自描述 Capability 契约、会话覆盖与动态 Schema 驱动",
            icon: "slider.horizontal.3",
            category: "核心技术设计",
            fileName: "CONFIGURATION_AND_SETTINGS_DESIGN.md",
            badge: "Schema"
        ),
        GuideDocItem(
            id: "gemini-rules",
            title: "项目核心规约 (GEMINI.md)",
            subtitle: "photools 持久化规约、用户画像与 7 步插件 SOP",
            icon: "shield.lefthalf.filled",
            category: "规约与文档",
            fileName: "GEMINI.md",
            badge: "SOP"
        ),
        GuideDocItem(
            id: "readme-overview",
            title: "项目总览 (README.md)",
            subtitle: "项目简介、CLI 命令速查表与全平台构建指南",
            icon: "doc.text.fill",
            category: "规约与文档",
            fileName: "README.md",
            badge: "README"
        )
    ]
}
