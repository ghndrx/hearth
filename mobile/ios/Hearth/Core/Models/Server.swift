import Foundation

struct Server: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let iconUrl: String?
    let bannerUrl: String?
    let description: String?
    let ownerId: String
    let features: [String]?
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case iconUrl = "icon_url"
        case bannerUrl = "banner_url"
        case description
        case ownerId = "owner_id"
        case features
        case createdAt = "created_at"
    }
}
