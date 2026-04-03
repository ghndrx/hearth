import SwiftUI

struct ServerListView: View {
    @Environment(AuthManager.self) private var authManager
    @State private var servers: [Server] = []
    @State private var selectedServer: Server?
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showAddServer = false

    private let apiClient: APIClient

    init(apiClient: APIClient = APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!)) {
        self.apiClient = apiClient
    }

    var body: some View {
        NavigationSplitView {
            ScrollView {
                LazyVStack(spacing: 8) {
                    // DMs Button
                    NavigationLink {
                        DMListView()
                    } label: {
                        ServerIconView(
                            imageURL: nil,
                            fallbackIcon: "bubble.left.and.bubble.right.fill",
                            isSelected: false
                        )
                    }

                    Divider()
                        .frame(width: 32)
                        .padding(.vertical, 4)

                    // Server List
                    if isLoading && servers.isEmpty {
                        ProgressView()
                            .padding(.top, 20)
                    } else {
                        ForEach(servers) { server in
                            ServerIconButton(
                                server: server,
                                isSelected: selectedServer?.id == server.id
                            ) {
                                selectedServer = server
                            }
                        }
                    }

                    Divider()
                        .frame(width: 32)
                        .padding(.vertical, 4)

                    // Add Server Button
                    Button {
                        showAddServer = true
                    } label: {
                        ServerIconView(
                            imageURL: nil,
                            fallbackIcon: "plus",
                            isSelected: false,
                            tintColor: .green
                        )
                    }
                }
                .padding(.vertical, 8)
            }
            .frame(width: 72)
            .background(Color(.systemGroupedBackground))
            .navigationBarHidden(true)
        } detail: {
            if let server = selectedServer {
                ChannelListView(server: server)
            } else {
                ContentUnavailableView(
                    "Select a Server",
                    systemImage: "server.rack",
                    description: Text("Choose a server from the sidebar to get started.")
                )
            }
        }
        .task {
            await loadServers()
        }
        .refreshable {
            await loadServers()
        }
        .alert("Add Server", isPresented: $showAddServer) {
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Server creation coming soon.")
        }
    }

    private func loadServers() async {
        isLoading = true
        defer { isLoading = false }

        do {
            servers = try await apiClient.fetchServers()
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Subviews

private struct ServerIconButton: View {
    let server: Server
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ServerIconView(
                imageURL: server.iconUrl.flatMap { URL(string: $0) },
                fallbackText: String(server.name.prefix(1)).uppercased(),
                isSelected: isSelected
            )
        }
        .overlay(alignment: .leading) {
            // Unread indicator pill
            if !isSelected {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color.primary)
                    .frame(width: 4, height: 8)
                    .opacity(0) // Set to 1 when unread logic is wired up
                    .offset(x: -4)
            }
        }
    }
}

private struct ServerIconView: View {
    let imageURL: URL?
    var fallbackText: String?
    var fallbackIcon: String?
    var isSelected: Bool = false
    var tintColor: Color = .orange

    var body: some View {
        Group {
            if let imageURL {
                AsyncImage(url: imageURL) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    fallbackContent
                }
            } else {
                fallbackContent
            }
        }
        .frame(width: 48, height: 48)
        .clipShape(
            RoundedRectangle(cornerRadius: isSelected ? 16 : 24, style: .continuous)
        )
        .animation(.easeInOut(duration: 0.2), value: isSelected)
    }

    @ViewBuilder
    private var fallbackContent: some View {
        ZStack {
            Color(.tertiarySystemGroupedBackground)
            if let fallbackIcon {
                Image(systemName: fallbackIcon)
                    .font(.title3)
                    .foregroundStyle(tintColor)
            } else if let fallbackText {
                Text(fallbackText)
                    .font(.title3.weight(.semibold))
                    .foregroundStyle(tintColor)
            }
        }
    }
}

#Preview {
    ServerListView()
        .environment(AuthManager())
}
