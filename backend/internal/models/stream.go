package models

import (
	"time"

	"github.com/google/uuid"
)

// LiveStreamType represents the type of live stream
type LiveStreamType int

const (
	LiveStreamTypeScreen      LiveStreamType = 1 // Screen share
	LiveStreamTypeApplication LiveStreamType = 2 // Application window
	LiveStreamTypeCamera      LiveStreamType = 3 // Camera feed
)

// LiveStreamQuality represents the quality level of a live stream
type LiveStreamQuality int

const (
	LiveStreamQuality480p  LiveStreamQuality = 1
	LiveStreamQuality720p  LiveStreamQuality = 2
	LiveStreamQuality1080p LiveStreamQuality = 3
)

// LiveStreamStatus represents the status of a live stream
type LiveStreamStatus int

const (
	LiveStreamStatusActive LiveStreamStatus = 1 // Stream is live
	LiveStreamStatusEnded  LiveStreamStatus = 2 // Stream has ended
)

// LiveStream represents a live stream session in a voice channel
type LiveStream struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	ChannelID   uuid.UUID         `json:"channel_id" db:"channel_id"`
	ServerID    uuid.UUID         `json:"server_id" db:"server_id"`
	StreamerID  uuid.UUID         `json:"streamer_id" db:"streamer_id"`
	Type        LiveStreamType    `json:"type" db:"type"`
	Quality     LiveStreamQuality `json:"quality" db:"quality"`
	Status      LiveStreamStatus  `json:"status" db:"status"`
	ViewerCount int               `json:"viewer_count" db:"viewer_count"`
	Viewers     []uuid.UUID       `json:"viewers" db:"viewers"`
	StartedAt   time.Time         `json:"started_at" db:"started_at"`
	EndedAt     *time.Time        `json:"ended_at,omitempty" db:"ended_at"`
}

// StartLiveStreamRequest is the input for starting a live stream
type StartLiveStreamRequest struct {
	StreamType LiveStreamType    `json:"stream_type" validate:"required,oneof=1 2 3"`
	Quality    LiveStreamQuality `json:"quality" validate:"omitempty,oneof=1 2 3"`
}

// LiveStreamInfo is the response for stream details
type LiveStreamInfo struct {
	ID          uuid.UUID         `json:"id"`
	ChannelID   uuid.UUID         `json:"channel_id"`
	ServerID    uuid.UUID         `json:"server_id"`
	StreamerID  uuid.UUID         `json:"streamer_id"`
	Streamer    *User             `json:"streamer,omitempty"`
	Type        LiveStreamType    `json:"type"`
	Quality     LiveStreamQuality `json:"quality"`
	Status      LiveStreamStatus  `json:"status"`
	ViewerCount int               `json:"viewer_count"`
	Viewers     []uuid.UUID       `json:"viewers"`
	StartedAt   time.Time         `json:"started_at"`
	EndedAt     *time.Time        `json:"ended_at,omitempty"`
}

// LiveStreamSettingsUpdate is used for updating stream settings
type LiveStreamSettingsUpdate struct {
	Quality *LiveStreamQuality `json:"quality,omitempty"`
}

// LiveStreamStartEvent is sent when a stream starts
type LiveStreamStartEvent struct {
	Stream    *LiveStreamInfo `json:"stream"`
	ChannelID uuid.UUID       `json:"channel_id"`
	ServerID  uuid.UUID       `json:"server_id"`
}

// LiveStreamEndEvent is sent when a stream ends
type LiveStreamEndEvent struct {
	StreamID  uuid.UUID `json:"stream_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
	UserID    uuid.UUID `json:"user_id"`
}

// LiveStreamViewerJoinEvent is sent when a viewer joins a stream
type LiveStreamViewerJoinEvent struct {
	StreamID    uuid.UUID   `json:"stream_id"`
	UserID      uuid.UUID   `json:"user_id"`
	ViewerCount int         `json:"viewer_count"`
	Viewers     []uuid.UUID `json:"viewers"`
	ChannelID   uuid.UUID   `json:"channel_id"`
	ServerID    uuid.UUID   `json:"server_id"`
}

// LiveStreamViewerLeaveEvent is sent when a viewer leaves a stream
type LiveStreamViewerLeaveEvent struct {
	StreamID    uuid.UUID   `json:"stream_id"`
	UserID      uuid.UUID   `json:"user_id"`
	ViewerCount int         `json:"viewer_count"`
	Viewers     []uuid.UUID `json:"viewers"`
	ChannelID   uuid.UUID   `json:"channel_id"`
	ServerID    uuid.UUID   `json:"server_id"`
}
