package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
			l.id, l.server_id, l.short_description, l.is_featured, l.is_verified,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description,
			c.name as category_name, c.slug as category_slug,
			COALESCE(ds.is_verified, false) as is_verified
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		LEFT JOIN discoverable_servers ds ON ds.server_id = l.server_id
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

		var isVerified bool
		err := rows.Scan(
			&sr.ID, &sr.ServerID, &sr.ShortDescription, &sr.IsFeatured, &sr.IsVerified,
			&sr.MemberCountSnapshot, &sr.OnlineCountSnapshot, &sr.WeeklyGrowthRate,
			&sr.EngagementScore, &region, &sr.Language, &sr.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
			&categoryName, &categorySlug, &isVerified,
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
		sr.IsVerified = isVerified

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

// GetTrendingServers returns trending servers based on growth rate and engagement
func (r *DiscoveryRepository) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServer, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			l.id, l.server_id, l.short_description, l.is_featured,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description,
			c.slug as category_slug,
			-- Calculate trend score: weighted combination of growth and engagement
			(l.weekly_growth_rate * 0.6 + l.engagement_score * 0.4) as trend_score,
			NOW() as last_trend_at
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		LEFT JOIN server_discovery_listing_categories lc ON lc.listing_id = l.id
		LEFT JOIN server_discovery_categories c ON c.id = lc.category_id
		WHERE l.is_listed = true AND l.approval_status = 'approved'
			AND l.weekly_growth_rate > 0
		ORDER BY trend_score DESC, l.weekly_growth_rate DESC
		LIMIT $1
	`

	rows, err := r.db.QueryxContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seenServers := make(map[uuid.UUID]bool)
	var results []*models.TrendingServer

	for rows.Next() {
		var ts models.TrendingServer
		var s models.Server
		var categorySlug sql.NullString
		var region, iconURL, bannerURL, description sql.NullString

		err := rows.Scan(
			&ts.ID, &ts.ServerID, &ts.ShortDescription, &ts.IsFeatured,
			&ts.MemberCountSnapshot, &ts.OnlineCountSnapshot, &ts.WeeklyGrowthRate,
			&ts.EngagementScore, &region, &ts.Language, &ts.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
			&categorySlug, &ts.TrendScore, &ts.LastTrendAt,
		)
		if err != nil {
			return nil, err
		}

		// Skip duplicate servers
		if seenServers[ts.ServerID] {
			continue
		}
		seenServers[ts.ServerID] = true

		if region.Valid {
			ts.Region = &region.String
		}
		if iconURL.Valid {
			ts.IconURL = &iconURL.String
		}
		if bannerURL.Valid {
			ts.BannerURL = &bannerURL.String
		}
		if description.Valid {
			ts.Description = &description.String
		}

		ts.ID = ts.ServerID
		ts.Name = s.Name
		ts.MemberCount = ts.MemberCountSnapshot
		ts.OnlineCount = ts.OnlineCountSnapshot
		ts.GrowthPercentage = ts.WeeklyGrowthRate

		if categorySlug.Valid {
			ts.Category = models.ServerCategory(categorySlug.String)
			ts.Categories = append(ts.Categories, ts.Category)
		}

		results = append(results, &ts)
	}

	return results, nil
}

// GetDiscoveryStats returns overall discovery statistics
func (r *DiscoveryRepository) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryStats, error) {
	var stats struct {
		TotalServers    int   `db:"total_servers"`
		TotalCategories int   `db:"total_categories"`
		TotalMembers    int64 `db:"total_members"`
	}

	statsQuery := `
		SELECT 
			(SELECT COUNT(*) FROM server_discovery_listings WHERE is_listed = true AND approval_status = 'approved') as total_servers,
			(SELECT COUNT(*) FROM server_discovery_categories WHERE is_active = true) as total_categories,
			(SELECT COALESCE(SUM(member_count_snapshot), 0) FROM server_discovery_listings WHERE is_listed = true AND approval_status = 'approved') as total_members
	`

	err := r.db.GetContext(ctx, &stats, statsQuery)
	if err != nil {
		return nil, err
	}

	return &models.DiscoveryStats{
		TotalServers:    int64(stats.TotalServers),
		TotalCategories: stats.TotalCategories,
		TotalMembers:    stats.TotalMembers,
	}, nil
}

// GetPopularTags returns the most popular discovery tags
func (r *DiscoveryRepository) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT t.id, t.name, t.slug, t.usage_count
		FROM server_discovery_tags t
		WHERE t.usage_count > 0
		ORDER BY t.usage_count DESC
		LIMIT $1
	`

	var tags []*models.DiscoveryTag
	err := r.db.SelectContext(ctx, &tags, query, limit)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// GetServersByTags returns servers matching the given tags
func (r *DiscoveryRepository) GetServersByTags(ctx context.Context, tags []string, limit, offset int) ([]*models.ServerListingResult, int, error) {
	if len(tags) == 0 {
		return nil, 0, nil
	}

	// Normalize tags
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		normalizedTags[i] = strings.ToLower(strings.ReplaceAll(tag, " ", "-"))
	}

	query, args, err := sqlx.In(`
		SELECT DISTINCT l.id, l.server_id, l.short_description, l.is_featured,
			l.member_count_snapshot, l.online_count_snapshot, l.weekly_growth_rate,
			l.engagement_score, l.region, l.language, l.created_at,
			s.name, s.icon_url, s.banner_url, s.description
		FROM server_discovery_listings l
		JOIN servers s ON s.id = l.server_id
		JOIN server_discovery_listing_tags lt ON lt.listing_id = l.id
		JOIN server_discovery_tags t ON t.id = lt.tag_id
		WHERE l.is_listed = true AND l.approval_status = 'approved'
			AND t.slug IN (?)
		ORDER BY l.engagement_score DESC
	`, normalizedTags)
	if err != nil {
		return nil, 0, err
	}
	query = r.db.Rebind(query)

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT l.id)
		FROM server_discovery_listings l
		JOIN server_discovery_listing_tags lt ON lt.listing_id = l.id
		JOIN server_discovery_tags t ON t.id = lt.tag_id
		WHERE l.is_listed = true AND l.approval_status = 'approved'
			AND t.slug IN (%s)
	`, strings.Join(make([]string, len(normalizedTags)), ","))

	var total int
	err = r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Apply limit and offset
	args = append(args, limit, offset)
	query += " LIMIT ? OFFSET ?"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*models.ServerListingResult
	for rows.Next() {
		var sr models.ServerListingResult
		var s models.Server
		var region, iconURL, bannerURL, description sql.NullString

		err := rows.Scan(
			&sr.ID, &sr.ServerID, &sr.ShortDescription, &sr.IsFeatured,
			&sr.MemberCountSnapshot, &sr.OnlineCountSnapshot, &sr.WeeklyGrowthRate,
			&sr.EngagementScore, &region, &sr.Language, &sr.CreatedAt,
			&s.Name, &iconURL, &bannerURL, &description,
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

		sr.ID = sr.ServerID
		sr.Name = s.Name
		sr.MemberCount = sr.MemberCountSnapshot
		sr.OnlineCount = sr.OnlineCountSnapshot

		results = append(results, &sr)
	}

	return results, total, nil
}

// SearchServersEnhanced searches servers with enhanced filters including tags and online_only
func (r *DiscoveryRepository) SearchServersEnhanced(ctx context.Context, filters *models.DiscoveryFilters) ([]*models.ServerListingResult, int, error) {
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

	// Online only filter - requires online_count_snapshot > 0
	if filters.OnlineOnly {
		conditions = append(conditions, "l.online_count_snapshot > 0")
	}

	// Tags filter
	if len(filters.Tags) > 0 {
		tagConditions := make([]string, len(filters.Tags))
		for i, tag := range filters.Tags {
			slug := strings.ToLower(strings.ReplaceAll(tag, " ", "-"))
			tagConditions[i] = fmt.Sprintf("EXISTS (SELECT 1 FROM server_discovery_listing_tags lt2 JOIN server_discovery_tags t2 ON t2.id = lt2.tag_id WHERE lt2.listing_id = l.id AND t2.slug = $%d)", argNum)
			args = append(args, slug)
			argNum++
		}
		conditions = append(conditions, "("+strings.Join(tagConditions, " OR ")+")")
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
	case "online":
		orderBy = "l.online_count_snapshot DESC"
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
			l.id, l.server_id, l.short_description, l.is_featured, l.is_verified,
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
			&sr.ID, &sr.ServerID, &sr.ShortDescription, &sr.IsFeatured, &sr.IsVerified,
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

		sr.ID = sr.ServerID
		sr.Name = s.Name
		sr.MemberCount = sr.MemberCountSnapshot
		sr.OnlineCount = sr.OnlineCountSnapshot

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
	GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error)
	GetCategories(ctx context.Context) ([]*models.CategoryInfo, error)
	SearchServers(ctx context.Context, query string, category models.ServerDiscoveryCategory, page, limit int) ([]*models.DiscoverableServerSearchResult, int, error)
	SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) ([]*models.DiscoverableServerSearchResult, int, error)
	GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error)
	GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error)
	GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error)
	GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error)
	GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error)
	GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error)
	GetInviteCode(ctx context.Context, serverID uuid.UUID) (string, error)
	Create(ctx context.Context, server *models.DiscoverableServer) error
	Update(ctx context.Context, server *models.DiscoverableServer) error
	Delete(ctx context.Context, id uuid.UUID) error
	TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error
	GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error)
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
		conditions = append(conditions, "(LOWER(name) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+") OR LOWER(description) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+"))")
		args = append(args, "%"+filters.Query+"%")
		argIdx++
	}

	if filters.Category != "" {
		conditions = append(conditions, "category = $"+fmt.Sprintf("%d", argIdx))
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
		       tags, member_count, is_verified, is_featured, created_at
		FROM discoverable_servers
		WHERE ` + whereClause + `
		ORDER BY member_count DESC, is_verified DESC, name ASC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

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
		       tags, member_count, is_verified, featured_at, created_at
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
		       tags, member_count, is_verified, is_public, is_featured, featured_at,
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
		       tags, member_count, is_verified, is_public, is_featured, featured_at,
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
		conditions = append(conditions, "(LOWER(name) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+") OR LOWER(description) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+"))")
		args = append(args, "%"+query+"%")
		argIdx++
	}

	if category != "" {
		conditions = append(conditions, "category = $"+fmt.Sprintf("%d", argIdx))
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
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, limit, offset)

	var servers []*models.DiscoverableServerSearchResult
	err = r.db.SelectContext(ctx, &servers, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// SearchServersEnhanced performs enhanced search with filters and sorting
func (r *DiscoverableServerRepository) SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) ([]*models.DiscoverableServerSearchResult, int, error) {
	// Normalize request
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 25
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "is_public = true")

	// Query filter
	if req.Query != "" {
		conditions = append(conditions, "(LOWER(name) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+") OR LOWER(description) LIKE LOWER($"+fmt.Sprintf("%d", argIdx)+"))")
		args = append(args, "%"+req.Query+"%")
		argIdx++
	}

	// Category filter
	if req.Category != "" {
		conditions = append(conditions, "category = $"+fmt.Sprintf("%d", argIdx))
		args = append(args, string(req.Category))
		argIdx++
	}

	// Multiple categories filter
	if len(req.Categories) > 0 {
		placeholders := make([]string, len(req.Categories))
		for i, cat := range req.Categories {
			placeholders[i] = "$" + fmt.Sprintf("%d", argIdx)
			args = append(args, string(cat))
			argIdx++
		}
		conditions = append(conditions, "category IN ("+strings.Join(placeholders, ",")+")")
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := "SELECT COUNT(*) FROM discoverable_servers WHERE " + whereClause
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Sorting
	orderBy := "member_count DESC, is_verified DESC"
	switch req.SortBy {
	case "new":
		orderBy = "created_at DESC"
	case "active":
		orderBy = "member_count DESC" // Using member_count as proxy for activity
	case "name":
		orderBy = "name ASC"
	default:
		orderBy = "member_count DESC, is_verified DESC"
	}

	if req.SortOrder == "asc" {
		orderBy = strings.Replace(orderBy, " DESC", " ASC", 1)
	}

	offset := (req.Page - 1) * req.Limit

	selectQuery := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, is_featured, created_at
		FROM discoverable_servers
		WHERE ` + whereClause + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, req.Limit, offset)

	var servers []*models.DiscoverableServerSearchResult
	err = r.db.SelectContext(ctx, &servers, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// GetTrendingServers returns trending servers based on growth metrics
func (r *DiscoverableServerRepository) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// For now, we'll use member_count growth as a proxy for trending
	// In a production system, you'd track historical member counts
	query := `
		SELECT id, server_id, name, description, category, icon_url, banner_url,
		       member_count, is_verified, is_featured, created_at
		FROM discoverable_servers
		WHERE is_public = true
		ORDER BY member_count DESC, is_featured DESC, created_at DESC
		LIMIT $1
	`

	var servers []*models.DiscoverableServerSearchResult
	err := r.db.SelectContext(ctx, &servers, query, limit)
	if err != nil {
		return nil, err
	}

	// Convert to trending info (simplified - real implementation would have trend metrics)
	result := make([]*models.TrendingServerInfo, len(servers))
	for i, s := range servers {
		// Generate mock trend data based on position
		trendScore := float64(len(servers) - i)
		result[i] = &models.TrendingServerInfo{
			Server:     s,
			TrendScore: trendScore,
			GrowthRate: trendScore * 2.5,
			RankChange: 0,
		}
	}

	return result, nil
}

