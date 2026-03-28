package models

import (
	"time"

	"github.com/google/uuid"
)

// RichActivity represents a user's rich activity (game, streaming, etc.)
// Note: Uses ActivityType and constants from presence.go
type RichActivity struct {
	ID      uuid.UUID    `json:"id" db:"id"`
	UserID  uuid.UUID    `json:"user_id" db:"user_id"`
	Type    ActivityType `json:"type" db:"type"`
	Name    string       `json:"name" db:"name"`
	Details *string      `json:"details,omitempty" db:"details"`
	State   *string      `json:"state,omitempty" db:"state"`

	// Rich presence fields
	ApplicationID *uuid.UUID `json:"application_id,omitempty" db:"application_id"`
	LargeImage    *string    `json:"large_image,omitempty" db:"large_image"`
	LargeText     *string    `json:"large_text,omitempty" db:"large_text"`
	SmallImage    *string    `json:"small_image,omitempty" db:"small_image"`
	SmallText     *string    `json:"small_text,omitempty" db:"small_text"`

	// Timestamps
	StartTime *time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`

	// Party/Group info
	PartyID   *string `json:"party_id,omitempty" db:"party_id"`
	PartySize *int    `json:"party_size,omitempty" db:"party_size"`
	PartyMax  *int    `json:"party_max,omitempty" db:"party_max"`

	// Secrets for joining/spectating
	JoinSecret     *string `json:"join_secret,omitempty" db:"join_secret"`
	SpectateSecret *string `json:"spectate_secret,omitempty" db:"spectate_secret"`
	MatchSecret    *string `json:"match_secret,omitempty" db:"match_secret"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// GameMetadata represents metadata about a game/application
type GameMetadata struct {
	ID            uuid.UUID `json:"id" db:"id"`
	ApplicationID string    `json:"application_id" db:"application_id"`
	Name          string    `json:"name" db:"name"`
	Icon          *string   `json:"icon,omitempty" db:"icon"`
	Summary       *string   `json:"summary,omitempty" db:"summary"`
	Developer     *string   `json:"developer,omitempty" db:"developer"`
	Publishers    []string  `json:"publishers,omitempty" db:"publishers"`
	Genres        []string  `json:"genres,omitempty" db:"genres"`
	Executables   []string  `json:"executables,omitempty" db:"executables"`
	Verified      bool      `json:"verified" db:"verified"`
	Aliases       []string  `json:"aliases,omitempty" db:"aliases"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// EnhancedPresence represents a user's enhanced presence with rich activities
type EnhancedPresence struct {
	UserID       uuid.UUID      `json:"user_id"`
	Status       PresenceStatus `json:"status"`
	CustomStatus *string        `json:"custom_status,omitempty"`
	Activities   []RichActivity `json:"activities,omitempty"`
	ClientStatus *ClientStatus  `json:"client_status,omitempty"`
	LastSeen     time.Time      `json:"last_seen"`
}

// SetActivityRequest is the input for setting a custom activity
type SetActivityRequest struct {
	Type       ActivityType `json:"type" validate:"required"`
	Name       string       `json:"name" validate:"required,min=1,max=128"`
	Details    *string      `json:"details,omitempty" validate:"omitempty,max=128"`
	State      *string      `json:"state,omitempty" validate:"omitempty,max=128"`
	LargeImage *string      `json:"large_image,omitempty"`
	LargeText  *string      `json:"large_text,omitempty" validate:"omitempty,max=128"`
	SmallImage *string      `json:"small_image,omitempty"`
	SmallText  *string      `json:"small_text,omitempty" validate:"omitempty,max=128"`
}

// UpdatePresenceRequest is the input for updating user presence
type UpdatePresenceRequest struct {
	Status       *PresenceStatus      `json:"status,omitempty"`
	CustomStatus *string              `json:"custom_status,omitempty" validate:"omitempty,max=128"`
	Activities   []SetActivityRequest `json:"activities,omitempty"`
}

// RichPresenceUpdateEvent is sent when a user's rich presence changes
type RichPresenceUpdateEvent struct {
	UserID   uuid.UUID         `json:"user_id"`
	ServerID *uuid.UUID        `json:"server_id,omitempty"`
	Presence *EnhancedPresence `json:"presence"`
}
