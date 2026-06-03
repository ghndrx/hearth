package handlers

import (
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// OAuthProviderHandler handles OAuth provider endpoints
type OAuthProviderHandler struct {
	oauthService *services.OAuthProviderService
}

// NewOAuthProviderHandler creates a new OAuth provider handler
func NewOAuthProviderHandler(oauthService *services.OAuthProviderService) *OAuthProviderHandler {
	return &OAuthProviderHandler{oauthService: oauthService}
}

// ======== App Management Endpoints ========

// CreateAppRequest represents the request to create an OAuth app
type CreateAppRequest struct {
	Name         string   `json:"name"`
	Description  *string  `json:"description,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	IconURL      *string  `json:"icon_url,omitempty"`
	HomepageURL  *string  `json:"homepage_url,omitempty"`
	PrivacyURL   *string  `json:"privacy_url,omitempty"`
	TermsURL     *string  `json:"terms_url,omitempty"`
	IsPublic     bool     `json:"is_public"`
}

// CreateApp creates a new OAuth application
// POST /oauth/apps
func (h *OAuthProviderHandler) CreateApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	// Validate
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "name is required",
		})
	}
	if len(req.RedirectURIs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "at least one redirect_uri is required",
		})
	}

	serviceReq := &services.CreateAppRequest{
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		IconURL:      req.IconURL,
		HomepageURL:  req.HomepageURL,
		PrivacyURL:   req.PrivacyURL,
		TermsURL:     req.TermsURL,
		IsPublic:     req.IsPublic,
	}

	result, err := h.oauthService.CreateApp(c.Context(), userID, serviceReq)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "creation_failed",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"app":           result.App.ToResponse(),
		"client_secret": result.ClientSecret,
		"message":       "Store the client_secret securely. It will not be shown again.",
	})
}

// GetApps returns all OAuth apps owned by the user
// GET /oauth/apps
func (h *OAuthProviderHandler) GetApps(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	apps, err := h.oauthService.GetAppsByOwner(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "failed to retrieve apps",
		})
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

	return c.JSON(fiber.Map{
		"apps": responses,
	})
}

// GetApp returns a specific OAuth app
// GET /oauth/apps/:id
func (h *OAuthProviderHandler) GetApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "invalid app ID",
		})
	}

	app, err := h.oauthService.GetApp(c.Context(), appID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "app not found",
		})
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

// UpdateApp updates an OAuth app
// PATCH /oauth/apps/:id
func (h *OAuthProviderHandler) UpdateApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "invalid app ID",
		})
	}

	var req CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	serviceReq := &services.CreateAppRequest{
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		IconURL:      req.IconURL,
		HomepageURL:  req.HomepageURL,
		PrivacyURL:   req.PrivacyURL,
		TermsURL:     req.TermsURL,
		IsPublic:     req.IsPublic,
	}

	app, err := h.oauthService.UpdateApp(c.Context(), appID, userID, serviceReq)
	if err != nil {
		if err == services.ErrOAuthAppNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "app not found",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(app.ToResponse())
}

// DeleteApp deletes an OAuth app
// DELETE /oauth/apps/:id
func (h *OAuthProviderHandler) DeleteApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "invalid app ID",
		})
	}

	if err := h.oauthService.DeleteApp(c.Context(), appID, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "app not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RegenerateSecret generates a new client secret
// POST /oauth/apps/:id/secret
func (h *OAuthProviderHandler) RegenerateSecret(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "invalid app ID",
		})
	}

	secret, err := h.oauthService.RegenerateClientSecret(c.Context(), appID, userID)
	if err != nil {
		if err == services.ErrOAuthAppNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "app not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "regeneration_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"client_secret": secret,
		"message":       "Store the new client_secret securely. It will not be shown again.",
	})
}

// ======== OAuth Authorization Endpoints ========

// Authorize handles the authorization endpoint
// GET /oauth/authorize
func (h *OAuthProviderHandler) Authorize(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	req := &services.AuthorizeRequest{
		ClientID:     c.Query("client_id"),
		RedirectURI:  c.Query("redirect_uri"),
		ResponseType: c.Query("response_type"),
		Scope:        c.Query("scope"),
	}

	// Optional parameters
	if state := c.Query("state"); state != "" {
		req.State = &state
	}
	if codeChallenge := c.Query("code_challenge"); codeChallenge != "" {
		req.CodeChallenge = &codeChallenge
	}
	if codeChallengeMethod := c.Query("code_challenge_method"); codeChallengeMethod != "" {
		req.CodeChallengeMethod = &codeChallengeMethod
	}
	if nonce := c.Query("nonce"); nonce != "" {
		req.Nonce = &nonce
	}

	// Validate required parameters
	if req.ClientID == "" {
		return oauthError(c, "", "", "invalid_request", "client_id is required")
	}
	if req.RedirectURI == "" {
		return oauthError(c, "", "", "invalid_request", "redirect_uri is required")
	}

	// Validate authorization request
	authResp, err := h.oauthService.ValidateAuthorization(c.Context(), userID, req)
	if err != nil {
		return oauthErrorRedirect(c, req.RedirectURI, req.State, mapOAuthError(err))
	}

	// If consent required, return consent info
	if authResp.RequiresConsent {
		return c.JSON(fiber.Map{
			"requires_consent": true,
			"app": fiber.Map{
				"id":           authResp.App.ID,
				"name":         authResp.App.Name,
				"description":  authResp.App.Description,
				"icon_url":     authResp.App.IconURL,
				"homepage_url": authResp.App.HomepageURL,
				"privacy_url":  authResp.App.PrivacyURL,
				"terms_url":    authResp.App.TermsURL,
				"is_verified":  authResp.App.IsVerified,
			},
			"scopes":   authResp.ScopeDetails,
			"redirect": req.RedirectURI,
			"state":    req.State,
		})
	}

	// User has already consented with same scopes, auto-approve
	code, err := h.oauthService.ApproveAuthorization(c.Context(), userID, req)
	if err != nil {
		return oauthErrorRedirect(c, req.RedirectURI, req.State, mapOAuthError(err))
	}

	return oauthRedirectWithCode(c, req.RedirectURI, code, req.State)
}

// AuthorizeConsent handles user consent approval
// POST /oauth/authorize
func (h *OAuthProviderHandler) AuthorizeConsent(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var body struct {
		ClientID            string  `json:"client_id"`
		RedirectURI         string  `json:"redirect_uri"`
		ResponseType        string  `json:"response_type"`
		Scope               string  `json:"scope"`
		State               *string `json:"state,omitempty"`
		CodeChallenge       *string `json:"code_challenge,omitempty"`
		CodeChallengeMethod *string `json:"code_challenge_method,omitempty"`
		Nonce               *string `json:"nonce,omitempty"`
		Approved            bool    `json:"approved"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	if !body.Approved {
		return oauthErrorRedirect(c, body.RedirectURI, body.State, oauthErr{
			Error:       "access_denied",
			Description: "user denied the authorization request",
		})
	}

	req := &services.AuthorizeRequest{
		ClientID:            body.ClientID,
		RedirectURI:         body.RedirectURI,
		ResponseType:        body.ResponseType,
		Scope:               body.Scope,
		State:               body.State,
		CodeChallenge:       body.CodeChallenge,
		CodeChallengeMethod: body.CodeChallengeMethod,
		Nonce:               body.Nonce,
	}

	code, err := h.oauthService.ApproveAuthorization(c.Context(), userID, req)
	if err != nil {
		return oauthErrorRedirect(c, body.RedirectURI, body.State, mapOAuthError(err))
	}

	// Return redirect URL for frontend to handle
	return c.JSON(fiber.Map{
		"redirect_uri": buildRedirectURI(body.RedirectURI, code, body.State),
	})
}

// Token handles the token endpoint
// POST /oauth/token
func (h *OAuthProviderHandler) Token(c *fiber.Ctx) error {
	// Parse request - support both form and JSON
	var req services.TokenRequest

	contentType := c.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		req.GrantType = c.FormValue("grant_type")
		req.ClientID = c.FormValue("client_id")
		if code := c.FormValue("code"); code != "" {
			req.Code = &code
		}
		if redirectURI := c.FormValue("redirect_uri"); redirectURI != "" {
			req.RedirectURI = &redirectURI
		}
		if clientSecret := c.FormValue("client_secret"); clientSecret != "" {
			req.ClientSecret = &clientSecret
		}
		if codeVerifier := c.FormValue("code_verifier"); codeVerifier != "" {
			req.CodeVerifier = &codeVerifier
		}
		if refreshToken := c.FormValue("refresh_token"); refreshToken != "" {
			req.RefreshToken = &refreshToken
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return tokenError(c, "invalid_request", "invalid request body")
		}
	}

	// Support client credentials in Authorization header
	if authHeader := c.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		credentials, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err == nil {
			parts := strings.SplitN(string(credentials), ":", 2)
			if len(parts) == 2 {
				req.ClientID = parts[0]
				req.ClientSecret = &parts[1]
			}
		}
	}

	// Validate
	if req.ClientID == "" {
		return tokenError(c, "invalid_request", "client_id is required")
	}
	if req.GrantType == "" {
		return tokenError(c, "invalid_request", "grant_type is required")
	}

	// Exchange token
	tokenResp, err := h.oauthService.ExchangeToken(c.Context(), &req)
	if err != nil {
		return tokenError(c, mapTokenError(err), err.Error())
	}

	return c.JSON(tokenResp)
}

