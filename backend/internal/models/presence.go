package models

import (
	"time"

	"github.com/google/uuid"
)

// PresenceStatus represents user online status
type PresenceStatus string

const (
	PresenceOnline    PresenceStatus = "online"
	PresenceIdle      PresenceStatus = "idle"
	PresenceDND       PresenceStatus = "dnd"
	PresenceInvisible PresenceStatus = "invisible"
	PresenceOffline   PresenceStatus = "offline"
)

// Presence represents a user's online status
type Presence struct {
	UserID       uuid.UUID      `json:"user_id" db:"user_id"`
	Status       PresenceStatus `json:"status" db:"status"`
	CustomStatus *string        `json:"custom_status,omitempty" db:"custom_status"`
	Activities   []Activity     `json:"activities,omitempty"`
	ClientStatus *ClientStatus  `json:"client_status,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// ClientStatus represents status per client
type ClientStatus struct {
	Desktop PresenceStatus `json:"desktop,omitempty"`
	Mobile  PresenceStatus `json:"mobile,omitempty"`
	Web     PresenceStatus `json:"web,omitempty"`
}

// Activity represents a user activity (playing game, listening, etc.)
type Activity struct {
	Name          string         `json:"name"`
	Type          ActivityType   `json:"type"`
	URL           string         `json:"url,omitempty"`
	Details       string         `json:"details,omitempty"`
	State         string         `json:"state,omitempty"`
	ApplicationID *uuid.UUID     `json:"application_id,omitempty"`
	Timestamps    *ActivityTime  `json:"timestamps,omitempty"`
	Assets        *ActivityAssets `json:"assets,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type ActivityType int

const (
	ActivityTypePlaying   ActivityType = 0
	ActivityTypeStreaming ActivityType = 1
	ActivityTypeListening ActivityType = 2
	ActivityTypeWatching  ActivityType = 3
	ActivityTypeCustom    ActivityType = 4
	ActivityTypeCompeting ActivityType = 5
)

type ActivityTime struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

type ActivityAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

// VoiceState represents a user's voice connection state
type VoiceState struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID   *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	ChannelID  *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"`
	SessionID  string     `json:"session_id" db:"session_id"`
	Deaf       bool       `json:"deaf" db:"deaf"`
	Mute       bool       `json:"mute" db:"mute"`
	SelfDeaf   bool       `json:"self_deaf" db:"self_deaf"`
	SelfMute   bool       `json:"self_mute" db:"self_mute"`
	SelfVideo  bool       `json:"self_video" db:"self_video"`
	SelfStream bool       `json:"self_stream" db:"self_stream"`
	Suppress   bool       `json:"suppress" db:"suppress"`
	
	// Populated on fetch
	Member *Member `json:"member,omitempty"`
}

// TypingIndicator represents a user typing in a channel
type TypingIndicator struct {
	UserID    uuid.UUID `json:"user_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	Timestamp time.Time `json:"timestamp"`
}
