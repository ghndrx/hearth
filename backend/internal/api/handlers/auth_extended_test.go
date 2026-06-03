package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// setupAuthTestAppWithUser creates an app with auth middleware that injects the given userID
func setupAuthTestAppWithUser(userID uuid.UUID) (*fiber.App, *mockAuthService) {
	service := &mockAuthService{}
	handler := NewAuthHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Post("/auth/register", handler.Register)
	app.Post("/auth/login", handler.Login)
	app.Post("/auth/login/mfa", handler.LoginWithMFA)
	app.Post("/auth/refresh", handler.Refresh)
	app.Post("/auth/logout", handler.Logout)
	app.Post("/auth/mfa/enable", handler.EnableMFA)
	app.Post("/auth/mfa/verify", handler.VerifyMFASetup)
	app.Post("/auth/mfa/disable", handler.DisableMFA)

	return app, service
}

// makeRawRequest sends a request with raw body bytes and returns the response
func makeRawRequest(app *fiber.App, method, path string, body []byte, contentType string) (*httptest.ResponseRecorder, map[string]interface{}) {
	reqBody := io.Reader(nil)
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, _ := app.Test(req, -1)

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	w := httptest.NewRecorder()
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	return w, result
}

// ========== Register Endpoint Tests ==========

func TestRegister_UsernameTaken(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrUsernameTaken
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "takenuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 409, resp.Code)
	assert.Equal(t, "username_taken", result["error"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrEmailTaken
	}

	body := map[string]string{
		"email":    "duplicate@example.com",
		"username": "newuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 409, resp.Code)
	assert.Equal(t, "email_taken", result["error"])
}

func TestRegister_EmptyFields(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "empty email",
			body: map[string]string{"email": "", "username": "test", "password": "password123"},
		},
		{
			name: "empty username",
			body: map[string]string{"email": "test@example.com", "username": "", "password": "password123"},
		},
		{
			name: "empty password",
			body: map[string]string{"email": "test@example.com", "username": "test", "password": ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, result := makeRequest(app, "POST", "/auth/register", tc.body)

			assert.Equal(t, 400, resp.Code)
			assert.Equal(t, "validation_error", result["error"])
		})
	}
}

func TestRegister_VeryLongUsername(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	body := map[string]string{
		"email":    "test@example.com",
		"username": strings.Repeat("a", 33),
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "validation_error", result["error"])
	assert.Contains(t, result["message"], "username must be between 2 and 32 characters")
}

func TestRegister_SQLInjectionUsername(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	callCount := 0
	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		callCount++
		user := &models.User{
			ID:            uuid.New(),
			Username:      username,
			Discriminator: "0001",
			Email:         email,
		}
		tokens := &services.AuthTokens{
			AccessToken:  "token",
			RefreshToken: "refresh",
			ExpiresIn:    900,
		}
		return user, tokens, nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "' OR '1'='1",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	// Handler does not do SQL sanitization; it passes through to service
	assert.Equal(t, 201, resp.Code)
	assert.NotNil(t, result["access_token"])
	assert.Equal(t, 1, callCount)
}

// ========== Login Endpoint Tests ==========

func TestLogin_WrongPassword(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrInvalidCredentials
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	assert.Equal(t, 401, resp.Code)
	assert.Equal(t, "invalid_credentials", result["error"])
}

func TestLogin_NonExistentUser(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrInvalidCredentials
	}

	body := map[string]string{
		"email":    "nobody@example.com",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	assert.Equal(t, 401, resp.Code)
	assert.Equal(t, "invalid_credentials", result["error"])
}

func TestLogin_AccountDisabled(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, errors.New("account disabled")
	}

	body := map[string]string{
		"email":    "banned@example.com",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	// Generic errors from service fall through to 500
	assert.Equal(t, 500, resp.Code)
	assert.Equal(t, "internal_error", result["error"])
}

func TestLogin_EmptyFields(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "empty email",
			body: map[string]string{"email": "", "password": "password123"},
		},
		{
			name: "empty password",
			body: map[string]string{"email": "test@example.com", "password": ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, result := makeRequest(app, "POST", "/auth/login", tc.body)

			assert.Equal(t, 400, resp.Code)
			assert.Equal(t, "validation_error", result["error"])
		})
	}
}

// ========== Refresh Token Tests ==========

