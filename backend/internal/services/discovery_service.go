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
	ErrDiscoveryListingNotFound = errors.New("discovery listing not found")
	ErrDiscoveryListingExists   = errors.New("server already has a discovery listing")
	ErrDiscoveryNotOwner        = errors.New("only server owner can modify discovery settings")
	ErrNotDiscoveryAdmin        = errors.New("admin permission required for this action")
	ErrServerNotListed          = errors.New("server is not listed in discovery")
	ErrInvalidCategory          = errors.New("invalid category")
	ErrDiscoveryReportNotFound  = errors.New("report not found")
)

// DiscoveryRepo defines discovery data operations
type DiscoveryRepo interface {
	GetFeaturedServers(ctx context.Context, limit int) ([]*models.FeaturedServer, error)
	SearchServers(ctx context.Context, filters *models.DiscoveryFilters) ([]*models.ServerListingResult, int, error)
	GetServersByCategory(ctx context.Context, categorySlug string, limit, offset int) ([]*models.ServerListingResult, int, error)
	GetCategories(ctx context.Context) ([]*models.DiscoveryCategory, error)
	GetServerListing(ctx context.Context, serverID uuid.UUID) (*models.DiscoveryListing, error)
	GetServerListingByID(ctx context.Context, listingID uuid.UUID) (*models.DiscoveryListing, error)
	CreateListing(ctx context.Context, listing *models.DiscoveryListing) error
	UpdateListing(ctx context.Context, listing *models.DiscoveryListing) error
	SetListingCategories(ctx context.Context, listingID uuid.UUID, categoryIDs []uuid.UUID) error
	GetListingCategories(ctx context.Context, listingID uuid.UUID) ([]uuid.UUID, error)
	GetListingTags(ctx context.Context, listingID uuid.UUID) ([]string, error)
	SetListingTags(ctx context.Context, listingID uuid.UUID, tagNames []string) error
	GetCategoryBySlug(ctx context.Context, slug string) (*models.DiscoveryCategory, error)
	GetCategoriesBySlug(ctx context.Context, slugs []string) ([]uuid.UUID, error)
	CreateReport(ctx context.Context, report *models.DiscoveryReport) error
	GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerListingResult, error)
}

// DiscoveryServerRepo for discovery (subset interface)
type DiscoveryServerRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error)
	GetOnlineCount(ctx context.Context, serverID uuid.UUID) (int, error)
	GetPublicInviteCode(ctx context.Context, serverID uuid.UUID) (string, error)
}

// DiscoveryInviteRepo for discovery
type DiscoveryInviteRepo interface {
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error)
}

// DiscoveryService handles server discovery business logic
type DiscoveryService struct {
	repo        DiscoveryRepo
	serverRepo  DiscoveryServerRepo
	inviteRepo  DiscoveryInviteRepo
	permService *PermissionService
	eventBus    EventBus
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(
	repo DiscoveryRepo,
	serverRepo DiscoveryServerRepo,
	inviteRepo DiscoveryInviteRepo,
	permService *PermissionService,
	eventBus EventBus,
) *DiscoveryService {
	return &DiscoveryService{
		repo:        repo,
		serverRepo:  serverRepo,
		inviteRepo:  inviteRepo,
		permService: permService,
		eventBus:    eventBus,
	}
}

// GetFeaturedServers returns featured servers for the discovery page
func (s *DiscoveryService) GetFeaturedServers(ctx context.Context, limit int) ([]*models.FeaturedServer, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	return s.repo.GetFeaturedServers(ctx, limit)
}

// SearchServers searches servers with filters
func (s *DiscoveryService) SearchServers(ctx context.Context, filters *models.DiscoveryFilters) ([]*models.ServerListingResult, int, error) {
	return s.repo.SearchServers(ctx, filters)
}

// GetServersByCategory returns servers for a category
func (s *DiscoveryService) GetServersByCategory(ctx context.Context, category string, limit, offset int) ([]*models.ServerListingResult, int, error) {
	return s.repo.GetServersByCategory(ctx, category, limit, offset)
}

// GetCategories returns all discovery categories
func (s *DiscoveryService) GetCategories(ctx context.Context) ([]*models.DiscoveryCategory, error) {
	return s.repo.GetCategories(ctx)
}

// GetRecommendations returns personalized recommendations for a user
func (s *DiscoveryService) GetRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerListingResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.repo.GetRecommendedServers(ctx, userID, limit)
}

