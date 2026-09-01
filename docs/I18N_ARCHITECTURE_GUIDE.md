# 🌐 photools 全链路国际化 (i18n) 架构设计与开发维护规约

本文档详细说明 `photools` 系统中多语言国际化（Internationalization / i18n）的设计理念、架构分层、单一事实源字典管理、跨语言 (Go / Swift C-ABI) 传递机制以及自动化质量守卫规约。

---

## 1. 核心设计原则

1. **单一事实源 (Single Source of Truth)**：
   - 所有的静态 UI 文本、CLI 参数说明、插件自描述元数据、运行时调度事件、错误诊断建议与落盘日志模板，统一定义在根目录的 `locales/zh-CN.json`（简体中文）与 `locales/en-US.json`（英文）中；
   - 严禁在 Go 代码或 Swift 代码中硬编码中英文业务字符串。

2. **零运行时额外 I/O 损耗 (Static Go Embed)**：
   - `locales/locales.go` 通过 Go 标准库 `//go:embed *.json` 将 JSON 字典静态编译到二进制产物中；
   - 运行时字典查找完全在内存哈希表（`map[string]string`）中毫秒级完成，杜绝外部磁盘读取。

3. **动态热切换与平滑覆盖 (Hot-Reload & State Sync)**：
   - 支持通过 CLI 参数 `--lang [zh|en]`、TUI 快捷键 `[l]`、macOS 设置面板无缝热切换语言；
   - 语言变更通过 C-Shared ABI 强同步至 Go 核心引擎与 Swift 客户端，确保全链路状态完全一致。

4. **100% 字典对称性与无汉字泄漏守卫 (Automated Guard Tests)**：
   - 自动化测试 [`internal/i18n/i18n_guard_test.go`](../internal/i18n/i18n_guard_test.go) 强制保证 `zh-CN.json` 与 `en-US.json` 键名 100% 镜像对称；
   - 英文模式下运行守卫测试，严格断言所有插件名称、描述、配置项与阶段名称中**零中文字符残留**。

---

## 2. 国际化系统架构拓扑

```mermaid
flowchart TD
    subgraph SSoT [单一事实源]
        DictZH[locales/zh-CN.json]
        DictEN[locales/en-US.json]
        EmbedFS[locales/locales.go: embed.FS]
        DictZH --> EmbedFS
        DictEN --> EmbedFS
    end

    subgraph CoreI18n [Go 核心 i18n 引擎 (internal/i18n)]
        I18nPkg[i18n.T / i18n.SetLanguage / i18n.GetLanguage]
        EmbedFS --> I18nPkg
    end

    subgraph Capabilities [四大能力插件 & 调度器]
        GPX[GPX Matching 插件]
        Interpolate[GPS Interpolate 插件]
        Geocode[Reverse Geocode 插件]
        Archive[Date Archive 插件]
        Orchestrator[Pipeline Orchestrator 调度器]
        
        I18nPkg --> GPX
        I18nPkg --> Interpolate
        I18nPkg --> Geocode
        I18nPkg --> Archive
        I18nPkg --> Orchestrator
    end

    subgraph Presentation [多端展现层]
        TUI[Bubble Tea 终端交互界面]
        CLI[命令行标准输出与帮助]
        CShared[cmd/photools-cshared C-ABI 桥接]
        SwiftApp[macOS 原生客户端 (LanguageManager)]
        LogFile[~/.logs/photools/photools_latest.log]
        
        Orchestrator --> TUI
        Orchestrator --> CLI
        Orchestrator --> LogFile
        Orchestrator --> CShared
        CShared --> SwiftApp
    end
```

---

## 3. 核心模块与实现规范

### 3.1 基础翻译调用 (`internal/i18n`)
- **常用翻译函数**：
  ```go
  import "github.com/vincentchyu/photools/internal/i18n"

  // 1. 无参数静态文案
  text := i18n.T("tuiMenuTitle")

  // 2. 带格式化参数动态文案
  msg := i18n.T("eventPhaseEntering", phaseTitle, totalAssets)
  ```
- **语言归一化与判断**：
  - `i18n.NormalizeLanguage("zh_CN.UTF-8")` ➔ 返回 `"zh-CN"`；
  - `i18n.NormalizeLanguage("en_GB")` ➔ 返回 `"en-US"`；
  - `i18n.IsChinese()` ➔ 快速判断当前是否为中文环境。

---

