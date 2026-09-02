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
        let candidates = [
            URL(fileURLWithPath: repoRoot).appendingPathComponent("photools").path,
            URL(fileURLWithPath: repoRoot).appendingPathComponent("dist").appendingPathComponent("photools").path,
            "/opt/homebrew/bin/photools",
            "/usr/local/bin/photools",
            FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".local/bin/photools").path
        ]
        for candidate in candidates {
            if FileManager.default.isExecutableFile(atPath: candidate) {
                return candidate
            }
        }
        return candidates[0]
    }
}
