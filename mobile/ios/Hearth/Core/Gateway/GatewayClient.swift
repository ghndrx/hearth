import Foundation

/// Events emitted by the gateway client to consumers.
enum GatewayEvent: Sendable {
    case connected
    case disconnected(Error?)
    case dispatch(eventName: String, data: [String: Any])
    case invalidSession(resumable: Bool)
}

actor GatewayClient {
    private let baseURL: URL
    private var webSocketTask: URLSessionWebSocketTask?
    private let session: URLSession

    private var token: String?
    private var sessionId: String?
    private var sequence: Int?
    private var heartbeatInterval: TimeInterval = 41.25
    private var heartbeatTask: Task<Void, Never>?
    private var receiveTask: Task<Void, Never>?
    private var lastHeartbeatAck = true
    private var reconnectAttempts = 0
    private let maxReconnectAttempts = 10
    private var isConnected = false

    private var eventContinuation: AsyncStream<GatewayEvent>.Continuation?

    /// An async stream of gateway events for consumers to observe.
    let events: AsyncStream<GatewayEvent>

    init(baseURL: URL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session

        var continuation: AsyncStream<GatewayEvent>.Continuation!
        self.events = AsyncStream { continuation = $0 }
        self.eventContinuation = continuation
    }

    deinit {
        eventContinuation?.finish()
    }

    // MARK: - Public API

    func connect(token: String) {
        self.token = token
        reconnectAttempts = 0
        establishConnection()
    }

    func disconnect() {
        heartbeatTask?.cancel()
        heartbeatTask = nil
        receiveTask?.cancel()
        receiveTask = nil
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        isConnected = false
        eventContinuation?.yield(.disconnected(nil))
    }

    func send<T: Encodable>(op: GatewayOpcode, data: T) async throws {
        let payload = GatewayOutbound(op: op.rawValue, d: data)
        let jsonData = try JSONEncoder().encode(payload)
        guard let string = String(data: jsonData, encoding: .utf8) else { return }
        try await webSocketTask?.send(.string(string))
    }

    func updatePresence(status: String, activities: [[String: Any]]? = nil) async throws {
        var payload: [String: Any] = ["status": status]
        if let activities {
            payload["activities"] = activities
        }
        let data = try JSONSerialization.data(withJSONObject: payload)
        let message = GatewayOutbound(
            op: GatewayOpcode.presenceUpdate.rawValue,
            d: AnyCodable(try JSONSerialization.jsonObject(with: data))
        )
        let encoded = try JSONEncoder().encode(message)
        guard let string = String(data: encoded, encoding: .utf8) else { return }
        try await webSocketTask?.send(.string(string))
    }

    var currentSequence: Int? {
        sequence
    }

    var currentSessionId: String? {
        sessionId
    }

    // MARK: - Connection Management

    private func establishConnection() {
        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("gateway"), resolvingAgainstBaseURL: true)
        if let token {
            urlComponents?.queryItems = [URLQueryItem(name: "token", value: token)]
        }

        guard let url = urlComponents?.url else { return }

        let task = session.webSocketTask(with: url)
        self.webSocketTask = task
        task.resume()

        receiveTask?.cancel()
        receiveTask = Task { [weak self] in
            await self?.receiveLoop()
        }
    }

    private func receiveLoop() async {
        guard let task = webSocketTask else { return }

        while !Task.isCancelled {
            do {
                let message = try await task.receive()
                switch message {
                case .string(let text):
                    if let data = text.data(using: .utf8) {
                        await handleMessage(data)
                    }
                case .data(let data):
                    await handleMessage(data)
                @unknown default:
                    break
                }
            } catch {
                if !Task.isCancelled {
                    await handleDisconnect(error: error)
                }
                return
            }
        }
    }

    // MARK: - Message Handling

    private func handleMessage(_ data: Data) async {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let opRaw = json["op"] as? Int,
              let op = GatewayOpcode(rawValue: opRaw) else {
            return
        }

        let d = json["d"]
        let s = json["s"] as? Int
        let t = json["t"] as? String

        if let s {
            sequence = s
        }

        switch op {
        case .hello:
            await handleHello(d: d)

        case .heartbeatAck:
            lastHeartbeatAck = true

        case .dispatch:
            await handleDispatch(eventName: t, data: d)

        case .heartbeat:
            await sendHeartbeat()

        case .reconnect:
            await attemptReconnect()

        case .invalidSession:
            let resumable = (d as? Bool) ?? false
            if !resumable {
                sessionId = nil
                sequence = nil
            }
            eventContinuation?.yield(.invalidSession(resumable: resumable))
            if resumable {
                try? await Task.sleep(nanoseconds: UInt64.random(in: 1_000_000_000...5_000_000_000))
                await attemptReconnect()
            } else {
                try? await Task.sleep(nanoseconds: UInt64.random(in: 1_000_000_000...5_000_000_000))
                establishConnection()
            }

        default:
            break
        }
    }

    private func handleHello(d: Any?) async {
        if let dict = d as? [String: Any],
           let interval = dict["heartbeat_interval"] as? Int {
            heartbeatInterval = Double(interval) / 1000.0
        }

        isConnected = true
        reconnectAttempts = 0
        lastHeartbeatAck = true
        eventContinuation?.yield(.connected)

        startHeartbeat()

        // Send IDENTIFY or RESUME
        if let sessionId, let sequence {
            await sendResume(sessionId: sessionId, sequence: sequence)
        } else {
            await sendIdentify()
        }
    }

    private func handleDispatch(eventName: String?, data: Any?) async {
        guard let eventName else { return }

        // Capture session_id from READY event
        if eventName == "READY", let dict = data as? [String: Any] {
            sessionId = dict["session_id"] as? String
        }

        let eventData = (data as? [String: Any]) ?? [:]
        eventContinuation?.yield(.dispatch(eventName: eventName, data: eventData))
    }

    private func handleDisconnect(error: Error?) async {
        isConnected = false
        heartbeatTask?.cancel()
        heartbeatTask = nil
        eventContinuation?.yield(.disconnected(error))
        await attemptReconnect()
    }

    // MARK: - Heartbeat

    private func startHeartbeat() {
        heartbeatTask?.cancel()
        heartbeatTask = Task { [weak self] in
            guard let self else { return }

            // Initial jitter: wait a random fraction of the heartbeat interval before the first heartbeat
            let jitterNanos = UInt64(Double.random(in: 0...1) * await self.heartbeatInterval * 1_000_000_000)
            try? await Task.sleep(nanoseconds: jitterNanos)

            while !Task.isCancelled {
                let interval = await self.heartbeatInterval
                let ackReceived = await self.lastHeartbeatAck

                if !ackReceived {
                    // Missed heartbeat ACK - reconnect
                    await self.attemptReconnect()
                    return
                }

                await self.sendHeartbeat()
                try? await Task.sleep(nanoseconds: UInt64(interval * 1_000_000_000))
            }
        }
    }

    private func sendHeartbeat() async {
        lastHeartbeatAck = false
        let payload: [String: Any?] = ["op": GatewayOpcode.heartbeat.rawValue, "d": sequence]
        guard let data = try? JSONSerialization.data(withJSONObject: payload.compactMapValues { $0 ?? NSNull() }),
              let string = String(data: data, encoding: .utf8) else {
            return
        }
        try? await webSocketTask?.send(.string(string))
    }

    // MARK: - Identify / Resume

    private func sendIdentify() async {
        guard let token else { return }

        let identify = GatewayIdentify(
            token: token,
            properties: GatewayIdentify.IdentifyProperties(
                os: "iOS",
                browser: "Hearth iOS",
                device: "iPhone"
            )
        )

        try? await send(op: .identify, data: identify)
    }

    private func sendResume(sessionId: String, sequence: Int) async {
        guard let token else { return }
        let resume = GatewayResume(token: token, sessionId: sessionId, seq: sequence)
        try? await send(op: .resume, data: resume)
    }

    // MARK: - Reconnect

    private func attemptReconnect() async {
        guard reconnectAttempts < maxReconnectAttempts else {
            eventContinuation?.yield(.disconnected(
                NSError(domain: "GatewayClient", code: -1,
                        userInfo: [NSLocalizedDescriptionKey: "Max reconnect attempts reached"])
            ))
            return
        }

        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        heartbeatTask?.cancel()
        heartbeatTask = nil

        reconnectAttempts += 1

        // Exponential backoff with jitter: base * 2^attempt + random jitter
        let base: Double = 1.0
        let backoff = base * pow(2.0, Double(min(reconnectAttempts, 6)))
        let jitter = Double.random(in: 0...backoff * 0.5)
        let delay = backoff + jitter

        try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))

        guard !Task.isCancelled else { return }
        establishConnection()
    }
}

// MARK: - Outbound payload helper

private struct GatewayOutbound<T: Encodable>: Encodable {
    let op: Int
    let d: T
}
