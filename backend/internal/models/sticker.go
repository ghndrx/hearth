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

// StickerPackTier represents the tier of a sticker pack (maps to subscription tiers)
type StickerPackTier string

const (
	StickerPackTierFree    StickerPackTier = "free"
	StickerPackTierBasic   StickerPackTier = "basic"
	StickerPackTierPremium StickerPackTier = "premium"
)

// Sticker represents a sticker in the system
type Sticker struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ServerID     *uuid.UUID      `json:"server_id,omitempty" db:"server_id"` // nil for global stickers
	Name         string          `json:"name" db:"name"`
	Tags         []string        `json:"tags" db:"tags"`
	URL          string          `json:"url" db:"url"`
	Format       StickerFormat   `json:"format" db:"format"`
	RequiredTier StickerPackTier `json:"required_tier" db:"required_tier"` // Minimum tier to use this sticker
	CreatedBy    uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// StickerPack represents a collection of stickers
type StickerPack struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Name         string          `json:"name" db:"name"`
	Description  *string         `json:"description,omitempty" db:"description"`
	IconURL      *string         `json:"icon_url,omitempty" db:"icon_url"`
	Tier         StickerPackTier `json:"tier" db:"tier"`
	StickerCount int             `json:"sticker_count" db:"sticker_count"`
	IsActive     bool            `json:"is_active" db:"is_active"`
	IsGlobal     bool            `json:"is_global" db:"is_global"`
	ServerID     *uuid.UUID      `json:"server_id,omitempty" db:"server_id"`
	CreatedBy    *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	Stickers     []*Sticker      `json:"stickers,omitempty"` // Populated when fetching pack with stickers
}

// PackSticker represents the many-to-many relationship between packs and stickers
type PackSticker struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PackID    uuid.UUID `json:"pack_id" db:"pack_id"`
	StickerID uuid.UUID `json:"sticker_id" db:"sticker_id"`
	Position  int       `json:"position" db:"position"`
	IsDefault bool      `json:"is_default" db:"is_default"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// StickerResponse is the API response for a sticker
type StickerResponse struct {
	ID           string   `json:"id"`
	ServerID     *string  `json:"guild_id,omitempty"`
	Name         string   `json:"name"`
	Tags         []string `json:"tags"`
	URL          string   `json:"url"`
	Format       string   `json:"format"`
	RequiredTier string   `json:"required_tier"`
	CreatedBy    string   `json:"creator_id"`
	CreatedAt    string   `json:"created_at"`
}

// StickerPackResponse is the API response for a sticker pack
type StickerPackResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  *string           `json:"description,omitempty"`
	IconURL      *string           `json:"icon_url,omitempty"`
	Tier         string            `json:"tier"`
	StickerCount int               `json:"sticker_count"`
	IsActive     bool              `json:"is_active"`
	IsGlobal     bool              `json:"is_global"`
	ServerID     *string           `json:"guild_id,omitempty"`
	CreatedBy    *string           `json:"creator_id,omitempty"`
	CreatedAt    string            `json:"created_at"`
	Stickers     []StickerResponse `json:"stickers,omitempty"`
}

// ToResponse converts a Sticker to StickerResponse
func (s *Sticker) ToResponse() StickerResponse {
	resp := StickerResponse{
		ID:           s.ID.String(),
		Name:         s.Name,
		Tags:         s.Tags,
		URL:          s.URL,
		Format:       string(s.Format),
		RequiredTier: string(s.RequiredTier),
		CreatedBy:    s.CreatedBy.String(),
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
	}
	if s.ServerID != nil {
		serverIDStr := s.ServerID.String()
		resp.ServerID = &serverIDStr
	}
	return resp
}

// ToPackResponse converts a StickerPack to StickerPackResponse
func (p *StickerPack) ToPackResponse() StickerPackResponse {
	resp := StickerPackResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Description:  p.Description,
		IconURL:      p.IconURL,
		Tier:         string(p.Tier),
		StickerCount: p.StickerCount,
		IsActive:     p.IsActive,
		IsGlobal:     p.IsGlobal,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		Stickers:     []StickerResponse{},
	}
	if p.ServerID != nil {
		serverIDStr := p.ServerID.String()
		resp.ServerID = &serverIDStr
	}
	if p.CreatedBy != nil {
		createdByStr := p.CreatedBy.String()
		resp.CreatedBy = &createdByStr
	}
	for _, s := range p.Stickers {
		resp.Stickers = append(resp.Stickers, s.ToResponse())
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

// CreateStickerPackRequest is the request to create a sticker pack
type CreateStickerPackRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Description *string `json:"description,omitempty" validate:"max=500"`
	IconURL     *string `json:"icon_url,omitempty"`
	Tier        string  `json:"tier" validate:"required,oneof=free basic premium"`
	IsGlobal    *bool   `json:"is_global,omitempty"`
}

// UpdateStickerPackRequest is the request to update a sticker pack
type UpdateStickerPackRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description,omitempty" validate:"max=500"`
	IconURL     *string `json:"icon_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// AddStickerToPackRequest is the request to add a sticker to a pack
type AddStickerToPackRequest struct {
	StickerID uuid.UUID `json:"sticker_id" validate:"required"`
	Position  int       `json:"position,omitempty"`
	IsDefault bool      `json:"is_default,omitempty"`
}

// StickerTierFromString converts a string to StickerPackTier
func StickerTierFromString(s string) StickerPackTier {
	switch s {
	case "basic":
		return StickerPackTierBasic
	case "premium":
		return StickerPackTierPremium
	default:
		return StickerPackTierFree
	}
}

// TierMeetsRequirement checks if a user's tier meets the required tier for a sticker/pack
func TierMeetsRequirement(userTier, requiredTier StickerPackTier) bool {
	tierOrder := map[StickerPackTier]int{
		StickerPackTierFree:    0,
		StickerPackTierBasic:   1,
		StickerPackTierPremium: 2,
	}
	return tierOrder[userTier] >= tierOrder[requiredTier]
}
