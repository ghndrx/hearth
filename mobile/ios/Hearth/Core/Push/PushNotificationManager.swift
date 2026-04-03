import Foundation
import UserNotifications
import UIKit

/// Manages the APNs push notification lifecycle on iOS.
///
/// Handles:
/// - Authorization requests (provisional on iOS 12+, upgrade to full)
/// - Device token management and server registration
/// - Incoming notification processing
/// - Badge count management
/// - Notification categories and actions
/// - Background notification handling
@MainActor
final class PushNotificationManager: NSObject, @unchecked Sendable {

    // MARK: - Singleton

    static let shared = PushNotificationManager()

    // MARK: - Published State

    private(set) var isAuthorized = false
    private(set) var deviceToken: String?
    private(set) var authorizationStatus: UNAuthorizationStatus = .notDetermined

    // MARK: - Dependencies

    private var apiClient: APIClient?
    private var authManager: AuthManager?

    // MARK: - Notification Categories

    /// Actions supported from notification UI.
    enum Category: String {
        case message = "MESSAGE"
        case mention = "MENTION"
        case dm = "DM"
        case serverActivity = "SERVER_ACTIVITY"
    }

    enum Action: String {
        case reply = "REPLY_ACTION"
        case markRead = "MARK_READ_ACTION"
    }

    // MARK: - Setup

    /// Call once from HearthApp when the app starts.
    func configure(apiClient: APIClient, authManager: AuthManager) {
        self.apiClient = apiClient
        self.authManager = authManager

        // Set up notification center delegate
        UNUserNotificationCenter.current().delegate = self

        // Register notification categories with actions
        registerCategories()

        // Check current authorization status
        Task {
            await refreshAuthorizationStatus()
        }
    }

    /// Call when user has just authenticated. Attempts to register for push.
    func registerIfAuthenticated() {
        guard authManager?.isAuthenticated == true else { return }
        Task {
            await requestAuthorization()
            await registerForRemoteNotifications()
        }
    }

    /// Call on logout. Unregisters device token from backend and clears badge.
    func unregister() {
        guard let token = deviceToken else { return }
        Task {
            do {
                try await apiClient?.deletePushToken(token: token)
            } catch {
                // Best-effort — don't block logout on push failure
            }
            deviceToken = nil
            await clearBadge()
        }
    }

    // MARK: - Authorization

    /// Refreshes and publishes the current authorization status from the system.
    func refreshAuthorizationStatus() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        authorizationStatus = settings.authorizationStatus
        isAuthorized = settings.authorizationStatus == .authorized ||
                       settings.authorizationStatus == .provisional
    }

    /// Requests notification authorization.
    ///
    /// Strategy:
    /// 1. iOS 12+ — request `.provisional` first so we can send silent background updates
    ///    without bothering the user with a permission dialog.
    /// 2. After the first notification arrives, iOS will prompt for full authorization.
    /// 3. We then call `upgradeToFullAuthorization()` to move to `.authorized`.
    private func requestAuthorization() async {
        // Already fully authorized
        guard authorizationStatus != .authorized else {
            isAuthorized = true
            return
        }

        // Already denied — don't prompt again
        guard authorizationStatus != .denied else {
            isAuthorized = false
            return
        }

        let provisional = await UNUserNotificationCenter.current().requestAuthorization(
            options: [.provisional, .badge, .sound, .alert]
        )

        await refreshAuthorizationStatus()

        if provisional {
            // iOS 12+ with provisional — we can send background updates
            isAuthorized = true
        }
    }

    /// Upgrades from `.provisional` to full `.authorized`.
    /// Call when the user has interacted positively with a notification.
    func upgradeToFullAuthorization() async {
        guard authorizationStatus == .provisional else { return }

        let granted = await UNUserNotificationCenter.current().requestAuthorization(
            options: [.badge, .sound, .alert]
        )
        await refreshAuthorizationStatus()
        isAuthorized = granted
    }

    // MARK: - Device Token

    /// Called by the system when a device token is received from APNs.
    /// Also called on token refresh.
    func didRegisterForRemoteNotifications(deviceToken: Data) {
        let tokenString = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        self.deviceToken = tokenString
        Task {
            await sendTokenToServer(token: tokenString)
        }
    }

    /// Called when APNs registration fails.
    func didFailToRegisterForRemoteNotifications(error: Error) {
        print("[PushNotificationManager] Failed to register for remote notifications: \(error)")
    }

    /// Registers the device with APNs to receive push notifications.
    /// Safe to call multiple times — the system is idempotent.
    private func registerForRemoteNotifications() async {
        guard isAuthorized || authorizationStatus == .provisional else { return }

        await MainActor.run {
            UIApplication.shared.registerForRemoteNotifications()
        }
    }

    /// Sends the APNs device token to the Hearth backend.
    /// Idempotent — the server will upsert.
    private func sendTokenToServer(token: String) async {
        guard let client = apiClient else { return }
        guard authManager?.isAuthenticated == true else { return }

        do {
            let request = PushTokenRequest(
                token: token,
                platform: "ios",
                deviceId: Self.deviceIdentifier,
                appVersion: Self.appVersion
            )
            try await client.registerPushToken(request)
        } catch {
            print("[PushNotificationManager] Failed to register token with backend: \(error)")
        }
    }

    // MARK: - Badge

    /// Clears the app badge to 0.
    func clearBadge() async {
        await MainActor.run {
            UNUserNotificationCenter.current().setBadgeCount(0)
        }
    }

    /// Sets the app badge to the given count.
    func setBadge(_ count: Int) async {
        await MainActor.run {
            UNUserNotificationCenter.current().setBadgeCount(count)
        }
    }

    // MARK: - Background Refresh

    /// Called when the app receives a background notification.
    /// Triggers a badge count refresh from the backend.
    func handleBackgroundNotification() async {
        await refreshUnreadCount()
    }

    /// Fetches the unread notification count from the backend and updates the badge.
    private func refreshUnreadCount() async {
        guard let client = apiClient else { return }

        do {
            let count = try await client.fetchUnreadNotificationCount()
            await setBadge(count)
        } catch {
            print("[PushNotificationManager] Failed to refresh unread count: \(error)")
        }
    }

    // MARK: - Categories

    private func registerCategories() {
        // Message actions
        let replyAction = UNTextInputNotificationAction(
            identifier: Action.reply.rawValue,
            title: "Reply",
            options: [],
            textInputButtonTitle: "Send",
            textInputPlaceholder: "Message..."
        )

        let markReadAction = UNNotificationAction(
            identifier: Action.markRead.rawValue,
            title: "Mark Read",
            options: [.destructive]
        )

        let messageCategory = UNNotificationCategory(
            identifier: Category.message.rawValue,
            actions: [replyAction, markReadAction],
            intentIdentifiers: [],
            options: [.customDismissAction]
        )

        let mentionCategory = UNNotificationCategory(
            identifier: Category.mention.rawValue,
            actions: [replyAction, markReadAction],
            intentIdentifiers: [],
            options: []
        )

        let dmCategory = UNNotificationCategory(
            identifier: Category.dm.rawValue,
            actions: [replyAction],
            intentIdentifiers: [],
            options: []
        )

        let serverActivityCategory = UNNotificationCategory(
            identifier: Category.serverActivity.rawValue,
            actions: [],
            intentIdentifiers: [],
            options: []
        )

        UNUserNotificationCenter.current().setNotificationCategories([
            messageCategory,
            mentionCategory,
            dmCategory,
            serverActivityCategory,
        ])
    }

    // MARK: - Helpers

    private static var deviceIdentifier: String {
        let key = "hearth_device_id"
        if let existing = UserDefaults.standard.string(forKey: key) {
            return existing
        }
        let newId = UUID().uuidString
        UserDefaults.standard.set(newId, forKey: key)
        return newId
    }

    private static var appVersion: String {
        Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
    }
}

