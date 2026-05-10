package geotag

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vincentchyu/photo-processing/internal/organizer"
)

func TestParseExtensions(t *testing.T) {
	got := ParseExtensions(".NEF, jpg, nef, cr3 ,, ORF")
	want := []string{"nef", "jpg", "cr3", "orf"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseExtensions() = %#v, want %#v", got, want)
	}
}

func TestDiscoverAssets(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(Config{BaseDir: root, RawExtensions: ParseExtensions("nef,cr3")})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer app.closeLogger()

	raw := filepath.Join(app.cfg.InboxDir, "A001.NEF")
	jpg := filepath.Join(app.cfg.InboxDir, "A001.JPG")
	xmp := filepath.Join(app.cfg.InboxDir, "A001.xmp")
	wav := filepath.Join(app.cfg.InboxDir, "A001.wav")
	loneJPG := filepath.Join(app.cfg.InboxDir, "B001.JPG")
	if err := osWriteFile(raw); err != nil {
		t.Fatalf("write raw: %v", err)
	}
	if err := osWriteFile(jpg); err != nil {
		t.Fatalf("write jpg: %v", err)
	}
	if err := osWriteFile(xmp); err != nil {
		t.Fatalf("write xmp: %v", err)
	}
	if err := osWriteFile(wav); err != nil {
		t.Fatalf("write wav: %v", err)
	}
	if err := osWriteFile(loneJPG); err != nil {
		t.Fatalf("write lone jpg: %v", err)
	}

	assets, jpgOnly, err := app.discoverAssets()
	if err != nil {
		t.Fatalf("discoverAssets() error = %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("discoverAssets() assets len = %d, want 1", len(assets))
	}
	if assets[0].RawPath != raw || assets[0].JPGPath != jpg {
		t.Fatalf("discoverAssets() asset = %#v", assets[0])
	}
	if assets[0].XMPPath != xmp {
		t.Fatalf("discoverAssets() XMPPath = %q, want %q", assets[0].XMPPath, xmp)
	}
	wantCompanions := []string{wav, xmp}
	if !reflect.DeepEqual(assets[0].CompanionPaths, wantCompanions) {
		t.Fatalf("discoverAssets() CompanionPaths = %#v, want %#v", assets[0].CompanionPaths, wantCompanions)
	}
	if len(jpgOnly) != 1 || jpgOnly[0] != loneJPG {
		t.Fatalf("discoverAssets() jpgOnly = %#v", jpgOnly)
	}
}

func TestWritePendingReport(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(Config{BaseDir: root, RawExtensions: ParseExtensions("nef")})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer app.closeLogger()

	reportPath, err := app.writePendingReport(
		[]processingIssue{
			{
				Kind:       issueKindTrackGap,
				Reason:     "照片时间未落在轨迹范围内，未写入有效 GPS 信息。",
				Suggestion: "请补充覆盖该拍摄时间段的 GPX 轨迹文件。",
				Asset: organizer.PhotoAsset{
					BaseName:       "A001",
					RawPath:        "/tmp/A001.NEF",
					JPGPath:        "/tmp/A001.JPG",
					CompanionPaths: []string{"/tmp/A001.wav"},
				},
				PhotoTime:          "2025:10:06 07:34:40",
				PhotoOffset:        "+08:00",
				ReferencedGPXFiles: []string{"/tmp/track1.gpx"},
			},
		},
	)
	if err != nil {
		t.Fatalf("writePendingReport() error = %v", err)
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	for _, want := range []string{"A001", "GPX", "/tmp/A001.NEF", "/tmp/track1.gpx"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q in %q", want, text)
		}
	}
}

func osWriteFile(path string) error {
	return os.WriteFile(path, []byte("x"), 0o644)
}

func TestReadMetadata(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(Config{BaseDir: root, RawExtensions: ParseExtensions("nef,cr3")})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer app.closeLogger()
	t.Skip("Skipping TestReadMetadata as it depends on external files")
	metadata, err := app.readMetadata("/Users/vincent/Pictures/GPS/Processed/2025/1115/DSC_2025-11-15_1456.JPG")
	if err != nil {
		t.Fatalf("readMetadata() error = %v", err)
	}
	fmt.Println(fmt.Sprintf("metadata:%v", metadata))
}
