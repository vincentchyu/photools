import PhotoolsCore
import SwiftUI
import QuickLook

public struct PhotoThumbnailView: View {
    let asset: PhotoAssetGroup
    var targetSize: CGSize
    var contentMode: ContentMode
    var cornerRadius: CGFloat
    var fixedSize: CGSize?

    @State private var thumbnail: NSImage?
    @State private var isLoading: Bool = true

    public init(
        asset: PhotoAssetGroup,
        targetSize: CGSize = CGSize(width: 80, height: 80),
        contentMode: ContentMode = .fill,
        cornerRadius: CGFloat = 8,
        fixedSize: CGSize? = nil
    ) {
        self.asset = asset
        self.targetSize = targetSize
        self.contentMode = contentMode
        self.cornerRadius = cornerRadius
        self.fixedSize = fixedSize
    }

    public var body: some View {
        ZStack {
            if let image = thumbnail {
                Image(nsImage: image)
                    .resizable()
                    .aspectRatio(contentMode: contentMode)
                    .transition(.opacity.animation(.easeInOut(duration: 0.2)))
            } else {
                ZStack {
                    Color(nsColor: .controlBackgroundColor).opacity(0.6)

                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.6)
                    } else {
                        Image(systemName: "photo")
                            .font(.system(size: 20))
                            .foregroundStyle(.secondary.opacity(0.5))
                    }
                }
            }
        }
        .frame(
            width: fixedSize?.width,
            height: fixedSize?.height
        )
        .clipped()
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
        .task(id: asset.id) {
            isLoading = true
            thumbnail = await PhotoThumbnailLoader.shared.loadThumbnail(for: asset, targetSize: targetSize)
            isLoading = false
        }
    }
}
