package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/services"
)

// MockOAuthCacheForHandler implements CacheService for handler tests
type MockOAuthCacheForHandler struct {
	data map[string][]byte
}

func NewMockOAuthCacheForHandler() *MockOAuthCacheForHandler {
	return &MockOAuthCacheForHandler{data: make(map[string][]byte)}
}

func (m *MockOAuthCacheForHandler) Get(ctx interface{}, key string) ([]byte, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return nil, services.ErrOAuthStateExpired
}

func (m *MockOAuthCacheForHandler) Set(ctx interface{}, key string, value []byte, ttl interface{}) error {
	m.data[key] = value
	return nil
}

func (m *MockOAuthCacheForHandler) Delete(ctx interface{}, key string) error {
	delete(m.data, key)
	return nil
}

func TestOAuthRedirect_NoOAuthService(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	app.Get("/oauth/:provider", handler.OAuthRedirect)

	req := httptest.NewRequest("GET", "/oauth/github", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusNotImplemented, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "not_configured", result["error"])
}

func TestOAuthRedirect_InvalidProvider(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured
	app.Get("/oauth/:provider", handler.OAuthRedirect)

	req := httptest.NewRequest("GET", "/oauth/invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "invalid_provider", result["error"])
}

func TestOAuthCallback_NoOAuthService(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	app.Get("/oauth/:provider/callback", handler.OAuthCallback)

	req := httptest.NewRequest("GET", "/oauth/github/callback?code=testcode&state=teststate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusNotImplemented, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "not_configured", result["error"])
}

func TestOAuthCallback_MissingCode(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured
	app.Get("/oauth/:provider/callback", handler.OAuthCallback)

	req := httptest.NewRequest("GET", "/oauth/github/callback?state=teststate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "missing_code", result["error"])
}

func TestOAuthCallback_MissingState(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured
	app.Get("/oauth/:provider/callback", handler.OAuthCallback)

	req := httptest.NewRequest("GET", "/oauth/github/callback?code=testcode", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "missing_state", result["error"])
}

func TestOAuthCallback_OAuthError(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured
	app.Get("/oauth/:provider/callback", handler.OAuthCallback)

	req := httptest.NewRequest("GET", "/oauth/github/callback?error=access_denied&error_description=User+denied+access", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "oauth_access_denied", result["error"])
	assert.Equal(t, "User denied access", result["message"])
}

func TestOAuthCallback_InvalidProvider(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured
	app.Get("/oauth/:provider/callback", handler.OAuthCallback)

	req := httptest.NewRequest("GET", "/oauth/invalid/callback?code=testcode&state=teststate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "invalid_provider", result["error"])
}

func TestOAuthProviders_AllSupported(t *testing.T) {
	providers := []string{"github", "google", "discord"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			app := fiber.New()
			t.Cleanup(func() { _ = app.Shutdown() })
			handler := NewAuthHandler(nil)
			app.Get("/oauth/:provider", handler.OAuthRedirect)

			req := httptest.NewRequest("GET", "/oauth/"+provider, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Should fail with "not_configured" not "invalid_provider"
			assert.Equal(t, fiber.StatusNotImplemented, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			assert.Equal(t, "not_configured", result["error"])
		})
	}
}

func TestGetEnabledProviders_NoOAuthService(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	app.Get("/oauth/providers", handler.GetEnabledProviders)

	req := httptest.NewRequest("GET", "/oauth/providers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	providers := result["providers"].([]interface{})
	assert.Len(t, providers, 0)
}

func TestOAuthUnlink_InvalidProvider(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.New())
		return c.Next()
	})
	app.Delete("/oauth/:provider", handler.OAuthUnlink)

	req := httptest.NewRequest("DELETE", "/oauth/invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "invalid_provider", result["error"])
}

func TestOAuthLinkRedirect_InvalidProvider(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)
	handler.oauthService = &services.OAuthService{} // Set but not configured

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.New())
		return c.Next()
	})
	app.Post("/oauth/:provider/link", handler.OAuthLinkRedirect)

	req := httptest.NewRequest("POST", "/oauth/invalid/link", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "invalid_provider", result["error"])
}

func TestGetLinkedProviders_NoOAuthService(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })
	handler := NewAuthHandler(nil)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.New())
		return c.Next()
	})
	app.Get("/oauth/linked", handler.GetLinkedProviders)

	req := httptest.NewRequest("GET", "/oauth/linked", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusNotImplemented, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "not_configured", result["error"])
}
