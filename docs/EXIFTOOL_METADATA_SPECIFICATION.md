# 📸 photools 元数据写入与 ExifTool 标签字典全景规范 (Metadata Specification)

本文档面向高级摄影师、数字资产管理（DAM）极客以及对照片元数据有严格规范要求的专业用户，全面揭示 **photools** 在底层通过 **ExifTool** 对照片（RAW / JPG）与 XMP 侧车文件进行元数据读写、修改与生成的完整清单、分层模型与可追溯性技术规范。

---

## 1. 核心设计哲学：按元数据性质分层 (Metadata Intent Model)

photools 遵循 **Metadata Working Group (MWG)** 国际元数据工作组规范，创新性地提出了 **按元数据性质分层（Metadata Tiering）** 的写入策略哲学，彻底化解“RAW 资产不可破坏性”与“补全摄影事实最大化生态兼容”之间的传统矛盾：

```
                    摄影元数据 (Metadata)
                              │
         ┌────────────────────┼────────────────────┐
         ↓                    ↓                    ↓
  第一层: 原始摄影事实   第二层: 修正事实      第三层: 派生地理 & 第四层: 工作流
   (Original Fact)     (Corrected Fact)       (Derived / Workflow)
         │                    │                    │
   相机参数/快门/光圈      GPX匹配GPS/插值GPS       离线逆地理中文地名/POI
   原始拍摄时间/内建GPS    时区偏差/时间补偿        Lightroom分层关键词树/评分
         │                    │                    │
         ↓                    ↓                    ↓
      RAW 严格保留          RAW EXIF 头部          NEF.XMP 侧车
                          伴随 JPG 内嵌           伴随 JPG 内嵌
                          NEF.XMP 同步           (绝不触碰 RAW)
```

### 1.1 四档写入策略体系 (`SidecarPolicy`)

| 策略标识 | 策略名称 | 修正事实 (GPS/时间修正) | 派生信息 (中文地名/分层标签) | 说明与适用场景 |
| :--- | :--- | :--- | :--- | :--- |
| **`smart`** *(默认/推荐)* | **智能分层模式** | **RAW 内嵌 + JPG 内嵌 + XMP 同步** (附溯源指纹) | **RAW 严格只读写 XMP + JPG 内嵌** | **黄金平衡点**：RAW 获得永久标准 GPS 事实，地名由 XMP 承载，JPG 即开即看 |
| **`sidecar_only`** | **纯 XMP 侧车模式** | 仅写入 `.xmp` 侧车 | 仅写入 `.xmp` 侧车 | 零触碰原图，适合严苛归档要求 |
| **`embed_and_sidecar`** | **原图与 XMP 双写同步** | RAW 与 JPG 均内嵌修改 | RAW 与 JPG 均内嵌修改 + 同步 XMP | 全量镜像模式 |
| **`embed_only`** | **纯原图内嵌模式** | RAW 与 JPG 均内嵌修改 | RAW 与 JPG 均内嵌修改 (不产生 XMP) | 极简模式 |

---

## 2. ExifTool 写入与生成的标签全景字典

### 2.1 GPS 轨迹与机位坐标 (GPS & Coordinates)

| 命名空间 (Namespace) | 标签名 (Tag Name) | 写入时机 / 插件 | 说明与格式示例 | 写入目标 |
| :--- | :--- | :--- | :--- | :--- |
| `GPS` (EXIF IFD 0x8825) | `GPSVersionID` | GPX匹配 / GPS插值 | `2.3.0.0` (标准 GPS 版本) | RAW / JPG (内嵌) |
| `GPS` | `GPSLatitude` & `GPSLatitudeRef` | GPX匹配 / GPS插值 | 如 `23.658583 N` (六位小数高精纬度) | RAW / JPG (内嵌) |
| `GPS` | `GPSLongitude` & `GPSLongitudeRef` | GPX匹配 / GPS插值 | 如 `116.631917 E` (六位小数高精经度) | RAW / JPG (内嵌) |
| `GPS` | `GPSAltitude` & `GPSAltitudeRef` | GPX匹配 / GPS插值 | 如 `23.00 m`，Ref: `0` (高于海平面) | RAW / JPG (内嵌) |
| `GPS` | `GPSDateStamp` & `GPSTimeStamp` | GPX匹配 | UTC 拍摄时间戳 (如 `09:34:05.67`) | RAW / JPG (内嵌) |
| `GPS` | `GPSMapDatum` | GPX匹配 / GPS插值 | `WGS-84` (全球通用空间参考基准) | RAW / JPG (内嵌) |
| `XMP-exif` | `GPSLatitude` | 智能分层 / 同步 | `23,39.5154N` 或 `23.658583 N` | `.xmp` 侧车文件 |
| `XMP-exif` | `GPSLongitude` | 智能分层 / 同步 | `116,37.9181E` 或 `116.631917 E` | `.xmp` 侧车文件 |
| `XMP-exif` | `GPSAltitude` & `GPSAltitudeRef` | 智能分层 / 同步 | `23/1` (有理数分数表示) | `.xmp` 侧车文件 |
| `XMP-exif` | `GPSDateTime` | GPX匹配 / 同步 | 带毫秒/亚秒 UTC 时间 | `.xmp` 侧车文件 |

