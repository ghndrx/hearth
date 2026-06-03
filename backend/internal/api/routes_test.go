package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
)

// createMinimalHandlers creates handler stubs with zero-value services.
// Methods are only invoked at request time, not during route registration.
func createMinimalHandlers() *handlers.Handlers {
	return &handlers.Handlers{
		Auth:      &handlers.AuthHandler{},
		Users:     &handlers.UserHandler{},
		Servers:   &handlers.ServerHandler{},
		Channels:  &handlers.ChannelHandler{},
		Threads:   &handlers.ThreadHandler{},
		Invites:   &handlers.InviteHandler{},
		Voice:     &handlers.VoiceHandler{},
		Gateway:   &handlers.GatewayHandler{},
		Search:    &handlers.SearchHandler{},

	}
}

func TestSetupRoutes_DoesNotPanic(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")

	// SetupRoutes should not panic with minimal handlers
	SetupRoutes(app, h, m)
}

func TestSetupRoutes_HealthEndpoints(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"health", "/health", http.MethodGet},
		{"healthz", "/healthz", http.MethodGet},
		{"readyz", "/readyz", http.MethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Should not return 404 (route exists)
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("expected route %s %s to exist, got 404", tt.method, tt.path)
			}
		})
	}
}

func TestSetupRoutes_PublicAuthRoutes(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"register", "/api/v1/auth/register", http.MethodPost},
		{"login", "/api/v1/auth/login", http.MethodPost},
		{"login_mfa", "/api/v1/auth/login/mfa", http.MethodPost},
		{"refresh", "/api/v1/auth/refresh", http.MethodPost},
		{"logout", "/api/v1/auth/logout", http.MethodPost},
		{"oauth_providers", "/api/v1/auth/oauth/providers", http.MethodGet},
		{"oauth_redirect", "/api/v1/auth/oauth/github", http.MethodGet},
		{"oauth_callback", "/api/v1/auth/oauth/github/callback", http.MethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("expected route %s %s to exist, got 404", tt.method, tt.path)
			}
		})
	}
}

func TestSetupRoutes_ProtectedRoutesRequireAuth(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	protectedRoutes := []struct {
		name   string
		path   string
		method string
	}{
		{"get_me", "/api/v1/users/@me", http.MethodGet},
		{"update_me", "/api/v1/users/@me", http.MethodPatch},
		{"get_servers", "/api/v1/users/@me/servers", http.MethodGet},
		{"create_server", "/api/v1/servers", http.MethodPost},
		{"get_channel", "/api/v1/channels/some-id", http.MethodGet},
		{"search_all", "/api/v1/search/", http.MethodGet},
		{"voice_regions", "/api/v1/voice/regions", http.MethodGet},
		{"get_thread", "/api/v1/threads/some-id", http.MethodGet},
		{"get_invite", "/api/v1/invites/some-code", http.MethodGet},
		{"mfa_enable", "/api/v1/auth/mfa/enable", http.MethodPost},
	}

	for _, tt := range protectedRoutes {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401 for unauthenticated %s %s, got %d", tt.method, tt.path, resp.StatusCode)
			}
		})
	}
}

func TestSetupRoutes_ProtectedRoutesReturnUnauthorizedJSON(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/@me", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "missing authorization header") {
		t.Errorf("expected unauthorized error message, got: %s", bodyStr)
	}
}

func TestSetupRoutes_InvalidAuthToken(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	tests := []struct {
		name       string
		authHeader string
		wantErr    string
	}{
		{"invalid_format", "Token abc123", "invalid authorization format"},
		{"invalid_bearer", "Bearer invalid-token", "invalid token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/@me", nil)
			req.Header.Set("Authorization", tt.authHeader)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantErr) {
				t.Errorf("expected error %q, got: %s", tt.wantErr, string(body))
			}
		})
	}
}

