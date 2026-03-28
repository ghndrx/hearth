package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type AuthHandler struct {
	authService  services.AuthService
	oauthService *services.OAuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SetOAuthService sets the OAuth service for the handler
func (h *AuthHandler) SetOAuthService(oauthService *services.OAuthService) {
	h.oauthService = oauthService
}

// RegisterRequest represents registration payload
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Username    string `json:"username" validate:"required,min=2,max=32"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password" validate:"required,min=8"`
	InviteCode  string `json:"invite_code"`
}

// LoginRequest represents login payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// RefreshRequest represents token refresh payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LoginWithMFARequest represents login with MFA payload
type LoginWithMFARequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFACode  string `json:"mfa_code" validate:"required"`
}

// MFASetupResponse represents the response when setting up MFA
type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// VerifyMFARequest represents MFA verification payload
type VerifyMFARequest struct {
	Code string `json:"code" validate:"required"`
}

// DisableMFARequest represents MFA disable payload
type DisableMFARequest struct {
	Password string `json:"password" validate:"required"`
}

// TokenResponse represents auth tokens - matches frontend expectations
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

// Register handles user registration
// @Summary Register a new user
// @Description Creates a new user account with email, username, and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Registration details"
// @Success 201 {object} TokenResponse "User created successfully"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 403 {object} fiber.Map "Registration closed or invite required"
// @Failure 409 {object} fiber.Map "Email or username already taken"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
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

	// Validate email format (basic check)
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
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

	// Call auth service
	_, tokens, err := h.authService.Register(c.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		return handleAuthError(c, err)
	}

	// Return tokens in the format frontend expects
	return c.Status(fiber.StatusCreated).JSON(TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Login handles user login
// @Summary Authenticate a user
// @Description Authenticates a user with email and password, returns access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} TokenResponse "Login successful"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Invalid credentials"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
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

	_, tokens, err := h.authService.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return handleAuthError(c, err)
	}

	// Return tokens in the format frontend expects
	return c.JSON(TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Refresh handles token refresh
// @Summary Refresh access token
// @Description Refreshes the access token using a valid refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "Refresh token"
// @Success 200 {object} TokenResponse "New tokens issued"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
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

	tokens, err := h.authService.RefreshTokens(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "invalid_refresh_token",
			"message": "Invalid or expired refresh token",
		})
	}

	return c.JSON(TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// Logout handles logout
// @Summary Logout user
// @Description Logs out the current user (client-side token removal)
// @Tags Auth
// @Success 204 "Logout successful"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// For stateless JWT, logout is handled client-side by removing tokens
	// In a production app, we could add refresh tokens to a revocation list
	return c.SendStatus(fiber.StatusNoContent)
}

// OAuthRedirect redirects to OAuth provider
// @Summary Redirect to OAuth provider
// @Description Generates and returns OAuth authorization URL for the specified provider
// @Tags Auth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider (google, github, discord)"
// @Success 200 {object} fiber.Map "Authorization URL"
// @Failure 400 {object} fiber.Map "Invalid provider"
// @Failure 501 {object} fiber.Map "OAuth not configured"
// @Router /auth/oauth/{provider} [get]
func (h *AuthHandler) OAuthRedirect(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Validate provider - case sensitive, must be lowercase
	validProviders := map[string]services.OAuthProvider{
		"google":  services.OAuthProviderGoogle,
		"github":  services.OAuthProviderGitHub,
		"discord": services.OAuthProviderDiscord,
	}

	oauthProvider, ok := validProviders[provider]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_provider",
			"message": "unsupported OAuth provider",
		})
	}

	// Check if OAuth service is configured
	if h.oauthService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth authentication is not configured",
		})
	}

	// Check if provider is enabled
	if !h.oauthService.IsProviderEnabled(oauthProvider) {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth provider is not configured",
		})
	}

	// Generate authorization URL
	authURL, err := h.oauthService.GetAuthorizationURL(c.Context(), oauthProvider, nil)
	if err != nil {
		return handleOAuthError(c, err)
	}

	// Return authorization URL for redirect
	return c.JSON(fiber.Map{
		"authorization_url": authURL,
	})
}