func TestRefresh_MalformedToken(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.refreshTokensFunc = func(ctx context.Context, refreshToken string) (*services.AuthTokens, error) {
		return nil, errors.New("token malformed")
	}

	body := map[string]string{
		"refresh_token": "malformed.token.here",
	}

	resp, result := makeRequest(app, "POST", "/auth/refresh", body)

	assert.Equal(t, 401, resp.Code)
	assert.Equal(t, "invalid_refresh_token", result["error"])
}

// ========== Logout Tests ==========

func TestLogout_WithInvalidToken(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	// Logout is client-side for JWT; handler always returns 204
	resp, _ := makeRequest(app, "POST", "/auth/logout", map[string]string{"token": "invalid"})

	assert.Equal(t, 204, resp.Code)
}

// ========== MFA Endpoint Tests ==========

func TestEnableMFA_Success(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.enableMFAFunc = func(ctx context.Context, uid uuid.UUID) (*services.MFASetupResponse, error) {
		return &services.MFASetupResponse{
			Secret:      "SECRET123",
			QRCodeURL:   "otpauth://totp/test",
			BackupCodes: []string{"code1", "code2"},
		}, nil
	}

	resp, result := makeRequest(app, "POST", "/auth/mfa/enable", nil)

	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "SECRET123", result["secret"])
	assert.Equal(t, "otpauth://totp/test", result["qr_code_url"])
}

func TestEnableMFA_AlreadyEnabled(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.enableMFAFunc = func(ctx context.Context, uid uuid.UUID) (*services.MFASetupResponse, error) {
		return nil, services.ErrMFAAlreadyEnabled
	}

	resp, result := makeRequest(app, "POST", "/auth/mfa/enable", nil)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "mfa_already_enabled", result["error"])
}

func TestEnableMFA_Unauthorized(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	// Use regular app without auth middleware
	service.enableMFAFunc = func(ctx context.Context, uid uuid.UUID) (*services.MFASetupResponse, error) {
		return &services.MFASetupResponse{}, nil
	}

	req := httptest.NewRequest("POST", "/auth/mfa/enable", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	assert.Equal(t, 404, resp.StatusCode) // route not registered on basic app
}

func TestVerifyMFASetup_Success(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.verifyMFASetupFunc = func(ctx context.Context, uid uuid.UUID, code string) error {
		return nil
	}

	body := map[string]string{"code": "123456"}
	resp, result := makeRequest(app, "POST", "/auth/mfa/verify", body)

	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "MFA enabled successfully", result["message"])
}

func TestVerifyMFASetup_InvalidCode(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.verifyMFASetupFunc = func(ctx context.Context, uid uuid.UUID, code string) error {
		return services.ErrInvalidMFACode
	}

	body := map[string]string{"code": "000000"}
	resp, result := makeRequest(app, "POST", "/auth/mfa/verify", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_mfa_code", result["error"])
}

func TestVerifyMFASetup_MissingCode(t *testing.T) {
	userID := uuid.New()
	app, _ := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	body := map[string]string{}
	resp, result := makeRequest(app, "POST", "/auth/mfa/verify", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "validation_error", result["error"])
	assert.Contains(t, result["message"], "code is required")
}

func TestDisableMFA_Success(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.disableMFAFunc = func(ctx context.Context, uid uuid.UUID, password string) error {
		return nil
	}

	body := map[string]string{"password": "correctpassword"}
	resp, result := makeRequest(app, "POST", "/auth/mfa/disable", body)

	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "MFA disabled successfully", result["message"])
}

func TestDisableMFA_WrongPassword(t *testing.T) {
	userID := uuid.New()
	app, service := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	service.disableMFAFunc = func(ctx context.Context, uid uuid.UUID, password string) error {
		return services.ErrInvalidCredentials
	}

	body := map[string]string{"password": "wrongpassword"}
	resp, result := makeRequest(app, "POST", "/auth/mfa/disable", body)

	assert.Equal(t, 401, resp.Code)
	assert.Equal(t, "invalid_credentials", result["error"])
}

func TestDisableMFA_MissingPassword(t *testing.T) {
	userID := uuid.New()
	app, _ := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	body := map[string]string{}
	resp, result := makeRequest(app, "POST", "/auth/mfa/disable", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "validation_error", result["error"])
	assert.Contains(t, result["message"], "password is required")
}

// ========== Edge Case Tests ==========

