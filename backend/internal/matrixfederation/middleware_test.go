package matrixfederation

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantOrigin string
		wantKeyID  string
		wantSig    string
		wantErr    bool
	}{
		{
			name:       "valid header",
			header:     `X-Matrix origin="example.com",key="ed25519:a_Obwu",sig="base64signaturehere"`,
			wantOrigin: "example.com",
			wantKeyID:  "ed25519:a_Obwu",
			wantSig:    "base64signaturehere",
			wantErr:    false,
		},
		{
			name:    "missing X-Matrix prefix",
			header:  `Bearer token`,
			wantErr: true,
		},
		{
			name:    "missing origin",
			header:  `X-Matrix key="ed25519:a",sig="sig"`,
			wantErr: true,
		},
		{
			name:    "missing key",
			header:  `X-Matrix origin="example.com",sig="sig"`,
			wantErr: true,
		},
		{
			name:    "missing sig",
			header:  `X-Matrix origin="example.com",key="ed25519:a"`,
			wantErr: true,
		},
		{
			name:    "empty header",
			header:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := ParseAuthorizationHeader(tt.header)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOrigin, auth.Origin)
			assert.Equal(t, tt.wantKeyID, auth.KeyID)
			assert.Equal(t, tt.wantSig, auth.Sig)
		})
	}
}

func TestParseAuthorizationHeader_RealWorldFormat(t *testing.T) {
	// Test with a real-world style header
	header := `X-Matrix origin="matrix.org",key="ed25519:a_Obwu",sig="cV6d5+g8Y7Z3VjGoNlC3lVhJxQ7w9E5R8xK2hJpLMxQjY6dH5xK2hJpLMxQjY6dH5xK2hJpLMxQjY6dH5x"`
	auth, err := ParseAuthorizationHeader(header)
	require.NoError(t, err)
	assert.Equal(t, "matrix.org", auth.Origin)
	assert.Equal(t, "ed25519:a_Obwu", auth.KeyID)
	assert.NotEmpty(t, auth.Sig)
}

func TestFederationMiddleware_VerifyXMatrix_SkipsNonFederationPaths(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")
	middleware := NewFederationMiddleware(keyStore)

	app.Use(middleware.VerifyXMatrix())
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFederationMiddleware_VerifyXMatrix_SkipsUnauthenticatedEndpoints(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")
	middleware := NewFederationMiddleware(keyStore)

	app.Use(middleware.VerifyXMatrix())
	app.Get("/_matrix/federation/v1/version", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{"server": map[string]interface{}{"version": "1.0"}})
	})

	req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFederationMiddleware_VerifyXMatrix_MissingAuthHeader(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")
	middleware := NewFederationMiddleware(keyStore)

	app.Use(middleware.VerifyXMatrix())
	app.Put("/_matrix/federation/v1/send/:txnId", func(c *fiber.Ctx) error {
		return c.SendString("should not reach here")
	})

	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader([]byte(`{}`)))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "M_UNAUTHORIZED")
}

func TestFederationMiddleware_VerifyXMatrix_InvalidAuthHeader(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")
	middleware := NewFederationMiddleware(keyStore)

	app.Use(middleware.VerifyXMatrix())
	app.Put("/_matrix/federation/v1/send/:txnId", func(c *fiber.Ctx) error {
		return c.SendString("should not reach here")
	})

	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "M_UNAUTHORIZED")
}

