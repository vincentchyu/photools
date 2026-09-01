import PhotoolsCore
import SwiftUI

public struct SettingsView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var selectedTab: Int = 0

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(spacing: 0) {
            // 1. 顶部标题与分段切换器
            headerView

            Divider()

            // 2. 内容滚动区
            ScrollView(.vertical, showsIndicators: true) {
                VStack(alignment: .leading, spacing: 16) {
                    if selectedTab == 0 {
                        sessionSettingsTab
                    } else {
                        globalSettingsTab
                    }
                }
                .padding(20)
            }

            Divider()

            // 3. 底部完成按钮栏
            footerView
        }
        .frame(width: 580, height: 530)
        .background(Color(nsColor: .windowBackgroundColor))
        .onDisappear {
            store.refresh()
        }
    }

    // MARK: - 顶部导航与分段切换
    private var headerView: some View {
        VStack(spacing: 12) {
            HStack {
                Text(lang.text(.preferencesTitle))
                    .font(.headline.weight(.semibold))

                Spacer()

                Button {
                    store.refresh()
                    dismiss()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .keyboardShortcut(.cancelAction)
                .keyboardShortcut("w", modifiers: .command)
                .help("\(lang.text(.close)) (Esc / ⌘W)")
            }

            Picker("", selection: $selectedTab) {
                Label(lang.text(.tabSession), systemImage: "bolt.badge.clock.fill").tag(0)
                Label(lang.text(.tabGlobal), systemImage: "gearshape.2.fill").tag(1)
            }
            .pickerStyle(.segmented)
            .labelsHidden()
        }
        .padding(.horizontal, 20)
        .padding(.top, 16)
        .padding(.bottom, 12)
    }

    // MARK: - Tab 1: 会话设置 (当前批次相册调度，临时生效)
    private var sessionSettingsTab: some View {
        VStack(alignment: .leading, spacing: 16) {
            // 工作区目录组
            settingsSection(title: lang.text(.workspaceConfig), icon: "folder.badge.gearshape") {
                pathInputField(
                    title: lang.text(.baseDirectory),
                    text: $store.baseDirectory,
                    icon: "folder.fill"
                )

                pathInputField(
                    title: lang.text(.sourceDirectory),
                    text: $store.sourceDirectory,
                    icon: "tray.and.arrow.down.fill"
                )

                pathInputField(
                    title: lang.text(.processedDirectory),
                    text: $store.processedDirectory,
                    icon: "archivebox.fill"
                )

                Divider()

                settingsToggleRow(
                    title: lang.text(.flatMode),
                    subtitle: lang.text(.flatModeDesc),
                    isOn: $store.flatMode
                )
            }

            // 会话执行调度组
            settingsSection(title: lang.text(.sessionSettings), icon: "slider.horizontal.3") {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(lang.text(.geosyncOffset))
                            .font(.subheadline)
                        Text(lang.text(.geosyncOffsetPrompt))
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    TextField("0, +15, -00:01:30", text: $store.geosync)
                        .textFieldStyle(.roundedBorder)
                        .frame(width: 160)
                        .font(.body.monospaced())
                }

                Divider()

                settingsToggleRow(
                    title: lang.text(.allowNoGps),
                    subtitle: lang.text(.allowNoGpsDesc),
                    isOn: $store.allowNoGPS
                )

                Divider()

                settingsToggleRow(
                    title: lang.text(.testBackupMode),
                    subtitle: lang.text(.testBackupDesc),
                    isOn: $store.testBackup
                )
            }
        }
    }

    // MARK: - Tab 2: 全局设置 (持久化至 plugins.json)
    private var globalSettingsTab: some View {
        VStack(alignment: .leading, spacing: 16) {
            // 语言设置
            settingsSection(title: lang.text(.languageSetting), icon: "globe") {
                HStack {
                    Text(lang.text(.languageSetting))
                        .font(.subheadline)

                    Spacer()

                    Picker("", selection: $lang.currentLanguage) {
                        ForEach(AppLanguage.allCases) { item in
                            Text(item.displayName).tag(item)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(width: 160)
                }
            }

            // 全局路径
            settingsSection(title: lang.text(.globalPreferences), icon: "internaldrive.fill") {
                pathInputField(
                    title: lang.text(.gpxDirectory),
                    text: $store.gpxDirectory,
                    icon: "map.fill"
                )

                pathInputField(
                    title: lang.text(.logDirectory),
                    text: $store.logDirectory,
                    icon: "doc.text.fill"
                )
            }

            // 元数据写入策略
            settingsSection(title: lang.text(.sidecarPolicy), icon: "shield.lefthalf.filled") {
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text(lang.text(.sidecarPolicy))
                            .font(.subheadline)

                        Spacer()

                        Picker("", selection: $store.sidecarPolicy) {
                            Text(lang.text(.policySmart)).tag("smart")
                            Text(lang.text(.policySidecarOnly)).tag("sidecar_only")
                            Text(lang.text(.policyEmbedAndSidecar)).tag("embed_and_sidecar")
                            Text(lang.text(.policyEmbedOnly)).tag("embed_only")
                        }
                        .pickerStyle(.menu)
                        .frame(width: 200)
                    }

                    Text(policyDescription(for: store.sidecarPolicy))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(8)
                        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
                }
            }

            // 性能与格式
            settingsSection(title: lang.text(.performanceSection), icon: "bolt.fill") {
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Text(lang.text(.rawExtensions))
                            .font(.subheadline)
                        Spacer()
                        TextField("nef,cr3,arw,dng...", text: $store.rawExtensions)
                            .textFieldStyle(.roundedBorder)
                            .frame(width: 220)
                            .font(.subheadline.monospaced())
                    }

                    Divider()

                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(lang.text(.companionExtensions))
                                .font(.subheadline)
                            Spacer()
                            TextField("wav,acr,exf...", text: $store.companionExtensions)
                                .textFieldStyle(.roundedBorder)
                                .frame(width: 220)
                                .font(.subheadline.monospaced())
                        }
                        Text(lang.text(.companionExtensionsDesc))
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }

                    Divider()

                    HStack {
                        Text(lang.text(.concurrencyWorkers))
                            .font(.subheadline)
                        Spacer()
                        Stepper("\(store.workers)", value: $store.workers, in: 1...32)
                            .font(.subheadline.monospaced())
                    }
                }
            }
        }
    }

    // MARK: - 基础组件封装 (统一卡片风格)
    private func settingsSection<Content: View>(
        title: String,
        icon: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color.accentColor)
                Text(title)
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.secondary)
            }
            .padding(.leading, 2)

            VStack(alignment: .leading, spacing: 10) {
                content()
            }
            .padding(12)
            .background(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(Color(nsColor: .controlBackgroundColor))
                    .overlay(
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
                    )
            )
        }
    }

    private func pathInputField(
        title: String,
        text: Binding<String>,
        icon: String
    ) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.subheadline)
                .foregroundStyle(.primary)

            HStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(width: 14)

                TextField("", text: text)
                    .textFieldStyle(.roundedBorder)
                    .font(.caption.monospaced())

                Button(lang.text(.chooseDirectory)) {
                    chooseDirectory { path in
                        text.wrappedValue = path
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        }
    }

    private func settingsToggleRow(
        title: String,
        subtitle: String? = nil,
        isOn: Binding<Bool>
    ) -> some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.subheadline)
                if let sub = subtitle {
                    Text(sub)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            Toggle("", isOn: isOn)
                .toggleStyle(.switch)
                .labelsHidden()
        }
    }

    // MARK: - 底部操作栏
    private var footerView: some View {
        HStack {
            Spacer()
            Button(lang.text(.done)) {
                store.refresh()
                dismiss()
            }
            .buttonStyle(.borderedProminent)
            .keyboardShortcut(.defaultAction)
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
    }

    private func policyDescription(for policy: String) -> String {
        switch policy {
        case "smart", "read_only":
            return lang.text(.policySmartDesc)
        case "sidecar_only":
            return lang.text(.policySidecarOnlyDesc)
        case "embed_and_sidecar":
            return lang.text(.policyEmbedAndSidecarDesc)
        case "embed_only":
            return lang.text(.policyEmbedOnlyDesc)
        default:
            return lang.text(.policySmartDesc)
        }
    }

    private func chooseDirectory(onSelect: (String) -> Void) {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.prompt = lang.text(.done)
        if panel.runModal() == .OK, let url = panel.url {
            onSelect(url.path)
        }
    }
}
