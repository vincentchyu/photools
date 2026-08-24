# PhotoTools 插件化流水线与能力解耦架构设计文档

本文档详细说明 `photools` 的系统分层架构、四大核心能力插件协议、基于 Priority 的分阶段同步屏障调度器、配置文件生命周期以及详尽的业务时序图。

---

## 1. 总体分层架构 (System Architecture)

```mermaid
flowchart TD
    subgraph Layer_UI["1. 交互与展示层 (Presentation Layer)"]
        CLI["CLI 命令行入口\n(photools geotag / geocode / pipeline / organize-by-date)"]
        TUI["TUI 交互工作台 (Bubble Tea)\n[✔] P10 GPX匹配\n[✔] P15 GPS插值推算\n[✔] P20 逆地理编码\n[✔] P100 拍摄日归档"]
    end

    subgraph Layer_Config["2. 配置与持久化层 (Configuration Layer)"]
        CFG["插件配置文件 (~/.config/photools/plugins.json)\n自动初始化 · 扩展 Options · 自定义 Priority · 自愈热重载"]
    end

    subgraph Layer_Orchestration["3. 流水线编排与调度层 (Orchestration Layer)"]
        Builder["流水线装配器 (PipelineBuilder)"]
        Scheduler["分阶段屏障调度器 (Phased Priority Scheduler)"]
        Pool["资产并发池 (WorkerPool)"]
        Ctx["资产流转上下文 (AssetContext)"]
    end

    subgraph Layer_Capabilities["4. 独立能力插件层 (Capability Plugins Layer)"]
        Cap1["能力 1: GPXMatchingCapability (P10)\nGPX 轨迹匹配 ➔ RAW GPS 写入 ➔ 二次校验 ➔ 同步 GPS 到 JPG/XMP"]
        Cap15["能力 1.5: GPSInterpolateCapability (P15) ✨\n同批次前后照片时间权重大圆插值 / 近邻机位推算 ➔ 补全 GPS 写入 RAW 并同步"]
        Cap2["能力 2: ReverseGeocodeCapability (P20)\n坐标提取 ➔ 3D KD-Tree 逆地理查询 ➔ 写入 IPTC/XMP 地名元数据"]
        Cap3["能力 3: DateArchiveCapability (P100)\n拍摄日期提取 ➔ 规范重命名 (YYYY-MM-DD) ➔ 整组原子归档移动"]
    end

    subgraph Layer_Engine["5. 核心引擎与底层执行层 (Engine & Drivers Layer)"]
        ExifDriver["ExifTool 交互驱动 (exiftool.CommandRunner)"]
        GeoEngine["离线逆地理空间索引 (geocoding.ReverseGeocoder - 3D KD-Tree)"]
        ArchiveEngine["文件安全归档引擎 (engine.Archiver)"]
    end

    CLI --> Builder
    TUI --> Builder
    CFG -.-> Builder
    Builder --> Scheduler
    Scheduler --> Pool
    Pool --> Ctx
    Ctx --> Cap1
    Ctx --> Cap15
    Ctx --> Cap2
    Ctx --> Cap3
    Cap1 --> ExifDriver
    Cap15 --> ExifDriver
    Cap2 --> GeoEngine
    Cap2 --> ExifDriver
    Cap3 --> ArchiveEngine
```

---

## 2. 领域模型与接口契约

### 2.1 能力插件接口 (`domain.Capability`)

```go
type Capability interface {
    // ID 返回插件全局唯一标识符 (如 "gpx_matching", "gps_interpolate", "reverse_geocode", "date_archive")
    ID() CapabilityID

    // Name 返回插件中文友好名称
    Name() string

    // Description 返回功能描述
    Description() string

    // RequiredStage 返回该能力对应的流水线阶段 (如 StageGeotag, StageGeocode, StageArchive)
    RequiredStage() PipelineStage

    // Priority 返回插件默认执行优先级 (数值越小越优先执行)
    Priority() int

    // Init 异步初始化自检与流式汇报 (如检查 ExifTool、加载 8 大洲 94 万离线点位并构建 KD-Tree)
    Init(ctx context.Context, report func(PluginInitReport)) error

    // PlanPrecheck 对单资产进行轻量预检 (Dry-Run)，评估依赖条件与动作描述 (严格区分阻塞 Warning 与良性跳过)
    PlanPrecheck(ctx context.Context, actx *AssetContext) CapabilityPlan

    // ExecuteProcess 执行具体业务逻辑 (写入 Exif、插值推算、查询地名、移动归档)
    ExecuteProcess(ctx context.Context, actx *AssetContext, sendEvent func(ProgressEvent)) error
}
```

