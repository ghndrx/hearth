import SwiftUI

struct ChannelListView: View {
    let server: Server

    @State private var channels: [Channel] = []
    @State private var isLoading = false
    @State private var selectedChannel: Channel?
    @State private var errorMessage: String?

    private let apiClient: APIClient

    init(
        server: Server,
        apiClient: APIClient = APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!)
    ) {
        self.server = server
        self.apiClient = apiClient
    }

    /// Channels grouped by their category. Uncategorized channels use `nil` key.
    private var groupedChannels: [(category: Channel?, channels: [Channel])] {
        let categories = channels
            .filter { $0.type == .category }
            .sorted { ($0.position ?? 0) < ($1.position ?? 0) }

        let nonCategoryChannels = channels.filter { $0.type != .category }

        var groups: [(category: Channel?, channels: [Channel])] = []

        // Uncategorized channels (no matching server_id category or position before first category)
        let uncategorized = nonCategoryChannels.filter { channel in
            !categories.contains { _ in false } // simplified: group by position relative to categories
        }

        // Build groups based on category positions
        if categories.isEmpty {
            groups.append((nil, nonCategoryChannels.sorted { ($0.position ?? 0) < ($1.position ?? 0) }))
        } else {
            // Place channels under the category that precedes them by position
            var channelsByCategory: [String?: [Channel]] = [nil: []]

            for category in categories {
                channelsByCategory[category.id] = []
            }

            for channel in nonCategoryChannels {
                // Find the nearest category with a lower position
                let owningCategory = categories.last { ($0.position ?? 0) < (channel.position ?? 0) }
                channelsByCategory[owningCategory?.id, default: []].append(channel)
            }

            // Add uncategorized first
            let uncatChannels = channelsByCategory[nil] ?? []
            if !uncatChannels.isEmpty {
                groups.append((nil, uncatChannels.sorted { ($0.position ?? 0) < ($1.position ?? 0) }))
            }

            for category in categories {
                let catChannels = channelsByCategory[category.id] ?? []
                groups.append((category, catChannels.sorted { ($0.position ?? 0) < ($1.position ?? 0) }))
            }
        }

        return groups
    }

    var body: some View {
        List {
            ForEach(groupedChannels, id: \.category?.id) { group in
                Section {
                    ForEach(group.channels) { channel in
                        NavigationLink {
                            MessageListView(channel: channel)
                        } label: {
                            ChannelRow(channel: channel)
                        }
                    }
                } header: {
                    if let category = group.category {
                        Text(category.name.uppercased())
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .listStyle(.sidebar)
        .navigationTitle(server.name)
        .navigationBarTitleDisplayMode(.large)
        .overlay {
            if isLoading && channels.isEmpty {
                ProgressView("Loading channels...")
            } else if channels.isEmpty && !isLoading {
                ContentUnavailableView(
                    "No Channels",
                    systemImage: "number",
                    description: Text("This server has no channels yet.")
                )
            }
        }
        .task {
            await loadChannels()
        }
        .refreshable {
            await loadChannels()
        }
    }

    private func loadChannels() async {
        isLoading = true
        defer { isLoading = false }

        do {
            channels = try await apiClient.fetchChannels(serverId: server.id)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

private struct ChannelRow: View {
    let channel: Channel

    var body: some View {
        HStack(spacing: 6) {
            channelIcon
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .frame(width: 20)

            Text(channel.name)
                .font(.body)
                .lineLimit(1)

            Spacer()

            if channel.nsfw == true {
                Text("NSFW")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(.red)
                    .padding(.horizontal, 4)
                    .padding(.vertical, 2)
                    .background(Color.red.opacity(0.15), in: RoundedRectangle(cornerRadius: 3))
            }
        }
    }

    @ViewBuilder
    private var channelIcon: some View {
        switch channel.type {
        case .text:
            Image(systemName: "number")
        case .voice:
            Image(systemName: "speaker.wave.2.fill")
        case .announcement:
            Image(systemName: "megaphone.fill")
        case .forum:
            Image(systemName: "text.bubble.fill")
        case .stage:
            Image(systemName: "person.wave.2.fill")
        default:
            Image(systemName: "number")
        }
    }
}