// Revoke handles the token revocation endpoint
// POST /oauth/revoke
func (h *OAuthProviderHandler) Revoke(c *fiber.Ctx) error {
	var token, tokenTypeHint string

	contentType := c.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		token = c.FormValue("token")
		tokenTypeHint = c.FormValue("token_type_hint")
	} else {
		var body struct {
			Token         string `json:"token"`
			TokenTypeHint string `json:"token_type_hint,omitempty"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":             "invalid_request",
				"error_description": "invalid request body",
			})
		}
		token = body.Token
		tokenTypeHint = body.TokenTypeHint
	}

	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": "token is required",
		})
	}

	// Revoke token (never returns error per RFC 7009)
	_ = h.oauthService.RevokeToken(c.Context(), token, tokenTypeHint)

	return c.SendStatus(fiber.StatusOK)
}

// Introspect handles the token introspection endpoint
// POST /oauth/introspect
func (h *OAuthProviderHandler) Introspect(c *fiber.Ctx) error {
	var token string

	contentType := c.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		token = c.FormValue("token")
	} else {
		var body struct {
			Token string `json:"token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":             "invalid_request",
				"error_description": "invalid request body",
			})
		}
		token = body.Token
	}

	if token == "" {
		return c.JSON(fiber.Map{"active": false})
	}

	validatedToken, err := h.oauthService.ValidateAccessToken(c.Context(), token)
	if err != nil {
		return c.JSON(fiber.Map{"active": false})
	}

	return c.JSON(fiber.Map{
		"active":    true,
		"scope":     strings.Join(validatedToken.Scopes, " "),
		"client_id": validatedToken.ClientID,
		"sub":       validatedToken.UserID.String(),
	})
}