// OAuthCallback handles OAuth callback
// @Summary Handle OAuth callback
// @Description Processes OAuth callback, exchanges code for tokens, and authenticates user
// @Tags Auth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider"
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} fiber.Map "Access token and user info"
// @Failure 400 {object} fiber.Map "Invalid parameters or OAuth error"
// @Failure 501 {object} fiber.Map "OAuth not configured"
// @Router /auth/oauth/{provider}/callback [get]
func (h *AuthHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Check for OAuth error response from provider
	if oauthError := c.Query("error"); oauthError != "" {
		errorDescription := c.Query("error_description")
		if errorDescription == "" {
			errorDescription = "authorization was denied"
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "oauth_" + oauthError,
			"message": errorDescription,
		})
	}

	code := c.Query("code")
	state := c.Query("state")

	// Validate required parameters
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_code",
			"message": "authorization code is required",
		})
	}

	if state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_state",
			"message": "state parameter is required",
		})
	}

	// Validate provider - case sensitive
	validProviders := map[string]services.OAuthProvider{
		"google":  services.OAuthProviderGoogle,
		"github":  services.OAuthProviderGitHub,
		"discord": services.OAuthProviderDiscord,
	}

	oauthProvider, ok := validProviders[provider]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_provider",
			"message": "unsupported OAuth provider",
		})
	}

	// Check if OAuth service is configured
	if h.oauthService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth authentication is not configured",
		})
	}

	// Handle the callback - exchange code for tokens and get/create user
	user, tokens, err := h.oauthService.HandleCallback(c.Context(), oauthProvider, code, state)
	if err != nil {
		return handleOAuthError(c, err)
	}

	// Return tokens and user info
	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
		"token_type":    "Bearer",
		"user":          toUserResponse(user),
	})
}

// GetEnabledProviders returns a list of enabled OAuth providers
// @Summary Get enabled OAuth providers
// @Description Returns a list of configured and enabled OAuth providers
// @Tags Auth
// @Produce json
// @Success 200 {object} fiber.Map "List of enabled providers"
// @Router /auth/oauth/providers [get]
func (h *AuthHandler) GetEnabledProviders(c *fiber.Ctx) error {
	// If OAuth service is not configured, return empty list
	if h.oauthService == nil {
		return c.JSON(fiber.Map{
			"providers": []string{},
		})
	}

	enabledProviders := h.oauthService.GetEnabledProviders()
	providers := make([]string, len(enabledProviders))
	for i, p := range enabledProviders {
		providers[i] = string(p)
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// OAuthLinkRedirect generates an OAuth URL for linking a provider to an existing account
// @Summary Generate OAuth link URL
// @Description Generates an OAuth authorization URL to link a provider to the current user's account
// @Tags Auth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider (google, github, discord)"
// @Success 200 {object} fiber.Map "Authorization URL"
// @Failure 400 {object} fiber.Map "Invalid provider"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 501 {object} fiber.Map "OAuth not configured"
// @Router /auth/oauth/{provider}/link [post]
func (h *AuthHandler) OAuthLinkRedirect(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Validate provider
	validProviders := map[string]services.OAuthProvider{
		"google":  services.OAuthProviderGoogle,
		"github":  services.OAuthProviderGitHub,
		"discord": services.OAuthProviderDiscord,
	}

	oauthProvider, ok := validProviders[provider]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_provider",
			"message": "unsupported OAuth provider",
		})
	}

	// Check if OAuth service is configured
	if h.oauthService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth authentication is not configured",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(uuid.UUID)

	// Generate link authorization URL
	authURL, err := h.oauthService.GetLinkAuthorizationURL(c.Context(), userID, oauthProvider)
	if err != nil {
		return handleOAuthError(c, err)
	}

	return c.JSON(fiber.Map{
		"authorization_url": authURL,
	})
}

// GetLinkedProviders returns all OAuth providers linked to the current user
// @Summary Get linked OAuth providers
// @Description Returns all OAuth providers linked to the current user's account
// @Tags Auth
// @Produce json
// @Success 200 {object} fiber.Map "List of linked providers"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 501 {object} fiber.Map "OAuth not configured"
// @Router /auth/oauth/linked [get]
func (h *AuthHandler) GetLinkedProviders(c *fiber.Ctx) error {
	// Check if OAuth service is configured
	if h.oauthService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth authentication is not configured",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(uuid.UUID)

	providers, err := h.oauthService.GetLinkedProviders(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "failed to retrieve linked providers",
		})
	}

	// Convert to response format
	responses := make([]models.OAuthProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	return c.JSON(fiber.Map{
		"providers": responses,
	})
}

