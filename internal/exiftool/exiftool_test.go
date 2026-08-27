package exiftool

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/vincentchyu/photools/internal/domain"
)

// mockCommandRunner 用于记录调用并模拟返回
type mockCommandRunner struct {
	lastCommand string
	lastArgs    []string
	output      []byte
	err         error
}

func (m *mockCommandRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	m.lastCommand = name
	m.lastArgs = args
	return m.output, m.err
}

func TestReadMetadata(t *testing.T) {
	t.Run("成功读取元数据", func(t *testing.T) {
		jsonOutput := []byte(`[{
			"DateTimeOriginal": "2026:08:23 15:30:00",
			"OffsetTimeOriginal": "+08:00",
			"GPSPosition": "39 deg 54' 15.00\" N, 116 deg 23' 30.00\" E",
			"GPSDateTime": "2026:08:23 07:30:00Z"
		}]`)
		runner := &mockCommandRunner{output: jsonOutput}

		meta, err := ReadMetadata(runner, "/path/to/photo.jpg")
		if err != nil {
			t.Fatalf("ReadMetadata 期望成功，得到错误: %v", err)
		}

		if meta.DateTimeOriginal != "2026:08:23 15:30:00" {
			t.Errorf("DateTimeOriginal 不匹配: %s", meta.DateTimeOriginal)
		}
		if meta.OffsetTimeOriginal != "+08:00" {
			t.Errorf("OffsetTimeOriginal 不匹配: %s", meta.OffsetTimeOriginal)
		}
		if meta.GPSPosition != `39 deg 54' 15.00" N, 116 deg 23' 30.00" E` {
			t.Errorf("GPSPosition 不匹配: %s", meta.GPSPosition)
		}
	})

	t.Run("ExifTool 执行失败报错", func(t *testing.T) {
		runner := &mockCommandRunner{err: errors.New("command not found")}
		_, err := ReadMetadata(runner, "/path/to/photo.jpg")
		if err == nil {
			t.Fatal("期望返回错误，但返回 nil")
		}
	})
}

