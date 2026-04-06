import Foundation

struct GatewayMessage: Codable, Sendable {
    let op: GatewayOpcode
    let d: AnyCodable?
    let s: Int?
    let t: String?
}

enum GatewayOpcode: Int, Codable, Sendable {
    case dispatch = 0
    case heartbeat = 1
    case identify = 2
    case presenceUpdate = 3
    case voiceStateUpdate = 4
    case resume = 6
    case reconnect = 7
    case invalidSession = 9
    case hello = 10
    case heartbeatAck = 11
}

struct GatewayHello: Codable, Sendable {
    let heartbeatInterval: Int

    enum CodingKeys: String, CodingKey {
        case heartbeatInterval = "heartbeat_interval"
    }
}

struct GatewayIdentify: Codable, Sendable {
    let token: String
    let properties: IdentifyProperties

    struct IdentifyProperties: Codable, Sendable {
        let os: String
        let browser: String
        let device: String
    }
}

struct GatewayResume: Codable, Sendable {
    let token: String
    let sessionId: String
    let seq: Int

    enum CodingKeys: String, CodingKey {
        case token
        case sessionId = "session_id"
        case seq
    }
}

/// A type-erased Codable value for gateway message payloads.
struct AnyCodable: Codable, Hashable, Sendable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() {
            value = NSNull()
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map(\.value)
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues(\.value)
        } else {
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Unsupported AnyCodable value"
            )
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()

        switch value {
        case is NSNull:
            try container.encodeNil()
        case let bool as Bool:
            try container.encode(bool)
        case let int as Int:
            try container.encode(int)
        case let double as Double:
            try container.encode(double)
        case let string as String:
            try container.encode(string)
        case let array as [Any]:
            try container.encode(array.map { AnyCodable($0) })
        case let dict as [String: Any]:
            try container.encode(dict.mapValues { AnyCodable($0) })
        default:
            throw EncodingError.invalidValue(
                value,
                EncodingError.Context(
                    codingPath: encoder.codingPath,
                    debugDescription: "Unsupported AnyCodable value"
                )
            )
        }
    }

    static func == (lhs: AnyCodable, rhs: AnyCodable) -> Bool {
        String(describing: lhs.value) == String(describing: rhs.value)
    }

    func hash(into hasher: inout Hasher) {
        hasher.combine(String(describing: value))
    }

    /// Convenience to extract the underlying value as a dictionary.
    var dictionary: [String: Any]? {
        value as? [String: Any]
    }
}
