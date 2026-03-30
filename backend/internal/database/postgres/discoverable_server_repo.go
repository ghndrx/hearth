package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// DiscoverableServerRepository handles discoverable server data operations
type DiscoverableServerRepository struct {
	db *sqlx.DB
}

// NewDiscoverableServerRepository creates a new discoverable server repository
func NewDiscoverableServerRepository(db *sqlx.DB) *DiscoverableServerRepository {
	return &DiscoverableServerRepository{db: db}
}

// DiscoverableServerRepo interface for dependency injection
type DiscoverableServerRepo interface {
	GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) ([]*models.DiscoverableServerSearchResult, int, error)
	GetFeaturedServers(ctx context.Context, limit int) ([]*models.FeaturedServer, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error)
	GetCategories(ctx context.Context) ([]*models.CategoryInfo, error)
	SearchServers(ctx context.Context, query string, category models.ServerDiscoveryCategory, page, limit int) ([]*models.DiscoverableServerSearchResult, int, error)
	GetInviteCode(ctx context.Context, serverID uuid.UUID) (string, error)
}

// GetDiscoverableServers returns paginated discoverable servers
func (r *DiscoverableServerRepository) GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) ([]*models.DiscoverableServerSearchResult, int, error) {
	models.NormalizeDiscoverFilters(filters)

	// Build the WHERE clause
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "is_public = true")

	if filters.Query != "" {
		conditions = append(conditions, "(LOWER(name) LIKE LOWER($"+string(rune('0'+argIdx))+") OR LOWER(description) LIKE LOWER($"+string(rune('0'+argIdx))+"))")
		args = append(args, "%"+filters.Query+"%")
		argIdx++
	}

	if filters.Category != "" {
		conditions = append(conditions, "category = $"+string(rune('0'+argIdx)))
		args = append(args, string(filters.Category))
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := "SELECT COUNT(*) FROM discoverable_servers WHERE " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (filters.Page - 1) * filters.Limit

	// Get servers sorted by member_count DESC, is_verified DESC, name ASC
	query := `
		SELECT id, server_id, name, description, category, icon_url, banner_url, 
		       member_count, is_verified, is_featured, created_at
		FROM discoverable_servers
		WHERE ` + whereClause + `
		ORDER BY member_count DESC, is_verified DESC, name ASC
		LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))

	args = append(args, filters.Limit, offset)

	var servers []*models.DiscoverableServerSearchResult
	err = r.db.SelectContext(ctx, &servers, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// GetFeaturedServers returns featured servers
func (r *DiscoverableServerRepository) GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	query := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, featured_at, created_at
		FROM discoverable_servers
		WHERE is_public = true AND is_featured = true
		ORDER BY featured_at DESC
		LIMIT $1
	`

	var servers []*models.DiscoverableFeaturedServer
	err := r.db.SelectContext(ctx, &servers, query, limit)
	if err != nil {
		return nil, err
	}

	return servers, nil
}

// GetByServerID returns a discoverable server by its server ID
func (r *DiscoverableServerRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error) {
	query := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, is_public, is_featured, featured_at,
		       created_at, updated_at
		FROM discoverable_servers
		WHERE server_id = $1
	`

	var server models.DiscoverableServer
	err := r.db.GetContext(ctx, &server, query, serverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &server, nil
}

// GetByID returns a discoverable server by its ID
func (r *DiscoverableServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error) {
	query := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, is_public, is_featured, featured_at,
		       created_at, updated_at
		FROM discoverable_servers
		WHERE id = $1
	`

	var server models.DiscoverableServer
	err := r.db.GetContext(ctx, &server, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &server, nil
}

// GetCategories returns all categories with server counts
func (r *DiscoverableServerRepository) GetCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	query := `
		SELECT category as name, category as slug, COUNT(*) as server_count
		FROM discoverable_servers
		WHERE is_public = true
		GROUP BY category
		ORDER BY server_count DESC
	`

	var categories []*models.CategoryInfo
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

// SearchServers performs a search on discoverable servers
func (r *DiscoverableServerRepository) SearchServers(ctx context.Context, query string, category models.ServerDiscoveryCategory, page, limit int) ([]*models.DiscoverableServerSearchResult, int, error) {
	models.NormalizeDiscoverFilters(&models.DiscoverFilters{Query: query, Category: category, Page: page, Limit: limit})

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "is_public = true")

	if query != "" {
		conditions = append(conditions, "(LOWER(name) LIKE LOWER($"+string(rune('0'+argIdx))+") OR LOWER(description) LIKE LOWER($"+string(rune('0'+argIdx))+"))")
		args = append(args, "%"+query+"%")
		argIdx++
	}

	if category != "" {
		conditions = append(conditions, "category = $"+string(rune('0'+argIdx)))
		args = append(args, string(category))
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := "SELECT COUNT(*) FROM discoverable_servers WHERE " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	selectQuery := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, is_featured, created_at
		FROM discoverable_servers
		WHERE ` + whereClause + `
		ORDER BY member_count DESC, is_verified DESC, name ASC
		LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))

	args = append(args, limit, offset)

	var servers []*models.DiscoverableServerSearchResult
	err = r.db.SelectContext(ctx, &servers, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// GetInviteCode returns a public invite code for a server
func (r *DiscoverableServerRepository) GetInviteCode(ctx context.Context, serverID uuid.UUID) (string, error) {
	query := `
		SELECT code FROM invites
		WHERE server_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY uses ASC
		LIMIT 1
	`

	var code string
	err := r.db.GetContext(ctx, &code, query, serverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return code, nil
}
