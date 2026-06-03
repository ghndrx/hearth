// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements X-Matrix signature verification middleware for incoming
// federation requests.
//
// Matrix Spec References:
//   - Request Authentication: https://spec.matrix.org/v1.16/server-server-api/#request-authentication
//   - Signature Format: https://spec.matrix.org/v1.16/appendices/#signing-json
package matrixfederation

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AuthorizationHeader is the Matrix federation authorization header format.
// Format: X-Matrix origin="<origin_server_name>",key="<key_id>",sig="<base64_signature>"
type AuthorizationHeader struct {
	Origin string // The server name that sent the request
	KeyID  string // The signing key ID used
	Sig    string // Base64-encoded signature
}

// ParseAuthorizationHeader parses an X-Matrix authorization header.
func ParseAuthorizationHeader(header string) (*AuthorizationHeader, error) {
	if !strings.HasPrefix(header, "X-Matrix ") {
		return nil, fmt.Errorf("authorization header must start with 'X-Matrix '")
	}

	// Remove the "X-Matrix " prefix
	content := header[len("X-Matrix "):]

	result := &AuthorizationHeader{}

	// Parse key="value" pairs
	for len(content) > 0 {
		// Find next key="value" pair
		var key, value string

		// Find the key name
		eqIdx := strings.Index(content, "=\"")
		if eqIdx < 0 {
			break
		}
		key = strings.TrimSpace(content[:eqIdx])

		// Find the closing quote
		content = content[eqIdx+2:]
		closeIdx := strings.Index(content, "\"")
		if closeIdx < 0 {
			return nil, fmt.Errorf("unclosed quote in authorization header")
		}
		value = content[:closeIdx]
		content = content[closeIdx+1:]

		// Skip comma if present
		content = strings.TrimLeft(content, ", ")

		switch key {
		case "origin":
			result.Origin = value
		case "key":
			result.KeyID = value
		case "sig":
			result.Sig = value
		}
	}

	if result.Origin == "" {
		return nil, fmt.Errorf("missing origin in authorization header")
	}
	if result.KeyID == "" {
		return nil, fmt.Errorf("missing key in authorization header")
	}
	if result.Sig == "" {
		return nil, fmt.Errorf("missing sig in authorization header")
	}

	return result, nil
}

// FederationMiddleware provides X-Matrix authentication for incoming requests.
type FederationMiddleware struct {
	// keyStore provides access to our signing keys (for verifying)
	keyStore *KeyStore

	// originValidator validates the origin server name
	originValidator func(origin string) bool

	// maxRequestAge is the maximum age of a request in milliseconds
	// to prevent replay attacks. Default: 5 minutes.
	maxRequestAge int64
}

// FederationMiddlewareOption configures the middleware.
type FederationMiddlewareOption func(*FederationMiddleware)

// WithMaxRequestAge sets the maximum request age for replay protection.
func WithMaxRequestAge(age time.Duration) FederationMiddlewareOption {
	return func(fm *FederationMiddleware) {
		fm.maxRequestAge = age.Milliseconds()
	}
}

// WithOriginValidator sets a custom origin validator function.
func WithOriginValidator(validator func(origin string) bool) FederationMiddlewareOption {
	return func(fm *FederationMiddleware) {
		fm.originValidator = validator
	}
}

// NewFederationMiddleware creates a new federation middleware.
func NewFederationMiddleware(keyStore *KeyStore, opts ...FederationMiddlewareOption) *FederationMiddleware {
	fm := &FederationMiddleware{
		keyStore:        keyStore,
		maxRequestAge:   5 * 60 * 1000, // 5 minutes in milliseconds
		originValidator: func(_ string) bool { return true },
	}

	for _, opt := range opts {
		opt(fm)
	}

	return fm
}

// VerifyXMatrix returns a Fiber middleware that verifies X-Matrix signatures.
func (fm *FederationMiddleware) VerifyXMatrix() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only verify federation endpoints (paths starting with /_matrix/federation)
		if !strings.HasPrefix(c.Path(), "/_matrix/federation") {
			return c.Next()
		}

		// Skip verification for endpoints that don't require it
		// (e.g., key queries, well-known)
		if isUnauthenticatedEndpoint(c.Path()) {
			return c.Next()
		}

		// Parse the Authorization header
		authHeader := string(c.Request().Header.Peek("Authorization"))
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(map[string]interface{}{
				"errcode": "M_UNAUTHORIZED",
				"error":   "Missing Authorization header",
			})
		}

		auth, err := ParseAuthorizationHeader(authHeader)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(map[string]interface{}{
				"errcode": "M_UNAUTHORIZED",
				"error":   fmt.Sprintf("Invalid authorization header: %v", err),
			})
		}

		// Validate origin
		if !fm.originValidator(auth.Origin) {
			return c.Status(http.StatusForbidden).JSON(map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Origin %q not allowed", auth.Origin),
			})
		}

		// Check request timestamp for replay protection
		if err := fm.checkRequestAge(c); err != nil {
			return c.Status(http.StatusForbidden).JSON(map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   err.Error(),
			})
		}

		// Verify the signature
		if err := fm.verifyRequestSignature(c, auth); err != nil {
			return c.Status(http.StatusForbidden).JSON(map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Signature verification failed: %v", err),
			})
		}

		// Store verified origin in context for handlers
		c.Locals("matrix_origin", auth.Origin)
		c.Locals("matrix_key_id", auth.KeyID)

		return c.Next()
	}
}