func TestReadBatchMetadataMap(t *testing.T) {
	t.Run("空切片直接返回空 map", func(t *testing.T) {
		runner := &mockCommandRunner{}
		res, err := ReadBatchMetadataMap(runner, nil)
		if err != nil {
			t.Fatalf("ReadBatchMetadataMap(nil) 出错: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("期望空 map，实际长度 %d", len(res))
		}
	})

	t.Run("批量成功读取与路径匹配", func(t *testing.T) {
		jsonOutput := []byte(`[
			{
				"SourceFile": "/path/to/photo1.jpg",
				"DateTimeOriginal": "2026:08:23 15:30:00",
				"GPSPosition": "39 deg 54' 15.00\" N, 116 deg 23' 30.00\" E"
			},
			{
				"SourceFile": "/path/to/photo2.NEF",
				"DateTimeOriginal": "2026:08:23 15:31:00",
				"GPSPosition": "39 deg 54' 16.00\" N, 116 deg 23' 31.00\" E"
			}
		]`)
		runner := &mockCommandRunner{output: jsonOutput}
		paths := []string{"/path/to/photo1.jpg", "/path/to/photo2.NEF"}

		res, err := ReadBatchMetadataMap(runner, paths)
		if err != nil {
			t.Fatalf("ReadBatchMetadataMap 报错: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("期望返回 2 条记录，实际返回 %d", len(res))
		}
		if res["/path/to/photo1.jpg"].DateTimeOriginal != "2026:08:23 15:30:00" {
			t.Errorf("photo1 时间不匹配: %s", res["/path/to/photo1.jpg"].DateTimeOriginal)
		}
		if res["/path/to/photo2.NEF"].DateTimeOriginal != "2026:08:23 15:31:00" {
			t.Errorf("photo2 时间不匹配: %s", res["/path/to/photo2.NEF"].DateTimeOriginal)
		}
	})
}

func TestParseBatchMetadataJSON(t *testing.T) {
	t.Run("空切片返回空切片", func(t *testing.T) {
		res, err := ParseBatchMetadataJSON([]byte(`[]`))
		if err != nil {
			t.Fatalf("期望正常解析空切片，实际报错: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("期望长度 0，实际长度 %d", len(res))
		}
	})

	t.Run("包含 Warning 干扰前缀时自动剥离成功解析", func(t *testing.T) {
		warningOutput := []byte("Warning: [minor] Fixed incorrect URI for schema\n" +
			"Warning: [minor] Non-standard format string\n" +
			`[{"SourceFile":"/path/to/img.nef","DateTimeOriginal":"2026:08:24 10:00:00"}]`)
		res, err := ParseBatchMetadataJSON(warningOutput)
		if err != nil {
			t.Fatalf("带 Warning 前缀应成功解析，实际报错: %v", err)
		}
		if len(res) != 1 || res[0].DateTimeOriginal != "2026:08:24 10:00:00" {
			t.Errorf("解析结果不匹配: %+v", res)
		}
	})

	t.Run("非法 JSON 报错", func(t *testing.T) {
		_, err := ParseBatchMetadataJSON([]byte(`invalid`))
		if err == nil {
			t.Fatal("期望报错，实际未报错")
		}
	})
}

func TestParseMetadataJSON(t *testing.T) {
	t.Run("空切片输出报错", func(t *testing.T) {
		_, err := ParseMetadataJSON([]byte(`[]`))
		if err == nil {
			t.Fatal("期望空数组报错，实际未报错")
		}
	})

	t.Run("非法 JSON 报错", func(t *testing.T) {
		_, err := ParseMetadataJSON([]byte(`invalid-json`))
		if err == nil {
			t.Fatal("期望非法 JSON 报错，实际未报错")
		}
	})
}

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		err      error
		expected string
	}{
		{
			name:     "无轨迹点命中",
			output:   "Warning: No track points found within time range",
			err:      errors.New("exit status 1"),
			expected: "照片时间未落在轨迹范围内，未写入有效 GPS 信息。",
		},
		{
			name:     "轨迹为空",
			output:   "Error: GPS track is empty",
			err:      errors.New("exit status 1"),
			expected: "照片时间未落在轨迹范围内，未写入有效 GPS 信息。",
		},
		{
			name:     "GPX 轨迹文件损坏或空",
			output:   "Error: empty track file /path/to/track.gpx",
			err:      errors.New("exit status 1"),
			expected: "GPX 轨迹文件为空或无法读取。",
		},
		{
			name:     "文件不存在",
			output:   "File not found: /path/to/missing.nef",
			err:      errors.New("exit status 1"),
			expected: "底层工具未找到待处理文件。",
		},
		{
			name:     "其他 exiftool 错误",
			output:   "unknown internal exif error",
			err:      errors.New("exit status 2"),
			expected: "调用 exiftool 失败：exit status 2",
		},
		{
			name:     "无错误但无有效输出",
			output:   "",
			err:      nil,
			expected: "未写入有效 GPS 信息。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyFailure([]byte(tt.output), tt.err)
			if got != tt.expected {
				t.Errorf("ClassifyFailure() = %q; 期望 %q", got, tt.expected)
			}
		})
	}
}

