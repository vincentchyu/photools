import XCTest
@testable import PhotoolsCore

final class PhotoolsCommandTests: XCTestCase {
    func testGeotagCommandKeepsArgumentsSeparate() {
        let command = PhotoolsCommand.geotag(
            executablePath: "/repo/photools",
            options: GeotagRunOptions(
                baseDirectory: "/workspace/GPS",
                geosync: "+00:00:05",
                rawExtensions: "nef,cr3,arw",
                workers: 4
            )
        )

        XCTAssertEqual(command.executablePath, "/repo/photools")
        XCTAssertEqual(command.arguments, [
            "geotag",
            "-base-dir", "/workspace/GPS",
            "-geosync", "+00:00:05",
            "-raw-exts", "nef,cr3,arw",
            "-workers", "4"
        ])
    }

    func testPipelineCommandWithFullCapabilities() {
        let options = PipelineRunOptions(
            baseDirectory: "/workspace/GPS",
            sourceDirectory: "/workspace/GPS/Inbox",
            gpxDirectory: "/workspace/GPS/GPX",
            processedDirectory: "/workspace/GPS/Processed",
            flatMode: false,
            inPlace: false,
            geosync: "0",
            rawExtensions: "nef,dng",
            workers: 8,
            enableGPX: true,
            enableInterpolate: true,
            interpolateWindow: "30m",
            enableGeocode: true,
            allowNoGPS: true,
            enableArchive: true,
            testBackup: true,
            backupDirectory: "/workspace/GPS/Inbox_bak"
        )

        let command = PhotoolsCommand.pipeline(executablePath: "/repo/photools", options: options)

        XCTAssertEqual(command.executablePath, "/repo/photools")
        XCTAssertEqual(command.arguments, [
            "pipeline",
            "-base-dir", "/workspace/GPS",
            "-source-dir", "/workspace/GPS/Inbox",
            "-gpx-dir", "/workspace/GPS/GPX",
            "-processed-dir", "/workspace/GPS/Processed",
            "-geosync", "0",
            "-raw-exts", "nef,dng",
            "-workers", "8",
            "-interpolate-window", "30m",
            "-backup-dir", "/workspace/GPS/Inbox_bak",
            "-sidecar-policy", "smart",
            "-companion-exts", "wav,acr,exf",
            "-gpx",
            "-interpolate",
            "-geocode",
            "-allow-no-gps",
            "-archive",
            "-test"
        ])
    }

    func testPipelineCommandFlatAndInPlace() {
        let options = PipelineRunOptions(
            baseDirectory: "/workspace/Photos2026",
            sourceDirectory: "/workspace/Photos2026",
            gpxDirectory: "/workspace/Photos2026/GPX",
            processedDirectory: "/workspace/Photos2026",
            flatMode: true,
            inPlace: true,
            geosync: "+00:00:10",
            rawExtensions: "nef",
            workers: 4,
            enableGPX: false,
            enableInterpolate: false,
            interpolateWindow: "15m",
            enableGeocode: true,
            allowNoGPS: false,
            enableArchive: true,
            testBackup: false,
            backupDirectory: ""
        )

        let command = PhotoolsCommand.pipeline(executablePath: "/repo/photools", options: options)

        XCTAssertEqual(command.arguments, [
            "pipeline",
            "-base-dir", "/workspace/Photos2026",
            "-source-dir", "/workspace/Photos2026",
            "-gpx-dir", "/workspace/Photos2026/GPX",
            "-processed-dir", "/workspace/Photos2026",
            "-geosync", "+00:00:10",
            "-raw-exts", "nef",
            "-workers", "4",
            "-interpolate-window", "15m",
            "-flat",
            "-sidecar-policy", "smart",
            "-companion-exts", "wav,acr,exf",
            "-in-place",
            "-geocode",
            "-archive"
        ])
    }

    func testRestoreTestCommand() {
        let command = PhotoolsCommand.restoreTest(
            executablePath: "/repo/photools",
            baseDirectory: "/workspace/GPS",
            backupDir: "/workspace/GPS/Inbox_bak",
            targetDir: "/workspace/GPS/Inbox",
            cleanProcessed: true
        )

        XCTAssertEqual(command.arguments, [
            "restore-test",
            "-base-dir", "/workspace/GPS",
            "-backup-dir", "/workspace/GPS/Inbox_bak",
            "-target-dir", "/workspace/GPS/Inbox",
            "-clean"
        ])
    }

    func testGeodataCommands() {
        let listCmd = PhotoolsCommand.geodataList(executablePath: "/repo/photools")
        XCTAssertEqual(listCmd.arguments, ["geodata", "list"])

        let installCmd = PhotoolsCommand.geodataInstall(executablePath: "/repo/photools", target: "china")
        XCTAssertEqual(installCmd.arguments, ["geodata", "install", "china"])

        let removeCmd = PhotoolsCommand.geodataRemove(executablePath: "/repo/photools", target: "asia")
        XCTAssertEqual(removeCmd.arguments, ["geodata", "remove", "asia"])

        let infoCmd = PhotoolsCommand.geodataInfo(executablePath: "/repo/photools")
        XCTAssertEqual(infoCmd.arguments, ["geodata", "info"])

        let testCmd = PhotoolsCommand.geodataTest(executablePath: "/repo/photools", latitude: 31.23, longitude: 121.47, altitude: 10.0, debug: true)
        XCTAssertEqual(testCmd.arguments, ["geodata", "test", "31.23", "121.47", "10", "--debug"])
    }
}
