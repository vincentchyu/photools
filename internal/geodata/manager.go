package geodata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vincentchyu/photo-processing/internal/geocoding"
)

// ContinentStatus 大洲安装状态
type ContinentStatus struct {
	Meta      ContinentMeta `json:"meta"`
	Installed bool          `json:"installed"`
	FilePath  string        `json:"file_path,omitempty"`
	FileSize  int64         `json:"file_size,omitempty"`
	Points    int           `json:"points,omitempty"`
}

// Manager 管理地理数据包的安装、更新、卸载与状态查询
type Manager struct {
	dataDir string
}

// NewManager 创建默认数据包管理器 (~/.config/photools/geodata/)
func NewManager() (*Manager, error) {
	dataDir, err := GetGeoDataDir()
	if err != nil {
		return nil, err
	}
	LoadMappingFilesFromDir(dataDir)
	return &Manager{dataDir: dataDir}, nil
}

// NewManagerWithDir 允许指定自定义数据目录（用于测试或便携部署）
func NewManagerWithDir(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	LoadMappingFilesFromDir(dir)
	return &Manager{dataDir: dir}, nil
}

// GetDataDir 获取标准地理数据包安装目录
func (m *Manager) GetDataDir() string {
	return m.dataDir
}

// GetGeoDataDir 获取标准地理数据包安装目录
func GetGeoDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "photools", "geodata")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ListContinents 列出所有大洲包及其本地安装状态
func (m *Manager) ListContinents() []ContinentStatus {
	var list []ContinentStatus
	for _, meta := range AvailableContinents {
		filePath := filepath.Join(m.dataDir, meta.Code+".json")
		status := ContinentStatus{
			Meta:      meta,
			Installed: false,
		}

		if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
			status.Installed = true
			status.FilePath = filePath
			status.FileSize = info.Size()

			if data, err := os.ReadFile(filePath); err == nil {
				var pts []geocoding.GeoPoint
				if err := json.Unmarshal(data, &pts); err == nil {
					status.Points = len(pts)
				}
			}
		}

		list = append(list, status)
	}
	return list
}

// Install 安装或更新指定大洲/国家的数据包
func (m *Manager) Install(ctx context.Context, nameOrAlias string, logFn func(string)) error {
	meta := FindContinent(nameOrAlias)
	if meta == nil {
		if strings.EqualFold(nameOrAlias, "all") || strings.EqualFold(nameOrAlias, "全部") {
			return m.InstallAll(ctx, logFn)
		}
		return fmt.Errorf("未找到数据包: %q（可用包: china, asia, europe, north-america, oceania, south-america, africa, all）", nameOrAlias)
	}

	// 同步 GeoNames 官方源映射表 (admin1CodesASCII.txt 与 admin2Codes.txt)，与各大洲文件一同保存在数据目录
	_ = DownloadAndCacheAdminCodes(ctx, m.dataDir, logFn)

	targetFile := filepath.Join(m.dataDir, meta.Code+".json")
	if logFn != nil {
		logFn(fmt.Sprintf("正在准备安装 [%s] 数据包...", meta.NameZH))
	}

	var downloadedPoints []geocoding.GeoPoint
	var downloadErr error

	// 1. 如果是中国全量高精包 (CN.zip)
	if meta.Code == "china" {
		pts, err := DownloadAndParseChinaZip(ctx, logFn)
		if err == nil && len(pts) > 0 {
			downloadedPoints = pts
			if logFn != nil {
				logFn(fmt.Sprintf("✅ 成功从 GeoNames 提取并中文化 [中国全境高精] %d 个行政区划与摄影地标点位", len(pts)))
			}
		} else {
			if err != nil {
				downloadErr = err
			} else {
				downloadErr = fmt.Errorf("解析 CN.zip 未获取到有效点位")
			}
		}
	} else {
		// 2. 从全球 cities15000.zip 提取目标大洲
		for _, url := range meta.DownloadURLs {
			if strings.HasSuffix(url, ".zip") {
				continentMap, err := DownloadAndParseGeoNamesZip(ctx, url, logFn)
				if err == nil {
					if pts, ok := continentMap[meta.Code]; ok && len(pts) > 0 {
						downloadedPoints = pts
						if logFn != nil {
							logFn(fmt.Sprintf("✅ 成功从 GeoNames 提取并中文化 [%s] %d 个城镇点位", meta.NameZH, len(pts)))
						}
						// 同时缓存其他大洲点位
						for cCode, cPts := range continentMap {
							if cCode != meta.Code && len(cPts) > 0 {
								_ = m.savePoints(cCode, cPts)
							}
						}
						break
					}
				}
				downloadErr = err
			}
		}
	}

	// 3. 容灾回退
	if len(downloadedPoints) == 0 {
		if len(meta.BasePoints) > 0 {
			if logFn != nil {
				if downloadErr != nil {
					logFn(fmt.Sprintf("⚠️ 官方源暂不可达 (%v)，自动载入内置离线精选数据（%d 个核心地标）...", downloadErr, len(meta.BasePoints)))
				} else {
					logFn(fmt.Sprintf("载入内置离线精选数据（%d 个核心地标）...", len(meta.BasePoints)))
				}
			}
			downloadedPoints = meta.BasePoints
		} else {
			if downloadErr != nil {
				return fmt.Errorf("下载失败且无可用离线基础包: %w", downloadErr)
			}
			return fmt.Errorf("未能从数据源解析出有效点位且无离线基础包")
		}
	}

	// 4. 写入目标文件
	if err := m.savePoints(meta.Code, downloadedPoints); err != nil {
		return err
	}

	if logFn != nil {
		logFn(fmt.Sprintf("🎉 [%s] 数据包已成功就绪并保存至: %s", meta.NameZH, targetFile))
	}

	return nil
}

