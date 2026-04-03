import Foundation

enum APIError: Error, LocalizedError {
    case invalidURL
    case invalidResponse
    case httpError(statusCode: Int, body: Data?)
    case decodingError(Error)
    case encodingError(Error)
    case unauthorized
    case tokenRefreshFailed
    case networkError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid URL"
        case .invalidResponse:
            return "Invalid server response"
        case .httpError(let code, _):
            return "HTTP error \(code)"
        case .decodingError(let error):
            return "Failed to decode response: \(error.localizedDescription)"
        case .encodingError(let error):
            return "Failed to encode request: \(error.localizedDescription)"
        case .unauthorized:
            return "Unauthorized"
        case .tokenRefreshFailed:
            return "Session expired. Please log in again."
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        }
    }
}

actor APIClient {
    private let session: URLSession
    private let baseURL: URL
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private weak var authManager: AuthManager?
    private var isRefreshing = false

    init(
        baseURL: URL,
        session: URLSession = .shared,
        authManager: AuthManager? = nil
    ) {
        self.baseURL = baseURL
        self.session = session
        self.authManager = authManager

        self.decoder = JSONDecoder()
        self.decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let dateString = try container.decode(String.self)

            let iso8601 = ISO8601DateFormatter()
            iso8601.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            if let date = iso8601.date(from: dateString) {
                return date
            }

            iso8601.formatOptions = [.withInternetDateTime]
            if let date = iso8601.date(from: dateString) {
                return date
            }

            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Cannot decode date: \(dateString)"
            )
        }

        self.encoder = JSONEncoder()
        self.encoder.dateEncodingStrategy = .iso8601
        self.encoder.keyEncodingStrategy = .convertToSnakeCase
    }

    func setAuthManager(_ manager: AuthManager) {
        self.authManager = manager
    }

    // MARK: - Generic Request

    func request<T: Decodable>(
        method: String,
        path: String,
        body: (any Encodable)? = nil,
        queryItems: [URLQueryItem]? = nil,
        authenticated: Bool = true,
        retryOnUnauthorized: Bool = true
    ) async throws -> T {
        let url = try buildURL(path: path, queryItems: queryItems)
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.setValue("application/json", forHTTPHeaderField: "Accept")

        if authenticated, let token = await authManager?.accessToken {
            urlRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body {
            do {
                urlRequest.httpBody = try encoder.encode(AnyEncodable(body))
            } catch {
                throw APIError.encodingError(error)
            }
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: urlRequest)
        } catch {
            throw APIError.networkError(error)
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }

        if httpResponse.statusCode == 401 && retryOnUnauthorized && authenticated {
            let refreshed = await refreshTokenIfNeeded()
            if refreshed {
                return try await request(
                    method: method,
                    path: path,
                    body: body,
                    queryItems: queryItems,
                    authenticated: true,
                    retryOnUnauthorized: false
                )
            } else {
                throw APIError.tokenRefreshFailed
            }
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            throw APIError.httpError(statusCode: httpResponse.statusCode, body: data)
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingError(error)
        }
    }

    @discardableResult
    func requestVoid(
        method: String,
        path: String,
        body: (any Encodable)? = nil,
        queryItems: [URLQueryItem]? = nil,
        authenticated: Bool = true
    ) async throws -> Data {
        let url = try buildURL(path: path, queryItems: queryItems)
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method
        urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if authenticated, let token = await authManager?.accessToken {
            urlRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body {
            urlRequest.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let (data, response) = try await session.data(for: urlRequest)

        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.invalidResponse
        }

        return data
    }

    // MARK: - Auth Endpoints

    func login(email: String, password: String) async throws -> AuthResponse {
        try await request(
            method: "POST",
            path: "/api/auth/login",
            body: LoginRequest(email: email, password: password),
            authenticated: false
        )
    }

    func register(email: String, username: String, password: String) async throws -> AuthResponse {
        try await request(
            method: "POST",
            path: "/api/auth/register",
            body: RegisterRequest(email: email, username: username, password: password),
            authenticated: false
        )
    }

    func refreshToken(token: String) async throws -> AuthResponse {
        try await request(
            method: "POST",
            path: "/api/auth/refresh",
            body: RefreshTokenRequest(refreshToken: token),
            authenticated: false,
            retryOnUnauthorized: false
        )
    }

    func fetchCurrentUser() async throws -> User {
        try await request(method: "GET", path: "/api/users/@me")
    }

    // MARK: - Server Endpoints

    func fetchServers() async throws -> [Server] {
        try await request(method: "GET", path: "/api/users/@me/servers")
    }

    func fetchChannels(serverId: String) async throws -> [Channel] {
        try await request(method: "GET", path: "/api/servers/\(serverId)/channels")
    }

    // MARK: - Message Endpoints

    func fetchMessages(
        channelId: String,
        before: String? = nil,
        limit: Int = 50
    ) async throws -> [Message] {
        var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
        if let before {
            queryItems.append(URLQueryItem(name: "before", value: before))
        }
        return try await request(
            method: "GET",
            path: "/api/channels/\(channelId)/messages",
            queryItems: queryItems
        )
    }

    func sendMessage(channelId: String, content: String) async throws -> Message {
        try await request(
            method: "POST",
            path: "/api/channels/\(channelId)/messages",
            body: SendMessageRequest(content: content)
        )
    }

    // MARK: - DM Endpoints

    func fetchDMs() async throws -> [Channel] {
        try await request(method: "GET", path: "/api/users/@me/channels")
    }

    func createDM(recipientId: String) async throws -> Channel {
        try await request(
            method: "POST",
            path: "/api/users/@me/channels",
            body: CreateDMRequest(recipientId: recipientId)
        )
    }

    // MARK: - Private Helpers

    private func buildURL(path: String, queryItems: [URLQueryItem]? = nil) throws -> URL {
        var components = URLComponents(url: baseURL.appendingPathComponent(path), resolvingAgainstBaseURL: true)
        components?.queryItems = queryItems

        guard let url = components?.url else {
            throw APIError.invalidURL
        }
        return url
    }

    private func refreshTokenIfNeeded() async -> Bool {
        guard !isRefreshing else { return false }
        isRefreshing = true
        defer { isRefreshing = false }

        guard let manager = authManager else { return false }

        do {
            try await manager.refreshToken()
            return true
        } catch {
            await manager.logout()
            return false
        }
    }
}

/// Type-erased Encodable wrapper for encoding heterogeneous body types.
private struct AnyEncodable: Encodable {
    private let _encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        self._encode = { encoder in
            try value.encode(to: encoder)
        }
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}
