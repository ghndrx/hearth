package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// ProviderType identifies the authentication backend
type ProviderType string

const (
	ProviderNative     ProviderType = "native"
	ProviderFusionAuth ProviderType = "fusionauth"
	ProviderAuthentik  ProviderType = "authentik"
	ProviderKeycloak   ProviderType = "keycloak"
	ProviderOIDC       ProviderType = "oidc"
)

// Provider defines the interface for authentication backends
type Provider interface {
	// Metadata
	Name() string
	Type() ProviderType

	// Core authentication
	Register(ctx context.Context, req *RegisterRequest) (*AuthResult, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResult, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)

	// Password management (native provider, others may delegate)
	ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error
	RequestPasswordReset(ctx context.Context, email string) error
	ConfirmPasswordReset(ctx context.Context, token, newPassword string) error

	// MFA
	EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetup, error)
	VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error
	DisableMFA(ctx context.Context, userID uuid.UUID) error

	// Sessions
	GetSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID uuid.UUID) error

	// User management
	GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error

	// OAuth2/OIDC (for external providers)
	GetAuthorizationURL(ctx context.Context, state string) (string, error)
	HandleCallback(ctx context.Context, code, state string) (*AuthResult, error)
}

// RegisterRequest is input for user registration
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=2,max=32"`
	Password string `json:"password" validate:"required,min=8,max=128"`

	// Optional OAuth token for account linking
	OAuthToken string `json:"oauth_token,omitempty"`

	// Client info for session
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

// LoginRequest is input for authentication
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFACode  string `json:"mfa_code,omitempty"`

	// For OAuth flows
	OAuthCode string `json:"oauth_code,omitempty"`

	// Client info for session
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

// ChangePasswordRequest is input for password changes
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// AuthResult is the response after successful authentication
type AuthResult struct {
	User         *models.PublicUser `json:"user"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	ExpiresIn    int                `json:"expires_in"` // seconds
	TokenType    string             `json:"token_type"` // "Bearer"

	// For MFA challenge
	MFARequired bool   `json:"mfa_required,omitempty"`
	MFAToken    string `json:"mfa_token,omitempty"`
}

// MFASetup is returned when enabling MFA
type MFASetup struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// ProviderClaims for external provider JWT tokens (distinct from internal Claims in jwt.go)
type ProviderClaims struct {
	UserID    uuid.UUID    `json:"uid"`
	Username  string       `json:"usr"`
	SessionID uuid.UUID    `json:"sid"`
	Provider  ProviderType `json:"prv"`

	// External provider ID (if applicable)
	ExternalID string `json:"ext,omitempty"`

	// User flags for quick permission checks
	Flags int64 `json:"flg,omitempty"`

	// Standard JWT claims
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
	Issuer    string    `json:"iss"`
	Audience  []string  `json:"aud"`
}

// Config holds authentication configuration
type Config struct {
	// Provider selection
	Provider ProviderType `yaml:"provider"`

	// Native provider config
	Native *NativeConfig `yaml:"native,omitempty"`

	// FusionAuth config
	FusionAuth *FusionAuthConfig `yaml:"fusionauth,omitempty"`

	// Generic OIDC config
	OIDC *OIDCConfig `yaml:"oidc,omitempty"`

	// JWT settings (used by all providers)
	JWT JWTConfig `yaml:"jwt"`

	// Hybrid mode: enable multiple providers
	Providers []ProviderConfig `yaml:"providers,omitempty"`

	// Account linking
	AllowAccountLinking bool `yaml:"allow_account_linking"`
	RequireEmailMatch   bool `yaml:"require_email_match"`
}

// NativeConfig for built-in authentication
type NativeConfig struct {
	// Password requirements
	PasswordMinLength        int  `yaml:"password_min_length"`
	PasswordRequireUppercase bool `yaml:"password_require_uppercase"`
	PasswordRequireNumber    bool `yaml:"password_require_number"`
	PasswordRequireSpecial   bool `yaml:"password_require_special"`

	// Security
	MaxLoginAttempts int           `yaml:"max_login_attempts"`
	LockoutDuration  time.Duration `yaml:"lockout_duration"`

	// Email
	RequireEmailVerification bool `yaml:"require_email_verification"`

	// MFA
	MFAIssuer string `yaml:"mfa_issuer"`
}

// FusionAuthConfig for FusionAuth integration
type FusionAuthConfig struct {
	// Server
	Host string `yaml:"host"`

	// Application credentials
	ApplicationID string `yaml:"application_id"`
	ClientID      string `yaml:"client_id"`
	ClientSecret  string `yaml:"client_secret"`

	// API key for admin operations
	APIKey string `yaml:"api_key"`

	// Tenant (multi-tenant mode)
	TenantID string `yaml:"tenant_id,omitempty"`

	// User sync
	SyncProfileFields bool          `yaml:"sync_profile_fields"`
	SyncInterval      time.Duration `yaml:"sync_interval"`

	// OAuth2 settings
	Scopes      []string `yaml:"scopes"`
	RedirectURI string   `yaml:"redirect_uri"`

	// Post-logout redirect
	PostLogoutURI string `yaml:"post_logout_uri"`
}

// OIDCConfig for generic OIDC providers
type OIDCConfig struct {
	// Discovery (auto-configures from .well-known)
	Issuer string `yaml:"issuer,omitempty"`

	// Manual endpoints (if not using discovery)
	AuthorizationEndpoint string `yaml:"authorization_endpoint,omitempty"`
	TokenEndpoint         string `yaml:"token_endpoint,omitempty"`
	UserinfoEndpoint      string `yaml:"userinfo_endpoint,omitempty"`
	JWKSURI               string `yaml:"jwks_uri,omitempty"`

	// Client credentials
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`

	// Scopes
	Scopes []string `yaml:"scopes"`

	// Claim mapping
	Claims ClaimMapping `yaml:"claims"`

	// Redirect URI
	RedirectURI string `yaml:"redirect_uri"`
}

