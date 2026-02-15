package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/api/middleware"
	"hearth/internal/auth"
	"hearth/internal/models"
	"hearth/internal/services"
)

// TestTokenIntegration tests the full flow: auth service generates token, middleware validates it
func TestTokenIntegration(t *testing.T) {
	const secretKey = "test-integration-secret-key"

	// Create real JWT service (same as used in auth service)
	jwtService := auth.NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	// Create middleware with the same secret
	mw := middleware.NewMiddleware(secretKey)

	// Create auth service mock that uses real JWT service
	authService := &mockAuthService{
		registerFunc: func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
			userID := uuid.New()
			user := &models.User{
				ID:            userID,
				Username:      username,
				Discriminator: "0001",
				Email:         email,
				CreatedAt:     time.Now(),
			}

			// Generate real tokens using the JWT service
			accessToken, refreshToken, err := jwtService.GenerateTokenPair(userID, username)
			if err != nil {
				return nil, nil, err
			}

			tokens := &services.AuthTokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    jwtService.GetExpirySeconds(),
			}
			return user, tokens, nil
		},
	}

	// Create auth handler
	authHandler := NewAuthHandler(authService)

	// Setup Fiber app
	app := fiber.New()

	// Public route - register
	app.Post("/auth/register", authHandler.Register)

	// Protected route - requires auth
	app.Get("/protected", mw.RequireAuth, func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		username := c.Locals("username").(string)
		return c.JSON(fiber.Map{
			"user_id":  userID.String(),
			"username": username,
		})
	})

	// Step 1: Register a user
	registerBody := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}
	bodyBytes, _ := json.Marshal(registerBody)

	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(bodyBytes))
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := app.Test(registerReq, -1)
	if err != nil {
		t.Fatalf("Failed to make register request: %v", err)
	}

	if registerResp.StatusCode != 201 {
		body, _ := io.ReadAll(registerResp.Body)
		t.Fatalf("Expected status 201, got %d: %s", registerResp.StatusCode, body)
	}

	// Parse register response to get tokens
	var registerResult map[string]interface{}
	respBody, _ := io.ReadAll(registerResp.Body)
	if err := json.Unmarshal(respBody, &registerResult); err != nil {
		t.Fatalf("Failed to parse register response: %v", err)
	}

	accessToken, ok := registerResult["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatal("No access_token in register response")
	}

	t.Logf("Got access token: %s", accessToken[:50]+"...")

	// Step 2: Use the token to access protected route
	protectedReq := httptest.NewRequest("GET", "/protected", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+accessToken)

	protectedResp, err := app.Test(protectedReq, -1)
	if err != nil {
		t.Fatalf("Failed to make protected request: %v", err)
	}

	protectedBody, _ := io.ReadAll(protectedResp.Body)

	if protectedResp.StatusCode != 200 {
		t.Fatalf("Protected route failed with status %d: %s", protectedResp.StatusCode, protectedBody)
	}

	var protectedResult map[string]interface{}
	if err := json.Unmarshal(protectedBody, &protectedResult); err != nil {
		t.Fatalf("Failed to parse protected response: %v", err)
	}

	if protectedResult["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", protectedResult["username"])
	}

	t.Logf("Protected route response: %v", protectedResult)
}

// TestTokenValidationFailureCases tests various failure scenarios
func TestTokenValidationFailureCases(t *testing.T) {
	const secretKey = "test-integration-secret-key"
	const differentSecret = "different-secret-key"

	// Create JWT service with original secret
	jwtService := auth.NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	// Create middleware with DIFFERENT secret (simulating misconfiguration)
	mwDifferentSecret := middleware.NewMiddleware(differentSecret)

	// Create middleware with correct secret
	mwCorrectSecret := middleware.NewMiddleware(secretKey)

	userID := uuid.New()
	username := "testuser"

	// Generate a valid token
	accessToken, _, err := jwtService.GenerateTokenPair(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("different_secret_fails", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", mwDifferentSecret.RequireAuth, func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, _ := app.Test(req, -1)

		if resp.StatusCode != 401 {
			t.Errorf("Expected 401 with different secret, got %d", resp.StatusCode)
		}
	})

	t.Run("correct_secret_succeeds", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", mwCorrectSecret.RequireAuth, func(c *fiber.Ctx) error {
			capturedUserID := c.Locals("userID").(uuid.UUID)
			if capturedUserID != userID {
				t.Errorf("UserID mismatch: expected %s, got %s", userID, capturedUserID)
			}
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, _ := app.Test(req, -1)

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 200 with correct secret, got %d: %s", resp.StatusCode, body)
		}
	})

	t.Run("missing_auth_header_fails", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", mwCorrectSecret.RequireAuth, func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		// No Authorization header

		resp, _ := app.Test(req, -1)

		if resp.StatusCode != 401 {
			t.Errorf("Expected 401 without auth header, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed_token_fails", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", mwCorrectSecret.RequireAuth, func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer not-a-valid-jwt-token")

		resp, _ := app.Test(req, -1)

		if resp.StatusCode != 401 {
			t.Errorf("Expected 401 with malformed token, got %d", resp.StatusCode)
		}
	})
}