// savePoints 将点位列表格式化保存为 JSON 文件
func (m *Manager) savePoints(code string, points []geocoding.GeoPoint) error {
	targetFile := filepath.Join(m.dataDir, code+".json")
	data, err := json.MarshalIndent(points, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化数据包失败: %w", err)
	}
	if err := os.WriteFile(targetFile, data, 0644); err != nil {
		return fmt.Errorf("写入数据包文件失败 (%s): %w", targetFile, err)
	}
	return nil
}

// InstallAll 一键安装所有可用大洲及国家高精数据包
func (m *Manager) InstallAll(ctx context.Context, logFn func(string)) error {
	if logFn != nil {
		logFn("🌐 开始一键同步并安装全部大洲与中国高精离线地理数据包...")
	}

	// 1. 同步官方映射表源文件
	_ = DownloadAndCacheAdminCodes(ctx, m.dataDir, logFn)

	// 2. 安装中国全量高精包
	if err := m.Install(ctx, "china", logFn); err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("⚠️ 安装中国高精数据包异常: %v", err))
		}
	}

	// 3. 安装全球各洲 cities15000 包
	continentMap, err := DownloadAndParseGeoNamesZip(ctx, GeoNamesCities15000URL, logFn)
	if err == nil && len(continentMap) > 0 {
		for _, meta := range AvailableContinents {
			if meta.Code == "china" {
				continue
			}
			if pts, ok := continentMap[meta.Code]; ok && len(pts) > 0 {
				_ = m.savePoints(meta.Code, pts)
				if logFn != nil {
					logFn(fmt.Sprintf("✅ [%s] 已写入 %d 个城镇点位", meta.NameZH, len(pts)))
				}
			}
		}
		if logFn != nil {
			logFn("🎉 全球各大洲及中国高精离线数据包已全部安装就绪！")
		}
		return nil
	}

	// 容灾离线保底
	if logFn != nil {
		logFn("⚠️ 官方源无法访问，批量载入内置各洲高精度离线精选数据...")
	}
	for _, meta := range AvailableContinents {
		if meta.Code == "china" {
			continue
		}
		if len(meta.BasePoints) > 0 {
			_ = m.savePoints(meta.Code, meta.BasePoints)
			if logFn != nil {
				logFn(fmt.Sprintf("✅ [%s] 已载入内置 %d 个重点地标", meta.NameZH, len(meta.BasePoints)))
			}
		}
	}
	return nil
}

// Remove 移除已安装的大洲数据包
func (m *Manager) Remove(nameOrAlias string) error {
	meta := FindContinent(nameOrAlias)
	if meta == nil {
		return fmt.Errorf("未找到对应数据包: %q", nameOrAlias)
	}

	targetFile := filepath.Join(m.dataDir, meta.Code+".json")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return fmt.Errorf("数据包 [%s] 尚未安装", meta.NameZH)
	}

	if err := os.Remove(targetFile); err != nil {
		return fmt.Errorf("删除数据包文件失败: %w", err)
	}

	return nil
}
