import PhotoolsCore
import SwiftUI

public struct PipelineStageItem: Identifiable, Sendable {
    public let id: String
    public let nameKey: L10nKey
    public let icon: String

    public init(id: String, nameKey: L10nKey, icon: String) {
        self.id = id
        self.nameKey = nameKey
        self.icon = icon
    }
}

public struct PipelineExecutionConsole: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var isAutoScroll: Bool = true

    private let stages: [PipelineStageItem] = [
        PipelineStageItem(id: "discover", nameKey: .stageDiscover, icon: "magnifyingglass"),
        PipelineStageItem(id: "geotag", nameKey: .stageGeotag, icon: "location.fill"),
        PipelineStageItem(id: "interpolate", nameKey: .stageInterpolate, icon: "point.topleft.filled.down.to.point.bottomright.curvepath"),
        PipelineStageItem(id: "geocode", nameKey: .stageGeocode, icon: "building.2.crop.circle.fill"),
        PipelineStageItem(id: "archive", nameKey: .stageArchive, icon: "calendar.badge.clock")
    ]

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // 1. 控制台头部状态栏
            headerBar

            // 2. 阶段流转指示器
            stageStepperBar

            // 3. 任务完成结算卡片 (若已完成)
            if case .succeeded = store.runState {
                taskCompletionCard
            } else if case .failed(let err) = store.runState {
                taskErrorCard(err)
            }

            // 4. 实时日志流窗口 - 撑满剩余全部可用高度
            terminalLogView
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    // 控制台头部
    private var headerBar: some View {
        HStack(spacing: 8) {
            if store.runState.isRunning {
                ProgressView()
                    .scaleEffect(0.7)
                    .frame(width: 16, height: 16)
                Text(lang.text(.pipelineRunning))
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(Color.accentColor)
            } else {
                Image(systemName: "terminal.fill")
                    .foregroundStyle(.secondary)
                Text(lang.text(.realtimeConsole))
                    .font(.subheadline.weight(.semibold))
            }

            Spacer()

            // 清除/重置当前任务状态
            if store.runState != .idle && !store.runState.isRunning {
                Button {
                    store.resetTaskStatus()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.counterclockwise.circle")
                        Text(lang.text(.clearStatus))
                    }
                    .font(.caption2.weight(.semibold))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help(lang.text(.clearStatusHelp))
                .pulseOnHover(scale: 1.05, glowColor: .secondary)
            }

            // 自动滚屏
            Toggle(isOn: $isAutoScroll) {
                Text(lang.text(.autoScroll))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            .toggleStyle(.checkbox)

            // 复制日志
            Button {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(store.liveLog, forType: .string)
            } label: {
                Image(systemName: "doc.on.doc")
                    .font(.caption)
            }
            .buttonStyle(.borderless)
            .help(lang.text(.copyAllLogs))
            .pulseOnHover(scale: 1.1, glowColor: .accentColor)

            // 清空日志
            Button {
                store.clearLiveLog()
            } label: {
                Image(systemName: "trash")
                    .font(.caption)
            }
            .buttonStyle(.borderless)
            .help(lang.text(.clearLogs))
            .pulseOnHover(scale: 1.1, glowColor: .red)
        }
    }

    private enum StageStatus {
        case pending    // 未执行 (灰底灰字)
        case running    // 正在执行 (蓝底蓝字 + Spinner 高亮)
        case completed  // 已完成 (绿底绿字 + 勾选)
        case failed     // 失败 (红底红字)
    }

    private func stageStatus(for index: Int) -> StageStatus {
        switch store.runState {
        case .succeeded:
            return .completed
        case .running:
            if store.currentStageIndex == index {
                return .running
            } else if store.currentStageIndex > index {
                return .completed
            } else {
                return .pending
            }
        case .failed:
            if store.currentStageIndex == index {
                return .failed
            } else if store.currentStageIndex > index {
                return .completed
            } else {
                return .pending
            }
        case .idle:
            return .pending
        }
    }

    // 阶段流转指示器
    private var stageStepperBar: some View {
        HStack(spacing: 4) {
            ForEach(Array(stages.enumerated()), id: \.element.id) { index, item in
                let status = stageStatus(for: index)

                HStack(spacing: 4) {
                    if status == .running {
                        ProgressView()
                            .scaleEffect(0.5)
                            .frame(width: 9, height: 9)
                    } else if status == .completed {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.system(size: 9, weight: .bold))
                    } else if status == .failed {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 9, weight: .bold))
                    } else {
                        Image(systemName: item.icon)
                            .font(.system(size: 9))
                    }

                    Text(lang.text(item.nameKey))
                        .font(.system(size: 10, weight: (status == .running || status == .completed) ? .semibold : .medium))
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 4)
                .background(stageBackground(for: status), in: RoundedRectangle(cornerRadius: 5, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 5, style: .continuous)
                        .stroke(status == .running ? Color.accentColor.opacity(0.4) : Color.clear, lineWidth: 1)
                )
                .foregroundStyle(stageForeground(for: status))

                if index != stages.count - 1 {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 7, weight: .bold))
                        .foregroundStyle(status == .completed ? Color.green.opacity(0.6) : Color.secondary.opacity(0.3))
                }
            }
        }
    }

    private func stageBackground(for status: StageStatus) -> Color {
        switch status {
        case .completed:
            return Color.green.opacity(0.16)
        case .running:
            return Color.accentColor.opacity(0.18)
        case .failed:
            return Color.red.opacity(0.16)
        case .pending:
            return Color.secondary.opacity(0.08)
        }
    }

    private func stageForeground(for status: StageStatus) -> Color {
        switch status {
        case .completed:
            return Color.green
        case .running:
            return Color.accentColor
        case .failed:
            return Color.red
        case .pending:
            return Color.secondary.opacity(0.6)
        }
    }

    // 任务成功完成卡片
    private var taskCompletionCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundStyle(.green)
                Text(lang.text(.taskCompletedTitle))
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(.green)
                Spacer()
            }

            HStack(spacing: 12) {
                if let processed = store.summary?.processedFileCount {
                    metricBadge(label: lang.text(.archivedCount), value: "\(processed)", color: .green)
                }
                if let ready = store.summary?.readyCount {
                    metricBadge(label: lang.text(.metricInbox), value: "\(ready)", color: .blue)
                }
            }

            HStack(spacing: 8) {
                Button {
                    if let path = store.summary?.logFilePath {
                        NSWorkspace.shared.open(URL(fileURLWithPath: path))
                    }
                } label: {
                    Label(lang.text(.viewFullLog), systemImage: "doc.text")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .pulseOnHover(scale: 1.04, glowColor: .green)

                if store.summary?.pendingReportExists == true {
                    Button {
                        if let path = store.summary?.pendingReportPath {
                            NSWorkspace.shared.open(URL(fileURLWithPath: path))
                        }
                    } label: {
                        Label(lang.text(.pendingReport), systemImage: "exclamationmark.triangle")
                            .font(.caption)
                    }
                    .buttonStyle(.bordered)
                    .pulseOnHover(scale: 1.04, glowColor: .orange)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.green.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.green.opacity(0.2), lineWidth: 1)
        )
    }

    // 任务失败卡片
    private func taskErrorCard(_ error: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "xmark.octagon.fill")
                    .foregroundStyle(.red)
                Text(lang.text(.taskErrorTitle))
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(.red)
                Spacer()

                Button {
                    store.resetTaskStatus()
                } label: {
                    Label(lang.text(.clearStatus), systemImage: "arrow.counterclockwise")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .pulseOnHover(scale: 1.04, glowColor: .red)
            }
            Text(error)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.red.opacity(0.2), lineWidth: 1)
        )
    }

    // 实时日志控制台视图 (撑满剩余高度，高性能无卡顿滚动黑客暗色终端)
    private var terminalLogView: some View {
        HackerTerminalView(
            logText: store.liveLog,
            isRunning: store.runState.isRunning,
            isAutoScroll: isAutoScroll,
            placeholder: lang.text(.waitingForTask)
        )
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func metricBadge(label: String, value: String, color: Color) -> some View {
        HStack(spacing: 4) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption.weight(.bold))
                .foregroundStyle(color)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(color.opacity(0.1), in: Capsule())
    }
}
