package models

import (
	"time"

	"github.com/google/uuid"
)

// Application represents a bot application in the ecosystem
type Application struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
}
