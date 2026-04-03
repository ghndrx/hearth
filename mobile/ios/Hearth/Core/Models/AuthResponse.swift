import Foundation

struct AuthResponse: Codable, Sendable {
    let token: String
    let refreshToken: String
    let user: User

    enum CodingKeys: String, CodingKey {
        case token
        case refreshToken = "refresh_token"
        case user
    }
}

struct LoginRequest: Codable, Sendable {
    let email: String
    let password: String
}

struct RegisterRequest: Codable, Sendable {
    let email: String
    let username: String
    let password: String
}

struct RefreshTokenRequest: Codable, Sendable {
    let refreshToken: String

    enum CodingKeys: String, CodingKey {
        case refreshToken = "refresh_token"
    }
}

struct SendMessageRequest: Codable, Sendable {
    let content: String
}

struct CreateDMRequest: Codable, Sendable {
    let recipientId: String

    enum CodingKeys: String, CodingKey {
        case recipientId = "recipient_id"
    }
}
