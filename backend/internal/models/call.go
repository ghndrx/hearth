package models

import (
	"time"

	"github.com/google/uuid"
)

// CallType represents the type of call
type CallType string

const (
	CallTypeDirect  CallType = "direct"
	CallTypeGroup   CallType = "group"
	CallTypeChannel CallType = "channel"
)

// CallStatus represents the current status of a call
type CallStatus string

const (
	CallStatusRinging CallStatus = "ringing"
	CallStatusActive  CallStatus = "active"
	CallStatusEnded   CallStatus = "ended"
)

// CallEndReason represents why a call ended
type CallEndReason string

const (
	CallEndReasonCompleted CallEndReason = "completed"
	CallEndReasonMissed    CallEndReason = "missed"
	CallEndReasonDeclined  CallEndReason = "declined"
	CallEndReasonError     CallEndReason = "error"
)

// Call represents a video/audio call
type Call struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ChannelID   uuid.UUID  `json:"channel_id" db:"channel_id"`
	ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	InitiatorID uuid.UUID  `json:"initiator_id" db:"initiator_id"`
	Type        CallType   `json:"type" db:"type"`
	Status      CallStatus `json:"status" db:"status"`
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	EndReason   string     `json:"end_reason,omitempty" db:"end_reason"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	Participants []CallParticipant `json:"participants,omitempty"`
}

// CallParticipant represents a user participating in a call
type CallParticipant struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CallID    uuid.UUID  `json:"call_id" db:"call_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
	LeftAt    *time.Time `json:"left_at,omitempty" db:"left_at"`
	IsMuted   bool       `json:"is_muted" db:"is_muted"`
	IsVideoOn bool       `json:"is_video_on" db:"is_video_on"`

	// Populated from joins
	Username    string  `json:"username,omitempty" db:"username"`
	DisplayName *string `json:"display_name,omitempty" db:"display_name"`
	Avatar      *string `json:"avatar,omitempty" db:"avatar"`
}

// CallSession tracks individual WebRTC sessions within a call
type CallSession struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	CallID         uuid.UUID  `json:"call_id" db:"call_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	SessionID      string     `json:"session_id" db:"session_id"`
	PeerID         string     `json:"peer_id" db:"peer_id"`
	ConnectedAt    time.Time  `json:"connected_at" db:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty" db:"disconnected_at"`
	ConnectionType string     `json:"connection_type" db:"connection_type"` // "peer" or "sfu"
}

// CreateCallRequest is the request body for creating a call
type CreateCallRequest struct {
	ChannelID    string `json:"channel_id"`
	ServerID     string `json:"server_id,omitempty"`
	Type         string `json:"type"`
	TargetUserID string `json:"target_user_id,omitempty"`
}

// JoinCallResponse is returned when a user joins a call
type JoinCallResponse struct {
	CallID   uuid.UUID  `json:"call_id"`
	UserID   uuid.UUID  `json:"user_id"`
	JoinedAt time.Time  `json:"joined_at"`
	ICEServers []ICEServer `json:"ice_servers"`
}

// ICEServer represents a STUN/TURN server configuration
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}
