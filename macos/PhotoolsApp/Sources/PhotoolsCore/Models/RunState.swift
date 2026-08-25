import Foundation

public enum RunState: Equatable, Sendable {
    case idle
    case running
    case succeeded
    case failed(String)

    public var title: String {
        switch self {
        case .idle:
            return "未运行"
        case .running:
            return "正在处理"
        case .succeeded:
            return "处理完成"
        case .failed:
            return "处理失败"
        }
    }

    public var isRunning: Bool {
        if case .running = self {
            return true
        }
        return false
    }
}
