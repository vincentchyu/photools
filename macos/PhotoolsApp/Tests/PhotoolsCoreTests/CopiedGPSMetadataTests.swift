import Foundation
@testable import PhotoolsCore
import XCTest

final class CopiedGPSMetadataTests: XCTestCase {
    func testCopiedGPSMetadataFormatting() {
        let meta = CopiedGPSMetadata(
            latitude: 31.230416,
            longitude: 121.473722,
            altitude: 15.5,
            sourceAssetBaseName: "DSC_1001",
            sourceFilePath: "/photos/DSC_1001.NEF",
            captureDate: "2026:09:02 12:00:00",
            gpsSource: "camera",
            locationSummary: "中国 · 上海市 · 黄浦区"
        )

        XCTAssertTrue(meta.formattedDecimal.contains("31.230416° N"))
        XCTAssertTrue(meta.formattedDecimal.contains("121.473722° E"))
        XCTAssertTrue(meta.formattedDMS.contains("31°13'49.50\" N"))
        XCTAssertTrue(meta.formattedDMS.contains("121°28'25.40\" E"))
        XCTAssertEqual(meta.formattedAltitude, "15.5 m")
        XCTAssertTrue(meta.plainTextSummary.contains("【GPS 坐标】"))
        XCTAssertTrue(meta.plainTextSummary.contains("【来源照片】 DSC_1001"))
    }

    func testNegativeCoordinatesDMS() {
        let meta = CopiedGPSMetadata(
            latitude: -33.8688,
            longitude: -151.2093,
            altitude: -5.0,
            sourceAssetBaseName: "IMG_2000",
            sourceFilePath: "/photos/IMG_2000.JPG"
        )

        XCTAssertTrue(meta.formattedDecimal.contains("33.868800° S"))
        XCTAssertTrue(meta.formattedDecimal.contains("151.209300° W"))
        XCTAssertTrue(meta.formattedDMS.contains("S"))
        XCTAssertTrue(meta.formattedDMS.contains("W"))
        XCTAssertEqual(meta.formattedAltitude, "-5.0 m")
    }

    func testCodableSerialization() throws {
        let original = CopiedGPSMetadata(
            latitude: 39.9042,
            longitude: 116.3917,
            altitude: 43.0,
            sourceAssetBaseName: "NIKON_001",
            sourceFilePath: "/test/NIKON_001.NEF"
        )

        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(CopiedGPSMetadata.self, from: data)

        XCTAssertEqual(original.latitude, decoded.latitude)
        XCTAssertEqual(original.longitude, decoded.longitude)
        XCTAssertEqual(original.altitude, decoded.altitude)
        XCTAssertEqual(original.sourceAssetBaseName, decoded.sourceAssetBaseName)
    }
}
