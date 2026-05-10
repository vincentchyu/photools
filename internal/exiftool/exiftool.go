package exiftool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Metadata struct {
	DateTimeOriginal   string `json:"DateTimeOriginal"`
	OffsetTimeOriginal string `json:"OffsetTimeOriginal"`
	GPSPosition        string `json:"GPSPosition"`
	GPSDateTime        string `json:"GPSDateTime"`
}

type CommandRunner interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func ReadMetadata(runner CommandRunner, path string) (Metadata, error) {
	output, err := runner.CombinedOutput(
		"exiftool", "-json", "-DateTimeOriginal", "-OffsetTimeOriginal", "-GPSPosition", "-GPSDateTime", path,
	)
	if err != nil {
		return Metadata{}, fmt.Errorf("exiftool 读取元数据失败: %w", err)
	}
	return ParseMetadataJSON(output)
}

func ParseMetadataJSON(output []byte) (Metadata, error) {
	var records []Metadata
	if err := json.Unmarshal(output, &records); err != nil {
		return Metadata{}, fmt.Errorf("解析 exiftool 输出失败: %w", err)
	}
	if len(records) == 0 {
		return Metadata{}, errors.New("exiftool 未返回任何元数据")
	}
	return records[0], nil
}

func WriteGeotag(runner CommandRunner, rawPath string, gpxFiles []string, geosync string) ([]byte, error) {
	args := []string{
		"-overwrite_original",
		"-P",
		fmt.Sprintf("-geosync=%s", geosync),
	}
	for _, gpx := range gpxFiles {
		args = append(args, "-geotag", gpx)
	}
	args = append(args, rawPath)
	return runner.CombinedOutput("exiftool", args...)
}

func ClassifyFailure(output []byte, err error) string {
	text := strings.ToLower(string(bytes.TrimSpace(output)))
	switch {
	case strings.Contains(text, "no track points found"), strings.Contains(text, "gps track is empty"):
		return "照片时间未落在轨迹范围内，未写入有效 GPS 信息。"
	case strings.Contains(text, "empty track file"):
		return "GPX 轨迹文件为空或无法读取。"
	case strings.Contains(text, "file not found"):
		return "底层工具未找到待处理文件。"
	case err != nil:
		return fmt.Sprintf("调用 exiftool 失败：%v", err)
	default:
		return "未写入有效 GPS 信息。"
	}
}

func SyncGPS(runner CommandRunner, sourceRaw, targetPath string) error {
	_, err := runner.CombinedOutput(
		"exiftool", "-overwrite_original", "-P", "-TagsFromFile", sourceRaw, "-GPS:all", targetPath,
	)
	if err != nil {
		return fmt.Errorf("同步 GPS 失败：%w", err)
	}
	return nil
}

func SyncXMPGPS(runner CommandRunner, sourceRaw, targetXMP string) error {
	args := []string{
		"-overwrite_original",
		"-P",
		"-TagsFromFile", sourceRaw,
		"-XMP-exif:GPSVersionID<GPSVersionID",
		"-XMP-exif:GPSLatitude<GPSLatitude",
		"-XMP-exif:GPSLongitude<GPSLongitude",
		"-XMP-exif:GPSAltitudeRef<GPSAltitudeRef",
		"-XMP-exif:GPSAltitude<GPSAltitude",
		"-XMP-exif:GPSDateTime<GPSDateTime",
		"-XMP-exif:GPSSatellites<GPSSatellites",
		"-XMP-exif:GPSMapDatum<GPSMapDatum",
		targetXMP,
	}
	_, err := runner.CombinedOutput("exiftool", args...)
	if err != nil {
		return fmt.Errorf("同步 XMP GPS 失败：%w", err)
	}
	return nil
}