// ======== User Authorization Management ========

// GetAuthorizations returns all apps the user has authorized
// GET /oauth/authorizations
func (h *OAuthProviderHandler) GetAuthorizations(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	auths, err := h.oauthService.GetUserAuthorizations(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "failed to retrieve authorizations",
		})
	}

	return c.JSON(fiber.Map{
		"authorizations": auths,
	})
}

// RevokeAuthorization revokes a user's authorization for an app
// DELETE /oauth/authorizations/:client_id
func (h *OAuthProviderHandler) RevokeAuthorization(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	clientID := c.Params("client_id")

	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "client_id is required",
		})
	}

	if err := h.oauthService.RevokeUserAuthorization(c.Context(), userID, clientID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "authorization not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ======== Helper Functions ========

type oauthErr struct {
	Error       string
	Description string
}

func mapOAuthError(err error) oauthErr {
	switch err {
	case services.ErrOAuthAppNotFound:
		return oauthErr{"invalid_client", "unknown client"}
	case services.ErrOAuthAppInactive:
		return oauthErr{"invalid_client", "client is inactive"}
	case services.ErrOAuthInvalidRedirectURI:
		return oauthErr{"invalid_request", "invalid redirect_uri"}
	case services.ErrOAuthInvalidScope:
		return oauthErr{"invalid_scope", "requested scope is invalid"}
	case services.ErrOAuthPKCERequired:
		return oauthErr{"invalid_request", "PKCE is required for this client"}
	case services.ErrOAuthInvalidCodeChallenge:
		return oauthErr{"invalid_request", "invalid code_challenge_method"}
	default:
		return oauthErr{"server_error", "an unexpected error occurred"}
	}
}

func mapTokenError(err error) string {
	switch err {
	case services.ErrOAuthAppNotFound, services.ErrOAuthInvalidClientSecret:
		return "invalid_client"
	case services.ErrOAuthCodeNotFound, services.ErrOAuthCodeExpired, services.ErrOAuthCodeAlreadyUsed:
		return "invalid_grant"
	case services.ErrOAuthInvalidCodeVerifier:
		return "invalid_grant"
	case services.ErrOAuthInvalidRedirectURI:
		return "invalid_grant"
	case services.ErrOAuthRefreshTokenExpired, services.ErrOAuthRefreshTokenRevoked, services.ErrOAuthRefreshTokenReused:
		return "invalid_grant"
	case services.ErrOAuthClientAuthRequired:
		return "invalid_client"
	case services.ErrOAuthAppInactive:
		return "invalid_client"
	default:
		return "server_error"
	}
}

func oauthError(c *fiber.Ctx, redirectURI, state, errCode, errDesc string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":             errCode,
		"error_description": errDesc,
	})
}