func TestWriteGeotag(t *testing.T) {
	runner := &mockCommandRunner{}
	rawPath := "/path/to/RAW.NEF"
	gpxFiles := []string{"/tracks/day1.gpx", "/tracks/day2.gpx"}
	geosync := "+00:00:05"

	_, err := WriteGeotag(runner, rawPath, gpxFiles, geosync)
	if err != nil {
		t.Fatalf("WriteGeotag 出错: %v", err)
	}

	argsStr := strings.Join(runner.lastArgs, " ")
	if !strings.Contains(argsStr, "-overwrite_original") {
		t.Errorf("缺少 -overwrite_original 参数")
	}
	if !strings.Contains(argsStr, "-geosync=+00:00:05") {
		t.Errorf("缺少 -geosync 参数: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-geotag /tracks/day1.gpx") || !strings.Contains(argsStr, "-geotag /tracks/day2.gpx") {
		t.Errorf("缺少 GPX 文件参数: %s", argsStr)
	}
	if !strings.HasSuffix(argsStr, rawPath) {
		t.Errorf("目标 RAW 文件应作为末尾参数: %s", argsStr)
	}
}

func TestSyncGPSToJPG(t *testing.T) {
	runner := &mockCommandRunner{}
	sourceRaw := "/path/to/photo.NEF"
	targetJPG := "/path/to/photo.JPG"

	err := SyncGPSToJPG(runner, sourceRaw, targetJPG)
	if err != nil {
		t.Fatalf("SyncGPSToJPG 出错: %v", err)
	}

	argsStr := strings.Join(runner.lastArgs, " ")
	if !strings.Contains(argsStr, "-TagsFromFile /path/to/photo.NEF") {
		t.Errorf("缺少 -TagsFromFile 参数: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-GPS:all") {
		t.Errorf("缺少 -GPS:all 标签同步参数: %s", argsStr)
	}
	if strings.Contains(argsStr, "-IPTC:all") || strings.Contains(argsStr, "photoshop") {
		t.Errorf("SyncGPSToJPG 不应包含地名元数据标签: %s", argsStr)
	}
}

func TestSyncGPSToXMP(t *testing.T) {
	runner := &mockCommandRunner{}
	sourceRaw := "/path/to/photo.NEF"
	targetXMP := "/path/to/photo.xmp"

	err := SyncGPSToXMP(runner, sourceRaw, targetXMP)
	if err != nil {
		t.Fatalf("SyncGPSToXMP 出错: %v", err)
	}

	argsStr := strings.Join(runner.lastArgs, " ")
	if !strings.Contains(argsStr, "-XMP-exif:GPSLatitude<GPSLatitude") {
		t.Errorf("缺少 XMP GPSLatitude 映射: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-XMP-exif:GPSLongitude<GPSLongitude") {
		t.Errorf("缺少 XMP GPSLongitude 映射: %s", argsStr)
	}
	if strings.Contains(argsStr, "Country") || strings.Contains(argsStr, "City") {
		t.Errorf("SyncGPSToXMP 不应包含地名标签: %s", argsStr)
	}
}

func TestWriteLocation(t *testing.T) {
	t.Run("空地理信息安全跳过", func(t *testing.T) {
		runner := &mockCommandRunner{}
		err := WriteLocation(runner, "/path/to/photo.JPG", domain.LocationInfo{})
		if err != nil {
			t.Fatalf("空地名信息应安全返回 nil: %v", err)
		}
		if runner.lastCommand != "" {
			t.Errorf("空地名信息不应调用 exiftool 执行命令")
		}
	})

	t.Run("完整写入国家省份城市与区县子地点", func(t *testing.T) {
		runner := &mockCommandRunner{}
		loc := domain.LocationInfo{
			Country:     "中国",
			CountryCode: "CN",
			Province:    "山西省",
			City:        "忻州市",
			District:    "大南庄",
		}
		err := WriteLocation(runner, "/path/to/photo.JPG", loc)
		if err != nil {
			t.Fatalf("WriteLocation 出错: %v", err)
		}

		argsStr := strings.Join(runner.lastArgs, " ")
		if !strings.Contains(argsStr, "-charset UTF8") || !strings.Contains(argsStr, "-codedcharacterset=utf8") {
			t.Errorf("缺少 UTF-8 编码或 codedcharacterset 声明: %s", argsStr)
		}
		if !strings.Contains(argsStr, "-XMP-photoshop:Country=中国") || !strings.Contains(argsStr, "-IPTC:Country-PrimaryLocationName=中国") {
			t.Errorf("国家字段写入不完整: %s", argsStr)
		}
		if !strings.Contains(argsStr, "-XMP-iptcCore:CountryCode=CN") || !strings.Contains(argsStr, "-IPTC:Country-PrimaryLocationCode=CN") {
			t.Errorf("国家代码写入不完整: %s", argsStr)
		}
		if !strings.Contains(argsStr, "-XMP-photoshop:State=山西省") {
			t.Errorf("省份字段写入不完整: %s", argsStr)
		}
		if !strings.Contains(argsStr, "-XMP-photoshop:City=忻州市") {
			t.Errorf("城市字段写入不完整: %s", argsStr)
		}
		if !strings.Contains(argsStr, "-XMP-iptcCore:Location=大南庄") || !strings.Contains(argsStr, "-IPTC:Sub-location=大南庄") {
			t.Errorf("子位置/区县字段写入不完整: %s", argsStr)
		}
	})
}

func TestSyncLocationToJPG(t *testing.T) {
	runner := &mockCommandRunner{}
	sourceRaw := "/path/to/photo.NEF"
	targetJPG := "/path/to/photo.JPG"

	err := SyncLocationToJPG(runner, sourceRaw, targetJPG)
	if err != nil {
		t.Fatalf("SyncLocationToJPG 出错: %v", err)
	}

	argsStr := strings.Join(runner.lastArgs, " ")
	if !strings.Contains(argsStr, "-charset UTF8") || !strings.Contains(argsStr, "-codedcharacterset=utf8") {
		t.Errorf("缺少 UTF-8 编码或 codedcharacterset 声明: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-IPTC:all") || !strings.Contains(argsStr, "-XMP-photoshop:all") || !strings.Contains(argsStr, "-XMP-iptcCore:all") || !strings.Contains(argsStr, "-XMP-iptcExt:all") {
		t.Errorf("SyncLocationToJPG 缺少地名元数据标签: %s", argsStr)
	}
	if strings.Contains(argsStr, "-GPS:all") {
		t.Errorf("SyncLocationToJPG 不应混入 -GPS:all 标签: %s", argsStr)
	}
}

func TestSyncLocationToXMP(t *testing.T) {
	runner := &mockCommandRunner{}
	sourceRaw := "/path/to/photo.NEF"
	targetXMP := "/path/to/photo.xmp"

	err := SyncLocationToXMP(runner, sourceRaw, targetXMP)
	if err != nil {
		t.Fatalf("SyncLocationToXMP 出错: %v", err)
	}

	argsStr := strings.Join(runner.lastArgs, " ")
	if !strings.Contains(argsStr, "-XMP-photoshop:Country<XMP-photoshop:Country") {
		t.Errorf("缺少 XMP Country 映射: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-XMP-photoshop:State<XMP-photoshop:State") {
		t.Errorf("缺少 XMP State 映射: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-XMP-photoshop:City<XMP-photoshop:City") {
		t.Errorf("缺少 XMP City 映射: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-XMP-iptcCore:Location<XMP-iptcCore:Location") {
		t.Errorf("缺少 XMP Location 映射: %s", argsStr)
	}
	if strings.Contains(argsStr, "GPSLatitude") {
		t.Errorf("SyncLocationToXMP 不应混入 GPSLatitude 标签: %s", argsStr)
	}
}

func TestParseCoordinates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedLat float64
		expectedLon float64
		shouldErr   bool
	}{
		{
			name:        "标准 DMS 格式 (北纬东经)",
			input:       `39 deg 54' 15.00" N, 116 deg 23' 30.00" E`,
			expectedLat: 39.90416666,
			expectedLon: 116.39166666,
			shouldErr:   false,
		},
		{
			name:        "南纬西经 DMS 格式",
			input:       `33 deg 52' 07.68" S, 151 deg 12' 33.48" W`,
			expectedLat: -33.8688,
			expectedLon: -151.2093,
			shouldErr:   false,
		},
		{
			name:        "十进制加方向 39.9042 N, 116.3917 E",
			input:       "39.9042 N, 116.3917 E",
			expectedLat: 39.9042,
			expectedLon: 116.3917,
			shouldErr:   false,
		},
		{
			name:        "十进制加方向南纬西经 45.1234 S, 75.5678 W",
			input:       "45.1234 S, 75.5678 W",
			expectedLat: -45.1234,
			expectedLon: -75.5678,
			shouldErr:   false,
		},
		{
			name:        "逗号分隔浮点数 39.9042, 116.3917",
			input:       "39.9042, 116.3917",
			expectedLat: 39.9042,
			expectedLon: 116.3917,
			shouldErr:   false,
		},
		{
			name:        "空格分隔纯浮点数 -33.8688 151.2093",
			input:       "-33.8688 151.2093",
			expectedLat: -33.8688,
			expectedLon: 151.2093,
			shouldErr:   false,
		},
		{
			name:      "空字符串",
			input:     "",
			shouldErr: true,
		},
		{
			name:      "非法文本",
			input:     "not-a-coordinate",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon, err := ParseCoordinates(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Fatalf("ParseCoordinates(%q) 期望报错，实际未报错", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCoordinates(%q) 出错: %v", tt.input, err)
			}
			if math.Abs(lat-tt.expectedLat) > 0.001 {
				t.Errorf("Lat = %f; 期望 %f", lat, tt.expectedLat)
			}
			if math.Abs(lon-tt.expectedLon) > 0.001 {
				t.Errorf("Lon = %f; 期望 %f", lon, tt.expectedLon)
			}
		})
	}
}

func TestInspectPhotoMetadata(t *testing.T) {
	t.Run("完整解析相机与曝光参数与GPS", func(t *testing.T) {
		jsonOutput := []byte(`[{
			"SourceFile": "/photos/DSC_8021.NEF",
			"File:FileSize": "45.2 MB",
			"File:FileModifyDate": "2026:08:25 10:00:00+08:00",
			"EXIF:Make": "NIKON CORPORATION",
			"EXIF:Model": "NIKON Z 8",
			"Composite:LensSpec": "24-70mm f/2.8",
			"EXIF:LensModel": "NIKKOR Z 24-70mm f/2.8 S",
			"EXIF:DateTimeOriginal": "2026:08:25 09:30:15",
			"EXIF:ExposureTime": "1/250",
			"EXIF:FNumber": 2.8,
			"EXIF:ISO": 100,
			"EXIF:FocalLength": 35.0,
			"Composite:GPSLatitude": 31.2304,
			"Composite:GPSLongitude": 121.4737,
			"Composite:GPSAltitude": 12.5,
			"XMP:Country": "中国",
			"XMP:State": "上海市",
			"XMP:City": "上海市",
			"XMP:Location": "外滩"
		}]`)

		runner := &mockCommandRunner{output: jsonOutput}
		meta, err := InspectPhotoMetadata(runner, "/photos/DSC_8021.NEF")
		if err != nil {
			t.Fatalf("InspectPhotoMetadata 报错: %v", err)
		}

		if meta.CameraMake != "NIKON CORPORATION" {
			t.Errorf("CameraMake 不匹配: %s", meta.CameraMake)
		}
		if meta.CameraModel != "NIKON Z 8" {
			t.Errorf("CameraModel 不匹配: %s", meta.CameraModel)
		}
		if meta.LensModel != "NIKKOR Z 24-70mm f/2.8 S" {
			t.Errorf("LensModel 不匹配: %s", meta.LensModel)
		}
		if meta.ExposureTime != "1/250" {
			t.Errorf("ExposureTime 不匹配: %s", meta.ExposureTime)
		}
		if meta.FNumber != "2.8" {
			t.Errorf("FNumber 不匹配: %s", meta.FNumber)
		}
		if meta.ISO != "100" {
			t.Errorf("ISO 不匹配: %s", meta.ISO)
		}
		if meta.FocalLength != "35" {
			t.Errorf("FocalLength 不匹配: %s", meta.FocalLength)
		}
		if meta.Latitude == nil || *meta.Latitude != 31.2304 {
			t.Errorf("Latitude 不匹配: %v", meta.Latitude)
		}
		if meta.Longitude == nil || *meta.Longitude != 121.4737 {
			t.Errorf("Longitude 不匹配: %v", meta.Longitude)
		}
		if meta.Country != "中国" || meta.District != "外滩" {
			t.Errorf("Location 不匹配: %s %s", meta.Country, meta.District)
		}
		if len(meta.RawTags) == 0 {
			t.Errorf("RawTags 期望非空")
		}
	})
}
