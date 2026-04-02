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

	// Sticker Pack operations
	CreatePack(ctx context.Context, pack *models.StickerPack) error
	GetPackByID(ctx context.Context, id uuid.UUID) (*models.StickerPack, error)
	UpdatePack(ctx context.Context, pack *models.StickerPack) error
	DeletePack(ctx context.Context, id uuid.UUID) error
	GetPacksByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StickerPack, error)
	GetGlobalPacks(ctx context.Context) ([]*models.StickerPack, error)
	GetPacksByTier(ctx context.Context, tier models.StickerPackTier) ([]*models.StickerPack, error)
	GetAvailablePacks(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error)

	// Pack-Sticker relationship operations
	AddStickerToPack(ctx context.Context, packID, stickerID uuid.UUID, position int, isDefault bool) error
	RemoveStickerFromPack(ctx context.Context, packID, stickerID uuid.UUID) error
	GetStickersInPack(ctx context.Context, packID uuid.UUID) ([]*models.Sticker, error)
	GetPacksContainingSticker(ctx context.Context, stickerID uuid.UUID) ([]*models.StickerPack, error)
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
		INSERT INTO stickers (id, server_id, name, tags, url, format, required_tier, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		sticker.ID,
		sticker.ServerID,
		sticker.Name,
		pq.Array(sticker.Tags),
		sticker.URL,
		sticker.Format,
		sticker.RequiredTier,
		sticker.CreatedBy,
		sticker.CreatedAt,
	)
	return err
}

func (r *stickerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Sticker, error) {
	var sticker models.Sticker
	query := `
		SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
		&sticker.RequiredTier,
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
		SET name = $2, tags = $3, required_tier = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		sticker.ID,
		sticker.Name,
		pq.Array(sticker.Tags),
		sticker.RequiredTier,
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
		SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
			&sticker.RequiredTier,
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
		SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
			&sticker.RequiredTier,
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
		SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
			&sticker.RequiredTier,
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
			SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
				&sticker.RequiredTier,
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
		SELECT id, server_id, name, tags, url, format, required_tier, created_by, created_at
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
			&sticker.RequiredTier,
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

// CreatePack creates a new sticker pack
func (r *stickerRepo) CreatePack(ctx context.Context, pack *models.StickerPack) error {
	query := `
		INSERT INTO sticker_packs (id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		pack.ID,
		pack.Name,
		pack.Description,
		pack.IconURL,
		pack.Tier,
		pack.StickerCount,
		pack.IsActive,
		pack.IsGlobal,
		pack.ServerID,
		pack.CreatedBy,
		pack.CreatedAt,
		pack.UpdatedAt,
	)
	return err
}

