import SwiftUI
import AppKit

public struct MarkdownRendererView: View {
    public let markdownText: String

    public init(markdownText: String) {
        self.markdownText = markdownText
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            let blocks = MarkdownBlockParser.parse(markdownText)
            ForEach(blocks) { block in
                renderBlock(block)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func renderBlock(_ block: MarkdownBlock) -> some View {
        switch block.type {
        case .h1(let text):
            VStack(alignment: .leading, spacing: 6) {
                Text(text)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
                Divider()
                    .padding(.bottom, 4)
            }
            .padding(.top, 8)

        case .h2(let text):
            HStack(spacing: 6) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(Color.teal)
                    .frame(width: 3, height: 16)
                Text(text)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }
            .padding(.top, 10)
            .padding(.bottom, 2)

        case .h3(let text):
            Text(text)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(.primary)
                .textSelection(.enabled)
                .padding(.top, 4)

        case .h4(let text):
            Text(text)
                .font(.system(size: 12.5, weight: .semibold))
                .foregroundStyle(.secondary)
                .textSelection(.enabled)

        case .paragraph(let text):
            Text(LocalizedStringKey(text))
                .font(.system(size: 13))
                .lineSpacing(4)
                .foregroundStyle(.primary)
                .textSelection(.enabled)

        case .codeBlock(let code, let language):
            codeBlockView(code: code, language: language)

        case .quote(let text):
            quoteBlockView(text: text)

        case .bulletItem(let text, let level):
            HStack(alignment: .top, spacing: 6) {
                Text("•")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(.teal)
                    .padding(.leading, CGFloat(level * 12))
                Text(LocalizedStringKey(text))
                    .font(.system(size: 12.5))
                    .lineSpacing(3)
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }

        case .numberedItem(let number, let text, let level):
            HStack(alignment: .top, spacing: 6) {
                Text("\(number).")
                    .font(.system(size: 12, weight: .semibold, design: .monospaced))
                    .foregroundStyle(.teal)
                    .padding(.leading, CGFloat(level * 12))
                Text(LocalizedStringKey(text))
                    .font(.system(size: 12.5))
                    .lineSpacing(3)
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }

        case .divider:
            Divider()
                .padding(.vertical, 4)
        }
    }

    private func codeBlockView(code: String, language: String) -> some View {
        VStack(spacing: 0) {
            HStack {
                HStack(spacing: 4) {
                    Circle().fill(Color.red.opacity(0.8)).frame(width: 7, height: 7)
                    Circle().fill(Color.yellow.opacity(0.8)).frame(width: 7, height: 7)
                    Circle().fill(Color.green.opacity(0.8)).frame(width: 7, height: 7)
                    if !language.isEmpty {
                        Text(language)
                            .font(.system(size: 10, weight: .semibold, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.6))
                            .padding(.leading, 4)
                    }
                }

                Spacer()

                Button {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(code, forType: .string)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "doc.on.doc")
                            .font(.system(size: 9))
                        Text("复制")
                            .font(.system(size: 9.5))
                    }
                    .foregroundStyle(.white.opacity(0.7))
                }
                .buttonStyle(.borderless)
                .help("复制代码内容")
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(Color(red: 0.12, green: 0.14, blue: 0.17))

            ScrollView(.horizontal, showsIndicators: true) {
                Text(code)
                    .font(.system(size: 11.5, design: .monospaced))
                    .foregroundStyle(Color(red: 0.88, green: 0.92, blue: 0.96))
                    .padding(10)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .textSelection(.enabled)
            }
            .background(Color(red: 0.07, green: 0.09, blue: 0.11))
        }
        .clipShape(RoundedRectangle(cornerRadius: 6))
        .overlay(
            RoundedRectangle(cornerRadius: 6)
                .stroke(Color.white.opacity(0.1), lineWidth: 1)
        )
    }