func TestRegister_MalformedJSON(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/register", []byte(`{"email": "test@example.com", "username":`), "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestLogin_MalformedJSON(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/login", []byte(`{"email": broken`), "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestRefresh_MalformedJSON(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/refresh", []byte(`not json`), "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestRegister_EmptyBody(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/register", []byte{}, "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestLogin_EmptyBody(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/login", []byte{}, "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestRefresh_EmptyBody(t *testing.T) {
	app, _ := setupTestApp()
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/refresh", []byte{}, "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestLogin_VeryLongPassword(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.loginFunc = func(ctx context.Context, email, password string) (*models.User, *services.AuthTokens, error) {
		// Service would return ErrPasswordTooLong for very long passwords
		return nil, nil, services.ErrPasswordTooLong
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": strings.Repeat("a", 100),
	}

	resp, result := makeRequest(app, "POST", "/auth/login", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "password_too_long", result["error"])
}

func TestRegister_VeryLongPassword(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrPasswordTooLong
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": strings.Repeat("a", 100),
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	// Handler only checks min 8, so very long password reaches service
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "password_too_long", result["error"])
}

func TestVerifyMFASetup_MalformedJSON(t *testing.T) {
	userID := uuid.New()
	app, _ := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/mfa/verify", []byte(`{code:`), "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func TestDisableMFA_MalformedJSON(t *testing.T) {
	userID := uuid.New()
	app, _ := setupAuthTestAppWithUser(userID)
	defer app.Shutdown()

	resp, result := makeRawRequest(app, "POST", "/auth/mfa/disable", []byte(`bad json`), "application/json")

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_request", result["error"])
}

func setupTestAppWithMFA() (*fiber.App, *mockAuthService) {
	service := &mockAuthService{}
	handler := NewAuthHandler(service)

	app := fiber.New()
	app.Post("/auth/register", handler.Register)
	app.Post("/auth/login", handler.Login)
	app.Post("/auth/login/mfa", handler.LoginWithMFA)
	app.Post("/auth/refresh", handler.Refresh)
	app.Post("/auth/logout", handler.Logout)

	return app, service
}

func TestLoginWithMFA_Success(t *testing.T) {
	app, service := setupTestAppWithMFA()
	defer app.Shutdown()

	service.loginWithMFAFunc = func(ctx context.Context, email, password, mfaCode string) (*models.User, *services.AuthTokens, error) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
			Email:         email,
		}
		tokens := &services.AuthTokens{
			AccessToken:  "mfa-access-token",
			RefreshToken: "mfa-refresh-token",
			ExpiresIn:    900,
		}
		return user, tokens, nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"mfa_code": "123456",
	}

	resp, result := makeRequest(app, "POST", "/auth/login/mfa", body)

	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "mfa-access-token", result["access_token"])
	assert.Equal(t, "Bearer", result["token_type"])
}

func TestLoginWithMFA_InvalidCode(t *testing.T) {
	app, service := setupTestAppWithMFA()
	defer app.Shutdown()

	service.loginWithMFAFunc = func(ctx context.Context, email, password, mfaCode string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrInvalidMFACode
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"mfa_code": "000000",
	}

	resp, result := makeRequest(app, "POST", "/auth/login/mfa", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "invalid_mfa_code", result["error"])
}

func TestLoginWithMFA_MissingFields(t *testing.T) {
	app, _ := setupTestAppWithMFA()
	defer app.Shutdown()

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "missing email",
			body: map[string]string{"password": "password123", "mfa_code": "123456"},
		},
		{
			name: "missing password",
			body: map[string]string{"email": "test@example.com", "mfa_code": "123456"},
		},
		{
			name: "missing mfa_code",
			body: map[string]string{"email": "test@example.com", "password": "password123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, result := makeRequest(app, "POST", "/auth/login/mfa", tc.body)

			assert.Equal(t, 400, resp.Code)
			assert.Equal(t, "validation_error", result["error"])
		})
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrPasswordWeak
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "password_weak", result["error"])
}

func TestRegister_RegistrationClosed(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrRegistrationClosed
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 403, resp.Code)
	assert.Equal(t, "registration_closed", result["error"])
}

func TestRegister_InviteRequired(t *testing.T) {
	app, service := setupTestApp()
	defer app.Shutdown()

	service.registerFunc = func(ctx context.Context, email, username, password string) (*models.User, *services.AuthTokens, error) {
		return nil, nil, services.ErrInviteRequired
	}

	body := map[string]string{
		"email":    "test@example.com",
		"username": "testuser",
		"password": "password123",
	}

	resp, result := makeRequest(app, "POST", "/auth/register", body)

	assert.Equal(t, 403, resp.Code)
	assert.Equal(t, "invite_required", result["error"])
}
