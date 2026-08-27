import PhotoolsCore
import SwiftUI

public struct SettingsView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(spacing: 0) {
            // 顶部标题与关闭按钮栏
            HStack {
                Text(lang.text(.preferencesTitle))
                    .font(.headline)
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
            .padding(.horizontal, 20)
            .padding(.top, 16)
            .padding(.bottom, 8)

            TabView {
                generalTab
                    .tabItem {
                        Label(lang.text(.tabGeneral), systemImage: "gearshape")
                    }

                pluginsTab
                    .tabItem {
                        Label(lang.text(.tabPlugins), systemImage: "slider.horizontal.3")
                    }
            }
            .padding(.horizontal, 20)

            Divider()

            // 底部操作栏
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
        .frame(width: 600, height: 530)
        .onDisappear {
            store.refresh()
        }
    }

    private var generalTab: some View {
        Form {
            Section(lang.text(.languageSetting)) {
                Picker(lang.text(.languageSetting), selection: $lang.currentLanguage) {
                    ForEach(AppLanguage.allCases) { item in
                        Text(item.displayName).tag(item)
                    }
                }
                .pickerStyle(.menu)
            }

            Section(lang.text(.workspaceConfig)) {
                HStack {
                    TextField(lang.text(.baseDirectory), text: $store.baseDirectory)
                    Button(lang.text(.chooseDirectory)) {
                        chooseDirectory { path in
                            store.setBaseDirectoryAndResetPaths(path)
                        }
                    }
                }

                Toggle(lang.text(.flatMode), isOn: $store.flatMode)

                HStack {
                    TextField(lang.text(.gpxDirectory), text: $store.gpxDirectory)
                    Button(lang.text(.chooseDirectory)) {
                        chooseDirectory { path in
                            store.gpxDirectory = path
                        }
                    }
                }

                if !store.flatMode {
                    HStack {
                        TextField(lang.text(.sourceDirectory), text: $store.sourceDirectory)
                        Button(lang.text(.chooseDirectory)) {
                            chooseDirectory { path in
                                store.sourceDirectory = path
                            }
                        }
                    }

                    HStack {
                        TextField(lang.text(.processedDirectory), text: $store.processedDirectory)
                        Button(lang.text(.chooseDirectory)) {
                            chooseDirectory { path in
                                store.processedDirectory = path
                            }
                        }
                    }
                }
            }

            Section(lang.text(.performanceSection)) {
                TextField(lang.text(.rawExtensions), text: $store.rawExtensions)
                Stepper("\(lang.text(.concurrencyWorkers))：\(store.workers)", value: $store.workers, in: 1...32)
                Toggle(lang.text(.testBackupMode), isOn: $store.testBackup)
            }
        }
    }

    private var pluginsTab: some View {
        Form {
            Section(lang.text(.pluginDefaults)) {
                Toggle(lang.text(.pluginGpxMatch), isOn: $store.enableGPXMatch)
                Toggle(lang.text(.pluginInterpolate), isOn: $store.enableInterpolate)
                Toggle(lang.text(.pluginGeocode), isOn: $store.enableGeocode)
                Toggle(lang.text(.pluginArchive), isOn: $store.enableArchive)
            }

            Section(lang.text(.pluginParams)) {
                TextField(lang.text(.geosyncOffset), text: $store.geosync)
                Picker(lang.text(.interpolateWindow), selection: $store.interpolateWindow) {
                    Text(lang.text(.window15m)).tag("15m")
                    Text(lang.text(.window30m)).tag("30m")
                    Text(lang.text(.window1h)).tag("1h")
                    Text(lang.text(.window2h)).tag("2h")
                    Text(lang.text(.window4h)).tag("4h")
                }
                Toggle(lang.text(.inPlaceRename), isOn: $store.inPlace)
                Toggle(lang.text(.allowNoGps), isOn: $store.allowNoGPS)
            }
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