### 2.2 插件自检汇报与健康模型 (`domain.PluginInitReport`)

```go
type PluginHealthStatus string

const (
    HealthReady    PluginHealthStatus = "ready"    // 正常就绪 (工具链完备，数据包已建树)
    HealthDegraded PluginHealthStatus = "degraded" // 降级运行 (如未装外挂数据包，降级到内置轻量库)
    HealthFailed   PluginHealthStatus = "failed"   // 初始化失败 (如关键命令行缺失)
)

type PluginInitReport struct {
    PluginID CapabilityID       `json:"plugin_id"`
    Name     string             `json:"name"`
    Stage    string             `json:"stage"`   // 当前步骤 (如 "装载离线数据包", "构建 KD-Tree")
    Message  string             `json:"message"` // 详细提示信息
    Percent  float64            // 0.0 ~ 1.0 (-1 表示不确定进度)
    Status   PluginHealthStatus // 当前健康状态
    Err      error              // 错误信息
}
```

### 2.3 资产流转上下文 (`domain.AssetContext`)

单个拍摄单元（同 basename 的 `RAW + JPG + XMP`）在整个流水线生命周期中共享同一个上下文对象，并可安全访问批次全景元数据（`Batch`），实现邻近照片时间权重计算：

```go
type AssetContext struct {
    mu sync.RWMutex

    // 拍摄单元物理文件
    Asset AssetGroup

    // 缓存的元数据与解析结果
    Metadata    Metadata
    HasGPS      bool
    Latitude    float64
    Longitude   float64
    Altitude    float64
    Location    *LocationInfo
    TargetDir   string
    NewBaseName string

    // 批次全量元数据引用 (用于前后邻近时间差空间插值)
    Batch []*AssetContext

    // 状态标记
    ModifiedFiles []string
    Skipped       bool
    SkipReason    string
}
```

---

## 3. 基于 Priority 的阶段同步屏障调度机制

### 3.1 调度规则
1. **分阶段分桶 (Priority Buckets)**：
   编排器将用户启用的所有 Capability 按 `Priority()` 升序分组为有序阶段列表：
   $$\text{Phase}_1 (P=10) \longrightarrow \text{Phase}_2 (P=15) \longrightarrow \text{Phase}_3 (P=20) \longrightarrow \text{Phase}_4 (P=100)$$
2. **串行阶段推进与屏障 (Stage Synchronization Barrier)**：
   全量照片必须在 $\text{Phase}_k$ 全部完成（或平滑流转/跳过）并通过阶段屏障后，调度器才会统一触发 $\text{Phase}_{k+1}$。
3. **多阶段平滑交接与容错降级**：
   - 阶段 1 未命中 GPX 轨迹时，平滑流转至阶段 2 进行时间权重空间插值；
   - 彻底无 GPS 的照片在开启 `--allow-no-gps` 时，在逆地理阶段良性跳过，安全进入阶段 4 按拍摄日期规范归档。
4. **同阶段安全并发**：
   同一个 Phase 内部如果配置了相同 Priority 的多个插件，将在当前阶段内对资产组协同并发执行。
5. **破坏性操作隔离**：
   归档移动插件（`DateArchiveCapability`）固定拥有最低优先级（$P=100$），确保绝不会在元数据清洗打标完成前提前移动文件。

---

## 4. 核心业务时序图 (Mermaid Sequence Diagrams)

### 4.1 启动阶段：插件并发自检与流式装载时序图

