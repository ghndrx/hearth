package models

import (
	"time"

	"github.com/google/uuid"
)

// PresenceStatus represents a user's online status
type PresenceStatus string

const (
	StatusOnline    PresenceStatus = "online"
	StatusIdle      PresenceStatus = "idle"
	StatusDND       PresenceStatus = "dnd"
	StatusInvisible PresenceStatus = "invisible"
	StatusOffline   PresenceStatus = "offline"
)

// User represents a Hearth user account
type User struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	Email         string         `json:"email" db:"email"`
	Username      string         `json:"username" db:"username"`
	Discriminator string         `json:"discriminator" db:"discriminator"`
	PasswordHash  string         `json:"-" db:"password_hash"`
	AvatarURL     *string        `json:"avatar_url,omitempty" db:"avatar_url"`
	BannerURL     *string        `json:"banner_url,omitempty" db:"banner_url"`
	Bio           *string        `json:"bio,omitempty" db:"bio"`
	Status        PresenceStatus `json:"status" db:"status"`
	CustomStatus  *string        `json:"custom_status,omitempty" db:"custom_status"`
	MFAEnabled    bool           `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret     *string        `json:"-" db:"mfa_secret"`
	Verified      bool           `json:"verified" db:"verified"`
	Flags         int64          `json:"flags" db:"flags"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// UserFlags for system-level user attributes
const (
	UserFlagStaff       int64 = 1 << 0
	UserFlagPartner     int64 = 1 << 1
	UserFlagBugHunter   int64 = 1 << 2
	UserFlagPremium     int64 = 1 << 3
	UserFlagSystemBot   int64 = 1 << 4
	UserFlagDeletedUser int64 = 1 << 5
)

// PublicUser is a safe representation for API responses
type PublicUser struct {
	ID            uuid.UUID      `json:"id"`
	Username      string         `json:"username"`
	Discriminator string         `json:"discriminator"`
	AvatarURL     *string        `json:"avatar_url,omitempty"`
	BannerURL     *string        `json:"banner_url,omitempty"`
	Bio           *string        `json:"bio,omitempty"`
	Status        PresenceStatus `json:"status"`
	CustomStatus  *string        `json:"custom_status,omitempty"`
	Flags         int64          `json:"flags"`
}

// ToPublic converts a User to a PublicUser (safe for API responses)
func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:            u.ID,
		Username:      u.Username,
		Discriminator: u.Discriminator,
		AvatarURL:     u.AvatarURL,
		BannerURL:     u.BannerURL,
		Bio:           u.Bio,
		Status:        u.Status,
		CustomStatus:  u.CustomStatus,
		Flags:         u.Flags,
	}
}

// Tag returns the full username with discriminator (e.g., "user#1234")
func (u *User) Tag() string {
	return u.Username + "#" + u.Discriminator
}

// Session represents an authenticated user session
type Session struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	RefreshToken string     `json:"-" db:"refresh_token_hash"`
	UserAgent    *string    `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress    *string    `json:"-" db:"ip_address"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// CreateUserRequest is the input for user registration
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=2,max=32"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// UpdateUserRequest is the input for updating user profile
type UpdateUserRequest struct {
	Username     *string `json:"username,omitempty" validate:"omitempty,min=2,max=32"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	BannerURL    *string `json:"banner_url,omitempty"`
	Bio          *string `json:"bio,omitempty" validate:"omitempty,max=190"`
	CustomStatus *string `json:"custom_status,omitempty" validate:"omitempty,max=128"`
}

// LoginRequest is the input for authentication
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse is returned after successful authentication
type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int        `json:"expires_in"`
	User         PublicUser `json:"user"`
}
