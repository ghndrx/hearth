package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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
}

// RefreshRequest represents token refresh payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenResponse represents auth tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// AuthResponse includes user and tokens
type AuthResponse struct {
	User   *UserResponse  `json:"user"`
	Tokens *TokenResponse `json:"tokens"`
}

// Register handles user registration
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
	user, tokens, err := h.authService.Register(c.Context(), &services.RegisterRequest{
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

	user, tokens, err := h.authService.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return handleAuthError(c, err)
	}

	return c.JSON(AuthResponse{
		User:   toUserResponse(user),
		Tokens: toTokenResponse(tokens),
	})
}

// Refresh handles token refresh
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

	tokens, err := h.authService.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "invalid_token",
			"message": "invalid or expired refresh token",
		})
	}

	return c.JSON(toTokenResponse(tokens))
}

// Logout handles logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Get tokens from request
	accessToken := extractBearerToken(c)

	var req RefreshRequest
	_ = c.BodyParser(&req) // Optional body with refresh token

	if accessToken != "" || req.RefreshToken != "" {
		_ = h.authService.Logout(c.Context(), accessToken, req.RefreshToken)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// OAuthRedirect redirects to OAuth provider
func (h *AuthHandler) OAuthRedirect(c *fiber.Ctx) error {
	provider := c.Params("provider")

	// Validate provider
	validProviders := map[string]bool{"google": true, "github": true, "discord": true}
	if !validProviders[provider] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_provider",
			"message": "unsupported OAuth provider",
		})
	}

	// TODO: Generate state, build OAuth URL based on provider
	// For now, return not implemented
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "OAuth login is not yet implemented",
	})
}

// OAuthCallback handles OAuth callback
func (h *AuthHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_code",
			"message": "authorization code is required",
		})
	}

	// TODO: Validate state
	// TODO: Exchange code for tokens
	// TODO: Get user info from provider
	// TODO: Create or link user
	// TODO: Generate tokens

	_, _, _ = provider, code, state

	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "OAuth login is not yet implemented",
	})
}

// Helper functions

func extractBearerToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

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
		Discriminator: user.Discriminator,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	}
}

func toTokenResponse(tokens *services.TokenResponse) *TokenResponse {
	if tokens == nil {
		return nil
	}
	return &TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	Email         *string   `json:"email,omitempty"` // Only set for self-user responses
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	BannerURL     *string   `json:"banner_url,omitempty"`
	Bio           *string   `json:"bio,omitempty"`
	CustomStatus  *string   `json:"custom_status,omitempty"`
	Flags         int64     `json:"flags"`
	CreatedAt     time.Time `json:"created_at"`
}
