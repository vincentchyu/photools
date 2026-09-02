cask "photools" do
  version "0.1.0"
  sha256 "966be8c742ead28d155b92ca70462ba11c8f980c641d5da827374b161be90263"

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
