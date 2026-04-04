package models

import (
	"time"

	"github.com/google/uuid"
)

// ServerFolder represents a user-created folder for organizing servers
type ServerFolder struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Name      string    `json:"name" db:"name"`
	Position  int       `json:"position" db:"position"`
	IsCollapsed bool    `json:"is_collapsed" db:"is_collapsed"`
	Depth     int       `json:"depth" db:"depth"` // 0 = root, 1 = nested once, 2 = nested twice (max 3 levels)
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Populated fields
	Children []*ServerFolder    `json:"children,omitempty"`
	Servers  []*ServerInFolder  `json:"servers,omitempty"`
}

// ServerInFolder represents a server's assignment to a folder
type ServerInFolder struct {
	ServerID   uuid.UUID `json:"server_id" db:"server_id"`
	FolderID   *uuid.UUID `json:"folder_id,omitempty" db:"folder_id"`
	Position   int       `json:"position" db:"position"`
	AssignedAt time.Time `json:"assigned_at" db:"assigned_at"`

	// Populated from joins
	Server *Server `json:"server,omitempty"`
}

// CreateServerFolderRequest is the input for creating a server folder
type CreateServerFolderRequest struct {
	Name     string     `json:"name" validate:"required,min=1,max=100"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Position *int       `json:"position,omitempty"`
}

// UpdateServerFolderRequest is the input for updating a server folder
type UpdateServerFolderRequest struct {
	Name       *string    `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Position   *int       `json:"position,omitempty"`
	IsCollapsed *bool    `json:"is_collapsed,omitempty"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
}

// ReorderServersRequest is the input for reordering servers within a folder
type ReorderServersRequest struct {
	ServerPositions []ServerPosition `json:"server_positions" validate:"required,min=1,dive"`
}

// ServerPosition represents a server's position in a folder
type ServerPosition struct {
	ServerID uuid.UUID `json:"server_id" validate:"required"`
	Position int       `json:"position" validate:"required"`
}

// MoveServersToFolderRequest is the input for moving servers to a folder
type MoveServersToFolderRequest struct {
	ServerIDs []uuid.UUID `json:"server_ids" validate:"required,min=1,dive"`
	FolderID  *uuid.UUID  `json:"folder_id,omitempty"` // nil means remove from folder
}

// ServerFolderTree represents the full folder tree for a user
type ServerFolderTree struct {
	Folders []*ServerFolder      `json:"folders"`
	Servers []*ServerInFolder    `json:"servers"` // Servers not in any folder
}

// MaxNestingDepth is the maximum folder nesting depth (3 levels: 0, 1, 2)
const MaxNestingDepth = 2
