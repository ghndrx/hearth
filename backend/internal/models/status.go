package models

import (
	"time"

	"github.com/google/uuid"
)

// UserActivityStatus represents a user's current status (online, idle, etc.)
type UserActivityStatus string

const (
	UserStatusOnline    UserActivityStatus = "online"
	UserStatusIdle      UserActivityStatus = "idle"
	UserStatusDND       UserActivityStatus = "dnd"
	UserStatusInvisible UserActivityStatus = "invisible"
	UserStatusOffline   UserActivityStatus = "offline"
)

// Status represents a user status update
type Status struct {
	ID              uuid.UUID          `json:"id" db:"id"`
	UserID          uuid.UUID          `json:"user_id" db:"user_id"`
	Status          UserActivityStatus `json:"status" db:"status"`
	GameID          *string            `json:"game_id,omitempty" db:"game_id"`
	ActivityDetails *string            `json:"activity_details,omitempty" db:"activity_details"`
	Timestamp       time.Time          `json:"timestamp" db:"timestamp"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

// Now returns the current time
func Now() time.Time {
	return time.Now()
}
