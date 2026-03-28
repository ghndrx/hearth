package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthScope represents a permission scope for OAuth tokens
type OAuthScope string

const (
	// OAuthScopeRead allows reading user data (profile, servers, channels)
	OAuthScopeRead OAuthScope = "read"
	// OAuthScopeWrite allows modifying user data (send messages, update profile)
	OAuthScopeWrite OAuthScope = "write"
	// OAuthScopeAdmin allows administrative actions (manage servers, roles)
	OAuthScopeAdmin OAuthScope = "admin"
	// OAuthScopeOpenID provides OpenID Connect identity
	OAuthScopeOpenID OAuthScope = "openid"
	// OAuthScopeProfile provides user profile information
	OAuthScopeProfile OAuthScope = "profile"
	// OAuthScopeEmail provides user email
	OAuthScopeEmail OAuthScope = "email"
	// OAuthScopeServers allows access to user's servers
	OAuthScopeServers OAuthScope = "servers"
	// OAuthScopeMessages allows sending/reading messages
	OAuthScopeMessages OAuthScope = "messages"
)

// ValidScopes contains all valid OAuth scopes
var ValidScopes = map[OAuthScope]bool{
	OAuthScopeRead:     true,
	OAuthScopeWrite:    true,
	OAuthScopeAdmin:    true,
	OAuthScopeOpenID:   true,
	OAuthScopeProfile:  true,
	OAuthScopeEmail:    true,
	OAuthScopeServers:  true,
	OAuthScopeMessages: true,
}

// ScopeDescriptions provides human-readable descriptions for scopes
var ScopeDescriptions = map[OAuthScope]string{
	OAuthScopeRead:     "Read your basic profile information",
	OAuthScopeWrite:    "Modify your profile and settings",
	OAuthScopeAdmin:    "Perform administrative actions on your behalf",
	OAuthScopeOpenID:   "Verify your identity",
	OAuthScopeProfile:  "Access your profile information",
	OAuthScopeEmail:    "Access your email address",
	OAuthScopeServers:  "Access your servers and memberships",
	OAuthScopeMessages: "Send and read messages on your behalf",
}

// OAuthApp represents a registered third-party OAuth application
type OAuthApp struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OwnerID      uuid.UUID `json:"owner_id" db:"owner_id"`
	Name         string    `json:"name" db:"name"`
	Description  *string   `json:"description,omitempty" db:"description"`
	ClientID     string    `json:"client_id" db:"client_id"`
	ClientSecret string    `json:"-" db:"client_secret_hash"` // bcrypt hash, never exposed
	RedirectURIs []string  `json:"redirect_uris" db:"redirect_uris"`
	Scopes       []string  `json:"scopes" db:"scopes"`
	IconURL      *string   `json:"icon_url,omitempty" db:"icon_url"`
	HomepageURL  *string   `json:"homepage_url,omitempty" db:"homepage_url"`
	PrivacyURL   *string   `json:"privacy_url,omitempty" db:"privacy_url"`
	TermsURL     *string   `json:"terms_url,omitempty" db:"terms_url"`
	IsPublic     bool      `json:"is_public" db:"is_public"`     // Public clients (mobile/SPA) use PKCE only
	IsVerified   bool      `json:"is_verified" db:"is_verified"` // Verified by platform admins
	IsActive     bool      `json:"is_active" db:"is_active"`     // Can be disabled by owner or admin
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// OAuthAppResponse is the safe API response format
type OAuthAppResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	ClientID     string    `json:"client_id"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	IconURL      *string   `json:"icon_url,omitempty"`
	HomepageURL  *string   `json:"homepage_url,omitempty"`
	IsPublic     bool      `json:"is_public"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToResponse converts OAuthApp to safe API response
func (a *OAuthApp) ToResponse() OAuthAppResponse {
	return OAuthAppResponse{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		ClientID:     a.ClientID,
		RedirectURIs: a.RedirectURIs,
		Scopes:       a.Scopes,
		IconURL:      a.IconURL,
		HomepageURL:  a.HomepageURL,
		IsPublic:     a.IsPublic,
		IsVerified:   a.IsVerified,
		CreatedAt:    a.CreatedAt,
	}
}

// OAuthAuthorizationCode represents a temporary authorization code
type OAuthAuthorizationCode struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	Code                string    `json:"-" db:"code"` // Hashed
	ClientID            string    `json:"client_id" db:"client_id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	Scopes              []string  `json:"scopes" db:"scopes"`
	RedirectURI         string    `json:"redirect_uri" db:"redirect_uri"`
	CodeChallenge       *string   `json:"-" db:"code_challenge"`        // For PKCE
	CodeChallengeMethod *string   `json:"-" db:"code_challenge_method"` // "plain" or "S256"
	Nonce               *string   `json:"-" db:"nonce"`                 // For OpenID Connect
	State               *string   `json:"-" db:"state"`
	ExpiresAt           time.Time `json:"expires_at" db:"expires_at"`
	Used                bool      `json:"used" db:"used"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// OAuthAccessToken represents an issued access token
type OAuthAccessToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TokenHash string     `json:"-" db:"token_hash"` // SHA256 hash of token
	ClientID  string     `json:"client_id" db:"client_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Scopes    []string   `json:"scopes" db:"scopes"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// OAuthRefreshToken represents an issued refresh token
type OAuthRefreshToken struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TokenHash     string     `json:"-" db:"token_hash"` // SHA256 hash of token
	AccessTokenID uuid.UUID  `json:"access_token_id" db:"access_token_id"`
	ClientID      string     `json:"client_id" db:"client_id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	Scopes        []string   `json:"scopes" db:"scopes"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	RotatedAt     *time.Time `json:"rotated_at,omitempty" db:"rotated_at"`       // When rotated to new token
	RotatedToID   *uuid.UUID `json:"rotated_to_id,omitempty" db:"rotated_to_id"` // Points to new refresh token
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevokedReason *string    `json:"revoked_reason,omitempty" db:"revoked_reason"` // For reuse detection
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// OAuthUserAuthorization tracks which apps a user has authorized
type OAuthUserAuthorization struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	ClientID     string     `json:"client_id" db:"client_id"`
	Scopes       []string   `json:"scopes" db:"scopes"`
	AuthorizedAt time.Time  `json:"authorized_at" db:"authorized_at"`
	LastUsedAt   time.Time  `json:"last_used_at" db:"last_used_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// OAuthUserAuthorizationResponse is the API response for authorized apps
type OAuthUserAuthorizationResponse struct {
	ID           uuid.UUID        `json:"id"`
	App          OAuthAppResponse `json:"app"`
	Scopes       []string         `json:"scopes"`
	AuthorizedAt time.Time        `json:"authorized_at"`
	LastUsedAt   time.Time        `json:"last_used_at"`
}
