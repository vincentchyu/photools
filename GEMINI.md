# 📸 photools 项目持久化记忆与核心架构规约 (GEMINI.md)

本文档作为 AI Agent 在本工作区（photools 代码库根目录）的**唯一核心持久化记忆、技术规范与架构约束文件（Single Source of Truth）**。后续所有新增功能、架构演进、代码重构或测试均必须严格遵守本文档。

---

## 1. 核心定位与用户画像

- **用户画像**：Golang 程序员与尼康单反/微单摄影师；
- **系统核心定位**：专注于相机 RAW/JPG 照片导入电脑后的 **GPS 轨迹匹配、智能时间插值推算、离线高精逆地理编码中文地名清洗与规范化拍摄日期归档**；
- **语言规约**：所有交互、思维推理、终端输出、用户日志与异常提示**统一强制使用简体中文**；
- **系统边界与非目标**：
  - 本系统不是完整 DAM，不管理全部摄影资产生命周期；
  - 本系统不是后期调色软件，不承担 Lightroom/Capture One 等工具的职责；
  - 默认不包含云同步、评分筛选、非 ExifTool 独立 Exif 写入等脱离核心定位的功能。

---

## 2. 摄影资产模型与元数据写入规则

- **资产主文件优先模型 (Primary Asset Model)**：
  - 同 basename 的配套文件是一个独立的拍摄单元；
  - 核心操作遵循 **“主文件优先（PrimaryPath）”** 抽象：若存在 `RAW`，以 `RAW` 为主决策源并自动同步至伴随的 `JPG` 与 `XMP`；若仅有单 `JPG` 或单 `RAW`，该文件本身即作为完整主文件，平等享受 GPX 匹配、GPS 插值、逆地理与归档能力；
  - 若存在同 basename 的伴随文件（`XMP`、`ACR`、`WAV` 等），作为伴随文件整体同步维护并一同归档。
- **时间与轨迹匹配**：
  - 默认信任照片内的 `DateTimeOriginal` 与 `OffsetTimeOriginal`；
  - 默认 `geosync=0`，但必须支持显式时间偏移补偿；
  - 多个 `GPX` 文件必须逐个传参，禁止拼接成无效字符串。
- **写入顺序与二次校验**：
  - 双格式时必须先写主文件 `RAW`，再同步至 `JPG` 和 `XMP`；单文件时直接写入主文件并同步 `XMP`；
  - 写入成功判定**绝不能仅依赖 ExifTool 退出码**，必须包含二次读取校验，至少确认主文件上存在有效 `GPSPosition`。

---

## 3. 四大能力插件与分阶段屏障调度器 (Capability Architecture)

系统采用接口解耦的插件化架构，严禁将 GPX 匹配、推算、逆地理、归档杂糅在同一个单体函数中。

### 3.1 四大核心能力插件
1. **`gpx_matching` (Priority 10 · 阶段 1)**:
   - 依赖 ExifTool 进行 GPX 轨迹时间轴精准匹配；
   - 写入 RAW 经纬度并二次校验，无污染同步至伴随 JPG/XMP。
2. **`gps_interpolate` (Priority 15 · 阶段 2)** ✨:
   - **极速时间分桶与二分查找索引 (`AnchorIndex`)**：
     - 内存按拍摄日期 `YYYY-MM-DD` 分桶维护时间有序锚点切片，通过 `sort.Search` 二分查找在 $O(\log K)$ 纳秒级（~200ns）定位前后最邻近机位；
     - 批次首轮统一提取拍摄时间与坐标，彻底杜绝在检索循环中重复启动外部 ExifTool 子进程；
     - 双向锚点采用球面大圆时间权重线性插值，单向锚点采用同机位近邻继承，推算后动态合入索引供同批次后序照片自愈继承；
     - 默认推算窗口 15m（支持 `--interpolate-window` 自定义），开启 `--allow-no-gps` 时超出窗口安全跳过而不阻断流水线。
3. **`reverse_geocode` (Priority 20 · 阶段 3)**:
   - 基于 3D KD-Tree 离线高精空间索引；
   - 写入国家、省份、城市、区县、风景区 POI 中文元数据（IPTC/XMP），支持 IPTC Extension 完整结构化位置模型 (`LocationCreated` / `LocationShown`) 与 Lightroom 分层关键词标签树 (`XMP-lr:HierarchicalSubject`)；
   - **离线地理数据包终端安装规约**：为保证地理数据库来源准确性、网络完整性与数据源校验，离线数据包严禁在 GUI 客户端内隐式下载安装，必须由使用者在终端（CLI）中显式执行 `photools geodata install <target>` 安装。GUI 仅负责状态探测、命令复制与 3D KD-Tree 坐标反查测试。
