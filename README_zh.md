# 📸 photools 摄影资产地理信息与归档工具箱

<p align="center">
  <img src="img/AppIcon.png" width="128" height="128" alt="photools App Icon" style="border-radius: 28px; box-shadow: 0 12px 30px rgba(0,0,0,0.35);" />
</p>

<p align="center">
  <b>让 GPX 轨迹一键化为带有精准 GPS 坐标、中文地名与拍摄日期的摄影作品库</b><br/>
  <i>专为摄影师打造的自动化 GPX 轨迹对齐、GPS 智能邻近推算、离线高精中文逆地理打标与拍摄日期原子归档工具箱</i>
</p>

<p align="center">
  <a href="README.md"><b>English</b></a> | <a href="README_zh.md"><b>简体中文</b></a>
</p>

<p align="center">
  <a href="https://github.com/vincentchyu/photools/releases/latest"><img src="https://img.shields.io/badge/立即下载-macOS%20客户端%20(.dmg)-blue?style=flat-square&logo=apple" alt="下载 macOS 客户端" /></a>
  <img src="https://img.shields.io/badge/平台-macOS%20%7C%20Linux-black?style=flat-square&logo=apple" alt="Platform" />
  <img src="https://img.shields.io/badge/RAW%20全格式支持-尼康%20%7C%20索尼%20%7C%20佳能%20%7C%20富士%20%7C%20徕卡-darkred?style=flat-square" alt="RAW Support" />
  <img src="https://img.shields.io/badge/Go%20版本-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/开源协议-MIT-green?style=flat-square" alt="License" />
</p>

---

## 💡 真实痛点与设计初衷

> *“去新疆徒步旅行拍摄了 4000 多张 RAW 照片。单反/微单（尼康、索尼、佳能、富士等）没有内置 GPS，但手机、Apple Watch 或佳明手表里记录了完整的两步路/户外 GPX 轨迹。如何把轨迹自动匹配到照片、把‘赛里木湖/夏特古道’等五级中文地名打进 EXIF，并按拍摄日期整理归档，且不把照片上传到任何云端？”*

**photools** 正是为解决这一摄影后期工作流痛点而生：

```
相机 RAW/JPG 照片 + 手表/手机 GPX 轨迹 (两步路 / Garmin / Apple Watch)
                           ↓
                   ┌──────────────┐
                   │   photools   │
                   └──────────────┘
                           ↓
  📍 GPS 经纬度坐标  +  🏙️ 五级规范中文地名  +  📅 按拍摄日期自动归档
```

---

## ✨ 效果实测对比 (Before vs After)

<table width="100%">
<tr>
<th width="50%">❌ 使用 photools 之前</th>
<th width="50%">✅ 使用 photools 之后</th>
</tr>
<tr>
<td>

```text
📁 待处理目录 (Inbox)/
├── DSC_8123.NEF  (尼康 RAW)
├── DSC_8123.JPG
├── DSC_8124.NEF
└── DSC_8125.NEF

元数据状态:
• GPS 坐标: 无 (0, 0)
• 地理名称: 无
• 文件名: 混乱的相机默认编号
```

</td>
<td>

```text
📁 归档库 (Processed/2025/0516/)/
├── 2025-05-16-DSC_8123.NEF
├── 2025-05-16-DSC_8123.JPG
├── 2025-05-16-DSC_8124.NEF
└── 2025-05-16-DSC_8125.NEF

元数据状态 (已写入 EXIF / IPTC / XMP):
• GPS 坐标: 44.5912° N, 81.1684° E
• 地理名称: 中国 / 新疆 / 伊犁 / 赛里木湖
• 在 Lightroom、Apple Photos、Capture One 中可直接按地名搜索！
```

</td>
</tr>
</table>

---

## 🍏 photools for macOS (原生桌面客户端)

photools 为 Mac 摄影师提供了基于 SwiftUI 打造的原生桌面应用。底层直接通过进程内 C-Shared FFI（`libphotools.dylib`）直通核心能力，响应时间仅需 ~0.1ms，**支持无需重启即时在偏好设置中切换简体中文与 English**。

<p align="center">
  <a href="https://github.com/vincentchyu/photools/releases/latest">
    <img src="https://img.shields.io/badge/点击下载-photools--macOS.dmg-blue?style=for-the-badge&logo=apple" alt="下载 macOS 客户端" />
  </a>
</p>

### 📸 客户端界面一览

