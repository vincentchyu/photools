import PhotoolsCore
import SwiftUI

struct LogsAndReportView: View {
    @ObservedObject var store: WorkspaceStore
    @State private var selectedTab: LogTab = .liveLog

    enum LogTab: String, CaseIterable, Identifiable {
        case liveLog = "实时中文日志流"
        case pendingReport = "待处理清单报告"

        var id: String { rawValue }
    }

    var body: some View {
        VStack(spacing: 0) {
            headerBar
            Divider()
            content
        }
    }

    private var headerBar: some View {
        HStack {
            Picker("", selection: $selectedTab) {
                ForEach(LogTab.allCases) { tab in
                    Text(tab.rawValue).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .frame(width: 280)

            Spacer()

            if selectedTab == .liveLog {
                Button {
                    openFile(store.summary?.logFilePath)
                } label: {
                    Label("打开日志文件", systemImage: "arrow.up.right.square")
                }
                .disabled(store.summary == nil)
            } else {
                Button {
                    openFile(store.summary?.pendingReportPath)
                } label: {
                    Label("打开报告 Markdown", systemImage: "doc.plaintext")
                }
                .disabled(store.summary?.pendingReportExists != true)
            }
        }
        .padding(12)
    }

    @ViewBuilder
    private var content: some View {
        switch selectedTab {
        case .liveLog:
            HackerTerminalView(
                logText: store.liveLog,
                isRunning: store.runState.isRunning,
                isAutoScroll: true,
                placeholder: "暂无实时运行日志。点击“立即执行流水线”开始处理。\n"
            )
            .padding(12)
        case .pendingReport:
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if store.summary?.pendingReportExists == true {
                        Text(store.summary?.pendingReportText ?? "")
                            .font(.system(.body, design: .monospaced))
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    } else {
                        VStack(spacing: 8) {
                            Image(systemName: "checkmark.seal.fill")
                                .font(.system(size: 40))
                                .foregroundStyle(.green)
                            Text("当前没有待处理或中断异常报告")
                                .font(.headline)
                            Text("所有资产均已成功归档或 Inbox 为空。")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .padding(60)
                    }
                }
                .padding(20)
            }
        }
    }

    private func openFile(_ path: String?) {
        guard let path else { return }
        NSWorkspace.shared.open(URL(fileURLWithPath: path))
    }
}
