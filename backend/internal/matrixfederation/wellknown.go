// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements .well-known discovery endpoints.
//
// Matrix Spec References:
//   - Client-Server API r0.6.1 § 4.1: Server Discovery
//   - Federation API r0.1.4 § 3.1: Resolving server names
//   - https://spec.matrix.org/v1.12/client-server-api/#well-known-uri
//   - https://spec.matrix.org/v1.12/server-server-api/#resolving-server-names
package matrixfederation

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"hearth/internal/matrix"
)

// ClientWellKnown represents the response for /.well-known/matrix/client
// This tells Matrix clients how to reach the homeserver.
//
// https://spec.matrix.org/v1.12/client-server-api/#getwell-knownmatrixclient
type ClientWellKnown struct {
	// Homeserver contains the base URL for the homeserver's Client-Server API
	Homeserver HomeserverInfo `json:"m.homeserver"`

	// IdentityServer is optional and contains the base URL for the identity server
	IdentityServer *IdentityServerInfo `json:"m.identity_server,omitempty"`

	// SlidingSyncProxy is optional for MSC3575 sliding sync support
	SlidingSyncProxy *SlidingSyncInfo `json:"org.matrix.msc3575.proxy,omitempty"`
}

// HomeserverInfo contains homeserver discovery information.
type HomeserverInfo struct {
	// BaseURL is the base URL of the homeserver's Client-Server API
	// Example: "https://matrix.example.com"
	BaseURL string `json:"base_url"`
}

// IdentityServerInfo contains identity server discovery information.
type IdentityServerInfo struct {
	// BaseURL is the base URL of the identity server
	BaseURL string `json:"base_url"`
}

// SlidingSyncInfo contains sliding sync proxy information (MSC3575).
type SlidingSyncInfo struct {
	// URL is the URL of the sliding sync proxy
	URL string `json:"url"`
}

// ServerWellKnown represents the response for /.well-known/matrix/server
// This tells other Matrix servers how to reach this homeserver for federation.
//
// https://spec.matrix.org/v1.12/server-server-api/#getwell-knownmatrixserver
type ServerWellKnown struct {
	// ServerName is the server name with optional port for federation
	// Example: "matrix.example.com:8448" or "matrix.example.com"
	ServerName string `json:"m.server"`
}

// SupportWellKnown represents the response for /.well-known/matrix/support
// This provides contact information for server administrators.
//
// https://spec.matrix.org/v1.12/client-server-api/#getwell-knownmatrixsupport
type SupportWellKnown struct {
	// Contacts is a list of contact methods
	Contacts []SupportContact `json:"contacts,omitempty"`

	// SupportPage is a URL to a support page
	SupportPage string `json:"support_page,omitempty"`
}

// SupportContact represents a single contact method.
type SupportContact struct {
	// Role describes the contact's role (e.g., "m.role.admin", "m.role.security")
	Role string `json:"role,omitempty"`

	// EmailAddress is the contact email
	EmailAddress string `json:"email_address,omitempty"`

	// MatrixID is the contact's Matrix ID
	MatrixID string `json:"matrix_id,omitempty"`
}

// WellKnownHandler handles Matrix .well-known discovery endpoints.
type WellKnownHandler struct {
	homeserverCfg *matrix.HomeserverConfig
	identityURL   string // Optional identity server URL
	supportPage   string // Optional support page URL
	adminEmail    string // Optional admin email
	adminMXID     string // Optional admin Matrix ID
}

// WellKnownOptions configures the well-known handler.
type WellKnownOptions struct {
	// IdentityServerURL is the identity server base URL (optional)
	IdentityServerURL string

	// SupportPageURL is the support page URL (optional)
	SupportPageURL string

	// AdminEmail is the admin contact email (optional)
	AdminEmail string

	// AdminMXID is the admin's Matrix ID (optional)
	AdminMXID string
}

// NewWellKnownHandler creates a new handler for .well-known endpoints.
func NewWellKnownHandler(cfg *matrix.HomeserverConfig, opts *WellKnownOptions) *WellKnownHandler {
	h := &WellKnownHandler{
		homeserverCfg: cfg,
	}

	if opts != nil {
		h.identityURL = opts.IdentityServerURL
		h.supportPage = opts.SupportPageURL
		h.adminEmail = opts.AdminEmail
		h.adminMXID = opts.AdminMXID
	}

	return h
}

