import PhotoolsCore
import SwiftUI

struct AssetListView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var filterType: FilterType = .all
    @State private var searchText: String = ""

    enum FilterType: String, CaseIterable, Identifiable {
        case all
        case rawPair
        case rawOnly
        case jpgOnly
        case companionOnly

        var id: String { rawValue }

        @MainActor
        var title: String {
            switch self {
            case .all:
                return LanguageManager.shared.text(.filterAll)
            case .rawPair:
                return LanguageManager.shared.text(.filterRawPair)
            case .rawOnly:
                return LanguageManager.shared.text(.filterRawOnly)
            case .jpgOnly:
                return LanguageManager.shared.text(.filterJpgOnly)
            case .companionOnly:
                return LanguageManager.shared.text(.filterCompanionOnly)
            }
        }
    }

    var body: some View {
        content
            .navigationTitle(store.selectedSection.title)
    }

    @ViewBuilder
    private var content: some View {
        switch store.selectedSection {
        case .pipeline:
            PipelineDashboardView(store: store)
        case .inbox:
            inboxView
        case .gpx:
            fileList(paths: store.summary?.gpxFiles ?? [], emptyText: lang.text(.emptyGpxFiles), systemImage: "map")
        case .processed:
            processedView
        case .geodata:
            GeodataManagerView(store: store)
        case .testRestore:
            TestRestoreView(store: store)
        case .guide:
            GuideListView(store: store)
        }
    }

    private var inboxView: some View {
        VStack(spacing: 0) {
            // 顶部搜索与类型过滤工具栏 (弹性自适应宽度，绝不溢出)
            HStack(spacing: 8) {
                Picker(lang.text(.filterAll), selection: $filterType) {
                    ForEach(FilterType.allCases) { type in
                        Text(type.title).tag(type)
                    }
                }
                .pickerStyle(.menu)
                .frame(width: 110)

                HStack(spacing: 4) {
                    Image(systemName: "magnifyingglass")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    TextField(lang.text(.searchPlaceholder), text: $searchText)
                        .textFieldStyle(.plain)
                        .font(.caption)
                    if !searchText.isEmpty {
                        Button {
                            searchText = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 6))
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)

            Divider()

            List(selection: $store.selectedAssetID) {
                ForEach(filteredAssets) { asset in
                    AssetRow(asset: asset, onDelete: {
                        deleteAssetGroup(asset)
                    })
                    .tag(asset.id)
                }
            }
            .listStyle(.inset(alternatesRowBackgrounds: true))
            .overlay {
                if filteredAssets.isEmpty {
                    EmptyStateView(
                        title: store.summary?.assetGroups.isEmpty == true ? lang.text(.emptyInbox) : lang.text(.emptyFiltered),
                        systemImage: "tray"
                    )
                }
            }
        }
    }

    private func deleteAssetGroup(_ asset: PhotoAssetGroup) {
        let fm = FileManager.default
        for path in asset.allFiles {
            let url = URL(fileURLWithPath: path)
            do {
                try fm.trashItem(at: url, resultingItemURL: nil)
            } catch {
                try? fm.removeItem(at: url)
            }
        }
        store.refresh()
    }

    private var filteredAssets: [PhotoAssetGroup] {
        guard let assets = store.summary?.assetGroups else { return [] }
        return assets.filter { asset in
            let matchesFilter: Bool
            switch filterType {
            case .all:
                matchesFilter = true
            case .rawPair:
                matchesFilter = asset.primaryType == .rawPair
            case .rawOnly:
                matchesFilter = asset.primaryType == .rawOnly
            case .jpgOnly:
                matchesFilter = asset.primaryType == .jpgOnly
            case .companionOnly:
                matchesFilter = asset.primaryType == .companionOnly
            }

            if !matchesFilter { return false }
            if searchText.isEmpty { return true }
            return asset.baseName.localizedCaseInsensitiveContains(searchText)
        }
    }

    private var processedView: some View {
        VStack(alignment: .leading, spacing: 16) {
            VStack(alignment: .leading, spacing: 6) {
                Text(lang.text(.processedTotalFiles))
                    .font(.headline)
                Text("\(store.summary?.processedFileCount ?? 0)")
                    .font(.system(size: 44, weight: .bold, design: .rounded))
                    .foregroundStyle(.blue)
                Text(store.summary?.processedDirectory ?? "")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            .padding(18)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))

            Button {
                if let path = store.summary?.processedDirectory {
                    NSWorkspace.shared.open(URL(fileURLWithPath: path))
                }
            } label: {
                Label(lang.text(.openInFinder), systemImage: "folder")
            }

            Spacer()
        }
        .padding(20)
    }

    private func fileList(paths: [String], emptyText: String, systemImage: String) -> some View {
        List(paths, id: \.self) { path in
            HStack(spacing: 12) {
                Image(systemName: "doc.text.fill")
                    .foregroundStyle(.blue)
                VStack(alignment: .leading, spacing: 2) {
                    Text(URL(fileURLWithPath: path).lastPathComponent)
                        .font(.body.weight(.medium))
                        .lineLimit(1)
                    Text(path)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }
            .padding(.vertical, 3)
            .contextMenu {
                Button {
                    NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
                } label: {
                    Label(lang.text(.showInFinder), systemImage: "folder")
                }
                Button {
                    NSWorkspace.shared.open(URL(fileURLWithPath: path))
                } label: {
                    Label(lang.text(.openFile), systemImage: "arrow.up.right.square")
                }
            }
        }
        .listStyle(.inset(alternatesRowBackgrounds: true))
        .overlay {
            if paths.isEmpty {
                EmptyStateView(title: emptyText, systemImage: systemImage)
            }
        }
    }
}

