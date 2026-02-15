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

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockAuthService implements services.AuthService for testing
type mockAuthService struct {
	registerFunc      func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error)
	loginFunc         func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error)
	refreshTokensFunc func(ctx context.Context, refreshToken string) (*services.AuthTokens, error)
	validateTokenFunc func(ctx context.Context, token string) (uuid.UUID, error)
}

func (m *mockAuthService) Register(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, email, username, password)
	}
	return nil, nil, nil
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, nil, nil
}

func (m *mockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*services.AuthTokens, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(ctx, refreshToken)
	}
	return nil, nil
}

func (m *mockAuthService) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, token)
	}
	return uuid.Nil, nil
}

func setupTestApp() (*fiber.App, *mockAuthService) {
	service := &mockAuthService{}
	handler := NewAuthHandler(service)

	app := fiber.New()
	app.Post("/auth/register", handler.Register)
	app.Post("/auth/login", handler.Login)
	app.Post("/auth/refresh", handler.Refresh)
	app.Post("/auth/logout", handler.Logout)

	return app, service
}

func makeRequest(app *fiber.App, method, path string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	w := httptest.NewRecorder()
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	return w, result
}

func TestRegister_Success(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      username,
			Discriminator: "0001",
			Email:         email,
			CreatedAt:     time.Now(),
		}
		tokens := &services.AuthTokens{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    900,
		}
		return user, tokens, nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 201 {
		t.Errorf("Expected status 201, got %d", resp.Code)
	}

	if result["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if result["refresh_token"] == nil {
		t.Error("Expected refresh_token in response")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	app, _ := setupTestApp()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "missing email",
			body: map[string]string{"username": "test", "password": "password123"},
		},
		{
			name: "missing username",
			body: map[string]string{"email": "test@example.com", "password": "password123"},
		},
		{
			name: "missing password",
			body: map[string]string{"email": "test@example.com", "username": "test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, result := makeRequest(app, "POST", "/auth/register", tc.body)

			if resp.Code != 400 {
				t.Errorf("Expected status 400, got %d", resp.Code)
			}

			if result["error"] != "validation_error" {
				t.Errorf("Expected error 'validation_error', got '%s'", result["error"])
			}
		})
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	app, _ := setupTestApp()

	body := map[string]string{
		"email":    "invalid-email",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 400 {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	if result["error"] != "validation_error" {
		t.Errorf("Expected error 'validation_error', got '%s'", result["error"])
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	app, _ := setupTestApp()

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "short",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 400 {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	if result["error"] != "validation_error" {
		t.Errorf("Expected error 'validation_error', got '%s'", result["error"])
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrEmailTaken
	}

	body := map[string]string{
		"email":    "taken@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 409 {
		t.Errorf("Expected status 409, got %d", resp.Code)
	}

	if result["error"] != "email_taken" {
		t.Errorf("Expected error 'email_taken', got '%s'", result["error"])
	}
}

func TestLogin_Success(t *testing.T) {
	app, service := setupTestApp()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
			Email:         email,
			CreatedAt:     time.Now(),
		}
		tokens := &services.AuthTokens{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    900,
		}
		return user, tokens, nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	if resp.Code != 200 {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	if result["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if result["refresh_token"] == nil {
		t.Error("Expected refresh_token in response")
	}
}

func TestLogin_MissingFields(t *testing.T) {
	app, _ := setupTestApp()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "missing email",
			body: map[string]string{"password": "password123"},
		},
		{
			name: "missing password",
			body: map[string]string{"email": "test@example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, result := makeRequest(app, "POST", "/auth/login", tc.body)

			if resp.Code != 400 {
				t.Errorf("Expected status 400, got %d", resp.Code)
			}

			if result["error"] != "validation_error" {
				t.Errorf("Expected error 'validation_error', got '%s'", result["error"])
			}
		})
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	app, service := setupTestApp()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrInvalidCredentials
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	if resp.Code != 401 {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if result["error"] != "invalid_credentials" {
		t.Errorf("Expected error 'invalid_credentials', got '%s'", result["error"])
	}
}

func TestRefresh_Success(t *testing.T) {
	app, service := setupTestApp()

	service.refreshTokensFunc = func(ctx context.Context, refreshToken string) (*services.AuthTokens, error) {
		return &services.AuthTokens{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    900,
		}, nil
	}

	body := map[string]string{
		"refresh_token": "valid-refresh-token",
	}

	resp, result := makeRequest(app, "POST", "/auth/refresh", body)

	if resp.Code != 200 {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	if result["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if result["refresh_token"] == nil {
		t.Error("Expected refresh_token in response")
	}
}

func TestRefresh_MissingToken(t *testing.T) {
	app, _ := setupTestApp()

	body := map[string]string{}

	resp, result := makeRequest(app, "POST", "/auth/refresh", body)

	if resp.Code != 400 {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	if result["error"] != "validation_error" {
		t.Errorf("Expected error 'validation_error', got '%s'", result["error"])
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	app, service := setupTestApp()

	service.refreshTokensFunc = func(ctx context.Context, refreshToken string) (*services.AuthTokens, error) {
		return nil, services.ErrInvalidCredentials
	}

	body := map[string]string{
		"refresh_token": "invalid-refresh-token",
	}

	resp, result := makeRequest(app, "POST", "/auth/refresh", body)

	if resp.Code != 401 {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if result["error"] != "invalid_refresh_token" {
		t.Errorf("Expected error 'invalid_refresh_token', got '%s'", result["error"])
	}
}

func TestLogout_Success(t *testing.T) {
	app, _ := setupTestApp()

	resp, _ := makeRequest(app, "POST", "/auth/logout", nil)

	if resp.Code != 204 {
		t.Errorf("Expected status 204, got %d", resp.Code)
	}
}