// GetPackByID retrieves a sticker pack by ID
func (r *stickerRepo) GetPackByID(ctx context.Context, id uuid.UUID) (*models.StickerPack, error) {
	var pack models.StickerPack
	query := `
		SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
		FROM sticker_packs
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pack.ID,
		&pack.Name,
		&pack.Description,
		&pack.IconURL,
		&pack.Tier,
		&pack.StickerCount,
		&pack.IsActive,
		&pack.IsGlobal,
		&pack.ServerID,
		&pack.CreatedBy,
		&pack.CreatedAt,
		&pack.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pack, nil
}

// UpdatePack updates a sticker pack
func (r *stickerRepo) UpdatePack(ctx context.Context, pack *models.StickerPack) error {
	query := `
		UPDATE sticker_packs
		SET name = $2, description = $3, icon_url = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		pack.ID,
		pack.Name,
		pack.Description,
		pack.IconURL,
		pack.IsActive,
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

// DeletePack deletes a sticker pack
func (r *stickerRepo) DeletePack(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sticker_packs WHERE id = $1`
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

// GetPacksByServer retrieves all sticker packs for a server
func (r *stickerRepo) GetPacksByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StickerPack, error) {
	query := `
		SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
		FROM sticker_packs
		WHERE server_id = $1 AND is_active = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []*models.StickerPack
	for rows.Next() {
		pack := &models.StickerPack{}
		err := rows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.Description,
			&pack.IconURL,
			&pack.Tier,
			&pack.StickerCount,
			&pack.IsActive,
			&pack.IsGlobal,
			&pack.ServerID,
			&pack.CreatedBy,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// GetGlobalPacks retrieves all global sticker packs
func (r *stickerRepo) GetGlobalPacks(ctx context.Context) ([]*models.StickerPack, error) {
	query := `
		SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
		FROM sticker_packs
		WHERE is_global = true AND is_active = true
		ORDER BY tier ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []*models.StickerPack
	for rows.Next() {
		pack := &models.StickerPack{}
		err := rows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.Description,
			&pack.IconURL,
			&pack.Tier,
			&pack.StickerCount,
			&pack.IsActive,
			&pack.IsGlobal,
			&pack.ServerID,
			&pack.CreatedBy,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// GetPacksByTier retrieves all sticker packs of a specific tier
func (r *stickerRepo) GetPacksByTier(ctx context.Context, tier models.StickerPackTier) ([]*models.StickerPack, error) {
	query := `
		SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
		FROM sticker_packs
		WHERE tier = $1 AND is_active = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []*models.StickerPack
	for rows.Next() {
		pack := &models.StickerPack{}
		err := rows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.Description,
			&pack.IconURL,
			&pack.Tier,
			&pack.StickerCount,
			&pack.IsActive,
			&pack.IsGlobal,
			&pack.ServerID,
			&pack.CreatedBy,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// GetAvailablePacks retrieves all packs available to a user based on their tier
func (r *stickerRepo) GetAvailablePacks(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error) {
	// First get global packs that user has access to based on tier
	tierOrder := map[models.StickerPackTier]int{
		models.StickerPackTierFree:    0,
		models.StickerPackTierBasic:   1,
		models.StickerPackTierPremium: 2,
	}
	userTierLevel := tierOrder[userTier]

	globalQuery := `
		SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
		FROM sticker_packs
		WHERE is_global = true AND is_active = true
		ORDER BY tier ASC, created_at ASC
	`

	globalRows, err := r.db.QueryContext(ctx, globalQuery)
	if err != nil {
		return nil, err
	}
	defer globalRows.Close()

	var packs []*models.StickerPack
	for globalRows.Next() {
		pack := &models.StickerPack{}
		err := globalRows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.Description,
			&pack.IconURL,
			&pack.Tier,
			&pack.StickerCount,
			&pack.IsActive,
			&pack.IsGlobal,
			&pack.ServerID,
			&pack.CreatedBy,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Filter by tier access
		if tierOrder[pack.Tier] <= userTierLevel {
			packs = append(packs, pack)
		}
	}

	// If serverID is provided, also get server-specific packs
	if serverID != nil {
		serverQuery := `
			SELECT id, name, description, icon_url, tier, sticker_count, is_active, is_global, server_id, created_by, created_at, updated_at
			FROM sticker_packs
			WHERE server_id = $1 AND is_active = true
			ORDER BY tier ASC, created_at ASC
		`

		serverRows, err := r.db.QueryContext(ctx, serverQuery, *serverID)
		if err != nil {
			return nil, err
		}
		defer serverRows.Close()

		for serverRows.Next() {
			pack := &models.StickerPack{}
			err := serverRows.Scan(
				&pack.ID,
				&pack.Name,
				&pack.Description,
				&pack.IconURL,
				&pack.Tier,
				&pack.StickerCount,
				&pack.IsActive,
				&pack.IsGlobal,
				&pack.ServerID,
				&pack.CreatedBy,
				&pack.CreatedAt,
				&pack.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			// Filter by tier access
			if tierOrder[pack.Tier] <= userTierLevel {
				packs = append(packs, pack)
			}
		}
	}

	return packs, nil
}

// AddStickerToPack adds a sticker to a pack
func (r *stickerRepo) AddStickerToPack(ctx context.Context, packID, stickerID uuid.UUID, position int, isDefault bool) error {
	query := `
		INSERT INTO pack_stickers (id, pack_id, sticker_id, position, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (pack_id, sticker_id) DO UPDATE SET position = $4, is_default = $5
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), packID, stickerID, position, isDefault)
	return err
}

// RemoveStickerFromPack removes a sticker from a pack
func (r *stickerRepo) RemoveStickerFromPack(ctx context.Context, packID, stickerID uuid.UUID) error {
	query := `DELETE FROM pack_stickers WHERE pack_id = $1 AND sticker_id = $2`
	_, err := r.db.ExecContext(ctx, query, packID, stickerID)
	return err
}

// GetStickersInPack retrieves all stickers in a pack
func (r *stickerRepo) GetStickersInPack(ctx context.Context, packID uuid.UUID) ([]*models.Sticker, error) {
	query := `
		SELECT s.id, s.server_id, s.name, s.tags, s.url, s.format, s.required_tier, s.created_by, s.created_at
		FROM stickers s
		INNER JOIN pack_stickers ps ON s.id = ps.sticker_id
		WHERE ps.pack_id = $1
		ORDER BY ps.position ASC, ps.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, packID)
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
			&sticker.RequiredTier,
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

// GetPacksContainingSticker retrieves all packs containing a specific sticker
func (r *stickerRepo) GetPacksContainingSticker(ctx context.Context, stickerID uuid.UUID) ([]*models.StickerPack, error) {
	query := `
		SELECT sp.id, sp.name, sp.description, sp.icon_url, sp.tier, sp.sticker_count, sp.is_active, sp.is_global, sp.server_id, sp.created_by, sp.created_at, sp.updated_at
		FROM sticker_packs sp
		INNER JOIN pack_stickers ps ON sp.id = ps.pack_id
		WHERE ps.sticker_id = $1 AND sp.is_active = true
		ORDER BY sp.tier ASC
	`

	rows, err := r.db.QueryContext(ctx, query, stickerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []*models.StickerPack
	for rows.Next() {
		pack := &models.StickerPack{}
		err := rows.Scan(
			&pack.ID,
			&pack.Name,
			&pack.Description,
			&pack.IconURL,
			&pack.Tier,
			&pack.StickerCount,
			&pack.IsActive,
			&pack.IsGlobal,
			&pack.ServerID,
			&pack.CreatedBy,
			&pack.CreatedAt,
			&pack.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}
	return packs, nil
}

// Ensure test helpers exist (used by sticker_service_test.go)
func (r *stickerRepo) Upload_Test_Add(sticker *models.Sticker) {
	// Test helper - not used in production
}

func (r *stickerRepo) GetByID_Test(stickerID uuid.UUID) *models.Sticker {
	// Test helper - not used in production
	return nil
}