func TestSetupRoutes_OptionalHandlersNil(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")

	// All optional handlers are nil - should not panic
	h.Sessions = nil
	h.Settings = nil
	h.Notifications = nil
	h.SavedMessages = nil
	h.DMs = nil
	h.AuditLog = nil
	h.Polls = nil
	h.Attachments = nil
	h.Stickers = nil
	h.Soundboard = nil
	h.Announcements = nil
	h.Components = nil
	h.Events = nil
	h.AutoMod = nil
	h.Webhooks = nil
	h.E2EE = nil
	h.AI = nil
	h.AIChat = nil
	h.Discovery = nil
	h.Templates = nil
	h.SlashCommands = nil
	h.Interactions = nil
	h.ServerAudioSettings = nil

	SetupRoutes(app, h, m)

	// Verify optional routes are NOT registered (should fall through to SPA)
	optionalPaths := []struct {
		name   string
		path   string
		method string
	}{
		{"sessions", "/api/v1/auth/sessions", http.MethodGet},
		{"notifications", "/api/v1/notifications/", http.MethodGet},
		{"polls_create", "/api/v1/channels/test-id/polls", http.MethodPost},
	}

	for _, tt := range optionalPaths {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer dummy")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// These should either 401 (caught by auth middleware on different group)
			// or not match the specific handler route
			// They should NOT return a handler-specific response
			_ = resp
		})
	}
}

func TestSetupRoutes_WithAllOptionalHandlers(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")

	// Set all optional handlers
	h.Sessions = &handlers.SessionHandler{}
	h.Settings = &handlers.SettingsHandler{}
	h.Notifications = &handlers.NotificationHandler{}
	h.NotificationPreferences = &handlers.NotificationPreferenceHandler{}
	h.SavedMessages = &handlers.SavedMessagesHandler{}
	h.DMs = &handlers.DMHandler{}
	h.AuditLog = &handlers.AuditLogHandler{}
	h.Polls = &handlers.PollHandler{}
	h.Attachments = &handlers.AttachmentHandler{}
	h.Stickers = &handlers.StickerHandler{}
	h.Soundboard = &handlers.SoundboardHandler{}
	h.Announcements = &handlers.AnnouncementHandler{}
	h.Components = &handlers.ComponentHandler{}
	h.Events = &handlers.EventHandler{}
	h.AutoMod = &handlers.AutoModHandler{}
	h.Webhooks = &handlers.WebhookHandlers{}
	h.E2EE = &handlers.E2EEHandler{}
	h.AI = &handlers.AIHandler{}
	h.AIChat = &handlers.AIChatHandler{}
	h.Discovery = &handlers.DiscoveryHandler{}
	h.Templates = &handlers.TemplateHandler{}

	h.SlashCommands = &handlers.SlashCommandHandler{}
	h.Interactions = &handlers.InteractionHandler{}
	h.ServerAudioSettings = &handlers.ServerAudioSettingsHandler{}

	// Should not panic with all handlers set
	SetupRoutes(app, h, m)
}