4. **`date_archive` (Priority 100 · 阶段 4)**:
   - 依据原始拍摄日期规范化重命名；
   - 原子安全归档至 `Processed/YYYY/MMDD/`（破坏性移动插件，必须具备最低优先级），完整保留 `.nef.xmp` / `.jpg.xmp` 复合侧车后缀。

### 3.2 阶段流转、调度器与性能契约
- **多阶段平滑交接与就绪判定**：
  - Dry-Run 预检判定：前序插件良性跳过（如已有 GPS 跳过插值）绝不阻断后序插件（如逆地理写入），只要整条链路无阻塞 Warning 且包含有效执行阶段，资产即判定为 `ReadyCount` 就绪；
  - 阶段 1 未命中 GPX 轨迹时，若流水线配置了阶段 2 插值插件，编排器平滑交接，严禁提前硬熔断。
- **全生命周期 `sync.Once` 与零冗余进程机制**：
  - 各插件 `Init()` 强制采用 `sync.Once` 缓存环境自检与 ExifTool 探针结果，预检与执行阶段毫秒级直通；
  - 离线地理数据包与 KD-Tree 建树全局单例装载一次，杜绝重复 I/O 与建树开销；
  - **ExifTool Stay-Open 常驻守护进程池 (`DefaultRunner`)**：生产环境默认启用常驻进程池（`StayOpenPool`），通过 `exiftool -stay_open True -@ -` 消除重复 `fork/exec` 子进程开销，单次读取开销从 ~30ms 骤降至 1~2ms，并具备进程崩溃自愈与优雅退出能力。
- **软降级容错 (`--allow-no-gps`)**：彻底无 GPS 照片在逆地理阶段良性跳过（不产生阻塞 Issue），安全进入阶段 4 按拍摄日期规范归档；
- **并发与待处理报告**：按 basename 资产组并发处理，处理中断或待补资产自动生成详尽的 Markdown 原因清单（默认 `~/.logs/photools/inbox_pending_report_latest.md`）；
- **全量实时中文日志流落盘 (Real-Time Log Streaming)**：
  - 流水线执行期间产生的全部中文实时事件流（含插件自检、阶段流转、每张照片推算/打标/归档进度与异常）强制实时流式落盘至用户主目录全局日志中心 `~/.logs/photools/photools_latest.log` 与带时间戳的 `~/.logs/photools/photools_YYYYMMDD_HHMMSS.log`，彻底杜绝在照片工作区目录产生 `Logs/` 垃圾文件；
  - 采用毫秒级带时区时间戳格式（`[HH:MM:SS.mmm] [LEVEL] [阶段] 消息`），并在任务结算时自动追加 Execution Summary 与异常清单；
- **扁平原地模式与就地保存 (Flat Mode & In-Place)**：支持忽略传统 `Inbox/` -> `Processed/YYYY/MMDD/` 分层，直接指定源目录就地扫描、就地逆地理/打标签、并原地规范化重命名，且同一目录下重命名严格免自冲突；
- **智能元数据分层模型与四档策略 (`SidecarPolicy` · `--sidecar-policy`)** ✨：
  1. `smart`（默认/推荐 · 智能分层模式）：
     - **第二层修正事实 (GPS/时间修正)**：写入 RAW EXIF 头部 + 伴随 JPG 内嵌 + 同步 `.xmp` 侧车（附带 Photools 溯源指纹）；
     - **第三层派生信息 (中文地名/分层标签)**：RAW 严格只读仅写 `.nef.xmp` 侧车，JPG 交付格式直接内嵌写入；
     - 黄金平衡点：RAW 永久拥有标准 GPS 事实，地名由 XMP 承载，JPG 跨设备即开即看。
  2. `sidecar_only`（纯 XMP 侧车模式）：RAW 与 JPG 均不触碰原图，所有修正与派生元数据严格输出为独立 `{file}.xmp` 侧车；
  3. `embed_and_sidecar`（双写同步模式）：直接修改原图内嵌 EXIF，同时维护配套的 `.xmp` 侧车文件；
  4. `embed_only`（纯原图内嵌模式）：直接修改原图内嵌 EXIF，不产生或更新任何 `.xmp` 侧车文件。
- **伴随文件扩展名白名单 (`CompanionExtensions` · `--companion-exts`)** ✨：支持针对 RAW/JPG 主文件配套的伴随文件（如 `wav` 录音、`acr` 调色、`exf`、`xmp` 等）进行拓扑发现、原子同步流转与规范重命名归档；
- **统一设置抽象与会话覆盖 (Session & Plugin Settings)**：
  - 核心配置由 `internal/config/schema.go` 统一定义，全局参数与插件专属参数清晰正交解耦；
  - TUI 中按 `[s]` 调出全局设置，光标选中插件按 `[o]` 调出插件专属设置，输入路径支持 `[Tab]` 智能补全与多候选轮转；
  - 运行时基于 `SessionConfig` 动态覆盖，支持 `Enter` 会话生效与 `Ctrl+S` 持久化写入 `~/.config/photools/plugins.json`。
