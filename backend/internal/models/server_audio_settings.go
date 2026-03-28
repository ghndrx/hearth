package models

import (
	"time"

	"github.com/google/uuid"
)

// ServerAudioSettings represents per-server audio device and volume preferences for a user
type ServerAudioSettings struct {
	UserID            uuid.UUID `json:"user_id" db:"user_id"`
	ServerID          uuid.UUID `json:"server_id" db:"server_id"`
	InputDeviceID     string    `json:"input_device_id" db:"input_device_id"`
	OutputDeviceID    string    `json:"output_device_id" db:"output_device_id"`
	InputVolume       int       `json:"input_volume" db:"input_volume"`
	OutputVolume      int       `json:"output_volume" db:"output_volume"`
	PushToTalkEnabled bool      `json:"push_to_talk_enabled" db:"push_to_talk_enabled"`
	PushToTalkKey     string    `json:"push_to_talk_key" db:"push_to_talk_key"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultServerAudioSettings returns the default audio settings for a user in a server
func DefaultServerAudioSettings(userID, serverID uuid.UUID) *ServerAudioSettings {
	return &ServerAudioSettings{
		UserID:            userID,
		ServerID:          serverID,
		InputDeviceID:     "",
		OutputDeviceID:    "",
		InputVolume:       100,
		OutputVolume:      100,
		PushToTalkEnabled: false,
		PushToTalkKey:     "",
		UpdatedAt:         time.Now(),
	}
}

// UpdateServerAudioSettingsRequest represents a request to update server audio settings
type UpdateServerAudioSettingsRequest struct {
	InputDeviceID     *string `json:"input_device_id,omitempty"`
	OutputDeviceID    *string `json:"output_device_id,omitempty"`
	InputVolume       *int    `json:"input_volume,omitempty" validate:"omitempty,min=0,max=100"`
	OutputVolume      *int    `json:"output_volume,omitempty" validate:"omitempty,min=0,max=100"`
	PushToTalkEnabled *bool   `json:"push_to_talk_enabled,omitempty"`
	PushToTalkKey     *string `json:"push_to_talk_key,omitempty"`
}
