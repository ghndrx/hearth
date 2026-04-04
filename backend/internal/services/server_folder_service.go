package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrFolderNotFound = errors.New("folder not found")
	ErrFolderNameRequired = errors.New("folder name is required")
)

// ServerFolderRepositoryInterface defines the methods needed from the repository
type ServerFolderRepositoryInterface interface {
	Create(ctx context.Context, folder *models.ServerFolder) error
	GetByID(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error)
	GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error)
	Update(ctx context.Context, folder *models.ServerFolder) error
	Delete(ctx context.Context, userID, folderID uuid.UUID) error
	GetServersInFolder(ctx context.Context, userID, folderID uuid.UUID) ([]*models.ServerInFolder, error)
	GetUnassignedServers(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error)
	GetAllServerAssignments(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error)
	AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error
	AssignServersToFolder(ctx context.Context, userID uuid.UUID, serverIDs []uuid.UUID, folderID *uuid.UUID) error
	UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error
	GetServerAssignment(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error)
	UserIsMemberOfServer(ctx context.Context, userID, serverID uuid.UUID) (bool, error)
}

// ServerFolderService handles business logic for server folders
type ServerFolderService struct {
	repo ServerFolderRepositoryInterface
}

// NewServerFolderService creates a new server folder service
func NewServerFolderService(repo ServerFolderRepositoryInterface) *ServerFolderService {
	return &ServerFolderService{repo: repo}
}

