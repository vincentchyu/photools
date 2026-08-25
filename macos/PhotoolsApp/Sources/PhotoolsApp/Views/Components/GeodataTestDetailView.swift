import PhotoolsCore
import SwiftUI

/// 离线地理反查引擎测试与结果展示面板
public struct GeodataTestDetailView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared

    private struct PresetLocation: Identifiable {
        let id = UUID()
        let nameZH: String
        let nameEN: String
        let lat: Double
        let lon: Double
        let alt: Double
        let icon: String
    }

    private let presets: [PresetLocation] = [
        PresetLocation(nameZH: "上海·陆家嘴", nameEN: "Shanghai Lujiazui", lat: 31.2400, lon: 121.5000, alt: 10, icon: "building.2"),
        PresetLocation(nameZH: "北京·故宫", nameEN: "Beijing Forbidden City", lat: 39.9163, lon: 116.3972, alt: 45, icon: "building.columns"),
        PresetLocation(nameZH: "杭州·西湖", nameEN: "Hangzhou West Lake", lat: 30.2460, lon: 120.1430, alt: 12, icon: "water.waves"),
        PresetLocation(nameZH: "西藏·布达拉宫", nameEN: "Tibet Potala Palace", lat: 29.6555, lon: 91.1186, alt: 3700, icon: "mountain.2"),
        PresetLocation(nameZH: "四川·九寨沟", nameEN: "Jiuzhaigou", lat: 33.2600, lon: 103.9200, alt: 2400, icon: "tree"),
        PresetLocation(nameZH: "日本·东京塔", nameEN: "Tokyo Tower", lat: 35.6586, lon: 139.7454, alt: 25, icon: "antenna.radiowaves.left.and.right"),
        PresetLocation(nameZH: "法国·巴黎铁塔", nameEN: "Eiffel Tower", lat: 48.8584, lon: 2.2945, alt: 35, icon: "sparkles"),
        PresetLocation(nameZH: "瑞士·少女峰", nameEN: "Jungfrau", lat: 46.5475, lon: 7.9825, alt: 4158, icon: "snowflake")
    ]

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // 1. 顶部引擎紧凑说明
            headerSection

            // 2. 坐标反查测试输入 View
            testInputView

            // 3. 执行过程黑客终端 View
            terminalSection

            // 4. 匹配结果展示 View
            testResultView
        }
        .padding(12)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    // 1. 顶部引擎紧凑说明
    private var headerSection: some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 6) {
                Image(systemName: "globe.asia.australia.fill")
                    .foregroundStyle(.teal)
                    .font(.subheadline.weight(.bold))
                Text(lang.text(.geoEngineTitle))
                    .font(.subheadline.weight(.bold))
                Spacer()
            }

            Text(lang.text(.geoEngineDesc))
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
                .lineLimit(2)
        }
    }

    // 2. 反地理编码测试输入 View
    private var testInputView: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                HStack(spacing: 3) {
                    Text(lang.text(.latShort))
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                    TextField("31.2304", text: $store.testLatitude)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(size: 11, design: .monospaced))
                }

                HStack(spacing: 3) {
                    Text(lang.text(.lonShort))
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                    TextField("121.4737", text: $store.testLongitude)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(size: 11, design: .monospaced))
                }

                HStack(spacing: 3) {
                    Text(lang.text(.altShort))
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                    TextField("10", text: $store.testAltitude)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(size: 11, design: .monospaced))
                        .frame(width: 45)
                }

                // 测试按钮
                Button {
                    store.testGeodataLookup()
                } label: {
                    HStack(spacing: 3) {
                        if store.isGeodataTesting {
                            ProgressView()
                                .scaleEffect(0.5)
                                .frame(width: 10, height: 10)
                        } else {
                            Image(systemName: "bolt.fill")
                                .font(.system(size: 9))
                        }
                        Text(lang.text(.testBtn))
                            .font(.system(size: 11, weight: .semibold))
                    }
                    .padding(.horizontal, 4)
                }
                .buttonStyle(.borderedProminent)
                .tint(.teal)
                .controlSize(.small)
                .disabled(store.isGeodataTesting)

                // Debug 空间分析模式开关
                Toggle(isOn: $store.isGeodataDebugMode) {
                    Text(lang.text(.spatialAnalysisToggle))
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                }
                .toggleStyle(.checkbox)
                .help(lang.text(.spatialAnalysisHelp))
            }

            // 预设地标
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 4) {
                    Text(lang.text(.presetsLabel))
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)

                    ForEach(presets) { p in
                        Button {
                            store.testLatitude = String(format: "%.4f", p.lat)
                            store.testLongitude = String(format: "%.4f", p.lon)
                            store.testAltitude = String(format: "%.0f", p.alt)
                            store.testGeodataLookup()
                        } label: {
                            HStack(spacing: 2) {
                                Image(systemName: p.icon)
                                    .font(.system(size: 8))
                                Text(lang.currentLanguage.isChinese ? p.nameZH : p.nameEN)
                                    .font(.system(size: 9.5))
                            }
                            .padding(.horizontal, 5)
                            .padding(.vertical, 2)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.mini)
                    }
                }
            }
        }
        .padding(8)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
        .overlay(
            RoundedRectangle(cornerRadius: 6)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // 3. 执行过程黑客终端 View
    private var terminalSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                HStack(spacing: 4) {
                    Image(systemName: "terminal.fill")
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                    Text(lang.text(.logOutput))
                        .font(.system(size: 11, weight: .semibold))
                }

                Spacer()

                Button {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(store.geodataLog, forType: .string)
                } label: {
                    Image(systemName: "doc.on.doc")
                        .font(.system(size: 10))
                }
                .buttonStyle(.borderless)
                .help(lang.text(.copyAllLogs))

                Button {
                    store.clearGeodataLog()
                } label: {
                    Image(systemName: "trash")
                        .font(.system(size: 10))
                }
                .buttonStyle(.borderless)
                .help(lang.text(.clearLogs))
            }

            HackerTerminalView(
                logText: store.geodataLog,
                isRunning: store.isGeodataTesting,
                isAutoScroll: true,
                placeholder: "Ready to test...\n"
            )
            .frame(minHeight: 110, maxHeight: .infinity)
        }
        .frame(maxHeight: .infinity)
    }

    // 4. 匹配结果展示 View
    private var testResultView: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Image(systemName: "checkmark.seal.fill")
                    .font(.system(size: 10))
                    .foregroundStyle(.teal)
                Text(lang.text(.testResultTitle))
                    .font(.system(size: 11, weight: .bold))

                Spacer()

                if let res = store.testLookupResult, !res.formattedSummary.isEmpty {
                    Text(res.source.isEmpty ? (lang.currentLanguage.isChinese ? "离线高精库" : "Offline 3D KD-Tree") : res.source)
                        .font(.system(size: 9))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 4)
                        .padding(.vertical, 1)
                        .background(Color.secondary.opacity(0.12), in: Capsule())
                }
            }

            if let res = store.testLookupResult, !res.formattedSummary.isEmpty {
                HStack(spacing: 6) {
                    Text(lang.text(.formattedIptcTag) + ":")
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                    Text(res.formattedSummary)
                        .font(.system(size: 11.5, weight: .bold))
                        .foregroundStyle(.teal)
                        .textSelection(.enabled)
                        .lineLimit(1)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.teal.opacity(0.08), in: RoundedRectangle(cornerRadius: 4))

                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 4) {
                    compactMetaItem(icon: "flag.fill", label: lang.text(.countryRegion), value: "\(res.country) (\(res.countryCode))", color: .indigo)
                    if !res.province.isEmpty {
                        compactMetaItem(icon: "building.columns.fill", label: lang.text(.stateProvince), value: res.province, color: .blue)
                    }
                    if !res.city.isEmpty {
                        compactMetaItem(icon: "building.2.fill", label: lang.text(.cityPrefecture), value: res.city, color: .cyan)
                    }
                    if !res.district.isEmpty {
                        compactMetaItem(icon: "mappin.and.ellipse", label: lang.text(.districtCounty), value: res.district, color: .teal)
                    }
                    if !res.timezone.isEmpty {
                        compactMetaItem(icon: "clock.fill", label: lang.text(.timezoneLabel), value: res.timezone, color: .orange)
                    }
                    compactMetaItem(icon: "mountain.2.fill", label: lang.text(.altShort), value: "\(res.elevation)m", color: .purple)
                    if res.distanceKm > 0 {
                        compactMetaItem(icon: "ruler.fill", label: lang.text(.nearestPointDist), value: "\(String(format: "%.2f", res.distanceKm))km", color: .mint)
                    }
                }
            } else {
                HStack(spacing: 6) {
                    Image(systemName: "info.circle")
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                    Text(lang.text(.instantLookup))
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                }
                .padding(6)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(8)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
        .overlay(
            RoundedRectangle(cornerRadius: 6)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func compactMetaItem(icon: String, label: String, value: String, color: Color) -> some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 9))
                .foregroundStyle(color)
                .frame(width: 12)

            Text(label + ":")
                .font(.system(size: 10))
                .foregroundStyle(.secondary)

            Text(value)
                .font(.system(size: 10, weight: .medium, design: .monospaced))
                .foregroundStyle(.primary)
                .lineLimit(1)
                .textSelection(.enabled)

            Spacer()
        }
        .padding(.horizontal, 4)
        .padding(.vertical, 2)
    }
}
