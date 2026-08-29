#!/usr/bin/env python3
"""
generate_app_icon.py
Generates the 1024x1024 AppIcon.png and compiles the multi-resolution AppIcon.icns
using native macOS CoreGraphics, sips, and iconutil.
"""

import os
import math
import shutil
import subprocess
import sys
import ctypes
from ctypes import (
    c_void_p, c_char_p, c_size_t, c_double, c_uint32, c_int, c_bool,
    Structure, byref, POINTER
)

# Load macOS System Frameworks
cg = ctypes.cdll.LoadLibrary('/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics')
iio = ctypes.cdll.LoadLibrary('/System/Library/Frameworks/ImageIO.framework/ImageIO')
cf = ctypes.cdll.LoadLibrary('/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation')

class CGPoint(Structure):
    _fields_ = [('x', c_double), ('y', c_double)]

class CGSize(Structure):
    _fields_ = [('width', c_double), ('height', c_double)]

class CGRect(Structure):
    _fields_ = [('origin', CGPoint), ('size', CGSize)]

# Setup C function prototypes
cf.CFStringCreateWithCString.restype = c_void_p
cf.CFStringCreateWithCString.argtypes = [c_void_p, c_char_p, c_uint32]
kCFStringEncodingUTF8 = 0x08000100

cf.CFURLCreateFromFileSystemRepresentation.restype = c_void_p
cf.CFURLCreateFromFileSystemRepresentation.argtypes = [c_void_p, c_char_p, c_size_t, c_bool]
cf.CFRelease.argtypes = [c_void_p]

cg.CGColorSpaceCreateDeviceRGB.restype = c_void_p

cg.CGBitmapContextCreate.restype = c_void_p
cg.CGBitmapContextCreate.argtypes = [
    c_void_p, c_size_t, c_size_t, c_size_t, c_size_t, c_void_p, c_uint32
]

cg.CGContextSaveGState.argtypes = [c_void_p]
cg.CGContextRestoreGState.argtypes = [c_void_p]

cg.CGPathCreateWithRoundedRect.restype = c_void_p
cg.CGPathCreateWithRoundedRect.argtypes = [CGRect, c_double, c_double, c_void_p]

cg.CGContextAddPath.argtypes = [c_void_p, c_void_p]
cg.CGContextClip.argtypes = [c_void_p]
cg.CGContextDrawImage.argtypes = [c_void_p, CGRect, c_void_p]

cg.CGColorCreate.restype = c_void_p
cg.CGColorCreate.argtypes = [c_void_p, POINTER(c_double)]

cg.CGContextSetShadowWithColor.argtypes = [c_void_p, CGSize, c_double, c_void_p]
cg.CGContextSetStrokeColorWithColor.argtypes = [c_void_p, c_void_p]
cg.CGContextSetFillColorWithColor.argtypes = [c_void_p, c_void_p]
cg.CGContextSetLineWidth.argtypes = [c_void_p, c_double]
cg.CGContextStrokePath.argtypes = [c_void_p]
cg.CGContextFillPath.argtypes = [c_void_p]

cg.CGContextAddArc.argtypes = [c_void_p, c_double, c_double, c_double, c_double, c_double, c_int]
cg.CGContextMoveToPoint.argtypes = [c_void_p, c_double, c_double]
cg.CGContextAddLineToPoint.argtypes = [c_void_p, c_double, c_double]
cg.CGContextAddCurveToPoint.argtypes = [c_void_p, c_double, c_double, c_double, c_double, c_double, c_double]
cg.CGContextClosePath.argtypes = [c_void_p]

# Gradients
cg.CGGradientCreateWithColors.restype = c_void_p
cg.CGGradientCreateWithColors.argtypes = [c_void_p, c_void_p, POINTER(c_double)]
cf.CFArrayCreate.restype = c_void_p
cf.CFArrayCreate.argtypes = [c_void_p, POINTER(c_void_p), c_size_t, c_void_p]

cg.CGContextDrawLinearGradient.argtypes = [c_void_p, c_void_p, CGPoint, CGPoint, c_uint32]
cg.CGContextDrawRadialGradient.argtypes = [c_void_p, c_void_p, CGPoint, c_double, CGPoint, c_double, c_uint32]

cg.CGBitmapContextCreateImage.restype = c_void_p
cg.CGBitmapContextCreateImage.argtypes = [c_void_p]

iio.CGImageDestinationCreateWithURL.restype = c_void_p
iio.CGImageDestinationCreateWithURL.argtypes = [c_void_p, c_void_p, c_size_t, c_void_p]
iio.CGImageDestinationAddImage.argtypes = [c_void_p, c_void_p, c_void_p]
iio.CGImageDestinationFinalize.restype = c_bool
iio.CGImageDestinationFinalize.argtypes = [c_void_p]

