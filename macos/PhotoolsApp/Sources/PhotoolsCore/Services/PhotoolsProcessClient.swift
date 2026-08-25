import Foundation

public enum PhotoolsProcessError: Error, LocalizedError {
    case executableMissing(String)
    case launchFailed(String)
    case cancelled
    case failed(exitCode: Int32)

    public var errorDescription: String? {
        switch self {
        case .executableMissing(let path):
            return "未找到 photools 可执行文件：\(path)"
        case .launchFailed(let message):
            return "启动 photools 失败：\(message)"
        case .cancelled:
            return "任务已被用户中断"
        case .failed(let exitCode):
            return "photools 执行退出码为 \(exitCode)"
        }
    }
}

public actor ProcessStateBox {
    private var process: Process?

    public init() {}

    public func setProcess(_ proc: Process?) {
        self.process = proc
    }

    public func terminate() {
        if let process, process.isRunning {
            process.terminate()
        }
        self.process = nil
    }
}

public final class PhotoolsProcessClient: Sendable {
    private let stateBox = ProcessStateBox()

    public init() {}

    public func cancel() {
        Task {
            await stateBox.terminate()
        }
    }

    public func run(
        command: PhotoolsCommand,
        onOutput: @escaping @Sendable (String) -> Void
    ) async throws {
        guard FileManager.default.isExecutableFile(atPath: command.executablePath) else {
            throw PhotoolsProcessError.executableMissing(command.executablePath)
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: command.executablePath)
        process.arguments = command.arguments

        let outputPipe = Pipe()
        let errorPipe = Pipe()
        process.standardOutput = outputPipe
        process.standardError = errorPipe

        outputPipe.fileHandleForReading.readabilityHandler = { handle in
            emitAvailableText(from: handle, onOutput: onOutput)
        }
        errorPipe.fileHandleForReading.readabilityHandler = { handle in
            emitAvailableText(from: handle, onOutput: onOutput)
        }

        await stateBox.setProcess(process)

        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            process.terminationHandler = { [weak self] proc in
                outputPipe.fileHandleForReading.readabilityHandler = nil
                errorPipe.fileHandleForReading.readabilityHandler = nil
                emitRemainingText(from: outputPipe.fileHandleForReading, onOutput: onOutput)
                emitRemainingText(from: errorPipe.fileHandleForReading, onOutput: onOutput)

                if let self {
                    Task {
                        await self.stateBox.setProcess(nil)
                    }
                }

                if proc.terminationStatus == 0 {
                    continuation.resume()
                } else if proc.terminationReason == .uncaughtSignal {
                    continuation.resume(throwing: PhotoolsProcessError.cancelled)
                } else {
                    continuation.resume(throwing: PhotoolsProcessError.failed(exitCode: proc.terminationStatus))
                }
            }

            do {
                try process.run()
            } catch {
                outputPipe.fileHandleForReading.readabilityHandler = nil
                errorPipe.fileHandleForReading.readabilityHandler = nil
                Task { [weak self] in
                    await self?.stateBox.setProcess(nil)
                }
                continuation.resume(throwing: PhotoolsProcessError.launchFailed(error.localizedDescription))
            }
        }
    }

    public func runGeotag(
        command: PhotoolsCommand,
        onOutput: @escaping @Sendable (String) -> Void
    ) async throws {
        try await run(command: command, onOutput: onOutput)
    }
}

private func emitAvailableText(from handle: FileHandle, onOutput: @escaping @Sendable (String) -> Void) {
    let data = handle.availableData
    guard !data.isEmpty, let text = String(data: data, encoding: .utf8), !text.isEmpty else {
        return
    }
    onOutput(text)
}

private func emitRemainingText(from handle: FileHandle, onOutput: @escaping @Sendable (String) -> Void) {
    let data = handle.readDataToEndOfFile()
    guard !data.isEmpty, let text = String(data: data, encoding: .utf8), !text.isEmpty else {
        return
    }
    onOutput(text)
}
