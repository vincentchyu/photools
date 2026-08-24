package geocoding

import (
	"math"
	"math/rand"
	"testing"
)

func TestKDTree_NearestAccuracy(t *testing.T) {
	points := []GeoPoint{
		{Lat: 39.9042, Lon: 116.4074, Country: "中国", City: "北京市"},
		{Lat: 31.2304, Lon: 121.4737, Country: "中国", City: "上海市"},
		{Lat: 23.1291, Lon: 113.2644, Country: "中国", City: "广州市"},
		{Lat: 30.5728, Lon: 104.0668, Country: "中国", City: "成都市"},
		{Lat: 35.6762, Lon: 139.6503, Country: "日本", City: "东京都"},
	}

	tree := NewKDTree(points)
	if tree.Size() != 5 {
		t.Fatalf("expected size 5, got %d", tree.Size())
	}

	// 查成都周边 (30.6, 104.1)
	p, dist := tree.Nearest(30.6000, 104.1000)
	if p == nil || p.City != "成都市" {
		t.Errorf("expected 成都市, got %+v", p)
	}
	if dist > 20.0 {
		t.Errorf("expected distance < 20km, got %f", dist)
	}

	// 查东京周边 (35.68, 139.7)
	p2, dist2 := tree.Nearest(35.6800, 139.7000)
	if p2 == nil || p2.City != "东京都" {
		t.Errorf("expected 东京都, got %+v", p2)
	}
	if dist2 > 10.0 {
		t.Errorf("expected distance < 10km, got %f", dist2)
	}
}

func TestKDTree_Empty(t *testing.T) {
	tree := NewKDTree([]GeoPoint{})
	p, dist := tree.Nearest(39.9, 116.4)
	if p != nil || dist != math.MaxFloat64 {
		t.Errorf("expected nil and max float for empty tree, got %v, %f", p, dist)
	}
}

// 暴力线性查找用于与 KD-Tree 对比基准正确性
func bruteForceNearest(points []GeoPoint, lat, lon float64) (*GeoPoint, float64) {
	var best *GeoPoint
	minDist := math.MaxFloat64
	for i := range points {
		d := haversineDistance(lat, lon, points[i].Lat, points[i].Lon)
		if d < minDist {
			minDist = d
			best = &points[i]
		}
	}
	return best, minDist
}

func TestKDTree_FuzzAgainstBruteForce(t *testing.T) {
	// 生成随机 500 个全球点
	r := rand.New(rand.NewSource(42))
	var pts []GeoPoint
	for i := 0; i < 500; i++ {
		lat := (r.Float64() * 180.0) - 90.0
		lon := (r.Float64() * 360.0) - 180.0
		pts = append(pts, GeoPoint{
			Lat:  lat,
			Lon:  lon,
			City: "City",
		})
	}

	tree := NewKDTree(pts)

	// 测试 100 个随机查询点，KD-Tree 的最近距离必须与暴力搜索一致（允许浮点微小误差）
	for i := 0; i < 100; i++ {
		qLat := (r.Float64() * 180.0) - 90.0
		qLon := (r.Float64() * 360.0) - 180.0

		_, bfDist := bruteForceNearest(pts, qLat, qLon)
		_, kdDist := tree.Nearest(qLat, qLon)

		if math.Abs(bfDist-kdDist) > 1e-4 {
			t.Fatalf("KDTree distance mismatch: brute_force=%f, kdtree=%f (query: %f, %f)", bfDist, kdDist, qLat, qLon)
		}
	}
}

func BenchmarkKDTree_Lookup(b *testing.B) {
	tree := NewKDTree(embeddedAsiaPoints)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Nearest(31.2304, 121.4737)
	}
}