def make_color(color_space, r, g, b, a):
    comps = (c_double * 4)(r, g, b, a)
    return cg.CGColorCreate(color_space, comps)

def make_gradient(color_space, color_list, locations=None):
    n = len(color_list)
    c_arr = (c_void_p * n)(*color_list)
    cf_arr = cf.CFArrayCreate(None, c_arr, n, None)
    if locations:
        locs = (c_double * n)(*locations)
        grad = cg.CGGradientCreateWithColors(color_space, cf_arr, locs)
    else:
        grad = cg.CGGradientCreateWithColors(color_space, cf_arr, None)
    cf.CFRelease(cf_arr)
    return grad

def save_image(img, filepath: str):
    b_path = filepath.encode('utf-8')
    url = cf.CFURLCreateFromFileSystemRepresentation(None, b_path, len(b_path), False)
    png_type = cf.CFStringCreateWithCString(None, b"public.png", kCFStringEncodingUTF8)
    dest = iio.CGImageDestinationCreateWithURL(url, png_type, 1, None)
    cf.CFRelease(url)
    cf.CFRelease(png_type)
    if not dest:
        return False
    iio.CGImageDestinationAddImage(dest, img, None)
    ok = iio.CGImageDestinationFinalize(dest)
    cf.CFRelease(dest)
    return ok


