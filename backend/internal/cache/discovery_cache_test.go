package cache

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

func TestDiscoveryCacheKeyGeneration(t *testing.T) {
	t.Run("discovery list key", func(t *testing.T) {
		key := discoveryListKey(1, 20, "gaming", "technology")
		assert.Equal(t, "discovery:list:p1:l20:qgaming:ctechnology", key)
	})

	t.Run("discovery list key with empty values", func(t *testing.T) {
		key := discoveryListKey(1, 20, "", "")
		assert.Equal(t, "discovery:list:p1:l20:q:c", key)
	})

	t.Run("discovery featured key", func(t *testing.T) {
		key := discoveryFeaturedKey(10)
		assert.Equal(t, "discovery:featured:10", key)
	})

	t.Run("discovery trending key", func(t *testing.T) {
		key := discoveryTrendingKey(15)
		assert.Equal(t, "discovery:trending:15", key)
	})

	t.Run("discovery recommended key", func(t *testing.T) {
		userID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
		key := discoveryRecommendedKey(userID, 10)
		assert.Equal(t, "discovery:recommended:12345678-1234-1234-1234-123456789abc:10", key)
	})

	t.Run("discovery categories key", func(t *testing.T) {
		key := discoveryCategoriesKey()
		assert.Equal(t, "discovery:categories", key)
	})

	t.Run("discovery categories with stats key", func(t *testing.T) {
		key := discoveryCategoriesWithStatsKey()
		assert.Equal(t, "discovery:categories:stats", key)
	})

	t.Run("discovery stats key", func(t *testing.T) {
		key := discoveryStatsKey()
		assert.Equal(t, "discovery:stats", key)
	})

	t.Run("discovery popular tags key", func(t *testing.T) {
		key := discoveryPopularTagsKey(20)
		assert.Equal(t, "discovery:tags:popular:20", key)
	})

	t.Run("discovery suggestions key", func(t *testing.T) {
		key := discoverySuggestionsKey("gam", 10)
		assert.Equal(t, "discovery:suggestions:gam:10", key)
	})

	t.Run("discovery server detail key", func(t *testing.T) {
		id := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
		key := discoveryServerDetailKey(id)
		assert.Equal(t, "discovery:server:12345678-1234-1234-1234-123456789abc", key)
	})

	t.Run("discovery search key", func(t *testing.T) {
		key := discoverySearchKey("gaming", "tech", "popular", "desc", 1, 25)
		assert.Equal(t, "discovery:search:qgaming:ctech:spopular:odesc:p1:l25", key)
	})

	t.Run("discovery home page key", func(t *testing.T) {
		userID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
		key := discoveryHomePageKey(userID, 5, 10, 10)
		assert.Equal(t, "discovery:home:12345678-1234-1234-1234-123456789abc:5:10:10", key)
	})

	t.Run("discovery directory key", func(t *testing.T) {
		key := discoveryDirectoryKey("test", "gaming", "popular", "desc", 25, 0)
		assert.Equal(t, "discovery:directory:qtest:cgaming:spopular:odesc:l25:off0", key)
	})
}

func TestDiscoveryCacheTTLValues(t *testing.T) {
	// Verify TTL values are reasonable
	assert.True(t, DiscoveryCacheTTL > 0, "DiscoveryCacheTTL should be positive")
	assert.True(t, DiscoveryFeaturedTTL > 0, "DiscoveryFeaturedTTL should be positive")
	assert.True(t, DiscoveryTrendingTTL > 0, "DiscoveryTrendingTTL should be positive")
	assert.True(t, DiscoveryCategoriesTTL > 0, "DiscoveryCategoriesTTL should be positive")
	assert.True(t, DiscoveryStatsTTL > 0, "DiscoveryStatsTTL should be positive")
	assert.True(t, DiscoveryTagsTTL > 0, "DiscoveryTagsTTL should be positive")
	assert.True(t, DiscoverySuggestionsTTL > 0, "DiscoverySuggestionsTTL should be positive")
	assert.True(t, DiscoveryHomePageTTL > 0, "DiscoveryHomePageTTL should be positive")
	assert.True(t, DiscoverySearchTTL > 0, "DiscoverySearchTTL should be positive")
	assert.True(t, DiscoveryRecommendedTTL > 0, "DiscoveryRecommendedTTL should be positive")
	assert.True(t, DiscoveryServerDetailTTL > 0, "DiscoveryServerDetailTTL should be positive")
	assert.True(t, DiscoveryDirectoryTTL > 0, "DiscoveryDirectoryTTL should be positive")

	// Featured and categories should have longer TTLs than search results
	assert.True(t, DiscoveryFeaturedTTL >= DiscoverySearchTTL, "Featured TTL should be >= search TTL")
	assert.True(t, DiscoveryCategoriesTTL >= DiscoveryCacheTTL, "Categories TTL should be >= general TTL")
}

