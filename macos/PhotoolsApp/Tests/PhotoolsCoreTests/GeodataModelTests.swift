import XCTest
@testable import PhotoolsCore

final class GeodataModelTests: XCTestCase {
    func testParseGeodataListOutput() {
        let output = """
        🗺️  【全球各洲离线逆地理编码数据包列表】
          • china           [中国离线高精地名库 (含港澳台)]: ✅ 已安装 (715000 点位, 42.5 MB)
            └─ 包含中国34个省级行政区、300+地级市、2800+区县及全国著名风景名胜区
          • asia            [亚洲其他国家与地区]: ❌ 未安装
            └─ 包含日本、韩国、东南亚等亚洲邻国地名数据
          • europe          [欧洲全境高精地名]: ✅ 已安装 (1200000 点位, 85.0 MB)
            └─ 覆盖欧洲全境阿尔卑斯山脉、著名城镇与旅游胜地
        """

        let packs = GeodataParser.parseListOutput(output)
        XCTAssertEqual(packs.count, 3)

        XCTAssertEqual(packs[0].code, "china")
        XCTAssertEqual(packs[0].nameZH, "中国离线高精地名库 (含港澳台)")
        XCTAssertTrue(packs[0].isInstalled)
        XCTAssertEqual(packs[0].pointCount, 715000)
        XCTAssertEqual(packs[0].sizeMB, 42.5)

        XCTAssertEqual(packs[1].code, "asia")
        XCTAssertEqual(packs[1].nameZH, "亚洲其他国家与地区")
        XCTAssertFalse(packs[1].isInstalled)
        XCTAssertEqual(packs[1].pointCount, 0)

        XCTAssertEqual(packs[2].code, "europe")
        XCTAssertTrue(packs[2].isInstalled)
        XCTAssertEqual(packs[2].pointCount, 1200000)
    }

    func testParseGeodataLookupOutput() {
        let output = """
        🔍 正在检索经纬度坐标: (31.230000, 121.470000) [海拔: 10.0m]

        📍 【逆地理编码匹配结果】
          • 国家:       中国 (CN)
          • 省份/州:    上海市
          • 城市/地区:  上海市
          • 区县/POI:   黄浦区
          • IANA时区:   Asia/Shanghai
          • 地形海拔:   4 米
          • 物理距离:   0.85 km
          • 数据源:     内置离线地名库
          • 规范全称:   中国 · 上海市 · 黄浦区
        """

        let result = GeodataParser.parseLookupOutput(output)
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.country, "中国")
        XCTAssertEqual(result?.countryCode, "CN")
        XCTAssertEqual(result?.province, "上海市")
        XCTAssertEqual(result?.city, "上海市")
        XCTAssertEqual(result?.district, "黄浦区")
        XCTAssertEqual(result?.timezone, "Asia/Shanghai")
        XCTAssertEqual(result?.distanceKm, 0.85)
        XCTAssertEqual(result?.formattedSummary, "中国 · 上海市 · 黄浦区")
    }
}
