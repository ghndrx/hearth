package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// --- Mock implementations ---

type MockDiscoverableServerRepo struct {
	mock.Mock
}

func (m *MockDiscoverableServerRepo) GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *MockDiscoverableServerRepo) GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscoverableFeaturedServer), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoverableServer), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoverableServer), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CategoryInfo), args.Error(1)
}

func (m *MockDiscoverableServerRepo) SearchServers(ctx context.Context, query string, category models.ServerDiscoveryCategory, page, limit int) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, query, category, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *MockDiscoverableServerRepo) SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *MockDiscoverableServerRepo) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TrendingServerInfo), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerRecommendation), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CategoryWithStats), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscoveryTag), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoveryPageStats), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SearchSuggestion), args.Error(1)
}

func (m *MockDiscoverableServerRepo) GetInviteCode(ctx context.Context, serverID uuid.UUID) (string, error) {
	args := m.Called(ctx, serverID)
	return args.String(0), args.Error(1)
}

func (m *MockDiscoverableServerRepo) Create(ctx context.Context, server *models.DiscoverableServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockDiscoverableServerRepo) Update(ctx context.Context, server *models.DiscoverableServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockDiscoverableServerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDiscoverableServerRepo) TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error {
	args := m.Called(ctx, serverID, userID, activityType, source)
	return args.Error(0)
}

func (m *MockDiscoverableServerRepo) GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerDiscoveryDailyStats), args.Error(1)
}

type MockServerRepoForDiscovery struct {
	mock.Mock
}

func (m *MockServerRepoForDiscovery) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

type MockMemberRepoForDiscovery struct {
	mock.Mock
}

func (m *MockMemberRepoForDiscovery) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockMemberRepoForDiscovery) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

// --- Helper ---

func newTestDiscoverableServerService() (*DiscoverableServerService, *MockDiscoverableServerRepo, *MockServerRepoForDiscovery, *MockMemberRepoForDiscovery) {
	repo := new(MockDiscoverableServerRepo)
	serverRepo := new(MockServerRepoForDiscovery)
	memberRepo := new(MockMemberRepoForDiscovery)
	svc := NewDiscoverableServerService(repo, serverRepo, memberRepo)
	return svc, repo, serverRepo, memberRepo
}

// --- Tests ---

func TestGetDiscoverableServers_Pagination(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	servers := []*models.DiscoverableServerSearchResult{
		{ID: uuid.New(), Name: "Server A", MemberCount: 100},
		{ID: uuid.New(), Name: "Server B", MemberCount: 50},
	}

	repo.On("GetDiscoverableServers", ctx, mock.AnythingOfType("*models.DiscoverFilters")).Return(servers, 42, nil)

	result, err := svc.GetDiscoverableServers(ctx, &models.DiscoverFilters{Page: 2, Limit: 20})
	assert.NoError(t, err)
	assert.Equal(t, 42, result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 3, result.TotalPages) // ceil(42/20) = 3
	assert.Len(t, result.Servers, 2)
}

func TestGetDiscoverableServers_NormalizesFilters(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	repo.On("GetDiscoverableServers", ctx, mock.AnythingOfType("*models.DiscoverFilters")).
		Return([]*models.DiscoverableServerSearchResult{}, 0, nil)

	// Zero page/limit should be normalized
	result, err := svc.GetDiscoverableServers(ctx, &models.DiscoverFilters{Page: 0, Limit: 0})
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
}

func TestGetDiscoverableServers_RepoError(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	repo.On("GetDiscoverableServers", ctx, mock.AnythingOfType("*models.DiscoverFilters")).
		Return(nil, 0, errors.New("db error"))

	_, err := svc.GetDiscoverableServers(ctx, &models.DiscoverFilters{})
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

func TestGetServerByID_Success(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()

	server := &models.DiscoverableServer{
		ID:       id,
		Name:     "Test Server",
		IsPublic: true,
	}
	repo.On("GetByID", ctx, id).Return(server, nil)

	result, err := svc.GetServerByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, "Test Server", result.Name)
}

func TestGetServerByID_NotFound(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, nil)

	_, err := svc.GetServerByID(ctx, id)
	assert.ErrorIs(t, err, ErrDiscoverableServerNotFound)
}

func TestGetServerByID_NotPublic(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()

	server := &models.DiscoverableServer{ID: id, IsPublic: false}
	repo.On("GetByID", ctx, id).Return(server, nil)

	_, err := svc.GetServerByID(ctx, id)
	assert.ErrorIs(t, err, ErrServerNotPublic)
}

