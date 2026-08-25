package exiftool

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	cachedPath string
	locateOnce sync.Once
)

// LocateExifTool 自动定位 ExifTool 可执行文件路径
// 优先级：
// 1. PHOTOOLS_EXIFTOOL 环境变量指定的路径
// 2. 当前运行可执行文件所在 App Bundle 内置路径 (../Resources/vendor/exiftool/exiftool 或 ../Resources/exiftool)
// 3. 系统 PATH 中的 exiftool
// 4. /opt/homebrew/bin/exiftool (macOS Apple Silicon 默认)
// 5. /usr/local/bin/exiftool (macOS Intel 默认)
func LocateExifTool() string {
	locateOnce.Do(func() {
		// 1. 环境变量
		if envPath := os.Getenv("PHOTOOLS_EXIFTOOL"); envPath != "" {
			if isExecutable(envPath) {
				cachedPath = envPath
				return
			}
		}

		// 2. App Bundle 内置资源探测
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			// Contents/MacOS/ -> Contents/Resources/vendor/exiftool/exiftool
			candidate1 := filepath.Join(exeDir, "..", "Resources", "vendor", "exiftool", "exiftool")
			if isExecutable(candidate1) {
				cachedPath = candidate1
				return
			}
			candidate2 := filepath.Join(exeDir, "..", "Resources", "exiftool")
			if isExecutable(candidate2) {
				cachedPath = candidate2
				return
			}
		}

		// 3. 系统 PATH 检索
		if sysPath, err := exec.LookPath("exiftool"); err == nil {
			cachedPath = sysPath
			return
		}

		// 4. macOS 常见固定路径
		candidates := []string{
			"/opt/homebrew/bin/exiftool",
			"/usr/local/bin/exiftool",
			"/usr/bin/exiftool",
		}
		for _, c := range candidates {
			if isExecutable(c) {
				cachedPath = c
				return
			}
		}

		// 降级使用默认命令名
		cachedPath = "exiftool"
	})

	return cachedPath
}

// ResetExifToolPath 重置缓存的 ExifTool 路径（用于测试）
func ResetExifToolPath() {
	cachedPath = ""
	locateOnce = sync.Once{}
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0111 != 0
}
