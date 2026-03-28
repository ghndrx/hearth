package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"hearth/internal/models"
)

// StickerRepository defines sticker data access operations
type StickerRepository interface {
	// CRUD operations
	Create(ctx context.Context, sticker *models.Sticker) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Sticker, error)
	Update(ctx context.Context, sticker *models.Sticker) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Sticker, error)
	GetGlobal(ctx context.Context) ([]*models.Sticker, error)
	GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.Sticker, error)
	Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.Sticker, error)
}

type stickerRepo struct {
	db *sql.DB
}

// NewStickerRepository creates a new sticker repository
func NewStickerRepository(db *sql.DB) StickerRepository {
	return &stickerRepo{db: db}
}

func (r *stickerRepo) Create(ctx context.Context, sticker *models.Sticker) error {
	query := `
		INSERT INTO stickers (id, server_id, name, tags, url, format, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		sticker.ID,
		sticker.ServerID,
		sticker.Name,
		pq.Array(sticker.Tags),
		sticker.URL,
		sticker.Format,
		sticker.CreatedBy,
		sticker.CreatedAt,
	)
	return err
}

func (r *stickerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Sticker, error) {
	var sticker models.Sticker
	query := `
		SELECT id, server_id, name, tags, url, format, created_by, created_at
		FROM stickers
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sticker.ID,
		&sticker.ServerID,
		&sticker.Name,
		pq.Array(&sticker.Tags),
		&sticker.URL,
		&sticker.Format,
		&sticker.CreatedBy,
		&sticker.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sticker, nil
}

func (r *stickerRepo) Update(ctx context.Context, sticker *models.Sticker) error {
	query := `
		UPDATE stickers
		SET name = $2, tags = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		sticker.ID,
		sticker.Name,
		pq.Array(sticker.Tags),
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *stickerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM stickers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *stickerRepo) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Sticker, error) {
	query := `
		SELECT id, server_id, name, tags, url, format, created_by, created_at
		FROM stickers
		WHERE server_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stickers []*models.Sticker
	for rows.Next() {
		sticker := &models.Sticker{}
		err := rows.Scan(
			&sticker.ID,
			&sticker.ServerID,
			&sticker.Name,
			pq.Array(&sticker.Tags),
			&sticker.URL,
			&sticker.Format,
			&sticker.CreatedBy,
			&sticker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		stickers = append(stickers, sticker)
	}
	return stickers, nil
}

func (r *stickerRepo) GetGlobal(ctx context.Context) ([]*models.Sticker, error) {
	query := `
		SELECT id, server_id, name, tags, url, format, created_by, created_at
		FROM stickers
		WHERE server_id IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stickers []*models.Sticker
	for rows.Next() {
		sticker := &models.Sticker{}
		err := rows.Scan(
			&sticker.ID,
			&sticker.ServerID,
			&sticker.Name,
			pq.Array(&sticker.Tags),
			&sticker.URL,
			&sticker.Format,
			&sticker.CreatedBy,
			&sticker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		stickers = append(stickers, sticker)
	}
	return stickers, nil
}

func (r *stickerRepo) GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.Sticker, error) {
	// Get global stickers
	globalQuery := `
		SELECT id, server_id, name, tags, url, format, created_by, created_at
		FROM stickers
		WHERE server_id IS NULL
		ORDER BY created_at ASC
	`

	globalRows, err := r.db.QueryContext(ctx, globalQuery)
	if err != nil {
		return nil, err
	}
	defer globalRows.Close()

	var stickers []*models.Sticker
	for globalRows.Next() {
		sticker := &models.Sticker{}
		err := globalRows.Scan(
			&sticker.ID,
			&sticker.ServerID,
			&sticker.Name,
			pq.Array(&sticker.Tags),
			&sticker.URL,
			&sticker.Format,
			&sticker.CreatedBy,
			&sticker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		stickers = append(stickers, sticker)
	}

	// If serverID is provided, also get server-specific stickers
	if serverID != nil {
		serverQuery := `
			SELECT id, server_id, name, tags, url, format, created_by, created_at
			FROM stickers
			WHERE server_id = $1
			ORDER BY created_at ASC
		`

		serverRows, err := r.db.QueryContext(ctx, serverQuery, *serverID)
		if err != nil {
			return nil, err
		}
		defer serverRows.Close()

		for serverRows.Next() {
			sticker := &models.Sticker{}
			err := serverRows.Scan(
				&sticker.ID,
				&sticker.ServerID,
				&sticker.Name,
				pq.Array(&sticker.Tags),
				&sticker.URL,
				&sticker.Format,
				&sticker.CreatedBy,
				&sticker.CreatedAt,
			)
			if err != nil {
				return nil, err
			}
			stickers = append(stickers, sticker)
		}
	}

	return stickers, nil
}

func (r *stickerRepo) Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.Sticker, error) {
	searchQuery := `
		SELECT id, server_id, name, tags, url, format, created_by, created_at
		FROM stickers
		WHERE (
			name ILIKE $1
			OR $1 = ANY(tags)
		)
	`
	args := []interface{}{"%" + query + "%"}

	if serverID != nil {
		searchQuery += ` AND (server_id IS NULL OR server_id = $2)`
		args = append(args, *serverID)
	}

	searchQuery += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stickers []*models.Sticker
	for rows.Next() {
		sticker := &models.Sticker{}
		err := rows.Scan(
			&sticker.ID,
			&sticker.ServerID,
			&sticker.Name,
			pq.Array(&sticker.Tags),
			&sticker.URL,
			&sticker.Format,
			&sticker.CreatedBy,
			&sticker.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		stickers = append(stickers, sticker)
	}
	return stickers, nil
}

// Ensure test helpers exist (used by sticker_service_test.go)
func (r *stickerRepo) Upload_Test_Add(sticker *models.Sticker) {
	// Test helper - not used in production
}

func (r *stickerRepo) GetByID_Test(stickerID uuid.UUID) *models.Sticker {
	// Test helper - not used in production
	return nil
}
