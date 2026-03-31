package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

func newTestCachedService() (*CachedDiscoverableServerService, *MockDiscoverableServerRepo, *MockServerRepoForDiscovery, *MockMemberRepoForDiscovery) {
	repo := new(MockDiscoverableServerRepo)
	serverRepo := new(MockServerRepoForDiscovery)
	memberRepo := new(MockMemberRepoForDiscovery)
	svc := NewDiscoverableServerService(repo, serverRepo, memberRepo)
	// nil cache means no caching - all calls go straight to the service
	cached := NewCachedDiscoverableServerService(svc, nil)
	return cached, repo, serverRepo, memberRepo
}

func TestCachedGetDiscoverableServers_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	servers := []*models.DiscoverableServerSearchResult{
		{ID: uuid.New(), Name: "Server A", MemberCount: 100},
	}
	repo.On("GetDiscoverableServers", ctx, mock.AnythingOfType("*models.DiscoverFilters")).Return(servers, 1, nil)

	result, err := cached.GetDiscoverableServers(ctx, &models.DiscoverFilters{Page: 1, Limit: 20})
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Servers, 1)
}

func TestCachedGetFeaturedServers_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	featured := []*models.DiscoverableFeaturedServer{{Name: "Featured1"}}
	repo.On("GetFeaturedServers", ctx, 5).Return(featured, nil)

	result, err := cached.GetFeaturedServers(ctx, 5)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetTrendingServers_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	trending := []*models.TrendingServerInfo{{TrendScore: 95.0}}
	repo.On("GetTrendingServers", ctx, 10).Return(trending, nil)

	result, err := cached.GetTrendingServers(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetRecommendedServers_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()
	userID := uuid.New()

	recs := []*models.ServerRecommendation{{Reason: "Popular"}}
	repo.On("GetRecommendedServers", ctx, userID, 10).Return(recs, nil)

	result, err := cached.GetRecommendedServers(ctx, userID, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetCategories_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	cats := []*models.CategoryInfo{{Name: "Gaming", Slug: "gaming"}}
	repo.On("GetCategories", ctx).Return(cats, nil)

	result, err := cached.GetCategories(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetCategoriesWithStats_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	cats := []*models.CategoryWithStats{{CategoryInfo: models.CategoryInfo{Name: "Gaming"}}}
	repo.On("GetCategoriesWithStats", ctx).Return(cats, nil)

	result, err := cached.GetCategoriesWithStats(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetDiscoveryStats_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	stats := &models.DiscoveryPageStats{TotalServers: 500}
	repo.On("GetDiscoveryStats", ctx).Return(stats, nil)

	result, err := cached.GetDiscoveryStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), result.TotalServers)
}

func TestCachedGetPopularTags_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	tags := []*models.DiscoveryTag{{Name: "gaming"}}
	repo.On("GetPopularTags", ctx, 20).Return(tags, nil)

	result, err := cached.GetPopularTags(ctx, 20)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedGetSearchSuggestions_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	suggestions := []*models.SearchSuggestion{{Type: "server", Value: "Gaming Hub"}}
	repo.On("GetSearchSuggestions", ctx, "gam", 10).Return(suggestions, nil)

	result, err := cached.GetSearchSuggestions(ctx, "gam", 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCachedSearchServersEnhanced_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	servers := []*models.DiscoverableServerSearchResult{{Name: "Gaming Hub"}}
	req := &models.DiscoverySearchRequest{Query: "gaming", Limit: 25}
	repo.On("SearchServersEnhanced", ctx, req).Return(servers, 1, nil)

	result, err := cached.SearchServersEnhanced(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

func TestCachedGetDiscoveryHomePage_WithoutCache(t *testing.T) {
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetFeaturedServers", ctx, 5).Return([]*models.DiscoverableFeaturedServer{}, nil)
	repo.On("GetTrendingServers", ctx, 10).Return([]*models.TrendingServerInfo{}, nil)
	repo.On("GetRecommendedServers", ctx, userID, 10).Return([]*models.ServerRecommendation{}, nil)
	repo.On("GetCategoriesWithStats", ctx).Return([]*models.CategoryWithStats{}, nil)
	repo.On("GetPopularTags", ctx, 10).Return([]*models.DiscoveryTag{}, nil)
	repo.On("GetDiscoveryStats", ctx).Return(&models.DiscoveryPageStats{}, nil)

	page, err := cached.GetDiscoveryHomePage(ctx, userID, 5, 10, 10)
	assert.NoError(t, err)
	assert.NotNil(t, page)
}

func TestCachedRegisterServer_InvalidatesCache(t *testing.T) {
	cached, repo, serverRepo, _ := newTestCachedService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("GetByServerID", ctx, serverID).Return(nil, nil)
	repo.On("Create", ctx, mock.AnythingOfType("*models.DiscoverableServer")).Return(nil)

	req := &models.RegisterServerRequest{
		Name:     "My Server",
		Category: models.DiscoveryCategoryGaming,
	}

	ds, err := cached.RegisterServer(ctx, serverID, ownerID, req)
	assert.NoError(t, err)
	assert.Equal(t, "My Server", ds.Name)
}

func TestCachedUpdateRegisteredServer_InvalidatesCache(t *testing.T) {
	cached, repo, serverRepo, _ := newTestCachedService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{ID: id, ServerID: serverID, Name: "Old Name", Category: models.DiscoveryCategoryGaming, IsPublic: true}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*models.DiscoverableServer")).Return(nil)

	newName := "New Name"
	req := &models.UpdateDiscoverableServerRequest{Name: &newName}

	updated, err := cached.UpdateRegisteredServer(ctx, id, ownerID, req)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestCachedDeleteRegisteredServer_InvalidatesCache(t *testing.T) {
	cached, repo, serverRepo, _ := newTestCachedService()
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	ds := &models.DiscoverableServer{ID: id, ServerID: serverID}
	server := &models.Server{ID: serverID, OwnerID: ownerID}

	repo.On("GetByID", ctx, id).Return(ds, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	repo.On("Delete", ctx, id).Return(nil)

	err := cached.DeleteRegisteredServer(ctx, id, ownerID)
	assert.NoError(t, err)
}

func TestCachedDiscoverableServerService_NilCacheFallsThrough(t *testing.T) {
	// Verifies that all methods work without a cache (nil cache)
	cached, repo, _, _ := newTestCachedService()
	ctx := context.Background()

	// Test multiple methods to ensure nil cache doesn't panic
	repo.On("GetDiscoverableServers", ctx, mock.Anything).Return([]*models.DiscoverableServerSearchResult{}, 0, nil)
	repo.On("GetFeaturedServers", ctx, 5).Return([]*models.DiscoverableFeaturedServer{}, nil)
	repo.On("GetTrendingServers", ctx, 10).Return([]*models.TrendingServerInfo{}, nil)
	repo.On("GetCategories", ctx).Return([]*models.CategoryInfo{}, nil)
	repo.On("GetCategoriesWithStats", ctx).Return([]*models.CategoryWithStats{}, nil)
	repo.On("GetDiscoveryStats", ctx).Return(&models.DiscoveryPageStats{}, nil)
	repo.On("GetPopularTags", ctx, 20).Return([]*models.DiscoveryTag{}, nil)
	repo.On("GetSearchSuggestions", ctx, "test", 10).Return([]*models.SearchSuggestion{}, nil)

	_, err := cached.GetDiscoverableServers(ctx, &models.DiscoverFilters{})
	assert.NoError(t, err)

	_, err = cached.GetFeaturedServers(ctx, 5)
	assert.NoError(t, err)

	_, err = cached.GetTrendingServers(ctx, 10)
	assert.NoError(t, err)

	_, err = cached.GetCategories(ctx)
	assert.NoError(t, err)

	_, err = cached.GetCategoriesWithStats(ctx)
	assert.NoError(t, err)

	_, err = cached.GetDiscoveryStats(ctx)
	assert.NoError(t, err)

	_, err = cached.GetPopularTags(ctx, 20)
	assert.NoError(t, err)

	_, err = cached.GetSearchSuggestions(ctx, "test", 10)
	assert.NoError(t, err)
}
