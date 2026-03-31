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
	TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error
	GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error)
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
