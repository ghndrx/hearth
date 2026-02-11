package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookType represents the type of webhook
type WebhookType int

const (
	// WebhookTypeIncoming is a standard webhook for sending messages
	WebhookTypeIncoming WebhookType = 1
	// WebhookTypeChannelFollower is for following announcements from another channel
	WebhookTypeChannelFollower WebhookType = 2
	// WebhookTypeApplication is for Discord App interactions
	WebhookTypeApplication WebhookType = 3
)

// Webhook represents a channel webhook
type Webhook struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	Type            WebhookType `json:"type" db:"type"`
	ServerID        *uuid.UUID  `json:"guild_id,omitempty" db:"server_id"`
	ChannelID       uuid.UUID   `json:"channel_id" db:"channel_id"`
	CreatorID       *uuid.UUID  `json:"user,omitempty" db:"creator_id"`
	Name            string      `json:"name" db:"name"`
	Avatar          *string     `json:"avatar,omitempty" db:"avatar"`
	Token           string      `json:"token,omitempty" db:"token"`
	ApplicationID   *uuid.UUID  `json:"application_id,omitempty" db:"application_id"`
	SourceServerID  *uuid.UUID  `json:"source_guild,omitempty" db:"source_server_id"`
	SourceChannelID *uuid.UUID  `json:"source_channel,omitempty" db:"source_channel_id"`
	URL             string      `json:"url,omitempty" db:"-"` // Computed
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`

	// Populated on fetch
	Creator *User    `json:"user_details,omitempty" db:"-"`
	Server  *Server  `json:"guild_details,omitempty" db:"-"`
	Channel *Channel `json:"channel_details,omitempty" db:"-"`
}

// WebhookMessage represents a message sent via webhook
type WebhookMessage struct {
	Content         string   `json:"content,omitempty"`
	Username        string   `json:"username,omitempty"`
	AvatarURL       string   `json:"avatar_url,omitempty"`
	TTS             bool     `json:"tts,omitempty"`
	Embeds          []Embed  `json:"embeds,omitempty"`
	AllowedMentions *struct {
		Parse       []string `json:"parse,omitempty"`
		Roles       []string `json:"roles,omitempty"`
		Users       []string `json:"users,omitempty"`
		RepliedUser bool     `json:"replied_user,omitempty"`
	} `json:"allowed_mentions,omitempty"`
	Components []interface{} `json:"components,omitempty"`
	Files      []interface{} `json:"files,omitempty"`
	Flags      int           `json:"flags,omitempty"`
	ThreadName string        `json:"thread_name,omitempty"`
}

// Validate validates a webhook message
func (m *WebhookMessage) Validate() error {
	if m.Content == "" && len(m.Embeds) == 0 && len(m.Files) == 0 {
		return ErrEmptyMessage
	}
	if len(m.Content) > 2000 {
		return ErrContentTooLong
	}
	if len(m.Embeds) > 10 {
		return ErrTooManyEmbeds
	}
	return nil
}

// Common errors
var (
	ErrEmptyMessage    = NewValidationError("message must have content, embeds, or files")
	ErrContentTooLong  = NewValidationError("content must be 2000 characters or less")
	ErrTooManyEmbeds   = NewValidationError("maximum 10 embeds allowed")
)

// ValidationError represents a validation error
type ValidationError struct {
	message string
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}

func (e *ValidationError) Error() string {
	return e.message
}
