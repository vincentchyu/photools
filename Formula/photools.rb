class Photools < Formula
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/vincentchyu/photools"
  version "0.0.3"
  license "MIT"

  depends_on "exiftool"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-darwin-arm64"
      sha256 "e06edd5e782f2847dac164e6170d10dac629aba22d65cef49b9695a8c06b5d0b"
    else
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-darwin-amd64"
      sha256 "593fd82a7952e42ab41239135e794356a2f1589ea49ee27d9e47772601c643a5"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-linux-amd64"
      sha256 "7e12c2641756d672edda36c534ec121c59bb00e78dbb605c722ccdf931157d11"
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
