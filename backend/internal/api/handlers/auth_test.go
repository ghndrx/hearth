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

// mockAuthService implements the auth operations needed by AuthHandler
type mockAuthService struct {
	registerFunc      func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error)
	loginFunc         func(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error)
	refreshTokenFunc  func(ctx context.Context, refreshToken string) (*services.TokenResponse, error)
	logoutFunc        func(ctx context.Context, accessToken, refreshToken string) error
}

// AuthServiceInterface defines what AuthHandler needs from the auth service
type AuthServiceInterface interface {
	Register(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error)
	Login(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*services.TokenResponse, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

// mockAuthHandler wraps a mock service for testing
type mockAuthHandler struct {
	service *mockAuthService
}

func newMockAuthHandler(service *mockAuthService) *mockAuthHandler {
	return &mockAuthHandler{service: service}
}

// Register handles user registration
func (h *mockAuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "email, username, and password are required",
		})
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "invalid email format",
		})
	}

	// Validate username length
	if len(req.Username) < 2 || len(req.Username) > 32 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "username must be between 2 and 32 characters",
		})
	}

	// Validate password length
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "password must be at least 8 characters",
		})
	}

	user, tokens, err := h.service.registerFunc(c.Context(), &services.RegisterRequest{
		Email:      req.Email,
		Username:   req.Username,
		Password:   req.Password,
		InviteCode: req.InviteCode,
	})

	if err != nil {
		return handleAuthError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		User:   toUserResponse(user),
		Tokens: toTokenResponse(tokens),
	})
}

// Login handles user login
func (h *mockAuthHandler) Login(c *fiber.Ctx) error {
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

	user, tokens, err := h.service.loginFunc(c.Context(), req.Email, req.Password)
	if err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(AuthResponse{
		User:   toUserResponse(user),
		Tokens: toTokenResponse(tokens),
	})
}

// Refresh handles token refresh
func (h *mockAuthHandler) Refresh(c *fiber.Ctx) error {
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

	tokens, err := h.service.refreshTokenFunc(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "invalid_token",
			"message": "invalid or expired refresh token",
		})
	}

	return c.JSON(toTokenResponse(tokens))
}

// Logout handles logout
func (h *mockAuthHandler) Logout(c *fiber.Ctx) error {
	accessToken := extractBearerToken(c)
	var req RefreshRequest
	_ = c.BodyParser(&req)

	if accessToken != "" || req.RefreshToken != "" {
		_ = h.service.logoutFunc(c.Context(), accessToken, req.RefreshToken)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func isValidEmail(email string) bool {
	// Simple email validation
	atIdx := -1
	dotIdx := -1
	for i, c := range email {
		if c == '@' {
			atIdx = i
		}
		if c == '.' && atIdx > 0 {
			dotIdx = i
		}
	}
	return atIdx > 0 && dotIdx > atIdx
}

func setupTestApp() (*fiber.App, *mockAuthService) {
	service := &mockAuthService{}
	handler := newMockAuthHandler(service)

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

	service.registerFunc = func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      req.Username,
			Discriminator: "0001",
			Email:         req.Email,
			CreatedAt:     time.Now(),
		}
		tokens := &services.TokenResponse{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    900,
			TokenType:    "Bearer",
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

	if result["tokens"] == nil {
		t.Error("Expected tokens in response")
	}
	if result["user"] == nil {
		t.Error("Expected user in response")
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
		t.Errorf("Expected error 'validation_error', got %v", result["error"])
	}
}

func TestRegister_MissingFields(t *testing.T) {
	app, _ := setupTestApp()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"username": "test", "password": "password123"}},
		{"missing username", map[string]string{"email": "test@example.com", "password": "password123"}},
		{"missing password", map[string]string{"email": "test@example.com", "username": "test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := makeRequest(app, "POST", "/auth/register", tc.body)
			if resp.Code != 400 {
				t.Errorf("Expected status 400, got %d", resp.Code)
			}
		})
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
		t.Errorf("Expected error 'validation_error', got %v", result["error"])
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
		return nil, nil, services.ErrEmailTaken
	}

	body := map[string]string{
		"email":    "taken@example.com",
		"username": "newuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 409 {
		t.Errorf("Expected status 409, got %d", resp.Code)
	}

	if result["error"] != "email_taken" {
		t.Errorf("Expected error 'email_taken', got %v", result["error"])
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
		return nil, nil, services.ErrUsernameTaken
	}

	body := map[string]string{
		"email":    "new@example.com",
		"username": "takenuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 409 {
		t.Errorf("Expected status 409, got %d", resp.Code)
	}

	if result["error"] != "username_taken" {
		t.Errorf("Expected error 'username_taken', got %v", result["error"])
	}
}

