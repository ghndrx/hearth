package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// Common errors
var (
	ErrDiscoverableServerNotFound = errors.New("discoverable server not found")
	ErrServerNotPublic            = errors.New("server is not public")
	ErrServerNotInDiscovery       = errors.New("server is not in discovery")
)

// MemberRepo interface for checking and adding server membership
type MemberRepo interface {
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	AddMember(ctx context.Context, member *models.Member) error
}

// DiscoverableServerRepo interface for discoverable server operations
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
}

// ServerRepo interface for basic server operations
type ServerRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
}

// DiscoverableServerService handles discoverable server business logic
type DiscoverableServerService struct {
	repo       DiscoverableServerRepo
	serverRepo ServerRepo
	memberRepo MemberRepo
}

// NewDiscoverableServerService creates a new discoverable server service
func NewDiscoverableServerService(
	repo DiscoverableServerRepo,
	serverRepo ServerRepo,
	memberRepo MemberRepo,
) *DiscoverableServerService {
	return &DiscoverableServerService{
		repo:       repo,
		serverRepo: serverRepo,
		memberRepo: memberRepo,
	}
}

// GetDiscoverableServers returns paginated discoverable servers
func (s *DiscoverableServerService) GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) (*models.PaginatedDiscoverableServers, error) {
	models.NormalizeDiscoverFilters(filters)

	servers, total, err := s.repo.GetDiscoverableServers(ctx, filters)
	if err != nil {
		return nil, err
	}

	totalPages := total / filters.Limit
	if total%filters.Limit > 0 {
		totalPages++
	}

	return &models.PaginatedDiscoverableServers{
		Servers:    servers,
		Total:      total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetFeaturedServers returns featured servers
func (s *DiscoverableServerService) GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	return s.repo.GetFeaturedServers(ctx, limit)
}

// GetServerByID returns a discoverable server by its ID
func (s *DiscoverableServerService) GetServerByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error) {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrDiscoverableServerNotFound
	}
	if !server.IsPublic {
		return nil, ErrServerNotPublic
	}
	return server, nil
}

// GetServerDetail returns detailed info about a discoverable server
func (s *DiscoverableServerService) GetServerDetail(ctx context.Context, id uuid.UUID) (*models.DiscoverableServerDetail, error) {
	server, err := s.GetServerByID(ctx, id)
	if err != nil {
		return nil, err
	}

	detail := &models.DiscoverableServerDetail{
		DiscoverableServer: *server,
	}

	// Get invite code
	inviteCode, err := s.repo.GetInviteCode(ctx, server.ServerID)
	if err == nil && inviteCode != "" {
		detail.InviteCode = &inviteCode
	}

	return detail, nil
}

// GetCategories returns all discovery categories
func (s *DiscoverableServerService) GetCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	return s.repo.GetCategories(ctx)
}

// CanJoinServer checks if a user can join a discoverable server
func (s *DiscoverableServerService) CanJoinServer(ctx context.Context, serverID, userID uuid.UUID) error {
	// Check if server exists and is public
	server, err := s.repo.GetByServerID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrDiscoverableServerNotFound
	}
	if !server.IsPublic {
		return ErrServerNotPublic
	}

	// Check if user is already a member
	member, err := s.memberRepo.GetMember(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if member != nil {
		return ErrAlreadyMember
	}

	return nil
}

// GetServerByServerID returns a discoverable server by its server ID
func (s *DiscoverableServerService) GetServerByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error) {
	server, err := s.repo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrDiscoverableServerNotFound
	}
	if !server.IsPublic {
		return nil, ErrServerNotPublic
	}
	return server, nil
}

// JoinServer adds a user to a discoverable server
func (s *DiscoverableServerService) JoinServer(ctx context.Context, serverID, userID uuid.UUID) error {
	// Check if server exists and is public
	server, err := s.repo.GetByServerID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrDiscoverableServerNotFound
	}
	if !server.IsPublic {
		return ErrServerNotPublic
	}

	// Check if user is already a member
	member, err := s.memberRepo.GetMember(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if member != nil {
		return ErrAlreadyMember
	}

	// Add the member
	newMember := &models.Member{
		UserID:   userID,
		ServerID: serverID,
		JoinedAt: time.Now(),
		Roles:    []uuid.UUID{},
	}

	return s.memberRepo.AddMember(ctx, newMember)
}

// SearchServersEnhanced performs enhanced search on discoverable servers
func (s *DiscoverableServerService) SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) (*models.DiscoverySearchResponse, error) {
	servers, total, err := s.repo.SearchServersEnhanced(ctx, req)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}

	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &models.DiscoverySearchResponse{
		Servers:    servers,
		Total:      total,
		Page:       req.Page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetTrendingServers returns trending servers
func (s *DiscoverableServerService) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	return s.repo.GetTrendingServers(ctx, limit)
}

// GetRecommendedServers returns personalized recommendations for a user
func (s *DiscoverableServerService) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	return s.repo.GetRecommendedServers(ctx, userID, limit)
}

// GetCategoriesWithStats returns categories with statistics
func (s *DiscoverableServerService) GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	return s.repo.GetCategoriesWithStats(ctx)
}

// GetPopularTags returns popular discovery tags
func (s *DiscoverableServerService) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	return s.repo.GetPopularTags(ctx, limit)
}

// GetDiscoveryStats returns overall discovery statistics
func (s *DiscoverableServerService) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	return s.repo.GetDiscoveryStats(ctx)
}

// GetSearchSuggestions returns search suggestions
func (s *DiscoverableServerService) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	return s.repo.GetSearchSuggestions(ctx, query, limit)
}

// GetDiscoveryHomePage returns the full discovery home page data
func (s *DiscoverableServerService) GetDiscoveryHomePage(ctx context.Context, userID uuid.UUID, featuredLimit, trendingLimit, recommendedLimit int) (*models.DiscoveryHomePage, error) {
	// Fetch all data in parallel (simplified for now)
	featured, err := s.GetFeaturedServers(ctx, featuredLimit)
	if err != nil {
		featured = []*models.DiscoverableFeaturedServer{}
	}

	trending, err := s.GetTrendingServers(ctx, trendingLimit)
	if err != nil {
		trending = []*models.TrendingServerInfo{}
	}

	var recommended []*models.ServerRecommendation
	if userID != uuid.Nil {
		recommended, err = s.GetRecommendedServers(ctx, userID, recommendedLimit)
		if err != nil {
			recommended = []*models.ServerRecommendation{}
		}
	}

	categories, err := s.GetCategoriesWithStats(ctx)
	if err != nil {
		categories = []*models.CategoryWithStats{}
	}

	tags, err := s.GetPopularTags(ctx, 10)
	if err != nil {
		tags = []*models.DiscoveryTag{}
	}

	stats, err := s.GetDiscoveryStats(ctx)
	if err != nil {
		stats = &models.DiscoveryPageStats{}
	}

	return &models.DiscoveryHomePage{
		Featured:      featured,
		Trending:      trending,
		Recommended:   recommended,
		Categories:    categories,
		PopularTags:   tags,
		Stats:         stats,
	}, nil
}