private struct EmptyStateView: View {
    let title: String
    let systemImage: String

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: systemImage)
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text(title)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

private struct AssetRow: View {
    let asset: PhotoAssetGroup
    var onDelete: () -> Void = {}
    @ObservedObject private var lang = LanguageManager.shared

    var body: some View {
        HStack(spacing: 10) {
            // 左侧等比缩略图预览 (固定 42x42)
            PhotoThumbnailView(
                asset: asset,
                targetSize: CGSize(width: 84, height: 84),
                contentMode: .fill,
                cornerRadius: 6,
                fixedSize: CGSize(width: 42, height: 42)
            )

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 4) {
                    Image(systemName: iconName)
                        .foregroundStyle(iconColor)
                        .font(.system(size: 10))

                    Text(asset.baseName)
                        .font(.system(.body, design: .default).weight(.medium))
                        .lineLimit(1)
                }

                HStack(spacing: 4) {
                    Text(asset.primaryType.title)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    if let primaryPath = asset.primaryPath {
                        Text("• " + (primaryPath as NSString).pathExtension.uppercased())
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(.tertiary)
                    }
                }
            }

            Spacer()

            shortFileBadges
        }
        .padding(.vertical, 2)
        .contentShape(Rectangle())
        .contextMenu {
            if let primaryPath = asset.primaryPath {
                Button {
                    NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: primaryPath)])
                } label: {
                    Label(lang.text(.showInFinder), systemImage: "folder")
                }
            }

            if let jpg = asset.jpgPath {
                Button {
                    NSWorkspace.shared.open(URL(fileURLWithPath: jpg))
                } label: {
                    Label(lang.text(.openJpgPreview), systemImage: "photo")
                }
            }

            if let raw = asset.rawPath {
                Button {
                    NSWorkspace.shared.open(URL(fileURLWithPath: raw))
                } label: {
                    Label(lang.text(.openRawOriginal), systemImage: "camera.fill")
                }
            }

            Divider()

            Button(role: .destructive) {
                onDelete()
            } label: {
                Label(lang.text(.moveToTrashDelete), systemImage: "trash")
            }
        }
    }

    private var iconName: String {
        switch asset.status {
        case .ready:
            return "checkmark.circle.fill"
        case .companionOnly:
            return "questionmark.circle"
        }
    }

    private var iconColor: Color {
        switch asset.status {
        case .ready:
            return .green
        case .companionOnly:
            return .secondary
        }
    }

    private var shortFileBadges: some View {
        HStack(spacing: 4) {
            if asset.rawPath != nil {
                badge("RAW", color: .purple)
            }
            if asset.jpgPath != nil {
                badge("JPG", color: .blue)
            }
            if asset.xmpPath != nil {
                badge("XMP", color: .orange)
            }
            if !asset.companionPaths.isEmpty {
                let otherCount = asset.companionPaths.filter { $0 != asset.xmpPath }.count
                if otherCount > 0 {
                    badge("+\(otherCount)", color: .gray)
                }
            }
        }
    }

    private func badge(_ text: String, color: Color) -> some View {
        Text(text)
            .font(.system(size: 9, weight: .bold))
            .padding(.horizontal, 4)
            .padding(.vertical, 2)
            .background(color.opacity(0.15), in: RoundedRectangle(cornerRadius: 4))
            .foregroundStyle(color)
    }
}
