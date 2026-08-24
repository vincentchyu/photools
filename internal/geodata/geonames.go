package geodata

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vincentchyu/photo-processing/internal/geocoding"
)

const (
	GeoNamesChinaURL       = "https://download.geonames.org/export/dump/CN.zip"
	GeoNamesCities15000URL = "https://download.geonames.org/export/dump/cities15000.zip"
	GeoNamesAdmin1CodesURL = "https://download.geonames.org/export/dump/admin1CodesASCII.txt"
	GeoNamesAdmin2CodesURL = "https://download.geonames.org/export/dump/admin2Codes.txt"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DownloadWithCache 针对大文件下载实现本地缓存与断点续传（优先使用本地已下载的缓存文件）
func DownloadWithCache(ctx context.Context, rawURL string, logFn func(string)) ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户主目录: %w", err)
	}

	cacheDir := filepath.Join(home, ".config", "photools", "geonames")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 GeoNames 缓存目录失败: %w", err)
	}

	fileName := filepath.Base(rawURL)
	cachedPath := filepath.Join(cacheDir, fileName)

	// 如果本地已有完整缓存，直接读取
	if fi, err := os.Stat(cachedPath); err == nil && fi.Size() > 0 {
		if logFn != nil {
			logFn(fmt.Sprintf("使用本地缓存文件: %s (%s)", cachedPath, formatBytes(fi.Size())))
		}
		return os.ReadFile(cachedPath)
	}

	if logFn != nil {
		logFn(fmt.Sprintf("正在从 GeoNames 官方开放源下载: %s ...", rawURL))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "PhotoTools/1.0 (Photography GPS & Geocoding Suite)")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 GeoNames 数据失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败，HTTP 状态码: %d %s", resp.StatusCode, resp.Status)
	}

	var buf bytes.Buffer
	var written int64
	tmpReader := &progressReader{
		Reader: resp.Body,
		OnProgress: func(read int64) {
			written += read
		},
	}

	_, err = io.Copy(&buf, tmpReader)
	if err != nil {
		return nil, fmt.Errorf("读取网络流失败: %w", err)
	}

	data := buf.Bytes()
	// 写入本地持久缓存
	_ = os.WriteFile(cachedPath, data, 0644)

	if logFn != nil {
		logFn(fmt.Sprintf("下载完成 (%s)，已缓存至: %s", formatBytes(int64(len(data))), cachedPath))
	}

	return data, nil
}

