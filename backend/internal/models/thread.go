package models

import (
	"time"

	"github.com/google/uuid"
)

// ThreadWithDetails includes additional details for API responses
type ThreadWithDetails struct {
	Thread
	Owner        *User           `json:"owner,omitempty"`
	LastMessage  *ThreadMessage  `json:"last_message,omitempty"`
	Participants []ThreadMember  `json:"participants,omitempty"`
}

// ThreadMember represents a user participating in a thread
type ThreadMember struct {
	ThreadID uuid.UUID `json:"thread_id" db:"thread_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`

	// Populated by join
	User *User `json:"user,omitempty" db:"-"`
}

// ThreadNotificationLevel represents notification preference levels
type ThreadNotificationLevel string

const (
	ThreadNotifyAll      ThreadNotificationLevel = "all"
	ThreadNotifyMentions ThreadNotificationLevel = "mentions"
	ThreadNotifyNone     ThreadNotificationLevel = "none"
)

// ThreadNotificationPreference represents a user's notification settings for a thread
type ThreadNotificationPreference struct {
	ThreadID  uuid.UUID               `json:"thread_id" db:"thread_id"`
	UserID    uuid.UUID               `json:"user_id" db:"user_id"`
	Level     ThreadNotificationLevel `json:"level" db:"level"`
	CreatedAt time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt time.Time               `json:"updated_at" db:"updated_at"`
}

// ThreadPresence represents a user currently viewing a thread
type ThreadPresence struct {
	ThreadID   uuid.UUID `json:"thread_id" db:"thread_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	LastSeenAt time.Time `json:"last_seen_at" db:"last_seen_at"`

	// Populated by join
	User *User `json:"user,omitempty" db:"-"`
}

// ThreadPresenceUser is a simplified user for presence display
type ThreadPresenceUser struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	DisplayName *string   `json:"display_name,omitempty" db:"display_name"`
	Avatar      *string   `json:"avatar,omitempty" db:"avatar"`
}

// UpdateThreadNotificationRequest is the request body for updating notification preferences
type UpdateThreadNotificationRequest struct {
	Level ThreadNotificationLevel `json:"level" validate:"required,oneof=all mentions none"`
}

// ThreadPresenceUpdateRequest is the request body for updating presence in a thread
type ThreadPresenceUpdateRequest struct {
	Active bool `json:"active"` // true = user is viewing, false = user left
}

// ThreadPresenceResponse is the response for thread presence queries
type ThreadPresenceResponse struct {
	ThreadID    uuid.UUID            `json:"thread_id"`
	ActiveUsers []ThreadPresenceUser `json:"active_users"`
}
