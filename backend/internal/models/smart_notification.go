package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	NotificationPriorityUrgent NotificationPriority = "urgent"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityLow    NotificationPriority = "low"
)

// NotificationDeliveryMode represents how a notification should be delivered
type NotificationDeliveryMode string

const (
	DeliveryImmediate NotificationDeliveryMode = "immediate"
	DeliveryBatched   NotificationDeliveryMode = "batched"
)

// NotificationCategory classifies notification content
type NotificationCategory string

const (
	NotifCategoryMention       NotificationCategory = "mention"
	NotifCategoryDirectMessage NotificationCategory = "direct_message"
	NotifCategoryReply         NotificationCategory = "reply"
	NotifCategoryReaction      NotificationCategory = "reaction"
	NotifCategoryServerEvent   NotificationCategory = "server_event"
	NotifCategorySystem        NotificationCategory = "system"
	NotifCategorySocial        NotificationCategory = "social"
)

// SmartNotification extends a notification with priority scoring and delivery metadata
type SmartNotification struct {
	Notification

	// Priority scoring
	PriorityScore    int                      `json:"priority_score" db:"priority_score"`       // 0-100
	Priority         NotificationPriority     `json:"priority" db:"priority"`
	DeliveryMode     NotificationDeliveryMode `json:"delivery_mode" db:"delivery_mode"`
	Category         NotificationCategory     `json:"category" db:"category"`

	// Delivery tracking
	DeliveredAt      *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	ClickedAt        *time.Time `json:"clicked_at,omitempty" db:"clicked_at"`
	DigestID         *uuid.UUID `json:"digest_id,omitempty" db:"digest_id"`
	DigestIncludedAt *time.Time `json:"digest_included_at,omitempty" db:"digest_included_at"`
}

// NotificationDigest represents a batched notification digest
type NotificationDigest struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Title           string     `json:"title" db:"title"`
	Summary         string     `json:"summary" db:"summary"`
	NotificationIDs []uuid.UUID `json:"notification_ids"`
	Count           int        `json:"count" db:"count"`
	DeliveredAt     *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt          *time.Time `json:"read_at,omitempty" db:"read_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// UserEngagement tracks user engagement with notifications
type UserEngagement struct {
	UserID              uuid.UUID         `json:"user_id" db:"user_id"`
	TotalReceived       int               `json:"total_received" db:"total_received"`
	TotalClicked        int               `json:"total_clicked" db:"total_clicked"`
	TotalDismissed      int               `json:"total_dismissed" db:"total_dismissed"`
	ClickRate           float64           `json:"click_rate"`
	TopChannels         []uuid.UUID       `json:"top_channels,omitempty"`
	LastActiveAt        *time.Time        `json:"last_active_at,omitempty" db:"last_active_at"`
	PreferredDelivery   NotificationDeliveryMode `json:"preferred_delivery"`
	UpdatedAt           time.Time         `json:"updated_at" db:"updated_at"`
}

// SnoozeConfig represents a notification snooze configuration
type SnoozeConfig struct {
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Active    bool       `json:"active" db:"active"`
	Until     time.Time  `json:"until" db:"until"`
	ServerID  *uuid.UUID `json:"server_id,omitempty" db:"server_id"`   // nil = global snooze
	ChannelID *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"` // nil = server-level or global
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// MuteConfig represents a notification mute configuration
type MuteConfig struct {
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID  *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"`
	Muted     bool       `json:"muted" db:"muted"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// SmartNotificationPreferences holds user preferences for smart notifications
type SmartNotificationPreferences struct {
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	Enabled             bool      `json:"enabled" db:"enabled"`
	DigestEnabled       bool      `json:"digest_enabled" db:"digest_enabled"`
	DigestIntervalMins  int       `json:"digest_interval_mins" db:"digest_interval_mins"` // default 30
	UrgentAlwaysDeliver bool      `json:"urgent_always_deliver" db:"urgent_always_deliver"`
	ClickTrackingEnabled bool     `json:"click_tracking_enabled" db:"click_tracking_enabled"`
}

// PriorityScoringInput holds the inputs needed to compute a priority score
type PriorityScoringInput struct {
	NotificationType NotificationType
	SenderID         *uuid.UUID
	RecipientID      uuid.UUID
	ServerID         *uuid.UUID
	ChannelID        *uuid.UUID
	HasMention       bool
	IsDM             bool
	IsReply          bool
}

// SnoozeRequest represents a request to snooze notifications
type SnoozeRequest struct {
	DurationMins int        `json:"duration_mins" validate:"required,min=1,max=10080"` // max 7 days
	ServerID     *uuid.UUID `json:"server_id,omitempty"`
	ChannelID    *uuid.UUID `json:"channel_id,omitempty"`
}

// MuteRequest represents a request to mute notifications
type MuteRequest struct {
	ServerID  *uuid.UUID `json:"server_id,omitempty"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	Muted     bool       `json:"muted"`
}

// NotificationClickEvent represents a notification click/open for tracking
type NotificationClickEvent struct {
	NotificationID uuid.UUID `json:"notification_id"`
	UserID         uuid.UUID `json:"user_id"`
	ClickedAt      time.Time `json:"clicked_at"`
}

// DigestListOptions represents options for listing digests
type DigestListOptions struct {
	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
	Unread *bool `json:"unread,omitempty"`
}
