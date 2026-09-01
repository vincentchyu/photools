import PhotoolsCore
import SwiftUI

public struct PipelineDashboardView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                // 1. 顶部工作区状态
                headerView

                // 2. 统计指标卡片组
                statusMetricsView

                // 3. 四大核心自动化处理能力
                capabilitiesSection
            }
            .padding(18)
        }
    }

    // 顶部工作区状态栏
    private var headerView: some View {
        HStack(alignment: .center) {
            VStack(alignment: .leading, spacing: 2) {
                // Text(lang.text(.workbenchTitle))
                //    .font(.title2.weight(.bold))
                Text(lang.text(.workbenchSubtitle))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            // 核心执行主按钮组
            HStack(spacing: 8) {
                if store.runState.isRunning {
                    Button(role: .destructive) {
                        store.cancelCurrentTask()
                    } label: {
                        Label(lang.text(.interruptPipeline), systemImage: "stop.fill")
                            .font(.subheadline.weight(.semibold))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.red)
                    .keyboardShortcut(".", modifiers: .command)
                    .help(lang.text(.interruptPipelineHelp))
                    .pulseOnHover(scale: 1.05, glowColor: .red)
                } else {
                    Button {
                        store.runPipeline()
                    } label: {
                        HStack(spacing: 6) {
                            Image(systemName: "play.fill")
                            Text(lang.text(.runPipeline))
                            Text("⌘↩")
                                .font(.caption2.weight(.semibold))
                                .padding(.horizontal, 4)
                                .padding(.vertical, 1)
                                .background(Color.white.opacity(0.2), in: RoundedRectangle(cornerRadius: 3))
                        }
                        .font(.subheadline.weight(.semibold))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.accentColor)
                    .keyboardShortcut(.return, modifiers: .command)
                    .help(lang.text(.runPipelineHelp))
                    .pulseOnHover(scale: 1.05, glowColor: .accentColor)
                }
            }
        }
    }

    // 核心统计指标 (自适应网格，窄屏自动换行，绝不横向撑大)
    private var statusMetricsView: some View {
        LazyVGrid(columns: [GridItem(.adaptive(minimum: 95), spacing: 8)], spacing: 8) {
            metricCard(title: lang.text(.metricGpx), value: "\(store.summary?.gpxFiles.count ?? 0)", icon: "map.fill", color: .blue)
            metricCard(title: lang.text(.metricInbox), value: "\(store.summary?.readyCount ?? 0)", icon: "tray.full.fill", color: .green)
            metricCard(title: lang.text(.metricRawPair), value: "\(store.summary?.rawPairCount ?? 0)", icon: "photo.stack.fill", color: .purple)
            metricCard(title: lang.text(.metricSingleFile), value: "\((store.summary?.rawOnlyCount ?? 0) + (store.summary?.jpgOnlyCount ?? 0))", icon: "photo.fill", color: .orange)
            metricCard(title: lang.text(.metricArchived), value: "\(store.summary?.processedFileCount ?? 0)", icon: "archivebox.fill", color: .indigo)
        }
    }

    private func metricCard(title: String, value: String, subtitle: String? = nil, icon: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 10))
                    .foregroundStyle(color)
                Text(title)
                    .font(.system(size: 10))
                    .foregroundStyle(.secondary)
            }
            Text(value)
                .font(.system(size: 17, weight: .bold, design: .rounded))
                .foregroundStyle(Color.primary)
            if let sub = subtitle {
                Text(sub)
                    .font(.system(size: 9))
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .background(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(Color(nsColor: .controlBackgroundColor).opacity(0.8))
                .overlay(
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
                )
        )
    }

    // 四大自动化能力卡片
    private var capabilitiesSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(lang.text(.pipelineTitle))
                .font(.headline)

            // 1. GPX 轨迹匹配
            capabilityCard(
                isEnabled: $store.enableGPXMatch,
                stepNumber: "1",
                title: store.pluginName(id: "gpx_matching", fallback: lang.text(.capGpxTitle)),
                desc: store.pluginDesc(id: "gpx_matching", fallback: lang.text(.capGpxDesc)),
                icon: "location.fill",
                accentColor: .blue
            ) {
                HStack(spacing: 10) {
                    Text(lang.text(.geosyncOffsetPrompt))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("0, +00:00:05", text: $store.geosync)
                        .textFieldStyle(.roundedBorder)
                        .frame(width: 140)
                        .font(.caption.monospaced())
                }
            }

            // 2. GPS 智能邻近推断与时间插值
            capabilityCard(
                isEnabled: $store.enableInterpolate,
                stepNumber: "2",
                title: store.pluginName(id: "gps_interpolate", fallback: lang.text(.capInterpolateTitle)),
                desc: store.pluginDesc(id: "gps_interpolate", fallback: lang.text(.capInterpolateDesc)),
                icon: "phone.badge.waveform.fill",
                accentColor: .orange
            ) {
                HStack(spacing: 10) {
                    Text(lang.text(.interpolateWindowPrompt))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Picker("", selection: $store.interpolateWindow) {
                        Text(lang.text(.window15m)).tag("15m")
                        Text(lang.text(.window30m)).tag("30m")
                        Text(lang.text(.window1h)).tag("1h")
                        Text(lang.text(.window2h)).tag("2h")
                        Text(lang.text(.window4h)).tag("4h")
                    }
                    .pickerStyle(.menu)
                    .frame(width: 150)
                }
            }

            // 3. 离线高精逆地理编码与地名写入
            capabilityCard(
                isEnabled: $store.enableGeocode,
                stepNumber: "3",
                title: store.pluginName(id: "reverse_geocode", fallback: lang.text(.capGeocodeTitle)),
                desc: store.pluginDesc(id: "reverse_geocode", fallback: lang.text(.capGeocodeDesc)),
                icon: "globe.asia.australia.fill",
                accentColor: .teal
            ) {
                Toggle(lang.text(.allowNoGpsDesc), isOn: $store.allowNoGPS)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            // 4. 拍摄日期重命名与安全归档
            capabilityCard(
                isEnabled: $store.enableArchive,
                stepNumber: "4",
                title: store.pluginName(id: "date_archive", fallback: lang.text(.capArchiveTitle)),
                desc: store.pluginDesc(id: "date_archive", fallback: lang.text(.capArchiveDesc)),
                icon: "archivebox.fill",
                accentColor: .indigo
            )
        }
    }

    private func capabilityCard<Content: View>(
        isEnabled: Binding<Bool>,
        stepNumber: String,
        title: String,
        desc: String,
        icon: String,
        accentColor: Color,
        @ViewBuilder extraContent: () -> Content = { EmptyView() }
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .top, spacing: 10) {
                Toggle("", isOn: isEnabled)
                    .toggleStyle(.switch)
                    .labelsHidden()

                Image(systemName: icon)
                    .foregroundStyle(isEnabled.wrappedValue ? accentColor : .secondary)
                    .font(.system(size: 14))
                    .frame(width: 20, height: 20)
                    .background((isEnabled.wrappedValue ? accentColor : .secondary).opacity(0.12), in: Circle())

                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(isEnabled.wrappedValue ? .primary : .secondary)
                    Text(desc)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Spacer()
            }

            if isEnabled.wrappedValue {
                let extra = extraContent()
                if !(extra is EmptyView) {
                    Divider()
                        .padding(.vertical, 2)
                    extra
                        .padding(.leading, 42)
                }
            }
        }
        .padding(12)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(Color(nsColor: .controlBackgroundColor).opacity(0.8))
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(isEnabled.wrappedValue ? accentColor.opacity(0.3) : Color.secondary.opacity(0.1), lineWidth: 1)
                )
        )
    }
}
