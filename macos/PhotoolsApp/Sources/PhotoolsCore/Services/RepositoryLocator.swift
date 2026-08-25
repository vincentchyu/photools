import Foundation

public struct RepositoryLocator: Sendable {
    public init() {}

    public func locateRepositoryRoot(bundleURL: URL = Bundle.main.bundleURL) -> String {
        let bundlePath = bundleURL.path
        if bundlePath.hasSuffix("/dist/PhotoolsApp.app") {
            return bundleURL
                .deletingLastPathComponent()
                .deletingLastPathComponent()
                .path
        }
        return FileManager.default.currentDirectoryPath
    }

    public func photoolsExecutablePath(repoRoot: String) -> String {
        let localBinary = URL(fileURLWithPath: repoRoot).appendingPathComponent("photools").path
        if FileManager.default.isExecutableFile(atPath: localBinary) {
            return localBinary
        }
        return URL(fileURLWithPath: repoRoot)
            .appendingPathComponent("dist")
            .appendingPathComponent("photools")
            .path
    }
}
