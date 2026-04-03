import Foundation

struct User: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let username: String
    let email: String?
    let displayName: String?
    let avatar: String?
    let avatarUrl: String?
    let status: UserStatus?
    let flags: Int?
    let createdAt: Date?

    enum UserStatus: String, Codable, Hashable, Sendable {
        case online
        case idle
        case dnd
        case offline
        case invisible
    }

    enum CodingKeys: String, CodingKey {
        case id
        case username
        case email
        case displayName = "display_name"
        case avatar
        case avatarUrl = "avatar_url"
        case status
        case flags
        case createdAt = "created_at"
    }
}
