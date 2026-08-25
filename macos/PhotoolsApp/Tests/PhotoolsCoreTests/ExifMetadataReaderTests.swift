import XCTest
@testable import PhotoolsCore

final class ExifMetadataReaderTests: XCTestCase {
    func testExifMetadataModel() {
        let metadata = ExifMetadata(
            filePath: "/dummy/test.nef",
            fileName: "test.nef",
            fileSize: "45.2 MB",
            fileModifyDate: "2026-08-25 10:00:00",
            cameraMake: "NIKON CORPORATION",
            cameraModel: "NIKON Z 8",
            lensModel: "NIKKOR Z 24-70mm f/2.8 S",
            dateTimeOriginal: "2026-08-25 09:30:15",
            exposureTime: "1/250",
            fNumber: "2.8",
            iso: "100",
            focalLength: "35",
            latitude: 31.2304,
            longitude: 121.4737,
            altitude: 12.5,
            country: "中国",
            province: "上海市",
            city: "上海市",
            district: "黄浦区",
            rawTags: [
                ExifTagItem(group: "EXIF", tag: "Model", value: "NIKON Z 8"),
                ExifTagItem(group: "GPS", tag: "GPSLatitude", value: "31.2304")
            ]
        )

        XCTAssertTrue(metadata.hasGPS)
        XCTAssertEqual(metadata.cameraSummary, "NIKON CORPORATION NIKON Z 8")
        XCTAssertEqual(metadata.exposureSummary, "1/250s · f/2.8 · ISO 100 · 35mm")
        XCTAssertEqual(metadata.locationSummary, "中国 · 上海市 · 黄浦区")
        XCTAssertEqual(metadata.rawTags.count, 2)
    }

    func testExifMetadataParseFromDictionary() {
        let dict: [String: Any] = [
            "file_path": "/dummy/DSC_0001.NEF",
            "camera_make": "NIKON CORPORATION",
            "camera_model": "NIKON Z6_3",
            "lens_model": "NIKKOR Z 24-120mm f/4 S",
            "latitude": 42.671970,
            "longitude": 80.591590,
            "altitude": 1955.0,
            "raw_tags": [
                ["group": "IFD0", "tag": "Make", "value": "NIKON CORPORATION"],
                ["group": "IFD0", "tag": "Model", "value": "NIKON Z6_3"],
                ["group": "GPS", "tag": "GPSLatitude", "value": "42.671970"]
            ]
        ]

        let meta = ExifMetadata.parse(from: dict, fallbackPath: "/dummy/DSC_0001.NEF")
        XCTAssertEqual(meta.fileName, "DSC_0001.NEF")
        XCTAssertEqual(meta.cameraModel, "NIKON Z6_3")
        XCTAssertEqual(meta.latitude, 42.671970)
        XCTAssertEqual(meta.longitude, 80.591590)
        XCTAssertEqual(meta.altitude, 1955.0)
        XCTAssertEqual(meta.rawTags.count, 3)
        XCTAssertTrue(meta.hasGPS)
    }

    func testReaderWithNonExistentFileThrows() async {
        let reader = ExifMetadataReader()
        do {
            _ = try await reader.readMetadata(for: "/non/existent/path/photo.jpg")
            XCTFail("Should have thrown error for non existent file")
        } catch {
            XCTAssertTrue(true)
        }
    }
}