func oauthErrorRedirect(c *fiber.Ctx, redirectURI string, state *string, err oauthErr) error {
	// Build redirect URL with error
	u, parseErr := url.Parse(redirectURI)
	if parseErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             err.Error,
			"error_description": err.Description,
		})
	}

	q := u.Query()
	q.Set("error", err.Error)
	q.Set("error_description", err.Description)
	if state != nil {
		q.Set("state", *state)
	}
	u.RawQuery = q.Encode()

	return c.JSON(fiber.Map{
		"redirect_uri":      u.String(),
		"error":             err.Error,
		"error_description": err.Description,
	})
}

func oauthRedirectWithCode(c *fiber.Ctx, redirectURI, code string, state *string) error {
	return c.JSON(fiber.Map{
		"redirect_uri": buildRedirectURI(redirectURI, code, state),
	})
}

func buildRedirectURI(redirectURI, code string, state *string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}

	q := u.Query()
	q.Set("code", code)
	if state != nil {
		q.Set("state", *state)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func tokenError(c *fiber.Ctx, errCode, errDesc string) error {
	status := fiber.StatusBadRequest
	if errCode == "invalid_client" {
		status = fiber.StatusUnauthorized
	}
	return c.Status(status).JSON(fiber.Map{
		"error":             errCode,
		"error_description": errDesc,
	})
}

// RegisterOAuthProviderRoutes registers OAuth provider routes
func RegisterOAuthProviderRoutes(router fiber.Router, handler *OAuthProviderHandler, authMiddleware fiber.Handler) {
	// Public endpoints (no auth required for OAuth flow)
	router.Post("/token", handler.Token)
	router.Post("/revoke", handler.Revoke)
	router.Post("/introspect", handler.Introspect)

	// Protected endpoints (require user auth)
	protected := router.Group("", authMiddleware)

	// Authorization endpoints
	protected.Get("/authorize", handler.Authorize)
	protected.Post("/authorize", handler.AuthorizeConsent)

	// App management endpoints
	protected.Post("/apps", handler.CreateApp)
	protected.Get("/apps", handler.GetApps)
	protected.Get("/apps/:id", handler.GetApp)
	protected.Patch("/apps/:id", handler.UpdateApp)
	protected.Delete("/apps/:id", handler.DeleteApp)
	protected.Post("/apps/:id/secret", handler.RegenerateSecret)

	// User authorization management
	protected.Get("/authorizations", handler.GetAuthorizations)
	protected.Delete("/authorizations/:client_id", handler.RevokeAuthorization)
}
