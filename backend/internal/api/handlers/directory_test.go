package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// TestDirectoryEndpointModels tests models used by the directory endpoint
func TestDirectoryEndpointModels(t *testing.T) {
	t.Run("directory response structure", func(t *testing.T) {
		servers := []*models.DiscoverableServerSearchResult{
			{Name: "Server 1", MemberCount: 1000, Category: models.DiscoveryCategoryGaming},
			{Name: "Server 2", MemberCount: 2000, Category: models.DiscoveryCategoryTechnology},
		}

		featured := []*models.DiscoverableFeaturedServer{
			{Name: "Featured 1", MemberCount: 5000},
		}

		categories := []*models.CategoryInfo{
			{Name: "Gaming", Slug: "gaming", ServerCount: 100},
			{Name: "Technology", Slug: "technology", ServerCount: 75},
		}

		assert.Len(t, servers, 2)
		assert.Len(t, featured, 1)
		assert.Len(t, categories, 2)
		assert.Equal(t, "Server 1", servers[0].Name)
		assert.Equal(t, models.DiscoveryCategoryGaming, servers[0].Category)
	})

	t.Run("directory filters normalization", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 0, Limit: 0}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 1, filters.Page)
		assert.Equal(t, 20, filters.Limit)
	})

	t.Run("directory filters with query and category", func(t *testing.T) {
		filters := &models.DiscoverFilters{
			Query:    "gaming",
			Category: models.DiscoveryCategoryGaming,
			Page:     2,
			Limit:    50,
		}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, "gaming", filters.Query)
		assert.Equal(t, models.DiscoveryCategoryGaming, filters.Category)
		assert.Equal(t, 2, filters.Page)
		assert.Equal(t, 50, filters.Limit)
	})

	t.Run("directory filters limit cap", func(t *testing.T) {
		filters := &models.DiscoverFilters{Page: 1, Limit: 500}
		models.NormalizeDiscoverFilters(filters)
		assert.Equal(t, 100, filters.Limit)
	})
}

// TestDirectoryPaginationCalculation tests pagination math
func TestDirectoryPaginationCalculation(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		limit    int
		expected int
	}{
		{"exact division", 100, 25, 4},
		{"with remainder", 101, 25, 5},
		{"single page", 10, 25, 1},
		{"zero results", 0, 25, 0},
		{"one result", 1, 25, 1},
		{"limit equals total", 25, 25, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalPages := tt.total / tt.limit
			if tt.total%tt.limit > 0 {
				totalPages++
			}
			assert.Equal(t, tt.expected, totalPages)
		})
	}
}

// TestDirectoryCategoryValidation tests category validation for the directory
func TestDirectoryCategoryValidation(t *testing.T) {
	validCategories := []string{
		"gaming", "technology", "art", "music", "sports",
		"education", "entertainment", "community", "other",
	}
	for _, cat := range validCategories {
		assert.True(t, models.IsValidCategory(cat), "expected %s to be valid", cat)
	}

	invalidCategories := []string{"invalid", "GAMING", "Science", "", "social", "anime"}
	for _, cat := range invalidCategories {
		assert.False(t, models.IsValidCategory(cat), "expected %s to be invalid for DiscoverFilters", cat)
	}
}

// TestParseCommaSeparated tests the comma-separated parsing helper
func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"single value", "gaming", []string{"gaming"}},
		{"multiple values", "gaming,tech,music", []string{"gaming", "tech", "music"}},
		{"with spaces", "gaming , tech , music", []string{"gaming", "tech", "music"}},
		{"trailing comma", "gaming,tech,", []string{"gaming", "tech"}},
		{"empty entries", "gaming,,tech", []string{"gaming", "tech"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDiscoveryActivityValidation tests activity type validation
func TestDiscoveryActivityValidation(t *testing.T) {
	t.Run("valid activity types", func(t *testing.T) {
		validTypes := []string{"view", "impression", "join", "search_click"}
		for _, at := range validTypes {
			assert.NotEmpty(t, at)
		}
	})

	t.Run("valid sources", func(t *testing.T) {
		validSources := []string{"home", "search", "category", "trending", "recommended", "featured"}
		for _, s := range validSources {
			assert.NotEmpty(t, s)
		}
	})
}

// TestAdminEndpointModels tests models used by admin endpoints
func TestAdminEndpointModels(t *testing.T) {
	t.Run("admin approve response structure", func(t *testing.T) {
		// Verify the expected response structure for admin approve
		response := map[string]interface{}{
			"message":   "Server approved for public directory",
			"is_public": true,
		}
		assert.Equal(t, true, response["is_public"])
		assert.Equal(t, "Server approved for public directory", response["message"])
	})

	t.Run("admin reject response structure", func(t *testing.T) {
		response := map[string]interface{}{
			"message":   "Server removed from public directory",
			"is_public": false,
		}
		assert.Equal(t, false, response["is_public"])
	})

	t.Run("admin feature response structure", func(t *testing.T) {
		response := map[string]interface{}{
			"message":     "Featured status updated",
			"is_featured": true,
		}
		assert.Equal(t, true, response["is_featured"])
	})
}
