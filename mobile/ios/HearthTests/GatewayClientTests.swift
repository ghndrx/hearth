import XCTest
@testable import Hearth

final class GatewayClientTests: XCTestCase {

    // MARK: - GatewayOpcode

    func testGatewayOpcodeValues() {
        XCTAssertEqual(GatewayOpcode.dispatch.rawValue, 0)
        XCTAssertEqual(GatewayOpcode.heartbeat.rawValue, 1)
        XCTAssertEqual(GatewayOpcode.identify.rawValue, 2)
        XCTAssertEqual(GatewayOpcode.presenceUpdate.rawValue, 3)
        XCTAssertEqual(GatewayOpcode.voiceStateUpdate.rawValue, 4)
        XCTAssertEqual(GatewayOpcode.resume.rawValue, 6)
        XCTAssertEqual(GatewayOpcode.reconnect.rawValue, 7)
        XCTAssertEqual(GatewayOpcode.invalidSession.rawValue, 9)
        XCTAssertEqual(GatewayOpcode.hello.rawValue, 10)
        XCTAssertEqual(GatewayOpcode.heartbeatAck.rawValue, 11)
    }

    // MARK: - ConnectionState

    func testConnectionStateEquatable() {
        XCTAssertEqual(ConnectionState.disconnected, .disconnected)
        XCTAssertEqual(ConnectionState.connecting, .connecting)
        XCTAssertEqual(ConnectionState.connected, .connected)
        XCTAssertEqual(ConnectionState.reconnecting, .reconnecting)
    }

    func testConnectionStateDescription() {
        XCTAssertEqual(ConnectionState.disconnected.description, "disconnected")
        XCTAssertEqual(ConnectionState.connecting.description, "connecting")
        XCTAssertEqual(ConnectionState.connected.description, "connected")
        XCTAssertEqual(ConnectionState.reconnecting.description, "reconnecting")
    }

    // MARK: - GatewayMessage Encoding

    func testGatewayMessageHeartbeatEncoding() throws {
        let encoder = JSONEncoder()
        let message = GatewayMessage(op: .heartbeat, d: nil)
        let data = try encoder.encode(message)

        let decoded = try JSONDecoder().decode([String: AnyCodable?].self, from: data)
        XCTAssertEqual(decoded["op"]?.value as? Int, 1)
        XCTAssertNil(decoded["d"]?.value)
    }

    func testGatewayMessageWithDataEncoding() throws {
        let identifyData: [String: Any] = [
            "token": "test-token",
            "properties": [
                "os": "iOS",
                "browser": "Hearth iOS",
                "device": "Hearth iOS"
            ]
        ]

        let message = GatewayMessage(
            op: .identify,
            d: AnyCodable(identifyData)
        )

        let encoder = JSONEncoder()
        let data = try encoder.encode(message)

        let decoded = try JSONDecoder().decode([String: AnyCodable?].self, from: data)
        XCTAssertEqual(decoded["op"]?.value as? Int, 2)
        XCTAssertNotNil(decoded["d"]?.value)
    }

    func testGatewayMessageDispatchDecoding() throws {
        let json = """
        {
            "op": 0,
            "t": "MESSAGE_CREATE",
            "s": 42,
            "d": {
                "id": "123",
                "content": "Hello"
            }
        }
        """

        let decoder = JSONDecoder()
        let message = try decoder.decode(GatewayMessage.self, from: Data(json.utf8))

        XCTAssertEqual(message.op, .dispatch)
        XCTAssertEqual(message.t, "MESSAGE_CREATE")
        XCTAssertEqual(message.s, 42)
    }

    func testGatewayMessageHelloDecoding() throws {
        let json = """
        {
            "op": 10,
            "d": {
                "heartbeat_interval": 41250
            }
        }
        """

        let decoder = JSONDecoder()
        let message = try decoder.decode(GatewayMessage.self, from: Data(json.utf8))

        XCTAssertEqual(message.op, .hello)
        XCTAssertNotNil(message.d)
    }

    // MARK: - GatewayEvent

    func testGatewayEventProperties() {
        let event = GatewayEvent(
            type: "MESSAGE_CREATE",
            data: ["id": "123", "content": "test"],
            sequence: 42
        )

        XCTAssertEqual(event.type, "MESSAGE_CREATE")
        XCTAssertEqual(event.sequence, 42)
    }

    // MARK: - AnyCodable

    func testAnyCodableEncodesDictionary() throws {
        let dict: [String: Any] = ["key": "value", "num": 123]
        let any = AnyCodable(dict)

        let encoder = JSONEncoder()
        let data = try encoder.encode(any)
        let decoded = try JSONDecoder().decode([String: AnyCodable?].self, from: data)

        XCTAssertEqual(decoded["key"]?.value as? String, "value")
        XCTAssertEqual(decoded["num"]?.value as? Int, 123)
    }

    func testAnyCodableEncodesArray() throws {
        let arr: [Any] = ["one", 2, true]
        let any = AnyCodable(arr)

        let encoder = JSONEncoder()
        let data = try encoder.encode(any)
        let decoded = try JSONDecoder().decode([AnyCodable?].self, from: data)

        XCTAssertEqual(decoded[0]?.value as? String, "one")
        XCTAssertEqual(decoded[1]?.value as? Int, 2)
        XCTAssertEqual(decoded[2]?.value as? Bool, true)
    }
}
