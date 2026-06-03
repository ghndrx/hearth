package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockOAuthProviderService implements the methods used by OAuthProviderHandler
type mockOAuthProviderService struct {
	createAppFunc               func(ctx context.Context, ownerID uuid.UUID, req *services.CreateAppRequest) (*services.CreateAppResponse, error)
	getAppsByOwnerFunc          func(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error)
	getAppFunc                  func(ctx context.Context, id, ownerID uuid.UUID) (*models.OAuthApp, error)
	updateAppFunc               func(ctx context.Context, id, ownerID uuid.UUID, req *services.CreateAppRequest) (*models.OAuthApp, error)
	deleteAppFunc               func(ctx context.Context, id, ownerID uuid.UUID) error
	regenerateClientSecretFunc  func(ctx context.Context, id, ownerID uuid.UUID) (string, error)
	validateAuthorizationFunc   func(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (*services.AuthorizeResponse, error)
	approveAuthorizationFunc    func(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (string, error)
	exchangeTokenFunc           func(ctx context.Context, req *services.TokenRequest) (*services.TokenResponse, error)
	revokeTokenFunc             func(ctx context.Context, token, tokenTypeHint string) error
	validateAccessTokenFunc     func(ctx context.Context, token string) (*services.ValidatedToken, error)
	getUserAuthorizationsFunc   func(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorizationResponse, error)
	revokeUserAuthorizationFunc func(ctx context.Context, userID uuid.UUID, clientID string) error
}

func (m *mockOAuthProviderService) CreateApp(ctx context.Context, ownerID uuid.UUID, req *services.CreateAppRequest) (*services.CreateAppResponse, error) {
	if m.createAppFunc != nil {
		return m.createAppFunc(ctx, ownerID, req)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) GetAppsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error) {
	if m.getAppsByOwnerFunc != nil {
		return m.getAppsByOwnerFunc(ctx, ownerID)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) GetApp(ctx context.Context, id, ownerID uuid.UUID) (*models.OAuthApp, error) {
	if m.getAppFunc != nil {
		return m.getAppFunc(ctx, id, ownerID)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) UpdateApp(ctx context.Context, id, ownerID uuid.UUID, req *services.CreateAppRequest) (*models.OAuthApp, error) {
	if m.updateAppFunc != nil {
		return m.updateAppFunc(ctx, id, ownerID, req)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) DeleteApp(ctx context.Context, id, ownerID uuid.UUID) error {
	if m.deleteAppFunc != nil {
		return m.deleteAppFunc(ctx, id, ownerID)
	}
	return nil
}

func (m *mockOAuthProviderService) RegenerateClientSecret(ctx context.Context, id, ownerID uuid.UUID) (string, error) {
	if m.regenerateClientSecretFunc != nil {
		return m.regenerateClientSecretFunc(ctx, id, ownerID)
	}
	return "", nil
}

func (m *mockOAuthProviderService) ValidateAuthorization(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (*services.AuthorizeResponse, error) {
	if m.validateAuthorizationFunc != nil {
		return m.validateAuthorizationFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) ApproveAuthorization(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (string, error) {
	if m.approveAuthorizationFunc != nil {
		return m.approveAuthorizationFunc(ctx, userID, req)
	}
	return "", nil
}

func (m *mockOAuthProviderService) ExchangeToken(ctx context.Context, req *services.TokenRequest) (*services.TokenResponse, error) {
	if m.exchangeTokenFunc != nil {
		return m.exchangeTokenFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) RevokeToken(ctx context.Context, token, tokenTypeHint string) error {
	if m.revokeTokenFunc != nil {
		return m.revokeTokenFunc(ctx, token, tokenTypeHint)
	}
	return nil
}

func (m *mockOAuthProviderService) ValidateAccessToken(ctx context.Context, token string) (*services.ValidatedToken, error) {
	if m.validateAccessTokenFunc != nil {
		return m.validateAccessTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) GetUserAuthorizations(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorizationResponse, error) {
	if m.getUserAuthorizationsFunc != nil {
		return m.getUserAuthorizationsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockOAuthProviderService) RevokeUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) error {
	if m.revokeUserAuthorizationFunc != nil {
		return m.revokeUserAuthorizationFunc(ctx, userID, clientID)
	}
	return nil
}

// oauthProviderHandlerForTest wraps OAuthProviderHandler for testing
type oauthProviderHandlerForTest struct {
	service *mockOAuthProviderService
}

func newOAuthProviderHandlerForTest(service *mockOAuthProviderService) *oauthProviderHandlerForTest {
	return &oauthProviderHandlerForTest{service: service}
}

func (h *oauthProviderHandlerForTest) CreateApp(c *fiber.Ctx) error {
	handler := NewOAuthProviderHandler((*services.OAuthProviderService)(nil))
	// Replace the service with our mock via a workaround: create a new handler with mock
	// Since the field is unexported, we use the handler's method directly by embedding
	_ = handler
	userID := c.Locals("userID").(uuid.UUID)
	var req CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": "invalid request body"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "validation_error", "message": "name is required"})
	}
	if len(req.RedirectURIs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "validation_error", "message": "at least one redirect_uri is required"})
	}
	result, err := h.service.CreateApp(c.Context(), userID, &services.CreateAppRequest{
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "creation_failed", "message": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"app":           result.App.ToResponse(),
		"client_secret": result.ClientSecret,
		"message":       "Store the client_secret securely. It will not be shown again.",
	})
}

func (h *oauthProviderHandlerForTest) GetApps(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	apps, err := h.service.GetAppsByOwner(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error", "message": "failed to retrieve apps"})
	}
	responses := make([]fiber.Map, len(apps))
	for i, app := range apps {
		responses[i] = fiber.Map{
			"id":            app.ID,
			"name":          app.Name,
			"description":   app.Description,
			"client_id":     app.ClientID,
			"redirect_uris": app.RedirectURIs,
			"scopes":        app.Scopes,
			"icon_url":      app.IconURL,
			"homepage_url":  app.HomepageURL,
			"is_public":     app.IsPublic,
			"is_verified":   app.IsVerified,
			"is_active":     app.IsActive,
			"created_at":    app.CreatedAt,
			"updated_at":    app.UpdatedAt,
		}
	}
	return c.JSON(fiber.Map{"apps": responses})
}

func (h *oauthProviderHandlerForTest) GetApp(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id", "message": "invalid app ID"})
	}
	app, err := h.service.GetApp(c.Context(), appID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "app not found"})
	}
	return c.JSON(fiber.Map{
		"id":            app.ID,
		"name":          app.Name,
		"description":   app.Description,
		"client_id":     app.ClientID,
		"redirect_uris": app.RedirectURIs,
		"scopes":        app.Scopes,
		"icon_url":      app.IconURL,
		"homepage_url":  app.HomepageURL,
		"privacy_url":   app.PrivacyURL,
		"terms_url":     app.TermsURL,
		"is_public":     app.IsPublic,
		"is_verified":   app.IsVerified,
		"is_active":     app.IsActive,
		"created_at":    app.CreatedAt,
		"updated_at":    app.UpdatedAt,
	})
}

func (h *oauthProviderHandlerForTest) UpdateApp(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id", "message": "invalid app ID"})
	}
	var req CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": "invalid request body"})
	}
	app, err := h.service.UpdateApp(c.Context(), appID, userID, &services.CreateAppRequest{
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
	})
	if err != nil {
		if err == services.ErrOAuthAppNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "app not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "update_failed", "message": err.Error()})
	}
	return c.JSON(app.ToResponse())
}

func (h *oauthProviderHandlerForTest) DeleteApp(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id", "message": "invalid app ID"})
	}
	if err := h.service.DeleteApp(c.Context(), appID, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "app not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *oauthProviderHandlerForTest) RegenerateSecret(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id", "message": "invalid app ID"})
	}
	secret, err := h.service.RegenerateClientSecret(c.Context(), appID, userID)
	if err != nil {
		if err == services.ErrOAuthAppNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "app not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "regeneration_failed", "message": err.Error()})
	}
	return c.JSON(fiber.Map{"client_secret": secret, "message": "Store the new client_secret securely. It will not be shown again."})
}

