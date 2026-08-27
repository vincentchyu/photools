// Package geodata 提供离线地理数据源管理、大洲数据包构建/下载/生命周期维护与权威行政区划映射字典。
//
// 本包内嵌了全球国家、中国省级（Admin1）、地级市（Admin2）以及全球数万个行政区划中英文元数据（通过 //go:embed 内嵌在二进制中），并支持从 GeoNames 官方开放源下载或自制离线空间数据包（.pack / .json）。
//
// 核心特性：
//   - 零路径依赖：内嵌 country_codes.json、admin1_codes.json、admin2_codes.json 等字典，编译期打包，随二进制分发；
//   - 线程安全字典映射：提供 GetCountryNameZH、GetAdmin1NameZH、GetAdmin2NameZH 等极速查询函数；
//   - 数据包生命周期管理：Manager 提供对大洲数据包（如 china, asia, europe 等）的安装、更新、卸载与状态探测；
//   - 用户自定义热覆写：支持探测 ~/.config/photools/geodata 增量合并自定义词条。
//
// 快速入门：
//
//	package main
//
//	import (
//	    "fmt"
//	    "github.com/vincentchyu/photools/pkg/geodata"
//	)
//
//	func main() {
//	    // 1. 查询国家中文名
//	    fmt.Println(geodata.GetCountryNameZH("CN")) // "中国"
//
//	    // 2. 查询省级行政区划 (GeoNames FIPS 代码)
//	    fmt.Println(geodata.GetAdmin1NameZH("13"))  // "新疆维吾尔自治区"
//
//	    // 3. 查询地级市行政区划 (GB/T 2260 4位代码)
//	    fmt.Println(geodata.GetAdmin2NameZH("6531")) // "喀什地区"
//	}
package geodata
