package models

import (
	"time"

	"github.com/google/uuid"
)

// SavedMessage represents a bookmarked/saved message
type SavedMessage struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	Note      *string   `json:"note,omitempty" db:"note"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Populated from joins
	Message *Message `json:"message,omitempty"`
}

// SaveMessageRequest is the input for saving/bookmarking a message
type SaveMessageRequest struct {
	MessageID string  `json:"message_id" validate:"required,uuid"`
	Note      *string `json:"note,omitempty" validate:"omitempty,max=500"`
}

// UpdateSavedMessageRequest is the input for updating a saved message note
type UpdateSavedMessageRequest struct {
	Note *string `json:"note,omitempty" validate:"omitempty,max=500"`
}

// SavedMessagesQueryOptions for fetching saved messages
type SavedMessagesQueryOptions struct {
	Before *uuid.UUID
	After  *uuid.UUID
	Limit  int // max 100, default 50
}
