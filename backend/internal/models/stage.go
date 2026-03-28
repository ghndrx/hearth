package models

import (
	"time"

	"github.com/google/uuid"
)

// StageStatus represents the current state of a stage
type StageStatus int

const (
	StageStatusScheduled StageStatus = 1 // Planned but not started
	StageStatusLive      StageStatus = 2 // Active with speakers
	StageStatusPaused    StageStatus = 3 // Temporarily paused
	StageStatusEnded     StageStatus = 4 // Completed
)

// StageRole represents a user's role within a stage
type StageRole int

const (
	StageRoleAudience  StageRole = 1 // Can only listen
	StageRoleSpeaker   StageRole = 2 // Can speak
	StageRoleModerator StageRole = 3 // Can manage speakers
	StageRoleHost      StageRole = 4 // Full stage control
)

// Stage represents a stage channel session
type Stage struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	ChannelID         uuid.UUID   `json:"channel_id" db:"channel_id"`
	Topic             string      `json:"topic" db:"topic"`
	Description       string      `json:"description" db:"description"`
	Status            StageStatus `json:"status" db:"status"`
	HostUserID        uuid.UUID   `json:"host_user_id" db:"host_user_id"`
	DiscoveryDisabled bool        `json:"discovery_disabled" db:"discovery_disabled"`
	RequestToSpeak    bool        `json:"request_to_speak" db:"request_to_speak"`
	ModeratorOnly     bool        `json:"moderator_only" db:"moderator_only"`
	MaxSpeakers       int         `json:"max_speakers" db:"max_speakers"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	StartedAt         *time.Time  `json:"started_at,omitempty" db:"started_at"`
	EndedAt           *time.Time  `json:"ended_at,omitempty" db:"ended_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

// StageParticipant represents a user's participation in a stage
type StageParticipant struct {
	StageID     uuid.UUID  `json:"stage_id" db:"stage_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Role        StageRole  `json:"role" db:"role"`
	JoinedAt    time.Time  `json:"joined_at" db:"joined_at"`
	IsMuted     bool       `json:"is_muted" db:"is_muted"`
	IsDeafened  bool       `json:"is_deafened" db:"is_deafened"`
	RequestedAt *time.Time `json:"requested_at,omitempty" db:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty" db:"approved_at"`
}

// CreateStageRequest is the input for creating/starting a stage
type CreateStageRequest struct {
	Topic             string `json:"topic" validate:"required,max=128"`
	Description       string `json:"description" validate:"omitempty,max=500"`
	DiscoveryDisabled *bool  `json:"discovery_disabled,omitempty"`
	RequestToSpeak    *bool  `json:"request_to_speak,omitempty"`
	ModeratorOnly     *bool  `json:"moderator_only,omitempty"`
	MaxSpeakers       *int   `json:"max_speakers,omitempty" validate:"omitempty,min=1,max=100"`
}

// UpdateStageRequest is the input for updating a stage
type UpdateStageRequest struct {
	Topic       *string `json:"topic,omitempty" validate:"omitempty,max=128"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// StageConfig is the input for updating stage configuration
type StageConfig struct {
	DiscoveryDisabled *bool `json:"discovery_disabled,omitempty"`
	RequestToSpeak    *bool `json:"request_to_speak,omitempty"`
	ModeratorOnly     *bool `json:"moderator_only,omitempty"`
	MaxSpeakers       *int  `json:"max_speakers,omitempty" validate:"omitempty,min=1,max=100"`
}

// StageInfo is the response for stage details
type StageInfo struct {
	ID                uuid.UUID   `json:"id"`
	ChannelID         uuid.UUID   `json:"channel_id"`
	Topic             string      `json:"topic"`
	Description       string      `json:"description"`
	Status            StageStatus `json:"status"`
	HostUserID        uuid.UUID   `json:"host_user_id"`
	DiscoveryDisabled bool        `json:"discovery_disabled"`
	RequestToSpeak    bool        `json:"request_to_speak"`
	ModeratorOnly     bool        `json:"moderator_only"`
	MaxSpeakers       int         `json:"max_speakers"`
	SpeakerCount      int         `json:"speaker_count"`
	AudienceCount     int         `json:"audience_count"`
	PendingCount      int         `json:"pending_request_count"`
	CreatedAt         time.Time   `json:"created_at"`
	StartedAt         *time.Time  `json:"started_at,omitempty"`
	EndedAt           *time.Time  `json:"ended_at,omitempty"`
}

// ParticipantInfo is the response for participant details
type ParticipantInfo struct {
	UserID            uuid.UUID  `json:"user_id"`
	Role              StageRole  `json:"role"`
	JoinedAt          time.Time  `json:"joined_at"`
	IsMuted           bool       `json:"is_muted"`
	IsDeafened        bool       `json:"is_deafened"`
	HasPendingRequest bool       `json:"has_pending_request"`
	RequestedAt       *time.Time `json:"requested_at,omitempty"`
}