func TestFederationMiddleware_VerifyXMatrix_ValidRequestSetsLocals(t *testing.T) {
	// Create a key pair for testing
	keyStore := NewKeyStore("example.com")
	key, err := GenerateSigningKey()
	require.NoError(t, err)
	keyStore.AddKey(key, true)

	app := fiber.New()
	middleware := NewFederationMiddleware(keyStore)

	var capturedOrigin, capturedKeyID string
	app.Use(middleware.VerifyXMatrix())
	app.Put("/_matrix/federation/v1/send/:txnId", func(c *fiber.Ctx) error {
		capturedOrigin = c.Locals("matrix_origin").(string)
		capturedKeyID = c.Locals("matrix_key_id").(string)
		return c.SendString("OK")
	})

	// Create a signed request
	body := []byte(`{"origin":"example.com","origin_server_ts":1234567890000,"pdus":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader(body))

	// Sign the request
	clientMiddleware := NewFederationClientMiddleware(keyStore, "example.com")
	stdReq := &http.Request{
		Method: "PUT",
		URL:    req.URL,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: make(http.Header),
	}
	stdReq.URL.Path = "/_matrix/federation/v1/send/txn123"
	require.NoError(t, clientMiddleware.SignRequest(stdReq))

	req.Header.Set("Authorization", stdReq.Header.Get("Authorization"))

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "example.com", capturedOrigin)
	assert.Equal(t, key.KeyID, capturedKeyID)
}

func TestFederationMiddleware_VerifyXMatrix_OriginNotAllowed(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")

	// Only allow specific origins
	middleware := NewFederationMiddleware(keyStore,
		WithOriginValidator(func(origin string) bool {
			return origin == "trusted.com"
		}),
	)

	app.Use(middleware.VerifyXMatrix())
	app.Put("/_matrix/federation/v1/send/:txnId", func(c *fiber.Ctx) error {
		return c.SendString("should not reach here")
	})

	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", `X-Matrix origin="untrusted.com",key="ed25519:test",sig="testsig"`)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "M_FORBIDDEN")
	assert.Contains(t, string(body), "not allowed")
}

func TestFederationMiddleware_VerifyXMatrix_KeyNotFound(t *testing.T) {
	app := fiber.New()
	keyStore := NewKeyStore("example.com")
	// Don't add any keys
	middleware := NewFederationMiddleware(keyStore)

	app.Use(middleware.VerifyXMatrix())
	app.Put("/_matrix/federation/v1/send/:txnId", func(c *fiber.Ctx) error {
		return c.SendString("should not reach here")
	})

	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", `X-Matrix origin="example.com",key="ed25519:missing",sig="testsig"`)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "M_FORBIDDEN")
	assert.Contains(t, string(body), "key")
}

func TestFederationClientMiddleware_SignRequest(t *testing.T) {
	keyStore := NewKeyStore("example.com")
	key, err := GenerateSigningKey()
	require.NoError(t, err)
	keyStore.AddKey(key, true)

	clientMiddleware := NewFederationClientMiddleware(keyStore, "example.com")

	body := []byte(`{"test":"data"}`)
	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", bytes.NewReader(body))

	// Convert to standard http.Request for signing
	stdReq := &http.Request{
		Method: "PUT",
		URL:    req.URL,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: make(http.Header),
	}
	stdReq.URL.Path = "/_matrix/federation/v1/send/txn123"

	err = clientMiddleware.SignRequest(stdReq)
	require.NoError(t, err)

	// Check Authorization header was added
	authHeader := stdReq.Header.Get("Authorization")
	assert.NotEmpty(t, authHeader)
	assert.Contains(t, authHeader, "X-Matrix")
	assert.Contains(t, authHeader, `origin="example.com"`)
	assert.Contains(t, authHeader, `key="`)
	assert.Contains(t, authHeader, `sig="`)
}

func TestFederationClientMiddleware_SignRequest_NoKey(t *testing.T) {
	keyStore := NewKeyStore("example.com")
	// Don't add any keys
	clientMiddleware := NewFederationClientMiddleware(keyStore, "example.com")

	req := httptest.NewRequest(http.MethodPut, "/_matrix/federation/v1/send/txn123", nil)
	stdReq := &http.Request{
		Method: "PUT",
		URL:    req.URL,
		Body:   nil,
		Header: make(http.Header),
	}
	stdReq.URL.Path = "/_matrix/federation/v1/send/txn123"

	err := clientMiddleware.SignRequest(stdReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no signing key")
}

func TestIsUnauthenticatedEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/_matrix/federation/v1/version", true},
		{"/_matrix/key/v2/server", true},
		{"/_matrix/key/v2/server/ed25519:a_Obwu", true},
		{"/_matrix/federation/v1/send/123", false},
		{"/_matrix/federation/v1/make_join/!room:server", false},
		{"/_matrix/federation/v1/send_join/!room:server", false},
		{"/_matrix/federation/v1/event/$abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isUnauthenticatedEndpoint(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFederationMiddlewareOptions(t *testing.T) {
	keyStore := NewKeyStore("example.com")

	// Test WithMaxRequestAge
	fm := NewFederationMiddleware(keyStore, WithMaxRequestAge(10*time.Minute))
	assert.Equal(t, int64(600000), fm.maxRequestAge)

	// Test WithOriginValidator
	called := false
	fm2 := NewFederationMiddleware(keyStore, WithOriginValidator(func(origin string) bool {
		called = true
		return true
	}))
	fm2.originValidator("test.com")
	assert.True(t, called)
}
