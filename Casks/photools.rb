cask "photools" do
  version "0.2.0"
  sha256 "41477bb5097c60b295ad1c207bef87c511a4b0ddfadb9cf3654e78c86a5e356a"

  url "https://github.com/vincentchyu/photools/releases/download/v0.2.0/photools-macOS.dmg"
  name "photools"
  desc "Turn GPX tracks into GPS-tagged, location-aware photo libraries with offline geocoding"
  homepage "https://github.com/vincentchyu/photools"

  depends_on formula: "exiftool"

  app "PhotoolsApp.app"
  binary "#{appdir}/PhotoolsApp.app/Contents/MacOS/photools"

  zap trash: [
    "~/.config/photools",
    "~/.logs/photools",
  ]
end
