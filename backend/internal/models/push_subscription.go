package models

import (
	"time"

	"github.com/google/uuid"
)

// PushSubscription represents a Web Push subscription
type PushSubscription struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Endpoint  string     `json:"endpoint" db:"endpoint"`
	P256dh    string     `json:"p256dh" db:"p256dh"`         // Public key for encryption
	Auth      string     `json:"auth" db:"auth"`             // Authentication secret
	UserAgent string     `json:"user_agent" db:"user_agent"` // Browser/device info
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// CreatePushSubscriptionRequest is the request to register a push subscription
type CreatePushSubscriptionRequest struct {
	Endpoint  string `json:"endpoint" validate:"required,url"`
	P256dh    string `json:"p256dh" validate:"required"`
	Auth      string `json:"auth" validate:"required"`
	UserAgent string `json:"user_agent,omitempty"`
}

// PushPayload represents the payload sent to push services
type PushPayload struct {
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Icon      string            `json:"icon,omitempty"`
	Badge     string            `json:"badge,omitempty"`
	Tag       string            `json:"tag,omitempty"`
	Data      map[string]string `json:"data,omitempty"`
	Actions   []PushAction      `json:"actions,omitempty"`
	Timestamp int64             `json:"timestamp,omitempty"`
}

// PushAction represents an action button in a push notification
type PushAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	// Push notification toggles
	PushEnabled        bool `json:"push_enabled" db:"push_enabled"`
	PushMentions       bool `json:"push_mentions" db:"push_mentions"`
	PushDirectMessages bool `json:"push_direct_messages" db:"push_direct_messages"`
	PushReplies        bool `json:"push_replies" db:"push_replies"`
	PushFriendRequests bool `json:"push_friend_requests" db:"push_friend_requests"`
	PushServerInvites  bool `json:"push_server_invites" db:"push_server_invites"`

	// Sound settings
	SoundEnabled bool   `json:"sound_enabled" db:"sound_enabled"`
	SoundMessage string `json:"sound_message" db:"sound_message"`
	SoundMention string `json:"sound_mention" db:"sound_mention"`

	// Desktop notification settings
	DesktopEnabled  bool `json:"desktop_enabled" db:"desktop_enabled"`
	DesktopPreviews bool `json:"desktop_previews" db:"desktop_previews"`

	// DND settings
	DoNotDisturb      bool       `json:"do_not_disturb" db:"do_not_disturb"`
	DoNotDisturbUntil *time.Time `json:"do_not_disturb_until,omitempty" db:"do_not_disturb_until"`

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultNotificationPreferences returns default notification preferences
func DefaultNotificationPreferences(userID uuid.UUID) *NotificationPreferences {
	return &NotificationPreferences{
		UserID:             userID,
		PushEnabled:        true,
		PushMentions:       true,
		PushDirectMessages: true,
		PushReplies:        true,
		PushFriendRequests: true,
		PushServerInvites:  true,
		SoundEnabled:       true,
		SoundMessage:       "default",
		SoundMention:       "mention",
		DesktopEnabled:     true,
		DesktopPreviews:    true,
		DoNotDisturb:       false,
		UpdatedAt:          time.Now(),
	}
}

// UpdateNotificationPreferencesRequest is the request to update notification preferences
type UpdateNotificationPreferencesRequest struct {
	PushEnabled        *bool      `json:"push_enabled,omitempty"`
	PushMentions       *bool      `json:"push_mentions,omitempty"`
	PushDirectMessages *bool      `json:"push_direct_messages,omitempty"`
	PushReplies        *bool      `json:"push_replies,omitempty"`
	PushFriendRequests *bool      `json:"push_friend_requests,omitempty"`
	PushServerInvites  *bool      `json:"push_server_invites,omitempty"`
	SoundEnabled       *bool      `json:"sound_enabled,omitempty"`
	SoundMessage       *string    `json:"sound_message,omitempty"`
	SoundMention       *string    `json:"sound_mention,omitempty"`
	DesktopEnabled     *bool      `json:"desktop_enabled,omitempty"`
	DesktopPreviews    *bool      `json:"desktop_previews,omitempty"`
	DoNotDisturb       *bool      `json:"do_not_disturb,omitempty"`
	DoNotDisturbUntil  *time.Time `json:"do_not_disturb_until,omitempty"`
}
