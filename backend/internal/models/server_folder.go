package models

import (
	"time"

	"github.com/google/uuid"
)

// ServerFolder represents a user-created folder for organizing servers
type ServerFolder struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	UserID    uuid.UUID      `json:"user_id" db:"user_id"`
	ParentID  *uuid.UUID     `json:"parent_id,omitempty" db:"parent_id"`
	Name      string         `json:"name" db:"name"`
	Position  int            `json:"position" db:"position"`
	IsCollapsed bool         `json:"is_collapsed" db:"is_collapsed"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// ServerInFolder represents a server assigned to a folder
type ServerInFolder struct {
	ServerID   uuid.UUID      `json:"server_id" db:"server_id"`
	FolderID   *uuid.UUID     `json:"folder_id,omitempty" db:"folder_id"`
	UserID     uuid.UUID      `json:"user_id" db:"user_id"`
	Position   int            `json:"position" db:"position"`
	AssignedAt time.Time      `json:"assigned_at" db:"assigned_at"`
}

// ServerFolderTree is the full folder tree with nested folders and servers
type ServerFolderTree struct {
	Folders []ServerFolderTreeItem `json:"folders"`
	Servers []ServerInFolder       `json:"servers"`
}

// ServerFolderTreeItem is a folder with its children and servers
type ServerFolderTreeItem struct {
	ID         uuid.UUID               `json:"id"`
	UserID     uuid.UUID               `json:"user_id"`
	ParentID   *uuid.UUID              `json:"parent_id,omitempty"`
	Name       string                  `json:"name"`
	Position   int                     `json:"position"`
	IsCollapsed bool                   `json:"is_collapsed"`
	Depth      int                     `json:"depth"`
	Children   []ServerFolderTreeItem  `json:"children,omitempty"`
	Servers    []ServerInFolder        `json:"servers,omitempty"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

// CreateServerFolderRequest is the input for creating a folder
type CreateServerFolderRequest struct {
	Name      string     `json:"name" validate:"required,min=1,max=100"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Position  *int       `json:"position,omitempty"`
}

// UpdateServerFolderRequest is the input for updating a folder
type UpdateServerFolderRequest struct {
	Name       *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Position   *int    `json:"position,omitempty"`
	IsCollapsed *bool  `json:"is_collapsed,omitempty"`
	ParentID   *string `json:"parent_id,omitempty"`
}

// MoveServerRequest is the input for moving a single server to a folder
type MoveServerRequest struct {
	ServerID string `json:"server_id" validate:"required"`
	FolderID *string `json:"folder_id,omitempty"`
}

// MoveServersToFolderRequest is the input for moving multiple servers to a folder
type MoveServersToFolderRequest struct {
	ServerIDs []string `json:"server_ids" validate:"required,min=1"`
	FolderID  *string   `json:"folder_id,omitempty"`
}

// ServerPosition represents a server's position
type ServerPosition struct {
	ServerID string `json:"server_id"`
	Position int    `json:"position"`
}

// ReorderServersRequest is the input for reordering servers within a folder
type ReorderServersRequest struct {
	FolderID        *string           `json:"folder_id,omitempty"`
	ServerPositions []ServerPosition  `json:"server_positions" validate:"required,min=1"`
}
