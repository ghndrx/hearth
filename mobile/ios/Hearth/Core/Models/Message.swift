import Foundation

struct Message: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let channelId: String
    let authorId: String
    let content: String
    let type: Int?
    let replyTo: String?
    let pinned: Bool?
    let tts: Bool?
    let flags: Int?
    let createdAt: Date?
    let editedAt: Date?
    let author: User?
    let attachments: [Attachment]?
    let reactions: [Reaction]?

    enum CodingKeys: String, CodingKey {
        case id
        case channelId = "channel_id"
        case authorId = "author_id"
        case content
        case type
        case replyTo = "reply_to"
        case pinned
        case tts
        case flags
        case createdAt = "created_at"
        case editedAt = "edited_at"
        case author
        case attachments
        case reactions
    }
}

struct Attachment: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let filename: String
    let contentType: String?
    let size: Int
    let url: String
    let proxyUrl: String?
    let width: Int?
    let height: Int?

    enum CodingKeys: String, CodingKey {
        case id
        case filename
        case contentType = "content_type"
        case size
        case url
        case proxyUrl = "proxy_url"
        case width
        case height
    }
}

struct Reaction: Codable, Hashable, Sendable {
    let emoji: String
    let count: Int
    let me: Bool
}
