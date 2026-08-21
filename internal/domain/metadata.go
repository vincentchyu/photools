package domain

// Metadata 封装由 ExifTool 提取的拍摄元数据
type Metadata struct {
	DateTimeOriginal   string `json:"DateTimeOriginal"`
	OffsetTimeOriginal string `json:"OffsetTimeOriginal"`
	GPSPosition        string `json:"GPSPosition"`
	GPSDateTime        string `json:"GPSDateTime"`
}

// HasGPS 判断是否存在有效 GPS 坐标
func (m Metadata) HasGPS() bool {
	return m.GPSPosition != ""
}
