package models

import (
	"time"

	"github.com/google/uuid"
)

// SvelteComponent represents a Svelte component stored in the system
type SvelteComponent struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Content   string    `json:"content" db:"content"`
	AuthorID  uuid.UUID `json:"author_id" db:"author_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