    private func quoteBlockView(text: String) -> some View {
        let (alertType, cleanText) = parseAlert(text)
        let alertColor: Color = {
            switch alertType {
            case .tip: return .green
            case .note: return .blue
            case .warning: return .orange
            case .important: return .purple
            case .none: return .teal
            }
        }()

        let iconName: String = {
            switch alertType {
            case .tip: return "lightbulb.fill"
            case .note: return "info.circle.fill"
            case .warning: return "exclamationmark.triangle.fill"
            case .important: return "flame.fill"
            case .none: return "quote.opening"
            }
        }()

        return HStack(alignment: .top, spacing: 8) {
            RoundedRectangle(cornerRadius: 2)
                .fill(alertColor)
                .frame(width: 3)

            VStack(alignment: .leading, spacing: 4) {
                if alertType != .none {
                    HStack(spacing: 4) {
                        Image(systemName: iconName)
                            .font(.system(size: 10))
                            .foregroundStyle(alertColor)
                        Text(alertType.title)
                            .font(.system(size: 11, weight: .bold))
                            .foregroundStyle(alertColor)
                    }
                }
                Text(LocalizedStringKey(cleanText))
                    .font(.system(size: 12))
                    .lineSpacing(3)
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }
            .padding(.vertical, 4)
            .padding(.trailing, 8)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(alertColor.opacity(0.06), in: RoundedRectangle(cornerRadius: 6))
        .overlay(
            RoundedRectangle(cornerRadius: 6)
                .stroke(alertColor.opacity(0.2), lineWidth: 1)
        )
    }

    private enum AlertType {
        case none, note, tip, warning, important

        var title: String {
            switch self {
            case .none: return ""
            case .note: return "提示 (NOTE)"
            case .tip: return "最佳实践 (TIP)"
            case .warning: return "注意事项 (WARNING)"
            case .important: return "关键约束 (IMPORTANT)"
            }
        }
    }

    private func parseAlert(_ raw: String) -> (AlertType, String) {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.hasPrefix("[!NOTE]") {
            return (.note, trimmed.replacingOccurrences(of: "[!NOTE]", with: "").trimmingCharacters(in: .whitespacesAndNewlines))
        }
        if trimmed.hasPrefix("[!TIP]") {
            return (.tip, trimmed.replacingOccurrences(of: "[!TIP]", with: "").trimmingCharacters(in: .whitespacesAndNewlines))
        }
        if trimmed.hasPrefix("[!WARNING]") || trimmed.hasPrefix("[!CAUTION]") {
            return (.warning, trimmed.replacingOccurrences(of: "[!WARNING]", with: "").replacingOccurrences(of: "[!CAUTION]", with: "").trimmingCharacters(in: .whitespacesAndNewlines))
        }
        if trimmed.hasPrefix("[!IMPORTANT]") {
            return (.important, trimmed.replacingOccurrences(of: "[!IMPORTANT]", with: "").trimmingCharacters(in: .whitespacesAndNewlines))
        }
        return (.none, raw)
    }
}

// MARK: - Markdown Block Parsing Model

public struct MarkdownBlock: Identifiable {
    public let id: String
    public let type: BlockType

    public enum BlockType {
        case h1(String)
        case h2(String)
        case h3(String)
        case h4(String)
        case paragraph(String)
        case codeBlock(code: String, language: String)
        case quote(String)
        case bulletItem(text: String, level: Int)
        case numberedItem(number: Int, text: String, level: Int)
        case divider
    }

