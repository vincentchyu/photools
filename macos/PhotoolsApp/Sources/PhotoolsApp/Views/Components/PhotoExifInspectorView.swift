import PhotoolsCore
import SwiftUI

private actor PreviewTextCollector {
    var text: String = ""
    func append(_ str: String) { text.append(str) }
    func getText() -> String { text }
}

public struct PhotoExifInspectorView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    let asset: PhotoAssetGroup

    @State private var tagSearchText: String = ""
    @State private var isRawTagsExpanded: Bool = false
    @State private var previewGeocodeResult: GeodataLookupResult?
    @State private var isPreviewLoading: Bool = false

    public init(store: WorkspaceStore, asset: PhotoAssetGroup) {
        self.store = store
        self.asset = asset
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // 0. 高清图片预览卡片
                largePhotoPreviewCard

                // 1. 资产头部摘要
                assetHeaderCard

                // 2. 核心拍摄参数卡片
                cameraAndExposureCard

                // 3. 地理位置与即时反查预览
                locationAndGeocodeCard

                // 3.5 元数据溯源与审计指纹 (若有)
                if let exif = store.selectedAssetExif, exif.hasProvenance {
                    provenanceAuditCard(exif)
                }

                // 4. 伴随文件与主资产关系
                companionFilesCard

                // 5. 待处理/异常原因分析 (若有)
                if let pendingSection = store.pendingReportSection(for: asset) {
                    pendingReportCard(pendingSection)
                }

                // 6. 原始 ExifTool 标签树浏览器
                rawExifTagsCard
            }
            .padding(12)
        }
        .onAppear {
            previewGeocodeResult = nil
            store.loadExifForSelectedAsset()
        }
        .onChange(of: asset.id) { _ in
            previewGeocodeResult = nil
        }
    }

    // 高清图片预览卡片
    private var largePhotoPreviewCard: some View {
        ZStack(alignment: .bottomTrailing) {
            PhotoThumbnailView(
                asset: asset,
                targetSize: CGSize(width: 900, height: 600),
                contentMode: .fill,
                cornerRadius: 8
            )
            .aspectRatio(3.0 / 2.0, contentMode: .fit)
            .frame(maxWidth: .infinity)
            .contentShape(Rectangle())
            .onTapGesture(count: 2) {
                openAssetPreview(asset)
            }

            // 底部浮动工具与提示栏
            HStack(spacing: 6) {
                if let primaryPath = asset.primaryPath {
                    Text((primaryPath as NSString).pathExtension.uppercased())
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.black.opacity(0.65), in: RoundedRectangle(cornerRadius: 4))
                }

                Spacer()

                // 在访达中定位高亮
                Button {
                    if let path = asset.primaryPath {
                        NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
                    }
                } label: {
                    Label(lang.text(.showInFinder), systemImage: "folder")
                        .font(.caption2.weight(.medium))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help(lang.text(.locateInFinderHelp))

                // 打开大图预览
                Button {
                    openAssetPreview(asset)
                } label: {
                    Label(asset.jpgPath != nil ? lang.text(.openLargeJpg) : lang.text(.openOriginalImage), systemImage: "arrow.up.right.square")
                        .font(.caption2.weight(.medium))
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
            }
            .padding(8)
        }
        .frame(maxWidth: .infinity)
        .background(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(Color(nsColor: .controlBackgroundColor).opacity(0.8))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func openAssetPreview(_ asset: PhotoAssetGroup) {
        if let jpg = asset.jpgPath {
            NSWorkspace.shared.open(URL(fileURLWithPath: jpg))
        } else if let primary = asset.primaryPath {
            NSWorkspace.shared.open(URL(fileURLWithPath: primary))
        }
    }

    // 资产头部摘要
    private var assetHeaderCard: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(asset.baseName)
                        .font(.headline.weight(.bold))
                        .textSelection(.enabled)

                    HStack(spacing: 6) {
                        typeBadge(for: asset.primaryType)

                        if let exif = store.selectedAssetExif, !exif.fileSize.isEmpty {
                            Text("·")
                                .foregroundStyle(.secondary)
                            Text(exif.fileSize)
                                .font(.caption2.monospaced())
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                Spacer()

                if store.isExifLoading {
                    ProgressView()
                        .scaleEffect(0.7)
                        .frame(width: 16, height: 16)
                }
            }

            if let path = asset.primaryPath {
                HStack(spacing: 4) {
                    Text(path)
                        .font(.caption2.monospaced())
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                        .textSelection(.enabled)

                    Button {
                        NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
                    } label: {
                        Image(systemName: "folder")
                            .font(.caption2)
                    }
                    .buttonStyle(.plain)
                    .help(lang.text(.showInFinder))
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // 核心拍摄参数卡片
    private var cameraAndExposureCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label(lang.text(.shootingParamsTitle), systemImage: "camera.fill")
                .font(.headline)
                .foregroundStyle(.primary)

            if let exif = store.selectedAssetExif {
                VStack(alignment: .leading, spacing: 8) {
                    // 相机与镜头
                    VStack(alignment: .leading, spacing: 2) {
                        Text(exif.cameraSummary)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.primary)
                        if let lens = exif.lensModel, !lens.isEmpty {
                            Text(lens)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }

                    Divider()

                    // 曝光四要素
                    HStack(spacing: 6) {
                        exposureItem(icon: "timer", label: lang.text(.shutterSpeed), value: formatExposure(exif.exposureTime))
                        exposureItem(icon: "camera.aperture", label: lang.text(.aperture), value: formatFNumber(exif.fNumber))
                        exposureItem(icon: "sun.max.fill", label: "ISO", value: exif.iso ?? "--")
                        exposureItem(icon: "scope", label: lang.text(.focalLength), value: formatFocal(exif.focalLength))
                    }

                    if let date = exif.dateTimeOriginal, !date.isEmpty {
                        HStack(spacing: 6) {
                            Image(systemName: "calendar")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                            Text(lang.text(.captureTime) + ":")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                            Text(date)
                                .font(.caption2.monospaced())
                                .foregroundStyle(.primary)
                        }
                    }
                }
            } else if store.isExifLoading {
                HStack {
                    Spacer()
                    ProgressView(lang.text(.readingExif))
                        .font(.caption)
                    Spacer()
                }
                .padding(10)
            } else {
                Text(lang.text(.noExifRead))
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func formatExposure(_ exp: String?) -> String {
        guard let exp, !exp.isEmpty, exp != "--" else { return "--" }
        return exp.hasSuffix("s") ? exp : "\(exp)s"
    }

    private func formatFNumber(_ fn: String?) -> String {
        guard let fn, !fn.isEmpty, fn != "--" else { return "--" }
        if fn.lowercased().starts(with: "f/") {
            return fn
        }
        return "f/\(fn)"
    }

    private func formatFocal(_ fl: String?) -> String {
        guard let fl, !fl.isEmpty, fl != "--" else { return "--" }
        let clean = fl.components(separatedBy: "(").first?.trimmingCharacters(in: .whitespacesAndNewlines) ?? fl
        return clean.hasSuffix("mm") ? clean : "\(clean)mm"
    }

    private func exposureItem(icon: String, label: String, value: String) -> some View {
        VStack(spacing: 2) {
            HStack(spacing: 2) {
                Image(systemName: icon)
                    .font(.system(size: 8))
                    .foregroundStyle(.secondary)
                Text(label)
                    .font(.system(size: 8))
                    .foregroundStyle(.secondary)
            }
            Text(value)
                .font(.system(size: 10, weight: .semibold, design: .monospaced))
                .foregroundStyle(.primary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 5)
        .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 6, style: .continuous))
    }

    // 地理位置与即时反查预览卡片
    private var locationAndGeocodeCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(lang.text(.gpsAndGeocodeTitle), systemImage: "location.fill")
                    .font(.headline)
                Spacer()
                if let exif = store.selectedAssetExif {
                    if exif.hasGPS {
                        if exif.locationSummary != nil {
                            Text(lang.text(.geocodedWrittenBadge))
                                .font(.caption2.weight(.bold))
                                .foregroundStyle(.green)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.green.opacity(0.12), in: Capsule())
                        } else {
                            Text(lang.text(.hasGpsNotGeocodedBadge))
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.blue)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.blue.opacity(0.12), in: Capsule())
                        }
                    } else if !store.isExifLoading {
                        Text(lang.text(.noGpsBadge))
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(.orange)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.orange.opacity(0.12), in: Capsule())
                    }
                }
            }

            if let exif = store.selectedAssetExif {
                if exif.hasGPS, let lat = exif.latitude, let lon = exif.longitude {
                    VStack(alignment: .leading, spacing: 6) {
                        // 1. 坐标与海拔
                        HStack {
                            Text(lang.text(.coordinatesLabel))
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                            Text(String(format: "%.6f°, %.6f°", lat, lon))
                                .font(.caption2.monospaced())
                                .foregroundStyle(.primary)
                            if let alt = exif.altitude {
                                Text("·")
                                    .foregroundStyle(.secondary)
                                Text(String(format: "%.1fm", alt))
                                    .font(.caption2.monospaced())
                                    .foregroundStyle(.secondary)
                            }
                        }

                        // 2. 已有地名展示
                        if let loc = exif.locationSummary {
                            HStack(alignment: .top, spacing: 6) {
                                Text(lang.text(.writtenLabel))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                                Text(loc)
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(.teal)

                                Spacer()

                                if let src = exif.geocodeSource {
                                    Text(src == "xmp" ? lang.text(.geocodeSourceXMP) : lang.text(.geocodeSourceEmbedded))
                                        .font(.system(size: 9, weight: .bold))
                                        .foregroundStyle(src == "xmp" ? Color.indigo : Color.purple)
                                        .padding(.horizontal, 5)
                                        .padding(.vertical, 1)
                                        .background((src == "xmp" ? Color.indigo : Color.purple).opacity(0.12), in: Capsule())
                                }
                            }
                        } else {
                            // 3. 未写入地名：提供即时反查预览按钮
                            VStack(alignment: .leading, spacing: 4) {
                                if let preview = previewGeocodeResult {
                                    HStack(alignment: .top, spacing: 6) {
                                        Image(systemName: "sparkles")
                                            .font(.caption2)
                                            .foregroundStyle(.teal)
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text("\(lang.text(.previewPrefix)) \(preview.formattedSummary)")
                                                .font(.caption2.weight(.semibold))
                                                .foregroundStyle(.teal)
                                            Text(lang.text(.geocodeNotice))
                                                .font(.system(size: 9))
                                                .foregroundStyle(.tertiary)
                                        }
                                    }
                                    .padding(6)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .background(Color.teal.opacity(0.1), in: RoundedRectangle(cornerRadius: 6))
                                } else {
                                    Button {
                                        fetchPreviewGeocode(lat: lat, lon: lon, alt: exif.altitude ?? 0)
                                    } label: {
                                        HStack(spacing: 4) {
                                            if isPreviewLoading {
                                                ProgressView()
                                                    .scaleEffect(0.6)
                                                    .frame(width: 12, height: 12)
                                            } else {
                                                Image(systemName: "sparkle.magnifyingglass")
                                            }
                                            Text(lang.text(.clickToPreviewGeocode))
                                                .font(.caption2.weight(.medium))
                                        }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .tint(.teal)
                                    .controlSize(.small)
                                    .disabled(isPreviewLoading)
                                }
                            }
                            .padding(.top, 2)
                        }
                    }
                } else {
                    Text(lang.text(.noGpsNotice))
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func fetchPreviewGeocode(lat: Double, lon: Double, alt: Double) {
        isPreviewLoading = true
        if PhotoolsEngine.shared.isLoaded {
            let res = PhotoolsEngine.shared.lookupCoordinates(latitude: lat, longitude: lon, altitude: alt)
            self.previewGeocodeResult = res
            self.isPreviewLoading = false
        } else {
            Task {
                let client = PhotoolsProcessClient()
                let cmd = PhotoolsCommand.geodataTest(
                    executablePath: store.photoolsExecutablePath,
                    latitude: lat,
                    longitude: lon,
                    altitude: alt
                )
                let collector = PreviewTextCollector()
                do {
                    try await client.run(command: cmd) { text in
                        Task { await collector.append(text) }
                    }
                    let out = await collector.getText()
                    self.previewGeocodeResult = GeodataParser.parseLookupOutput(out)
                    self.isPreviewLoading = false
                } catch {
                    self.previewGeocodeResult = GeodataLookupResult(formattedSummary: "Error: \(error.localizedDescription)")
                    self.isPreviewLoading = false
                }
            }
        }
    }

    // 伴随文件卡片
    private var companionFilesCard: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label(lang.text(.companionListTitle), systemImage: "doc.on.doc.fill")
                .font(.headline)

            VStack(alignment: .leading, spacing: 4) {
                ForEach(asset.allFiles, id: \.self) { path in
                    HStack(spacing: 6) {
                        let ext = (path as NSString).pathExtension.uppercased()
                        let isPrimary = (path == asset.primaryPath)

                        Text(ext)
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                            .foregroundStyle(isPrimary ? Color.accentColor : Color.secondary)
                            .padding(.horizontal, 5)
                            .padding(.vertical, 2)
                            .background(isPrimary ? Color.accentColor.opacity(0.15) : Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 4))

                        Text((path as NSString).lastPathComponent)
                            .font(.caption2.monospaced())
                            .foregroundStyle(isPrimary ? .primary : .secondary)
                            .lineLimit(1)
                            .truncationMode(.middle)

                        Spacer()

                        if isPrimary {
                            Text(lang.text(.primaryDecisionSource))
                                .font(.system(size: 9, weight: .semibold))
                                .foregroundStyle(.green)
                        }

                        Button {
                            NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
                        } label: {
                            Image(systemName: "folder")
                                .font(.system(size: 11))
                        }
                        .buttonStyle(.plain)
                        .help(lang.text(.showInFinder))
                    }
                    .padding(.vertical, 1)
                    .contentShape(Rectangle())
                    .contextMenu {
                        Button {
                            NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
                        } label: {
                            Label(lang.text(.showInFinder), systemImage: "folder")
                        }
                        Button {
                            NSWorkspace.shared.open(URL(fileURLWithPath: path))
                        } label: {
                            Label(lang.text(.openThisFile), systemImage: "arrow.up.right.square")
                        }
                    }
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // 待处理报告卡片
    private func pendingReportCard(_ text: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Label(lang.text(.pendingReasonTitle), systemImage: "exclamationmark.triangle.fill")
                .font(.headline)
                .foregroundStyle(.orange)

            Text(text)
                .font(.system(size: 11, design: .monospaced))
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.orange.opacity(0.08), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.orange.opacity(0.25), lineWidth: 1)
        )
    }

    // 原始 Exif 标签卡片
    private var rawExifTagsCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Button {
                withAnimation(.easeInOut(duration: 0.2)) {
                    isRawTagsExpanded.toggle()
                }
            } label: {
                HStack {
                    Label("\(lang.text(.allExifTagsTitle)) (\(store.selectedAssetExif?.rawTags.count ?? 0))", systemImage: "list.bullet.rectangle")
                        .font(.headline)
                        .foregroundStyle(.primary)
                    Spacer()
                    Image(systemName: isRawTagsExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(.secondary)
                }
            }
            .buttonStyle(.plain)

            if isRawTagsExpanded {
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Image(systemName: "magnifyingglass")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        TextField(lang.text(.filterTagsPlaceholder), text: $tagSearchText)
                            .textFieldStyle(.plain)
                            .font(.caption2)
                    }
                    .padding(5)
                    .background(Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 6))

                    let filteredTags = (store.selectedAssetExif?.rawTags ?? []).filter { tag in
                        if tagSearchText.isEmpty { return true }
                        return tag.tag.localizedCaseInsensitiveContains(tagSearchText) ||
                               tag.value.localizedCaseInsensitiveContains(tagSearchText) ||
                               tag.group.localizedCaseInsensitiveContains(tagSearchText)
                    }

                    if filteredTags.isEmpty {
                        Text(lang.text(.noMatchingTags))
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                            .padding(.vertical, 2)
                    } else {
                        VStack(spacing: 2) {
                            ForEach(filteredTags) { item in
                                HStack(alignment: .top, spacing: 6) {
                                    Text(item.group)
                                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                        .padding(.horizontal, 4)
                                        .padding(.vertical, 1)
                                        .background(Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 3))

                                    Text(item.tag)
                                        .font(.caption2.weight(.semibold))
                                        .foregroundStyle(.primary)

                                    Spacer()

                                    Text(item.value)
                                        .font(.caption2.monospaced())
                                        .foregroundStyle(.secondary)
                                        .lineLimit(2)
                                        .multilineTextAlignment(.trailing)
                                        .textSelection(.enabled)
                                }
                                .padding(.vertical, 2)
                                Divider()
                            }
                        }
                    }
                }
                .padding(.top, 2)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor).opacity(0.8), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func typeBadge(for type: PrimaryAssetType) -> some View {
        Text(type.title)
            .font(.caption2.weight(.bold))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(typeColor(for: type).opacity(0.15), in: Capsule())
            .foregroundStyle(typeColor(for: type))
    }

    private func typeColor(for type: PrimaryAssetType) -> Color {
        switch type {
        case .rawPair:
            return .purple
        case .rawOnly:
            return .indigo
        case .jpgOnly:
            return .blue
        case .companionOnly:
            return .secondary
        }
    }

    // 元数据溯源与审计指纹卡片
    private func provenanceAuditCard(_ exif: ExifMetadata) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(lang.text(.provenanceTitle), systemImage: "signature")
                    .font(.headline)
                    .foregroundStyle(.indigo)
                Spacer()
                Text("photools Provenance")
                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                    .foregroundStyle(.indigo)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.indigo.opacity(0.12), in: Capsule())
            }

            VStack(alignment: .leading, spacing: 8) {
                // 坐标来源与算法
                HStack(alignment: .top, spacing: 14) {
                    if let src = exif.gpsSource, !src.isEmpty {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lang.text(.provenanceSourceLabel))
                                .font(.system(size: 9))
                                .foregroundStyle(.secondary)
                            Text(formatGpsSource(src))
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.primary)
                        }
                    }

                    if let method = exif.gpsMatchMethod, !method.isEmpty {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lang.text(.provenanceMethodLabel))
                                .font(.system(size: 9))
                                .foregroundStyle(.secondary)
                            Text(formatMatchMethod(method))
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                Divider()

                // 处理引擎与时间
                HStack(alignment: .top, spacing: 14) {
                    if let proc = exif.processor, !proc.isEmpty {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lang.text(.provenanceProcessorLabel))
                                .font(.system(size: 9))
                                .foregroundStyle(.secondary)
                            Text(proc)
                                .font(.caption2.monospaced())
                                .foregroundStyle(.primary)
                        }
                    }

                    if let dt = exif.processedDate, !dt.isEmpty {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lang.text(.provenanceDateLabel))
                                .font(.system(size: 9))
                                .foregroundStyle(.secondary)
                            Text(dt)
                                .font(.caption2.monospaced())
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                if let sidecar = exif.sidecarPath, !sidecar.isEmpty {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text")
                            .font(.system(size: 9))
                            .foregroundStyle(.indigo)
                        Text(lang.text(.sidecarFileLabel) + ":")
                            .font(.system(size: 9))
                            .foregroundStyle(.secondary)
                        Text((sidecar as NSString).lastPathComponent)
                            .font(.system(size: 9, weight: .semibold, design: .monospaced))
                            .foregroundStyle(.indigo)
                    }
                    .padding(.top, 2)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(Color.indigo.opacity(0.04))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(Color.indigo.opacity(0.2), lineWidth: 1)
        )
    }

    private func formatGpsSource(_ src: String) -> String {
        switch src.lowercased() {
        case "gpx":
            return lang.text(.gpsSourceGpx)
        case "interpolated":
            return lang.text(.gpsSourceInterpolated)
        case "camera", "original":
            return lang.text(.gpsSourceCamera)
        default:
            return src
        }
    }

    private func formatMatchMethod(_ method: String) -> String {
        switch method.lowercased() {
        case "time_proximity":
            return lang.text(.methodTimeProximity)
        case "spherical_linear_interpolation":
            return lang.text(.methodSphericalLinear)
        case "nearest_neighbor_anchor":
            return lang.text(.methodNearestNeighbor)
        default:
            return method
        }
    }
}
