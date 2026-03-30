package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// TestDiscoverFiltersNormalization tests filter normalization
func TestDiscoverFiltersNormalization(t *testing.T) {
	t.Run("normalize zero page to 1", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 0, Limit: 20}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 1, filters.Page)
	})

	t.Run("normalize negative page to 1", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: -5, Limit: 20}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 1, filters.Page)
	})

	t.Run("normalize zero limit to 20", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 0}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 20, filters.Limit)
	})

	t.Run("cap limit at 100", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 500}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 100, filters.Limit)
	})

	t.Run("default values", func(t *testing.T) {
		filters := &models.DiscoverFilters{}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 1, filters.Page)
		assert.Equal(t, 20, filters.Limit)
	})
}

// TestIsValidCategory tests category validation
func TestIsValidCategory(t *testing.T) {
	t.Run("valid categories", func(t *testing.T) {
		validCategories := []string{"gaming", "technology", "art", "music", "sports", "education", "entertainment", "community", "other"}
		for _, cat := range validCategories {
			assert.True(t, models.IsValidCategory(cat), "expected %s to be valid", cat)
		}
	})

	t.Run("invalid categories", func(t *testing.T) {
		invalidCategories := []string{"invalid", "GAMING", "Science", ""}
		for _, cat := range invalidCategories {
			assert.False(t, models.IsValidCategory(cat), "expected %s to be invalid", cat)
		}
	})
}

// TestAllDiscoveryCategories tests that all categories are properly defined
func TestAllDiscoveryCategories(t *testing.T) {
	categories := models.AllDiscoveryCategories()
	assert.Len(t, categories, 9)
	
	expected := []models.ServerDiscoveryCategory{
		models.DiscoveryCategoryGaming,
		models.DiscoveryCategoryTechnology,
		models.DiscoveryCategoryArt,
		models.DiscoveryCategoryMusic,
		models.DiscoveryCategorySports,
		models.DiscoveryCategoryEducation,
		models.DiscoveryCategoryEntertainment,
		models.DiscoveryCategoryCommunity,
		models.DiscoveryCategoryOther,
	}
	
	for _, exp := range expected {
		found := false
		for _, cat := range categories {
			if cat == exp {
				found = true
				break
			}
		}
		assert.True(t, found, "expected category %s not found", exp)
	}
}

// TestPaginatedDiscoverableServersResponse tests the response structure
func TestPaginatedDiscoverableServersResponse(t *testing.T) {
	t.Run("response has correct structure", func(t *testing.T) {
		result := &models.PaginatedDiscoverableServers{
			Servers:    []*models.DiscoverableServerSearchResult{},
			Total:      100,
			Page:       1,
			Limit:      20,
			TotalPages: 5,
		}

		assert.Equal(t, 100, result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.Limit)
		assert.Equal(t, 5, result.TotalPages)
	})
}

// TestDiscoverySearchRequest tests the search request model
func TestDiscoverySearchRequest(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		req := &models.DiscoverySearchRequest{}
		
		assert.Equal(t, "", req.Query)
		assert.Equal(t, models.ServerDiscoveryCategory(""), req.Category)
		assert.Nil(t, req.Categories)
		assert.Nil(t, req.Tags)
		assert.Equal(t, "", req.SortBy)
		assert.Equal(t, "", req.SortOrder)
		assert.Equal(t, 0, req.Page)
		assert.Equal(t, 0, req.Limit)
	})

	t.Run("with values", func(t *testing.T) {
		req := &models.DiscoverySearchRequest{
			Query:     "gaming",
			Category:  models.DiscoveryCategoryGaming,
			SortBy:    "popular",
			SortOrder: "desc",
			Page:      1,
			Limit:     25,
		}
		
		assert.Equal(t, "gaming", req.Query)
		assert.Equal(t, models.DiscoveryCategoryGaming, req.Category)
		assert.Equal(t, "popular", req.SortBy)
		assert.Equal(t, "desc", req.SortOrder)
		assert.Equal(t, 1, req.Page)
		assert.Equal(t, 25, req.Limit)
	})
}

// TestDiscoverySearchResponse tests the search response model
func TestDiscoverySearchResponse(t *testing.T) {
	t.Run("response pagination", func(t *testing.T) {
		resp := &models.DiscoverySearchResponse{
			Servers:    []*models.DiscoverableServerSearchResult{},
			Total:      150,
			Page:       2,
			Limit:      25,
			TotalPages: 6,
		}

		assert.Equal(t, 150, resp.Total)
		assert.Equal(t, 2, resp.Page)
		assert.Equal(t, 25, resp.Limit)
		assert.Equal(t, 6, resp.TotalPages)
		assert.Len(t, resp.Servers, 0)
	})
}

// TestDiscoveryHomePage tests the home page model
func TestDiscoveryHomePage(t *testing.T) {
	t.Run("home page structure", func(t *testing.T) {
		page := &models.DiscoveryHomePage{
			Featured:      []*models.DiscoverableFeaturedServer{},
			Trending:      []*models.TrendingServerInfo{},
			Recommended:   []*models.ServerRecommendation{},
			Categories:    []*models.CategoryWithStats{},
			PopularTags:   []*models.DiscoveryTag{},
			Stats: &models.DiscoveryPageStats{
				TotalServers:       1000,
				TotalMembers:       50000,
				TotalCategories:    9,
				NewServersThisWeek: 25,
			},
		}

		assert.NotNil(t, page.Featured)
		assert.NotNil(t, page.Trending)
		assert.NotNil(t, page.Recommended)
		assert.NotNil(t, page.Categories)
		assert.NotNil(t, page.PopularTags)
		assert.NotNil(t, page.Stats)
		assert.Equal(t, int64(1000), page.Stats.TotalServers)
		assert.Equal(t, int64(50000), page.Stats.TotalMembers)
		assert.Equal(t, 9, page.Stats.TotalCategories)
		assert.Equal(t, 25, page.Stats.NewServersThisWeek)
	})
}

