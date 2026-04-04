package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrFolderNotFound      = errors.New("folder not found")
	ErrMaxNestingExceeded  = errors.New("maximum folder nesting level exceeded")
	ErrInvalidParentFolder = errors.New("invalid parent folder")
	ErrServerNotInFolder   = errors.New("server not found in folder")
)

// ServerFolderRepository defines the interface for server folder data access
type ServerFolderRepository interface {
	Create(ctx context.Context, folder *models.ServerFolder) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ServerFolder, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error)
	Update(ctx context.Context, folder *models.ServerFolder) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetChildFolders(ctx context.Context, parentID uuid.UUID) ([]*models.ServerFolder, error)
	GetMaxPositionAtLevel(ctx context.Context, userID uuid.UUID, depth int, parentID *uuid.UUID) (int, error)
	AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID, position int) error
	RemoveServerFromFolder(ctx context.Context, userID, serverID uuid.UUID) error
	GetServerFolder(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error)
	GetServersInFolder(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID) ([]*models.ServerInFolder, error)
	GetAllUserServersWithFolders(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error)
	UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error
	GetChildFolderIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error)
}

// ServerFolderService handles server folder-related business logic
type ServerFolderService struct {
	repo         ServerFolderRepository
	serverRepo   ServerRepository
	eventBus     EventBus
}

// NewServerFolderService creates a new server folder service
func NewServerFolderService(
	repo ServerFolderRepository,
	serverRepo ServerRepository,
	eventBus EventBus,
) *ServerFolderService {
	return &ServerFolderService{
		repo:       repo,
		serverRepo: serverRepo,
		eventBus:   eventBus,
	}
}

// CreateFolder creates a new server folder
func (s *ServerFolderService) CreateFolder(ctx context.Context, userID uuid.UUID, req *models.CreateServerFolderRequest) (*models.ServerFolder, error) {
	now := time.Now()
	depth := 0

	// Calculate depth if parent is specified
	if req.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *req.ParentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrInvalidParentFolder
			}
			return nil, err
		}

		// Check user owns the parent
		if parent.UserID != userID {
			return nil, ErrInvalidParentFolder
		}

		// Check nesting depth
		if parent.Depth >= models.MaxNestingDepth {
			return nil, ErrMaxNestingExceeded
		}
		depth = parent.Depth + 1
	}

	// Get max position at this level
	maxPos, err := s.repo.GetMaxPositionAtLevel(ctx, userID, depth, req.ParentID)
	if err != nil {
		return nil, err
	}

	position := maxPos + 1
	if req.Position != nil {
		position = *req.Position
	}

	folder := &models.ServerFolder{
		ID:          uuid.New(),
		UserID:      userID,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Position:    position,
		IsCollapsed: false,
		Depth:       depth,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_created", map[string]interface{}{
			"user_id":  userID,
			"folder":   folder,
		})
	}

	return folder, nil
}

// GetUserFolders gets all folders for a user with their servers
func (s *ServerFolderService) GetUserFolders(ctx context.Context, userID uuid.UUID) (*models.ServerFolderTree, error) {
	folders, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build folder tree
	folderMap := make(map[uuid.UUID]*models.ServerFolder)
	rootFolders := []*models.ServerFolder{}

	for _, f := range folders {
		f.Children = []*models.ServerFolder{}
		f.Servers = []*models.ServerInFolder{}
		folderMap[f.ID] = f
	}

	for _, f := range folders {
		if f.ParentID == nil {
			rootFolders = append(rootFolders, f)
		} else {
			if parent, ok := folderMap[*f.ParentID]; ok {
				parent.Children = append(parent.Children, f)
			} else {
				// Orphaned folder, treat as root
				rootFolders = append(rootFolders, f)
			}
		}
	}

	// Get servers for each folder
	servers, err := s.repo.GetAllUserServersWithFolders(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Group servers by folder
	unassignedServers := []*models.ServerInFolder{}
	for _, srv := range servers {
		if srv.FolderID == nil {
			unassignedServers = append(unassignedServers, srv)
		} else if folder, ok := folderMap[*srv.FolderID]; ok {
			folder.Servers = append(folder.Servers, srv)
		}
	}

	return &models.ServerFolderTree{
		Folders: rootFolders,
		Servers: unassignedServers,
	}, nil
}

// GetFolder gets a folder by ID
func (s *ServerFolderService) GetFolder(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	folder, err := s.repo.GetByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}

	if folder.UserID != userID {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

// UpdateFolder updates a server folder
func (s *ServerFolderService) UpdateFolder(ctx context.Context, userID, folderID uuid.UUID, req *models.UpdateServerFolderRequest) (*models.ServerFolder, error) {
	folder, err := s.repo.GetByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFolderNotFound
		}
		return nil, err
	}

	if folder.UserID != userID {
		return nil, ErrFolderNotFound
	}

	// Handle parent change (move to new parent)
	if req.ParentID != nil || folder.ParentID != nil {
		if req.ParentID != nil {
			parent, err := s.repo.GetByID(ctx, *req.ParentID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, ErrInvalidParentFolder
				}
				return nil, err
			}
			if parent.UserID != userID {
				return nil, ErrInvalidParentFolder
			}
			// Cannot move folder to itself
			if parent.ID == folderID {
				return nil, ErrInvalidParentFolder
			}
			// Check if new parent would exceed max depth
			if parent.Depth >= models.MaxNestingDepth {
				return nil, ErrMaxNestingExceeded
			}
			folder.ParentID = req.ParentID
			folder.Depth = parent.Depth + 1

			// Update depth of all descendants
			if err := s.updateDescendantDepths(ctx, folderID, folder.Depth); err != nil {
				return nil, err
			}
		} else {
			// Moving to root level
			folder.ParentID = nil
			folder.Depth = 0
			if err := s.updateDescendantDepths(ctx, folderID, 0); err != nil {
				return nil, err
			}
		}
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

	if err := s.repo.Update(ctx, folder); err != nil {
		return nil, err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_updated", map[string]interface{}{
			"user_id":  userID,
			"folder":   folder,
		})
	}

	return folder, nil
}