// MARK: - UNUserNotificationCenterDelegate

/// Handles notification delivery while the app is in the foreground
/// and notification interaction callbacks.
extension PushNotificationManager: UNUserNotificationCenterDelegate {

    /// Called when a notification is delivered while the app is in the foreground.
    /// We show it manually via the system's notification center presentation.
    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        // Always show banner + sound when app is in foreground
        return [.banner, .sound, .badge]
    }

    /// Called when the user interacts with a delivered notification.
    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        let userInfo = response.notification.request.content.userInfo

        switch response.actionIdentifier {
        case Action.reply.rawValue:
            // User typed a quick reply — this would be handled by a
            // chat-focused implementation (future voice/DM feature)
            break

        case Action.markRead.rawValue:
            // Mark channel/message as read — refresh badge
            await handleBackgroundNotification()

        case UNNotificationDefaultActionIdentifier:
            // User tapped the notification itself — navigate to the relevant channel
            await navigateToNotificationDestination(userInfo: userInfo)

        case UNNotificationDismissActionIdentifier:
            // User dismissed — mark as read
            await handleBackgroundNotification()

        default:
            break
        }
    }

    private func navigateToNotificationDestination(userInfo: [AnyHashable: Any]) async {
        // Extract hearth data from userInfo
        guard let hearthData = userInfo["hearth"] as? [String: Any],
              let type = hearthData["type"] as? String else {
            return
        }

        // Post a notification that the navigation layer can observe
        // to navigate to the correct screen. This decouples the push
        // manager from the navigation layer.
        let navEvent = PushNavigationEvent(
            type: type,
            channelId: hearthData["channelId"] as? String,
            serverId: hearthData["serverId"] as? String,
            messageId: hearthData["messageId"] as? String
        )
        await MainActor.run {
            NotificationCenter.default.post(
                name: .pushNotificationTapped,
                object: nil,
                userInfo: ["event": navEvent]
            )
        }
    }
}

// MARK: - Supporting Types

struct PushTokenRequest: Encodable {
    let token: String
    let platform: String
    let deviceId: String
    let appVersion: String
}

struct PushNavigationEvent: Sendable {
    let type: String
    let channelId: String?
    let serverId: String?
    let messageId: String?
}

extension Notification.Name {
    static let pushNotificationTapped = Notification.Name("hearth.pushNotificationTapped")
}
