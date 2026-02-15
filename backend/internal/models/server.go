package models

import (
	"time"

	"github.com/google/uuid"
)

// Server represents a Hearth server (community)
type Server struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	Name                  string     `json:"name" db:"name"`
	IconURL               *string    `json:"icon_url,omitempty" db:"icon_url"`
	BannerURL             *string    `json:"banner_url,omitempty" db:"banner_url"`
	Description           *string    `json:"description,omitempty" db:"description"`
	OwnerID               uuid.UUID  `json:"owner_id" db:"owner_id"`
	DefaultChannelID      *uuid.UUID `json:"default_channel_id,omitempty" db:"default_channel_id"`
	AFKChannelID          *uuid.UUID `json:"afk_channel_id,omitempty" db:"afk_channel_id"`
	AFKTimeout            int        `json:"afk_timeout" db:"afk_timeout"`
	VerificationLevel     int        `json:"verification_level" db:"verification_level"`
	ExplicitContentFilter int        `json:"explicit_content_filter" db:"explicit_content_filter"`
	DefaultNotifications  int        `json:"default_notifications" db:"default_notifications"`
	Features              []string   `json:"features" db:"features"`
	MaxMembers            int        `json:"max_members" db:"max_members"`
	VanityURLCode         *string    `json:"vanity_url_code,omitempty" db:"vanity_url_code"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// VerificationLevel constants
const (
	VerificationNone     = 0 // Unrestricted
	VerificationLow      = 1 // Must have verified email
	VerificationMedium   = 2 // Must be registered for 5+ minutes
	VerificationHigh     = 3 // Must be member for 10+ minutes
	VerificationVeryHigh = 4 // Must have verified phone
)

// ExplicitContentFilter constants
const (
	ExplicitFilterDisabled    = 0 // Don't scan
	ExplicitFilterNoRole      = 1 // Scan messages from members without roles
	ExplicitFilterAllMembers  = 2 // Scan all messages
)

// DefaultNotificationLevel constants
const (
	NotifyAllMessages = 0 // Notify for all messages
	NotifyMentionsOnly = 1 // Only notify for mentions
)

// CreateServerRequest is the input for creating a server
type CreateServerRequest struct {
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	IconURL  *string `json:"icon_url,omitempty"`
	Template *string `json:"template,omitempty"`
}

// UpdateServerRequest is the input for updating server settings
type UpdateServerRequest struct {
	Name                  *string  `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	IconURL               *string  `json:"icon_url,omitempty"`
	BannerURL             *string  `json:"banner_url,omitempty"`
	Description           *string  `json:"description,omitempty" validate:"omitempty,max=300"`
	AFKChannelID          *string  `json:"afk_channel_id,omitempty"`
	AFKTimeout            *int     `json:"afk_timeout,omitempty"`
	VerificationLevel     *int     `json:"verification_level,omitempty"`
	ExplicitContentFilter *int     `json:"explicit_content_filter,omitempty"`
	DefaultNotifications  *int     `json:"default_notifications,omitempty"`
}

// Member represents a user's membership in a server
type Member struct {
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID     uuid.UUID  `json:"server_id" db:"server_id"`
	Nickname     *string    `json:"nickname,omitempty" db:"nickname"`
	JoinedAt     time.Time  `json:"joined_at" db:"joined_at"`
	PremiumSince *time.Time `json:"premium_since,omitempty" db:"premium_since"`
	Deaf         bool       `json:"deaf" db:"deaf"`
	Mute         bool       `json:"mute" db:"mute"`
	Pending      bool       `json:"pending" db:"pending"`
	Temporary    bool       `json:"temporary" db:"temporary"`

	// Populated from joins
	User  *PublicUser `json:"user,omitempty"`
	Roles []uuid.UUID `json:"roles,omitempty"`
}

// DisplayName returns the member's display name (nickname or username)
func (m *Member) DisplayName(user *User) string {
	if m.Nickname != nil && *m.Nickname != "" {
		return *m.Nickname
	}
	if user != nil {
		return user.Username
	}
	return ""
}

// Ban represents a server ban
type Ban struct {
	ServerID  uuid.UUID  `json:"server_id" db:"server_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Reason    *string    `json:"reason,omitempty" db:"reason"`
	BannedBy  *uuid.UUID `json:"banned_by,omitempty" db:"banned_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	User *PublicUser `json:"user,omitempty"`
}

// Invite represents a server invite
type Invite struct {
	Code      string     `json:"code" db:"code"`
	ServerID  uuid.UUID  `json:"server_id" db:"server_id"`
	ChannelID uuid.UUID  `json:"channel_id" db:"channel_id"`
	CreatorID uuid.UUID  `json:"creator_id" db:"creator_id"`
	MaxUses   int        `json:"max_uses" db:"max_uses"`
	Uses      int        `json:"uses" db:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Temporary bool       `json:"temporary" db:"temporary"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	Server  *Server     `json:"server,omitempty"`
	Channel *Channel    `json:"channel,omitempty"`
	Creator *PublicUser `json:"creator,omitempty"`
}

// IsExpired checks if the invite has expired
func (i *Invite) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// IsMaxUsesReached checks if the invite has reached max uses
func (i *Invite) IsMaxUsesReached() bool {
	if i.MaxUses == 0 {
		return false
	}
	return i.Uses >= i.MaxUses
}

// IsValid checks if the invite can be used
func (i *Invite) IsValid() bool {
	return !i.IsExpired() && !i.IsMaxUsesReached()
}

// CreateInviteRequest is the input for creating an invite
type CreateInviteRequest struct {
	MaxAge    *int  `json:"max_age,omitempty"`    // seconds, 0 = never
	MaxUses   *int  `json:"max_uses,omitempty"`   // 0 = unlimited
	Temporary *bool `json:"temporary,omitempty"` // kick when disconnect
}

// ServerWithCounts includes member and online counts
type ServerWithCounts struct {
	Server
	MemberCount int `json:"member_count"`
	OnlineCount int `json:"online_count"`
}