// updateDescendantDepths updates the depth of all descendants when a folder is moved
func (s *ServerFolderService) updateDescendantDepths(ctx context.Context, folderID uuid.UUID, parentDepth int) error {
	childFolders, err := s.repo.GetChildFolders(ctx, folderID)
	if err != nil {
		return err
	}

	for _, child := range childFolders {
		child.Depth = parentDepth + 1
		if err := s.repo.Update(ctx, child); err != nil {
			return err
		}
		// Recursively update children
		if err := s.updateDescendantDepths(ctx, child.ID, child.Depth); err != nil {
			return err
		}
	}
	return nil
}

// DeleteFolder deletes a server folder
func (s *ServerFolderService) DeleteFolder(ctx context.Context, userID, folderID uuid.UUID) error {
	folder, err := s.repo.GetByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFolderNotFound
		}
		return err
	}

	if folder.UserID != userID {
		return ErrFolderNotFound
	}

	// Move servers in this folder to unassigned
	childFolders, err := s.repo.GetChildFolderIDs(ctx, folderID)
	if err != nil {
		return err
	}

	// Include the folder itself
	allFolderIDs := append(childFolders, folderID)

	// Move all servers in deleted folders to unassigned
	for _, fid := range allFolderIDs {
		servers, err := s.repo.GetServersInFolder(ctx, userID, &fid)
		if err != nil {
			return err
		}
		for _, srv := range servers {
			if err := s.repo.AssignServerToFolder(ctx, userID, srv.ServerID, nil, srv.Position); err != nil {
				return err
			}
		}
	}

	if err := s.repo.Delete(ctx, folderID); err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_deleted", map[string]interface{}{
			"user_id":   userID,
			"folder_id": folderID,
		})
	}

	return nil
}

// MoveServerToFolder moves a server to a folder
func (s *ServerFolderService) MoveServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	// Verify server belongs to user
	member, err := s.serverRepo.GetMember(ctx, serverID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServerNotInFolder // Or a more appropriate error
		}
		return err
	}
	if member == nil {
		return ErrServerNotInFolder
	}

	// Verify folder exists and belongs to user if specified
	if folderID != nil {
		folder, err := s.repo.GetByID(ctx, *folderID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrFolderNotFound
			}
			return err
		}
		if folder.UserID != userID {
			return ErrFolderNotFound
		}
	}

	// Get current max position in target folder
	maxPos, err := s.repo.GetMaxPositionAtLevel(ctx, userID, 0, folderID)
	if err != nil {
		return err
	}

	if err := s.repo.AssignServerToFolder(ctx, userID, serverID, folderID, maxPos+1); err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_moved", map[string]interface{}{
			"user_id":   userID,
			"server_id": serverID,
			"folder_id": folderID,
		})
	}

	return nil
}

// MoveServersToFolder moves multiple servers to a folder
func (s *ServerFolderService) MoveServersToFolder(ctx context.Context, userID uuid.UUID, req *models.MoveServersToFolderRequest) error {
	// Verify folder exists and belongs to user if specified
	if req.FolderID != nil {
		folder, err := s.repo.GetByID(ctx, *req.FolderID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrFolderNotFound
			}
			return err
		}
		if folder.UserID != userID {
			return ErrFolderNotFound
		}
	}

	// Get current max position in target folder
	maxPos, err := s.repo.GetMaxPositionAtLevel(ctx, userID, 0, req.FolderID)
	if err != nil {
		return err
	}

	position := maxPos + 1
	for _, serverID := range req.ServerIDs {
		// Verify server belongs to user
		member, err := s.serverRepo.GetMember(ctx, serverID, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // Skip servers not in
			}
			return err
		}
		if member == nil {
			continue
		}

		if err := s.repo.AssignServerToFolder(ctx, userID, serverID, req.FolderID, position); err != nil {
			return err
		}
		position++
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_bulk_moved", map[string]interface{}{
			"user_id":    userID,
			"server_ids": req.ServerIDs,
			"folder_id":  req.FolderID,
		})
	}

	return nil
}

// ReorderServers reorders servers within a folder
func (s *ServerFolderService) ReorderServers(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, req *models.ReorderServersRequest) error {
	// Verify folder exists and belongs to user if specified
	if folderID != nil {
		folder, err := s.repo.GetByID(ctx, *folderID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrFolderNotFound
			}
			return err
		}
		if folder.UserID != userID {
			return ErrFolderNotFound
		}
	}

	if err := s.repo.UpdateServerPositions(ctx, userID, req.ServerPositions); err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("server_folder_reordered", map[string]interface{}{
			"user_id":    userID,
			"folder_id":  folderID,
			"positions":  req.ServerPositions,
		})
	}

	return nil
}
