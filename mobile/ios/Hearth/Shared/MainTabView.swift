import SwiftUI

struct MainTabView: View {
    @State private var selectedTab: Tab = .servers

    enum Tab: String, CaseIterable {
        case servers
        case dms
        case notifications
        case profile
    }

    var body: some View {
        TabView(selection: $selectedTab) {
            ServerListView()
                .tabItem {
                    Label("Servers", systemImage: "server.rack")
                }
                .tag(Tab.servers)

            DMListView()
                .tabItem {
                    Label("Messages", systemImage: "bubble.left.and.bubble.right.fill")
                }
                .tag(Tab.dms)

            NotificationsPlaceholderView()
                .tabItem {
                    Label("Notifications", systemImage: "bell.fill")
                }
                .tag(Tab.notifications)

            ProfileView()
                .tabItem {
                    Label("Profile", systemImage: "person.fill")
                }
                .tag(Tab.profile)
        }
        .tint(.orange)
    }
}

struct NotificationsPlaceholderView: View {
    var body: some View {
        NavigationStack {
            ContentUnavailableView(
                "No Notifications",
                systemImage: "bell.slash",
                description: Text("You're all caught up! Notifications will appear here.")
            )
            .navigationTitle("Notifications")
        }
    }
}

#Preview {
    MainTabView()
        .environment(AuthManager())
}