// ClaimMapping maps OIDC claims to Hearth user fields
type ClaimMapping struct {
	UserID      string `yaml:"user_id"`      // default: sub
	Email       string `yaml:"email"`        // default: email
	Username    string `yaml:"username"`     // default: preferred_username
	Avatar      string `yaml:"avatar"`       // default: picture
	DisplayName string `yaml:"display_name"` // default: name
}

// JWTConfig for token generation
type JWTConfig struct {
	// Secret key for signing (HS256) or path to private key (RS256)
	Secret     string `yaml:"secret"`
	PrivateKey string `yaml:"private_key,omitempty"`

	// Token lifetimes
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`

	// Issuer and audience
	Issuer   string   `yaml:"issuer"`
	Audience []string `yaml:"audience"`
}

// ProviderConfig for hybrid mode
type ProviderConfig struct {
	Name    string       `yaml:"name"`
	Type    ProviderType `yaml:"type"`
	Enabled bool         `yaml:"enabled"`

	// Type-specific config
	Native     *NativeConfig     `yaml:"native,omitempty"`
	FusionAuth *FusionAuthConfig `yaml:"fusionauth,omitempty"`
	OIDC       *OIDCConfig       `yaml:"oidc,omitempty"`
}

// DefaultNativeConfig returns sensible defaults
func DefaultNativeConfig() *NativeConfig {
	return &NativeConfig{
		PasswordMinLength:        8,
		PasswordRequireUppercase: false,
		PasswordRequireNumber:    false,
		PasswordRequireSpecial:   false,
		MaxLoginAttempts:         5,
		LockoutDuration:          15 * time.Minute,
		RequireEmailVerification: false,
		MFAIssuer:                "Hearth",
	}
}

// DefaultJWTConfig returns sensible defaults
func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "hearth",
		Audience:        []string{"hearth-api"},
	}
}
