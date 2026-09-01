cask "photools" do
  version "0.1.0"
  sha256 "98b33919d52201b7f2709de9ddc3a6d795b024a59da90f8a023e1c071467146f"

  url "https://github.com/vincentchyu/photools/releases/download/v0.1.0/photools-macOS.dmg"
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