- **插件自描述与自配置契约 (Self-Describing & Configurable Capability)**：
  - 插件接口强制实现 `SupportedOptions() []OptionSpec` 与 `Configure(opts map[string]any) error`；
  - 插件自身完全内聚掌管自己的可配置选项、默认值、说明与预设候选值（Choices）；
  - TUI 界面与 Pipeline 构建器基于元数据纯动态驱动，彻底杜绝针对具体插件的外部硬编码 `switch-case`。

---

## 4. 核心工具包单元测试 100% 闭环规范 (AI 必遵)

所有核心底层工具包（`internal/exiftool`、`pkg/geocoding`、`internal/engine`、`pkg/geodata`、`internal/pipeline` 等）必须提供 **100% 完整的对外接口测试闭环**：

1. **导出函数全覆盖**：所有首字母大写的 Exported 函数必须在 `*_test.go` 中有直接对应的单元测试，严禁只测辅助函数漏测核心业务函数；
2. **测试隔离性与 Mock**：外部系统调用（如 `exiftool` 二进制执行、网络下载）必须通过 Runner/Client 抽象注入 Mock，`go test ./...` 必须在纯净环境中秒级通过；
3. **元数据防污染断言**：
   - GPS 写入/同步函数必须断言**仅操作 `-GPS:*`**，绝不能混入 IPTC/Photoshop 地名标签；
   - 逆地理地名写入函数必须断言**仅操作地名标签**，绝不能混入 GPS 坐标标签。
4. **全分支边界覆盖**：覆盖成功路径、空入参安全返回、格式畸变、错误分类（`ClassifyFailure`）等全部分支。

---

## 5. 主动式零技术债务与工程纪律 (AI 必遵)

1. **严禁硬编码魔法值 (No Magic Values/Strings)**：
   - 严禁在业务代码、插件逻辑或测试中散落硬编码字符串（如插件 ID、GPS 来源、推算算法、侧车策略、阶段名称、日志级别、默认扩展名等）；
   - **全局枚举与通用常量必须统一定义在根目录的 `common` 包中**（`common/enums.go`），其他模块（如 `internal/domain`、各能力插件等）通过引用或类型别名（Type Alias）使用，保证全局唯一事实源；
2. **禁止被动等待发现**：AI 助手在每次架构调整或功能演进时，必须主动全库扫描死代码、过时包与残留引用，主动清理与重构；
3. **历史废弃包物理移除**：一旦新架构方案落地并验证无误，被替代的旧包必须立即物理删除；
4. **CLI 与 Shell Tab 自动补全 100% 同步**：新增、修改或弃用任何 CLI 命令或参数时，必须同步增量维护 `internal/completion/`，确保 Zsh、Bash、Fish 补全脚本与 CLI 保持一致。

---

## 6. 未来新增/重构能力插件标准开发闭环规范 (Plugin Development SOP)

未来任何时候为 photools 扩展或重构能力插件，AI Agent **必须无条件、按序且 100% 完整执行以下 7 步闭环流程**：

1. **`common/enums.go` & `internal/domain/capability.go`**：在 `common` 注册 `CapabilityID` 常量并实现 `Capability` 接口（包含 `SupportedOptions()` 与 `Configure()`）；
2. **`internal/capabilities/<plugin>/`**：在能力包内部实现业务逻辑、自描述配置契约 `SupportedOptions()` 与 `Configure()`，并提供独立单元测试；
3. **`internal/config/plugins.go`**：在 `defaultMetas` 注册并实现 `~/.config/photools/plugins.json` 自动自愈迁移逻辑，同步更新 `plugins_test.go`；
4. **`internal/pipeline/`**：在 `builder.go` 接入 `PipelineOptions`，并在 `orchestrator.go` 编排阶段流转与降级容错；
5. **`cmd/photools/` & `internal/completion/`**：增加 CLI 选项，并**100% 同步维护 Zsh / Bash / Fish Tab 自动补全脚本**；
6. **`internal/tui/`**：在 TUI 注册插件项、分配快捷键（如 `[1/2/3/4]`）、光标空格勾选，由自描述 Schema 自动渲染专属设置面板，同步更新 `model_test.go`；
7. **规约与持久化文档同步**：同步更新 `GEMINI.md` 与 `README.md`。

---

## 7. macOS 原生客户端与跨语言 (C-Shared FFI) 开发规约

1. **架构文档**：所有 macOS 相关的系统分层、C ABI 导出符号、内存管理契约、EXIF 解析流水线与 UI/UX 规范统一维护在 [`docs/MACOS_CLIENT_TECHNICAL_DESIGN.md`](docs/MACOS_CLIENT_TECHNICAL_DESIGN.md)；
2. **双引擎直通与降级**：
   - 优先通过 `libphotools.dylib` 进行进程内 C-Shared FFI 极速直通（~0.1ms，单例常驻）；
   - 缺失动态库时必须平滑无缝降级至 CLI 异步子进程模式，严禁界面崩溃；