```mermaid
sequenceDiagram
    autonumber
    actor User as 用户
    participant TUI as TUI 工作台 (Bubble Tea)
    participant Cap1 as GPXMatching (P10)
    participant Cap15 as GPSInterpolate (P15)
    participant Cap2 as ReverseGeocode (P20)
    participant Cap3 as DateArchive (P100)
    participant GeoDB as 离线地理库 (KDTree)

    User->>TUI: 启动 photools / photools tui
    TUI->>TUI: 进入 stateInitializing 渲染加载进度界面
    par 并发异步自检装载
        TUI->>Cap1: Init(ctx, reportCb)
        Cap1->>Cap1: 检查 exiftool -ver
        Cap1-->>TUI: report(ExifTool 核心引擎就绪 v13.55, 100%)
    and
        TUI->>Cap15: Init(ctx, reportCb)
        Cap15->>Cap15: 检查推算窗口 (默认 15m) 与 ExifTool
        Cap15-->>TUI: report(GPS 插值引擎就绪, 100%)
    and
        TUI->>Cap2: Init(ctx, reportCb)
        Cap2->>GeoDB: 装载内嵌基础库 (10%)
        GeoDB-->>TUI: report("装载基础库", 10%)
        Cap2->>GeoDB: 扫描并解析 ~/.config/photools/geodata/*.json (20%~75%)
        GeoDB-->>TUI: report("解析离线包 [china.json]...", 35%)
        Cap2->>GeoDB: 构建 3D 球面 KD-Tree 空间索引 (85%)
        GeoDB-->>TUI: report("构建空间索引...", 85%)
        Cap2-->>TUI: report(离线地理库就绪 942k 点位 / 8 包, 100%)
    and
        TUI->>Cap3: Init(ctx, reportCb)
        Cap3-->>TUI: report(拍摄日期归档引擎就绪, 100%)
    end
    Note over TUI: 全部插件就绪后平滑过渡 (或按 Enter 立即进入)
    TUI->>TUI: 切换至 stateMenu 主工作台
```

---

### 4.2 TUI 插件选择与流水线动态装配时序图

```mermaid
sequenceDiagram
    autonumber
    actor User as 用户
    participant TUI as TUI 工作台 (Bubble Tea)
    participant Cfg as 配置管理器 (~/.config/photools/plugins.json)
    participant Builder as 流水线工厂 (PipelineBuilder)
    participant Orch as 编排器 (Orchestrator)

    TUI->>Cfg: LoadPluginsConfig("")
    Cfg-->>TUI: 返回插件配置 (Priority、Enabled、Options)
    TUI->>User: 渲染插件勾选清单 (展示 [P10 阶段1] [P15 阶段2] [P20 阶段3] [P100 阶段4] 及自检状态)
    User->>TUI: 按 [1/2/3/4/空格] 切换能力组合，按 [Enter]
    TUI->>Builder: Build(PipelineOptions{EnableGPX, EnableInterpolate, EnableGeocode, EnableArchive})
    Builder->>Cfg: 读取各插件 Priority 与 Options (如 window: 15m)
    Builder->>Orch: NewOrchestrator(按 Priority 升序划分为 Phase 1, Phase 2, Phase 3, Phase 4)
    Builder-->>TUI: 返回 Task 实例
    TUI->>Orch: Plan(ctx) 执行 Dry-Run 预检
    Orch-->>TUI: 返回 PlanResult (就绪/待补/预警清单)
    TUI->>User: 渲染预检计划清单，等待确认执行
```

---

### 4.3 全量资产分阶段同步屏障执行时序图

```mermaid
sequenceDiagram
    autonumber
    participant TUI as TUI / 日志流
    participant Orch as 调度编排器 (Orchestrator)
    participant Pool as 并发处理池 (WorkerPool)
    participant Cap1 as 阶段 1: GPX 匹配 (P10)
    participant Cap15 as 阶段 2: GPS 插值 (P15)
    participant Cap2 as 阶段 3: 逆地理编码 (P20)
    participant Cap3 as 阶段 4: 日期归档 (P100)

    TUI->>Orch: Execute(ctx, eventCh)
    Orch->>TUI: [StageDiscover] 扫描 Inbox 发现 N 组照片资产
    
    rect rgb(235, 248, 255)
        note over Orch,Cap1: 🚀 进入 阶段 1 (Phase 1 · Priority 10)
        Orch->>Pool: 并发分发全量 AssetContext 执行 Cap1 (GPX 轨迹匹配)
        Pool->>Cap1: ExecuteProcess(RAW 写入 GPS ➔ 二次校验 ➔ 同步 GPS 到 JPG/XMP)
        Cap1-->>Pool: 命中写入成功 / 轨迹未命中(平滑转交下游)
        Pool-->>Orch: 全量资产 Phase 1 执行完毕
        Orch->>TUI: 🚦 【阶段同步屏障 1】GPX 轨迹匹配完成，通过屏障！
    end

    rect rgb(254, 249, 195)
        note over Orch,Cap15: ✨ 进入 阶段 2 (Phase 2 · Priority 15)
        Orch->>Pool: 并发分发未获 GPS 资产执行 Cap15 (智能时间插值推算)
        Pool->>Cap15: ExecuteProcess(计算同批次前后锚点 ➔ 空间线性插值 ➔ 写入 RAW 并同步)
        Cap15-->>Pool: 成功补全坐标 / 窗口超限(允许降级)
        Pool-->>Orch: 全量资产 Phase 2 执行完毕
        Orch->>TUI: 🚦 【阶段同步屏障 2】GPS 智能推算完成，通过屏障！
    end

    rect rgb(240, 253, 244)
        note over Orch,Cap2: 🧭 进入 阶段 3 (Phase 3 · Priority 20)
        Orch->>Pool: 并发分发资产执行 Cap2 (逆地理编码地名打标)
        Pool->>Cap2: ExecuteProcess(3D KD-Tree 地名检索 ➔ 写入 IPTC/XMP 地名)
        Cap2-->>Pool: 返回地名打标完成 (中国 · 新疆 · 特克斯)
        Pool-->>Orch: 全量资产 Phase 3 执行完毕
        Orch->>TUI: 🚦 【阶段同步屏障 3】地名元数据写入完成，通过屏障！
    end

    rect rgb(254, 242, 242)
        note over Orch,Cap3: 📦 进入 阶段 4 (Phase 4 · Priority 100 - 终态归档)
        Orch->>Pool: 并发分发资产执行 Cap3 (日期归档移动)
        Pool->>Cap3: ExecuteProcess(拍摄日提取 ➔ YYYY-MM-DD 重命名 ➔ 移动整组文件)
        Cap3-->>Pool: 归档到 Processed/YYYY/MMDD/ (原子移动)
        Pool-->>Orch: 全量资产 Phase 4 执行完毕
        Orch->>TUI: 🏁 【流水线完成】全部阶段执行完毕，输出汇总与待处理报告！
    end
```

