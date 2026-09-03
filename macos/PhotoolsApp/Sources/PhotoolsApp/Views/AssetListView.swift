import PhotoolsCore
import QuickLook
import SwiftUI

struct AssetListView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var filterType: FilterType = .all
    @State private var searchText: String = ""
    @FocusState private var isSearchFocused: Bool

    enum FilterType: String, CaseIterable, Identifiable {
        case all
        case hasGPS
        case noGPS
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
            case .hasGPS:
                return LanguageManager.shared.text(.filterHasGPS)
            case .noGPS:
                return LanguageManager.shared.text(.filterNoGPS)
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

    @State private var eventMonitor: Any?

    var body: some View {
        content
            .navigationTitle(store.selectedSection.title)
            .quickLookPreview($store.quickLookURL)
            .onAppear {
                setupKeyboardMonitor()
            }
            .onDisappear {
                removeKeyboardMonitor()
            }
    }

    private var currentDisplayedAssets: [PhotoAssetGroup] {
        switch store.selectedSection {
        case .inbox:
            return filteredAssets
        case .processed:
            return filteredProcessedAssets
        default:
            return []
        }
    }

    private func selectNext() {
        let assets = currentDisplayedAssets
        guard !assets.isEmpty else { return }
        guard let currentID = store.selectedAssetID, let idx = assets.firstIndex(where: { $0.id == currentID }) else {
            store.selectedAssetID = assets.first?.id
            return
        }
        let nextIdx = min(idx + 1, assets.count - 1)
        if nextIdx != idx {
            store.selectedAssetID = assets[nextIdx].id
        }
    }

    private func selectPrevious() {
        let assets = currentDisplayedAssets
        guard !assets.isEmpty else { return }
        guard let currentID = store.selectedAssetID, let idx = assets.firstIndex(where: { $0.id == currentID }) else {
            store.selectedAssetID = assets.first?.id
            return
        }
        let prevIdx = max(idx - 1, 0)
        if prevIdx != idx {
            store.selectedAssetID = assets[prevIdx].id
        }
    }

    private func setupKeyboardMonitor() {
        guard eventMonitor == nil else { return }
        eventMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            // 1. 若当前第一响应者为文本输入组件 (NSTextView / Field Editor)，绝不拦截按键
            if let responder = NSApp.keyWindow?.firstResponder, responder is NSTextView {
                return event
            }

            // 2. 仅在待选照片或归档照片视图下响应
            guard store.selectedSection == .inbox || store.selectedSection == .processed else {
                return event
            }

            switch event.keyCode {
            case 49: // 空格键 Space：唤起/关闭 QuickLook 快速大图预览
                DispatchQueue.main.async {
                    store.toggleQuickLookForSelectedAsset()
                }
                return nil

            case 125, 124: // 下方向键 ↓ 或 右方向键 →：切换到下一张照片
                DispatchQueue.main.async {
                    selectNext()
                }
                return nil

            case 126, 123: // 上方向键 ↑ 或 左方向键 ←：切换到上一张照片
                DispatchQueue.main.async {
                    selectPrevious()
                }
                return nil

            default:
                return event
            }
        }
    }

    private func removeKeyboardMonitor() {
        if let monitor = eventMonitor {
            NSEvent.removeMonitor(monitor)
            eventMonitor = nil
        }
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
                Picker(lang.text(.filterLabel), selection: $filterType) {
                    ForEach(FilterType.allCases) { type in
                        Text(type.title).tag(type)
                    }
                }
                .pickerStyle(.menu)
                .fixedSize()

                HStack(spacing: 4) {
                    Image(systemName: "magnifyingglass")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    TextField(lang.text(.searchPlaceholder), text: $searchText)
                        .textFieldStyle(.plain)
                        .font(.caption)
                        .focused($isSearchFocused)
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

            ScrollViewReader { proxy in
                List(selection: $store.selectedAssetID) {
                    ForEach(filteredAssets) { asset in
                        AssetRow(store: store, asset: asset, onDelete: {
                            deleteAssetGroup(asset)
                        })
                        .tag(asset.id)
                        .id(asset.id)
                    }
                }
                .listStyle(.inset(alternatesRowBackgrounds: true))
                .onChange(of: store.selectedAssetID) { newID in
                    if let id = newID {
                        withAnimation(.easeInOut(duration: 0.15)) {
                            proxy.scrollTo(id, anchor: .center)
                        }
                    }
                }
            }
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
            case .hasGPS:
                matchesFilter = (store.assetGPSMap[asset.id] == true)
            case .noGPS:
                matchesFilter = (store.assetGPSMap[asset.id] == false)
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

    private var filteredProcessedAssets: [PhotoAssetGroup] {
        let assets = store.processedCurrentAssets
        return assets.filter { asset in
            let matchesFilter: Bool
            switch filterType {
            case .all:
                matchesFilter = true
            case .hasGPS:
                matchesFilter = (store.assetGPSMap[asset.id] == true)
            case .noGPS:
                matchesFilter = (store.assetGPSMap[asset.id] == false)
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
        VStack(spacing: 0) {
            // 1. 顶部面包屑路径导航、冰蓝冻结徽章与平铺切换栏
            HStack(spacing: 8) {
                // 面包屑导航
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 4) {
                        Button {
                            store.resetProcessedNavigation()
                        } label: {
                            HStack(spacing: 3) {
                                Image(systemName: "archivebox.fill")
                                    .font(.system(size: 10))
                                Text("Processed")
                                    .font(.caption.weight(store.processedCurrentSubdir.isEmpty ? .bold : .medium))
                            }
                            .foregroundStyle(store.processedCurrentSubdir.isEmpty ? Color.cyan : Color.secondary)
                        }
                        .buttonStyle(.plain)

                        let crumbs = store.processedBreadcrumbs
                        ForEach(Array(crumbs.enumerated()), id: \.offset) { idx, crumb in
                            Image(systemName: "chevron.right")
                                .font(.system(size: 8))
                                .foregroundStyle(.tertiary)

                            Button {
                                store.navigateToProcessedBreadcrumb(at: idx)
                            } label: {
                                Text(crumb)
                                    .font(.caption.weight(idx == crumbs.count - 1 ? .bold : .medium))
                                    .foregroundStyle(idx == crumbs.count - 1 ? Color.cyan : Color.secondary)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }

                Spacer()

                // 极地冰蓝冻结只读徽章
                HStack(spacing: 3) {
                    Image(systemName: "snowflake")
                        .font(.system(size: 9))
                    Text("已冻结")
                        .font(.system(size: 10, weight: .bold))
                }
                .foregroundStyle(Color.cyan)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.cyan.opacity(0.12), in: Capsule())

                // 平铺/层级下探切换按钮
                Button {
                    store.processedIsFlatRecursive.toggle()
                    store.selectedAssetID = store.processedCurrentAssets.first?.id
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: store.processedIsFlatRecursive ? "square.grid.2x2.fill" : "folder.fill")
                            .font(.system(size: 9))
                        Text(store.processedIsFlatRecursive ? "平铺" : "目录")
                            .font(.system(size: 10, weight: .medium))
                    }
                    .foregroundStyle(store.processedIsFlatRecursive ? Color.cyan : Color.secondary)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 4))
                }
                .buttonStyle(.plain)
                .help(store.processedIsFlatRecursive ? "当前为递归平铺所有照片，点击切换为按子目录层级浏览" : "当前为按目录层级浏览，点击切换为递归平铺所有归档照片")
            }
            .padding(.horizontal, 10)
            .padding(.top, 8)
            .padding(.bottom, 4)

            // 2. 筛选与搜索条
            HStack(spacing: 8) {
                Picker(lang.text(.filterLabel), selection: $filterType) {
                    ForEach(FilterType.allCases) { type in
                        Text(type.title).tag(type)
                    }
                }
                .pickerStyle(.menu)
                .fixedSize()

                HStack(spacing: 4) {
                    Image(systemName: "magnifyingglass")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    TextField(lang.text(.searchPlaceholder), text: $searchText)
                        .textFieldStyle(.plain)
                        .font(.caption)
                        .focused($isSearchFocused)
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
            .padding(.bottom, 6)

            Divider()

            // 3. 目录与照片列表
            ScrollViewReader { proxy in
                List(selection: $store.selectedAssetID) {
                    // A. 归档子目录列表 (非平铺模式)
                    if !store.processedIsFlatRecursive && !store.processedDrillDownFolders.isEmpty {
                        Section {
                            ForEach(store.processedDrillDownFolders) { folder in
                                Button {
                                    store.enterProcessedFolder(folder.relativePath)
                                } label: {
                                    HStack(spacing: 10) {
                                        Image(systemName: "folder.fill")
                                            .font(.body)
                                            .foregroundStyle(Color.cyan)
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(folder.name)
                                                .font(.body.weight(.medium))
                                                .foregroundStyle(Color.primary)
                                            Text("\(folder.photoCount) 组归档照片")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Image(systemName: "chevron.right")
                                            .font(.caption2)
                                            .foregroundStyle(.tertiary)
                                    }
                                    .padding(.vertical, 3)
                                    .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                            }
                        } header: {
                            Text("归档子目录 (\(store.processedDrillDownFolders.count))")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.secondary)
                        }
                    }

                    // B. 已归档照片列表
                    if !filteredProcessedAssets.isEmpty {
                        Section {
                            ForEach(filteredProcessedAssets) { asset in
                                ArchivedAssetRow(store: store, asset: asset)
                                    .tag(asset.id)
                                    .id(asset.id)
                            }
                        } header: {
                            Text(store.processedIsFlatRecursive ? "全部已归档照片 (\(filteredProcessedAssets.count))" : "当前目录照片 (\(filteredProcessedAssets.count))")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.secondary)
                        }
                    }
                }
                .listStyle(.inset(alternatesRowBackgrounds: true))
                .onChange(of: store.selectedAssetID) { newID in
                    if let id = newID {
                        withAnimation(.easeInOut(duration: 0.15)) {
                            proxy.scrollTo(id, anchor: .center)
                        }
                    }
                }
            }
            .overlay {
                if filteredProcessedAssets.isEmpty && store.processedDrillDownFolders.isEmpty {
                    EmptyStateView(
                        title: store.summary?.processedFileCount == 0 ? "归档库暂无照片" : "当前目录下无匹配照片",
                        systemImage: "archivebox"
                    )
                }
            }
        }
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
    @ObservedObject var store: WorkspaceStore
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
            Button {
                store.selectedAssetID = asset.id
                store.toggleQuickLookForSelectedAsset()
            } label: {
                Label("快速预览 (空格)", systemImage: "eye")
            }

            Divider()

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

            Button {
                store.selectedAssetID = asset.id
                store.copySelectedAssetGPS()
            } label: {
                Label(lang.text(.copyGPSAction), systemImage: "doc.on.doc")
            }

            if store.copiedGPSMetadata != nil {
                Button {
                    store.selectedAssetID = asset.id
                    store.pasteGPSToSelectedAsset()
                } label: {
                    Label(lang.text(.pasteGPSAction), systemImage: "location.fill.viewfinder")
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
            // GPS 状态 Tag (绿色 GPS，灰色 无GPS)
            if let hasGPS = store.assetGPSMap[asset.id] {
                if hasGPS {
                    HStack(spacing: 2) {
                        Image(systemName: "location.fill")
                            .font(.system(size: 7))
                        Text("GPS")
                            .font(.system(size: 9, weight: .bold))
                    }
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(Color.green.opacity(0.18), in: RoundedRectangle(cornerRadius: 4))
                    .foregroundStyle(.green)
                } else {
                    Text(lang.text(.filterNoGPS))
                        .font(.system(size: 8, weight: .semibold))
                        .padding(.horizontal, 4)
                        .padding(.vertical, 2)
                        .background(Color.secondary.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .foregroundStyle(.secondary)
                }
            }

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

// MARK: - 已归档照片资产行 (极地冰蓝冻结只读视觉质感)
private struct ArchivedAssetRow: View {
    @ObservedObject var store: WorkspaceStore
    let asset: PhotoAssetGroup
    @ObservedObject private var lang = LanguageManager.shared

    var body: some View {
        HStack(spacing: 10) {
            // 左侧等比缩略图预览
            ZStack(alignment: .bottomTrailing) {
                PhotoThumbnailView(
                    asset: asset,
                    targetSize: CGSize(width: 84, height: 84),
                    contentMode: .fill,
                    cornerRadius: 6,
                    fixedSize: CGSize(width: 42, height: 42)
                )

                // 冰蓝冻结微角标
                Image(systemName: "snowflake")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(2)
                    .background(Color.cyan.opacity(0.85), in: Circle())
                    .offset(x: 2, y: 2)
            }

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 4) {
                    Image(systemName: "lock.shield.fill")
                        .foregroundStyle(Color.cyan)
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

                    // 相对归档子路径提示 (如 2026/0101)
                    if let relDir = relativeSubdirectory {
                        Text("• " + relDir)
                            .font(.caption2.monospaced())
                            .foregroundStyle(Color.cyan.opacity(0.85))
                    }
                }
            }

            Spacer()

            // 右侧格式与冻结角标
            shortFileBadges
        }
        .padding(.vertical, 2)
        .contentShape(Rectangle())
        .contextMenu {
            Button {
                store.selectedAssetID = asset.id
                store.toggleQuickLookForSelectedAsset()
            } label: {
                Label("快速预览 (空格)", systemImage: "eye")
            }

            Divider()

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
            } else if let primary = asset.primaryPath {
                Button {
                    NSWorkspace.shared.open(URL(fileURLWithPath: primary))
                } label: {
                    Label(lang.text(.openFile), systemImage: "arrow.up.right.square")
                }
            }

            Divider()

            Button {
                store.copySelectedAssetGPS()
            } label: {
                Label(lang.text(.copyGPSAction), systemImage: "doc.on.doc")
            }
        }
    }

    private var relativeSubdirectory: String? {
        guard let procDir = store.summary?.processedDirectory, !procDir.isEmpty else { return nil }
        let dir = asset.directory
        guard dir.hasPrefix(procDir) else { return nil }
        var rel = String(dir.dropFirst(procDir.count))
        if rel.hasPrefix("/") {
            rel.removeFirst()
        }
        return rel.isEmpty ? nil : rel
    }

    @ViewBuilder
    private var shortFileBadges: some View {
        HStack(spacing: 3) {
            if asset.hasRaw {
                Text("RAW")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.purple)
                    .padding(.horizontal, 3)
                    .padding(.vertical, 1)
                    .background(Color.purple.opacity(0.12), in: RoundedRectangle(cornerRadius: 3))
            }
            if asset.hasJpg {
                Text("JPG")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.blue)
                    .padding(.horizontal, 3)
                    .padding(.vertical, 1)
                    .background(Color.blue.opacity(0.12), in: RoundedRectangle(cornerRadius: 3))
            }
            if asset.hasXmp {
                Text("XMP")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.teal)
                    .padding(.horizontal, 3)
                    .padding(.vertical, 1)
                    .background(Color.teal.opacity(0.12), in: RoundedRectangle(cornerRadius: 3))
            }
        }
    }
}