// TestTrendingServerInfo tests the trending server model
func TestTrendingServerInfo(t *testing.T) {
	t.Run("trending server values", func(t *testing.T) {
		trending := &models.TrendingServerInfo{
			Server: &models.DiscoverableServerSearchResult{
				Name: "Test Server",
			},
			TrendScore:          85.5,
			GrowthRate:          12.3,
			ActiveMembersRatio:  0.45,
			RankChange:          3,
		}

		assert.Equal(t, 85.5, trending.TrendScore)
		assert.Equal(t, 12.3, trending.GrowthRate)
		assert.Equal(t, 0.45, trending.ActiveMembersRatio)
		assert.Equal(t, 3, trending.RankChange)
		assert.Equal(t, "Test Server", trending.Server.Name)
	})
}

// TestServerRecommendation tests the recommendation model
func TestServerRecommendation(t *testing.T) {
	t.Run("recommendation with reason", func(t *testing.T) {
		rec := &models.ServerRecommendation{
			DiscoverableServerSearchResult: models.DiscoverableServerSearchResult{
				Name:        "Recommended Server",
				MemberCount: 5000,
			},
			Reason:           "Popular in your interests",
			MutualMemberCount: 42,
			MutualServers:    []string{"server-1", "server-2"},
		}

		assert.Equal(t, "Popular in your interests", rec.Reason)
		assert.Equal(t, 42, rec.MutualMemberCount)
		assert.Len(t, rec.MutualServers, 2)
	})
}

// TestCategoryWithStats tests the category with stats model
func TestCategoryWithStats(t *testing.T) {
	t.Run("category stats", func(t *testing.T) {
		cat := &models.CategoryWithStats{
			CategoryInfo: models.CategoryInfo{
				Name:        "Gaming",
				Slug:        "gaming",
				ServerCount: 150,
			},
			TotalMembers:   25000,
			AvgMemberCount: 166.67,
			GrowthRate:     5.5,
		}

		assert.Equal(t, "Gaming", cat.Name)
		assert.Equal(t, "gaming", cat.Slug)
		assert.Equal(t, 150, cat.ServerCount)
		assert.Equal(t, 25000, cat.TotalMembers)
		assert.InDelta(t, 166.67, cat.AvgMemberCount, 0.01)
		assert.Equal(t, 5.5, cat.GrowthRate)
	})
}

// TestSearchSuggestion tests the search suggestion model
func TestSearchSuggestion(t *testing.T) {
	t.Run("search suggestion types", func(t *testing.T) {
		suggestions := []*models.SearchSuggestion{
			{Type: "server", Value: "Gaming Hub"},
			{Type: "category", Value: "gaming", Count: 150},
			{Type: "tag", Value: "esports"},
		}

		assert.Equal(t, "server", suggestions[0].Type)
		assert.Equal(t, "Gaming Hub", suggestions[0].Value)
		assert.Equal(t, "category", suggestions[1].Type)
		assert.Equal(t, 150, suggestions[1].Count)
		assert.Equal(t, "tag", suggestions[2].Type)
	})
}

// TestDiscoverableServerSearchResult tests the search result model
func TestDiscoverableServerSearchResult(t *testing.T) {
	t.Run("search result fields", func(t *testing.T) {
		desc := "A test server description"
		result := &models.DiscoverableServerSearchResult{
			Name:        "Test Server",
			Description: &desc,
			Category:    models.DiscoveryCategoryGaming,
			MemberCount: 10000,
			IsFeatured:  true,
			IsVerified:  true,
		}

		assert.Equal(t, "Test Server", result.Name)
		assert.NotNil(t, result.Description)
		assert.Equal(t, "A test server description", *result.Description)
		assert.Equal(t, models.DiscoveryCategoryGaming, result.Category)
		assert.Equal(t, 10000, result.MemberCount)
		assert.True(t, result.IsFeatured)
		assert.True(t, result.IsVerified)
	})
}

// TestDiscoverableFeaturedServer tests the featured server model
func TestDiscoverableFeaturedServer(t *testing.T) {
	t.Run("featured server fields", func(t *testing.T) {
		now := time.Now()
		result := &models.DiscoverableFeaturedServer{
			Name:        "Featured Server",
			MemberCount: 50000,
			FeaturedAt:  now,
		}

		assert.Equal(t, "Featured Server", result.Name)
		assert.Equal(t, 50000, result.MemberCount)
		assert.NotNil(t, result.FeaturedAt)
	})
}

// TestCategoryInfo tests the category info model
func TestCategoryInfo(t *testing.T) {
	t.Run("category info basic", func(t *testing.T) {
		info := &models.CategoryInfo{
			Name:        "Music",
			Slug:        "music",
			ServerCount: 75,
		}

		assert.Equal(t, "Music", info.Name)
		assert.Equal(t, "music", info.Slug)
		assert.Equal(t, 75, info.ServerCount)
	})
}
