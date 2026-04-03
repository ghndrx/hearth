import SwiftUI

struct DMListView: View {
    @State private var dmChannels: [Channel] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    private let apiClient: APIClient

    init(apiClient: APIClient = APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!)) {
        self.apiClient = apiClient
    }

    var body: some View {
        NavigationStack {
            List {
                if isLoading && dmChannels.isEmpty {
                    HStack {
                        Spacer()
                        ProgressView()
                        Spacer()
                    }
                    .listRowSeparator(.hidden)
                } else if dmChannels.isEmpty {
                    ContentUnavailableView(
                        "No Direct Messages",
                        systemImage: "bubble.left.and.bubble.right",
                        description: Text("Start a conversation with someone!")
                    )
                    .listRowSeparator(.hidden)
                } else {
                    ForEach(dmChannels) { channel in
                        NavigationLink {
                            MessageListView(channel: channel)
                        } label: {
                            DMRow(channel: channel)
                        }
                    }
                }
            }
            .listStyle(.plain)
            .navigationTitle("Messages")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        // TODO: New DM flow
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                }
            }
            .task {
                await loadDMs()
            }
            .refreshable {
                await loadDMs()
            }
        }
    }

    private func loadDMs() async {
        isLoading = true
        defer { isLoading = false }

        do {
            dmChannels = try await apiClient.fetchDMs()
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

private struct DMRow: View {
    let channel: Channel

    var body: some View {
        HStack(spacing: 12) {
            // Avatar placeholder
            Circle()
                .fill(Color.orange.opacity(0.2))
                .frame(width: 44, height: 44)
                .overlay {
                    Text(String(channel.name.prefix(1)).uppercased())
                        .font(.headline)
                        .foregroundStyle(.orange)
                }

            VStack(alignment: .leading, spacing: 3) {
                Text(channel.name)
                    .font(.body.weight(.medium))
                    .lineLimit(1)

                if let topic = channel.topic {
                    Text(topic)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                } else {
                    Text("No messages yet")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }

            Spacer()

            // Unread dot placeholder
            // Circle()
            //     .fill(.orange)
            //     .frame(width: 10, height: 10)
        }
        .padding(.vertical, 4)
    }
}

#Preview {
    DMListView()
}
