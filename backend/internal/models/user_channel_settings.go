package models

import (
	"time"

	"github.com/google/uuid"
)

// UserChannelSettings represents per-user settings for a specific channel
type UserChannelSettings struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	Muted     bool      `json:"muted" db:"muted"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateChannelMuteRequest represents a request to mute/unmute a channel
type UpdateChannelMuteRequest struct {
	Muted bool `json:"muted"`
}