// OAuthUnlink removes an OAuth provider link from the current user
// @Summary Unlink OAuth provider
// @Description Removes an OAuth provider link from the current user's account
// @Tags Auth
// @Param provider path string true "OAuth provider to unlink"
// @Success 204 "Provider unlinked successfully"
// @Failure 400 {object} fiber.Map "Invalid provider"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Provider not linked"
// @Failure 409 {object} fiber.Map "Cannot unlink last authentication method"
// @Failure 501 {object} fiber.Map "OAuth not configured"
// @Router /auth/oauth/{provider}/unlink [delete]
func (h *AuthHandler) OAuthUnlink(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Validate provider
	validProviders := map[string]services.OAuthProvider{
		"google":  services.OAuthProviderGoogle,
		"github":  services.OAuthProviderGitHub,
		"discord": services.OAuthProviderDiscord,
	}

	oauthProvider, ok := validProviders[provider]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_provider",
			"message": "unsupported OAuth provider",
		})
	}

	// Check if OAuth service is configured
	if h.oauthService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_configured",
			"message": "OAuth authentication is not configured",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(uuid.UUID)

	err := h.oauthService.UnlinkProvider(c.Context(), userID, oauthProvider)
	if err != nil {
		return handleOAuthError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// handleOAuthError converts OAuth service errors to appropriate HTTP responses
func handleOAuthError(c *fiber.Ctx, err error) error {
	// Map errors to appropriate HTTP responses
	switch {
	case errors.Is(err, services.ErrOAuthProviderNotSupported):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "provider_not_supported",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthStateMismatch):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "state_mismatch",
			"message": "OAuth state validation failed - possible CSRF attack",
		})

	case errors.Is(err, services.ErrOAuthStateExpired):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "state_expired",
			"message": "OAuth state has expired - please try again",
		})

	case errors.Is(err, services.ErrOAuthCodeExchange):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "token_exchange_failed",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthMalformedResponse):
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":   "provider_error",
			"message": "received malformed response from OAuth provider",
		})

	case errors.Is(err, services.ErrOAuthProviderUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "provider_unavailable",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthTokenRevoked):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "token_revoked",
			"message": "OAuth token has been revoked",
		})

	case errors.Is(err, services.ErrOAuthInsufficientScope):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "insufficient_scope",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthRateLimited):
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "rate_limited",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthUserInfo):
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":   "user_info_failed",
			"message": err.Error(),
		})

	case errors.Is(err, services.ErrOAuthEmailNotVerified):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "email_not_verified",
			"message": "email address must be verified with the OAuth provider",
		})

	case errors.Is(err, services.ErrOAuthAccountLinked):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "account_already_linked",
			"message": "this OAuth account is already linked to another user",
		})

	case errors.Is(err, services.ErrOAuthProviderNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "provider_not_linked",
			"message": "OAuth provider is not linked to this account",
		})

	case errors.Is(err, services.ErrOAuthProviderAlreadyLinked):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "provider_already_linked",
			"message": "this OAuth provider is already linked to your account",
		})

	case errors.Is(err, services.ErrOAuthCannotUnlinkLast):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "cannot_unlink_last",
			"message": "cannot unlink the last authentication method - add a password or link another provider first",
		})

	case errors.Is(err, services.ErrUserNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "user_not_found",
			"message": "user not found",
		})

	default:
		// Log the error for debugging
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "an unexpected error occurred during OAuth authentication",
		})
	}
}

// LoginWithMFA handles user login with MFA verification
// @Summary Login with MFA
// @Description Authenticates a user with email, password, and MFA code
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginWithMFARequest true "Login credentials with MFA code"
// @Success 200 {object} TokenResponse "Login successful"
// @Failure 400 {object} fiber.Map "Invalid credentials or MFA code"
// @Router /auth/login/mfa [post]
func (h *AuthHandler) LoginWithMFA(c *fiber.Ctx) error {
	var req LoginWithMFARequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" || req.MFACode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "email, password, and mfa_code are required",
		})
	}

	_, tokens, err := h.authService.LoginWithMFA(c.Context(), req.Email, req.Password, req.MFACode)
	if err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    "Bearer",
	})
}