// GetRecommendedServers returns personalized recommendations for a user
func (r *DiscoverableServerRepository) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Enhanced recommendation algorithm that considers:
	// 1. Servers in categories the user frequently engages with
	// 2. Mutual members with servers the user has joined
	// 3. Featured and trending servers not yet joined
	// 4. Similar servers based on user's existing server categories

	// First, get the user's joined server categories to understand their interests
	categoryQuery := `
		SELECT DISTINCT ds.category
		FROM discoverable_servers ds
		JOIN members m ON m.server_id = ds.server_id
		WHERE m.user_id = $1 AND ds.is_public = true
	`
	var userCategories []string
	err := r.db.SelectContext(ctx, &userCategories, categoryQuery, userID)
	if err != nil {
		// Fallback to basic recommendations if this fails
		return r.getBasicRecommendations(ctx, userID, limit)
	}

	// If user has no interests yet, return featured/popular servers
	if len(userCategories) == 0 {
		return r.getBasicRecommendations(ctx, userID, limit)
	}

	// Build recommendation query based on user's categories
	// This prioritizes servers in categories the user already engages with
	categoryPlaceholders := make([]string, len(userCategories))
	for i := range userCategories {
		categoryPlaceholders[i] = fmt.Sprintf("$%d", i+2)
	}

	query := fmt.Sprintf(`
		SELECT ds.id, ds.server_id, ds.name, ds.description, ds.category, ds.icon_url, ds.banner_url,
		       ds.tags, ds.member_count, ds.is_verified, ds.is_featured, ds.created_at,
		       -- Score based on category match and popularity
		       CASE WHEN ds.category IN (%s) THEN 100 ELSE 0 END +
		       CASE WHEN ds.is_featured THEN 50 ELSE 0 END +
		       CASE WHEN ds.is_verified THEN 25 ELSE 0 END +
		       LEAST(ds.member_count / 100, 50) as relevance_score
		FROM discoverable_servers ds
		WHERE ds.is_public = true
			AND ds.server_id NOT IN (
				SELECT server_id FROM members WHERE user_id = $1
			)
		ORDER BY relevance_score DESC, ds.member_count DESC, ds.is_featured DESC
		LIMIT $%d
	`, strings.Join(categoryPlaceholders, ","), len(userCategories)+2)

	args := make([]interface{}, 0, len(userCategories)+2)
	args = append(args, userID)
	args = append(args, interfaceSlice(userCategories)...)
	args = append(args, limit)

	var servers []*models.DiscoverableServerSearchResult
	err = r.db.SelectContext(ctx, &servers, query, args...)
	if err != nil {
		return nil, err
	}

	// Get mutual servers for recommendation context
	mutualServersQuery := `
		SELECT s.name FROM servers s
		JOIN members m ON m.server_id = s.id
		WHERE m.user_id = $1
		LIMIT 5
	`
	var mutualServerNames []string
	r.db.SelectContext(ctx, &mutualServerNames, mutualServersQuery, userID)

	// Convert to recommendations with contextual reasons
	result := make([]*models.ServerRecommendation, len(servers))
	for i, s := range servers {
		reason := r.getRecommendationReason(s, userCategories, mutualServerNames)
		result[i] = &models.ServerRecommendation{
			DiscoverableServerSearchResult: *s,
			Reason:                         reason,
			MutualMemberCount:              0, // Would require a more complex query to calculate accurately
			MutualServers:                  nil,
		}
	}

	return result, nil
}

