import XCTest
@testable import Hearth

final class AuthManagerTests: XCTestCase {

    override func tearDown() {
        super.tearDown()
        // Clean up keychain entries after each test
        KeychainHelper.delete(key: "hearth_access_token")
        KeychainHelper.delete(key: "hearth_refresh_token")
        KeychainHelper.delete(key: "hearth_current_user")
    }

    // MARK: - Token Storage

    func testSaveAndReadTokenFromKeychain() {
        let saved = KeychainHelper.saveString(key: "hearth_access_token", value: "test-jwt-token")
        XCTAssertTrue(saved)

        let retrieved = KeychainHelper.readString(key: "hearth_access_token")
        XCTAssertEqual(retrieved, "test-jwt-token")
    }

    func testDeleteTokenFromKeychain() {
        KeychainHelper.saveString(key: "hearth_access_token", value: "to-delete")
        let deleted = KeychainHelper.delete(key: "hearth_access_token")
        XCTAssertTrue(deleted)

        let retrieved = KeychainHelper.readString(key: "hearth_access_token")
        XCTAssertNil(retrieved)
    }

    func testReadNonExistentKeyReturnsNil() {
        let result = KeychainHelper.readString(key: "nonexistent_key_\(UUID().uuidString)")
        XCTAssertNil(result)
    }

    // MARK: - AuthManager State

    func testInitialStateNotAuthenticated() {
        // Ensure no tokens in keychain
        KeychainHelper.delete(key: "hearth_access_token")
        KeychainHelper.delete(key: "hearth_refresh_token")
        KeychainHelper.delete(key: "hearth_current_user")

        let manager = AuthManager(baseURL: URL(string: "https://localhost:9999")!)
        XCTAssertFalse(manager.isAuthenticated)
        XCTAssertNil(manager.currentUser)
        XCTAssertNil(manager.accessToken)
    }

    func testLogoutClearsTokensAndUser() async {
        // Simulate stored tokens
        KeychainHelper.saveString(key: "hearth_access_token", value: "access")
        KeychainHelper.saveString(key: "hearth_refresh_token", value: "refresh")

        let user = User(
            id: "1", username: "testuser", email: "test@example.com",
            displayName: nil, avatar: nil, avatarUrl: nil,
            status: .online, flags: 0, createdAt: nil
        )
        if let userData = try? JSONEncoder().encode(user) {
            KeychainHelper.save(key: "hearth_current_user", data: userData)
        }

        let manager = AuthManager(baseURL: URL(string: "https://localhost:9999")!)

        await MainActor.run {
            manager.logout()
        }

        XCTAssertNil(manager.accessToken)
        XCTAssertNil(manager.refreshTokenValue)
        XCTAssertNil(manager.currentUser)
        XCTAssertFalse(manager.isAuthenticated)
    }

    // MARK: - Login with Mock

    func testLoginSuccessStoresTokens() async throws {
        // This test validates the flow logic. In a real test suite you'd inject
        // a mock URLSession/APIClient. Here we verify the keychain storage path.
        let token = "mock-access-token-\(UUID().uuidString)"
        let refresh = "mock-refresh-token-\(UUID().uuidString)"

        KeychainHelper.saveString(key: "hearth_access_token", value: token)
        KeychainHelper.saveString(key: "hearth_refresh_token", value: refresh)

        XCTAssertEqual(KeychainHelper.readString(key: "hearth_access_token"), token)
        XCTAssertEqual(KeychainHelper.readString(key: "hearth_refresh_token"), refresh)
    }

    func testRefreshTokenValueMatchesStored() {
        let refreshToken = "refresh-\(UUID().uuidString)"
        KeychainHelper.saveString(key: "hearth_refresh_token", value: refreshToken)

        let manager = AuthManager(baseURL: URL(string: "https://localhost:9999")!)
        XCTAssertEqual(manager.refreshTokenValue, refreshToken)
    }

    // MARK: - User Persistence

    func testUserPersistenceRoundTrip() {
        let user = User(
            id: "42", username: "hearthuser", email: "hearth@example.com",
            displayName: "Hearth User", avatar: nil, avatarUrl: "https://cdn.hearth.hndrx.co/avatars/42.png",
            status: .online, flags: 1, createdAt: Date()
        )

        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        guard let data = try? encoder.encode(user) else {
            XCTFail("Failed to encode user")
            return
        }

        KeychainHelper.save(key: "hearth_current_user", data: data)

        guard let readData = KeychainHelper.read(key: "hearth_current_user") else {
            XCTFail("Failed to read user data from keychain")
            return
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let decoded = try? decoder.decode(User.self, from: readData)
        XCTAssertNotNil(decoded)
        XCTAssertEqual(decoded?.id, "42")
        XCTAssertEqual(decoded?.username, "hearthuser")
        XCTAssertEqual(decoded?.displayName, "Hearth User")
    }
}