type progressReader struct {
	io.Reader
	OnProgress func(read int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 && pr.OnProgress != nil {
		pr.OnProgress(int64(n))
	}
	return n, err
}

// DownloadAndCacheAdminCodes 下载并持久化 GeoNames 官方原始映射文件 (admin1CodesASCII.txt, admin2Codes.txt) 到 geodata 目录
func DownloadAndCacheAdminCodes(ctx context.Context, destDir string, logFn func(string)) error {
	if destDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		destDir = filepath.Join(home, ".config", "photools", "geodata")
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// 1. 下载并保存 admin1CodesASCII.txt
	admin1Path := filepath.Join(destDir, "admin1CodesASCII.txt")
	if fi, err := os.Stat(admin1Path); err != nil || fi.Size() == 0 {
		if logFn != nil {
			logFn("正在从 GeoNames 同步省级行政区划源映射表 (admin1CodesASCII.txt)...")
		}
		data, err := DownloadWithCache(ctx, GeoNamesAdmin1CodesURL, logFn)
		if err == nil && len(data) > 0 {
			_ = os.WriteFile(admin1Path, data, 0644)
		}
	}

	// 2. 下载并保存 admin2Codes.txt
	admin2Path := filepath.Join(destDir, "admin2Codes.txt")
	if fi, err := os.Stat(admin2Path); err != nil || fi.Size() == 0 {
		if logFn != nil {
			logFn("正在从 GeoNames 同步地级市/区县源映射表 (admin2Codes.txt)...")
		}
		data, err := DownloadWithCache(ctx, GeoNamesAdmin2CodesURL, logFn)
		if err == nil && len(data) > 0 {
			_ = os.WriteFile(admin2Path, data, 0644)
		}
	}

	// 3. 动态热载入到内存映射表
	LoadMappingFilesFromDir(destDir)
	return nil
}

// ParseGeoNamesTSV 解析 GeoNames 标准 TSV 格式数据（19 列），提取各大洲城镇与摄影点位
func ParseGeoNamesTSV(reader io.Reader) (map[string][]geocoding.GeoPoint, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 512*1024)
	scanner.Buffer(buf, 2*1024*1024)

	continentMap := make(map[string][]geocoding.GeoPoint)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		cols := strings.Split(line, "\t")
		if len(cols) < 11 {
			continue
		}

		var geonameID int64
		if len(cols) > 0 {
			geonameID, _ = strconv.ParseInt(cols[0], 10, 64)
		}
		nameUTF8 := cols[1]
		asciiname := cols[2]
		altNames := cols[3]
		latStr := cols[4]
		lonStr := cols[5]
		fClass := cols[6]
		fCode := cols[7]
		countryCode := cols[8]
		admin1Code := cols[10]

		var admin2Code string
		if len(cols) > 11 {
			admin2Code = cols[11]
		}

		var population int64
		if len(cols) > 14 && cols[14] != "" {
			population, _ = strconv.ParseInt(cols[14], 10, 64)
		}

		var elevation int
		if len(cols) > 15 && cols[15] != "" {
			elevation, _ = strconv.Atoi(cols[15])
		}

		var dem int
		if len(cols) > 16 && cols[16] != "" {
			dem, _ = strconv.Atoi(cols[16])
		}

		var timezone string
		if len(cols) > 17 {
			timezone = cols[17]
		}

		var modDate string
		if len(cols) > 18 {
			modDate = cols[18]
		}

		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		countryMeta := GetCountryMeta(countryCode)
		pointNameZH := ExtractChineseName(asciiname, altNames)
		if pointNameZH == "" {
			pointNameZH = nameUTF8
		}

		province := admin1Code
		cityName := pointNameZH
		district := ""

		if countryCode == "CN" || countryCode == "HK" || countryCode == "MO" || countryCode == "TW" {
			province = GetChinaProvinceName(admin1Code, lat, lon)
			if province == "" {
				province = "中国"
			}
			resolvedCity := GetChinaCityName(admin2Code, province, fClass, fCode, pointNameZH, lat, lon)
			if resolvedCity != "" {
				cityName = resolvedCity
				if pointNameZH != cityName && pointNameZH != province {
					district = pointNameZH
				}
			}
		}

		point := geocoding.GeoPoint{
			GeoNameID:    geonameID,
			Name:         nameUTF8,
			NameASCII:    asciiname,
			NameZH:       pointNameZH,
			Lat:          lat,
			Lon:          lon,
			FeatureClass: fClass,
			FeatureCode:  fCode,
			CountryCode:  countryMeta.Code,
			Country:      countryMeta.NameZH,
			Admin1Code:   admin1Code,
			Admin2Code:   admin2Code,
			Province:     province,
			City:         cityName,
			District:     district,
			Population:   population,
			Elevation:    elevation,
			DEM:          dem,
			Timezone:     timezone,
			ModDate:      modDate,
			Source:       "geonames_" + countryMeta.Continent,
		}

		continent := countryMeta.Continent
		continentMap[continent] = append(continentMap[continent], point)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 GeoNames 数据流失败: %w", err)
	}

	return continentMap, nil
}

// isUsefulChinaFeature 智能筛选中国全量数据中适合摄影工作流的区划与自然地标
func isUsefulChinaFeature(fClass, fCode string) bool {
	// 行政区划城市/区县/乡镇 (P 类)
	if fClass == "P" {
		switch fCode {
		case "PPLC", "PPLA", "PPLA2", "PPLA3", "PPLA4", "PPL", "PPLX":
			return true
		}
	}
	// 行政区域地级市/州/盟/县 (A 类)
	if fClass == "A" {
		switch fCode {
		case "ADM1", "ADM2", "ADM3", "ADM4", "ADMD":
			return true
		}
	}
	// 山脉/山峰/峡谷/冰川/垭口/公路奇观 (T 类)
	if fClass == "T" {
		switch fCode {
		case "MT", "MTS", "PK", "PKS", "VLY", "GLCR", "PASS", "GAP", "RDGE", "CLIFF", "ISL", "PT", "HLL":
			return true
		}
	}
	// 湖泊/河流/瀑布/海湾/水库 (H 类)
	if fClass == "H" {
		switch fCode {
		case "LK", "LKS", "WTRH", "FLL", "STM", "BAY", "RSV":
			return true
		}
	}
	// 国家公园/风景名胜区/寺庙/古迹/地质公园 (L / S 类)
	if fClass == "L" || fClass == "S" {
		switch fCode {
		case "PRK", "RES", "TMPL", "MNMT", "RUIN", "SPA", "CST", "MUS", "AIRP", "RSTN", "BDG":
			return true
		}
	}
	return false
}

