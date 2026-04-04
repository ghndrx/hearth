package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/models"
)

// ServerFolderRepository handles database operations for server folders
type ServerFolderRepository struct {
	db *sqlx.DB
}

// NewServerFolderRepository creates a new server folder repository
func NewServerFolderRepository(db *sqlx.DB) *ServerFolderRepository {
	return &ServerFolderRepository{db: db}
}

// Create creates a new server folder
func (r *ServerFolderRepository) Create(ctx context.Context, folder *models.ServerFolder) error {
	query := `
		INSERT INTO server_folders (id, user_id, parent_id, name, position, is_collapsed, depth, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		folder.ID,
		folder.UserID,
		folder.ParentID,
		folder.Name,
		folder.Position,
		folder.IsCollapsed,
		folder.Depth,
		folder.CreatedAt,
		folder.UpdatedAt,
	)
	return err
}

// GetByID gets a server folder by ID
func (r *ServerFolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ServerFolder, error) {
	query := `
		SELECT id, user_id, parent_id, name, position, is_collapsed, depth, created_at, updated_at
		FROM server_folders
		WHERE id = $1
	`
	folder := &models.ServerFolder{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.ParentID,
		&folder.Name,
		&folder.Position,
		&folder.IsCollapsed,
		&folder.Depth,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

// GetByUserID gets all server folders for a user
func (r *ServerFolderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error) {
	query := `
		SELECT id, user_id, parent_id, name, position, is_collapsed, depth, created_at, updated_at
		FROM server_folders
		WHERE user_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.ServerFolder
	for rows.Next() {
		folder := &models.ServerFolder{}
		err := rows.Scan(
			&folder.ID,
			&folder.UserID,
			&folder.ParentID,
			&folder.Name,
			&folder.Position,
			&folder.IsCollapsed,
			&folder.Depth,
			&folder.CreatedAt,
			&folder.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

// Update updates a server folder
func (r *ServerFolderRepository) Update(ctx context.Context, folder *models.ServerFolder) error {
	query := `
		UPDATE server_folders
		SET name = $1, position = $2, is_collapsed = $3, parent_id = $4, depth = $5, updated_at = $6
		WHERE id = $7
	`
	folder.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		folder.Name,
		folder.Position,
		folder.IsCollapsed,
		folder.ParentID,
		folder.Depth,
		folder.UpdatedAt,
		folder.ID,
	)
	return err
}

// Delete deletes a server folder
func (r *ServerFolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM server_folders WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetChildFolders gets all child folders of a folder
func (r *ServerFolderRepository) GetChildFolders(ctx context.Context, parentID uuid.UUID) ([]*models.ServerFolder, error) {
	query := `
		SELECT id, user_id, parent_id, name, position, is_collapsed, depth, created_at, updated_at
		FROM server_folders
		WHERE parent_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.ServerFolder
	for rows.Next() {
		folder := &models.ServerFolder{}
		err := rows.Scan(
			&folder.ID,
			&folder.UserID,
			&folder.ParentID,
			&folder.Name,
			&folder.Position,
			&folder.IsCollapsed,
			&folder.Depth,
			&folder.CreatedAt,
			&folder.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

// GetMaxPositionAtLevel gets the maximum position at a given depth for a user
func (r *ServerFolderRepository) GetMaxPositionAtLevel(ctx context.Context, userID uuid.UUID, depth int, parentID *uuid.UUID) (int, error) {
	var query string
	var args []interface{}

	if parentID == nil {
		query = `SELECT COALESCE(MAX(position), -1) FROM server_folders WHERE user_id = $1 AND parent_id IS NULL`
		args = []interface{}{userID}
	} else {
		query = `SELECT COALESCE(MAX(position), -1) FROM server_folders WHERE user_id = $1 AND parent_id = $2`
		args = []interface{}{userID, *parentID}
	}

	var maxPos int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&maxPos)
	if err != nil {
		return -1, err
	}
	return maxPos, nil
}

// AssignServerToFolder assigns a server to a folder for a user
func (r *ServerFolderRepository) AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID, position int) error {
	query := `
		INSERT INTO user_server_folder (user_id, server_id, folder_id, position, assigned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			folder_id = EXCLUDED.folder_id,
			position = EXCLUDED.position,
			assigned_at = EXCLUDED.assigned_at
	`
	_, err := r.db.ExecContext(ctx, query, userID, serverID, folderID, position, time.Now())
	return err
}

// RemoveServerFromFolder removes a server from its folder
func (r *ServerFolderRepository) RemoveServerFromFolder(ctx context.Context, userID, serverID uuid.UUID) error {
	query := `DELETE FROM user_server_folder WHERE user_id = $1 AND server_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, serverID)
	return err
}

// GetServerFolder gets the folder assignment for a server
func (r *ServerFolderRepository) GetServerFolder(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error) {
	query := `
		SELECT usf.server_id, usf.folder_id, usf.position, usf.assigned_at,
			   s.id, s.name, s.icon_url, s.banner_url, s.description, s.owner_id,
			   s.default_channel_id, s.afk_channel_id, s.afk_timeout, s.verification_level,
			   s.explicit_content_filter, s.default_notifications, s.features, s.max_members,
			   s.vanity_url_code, s.created_at, s.updated_at
		FROM user_server_folder usf
		JOIN servers s ON s.id = usf.server_id
		WHERE usf.user_id = $1 AND usf.server_id = $2
	`
	sif := &models.ServerInFolder{Server: &models.Server{}}
	err := r.db.QueryRowContext(ctx, query, userID, serverID).Scan(
		&sif.ServerID,
		&sif.FolderID,
		&sif.Position,
		&sif.AssignedAt,
		&sif.Server.ID,
		&sif.Server.Name,
		&sif.Server.IconURL,
		&sif.Server.BannerURL,
		&sif.Server.Description,
		&sif.Server.OwnerID,
		&sif.Server.DefaultChannelID,
		&sif.Server.AFKChannelID,
		&sif.Server.AFKTimeout,
		&sif.Server.VerificationLevel,
		&sif.Server.ExplicitContentFilter,
		&sif.Server.DefaultNotifications,
		&sif.Server.Features,
		&sif.Server.MaxMembers,
		&sif.Server.VanityURLCode,
		&sif.Server.CreatedAt,
		&sif.Server.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sif, nil
}

// GetServersInFolder gets all servers in a folder for a user
func (r *ServerFolderRepository) GetServersInFolder(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID) ([]*models.ServerInFolder, error) {
	var query string
	var args []interface{}

	if folderID == nil {
		query = `
			SELECT usf.server_id, usf.folder_id, usf.position, usf.assigned_at,
				   s.id, s.name, s.icon_url, s.banner_url, s.description, s.owner_id,
				   s.default_channel_id, s.afk_channel_id, s.afk_timeout, s.verification_level,
				   s.explicit_content_filter, s.default_notifications, s.features, s.max_members,
				   s.vanity_url_code, s.created_at, s.updated_at
			FROM user_server_folder usf
			JOIN servers s ON s.id = usf.server_id
			WHERE usf.user_id = $1 AND usf.folder_id IS NULL
			ORDER BY usf.position ASC
		`
		args = []interface{}{userID}
	} else {
		query = `
			SELECT usf.server_id, usf.folder_id, usf.position, usf.assigned_at,
				   s.id, s.name, s.icon_url, s.banner_url, s.description, s.owner_id,
				   s.default_channel_id, s.afk_channel_id, s.afk_timeout, s.verification_level,
				   s.explicit_content_filter, s.default_notifications, s.features, s.max_members,
				   s.vanity_url_code, s.created_at, s.updated_at
			FROM user_server_folder usf
			JOIN servers s ON s.id = usf.server_id
			WHERE usf.user_id = $1 AND usf.folder_id = $2
			ORDER BY usf.position ASC
		`
		args = []interface{}{userID, *folderID}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.ServerInFolder
	for rows.Next() {
		sif := &models.ServerInFolder{Server: &models.Server{}}
		err := rows.Scan(
			&sif.ServerID,
			&sif.FolderID,
			&sif.Position,
			&sif.AssignedAt,
			&sif.Server.ID,
			&sif.Server.Name,
			&sif.Server.IconURL,
			&sif.Server.BannerURL,
			&sif.Server.Description,
			&sif.Server.OwnerID,
			&sif.Server.DefaultChannelID,
			&sif.Server.AFKChannelID,
			&sif.Server.AFKTimeout,
			&sif.Server.VerificationLevel,
			&sif.Server.ExplicitContentFilter,
			&sif.Server.DefaultNotifications,
			&sif.Server.Features,
			&sif.Server.MaxMembers,
			&sif.Server.VanityURLCode,
			&sif.Server.CreatedAt,
			&sif.Server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, sif)
	}
	return servers, rows.Err()
}

// GetAllUserServersWithFolders gets all servers for a user with their folder assignments
func (r *ServerFolderRepository) GetAllUserServersWithFolders(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	query := `
		SELECT usf.server_id, usf.folder_id, usf.position, usf.assigned_at,
			   s.id, s.name, s.icon_url, s.banner_url, s.description, s.owner_id,
			   s.default_channel_id, s.afk_channel_id, s.afk_timeout, s.verification_level,
			   s.explicit_content_filter, s.default_notifications, s.features, s.max_members,
			   s.vanity_url_code, s.created_at, s.updated_at
		FROM user_server_folder usf
		JOIN servers s ON s.id = usf.server_id
		JOIN members m ON m.server_id = s.id AND m.user_id = usf.user_id
		WHERE usf.user_id = $1
		ORDER BY usf.folder_id NULLS FIRST, usf.position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.ServerInFolder
	for rows.Next() {
		sif := &models.ServerInFolder{Server: &models.Server{}}
		err := rows.Scan(
			&sif.ServerID,
			&sif.FolderID,
			&sif.Position,
			&sif.AssignedAt,
			&sif.Server.ID,
			&sif.Server.Name,
			&sif.Server.IconURL,
			&sif.Server.BannerURL,
			&sif.Server.Description,
			&sif.Server.OwnerID,
			&sif.Server.DefaultChannelID,
			&sif.Server.AFKChannelID,
			&sif.Server.AFKTimeout,
			&sif.Server.VerificationLevel,
			&sif.Server.ExplicitContentFilter,
			&sif.Server.DefaultNotifications,
			&sif.Server.Features,
			&sif.Server.MaxMembers,
			&sif.Server.VanityURLCode,
			&sif.Server.CreatedAt,
			&sif.Server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, sif)
	}
	return servers, rows.Err()
}

// UpdateServerPositions updates positions of multiple servers in a folder
func (r *ServerFolderRepository) UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE user_server_folder SET position = $1 WHERE user_id = $2 AND server_id = $3
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, pos := range positions {
		_, err := stmt.ExecContext(ctx, pos.Position, userID, pos.ServerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetChildFolderIDs gets all descendant folder IDs (for cascade delete check)
func (r *ServerFolderRepository) GetChildFolderIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id FROM server_folders WHERE parent_id = $1
			UNION ALL
			SELECT sf.id FROM server_folders sf
			INNER JOIN descendants d ON sf.parent_id = d.id
		)
		SELECT id FROM descendants
	`
	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
