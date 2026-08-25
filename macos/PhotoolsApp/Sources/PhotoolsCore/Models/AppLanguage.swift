import Foundation

/// 应用程序支持的界面语言
public enum AppLanguage: String, CaseIterable, Identifiable, Codable, Sendable {
    case system = "system"
    case zhHans = "zh-Hans"
    case en = "en"

    public var id: String { rawValue }

    public var displayName: String {
        switch self {
        case .system:
            return "跟随系统 (System Default)"
        case .zhHans:
            return "简体中文 (Chinese)"
        case .en:
            return "English"
        }
    }

    /// 实际生效的语言代码 (如果是 system 则根据系统语言裁决)
    public var effectiveLanguageCode: String {
        switch self {
        case .zhHans:
            return "zh-Hans"
        case .en:
            return "en"
        case .system:
            let preferred = Locale.preferredLanguages.first ?? "en"
            if preferred.starts(with: "zh") {
                return "zh-Hans"
            }
            return "en"
        }
    }

    public var isChinese: Bool {
        effectiveLanguageCode == "zh-Hans"
    }
}
