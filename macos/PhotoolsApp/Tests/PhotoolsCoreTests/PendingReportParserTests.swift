import XCTest
@testable import PhotoolsCore

final class PendingReportParserTests: XCTestCase {
    func testFindsAssetSection() {
        let report = """
        # Inbox 待处理清单

        ## 1. A001

        - 原因：缺少同 basename 的 JPG

        ## 2. B001

        - 原因：轨迹时间不匹配
        """

        let section = PendingReportParser().section(for: "B001", in: report)

        XCTAssertNotNil(section)
        XCTAssertTrue(section?.contains("轨迹时间不匹配") == true)
    }
}