// ParseChinaGeoNamesTSV 针对中国 CN.txt 全量高精数据包进行深度中文清洗与全字段结构化提取
func ParseChinaGeoNamesTSV(reader io.Reader) ([]geocoding.GeoPoint, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 512*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var points []geocoding.GeoPoint
	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		cols := strings.Split(line, "\t")
		if len(cols) < 11 {
			continue
		}

		var geonameID int64
		if len(cols) > 0 {
			geonameID, _ = strconv.ParseInt(cols[0], 10, 64)
		}
		nameUTF8 := cols[1]
		asciiname := cols[2]
		altNames := cols[3]
		latStr := cols[4]
		lonStr := cols[5]
		fClass := cols[6]
		fCode := cols[7]
		countryCode := cols[8]
		admin1Code := cols[10]

		var admin2Code string
		if len(cols) > 11 {
			admin2Code = cols[11]
		}

		var population int64
		if len(cols) > 14 && cols[14] != "" {
			population, _ = strconv.ParseInt(cols[14], 10, 64)
		}

		var elevation int
		if len(cols) > 15 && cols[15] != "" {
			elevation, _ = strconv.Atoi(cols[15])
		}

		var dem int
		if len(cols) > 16 && cols[16] != "" {
			dem, _ = strconv.Atoi(cols[16])
		}

		var timezone string
		if len(cols) > 17 {
			timezone = cols[17]
		}

		var modDate string
		if len(cols) > 18 {
			modDate = cols[18]
		}

		if countryCode != "CN" && countryCode != "HK" && countryCode != "MO" && countryCode != "TW" {
			continue
		}

		// 过滤出有意义的行政区划与自然/人文地标
		if !isUsefulChinaFeature(fClass, fCode) {
			continue
		}

		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		countryMeta := GetCountryMeta(countryCode)
		pointNameZH := ExtractChineseName(asciiname, altNames)
		if pointNameZH == "" {
			pointNameZH = nameUTF8
		}

		province := GetChinaProvinceName(admin1Code, lat, lon)
		if province == "" {
			province = countryMeta.NameZH
		}

		cityName := GetChinaCityName(admin2Code, province, fClass, fCode, pointNameZH, lat, lon)
		if cityName == "" {
			if fClass == "P" {
				cityName = pointNameZH
			} else {
				cityName = province
			}
		}

		district := ""
		if pointNameZH != "" && pointNameZH != cityName && pointNameZH != province {
			district = pointNameZH
		}

		// 去重 (相同省份+相同地级市+相同地名+相近坐标)
		dedupKey := fmt.Sprintf("%s_%s_%s_%.3f_%.3f", province, cityName, pointNameZH, lat, lon)
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}

		pt := geocoding.GeoPoint{
			GeoNameID:    geonameID,
			Name:         nameUTF8,
			NameASCII:    asciiname,
			NameZH:       pointNameZH,
			Lat:          lat,
			Lon:          lon,
			FeatureClass: fClass,
			FeatureCode:  fCode,
			CountryCode:  countryMeta.Code,
			Country:      countryMeta.NameZH,
			Admin1Code:   admin1Code,
			Admin2Code:   admin2Code,
			Province:     province,
			City:         cityName,
			District:     district,
			Population:   population,
			Elevation:    elevation,
			DEM:          dem,
			Timezone:     timezone,
			ModDate:      modDate,
			Source:       "geonames_china_ultra",
		}
		points = append(points, pt)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取中国全量数据流失败: %w", err)
	}

	return points, nil
}

// DownloadAndParseGeoNamesZip 下载并解压 GeoNames ZIP 压缩包 (支持本地缓存与交互覆盖)
func DownloadAndParseGeoNamesZip(ctx context.Context, url string, logFn func(string)) (
	map[string][]geocoding.GeoPoint, error,
) {
	data, err := DownloadWithCache(ctx, url, logFn)
	if err != nil {
		return nil, err
	}

	if logFn != nil {
		logFn("正在解压并解析全球城镇空间数据...")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("解析 ZIP 压缩结构失败: %w", err)
	}

	for _, file := range zipReader.File {
		baseName := filepath.Base(file.Name)
		if strings.HasSuffix(file.Name, ".txt") && !strings.EqualFold(baseName, "readme.txt") {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return ParseGeoNamesTSV(rc)
		}
	}

	return nil, fmt.Errorf("在 ZIP 文件中未找到有效的 GeoNames TSV 数据文件")
}

// DownloadAndParseChinaZip 专门下载并解析中国全境 GeoNames 数据包 (CN.zip)
func DownloadAndParseChinaZip(ctx context.Context, logFn func(string)) (
	[]geocoding.GeoPoint, error,
) {
	return DownloadAndParseChinaZipFromURL(ctx, GeoNamesChinaURL, logFn)
}

// DownloadAndParseChinaZipFromURL 从指定 URL 下载并解析中国 GeoNames ZIP 数据包
func DownloadAndParseChinaZipFromURL(ctx context.Context, url string, logFn func(string)) (
	[]geocoding.GeoPoint, error,
) {
	data, err := DownloadWithCache(ctx, url, logFn)
	if err != nil {
		return nil, err
	}

	if logFn != nil {
		logFn("正在流式提取并中文化中国 34 省市区县全量行政区划与摄影名胜点位...")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("解析 ZIP 压缩结构失败: %w", err)
	}

	for _, file := range zipReader.File {
		baseName := filepath.Base(file.Name)
		if strings.HasSuffix(file.Name, ".txt") && !strings.EqualFold(baseName, "readme.txt") {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return ParseChinaGeoNamesTSV(rc)
		}
	}

	return nil, fmt.Errorf("在 ZIP 文件中未找到有效的 CN.txt 数据文件")
}
