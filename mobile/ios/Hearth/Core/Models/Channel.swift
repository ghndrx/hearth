import Foundation

struct Channel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let serverId: String?
    let name: String
    let type: ChannelType
    let topic: String?
    let position: Int?
    let nsfw: Bool?
    let lastMessageId: String?
    let createdAt: Date?

    enum ChannelType: String, Codable, Hashable, Sendable {
        case text
        case voice
        case category
        case dm
        case groupDm = "group_dm"
        case announcement
        case forum
        case stage
    }

    enum CodingKeys: String, CodingKey {
        case id
        case serverId = "server_id"
        case name
        case type
        case topic
        case position
        case nsfw
        case lastMessageId = "last_message_id"
        case createdAt = "created_at"
    }
}
