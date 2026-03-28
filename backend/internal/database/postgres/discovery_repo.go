package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// DiscoveryRepository handles discovery data operations
type DiscoveryRepository struct {
	db *sqlx.DB
}

// NewDiscoveryRepository creates a new discovery repository
func NewDiscoveryRepository(db *sqlx.DB) *DiscoveryRepository {
	return &DiscoveryRepository{db: db}
}

// GetFeaturedServers returns featured servers for the discovery page
func (r *DiscoveryRepository) GetFeaturedServers(ctx context.Context, limit int) ([]*models.FeaturedServer, error) {
	query := `
		SELECT 
			l.id, l.server_id, l.short_description, l.is_featured, l.featured_at,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description,
			c.name as category_name, c.slug as category_slug
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		LEFT JOIN server_discovery_listing_categories lc ON lc.listing_id = l.id
		LEFT JOIN server_discovery_categories c ON c.id = lc.category_id
		WHERE l.is_listed = true AND l.is_featured = true AND l.approval_status = 'approved'
		ORDER BY l.featured_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryxContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map to deduplicate servers (a server can have multiple categories)
	serverMap := make(map[uuid.UUID]*models.FeaturedServer)

	for rows.Next() {
		var fs models.FeaturedServer
		var s models.Server
		var categoryName, categorySlug sql.NullString
		var region, iconURL, bannerURL, description sql.NullString

		err := rows.Scan(
			&fs.ID, &fs.ServerID, &fs.ShortDescription, &fs.IsFeatured, &fs.FeaturedAt,
			&fs.MemberCountSnapshot, &fs.OnlineCountSnapshot, &fs.WeeklyGrowthRate,
			&fs.EngagementScore, &region, &fs.Language, &fs.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
			&categoryName, &categorySlug,
		)
		if err != nil {
			return nil, err
		}

		if region.Valid {
			fs.Region = &region.String
		}
		if iconURL.Valid {
			s.IconURL = &iconURL.String
		}
		if bannerURL.Valid {
			s.BannerURL = &bannerURL.String
		}
		if description.Valid {
			s.Description = &description.String
		}

		fs.Name = s.Name
		fs.IconURL = s.IconURL
		fs.BannerURL = s.BannerURL
		fs.Description = s.Description
		fs.MemberCount = fs.MemberCountSnapshot
		fs.OnlineCount = fs.OnlineCountSnapshot

		// Add category
		if categoryName.Valid && categorySlug.Valid {
			cat := models.ServerCategory(categorySlug.String)
			fs.Categories = append(fs.Categories, cat)
			if len(fs.Categories) == 1 {
				fs.Category = cat
			}
		}

		serverMap[fs.ServerID] = &fs
	}

	result := make([]*models.FeaturedServer, 0, len(serverMap))
	for _, v := range serverMap {
		result = append(result, v)
	}

	return result, nil
}

// SearchServers searches servers based on filters
func (r *DiscoveryRepository) SearchServers(ctx context.Context, filters *models.DiscoveryFilters) ([]*models.ServerListingResult, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, "l.is_listed = true AND l.approval_status = 'approved'")

	if filters.Query != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(s.name ILIKE $%d OR s.description ILIKE $%d OR l.short_description ILIKE $%d)",
			argNum, argNum, argNum,
		))
		args = append(args, "%"+filters.Query+"%")
		argNum++
	}

	if filters.Category != "" {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM server_discovery_listing_categories lc2 JOIN server_discovery_categories c2 ON c2.id = lc2.category_id WHERE lc2.listing_id = l.id AND c2.slug = $%d)",
			argNum,
		))
		args = append(args, string(filters.Category))
		argNum++
	}

	if len(filters.Region) > 0 {
		conditions = append(conditions, fmt.Sprintf("l.region = $%d", argNum))
		args = append(args, filters.Region)
		argNum++
	}

	if filters.Language != "" {
		conditions = append(conditions, fmt.Sprintf("l.language = $%d", argNum))
		args = append(args, filters.Language)
		argNum++
	}

	if filters.MinMembers > 0 {
		conditions = append(conditions, fmt.Sprintf("l.member_count_snapshot >= $%d", argNum))
		args = append(args, filters.MinMembers)
		argNum++
	}

	if filters.MaxMembers > 0 {
		conditions = append(conditions, fmt.Sprintf("l.member_count_snapshot <= $%d", argNum))
		args = append(args, filters.MaxMembers)
		argNum++
	}

	if filters.Featured != nil && *filters.Featured {
		conditions = append(conditions, "l.is_featured = true")
	}

	// Sorting
	orderBy := "l.member_count_snapshot DESC"
	switch filters.SortBy {
	case "growth":
		orderBy = "l.weekly_growth_rate DESC"
	case "engagement":
		orderBy = "l.engagement_score DESC"
	case "newest":
		orderBy = "l.created_at DESC"
	default:
		orderBy = "l.member_count_snapshot DESC"
	}

	if filters.SortOrder == "asc" {
		orderBy = strings.Replace(orderBy, " DESC", " ASC", 1)
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT l.id)
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		WHERE %s
	`, strings.Join(conditions, " AND "))

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Main query
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset := filters.Offset

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (l.id)
			l.id, l.server_id, l.short_description, l.is_featured,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description,
			c.name as category_name, c.slug as category_slug
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		LEFT JOIN server_discovery_listing_categories lc ON lc.listing_id = l.id
		LEFT JOIN server_discovery_categories c ON c.id = lc.category_id AND lc.category_id = (
			SELECT lc2.category_id FROM server_discovery_listing_categories lc2 WHERE lc2.listing_id = l.id ORDER BY c.sort_order LIMIT 1
		)
		WHERE %s
		ORDER BY l.id, %s
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), orderBy, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	serverMap := make(map[uuid.UUID]*models.ServerListingResult)

	for rows.Next() {
		var sr models.ServerListingResult
		var s models.Server
		var categoryName, categorySlug sql.NullString
		var region, iconURL, bannerURL, description sql.NullString

		err := rows.Scan(
			&sr.ID, &sr.ServerID, &sr.ShortDescription, &sr.IsFeatured,
			&sr.MemberCountSnapshot, &sr.OnlineCountSnapshot, &sr.WeeklyGrowthRate,
			&sr.EngagementScore, &region, &sr.Language, &sr.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
			&categoryName, &categorySlug,
		)
		if err != nil {
			return nil, 0, err
		}

		if region.Valid {
			sr.Region = &region.String
		}
		if iconURL.Valid {
			sr.IconURL = &iconURL.String
		}
		if bannerURL.Valid {
			sr.BannerURL = &bannerURL.String
		}
		if description.Valid {
			sr.Description = &description.String
		}

		sr.ID = sr.ServerID // Use server ID as listing ID for simplicity
		sr.Name = s.Name
		sr.MemberCount = sr.MemberCountSnapshot
		sr.OnlineCount = sr.OnlineCountSnapshot
		sr.IsVerified = false // TODO: Link to server verification

		if categorySlug.Valid {
			sr.Category = models.ServerCategory(categorySlug.String)
			sr.Categories = append(sr.Categories, sr.Category)
		}

		serverMap[sr.ServerID] = &sr
	}

	result := make([]*models.ServerListingResult, 0, len(serverMap))
	for _, v := range serverMap {
		result = append(result, v)
	}

	return result, total, nil
}