func TestGetServerDetail_IncludesInviteCode(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()

	server := &models.DiscoverableServer{ID: id, ServerID: serverID, Name: "Test", IsPublic: true}
	repo.On("GetByID", ctx, id).Return(server, nil)
	repo.On("GetInviteCode", ctx, serverID).Return("abc123", nil)

	detail, err := svc.GetServerDetail(ctx, id)
	assert.NoError(t, err)
	assert.NotNil(t, detail.InviteCode)
	assert.Equal(t, "abc123", *detail.InviteCode)
}

func TestSearchServersEnhanced_Pagination(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	servers := []*models.DiscoverableServerSearchResult{
		{ID: uuid.New(), Name: "Gaming Hub", MemberCount: 500},
	}
	req := &models.DiscoverySearchRequest{
		Query:    "gaming",
		Category: models.DiscoveryCategoryGaming,
		SortBy:   "popular",
		Page:     1,
		Limit:    25,
	}

	repo.On("SearchServersEnhanced", ctx, req).Return(servers, 1, nil)

	result, err := svc.SearchServersEnhanced(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.TotalPages)
	assert.Len(t, result.Servers, 1)
	assert.Equal(t, "Gaming Hub", result.Servers[0].Name)
}

func TestSearchServersEnhanced_DefaultLimit(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	req := &models.DiscoverySearchRequest{Query: "test", Limit: 0}
	repo.On("SearchServersEnhanced", ctx, req).Return([]*models.DiscoverableServerSearchResult{}, 0, nil)

	result, err := svc.SearchServersEnhanced(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, 25, result.Limit) // default limit
}

func TestGetDiscoveryHomePage_Success(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	userID := uuid.New()

	featured := []*models.DiscoverableFeaturedServer{{Name: "Featured1"}}
	trending := []*models.TrendingServerInfo{{TrendScore: 10.0}}
	recommended := []*models.ServerRecommendation{{Reason: "test"}}
	categories := []*models.CategoryWithStats{{}}
	tags := []*models.DiscoveryTag{{Name: "gaming"}}
	stats := &models.DiscoveryPageStats{TotalServers: 100}

	repo.On("GetFeaturedServers", ctx, 5).Return(featured, nil)
	repo.On("GetTrendingServers", ctx, 10).Return(trending, nil)
	repo.On("GetRecommendedServers", ctx, userID, 10).Return(recommended, nil)
	repo.On("GetCategoriesWithStats", ctx).Return(categories, nil)
	repo.On("GetPopularTags", ctx, 10).Return(tags, nil)
	repo.On("GetDiscoveryStats", ctx).Return(stats, nil)

	page, err := svc.GetDiscoveryHomePage(ctx, userID, 5, 10, 10)
	assert.NoError(t, err)
	assert.Len(t, page.Featured, 1)
	assert.Len(t, page.Trending, 1)
	assert.Len(t, page.Recommended, 1)
	assert.Equal(t, int64(100), page.Stats.TotalServers)
}

func TestGetDiscoveryHomePage_NoUser(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	repo.On("GetFeaturedServers", ctx, 5).Return([]*models.DiscoverableFeaturedServer{}, nil)
	repo.On("GetTrendingServers", ctx, 10).Return([]*models.TrendingServerInfo{}, nil)
	repo.On("GetCategoriesWithStats", ctx).Return([]*models.CategoryWithStats{}, nil)
	repo.On("GetPopularTags", ctx, 10).Return([]*models.DiscoveryTag{}, nil)
	repo.On("GetDiscoveryStats", ctx).Return(&models.DiscoveryPageStats{}, nil)

	page, err := svc.GetDiscoveryHomePage(ctx, uuid.Nil, 5, 10, 10)
	assert.NoError(t, err)
	assert.Nil(t, page.Recommended) // No recommendations for anonymous user
}

func TestDiscoveryJoinServer_Success(t *testing.T) {
	svc, repo, _, memberRepo := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	ds := &models.DiscoverableServer{ServerID: serverID, IsPublic: true}
	repo.On("GetByServerID", ctx, serverID).Return(ds, nil)
	memberRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)
	memberRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil)

	err := svc.JoinServer(ctx, serverID, userID)
	assert.NoError(t, err)
	memberRepo.AssertCalled(t, "AddMember", ctx, mock.AnythingOfType("*models.Member"))
}

func TestDiscoveryJoinServer_AlreadyMember(t *testing.T) {
	svc, repo, _, memberRepo := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	ds := &models.DiscoverableServer{ServerID: serverID, IsPublic: true}
	repo.On("GetByServerID", ctx, serverID).Return(ds, nil)
	memberRepo.On("GetMember", ctx, serverID, userID).Return(&models.Member{}, nil)

	err := svc.JoinServer(ctx, serverID, userID)
	assert.ErrorIs(t, err, ErrAlreadyMember)
}

