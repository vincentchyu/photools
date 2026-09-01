#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# photools Homebrew Formula & Cask 自动生成与发布同步脚本
# ==============================================================================
# 用法:
#   ./script/release_homebrew.sh [版本号，如 v0.0.2] [--push]
#
# 行为:
#   1. 读取指定版本（默认读取最新 git tag 或 common.CurrentVersion）；
#   2. 计算本地预编译二进制文件 SHA256；
#   3. 计算 dist/photools-macOS.dmg 的 SHA256；
#   4. 渲染更新本地 Formula/photools.rb 与 Casks/photools.rb；
#   5. 若带有 --push 参数，自动克隆/同步推送至 github.com/vincentchyu/homebrew-tap。
# ==============================================================================

REPO="vincentchyu/photools"
TAP_REPO="vincentchyu/homebrew-tap"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

TAG=""
PUSH_TO_TAP=false

for arg in "$@"; do
    case "$arg" in
        --push)
            PUSH_TO_TAP=true
            ;;
        v*|V*|[0-9]*)
            TAG="$arg"
            ;;
        *)
            ;;
    esac
done

if [ -z "$TAG" ]; then
    # 尝试从 git tag 获取
    TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "")"
    if [ -z "$TAG" ]; then
        TAG="v0.0.2"
    fi
fi

# 确保 TAG 带有 'v' 前缀
if [[ ! "$TAG" =~ ^v ]]; then
    TAG="v${TAG}"
fi

VERSION="${TAG#v}"

echo "========================================================"
echo "📸 photools Homebrew Release Generator"
echo "========================================================"
echo "• 目标版本 Tag:   ${TAG}"
echo "• 纯版本号 Version: ${VERSION}"
echo "• 主仓库:         https://github.com/${REPO}"
echo "• Tap 仓库:       https://github.com/${TAP_REPO}"
echo "--------------------------------------------------------"

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT

resolve_sha256() {
    local name="$1"
    local local_file="${ROOT_DIR}/dist/${name}"
    local sha=""

    if [ -f "${local_file}" ]; then
        sha="$(shasum -a 256 "${local_file}" | awk '{print $1}')"
        echo "  ✅ 本地 ${name} SHA256: ${sha}" >&2
        echo "${sha}"
        return
    fi

    # 尝试从线上 GitHub Release 下载
    local online_url="https://github.com/${REPO}/releases/download/${TAG}/${name}"
    local temp_file="${TEMP_DIR}/${name}"
    if curl -sSL -f -o "${temp_file}" "${online_url}" 2>/dev/null; then
        sha="$(shasum -a 256 "${temp_file}" | awk '{print $1}')"
        echo "  ✅ 线上 Release ${name} SHA256: ${sha}" >&2
        echo "${sha}"
        return
    fi

    echo "  ⚠️ ${name} 本地与线上均未发现，使用全零占位符" >&2
    echo "0000000000000000000000000000000000000000000000000000000000000000"
}

echo "🔍 正在解析预编译二进制文件与 DMG 镜像 SHA256 ..."
MAC_ARM_SHA="$(resolve_sha256 "photools-darwin-arm64")"
MAC_AMD_SHA="$(resolve_sha256 "photools-darwin-amd64")"
LINUX_AMD_SHA="$(resolve_sha256 "photools-linux-amd64")"
DMG_SHA="$(resolve_sha256 "photools-macOS.dmg")"

# 1. 渲染 Formula/photools.rb
mkdir -p "${ROOT_DIR}/Formula"
cat <<INNEREOF > "${ROOT_DIR}/Formula/photools.rb"
class Photools < Formula
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/${REPO}"
  version "${VERSION}"
  license "MIT"

  depends_on "exiftool"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${REPO}/releases/download/v${VERSION}/photools-darwin-arm64"
      sha256 "${MAC_ARM_SHA}"
    else
      url "https://github.com/${REPO}/releases/download/v${VERSION}/photools-darwin-amd64"
      sha256 "${MAC_AMD_SHA}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/${REPO}/releases/download/v${VERSION}/photools-linux-amd64"
      sha256 "${LINUX_AMD_SHA}"
    end
  end

  def install
    if OS.mac? && Hardware::CPU.arm?
      bin.install "photools-darwin-arm64" => "photools"
    elsif OS.mac? && Hardware::CPU.intel?
      bin.install "photools-darwin-amd64" => "photools"
    elsif OS.linux? && Hardware::CPU.intel?
      bin.install "photools-linux-amd64" => "photools"
    end

    chmod 0555, bin/"photools"

    # 生成并安装 Zsh, Bash, Fish 全功能中文 Shell 自动补全脚本
    generate_completions_from_executable(bin/"photools", "completion")
  end

  def caveats
    <<~EOS
      📸 photools 已成功安装！

      提示：
      1. 核心底层依赖 exiftool 已由 Homebrew 自动装配就绪；
      2. Zsh/Bash/Fish 自动补全脚本已安装至系统补全目录；
      3. 在终端中直接运行 \\\`photools\\\` 或 \\\`photools tui\\\` 即可开启交互式工作台；
      4. 内置离线高精逆地理编码数据已随包分发。
    EOS
  end

  test do
    assert_match "photools", shell_output("#{bin}/photools --help")
    assert_match "v#{version}", shell_output("#{bin}/photools version")
  end
end
INNEREOF
echo "✅ 已生成: Formula/photools.rb"

# 2. 渲染 Casks/photools.rb
mkdir -p "${ROOT_DIR}/Casks"
cat <<INNEREOF > "${ROOT_DIR}/Casks/photools.rb"
cask "photools" do
  version "${VERSION}"
  sha256 "${DMG_SHA}"

  url "https://github.com/${REPO}/releases/download/v${VERSION}/photools-macOS.dmg"
  name "photools"
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/${REPO}"

  depends_on formula: "exiftool"

  app "PhotoolsApp.app"
  binary "#{appdir}/PhotoolsApp.app/Contents/MacOS/photools"

  zap trash: [
    "~/.config/photools",
    "~/.logs/photools",
  ]
end
INNEREOF
echo "✅ 已生成: Casks/photools.rb"

echo "--------------------------------------------------------"
echo "🎉 Formula 与 Cask 生成完毕！"

if [ "$PUSH_TO_TAP" = true ]; then
    echo "🚀 正在推送至远程 Tap 仓库: git@github.com:${TAP_REPO}.git ..."
    TAP_CLONE_DIR="${TEMP_DIR}/tap"
    git clone "git@github.com:${TAP_REPO}.git" "${TAP_CLONE_DIR}"
    mkdir -p "${TAP_CLONE_DIR}/Formula" "${TAP_CLONE_DIR}/Casks"
    cp "${ROOT_DIR}/Formula/photools.rb" "${TAP_CLONE_DIR}/Formula/photools.rb"
    cp "${ROOT_DIR}/Casks/photools.rb" "${TAP_CLONE_DIR}/Casks/photools.rb"
    cd "${TAP_CLONE_DIR}"
    git add Formula/ Casks/
    git commit -m "feat(release): update photools to ${TAG}" || true
    git push origin main || git push origin master
    echo "✨ 成功推送更新到 ${TAP_REPO}！"
else
    echo "💡 如需同步到远程 Tap 仓库，请执行:"
    echo "   1. git clone git@github.com:${TAP_REPO}.git /tmp/tap"
    echo "   2. cp -r Formula/ Casks/ /tmp/tap/"
    echo "   3. cd /tmp/tap && git commit -am 'feat: update photools to ${TAG}' && git push"
    echo "   或直接附带参数运行: ./script/release_homebrew.sh ${TAG} --push"
fi
echo "========================================================"
