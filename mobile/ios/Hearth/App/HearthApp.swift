import SwiftUI

@main
struct HearthApp: App {
    @State private var authManager = AuthManager()

    init() {
        // Configure push notification manager with dependencies.
        // This must happen before the first view is rendered.
        PushNotificationManager.shared.configure(
            apiClient: APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!),
            authManager: authManager
        )
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if authManager.isAuthenticated {
                    MainTabView()
                        .environment(authManager)
                        // Trigger push registration after successful auth
                        .onAppear {
                            PushNotificationManager.shared.registerIfAuthenticated()
                        }
                } else {
                    LoginView()
                        .environment(authManager)
                }
            }
            .onReceive(NotificationCenter.default.publisher(for: .pushNotificationTapped)) { notification in
                // Handle navigation to the tapped notification's destination
                if let event = notification.userInfo?["event"] as? PushNavigationEvent {
                    handlePushNavigation(event: event)
                }
            }
        }
    }

    private func handlePushNavigation(event: PushNavigationEvent) {
        // The MainTabView or its child views observe this notification
        // and navigate to the appropriate channel/DM screen.
        // Full implementation will be wired up in the Navigation feature.
        print("[HearthApp] Push navigation: type=\(event.type) channelId=\(event.channelId ?? "nil")")
    }
}
