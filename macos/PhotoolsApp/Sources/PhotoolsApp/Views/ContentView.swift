import PhotoolsCore
import SwiftUI

public struct ContentView: View {
    @ObservedObject var store: WorkspaceStore
    @State private var showingSettings = false

    public init(store: WorkspaceStore? = nil) {
        self.store = store ?? WorkspaceStore()
    }

    public var body: some View {
        VStack(spacing: 0) {
            NavigationSplitView {
                SidebarView(store: store)
                    .navigationSplitViewColumnWidth(min: 190, ideal: 210, max: 240)
            } content: {
                AssetListView(store: store)
                    .navigationSplitViewColumnWidth(min: 320, ideal: 360, max: 440)
            } detail: {
                DetailView(store: store)
                    .navigationSplitViewColumnWidth(min: 380, ideal: 440)
            }

            Divider()

            // 全局常驻底部状态栏
            statusBarView
        }
        .frame(minWidth: 920, minHeight: 600)
        .toolbar {
            ToolbarItemGroup(placement: .automatic) {
                Button {
                    chooseBaseDirectory()
                } label: {
                    Label(LanguageManager.shared.text(.chooseDirectory), systemImage: "folder")
                }
                .help(LanguageManager.shared.text(.chooseDirHelp))
                .keyboardShortcut("o", modifiers: .command)
                .pulseOnHover(scale: 1.08, glowColor: .blue)

                Button {
                    store.refresh()
                } label: {
                    Label(LanguageManager.shared.text(.refresh), systemImage: "arrow.clockwise")
                }
                .keyboardShortcut("r", modifiers: .command)
                .help(LanguageManager.shared.text(.refreshHelp))
                .pulseOnHover(scale: 1.08, glowColor: .green)

                Divider()

                if store.runState.isRunning {
                    Button(role: .destructive) {
                        store.cancelCurrentTask()
                    } label: {
                        Label(LanguageManager.shared.text(.interruptPipeline), systemImage: "stop.fill")
                    }
                    .keyboardShortcut(".", modifiers: .command)
                    .help(LanguageManager.shared.text(.interruptPipelineHelp))
                    .pulseOnHover(scale: 1.08, glowColor: .red)
                } else {
                    Button {
                        store.runPipeline()
                    } label: {
                        Label(LanguageManager.shared.text(.runPipeline), systemImage: "play.fill")
                    }
                    .keyboardShortcut(.return, modifiers: .command)
                    .help(LanguageManager.shared.text(.runPipelineHelp))
                    .pulseOnHover(scale: 1.08, glowColor: .accentColor)
                }

                Divider()

                // GPS 剪贴板快速查看与状态入口 (⌥G)
                Button {
                    store.showCopiedGPSInspector()
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: store.copiedGPSMetadata != nil ? "location.circle.fill" : "location.circle")
                            .foregroundStyle(store.copiedGPSMetadata != nil ? Color.green : Color.secondary)
                        if let copied = store.copiedGPSMetadata {
                            Text(copied.formattedDecimal)
                                .font(.caption2.monospaced())
                                .foregroundStyle(.primary)
                        }
                    }
                }
                .help(LanguageManager.shared.text(.viewCopiedGPSAction))
                .pulseOnHover(scale: 1.05, glowColor: .teal)

                Divider()

                Button {
                    showingSettings = true
                } label: {
                    Label(LanguageManager.shared.text(.openSettings), systemImage: "gearshape")
                }
                .keyboardShortcut(",", modifiers: .command)
                .help(LanguageManager.shared.text(.openSettingsHelp))
                .pulseOnHover(scale: 1.08, glowColor: .purple)
            }
        }
        // 快捷键定义与触发
        .background(
            Group {
                // ⌘G: 快速拷贝 GPS
                Button("") {
                    store.copySelectedAssetGPS()
                }
                .keyboardShortcut("g", modifiers: .command)
                .opacity(0)

                // ⌥⌘G: 写入已拷贝 GPS
                Button("") {
                    store.pasteGPSToSelectedAsset()
                }
                .keyboardShortcut("g", modifiers: [.option, .command])
                .opacity(0)

                // ⌥G: 查看已拷贝 GPS 渲染详情
                Button("") {
                    store.showCopiedGPSInspector()
                }
                .keyboardShortcut("g", modifiers: .option)
                .opacity(0)
            }
        )
        // 全局浮动 HUD Toast 提示
        .overlay(alignment: .top) {
            if let msg = store.hudMessage {
                HStack(spacing: 8) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                    Text(msg)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.primary)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
                .background(.regularMaterial, in: Capsule())
                .overlay(
                    Capsule()
                        .stroke(Color.primary.opacity(0.12), lineWidth: 1)
                )
                .shadow(color: .black.opacity(0.18), radius: 10, x: 0, y: 5)
                .padding(.top, 14)
                .transition(.asymmetric(
                    insertion: .move(edge: .top).combined(with: .opacity),
                    removal: .opacity
                ))
                .zIndex(999)
            }
        }
        .animation(.spring(response: 0.3, dampingFraction: 0.75), value: store.hudMessage)
        .sheet(isPresented: $showingSettings) {
            SettingsView(store: store)
        }
        .sheet(isPresented: $store.showingCopiedGPSInspector) {
            CopiedGPSInspectorSheet(store: store)
        }
        .onAppear {
            store.refresh()
        }
    }

    private func chooseBaseDirectory() {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.prompt = LanguageManager.shared.text(.chooseBaseDirPrompt)

        if panel.runModal() == .OK, let url = panel.url {
            store.setBaseDirectoryAndResetPaths(url.path)
        }
    }

    // MARK: - 全局常驻底部状态栏
    private var statusBarView: some View {
        HStack(spacing: 12) {
            // 1. 左侧：工作区状态与资产统计
            HStack(spacing: 6) {
                Circle()
                    .fill(store.runState.isRunning ? Color.orange : Color.green)
                    .frame(width: 7, height: 7)
                if store.runState.isRunning {
                    Text(LanguageManager.shared.text(.pipelineRunning))
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                } else {
                    Text(store.baseDirectory.isEmpty ? LanguageManager.shared.text(.chooseDirectory) : URL(fileURLWithPath: store.baseDirectory).lastPathComponent)
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.secondary)
                }

                if let summary = store.summary, summary.readyCount > 0 {
                    Text("•")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    Text("\(summary.readyCount) \(LanguageManager.shared.text(.metricInbox))")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer()

            // 2. 中央/核心：元数据写入策略状态（高亮卡片，带切换菜单与说明，让摄影师一目了然）
            Menu {
                Section(LanguageManager.shared.text(.sidecarPolicy)) {
                    Button {
                        store.sidecarPolicy = "smart"
                    } label: {
                        HStack {
                            Text(LanguageManager.shared.text(.policySmart))
                            if store.sidecarPolicy == "smart" {
                                Image(systemName: "checkmark")
                            }
                        }
                    }

                    Button {
                        store.sidecarPolicy = "sidecar_only"
                    } label: {
                        HStack {
                            Text(LanguageManager.shared.text(.policySidecarOnly))
                            if store.sidecarPolicy == "sidecar_only" {
                                Image(systemName: "checkmark")
                            }
                        }
                    }

                    Button {
                        store.sidecarPolicy = "embed_and_sidecar"
                    } label: {
                        HStack {
                            Text(LanguageManager.shared.text(.policyEmbedAndSidecar))
                            if store.sidecarPolicy == "embed_and_sidecar" {
                                Image(systemName: "checkmark")
                            }
                        }
                    }

                    Button {
                        store.sidecarPolicy = "embed_only"
                    } label: {
                        HStack {
                            Text(LanguageManager.shared.text(.policyEmbedOnly))
                            if store.sidecarPolicy == "embed_only" {
                                Image(systemName: "checkmark")
                            }
                        }
                    }
                }
            } label: {
                HStack(spacing: 5) {
                    Image(systemName: store.sidecarPolicyIcon)
                        .foregroundStyle(store.sidecarPolicyColor)
                        .font(.caption2)
                    Text(LanguageManager.shared.text(.sidecarPolicy) + ":")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    Text(store.sidecarPolicyTitle)
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(store.sidecarPolicyColor)
                    Image(systemName: "chevron.up.chevron.down")
                        .font(.system(size: 8))
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(store.sidecarPolicyColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 6))
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .fixedSize()
            .help("\(LanguageManager.shared.text(.sidecarPolicy)): \(store.sidecarPolicyDescription)")

            Divider()
                .frame(height: 12)

            // 3. 右侧：引擎状态与语言
            HStack(spacing: 8) {
                HStack(spacing: 4) {
                    Image(systemName: PhotoolsEngine.shared.isLoaded ? "bolt.fill" : "terminal.fill")
                        .font(.system(size: 9))
                        .foregroundStyle(PhotoolsEngine.shared.isLoaded ? Color.green : Color.orange)
                    Text(PhotoolsEngine.shared.isLoaded ? "FFI 直通" : "CLI 进程")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Text(LanguageManager.shared.currentLanguage.displayName)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 5)
        .background(Color(nsColor: .windowBackgroundColor))
    }
}
