import PhotoolsCore
import SwiftUI

struct GuideListView: View {
    @ObservedObject var store: WorkspaceStore
    @State private var searchText: String = ""

    private var filteredDocs: [GuideDocItem] {
        if searchText.trimmingCharacters(in: .whitespaces).isEmpty {
            return GuideDocItem.allDocs
        }
        return GuideDocItem.allDocs.filter {
            $0.title.localizedCaseInsensitiveContains(searchText) ||
            $0.subtitle.localizedCaseInsensitiveContains(searchText) ||
            ($0.fileName?.localizedCaseInsensitiveContains(searchText) ?? false) ||
            $0.category.localizedCaseInsensitiveContains(searchText)
        }
    }

    private var categories: [String] {
        var set = [String]()
        for doc in filteredDocs {
            if !set.contains(doc.category) {
                set.append(doc.category)
            }
        }
        return set
    }

    var body: some View {
        VStack(spacing: 0) {
            // 顶部搜索过滤栏
            HStack(spacing: 6) {
                Image(systemName: "magnifyingglass")
                    .font(.caption2)
                    .foregroundStyle(.secondary)

                TextField("搜索文档、架构或规范...", text: $searchText)
                    .textFieldStyle(.plain)
                    .font(.system(size: 11.5))

                if !searchText.isEmpty {
                    Button {
                        searchText = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
            .overlay(
                RoundedRectangle(cornerRadius: 6)
                    .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
            )
            .padding(.horizontal, 12)
            .padding(.vertical, 8)

            Divider()

            // 文档分组列表
            ScrollView {
                LazyVStack(spacing: 12, pinnedViews: [.sectionHeaders]) {
                    ForEach(categories, id: \.self) { cat in
                        Section {
                            VStack(spacing: 4) {
                                ForEach(filteredDocs.filter { $0.category == cat }) { doc in
                                    docRow(doc)
                                }
                            }
                        } header: {
                            HStack {
                                Text(cat)
                                    .font(.system(size: 10, weight: .bold))
                                    .foregroundStyle(.secondary)
                                Spacer()
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 4)
                            .background(Color(nsColor: .windowBackgroundColor))
                        }
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 6)
            }
        }
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private func docRow(_ doc: GuideDocItem) -> some View {
        let isSelected = store.selectedGuideDoc.id == doc.id

        return Button {
            store.selectedGuideDoc = doc
        } label: {
            HStack(alignment: .top, spacing: 8) {
                // 图标
                ZStack {
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .fill(isSelected ? Color.teal.opacity(0.2) : Color.secondary.opacity(0.1))
                        .frame(width: 28, height: 28)

                    Image(systemName: doc.icon)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(isSelected ? Color.teal : Color.primary)
                }

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 4) {
                        Text(doc.title)
                            .font(.system(size: 11.5, weight: isSelected ? .bold : .medium))
                            .foregroundStyle(isSelected ? Color.teal : Color.primary)
                            .lineLimit(1)

                        Spacer()

                        if let badge = doc.badge {
                            Text(badge)
                                .font(.system(size: 8.5, weight: .semibold))
                                .foregroundStyle(isSelected ? Color.teal : Color.secondary)
                                .padding(.horizontal, 4)
                                .padding(.vertical, 1)
                                .background(
                                    isSelected ? Color.teal.opacity(0.15) : Color.secondary.opacity(0.12),
                                    in: Capsule()
                                )
                        }
                    }

                    Text(doc.subtitle)
                        .font(.system(size: 10))
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)

                    if let fn = doc.fileName {
                        HStack(spacing: 3) {
                            Image(systemName: "doc.text")
                                .font(.system(size: 8))
                            Text(fn)
                                .font(.system(size: 8.5, design: .monospaced))
                        }
                        .foregroundStyle(.secondary.opacity(0.7))
                        .padding(.top, 1)
                    }
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
            .background(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(isSelected ? Color.teal.opacity(0.1) : Color.clear)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .stroke(isSelected ? Color.teal.opacity(0.4) : Color.clear, lineWidth: 1)
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
