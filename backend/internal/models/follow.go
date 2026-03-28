package models

import (
	"time"

	"github.com/google/uuid"
)

// FollowedChannel represents a channel following another channel
type FollowedChannel struct {
	ChannelID         uuid.UUID `json:"channel_id" db:"channel_id"`
	FollowerChannelID uuid.UUID `json:"follower_channel_id" db:"follower_channel_id"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}
