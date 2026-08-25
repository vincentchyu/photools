package geocoding

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FormatDebugReport 统一生成 CLI 与 macOS 客户端通用的高精空间分析诊断纯文本报告
func FormatDebugReport(
	lat, lon, alt float64,
	loc *LocationInfo,
	bestPt *GeoPoint,
	distKm float64,
	debugStats QueryDebugStats,
	loadStats GeocoderLoadStats,
	queryDur time.Duration,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔍 正在检索经纬度坐标: (%.6f, %.6f)", lat, lon))
	if alt != 0 {
		sb.WriteString(fmt.Sprintf(" [海拔: %.1fm]", alt))
	}
	sb.WriteString("\n\n")

	if loc == nil {
		sb.WriteString("❌ 未在离线地理库中匹配到有效地点\n")
		return sb.String()
	}

	sb.WriteString("📍 【逆地理编码匹配结果】\n")
	sb.WriteString(fmt.Sprintf("  • 国家:       %s (%s)\n", loc.Country, loc.CountryCode))
	if loc.Province != "" {
		sb.WriteString(fmt.Sprintf("  • 省份/州:    %s\n", loc.Province))
	}
	if loc.City != "" {
		sb.WriteString(fmt.Sprintf("  • 城市/地区:  %s\n", loc.City))
	}
	if loc.District != "" {
		sb.WriteString(fmt.Sprintf("  • 区县/POI:   %s\n", loc.District))
	}
	if loc.Timezone != "" {
		sb.WriteString(fmt.Sprintf("  • IANA时区:   %s\n", loc.Timezone))
	}
	if loc.Elevation != 0 {
		sb.WriteString(fmt.Sprintf("  • 地形海拔:   %d 米\n", loc.Elevation))
	}
	sb.WriteString(fmt.Sprintf("  • 物理距离:   %.2f km\n", distKm))
	sb.WriteString(fmt.Sprintf("  • 数据源:     %s\n", loc.Source))
	sb.WriteString(fmt.Sprintf("  • 规范全称:   %s\n", loc.FormatSummary()))

	sb.WriteString("\n🛠️  【空间索引搜索诊断 (Debug)】\n")
	sb.WriteString(fmt.Sprintf("  • 离线库总点数:   %d 个 (内置: %d, 用户自定义: %d)\n",
		loadStats.TotalPoints, loadStats.BuiltinPoints, loadStats.CustomPoints))
	if len(loadStats.Packs) > 0 {
		sb.WriteString(fmt.Sprintf("  • 外挂大洲数据包: %d 个\n", len(loadStats.Packs)))
		for _, pk := range loadStats.Packs {
			sb.WriteString(fmt.Sprintf("    - %-20s: %6d 点位 (%.1f MB, 加载 %.2fms)\n",
				pk.Name, pk.Points, float64(pk.SizeBytes)/(1024*1024), float64(pk.LoadTime.Microseconds())/1000.0))
		}
	}
	sb.WriteString(fmt.Sprintf("  • KD-Tree建树耗时: %.2f ms\n", float64(loadStats.TreeBuildTime.Microseconds())/1000.0))
	sb.WriteString(fmt.Sprintf("  • 数据库全量冷启:  %.2f ms\n", float64(loadStats.TotalInitTime.Microseconds())/1000.0))
	sb.WriteString(fmt.Sprintf("  • KD-Tree遍历节点: %d 次 (剪枝率: %.2f%%)\n", debugStats.VisitedNodes, debugStats.PruneRate))
	sb.WriteString(fmt.Sprintf("  • 3D点位检索总耗时: %.4f ms (%d µs)\n",
		float64(queryDur.Nanoseconds())/1e6, queryDur.Microseconds()))

	if len(debugStats.TopCandidates) > 0 {
		sb.WriteString("\n🗺️  【Top-5 最近邻拓扑候选点】\n")
		for i, c := range debugStats.TopCandidates {
			prefix := " "
			if i == 0 {
				prefix = "★"
			}
			pt := c.Point
			displayName := pt.NameZH
			if displayName == "" {
				displayName = pt.Name
			}
			if displayName == "" {
				if pt.District != "" {
					displayName = pt.District
				} else if pt.City != "" {
					displayName = pt.City
				} else {
					displayName = pt.Province
				}
			}
			nameDesc := displayName
			if pt.Name != "" && pt.Name != displayName {
				nameDesc = fmt.Sprintf("%s (%s)", displayName, pt.Name)
			}

			featureDesc := FormatFeatureCodeZH(pt.FeatureClass, pt.FeatureCode)
			locHierarchy := FormatPointLocation(&pt)

			var metaParts []string
			metaParts = append(metaParts, fmt.Sprintf("坐标: (%.6f, %.6f)", pt.Lat, pt.Lon))
			if pt.DEM != 0 {
				metaParts = append(metaParts, fmt.Sprintf("高程: %dm", pt.DEM))
			} else if pt.Elevation != 0 {
				metaParts = append(metaParts, fmt.Sprintf("海拔: %dm", pt.Elevation))
			}
			if pt.GeoNameID != 0 {
				metaParts = append(metaParts, fmt.Sprintf("ID: %d", pt.GeoNameID))
			}
			if pt.Source != "" {
				metaParts = append(metaParts, pt.Source)
			}

			sb.WriteString(fmt.Sprintf("  %s [%d] 距离 %6.2f km ➔ %s [%s]\n", prefix, i+1, c.DistanceKm, nameDesc, featureDesc))
			sb.WriteString(fmt.Sprintf("        • 归属: %s\n", locHierarchy))
			sb.WriteString(fmt.Sprintf("        • %s\n", strings.Join(metaParts, " | ")))
		}
	}

	if bestPt != nil {
		rawJSON, _ := json.MarshalIndent(bestPt, "  ", "  ")
		sb.WriteString(fmt.Sprintf("\n📄 【底层命中点位原始 GeoPoint 结构】\n  %s\n", string(rawJSON)))
	}

	return sb.String()
}

