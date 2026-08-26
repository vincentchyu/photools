import XCTest
@testable import PhotoolsCore

final class WorkspaceScannerTests: XCTestCase {
    func testScanGroupsAndSummaryWithPrimaryAssetModel() throws {
        let root = try makeWorkspace()
        // RAW + JPG + companion
        try write(root.appendingPathComponent("Inbox/A001.NEF"))
        try write(root.appendingPathComponent("Inbox/A001.JPG"))
        try write(root.appendingPathComponent("Inbox/A001.xmp"))
        try write(root.appendingPathComponent("Inbox/A001.wav"))
        // Single RAW
        try write(root.appendingPathComponent("Inbox/B001.NEF"))
        // Single JPG
        try write(root.appendingPathComponent("Inbox/C001.JPG"))
        // 误放在 Inbox 中的轨迹或文档文件（必须被忽略，不作为独立照片资产）
        try write(root.appendingPathComponent("Inbox/track_wrong.gpx"))
        try write(root.appendingPathComponent("Inbox/notes.txt"))

        try write(root.appendingPathComponent("GPX/track.gpx"))
        try write(root.appendingPathComponent("GPX/notes.txt"))
        try write(root.appendingPathComponent("Processed/2025/1006/DSC_2025-10-06_1001.NEF"))
        try write(root.appendingPathComponent("Inbox_bak/A001.NEF"))
        try write(root.appendingPathComponent("Logs/inbox_pending_report_latest.md"), text: "# Inbox 待处理清单\n")

        let summary = try WorkspaceScanner().scan(baseDirectory: root.path, rawExtensions: ["nef"])

        // GPX 目录下只有 track.gpx 被纳入轨迹列表
        XCTAssertEqual(summary.gpxFiles.map { URL(fileURLWithPath: $0).lastPathComponent }, ["track.gpx"])
        
        // A001 (raw+jpg), B001 (raw), C001 (jpg) 均为主资产，总数为 3
        XCTAssertEqual(summary.assetGroups.count, 3)
        XCTAssertEqual(summary.readyCount, 3)
        XCTAssertEqual(summary.rawPairCount, 1)
        XCTAssertEqual(summary.rawOnlyCount, 1)
        XCTAssertEqual(summary.jpgOnlyCount, 1)
        XCTAssertEqual(summary.processedFileCount, 1)
        XCTAssertTrue(summary.hasBackup)
        XCTAssertEqual(summary.backupFileCount, 1)
        XCTAssertTrue(summary.pendingReportExists)

        let a001 = try XCTUnwrap(summary.assetGroups.first { $0.baseName == "A001" })
        XCTAssertEqual(a001.status, .ready)
        XCTAssertEqual(a001.primaryType, .rawPair)
        XCTAssertEqual(a001.companionPaths.map { URL(fileURLWithPath: $0).lastPathComponent }, ["A001.wav", "A001.xmp"])
        XCTAssertEqual(a001.xmpPath.map { URL(fileURLWithPath: $0).lastPathComponent }, "A001.xmp")

        let b001 = try XCTUnwrap(summary.assetGroups.first { $0.baseName == "B001" })
        XCTAssertEqual(b001.status, .ready)
        XCTAssertEqual(b001.primaryType, .rawOnly)

        let c001 = try XCTUnwrap(summary.assetGroups.first { $0.baseName == "C001" })
        XCTAssertEqual(c001.status, .ready)
        XCTAssertEqual(c001.primaryType, .jpgOnly)
    }

    func testScanShieldsBackupAndSystemDirectories() throws {
        let root = try makeWorkspace()
        // Inbox 下正常待处理照片
        try write(root.appendingPathComponent("Inbox/P001.NEF"))
        try write(root.appendingPathComponent("Inbox/P001.JPG"))

        // 在 baseDirectory 根下和 Inbox 子目录下放置备份目录与已归档目录
        try write(root.appendingPathComponent("Inbox_bak/P001.NEF"))
        try write(root.appendingPathComponent("Inbox_bak/P001.JPG"))
        try write(root.appendingPathComponent("Inbox/test_bak/P002.NEF"))
        try write(root.appendingPathComponent("Inbox/backup/P003.NEF"))
        try write(root.appendingPathComponent("Processed/2026/0825/P004.NEF"))
        try write(root.appendingPathComponent("GPX/track.gpx"))
        try write(root.appendingPathComponent("Logs/photools_latest.log"))

        // 1. 分层模式扫描 (sourceDirectory = Inbox)
        let hierSummary = try WorkspaceScanner().scan(
            baseDirectory: root.path,
            sourceDirectory: root.appendingPathComponent("Inbox").path,
            rawExtensions: ["nef"]
        )
        // 应该只有 P001，Inbox/test_bak 和 Inbox/backup 均被屏蔽
        XCTAssertEqual(hierSummary.assetGroups.count, 1)
        XCTAssertEqual(hierSummary.assetGroups.first?.baseName, "P001")

        // 2. 扁平模式扫描 (sourceDirectory = baseDirectory)
        let flatSummary = try WorkspaceScanner().scan(
            baseDirectory: root.path,
            sourceDirectory: root.path,
            rawExtensions: ["nef"]
        )
        // 扁平模式下扫描 baseDirectory，Inbox_bak / Processed / GPX / Logs / *_bak 均被完全屏蔽，只应识别 Inbox 下的照片
        XCTAssertEqual(flatSummary.assetGroups.count, 1)
        XCTAssertEqual(flatSummary.assetGroups.first?.baseName, "P001")
    }

    func testPendingReportCanBeMissing() throws {
        let root = try makeWorkspace()
        let summary = try WorkspaceScanner().scan(baseDirectory: root.path, rawExtensions: ["nef"])

        XCTAssertFalse(summary.pendingReportExists)
        XCTAssertEqual(summary.pendingReportText, "")
    }

    func testScanMissingDirectoryThrows() throws {
        XCTAssertThrowsError(try WorkspaceScanner().scan(baseDirectory: "/non/existent/photools/path"))
    }

    private func makeWorkspace() throws -> URL {
        let root = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent(UUID().uuidString)
        for dir in ["Inbox", "GPX", "Processed", "Logs", "Inbox_bak"] {
            try FileManager.default.createDirectory(
                at: root.appendingPathComponent(dir),
                withIntermediateDirectories: true
            )
        }
        return root
    }

    private func write(_ url: URL, text: String = "x") throws {
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        try text.write(to: url, atomically: true, encoding: .utf8)
    }
}
