package models

import "github.com/google/uuid"

// UserUpdate represents a partial update to a user
type UserUpdate struct {
	Username     *string `json:"username,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	BannerURL    *string `json:"banner_url,omitempty"`
	Bio          *string `json:"bio,omitempty"`
	CustomStatus *string `json:"custom_status,omitempty"`
}

// ServerUpdate represents a partial update to a server
type ServerUpdate struct {
	Name                  *string    `json:"name,omitempty"`
	IconURL               *string    `json:"icon_url,omitempty"`
	BannerURL             *string    `json:"banner_url,omitempty"`
	Description           *string    `json:"description,omitempty"`
	DefaultChannelID      *uuid.UUID `json:"default_channel_id,omitempty"`
	AFKChannelID          *uuid.UUID `json:"afk_channel_id,omitempty"`
	AFKTimeout            *int       `json:"afk_timeout,omitempty"`
	VerificationLevel     *int       `json:"verification_level,omitempty"`
	ExplicitContentFilter *int       `json:"explicit_content_filter,omitempty"`
	DefaultNotifications  *int       `json:"default_notifications,omitempty"`
}

// RoleUpdate represents a partial update to a role
type RoleUpdate struct {
	Name        *string `json:"name,omitempty"`
	Color       *int    `json:"color,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
	Permissions *int64  `json:"permissions,omitempty"`
	Position    *int    `json:"position,omitempty"`
}
