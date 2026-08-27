package geocoding

import (
	"math"
	"sort"
	"time"
)

const earthRadiusKm = 6371.0

// point3D 三维笛卡尔单位球面坐标
type point3D struct {
	x, y, z float64
	geo     GeoPoint
}

func toPoint3D(g GeoPoint) point3D {
	latRad := g.Lat * (math.Pi / 180.0)
	lonRad := g.Lon * (math.Pi / 180.0)
	return point3D{
		x:   math.Cos(latRad) * math.Cos(lonRad),
		y:   math.Cos(latRad) * math.Sin(lonRad),
		z:   math.Sin(latRad),
		geo: g,
	}
}

// kdNode 3D KD-Tree 树节点
type kdNode struct {
	point point3D
	axis  int // 0: X, 1: Y, 2: Z
	left  *kdNode
	right *kdNode
}

// CandidatePoint 候选匹配点及球面大圆距离
type CandidatePoint struct {
	Point      GeoPoint `json:"point"`
	DistanceKm float64  `json:"distance_km"`
}

// QueryDebugStats KD-Tree 查询调试指标
type QueryDebugStats struct {
	TargetLat     float64          `json:"target_lat"`
	TargetLon     float64          `json:"target_lon"`
	TargetX       float64          `json:"target_x"`
	TargetY       float64          `json:"target_y"`
	TargetZ       float64          `json:"target_z"`
	VisitedNodes  int              `json:"visited_nodes"`
	TotalNodes    int              `json:"total_nodes"`
	PruneRate     float64          `json:"prune_rate"`
	Duration      time.Duration    `json:"duration"`
	TopCandidates []CandidatePoint `json:"top_candidates"`
}

// KDTree 三维球面空间 KD 树索引结构（彻底杜绝极点与 ±180° 日界线缝隙问题）
type KDTree struct {
	root  *kdNode
	count int
}

// NewKDTree 根据地理点集构建 3D 球面 KD-Tree
func NewKDTree(points []GeoPoint) *KDTree {
	pts := make([]point3D, len(points))
	for i, p := range points {
		pts[i] = toPoint3D(p)
	}
	return &KDTree{
		root:  buildKDTree(pts, 0),
		count: len(pts),
	}
}

func buildKDTree(points []point3D, depth int) *kdNode {
	if len(points) == 0 {
		return nil
	}

	axis := depth % 3 // 0: X, 1: Y, 2: Z

	switch axis {
	case 0:
		sort.Slice(points, func(i, j int) bool { return points[i].x < points[j].x })
	case 1:
		sort.Slice(points, func(i, j int) bool { return points[i].y < points[j].y })
	default:
		sort.Slice(points, func(i, j int) bool { return points[i].z < points[j].z })
	}

	medianIdx := len(points) / 2
	node := &kdNode{
		point: points[medianIdx],
		axis:  axis,
	}

	node.left = buildKDTree(points[:medianIdx], depth+1)
	node.right = buildKDTree(points[medianIdx+1:], depth+1)

	return node
}

// chordSqToKm 将三维弦长欧氏距离平方转换为球面大圆物理距离 (km)
func chordSqToKm(distSq float64) float64 {
	chordLen := math.Sqrt(distSq)
	if chordLen > 2.0 {
		chordLen = 2.0
	}
	centralAngle := 2.0 * math.Asin(chordLen/2.0)
	return centralAngle * earthRadiusKm
}

