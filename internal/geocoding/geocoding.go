package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

// PackLoadInfo 记录单个数据包的装载详情
type PackLoadInfo struct {
	Name         string        `json:"name"`
	Points       int           `json:"points"`
	SizeBytes    int64         `json:"size_bytes"`
	FileSize     int64         `json:"file_size,omitempty"`
	LoadTime     time.Duration `json:"load_time"`
	LoadDuration time.Duration `json:"load_duration,omitempty"`
}

// GeocoderLoadStats 记录离线地理编码器的冷启装载指标
type GeocoderLoadStats struct {
	BuiltinPoints int            `json:"builtin_points"`
	CustomPoints  int            `json:"custom_points"`
	Packs         []PackLoadInfo `json:"packs"`
	TotalPoints   int            `json:"total_points"`
	TreeBuildTime time.Duration  `json:"tree_build_time"`
	TotalInitTime time.Duration  `json:"total_init_time"`
}

// ReverseGeocoder 离线逆地理编码器（基于 3D 球面 KD-Tree 空间加速索引）
type ReverseGeocoder struct {
	mu           sync.RWMutex
	tree         *KDTree
	allPoints    []GeoPoint
	customPoints []GeoPoint
	stats        GeocoderLoadStats
	initialized  bool
	initOnce     sync.Once
	initErr      error
}

var (
	defaultGeocoderInstance *ReverseGeocoder
	defaultMu               sync.Mutex
)

// GetDefault 返回全局默认逆地理编码器单例（轻量返回，不阻塞全量 I/O，全量加载由 InitProgressive 异步汇报）
func GetDefault() *ReverseGeocoder {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	if defaultGeocoderInstance == nil {
		points := make([]GeoPoint, len(embeddedAsiaPoints))
		copy(points, embeddedAsiaPoints)
		tree := NewKDTree(points)
		defaultGeocoderInstance = &ReverseGeocoder{
			allPoints: points,
			tree:      tree,
			stats: GeocoderLoadStats{
				BuiltinPoints: len(points),
				TotalPoints:   len(points),
			},
			initialized: false,
		}
	}
	return defaultGeocoderInstance
}

// ResetDefault 重置全局默认逆地理编码器单例（常用于安装或卸载数据包后热重载）
func ResetDefault() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultGeocoderInstance = nil
}

// IsInitialized 返回当前逆地理编码器是否已完成全量初始化
func (rg *ReverseGeocoder) IsInitialized() bool {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return rg.initialized
}

// ensureInitialized 兜底保证数据加载（当未通过 InitProgressive 预加载而直接调用 Lookup 时）
func (rg *ReverseGeocoder) ensureInitialized() {
	if rg.initialized && rg.tree != nil {
		return
	}
	_ = rg.InitProgressive(context.Background(), nil)
}

