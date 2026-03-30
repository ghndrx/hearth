package handlers

import (
	"testing"

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