// GetClientWellKnown handles GET /.well-known/matrix/client
//
// Returns information about the homeserver for Matrix clients to discover
// the correct API endpoints.
func (h *WellKnownHandler) GetClientWellKnown(c *fiber.Ctx) error {
	resp := ClientWellKnown{
		Homeserver: HomeserverInfo{
			BaseURL: h.homeserverCfg.GetClientURL(),
		},
	}

	// Add identity server if configured
	if h.identityURL != "" {
		resp.IdentityServer = &IdentityServerInfo{
			BaseURL: h.identityURL,
		}
	} else if h.homeserverCfg.DefaultIdentityURL != "" {
		resp.IdentityServer = &IdentityServerInfo{
			BaseURL: h.homeserverCfg.DefaultIdentityURL,
		}
	}

	// Set cache headers (spec recommends caching for at least 24 hours)
	c.Set("Cache-Control", "public, max-age=86400")
	c.Set("Access-Control-Allow-Origin", "*")

	return c.JSON(resp)
}

// GetServerWellKnown handles GET /.well-known/matrix/server
//
// Returns the server name and port that other homeservers should use
// for federation.
func (h *WellKnownHandler) GetServerWellKnown(c *fiber.Ctx) error {
	serverName := h.homeserverCfg.ServerName

	// If a specific federation port is configured, include it
	if h.homeserverCfg.Port > 0 && h.homeserverCfg.Port != 8448 {
		serverName = h.homeserverCfg.ServerName + ":" + string(rune(h.homeserverCfg.Port))
	}

	resp := ServerWellKnown{
		ServerName: serverName,
	}

	// Set cache headers
	c.Set("Cache-Control", "public, max-age=86400")

	return c.JSON(resp)
}

// GetSupportWellKnown handles GET /.well-known/matrix/support
//
// Returns contact information for server administrators.
func (h *WellKnownHandler) GetSupportWellKnown(c *fiber.Ctx) error {
	resp := SupportWellKnown{}

	if h.supportPage != "" {
		resp.SupportPage = h.supportPage
	}

	// Add admin contact if configured
	if h.adminEmail != "" || h.adminMXID != "" {
		contact := SupportContact{
			Role: "m.role.admin",
		}
		if h.adminEmail != "" {
			contact.EmailAddress = h.adminEmail
		}
		if h.adminMXID != "" {
			contact.MatrixID = h.adminMXID
		}
		resp.Contacts = append(resp.Contacts, contact)
	}

	// If no support info is configured, return 404
	if resp.SupportPage == "" && len(resp.Contacts) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "Support information not configured",
		})
	}

	c.Set("Cache-Control", "public, max-age=86400")

	return c.JSON(resp)
}

// SetupWellKnownRoutes registers .well-known endpoints on a Fiber app.
func SetupWellKnownRoutes(app *fiber.App, handler *WellKnownHandler) {
	wellKnown := app.Group("/.well-known/matrix")

	wellKnown.Get("/client", handler.GetClientWellKnown)
	wellKnown.Get("/server", handler.GetServerWellKnown)
	wellKnown.Get("/support", handler.GetSupportWellKnown)
}

// VersionsResponse represents the response for GET /_matrix/client/versions
// and GET /_matrix/federation/v1/version
//
// https://spec.matrix.org/v1.12/client-server-api/#get_matrixclientversions
type VersionsResponse struct {
	// Versions is a list of supported Matrix protocol versions
	Versions []string `json:"versions"`

	// UnstableFeatures is a map of unstable feature flags
	UnstableFeatures map[string]bool `json:"unstable_features,omitempty"`
}

// ServerVersionResponse represents the response for GET /_matrix/federation/v1/version
type ServerVersionResponse struct {
	// Server contains information about this server implementation
	Server ServerVersionInfo `json:"server"`
}

// ServerVersionInfo contains server software information.
type ServerVersionInfo struct {
	// Name is the server software name
	Name string `json:"name"`

	// Version is the server software version
	Version string `json:"version"`
}

