import Foundation

public final class ExifMetadataReader: Sendable {
    public init() {}

    public func readMetadata(for filePath: String) async throws -> ExifMetadata {
        guard FileManager.default.fileExists(atPath: filePath) else {
            throw NSError(domain: "ExifMetadataReader", code: 404, userInfo: [NSLocalizedDescriptionKey: "文件不存在: \(filePath)"])
        }

        // 1. 优先使用进程内 FFI 引擎极速直通读取
        if PhotoolsEngine.shared.isLoaded {
            if let meta = PhotoolsEngine.shared.inspectPhotoMetadata(filePath: filePath) {
                return meta
            }
        }

        // 2. Fallback: 外部子进程模式
        let exiftoolPath = findExifToolPath()
        guard let exiftoolPath, FileManager.default.isExecutableFile(atPath: exiftoolPath) else {
            return fallbackMetadata(for: filePath)
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: exiftoolPath)

        var args = ["-m", "-q", "-j", "-G1", "-a", "-s", "-c", "%.6f", filePath]
        let sidecarCandidates = [
            filePath + ".xmp",
            filePath + ".XMP",
            (filePath as NSString).deletingPathExtension + ".xmp",
            (filePath as NSString).deletingPathExtension + ".XMP"
        ]
        var foundSidecar: String?
        for cand in sidecarCandidates {
            if FileManager.default.fileExists(atPath: cand) && cand != filePath {
                foundSidecar = cand
                args.append(cand)
                break
            }
        }
        process.arguments = args

        let outputPipe = Pipe()
        let errorPipe = Pipe()
        process.standardOutput = outputPipe
        process.standardError = errorPipe

        do {
            try process.run()
        } catch {
            return fallbackMetadata(for: filePath)
        }

        let data = outputPipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()

        guard !data.isEmpty,
              let jsonArray = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]],
              let firstItem = jsonArray.first else {
            return fallbackMetadata(for: filePath)
        }

        var merged = firstItem
        if jsonArray.count > 1 {
            let xmpDict = jsonArray[1]
            for (k, v) in xmpDict {
                if merged[k] == nil {
                    merged[k] = v
                }
            }
            if let sidecar = foundSidecar {
                merged["sidecar_path"] = sidecar
                merged["geocode_source"] = "xmp"
            }
        }

        return ExifMetadata.parse(from: merged, fallbackPath: filePath)
    }

    private func findExifToolPath() -> String? {
        let candidates = [
            "/opt/homebrew/bin/exiftool",
            "/usr/local/bin/exiftool",
            "/usr/bin/exiftool",
            Bundle.main.resourcePath.map { ($0 as NSString).appendingPathComponent("vendor/exiftool/exiftool") } ?? ""
        ]
        for path in candidates where !path.isEmpty {
            if FileManager.default.isExecutableFile(atPath: path) {
                return path
            }
        }
        return nil
    }

    private func fallbackMetadata(for filePath: String) -> ExifMetadata {
        let fileURL = URL(fileURLWithPath: filePath)
        let attrs = (try? FileManager.default.attributesOfItem(atPath: filePath)) ?? [:]
        let sizeBytes = (attrs[.size] as? Int64) ?? 0
        let modDate = (attrs[.modificationDate] as? Date) ?? Date()

        let formatter = ByteCountFormatter()
        formatter.allowedUnits = [.useMB, .useKB]
        formatter.countStyle = .file
        let sizeStr = formatter.string(fromByteCount: sizeBytes)

        let dateFmt = DateFormatter()
        dateFmt.dateFormat = "yyyy-MM-dd HH:mm:ss"

        return ExifMetadata(
            filePath: filePath,
            fileName: fileURL.lastPathComponent,
            fileSize: sizeStr,
            fileModifyDate: dateFmt.string(from: modDate)
        )
    }
}
