class Photools < Formula
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/vincentchyu/photools"
  version "0.2.0"
  license "MIT"

  depends_on "exiftool"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/vincentchyu/photools/releases/download/v0.2.0/photools-darwin-arm64"
      sha256 "37730d3a7223997a388328f8706cd90a09c1edf2edf90ec7e05847259d3906e2"
    else
      url "https://github.com/vincentchyu/photools/releases/download/v0.2.0/photools-darwin-amd64"
      sha256 "c5d90d7ae9bced2eb7ff6bfa5cd8dd9146add0cf7ffb603433a8f676f67d4eba"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/vincentchyu/photools/releases/download/v0.2.0/photools-linux-amd64"
      sha256 "ca22f57fe350e5a3b1eb362d621b1d3fdb61386b6aa40973c6880dc325930dab"
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
