import AppKit
import SwiftUI

/// 鼠标悬停时的动态脉冲与微交互效果修饰器
public struct PulseHoverModifier: ViewModifier {
    @State private var isHovered: Bool = false
    @State private var isPulsing: Bool = false

    public var scale: CGFloat
    public var glowColor: Color
    public var isEnabled: Bool
    public var changeCursor: Bool

    public init(
        scale: CGFloat = 1.06,
        glowColor: Color = Color.accentColor,
        isEnabled: Bool = true,
        changeCursor: Bool = true
    ) {
        self.scale = scale
        self.glowColor = glowColor
        self.isEnabled = isEnabled
        self.changeCursor = changeCursor
    }

    public func body(content: Content) -> some View {
        if isEnabled {
            content
                .scaleEffect(isHovered ? (isPulsing ? scale * 1.025 : scale) : 1.0)
                .shadow(
                    color: isHovered ? glowColor.opacity(isPulsing ? 0.45 : 0.22) : Color.clear,
                    radius: isHovered ? (isPulsing ? 8 : 4) : 0,
                    x: 0,
                    y: isHovered ? 2 : 0
                )
                .animation(.spring(response: 0.28, dampingFraction: 0.62), value: isHovered)
                .animation(
                    isHovered
                        ? .easeInOut(duration: 0.85).repeatForever(autoreverses: true)
                        : .default,
                    value: isPulsing
                )
                .onHover { hovering in
                    isHovered = hovering
                    isPulsing = hovering
                }
        } else {
            content
        }
    }
}

public extension View {
    /// 为按钮或视图添加鼠标悬停时的脉冲呼吸与吸引点击效果
    func pulseOnHover(
        scale: CGFloat = 1.06,
        glowColor: Color = Color.accentColor,
        isEnabled: Bool = true,
        changeCursor: Bool = true
    ) -> some View {
        self.modifier(
            PulseHoverModifier(
                scale: scale,
                glowColor: glowColor,
                isEnabled: isEnabled,
                changeCursor: changeCursor
            )
        )
    }
}