// haversineDistance 计算两点间的球面大圆距离 (km)，用于基准对比
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// Nearest 寻找离 (lat, lon) 最近的点及其实际物理大圆距离 (km)
func (t *KDTree) Nearest(lat, lon float64) (*GeoPoint, float64) {
	if t.root == nil {
		return nil, math.MaxFloat64
	}

	target := toPoint3D(GeoPoint{Lat: lat, Lon: lon})

	var bestPoint *GeoPoint
	bestDistSq := math.MaxFloat64

	var search func(n *kdNode)
	search = func(n *kdNode) {
		if n == nil {
			return
		}

		dx := target.x - n.point.x
		dy := target.y - n.point.y
		dz := target.z - n.point.z
		distSq := dx*dx + dy*dy + dz*dz

		if distSq < bestDistSq {
			bestDistSq = distSq
			bestPoint = &n.point.geo
		}

		var targetVal, nodeVal float64
		switch n.axis {
		case 0:
			targetVal, nodeVal = target.x, n.point.x
		case 1:
			targetVal, nodeVal = target.y, n.point.y
		default:
			targetVal, nodeVal = target.z, n.point.z
		}

		var first, second *kdNode
		if targetVal < nodeVal {
			first, second = n.left, n.right
		} else {
			first, second = n.right, n.left
		}

		search(first)

		planeDist := targetVal - nodeVal
		if (planeDist * planeDist) < bestDistSq {
			search(second)
		}
	}

	search(t.root)

	if bestPoint == nil {
		return nil, math.MaxFloat64
	}

	return bestPoint, chordSqToKm(bestDistSq)
}

// NearestKWithStats 寻找离 (lat, lon) 最近的前 K 个候选点位并返回性能统计
func (t *KDTree) NearestKWithStats(lat, lon float64, k int) (*GeoPoint, float64, QueryDebugStats) {
	startTime := time.Now()
	target := toPoint3D(GeoPoint{Lat: lat, Lon: lon})

	stats := QueryDebugStats{
		TargetLat:  lat,
		TargetLon:  lon,
		TargetX:    target.x,
		TargetY:    target.y,
		TargetZ:    target.z,
		TotalNodes: t.count,
	}

	if t.root == nil || k <= 0 {
		stats.Duration = time.Since(startTime)
		return nil, math.MaxFloat64, stats
	}

	type candidateItem struct {
		point  GeoPoint
		distSq float64
	}

	var candidates []candidateItem
	visited := 0

	var search func(n *kdNode)
	search = func(n *kdNode) {
		if n == nil {
			return
		}
		visited++

		dx := target.x - n.point.x
		dy := target.y - n.point.y
		dz := target.z - n.point.z
		distSq := dx*dx + dy*dy + dz*dz

		// 插入候选集（保持升序）
		idx := sort.Search(len(candidates), func(i int) bool {
			return candidates[i].distSq >= distSq
		})
		if idx < k {
			item := candidateItem{point: n.point.geo, distSq: distSq}
			if len(candidates) < k {
				candidates = append(candidates, candidateItem{})
			}
			copy(candidates[idx+1:], candidates[idx:])
			candidates[idx] = item
			if len(candidates) > k {
				candidates = candidates[:k]
			}
		}

		var targetVal, nodeVal float64
		switch n.axis {
		case 0:
			targetVal, nodeVal = target.x, n.point.x
		case 1:
			targetVal, nodeVal = target.y, n.point.y
		default:
			targetVal, nodeVal = target.z, n.point.z
		}

		var first, second *kdNode
		if targetVal < nodeVal {
			first, second = n.left, n.right
		} else {
			first, second = n.right, n.left
		}

		search(first)

		worstDistSq := math.MaxFloat64
		if len(candidates) == k {
			worstDistSq = candidates[k-1].distSq
		}

		planeDist := targetVal - nodeVal
		if (planeDist * planeDist) < worstDistSq {
			search(second)
		}
	}

	search(t.root)

	stats.VisitedNodes = visited
	stats.Duration = time.Since(startTime)
	if t.count > 0 {
		stats.PruneRate = (1.0 - float64(visited)/float64(t.count)) * 100.0
	}

	for _, c := range candidates {
		stats.TopCandidates = append(stats.TopCandidates, CandidatePoint{
			Point:      c.point,
			DistanceKm: math.Round(chordSqToKm(c.distSq)*100) / 100,
		})
	}

	if len(candidates) == 0 {
		return nil, math.MaxFloat64, stats
	}

	best := &candidates[0].point
	bestKm := chordSqToKm(candidates[0].distSq)
	return best, bestKm, stats
}

// Size 返回索引的点数量
func (t *KDTree) Size() int {
	return t.count
}
