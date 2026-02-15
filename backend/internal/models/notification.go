package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeMention       NotificationType = "mention"
	NotificationTypeReply         NotificationType = "reply"
	NotificationTypeDirectMessage NotificationType = "direct_message"
	NotificationTypeFriendRequest NotificationType = "friend_request"
	NotificationTypeFriendAccept  NotificationType = "friend_accept"
	NotificationTypeServerInvite  NotificationType = "server_invite"
	NotificationTypeServerJoin    NotificationType = "server_join"
	NotificationTypeReaction      NotificationType = "reaction"
	NotificationTypeSystem        NotificationType = "system"
)

// Notification represents a user notification
type Notification struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	Type      NotificationType `json:"type" db:"type"`
	Title     string           `json:"title" db:"title"`
	Body      string           `json:"body" db:"body"`
	Read      bool             `json:"read" db:"read"`
	Data      *string          `json:"data,omitempty" db:"data"` // JSON encoded extra data

	// References
	ActorID   *uuid.UUID `json:"actor_id,omitempty" db:"actor_id"`     // User who triggered the notification
	ServerID  *uuid.UUID `json:"server_id,omitempty" db:"server_id"`   // Related server
	ChannelID *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"` // Related channel
	MessageID *uuid.UUID `json:"message_id,omitempty" db:"message_id"` // Related message

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NotificationWithActor includes actor user info for display
type NotificationWithActor struct {
	Notification
	ActorUsername   *string `json:"actor_username,omitempty" db:"actor_username"`
	ActorAvatar     *string `json:"actor_avatar,omitempty" db:"actor_avatar"`
	ServerName      *string `json:"server_name,omitempty" db:"server_name"`
	ChannelName     *string `json:"channel_name,omitempty" db:"channel_name"`
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	UserID    uuid.UUID        `json:"user_id" validate:"required"`
	Type      NotificationType `json:"type" validate:"required"`
	Title     string           `json:"title" validate:"required,max=200"`
	Body      string           `json:"body" validate:"required,max=2000"`
	Data      *string          `json:"data,omitempty"`
	ActorID   *uuid.UUID       `json:"actor_id,omitempty"`
	ServerID  *uuid.UUID       `json:"server_id,omitempty"`
	ChannelID *uuid.UUID       `json:"channel_id,omitempty"`
	MessageID *uuid.UUID       `json:"message_id,omitempty"`
}

// NotificationListOptions represents options for listing notifications
type NotificationListOptions struct {
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	Unread     *bool             `json:"unread,omitempty"`
	Types      []NotificationType `json:"types,omitempty"`
}

// NotificationStats contains notification statistics for a user
type NotificationStats struct {
	Total   int `json:"total"`
	Unread  int `json:"unread"`
}