// GetServersByCategory returns servers for a specific category
func (r *DiscoveryRepository) GetServersByCategory(ctx context.Context, categorySlug string, limit, offset int) ([]*models.ServerListingResult, int, error) {
	filters := &models.DiscoveryFilters{
		Category: models.ServerCategory(categorySlug),
		Limit:    limit,
		Offset:   offset,
		SortBy:   "members",
	}
	return r.SearchServers(ctx, filters)
}

// GetCategories returns all active discovery categories
func (r *DiscoveryRepository) GetCategories(ctx context.Context) ([]*models.DiscoveryCategory, error) {
	query := `
		SELECT c.id, c.name, c.slug, c.icon, c.description, c.sort_order, c.is_active,
			COUNT(l.id) as server_count
		FROM server_discovery_categories c
		LEFT JOIN server_discovery_listing_categories lc ON lc.category_id = c.id
		LEFT JOIN server_discovery_listings l ON l.id = lc.listing_id AND l.is_listed = true AND l.approval_status = 'approved'
		WHERE c.is_active = true
		GROUP BY c.id
		ORDER BY c.sort_order
	`

	var categories []*models.DiscoveryCategory
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

// GetServerListing returns a discovery listing for a server
func (r *DiscoveryRepository) GetServerListing(ctx context.Context, serverID uuid.UUID) (*models.DiscoveryListing, error) {
	query := `
		SELECT id, server_id, short_description, is_listed, is_featured, featured_at,
			approval_status, approved_at, approved_by, rejection_reason,
			member_count_snapshot, online_count_snapshot, weekly_growth_rate,
			engagement_score, region, language, created_at, updated_at
		FROM server_discovery_listings
		WHERE server_id = $1
	`

	var listing models.DiscoveryListing
	err := r.db.GetContext(ctx, &listing, query, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &listing, nil
}

// GetServerListingByID returns a discovery listing by its ID
func (r *DiscoveryRepository) GetServerListingByID(ctx context.Context, listingID uuid.UUID) (*models.DiscoveryListing, error) {
	query := `
		SELECT id, server_id, short_description, is_listed, is_featured, featured_at,
			approval_status, approved_at, approved_by, rejection_reason,
			member_count_snapshot, online_count_snapshot, weekly_growth_rate,
			engagement_score, region, language, created_at, updated_at
		FROM server_discovery_listings
		WHERE id = $1
	`

	var listing models.DiscoveryListing
	err := r.db.GetContext(ctx, &listing, query, listingID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &listing, nil
}

// CreateListing creates a new discovery listing
func (r *DiscoveryRepository) CreateListing(ctx context.Context, listing *models.DiscoveryListing) error {
	query := `
		INSERT INTO server_discovery_listings (
			id, server_id, short_description, is_listed, approval_status,
			member_count_snapshot, language, region, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		listing.ID, listing.ServerID, listing.ShortDescription, listing.IsListed,
		listing.ApprovalStatus, listing.MemberCountSnapshot, listing.Language,
		listing.Region, listing.CreatedAt, listing.UpdatedAt,
	)
	return err
}

// UpdateListing updates a discovery listing
func (r *DiscoveryRepository) UpdateListing(ctx context.Context, listing *models.DiscoveryListing) error {
	query := `
		UPDATE server_discovery_listings SET
			short_description = $2, is_listed = $3, is_featured = $4, featured_at = $5,
			approval_status = $6, approved_at = $7, approved_by = $8, rejection_reason = $9,
			member_count_snapshot = $10, online_count_snapshot = $11, weekly_growth_rate = $12,
			engagement_score = $13, region = $14, language = $15, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		listing.ID, listing.ShortDescription, listing.IsListed, listing.IsFeatured,
		listing.FeaturedAt, listing.ApprovalStatus, listing.ApprovedAt, listing.ApprovedBy,
		listing.RejectionReason, listing.MemberCountSnapshot, listing.OnlineCountSnapshot,
		listing.WeeklyGrowthRate, listing.EngagementScore, listing.Region, listing.Language,
	)
	return err
}

// SetListingCategories sets the categories for a listing
func (r *DiscoveryRepository) SetListingCategories(ctx context.Context, listingID uuid.UUID, categoryIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing
	_, err = tx.ExecContext(ctx, "DELETE FROM server_discovery_listing_categories WHERE listing_id = $1", listingID)
	if err != nil {
		return err
	}

	// Insert new
	for _, catID := range categoryIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO server_discovery_listing_categories (listing_id, category_id) VALUES ($1, $2)",
			listingID, catID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetListingCategories returns category IDs for a listing
func (r *DiscoveryRepository) GetListingCategories(ctx context.Context, listingID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT category_id FROM server_discovery_listing_categories WHERE listing_id = $1
	`
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids, query, listingID)
	return ids, err
}

// GetListingTags returns tags for a listing
func (r *DiscoveryRepository) GetListingTags(ctx context.Context, listingID uuid.UUID) ([]string, error) {
	query := `
		SELECT t.name FROM server_discovery_tags t
		JOIN server_discovery_listing_tags lt ON lt.tag_id = t.id
		WHERE lt.listing_id = $1
	`
	var tags []string
	err := r.db.SelectContext(ctx, &tags, query, listingID)
	return tags, err
}

// GetOrCreateTag gets or creates a tag by name
func (r *DiscoveryRepository) GetOrCreateTag(ctx context.Context, name string) (uuid.UUID, error) {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	var existing uuid.UUID
	err := r.db.GetContext(ctx, &existing, "SELECT id FROM server_discovery_tags WHERE slug = $1", slug)
	if err == nil {
		// Increment usage count
		r.db.ExecContext(ctx, "UPDATE server_discovery_tags SET usage_count = usage_count + 1 WHERE id = $1", existing)
		return existing, nil
	}

	var id uuid.UUID
	err = r.db.GetContext(ctx, &id,
		"INSERT INTO server_discovery_tags (id, name, slug) VALUES ($1, $2, $3) RETURNING id",
		uuid.New(), name, slug,
	)
	return id, err
}

// SetListingTags sets tags for a listing
func (r *DiscoveryRepository) SetListingTags(ctx context.Context, listingID uuid.UUID, tagNames []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing
	_, err = tx.ExecContext(ctx, "DELETE FROM server_discovery_listing_tags WHERE listing_id = $1", listingID)
	if err != nil {
		return err
	}

	// Insert new
	for _, tagName := range tagNames {
		tagID, err := r.getOrCreateTagTx(ctx, tx, tagName)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO server_discovery_listing_tags (listing_id, tag_id) VALUES ($1, $2)",
			listingID, tagID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *DiscoveryRepository) getOrCreateTagTx(ctx context.Context, tx *sqlx.Tx, name string) (uuid.UUID, error) {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	var existing uuid.UUID
	err := tx.GetContext(ctx, &existing, "SELECT id FROM server_discovery_tags WHERE slug = $1", slug)
	if err == nil {
		tx.ExecContext(ctx, "UPDATE server_discovery_tags SET usage_count = usage_count + 1 WHERE id = $1", existing)
		return existing, nil
	}

	var id uuid.UUID
	err = tx.GetContext(ctx, &id,
		"INSERT INTO server_discovery_tags (id, name, slug) VALUES ($1, $2, $3) RETURNING id",
		uuid.New(), name, slug,
	)
	return id, err
}

// GetCategoryBySlug returns a category by slug
func (r *DiscoveryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*models.DiscoveryCategory, error) {
	var cat models.DiscoveryCategory
	err := r.db.GetContext(ctx, &cat, `
		SELECT id, name, slug, icon, description, sort_order, is_active
		FROM server_discovery_categories WHERE slug = $1 AND is_active = true
	`, slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &cat, err
}

// GetCategoriesBySlug returns categories by slugs
func (r *DiscoveryRepository) GetCategoriesBySlug(ctx context.Context, slugs []string) ([]uuid.UUID, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`
		SELECT id FROM server_discovery_categories WHERE slug IN (?) AND is_active = true
	`, slugs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var ids []uuid.UUID
	err = r.db.SelectContext(ctx, &ids, query, args...)
	return ids, err
}

// CreateReport creates a discovery report
func (r *DiscoveryRepository) CreateReport(ctx context.Context, report *models.DiscoveryReport) error {
	query := `
		INSERT INTO server_discovery_reports (id, listing_id, reporter_id, reason, details, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		report.ID, report.ListingID, report.ReporterID, report.Reason,
		report.Details, report.Status, report.CreatedAt,
	)
	return err
}

// GetRecommendedServers returns personalized recommendations for a user
func (r *DiscoveryRepository) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerListingResult, error) {
	// Get user's joined servers' categories and suggest similar servers
	// This is a simplified algorithm - in production you'd use ML
	query := `
		SELECT DISTINCT ON (l.id)
			l.id, l.server_id, l.short_description, l.is_featured,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description,
			c.slug as category_slug
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		LEFT JOIN server_discovery_listing_categories lc ON lc.listing_id = l.id
		LEFT JOIN server_discovery_categories c ON c.id = lc.category_id
		WHERE l.is_listed = true AND l.approval_status = 'approved'
			AND l.server_id NOT IN (
				SELECT server_id FROM members WHERE user_id = $1
			)
		ORDER BY l.id, l.engagement_score DESC, l.weekly_growth_rate DESC
		LIMIT $2
	`

	rows, err := r.db.QueryxContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.ServerListingResult
	for rows.Next() {
		var sr models.ServerListingResult
		var s models.Server
		var categorySlug sql.NullString
		var region, iconURL, bannerURL, description sql.NullString

		err := rows.Scan(
			&sr.ID, &sr.ServerID, &sr.ShortDescription, &sr.IsFeatured,
			&sr.MemberCountSnapshot, &sr.OnlineCountSnapshot, &sr.WeeklyGrowthRate,
			&sr.EngagementScore, &region, &sr.Language, &sr.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
			&categorySlug,
		)
		if err != nil {
			return nil, err
		}

		if region.Valid {
			sr.Region = &region.String
		}
		if iconURL.Valid {
			sr.IconURL = &iconURL.String
		}
		if bannerURL.Valid {
			sr.BannerURL = &bannerURL.String
		}
		if description.Valid {
			s.Description = &description.String
		}

		sr.ID = sr.ServerID
		sr.Name = s.Name
		sr.MemberCount = sr.MemberCountSnapshot
		sr.OnlineCount = sr.OnlineCountSnapshot

		if categorySlug.Valid {
			sr.Category = models.ServerCategory(categorySlug.String)
			sr.Categories = append(sr.Categories, sr.Category)
		}

		results = append(results, &sr)
	}

	return results, nil
}
