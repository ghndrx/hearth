package models

import (
	"time"

	"github.com/google/uuid"
)

// EmbedTemplate represents a saved embed template for reuse
type EmbedTemplate struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Title       *string   `json:"title,omitempty" db:"title"`
	Description *string   `json:"description,omitempty" db:"description"`
	URL         *string   `json:"url,omitempty" db:"url"`
	Color       *int      `json:"color,omitempty" db:"color"`
	AuthorName  *string   `json:"author_name,omitempty" db:"author_name"`
	AuthorURL   *string   `json:"author_url,omitempty" db:"author_url"`
	AuthorIcon  *string   `json:"author_icon,omitempty" db:"author_icon"`
	FooterText  *string   `json:"footer_text,omitempty" db:"footer_text"`
	FooterIcon  *string   `json:"footer_icon,omitempty" db:"footer_icon"`
	ImageURL    *string   `json:"image_url,omitempty" db:"image_url"`
	ThumbnailURL *string  `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateEmbedTemplateRequest is the input for creating an embed template
type CreateEmbedTemplateRequest struct {
	Name         string  `json:"name" validate:"required,min=1,max=100"`
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	URL          *string `json:"url,omitempty"`
	Color        *int    `json:"color,omitempty"`
	AuthorName   *string `json:"author_name,omitempty"`
	AuthorURL    *string `json:"author_url,omitempty"`
	AuthorIcon   *string `json:"author_icon,omitempty"`
	FooterText   *string `json:"footer_text,omitempty"`
	FooterIcon   *string `json:"footer_icon,omitempty"`
	ImageURL     *string `json:"image_url,omitempty"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
}

// UpdateEmbedTemplateRequest is the input for updating an embed template
type UpdateEmbedTemplateRequest struct {
	Name         *string `json:"name,omitempty"`
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	URL          *string `json:"url,omitempty"`
	Color        *int    `json:"color,omitempty"`
	AuthorName   *string `json:"author_name,omitempty"`
	AuthorURL    *string `json:"author_url,omitempty"`
	AuthorIcon   *string `json:"author_icon,omitempty"`
	FooterText   *string `json:"footer_text,omitempty"`
	FooterIcon   *string `json:"footer_icon,omitempty"`
	ImageURL     *string `json:"image_url,omitempty"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
}

// EmbedTemplateResponse is the API response for an embed template
type EmbedTemplateResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Title        *string   `json:"title,omitempty"`
	Description  *string   `json:"description,omitempty"`
	URL          *string   `json:"url,omitempty"`
	Color        *int      `json:"color,omitempty"`
	Author       *struct {
		Name *string `json:"name,omitempty"`
		URL  *string `json:"url,omitempty"`
		Icon *string `json:"icon,omitempty"`
	} `json:"author,omitempty"`
	Footer       *struct {
		Text *string `json:"text,omitempty"`
		Icon *string `json:"icon,omitempty"`
	} `json:"footer,omitempty"`
	ImageURL     *string   `json:"image_url,omitempty"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToTemplateResponse converts an EmbedTemplate to its API response format
func (et *EmbedTemplate) ToTemplateResponse() *EmbedTemplateResponse {
	resp := &EmbedTemplateResponse{
		ID:          et.ID,
		UserID:      et.UserID,
		Name:        et.Name,
		Title:       et.Title,
		Description: et.Description,
		URL:         et.URL,
		Color:       et.Color,
		ImageURL:    et.ImageURL,
		ThumbnailURL: et.ThumbnailURL,
		CreatedAt:   et.CreatedAt,
		UpdatedAt:   et.UpdatedAt,
	}

	if et.AuthorName != nil || et.AuthorURL != nil || et.AuthorIcon != nil {
		resp.Author = &struct {
			Name *string `json:"name,omitempty"`
			URL  *string `json:"url,omitempty"`
			Icon *string `json:"icon,omitempty"`
		}{
			Name: et.AuthorName,
			URL:  et.AuthorURL,
			Icon: et.AuthorIcon,
		}
	}

	if et.FooterText != nil || et.FooterIcon != nil {
		resp.Footer = &struct {
			Text *string `json:"text,omitempty"`
			Icon *string `json:"icon,omitempty"`
		}{
			Text: et.FooterText,
			Icon: et.FooterIcon,
		}
	}

	return resp
}