func TestRegister_RegistrationClosed(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
		return nil, nil, services.ErrRegistrationClosed
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 403 {
		t.Errorf("Expected status 403, got %d", resp.Code)
	}

	if result["error"] != "registration_closed" {
		t.Errorf("Expected error 'registration_closed', got %v", result["error"])
	}
}

func TestRegister_InviteRequired(t *testing.T) {
	app, service := setupTestApp()

	service.registerFunc = func(ctx context.Context, req *services.RegisterRequest) (*models.User, *services.TokenResponse, error) {
		return nil, nil, services.ErrInviteRequired
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 403 {
		t.Errorf("Expected status 403, got %d", resp.Code)
	}

	if result["error"] != "invite_required" {
		t.Errorf("Expected error 'invite_required', got %v", result["error"])
	}
}

func TestLogin_Success(t *testing.T) {
	app, service := setupTestApp()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
			Email:         email,
			CreatedAt:     time.Now(),
		}
		tokens := &services.TokenResponse{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    900,
			TokenType:    "Bearer",
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

	if result["tokens"] == nil {
		t.Error("Expected tokens in response")
	}
	if result["user"] == nil {
		t.Error("Expected user in response")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	app, service := setupTestApp()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.TokenResponse, error) {
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
		t.Errorf("Expected error 'invalid_credentials', got %v", result["error"])
	}
}

func TestLogin_MissingFields(t *testing.T) {
	app, _ := setupTestApp()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"password": "password123"}},
		{"missing password", map[string]string{"email": "test@example.com"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := makeRequest(app, "POST", "/auth/login", tc.body)
			if resp.Code != 400 {
				t.Errorf("Expected status 400, got %d", resp.Code)
			}
		})
	}
}

func TestRefresh_Success(t *testing.T) {
	app, service := setupTestApp()

	service.refreshTokenFunc = func(ctx context.Context, refreshToken string) (*services.TokenResponse, error) {
		return &services.TokenResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    900,
			TokenType:    "Bearer",
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

func TestRefresh_InvalidToken(t *testing.T) {
	app, service := setupTestApp()

	service.refreshTokenFunc = func(ctx context.Context, refreshToken string) (*services.TokenResponse, error) {
		return nil, services.ErrInvalidCredentials
	}

	body := map[string]string{
		"refresh_token": "invalid-token",
	}

	resp, result := makeRequest(app, "POST", "/auth/refresh", body)

	if resp.Code != 401 {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if result["error"] != "invalid_token" {
		t.Errorf("Expected error 'invalid_token', got %v", result["error"])
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
		t.Errorf("Expected error 'validation_error', got %v", result["error"])
	}
}

func TestLogout_Success(t *testing.T) {
	app, service := setupTestApp()

	logoutCalled := false
	service.logoutFunc = func(ctx context.Context, accessToken, refreshToken string) error {
		logoutCalled = true
		return nil
	}

	body := map[string]string{
		"refresh_token": "token-to-revoke",
	}

	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewReader(mustJSON(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer access-token")

	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 204 {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	if !logoutCalled {
		t.Error("Expected logout to be called")
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestRefresh_InvalidBody(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestRegister_UsernameTooShort(t *testing.T) {
	app, _ := setupTestApp()

	body := map[string]string{
		"email":    "test@example.com",
		"username": "a",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 400 {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	if result["error"] != "validation_error" {
		t.Errorf("Expected error 'validation_error', got %v", result["error"])
	}
}

func TestRegister_UsernameTooLong(t *testing.T) {
	app, _ := setupTestApp()

	body := map[string]string{
		"email":    "test@example.com",
		"username": "thisusernameiswaytoolongandexceedsthemaximumlengthallowed",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	if resp.Code != 400 {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	if result["error"] != "validation_error" {
		t.Errorf("Expected error 'validation_error', got %v", result["error"])
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
