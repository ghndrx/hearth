package models

import (
	"time"

	"github.com/google/uuid"
)

// Attachment represents a file attached to a message
type Attachment struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MessageID   uuid.UUID `json:"message_id" db:"message_id"`
	Filename    string    `json:"filename" db:"filename"`
	URL         string    `json:"url" db:"url"`
	ProxyURL    string    `json:"proxy_url,omitempty" db:"proxy_url"`
	ContentType string    `json:"content_type" db:"content_type"`
	Size        int64     `json:"size" db:"size"`
	Width       *int      `json:"width,omitempty" db:"width"`
	Height      *int      `json:"height,omitempty" db:"height"`
	Ephemeral   bool      `json:"ephemeral" db:"ephemeral"`
	
	// E2EE fields
	Encrypted   bool   `json:"encrypted" db:"encrypted"`
	EncryptedKey string `json:"encrypted_key,omitempty" db:"encrypted_key"` // Key encrypted with recipient's public key
	IV          string `json:"iv,omitempty" db:"iv"`
	
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Embed represents an embedded content (link preview, etc.)
type Embed struct {
	Type        string       `json:"type"` // "rich", "image", "video", "link"
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Timestamp   *time.Time   `json:"timestamp,omitempty"`
	Color       int          `json:"color,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Image       *EmbedMedia  `json:"image,omitempty"`
	Thumbnail   *EmbedMedia  `json:"thumbnail,omitempty"`
	Video       *EmbedMedia  `json:"video,omitempty"`
	Provider    *EmbedProvider `json:"provider,omitempty"`
	Author      *EmbedAuthor   `json:"author,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedMedia struct {
	URL      string `json:"url"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Reaction represents a reaction on a message
type Reaction struct {
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ReactionCount represents aggregated reaction counts
type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me"` // Whether current user reacted
}

// Mention types
type MentionType int

const (
	MentionTypeUser    MentionType = iota
	MentionTypeRole
	MentionTypeChannel
	MentionTypeEveryone
	MentionTypeHere
)

// Mention represents a mention in a message
type Mention struct {
	Type MentionType `json:"type"`
	ID   uuid.UUID   `json:"id,omitempty"`
}