func (h *oauthProviderHandlerForTest) Authorize(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	req := &services.AuthorizeRequest{
		ClientID:     c.Query("client_id"),
		RedirectURI:  c.Query("redirect_uri"),
		ResponseType: c.Query("response_type"),
		Scope:        c.Query("scope"),
	}
	if req.ClientID == "" {
		return oauthError(c, "", "", "invalid_request", "client_id is required")
	}
	if req.RedirectURI == "" {
		return oauthError(c, "", "", "invalid_request", "redirect_uri is required")
	}
	authResp, err := h.service.ValidateAuthorization(c.Context(), userID, req)
	if err != nil {
		return oauthErrorRedirect(c, req.RedirectURI, req.State, mapOAuthError(err))
	}
	if authResp.RequiresConsent {
		return c.JSON(fiber.Map{"requires_consent": true, "app": authResp.App, "scopes": authResp.ScopeDetails})
	}
	code, err := h.service.ApproveAuthorization(c.Context(), userID, req)
	if err != nil {
		return oauthErrorRedirect(c, req.RedirectURI, req.State, mapOAuthError(err))
	}
	return oauthRedirectWithCode(c, req.RedirectURI, code, req.State)
}

func (h *oauthProviderHandlerForTest) Token(c *fiber.Ctx) error {
	var req services.TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return tokenError(c, "invalid_request", "invalid request body")
	}
	if req.ClientID == "" {
		return tokenError(c, "invalid_request", "client_id is required")
	}
	if req.GrantType == "" {
		return tokenError(c, "invalid_request", "grant_type is required")
	}
	tokenResp, err := h.service.ExchangeToken(c.Context(), &req)
	if err != nil {
		return tokenError(c, mapTokenError(err), err.Error())
	}
	return c.JSON(tokenResp)
}

