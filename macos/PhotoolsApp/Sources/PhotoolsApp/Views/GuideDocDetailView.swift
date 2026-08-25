import PhotoolsCore
import SwiftUI
import AppKit

public struct GuideDocDetailView: View {
    @ObservedObject var store: WorkspaceStore
    @State private var docContent: String = ""
    @State private var docFilePath: String = ""
    @State private var isLoading: Bool = false

    public init(store: WorkspaceStore) {
        self.store = store
    }

    public var body: some View {
        VStack(spacing: 0) {
            // 顶部文档导航与操作工具栏
            headerToolbar

            Divider()

            // 正文 Markdown 渲染展示区
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if isLoading {
                        HStack {
                            Spacer()
                            ProgressView("正在载入文档内容...")
                            Spacer()
                        }
                        .padding(50)
                    } else {
                        MarkdownRendererView(markdownText: docContent)
                    }
                }
                .padding(20)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .background(Color(nsColor: .textBackgroundColor))
        .onChange(of: store.selectedGuideDoc) { _ in
            loadDocContent(for: store.selectedGuideDoc)
        }
        .onAppear {
            loadDocContent(for: store.selectedGuideDoc)
        }
    }

    private var headerToolbar: some View {
        HStack(alignment: .center, spacing: 10) {
            ZStack {
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(Color.teal.opacity(0.15))
                    .frame(width: 32, height: 32)
                Image(systemName: store.selectedGuideDoc.icon)
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(Color.teal)
            }

            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(store.selectedGuideDoc.title)
                        .font(.system(size: 13, weight: .bold))

                    if let badge = store.selectedGuideDoc.badge {
                        Text(badge)
                            .font(.system(size: 9, weight: .semibold))
                            .foregroundStyle(.teal)
                            .padding(.horizontal, 5)
                            .padding(.vertical, 1)
                            .background(Color.teal.opacity(0.12), in: Capsule())
                    }
                }

                if !docFilePath.isEmpty {
                    Text(docFilePath)
                        .font(.system(size: 9.5, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .textSelection(.enabled)
                } else {
                    Text(store.selectedGuideDoc.subtitle)
                        .font(.system(size: 10.5))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }

            Spacer()

            if !docFilePath.isEmpty {
                Button {
                    NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: docFilePath)])
                } label: {
                    Label("在访达中显示", systemImage: "folder")
                        .font(.system(size: 10))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("在访达 (Finder) 中高亮此 Markdown 文件")

                Button {
                    NSWorkspace.shared.open(URL(fileURLWithPath: docFilePath))
                } label: {
                    Label("外部打开", systemImage: "arrow.up.right.square")
                        .font(.system(size: 10))
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("在系统默认 Markdown 编辑器中打开")
            }

            Button {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(docContent, forType: .string)
            } label: {
                Label("复制全文", systemImage: "doc.on.doc")
                    .font(.system(size: 10))
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
            .help("复制 Markdown 全文内容至剪贴板")
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private func locateDocPath(for doc: GuideDocItem) -> String? {
        guard let fileName = doc.fileName else { return nil }

        // 1. App Bundle Resources/docs/ 或 Resources/
        if let bundleDocPath = Bundle.main.resourceURL?.appendingPathComponent("docs/\(fileName)").path,
           FileManager.default.fileExists(atPath: bundleDocPath) {
            return bundleDocPath
        }
        if let bundleRootPath = Bundle.main.resourceURL?.appendingPathComponent(fileName).path,
           FileManager.default.fileExists(atPath: bundleRootPath) {
            return bundleRootPath
        }

        // 2. 源码仓库根目录与 docs/ 目录
        let repoRoot = RepositoryLocator().locateRepositoryRoot()
        let candidates = [
            (repoRoot as NSString).appendingPathComponent("docs/\(fileName)"),
            (repoRoot as NSString).appendingPathComponent(fileName),
            "/Users/vincent/Pictures/GPS/docs/\(fileName)",
            "/Users/vincent/Pictures/GPS/\(fileName)"
        ]

        for path in candidates {
            if FileManager.default.fileExists(atPath: path) {
                return path
            }
        }

        return nil
    }

    private func loadDocContent(for doc: GuideDocItem) {
        if doc.id == "quick-guide" {
            self.docFilePath = ""
            self.docContent = defaultQuickGuideMarkdown
            return
        }

        guard let path = locateDocPath(for: doc) else {
            self.docFilePath = ""
            self.docContent = "⚠️ 未找到相关文档文件: `\(doc.fileName ?? "")`\n\n请确认该文档存在于仓库根目录或 `docs/` 文件夹中。"
            return
        }

        self.docFilePath = path
        do {
            let text = try String(contentsOfFile: path, encoding: .utf8)
            self.docContent = text
        } catch {
            self.docContent = "⚠️ 读取 Markdown 文档失败: \(error.localizedDescription)\n\n路径: `\(path)`"
        }
    }

    private var defaultQuickGuideMarkdown: String {
        """
        # 📸 photools 摄影师工作台使用指南

        > [!TIP]
        > 本系统专为 **Golang 程序员与尼康摄影师** 打造，专注于相机 RAW/JPG 照片导入电脑后的 **GPX 轨迹匹配、智能时间插值推算、离线高精逆地理编码中文地名清洗与规范化拍摄日期归档**。

        ---

        ## 1. 摄影资产主文件优先模型 (Primary Asset Model)

        - **配套文件绑定**：同 basename 的配套文件（`.NEF` / `.JPG` / `.xmp`）视为一个独立的拍摄单元；
        - **RAW 主文件决策源**：若存在 `RAW`，以 `RAW` 为主决策源并自动同步至伴随的 `JPG` 与 `XMP`；
        - **独立单文件平等支持**：单 `JPG` 或单 `RAW` 文件平等享受 GPX 匹配、GPS 插值、逆地理与归档能力；
        - **伴随文件整体归档**：同 basename 的伴随文件（`XMP`、`ACR`、`WAV` 等）作为伴随文件整体同步维护并一同归档。

        ---

        ## 2. 四大能力插件矩阵与分阶段流转

        ```
        [RAW+JPG 配套组]
             │
             ▼
        阶段 1: gpx_matching (优先级 10) ────> 依赖 GPX 轨迹时间轴二分匹配，写入经纬度并严格二次校验
             │ (未命中轨迹平滑交接)
             ▼
        阶段 2: gps_interpolate (优先级 15) ──> 内存按日分桶二分查找前后最近机位，球面大圆时间权重插值推算
             │
             ▼
        阶段 3: reverse_geocode (优先级 20) ──> 基于 3D KD-Tree 毫秒级空间索引，写入国家/省/市/区/POI 中文地名
             │
             ▼
        阶段 4: date_archive (优先级 100) ───> 提取原始 EXIF 拍摄时间，规范重命名并安全归档至 Processed/YYYY/MMDD/
        ```

        ### 插件能力与优先级说明：
        1. **`gpx_matching` (优先级 10)**: 依赖 ExifTool 进行 GPX 轨迹时间轴精准匹配；
        2. **`gps_interpolate` (优先级 15)**: 智能时间分桶与 $O(\\log K)$ 二分查找，双向/单向插值推算；
        3. **`reverse_geocode` (优先级 20)**: 基于 3D KD-Tree 离线高精空间索引，写入中文规范地名；
        4. **`date_archive` (优先级 100)**: 破坏性重命名与归档插件，具备最低优先级。

        ---

        ## 3. 离线高精逆地理编码与数据包

        - **全球 94 万+ 离线地名点位**：内置中国高精库（2800+ 区县、70万+ 景点 POI）；
        - **外挂大洲扩展包**：支持在「离线地理库」中一键下载安装亚洲、欧洲、北美、大洋洲、南美、非洲等离线包；
        - **3D 球面 KD-Tree 加速**：笛卡尔坐标转换与二叉空间剪枝，点位检索平均仅耗时 **10~20 微秒**（剪枝率 >99.9%）。

        ---

        ## 4. 高级模式与就地保存

        - **扁平原地模式 (`--flat`)**：忽略传统 `Inbox/` -> `Processed/YYYY/MMDD/` 分层，直接指定源目录就地扫描、打标并原地规范化重命名；
        - **原地重命名 (`--in-place`)**：就地规范化重命名，不创建子目录；
        - **软降级容错 (`--allow-no-gps`)**：彻底无 GPS 照片在逆地理阶段良性跳过，安全进入阶段 4 按拍摄日期规范归档；
        - **测试快照模式 (`--test`)**：处理前全量快照备份至 `Inbox_bak`，可在「测试快照还原」模块一键恢复。
        """
    }
}