// GetUserFolders returns the full folder tree for a user
func (s *ServerFolderService) GetUserFolders(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error) {
	folders, err := s.repo.GetAllForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build folder lookup map
	folderMap := make(map[uuid.UUID]*models.ServerFolderTreeItem)
	for _, f := range folders {
		folderMap[f.ID] = &models.ServerFolderTreeItem{
			ID:          f.ID,
			UserID:      f.UserID,
			ParentID:    f.ParentID,
			Name:        f.Name,
			Position:    f.Position,
			IsCollapsed: f.IsCollapsed,
			Depth:       0,
			Children:    []models.ServerFolderTreeItem{},
			Servers:     []models.ServerInFolder{},
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
	}

	// Get all server assignments
	allAssignments, err := s.repo.GetAllServerAssignments(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build assignment map by folder
	assignmentsByFolder := make(map[string][]models.ServerInFolder)
	var unassigned []models.ServerInFolder
	for _, a := range allAssignments {
		if a.FolderID != nil {
			key := a.FolderID.String()
			assignmentsByFolder[key] = append(assignmentsByFolder[key], *a)
		} else {
			unassigned = append(unassigned, *a)
		}
	}

	// Build tree structure
	// First pass: build all items with their server assignments
	var rootFolders []*models.ServerFolderTreeItem
	for _, f := range folders {
		item := folderMap[f.ID]
		// Add servers to this folder
		if servers, ok := assignmentsByFolder[f.ID.String()]; ok {
			item.Servers = servers
		}
		// Add children - for items with parents, add to parent's children
		// For root items, add to rootFolders
		if f.ParentID != nil {
			if parent, ok := folderMap[*f.ParentID]; ok {
				parent.Children = append(parent.Children, *item)
			} else {
				rootFolders = append(rootFolders, item)
			}
		} else {
			rootFolders = append(rootFolders, item)
		}
	}

	// Convert rootFolders to value type for return
	rootFoldersVals := make([]models.ServerFolderTreeItem, len(rootFolders))
	for i, rf := range rootFolders {
		rootFoldersVals[i] = *rf
	}

	// Calculate depth recursively
	var setDepth func(items []models.ServerFolderTreeItem, depth int)
	setDepth = func(items []models.ServerFolderTreeItem, depth int) {
		for i := range items {
			items[i].Depth = depth
			if len(items[i].Children) > 0 {
				setDepth(items[i].Children, depth+1)
			}
		}
	}
	setDepth(rootFoldersVals, 0)

	return &models.ServerFolderTree{
		Folders: rootFoldersVals,
		Servers: unassigned,
	}, nil
}

// GetFolder returns a single folder by ID
func (s *ServerFolderService) GetFolder(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	folder, err := s.repo.GetByID(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	return folder, nil
}

// CreateFolder creates a new folder
func (s *ServerFolderService) CreateFolder(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error) {
	if req.Name == "" {
		return nil, ErrFolderNameRequired
	}

	now := time.Now()
	folder := &models.ServerFolder{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Position:    0,
		IsCollapsed: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, errors.New("invalid parent_id")
		}
		folder.ParentID = &parentID
	}

	if req.Position != nil {
		folder.Position = *req.Position
	}

	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

// UpdateFolder updates a folder
func (s *ServerFolderService) UpdateFolder(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
	folder, err := s.repo.GetByID(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	if req.Name != nil {
		folder.Name = *req.Name
	}
	if req.Position != nil {
		folder.Position = *req.Position
	}
	if req.IsCollapsed != nil {
		folder.IsCollapsed = *req.IsCollapsed
	}
	if req.ParentID != nil {
		if *req.ParentID == "" {
			folder.ParentID = nil
		} else {
			parentID, err := uuid.Parse(*req.ParentID)
			if err != nil {
				return nil, errors.New("invalid parent_id")
			}
			folder.ParentID = &parentID
		}
	}
	folder.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

// DeleteFolder deletes a folder
func (s *ServerFolderService) DeleteFolder(ctx context.Context, userID, folderID uuid.UUID) error {
	folder, err := s.repo.GetByID(ctx, userID, folderID)
	if err != nil {
		return err
	}
	if folder == nil {
		return ErrFolderNotFound
	}

	return s.repo.Delete(ctx, userID, folderID)
}

// MoveServerToFolder moves a single server to a folder
func (s *ServerFolderService) MoveServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	// Verify user is member of server
	isMember, err := s.repo.UserIsMemberOfServer(ctx, userID, serverID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotServerMember
	}

	// Verify folder exists if provided
	if folderID != nil {
		folder, err := s.repo.GetByID(ctx, userID, *folderID)
		if err != nil {
			return err
		}
		if folder == nil {
			return ErrFolderNotFound
		}
	}

	return s.repo.AssignServerToFolder(ctx, userID, serverID, folderID)
}

// MoveServersToFolder moves multiple servers to a folder
func (s *ServerFolderService) MoveServersToFolder(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error {
	// Parse server IDs
	serverIDs := make([]uuid.UUID, len(req.ServerIDs))
	for i, sid := range req.ServerIDs {
		serverID, err := uuid.Parse(sid)
		if err != nil {
			return errors.New("invalid server_id")
		}
		serverIDs[i] = serverID
	}

	// Parse folder ID if provided
	var folderID *uuid.UUID
	if req.FolderID != nil && *req.FolderID != "" {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return errors.New("invalid folder_id")
		}
		folderID = &fid
	}

	// Verify folder exists if provided
	if folderID != nil {
		folder, err := s.repo.GetByID(ctx, userID, *folderID)
		if err != nil {
			return err
		}
		if folder == nil {
			return ErrFolderNotFound
		}
	}

	// Verify user is member of all servers
	for _, serverID := range serverIDs {
		isMember, err := s.repo.UserIsMemberOfServer(ctx, userID, serverID)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrNotServerMember
		}
	}

	return s.repo.AssignServersToFolder(ctx, userID, serverIDs, folderID)
}

// ReorderServers reorders servers within a folder
func (s *ServerFolderService) ReorderServers(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error {
	// Verify folder exists if provided
	if folderID != nil {
		folder, err := s.repo.GetByID(ctx, userID, *folderID)
		if err != nil {
			return err
		}
		if folder == nil {
			return ErrFolderNotFound
		}
	}

	return s.repo.UpdateServerPositions(ctx, userID, req.ServerPositions)
}