#### 1. 多阶段流水线控制台与实时日志
> 直观配置工作区目录、自由勾选处理能力，并实时查看每张照片的打标与推算进度。

<p align="center">
  <img src="img/zh/1-Work-Pipeline-Steps.png" alt="多阶段流水线执行全景" width="95%" />
</p>

#### 2. 照片图库与实时 EXIF 元数据检查器
> 浏览待处理照片，深度检查光学曝光参数（快门、光圈、ISO、镜头型号），支持 3D KD-Tree 离线坐标反查实时测试。

<p align="center">
  <img src="img/zh/2-Results-And-Pending.png" alt="处理完成汇总与待处理诊断清单" width="95%" />
</p>

#### 3. 离线高精地理数据包与内置使用文档
> 离线地理数据包管理，免联网极速查询任意经纬度对应的中文地名。

<p align="center">
  <img src="img/zh/3-Guide-And-Geocoding.png" alt="工作区引导与离线逆地理反查工具" width="95%" />
</p>

---

## ⚡ 四步自动化摄影工作流 (The 4-Step Workflow)

```
[待处理 RAW/JPG] ──▶ 1. GPX 轨迹对齐 ──▶ 2. GPS 智能推算 ──▶ 3. 离线地名打标 ──▶ 4. 拍摄日期归档 ──▶ [归档库/YYYY/MMDD/]
```

1. **🗺️ GPX 轨迹精准对齐 (`gpx_matching`)**
   - 自动匹配照片拍摄时间戳与多个 `.gpx` 轨迹文件，支持亚秒级精度与时间偏移补偿（`--geosync`）；
   - 优先写入 RAW 主文件（`.NEF`、`.ARW`、`.CR2/.CR3`、`.RAF`、`.DNG`）并无污染同步到伴随 `.JPG` 与 `.XMP`。
2. **🧠 GPS 智能邻近推算 (`gps_interpolate`)**
   - 高速连拍、室内穿行或峡谷遮挡导致部分照片未记录 GPS？
   - photools 采用大圆球面时间权重插值与同机位近邻继承算法，自动利用前后锚点照片平滑补齐缺失坐标。
3. **📍 离线五级高精中文逆地理 (`reverse_geocode`)**
   - 基于 3D KD-Tree 离线空间索引与本土化地名库（覆盖 71.5 万+ POI 点位）；
   - 毫秒级写入国家、省份、城市、区县及风景区 POI 中文地名至 IPTC/XMP 标签，**100% 离线运行，严守隐私**。
4. **📅 按拍摄日期原子归档 (`date_archive`)**
   - 读取真实 `DateTimeOriginal`，规范重命名为 `YYYY-MM-DD-原文件名`，并将同组配套文件原子安全移动至 `Processed/YYYY/MMDD/` 归档目录。

---

## 🚀 快速上手 (Quick Start)

