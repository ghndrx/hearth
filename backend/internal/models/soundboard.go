package models

import (
	"time"

	"github.com/google/uuid"
)

// SoundboardSound represents a soundboard sound clip
type SoundboardSound struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ServerID   *uuid.UUID `json:"server_id,omitempty" db:"server_id"` // nil for default sounds
	Name       string     `json:"name" db:"name"`
	EmojiName  string     `json:"emoji_name,omitempty" db:"emoji_name"`
	Volume     float64    `json:"volume" db:"volume"`
	AudioURL   string     `json:"audio_url" db:"audio_url"`
	DurationMs int        `json:"duration_ms" db:"duration_ms"`
	Available  bool       `json:"available" db:"available"`
	CreatorID  uuid.UUID  `json:"creator_id" db:"creator_id"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// SoundboardSoundResponse is the API response for a soundboard sound
type SoundboardSoundResponse struct {
	ID         string  `json:"id"`
	ServerID   *string `json:"guild_id,omitempty"`
	Name       string  `json:"name"`
	EmojiName  string  `json:"emoji_name,omitempty"`
	Volume     float64 `json:"volume"`
	AudioURL   string  `json:"audio_url"`
	DurationMs int     `json:"duration_ms"`
	Available  bool    `json:"available"`
	CreatorID  string  `json:"creator_id"`
	CreatedAt  string  `json:"created_at"`
}

// ToResponse converts a SoundboardSound to SoundboardSoundResponse
func (s *SoundboardSound) ToResponse() SoundboardSoundResponse {
	resp := SoundboardSoundResponse{
		ID:         s.ID.String(),
		Name:       s.Name,
		EmojiName:  s.EmojiName,
		Volume:     s.Volume,
		AudioURL:   s.AudioURL,
		DurationMs: s.DurationMs,
		Available:  s.Available,
		CreatorID:  s.CreatorID.String(),
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
	}
	if s.ServerID != nil {
		serverIDStr := s.ServerID.String()
		resp.ServerID = &serverIDStr
	}
	return resp
}

// CreateSoundboardSoundRequest is the request to create a soundboard sound
type CreateSoundboardSoundRequest struct {
	Name      string  `json:"name" validate:"required,min=2,max=100"`
	EmojiName string  `json:"emoji_name,omitempty" validate:"max=100"`
	Volume    float64 `json:"volume,omitempty" validate:"gte=0,lte=1"`
}

// UpdateSoundboardSoundRequest is the request to update a soundboard sound
type UpdateSoundboardSoundRequest struct {
	Name      string   `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	EmojiName string   `json:"emoji_name,omitempty" validate:"max=100"`
	Volume    *float64 `json:"volume,omitempty" validate:"omitempty,gte=0,lte=1"`
	Available *bool    `json:"available,omitempty"`
}

// SoundboardPlayEvent is the WebSocket event sent when a sound is played
type SoundboardPlayEvent struct {
	SoundID    string  `json:"sound_id"`
	SoundName  string  `json:"sound_name"`
	EmojiName  string  `json:"emoji_name,omitempty"`
	AudioURL   string  `json:"audio_url"`
	Volume     float64 `json:"volume"`
	DurationMs int     `json:"duration_ms"`
	UserID     string  `json:"user_id"`
	ChannelID  string  `json:"channel_id"`
	ServerID   string  `json:"server_id"`
}

// SoundboardSoundPack represents a collection of soundboard sounds
type SoundboardSoundPack struct {
	ID        uuid.UUID          `json:"id" db:"id"`
	ServerID  *uuid.UUID         `json:"server_id,omitempty" db:"server_id"`
	Name      string             `json:"name" db:"name"`
	EmojiName string             `json:"emoji_name,omitempty" db:"emoji_name"`
	Sounds    []*SoundboardSound `json:"sounds,omitempty" db:"-"`
	IsDefault bool               `json:"is_default" db:"is_default"`
	Position  int                `json:"position" db:"position"`
	CreatedAt time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" db:"updated_at"`
}

// SoundboardSoundPackResponse is the API response for a soundboard pack
type SoundboardSoundPackResponse struct {
	ID        string                      `json:"id"`
	ServerID  *string                     `json:"guild_id,omitempty"`
	Name      string                      `json:"name"`
	EmojiName string                      `json:"emoji_name,omitempty"`
	Sounds    []SoundboardSoundResponse   `json:"sounds"`
	IsDefault bool                        `json:"is_default"`
	Position  int                         `json:"position"`
	CreatedAt string                      `json:"created_at"`
	UpdatedAt string                      `json:"updated_at"`
}

// ToResponse converts a SoundboardSoundPack to SoundboardSoundPackResponse
func (p *SoundboardSoundPack) ToResponse() SoundboardSoundPackResponse {
	resp := SoundboardSoundPackResponse{
		ID:        p.ID.String(),
		Name:      p.Name,
		EmojiName: p.EmojiName,
		IsDefault: p.IsDefault,
		Position:  p.Position,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
		Sounds:    make([]SoundboardSoundResponse, 0, len(p.Sounds)),
	}
	if p.ServerID != nil {
		serverIDStr := p.ServerID.String()
		resp.ServerID = &serverIDStr
	}
	for _, sound := range p.Sounds {
		resp.Sounds = append(resp.Sounds, sound.ToResponse())
	}
	return resp
}

// SoundboardPlayingSound represents a sound currently being played
type SoundboardPlayingSound struct {
	SoundID    uuid.UUID `json:"sound_id"`
	SoundName  string    `json:"sound_name"`
	EmojiName  string    `json:"emoji_name,omitempty"`
	AudioURL   string    `json:"audio_url"`
	Volume     float64   `json:"volume"`
	DurationMs int       `json:"duration_ms"`
	StartedAt  time.Time `json:"started_at"`
	PlayedBy   uuid.UUID `json:"played_by"`
}
