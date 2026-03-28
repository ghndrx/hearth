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
	DisplayName   *string        `json:"display_name,omitempty" db:"display_name"`
	Discriminator string         `json:"discriminator" db:"discriminator"`
	PasswordHash  string         `json:"-" db:"password_hash"`
	AvatarURL     *string        `json:"avatar_url,omitempty" db:"avatar_url"`
	BannerURL     *string        `json:"banner_url,omitempty" db:"banner_url"`
	Bio           *string        `json:"bio,omitempty" db:"bio"`
	AboutMe       *string        `json:"about_me,omitempty" db:"about_me"`
	Pronouns      *string        `json:"pronouns,omitempty" db:"pronouns"`
	AccentColor   *int           `json:"accent_color,omitempty" db:"accent_color"`
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
	ID            uuid.UUID      `json:"id" db:"id"`
	Username      string         `json:"username" db:"username"`
	DisplayName   *string        `json:"display_name,omitempty" db:"display_name"`
	Discriminator string         `json:"discriminator" db:"discriminator"`
	AvatarURL     *string        `json:"avatar_url,omitempty" db:"avatar_url"`
	BannerURL     *string        `json:"banner_url,omitempty" db:"banner_url"`
	Bio           *string        `json:"bio,omitempty" db:"bio"`
	AboutMe       *string        `json:"about_me,omitempty" db:"about_me"`
	Pronouns      *string        `json:"pronouns,omitempty" db:"pronouns"`
	AccentColor   *int           `json:"accent_color,omitempty" db:"accent_color"`
	Status        PresenceStatus `json:"status" db:"status"`
	CustomStatus  *string        `json:"custom_status,omitempty" db:"custom_status"`
	Flags         int64          `json:"flags" db:"flags"`
}

// ToPublic converts a User to a PublicUser (safe for API responses)
func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:            u.ID,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Discriminator: u.Discriminator,
		AvatarURL:     u.AvatarURL,
		BannerURL:     u.BannerURL,
		Bio:           u.Bio,
		AboutMe:       u.AboutMe,
		Pronouns:      u.Pronouns,
		AccentColor:   u.AccentColor,
		Status:        u.Status,
		CustomStatus:  u.CustomStatus,
		Flags:         u.Flags,
	}
}

// ConnectedAccountType represents types of connected accounts
type ConnectedAccountType string

const (
	ConnectedAccountGitHub      ConnectedAccountType = "github"
	ConnectedAccountTwitter     ConnectedAccountType = "twitter"
	ConnectedAccountSpotify     ConnectedAccountType = "spotify"
	ConnectedAccountSteam       ConnectedAccountType = "steam"
	ConnectedAccountTwitch      ConnectedAccountType = "twitch"
	ConnectedAccountYouTube     ConnectedAccountType = "youtube"
	ConnectedAccountReddit      ConnectedAccountType = "reddit"
	ConnectedAccountPlayStation ConnectedAccountType = "playstation"
	ConnectedAccountXbox        ConnectedAccountType = "xbox"
)

// ConnectedAccountVisibility defines visibility levels
type ConnectedAccountVisibility int

const (
	VisibilityPrivate     ConnectedAccountVisibility = 0
	VisibilityFriendsOnly ConnectedAccountVisibility = 1
	VisibilityEveryone    ConnectedAccountVisibility = 2
)

// ConnectedAccount represents a linked external account
type ConnectedAccount struct {
	ID           uuid.UUID                  `json:"id" db:"id"`
	UserID       uuid.UUID                  `json:"user_id" db:"user_id"`
	Type         ConnectedAccountType       `json:"type" db:"type"`
	AccountID    string                     `json:"account_id" db:"account_id"`
	AccountName  *string                    `json:"account_name,omitempty" db:"account_name"`
	Verified     bool                       `json:"verified" db:"verified"`
	Visibility   ConnectedAccountVisibility `json:"visibility" db:"visibility"`
	ShowActivity bool                       `json:"show_activity" db:"show_activity"`
	Metadata     map[string]interface{}     `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at" db:"updated_at"`
}

