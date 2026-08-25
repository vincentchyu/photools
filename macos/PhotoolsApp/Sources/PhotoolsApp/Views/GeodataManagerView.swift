import PhotoolsCore
import SwiftUI

/// 离线地理数据包状态与终端操作指引视图
struct GeodataManagerView: View {
    @ObservedObject var store: WorkspaceStore
    @ObservedObject private var lang = LanguageManager.shared
    @State private var copiedCode: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                headerView
                policyBanner
                Divider()
                continentPacksSection
            }
            .padding(16)
        }
        .onAppear {
            if store.continentPacks.isEmpty {
                store.loadGeodataList()
            }
        }
    }

    private var headerView: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Image(systemName: "globe.asia.australia.fill")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.teal)
                Text(lang.text(.globalGeodataTitle))
                    .font(.headline.weight(.bold))
            }
            Text(lang.text(.globalGeodataDesc))
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    // 终端操作原则提示横幅
    private var policyBanner: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: "terminal.fill")
                .font(.title3)
                .foregroundStyle(.teal)

            VStack(alignment: .leading, spacing: 3) {
                Text(lang.text(.geodataTerminalPolicyNotice))
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.primary)
                Text(lang.text(.geodataTerminalPolicyDesc))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            Button {
                copyCommand("photools geodata install all", code: "all")
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: copiedCode == "all" ? "checkmark" : "doc.on.doc")
                    Text(copiedCode == "all" ? lang.text(.copiedToClipboard) : lang.text(.copyInstallAllCmd))
                }
                .font(.caption2.weight(.semibold))
            }
            .buttonStyle(.borderedProminent)
            .tint(.teal)
            .controlSize(.small)
        }
        .padding(12)
        .background(Color.teal.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.teal.opacity(0.2), lineWidth: 1)
        )
    }

    private var continentPacksSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("\(lang.text(.continentListTitle)) (\(store.continentPacks.count))")
                    .font(.subheadline.weight(.bold))
                Spacer()
                Button {
                    store.loadGeodataList()
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .buttonStyle(.borderless)
                .help(lang.text(.refreshStatus))
                .disabled(store.isGeodataLoading)
            }

            if store.isGeodataLoading && store.continentPacks.isEmpty {
                HStack {
                    Spacer()
                    ProgressView(lang.text(.checkingLocalGeodata))
                        .font(.caption)
                    Spacer()
                }
                .padding(24)
            } else if store.continentPacks.isEmpty {
                VStack(spacing: 8) {
                    Text(lang.text(.noGeodataPacks))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity)
                .padding(24)
                .background(Color(nsColor: .controlBackgroundColor).opacity(0.6), in: RoundedRectangle(cornerRadius: 8))
            } else {
                VStack(spacing: 8) {
                    ForEach(store.continentPacks) { pack in
                        continentPackRow(pack)
                    }
                }
            }
        }
    }

    private func continentPackRow(_ pack: GeodataContinentPack) -> some View {
        HStack(alignment: .center, spacing: 10) {
            Image(systemName: pack.isInstalled ? "checkmark.circle.fill" : "circle.dashed")
                .font(.title3)
                .foregroundStyle(pack.isInstalled ? .green : .secondary)

            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(lang.currentLanguage.isChinese ? pack.nameZH : pack.code.uppercased() + " - " + pack.nameZH)
                        .font(.caption.weight(.semibold))
                    Text("[\(pack.code)]")
                        .font(.system(size: 9.5, design: .monospaced))
                        .foregroundStyle(.secondary)
                }

                if pack.isInstalled {
                    Text("\(lang.text(.statusInstalled)) · \(pack.pointCount) pts · \(String(format: "%.1f", pack.sizeMB)) MB")
                        .font(.system(size: 9.5))
                        .foregroundStyle(.green)
                } else {
                    Text(pack.description)
                        .font(.system(size: 9.5))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }

            Spacer()

            // 终端命令复制按钮
            let isCopied = (copiedCode == pack.code)
            if pack.isInstalled {
                Button {
                    copyCommand("photools geodata remove \(pack.code)", code: pack.code)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: isCopied ? "checkmark" : "doc.on.doc")
                        Text(isCopied ? lang.text(.copiedToClipboard) : lang.text(.uninstallCmdCopied))
                    }
                    .font(.caption2)
                }
                .buttonStyle(.borderless)
                .foregroundStyle(.secondary)
                .help("复制命令: photools geodata remove \(pack.code)")
            } else {
                Button {
                    copyCommand("photools geodata install \(pack.code)", code: pack.code)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: isCopied ? "checkmark" : "terminal")
                        Text(isCopied ? lang.text(.copiedToClipboard) : lang.text(.installCmdCopied))
                    }
                    .font(.caption2)
                }
                .buttonStyle(.bordered)
                .controlSize(.mini)
                .help("复制命令: photools geodata install \(pack.code)")
            }
        }
        .padding(10)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    private func copyCommand(_ cmd: String, code: String) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(cmd, forType: .string)
        withAnimation {
            copiedCode = code
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) {
            if copiedCode == code {
                withAnimation {
                    copiedCode = nil
                }
            }
        }
    }
}