// getBasicRecommendations returns basic recommendations when user has no interests
func (r *DiscoverableServerRepository) getBasicRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	query := `
		SELECT ds.id, ds.server_id, ds.name, ds.description, ds.category, ds.icon_url, ds.banner_url,
		       ds.tags, ds.member_count, ds.is_verified, ds.is_featured, ds.created_at
		FROM discoverable_servers ds
		WHERE ds.is_public = true
			AND ds.server_id NOT IN (
				SELECT server_id FROM members WHERE user_id = $1
			)
		ORDER BY ds.is_featured DESC, ds.member_count DESC, ds.is_verified DESC
		LIMIT $2
	`

	var servers []*models.DiscoverableServerSearchResult
	err := r.db.SelectContext(ctx, &servers, query, userID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*models.ServerRecommendation, len(servers))
	reasons := []string{
		"Popular server",
		"Featured community",
		"Trending server",
		"Highly active",
		"Recommended for you",
	}
	for i, s := range servers {
		result[i] = &models.ServerRecommendation{
			DiscoverableServerSearchResult: *s,
			Reason:                         reasons[i%len(reasons)],
		}
	}

	return result, nil
}

// getRecommendationReason determines the reason for a recommendation
func (r *DiscoverableServerRepository) getRecommendationReason(server *models.DiscoverableServerSearchResult, userCategories []string, mutualServers []string) string {
	// Check if server is in user's preferred categories
	for _, cat := range userCategories {
		if string(server.Category) == cat {
			if len(mutualServers) > 0 {
				return "Similar to servers you're in"
			}
			return "Popular in categories you like"
		}
	}

	if server.IsFeatured {
		return "Featured community"
	}

	if server.IsVerified {
		return "Verified server"
	}

	return "Recommended for you"
}

