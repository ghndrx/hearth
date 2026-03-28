package models

import (
	"time"

	"github.com/google/uuid"
)

// StagePrivacyLevel represents who can see the stage
type StagePrivacyLevel int

const (
	StagePrivacyPublic    StagePrivacyLevel = 1 // Visible to everyone
	StagePrivacyGuildOnly StagePrivacyLevel = 2 // Visible to server members only
)

// StageParticipantRole represents a participant's role in the stage
type StageParticipantRole string

const (
	StageRoleSpeaker  StageParticipantRole = "speaker"
	StageRoleAudience StageParticipantRole = "audience"
)

// StageInstance represents an active stage session
type StageInstance struct {
	ID            uuid.UUID         `json:"id" db:"id"`
	ChannelID     uuid.UUID         `json:"channel_id" db:"channel_id"`
	ServerID      uuid.UUID         `json:"server_id" db:"server_id"`
	Topic         string            `json:"topic" db:"topic"`
	PrivacyLevel  StagePrivacyLevel `json:"privacy_level" db:"privacy_level"`
	StartedBy     uuid.UUID         `json:"started_by" db:"started_by"`
	SpeakerCount  int               `json:"speaker_count" db:"speaker_count"`
	AudienceCount int               `json:"audience_count" db:"audience_count"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	EndedAt       *time.Time        `json:"ended_at,omitempty" db:"ended_at"`

	// Populated from joins
	Speakers []StageParticipant `json:"speakers,omitempty"`
	Audience []StageParticipant `json:"audience,omitempty"`
}

// StageParticipant represents a user in a stage
type StageParticipant struct {
	StageID  uuid.UUID            `json:"stage_id" db:"stage_id"`
	UserID   uuid.UUID            `json:"user_id" db:"user_id"`
	Role     StageParticipantRole `json:"role" db:"role"`
	JoinedAt time.Time            `json:"joined_at" db:"joined_at"`

	// Populated from joins
	User *PublicUser `json:"user,omitempty"`
}

// CreateStageRequest is the input for starting a stage
type CreateStageRequest struct {
	Topic        string            `json:"topic" validate:"required,min=1,max=120"`
	PrivacyLevel StagePrivacyLevel `json:"privacy_level" validate:"omitempty,oneof=1 2"`
}

// UpdateStageRequest is the input for updating a stage
type UpdateStageRequest struct {
	Topic        *string            `json:"topic,omitempty" validate:"omitempty,min=1,max=120"`
	PrivacyLevel *StagePrivacyLevel `json:"privacy_level,omitempty" validate:"omitempty,oneof=1 2"`
}

// StageParticipantUpdateRequest is the input for updating a participant's role
type StageParticipantUpdateRequest struct {
	Role StageParticipantRole `json:"role" validate:"required,oneof=speaker audience"`
}

// WebSocket event payloads

// StageInstanceCreateEvent is sent when a stage starts
type StageInstanceCreateEvent struct {
	StageInstance *StageInstance `json:"stage_instance"`
	ChannelID     uuid.UUID     `json:"channel_id"`
	ServerID      uuid.UUID     `json:"server_id"`
}

// StageInstanceDeleteEvent is sent when a stage ends
type StageInstanceDeleteEvent struct {
	StageID   uuid.UUID `json:"stage_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
}

// StageParticipantEvent is sent when a participant joins, leaves, or changes role
type StageParticipantEvent struct {
	StageID   uuid.UUID            `json:"stage_id"`
	UserID    uuid.UUID            `json:"user_id"`
	Role      StageParticipantRole `json:"role"`
	ChannelID uuid.UUID            `json:"channel_id"`
	ServerID  uuid.UUID            `json:"server_id"`
}
