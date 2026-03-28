package models

import (
	"time"

	"github.com/google/uuid"
)

// StreamType represents the type of screen/application stream
type StreamType int

const (
	StreamTypeScreen      StreamType = 1 // Screen share
	StreamTypeApplication StreamType = 2 // Application window
)

// StreamStatus represents the status of a stream session
type StreamStatus int

const (
	StreamStatusActive StreamStatus = 1 // Stream is live
	StreamStatusEnded  StreamStatus = 2 // Stream has ended
)

// StreamSession represents an active screen share or application stream
type StreamSession struct {
	ID         uuid.UUID    `json:"id" db:"id"`
	ServerID   uuid.UUID    `json:"server_id" db:"server_id"`
	ChannelID  uuid.UUID    `json:"channel_id" db:"channel_id"`
	UserID     uuid.UUID    `json:"user_id" db:"user_id"`
	StreamType StreamType   `json:"stream_type" db:"stream_type"`
	Status     StreamStatus `json:"status" db:"status"`
	Resolution string       `json:"resolution" db:"resolution"`
	FrameRate  int          `json:"frame_rate" db:"frame_rate"`
	StartedAt  time.Time    `json:"started_at" db:"started_at"`
	EndedAt    *time.Time   `json:"ended_at,omitempty" db:"ended_at"`
}

// StreamViewer represents a user viewing a stream
type StreamViewer struct {
	SessionID uuid.UUID `json:"session_id" db:"session_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	JoinedAt  time.Time `json:"joined_at" db:"joined_at"`
}

// StartStreamRequest is the input for starting a stream
type StartStreamRequest struct {
	StreamType StreamType `json:"stream_type" validate:"required,oneof=1 2"`
	Resolution string     `json:"resolution" validate:"omitempty,oneof=720p 1080p 1440p"`
	FrameRate  int        `json:"frame_rate" validate:"omitempty,oneof=30 60"`
}

// StreamInfo is the response for stream details
type StreamInfo struct {
	ID          uuid.UUID    `json:"id"`
	ServerID    uuid.UUID    `json:"server_id"`
	ChannelID   uuid.UUID    `json:"channel_id"`
	UserID      uuid.UUID    `json:"user_id"`
	StreamType  StreamType   `json:"stream_type"`
	Status      StreamStatus `json:"status"`
	Resolution  string       `json:"resolution"`
	FrameRate   int          `json:"frame_rate"`
	ViewerCount int          `json:"viewer_count"`
	StartedAt   time.Time    `json:"started_at"`
	EndedAt     *time.Time   `json:"ended_at,omitempty"`
}

// StreamUpdate is used for updating stream settings
type StreamUpdate struct {
	Resolution *string `json:"resolution,omitempty"`
	FrameRate  *int    `json:"frame_rate,omitempty"`
}
