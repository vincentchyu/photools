package geotag

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/qichengzx/coordtransform"
)

// Gcj02ToWGS84 坐标转换
func Gcj02ToWGS84(lat, lon float64) (float64, float64) {
	return coordtransform.GCJ02toWGS84(lon, lat)
}

// Wgs84ToGCJ02 坐标转换
func Wgs84ToGCJ02(lat, lon float64) (float64, float64) {
	return coordtransform.WGS84toGCJ02(lon, lat)
}

// DecimalToDMS 将十进制度数转为 DMS 格式
func DecimalToDMS(lng, lat float64) string {
	latDeg, latMin, latSec := decimalToDMS(lat)
	latHemi := "N"
	if lat < 0 {
		latHemi = "S"
	}

	lngDeg, lngMin, lngSec := decimalToDMS(lng)
	lngHemi := "E"
	if lng < 0 {
		lngHemi = "W"
	}

	return fmt.Sprintf(
		"%d°%02d'%04.1f\"%s %d°%02d'%04.1f\"%s",
		int(math.Abs(float64(latDeg))), latMin, latSec, latHemi,
		int(math.Abs(float64(lngDeg))), lngMin, lngSec, lngHemi,
	)
}

func decimalToDMS(deg float64) (int, int, float64) {
	absDeg := math.Abs(deg)
	degree := math.Floor(absDeg)
	minutesFull := (absDeg - degree) * 60
	minute := math.Floor(minutesFull)
	second := (minutesFull - minute) * 60

	return int(degree), int(minute), second
}

// ListGPXFiles 扫描指定目录下的全部 .gpx 文件
func ListGPXFiles(gpxDir string) ([]string, error) {
	var entries []string
	err := filepath.WalkDir(
		gpxDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".gpx") {
				entries = append(entries, path)
			}
			return nil
		},
	)
	return entries, err
}
