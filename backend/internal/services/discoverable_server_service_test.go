package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// --- Mock implementations ---

type mockDiscoverableServerRepo struct {
	mock.Mock
}

func (m *mockDiscoverableServerRepo) GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *mockDiscoverableServerRepo) GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscoverableFeaturedServer), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) (*models.DiscoverableServer, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoverableServer), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoverableServer), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CategoryInfo), args.Error(1)
}

func (m *mockDiscoverableServerRepo) SearchServers(ctx context.Context, query string, category models.ServerDiscoveryCategory, page, limit int) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, query, category, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *mockDiscoverableServerRepo) SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) ([]*models.DiscoverableServerSearchResult, int, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DiscoverableServerSearchResult), args.Int(1), args.Error(2)
}

func (m *mockDiscoverableServerRepo) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TrendingServerInfo), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerRecommendation), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CategoryWithStats), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscoveryTag), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscoveryPageStats), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SearchSuggestion), args.Error(1)
}

func (m *mockDiscoverableServerRepo) GetInviteCode(ctx context.Context, serverID uuid.UUID) (string, error) {
	args := m.Called(ctx, serverID)
	return args.String(0), args.Error(1)
}

func (m *mockDiscoverableServerRepo) TrackActivity(ctx context.Context, serverID uuid.UUID, userID *uuid.UUID, activityType, source string) error {
	args := m.Called(ctx, serverID, userID, activityType, source)
	return args.Error(0)
}

func (m *mockDiscoverableServerRepo) GetServerDailyStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ServerDiscoveryDailyStats, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerDiscoveryDailyStats), args.Error(1)
}

type mockServerRepo struct {
	mock.Mock
}

func (m *mockServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

type mockMemberRepo struct {
	mock.Mock
}

func (m *mockMemberRepo) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *mockMemberRepo) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

// --- Helper ---

func newTestService() (*DiscoverableServerService, *mockDiscoverableServerRepo, *mockServerRepo, *mockMemberRepo) {
	repo := new(mockDiscoverableServerRepo)
	serverRepo := new(mockServerRepo)
	memberRepo := new(mockMemberRepo)
	svc := NewDiscoverableServerService(repo, serverRepo, memberRepo)
	return svc, repo, serverRepo, memberRepo
}

func makeServer(name string, memberCount int, public bool) *models.DiscoverableServer {
	return &models.DiscoverableServer{
		ID:          uuid.New(),
		ServerID:    uuid.New(),
		Name:        name,
		Category:    models.DiscoveryCategoryGaming,
		MemberCount: memberCount,
		IsVerified:  true,
		IsPublic:    public,
		Language:    "en",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// --- Tests ---

func TestGetDiscoverableServers(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	servers := []*models.DiscoverableServerSearchResult{
		{Name: "Server A", MemberCount: 1000},
		{Name: "Server B", MemberCount: 500},
	}

	t.Run("returns paginated results", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 20}
		repo.On("GetDiscoverableServers", ctx, filters).Return(servers, 2, nil).Once()

		result, err := svc.GetDiscoverableServers(ctx, filters)
		assert.NoError(t, err)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.Limit)
		assert.Equal(t, 1, result.TotalPages)
		assert.Len(t, result.Servers, 2)
	})

	t.Run("normalizes zero page", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 0, Limit: 0}
		// After normalization: Page=1, Limit=20
		repo.On("GetDiscoverableServers", ctx, mock.AnythingOfType("*models.DiscoverFilters")).Return(servers, 2, nil).Once()

		result, err := svc.GetDiscoverableServers(ctx, filters)
		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.Limit)
	})

	t.Run("calculates total pages correctly", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 3}
		repo.On("GetDiscoverableServers", ctx, filters).Return(servers, 10, nil).Once()

		result, err := svc.GetDiscoverableServers(ctx, filters)
		assert.NoError(t, err)
		assert.Equal(t, 4, result.TotalPages) // ceil(10/3) = 4
	})

	t.Run("handles repo error", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 20}
		repo.On("GetDiscoverableServers", ctx, filters).Return(nil, 0, errors.New("db error")).Once()

		result, err := svc.GetDiscoverableServers(ctx, filters)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetServerByID(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns public server", func(t *testing.T) {
		server := makeServer("Test", 100, true)
		repo.On("GetByID", ctx, server.ID).Return(server, nil).Once()

		result, err := svc.GetServerByID(ctx, server.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
	})

	t.Run("rejects non-public server", func(t *testing.T) {
		server := makeServer("Private", 100, false)
		repo.On("GetByID", ctx, server.ID).Return(server, nil).Once()

		result, err := svc.GetServerByID(ctx, server.ID)
		assert.ErrorIs(t, err, ErrServerNotPublic)
		assert.Nil(t, result)
	})

	t.Run("returns not found for nil", func(t *testing.T) {
		id := uuid.New()
		repo.On("GetByID", ctx, id).Return(nil, nil).Once()

		result, err := svc.GetServerByID(ctx, id)
		assert.ErrorIs(t, err, ErrDiscoverableServerNotFound)
		assert.Nil(t, result)
	})
}

