import Foundation
import AppKit
import QuickLookThumbnailing
import ImageIO

public actor PhotoThumbnailLoader {
    public static let shared = PhotoThumbnailLoader()

    private let cache = NSCache<NSString, NSImage>()

    public init() {
        cache.countLimit = 400
        cache.totalCostLimit = 200 * 1024 * 1024 // 200MB
    }

    public func loadThumbnail(for asset: PhotoAssetGroup, targetSize: CGSize = CGSize(width: 300, height: 300)) async -> NSImage? {
        let cacheKey = "\(asset.id)::\(Int(targetSize.width))x\(Int(targetSize.height))" as NSString
        if let cached = cache.object(forKey: cacheKey) {
            return cached
        }

        // 1. 如果存在配对的 JPG，优先极速从 JPG 渲染（毫秒级）
        if let jpgPath = asset.jpgPath, FileManager.default.fileExists(atPath: jpgPath) {
            if let image = createThumbnail(at: jpgPath, maxPixelSize: max(targetSize.width, targetSize.height) * 2) {
                cache.setObject(image, forKey: cacheKey)
                return image
            }
        }

        // 2. 纯 RAW / NEF 模式：优先使用 macOS 原生 QLThumbnailGenerator 提取 RAW 内嵌高质量 JPEG 预览
        if let rawPath = asset.rawPath, FileManager.default.fileExists(atPath: rawPath) {
            if let image = await generateQuickLookThumbnail(at: rawPath, targetSize: targetSize) {
                cache.setObject(image, forKey: cacheKey)
                return image
            }
            if let image = createThumbnail(at: rawPath, maxPixelSize: max(targetSize.width, targetSize.height) * 2) {
                cache.setObject(image, forKey: cacheKey)
                return image
            }
        }

        // 3. Fallback: 主路径
        if let primary = asset.primaryPath, FileManager.default.fileExists(atPath: primary) {
            if let image = await generateQuickLookThumbnail(at: primary, targetSize: targetSize) {
                cache.setObject(image, forKey: cacheKey)
                return image
            }
            if let image = createThumbnail(at: primary, maxPixelSize: max(targetSize.width, targetSize.height) * 2) {
                cache.setObject(image, forKey: cacheKey)
                return image
            }
        }

        return nil
    }

    private func generateQuickLookThumbnail(at filePath: String, targetSize: CGSize) async -> NSImage? {
        let url = URL(fileURLWithPath: filePath)
        let scale = NSScreen.main?.backingScaleFactor ?? 2.0
        let request = QLThumbnailGenerator.Request(
            fileAt: url,
            size: targetSize,
            scale: scale,
            representationTypes: .thumbnail
        )

        do {
            let rep = try await QLThumbnailGenerator.shared.generateBestRepresentation(for: request)
            return rep.nsImage
        } catch {
            return nil
        }
    }

    private func createThumbnail(at filePath: String, maxPixelSize: CGFloat) -> NSImage? {
        let url = URL(fileURLWithPath: filePath) as CFURL
        guard let source = CGImageSourceCreateWithURL(url, nil) else { return nil }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: maxPixelSize,
            kCGImageSourceShouldCacheImmediately: true
        ]

        guard let cgImage = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }
        return NSImage(cgImage: cgImage, size: NSSize(width: cgImage.width, height: cgImage.height))
    }
}