// isUnauthenticatedEndpoint returns true for endpoints that don't require auth.
func isUnauthenticatedEndpoint(path string) bool {
	unauthenticated := []string{
		"/_matrix/federation/v1/version",
		"/_matrix/key/v2/server",  // Key queries
		"/_matrix/key/v2/server/", // Key queries with key ID
	}

	for _, prefix := range unauthenticated {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// checkRequestAge verifies the request timestamp is within acceptable bounds.
func (fm *FederationMiddleware) checkRequestAge(c *fiber.Ctx) error {
	now := time.Now()
	maxAge := time.Duration(fm.maxRequestAge) * time.Millisecond
	futureTolerance := time.Minute

	// Try X-Matrix-Origin-TS header first
	tsHeader := string(c.Request().Header.Peek("X-Matrix-Origin-TS"))
	if tsHeader != "" {
		ts, err := strconv.ParseInt(tsHeader, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid timestamp header")
		}
		reqTime := time.UnixMilli(ts)
		if now.Sub(reqTime) > maxAge {
			return fmt.Errorf("request too old")
		}
		if reqTime.Sub(now) > futureTolerance {
			return fmt.Errorf("request timestamp is in the future")
		}
		return nil
	}

	// Fall back to Date header
	dateHeader := string(c.Request().Header.Peek("Date"))
	if dateHeader != "" {
		reqTime, err := http.ParseTime(dateHeader)
		if err != nil {
			return fmt.Errorf("invalid date header")
		}
		if now.Sub(reqTime) > maxAge {
			return fmt.Errorf("request too old")
		}
		if reqTime.Sub(now) > futureTolerance {
			return fmt.Errorf("request timestamp is in the future")
		}
		return nil
	}

	return fmt.Errorf("missing timestamp header")
}

// verifyRequestSignature verifies the X-Matrix signature on the request.
func (fm *FederationMiddleware) verifyRequestSignature(c *fiber.Ctx, auth *AuthorizationHeader) error {
	// Get the key for verification
	key, err := fm.keyStore.GetKey(auth.KeyID)
	if err != nil {
		// If key not found, try fetching from the origin server
		// For now, return error
		return fmt.Errorf("key %q not found: %w", auth.KeyID, err)
	}

	// Build the canonical JSON to verify
	canonicalJSON, err := fm.buildCanonicalJSON(c, auth.Origin)
	if err != nil {
		return fmt.Errorf("failed to build canonical JSON: %w", err)
	}

	// Verify (signature is already base64-encoded string)
	if err := key.Verify(canonicalJSON, auth.Sig); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// buildCanonicalJSON builds the canonical JSON string for signature verification.
// Per Matrix spec, this includes the method, path, and content.
func (fm *FederationMiddleware) buildCanonicalJSON(c *fiber.Ctx, origin string) ([]byte, error) {
	method := string(c.Request().Header.Method())
	path := c.Request().URI().RequestURI()

	// Read body
	body := c.Body()

	// Build the canonical JSON object
	canonical := map[string]interface{}{
		"method":  method,
		"uri":     string(path),
		"origin":  origin,
		"content": nil,
	}

	// If there's a body, hash it
	if len(body) > 0 {
		canonical["content"] = string(body)
	}

	return eventCanonicalJSON(canonical)
}

// FederationClientMiddleware provides outbound request signing.
// This wraps an HTTP client to automatically sign requests.
type FederationClientMiddleware struct {
	keyStore   *KeyStore
	serverName string
}

// NewFederationClientMiddleware creates a new client middleware for signing outbound requests.
func NewFederationClientMiddleware(keyStore *KeyStore, serverName string) *FederationClientMiddleware {
	return &FederationClientMiddleware{
		keyStore:   keyStore,
		serverName: serverName,
	}
}

// SignRequest signs an outbound HTTP request with X-Matrix authorization.
func (fcm *FederationClientMiddleware) SignRequest(req *http.Request) error {
	// Get the primary signing key
	key, err := fcm.keyStore.GetPrimaryKey()
	if err != nil {
		return fmt.Errorf("no signing key available: %w", err)
	}

	// Read the request body
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Build canonical JSON
	canonical := map[string]interface{}{
		"method":  req.Method,
		"uri":     req.URL.RequestURI(),
		"origin":  fcm.serverName,
		"content": nil,
	}

	if len(body) > 0 {
		canonical["content"] = string(body)
	}

	canonicalJSON, err := eventCanonicalJSON(canonical)
	if err != nil {
		return fmt.Errorf("failed to canonicalize request: %w", err)
	}

	// Sign (returns base64-encoded string)
	sig, err := key.Sign(canonicalJSON)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Add authorization header
	authHeader := fmt.Sprintf(
		`X-Matrix origin="%s",key="%s",sig="%s"`,
		fcm.serverName,
		key.KeyID,
		sig,
	)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Matrix-Origin-TS", strconv.FormatInt(time.Now().UnixMilli(), 10))

	return nil
}