// SubmitForDiscovery submits a server for discovery listing
func (s *DiscoveryService) SubmitForDiscovery(ctx context.Context, serverID, userID uuid.UUID, req *models.SubmitDiscoveryRequest) error {
	// Check server exists
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil || server == nil {
		return ErrServerNotFound
	}

	// Check user is server owner
	if server.OwnerID != userID {
		return ErrDiscoveryNotOwner
	}

	// Check if listing already exists
	existing, err := s.repo.GetServerListing(ctx, serverID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrDiscoveryListingExists
	}

	// Validate categories
	categoryIDs, err := s.repo.GetCategoriesBySlug(ctx, toStringSlice(req.Categories))
	if err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return ErrInvalidCategory
	}

	// Get current member count
	memberCount, _ := s.serverRepo.GetMemberCount(ctx, serverID)
	onlineCount, _ := s.serverRepo.GetOnlineCount(ctx, serverID)

	language := req.Language
	if language == "" {
		language = "en"
	}

	listing := &models.DiscoveryListing{
		ID:                  uuid.New(),
		ServerID:            serverID,
		ShortDescription:    req.ShortDescription,
		IsListed:            false, // Requires approval
		IsFeatured:          false,
		ApprovalStatus:      models.ApprovalPending,
		MemberCountSnapshot: memberCount,
		OnlineCountSnapshot: onlineCount,
		Region:              req.Region,
		Language:            language,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := s.repo.CreateListing(ctx, listing); err != nil {
		return err
	}

	// Set categories
	if err := s.repo.SetListingCategories(ctx, listing.ID, categoryIDs); err != nil {
		return err
	}

	// Set tags
	if len(req.Tags) > 0 {
		if err := s.repo.SetListingTags(ctx, listing.ID, req.Tags); err != nil {
			return err
		}
	}

	return nil
}

// UpdateListing updates a discovery listing
func (s *DiscoveryService) UpdateListing(ctx context.Context, serverID, userID uuid.UUID, req *models.UpdateDiscoveryRequest) error {
	// Check server exists
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil || server == nil {
		return ErrServerNotFound
	}

	// Check user is server owner
	if server.OwnerID != userID {
		return ErrDiscoveryNotOwner
	}

	listing, err := s.repo.GetServerListing(ctx, serverID)
	if err != nil {
		return err
	}
	if listing == nil {
		return ErrDiscoveryListingNotFound
	}

	// Update fields
	if req.ShortDescription != nil {
		listing.ShortDescription = *req.ShortDescription
	}
	if req.Region != nil {
		listing.Region = req.Region
	}
	if req.Language != nil {
		listing.Language = *req.Language
	}

	// Update categories
	if len(req.Categories) > 0 {
		categoryIDs, err := s.repo.GetCategoriesBySlug(ctx, toStringSlice(req.Categories))
		if err != nil {
			return err
		}
		if err := s.repo.SetListingCategories(ctx, listing.ID, categoryIDs); err != nil {
			return err
		}
	}

	// Update tags
	if req.Tags != nil {
		if err := s.repo.SetListingTags(ctx, listing.ID, req.Tags); err != nil {
			return err
		}
	}

	listing.UpdatedAt = time.Now()

	return s.repo.UpdateListing(ctx, listing)
}