def draw_app_icon(out_png_path: str):
    size = 1024
    color_space = cg.CGColorSpaceCreateDeviceRGB()
    ctx = cg.CGBitmapContextCreate(
        None, size, size, 8, size * 4, color_space, 1
    )

    # 1. Base Squircle (x=100, y=100, w=824, h=824, r=185)
    base_rect = CGRect(CGPoint(100.0, 100.0), CGSize(824.0, 824.0))
    base_path = cg.CGPathCreateWithRoundedRect(base_rect, 185.0, 185.0, None)

    # Base Drop Shadow
    cg.CGContextSaveGState(ctx)
    shadow_col = make_color(color_space, 0.0, 0.0, 0.0, 0.55)
    cg.CGContextSetShadowWithColor(ctx, CGSize(0.0, -28.0), 38.0, shadow_col)
    cf.CFRelease(shadow_col)

    # Draw Squircle Gradient
    cg.CGContextAddPath(ctx, base_path)
    cg.CGContextClip(ctx)

    c1 = make_color(color_space, 0.16, 0.18, 0.22, 1.0)
    c2 = make_color(color_space, 0.09, 0.10, 0.13, 1.0)
    c3 = make_color(color_space, 0.04, 0.05, 0.07, 1.0)
    bg_grad = make_gradient(color_space, [c1, c2, c3], [0.0, 0.5, 1.0])
    cg.CGContextDrawLinearGradient(ctx, bg_grad, CGPoint(512.0, 924.0), CGPoint(512.0, 100.0), 0)
    cf.CFRelease(c1); cf.CFRelease(c2); cf.CFRelease(c3); cf.CFRelease(bg_grad)

    # Squircle Border Highlight
    cg.CGContextRestoreGState(ctx)
    cg.CGContextSaveGState(ctx)
    cg.CGContextAddPath(ctx, base_path)
    border_col = make_color(color_space, 1.0, 1.0, 1.0, 0.22)
    cg.CGContextSetStrokeColorWithColor(ctx, border_col)
    cg.CGContextSetLineWidth(ctx, 3.0)
    cg.CGContextStrokePath(ctx)
    cf.CFRelease(border_col)
    cg.CGContextRestoreGState(ctx)

    # 2. Outer Metal Lens Mount (r=300)
    cx, cy = 512.0, 512.0
    cg.CGContextSaveGState(ctx)
    cg.CGContextAddArc(ctx, cx, cy, 300.0, 0, 2 * math.pi, 0)
    m1 = make_color(color_space, 0.35, 0.38, 0.45, 1.0)
    m2 = make_color(color_space, 0.15, 0.17, 0.21, 1.0)
    metal_grad = make_gradient(color_space, [m1, m2], [0.0, 1.0])
    cg.CGContextClip(ctx)
    cg.CGContextDrawLinearGradient(ctx, metal_grad, CGPoint(212.0, 812.0), CGPoint(812.0, 212.0), 0)
    cf.CFRelease(m1); cf.CFRelease(m2); cf.CFRelease(metal_grad)
    cg.CGContextRestoreGState(ctx)

    # Stepped Black Ring (r=260)
    cg.CGContextSaveGState(ctx)
    cg.CGContextAddArc(ctx, cx, cy, 260.0, 0, 2 * math.pi, 0)
    b_col = make_color(color_space, 0.04, 0.05, 0.07, 1.0)
    cg.CGContextSetFillColorWithColor(ctx, b_col)
    cg.CGContextFillPath(ctx)
    cf.CFRelease(b_col)
    cg.CGContextRestoreGState(ctx)

    # 3. Deep Optical Glass Basin (r=230)
    cg.CGContextSaveGState(ctx)
    cg.CGContextAddArc(ctx, cx, cy, 230.0, 0, 2 * math.pi, 0)
    cg.CGContextClip(ctx)

    g1 = make_color(color_space, 0.06, 0.22, 0.33, 1.0)
    g2 = make_color(color_space, 0.03, 0.10, 0.18, 1.0)
    g3 = make_color(color_space, 0.01, 0.03, 0.06, 1.0)
    glass_grad = make_gradient(color_space, [g1, g2, g3], [0.0, 0.5, 1.0])
    cg.CGContextDrawRadialGradient(ctx, glass_grad, CGPoint(480.0, 550.0), 10.0, CGPoint(512.0, 512.0), 240.0, 0)
    cf.CFRelease(g1); cf.CFRelease(g2); cf.CFRelease(g3); cf.CFRelease(glass_grad)

    # Optical Coating Highlights (Cyan / Violet flare)
    f1 = make_color(color_space, 0.0, 0.95, 1.0, 0.6)
    f2 = make_color(color_space, 0.3, 0.67, 1.0, 0.2)
    f3 = make_color(color_space, 0.6, 0.32, 0.88, 0.3)
    f4 = make_color(color_space, 0.0, 0.0, 0.0, 0.0)
    flare_grad = make_gradient(color_space, [f1, f2, f3, f4], [0.0, 0.35, 0.7, 1.0])
    cg.CGContextDrawLinearGradient(ctx, flare_grad, CGPoint(320.0, 680.0), CGPoint(680.0, 340.0), 0)
    cf.CFRelease(f1); cf.CFRelease(f2); cf.CFRelease(f3); cf.CFRelease(f4); cf.CFRelease(flare_grad)
    cg.CGContextRestoreGState(ctx)

    # 4. Glowing GPS Orbital Track Wave
    cg.CGContextSaveGState(ctx)
    track_glow = make_color(color_space, 0.0, 0.95, 1.0, 0.8)
    cg.CGContextSetShadowWithColor(ctx, CGSize(0.0, 0.0), 22.0, track_glow)
    cf.CFRelease(track_glow)

    track_stroke = make_color(color_space, 0.1, 0.95, 0.8, 1.0)
    cg.CGContextSetStrokeColorWithColor(ctx, track_stroke)
    cg.CGContextSetLineWidth(ctx, 10.0)
    cg.CGContextMoveToPoint(ctx, 240.0, 360.0)
    cg.CGContextAddCurveToPoint(ctx, 360.0, 300.0, 440.0, 420.0, 520.0, 520.0)
    cg.CGContextAddCurveToPoint(ctx, 600.0, 620.0, 680.0, 700.0, 780.0, 680.0)
    cg.CGContextStrokePath(ctx)
    cf.CFRelease(track_stroke)

    # Orbital Waypoint Nodes
    nodes = [(320.0, 325.0), (430.0, 410.0), (610.0, 630.0), (720.0, 685.0)]
    for nx, ny in nodes:
        cg.CGContextAddArc(ctx, nx, ny, 8.0, 0, 2 * math.pi, 0)
        node_col = make_color(color_space, 0.2, 0.95, 0.6, 1.0)
        cg.CGContextSetFillColorWithColor(ctx, node_col)
        cg.CGContextFillPath(ctx)
        cf.CFRelease(node_col)
    cg.CGContextRestoreGState(ctx)

    # 5. Geolocation Pin Hero Badge (Center)
    cg.CGContextSaveGState(ctx)
    pin_shadow = make_color(color_space, 0.0, 0.0, 0.0, 0.5)
    cg.CGContextSetShadowWithColor(ctx, CGSize(0.0, -18.0), 26.0, pin_shadow)
    cf.CFRelease(pin_shadow)

    # Pin Body Path
    cg.CGContextMoveToPoint(ctx, 512.0, 390.0) # Tip pointing down
    cg.CGContextAddCurveToPoint(ctx, 480.0, 440.0, 420.0, 520.0, 420.0, 580.0)
    cg.CGContextAddArc(ctx, 512.0, 580.0, 92.0, math.pi, 0, 0)
    cg.CGContextAddCurveToPoint(ctx, 604.0, 520.0, 544.0, 440.0, 512.0, 390.0)
    cg.CGContextClosePath(ctx)

    p1 = make_color(color_space, 1.0, 0.32, 0.32, 1.0)
    p2 = make_color(color_space, 0.95, 0.08, 0.22, 1.0)
    pin_grad = make_gradient(color_space, [p1, p2], [0.0, 1.0])
    cg.CGContextClip(ctx)
    cg.CGContextDrawLinearGradient(ctx, pin_grad, CGPoint(512.0, 680.0), CGPoint(512.0, 390.0), 0)
    cf.CFRelease(p1); cf.CFRelease(p2); cf.CFRelease(pin_grad)
    cg.CGContextRestoreGState(ctx)

    # Pin Inner Aperture Core (White Ring + Dark Center + Cyan Pulse)
    cg.CGContextSaveGState(ctx)
    cg.CGContextAddArc(ctx, 512.0, 580.0, 42.0, 0, 2 * math.pi, 0)
    w_col = make_color(color_space, 1.0, 1.0, 1.0, 1.0)
    cg.CGContextSetFillColorWithColor(ctx, w_col)
    cg.CGContextFillPath(ctx)
    cf.CFRelease(w_col)

    cg.CGContextAddArc(ctx, 512.0, 580.0, 30.0, 0, 2 * math.pi, 0)
    d_col = make_color(color_space, 0.08, 0.10, 0.14, 1.0)
    cg.CGContextSetFillColorWithColor(ctx, d_col)
    cg.CGContextFillPath(ctx)
    cf.CFRelease(d_col)

    # Cyan Pulse Core
    core_glow = make_color(color_space, 0.0, 0.95, 1.0, 0.9)
    cg.CGContextSetShadowWithColor(ctx, CGSize(0.0, 0.0), 16.0, core_glow)
    cf.CFRelease(core_glow)
    cg.CGContextAddArc(ctx, 512.0, 580.0, 14.0, 0, 2 * math.pi, 0)
    c_col = make_color(color_space, 0.0, 0.95, 1.0, 1.0)
    cg.CGContextSetFillColorWithColor(ctx, c_col)
    cg.CGContextFillPath(ctx)
    cf.CFRelease(c_col)

    # White Pinpoint Center
    cg.CGContextAddArc(ctx, 512.0, 580.0, 6.0, 0, 2 * math.pi, 0)
    cg.CGContextSetFillColorWithColor(ctx, make_color(color_space, 1.0, 1.0, 1.0, 1.0))
    cg.CGContextFillPath(ctx)
    cg.CGContextRestoreGState(ctx)

    # Output Image
    res_img = cg.CGBitmapContextCreateImage(ctx)
    ok = save_image(res_img, out_png_path)
    cf.CFRelease(res_img)
    cf.CFRelease(base_path)
    cf.CFRelease(color_space)
    return ok