// InitProgressive 渐进式异步初始化逆地理编码器，提供细粒度的步骤与进度汇报 (使用 sync.Once 保证全局只装载一次)
func (rg *ReverseGeocoder) InitProgressive(ctx context.Context, cb func(stage string, percent float64, msg string, status domain.PluginHealthStatus, err error)) error {
	rg.mu.RLock()
	alreadyInit := rg.initialized && rg.tree != nil
	stats := rg.stats
	rg.mu.RUnlock()

	if alreadyInit {
		if cb != nil {
			status := domain.HealthReady
			if len(stats.Packs) == 0 {
				status = domain.HealthDegraded
			}
			cb("就绪", 1.0, fmt.Sprintf("离线地理库已就绪 (共 %d 个点位 / %d 个数据包)", stats.TotalPoints, len(stats.Packs)), status, nil)
		}
		return nil
	}

	rg.initOnce.Do(func() {
		rg.mu.Lock()
		defer rg.mu.Unlock()

		initStart := time.Now()

		// 1. 装载内嵌基础亚洲地名
		if cb != nil {
			cb("装载基础库", 0.1, fmt.Sprintf("正在装载内嵌核心地名库 (%d 个点位)...", len(embeddedAsiaPoints)), domain.HealthReady, nil)
		}
		rg.allPoints = make([]GeoPoint, len(embeddedAsiaPoints))
		copy(rg.allPoints, embeddedAsiaPoints)
		rg.stats.BuiltinPoints = len(rg.allPoints)

		// 2. 检查用户自定义地点 places.json
		home, _ := os.UserHomeDir()
		if home != "" {
			userPlaces := filepath.Join(home, ".config", "photools", "places.json")
			if data, err := os.ReadFile(userPlaces); err == nil {
				var customList []GeoPoint
				if err := json.Unmarshal(data, &customList); err == nil && len(customList) > 0 {
					for i := range customList {
						if customList[i].Source == "" {
							customList[i].Source = "user_custom"
						}
					}
					rg.customPoints = customList
					rg.stats.CustomPoints = len(customList)
					if cb != nil {
						cb("用户自定义地名", 0.2, fmt.Sprintf("已装载用户自定义地点 (%d 个点位)", len(customList)), domain.HealthReady, nil)
					}
				}
			}
		}

		// 3. 渐进式扫描装载外挂大洲离线数据包
		var added []GeoPoint
		var geoPackDir string
		if home != "" {
			geoPackDir = filepath.Join(home, ".config", "photools", "geodata")
		}

		var packFiles []os.DirEntry
		if geoPackDir != "" {
			if entries, err := os.ReadDir(geoPackDir); err == nil {
				for _, e := range entries {
					if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
						packFiles = append(packFiles, e)
					}
				}
			}
		}

		if len(packFiles) > 0 {
			for idx, entry := range packFiles {
				select {
				case <-ctx.Done():
					rg.initErr = ctx.Err()
					return
				default:
				}

				fullPath := filepath.Join(geoPackDir, entry.Name())
				fi, err := entry.Info()
				if err != nil {
					continue
				}

				packPercent := 0.2 + (0.6 * float64(idx) / float64(len(packFiles)))
				if cb != nil {
					cb("装载离线数据包", packPercent, fmt.Sprintf("正在解析离线地理包 [%s] (%d/%d)...", entry.Name(), idx+1, len(packFiles)), domain.HealthReady, nil)
				}

				loadStart := time.Now()
				data, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				var packPoints []GeoPoint
				if err := json.Unmarshal(data, &packPoints); err == nil && len(packPoints) > 0 {
					packSource := "pack_" + strings.TrimSuffix(entry.Name(), ".json")
					for i := range packPoints {
						if packPoints[i].Source == "" {
							packPoints[i].Source = packSource
						}
					}
					added = append(added, packPoints...)
					dur := time.Since(loadStart)
					rg.stats.Packs = append(rg.stats.Packs, PackLoadInfo{
						Name:         entry.Name(),
						Points:       len(packPoints),
						SizeBytes:    fi.Size(),
						FileSize:     fi.Size(),
						LoadTime:     dur,
						LoadDuration: dur,
					})
				}
			}
		}

		// 4. 构建 3D 球面 KD-Tree 空间索引
		if cb != nil {
			cb("构建空间索引", 0.85, "正在构建 3D 球面 KD-Tree 空间加速索引...", domain.HealthReady, nil)
		}

		if len(added) > 0 {
			rg.allPoints = append(rg.allPoints, added...)
		}

		combined := make([]GeoPoint, 0, len(rg.allPoints)+len(rg.customPoints))
		combined = append(combined, rg.allPoints...)
		combined = append(combined, customListOrEmpty(rg.customPoints)...)

		treeBuildStart := time.Now()
		rg.tree = NewKDTree(combined)
		rg.stats.TreeBuildTime = time.Since(treeBuildStart)
		rg.stats.TotalInitTime = time.Since(initStart)
		rg.stats.TotalPoints = len(combined)
		rg.initialized = true

		// 注册到全局单例
		defaultMu.Lock()
		defaultGeocoderInstance = rg
		defaultMu.Unlock()

		// 5. 汇报最终就绪状态
		if len(rg.stats.Packs) == 0 {
			if cb != nil {
				cb("降级就绪", 1.0, fmt.Sprintf("⚠️ 未安装外挂离线数据包，已启用内置基础库 (%d 点位)", rg.stats.TotalPoints), domain.HealthDegraded, nil)
			}
		} else {
			if cb != nil {
				cb("就绪", 1.0, fmt.Sprintf("离线地理库就绪 (已加载 %d 点位 / %d 个数据包，建树 %.2fs)", rg.stats.TotalPoints, len(rg.stats.Packs), rg.stats.TreeBuildTime.Seconds()), domain.HealthReady, nil)
			}
		}
	})

	return rg.initErr
}