// VersionsHandler handles Matrix version discovery endpoints.
type VersionsHandler struct {
	serverName    string
	serverVersion string
}

// NewVersionsHandler creates a new versions handler.
func NewVersionsHandler(serverName, serverVersion string) *VersionsHandler {
	return &VersionsHandler{
		serverName:    serverName,
		serverVersion: serverVersion,
	}
}

// GetClientVersions handles GET /_matrix/client/versions
func (h *VersionsHandler) GetClientVersions(c *fiber.Ctx) error {
	resp := VersionsResponse{
		Versions: []string{
			"r0.0.1",
			"r0.1.0",
			"r0.2.0",
			"r0.3.0",
			"r0.4.0",
			"r0.5.0",
			"r0.6.0",
			"r0.6.1",
			"v1.1",
			"v1.2",
			"v1.3",
			"v1.4",
			"v1.5",
			"v1.6",
			"v1.7",
			"v1.8",
			"v1.9",
			"v1.10",
			"v1.11",
			"v1.12",
		},
		UnstableFeatures: map[string]bool{
			"org.matrix.e2e_cross_signing": true,
			"org.matrix.msc2432":           true, // Rooms v6
			"org.matrix.msc3440":           true, // Threading
		},
	}

	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("Access-Control-Allow-Origin", "*")

	return c.JSON(resp)
}

// GetFederationVersion handles GET /_matrix/federation/v1/version
func (h *VersionsHandler) GetFederationVersion(c *fiber.Ctx) error {
	resp := ServerVersionResponse{
		Server: ServerVersionInfo{
			Name:    h.serverName,
			Version: h.serverVersion,
		},
	}

	return c.JSON(resp)
}

// SetupVersionRoutes registers version endpoints.
func SetupVersionRoutes(app *fiber.App, handler *VersionsHandler) {
	app.Get("/_matrix/client/versions", handler.GetClientVersions)
	app.Get("/_matrix/federation/v1/version", handler.GetFederationVersion)
}

// KeyServerHandler handles /_matrix/key endpoints for server key discovery.
type KeyServerHandler struct {
	keyStore      *KeyStore
	validDuration time.Duration
}

// NewKeyServerHandler creates a new key server handler.
func NewKeyServerHandler(keyStore *KeyStore, validDuration time.Duration) *KeyServerHandler {
	if validDuration == 0 {
		validDuration = 24 * time.Hour
	}
	return &KeyServerHandler{
		keyStore:      keyStore,
		validDuration: validDuration,
	}
}

// GetServerKeys handles GET /_matrix/key/v2/server
// Returns the server's signing keys.
//
// https://spec.matrix.org/v1.12/server-server-api/#get_matrixkeyv2server
func (h *KeyServerHandler) GetServerKeys(c *fiber.Ctx) error {
	resp, err := h.keyStore.GetServerKeyResponse(h.validDuration)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to generate key response",
		})
	}

	c.Set("Cache-Control", "public, max-age=86400")

	return c.JSON(resp)
}

// GetServerKeyByID handles GET /_matrix/key/v2/server/{keyId}
// Returns a specific signing key by ID.
func (h *KeyServerHandler) GetServerKeyByID(c *fiber.Ctx) error {
	keyID := c.Params("keyId")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_INVALID_PARAM",
			"error":   "keyId is required",
		})
	}

	// URL-decode the keyId (e.g., ed25519%3Akey_name -> ed25519:key_name)
	// Fiber should handle this automatically via c.Params

	resp, err := h.keyStore.GetServerKeyResponse(h.validDuration)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to generate key response",
		})
	}

	// Check if the key exists
	_, currentOK := resp.VerifyKeys[keyID]
	_, oldOK := resp.OldVerifyKeys[keyID]

	if !currentOK && !oldOK {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"errcode": "M_NOT_FOUND",
			"error":   "Key not found",
		})
	}

	return c.JSON(resp)
}

// SetupKeyServerRoutes registers key server endpoints.
func SetupKeyServerRoutes(app *fiber.App, handler *KeyServerHandler) {
	app.Get("/_matrix/key/v2/server", handler.GetServerKeys)
	app.Get("/_matrix/key/v2/server/:keyId", handler.GetServerKeyByID)
}
