import SwiftUI

/// 极客/黑客暗黑终端视窗组件 (Dark Hacker Console)
/// 无论系统当前处于日间模式 (Light) 还是夜间模式 (Dark)，
/// 该视窗均保持纯粹沉浸式的黑客深色终端风格。
public struct HackerTerminalView: View {
    public let logText: String
    public let isRunning: Bool
    public var isAutoScroll: Bool
    public var placeholder: String

    @State private var blinkCursor: Bool = true
    private let timer = Timer.publish(every: 0.6, on: .main, in: .common).autoconnect()

    public init(
        logText: String,
        isRunning: Bool = false,
        isAutoScroll: Bool = true,
        placeholder: String = "等待任务启动...\n"
    ) {
        self.logText = logText
        self.isRunning = isRunning
        self.isAutoScroll = isAutoScroll
        self.placeholder = placeholder
    }

    public var body: some View {
        VStack(spacing: 0) {
            // 终端顶部标题栏 (macOS 三色红黄绿圆点 + 终端状态)
            terminalHeader

            Divider()
                .background(Color(red: 0.2, green: 0.23, blue: 0.28))

            // 终端日志文本流视窗
            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 3) {
                        if logText.isEmpty {
                            Text(placeholder)
                                .font(.system(size: 11.5, weight: .regular, design: .monospaced))
                                .foregroundStyle(Color(red: 0.48, green: 0.48, blue: 0.48)) // #7A7A7A 呼应 scheme3
                                .padding(.horizontal, 10)
                                .padding(.vertical, 8)
                        } else {
                            LazyVStack(alignment: .leading, spacing: 2) {
                                ForEach(Array(parsedLines.enumerated()), id: \.offset) { _, item in
                                    item
                                        .font(.system(size: 11.5, weight: .regular, design: .monospaced))
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                }
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 8)
                        }

                        // 运行中的黑客终端闪烁光标
                        if isRunning {
                            HStack(spacing: 4) {
                                Text("❯")
                                    .font(.system(size: 11.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(Color(red: 0.2, green: 0.83, blue: 0.6)) // #34D399
                                Text("▌")
                                    .font(.system(size: 11.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(Color(red: 0.2, green: 0.83, blue: 0.6))
                                    .opacity(blinkCursor ? 1.0 : 0.0)
                            }
                            .padding(.horizontal, 10)
                            .padding(.bottom, 6)
                        }

                        // 底部定位锚点
                        Color.clear
                            .frame(height: 1)
                            .id("terminal_bottom_anchor")
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .onChange(of: logText) { _ in
                    if isAutoScroll {
                        proxy.scrollTo("terminal_bottom_anchor", anchor: .bottom)
                    }
                }
            }
        }
        .background(Color(red: 0.05, green: 0.07, blue: 0.09)) // 极暗黑客炭黑 #0D1117
        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color(red: 0.19, green: 0.21, blue: 0.24), lineWidth: 1) // #30363D
        )
        .shadow(color: Color.black.opacity(0.18), radius: 6, x: 0, y: 3)
        .onReceive(timer) { _ in
            if isRunning {
                blinkCursor.toggle()
            }
        }
    }

    // 终端顶部栏
    private var terminalHeader: some View {
        HStack(spacing: 8) {
            // 红黄绿三色控制圆点
            HStack(spacing: 6) {
                Circle()
                    .fill(Color(red: 1.0, green: 0.37, blue: 0.34)) // #FF5F56
                    .frame(width: 9, height: 9)
                Circle()
                    .fill(Color(red: 1.0, green: 0.74, blue: 0.18)) // #FFBD2E
                    .frame(width: 9, height: 9)
                Circle()
                    .fill(Color(red: 0.15, green: 0.79, blue: 0.25)) // #27C93F
                    .frame(width: 9, height: 9)
            }
            .padding(.leading, 10)

            Spacer()

            // 终端标题
            HStack(spacing: 5) {
                Image(systemName: "terminal")
                    .font(.system(size: 9.5))
                    .foregroundStyle(Color(red: 0.48, green: 0.48, blue: 0.48)) // #7A7A7A
                Text("photools ~ stream")
                    .font(.system(size: 10, weight: .semibold, design: .monospaced))
                    .foregroundStyle(Color(red: 0.55, green: 0.58, blue: 0.62))
            }

            Spacer()

            // 状态徽章
            if isRunning {
                HStack(spacing: 3) {
                    Circle()
                        .fill(Color(red: 0.2, green: 0.83, blue: 0.6))
                        .frame(width: 6, height: 6)
                    Text("LIVE")
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(Color(red: 0.2, green: 0.83, blue: 0.6))
                }
                .padding(.trailing, 10)
            } else {
                Text("IDLE")
                    .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                    .foregroundStyle(Color(red: 0.48, green: 0.48, blue: 0.48))
                    .padding(.trailing, 10)
            }
        }
        .frame(height: 26)
        .background(Color(red: 0.08, green: 0.10, blue: 0.13)) // #161B22
    }

    // 将单行日志解析为语法着色的 Text
    private var parsedLines: [Text] {
        let lines = logText.components(separatedBy: "\n")
        return lines.compactMap { line in
            let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.isEmpty { return nil }
            return highlightLogLine(line)
        }
    }

    private func highlightLogLine(_ rawLine: String) -> Text {
        var text = Text("")
        let line = rawLine

        // 1. 匹配时间戳前缀 [HH:mm:ss.SSS] 或 [HH:mm:ss]
        var remaining = line
        if let timeMatch = line.range(of: #"^\[\d{2}:\d{2}:\d{2}(\.\d{1,3})?\]"#, options: .regularExpression) {
            let timeStr = String(line[timeMatch])
            text = text + Text(timeStr + " ").foregroundColor(Color(red: 0.48, green: 0.48, blue: 0.48)) // #7A7A7A 呼应 scheme3
            remaining = String(line[timeMatch.upperBound...]).trimmingCharacters(in: .whitespaces)
        }

        // 2. 匹配日志级别标签 [LEVEL]
        if let levelMatch = remaining.range(of: #"^\[(INFO|WARN|ERROR|FATAL|DEBUG|SUCCESS)\]"#, options: .regularExpression) {
            let levelStr = String(remaining[levelMatch])
            let levelColor: Color
            if levelStr.contains("ERROR") || levelStr.contains("FATAL") {
                levelColor = Color(red: 0.97, green: 0.44, blue: 0.44) // #F87171 荧光红
            } else if levelStr.contains("WARN") {
                levelColor = Color(red: 0.98, green: 0.75, blue: 0.14) // #FBBF24 极客金
            } else {
                levelColor = Color(red: 0.2, green: 0.83, blue: 0.6) // #34D399 翠绿
            }
            text = text + Text(levelStr + " ").foregroundColor(levelColor).bold()
            remaining = String(remaining[levelMatch.upperBound...]).trimmingCharacters(in: .whitespaces)
        }

        // 3. 匹配阶段标识 [阶段 X] 或 [模块]
        if let stageMatch = remaining.range(of: #"^\[阶段\s*\d+[^\]]*\]"#, options: .regularExpression) {
            let stageStr = String(remaining[stageMatch])
            text = text + Text(stageStr + " ").foregroundColor(Color(red: 0.22, green: 0.74, blue: 0.97)).bold() // #38BDF8 青蓝
            remaining = String(remaining[stageMatch.upperBound...]).trimmingCharacters(in: .whitespaces)
        } else if let genericTagMatch = remaining.range(of: #"^\[[^\]]+\]"#, options: .regularExpression) {
            let tagStr = String(remaining[genericTagMatch])
            text = text + Text(tagStr + " ").foregroundColor(Color(red: 0.64, green: 0.9, blue: 0.21)) // #A3E635 草绿
            remaining = String(remaining[genericTagMatch.upperBound...]).trimmingCharacters(in: .whitespaces)
        }

        // 4. 正文渲染 (根据关键词着色)
        if remaining.contains("失败") || remaining.contains("错误") || remaining.contains("error") {
            text = text + Text(remaining).foregroundColor(Color(red: 0.97, green: 0.44, blue: 0.44))
        } else if remaining.contains("跳过") || remaining.contains("warning") || remaining.contains("未命中") {
            text = text + Text(remaining).foregroundColor(Color(red: 0.95, green: 0.82, blue: 0.55))
        } else if remaining.contains("成功") || remaining.contains("就绪") || remaining.contains("已写入") || remaining.contains("已完成") {
            text = text + Text(remaining).foregroundColor(Color(red: 0.9, green: 0.93, blue: 0.95))
        } else {
            text = text + Text(remaining).foregroundColor(Color(red: 0.85, green: 0.88, blue: 0.91)) // #D1D5DB
        }

        return text
    }
}