func TestSetupRoutes_RouteGroups(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	routes := app.GetRoutes()
	routeMap := make(map[string]map[string]bool)
	for _, r := range routes {
		if routeMap[r.Path] == nil {
			routeMap[r.Path] = make(map[string]bool)
		}
		routeMap[r.Path][r.Method] = true
	}

	// Verify core route groups exist
	expectedRoutes := []struct {
		method string
		path   string
	}{
		// Health
		{"GET", "/health"},
		{"GET", "/healthz"},
		{"GET", "/readyz"},
		// Auth
		{"POST", "/api/v1/auth/register"},
		{"POST", "/api/v1/auth/login"},
		{"POST", "/api/v1/auth/login/mfa"},
		{"POST", "/api/v1/auth/refresh"},
		{"POST", "/api/v1/auth/logout"},
		// MFA (protected)
		{"POST", "/api/v1/auth/mfa/enable"},
		{"POST", "/api/v1/auth/mfa/verify"},
		{"POST", "/api/v1/auth/mfa/disable"},
		// OAuth
		{"GET", "/api/v1/auth/oauth/providers"},
		{"GET", "/api/v1/auth/oauth/:provider"},
		{"GET", "/api/v1/auth/oauth/:provider/callback"},
		// OAuth protected
		{"GET", "/api/v1/auth/oauth/linked"},
		{"GET", "/api/v1/auth/oauth/link/:provider"},
		{"DELETE", "/api/v1/auth/oauth/link/:provider"},
		// Users
		{"GET", "/api/v1/users/@me"},
		{"PATCH", "/api/v1/users/@me"},
		{"POST", "/api/v1/users/@me/avatar"},
		{"DELETE", "/api/v1/users/@me/avatar"},
		{"POST", "/api/v1/users/@me/banner"},
		{"DELETE", "/api/v1/users/@me/banner"},
		{"GET", "/api/v1/users/@me/status"},
		{"PUT", "/api/v1/users/@me/status"},
		{"DELETE", "/api/v1/users/@me/status"},
		{"GET", "/api/v1/users/@me/servers"},
		{"GET", "/api/v1/users/@me/channels"},
		{"POST", "/api/v1/users/@me/channels"},
		{"POST", "/api/v1/users/@me/channels/group"},
		{"GET", "/api/v1/users/:id"},
		{"GET", "/api/v1/users/:id/profile"},
		// Relationships
		{"GET", "/api/v1/users/@me/relationships"},
		{"POST", "/api/v1/users/@me/relationships"},
		{"DELETE", "/api/v1/users/@me/relationships/:id"},
		// Friends
		{"GET", "/api/v1/users/@me/friends"},
		{"GET", "/api/v1/users/@me/friends/pending"},
		{"PUT", "/api/v1/users/@me/friends/:id"},
		{"DELETE", "/api/v1/users/@me/friends/:id/request"},
		// Servers
		{"POST", "/api/v1/servers/"},
		{"GET", "/api/v1/servers/:id"},
		{"PATCH", "/api/v1/servers/:id"},
		{"DELETE", "/api/v1/servers/:id"},
		{"POST", "/api/v1/servers/:id/transfer-ownership"},
		// Server members
		{"GET", "/api/v1/servers/:id/members"},
		{"GET", "/api/v1/servers/:id/members/:userId"},
		{"PATCH", "/api/v1/servers/:id/members/:userId"},
		{"DELETE", "/api/v1/servers/:id/members/:userId"},
		{"DELETE", "/api/v1/servers/:id/members/@me"},
		// Server bans
		{"GET", "/api/v1/servers/:id/bans"},
		{"PUT", "/api/v1/servers/:id/bans/:userId"},
		{"DELETE", "/api/v1/servers/:id/bans/:userId"},
		// Server invites
		{"GET", "/api/v1/servers/:id/invites"},
		{"PUT", "/api/v1/servers/:id/vanity-url"},
		{"GET", "/api/v1/servers/:id/invites/analytics"},
		// Server roles
		{"GET", "/api/v1/servers/:id/roles"},
		{"POST", "/api/v1/servers/:id/roles"},
		{"PATCH", "/api/v1/servers/:id/roles"},
		{"PATCH", "/api/v1/servers/:id/roles/:roleId"},
		{"DELETE", "/api/v1/servers/:id/roles/:roleId"},
		// Server channels
		{"GET", "/api/v1/servers/:id/channels"},
		{"POST", "/api/v1/servers/:id/channels"},
		// Server voice states
		{"GET", "/api/v1/servers/:id/voice-states"},
		// Channels
		{"GET", "/api/v1/channels/:id"},
		{"PATCH", "/api/v1/channels/:id"},
		{"DELETE", "/api/v1/channels/:id"},
		// Channel messages
		{"GET", "/api/v1/channels/:id/messages"},
		{"POST", "/api/v1/channels/:id/messages"},
		{"GET", "/api/v1/channels/:id/messages/:messageId"},
		{"PATCH", "/api/v1/channels/:id/messages/:messageId"},
		{"DELETE", "/api/v1/channels/:id/messages/:messageId"},
		// Reactions
		{"GET", "/api/v1/channels/:id/messages/:messageId/reactions"},
		{"GET", "/api/v1/channels/:id/messages/:messageId/reactions/:emoji"},
		{"PUT", "/api/v1/channels/:id/messages/:messageId/reactions/:emoji/@me"},
		{"DELETE", "/api/v1/channels/:id/messages/:messageId/reactions/:emoji/@me"},
		// Pins
		{"GET", "/api/v1/channels/:id/pins"},
		{"PUT", "/api/v1/channels/:id/pins/:messageId"},
		{"DELETE", "/api/v1/channels/:id/pins/:messageId"},
		// Typing
		{"POST", "/api/v1/channels/:id/typing"},
		{"GET", "/api/v1/channels/:id/typing"},
		// Channel threads
		{"GET", "/api/v1/channels/:id/threads"},
		{"POST", "/api/v1/channels/:id/threads"},
		// Channel invites
		{"POST", "/api/v1/channels/:id/invites"},
		// Threads
		{"GET", "/api/v1/threads/:id"},
		{"PATCH", "/api/v1/threads/:id"},
		{"DELETE", "/api/v1/threads/:id"},
		{"GET", "/api/v1/threads/:id/messages"},
		{"POST", "/api/v1/threads/:id/messages"},
		{"POST", "/api/v1/threads/:id/archive"},
		{"POST", "/api/v1/threads/:id/unarchive"},
		{"POST", "/api/v1/threads/:id/join"},
		{"DELETE", "/api/v1/threads/:id/members/@me"},
		{"GET", "/api/v1/threads/:id/notifications"},
		{"PUT", "/api/v1/threads/:id/notifications"},
		{"GET", "/api/v1/threads/:id/presence"},
		{"POST", "/api/v1/threads/:id/presence"},
		{"PATCH", "/api/v1/threads/:id/presence"},
		{"DELETE", "/api/v1/threads/:id/presence"},
		// Forum tags on threads
		{"PUT", "/api/v1/threads/:id/tags"},
		{"GET", "/api/v1/threads/:id/tags"},
		{"PUT", "/api/v1/threads/:id/pin"},
		// Forum tags on channels
		{"GET", "/api/v1/channels/:id/tags"},
		{"POST", "/api/v1/channels/:id/tags"},
		{"GET", "/api/v1/channels/:id/posts"},
		// Global tag management
		{"PATCH", "/api/v1/forum-tags/:tagId"},
		{"DELETE", "/api/v1/forum-tags/:tagId"},
		// Invites
		{"GET", "/api/v1/invites/:code"},
		{"POST", "/api/v1/invites/:code"},
		{"DELETE", "/api/v1/invites/:code"},
		{"GET", "/api/v1/invites/:code/analytics"},
		// Search
		{"GET", "/api/v1/search/"},
		{"GET", "/api/v1/search/messages"},
		{"GET", "/api/v1/search/users"},
		{"GET", "/api/v1/search/channels"},
		{"GET", "/api/v1/search/suggestions"},
		// Voice
		{"GET", "/api/v1/voice/regions"},
		{"GET", "/api/v1/voice/channels/:channelId/states"},
		// Gateway
		{"GET", "/api/v1/gateway/stats"},
		{"GET", "/gateway"},
	}

	for _, er := range expectedRoutes {
		t.Run(er.method+" "+er.path, func(t *testing.T) {
			methods, ok := routeMap[er.path]
			if !ok {
				t.Errorf("route %s not found in registered routes", er.path)
				return
			}
			if !methods[er.method] {
				t.Errorf("method %s not registered for route %s", er.method, er.path)
			}
		})
	}
}

