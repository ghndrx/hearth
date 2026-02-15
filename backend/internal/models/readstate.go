package models

import (
	"time"

	"github.com/google/uuid"
)

// ReadState tracks the last read message for a user in a channel
type ReadState struct {
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	ChannelID     uuid.UUID  `json:"channel_id" db:"channel_id"`
	LastMessageID *uuid.UUID `json:"last_message_id,omitempty" db:"last_message_id"`
	MentionCount  int        `json:"mention_count" db:"mention_count"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ReadStateWithUnread includes the unread message count
type ReadStateWithUnread struct {
	ReadState
	UnreadCount int `json:"unread_count"`
}

// ChannelUnreadInfo provides unread information for a channel
type ChannelUnreadInfo struct {
	ChannelID     uuid.UUID  `json:"channel_id"`
	LastMessageID *uuid.UUID `json:"last_message_id,omitempty"`
	UnreadCount   int        `json:"unread_count"`
	MentionCount  int        `json:"mention_count"`
	HasUnread     bool       `json:"has_unread"`
}

// UnreadSummary provides a summary of all unread channels
type UnreadSummary struct {
	TotalUnread    int                 `json:"total_unread"`
	TotalMentions  int                 `json:"total_mentions"`
	Channels       []ChannelUnreadInfo `json:"channels"`
}

// MarkReadRequest is the request to mark a channel as read
type MarkReadRequest struct {
	MessageID *uuid.UUID `json:"message_id,omitempty"` // Optional: mark up to specific message
}

// AckResponse is the response after marking a channel as read
type AckResponse struct {
	ChannelID     uuid.UUID  `json:"channel_id"`
	LastMessageID *uuid.UUID `json:"last_message_id,omitempty"`
	MentionCount  int        `json:"mention_count"`
}
