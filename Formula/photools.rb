class Photools < Formula
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/vincentchyu/photools"
  version "0.1.0"
  license "MIT"

  depends_on "exiftool"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/vincentchyu/photools/releases/download/v0.1.0/photools-darwin-arm64"
      sha256 "68581d17f6e3a2791e0a7f2d9e5730b7bc2e6983c258d4a4101b5a5e40c54168"
    else
      url "https://github.com/vincentchyu/photools/releases/download/v0.1.0/photools-darwin-amd64"
      sha256 "0ea2ebf7a59c12e1a3ec70d612058c6b4e9f374758505958ca406d004fb03e5a"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vincentchyu/photools/releases/download/v0.1.0/photools-linux-amd64"
      sha256 "58d5873f2c9b21d431ad7efbc9ab0db209b7abb7357b799bdc7862fd1e97c497"
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
