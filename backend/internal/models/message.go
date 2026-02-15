package models

import (
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeDefault            MessageType = "default"
	MessageTypeReply              MessageType = "reply"
	MessageTypeRecipientAdd       MessageType = "recipient_add"
	MessageTypeRecipientRemove    MessageType = "recipient_remove"
	MessageTypeCall               MessageType = "call"
	MessageTypeChannelNameChange  MessageType = "channel_name_change"
	MessageTypeChannelIconChange  MessageType = "channel_icon_change"
	MessageTypePinned             MessageType = "pinned"
	MessageTypeMemberJoin         MessageType = "member_join"
	MessageTypeThreadCreated      MessageType = "thread_created"
)

// Message represents a chat message
type Message struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	ChannelID        uuid.UUID   `json:"channel_id" db:"channel_id"`
	ServerID         *uuid.UUID  `json:"server_id,omitempty" db:"server_id"`
	AuthorID         uuid.UUID   `json:"author_id" db:"author_id"`
	Content          string      `json:"content" db:"content"`
	EncryptedContent string      `json:"encrypted_content,omitempty" db:"encrypted_content"`
	Type             MessageType `json:"type" db:"type"`
	ReplyToID        *uuid.UUID  `json:"reply_to_id,omitempty" db:"reply_to_id"`
	ThreadID         *uuid.UUID  `json:"thread_id,omitempty" db:"thread_id"`
	Pinned           bool        `json:"pinned" db:"pinned"`
	TTS              bool        `json:"tts" db:"tts"`
	MentionEveryone  bool        `json:"mention_everyone" db:"mention_everyone"`
	Flags            int         `json:"flags" db:"flags"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	EditedAt         *time.Time  `json:"edited_at,omitempty" db:"edited_at"`

	// Populated from joins/aggregations
	Author        *PublicUser  `json:"author,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
	Embeds        []Embed      `json:"embeds,omitempty"`
	Reactions     []Reaction   `json:"reactions,omitempty"`
	Mentions      []uuid.UUID  `json:"mentions,omitempty"`
	MentionRoles  []uuid.UUID  `json:"mention_roles,omitempty"`
	ReferencedMsg *Message     `json:"referenced_message,omitempty"`
}

// MessageFlags
const (
	MessageFlagCrossposted          = 1 << 0
	MessageFlagIsCrosspost          = 1 << 1
	MessageFlagSuppressEmbeds       = 1 << 2
	MessageFlagSourceMsgDeleted     = 1 << 3
	MessageFlagUrgent               = 1 << 4
	MessageFlagHasThread            = 1 << 5
	MessageFlagEphemeral            = 1 << 6
	MessageFlagLoading              = 1 << 7
	MessageFlagFailedToMention      = 1 << 8
)

// Attachment represents a file attached to a message
type Attachment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MessageID    uuid.UUID `json:"message_id" db:"message_id"`
	Filename     string    `json:"filename" db:"filename"`
	URL          string    `json:"url" db:"url"`
	ProxyURL     *string   `json:"proxy_url,omitempty" db:"proxy_url"`
	Size         int64     `json:"size" db:"size"`
	ContentType  *string   `json:"content_type,omitempty" db:"content_type"`
	Width        *int      `json:"width,omitempty" db:"width"`
	Height       *int      `json:"height,omitempty" db:"height"`
	AltText      *string   `json:"alt_text,omitempty" db:"alt_text"` // Accessibility: description for screen readers
	Ephemeral    bool      `json:"ephemeral" db:"ephemeral"`
	Encrypted    bool      `json:"encrypted" db:"encrypted"`
	EncryptedKey string    `json:"encrypted_key,omitempty" db:"encrypted_key"`
	IV           string    `json:"iv,omitempty" db:"iv"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Embed represents a rich embed in a message
type Embed struct {
	Type        string        `json:"type,omitempty"`
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	URL         *string       `json:"url,omitempty"`
	Timestamp   *time.Time    `json:"timestamp,omitempty"`
	Color       *int          `json:"color,omitempty"`
	Footer      *EmbedFooter  `json:"footer,omitempty"`
	Image       *EmbedMedia   `json:"image,omitempty"`
	Thumbnail   *EmbedMedia   `json:"thumbnail,omitempty"`
	Video       *EmbedMedia   `json:"video,omitempty"`
	Provider    *EmbedProvider `json:"provider,omitempty"`
	Author      *EmbedAuthor  `json:"author,omitempty"`
	Fields      []EmbedField  `json:"fields,omitempty"`
}

type EmbedFooter struct {
	Text    string  `json:"text"`
	IconURL *string `json:"icon_url,omitempty"`
}

type EmbedMedia struct {
	URL      string `json:"url"`
	ProxyURL *string `json:"proxy_url,omitempty"`
	Width    *int   `json:"width,omitempty"`
	Height   *int   `json:"height,omitempty"`
}

type EmbedProvider struct {
	Name *string `json:"name,omitempty"`
	URL  *string `json:"url,omitempty"`
}

type EmbedAuthor struct {
	Name    string  `json:"name"`
	URL     *string `json:"url,omitempty"`
	IconURL *string `json:"icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Reaction represents emoji reactions on a message
type Reaction struct {
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	Count     int       `json:"count"`
	Me        bool      `json:"me"` // Did the current user react
}

// ReactionUser tracks individual user reactions
type ReactionUser struct {
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MentionType represents different types of mentions
type MentionType int

const (
	MentionTypeUser MentionType = iota
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

// CreateMessageRequest is the input for sending a message
type CreateMessageRequest struct {
	Content   string  `json:"content" validate:"required,max=2000"`
	ReplyToID *string `json:"reply_to_id,omitempty"`
	TTS       *bool   `json:"tts,omitempty"`
	// Attachments are handled separately via multipart upload
}

// UpdateMessageRequest is the input for editing a message
type UpdateMessageRequest struct {
	Content *string `json:"content,omitempty" validate:"omitempty,max=2000"`
}

// MessageQueryOptions for fetching message history
type MessageQueryOptions struct {
	Before *uuid.UUID
	After  *uuid.UUID
	Around *uuid.UUID
	Limit  int // max 100, default 50
}

// Pin represents a pinned message
type Pin struct {
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	PinnedBy  uuid.UUID `json:"pinned_by" db:"pinned_by"`
	PinnedAt  time.Time `json:"pinned_at" db:"pinned_at"`
}

// TypingIndicator tracks who is typing
type TypingIndicator struct {
	ChannelID uuid.UUID `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadState is defined in readstate.go
