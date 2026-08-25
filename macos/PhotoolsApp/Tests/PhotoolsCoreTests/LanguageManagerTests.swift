import XCTest
@testable import PhotoolsCore

@MainActor
final class LanguageManagerTests: XCTestCase {

    func testLanguageSwitching() {
        let manager = LanguageManager.shared

        // 1. 测试切换至中文
        manager.setLanguage(.zhHans)
        XCTAssertEqual(manager.currentLanguage, .zhHans)
        XCTAssertTrue(manager.currentLanguage.isChinese)
        XCTAssertEqual(manager.text(.preferencesTitle), "偏好设置")
        XCTAssertEqual(manager.text(.runPipeline), "执行流水线")
        XCTAssertEqual(manager.text(.sectionInbox), "待处理照片")

        // 2. 测试切换至英文
        manager.setLanguage(.en)
        XCTAssertEqual(manager.currentLanguage, .en)
        XCTAssertFalse(manager.currentLanguage.isChinese)
        XCTAssertEqual(manager.text(.preferencesTitle), "Preferences")
        XCTAssertEqual(manager.text(.runPipeline), "Run Pipeline")
        XCTAssertEqual(manager.text(.sectionInbox), "Inbox Photos")

        // 3. 测试所有 L10nKey 在中英文模式下均非空
        for key in L10nKey.allCases {
            manager.setLanguage(.zhHans)
            let zhVal = manager.text(key)
            XCTAssertFalse(zhVal.isEmpty, "Missing Chinese localization for key: \(key)")

            manager.setLanguage(.en)
            let enVal = manager.text(key)
            XCTAssertFalse(enVal.isEmpty, "Missing English localization for key: \(key)")
        }
    }

    func testAppLanguageProperties() {
        XCTAssertEqual(AppLanguage.zhHans.effectiveLanguageCode, "zh-Hans")
        XCTAssertEqual(AppLanguage.en.effectiveLanguageCode, "en")
        XCTAssertTrue(AppLanguage.zhHans.isChinese)
        XCTAssertFalse(AppLanguage.en.isChinese)
    }
}
