package models

import (
	"time"

	"github.com/google/uuid"
)

// ChannelNotificationPreference represents per-user, per-channel notification settings
type ChannelNotificationPreference struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	ChannelID  uuid.UUID `json:"channel_id" db:"channel_id"`
	ServerID   uuid.UUID `json:"server_id" db:"server_id"` // denormalized for queries

	// Notification type toggles (true = allow, false = suppress)
	EnableMentions       bool `json:"enable_mentions" db:"enable_mentions"`
	EnableMessages       bool `json:"enable_messages" db:"enable_messages"`
	EnableReactions      bool `json:"enable_reactions" db:"enable_reactions"`
	EnableThreads        bool `json:"enable_threads" db:"enable_threads"`
	EnablePins           bool `json:"enable_pins" db:"enable_pins"`
	EnableVoiceActivity  bool `json:"enable_voice_activity" db:"enable_voice_activity"`

	// Delivery mode for this channel
	DeliveryMode NotificationDeliveryMode `json:"delivery_mode" db:"delivery_mode"` // inherit, immediate, batched

	// Muting
	Muted bool `json:"muted" db:"muted"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultChannelNotificationPreference returns default channel notification settings
func DefaultChannelNotificationPreference(userID, channelID, serverID uuid.UUID) *ChannelNotificationPreference {
	return &ChannelNotificationPreference{
		UserID:            userID,
		ChannelID:         channelID,
		ServerID:          serverID,
		EnableMentions:    true,
		EnableMessages:    true,
		EnableReactions:   true,
		EnableThreads:     true,
		EnablePins:        true,
		EnableVoiceActivity: true,
		DeliveryMode:      DeliveryBatched,
		Muted:             false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// ServerNotificationPreference represents per-user, per-server notification settings
type ServerNotificationPreference struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`

	// Global notification toggles for this server
	EnableMentions       bool `json:"enable_mentions" db:"enable_mentions"`
	EnableMessages       bool `json:"enable_messages" db:"enable_messages"`
	EnableReactions      bool `json:"enable_reactions" db:"enable_reactions"`
	EnableThreads        bool `json:"enable_threads" db:"enable_threads"`

	// Supercategory overrides (roles-based filtering)
	NotifyRoles []uuid.UUID `json:"notify_roles,omitempty" db:"notify_roles"` // only notify for these roles; empty = all

	// Server-level muting
	Muted       bool       `json:"muted" db:"muted"`
	MutedUntil  *time.Time `json:"muted_until,omitempty" db:"muted_until"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultServerNotificationPreference returns default server notification settings
func DefaultServerNotificationPreference(userID, serverID uuid.UUID) *ServerNotificationPreference {
	return &ServerNotificationPreference{
		UserID:            userID,
		ServerID:          serverID,
		EnableMentions:    true,
		EnableMessages:    true,
		EnableReactions:   true,
		EnableThreads:     true,
		Muted:             false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Until is a helper type for nullable time
type Until struct {
	Time  time.Time
	Valid bool
}

// NotificationRoutingDecision represents the result of routing a notification
type NotificationRoutingDecision struct {
	Channel    NotificationChannel     `json:"channel"`     // push, email, in_app, suppressed
	ShouldSend bool                   `json:"should_send"`
	DelaySecs  int                    `json:"delay_secs"`  // seconds to wait before sending
	BatchID    *uuid.UUID            `json:"batch_id,omitempty"`
	Reason     string                `json:"reason"`      // human-readable reason for decision
	Priority   NotificationPriority  `json:"priority"`    // priority level assigned
}

// NotificationChannel represents the delivery channel for a notification
type NotificationChannel string

const (
	NotificationChannelPush   NotificationChannel = "push"
	NotificationChannelEmail   NotificationChannel = "email"
	NotificationChannelInApp   NotificationChannel = "in_app"
	NotificationChannelSuppressed NotificationChannel = "suppressed"
)

// UpdateChannelNotificationPreferenceRequest is the request to update channel notification preferences
type UpdateChannelNotificationPreferenceRequest struct {
	EnableMentions      *bool                    `json:"enable_mentions,omitempty"`
	EnableMessages      *bool                    `json:"enable_messages,omitempty"`
	EnableReactions     *bool                    `json:"enable_reactions,omitempty"`
	EnableThreads       *bool                    `json:"enable_threads,omitempty"`
	EnablePins          *bool                    `json:"enable_pins,omitempty"`
	EnableVoiceActivity *bool                    `json:"enable_voice_activity,omitempty"`
	DeliveryMode        *NotificationDeliveryMode `json:"delivery_mode,omitempty"`
	Muted               *bool                    `json:"muted,omitempty"`
}

// UpdateServerNotificationPreferenceRequest is the request to update server notification preferences
type UpdateServerNotificationPreferenceRequest struct {
	EnableMentions   *bool   `json:"enable_mentions,omitempty"`
	EnableMessages   *bool   `json:"enable_messages,omitempty"`
	EnableReactions  *bool   `json:"enable_reactions,omitempty"`
	EnableThreads    *bool   `json:"enable_threads,omitempty"`
	NotifyRoles      []uuid.UUID `json:"notify_roles,omitempty"`
	Muted            *bool  `json:"muted,omitempty"`
}

// NotificationQueueItem represents an item in the priority notification queue
type NotificationQueueItem struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	NotificationID uuid.UUID       `json:"notification_id" db:"notification_id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	Channel        NotificationChannel `json:"channel" db:"channel"`
	Priority       NotificationPriority `json:"priority" db:"priority"`
	Score          int              `json:"score" db:"score"`
	Status         QueueItemStatus  `json:"status" db:"status"`
	ScheduledAt    time.Time       `json:"scheduled_at" db:"scheduled_at"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	Attempts       int              `json:"attempts" db:"attempts"`
	LastError      *string         `json:"last_error,omitempty" db:"last_error"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// QueueItemStatus represents the status of a queued notification item
type QueueItemStatus string

const (
	QueueItemStatusPending   QueueItemStatus = "pending"
	QueueItemStatusProcessed QueueItemStatus = "processed"
	QueueItemStatusFailed    QueueItemStatus = "failed"
	QueueItemStatusExpired   QueueItemStatus = "expired"
)
