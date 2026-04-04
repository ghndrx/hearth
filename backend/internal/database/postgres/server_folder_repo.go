package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type ServerFolderRepository struct {
	db *sqlx.DB
}

func NewServerFolderRepository(db *sqlx.DB) *ServerFolderRepository {
	return &ServerFolderRepository{db: db}
}

// Create creates a new server folder
func (r *ServerFolderRepository) Create(ctx context.Context, folder *models.ServerFolder) error {
	query := `
		INSERT INTO server_folders (id, user_id, parent_id, name, position, is_collapsed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		folder.ID, folder.UserID, folder.ParentID, folder.Name,
		folder.Position, folder.IsCollapsed, folder.CreatedAt, folder.UpdatedAt,
	)
	return err
}

// GetByID retrieves a folder by ID for a specific user
func (r *ServerFolderRepository) GetByID(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	var folder models.ServerFolder
	query := `
		SELECT id, user_id, parent_id, name, position, is_collapsed, created_at, updated_at
		FROM server_folders
		WHERE id = $1 AND user_id = $2
	`
	err := r.db.GetContext(ctx, &folder, query, folderID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &folder, err
}

// GetAllForUser retrieves all folders for a user
func (r *ServerFolderRepository) GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error) {
	var folders []*models.ServerFolder
	query := `
		SELECT id, user_id, parent_id, name, position, is_collapsed, created_at, updated_at
		FROM server_folders
		WHERE user_id = $1
		ORDER BY position ASC
	`
	err := r.db.SelectContext(ctx, &folders, query, userID)
	return folders, err
}

// Update updates a folder
func (r *ServerFolderRepository) Update(ctx context.Context, folder *models.ServerFolder) error {
	query := `
		UPDATE server_folders SET
			name = $3, parent_id = $4, position = $5, is_collapsed = $6, updated_at = $7
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query,
		folder.ID, folder.UserID, folder.Name, folder.ParentID,
		folder.Position, folder.IsCollapsed, folder.UpdatedAt,
	)
	return err
}

// Delete deletes a folder
func (r *ServerFolderRepository) Delete(ctx context.Context, userID, folderID uuid.UUID) error {
	query := `DELETE FROM server_folders WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, folderID, userID)
	return err
}

// GetServersInFolder retrieves all servers assigned to a folder
func (r *ServerFolderRepository) GetServersInFolder(ctx context.Context, userID, folderID uuid.UUID) ([]*models.ServerInFolder, error) {
	var servers []*models.ServerInFolder
	query := `
		SELECT sfs.server_id, sfs.folder_id, sfs.user_id, sfs.position, sfs.assigned_at
		FROM server_folder_servers sfs
		WHERE sfs.user_id = $1 AND sfs.folder_id = $2
		ORDER BY sfs.position ASC
	`
	err := r.db.SelectContext(ctx, &servers, query, userID, folderID)
	return servers, err
}

// GetUnassignedServers retrieves all servers not assigned to any folder for a user
func (r *ServerFolderRepository) GetUnassignedServers(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	var servers []*models.ServerInFolder
	query := `
		SELECT sfs.server_id, sfs.folder_id, sfs.user_id, sfs.position, sfs.assigned_at
		FROM server_folder_servers sfs
		WHERE sfs.user_id = $1 AND sfs.folder_id IS NULL
		ORDER BY sfs.position ASC
	`
	err := r.db.SelectContext(ctx, &servers, query, userID)
	return servers, err
}

// GetAllServerAssignments retrieves all server assignments for a user
func (r *ServerFolderRepository) GetAllServerAssignments(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	var servers []*models.ServerInFolder
	query := `
		SELECT sfs.server_id, sfs.folder_id, sfs.user_id, sfs.position, sfs.assigned_at
		FROM server_folder_servers sfs
		WHERE sfs.user_id = $1
		ORDER BY sfs.position ASC
	`
	err := r.db.SelectContext(ctx, &servers, query, userID)
	return servers, err
}

// AssignServerToFolder assigns a server to a folder (or removes from folder if folderID is nil)
func (r *ServerFolderRepository) AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	query := `
		INSERT INTO server_folder_servers (server_id, folder_id, user_id, position, assigned_at)
		VALUES ($1, $2, $3, 
			COALESCE((SELECT MAX(position) + 1 FROM server_folder_servers WHERE user_id = $3 AND folder_id IS NOT DISTINCT FROM $2), 0),
			NOW())
		ON CONFLICT (server_id, user_id) DO UPDATE SET folder_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, serverID, folderID, userID)
	return err
}

// AssignServersToFolder assigns multiple servers to a folder
func (r *ServerFolderRepository) AssignServersToFolder(ctx context.Context, userID uuid.UUID, serverIDs []uuid.UUID, folderID *uuid.UUID) error {
	if len(serverIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get max position in the target folder
	var maxPos sql.NullInt64
	posQuery := `SELECT MAX(position) FROM server_folder_servers WHERE user_id = $1 AND folder_id IS NOT DISTINCT FROM $2`
	err = tx.GetContext(ctx, &maxPos, posQuery, userID, folderID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	startPos := 0
	if maxPos.Valid {
		startPos = int(maxPos.Int64) + 1
	}

	for i, serverID := range serverIDs {
		query := `
			INSERT INTO server_folder_servers (server_id, folder_id, user_id, position, assigned_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (server_id, user_id) DO UPDATE SET folder_id = $2, position = $4
		`
		_, err := tx.ExecContext(ctx, query, serverID, folderID, userID, startPos+i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateServerPositions updates positions for servers in a folder
func (r *ServerFolderRepository) UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error {
	if len(positions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, pos := range positions {
		serverID, err := uuid.Parse(pos.ServerID)
		if err != nil {
			return fmt.Errorf("invalid server_id: %w", err)
		}
		query := `
			UPDATE server_folder_servers SET position = $1
			WHERE server_id = $2 AND user_id = $3
		`
		_, err = tx.ExecContext(ctx, query, pos.Position, serverID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetServerAssignment gets a server's folder assignment
func (r *ServerFolderRepository) GetServerAssignment(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error) {
	var assignment models.ServerInFolder
	query := `
		SELECT server_id, folder_id, user_id, position, assigned_at
		FROM server_folder_servers
		WHERE user_id = $1 AND server_id = $2
	`
	err := r.db.GetContext(ctx, &assignment, query, userID, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &assignment, err
}

// UserIsMemberOfServer checks if user is a member of the server
func (r *ServerFolderRepository) UserIsMemberOfServer(ctx context.Context, userID, serverID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM members WHERE user_id = $1 AND server_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, userID, serverID)
	return exists, err
}
