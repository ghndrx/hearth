import Foundation
import SwiftUI

private let kAccessTokenKey = "hearth_access_token"
private let kRefreshTokenKey = "hearth_refresh_token"
private let kCurrentUserKey = "hearth_current_user"

@Observable
final class AuthManager: @unchecked Sendable {
    private(set) var currentUser: User?
    private(set) var isLoading = false
    var error: String?

    private let apiClient: APIClient
    private let lock = NSLock()

    var accessToken: String? {
        KeychainHelper.readString(key: kAccessTokenKey)
    }

    var refreshTokenValue: String? {
        KeychainHelper.readString(key: kRefreshTokenKey)
    }

    var isAuthenticated: Bool {
        accessToken != nil && currentUser != nil
    }

    init(baseURL: URL = URL(string: "https://api.hearth.hndrx.co")!) {
        self.apiClient = APIClient(baseURL: baseURL)

        // Restore user from persisted data if tokens exist
        if let token = KeychainHelper.readString(key: kAccessTokenKey),
           !token.isEmpty,
           let userData = KeychainHelper.read(key: kCurrentUserKey) {
            self.currentUser = try? JSONDecoder().decode(User.self, from: userData)
        }

        Task { [weak self] in
            await self?.apiClient.setAuthManager(self!)
        }
    }

    /// For dependency injection in tests.
    init(apiClient: APIClient) {
        self.apiClient = apiClient
        Task { [weak self] in
            guard let self else { return }
            await apiClient.setAuthManager(self)
        }
    }

    // MARK: - Auth Actions

    @MainActor
    func login(email: String, password: String) async throws -> User {
        isLoading = true
        error = nil
        defer { isLoading = false }

        do {
            let response = try await apiClient.login(email: email, password: password)
            storeTokens(access: response.token, refresh: response.refreshToken)
            persistUser(response.user)
            currentUser = response.user
            return response.user
        } catch {
            self.error = error.localizedDescription
            throw error
        }
    }

    @MainActor
    func register(email: String, username: String, password: String) async throws -> User {
        isLoading = true
        error = nil
        defer { isLoading = false }

        do {
            let response = try await apiClient.register(
                email: email, username: username, password: password
            )
            storeTokens(access: response.token, refresh: response.refreshToken)
            persistUser(response.user)
            currentUser = response.user
            return response.user
        } catch {
            self.error = error.localizedDescription
            throw error
        }
    }

    func refreshToken() async throws {
        guard let token = refreshTokenValue else {
            throw APIError.tokenRefreshFailed
        }

        let response = try await apiClient.refreshToken(token: token)
        storeTokens(access: response.token, refresh: response.refreshToken)
        persistUser(response.user)

        await MainActor.run {
            self.currentUser = response.user
        }
    }

    @MainActor
    func logout() {
        KeychainHelper.delete(key: kAccessTokenKey)
        KeychainHelper.delete(key: kRefreshTokenKey)
        KeychainHelper.delete(key: kCurrentUserKey)
        currentUser = nil
        error = nil
    }

    @MainActor
    func fetchCurrentUser() async throws {
        let user = try await apiClient.fetchCurrentUser()
        persistUser(user)
        currentUser = user
    }

    // MARK: - Helpers

    private func storeTokens(access: String, refresh: String) {
        KeychainHelper.saveString(key: kAccessTokenKey, value: access)
        KeychainHelper.saveString(key: kRefreshTokenKey, value: refresh)
    }

    private func persistUser(_ user: User) {
        if let data = try? JSONEncoder().encode(user) {
            KeychainHelper.save(key: kCurrentUserKey, data: data)
        }
    }
}