func TestGetServerDetail(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("includes invite code", func(t *testing.T) {
		server := makeServer("Detail Server", 500, true)
		repo.On("GetByID", ctx, server.ID).Return(server, nil).Once()
		repo.On("GetInviteCode", ctx, server.ServerID).Return("abc123", nil).Once()

		detail, err := svc.GetServerDetail(ctx, server.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Detail Server", detail.Name)
		assert.NotNil(t, detail.InviteCode)
		assert.Equal(t, "abc123", *detail.InviteCode)
	})

	t.Run("works without invite code", func(t *testing.T) {
		server := makeServer("No Invite", 100, true)
		repo.On("GetByID", ctx, server.ID).Return(server, nil).Once()
		repo.On("GetInviteCode", ctx, server.ServerID).Return("", nil).Once()

		detail, err := svc.GetServerDetail(ctx, server.ID)
		assert.NoError(t, err)
		assert.Nil(t, detail.InviteCode)
	})
}

func TestJoinServer(t *testing.T) {
	svc, repo, _, memberRepo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	t.Run("successfully joins server", func(t *testing.T) {
		server := makeServer("Join Me", 500, true)
		repo.On("GetByServerID", ctx, server.ServerID).Return(server, nil).Once()
		memberRepo.On("GetMember", ctx, server.ServerID, userID).Return(nil, nil).Once()
		memberRepo.On("AddMember", ctx, mock.AnythingOfType("*models.Member")).Return(nil).Once()

		err := svc.JoinServer(ctx, server.ServerID, userID)
		assert.NoError(t, err)
		memberRepo.AssertCalled(t, "AddMember", ctx, mock.AnythingOfType("*models.Member"))
	})

	t.Run("rejects already member", func(t *testing.T) {
		server := makeServer("Already In", 500, true)
		member := &models.Member{UserID: userID, ServerID: server.ServerID}
		repo.On("GetByServerID", ctx, server.ServerID).Return(server, nil).Once()
		memberRepo.On("GetMember", ctx, server.ServerID, userID).Return(member, nil).Once()

		err := svc.JoinServer(ctx, server.ServerID, userID)
		assert.ErrorIs(t, err, ErrAlreadyMember)
	})

	t.Run("rejects private server", func(t *testing.T) {
		server := makeServer("Private", 500, false)
		repo.On("GetByServerID", ctx, server.ServerID).Return(server, nil).Once()

		err := svc.JoinServer(ctx, server.ServerID, userID)
		assert.ErrorIs(t, err, ErrServerNotPublic)
	})

	t.Run("rejects non-existent server", func(t *testing.T) {
		serverID := uuid.New()
		repo.On("GetByServerID", ctx, serverID).Return(nil, nil).Once()

		err := svc.JoinServer(ctx, serverID, userID)
		assert.ErrorIs(t, err, ErrDiscoverableServerNotFound)
	})
}

func TestSearchServersEnhanced(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns search results with pagination", func(t *testing.T) {
		servers := []*models.DiscoverableServerSearchResult{
			{Name: "Match A"},
			{Name: "Match B"},
		}
		req := &models.DiscoverySearchRequest{
			Query:  "match",
			SortBy: "popular",
			Page:   1,
			Limit:  25,
		}
		repo.On("SearchServersEnhanced", ctx, req).Return(servers, 50, nil).Once()

		result, err := svc.SearchServersEnhanced(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 50, result.Total)
		assert.Equal(t, 2, result.TotalPages)
		assert.Len(t, result.Servers, 2)
	})

	t.Run("defaults limit when zero", func(t *testing.T) {
		req := &models.DiscoverySearchRequest{Query: "test", Limit: 0}
		repo.On("SearchServersEnhanced", ctx, req).Return([]*models.DiscoverableServerSearchResult{}, 0, nil).Once()

		result, err := svc.SearchServersEnhanced(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 25, result.Limit)
	})
}