// UserBadge represents an achievement or badge
type UserBadge struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	BadgeType   string                 `json:"badge_type" db:"badge_type"`
	BadgeName   string                 `json:"badge_name" db:"badge_name"`
	BadgeIcon   *string                `json:"badge_icon,omitempty" db:"badge_icon"`
	Description *string                `json:"description,omitempty" db:"description"`
	EarnedAt    time.Time              `json:"earned_at" db:"earned_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// Badge types
const (
	BadgeEarlySupporter = "early_supporter"
	BadgeVerifiedBot    = "verified_bot"
	BadgeBugHunter      = "bug_hunter"
	BadgePremium        = "premium"
	BadgeStaff          = "staff"
	BadgePartner        = "partner"
	BadgeHypeSquad      = "hypesquad"
	BadgeNitro          = "nitro"
)

// UserCustomStatus represents a user's custom status with emoji
type UserCustomStatus struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	CustomText *string    `json:"custom_text,omitempty" db:"custom_text"`
	Emoji      *string    `json:"emoji,omitempty" db:"emoji"`
	EmojiID    *uuid.UUID `json:"emoji_id,omitempty" db:"emoji_id"`
	EmojiName  *string    `json:"emoji_name,omitempty" db:"emoji_name"`
	ClearAfter *time.Time `json:"clear_after,omitempty" db:"clear_after"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// Tag returns the full username with discriminator (e.g., "user#1234")
func (u *User) Tag() string {
	return u.Username + "#" + u.Discriminator
}

// Note: Session type is defined in session.go

// CreateUserRequest is the input for user registration
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=2,max=32"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// UpdateUserRequest is the input for updating user profile
type UpdateUserRequest struct {
	Username     *string `json:"username,omitempty" validate:"omitempty,min=2,max=32"`
	DisplayName  *string `json:"display_name,omitempty" validate:"omitempty,max=32"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	BannerURL    *string `json:"banner_url,omitempty"`
	Bio          *string `json:"bio,omitempty" validate:"omitempty,max=190"`
	AboutMe      *string `json:"about_me,omitempty" validate:"omitempty,max=2000"`
	Pronouns     *string `json:"pronouns,omitempty" validate:"omitempty,max=32"`
	AccentColor  *int    `json:"accent_color,omitempty"`
	CustomStatus *string `json:"custom_status,omitempty" validate:"omitempty,max=128"`
}

// UpdateStatusRequest is the input for updating user status
type UpdateStatusRequest struct {
	Status     *PresenceStatus `json:"status,omitempty"`
	CustomText *string         `json:"custom_text,omitempty" validate:"omitempty,max=128"`
	Emoji      *string         `json:"emoji,omitempty" validate:"omitempty,max=64"`
	EmojiID    *uuid.UUID      `json:"emoji_id,omitempty"`
	EmojiName  *string         `json:"emoji_name,omitempty" validate:"omitempty,max=64"`
	ClearAfter *time.Time      `json:"clear_after,omitempty"`
}

// CreateConnectedAccountRequest is the input for linking an account
type CreateConnectedAccountRequest struct {
	Type         ConnectedAccountType       `json:"type" validate:"required"`
	AccountID    string                     `json:"account_id" validate:"required"`
	AccountName  *string                    `json:"account_name,omitempty"`
	Visibility   ConnectedAccountVisibility `json:"visibility"`
	ShowActivity bool                       `json:"show_activity"`
	AccessToken  string                     `json:"access_token,omitempty"`
	RefreshToken string                     `json:"refresh_token,omitempty"`
}

// UpdateConnectedAccountRequest is the input for updating a connected account
type UpdateConnectedAccountRequest struct {
	Visibility   *ConnectedAccountVisibility `json:"visibility,omitempty"`
	ShowActivity *bool                       `json:"show_activity,omitempty"`
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
