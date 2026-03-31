package cache

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
