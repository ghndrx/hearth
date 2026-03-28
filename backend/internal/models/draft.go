package models

import (
	"time"

	"github.com/google/uuid"
)

// DraftStatus represents the status of a draft
type DraftStatus string

const (
	DraftStatusDraft     DraftStatus = "draft"
	DraftStatusPublished DraftStatus = "published"
)

// Draft represents a message draft
type Draft struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	Title      string      `json:"title" db:"title"`
	Content    string      `json:"content" db:"content"`
	GuildID    uuid.UUID   `json:"guild_id" db:"guild_id"`
	ChannelID  uuid.UUID   `json:"channel_id" db:"channel_id"`
	Status     DraftStatus `json:"status" db:"status"`
	CreatedBy  uuid.UUID   `json:"created_by" db:"created_by"`
	LastEdited *time.Time  `json:"last_edited,omitempty" db:"last_edited"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
}

// CreateDraftRequest is the input for creating a draft
type CreateDraftRequest struct {
	Title     string    `json:"title" validate:"required"`
	Content   string    `json:"content"`
	GuildID   uuid.UUID `json:"guild_id" validate:"required"`
	ChannelID uuid.UUID `json:"channel_id" validate:"required"`
	CreatedBy uuid.UUID `json:"created_by" validate:"required"`
}

// UpdateDraftRequest is the input for updating a draft
type UpdateDraftRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}