func TestGetTrendingServers(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns trending servers", func(t *testing.T) {
		trending := []*models.TrendingServerInfo{
			{Server: &models.DiscoverableServerSearchResult{Name: "Hot Server"}, TrendScore: 95.0, GrowthRate: 15.0},
		}
		repo.On("GetTrendingServers", ctx, 10).Return(trending, nil).Once()

		result, err := svc.GetTrendingServers(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 95.0, result[0].TrendScore)
	})
}

func TestGetRecommendedServers(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	t.Run("returns recommendations", func(t *testing.T) {
		recs := []*models.ServerRecommendation{
			{
				DiscoverableServerSearchResult: models.DiscoverableServerSearchResult{Name: "Rec Server"},
				Reason:                         "Popular in categories you enjoy",
				MutualMemberCount:              10,
			},
		}
		repo.On("GetRecommendedServers", ctx, userID, 10).Return(recs, nil).Once()

		result, err := svc.GetRecommendedServers(ctx, userID, 10)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Rec Server", result[0].Name)
		assert.Equal(t, 10, result[0].MutualMemberCount)
	})
}

func TestGetDiscoveryHomePage(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	t.Run("returns full home page", func(t *testing.T) {
		featured := []*models.DiscoverableFeaturedServer{{Name: "Featured"}}
		trending := []*models.TrendingServerInfo{{Server: &models.DiscoverableServerSearchResult{Name: "Trending"}}}
		recs := []*models.ServerRecommendation{{DiscoverableServerSearchResult: models.DiscoverableServerSearchResult{Name: "Rec"}}}
		cats := []*models.CategoryWithStats{{CategoryInfo: models.CategoryInfo{Name: "Gaming"}}}
		tags := []*models.DiscoveryTag{{Name: "Competitive"}}
		stats := &models.DiscoveryPageStats{TotalServers: 100}

		repo.On("GetFeaturedServers", ctx, 5).Return(featured, nil).Once()
		repo.On("GetTrendingServers", ctx, 10).Return(trending, nil).Once()
		repo.On("GetRecommendedServers", ctx, userID, 10).Return(recs, nil).Once()
		repo.On("GetCategoriesWithStats", ctx).Return(cats, nil).Once()
		repo.On("GetPopularTags", ctx, 10).Return(tags, nil).Once()
		repo.On("GetDiscoveryStats", ctx).Return(stats, nil).Once()

		page, err := svc.GetDiscoveryHomePage(ctx, userID, 5, 10, 10)
		assert.NoError(t, err)
		assert.Len(t, page.Featured, 1)
		assert.Len(t, page.Trending, 1)
		assert.Len(t, page.Recommended, 1)
		assert.Len(t, page.Categories, 1)
		assert.Len(t, page.PopularTags, 1)
		assert.Equal(t, int64(100), page.Stats.TotalServers)
	})

	t.Run("skips recommendations for anonymous user", func(t *testing.T) {
		repo.On("GetFeaturedServers", ctx, 5).Return([]*models.DiscoverableFeaturedServer{}, nil).Once()
		repo.On("GetTrendingServers", ctx, 10).Return([]*models.TrendingServerInfo{}, nil).Once()
		repo.On("GetCategoriesWithStats", ctx).Return([]*models.CategoryWithStats{}, nil).Once()
		repo.On("GetPopularTags", ctx, 10).Return([]*models.DiscoveryTag{}, nil).Once()
		repo.On("GetDiscoveryStats", ctx).Return(&models.DiscoveryPageStats{}, nil).Once()

		page, err := svc.GetDiscoveryHomePage(ctx, uuid.Nil, 5, 10, 10)
		assert.NoError(t, err)
		assert.Nil(t, page.Recommended)
	})
}