// interfaceSlice converts a slice of strings to []interface{}
func interfaceSlice(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// GetCategoriesWithStats returns categories with additional statistics
func (r *DiscoverableServerRepository) GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	query := `
		SELECT 
			category as name, 
			category as slug, 
			COUNT(*) as server_count,
			COALESCE(SUM(member_count), 0) as total_members,
			COALESCE(AVG(member_count)::float, 0) as avg_member_count,
			0.0 as growth_rate
		FROM discoverable_servers
		WHERE is_public = true
		GROUP BY category
		ORDER BY server_count DESC
	`

	var categories []*models.CategoryWithStats
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

// GetPopularTags returns popular discovery tags
func (r *DiscoverableServerRepository) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	if limit <= 0 {
		limit = 20
	}

	// Return default popular tags since we don't have a tags table in discoverable_servers
	defaultTags := []*models.DiscoveryTag{
		{Name: "Gaming", Slug: "gaming", UsageCount: 1234},
		{Name: "Music", Slug: "music", UsageCount: 987},
		{Name: "Art", Slug: "art", UsageCount: 876},
		{Name: "Technology", Slug: "technology", UsageCount: 765},
		{Name: "Education", Slug: "education", UsageCount: 654},
		{Name: "Entertainment", Slug: "entertainment", UsageCount: 543},
		{Name: "Social", Slug: "social", UsageCount: 432},
		{Name: "Sports", Slug: "sports", UsageCount: 321},
	}

	if limit < len(defaultTags) {
		return defaultTags[:limit], nil
	}
	return defaultTags, nil
}

