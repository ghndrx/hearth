package models

import (
	"time"

	"github.com/google/uuid"
)

// MentionKind represents different types of mentions
type MentionKind string

const (
	MentionKindUser     MentionKind = "user"
	MentionKindRole     MentionKind = "role"
	MentionKindChannel  MentionKind = "channel"
	MentionKindEveryone MentionKind = "everyone"
	MentionKindHere     MentionKind = "here"
)

// Mention represents a mention record in the database
type Mention struct {
	ID                 uuid.UUID   `json:"id" db:"id"`
	UserID             uuid.UUID   `json:"user_id" db:"user_id"`                                     // Who is mentioned
	MessageID          uuid.UUID   `json:"message_id" db:"message_id"`                               // The message containing the mention
	MentionedBy        uuid.UUID   `json:"mentioned_by" db:"mentioned_by"`                           // Who did the mentioning
	ChannelID          uuid.UUID   `json:"channel_id" db:"channel_id"`                               // Channel where mention occurred
	GuildID            *uuid.UUID  `json:"guild_id,omitempty" db:"guild_id"`                         // Server ID (null for DMs)
	MentionType        MentionKind `json:"mention_type" db:"mention_type"`                           // Type of mention
	MentionedRoleID    *uuid.UUID  `json:"mentioned_role_id,omitempty" db:"mentioned_role_id"`       // For role mentions
	MentionedChannelID *uuid.UUID  `json:"mentioned_channel_id,omitempty" db:"mentioned_channel_id"` // For channel mentions
	ReadAt             *time.Time  `json:"read_at,omitempty" db:"read_at"`                           // When the mention was read
	CreatedAt          time.Time   `json:"created_at" db:"created_at"`

	// Populated from joins
	Author  *PublicUser `json:"author,omitempty"`  // Who mentioned
	Message *Message    `json:"message,omitempty"` // The message
	Channel *Channel    `json:"channel,omitempty"` // The channel
}

// MentionWithContext extends Mention with additional context for API responses
type MentionWithContext struct {
	Mention
	ServerName   *string `json:"server_name,omitempty"`
	ChannelName  *string `json:"channel_name,omitempty"`
	AuthorName   string  `json:"author_name"`
	AuthorAvatar *string `json:"author_avatar,omitempty"`
	Preview      string  `json:"preview"` // Truncated message content
}

// MentionsListResponse is the API response for listing mentions
type MentionsListResponse struct {
	Mentions   []MentionWithContext `json:"mentions"`
	TotalCount int                  `json:"total_count"`
	HasMore    bool                 `json:"has_more"`
}

// MentionStats contains statistics about a user's mentions
type MentionStats struct {
	TotalCount  int `json:"total_count"`
	UnreadCount int `json:"unread_count"`
	TodayCount  int `json:"today_count"`
}

// CreateMentionRequest is the request to create a mention (internal use)
type CreateMentionRequest struct {
	UserID             uuid.UUID
	MessageID          uuid.UUID
	MentionedBy        uuid.UUID
	ChannelID          uuid.UUID
	GuildID            *uuid.UUID
	MentionType        MentionKind
	MentionedRoleID    *uuid.UUID
	MentionedChannelID *uuid.UUID
}

// MentionFilter is used to filter mention queries
type MentionFilter struct {
	UserID      uuid.UUID
	Unread      *bool        // Filter by read status
	MentionType *MentionKind // Filter by mention type
	ChannelID   *uuid.UUID   // Filter by channel
	GuildID     *uuid.UUID   // Filter by server
	Before      *time.Time   // Pagination: mentions before this time
	After       *time.Time   // Pagination: mentions after this time
	Limit       int          // Max results (default 50, max 100)
	Offset      int          // Skip N results
}

// SetDefaults sets default values for the filter
func (f *MentionFilter) SetDefaults() {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}
