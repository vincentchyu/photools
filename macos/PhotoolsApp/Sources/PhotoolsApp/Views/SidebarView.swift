import PhotoolsCore
import SwiftUI

public struct SidebarView: View {
    @ObservedObject var store: WorkspaceStore

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        List(selection: $store.selectedSection) {
            Section(LanguageManager.shared.text(.groupWorkspace)) {
                VStack(alignment: .leading, spacing: 3) {
                    HStack {
                        Image(systemName: "folder.fill")
                            .foregroundStyle(.blue)
                        Text(LanguageManager.shared.text(.groupRootDirectory))
                            .font(.caption.weight(.semibold))
                    }
                    Text(store.baseDirectory)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                        .textSelection(.enabled)
                }
                .padding(.vertical, 2)
            }

            Section(LanguageManager.shared.text(.groupWorkflow)) {
                sidebarRow(for: .pipeline, color: .blue)
                sidebarRow(for: .inbox, color: .purple, badgeCount: store.summary?.readyCount)
                sidebarRow(for: .processed, color: .indigo, badgeCount: store.summary?.processedFileCount)
            }

            Section(LanguageManager.shared.text(.groupTools)) {
                sidebarRow(for: .geodata, color: .teal)
                sidebarRow(for: .gpx, color: .cyan, badgeCount: store.summary?.gpxFiles.count)
                sidebarRow(for: .testRestore, color: .orange)
                sidebarRow(for: .guide, color: .secondary)
            }
        }
        .listStyle(.sidebar)
        .navigationTitle("photools")
    }

    private func sidebarRow(for section: WorkspaceSection, color: Color, badgeCount: Int? = nil) -> some View {
        NavigationLink(value: section) {
            HStack {
                Label {
                    Text(section.title)
                        .font(.system(.body, weight: .regular))
                } icon: {
                    Image(systemName: section.systemImage)
                        .foregroundStyle(color)
                }

                Spacer()

                if section == .pipeline && store.runState.isRunning {
                    ProgressView()
                        .scaleEffect(0.6)
                        .frame(width: 12, height: 12)
                } else if let count = badgeCount, count > 0 {
                    Text("\(count)")
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 1)
                        .background(Color.secondary.opacity(0.12), in: Capsule())
                }
            }
        }
    }
}
