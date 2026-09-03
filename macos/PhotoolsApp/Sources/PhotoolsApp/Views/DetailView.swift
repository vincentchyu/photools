import PhotoolsCore
import SwiftUI

public struct DetailView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        Group {
            switch store.selectedSection {
            case .pipeline:
                PipelineExecutionConsole(store: store)

            case .inbox:
                if let asset = store.selectedAsset {
                    PhotoExifInspectorView(store: store, asset: asset)
                } else {
                    EmptyDetailStateView(
                        title: lang.text(.noPhotoSelectedTitle),
                        subtitle: lang.text(.noPhotoSelectedSubtitle)
                    )
                }

            case .processed:
                if let asset = store.selectedAsset {
                    PhotoExifInspectorView(store: store, asset: asset, isArchivedFrozen: true)
                } else {
                    processedInspectorView
                }

            case .geodata:
                GeodataTestDetailView(store: store)

            case .gpx:
                gpxInspectorView

            case .testRestore:
                testRestoreHelpDetail

            case .guide:
                GuideDocDetailView(store: store)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var processedInspectorView: some View {
        VStack(alignment: .leading, spacing: 16) {
            Label(lang.text(.processedArchiveInfo), systemImage: "archivebox.fill")
                .font(.headline)

            VStack(alignment: .leading, spacing: 8) {
                Text(lang.text(.processedArchiveCount))
                    .font(.caption)
                    .foregroundStyle(.secondary)
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text("\(store.summary?.processedAssetGroupCount ?? 0)")
                        .font(.system(size: 36, weight: .bold, design: .rounded))
                        .foregroundStyle(.indigo)
                    Text("组照片 (\(store.summary?.processedFileCount ?? 0) 个物理文件)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                if let path = store.summary?.processedDirectory {
                    Text(path)
                        .font(.caption2.monospaced())
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                        .textSelection(.enabled)
                }
            }
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 10))

            Button {
                if let path = store.summary?.processedDirectory {
                    NSWorkspace.shared.open(URL(fileURLWithPath: path))
                }
            } label: {
                Label(lang.text(.openInFinder), systemImage: "folder")
            }

            Spacer()
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var gpxInspectorView: some View {
        VStack(alignment: .leading, spacing: 16) {
            Label(lang.text(.gpxMatchOverview), systemImage: "map.fill")
                .font(.headline)

            VStack(alignment: .leading, spacing: 8) {
                Text(lang.text(.gpxFilesCount))
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Text("\(store.summary?.gpxFiles.count ?? 0)")
                    .font(.system(size: 36, weight: .bold, design: .rounded))
                    .foregroundStyle(.blue)
            }
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 10))

            Text(lang.text(.gpxPipelineDesc))
                .font(.caption)
                .foregroundStyle(.secondary)

            Spacer()
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var testRestoreHelpDetail: some View {
        VStack(alignment: .leading, spacing: 14) {
            Label(lang.text(.testRestoreMechTitle), systemImage: "arrow.counterclockwise.circle.fill")
                .font(.headline)

            Text(lang.text(.testRestoreMechDesc))
                .font(.caption)
                .foregroundStyle(.secondary)

            Spacer()
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
}

public struct EmptyDetailStateView: View {
    let title: String
    let subtitle: String

    public init(title: String, subtitle: String) {
        self.title = title
        self.subtitle = subtitle
    }

    public var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 44))
                .foregroundStyle(.secondary.opacity(0.6))
            Text(title)
                .font(.headline)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
