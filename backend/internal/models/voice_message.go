package models

import (
	"time"

	"github.com/google/uuid"
)

// VoiceMessage represents a voice message recording
type VoiceMessage struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ChannelID    uuid.UUID  `json:"channel_id" db:"channel_id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	FileURL      string     `json:"file_url" db:"file_url"`
	DurationMs   int        `json:"duration_ms" db:"duration_ms"`
	WaveformData []float64  `json:"waveform_data" db:"waveform_data"` // Array of amplitude values (0.0-1.0) for visualization
	Transcription *string   `json:"transcription,omitempty" db:"transcription"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	User *PublicUser `json:"user,omitempty"`
}

// VoiceMessageResponse is the API response for a voice message
type VoiceMessageResponse struct {
	ID           string    `json:"id"`
	ChannelID    string    `json:"channel_id"`
	UserID       string    `json:"user_id"`
	FileURL      string    `json:"file_url"`
	DurationMs   int       `json:"duration_ms"`
	WaveformData []float64 `json:"waveform_data"`
	Transcription *string  `json:"transcription,omitempty"`
	CreatedAt    string    `json:"created_at"`

	// Optional user info
	Username string `json:"username,omitempty"`
}

// ToResponse converts a VoiceMessage to VoiceMessageResponse
func (v *VoiceMessage) ToResponse() VoiceMessageResponse {
	resp := VoiceMessageResponse{
		ID:           v.ID.String(),
		ChannelID:    v.ChannelID.String(),
		UserID:       v.UserID.String(),
		FileURL:      v.FileURL,
		DurationMs:   v.DurationMs,
		WaveformData: v.WaveformData,
		Transcription: v.Transcription,
		CreatedAt:    v.CreatedAt.Format(time.RFC3339),
	}
	if v.User != nil {
		resp.Username = v.User.Username
	}
	return resp
}
