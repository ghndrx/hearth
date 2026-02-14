package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret"

// generateTestToken creates a valid JWT for testing
func generateTestToken(userID uuid.UUID, tokenType string, expired bool) string {
	expiresAt := time.Now().Add(time.Hour)
	if expired {
		expiresAt = time.Now().Add(-time.Hour)
	}

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   userID.String(),
		Username: "testuser",
		Type:     tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	return tokenString
}

func TestNewMiddleware(t *testing.T) {
	secret := "test-secret-key"
	m := NewMiddleware(secret)

	if m == nil {
		t.Fatal("NewMiddleware returned nil")
	}
	if string(m.jwtSecret) != secret {
		t.Errorf("expected jwtSecret %q, got %q", secret, string(m.jwtSecret))
	}
}

func TestRequireAuth(t *testing.T) {
	m := NewMiddleware(testSecret)
	validUserID := uuid.New()
	validToken := generateTestToken(validUserID, "access", false)
	expiredToken := generateTestToken(validUserID, "access", true)
	refreshToken := generateTestToken(validUserID, "refresh", false)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
		shouldSetUser  bool
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "missing authorization header",
			shouldSetUser:  false,
		},
		{
			name:           "invalid format - no bearer prefix",
			authHeader:     "invalid-token",
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "invalid authorization format",
			shouldSetUser:  false,
		},
		{
			name:           "invalid format - wrong prefix",
			authHeader:     "Basic sometoken",
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "invalid authorization format",
			shouldSetUser:  false,
		},
		{
			name:           "invalid format - too many parts",
			authHeader:     "Bearer token extra",
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "invalid authorization format",
			shouldSetUser:  false,
		},
		{
			name:           "invalid token - malformed JWT",
			authHeader:     "Bearer invalid-not-jwt",
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "invalid token",
			shouldSetUser:  false,
		},
		{
			name:           "invalid token - expired",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "token expired",
			shouldSetUser:  false,
		},
		{
			name:           "invalid token - wrong type (refresh instead of access)",
			authHeader:     "Bearer " + refreshToken,
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "invalid token type",
			shouldSetUser:  false,
		},
		{
			name:           "valid JWT access token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: fiber.StatusOK,
			expectedError:  "",
			shouldSetUser:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var capturedUserID uuid.UUID

			app.Use(m.RequireAuth)
			app.Get("/test", func(c *fiber.Ctx) error {
				if userID, ok := c.Locals("userID").(uuid.UUID); ok {
					capturedUserID = userID
				}
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.shouldSetUser && capturedUserID == uuid.Nil {
				t.Error("expected userID to be set in context, but it wasn't")
			}
		})
	}
}

func TestCORS(t *testing.T) {
	m := NewMiddleware("test-secret")

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "GET request passes through with CORS headers",
			method:         "GET",
			expectedStatus: fiber.StatusOK,
			checkHeaders:   true,
		},
		{
			name:           "POST request passes through with CORS headers",
			method:         "POST",
			expectedStatus: fiber.StatusOK,
			checkHeaders:   true,
		},
		{
			name:           "OPTIONS request returns no content (preflight)",
			method:         "OPTIONS",
			expectedStatus: fiber.StatusNoContent,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(m.CORS())
			app.All("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkHeaders {
				if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
					t.Error("expected Access-Control-Allow-Origin header to be '*'")
				}
				if resp.Header.Get("Access-Control-Allow-Methods") == "" {
					t.Error("expected Access-Control-Allow-Methods header to be set")
				}
				if resp.Header.Get("Access-Control-Allow-Headers") == "" {
					t.Error("expected Access-Control-Allow-Headers header to be set")
				}
			}
		})
	}
}

func TestRequestID(t *testing.T) {
	m := NewMiddleware("test-secret")

	tests := []struct {
		name             string
		incomingID       string
		expectSameID     bool
		expectGenerated  bool
	}{
		{
			name:            "generates new ID when none provided",
			incomingID:      "",
			expectSameID:    false,
			expectGenerated: true,
		},
		{
			name:            "preserves incoming X-Request-ID",
			incomingID:      "custom-request-id-123",
			expectSameID:    true,
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var capturedID string

			app.Use(m.RequestID())
			app.Get("/test", func(c *fiber.Ctx) error {
				if id, ok := c.Locals("requestID").(string); ok {
					capturedID = id
				}
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.incomingID != "" {
				req.Header.Set("X-Request-ID", tt.incomingID)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			responseID := resp.Header.Get("X-Request-ID")

			if responseID == "" {
				t.Error("expected X-Request-ID header in response")
			}

			if tt.expectSameID && responseID != tt.incomingID {
				t.Errorf("expected response ID %q, got %q", tt.incomingID, responseID)
			}

			if capturedID == "" {
				t.Error("expected requestID to be set in context")
			}

			if tt.expectSameID && capturedID != tt.incomingID {
				t.Errorf("expected context ID %q, got %q", tt.incomingID, capturedID)
			}

			if tt.expectGenerated {
				// Verify it's a valid UUID
				_, err := uuid.Parse(responseID)
				if err != nil {
					t.Errorf("expected generated ID to be a valid UUID, got %q", responseID)
				}
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	m := NewMiddleware("test-secret")

	// RateLimit currently just passes through (TODO implementation)
	// Test that it doesn't break requests
	app := fiber.New()
	app.Use(m.RateLimit(100, 60))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestLogger(t *testing.T) {
	m := NewMiddleware("test-secret")

	// Logger currently just passes through (TODO implementation)
	// Test that it doesn't break requests
	app := fiber.New()
	app.Use(m.Logger())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestRecover(t *testing.T) {
	m := NewMiddleware("test-secret")

	tests := []struct {
		name           string
		shouldPanic    bool
		expectedStatus int
	}{
		{
			name:           "normal request passes through",
			shouldPanic:    false,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "panic is recovered and returns 500",
			shouldPanic:    true,
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(m.Recover())
			app.Get("/test", func(c *fiber.Ctx) error {
				if tt.shouldPanic {
					panic("test panic")
				}
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.shouldPanic {
				body, _ := io.ReadAll(resp.Body)
				if len(body) == 0 {
					t.Error("expected error response body for panic recovery")
				}
			}
		})
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	m := NewMiddleware("test-secret")

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "non-websocket request rejected",
			headers:        map[string]string{},
			expectedStatus: fiber.StatusUpgradeRequired,
		},
		{
			name: "websocket upgrade request passes",
			headers: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
			},
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(m.WebSocketUpgrade)
			app.Get("/ws", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/ws", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// Test middleware chaining
func TestMiddlewareChaining(t *testing.T) {
	m := NewMiddleware(testSecret)
	validUserID := uuid.New()
	validToken := generateTestToken(validUserID, "access", false)

	app := fiber.New()
	app.Use(m.Recover())
	app.Use(m.RequestID())
	app.Use(m.CORS())
	app.Use(m.Logger())

	// Protected route
	app.Get("/protected", m.RequireAuth, func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		return c.JSON(fiber.Map{"userID": userID.String()})
	})

	// Public route
	app.Get("/public", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("public route with all middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/public", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
		}

		// Check middleware effects
		if resp.Header.Get("X-Request-ID") == "" {
			t.Error("expected X-Request-ID header")
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected CORS header")
		}
	})

	t.Run("protected route without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", fiber.StatusUnauthorized, resp.StatusCode)
		}
	})

	t.Run("protected route with valid auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
		}
	})
}