---

### 2.2 Photools 专属可追溯性指纹与审计标签 (Provenance & Audit)

当 Photools 为照片补全或推算元数据时，在 XMP 侧车中自动注入专属的审计指纹，明确区分“相机原始自带”与“算法推算补全”：

| 命名空间 | 标签名 | 示例值 | 语义与可追溯性作用 |
| :--- | :--- | :--- | :--- |
| `XMP-photools` | `GPSSource` | `gpx` / `interpolated` / `original` / `manual` | **坐标来源溯源**：标记是通过轨迹匹配还是时间插值补全 |
| `XMP-photools` | `GPSMatchMethod` | `time_proximity` / `spherical_linear_interpolation` / `nearest_neighbor_anchor` | **推算算法**：球面线性插值还是同机位近邻继承 |
| `XMP-photools` | `InterpolateWindow` | `15m` / `30m` / `1h` | **推算时间容差窗口** |
| `XMP-photools` | `Processor` | `photools v1.2.0` | **处理引擎与版本号** |
| `XMP-photools` | `ProcessedDate` | `2026-08-29T14:30:00+08:00` | **处理发生时间戳 (ISO 8601)** |
| `XMP-xmp` | `CreatorTool` | `photools v1.2.0` | **标准 XMP 创建工具标识** |
| `XMP-xmp` | `MetadataDate` | `2026-08-29T14:30:00+08:00` | **标准 XMP 最后元数据修改时间** |

---

### 2.2 逆地理编码中文地名 (Reverse Geocoding & Locations)

系统采用 **IPTC IIM + XMP-photoshop + XMP-iptcCore + IPTC Extension 四层元数据立体对齐**，彻底杜绝字段孤岛：

| 命名空间 | 标签名 | 示例值 | 语义说明 |
| :--- | :--- | :--- | :--- |
| `XMP-photoshop` | `Country` | `中国` | 国家全称 |
| `IPTC` | `Country-PrimaryLocationName` | `中国` | IPTC 传统国家名称 |
| `XMP-iptcCore` | `CountryCode` | `CN` | ISO 3166-1 alpha-2 标准二字码（Lightroom / XMP 核心） |
| `IPTC` | `Country-PrimaryLocationCode` | `CHN` | IPTC 传统国家代码（严格遵循 ISO 3166-1 alpha-3 三字码，digiKam / 传统 IPTC 规范） |
| `XMP-photoshop` | `State` | `广东省` | 省份 / 州 / 直辖市 |
| `IPTC` | `Province-State` | `广东省` | IPTC 传统省份 |
| `XMP-photoshop` | `City` | `潮州市` | 地级市 / 县级市（规范清洗，非区县） |
| `IPTC` | `City` | `潮州市` | IPTC 传统城市 |
| `XMP-iptcCore` | `Location` | `吉利村` / `湘桥区` | 区县、行政村或风景区 POI 名称 |
| `IPTC` | `Sub-location` | `吉利村` / `湘桥区` | IPTC 传统次级地点 / 拍摄点 |

> [!NOTE]
> **国家代码标准体系分流（Alpha-2 vs Alpha-3）**：
> - **GeoNames 数据库** 原始返回 ISO 3166-1 Alpha-2 二字码（如 `CN`, `US`, `JP`）。
> - **XMP 命名空间 (`XMP-iptcCore`, `XMP-iptcExt`)** 严格遵循 ISO 3166-1 Alpha-2 二字码标准。
> - **IPTC IIM 传统字段 (`IPTC:Country-PrimaryLocationCode`)** 规范定义为 ISO 3166-1 Alpha-3 三字码（如 `CHN`, `USA`, `JPN`）。
> - photools 内部建立双向标准转换层，保证在 digiKam、Lightroom Classic、Capture One 与 Nikon NX Studio 间实现 100% 正确互操作与原样呈现。

---

### 2.3 IPTC Extension 完整结构化位置 (Location Structure)

为专业图库与新闻摄影管理系统（如 Photo Mechanic、Lightroom Classic 复杂元数据面板）提供最高等级的结构化对象：