### 方式 A：macOS 原生桌面应用（推荐摄影师使用）
1. 前往 [Releases](https://github.com/vincentchyu/photools/releases/latest) 下载 `photools-macOS.dmg`；
2. 双击打开，将 `photoolsApp.app` 拖入 `/Applications` 应用程序文件夹；
3. 打开应用，选择照片与 GPX 目录，点击 **开始执行** 即可。

### 方式 B：CLI 命令行自动化
```bash
# 执行完整自动化流水线 (GPX 匹配 + 地名打标 + 日期归档)
photools pipeline

# 开启 GPS 智能时间权重插值推算 (例如设置 30 分钟推算窗口)
photools pipeline --interpolate=true --interpolate-window=30m

# 扁平原地模式（在原目录就地逆地理打标并原地规范化重命名）
photools pipeline --flat=true --in-place=true

# 软降级容错模式（无 GPS 照片跳过地名打标直接按日期安全归档）
photools pipeline --allow-no-gps=true
```

### 方式 C：交互式终端 TUI 工作台
针对终端极客与键盘流用户，photools 内置了基于 Bubble Tea 的交互式终端界面：

```bash
photools tui
```
- **`[1/2/3/4]` 或 `[空格]`**：自由勾选或关闭特定插件；
- **`[O]`**：打开当前插件的专属自描述配置弹窗（如推算窗口、时间偏移）；
- **`[S]`**：全局设置弹窗（支持 `[Tab]` 路径智能补全）；
- **`[Enter]`**：进入 Dry-Run 预检，再次回车即刻执行。

---

## 🛠️ 开发者与高级进阶 (Architecture & Benchmarks)

photools 在工程设计上追求极致的性能、模块化解耦与零内存泄漏。

### 🏗️ 技术架构与底层亮点
* **ExifTool Stay-Open 常驻守护池**：消除重复 `fork/exec` 外部子进程开销（`exiftool -stay_open True -@ -`），元数据单次读取耗时从 ~30ms 骤降至 **1.2ms/op**；
* **$O(\log K)$ 球面 `AnchorIndex`**：内存按拍摄日期分桶的时间有序二分查找索引，**205 ns/op** 定位前后最邻近机位；
* **3D KD-Tree 离线空间索引**：将经纬度映射至三维笛卡尔坐标系（地球半径 $R = 6371	ext{ km}$），毫秒级完成最近邻地名检索，杜绝网络 I/O 开销；
* **Go C-Shared FFI 极速直通**：SwiftUI 原生调用 Go 核心能力，严格保证内存指针生命周期与安全释放（`defer { fnFreeString(ptr) }`）。

### 📊 性能基准测试 (Apple M1 Max, `arm64`)

```bash
go test -run=^$ -bench=. -benchmem ./internal/capabilities/gpsinterpolate/...
```

| 评测目标 | 单次延迟 (Latency) | 吞吐量 (Throughput) | 内存分配 | 说明 |
| :--- | :---: | :---: | :---: | :--- |
| **`AnchorIndex` 二分查找 (10,000 锚点)** | **`205 ns/op`** | `4,870,000 ops/sec` | `0 B/op` | 零堆内存分配 |
| **`AnchorIndex` 批量建树 (1,000 资产)** | **`0.52 ms/op`** | `1,920 batches/sec` | `24 KB/op` | 日期分桶轻量索引 |
| **`ExecuteProcess` 单图推算与动态合入** | **`0.11 ms/op`** | `8,880 photos/sec` | `1.2 KB/op` | 大圆加权计算 |
| **`ExifTool` Stay-Open 守护进程池** | **`1.2 ms/op`** | `830 reads/sec` | 0 fork 开销 | 进程崩溃自动自愈 |

---

### 📦 从源码构建与打包 (Building from Source)

#### 环境依赖
- **Go 语言**: `1.21+`
- **Swift / Xcode 命令行工具**: `Swift 5.9+` (构建 macOS 原生应用)
- **ExifTool**: `12.0+` (`brew install exiftool`)

#### 1. 编译 Go 核心二进制与动态库
```bash
# 1. 编译 Go C-Shared 动态库 (供 Swift GUI 调用)
go build -buildmode=c-shared -o dist/libphotools.dylib ./cmd/photools-cshared

# 2. 编译独立 CLI & TUI 二进制程序
go build -ldflags="-s -w" -o dist/photools ./cmd/photools
```

#### 2. 安装离线高精地理数据包
```bash
# 安装中国高精离线地名库 (71.5 万+ POI，保存至 ~/.config/photools/geodata/)
./dist/photools geodata install china

# 可选：按需安装其他大洲数据包
./dist/photools geodata install asia
./dist/photools geodata install europe
./dist/photools geodata install north-america
```

#### 3. 构建 macOS 原生 App Bundle 与 DMG 安装包
```bash
# 构建独立的 App Bundle (产物位于 dist/photoolsApp.app)
./script/build_and_run.sh --build-only

# 打包为可分发的标准 macOS .dmg 安装镜像
hdiutil create -volname "photools" -srcfolder dist/photoolsApp.app -ov -format UDZO dist/photools-macOS.dmg
```

---

## 📚 深度技术设计文档 (Architecture Design Docs)

更详尽的架构设计与实现细节，请参阅各专项技术文档：

- 📖 **[系统分阶段调度器与插件化架构设计](docs/ARCHITECTURE_AND_DESIGN.md)**
- 🍏 **[macOS 原生客户端与跨语言 C-Shared FFI 设计](docs/MACOS_CLIENT_TECHNICAL_DESIGN.md)**
- 🛡️ **[ExifTool 常驻守护池与数据安全防污染设计](docs/EXIFTOOL_IO_AND_SAFETY_DESIGN.md)**
- 🗺️ **[GeoNames 离线数据包与 3D KD-Tree 空间索引设计](docs/GEONAMES_AND_GEOCODING_DESIGN.md)**
- ⚙️ **[动态配置 Schema 与设置面板自描述设计](docs/CONFIGURATION_AND_SETTINGS_DESIGN.md)**

---

## 📄 开源许可证 (License)

本项目采用 [MIT License](LICENSE) 开源许可证。