func TestDiscoveryJoinServer_NotPublic(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()

	ds := &models.DiscoverableServer{ServerID: serverID, IsPublic: false}
	repo.On("GetByServerID", ctx, serverID).Return(ds, nil)

	err := svc.JoinServer(ctx, serverID, uuid.New())
	assert.ErrorIs(t, err, ErrServerNotPublic)
}

func TestRegisterServer_Success(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("GetByServerID", ctx, serverID).Return(nil, nil)
	repo.On("Create", ctx, mock.AnythingOfType("*models.DiscoverableServer")).Return(nil)

	req := &models.RegisterServerRequest{
		Name:        "My Server",
		Description: "A test server",
		Category:    models.DiscoveryCategoryGaming,
		Tags:        []string{"fps", "competitive"},
	}

	ds, err := svc.RegisterServer(ctx, serverID, ownerID, req)
	assert.NoError(t, err)
	assert.Equal(t, "My Server", ds.Name)
	assert.Equal(t, models.DiscoveryCategoryGaming, ds.Category)
	assert.Equal(t, pq.StringArray([]string{"fps", "competitive"}), ds.Tags)
	assert.True(t, ds.IsPublic)
	assert.False(t, ds.IsFeatured)
}

func TestRegisterServer_NotOwner(t *testing.T) {
	svc, _, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	otherUser := uuid.New()

	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	req := &models.RegisterServerRequest{
		Name:     "My Server",
		Category: models.DiscoveryCategoryGaming,
	}

	_, err := svc.RegisterServer(ctx, serverID, otherUser, req)
	assert.ErrorIs(t, err, ErrNotServerOwner)
}

func TestRegisterServer_ServerNotFound(t *testing.T) {
	svc, _, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	req := &models.RegisterServerRequest{
		Name:     "My Server",
		Category: models.DiscoveryCategoryGaming,
	}

	_, err := svc.RegisterServer(ctx, serverID, uuid.New(), req)
	assert.ErrorIs(t, err, ErrServerNotFound)
}

func TestRegisterServer_AlreadyRegistered(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("GetByServerID", ctx, serverID).Return(&models.DiscoverableServer{}, nil)

	req := &models.RegisterServerRequest{
		Name:     "My Server",
		Category: models.DiscoveryCategoryGaming,
	}

	_, err := svc.RegisterServer(ctx, serverID, ownerID, req)
	assert.Error(t, err)
	assert.Equal(t, "server already registered for discovery", err.Error())
}

func TestRegisterServer_InvalidCategory(t *testing.T) {
	svc, _, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	req := &models.RegisterServerRequest{
		Name:     "My Server",
		Category: "invalid_category",
	}

	_, err := svc.RegisterServer(ctx, serverID, ownerID, req)
	assert.Error(t, err)
	assert.Equal(t, "invalid category", err.Error())
}

func TestUpdateRegisteredServer_Success(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{
		ID:       id,
		ServerID: serverID,
		Name:     "Old Name",
		Category: models.DiscoveryCategoryGaming,
		IsPublic: true,
	}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*models.DiscoverableServer")).Return(nil)

	newName := "New Name"
	newCat := models.DiscoveryCategoryTechnology
	req := &models.UpdateDiscoverableServerRequest{
		Name:     &newName,
		Category: &newCat,
		Tags:     []string{"coding", "devops"},
	}

	updated, err := svc.UpdateRegisteredServer(ctx, id, ownerID, req)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, models.DiscoveryCategoryTechnology, updated.Category)
	assert.Equal(t, pq.StringArray([]string{"coding", "devops"}), updated.Tags)
}

func TestUpdateRegisteredServer_NotOwner(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{ID: id, ServerID: serverID}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	_, err := svc.UpdateRegisteredServer(ctx, id, uuid.New(), &models.UpdateDiscoverableServerRequest{})
	assert.ErrorIs(t, err, ErrNotServerOwner)
}

func TestUpdateRegisteredServer_NotFound(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, nil)

	_, err := svc.UpdateRegisteredServer(ctx, id, uuid.New(), &models.UpdateDiscoverableServerRequest{})
	assert.ErrorIs(t, err, ErrDiscoverableServerNotFound)
}

func TestDeleteRegisteredServer_Success(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{ID: id, ServerID: serverID}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("Delete", ctx, id).Return(nil)

	err := svc.DeleteRegisteredServer(ctx, id, ownerID)
	assert.NoError(t, err)
	repo.AssertCalled(t, "Delete", ctx, id)
}