### 3.2 阶段枚举本地化 (`domain.StageDisplayName`)
在 [`internal/domain/event.go`](../internal/domain/event.go) 中定义了阶段本地化投影函数，杜绝日志中写死中文前缀：
```go
func StageDisplayName(s PipelineStage) string {
    switch s {
    case StageInit:        return i18n.T("stageInit")
    case StageDiscover:    return i18n.T("stageDiscover")
    case StagePrecheck:    return i18n.T("stagePrecheck")
    case StageGeotag:      return i18n.T("stageGeotag")
    case StageInterpolate: return i18n.T("stageInterpolate")
    case StageGeocode:     return i18n.T("stageGeocode")
    case StageSync:        return i18n.T("stageSync")
    case StageArchive:     return i18n.T("stageArchive")
    case StageBackup:      return i18n.T("stageBackup")
    case StageRestore:     return i18n.T("stageRestore")
    case StageSummary:     return i18n.T("stageSummary")
    case StageComplete:    return i18n.T("stageComplete")
    default:               return string(s)
    }
}
```

---

### 3.3 插件可配置项自描述契约 (`domain.OptionSpec`)
所有插件与全局配置项的名称和说明，均通过 `NameKey` 和 `DescKey` 指向字典键名：
```go
type OptionSpec struct {
    Key          string      `json:"key"`
    NameKey      string      `json:"name_key,omitempty"` // 对应字典键
    DescKey      string      `json:"desc_key,omitempty"` // 对应字典键
    Type         OptionType  `json:"type"`
    DefaultValue any         `json:"default_value"`
    Choices      []string    `json:"choices,omitempty"`
}

// 动态求值
func (o OptionSpec) DisplayName() string {
    if o.NameKey != "" { return i18n.T(o.NameKey) }
    return o.Key
}
```

---

### 3.4 C-Shared FFI 与 Swift 客户端语言透传
- **Go 动态库层 (`cmd/photools-cshared/main.go`)**：
  - 导出符号 `Photools_SetLanguage(cLang *C.char)`；
  - `Photools_RunPipeline` 在解析配置 JSON 时，首先读取 `opts.Language` 并调用 `i18n.SetLanguage(...)`；
  - 事件回调向 Swift 上报时，`cStage` 自动传递 `domain.StageDisplayName(evt.Stage)`。
- **Swift 原生层 (`LanguageManager.swift` & `PhotoolsEngine.swift`)**：
  - `LanguageManager.shared.currentLanguage` 发生变更时，自动调用 `PhotoolsEngine.shared.setLanguage(...)`；
  - `PipelineRunOptions` 显式携带 `language: String` 参数，确保后台执行协程无条件处于正确语言环境。

---

## 4. 字典命名空间与命名规范

在 `locales/zh-CN.json` 和 `locales/en-US.json` 中，采用**驼峰命名法（CamelCase）**并按功能前缀清晰划分命名空间：

| 命名空间前缀 | 用途说明 | 示例 |
| :--- | :--- | :--- |
| `tui*` | 终端 TUI 菜单、看板、按键提示与状态条 | `tuiMenuTitle`, `tuiKeyRun` |
| `cli*` | CLI 命令行参数、子命令帮助与用法提示 | `cliUsagePipeline`, `cliOptGpxDir` |
| `opt*` | 全局及插件配置项名称与描述 | `optBaseDirName`, `optGeosyncDesc` |
| `cap*` | 四大能力插件名称与详细描述 | `capGpxName`, `capArchiveDesc` |
| `stage*` | 流水线各阶段本地化名称 | `stageGeotag`, `stageArchive` |
| `event*` | 流水线调度、屏障放行、并发进度事件 | `eventPhaseEntering`, `eventPipelineSummaryDone` |
| `log*` | 各插件专有执行日志、自检与落盘头尾 | `logGpxDriftCalibratedSuccess`, `logFileHeaderTitle` |
| `suggestion*`| 待补报告与异常诊断建议 | `suggestionArchiveTargetExists`, `suggestionGpxMissingTrack` |
| `issue*` | 问题诊断状态与原因模板 | `issueStatusPreservedInSource`, `issueReasonExecutionFailed` |

---

## 5. 新增文案与插件国际化开发 SOP

未来为系统新增功能或新能力插件时，**必须严格执行以下 4 步闭环**：

1. **增补单一事实源字典**：
   - 在 `locales/zh-CN.json` 添加规范中文键值；
   - **立即且同步**在 `locales/en-US.json` 添加对应英文键值（键名完全一致）。

2. **代码引用替换**：
   - 业务逻辑中调用 `i18n.T("yourNewKey", args...)`；
   - 若涉及配置项，在 `OptionSpec` 中指定 `NameKey: "optYourKeyName"`, `DescKey: "optYourKeyDesc"`。

3. **Swift 端映射（若涉及 GUI）**：
   - 在 `macos/.../LanguageManager.swift` 的 `L10nKey` 枚举、`chineseDictionary` 与 `englishDictionary` 中同步补充。

4. **自动化守卫验证**：
   - 运行国际化守卫测试，确保 100% 镜像对称且无中文泄露：
     ```bash
     go test -v ./internal/i18n/...
     ```