// NewReverseGeocoder 初始化轻量级基础逆地理编码器（只包含内置点，不阻塞加载磁盘包）
func NewReverseGeocoder() *ReverseGeocoder {
	points := make([]GeoPoint, len(embeddedAsiaPoints))
	copy(points, embeddedAsiaPoints)

	treeBuildStart := time.Now()
	tree := NewKDTree(points)
	treeBuildDuration := time.Since(treeBuildStart)

	rg := &ReverseGeocoder{
		allPoints: points,
		tree:      tree,
		stats: GeocoderLoadStats{
			BuiltinPoints: len(points),
			TreeBuildTime: treeBuildDuration,
			TotalPoints:   len(points),
		},
		initialized: true,
	}
	return rg
}

func pointToLocationInfo(p *GeoPoint, distKm float64) *LocationInfo {
	if p == nil {
		return nil
	}
	elev := p.Elevation
	if elev == 0 && p.DEM != 0 {
		elev = p.DEM
	}
	return &LocationInfo{
		Country:      p.Country,
		CountryCode:  p.CountryCode,
		Province:     p.Province,
		City:         p.City,
		District:     p.District,
		DistanceKm:   distKm,
		Source:       p.Source,
		Timezone:     p.Timezone,
		Elevation:    elev,
		FeatureCode:  p.FeatureCode,
		FeatureClass: p.FeatureClass,
		GeoNameID:    p.GeoNameID,
	}
}

// Size 返回当前已加载的点位总数
func (rg *ReverseGeocoder) Size() int {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return len(rg.allPoints) + len(rg.customPoints)
}

// Lookup 根据经纬度查询最近的地理位置信息
func (rg *ReverseGeocoder) Lookup(lat, lon float64) *LocationInfo {
	rg.mu.RLock()
	if !rg.initialized {
		rg.mu.RUnlock()
		rg.ensureInitialized()
		rg.mu.RLock()
	}
	defer rg.mu.RUnlock()

	if rg.tree == nil {
		return nil
	}

	bestPoint, distKm := rg.tree.Nearest(lat, lon)
	if bestPoint == nil {
		return nil
	}

	return pointToLocationInfo(bestPoint, distKm)
}

// LookupDetailed 根据经纬度查询最近地点，同时返回原始 GeoPoint 结构及距离
func (rg *ReverseGeocoder) LookupDetailed(lat, lon float64) (*LocationInfo, *GeoPoint, float64) {
	rg.mu.RLock()
	if !rg.initialized {
		rg.mu.RUnlock()
		rg.ensureInitialized()
		rg.mu.RLock()
	}
	defer rg.mu.RUnlock()

	if rg.tree == nil {
		return nil, nil, 0
	}

	bestPoint, distKm := rg.tree.Nearest(lat, lon)
	if bestPoint == nil {
		return nil, nil, 0
	}

	return pointToLocationInfo(bestPoint, distKm), bestPoint, distKm
}

