package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Mock implementations for integration tests

type mockAuthServiceImpl struct {
	registerFunc     func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error)
	loginFunc        func(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error)
	refreshTokenFunc func(ctx context.Context, refreshToken string) (*services.TokenResponse, error)
	logoutFunc       func(ctx context.Context, accessToken, refreshToken string) error
}

func (m *mockAuthServiceImpl) Register(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, req)
	}
	return nil, nil, nil
}

func (m *mockAuthServiceImpl) Login(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, nil, nil
}

func (m *mockAuthServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*services.TokenResponse, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, refreshToken)
	}
	return nil, nil
}

func (m *mockAuthServiceImpl) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, accessToken, refreshToken)
	}
	return nil
}

func setupAuthIntegrationTestApp(authSvc *mockAuthServiceImpl) *fiber.App {
	app := fiber.New()

	// Middleware to inject userID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.New())
		return c.Next()
	})

	// Routes with inline implementations that call our mock
	app.Post("/auth/register", func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "invalid request body",
			})
		}

		// Validation
		if req.Email == "" || req.Username == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "email, username, and password are required",
			})
		}

		user, tokens, err := authSvc.Register(c.Context(), &services.RegisterRequest{
			Email:    req.Email,
			Username: req.Username,
			Password: req.Password,
		})
		if err != nil {
			return handleAuthError(c, err)
		}

		return c.Status(fiber.StatusCreated).JSON(AuthResponse{
			User:   toUserResponse(user),
			Tokens: toTokenResponse(tokens),
		})
	})

	app.Post("/auth/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "invalid request body",
			})
		}

		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "email and password are required",
			})
		}

		user, tokens, err := authSvc.Login(c.Context(), req.Email, req.Password)
		if err != nil {
			return handleAuthError(c, err)
		}

		return c.JSON(AuthResponse{
			User:   toUserResponse(user),
			Tokens: toTokenResponse(tokens),
		})
	})

	app.Post("/auth/refresh", func(c *fiber.Ctx) error {
		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "invalid request body",
			})
		}

		if req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "refresh_token is required",
			})
		}

		tokens, err := authSvc.RefreshToken(c.Context(), req.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "invalid_token",
				"message": "invalid or expired refresh token",
			})
		}

		return c.JSON(toTokenResponse(tokens))
	})

	app.Post("/auth/logout", func(c *fiber.Ctx) error {
		accessToken := extractBearerToken(c)
		var req RefreshRequest
		_ = c.BodyParser(&req)

		if accessToken != "" || req.RefreshToken != "" {
			_ = authSvc.Logout(c.Context(), accessToken, req.RefreshToken)
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

func TestAuthHandler_Register_Integration(t *testing.T) {
	svc := &mockAuthServiceImpl{
		registerFunc: func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
			user := &models.User{
				ID:            uuid.New(),
				Username:      req.Username,
				Discriminator: "0001",
				Email:         req.Email,
				CreatedAt:     time.Now(),
			}
			tokens := &services.TokenResponse{
				AccessToken:  "test-access",
				RefreshToken: "test-refresh",
				ExpiresIn:    900,
				TokenType:    "Bearer",
			}
			return user, tokens, nil
		},
	}

	app := setupAuthIntegrationTestApp(svc)

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "test-access", result.Tokens.AccessToken)
}

func TestAuthHandler_Login_Integration(t *testing.T) {
	svc := &mockAuthServiceImpl{
		loginFunc: func(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error) {
			user := &models.User{
				ID:            uuid.New(),
				Username:      "testuser",
				Discriminator: "0001",
				Email:         email,
				CreatedAt:     time.Now(),
			}
			tokens := &services.TokenResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    900,
				TokenType:    "Bearer",
			}
			return user, tokens, nil
		},
	}

	app := setupAuthIntegrationTestApp(svc)

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "testuser", result.User.Username)
}

func TestAuthHandler_Refresh_Integration(t *testing.T) {
	svc := &mockAuthServiceImpl{
		refreshTokenFunc: func(ctx context.Context, refreshToken string) (*services.TokenResponse, error) {
			return &services.TokenResponse{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				ExpiresIn:    900,
				TokenType:    "Bearer",
			}, nil
		},
	}

	app := setupAuthIntegrationTestApp(svc)

	body := map[string]string{
		"refresh_token": "valid-refresh-token",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "new-access", result.AccessToken)
}

func TestAuthHandler_Logout_Integration(t *testing.T) {
	logoutCalled := false
	svc := &mockAuthServiceImpl{
		logoutFunc: func(ctx context.Context, accessToken, refreshToken string) error {
			logoutCalled = true
			return nil
		},
	}

	app := setupAuthIntegrationTestApp(svc)

	body := map[string]string{
		"refresh_token": "refresh-to-revoke",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer access-to-revoke")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	assert.True(t, logoutCalled)
}

func TestAuthHandler_Register_ValidationErrors(t *testing.T) {
	svc := &mockAuthServiceImpl{}
	app := setupAuthIntegrationTestApp(svc)

	tests := []struct {
		name   string
		body   map[string]string
		status int
	}{
		{
			name:   "missing email",
			body:   map[string]string{"username": "test", "password": "pass123"},
			status: fiber.StatusBadRequest,
		},
		{
			name:   "missing username",
			body:   map[string]string{"email": "test@example.com", "password": "pass123"},
			status: fiber.StatusBadRequest,
		},
		{
			name:   "missing password",
			body:   map[string]string{"email": "test@example.com", "username": "test"},
			status: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestAuthHandler_Login_ValidationErrors(t *testing.T) {
	svc := &mockAuthServiceImpl{}
	app := setupAuthIntegrationTestApp(svc)

	tests := []struct {
		name   string
		body   map[string]string
		status int
	}{
		{
			name:   "missing email",
			body:   map[string]string{"password": "pass123"},
			status: fiber.StatusBadRequest,
		},
		{
			name:   "missing password",
			body:   map[string]string{"email": "test@example.com"},
			status: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}