func TestDiscoveryCacheKeyUniqueness(t *testing.T) {
	// Verify different parameters produce different keys
	key1 := discoveryListKey(1, 20, "gaming", "")
	key2 := discoveryListKey(2, 20, "gaming", "")
	key3 := discoveryListKey(1, 20, "", "gaming")
	key4 := discoveryListKey(1, 25, "gaming", "")

	assert.NotEqual(t, key1, key2, "Different pages should produce different keys")
	assert.NotEqual(t, key1, key3, "Query vs category should produce different keys")
	assert.NotEqual(t, key1, key4, "Different limits should produce different keys")
}

// TestDiscoveryCache_GetDiscoveryServers tests GetDiscoveryServers
func TestDiscoveryCache_GetDiscoveryServers(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss should return error
	_, err := cache.GetDiscoveryServers(ctx, 1, 20, "gaming", "technology")
	assert.Error(t, err)
}

// TestDiscoveryCache_SetAndGetDiscoveryServers tests SetDiscoveryServers and GetDiscoveryServers
func TestDiscoveryCache_SetAndGetDiscoveryServers(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	page, limit := 1, 20
	query, category := "gaming", "technology"

	servers := &models.PaginatedDiscoverableServers{
		Servers:    []*models.DiscoverableServerSearchResult{},
		Total:      100,
		Page:       page,
		Limit:      limit,
		TotalPages: 5,
	}

	// Set then get
	err := cache.SetDiscoveryServers(ctx, page, limit, query, category, servers)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryServers(ctx, page, limit, query, category)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, servers.Total, result.Total)
	assert.Equal(t, servers.Page, result.Page)
	assert.Equal(t, servers.Limit, result.Limit)
	assert.Equal(t, servers.TotalPages, result.TotalPages)
}

// TestDiscoveryCache_GetDiscoveryFeatured tests GetDiscoveryFeatured
func TestDiscoveryCache_GetDiscoveryFeatured(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss
	_, err := cache.GetDiscoveryFeatured(ctx, 10)
	assert.Error(t, err)
}

// TestDiscoveryCache_SetAndGetDiscoveryFeatured tests SetDiscoveryFeatured and GetDiscoveryFeatured
func TestDiscoveryCache_SetAndGetDiscoveryFeatured(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	desc := "A great gaming server"
	servers := []*models.DiscoverableFeaturedServer{
		{
			ID:          uuid.New(),
			ServerID:    uuid.New(),
			Name:        "Gaming Hub",
			Description: &desc,
			MemberCount: 5000,
			IsVerified:  true,
		},
	}

	err := cache.SetDiscoveryFeatured(ctx, 10, servers)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryFeatured(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, servers[0].Name, result[0].Name)
	assert.Equal(t, servers[0].MemberCount, result[0].MemberCount)
	assert.True(t, result[0].IsVerified)
}

// TestDiscoveryCache_SetAndGetDiscoveryTrending tests SetDiscoveryTrending and GetDiscoveryTrending
func TestDiscoveryCache_SetAndGetDiscoveryTrending(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoveryTrending(ctx, 10)
	assert.Error(t, err)

	servers := []*models.TrendingServerInfo{
		{
			TrendScore:         95.5,
			GrowthRate:         12.3,
			ActiveMembersRatio: 0.75,
			RankChange:         2,
		},
	}

	err = cache.SetDiscoveryTrending(ctx, 10, servers)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryTrending(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, servers[0].TrendScore, result[0].TrendScore)
	assert.Equal(t, servers[0].GrowthRate, result[0].GrowthRate)
	assert.Equal(t, servers[0].RankChange, result[0].RankChange)
}

// TestDiscoveryCache_SetAndGetDiscoveryRecommended tests SetDiscoveryRecommended and GetDiscoveryRecommended
func TestDiscoveryCache_SetAndGetDiscoveryRecommended(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Cache miss first
	_, err := cache.GetDiscoveryRecommended(ctx, userID, 10)
	assert.Error(t, err)

	servers := []*models.ServerRecommendation{
		{
			Reason:            "You have mutual friends",
			MutualMemberCount: 15,
		},
	}

	err = cache.SetDiscoveryRecommended(ctx, userID, 10, servers)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryRecommended(ctx, userID, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, servers[0].Reason, result[0].Reason)
	assert.Equal(t, servers[0].MutualMemberCount, result[0].MutualMemberCount)
}