// FormatPointLocation 格式化点位的完整层级归属
func FormatPointLocation(pt *GeoPoint) string {
	var parts []string
	if pt.Country != "" {
		parts = append(parts, pt.Country)
	}
	if pt.Province != "" && pt.Province != pt.Country {
		parts = append(parts, pt.Province)
	}
	if pt.City != "" && pt.City != pt.Province {
		parts = append(parts, pt.City)
	}
	if pt.District != "" && pt.District != pt.City && pt.District != pt.Province {
		parts = append(parts, pt.District)
	}
	if len(parts) == 0 {
		return "未知位置"
	}
	return strings.Join(parts, " · ")
}

// FormatFeatureCodeZH 将 GeoNames 的 FeatureClass 与 FeatureCode 转换为通俗中文描述
func FormatFeatureCodeZH(fClass, fCode string) string {
	fClass = strings.ToUpper(strings.TrimSpace(fClass))
	fCode = strings.ToUpper(strings.TrimSpace(fCode))
	if fCode == "" {
		return fClass
	}

	code := fClass + "/" + fCode
	switch fCode {
	case "PPLC":
		return code + " 首都/国家级行政中心"
	case "PPLA":
		return code + " 省会/首府/一级行政中心"
	case "PPLA2":
		return code + " 地级市/二级行政中心"
	case "PPLA3":
		return code + " 县级市/区县中心"
	case "PPLA4":
		return code + " 乡镇/四级行政中心"
	case "PPL":
		return code + " 城镇/村落/居民点"
	case "MT", "MTS", "PK":
		return code + " 山峰/山脉/名山"
	case "LK", "LKS":
		return code + " 湖泊/水库"
	case "STM":
		return code + " 河流/溪流"
	case "VAL":
		return code + " 峡谷/山谷"
	case "PASS":
		return code + " 垭口/山口/关隘"
	case "PARK":
		return code + " 国家公园/自然保护区/公园"
	case "AIRP":
		return code + " 机场/航空港"
	case "RSTN":
		return code + " 火车站/高铁站"
	default:
		return code
	}
}
