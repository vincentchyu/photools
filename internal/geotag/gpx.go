package geotag

import (
	"fmt"
	"math"

	"github.com/qichengzx/coordtransform"
)

type coordSystem string

const (
	coordWGS84 coordSystem = "wgs84"
	coordGCJ02 coordSystem = "gcj02"
)

type gpxSourceConfig struct {
	Default coordSystemRule `json:"default"`
	Rules   []gpxSourceRule `json:"rules"`
}

type coordSystemRule string

type gpxSourceRule struct {
	Pattern     string          `json:"pattern"`
	CoordSystem coordSystemRule `json:"coord_system"`
}

type normalizedGPXFile struct {
	OriginalPath   string
	NormalizedPath string
	CoordSystem    coordSystem
	Converted      bool
}

func Gcj02ToWGS84(lat, lon float64) (float64, float64) {
	return coordtransform.GCJ02toWGS84(lon, lat)
}

func Wgs84ToGCJ02(lat, lon float64) (float64, float64) {
	return coordtransform.WGS84toGCJ02(lon, lat)
	/*dLat := transformLat(lon-105.0, lat-35.0)
	dLon := transformLon(lon-105.0, lat-35.0)
	radLat := lat / 180.0 * math.Pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	dLat = (dLat * 180.0) / ((aCoord * (1 - ee)) / (magic * sqrtMagic) * math.Pi)
	dLon = (dLon * 180.0) / (aCoord / sqrtMagic * math.Cos(radLat) * math.Pi)
	return lat + dLat, lon + dLon*/
}

// DecimalToDMS 将十进制度数转为 DMS 格式（符合地理坐标规范）
// lat: 纬度 (-90 ~ 90)
// lng: 经度 (-180 ~ 180)
// return: 如 23°08'46.7"N 113°16'18.4"E
func DecimalToDMS(lng, lat float64) string {
	// 处理纬度
	latDeg, latMin, latSec := decimalToDMS(lat)
	latHemi := "N"
	if lat < 0 {
		latHemi = "S"
	}

	// 处理经度
	lngDeg, lngMin, lngSec := decimalToDMS(lng)
	lngHemi := "E"
	if lng < 0 {
		lngHemi = "W"
	}

	// 格式化输出（保留1位小数，和你示例一致）
	return fmt.Sprintf(
		"%d°%02d'%04.1f\"%s %d°%02d'%04.1f\"%s",
		int(math.Abs(float64(latDeg))), latMin, latSec, latHemi,
		int(math.Abs(float64(lngDeg))), lngMin, lngSec, lngHemi,
	)
}

// decimalToDMS 内部工具：十进制度 → 度、分、秒
func decimalToDMS(deg float64) (int, int, float64) {
	absDeg := math.Abs(deg)
	degree := math.Floor(absDeg)
	minutesFull := (absDeg - degree) * 60
	minute := math.Floor(minutesFull)
	second := (minutesFull - minute) * 60

	return int(degree), int(minute), second
}