// GetDiscoveryStats returns overall discovery statistics
func (r *DiscoverableServerRepository) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	var stats struct {
		TotalServers       int64 `db:"total_servers"`
		TotalMembers       int64 `db:"total_members"`
		TotalCategories    int   `db:"total_categories"`
		NewServersThisWeek int   `db:"new_servers_this_week"`
	}

	statsQuery := `
		SELECT 
			COUNT(*) as total_servers,
			COALESCE(SUM(member_count), 0) as total_members,
			COUNT(DISTINCT category) as total_categories,
			0 as new_servers_this_week
		FROM discoverable_servers
		WHERE is_public = true
	`

	err := r.db.GetContext(ctx, &stats, statsQuery)
	if err != nil {
		return nil, err
	}

	// Get new servers this week
	weekQuery := `
		SELECT COUNT(*) FROM discoverable_servers
		WHERE is_public = true AND created_at > NOW() - INTERVAL '7 days'
	`
	r.db.GetContext(ctx, &stats.NewServersThisWeek, weekQuery)

	return &models.DiscoveryPageStats{
		TotalServers:       stats.TotalServers,
		TotalMembers:       stats.TotalMembers,
		TotalCategories:    stats.TotalCategories,
		NewServersThisWeek: stats.NewServersThisWeek,
	}, nil
}

