// Package geocoding 提供基于 3D 球面 KD-Tree 空间索引的高性能离线逆地理编码（Reverse Geocoding）计算引擎。
//
// 本包专注于将 GPS 经纬度（WGS84）高频、极速反查为结构化、规范清洗的中文地理位置信息（国家、省份、城市、区县、风景区 POI）。
//
// 核心特性：
//   - 极致性能：全内存 3D 球面 KD-Tree 加速结构，$O(\log N)$ 空间最近邻搜索，单次反查耗时约 200ns；
//   - 内置离线数据：内嵌轻量亚洲核心摄影与城市数据集（//go:embed），零外部依赖即可开箱即用；
//   - 动态包扩展：支持渐进式加载外挂离线大洲数据包（如 China Ultra、Europe、North America）；
//   - 线程安全：内置单例模式（GetDefault()）与读写锁保护，支持高并发并发调用；
//   - 中文地名清洗：自动过滤冗余前缀，识别地级市与特色风景区 POI。
//
// 快速入门：
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "github.com/vincentchyu/photools/pkg/geocoding"
//	)
//
//	func main() {
//	    // 1. 获取全局单例并预热装载数据包
//	    geocoder := geocoding.GetDefault()
//	    _ = geocoder.InitProgressive(context.Background(), nil)
//
//	    // 2. 高频并发查询坐标
//	    loc, err := geocoder.Lookup(39.9042, 116.4074) // 北京天安门
//	    if err == nil {
//	        fmt.Println(loc.FormatSummary()) // "中国 · 北京市 · 东城区"
//	    }
//	}
package geocoding