func TestDeleteRegisteredServer_NotOwner(t *testing.T) {
	svc, repo, serverRepo, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{ID: id, ServerID: serverID}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	err := svc.DeleteRegisteredServer(ctx, id, uuid.New())
	assert.ErrorIs(t, err, ErrNotServerOwner)
}

func TestCanJoinServer_Success(t *testing.T) {
	svc, repo, _, memberRepo := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	ds := &models.DiscoverableServer{ServerID: serverID, IsPublic: true}
	repo.On("GetByServerID", ctx, serverID).Return(ds, nil)
	memberRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)

	err := svc.CanJoinServer(ctx, serverID, userID)
	assert.NoError(t, err)
}

func TestCanJoinServer_AlreadyMember(t *testing.T) {
	svc, repo, _, memberRepo := newTestDiscoverableServerService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	ds := &models.DiscoverableServer{ServerID: serverID, IsPublic: true}
	repo.On("GetByServerID", ctx, serverID).Return(ds, nil)
	memberRepo.On("GetMember", ctx, serverID, userID).Return(&models.Member{
		UserID:   userID,
		ServerID: serverID,
		JoinedAt: time.Now(),
	}, nil)

	err := svc.CanJoinServer(ctx, serverID, userID)
	assert.ErrorIs(t, err, ErrAlreadyMember)
}

func TestNormalizeDiscoverFilters(t *testing.T) {
	tests := []struct {
		name          string
		input         *models.DiscoverFilters
		expectedPage  int
		expectedLimit int
	}{
		{"zero values", &models.DiscoverFilters{Page: 0, Limit: 0}, 1, 20},
		{"negative values", &models.DiscoverFilters{Page: -1, Limit: -5}, 1, 20},
		{"limit too high", &models.DiscoverFilters{Page: 1, Limit: 200}, 1, 100},
		{"valid values", &models.DiscoverFilters{Page: 3, Limit: 50}, 3, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models.NormalizeDiscoverFilters(tt.input)
			assert.Equal(t, tt.expectedPage, tt.input.Page)
			assert.Equal(t, tt.expectedLimit, tt.input.Limit)
		})
	}
}

func TestIsValidCategory(t *testing.T) {
	assert.True(t, models.IsValidCategory("gaming"))
	assert.True(t, models.IsValidCategory("technology"))
	assert.True(t, models.IsValidCategory("art"))
	assert.True(t, models.IsValidCategory("other"))
	assert.False(t, models.IsValidCategory("invalid"))
	assert.False(t, models.IsValidCategory(""))
}

func TestAllDiscoveryCategories(t *testing.T) {
	categories := models.AllDiscoveryCategories()
	assert.Equal(t, 9, len(categories))
	assert.Contains(t, categories, models.DiscoveryCategoryGaming)
	assert.Contains(t, categories, models.DiscoveryCategoryOther)
}

func TestGetFeaturedServers(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	featured := []*models.DiscoverableFeaturedServer{
		{Name: "Featured1", MemberCount: 1000, IsVerified: true},
		{Name: "Featured2", MemberCount: 500},
	}
	repo.On("GetFeaturedServers", ctx, 5).Return(featured, nil)

	result, err := svc.GetFeaturedServers(ctx, 5)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Featured1", result[0].Name)
}

func TestGetTrendingServers(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	trending := []*models.TrendingServerInfo{
		{TrendScore: 95.0, GrowthRate: 12.5},
	}
	repo.On("GetTrendingServers", ctx, 10).Return(trending, nil)

	result, err := svc.GetTrendingServers(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 95.0, result[0].TrendScore)
}

func TestGetRecommendedServers(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()
	userID := uuid.New()

	recs := []*models.ServerRecommendation{
		{Reason: "Popular in your interests"},
	}
	repo.On("GetRecommendedServers", ctx, userID, 10).Return(recs, nil)

	result, err := svc.GetRecommendedServers(ctx, userID, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Popular in your interests", result[0].Reason)
}

func TestGetSearchSuggestions(t *testing.T) {
	svc, repo, _, _ := newTestDiscoverableServerService()
	ctx := context.Background()

	suggestions := []*models.SearchSuggestion{
		{Type: "server", Value: "Gaming Hub"},
		{Type: "category", Value: "gaming", Count: 50},
	}
	repo.On("GetSearchSuggestions", ctx, "gam", 10).Return(suggestions, nil)

	result, err := svc.GetSearchSuggestions(ctx, "gam", 10)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}
