# 📦 photools Go SDK & Public API

`photools` 提供了经过高并发生产验证的、高性能纯内存离线逆地理编码（Reverse Geocoding）与地理数据包管理能力库。

本目录包含所有对外部 Go 项目公开的 SDK 包。

---

## 目录结构与包分工

```
pkg/
├── geocoding/    # 纯内存 3D 球面 KD-Tree 极速空间计算引擎、坐标反查与地名清洗
└── geodata/      # //go:embed 内嵌标准行政区划字典、大洲离线数据包下载与生命周期管理
```

| 包路径 | 核心能力 | 典型耗时 | 外部依赖 |
| :--- | :--- | :---: | :---: |
| **`github.com/vincentchyu/photools/pkg/geocoding`** | 3D 球面 KD-Tree 空间最近邻搜索、坐标反查、地名结构清洗格式化 | **~200 ns** / 次 | **零依赖** (全内存) |
| **`github.com/vincentchyu/photools/pkg/geodata`** | 国家/省级/地级市行政代码映射字典、GeoNames 大洲包下载与安装 | **微秒级** (Map 查找) | **零依赖** (`//go:embed`) |

---

## 🛠️ 安装与配置

### 方式 A：本地多项目协同 (`go.work` 推荐)
如果你的项目与 `photools` 位于同一台开发机，推荐使用 Go 1.18+ 的 `go.work` 工作区模式：

```bash
go work init
go work use ./photools
go work use ./your-project
```

### 方式 B：在 `go.mod` 中使用 `replace`
```go
module your-project

go 1.22

require (
    github.com/vincentchyu/photools v0.0.0
)

replace github.com/vincentchyu/photools => ../photools
```

---

## 🚀 快速上手与使用示例

### 1. 离线坐标极速反查 (`pkg/geocoding`)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/vincentchyu/photools/pkg/geocoding"
)

func main() {
    ctx := context.Background()

    // 1. 获取全局单例并执行渐进式预热（内置亚洲点位 + ~/.config/photools/geodata 数据包）
    geocoder := geocoding.GetDefault()
    err := geocoder.InitProgressive(ctx, func(stage string, percent float64, msg string, status geocoding.HealthStatus, err error) {
        fmt.Printf("[初始化] %s - %s (%.0f%%)\n", stage, msg, percent*100)
    })
    if err != nil {
        log.Fatalf("初始化失败: %v", err)
    }

    // 2. 200ns 全内存并发极速反查 (经度, 纬度 WGS84)
    loc := geocoder.Lookup(31.2304, 121.4737) // 上海人民广场
    if loc != nil {
        fmt.Printf("国家: %s (%s)\n", loc.Country, loc.CountryCode)
        fmt.Printf("省份: %s\n", loc.Province)
        fmt.Printf("城市: %s\n", loc.City)
        fmt.Printf("区县/景区: %s\n", loc.District)
        fmt.Printf("距离参考点: %.2f km\n", loc.DistanceKm)
        fmt.Printf("格式化地名: %s\n", loc.FormatSummary()) // "中国 · 上海市 · 黄浦区"
    }

    // 3. 查看当前装载性能指标
    stats := geocoder.GetStats()
    fmt.Printf("总点位数: %d, 建树耗时: %v\n", stats.TotalPoints, stats.TreeBuildTime)
}
```

---

### 2. 标准行政区划与国家字典查询 (`pkg/geodata`)

所有字典均通过 `//go:embed data` 编译期内嵌进二进制，无需配置任何外部 JSON 路径：

```go
package main

import (
    "fmt"

    "github.com/vincentchyu/photools/pkg/geodata"
)

func main() {
    // 1. 国家中英文代码映射
    fmt.Println(geodata.GetCountryNameZH("CN")) // "中国"
    fmt.Println(geodata.GetCountryNameZH("US")) // "美国"
    fmt.Println(geodata.GetCountryNameZH("IS")) // "冰岛"

    // 2. 一级行政区划 (GeoNames FIPS / 国际代码)
    fmt.Println(geodata.GetAdmin1NameZH("13"))    // "新疆维吾尔自治区"
    fmt.Println(geodata.GetAdmin1NameZH("31"))    // "海南省"
    fmt.Println(geodata.GetAdmin1NameZH("JP.40")) // "东京都"

    // 3. 二级行政区划 (地级市 333+ / 区县 47000+)
    fmt.Println(geodata.GetAdmin2NameZH("6531"))         // "喀什地区"
    fmt.Println(geodata.GetAdmin2NameZH("CN.02.3301"))   // "杭州市"

    // 4. 获取全量只读快照 Map
    allProvinces := geodata.GetAllAdmin1ZH()
    fmt.Printf("全球一级行政区元数据条目数: %d\n", len(allProvinces))
}
```

---

### 3. 数据包生命周期管理 (`pkg/geodata.Manager`)

管理磁盘上（默认 `~/.config/photools/geodata`）的大洲离线高精数据包：

```go
package main

import (
    "context"
    "fmt"

    "github.com/vincentchyu/photools/pkg/geodata"
)

func main() {
    mgr, err := geodata.NewManager("")
    if err != nil {
        panic(err)
    }

    // 探测支持的大洲及本地安装状态
    statuses, _ := mgr.ListInstalled()
    for _, st := range statuses {
        fmt.Printf("数据包: %-15s 已安装: %-5v (文件大小: %d B)\n",
            st.Meta.NameZH, st.Installed, st.FileSize)
    }

    // 动态安装中国全境高精包 (自动解压与编译空间索引包)
    // _ = mgr.Install(context.Background(), "china", func(msg string, percent float64) {
    //     fmt.Printf("下载进度: %s (%.0f%%)\n", msg, percent*100)
    // })
}
```

---

## 📊 性能基准与压测表现 (Benchmarks)

在 **Apple Silicon (M1 Max)** 环境下的实测性能数据：

| 测试项 | 耗时 / 性能 | 内存分配 | 说明 |
| :--- | :---: | :---: | :--- |
| **`geocoding.Lookup`** (单次坐标最近邻反查) | **`198 ns/op`** | `0 B/op` | 3D 球面 KD-Tree 二分查找，纯内存零堆分配 |
| **`geocoding.NewKDTree`** (10 万点建树) | **`18 ms`** | 线性分配 | 启动时单例一次性构建，支持并发查询 |
| **`geodata.GetAdmin1NameZH`** (字典映射) | **`35 ns/op`** | `0 B/op` | 线程安全 RWMutex 全内存 Map 命中 |