    public init(id: String = UUID().uuidString, type: BlockType) {
        self.id = id
        self.type = type
    }
}

public struct MarkdownBlockParser {
    public static func parse(_ raw: String) -> [MarkdownBlock] {
        var blocks: [MarkdownBlock] = []
        let lines = raw.components(separatedBy: .newlines)
        var i = 0
        let count = lines.count

        while i < count {
            let line = lines[i]
            let trimmed = line.trimmingCharacters(in: .whitespaces)

            if trimmed.isEmpty {
                i += 1
                continue
            }

            // 1. 代码块 ```
            if trimmed.hasPrefix("```") {
                let lang = String(trimmed.dropFirst(3)).trimmingCharacters(in: .whitespaces)
                var codeLines: [String] = []
                i += 1
                while i < count {
                    let cLine = lines[i]
                    if cLine.trimmingCharacters(in: .whitespaces).hasPrefix("```") {
                        i += 1
                        break
                    }
                    codeLines.append(cLine)
                    i += 1
                }
                blocks.append(MarkdownBlock(type: .codeBlock(code: codeLines.joined(separator: "\n"), language: lang)))
                continue
            }

            // 2. 分隔线 --- or ***
            if trimmed == "---" || trimmed == "***" || trimmed == "___" {
                blocks.append(MarkdownBlock(type: .divider))
                i += 1
                continue
            }

            // 3. 标题
            if trimmed.hasPrefix("# ") {
                blocks.append(MarkdownBlock(type: .h1(String(trimmed.dropFirst(2)))))
                i += 1
                continue
            }
            if trimmed.hasPrefix("## ") {
                blocks.append(MarkdownBlock(type: .h2(String(trimmed.dropFirst(3)))))
                i += 1
                continue
            }
            if trimmed.hasPrefix("### ") {
                blocks.append(MarkdownBlock(type: .h3(String(trimmed.dropFirst(4)))))
                i += 1
                continue
            }
            if trimmed.hasPrefix("#### ") {
                blocks.append(MarkdownBlock(type: .h4(String(trimmed.dropFirst(5)))))
                i += 1
                continue
            }

            // 4. 引用块 >
            if trimmed.hasPrefix(">") {
                var quoteLines: [String] = []
                while i < count && lines[i].trimmingCharacters(in: .whitespaces).hasPrefix(">") {
                    let qLine = lines[i].trimmingCharacters(in: .whitespaces)
                    let text = String(qLine.dropFirst()).trimmingCharacters(in: .whitespaces)
                    quoteLines.append(text)
                    i += 1
                }
                blocks.append(MarkdownBlock(type: .quote(quoteLines.joined(separator: "\n"))))
                continue
            }

            // 5. 无序列表 - or *
            if trimmed.hasPrefix("- ") || trimmed.hasPrefix("* ") {
                let text = String(trimmed.dropFirst(2))
                let indent = line.prefix(while: { $0 == " " || $0 == "\t" }).count / 2
                blocks.append(MarkdownBlock(type: .bulletItem(text: text, level: indent)))
                i += 1
                continue
            }

            // 6. 有序列表 1. 2.
            if let firstSpace = trimmed.firstIndex(of: " "),
               let num = Int(trimmed[..<firstSpace].replacingOccurrences(of: ".", with: "")),
               trimmed[..<firstSpace].hasSuffix(".") {
                let text = String(trimmed[trimmed.index(after: firstSpace)...])
                let indent = line.prefix(while: { $0 == " " || $0 == "\t" }).count / 2
                blocks.append(MarkdownBlock(type: .numberedItem(number: num, text: text, level: indent)))
                i += 1
                continue
            }

            // 7. 普通段落 (合并相邻非空文本行)
            var pLines: [String] = [trimmed]
            i += 1
            while i < count {
                let nextLine = lines[i].trimmingCharacters(in: .whitespaces)
                if nextLine.isEmpty || nextLine.hasPrefix("#") || nextLine.hasPrefix("```") || nextLine.hasPrefix(">") || nextLine.hasPrefix("- ") || nextLine.hasPrefix("* ") || nextLine == "---" {
                    break
                }
                pLines.append(nextLine)
                i += 1
            }
            blocks.append(MarkdownBlock(type: .paragraph(pLines.joined(separator: "\n"))))
        }

        return blocks
    }
}