// TestDiscoveryCache_SetAndGetDiscoveryCategories tests SetDiscoveryCategories and GetDiscoveryCategories
func TestDiscoveryCache_SetAndGetDiscoveryCategories(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoveryCategories(ctx)
	assert.Error(t, err)

	categories := []*models.CategoryInfo{
		{Name: "Gaming", Slug: "gaming", ServerCount: 1500},
		{Name: "Technology", Slug: "technology", ServerCount: 800},
	}

	err = cache.SetDiscoveryCategories(ctx, categories)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryCategories(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, categories[0].Name, result[0].Name)
	assert.Equal(t, categories[0].Slug, result[0].Slug)
	assert.Equal(t, categories[0].ServerCount, result[0].ServerCount)
}

// TestDiscoveryCache_SetAndGetDiscoveryCategoriesWithStats tests SetDiscoveryCategoriesWithStats and GetDiscoveryCategoriesWithStats
func TestDiscoveryCache_SetAndGetDiscoveryCategoriesWithStats(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoveryCategoriesWithStats(ctx)
	assert.Error(t, err)

	categories := []*models.CategoryWithStats{
		{
			CategoryInfo:    models.CategoryInfo{Name: "Gaming", Slug: "gaming", ServerCount: 1500},
			TotalMembers:    50000,
			AvgMemberCount:  33.3,
			GrowthRate:      5.2,
		},
	}

	err = cache.SetDiscoveryCategoriesWithStats(ctx, categories)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryCategoriesWithStats(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, categories[0].TotalMembers, result[0].TotalMembers)
	assert.Equal(t, categories[0].GrowthRate, result[0].GrowthRate)
}

// TestDiscoveryCache_SetAndGetDiscoveryStats tests SetDiscoveryStats and GetDiscoveryStats
func TestDiscoveryCache_SetAndGetDiscoveryStats(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoveryStats(ctx)
	assert.Error(t, err)

	stats := &models.DiscoveryPageStats{
		TotalServers:       10000,
		TotalMembers:       500000,
		TotalCategories:    25,
		NewServersThisWeek: 150,
	}

	err = cache.SetDiscoveryStats(ctx, stats)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, stats.TotalServers, result.TotalServers)
	assert.Equal(t, stats.TotalMembers, result.TotalMembers)
	assert.Equal(t, stats.TotalCategories, result.TotalCategories)
	assert.Equal(t, stats.NewServersThisWeek, result.NewServersThisWeek)
}

// TestDiscoveryCache_SetAndGetDiscoveryPopularTags tests SetDiscoveryPopularTags and GetDiscoveryPopularTags
func TestDiscoveryCache_SetAndGetDiscoveryPopularTags(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoveryPopularTags(ctx, 20)
	assert.Error(t, err)

	tags := []*models.DiscoveryTag{
		{ID: uuid.New(), Name: "minecraft", Slug: "minecraft", UsageCount: 5000},
		{ID: uuid.New(), Name: "anime", Slug: "anime", UsageCount: 3500},
	}

	err = cache.SetDiscoveryPopularTags(ctx, 20, tags)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryPopularTags(ctx, 20)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, tags[0].Name, result[0].Name)
	assert.Equal(t, tags[0].UsageCount, result[0].UsageCount)
}

// TestDiscoveryCache_SetAndGetDiscoverySuggestions tests SetDiscoverySuggestions and GetDiscoverySuggestions
func TestDiscoveryCache_SetAndGetDiscoverySuggestions(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Cache miss first
	_, err := cache.GetDiscoverySuggestions(ctx, "gam", 10)
	assert.Error(t, err)

	suggestions := []*models.SearchSuggestion{
		{Type: "tag", Value: "gaming", Count: 500},
		{Type: "category", Value: "Gaming", Count: 0},
	}

	err = cache.SetDiscoverySuggestions(ctx, "gam", 10, suggestions)
	assert.NoError(t, err)

	result, err := cache.GetDiscoverySuggestions(ctx, "gam", 10)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, suggestions[0].Type, result[0].Type)
	assert.Equal(t, suggestions[0].Value, result[0].Value)
	assert.Equal(t, suggestions[0].Count, result[0].Count)
}

