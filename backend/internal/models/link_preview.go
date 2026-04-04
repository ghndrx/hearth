package models

import (
	"time"

	"github.com/google/uuid"
)

// LinkPreview represents cached OpenGraph/metadata for a URL
type LinkPreview struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	URL         string     `json:"url" db:"url"`
	Title       *string    `json:"title,omitempty" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	ImageURL    *string    `json:"image_url,omitempty" db:"image_url"`
	VideoURL    *string    `json:"video_url,omitempty" db:"video_url"`
	SiteName    *string    `json:"site_name,omitempty" db:"site_name"`
	Type        string     `json:"type" db:"type"` // website, video, image, rich, audio
	Width       *int       `json:"width,omitempty" db:"width"`
	Height      *int       `json:"height,omitempty" db:"height"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// MessageLinkPreview associates a link preview with a message
type MessageLinkPreview struct {
	MessageID     uuid.UUID `json:"message_id" db:"message_id"`
	LinkPreviewID uuid.UUID `json:"link_preview_id" db:"link_preview_id"`
}

// CreateLinkPreviewRequest is the input for creating a link preview
type CreateLinkPreviewRequest struct {
	URL string `json:"url" validate:"required,url"`
}

// LinkPreviewResponse is the API response for a link preview
type LinkPreviewResponse struct {
	ID          uuid.UUID  `json:"id"`
	URL         string     `json:"url"`
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	ImageURL    *string     `json:"image_url,omitempty"`
	VideoURL    *string     `json:"video_url,omitempty"`
	SiteName    *string     `json:"site_name,omitempty"`
	Type        string      `json:"type"`
	Width       *int        `json:"width,omitempty"`
	Height      *int        `json:"height,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ToResponse converts a LinkPreview to its API response format
func (lp *LinkPreview) ToResponse() *LinkPreviewResponse {
	return &LinkPreviewResponse{
		ID:          lp.ID,
		URL:         lp.URL,
		Title:       lp.Title,
		Description: lp.Description,
		ImageURL:    lp.ImageURL,
		VideoURL:    lp.VideoURL,
		SiteName:    lp.SiteName,
		Type:        lp.Type,
		Width:       lp.Width,
		Height:      lp.Height,
		ExpiresAt:   lp.ExpiresAt,
	}
}