// EnableMFA generates MFA setup data for a user
// @Summary Enable MFA
// @Description Generates TOTP secret and backup codes for MFA setup
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MFASetupResponse "MFA setup data generated"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 400 {object} fiber.Map "MFA already enabled"
// @Router /auth/mfa/enable [post]
func (h *AuthHandler) EnableMFA(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "invalid or missing authentication token",
		})
	}

	setup, err := h.authService.EnableMFA(c.Context(), userID)
	if err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(setup)
}

// VerifyMFASetup verifies the TOTP code and completes MFA setup
// @Summary Verify MFA Setup
// @Description Verifies TOTP code and enables MFA for the user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body VerifyMFARequest true "TOTP verification code"
// @Success 200 {object} fiber.Map "MFA enabled successfully"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 400 {object} fiber.Map "Invalid MFA code"
// @Router /auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFASetup(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "invalid or missing authentication token",
		})
	}

	var req VerifyMFARequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "code is required",
		})
	}

	if err := h.authService.VerifyMFASetup(c.Context(), userID, req.Code); err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "MFA enabled successfully",
	})
}

// DisableMFA disables MFA for a user
// @Summary Disable MFA
// @Description Disables MFA after password verification
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body DisableMFARequest true "Password verification"
// @Success 200 {object} fiber.Map "MFA disabled successfully"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 400 {object} fiber.Map "Invalid password"
// @Router /auth/mfa/disable [post]
func (h *AuthHandler) DisableMFA(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "invalid or missing authentication token",
		})
	}

	var req DisableMFARequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
	}

	if req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "password is required",
		})
	}

	if err := h.authService.DisableMFA(c.Context(), userID, req.Password); err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "MFA disabled successfully",
	})
}

// Helper functions

func handleAuthError(c *fiber.Ctx, err error) error {
	switch err {
	case services.ErrRegistrationClosed:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "registration_closed",
			"message": "registration is currently closed",
		})
	case services.ErrInviteRequired:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "invite_required",
			"message": "an invite code is required to register",
		})
	case services.ErrEmailTaken:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "email_taken",
			"message": "email is already registered",
		})
	case services.ErrUsernameTaken:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "username_taken",
			"message": "username is already taken",
		})
	case services.ErrInvalidCredentials:
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "invalid_credentials",
			"message": "invalid email or password",
		})
	case services.ErrPasswordTooShort:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "password_too_short",
			"message": "password must be at least 8 characters",
		})
	case services.ErrPasswordTooLong:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "password_too_long",
			"message": "password must be at most 72 characters",
		})
	case services.ErrPasswordWeak:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "password_weak",
			"message": "password must contain at least one uppercase, lowercase, and number",
		})
	case services.ErrMFARequired:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "mfa_required",
			"message": "MFA code is required",
		})
	case services.ErrInvalidMFACode:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_mfa_code",
			"message": "invalid MFA code",
		})
	case services.ErrMFAAlreadyEnabled:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "mfa_already_enabled",
			"message": "MFA is already enabled",
		})
	case services.ErrMFANotEnabled:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "mfa_not_enabled",
			"message": "MFA is not enabled",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "an unexpected error occurred",
		})
	}
}

func toUserResponse(user *models.User) *UserResponse {
	if user == nil {
		return nil
	}
	return &UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	}
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   *string   `json:"display_name,omitempty"`
	Discriminator string    `json:"discriminator"`
	Email         *string   `json:"email,omitempty"` // Only set for self-user responses
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	BannerURL     *string   `json:"banner_url,omitempty"`
	Bio           *string   `json:"bio,omitempty"`
	AboutMe       *string   `json:"about_me,omitempty"`
	Pronouns      *string   `json:"pronouns,omitempty"`
	AccentColor   *int      `json:"accent_color,omitempty"`
	CustomStatus  *string   `json:"custom_status,omitempty"`
	Flags         int64     `json:"flags"`
	CreatedAt     time.Time `json:"created_at"`
}
