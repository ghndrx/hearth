package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	Create(ctx context.Context, server *models.DiscoverableServer) error
	Update(ctx context.Context, server *models.DiscoverableServer) error
	Delete(ctx context.Context, id uuid.UUID) error
	TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error
	GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error)
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

// RegisterServer registers a server for public discovery (server owner only)
func (s *DiscoverableServerService) RegisterServer(ctx context.Context, serverID, ownerID uuid.UUID, req *models.RegisterServerRequest) (*models.DiscoverableServer, error) {
	// Verify the server exists
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	// Verify the user is the server owner
	if server.OwnerID != ownerID {
		return nil, ErrNotServerOwner
	}

	// Validate category
	if !models.IsValidCategory(string(req.Category)) {
		return nil, errors.New("invalid category")
	}

	// Check if already registered
	existing, err := s.repo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("server already registered for discovery")
	}

	desc := req.Description
	ds := &models.DiscoverableServer{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        req.Name,
		Description: &desc,
		Category:    req.Category,
		IconURL:     server.IconURL,
		BannerURL:   server.BannerURL,
		Tags:        pq.StringArray(req.Tags),
		MemberCount: 0,
		IsVerified:  false,
		IsPublic:    true,
		IsFeatured:  false,
	}

	if err := s.repo.Create(ctx, ds); err != nil {
		return nil, err
	}

	return ds, nil
}

// UpdateRegisteredServer updates a server's discovery listing (server owner only)
func (s *DiscoverableServerService) UpdateRegisteredServer(ctx context.Context, id, ownerID uuid.UUID, req *models.UpdateDiscoverableServerRequest) (*models.DiscoverableServer, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil || ds == nil {
		return nil, ErrDiscoverableServerNotFound
	}

	// Verify ownership
	server, err := s.serverRepo.GetByID(ctx, ds.ServerID)
	if err != nil || server == nil {
		return nil, ErrServerNotFound
	}
	if server.OwnerID != ownerID {
		return nil, ErrNotServerOwner
	}

	if req.Name != nil {
		ds.Name = *req.Name
	}
	if req.Description != nil {
		ds.Description = req.Description
	}
	if req.Category != nil {
		if !models.IsValidCategory(string(*req.Category)) {
			return nil, errors.New("invalid category")
		}
		ds.Category = *req.Category
	}
	if req.Tags != nil {
		ds.Tags = pq.StringArray(req.Tags)
	}

	if err := s.repo.Update(ctx, ds); err != nil {
		return nil, err
	}

	return ds, nil
}

// DeleteRegisteredServer removes a server from discovery (server owner only)
func (s *DiscoverableServerService) DeleteRegisteredServer(ctx context.Context, id, ownerID uuid.UUID) error {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil || ds == nil {
		return ErrDiscoverableServerNotFound
	}

	server, err := s.serverRepo.GetByID(ctx, ds.ServerID)
	if err != nil || server == nil {
		return ErrServerNotFound
	}
	if server.OwnerID != ownerID {
		return ErrNotServerOwner
	}

	return s.repo.Delete(ctx, id)
}

// SetServerPublicStatus is an admin method to force-set a server's public status
func (s *DiscoverableServerService) SetServerPublicStatus(ctx context.Context, id uuid.UUID, isPublic bool) error {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ds == nil {
		return ErrDiscoverableServerNotFound
	}

	ds.IsPublic = isPublic
	return s.repo.Update(ctx, ds)
}

// validActivityTypes are the allowed activity types for discovery tracking
var validActivityTypes = map[string]bool{
	"view":         true,
	"impression":   true,
	"join":         true,
	"search_click": true,
}

// validDiscoverySources are the allowed discovery sources
var validDiscoverySources = map[string]bool{
	"home":        true,
	"search":      true,
	"category":    true,
	"trending":    true,
	"recommended": true,
	"featured":    true,
}

// TrackActivity records a discovery activity event for a server
func (s *DiscoverableServerService) TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error {
	if !validActivityTypes[activityType] {
		return fmt.Errorf("invalid activity type: %s", activityType)
	}
	if source != "" && !validDiscoverySources[source] {
		return fmt.Errorf("invalid discovery source: %s", source)
	}
	return s.repo.TrackActivity(ctx, serverID, userID, activityType, source)
}

// GetServerDailyStats returns daily discovery stats for a server
func (s *DiscoverableServerService) GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error) {
	return s.repo.GetServerDailyStats(ctx, serverID, days)
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