func TestTrackActivity(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	t.Run("tracks valid activity", func(t *testing.T) {
		repo.On("TrackActivity", ctx, serverID, &userID, "view", "home").Return(nil).Once()

		err := svc.TrackActivity(ctx, serverID, &userID, "view", "home")
		assert.NoError(t, err)
	})

	t.Run("rejects invalid activity type", func(t *testing.T) {
		err := svc.TrackActivity(ctx, serverID, &userID, "invalid_type", "home")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid activity type")
	})

	t.Run("rejects invalid source", func(t *testing.T) {
		err := svc.TrackActivity(ctx, serverID, &userID, "view", "invalid_source")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid discovery source")
	})

	t.Run("allows empty source", func(t *testing.T) {
		repo.On("TrackActivity", ctx, serverID, &userID, "impression", "").Return(nil).Once()

		err := svc.TrackActivity(ctx, serverID, &userID, "impression", "")
		assert.NoError(t, err)
	})
}

func TestGetCategoriesWithStats(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns categories with stats", func(t *testing.T) {
		cats := []*models.CategoryWithStats{
			{CategoryInfo: models.CategoryInfo{Name: "Gaming", Slug: "gaming", ServerCount: 50}, TotalMembers: 10000, AvgMemberCount: 200, GrowthRate: 5.0},
			{CategoryInfo: models.CategoryInfo{Name: "Music", Slug: "music", ServerCount: 30}, TotalMembers: 5000, AvgMemberCount: 166.67, GrowthRate: 3.0},
		}
		repo.On("GetCategoriesWithStats", ctx).Return(cats, nil).Once()

		result, err := svc.GetCategoriesWithStats(ctx)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 50, result[0].ServerCount)
		assert.Equal(t, 10000, result[0].TotalMembers)
	})
}

func TestGetPopularTags(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns tags", func(t *testing.T) {
		tags := []*models.DiscoveryTag{
			{Name: "Competitive", Slug: "competitive", UsageCount: 100},
			{Name: "Casual", Slug: "casual", UsageCount: 80},
		}
		repo.On("GetPopularTags", ctx, 20).Return(tags, nil).Once()

		result, err := svc.GetPopularTags(ctx, 20)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestGetSearchSuggestions(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns suggestions", func(t *testing.T) {
		suggestions := []*models.SearchSuggestion{
			{Type: "server", Value: "Gaming Hub"},
			{Type: "category", Value: "gaming"},
		}
		repo.On("GetSearchSuggestions", ctx, "gam", 10).Return(suggestions, nil).Once()

		result, err := svc.GetSearchSuggestions(ctx, "gam", 10)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestGetFeaturedServers(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()

	t.Run("returns featured servers", func(t *testing.T) {
		featured := []*models.DiscoverableFeaturedServer{
			{Name: "Featured A", MemberCount: 50000},
		}
		repo.On("GetFeaturedServers", ctx, 5).Return(featured, nil).Once()

		result, err := svc.GetFeaturedServers(ctx, 5)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Featured A", result[0].Name)
	})
}

func TestCanJoinServer(t *testing.T) {
	svc, repo, _, memberRepo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	t.Run("allows joining public server", func(t *testing.T) {
		server := makeServer("Joinable", 100, true)
		repo.On("GetByServerID", ctx, server.ServerID).Return(server, nil).Once()
		memberRepo.On("GetMember", ctx, server.ServerID, userID).Return(nil, nil).Once()

		err := svc.CanJoinServer(ctx, server.ServerID, userID)
		assert.NoError(t, err)
	})

	t.Run("rejects already member", func(t *testing.T) {
		server := makeServer("AlreadyIn", 100, true)
		member := &models.Member{UserID: userID, ServerID: server.ServerID}
		repo.On("GetByServerID", ctx, server.ServerID).Return(server, nil).Once()
		memberRepo.On("GetMember", ctx, server.ServerID, userID).Return(member, nil).Once()

		err := svc.CanJoinServer(ctx, server.ServerID, userID)
		assert.ErrorIs(t, err, ErrAlreadyMember)
	})
}
