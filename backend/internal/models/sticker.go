package models

import (
	"time"

	"github.com/google/uuid"
)

// StickerFormat represents the format of a sticker
type StickerFormat string

const (
	StickerFormatPNG  StickerFormat = "PNG"
	StickerFormatAPNG StickerFormat = "APNG"
	StickerFormatGIF  StickerFormat = "GIF"
)

// Sticker represents a sticker in the system
type Sticker struct {
	ID        uuid.UUID     `json:"id" db:"id"`
	ServerID  *uuid.UUID    `json:"server_id,omitempty" db:"server_id"` // nil for global stickers
	Name      string        `json:"name" db:"name"`
	Tags      []string      `json:"tags" db:"tags"`
	URL       string        `json:"url" db:"url"`
	Format    StickerFormat `json:"format" db:"format"`
	CreatedBy uuid.UUID     `json:"created_by" db:"created_by"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}

// StickerResponse is the API response for a sticker
type StickerResponse struct {
	ID        string   `json:"id"`
	ServerID  *string  `json:"guild_id,omitempty"`
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	URL       string   `json:"url"`
	Format    string   `json:"format"`
	CreatedBy string   `json:"creator_id"`
	CreatedAt string   `json:"created_at"`
}

// ToResponse converts a Sticker to StickerResponse
func (s *Sticker) ToResponse() StickerResponse {
	resp := StickerResponse{
		ID:        s.ID.String(),
		Name:      s.Name,
		Tags:      s.Tags,
		URL:       s.URL,
		Format:    string(s.Format),
		CreatedBy: s.CreatedBy.String(),
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
	if s.ServerID != nil {
		serverIDStr := s.ServerID.String()
		resp.ServerID = &serverIDStr
	}
	return resp
}

// CreateStickerRequest is the request to create a sticker
type CreateStickerRequest struct {
	Name string   `json:"name" validate:"required,min=2,max=30"`
	Tags []string `json:"tags" validate:"max=10"`
}

// UpdateStickerRequest is the request to update a sticker
type UpdateStickerRequest struct {
	Name string   `json:"name,omitempty" validate:"omitempty,min=2,max=30"`
	Tags []string `json:"tags,omitempty" validate:"max=10"`
}
