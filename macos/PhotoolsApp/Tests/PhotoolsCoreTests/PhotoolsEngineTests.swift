import XCTest
@testable import PhotoolsCore

final class PhotoolsEngineTests: XCTestCase {
    func testEngineLoadingAndLookup() {
        let repoRoot = RepositoryLocator().locateRepositoryRoot()
        let dylibPath = (repoRoot as NSString).appendingPathComponent("dist/libphotools.dylib")

        guard FileManager.default.fileExists(atPath: dylibPath) else {
            // 如果是在纯隔离环境且未编译 dylib 则跳过真实 FFI
            return
        }

        let engine = PhotoolsEngine(customDylibPath: dylibPath)
        XCTAssertTrue(engine.isLoaded, "应该成功加载 libphotools.dylib")

        // 1. 测试初始化
        var initReports: [PhotoolsInitReport] = []
        engine.initialize { rep in
            initReports.append(rep)
        }
        XCTAssertTrue(engine.isInitialized)

        // 2. 测试内存经纬度高精反查 (上海 31.2304, 121.4737)
        let result = engine.lookupCoordinates(latitude: 31.2304, longitude: 121.4737)
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.country, "中国")
        XCTAssertEqual(result?.province, "上海市")

        // 3. 测试大洲数据包列表
        let packs = engine.listGeodataPacks()
        XCTAssertFalse(packs.isEmpty)
        XCTAssertTrue(packs.contains { $0.code == "china" })
    }

    func testEngineBackupAndRestoreFFI() async throws {
        let repoRoot = RepositoryLocator().locateRepositoryRoot()
        let dylibPath = (repoRoot as NSString).appendingPathComponent("dist/libphotools.dylib")

        guard FileManager.default.fileExists(atPath: dylibPath) else {
            return
        }

        let engine = PhotoolsEngine(customDylibPath: dylibPath)
        guard engine.isLoaded else { return }

        // 创建临时测试目录
        let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let inboxDir = tempDir.appendingPathComponent("Inbox")
        let backupDir = tempDir.appendingPathComponent("Inbox_bak")
        let processedDir = tempDir.appendingPathComponent("Processed")

        try FileManager.default.createDirectory(at: inboxDir, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: processedDir, withIntermediateDirectories: true)

        let dummyFile = inboxDir.appendingPathComponent("test.jpg")
        try "dummy photo".write(to: dummyFile, atomically: true, encoding: .utf8)

        // 1. 测试 C ABI createBackup
        let backupCount = try await engine.createBackup(sourceDir: inboxDir.path, backupDir: backupDir.path)
        XCTAssertEqual(backupCount, 1)
        XCTAssertTrue(FileManager.default.fileExists(atPath: backupDir.appendingPathComponent("test.jpg").path))

        // 模拟清空 Inbox 并测试 restoreBackup FFI (4 个参数正确匹配)
        try FileManager.default.removeItem(at: dummyFile)
        XCTAssertFalse(FileManager.default.fileExists(atPath: dummyFile.path))

        let restoreCount = try await engine.restoreBackup(
            baseDirectory: tempDir.path,
            backupDir: backupDir.path,
            targetDir: inboxDir.path,
            cleanProcessed: true
        )
        XCTAssertEqual(restoreCount, 1)
        XCTAssertTrue(FileManager.default.fileExists(atPath: dummyFile.path))

        try? FileManager.default.removeItem(at: tempDir)
    }
}
