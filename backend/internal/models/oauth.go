package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthProvider represents a linked OAuth provider account
type OAuthProvider struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Provider       string     `json:"provider" db:"provider"` // github, google, discord
	ProviderUserID string     `json:"provider_user_id" db:"provider_user_id"`
	Email          string     `json:"email" db:"email"`
	Username       *string    `json:"username,omitempty" db:"username"`
	DisplayName    *string    `json:"display_name,omitempty" db:"display_name"`
	AvatarURL      *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	AccessToken    *string    `json:"-" db:"access_token"`  // Encrypted, not exposed in JSON
	RefreshToken   *string    `json:"-" db:"refresh_token"` // Encrypted, not exposed in JSON
	TokenExpiresAt *time.Time `json:"-" db:"token_expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// OAuthProviderResponse is a safe representation for API responses
type OAuthProviderResponse struct {
	ID             uuid.UUID `json:"id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	Email          string    `json:"email"`
	Username       *string   `json:"username,omitempty"`
	DisplayName    *string   `json:"display_name,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToResponse converts OAuthProvider to a safe API response
func (o *OAuthProvider) ToResponse() OAuthProviderResponse {
	return OAuthProviderResponse{
		ID:             o.ID,
		Provider:       o.Provider,
		ProviderUserID: o.ProviderUserID,
		Email:          o.Email,
		Username:       o.Username,
		DisplayName:    o.DisplayName,
		AvatarURL:      o.AvatarURL,
		CreatedAt:      o.CreatedAt,
	}
}

// LinkOAuthRequest is the input for linking an OAuth provider to an existing account
type LinkOAuthRequest struct {
	Provider string `json:"provider" validate:"required,oneof=github google discord"`
}

// OAuthLinkState represents the state stored during OAuth linking flow
type OAuthLinkState struct {
	UserID    uuid.UUID `json:"user_id"`
	Provider  string    `json:"provider"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}