def create_icns(master_png: str, out_icns: str):
    iconset_dir = master_png.replace(".png", ".iconset")
    if os.path.exists(iconset_dir):
        shutil.rmtree(iconset_dir)
    os.makedirs(iconset_dir, exist_ok=True)

    sizes = [
        ("icon_16x16.png", 16),
        ("icon_16x16@2x.png", 32),
        ("icon_32x32.png", 32),
        ("icon_32x32@2x.png", 64),
        ("icon_128x128.png", 128),
        ("icon_128x128@2x.png", 256),
        ("icon_256x256.png", 256),
        ("icon_256x256@2x.png", 512),
        ("icon_512x512.png", 512),
        ("icon_512x512@2x.png", 1024),
    ]

    for name, s in sizes:
        dst = os.path.join(iconset_dir, name)
        subprocess.run(["sips", "-z", str(s), str(s), master_png, "--out", dst], check=True, capture_output=True)

    subprocess.run(["iconutil", "-c", "icns", iconset_dir, "-o", out_icns], check=True, capture_output=True)
    shutil.rmtree(iconset_dir, ignore_errors=True)
    return True


def main():
    base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    img_dir = os.path.join(base_dir, "img")
    master_png = os.path.join(img_dir, "AppIcon.png")
    out_icns = os.path.join(img_dir, "AppIcon.icns")
    res_icns = os.path.join(base_dir, "macos", "PhotoolsApp", "Resources", "AppIcon.icns")

    os.makedirs(os.path.dirname(res_icns), exist_ok=True)

    sys.stdout.write("🎨 [1/3] 渲染 1024x1024 Retina 高清主图标...\n")
    sys.stdout.flush()
    ok = draw_app_icon(master_png)
    if not ok:
        sys.stderr.write("绘制 App 图标失败\n")
        sys.exit(1)

    sys.stdout.write(f"✨ 成功生成: {master_png}\n")
    sys.stdout.write("📦 [2/3] 编译多分辨率 Apple Iconset 为 AppIcon.icns...\n")
    sys.stdout.flush()
    create_icns(master_png, out_icns)
    shutil.copyfile(out_icns, res_icns)
    sys.stdout.write(f"✨ 已生成 AppIcon.icns 并复制到: {res_icns}\n")
    sys.stdout.write("🎉 [3/3] macOS 原生图标包构建完毕！\n")
    sys.stdout.flush()


if __name__ == "__main__":
    main()