---

### 4.4 单拍摄单元 (RAW + JPG + XMP) 底层执行时序图

```mermaid
sequenceDiagram
    autonumber
    participant Orch as 编排器 (Orchestrator)
    participant Actx as 资产上下文 (AssetContext)
    participant Cap as Capability 插件
    participant Exif as ExifTool 执行驱动
    participant FS as 物理文件系统

    Orch->>Actx: 初始化 NewAssetContext(RAW+JPG+XMP, Batch)
    
    note over Cap,Exif: 1. GPX 轨迹匹配阶段 (P10)
    Cap->>Exif: ReadMetadata(RAW)
    Exif-->>Actx: 缓存 DateTimeOriginal, OffsetTimeOriginal
    Cap->>Exif: WriteGeotag(RAW, gpxFiles, geosync)
    Cap->>Exif: ReadMetadata(RAW) 二次校验 GPSPosition
    Cap->>Actx: SetGPS(lat, lon) ➔ 同步 GPS 到 JPG/XMP

    note over Cap,Exif: 2. GPS 智能时间插值推算阶段 (P15)
    opt 未命中 GPX 轨迹但启用了智能推算
        Cap->>Actx: 检索 Batch 中前后时间差 <= window 的 GPS 锚点
        Cap->>Cap: 计算时间权重球面大圆经纬度/海拔空间插值
        Cap->>Exif: WriteCoordinates(RAW, lat, lon, alt)
        Cap->>Actx: SetGPS(lat, lon) ➔ 同步 GPS 到 JPG/XMP
    end
    
    note over Cap,Exif: 3. 逆地理编码阶段 (P20)
    Cap->>Actx: 获取 Latitude, Longitude
    Cap->>Exif: WriteLocation(RAW, LocationInfo) [纯地名标签]
    Cap->>Exif: SyncLocationToJPG(RAW ➔ JPG) [纯地名标签]
    Cap->>Exif: SyncLocationToXMP(RAW ➔ XMP) [纯地名标签]

    note over Cap,FS: 4. 拍摄日期归档阶段 (P100)
    Cap->>Actx: 获取 DateTimeOriginal
    Cap->>FS: 计算目标目录 Processed/YYYY/MMDD/ 并检查冲突
    Cap->>FS: 原子移动 RAW, JPG, XMP 至目标目录并规范化重命名
```

---

## 5. 配置文件生命周期与规范 (`plugins.json`)

### 5.1 存储路径与自愈机制
- **默认全局路径**：`~/.config/photools/plugins.json`
- **环境变量覆盖**：`PHOTOOLS_PLUGINS_CONFIG`
- **增量自愈机制**：系统冷启时自动比对当前文件与内置默认配置，若缺少新插件（如 `gps_interpolate`）或缺少扩展 `options`（如 `window: "15m"`），将自动增量补齐并重新排序写回磁盘，同时完好保留用户已自定义的优先级数值与开关。

### 5.2 配置文件完整结构示例
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
