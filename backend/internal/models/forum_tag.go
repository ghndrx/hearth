package models

import (
	"time"

	"github.com/google/uuid"
)

// ForumTag represents a tag that can be applied to forum posts
type ForumTag struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	Name      string    `json:"name" db:"name"`
	EmojiName *string   `json:"emoji_name,omitempty" db:"emoji_name"`
	Moderated bool      `json:"moderated" db:"moderated"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ForumConfig contains forum-channel-specific settings
type ForumConfig struct {
	DefaultReactionEmoji *string    `json:"default_reaction_emoji,omitempty"`
	DefaultSortOrder     int        `json:"default_sort_order"`   // 0=latest_activity, 1=creation_date, 2=pin_weight
	DefaultAutoArchive   int        `json:"default_auto_archive"` // minutes
	AvailableTags        []ForumTag `json:"available_tags,omitempty"`
	RequireTag           bool       `json:"require_tag"`
	DefaultLayout        int        `json:"default_layout"` // 0=list, 1=gallery
	PostGuidelines       *string    `json:"post_guidelines,omitempty"`
}

// CreateForumTagRequest is the input for creating a tag
type CreateForumTagRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=100"`
	EmojiName *string `json:"emoji_name,omitempty" validate:"omitempty,max=128"`
	Moderated bool    `json:"moderated"`
}

// UpdateForumTagRequest is the input for updating a tag
type UpdateForumTagRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	EmojiName *string `json:"emoji_name,omitempty" validate:"omitempty,max=128"`
	Moderated *bool   `json:"moderated,omitempty"`
}

// ForumPostFilter contains filtering options for forum posts
type ForumPostFilter struct {
	TagIDs      []uuid.UUID `json:"tag_ids,omitempty"`
	SortOrder   int         `json:"sort_order"` // 0=latest_activity, 1=creation_date, 2=pin_weight
	AuthorID    *uuid.UUID  `json:"author_id,omitempty"`
	PinnedOnly  bool        `json:"pinned_only"`
	SearchQuery string      `json:"search_query,omitempty"`
}
