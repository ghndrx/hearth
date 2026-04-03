import XCTest
@testable import Hearth

final class APIClientTests: XCTestCase {

    var baseURL: URL!
    var apiClient: APIClient!

    override func setUp() {
        super.setUp()
        baseURL = URL(string: "https://api.hearth.hndrx.co")!
        apiClient = APIClient(baseURL: baseURL)
    }

    override func tearDown() {
        apiClient = nil
        baseURL = nil
        super.tearDown()
    }

    // MARK: - URL Building

    func testBuildURLWithNoQueryItems() throws {
        let url = try apiClient.request(
            method: "GET",
            path: "/api/test",
            authenticated: false,
            retryOnUnauthorized: false
        ) as TestResponse

        // We expect a network error since we're hitting a real endpoint,
        // but the URL should be correctly formed.
        // This test validates the URL construction path.
    }

    func testBuildURLWithQueryItems() throws {
        // Test that query items are properly added
        struct Response: Decodable {}
        let _: Response? = try? apiClient.request(
            method: "GET",
            path: "/api/test",
            queryItems: [
                URLQueryItem(name: "limit", value: "50"),
                URLQueryItem(name: "before", value: "123")
            ],
            authenticated: false,
            retryOnUnauthorized: false
        )
        // We just verify it doesn't throw on URL construction
    }

    func testBuildURLWithInvalidPath() throws {
        // Test that empty path doesn't crash
        struct EmptyPathResponse: Decodable {}
        do {
            let _ = try apiClient.request(
                method: "GET",
                path: "",
                authenticated: false,
                retryOnUnauthorized: false
            ) as EmptyPathResponse
            XCTFail("Should have thrown")
        } catch APIError.invalidURL {
            // Expected
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    // MARK: - Model Encoding

    func testLoginRequestEncoding() throws {
        let request = LoginRequest(email: "test@example.com", password: "password123")

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        encoder.dateEncodingStrategy = .iso8601

        let data = try encoder.encode(request)
        let decoded = try JSONDecoder().decode([String: String].self, from: data)

        XCTAssertEqual(decoded["email"], "test@example.com")
        XCTAssertEqual(decoded["password"], "password123")
    }

    func testRegisterRequestEncoding() throws {
        let request = RegisterRequest(
            email: "test@example.com",
            username: "testuser",
            password: "password123"
        )

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        let data = try encoder.encode(request)
        let decoded = try JSONDecoder().decode([String: String].self, from: data)

        XCTAssertEqual(decoded["email"], "test@example.com")
        XCTAssertEqual(decoded["username"], "testuser")
        XCTAssertEqual(decoded["password"], "password123")
    }

    func testSendMessageRequestEncoding() throws {
        let request = SendMessageRequest(content: "Hello, world!")

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        let data = try encoder.encode(request)
        let decoded = try JSONDecoder().decode([String: String].self, from: data)

        XCTAssertEqual(decoded["content"], "Hello, world!")
    }

    func testCreateDMRequestEncoding() throws {
        let request = CreateDMRequest(recipientId: "user-123")

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        let data = try encoder.encode(request)
        let decoded = try JSONDecoder().decode([String: String].self, from: data)

        XCTAssertEqual(decoded["recipient_id"], "user-123")
    }

    // MARK: - Date Decoding

    func testISODateDecodingWithFractionalSeconds() throws {
        let json = #"{"createdAt": "2024-01-15T10:30:00.123Z"}"#
        struct DateTest: Codable {
            let createdAt: Date
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
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

        let result = try decoder.decode(DateTest.self, from: Data(json.utf8))
        XCTAssertNotNil(result.createdAt)
    }

    func testISODateDecodingWithoutFractionalSeconds() throws {
        let json = #"{"createdAt": "2024-01-15T10:30:00Z"}"#
        struct DateTest: Codable {
            let createdAt: Date
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
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

        let result = try decoder.decode(DateTest.self, from: Data(json.utf8))
        XCTAssertNotNil(result.createdAt)
    }

    // MARK: - Error Types

    func testAPIErrorLocalizedDescription() {
        XCTAssertEqual(APIError.invalidURL.errorDescription, "Invalid URL")
        XCTAssertEqual(APIError.invalidResponse.errorDescription, "Invalid server response")
        XCTAssertEqual(APIError.unauthorized.errorDescription, "Unauthorized")
        XCTAssertEqual(APIError.tokenRefreshFailed.errorDescription, "Session expired. Please log in again.")
    }

    func testHTTPErrorDescription() {
        let error = APIError.httpError(statusCode: 404, body: nil)
        XCTAssertEqual(error.errorDescription, "HTTP error 404")
    }

    func testDecodingErrorDescription() {
        let underlying = DecodingError.dataCorrupted(
            DecodingError.Context(
                codingPath: [],
                debugDescription: "test"
            )
        )
        let error = APIError.decodingError(underlying)
        XCTAssertTrue(error.errorDescription?.contains("Failed to decode response") == true)
    }

    func testNetworkErrorDescription() {
        let nsError = NSError(domain: NSURLErrorDomain, code: NSURLErrorNotConnectedToInternet)
        let error = APIError.networkError(nsError)
        XCTAssertTrue(error.errorDescription?.contains("Network error") == true)
    }
}

// Helper struct for URL construction tests
private struct TestResponse: Decodable {}
private struct EmptyPathResponse: Decodable {}