3. **C-ABI 内存安全绝对纪律**：
   - 所有 Go 端 `C.CString` 返回给 Swift 的指针，Swift 端必须通过 `defer { fnFreeString?(ptr) }` 释放，严禁内存泄漏或野指针重复释放；
4. **主资产与排序**：
   - 资产列表必须以 `PhotoAssetGroup`（配套单元组）为核心展示单位；
   - 扫描阶段必须提取主文件 `fileModificationDate`，默认提供时间升序/降序及文件名升序/降序多维排序。

---

## 8. Homebrew Tap 双通道分发与维护规约 (vincentchyu/tap)

1. **双通道架构与分工**：
   - **默认通道 (`Formula/photools.rb`)**：`brew install vincentchyu/tap/photools`，面向开发者与全键盘终端用户。声明 `depends_on "exiftool"` 自动解决核心底层依赖，在 `install` 阶段自动生成并注入 Zsh、Bash、Fish 自动补全脚本；
   - **桌面通道 (`Casks/photools.rb`)**：`brew install --cask vincentchyu/tap/photools`，面向桌面摄影师。自动拉取 `photools-macOS.dmg` 并将 `PhotoolsApp.app` 安装至 `/Applications`；
2. **自动化发布闭环**：
   - 发布新版本时执行 `./script/release_homebrew.sh <tag>` 自动计算源码 Tarball 与 DMG 的 SHA256 并更新 Formula/Cask；
   - 配合 GitHub Actions (`.github/workflows/homebrew-release.yml`) 自动同步推送到 `vincentchyu/homebrew-tap` 仓库；
   - **动态 Git Tag 注入与 BuildInfo 兜底**：编译期通过 `-ldflags "-s -w -X 'github.com/vincentchyu/photools/common.CurrentVersion=$(git describe --tags --always)' -X 'main.Version=$(git describe --tags --always)'"` 动态注入版本，结合 `runtime/debug.ReadBuildInfo()` 智能兜底。

---

## 9. 全局多语言 (i18n) 统一字典与运行时规约

详细架构与开发规约见独立技术文档：[`docs/I18N_ARCHITECTURE_GUIDE.md`](docs/I18N_ARCHITECTURE_GUIDE.md)。

1. **单一事实源 (Single Source of Truth) JSON**：
   - 全局字典统一定义在 `locales/zh-CN.json`（简体中文）与 `locales/en-US.json`（英文）中；
   - Go 端通过 `locales/locales.go` 的 `//go:embed *.json` 静态内嵌，运行时零额外 I/O 损耗；
   - Swift macOS 端共用相同键名模型并由 `LanguageManager.swift` 维护；
2. **全链路运行时事件与控制台日志 100% 本地化**：
   - 流水线调度器、阶段屏障流转、四大能力插件内部执行日志、异常诊断建议与落盘日志（`~/.logs/photools/photools_latest.log`）全部基于 `i18n.T` 动态本地化；
   - 阶段名称通过 `domain.StageDisplayName` 动态投影，彻底杜绝日志中硬编码中文字符；
3. **默认中文优先、热切换与自动化守卫测试**：
   - 默认模式符合核心摄影师用户画像（`zh-CN` 优先），支持 CLI `--lang [zh|en]`、TUI 快捷键 `[l]`、macOS 设置面板无缝热切换；
   - 强制运行 `internal/i18n/i18n_guard_test.go` 验证中英文字典 100% 镜像对称与纯英文模式下零中文字符泄漏。

---

## 10. 本地开发环境与 Go 工具链执行规约 (asdf & Go Runtime Protocol)

1. **Golang 版本管理体系 (asdf)**：
    - 使用者的 Go 环境通过 `asdf` 进行多版本管理；
    - 当前项目活跃使用的 Go 版本为 `1.27.1`（二进制路径：`/Users/vincent/.asdf/installs/golang/1.27.1/go/bin/go`）；
    - 可用版本池包括：`1.24.3`、`1.25.1`、`1.26.2`、`1.27.1`；
2. **沙盒与终端命令执行要求**：
    - 默认沙盒环境下直接调用系统 PATH 中的 `go` 可能会因沙盒文件访问隔离提示无权限（`Operation not permitted`）；
    - AI Agent 在执行 `go test`、`go build`、`go mod` 等 Go 相关命令时，知晓宿主机的真实 Go 二进制位于
      `/Users/vincent/.asdf/installs/golang/1.27.1/go/bin/go`，并在需要外部 Go 工具链时显式请求沙盒穿透（
      `BypassSandbox: true`）或使用绝对路径。



