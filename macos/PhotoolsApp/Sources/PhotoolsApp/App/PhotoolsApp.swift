import PhotoolsCore
import SwiftUI

@main
struct PhotoolsApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var store = WorkspaceStore()

    var body: some Scene {
        WindowGroup("photools") {
            ContentView(store: store)
                .frame(minWidth: 1080, minHeight: 680)
        }
        .commands {
            CommandGroup(replacing: .newItem) {}
        }

        Settings {
            SettingsView(store: store)
        }
    }
}