// GetSearchSuggestions returns search suggestions based on partial query
func (r *DiscoverableServerRepository) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if query == "" {
		return nil, nil
	}

	// Search for matching server names
	serverQuery := `
		SELECT name FROM discoverable_servers
		WHERE is_public = true AND LOWER(name) LIKE LOWER($1)
		LIMIT $2
	`

	var serverNames []string
	err := r.db.SelectContext(ctx, &serverNames, serverQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}

	suggestions := make([]*models.SearchSuggestion, len(serverNames))
	for i, name := range serverNames {
		suggestions[i] = &models.SearchSuggestion{
			Type:  "server",
			Value: name,
		}
	}

	return suggestions, nil
}

// Create inserts a new discoverable server listing
func (r *DiscoverableServerRepository) Create(ctx context.Context, server *models.DiscoverableServer) error {
	query := `
		INSERT INTO discoverable_servers (id, server_id, name, description, category, icon_url, banner_url, tags, member_count, is_verified, is_public, is_featured)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		server.ID, server.ServerID, server.Name, server.Description,
		string(server.Category), server.IconURL, server.BannerURL,
		pq.Array(server.Tags), server.MemberCount, server.IsVerified,
		server.IsPublic, server.IsFeatured,
	).Scan(&server.CreatedAt, &server.UpdatedAt)
}

// Update updates an existing discoverable server listing
func (r *DiscoverableServerRepository) Update(ctx context.Context, server *models.DiscoverableServer) error {
	query := `
		UPDATE discoverable_servers
		SET name = $1, description = $2, category = $3, icon_url = $4,
		    banner_url = $5, tags = $6, member_count = $7
		WHERE id = $8
		RETURNING updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		server.Name, server.Description, string(server.Category),
		server.IconURL, server.BannerURL, pq.Array(server.Tags),
		server.MemberCount, server.ID,
	).Scan(&server.UpdatedAt)
}

// Delete removes a discoverable server listing
func (r *DiscoverableServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM discoverable_servers WHERE id = $1", id)
	return err
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

// TrackActivity records a discovery activity event for a server
func (r *DiscoverableServerRepository) TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error {
	query := `
		INSERT INTO discovery_activity (server_id, user_id, activity_type, source)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, serverID, userID, activityType, source)
	return err
}

// GetServerDailyStats returns daily discovery stats for a server
func (r *DiscoverableServerRepository) GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error) {
	query := `
		SELECT id, server_id, stat_date, views, impressions, joins, search_clicks
		FROM discovery_daily_stats
		WHERE server_id = $1 AND stat_date >= CURRENT_DATE - INTERVAL '1 day' * $2
		ORDER BY stat_date DESC
	`

	var stats []*models.ServerDiscoveryDailyStats
	err := r.db.SelectContext(ctx, &stats, query, serverID, days)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
