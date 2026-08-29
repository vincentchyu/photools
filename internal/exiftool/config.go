package exiftool

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	configPathOnce   sync.Once
	cachedConfigPath string
)

const photoolsConfigContent = `%Image::ExifTool::UserDefined = (
    'Image::ExifTool::XMP::Main' => {
        photools => {
            SubDirectory => {
                TagTable => 'Image::ExifTool::UserDefined::photools',
            },
        },
    },
);

%Image::ExifTool::UserDefined::photools = (
    GROUPS => { 0 => 'XMP', 1 => 'XMP-photools', 2 => 'Image' },
    NAMESPACE => { 'photools' => 'http://ns.photools.app/1.0/' },
    WRITABLE => 'string',
    GPSSource => { },
    GPSMatchMethod => { },
    InterpolateWindow => { },
    Processor => { },
    ProcessedDate => { },
);
1;
`

// EnsureConfigFile 确保本地存在 photools 专属 ExifTool 配置文件，并返回其绝对路径
func EnsureConfigFile() string {
	configPathOnce.Do(func() {
		home, err := os.UserHomeDir()
		var targetDir string
		if err == nil && home != "" {
			targetDir = filepath.Join(home, ".config", "photools")
		} else {
			targetDir = os.TempDir()
		}
		_ = os.MkdirAll(targetDir, 0o755)
		cfgPath := filepath.Join(targetDir, "exiftool.config")
		_ = os.WriteFile(cfgPath, []byte(photoolsConfigContent), 0o644)
		cachedConfigPath = cfgPath
	})
	return cachedConfigPath
}
