import PhotoolsCore
import SwiftUI

public struct CopiedGPSInspectorSheet: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @Environment(\.dismiss) private var dismiss
    @State private var geocodeResult: GeodataLookupResult?
    @State private var selectedTab: DetailTab = .allTags

    enum DetailTab: String, CaseIterable, Identifiable {
        case allTags = "全部待写入标签"
        case overview = "核心坐标概览"
        case json = "JSON 导出"

        var id: String { rawValue }
    }

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(spacing: 0) {
            // 头部标题栏
            HStack {
                HStack(spacing: 8) {
                    Image(systemName: "location.circle.fill")
                        .font(.title2)
                        .foregroundStyle(.green)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(lang.text(.copiedGpsInspectorTitle))
                            .font(.headline)
                            .foregroundStyle(.primary)
                        if let copied = store.copiedGPSMetadata {
                            Text("来源: \(copied.sourceAssetBaseName) · 包含 \(copied.rawGPSTags.count) 项待写入物理 GPS 字段")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                Spacer()

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .keyboardShortcut(.escape, modifiers: [])
            }
            .padding(16)

            Divider()

            if let copied = store.copiedGPSMetadata {
                // 分页选择器
                Picker("", selection: $selectedTab) {
                    ForEach(DetailTab.allCases) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 16)
                .padding(.top, 12)
                .padding(.bottom, 6)

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        switch selectedTab {
                        case .allTags:
                            // 🌟 1. 摄影师专属：全部待写入 GPS 标签明细表 (完全透明，非黑盒)
                            fullTagsTable(copied)
                        case .overview:
                            // 2. 核心坐标卡片与逆地理
                            coordinatesCard(copied)
                            reverseGeocodeCard(copied)
                            sourceInfoCard(copied)
                        case .json:
                            // 3. 完整 JSON 视图
                            jsonExportCard(copied)
                        }
                    }
                    .padding(16)
                }

                Divider()

                // 写入策略模式选择条 (让摄影师掌控写入目标)
                HStack(spacing: 8) {
                    Label("写入策略:", systemImage: "shield.lefthalf.filled")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)

                    Picker("", selection: $store.sidecarPolicy) {
                        Text("智能分层模式 (RAW+JPG+XMP指纹) 🌟").tag("smart")
                        Text("纯侧车模式 (原图只读，仅写XMP)").tag("sidecar_only")
                        Text("双写同步模式 (内嵌+XMP)").tag("embed_and_sidecar")
                        Text("纯原图内嵌模式 (仅改RAW/JPG)").tag("embed_only")
                    }
                    .pickerStyle(.menu)
                    .frame(width: 290)

                    Spacer()

                    if let sel = store.selectedAsset {
                        Text("目标: \(sel.baseName)")
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(.primary)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.secondary.opacity(0.1), in: RoundedRectangle(cornerRadius: 4))
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .background(Color.secondary.opacity(0.05))

                Divider()

                // 底部操作按钮栏
                HStack(spacing: 12) {
                    Button(role: .destructive) {
                        store.clearCopiedGPS()
                    } label: {
                        Label(lang.text(.copiedGpsClear), systemImage: "trash")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)

                    Button {
                        openInAppleMaps(lat: copied.latitude, lon: copied.longitude)
                    } label: {
                        Label(lang.text(.copiedGpsOpenInMaps), systemImage: "map")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)

                    Button {
                        copyAllAsJSON(copied)
                    } label: {
                        Label("复制全量 JSON", systemImage: "curlybraces")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)

                    Spacer()

                    Button {
                        store.pasteGPSToSelectedAsset()
                    } label: {
                        let policyTag: String = {
                            switch store.sidecarPolicy {
                            case "smart": return "智能模式"
                            case "sidecar_only": return "纯侧车"
                            case "embed_only": return "纯原图"
                            default: return "双写"
                            }
                        }()
                        Label("按[\(policyTag)]写入选中照片 (⌥⌘G)", systemImage: "location.fill.viewfinder")
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.regular)
                    .disabled(store.selectedAsset == nil)
                }
                .padding(14)
                .background(.regularMaterial)
            } else {
                emptyView
            }
        }
        .frame(minWidth: 580, minHeight: 480)
        .onAppear {
            if let copied = store.copiedGPSMetadata {
                resolveGeocode(copied)
            }
        }
        .onChange(of: store.copiedGPSMetadata) { newCopied in
            if let newCopied {
                resolveGeocode(newCopied)
            } else {
                geocodeResult = nil
            }
        }
    }

    private var emptyView: some View {
        VStack(spacing: 14) {
            Spacer()
            Image(systemName: "location.slash.fill")
                .font(.system(size: 48))
                .foregroundStyle(.secondary.opacity(0.6))

            Text(lang.text(.copiedGpsEmptyTitle))
                .font(.headline)
                .foregroundStyle(.primary)

            Text(lang.text(.copiedGpsEmptySubtitle))
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // 🌟 全量待写入 GPS 标签明细表 (非黑盒，全透明)
    private func fullTagsTable(_ copied: CopiedGPSMetadata) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label("已采集待写入数据包 (\(copied.rawGPSTags.count) 项物理字段)", systemImage: "tablecells.badge.ellipsis")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)

                Spacer()

                Text("RAW / JPG / XMP 同步注入")
                    .font(.caption2.weight(.medium))
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.blue.opacity(0.12), in: Capsule())
                    .foregroundStyle(.blue)
            }

            VStack(spacing: 0) {
                // 表头
                HStack {
                    Text("EXIF 标签名称 (Tag)")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                        .frame(width: 170, alignment: .leading)
                    Text("采集数值 (Value)")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text("物理含义")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                        .frame(width: 110, alignment: .trailing)
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color.secondary.opacity(0.12))

                Divider()

                // 表体行
                ForEach(copied.sortedGPSTags, id: \.key) { item in
                    HStack(alignment: .top) {
                        Text(item.key)
                            .font(.system(.caption, design: .monospaced).weight(.semibold))
                            .foregroundStyle(.primary)
                            .frame(width: 170, alignment: .leading)
                            .textSelection(.enabled)

                        Text(item.value)
                            .font(.system(.caption, design: .monospaced))
                            .foregroundStyle(.indigo)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .textSelection(.enabled)

                        Text(tagMeaning(item.key))
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .frame(width: 110, alignment: .trailing)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.secondary.opacity(0.03))

                    Divider()
                }

                // 溯源指纹行 (Photools Provenance)
                HStack(alignment: .top) {
                    Text("XMP:GPSSource")
                        .font(.system(.caption, design: .monospaced).weight(.semibold))
                        .foregroundStyle(.primary)
                        .frame(width: 170, alignment: .leading)
                    Text(copied.gpsSource ?? "manual_copied")
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(.green)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text("溯源标记")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .frame(width: 110, alignment: .trailing)
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(Color.green.opacity(0.04))

                Divider()

                HStack(alignment: .top) {
                    Text("XMP:GPSMatchMethod")
                        .font(.system(.caption, design: .monospaced).weight(.semibold))
                        .foregroundStyle(.primary)
                        .frame(width: 170, alignment: .leading)
                    Text(copied.gpsMatchMethod ?? "clipboard_paste")
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(.green)
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text("匹配/赋值方式")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .frame(width: 110, alignment: .trailing)
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(Color.green.opacity(0.04))
            }
            .background(Color.secondary.opacity(0.04), in: RoundedRectangle(cornerRadius: 8))
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(Color.secondary.opacity(0.15), lineWidth: 1)
            )
        }
    }

    private func tagMeaning(_ tag: String) -> String {
        switch tag {
        case "GPSVersionID": return "GPS 协议规范"
        case "GPSLatitudeRef": return "纬度半球 (N/S)"
        case "GPSLatitude": return "纬度数值"
        case "GPSLongitudeRef": return "经度半球 (E/W)"
        case "GPSLongitude": return "经度数值"
        case "GPSPosition": return "复合经纬度"
        case "GPSAltitudeRef": return "海平面参考系"
        case "GPSAltitude": return "海拔高度"
        case "GPSTimeStamp": return "卫星 UTC 时间"
        case "GPSDateStamp": return "卫星 UTC 日期"
        case "GPSDateTime": return "卫星复合时间戳"
        case "GPSSatellites": return "接收卫星编号"
        case "GPSMapDatum": return "大地测量基准"
        case "GPSImgDirectionRef": return "成像方位基准"
        case "GPSImgDirection": return "成像方向角度"
        case "GPSHPositioningError": return "水平定位误差"
        default: return "GPS 物理元数据"
        }
    }

    private func coordinatesCard(_ copied: CopiedGPSMetadata) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Label(lang.text(.copiedGpsCoordinates), systemImage: "scope")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.primary)

            VStack(spacing: 8) {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("十进制格式 (Decimal)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        Text(copied.formattedDecimal)
                            .font(.system(.body, design: .monospaced).weight(.semibold))
                            .textSelection(.enabled)
                    }
                    Spacer()
                    copyButton(text: copied.formattedDecimal)
                }

                Divider()

                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("度分秒格式 (DMS)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        Text(copied.formattedDMS)
                            .font(.system(.body, design: .monospaced).weight(.semibold))
                            .textSelection(.enabled)
                    }
                    Spacer()
                    copyButton(text: copied.formattedDMS)
                }

                if let alt = copied.formattedAltitude {
                    Divider()
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(lang.text(.copiedGpsAltitude))
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                            Text(alt)
                                .font(.system(.body, design: .monospaced).weight(.semibold))
                                .textSelection(.enabled)
                        }
                        Spacer()
                        copyButton(text: alt)
                    }
                }
            }
            .padding(12)
            .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private func reverseGeocodeCard(_ copied: CopiedGPSMetadata) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Label(lang.text(.copiedGpsReverseGeocode), systemImage: "globe.asia.australia.fill")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.primary)

            VStack(alignment: .leading, spacing: 6) {
                if let geo = geocodeResult {
                    HStack(alignment: .top, spacing: 6) {
                        Image(systemName: "mappin.circle.fill")
                            .foregroundStyle(.teal)
                            .font(.subheadline)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(geo.formattedSummary)
                                .font(.body.weight(.semibold))
                                .foregroundStyle(.primary)
                            if !geo.district.isEmpty && geo.district != geo.city {
                                Text("\(geo.country) · \(geo.province) · \(geo.city) · \(geo.district)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                } else if let loc = copied.locationSummary, !loc.isEmpty {
                    HStack(alignment: .top, spacing: 6) {
                        Image(systemName: "mappin.circle.fill")
                            .foregroundStyle(.teal)
                        Text(loc)
                            .font(.body.weight(.semibold))
                            .foregroundStyle(.primary)
                    }
                } else {
                    HStack(spacing: 6) {
                        ProgressView()
                            .scaleEffect(0.6)
                            .frame(width: 14, height: 14)
                        Text("正在离线反查地理地名...")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.teal.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private func sourceInfoCard(_ copied: CopiedGPSMetadata) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(lang.text(.copiedGpsSourceInfo), systemImage: "camera.fill")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.primary)

            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Text("来源照片:")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    Text(copied.sourceAssetBaseName)
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.primary)
                }

                if let date = copied.captureDate, !date.isEmpty {
                    HStack(spacing: 6) {
                        Image(systemName: "calendar")
                            .font(.caption2)
                            .foregroundStyle(Color.blue)
                        Text(lang.text(.captureTime) + ":")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        Text(date)
                            .font(.caption2.monospaced().weight(.semibold))
                            .foregroundStyle(Color.blue)
                    }
                }

                if let src = copied.gpsSource, !src.isEmpty {
                    HStack {
                        Text("GPS 来源:")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        Text(src)
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(.indigo)
                    }
                }
            }
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color.secondary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private func jsonExportCard(_ copied: CopiedGPSMetadata) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label("JSON 数据预览与导出", systemImage: "curlybraces")
                    .font(.subheadline.weight(.semibold))
                Spacer()
                Button {
                    copyAllAsJSON(copied)
                } label: {
                    Label("复制 JSON", systemImage: "doc.on.doc")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }

            let jsonString: String = {
                var dict = copied.rawGPSTags
                dict["_latitude"] = "\(copied.latitude)"
                dict["_longitude"] = "\(copied.longitude)"
                if let alt = copied.altitude { dict["_altitude"] = "\(alt)" }
                dict["_sourceAsset"] = copied.sourceAssetBaseName
                dict["_sourceFilePath"] = copied.sourceFilePath
                if let data = try? JSONSerialization.data(withJSONObject: dict, options: [.prettyPrinted, .sortedKeys]),
                   let str = String(data: data, encoding: .utf8) {
                    return str
                }
                return "{}"
            }()

            ScrollView {
                Text(jsonString)
                    .font(.system(.caption, design: .monospaced))
                    .padding(10)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .textSelection(.enabled)
            }
            .frame(maxHeight: 280)
            .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 6))
        }
    }

    private func copyButton(text: String) -> some View {
        Button {
            NSPasteboard.general.clearContents()
            NSPasteboard.general.setString(text, forType: .string)
            store.showHUD("已复制: \(text)")
        } label: {
            Image(systemName: "doc.on.doc")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .buttonStyle(.plain)
        .help("复制文本")
    }

    private func copyAllAsJSON(_ copied: CopiedGPSMetadata) {
        var dict = copied.rawGPSTags
        dict["_latitude"] = "\(copied.latitude)"
        dict["_longitude"] = "\(copied.longitude)"
        if let alt = copied.altitude { dict["_altitude"] = "\(alt)" }
        dict["_sourceAsset"] = copied.sourceAssetBaseName
        dict["_sourceFilePath"] = copied.sourceFilePath

        if let data = try? JSONSerialization.data(withJSONObject: dict, options: [.prettyPrinted, .sortedKeys]),
           let str = String(data: data, encoding: .utf8) {
            NSPasteboard.general.clearContents()
            NSPasteboard.general.setString(str, forType: .string)
            store.showHUD("已复制全量 GPS JSON 数据 (\(dict.count) 项)")
        }
    }

    private func resolveGeocode(_ copied: CopiedGPSMetadata) {
        if PhotoolsEngine.shared.isLoaded {
            self.geocodeResult = PhotoolsEngine.shared.lookupCoordinates(
                latitude: copied.latitude,
                longitude: copied.longitude,
                altitude: copied.altitude ?? 0.0
            )
        }
    }

    private func openInAppleMaps(lat: Double, lon: Double) {
        if let url = URL(string: "http://maps.apple.com/?ll=\(lat),\(lon)&q=\(lat),\(lon)") {
            NSWorkspace.shared.open(url)
        }
    }
}
