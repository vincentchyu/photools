cask "photools" do
  version "0.0.3"
  sha256 "d88688eb3aa7d9cd9bc71356b6bc9e06a5bf8d181ea886bbde1b8ee2f847616b"

  url "https://github.com/vincentchyu/photools/releases/download/v0.0.3/photools-macOS.dmg"
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