// LookupDetailedWithDebug 根据经纬度查询最近地点并返回调试与统计信息
func (rg *ReverseGeocoder) LookupDetailedWithDebug(lat, lon float64, k int) (*LocationInfo, *GeoPoint, float64, QueryDebugStats, GeocoderLoadStats) {
	rg.mu.RLock()
	if !rg.initialized {
		rg.mu.RUnlock()
		rg.ensureInitialized()
		rg.mu.RLock()
	}
	defer rg.mu.RUnlock()

	if rg.tree == nil {
		return nil, nil, 0, QueryDebugStats{}, rg.stats
	}

	bestPoint, distKm, debugStats := rg.tree.NearestKWithStats(lat, lon, k)
	if bestPoint == nil {
		return nil, nil, 0, debugStats, rg.stats
	}

	return pointToLocationInfo(bestPoint, distKm), bestPoint, distKm, debugStats, rg.stats
}

// LoadCustomPlaces 加载用户自定义的 places.json
func (rg *ReverseGeocoder) LoadCustomPlaces(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var customPoints []GeoPoint
	if err := json.Unmarshal(data, &customPoints); err != nil {
		return fmt.Errorf("解析用户地点文件失败: %w", err)
	}

	for i := range customPoints {
		if customPoints[i].Source == "" {
			customPoints[i].Source = "user_custom"
		}
	}

	rg.mu.Lock()
	defer rg.mu.Unlock()

	rg.customPoints = customPoints
	rg.stats.CustomPoints = len(customPoints)

	combined := make([]GeoPoint, 0, len(rg.allPoints)+len(rg.customPoints))
	combined = append(combined, rg.allPoints...)
	combined = append(combined, rg.customPoints...)

	treeBuildStart := time.Now()
	rg.tree = NewKDTree(combined)
	rg.stats.TreeBuildTime = time.Since(treeBuildStart)
	rg.stats.TotalPoints = len(combined)

	return nil
}

// LoadGeoPackDirectory 扫描并装载指定目录下的全部大洲离线数据包
func (rg *ReverseGeocoder) LoadGeoPackDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	rg.mu.Lock()
	defer rg.mu.Unlock()

	var added []GeoPoint
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		fi, err := entry.Info()
		if err != nil {
			continue
		}

		loadStart := time.Now()
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var packPoints []GeoPoint
		if err := json.Unmarshal(data, &packPoints); err != nil {
			continue
		}

		packSource := "pack_" + strings.TrimSuffix(entry.Name(), ".json")
		for i := range packPoints {
			if packPoints[i].Source == "" {
				packPoints[i].Source = packSource
			}
		}

		added = append(added, packPoints...)
		dur := time.Since(loadStart)

		rg.stats.Packs = append(rg.stats.Packs, PackLoadInfo{
			Name:         entry.Name(),
			Points:       len(packPoints),
			SizeBytes:    fi.Size(),
			FileSize:     fi.Size(),
			LoadTime:     dur,
			LoadDuration: dur,
		})
	}

	if len(added) > 0 {
		rg.allPoints = append(rg.allPoints, added...)
		combined := make([]GeoPoint, 0, len(rg.allPoints)+len(rg.customPoints))
		combined = append(combined, rg.allPoints...)
		combined = append(combined, customListOrEmpty(rg.customPoints)...)

		treeBuildStart := time.Now()
		rg.tree = NewKDTree(combined)
		rg.stats.TreeBuildTime = time.Since(treeBuildStart)
		rg.stats.TotalPoints = len(combined)
	}

	return nil
}

// GetStats 返回离线数据包加载统计信息（只读快照，不阻塞执行全量 I/O）
func (rg *ReverseGeocoder) GetStats() GeocoderLoadStats {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return rg.stats
}

func customListOrEmpty(list []GeoPoint) []GeoPoint {
	if list == nil {
		return []GeoPoint{}
	}
	return list
}