// ApproveListing approves a server for discovery
func (s *DiscoveryService) ApproveListing(ctx context.Context, listingID, adminID uuid.UUID) error {
	listing, err := s.repo.GetServerListingByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing == nil {
		return ErrDiscoveryListingNotFound
	}

	now := time.Now()
	listing.ApprovalStatus = models.ApprovalApproved
	listing.ApprovedAt = &now
	listing.ApprovedBy = &adminID
	listing.IsListed = true
	listing.UpdatedAt = now

	return s.repo.UpdateListing(ctx, listing)
}

// RejectListing rejects a server from discovery
func (s *DiscoveryService) RejectListing(ctx context.Context, listingID, adminID uuid.UUID, reason string) error {
	listing, err := s.repo.GetServerListingByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing == nil {
		return ErrDiscoveryListingNotFound
	}

	listing.ApprovalStatus = models.ApprovalRejected
	listing.RejectionReason = &reason
	listing.IsListed = false
	listing.UpdatedAt = time.Now()

	return s.repo.UpdateListing(ctx, listing)
}

// SetFeatured marks a server as featured
func (s *DiscoveryService) SetFeatured(ctx context.Context, listingID uuid.UUID, featured bool) error {
	listing, err := s.repo.GetServerListingByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing == nil {
		return ErrDiscoveryListingNotFound
	}

	listing.IsFeatured = featured
	if featured {
		now := time.Now()
		listing.FeaturedAt = &now
	}
	listing.UpdatedAt = time.Now()

	return s.repo.UpdateListing(ctx, listing)
}

// ReportServer allows a user to report a server
func (s *DiscoveryService) ReportServer(ctx context.Context, serverID, reporterID uuid.UUID, req *models.ReportServerRequest) error {
	listing, err := s.repo.GetServerListing(ctx, serverID)
	if err != nil {
		return err
	}
	if listing == nil {
		return ErrDiscoveryListingNotFound
	}

	report := &models.DiscoveryReport{
		ID:         uuid.New(),
		ListingID:  listing.ID,
		ReporterID: reporterID,
		Reason:     req.Reason,
		Details:    &req.Details,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	return s.repo.CreateReport(ctx, report)
}

// GetServerListing returns listing details for a server
func (s *DiscoveryService) GetServerListing(ctx context.Context, serverID uuid.UUID) (*models.DiscoveryListing, error) {
	return s.repo.GetServerListing(ctx, serverID)
}

// GetDiscoveryListingWithDetails returns listing with server details
func (s *DiscoveryService) GetDiscoveryListingWithDetails(ctx context.Context, serverID uuid.UUID) (*models.ServerListingResult, error) {
	listing, err := s.repo.GetServerListing(ctx, serverID)
	if err != nil || listing == nil {
		return nil, err
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil || server == nil {
		return nil, ErrServerNotFound
	}

	result := &models.ServerListingResult{
		ID:               listing.ID,
		ServerID:         server.ID,
		Name:             server.Name,
		IconURL:          server.IconURL,
		BannerURL:        server.BannerURL,
		Description:      server.Description,
		ShortDescription: listing.ShortDescription,
		MemberCount:      listing.MemberCountSnapshot,
		OnlineCount:      listing.OnlineCountSnapshot,
		IsFeatured:       listing.IsFeatured,
		Region:           listing.Region,
		Language:         listing.Language,
		WeeklyGrowthRate: listing.WeeklyGrowthRate,
		EngagementScore:  listing.EngagementScore,
		CreatedAt:        listing.CreatedAt,
	}

	// Get invite code
	invites, err := s.inviteRepo.GetByServerID(ctx, serverID)
	if err == nil && len(invites) > 0 {
		result.InviteCode = invites[0].Code
	}

	return result, nil
}

// Helper to convert category slice
func toStringSlice(cats []models.ServerCategory) []string {
	result := make([]string, len(cats))
	for i, c := range cats {
		result[i] = string(c)
	}
	return result
}