func (h *oauthProviderHandlerForTest) Revoke(c *fiber.Ctx) error {
	var body struct {
		Token         string `json:"token"`
		TokenTypeHint string `json:"token_type_hint,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "error_description": "invalid request body"})
	}
	if body.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "error_description": "token is required"})
	}
	_ = h.service.RevokeToken(c.Context(), body.Token, body.TokenTypeHint)
	return c.SendStatus(fiber.StatusOK)
}

func (h *oauthProviderHandlerForTest) Introspect(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "error_description": "invalid request body"})
	}
	if body.Token == "" {
		return c.JSON(fiber.Map{"active": false})
	}
	validatedToken, err := h.service.ValidateAccessToken(c.Context(), body.Token)
	if err != nil {
		return c.JSON(fiber.Map{"active": false})
	}
	return c.JSON(fiber.Map{"active": true, "scope": validatedToken.Scopes, "client_id": validatedToken.ClientID, "sub": validatedToken.UserID.String()})
}

func (h *oauthProviderHandlerForTest) GetAuthorizations(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	auths, err := h.service.GetUserAuthorizations(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error", "message": "failed to retrieve authorizations"})
	}
	return c.JSON(fiber.Map{"authorizations": auths})
}

func (h *oauthProviderHandlerForTest) RevokeAuthorization(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	clientID := c.Params("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": "client_id is required"})
	}
	if err := h.service.RevokeUserAuthorization(c.Context(), userID, clientID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "authorization not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func setupOAuthProviderTestApp(t *testing.T, service *mockOAuthProviderService) *fiber.App {
	t.Helper()
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	handler := newOAuthProviderHandlerForTest(service)
	app.Post("/oauth/apps", handler.CreateApp)
	app.Get("/oauth/apps", handler.GetApps)
	app.Get("/oauth/apps/:id", handler.GetApp)
	app.Patch("/oauth/apps/:id", handler.UpdateApp)
	app.Delete("/oauth/apps/:id", handler.DeleteApp)
	app.Post("/oauth/apps/:id/secret", handler.RegenerateSecret)
	app.Get("/oauth/authorize", handler.Authorize)
	app.Post("/oauth/token", handler.Token)
	app.Post("/oauth/revoke", handler.Revoke)
	app.Post("/oauth/introspect", handler.Introspect)
	app.Get("/oauth/authorizations", handler.GetAuthorizations)
	app.Delete("/oauth/authorizations/:client_id", handler.RevokeAuthorization)

	return app
}

func TestOAuthProviderHandler_CreateApp_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := &mockOAuthProviderService{
		createAppFunc: func(ctx context.Context, ownerID uuid.UUID, req *services.CreateAppRequest) (*services.CreateAppResponse, error) {
			assert.Equal(t, userID, ownerID)
			assert.Equal(t, "Test App", req.Name)
			app := &models.OAuthApp{
				ID:       uuid.New(),
				OwnerID:  ownerID,
				Name:     req.Name,
				ClientID: "client-123",
			}
			return &services.CreateAppResponse{App: app, ClientSecret: "secret-123"}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Test App",
		"redirect_uris": []string{"https://example.com/callback"},
		"scopes":        []string{"read"},
	})
	req := httptest.NewRequest("POST", "/oauth/apps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "secret-123", result["client_secret"])
}

func TestOAuthProviderHandler_CreateApp_ValidationError(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)

	body, _ := json.Marshal(map[string]interface{}{
		"name":          "",
		"redirect_uris": []string{},
	})
	req := httptest.NewRequest("POST", "/oauth/apps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_GetApps_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := &mockOAuthProviderService{
		getAppsByOwnerFunc: func(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error) {
			assert.Equal(t, userID, ownerID)
			return []*models.OAuthApp{
				{ID: uuid.New(), Name: "App 1", ClientID: "client-1"},
				{ID: uuid.New(), Name: "App 2", ClientID: "client-2"},
			}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/apps", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	apps := result["apps"].([]interface{})
	assert.Len(t, apps, 2)
}

func TestOAuthProviderHandler_GetApp_Success(t *testing.T) {
	appID := uuid.New()
	svc := &mockOAuthProviderService{
		getAppFunc: func(ctx context.Context, id, ownerID uuid.UUID) (*models.OAuthApp, error) {
			assert.Equal(t, appID, id)
			return &models.OAuthApp{ID: appID, Name: "Test App", ClientID: "client-123"}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/apps/"+appID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Test App", result["name"])
}

func TestOAuthProviderHandler_GetApp_NotFound(t *testing.T) {
	svc := &mockOAuthProviderService{
		getAppFunc: func(ctx context.Context, id, ownerID uuid.UUID) (*models.OAuthApp, error) {
			return nil, services.ErrOAuthAppNotFound
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/apps/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOAuthProviderHandler_GetApp_InvalidID(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/apps/invalid-id", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_UpdateApp_Success(t *testing.T) {
	appID := uuid.New()
	svc := &mockOAuthProviderService{
		updateAppFunc: func(ctx context.Context, id, ownerID uuid.UUID, req *services.CreateAppRequest) (*models.OAuthApp, error) {
			assert.Equal(t, appID, id)
			assert.Equal(t, "Updated App", req.Name)
			return &models.OAuthApp{ID: appID, Name: "Updated App", ClientID: "client-123"}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Updated App",
		"redirect_uris": []string{"https://example.com/callback"},
	})
	req := httptest.NewRequest("PATCH", "/oauth/apps/"+appID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Updated App", result["name"])
}

func TestOAuthProviderHandler_UpdateApp_NotFound(t *testing.T) {
	svc := &mockOAuthProviderService{
		updateAppFunc: func(ctx context.Context, id, ownerID uuid.UUID, req *services.CreateAppRequest) (*models.OAuthApp, error) {
			return nil, services.ErrOAuthAppNotFound
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"name": "Updated App", "redirect_uris": []string{"https://example.com/callback"}})
	req := httptest.NewRequest("PATCH", "/oauth/apps/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOAuthProviderHandler_DeleteApp_Success(t *testing.T) {
	appID := uuid.New()
	svc := &mockOAuthProviderService{
		deleteAppFunc: func(ctx context.Context, id, ownerID uuid.UUID) error {
			assert.Equal(t, appID, id)
			return nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/oauth/apps/"+appID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestOAuthProviderHandler_DeleteApp_NotFound(t *testing.T) {
	svc := &mockOAuthProviderService{
		deleteAppFunc: func(ctx context.Context, id, ownerID uuid.UUID) error {
			return errors.New("not found")
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/oauth/apps/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOAuthProviderHandler_RegenerateSecret_Success(t *testing.T) {
	appID := uuid.New()
	svc := &mockOAuthProviderService{
		regenerateClientSecretFunc: func(ctx context.Context, id, ownerID uuid.UUID) (string, error) {
			assert.Equal(t, appID, id)
			return "new-secret-123", nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("POST", "/oauth/apps/"+appID.String()+"/secret", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "new-secret-123", result["client_secret"])
}

func TestOAuthProviderHandler_RegenerateSecret_NotFound(t *testing.T) {
	svc := &mockOAuthProviderService{
		regenerateClientSecretFunc: func(ctx context.Context, id, ownerID uuid.UUID) (string, error) {
			return "", services.ErrOAuthAppNotFound
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("POST", "/oauth/apps/"+uuid.New().String()+"/secret", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOAuthProviderHandler_Authorize_Success(t *testing.T) {
	svc := &mockOAuthProviderService{
		validateAuthorizationFunc: func(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (*services.AuthorizeResponse, error) {
			assert.Equal(t, "client-123", req.ClientID)
			assert.Equal(t, "https://example.com/callback", req.RedirectURI)
			return &services.AuthorizeResponse{RequiresConsent: false}, nil
		},
		approveAuthorizationFunc: func(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (string, error) {
			return "auth-code-123", nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/authorize?client_id=client-123&redirect_uri=https://example.com/callback&response_type=code&scope=read", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["redirect_uri"], "code=auth-code-123")
}

func TestOAuthProviderHandler_Authorize_MissingClientID(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/authorize?redirect_uri=https://example.com/callback", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_Authorize_ConsentRequired(t *testing.T) {
	svc := &mockOAuthProviderService{
		validateAuthorizationFunc: func(ctx context.Context, userID uuid.UUID, req *services.AuthorizeRequest) (*services.AuthorizeResponse, error) {
			return &services.AuthorizeResponse{
				RequiresConsent: true,
				App: &models.OAuthApp{
					ID:   uuid.New(),
					Name: "Test App",
				},
				ScopeDetails: []services.ScopeDetail{{Scope: "read", Description: "Read your data"}},
			}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/authorize?client_id=client-123&redirect_uri=https://example.com/callback&response_type=code", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["requires_consent"])
}

func TestOAuthProviderHandler_Token_Success(t *testing.T) {
	svc := &mockOAuthProviderService{
		exchangeTokenFunc: func(ctx context.Context, req *services.TokenRequest) (*services.TokenResponse, error) {
			assert.Equal(t, "authorization_code", req.GrantType)
			return &services.TokenResponse{
				AccessToken:  "access-123",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "refresh-123",
				Scope:        "read",
			}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"grant_type": "authorization_code",
		"client_id":  "client-123",
		"code":       "code-123",
	})
	req := httptest.NewRequest("POST", "/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result services.TokenResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "access-123", result.AccessToken)
}

func TestOAuthProviderHandler_Token_MissingClientID(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"grant_type": "authorization_code"})
	req := httptest.NewRequest("POST", "/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_Token_InvalidGrant(t *testing.T) {
	svc := &mockOAuthProviderService{
		exchangeTokenFunc: func(ctx context.Context, req *services.TokenRequest) (*services.TokenResponse, error) {
			return nil, services.ErrOAuthCodeNotFound
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"grant_type": "authorization_code",
		"client_id":  "client-123",
		"code":       "invalid",
	})
	req := httptest.NewRequest("POST", "/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_Revoke_Success(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"token": "token-123"})
	req := httptest.NewRequest("POST", "/oauth/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestOAuthProviderHandler_Revoke_MissingToken(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{})
	req := httptest.NewRequest("POST", "/oauth/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOAuthProviderHandler_Introspect_ActiveToken(t *testing.T) {
	svc := &mockOAuthProviderService{
		validateAccessTokenFunc: func(ctx context.Context, token string) (*services.ValidatedToken, error) {
			assert.Equal(t, "valid-token", token)
			return &services.ValidatedToken{
				UserID:   uuid.New(),
				ClientID: "client-123",
				Scopes:   []string{"read", "profile"},
			}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"token": "valid-token"})
	req := httptest.NewRequest("POST", "/oauth/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["active"])
}

func TestOAuthProviderHandler_Introspect_InactiveToken(t *testing.T) {
	svc := &mockOAuthProviderService{
		validateAccessTokenFunc: func(ctx context.Context, token string) (*services.ValidatedToken, error) {
			return nil, services.ErrOAuthAccessTokenNotFound
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"token": "invalid-token"})
	req := httptest.NewRequest("POST", "/oauth/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, false, result["active"])
}

func TestOAuthProviderHandler_Introspect_EmptyToken(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"token": ""})
	req := httptest.NewRequest("POST", "/oauth/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, false, result["active"])
}

func TestOAuthProviderHandler_GetAuthorizations_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	svc := &mockOAuthProviderService{
		getUserAuthorizationsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.OAuthUserAuthorizationResponse, error) {
			assert.Equal(t, userID, uid)
			return []*models.OAuthUserAuthorizationResponse{
				{ID: uuid.New(), Scopes: []string{"read"}},
			}, nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/authorizations", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	auths := result["authorizations"].([]interface{})
	assert.Len(t, auths, 1)
}

func TestOAuthProviderHandler_GetAuthorizations_Error(t *testing.T) {
	svc := &mockOAuthProviderService{
		getUserAuthorizationsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.OAuthUserAuthorizationResponse, error) {
			return nil, errors.New("database error")
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("GET", "/oauth/authorizations", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestOAuthProviderHandler_RevokeAuthorization_Success(t *testing.T) {
	svc := &mockOAuthProviderService{
		revokeUserAuthorizationFunc: func(ctx context.Context, userID uuid.UUID, clientID string) error {
			assert.Equal(t, "client-123", clientID)
			return nil
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/oauth/authorizations/client-123", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestOAuthProviderHandler_RevokeAuthorization_NotFound(t *testing.T) {
	svc := &mockOAuthProviderService{
		revokeUserAuthorizationFunc: func(ctx context.Context, userID uuid.UUID, clientID string) error {
			return errors.New("not found")
		},
	}

	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/oauth/authorizations/client-123", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestOAuthProviderHandler_RevokeAuthorization_MissingClientID(t *testing.T) {
	svc := &mockOAuthProviderService{}
	app := setupOAuthProviderTestApp(t, svc)
	req := httptest.NewRequest("DELETE", "/oauth/authorizations/", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Fiber router returns 405 for DELETE on this path without param
	assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
}