// TestDiscoveryCache_SetAndGetDiscoveryServerDetail tests SetDiscoveryServerDetail and GetDiscoveryServerDetail
func TestDiscoveryCache_SetAndGetDiscoveryServerDetail(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	serverID := uuid.New()

	// Cache miss first
	_, err := cache.GetDiscoveryServerDetail(ctx, serverID)
	assert.Error(t, err)

	desc := "The best gaming community"
	server := &models.DiscoverableServer{
		ID:          serverID,
		Name:        "Epic Gaming Server",
		Description: &desc,
	}

	err = cache.SetDiscoveryServerDetail(ctx, serverID, server)
	assert.NoError(t, err)

	result, err := cache.GetDiscoveryServerDetail(ctx, serverID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, server.Name, result.Name)
	assert.Equal(t, server.Description, result.Description)
}

// TestDiscoveryCache_SetAndGetDiscoverySearch tests SetDiscoverySearch and GetDiscoverySearch
func TestDiscoveryCache_SetAndGetDiscoverySearch(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	query, category, sortBy, sortOrder := "minecraft", "gaming", "popular", "desc"
	page, limit := 1, 25

	// Cache miss first
	_, err := cache.GetDiscoverySearch(ctx, query, category, sortBy, sortOrder, page, limit)
	assert.Error(t, err)

	result := &models.DiscoverySearchResponse{
		Servers:    []*models.DiscoverableServerSearchResult{},
		Total:      100,
		Page:       page,
		Limit:      limit,
		TotalPages: 4,
	}

	err = cache.SetDiscoverySearch(ctx, query, category, sortBy, sortOrder, page, limit, result)
	assert.NoError(t, err)

	res, err := cache.GetDiscoverySearch(ctx, query, category, sortBy, sortOrder, page, limit)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, result.Total, res.Total)
	assert.Equal(t, result.Page, res.Page)
}

// TestDiscoveryCache_SetAndGetDiscoveryHomePage tests SetDiscoveryHomePage and GetDiscoveryHomePage
func TestDiscoveryCache_SetAndGetDiscoveryHomePage(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Cache miss first
	_, err := cache.GetDiscoveryHomePage(ctx, userID, 5, 10, 10)
	assert.Error(t, err)

	page := &models.DiscoveryHomePage{
		Featured:    []*models.DiscoverableFeaturedServer{},
		Trending:    []*models.TrendingServerInfo{},
		Recommended: []*models.ServerRecommendation{},
		Categories:  []*models.CategoryWithStats{},
		PopularTags: []*models.DiscoveryTag{},
		Stats:       &models.DiscoveryPageStats{TotalServers: 5000},
	}

	err = cache.SetDiscoveryHomePage(ctx, userID, 5, 10, 10, page)
	assert.NoError(t, err)

	res, err := cache.GetDiscoveryHomePage(ctx, userID, 5, 10, 10)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotNil(t, res.Stats)
	assert.Equal(t, page.Stats.TotalServers, res.Stats.TotalServers)
}

// TestDiscoveryCache_InvalidateDiscoveryCache tests InvalidateDiscoveryCache
func TestDiscoveryCache_InvalidateDiscoveryCache(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// First, set some discovery cache entries
	servers := &models.PaginatedDiscoverableServers{Total: 100}
	err := cache.SetDiscoveryServers(ctx, 1, 20, "gaming", "", servers)
	assert.NoError(t, err)

	featured := []*models.DiscoverableFeaturedServer{}
	err = cache.SetDiscoveryFeatured(ctx, 10, featured)
	assert.NoError(t, err)

	// Verify data is cached
	result, err := cache.GetDiscoveryServers(ctx, 1, 20, "gaming", "")
	assert.NoError(t, err)
	assert.Equal(t, servers.Total, result.Total)

	// Invalidate all discovery cache
	err = cache.InvalidateDiscoveryCache(ctx)
	assert.NoError(t, err)

	// Verify data is gone
	_, err = cache.GetDiscoveryServers(ctx, 1, 20, "gaming", "")
	assert.Error(t, err)

	_, err = cache.GetDiscoveryFeatured(ctx, 10)
	assert.Error(t, err)
}

// TestDiscoveryCache_InvalidJSON tests handling of invalid JSON stored directly
func TestDiscoveryCache_InvalidJSON(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Directly set invalid JSON in Redis (bypassing the cache layer)
	key := cache.prefix + "discovery:servers"
	mr.Set(key, "not valid json")

	// GetDiscoveryServers should return error on invalid JSON
	_, err := cache.GetDiscoveryServers(ctx, 1, 20, "", "")
	assert.Error(t, err)
}
