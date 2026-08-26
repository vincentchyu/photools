import AppKit
import PhotoolsCore

public final class AppDelegate: NSObject, NSApplicationDelegate {
    public func applicationDidFinishLaunching(_ notification: Notification) {
        // 注册应用即将终止的全局通知作为双重保险
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleWillTerminate),
            name: NSApplication.willTerminateNotification,
            object: nil
        )
    }

    public func applicationWillTerminate(_ notification: Notification) {
        cleanupResources()
    }

    public func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        // 主窗口关闭后自动退出 App，确保彻底释放进程与内存
        return true
    }

    @objc private func handleWillTerminate() {
        cleanupResources()
    }

    private func cleanupResources() {
        // 优雅退出 Go 动态库与常驻 ExifTool 进程池
        PhotoolsEngine.shared.shutdown()
    }
}