| 命名空间 | 结构体字段 | 示例值 | 语义角色 |
| :--- | :--- | :--- | :--- |
| `XMP-iptcExt` | `LocationCreatedCountryName` | `中国` | 相机拍摄机位所在国家 |
| `XMP-iptcExt` | `LocationCreatedCountryCode` | `CN` | 相机拍摄机位国家代码 |
| `XMP-iptcExt` | `LocationCreatedProvinceState`| `广东省` | 相机拍摄机位所在省份 |
| `XMP-iptcExt` | `LocationCreatedCity` | `潮州市` | 相机拍摄机位所在城市 |
| `XMP-iptcExt` | `LocationCreatedSublocation` | `吉利村` | 相机拍摄机位具体 POI / 区县 |
| `XMP-iptcExt` | `LocationShownCountryName` | `中国` | 画面主体所在国家 |
| `XMP-iptcExt` | `LocationShownProvinceState` | `广东省` | 画面主体所在省份 |
| `XMP-iptcExt` | `LocationShownCity` | `潮州市` | 画面主体所在城市 |
| `XMP-iptcExt` | `LocationShownSublocation` | `吉利村` | 画面主体所在区县/景点 |

---

### 2.4 Lightroom 分层关键词标签树与检索词 (Hierarchical Keywords)

| 命名空间 | 标签名 | 写入内容与格式 | 摄影师体验与生态联动 |
| :--- | :--- | :--- | :--- |
| `XMP-lr` | `HierarchicalSubject` | `中国|广东省|潮州市|吉利村` | **Lightroom / Capture One 关键词树**：在侧边栏自动生成可折叠的树状地点目录 |
| `XMP-dc` | `subject` | `["中国", "广东省", "潮州市", "吉利村"]` | **通用关键词列表**：支持 macOS 聚焦搜索、Windows 索引与图库搜索 |
| `IPTC` | `Keywords` | `中国, 广东省, 潮州市, 吉利村` | **传统 IPTC 关键词**：兼容老旧图片浏览器与归档软件 |

---

## 3. 生成与维护的文件规则

### 3.1 侧车文件命名约定 (Sidecar Naming)
- **复合格式（严格遵循业界最佳实践）**：
  - `DSC_2948.nef` $\rightarrow$ `DSC_2948.nef.xmp`
  - `DSC_2948.JPG` $\rightarrow$ `DSC_2948.JPG.xmp`
  - `DSC_2948.CR3` $\rightarrow$ `DSC_2948.cr3.xmp`
- **单格式历史兼容**：
  - 若输入目录中已存在 `DSC_2948.xmp`，系统自动识别并无缝升级更新。

### 3.2 伴随文件一同归档 (Companion File Bundle)
当执行阶段 4 日期归档与规范化重命名（例如 `DSC_2948` $\rightarrow$ `DSC_2026-01-01_2948`）时：
- 主文件与所有伴随文件**整体成组移动**；
- 复合后缀结构**完整保留**：
  - `DSC_2948.nef` $\rightarrow$ `Processed/2026/0101/DSC_2026-01-01_2948.nef`
  - `DSC_2948.nef.xmp` $\rightarrow$ `Processed/2026/0101/DSC_2026-01-01_2948.nef.xmp`
  - `DSC_2948.JPG.xmp` $\rightarrow$ `Processed/2026/0101/DSC_2026-01-01_2948.jpg.xmp`
  - `DSC_2948.WAV` $\rightarrow$ `Processed/2026/0101/DSC_2026-01-01_2948.wav`

---

## 4. 主流后期与摄影管理软件兼容矩阵

| 软件 / 平台 | 读取原生 EXIF GPS (默认模式) | 读取 XMP 侧车 GPS (--sidecar-only) | 读取中文地名 (IPTC/XMP) | 读取分层标签树 (HierarchicalSubject) |
| :--- | :---: | :---: | :---: | :---: |
| **Nikon NX Studio** | 🌟 完美原生支持 | ❌ 仅读内嵌 | 🌟 完美支持 | ⚠️ 仅识别扁平词 |
| **Adobe Lightroom Classic** | 🌟 完美原生支持 | 🌟 完美原生支持 | 🌟 完美支持 | 🌟 自动渲染为层级标签树 |
| **Capture One Pro** | 🌟 完美原生支持 | 🌟 完美原生支持 | 🌟 完美支持 | 🌟 自动渲染为层级标签树 |
| **macOS 访达 (Finder/QuickLook)** | 🌟 原生地图定位 | ❌ 仅读内嵌 | 🌟 访达简介可见 | 🌟 聚焦搜索可检索 |
| **iOS / iPadOS 系统相册** | 🌟 原生地图相册 | ❌ 仅读内嵌 | 🌟 详情页可见 | 🌟 搜索栏可直搜 |
| **群晖 Synology Photos** | 🌟 原生地图足迹 | ❌ 仅读内嵌 | 🌟 地点相册聚合 | 🌟 标签相册聚合 |
| **飞牛 fnOS 私有云相册** | 🌟 原生地图足迹 | ❌ 仅读内嵌 | 🌟 智能地点识别 | 🌟 标签检索 |
| **Photo Mechanic** | 🌟 完美原生支持 | 🌟 完美原生支持 | 🌟 完整解析结构体 | 🌟 完整支持 |
