class Photools < Formula
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/vincentchyu/photools"
  version "0.0.3"
  license "MIT"

  depends_on "exiftool"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-darwin-arm64"
      sha256 "9ddea6edde7261d94fb4b51532aa6cb766372cd3b89fe07f17494ccd55bf22ed"
    else
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-darwin-amd64"
      sha256 "5120ae4751de765722c81e3421fe21a7f5ae977b3b80e9580c203cc7d98bcfd0"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-linux-amd64"
      sha256 "2deea0ab515349fe2d7b687146fc2ff6d026b07245c647524b5ff35ac7bf8181"
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
      3. 在终端中直接运行 \`photools\` 或 \`photools tui\` 即可开启交互式工作台；
      4. 内置离线高精逆地理编码数据已随包分发。
    EOS
  end

  test do
    assert_match "photools", shell_output("#{bin}/photools --help")
    assert_match "v#{version}", shell_output("#{bin}/photools version")
  end
end
