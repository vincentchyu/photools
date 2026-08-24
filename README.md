[简体中文](#photo-processing) | [English](#photo-processing-en)

<h1 id="photo-processing">photo-processing</h1>

这是一个面向摄影师工作流的自动化处理工具集，基于**插件化能力架构（Capability Architecture）**与**分阶段屏障调度器（Phased Priority Scheduler）**构建，提供四大核心能力：
1. **GPS 轨迹匹配与修正 (`gpx_matching` · P10)**：从相机同步照片后，结合移动设备导出的 `GPX` 轨迹，批量匹配并写入 RAW 经纬度并同步至 JPG/XMP。
2. **GPS 智能邻近推断与时间插值 (`gps_interpolate` · P15)** ✨：对未命中 GPX 轨迹的照片，根据同批次前后邻近照片的时间差做球面大圆线性插值或同机位邻近推断，自动补全 GPS 坐标。
3. **离线逆地理编码与地名打标 (`reverse_geocode` · P20)**：基于 3D KD-Tree 空间加速索引，将国家、省、市、区、POI 等中文地名写入 IPTC/XMP 元数据。
4. **按拍摄日期归档与规范重命名 (`date_archive` · P100)**：提取 EXIF 拍摄日期，规范重命名并在 `Processed/YYYY/MMDD/` 目录下安全原子归档。

项目当前采用 `Go CLI + TUI + exiftool` 的方式工作，重点解决“导入电脑后的元数据补充、地名解析与文件整理”环节，不扩展成复杂的 DAM 或后期修图系统。

![整理后的轨迹与图片效果](static/img.png)

## 核心特性与架构设计

- **四大能力解耦**：每个能力独立实现 `Capability` 接口，各自可独立调用，亦可自由组合；
- **GPS 丢失智能自愈**：通过前后照片时间权重自动推算坐标（默认推算窗口 15 分钟，支持自定义）；
- **容错降级策略 (`--allow-no-gps`)**：对彻底无 GPS 照片支持跳过地名打标、直接按拍摄日期规范归档；
- **分阶段屏障调度 (Phased Priority Scheduler)**：
  - 各插件按 `Priority` 优先级数值划分阶段（Phase 1 ➔ Phase 2 ➔ Phase 3 ➔ Phase 4）；
  - 全量照片在当前阶段完成元数据清洗后才穿过同步屏障，确保在进入归档移动前元数据彻底闭环，杜绝文件竞争；
- **配置文件持久化 (`~/.config/photools/plugins.json`)**：系统自动生成默认配置，支持用户自定义修改优先级数值与启用开关；
- **TUI 插件化多选工作台**：在交互终端中通过快捷键 `[1/2/3/4]` 自由勾选组合所需能力；
- **双端支持**：统一的 `photools` CLI 命令行 + 薄 Mac SwiftUI 原生查看器。

详细技术文档与设计规范请查阅：
- 📖 **[架构与时序图设计文档](docs/ARCHITECTURE_AND_DESIGN.md)**
- ⚙️ **[配置与设置面板设计文档](docs/CONFIGURATION_AND_SETTINGS_DESIGN.md)**
- 🛡️ **[ExifTool I/O 读写机制与数据安全设计文档](docs/EXIFTOOL_IO_AND_SAFETY_DESIGN.md)**
- 🗺️ **[GeoNames 地理数据体系、中文化清洗与 3D KD-Tree 索引设计文档](docs/GEONAMES_AND_GEOCODING_DESIGN.md)**

---

## 目录结构

```text
GPS/
├── cmd/photools/          # 统一 CLI 工具入口
├── docs/                  # 系统架构、配置设计与底层安全设计文档
├── macos/PhotoToolsApp/   # Mac 原生 SwiftUI 工作台
├── internal/              # 内部核心逻辑
│   ├── capabilities/      # 四大独立能力插件 (gpxmatch / gpsinterpolate / reversegeocode / datearchive)
│   ├── config/            # 插件优先级与持久化配置管理
│   ├── pipeline/          # 分阶段屏障流水线编排器 (Orchestrator)
│   ├── domain/            # 领域模型 (Capability, AssetContext, Event)
│   ├── geocoding/         # 离线 3D KD-Tree 逆地理空间索引
│   ├── geodata/           # 大洲高精离线地理数据包管理器
│   ├── exiftool/          # ExifTool 交互与元数据操作
│   └── engine/            # 资产发现、并发池与归档引擎
├── Inbox/                 # 相机同步到电脑后的待处理照片
├── GPX/                   # 移动设备导出的轨迹文件
├── Processed/             # 按拍摄日期归档后的照片
├── Logs/                  # 实时中文运行日志 (photools_latest.log) 与待处理报告 (inbox_pending_report_latest.md)
└── GEMINI.md              # 项目核心架构规约与持久化记忆 (Single Source of Truth)
```

---

## 依赖要求

- Go `1.21+`
- [ExifTool](https://exiftool.org/)

先确认环境可用：

```bash
go version
exiftool -ver
```

---

## 使用方式

### 1. 交互式 TUI 终端工作台（🌟 强烈推荐）

直接在终端运行（无需记复杂参数）：
```bash
./photools
```
或显式进入 TUI：
```bash
./photools tui
```

#### 启动阶段：插件并发自检与渐进式装载 (Progressive Initialization)
启动时，系统自动并发执行各插件的 `Init` 自检与流式装载，实时显示步骤与进度条：
```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ⚡ 流水线能力插件自检与装载 (Capabilities Self-Check & Loading)...         │
│                                                                            │
│  [✔] 就绪   GPX 轨迹匹配与 GPS 修正 (GPX Matching)                         │
│      └─ [环境自检] ExifTool 核心引擎就绪 (v12.76)                           │
│                                                                            │
│  [⚙️] 装载中 逆地理编码与地名元数据写入 (Reverse Geocode)                   │
│      └─ [装载离线数据包] 正在解析离线地理包 [china.json] (3/8)...           │
│      ████████████████████░░░░░░░░░░░░░░░░   62%                            │
│                                                                            │
│  [✔] 就绪   按拍摄日期归档与规范重命名 (Date Archive)                      │
│      └─ [环境自检] 拍摄日期归档引擎与规范化重命名模板已就绪                 │
│                                                                            │
│ 已就绪: 2 / 3 个能力插件                                                   │
│ ⚙️ 系统正在并发自检环境与流式装载本地离线地理数据包，请稍候...             │
╰────────────────────────────────────────────────────────────────────────────╯
```

#### 主菜单：能力勾选与健康状态看板 (Workspace & Capabilities)
初始化完成后平滑过渡到主工作台，每个插件卡片深度融合自检环境与功能说明：
```text
当前工作区: /Users/vincent/Pictures/GPS
Inbox:  12 组   |  GPX:  2 个   |  已归档:  85 组

🧩 摄影处理流水线能力插件 (按 [1/2/3/4/空格] 切换勾选，按 [o] 调出当前插件设置，[s] 全局设置)：

 ▶ [✔]  P10 · 阶段 1   GPX 轨迹匹配与 GPS 修正 (GPX Matching)
     ├─ 环境自检:  ✔ 正常  ExifTool 核心引擎就绪 (v12.76)
     └─ 功能说明: 从 GPX 目录读取轨迹，为 RAW 写入经纬度并同步到 JPG/XMP

   [ ]  P15 (未激活)   GPS 智能邻近推断与时间插值 (GPS Interpolation)  推算窗口:15m
     ├─ 环境自检:  ✔ 正常  GPS 插值引擎就绪 (推算窗口: 15m, ExifTool v12.76)
     └─ 功能说明: 根据同批次前后邻近照片时间权重，自动推算补全无轨迹照片 GPS 坐标

   [✔]  P20 · 阶段 2   逆地理编码与地名元数据写入 (Reverse Geocode)
     ├─ 环境自检:  ✔ 正常  离线地理库就绪 (已加载 927,314 点位 / 8 个数据包，建树 0.28s)
     └─ 功能说明: 根据 GPS 坐标检索国家/省/市/区/POI，写入 IPTC/XMP 地名元数据

   [✔]  P100 · 阶段 3  按拍摄日期归档与规范重命名 (Date Archive)
     ├─ 环境自检:  ✔ 正常  拍摄日期归档引擎与规范化重命名模板已就绪
     └─ 功能说明: 提取 EXIF 拍摄日期，规范重命名并安全归档至 Processed/YYYY/MMDD/

⚙️ 会话配置已载入: ~/.config/photools/plugins.json (按 [o] 调整当前插件专属参数，按 [s] 进入全局设置)
─────────────────────────────────────────────────────────────────────────────────
 [1/2/3/4/空格] 切换勾选  [o] 插件设置  [s] 全局设置  [a] 全选  [c] 清空  [Enter] 预检执行  [r] 刷新  [q] 退出
```

- **[1/2/3/4]** 或 **[空格]**：切换对应能力插件开关；
- **[O]**：调出当前光标选中插件的**专属自描述设置面板**（如调整 P15 插值时间窗口、P10 Geosync、P100 In-Place 模式）；
- **[S]**：进入**全局环境与安全调度设置**（工作区路径、扁平模式、并发 Worker 数、无 GPS 容错策略、快照备份开关，支持 `[Tab]` 路径智能自动补全）；
- **[A]**：一键全选；**[C]**：一键清空；
- **[Enter]**：进入流水线参数全景看板与 Dry-Run 预检清单；再次按 `[Enter]` 正式执行；
- **[R]**：重新触发插件环境自检并刷新工作区资产。

---

### 2. 命令行（CLI）独立与复合模式

#### 2.1 复合流水线 (`pipeline`)
自由勾选开启或关闭任一能力组合：
```bash
# 执行全部流水线能力 (默认启用 GPX 匹配 + 逆地理 + 归档)
photools pipeline

# 开启 GPS 智能插值推算 (设定推算时间窗口为 1 小时)
photools pipeline --interpolate=true --interpolate-window=1h

# 启用扁平原地模式（原地扫描、原地逆地理打标并原地规范化重命名）
photools pipeline --flat=true --in-place=true

# 软降级容错模式（无 GPS 照片跳过地名打标直接安全归档）
photools pipeline --allow-no-gps=true
```

#### 2.2 独立子命令与测试辅助
- **标准 GPS 修正并归档 (`geotag`)**：
  ```bash
  photools geotag
  photools geotag -geosync +00:00:05 -workers 8 -test
  ```
- **独立离线逆地理地名打标 (`geocode`)**：
  ```bash
  photools geocode -dir /path/to/Inbox -test
  ```
- **独立按拍摄日期整理归档 (`organize-by-date`)**：
  ```bash
  photools organize-by-date -source-dir /path/to/source -target-dir /path/to/output -test
  ```
- **测试备份模式与一键还原 (`-test` & `restore-test`)**：
  ```bash
  # 在任何命令后附加 -test 即可在执行处理前自动快照备份 Inbox 到 Inbox_bak
  photools pipeline -test

  # 测试完毕后，一键将 Inbox_bak 还原回 Inbox，并清理测试产生的 Processed 归档产物
  photools restore-test -clean
  ```
- **离线地理数据包管理 (`geodata`)**：
  ```bash
  photools geodata list                   # 查看各大洲离线数据包状态
  photools geodata install all            # 一键下载并安装全球离线地名数据库
  photools geodata test 31.2304 121.4737  # 测试指定经纬度反查效果
  ```

---

### 3. Mac 原生图形工作台

轻量 SwiftUI 原生应用，用于可视化查看 `Inbox/GPX/Processed/Logs` 状态：
```bash
./script/build_and_run.sh
```

---

## 插件优先级配置文件 (`plugins.json`)

系统在首次启动时会自动在 `~/.config/photools/plugins.json` 生成默认配置：

```json
{
  "plugins": [
    {
      "id": "gpx_matching",
      "name": "GPX 轨迹匹配与 GPS 修正",
      "priority": 10,
      "enabled": true,
      "description": "从 GPX 目录读取轨迹为 RAW 写入经纬度并同步到 JPG/XMP"
    },
    {
      "id": "gps_interpolate",
      "name": "GPS 智能邻近推断与时间插值",
      "priority": 15,
      "enabled": true,
      "description": "根据同批次前后邻近照片时间权重，自动推算补全无轨迹照片 GPS 坐标",
      "options": {
        "window": "15m"
      }
    },
    {
      "id": "reverse_geocode",
      "name": "逆地理编码与地名元数据写入",
      "priority": 20,
      "enabled": true,
      "description": "根据 GPS 坐标检索国家/省/市/区/POI，写入 IPTC/XMP 地名元数据"
    },
    {
      "id": "date_archive",
      "name": "按拍摄日期归档与规范重命名",
      "priority": 100,
      "enabled": true,
      "description": "提取 EXIF 拍摄日期，规范重命名并安全归档至 Processed/YYYY/MMDD/"
    }
  ]
}
```

- **调度规则**：
  - 数值越小越优先执行；
  - **不同 Priority**：分属不同 Phase，按顺序严格串行推进；
  - **相同 Priority**：归入同一 Phase，在当前阶段内安全并发并行处理。

---

## 测试与性能基准 (Testing & Benchmarks)

### 1. 单元测试全覆盖
运行全量测试套件（100% 隔离闭环）：
```bash
go test -v ./...
```

### 2. 空间索引与智能插值基准性能 (Benchmarks)
在 Apple Silicon (M1 Max) 环境下执行基准测试：
```bash
go test -run=^$ -bench=. -benchmem ./internal/capabilities/gpsinterpolate/...
```

压测性能表现：
- **`AnchorIndex` 二分检索 (`10,000` 点位)**：`205 ns/op`，吞吐量高达 **`4,870,000 ops/sec`**（近 500 万 QPS）；
- **`AnchorIndex` 批次索引构建 (`1,000` 资产)**：`0.52 ms/op`，吞吐量达 **`1,920 批次/秒`**；
- **`ExecuteProcess` 单张插值与动态合入**：`0.11 ms/op`，吞吐量达 **`8,880 张/秒`**。

---

<h1 id="photo-processing-en">photo-processing (English)</h1>

An automated photo processing toolkit designed for photographer workflows, built on a **modular Capability Architecture** and a **Phased Priority Scheduler**.

### Core Capabilities:
1. **GPX Track Matching & Geotagging (`gpx_matching` · P10)**: Matches photos with GPS tracks exported from mobile devices, writes coordinates to RAW, and syncs to JPG/XMP.
2. **Offline Reverse Geocoding & Location Tagging (`reverse_geocode` · P20)**: Queries an offline 3D KD-Tree spatial index to embed Country, Province, City, District, and POI into IPTC/XMP tags.
3. **Date-based Normalization & Archiving (`date_archive` · P100)**: Extracts `DateTimeOriginal`, renames files with `YYYY-MM-DD`, and atomically moves companion files to `Processed/YYYY/MMDD/`.

📖 Documentation:
- **[Architecture & Design Document](docs/ARCHITECTURE_AND_DESIGN.md)**
- **[Configuration & Settings Design Document](docs/CONFIGURATION_AND_SETTINGS_DESIGN.md)**
- **[ExifTool I/O & Data Safety Design Document](docs/EXIFTOOL_IO_AND_SAFETY_DESIGN.md)**
