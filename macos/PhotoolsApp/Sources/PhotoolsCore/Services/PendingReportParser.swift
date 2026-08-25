import Foundation

public struct PendingReportParser: Sendable {
    public init() {}

    public func section(for baseName: String, in report: String) -> String? {
        let sections = report.components(separatedBy: "\n## ")
        return sections.first { section in
            guard let titleLine = section.split(separator: "\n", maxSplits: 1).first else {
                return false
            }
            return titleLine.trimmingCharacters(in: .whitespacesAndNewlines).hasSuffix(". \(baseName)")
        }
    }
}
