import PhotoolsCore
import SwiftUI

public struct ContentView: View {
    @ObservedObject var store: WorkspaceStore
    @State private var showingSettings = false

    public init(store: WorkspaceStore? = nil) {
        self.store = store ?? WorkspaceStore()
    }

    public var body: some View {
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

                Button {
                    store.refresh()
                } label: {
                    Label(LanguageManager.shared.text(.refresh), systemImage: "arrow.clockwise")
                }
                .keyboardShortcut("r", modifiers: .command)
                .help(LanguageManager.shared.text(.refreshHelp))

                Divider()

                if store.runState.isRunning {
                    Button(role: .destructive) {
                        store.cancelCurrentTask()
                    } label: {
                        Label(LanguageManager.shared.text(.interruptPipeline), systemImage: "stop.fill")
                    }
                    .keyboardShortcut(".", modifiers: .command)
                    .help(LanguageManager.shared.text(.interruptPipelineHelp))
                } else {
                    Button {
                        store.runPipeline()
                    } label: {
                        Label(LanguageManager.shared.text(.runPipeline), systemImage: "play.fill")
                    }
                    .keyboardShortcut(.return, modifiers: .command)
                    .help(LanguageManager.shared.text(.runPipelineHelp))
                }

                Divider()

                Button {
                    showingSettings = true
                } label: {
                    Label(LanguageManager.shared.text(.openSettings), systemImage: "gearshape")
                }
                .keyboardShortcut(",", modifiers: .command)
                .help(LanguageManager.shared.text(.openSettingsHelp))
            }
        }
        .sheet(isPresented: $showingSettings) {
            SettingsView(store: store)
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
}