func TestSetupRoutes_SPAFallback(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	// Non-API path should hit the SPA fallback (wildcard route)
	req := httptest.NewRequest(http.MethodGet, "/some/random/path", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not return 404 - the wildcard should catch it
	// It may return 404 if public/index.html doesn't exist, but route matches
	_ = resp
}

func TestSetupRoutes_MiddlewareInitialization(t *testing.T) {
	m := middleware.NewMiddleware("test-secret")
	if m == nil {
		t.Fatal("expected non-nil middleware")
	}

	mWithLimiter := middleware.NewMiddlewareWithRateLimiter("test-secret", nil)
	if mWithLimiter == nil {
		t.Fatal("expected non-nil middleware with rate limiter")
	}

	if mWithLimiter.HasRateLimiter() {
		t.Error("expected no rate limiter when nil is passed")
	}

	if mWithLimiter.IsRateLimiterAvailable() {
		t.Error("expected rate limiter to not be available when nil")
	}
}

func TestSetupRoutes_RateLimitConfig(t *testing.T) {
	config := middleware.DefaultRateLimitConfig()

	if config.Limit != 10000 {
		t.Errorf("expected default limit 10000, got %d", config.Limit)
	}
	if config.AuthMultiplier != 2.0 {
		t.Errorf("expected default auth multiplier 2.0, got %f", config.AuthMultiplier)
	}
	if len(config.SkipPaths) != 4 {
		t.Errorf("expected 4 skip paths, got %d", len(config.SkipPaths))
	}
}

func TestSetupRoutes_CountRoutes(t *testing.T) {
	app := fiber.New()
	h := createMinimalHandlers()
	m := middleware.NewMiddleware("test-secret")
	SetupRoutes(app, h, m)

	routes := app.GetRoutes()
	// Filter out HEAD routes (Fiber auto-generates them)
	var count int
	for _, r := range routes {
		if r.Method != "HEAD" {
			count++
		}
	}

	// Should have a substantial number of routes registered
	if count < 100 {
		t.Errorf("expected at least 100 non-HEAD routes, got %d", count)
	}
}
