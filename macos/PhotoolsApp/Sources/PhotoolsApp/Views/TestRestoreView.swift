import PhotoolsCore
import SwiftUI

struct TestRestoreView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var cleanProcessed: Bool = true

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                headerView
                statusSection
                backupActionCard
                restoreActionCard
            }
            .padding(24)
        }
    }

    private var headerView: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Image(systemName: "arrow.triangle.2.circlepath.circle.fill")
                    .font(.title.weight(.bold))
                    .foregroundStyle(.purple)
                Text(lang.text(.testRestoreConsoleTitle))
                    .font(.title2.weight(.bold))
            }
            Text(lang.text(.testRestoreConsoleDesc))
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
    }

    private var statusSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(lang.text(.directoryStatusTitle))
                .font(.headline)

            HStack(spacing: 16) {
                statusBlock(
                    title: lang.text(.inboxSourcePhotos),
                    value: "\(store.summary?.readyCount ?? 0)",
                    subtitle: store.summary?.inboxDirectory ?? "",
                    icon: "tray.full.fill",
                    color: .green
                )

                statusBlock(
                    title: lang.text(.inboxBakSnapshot),
                    value: "\(store.summary?.backupFileCount ?? 0)",
                    subtitle: store.summary?.inboxBakDirectory ?? "",
                    icon: "externaldrive.fill.badge.checkmark",
                    color: (store.summary?.hasBackup == true) ? .purple : .secondary
                )

                statusBlock(
                    title: lang.text(.processedArchiveResult),
                    value: "\(store.summary?.processedFileCount ?? 0)",
                    subtitle: store.summary?.processedDirectory ?? "",
                    icon: "archivebox.fill",
                    color: .blue
                )
            }
        }
    }

    private func statusBlock(title: String, value: String, subtitle: String, icon: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Image(systemName: icon)
                    .foregroundStyle(color)
                Text(title)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Text(value)
                .font(.title3.weight(.bold))
            Text(subtitle)
                .font(.caption2)
                .foregroundStyle(.tertiary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // 操作 1：一键快照备份
    private var backupActionCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "camera.badge.ellipsis")
                    .font(.headline)
                    .foregroundStyle(.purple)
                Text(lang.text(.backupStep1Title))
                    .font(.headline)
            }

            Text(lang.text(.backupStep1Desc))
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 16) {
                Button {
                    store.createBackup()
                } label: {
                    Label(lang.text(.backupNowBtn), systemImage: "arrow.down.doc.fill")
                        .font(.body.weight(.semibold))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 6)
                }
                .buttonStyle(.borderedProminent)
                .tint(.purple)
                .disabled(store.runState.isRunning || store.summary?.readyCount == 0)

                if store.summary?.readyCount == 0 {
                    Text(lang.text(.inboxEmptyWarning))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // 操作 2：一键快照还原
    private var restoreActionCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "arrow.counterclockwise.circle.fill")
                    .font(.headline)
                    .foregroundStyle(.orange)
                Text(lang.text(.restoreStep2Title))
                    .font(.headline)
            }

            Text(lang.text(.restoreStep2Desc))
                .font(.caption)
                .foregroundStyle(.secondary)

            Toggle(lang.text(.cleanProcessedToggle), isOn: $cleanProcessed)
                .font(.body)

            HStack(spacing: 16) {
                Button(role: .destructive) {
                    store.restoreTest(cleanProcessed: cleanProcessed)
                } label: {
                    Label(lang.text(.restoreNowBtn), systemImage: "arrow.counterclockwise")
                        .font(.body.weight(.semibold))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 6)
                }
                .buttonStyle(.borderedProminent)
                .disabled(store.runState.isRunning || store.summary?.hasBackup != true)

                if store.summary?.hasBackup != true {
                    Text(lang.text(.noBackupAvailableWarning))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }
}
