package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hearth/internal/models"

	"github.com/google/uuid"
)

// Discovery cache TTLs
const (
	DiscoveryCacheTTL        = 2 * time.Minute
	DiscoveryFeaturedTTL     = 5 * time.Minute
	DiscoveryTrendingTTL     = 3 * time.Minute
	DiscoveryCategoriesTTL   = 10 * time.Minute
	DiscoveryStatsTTL        = 5 * time.Minute
	DiscoveryTagsTTL         = 5 * time.Minute
	DiscoverySuggestionsTTL  = 1 * time.Minute
	DiscoveryHomePageTTL     = 2 * time.Minute
	DiscoverySearchTTL       = 1 * time.Minute
	DiscoveryRecommendedTTL  = 3 * time.Minute
	DiscoveryServerDetailTTL = 5 * time.Minute
	DiscoveryDirectoryTTL    = 2 * time.Minute
)

// Discovery cache key helpers

func discoveryListKey(page, limit int, query, category string) string {
	return fmt.Sprintf("discovery:list:p%d:l%d:q%s:c%s", page, limit, query, category)
}

func discoveryFeaturedKey(limit int) string {
	return fmt.Sprintf("discovery:featured:%d", limit)
}

func discoveryTrendingKey(limit int) string {
	return fmt.Sprintf("discovery:trending:%d", limit)
}

func discoveryRecommendedKey(userID uuid.UUID, limit int) string {
	return fmt.Sprintf("discovery:recommended:%s:%d", userID.String(), limit)
}

func discoveryCategoriesKey() string {
	return "discovery:categories"
}

func discoveryCategoriesWithStatsKey() string {
	return "discovery:categories:stats"
}

func discoveryStatsKey() string {
	return "discovery:stats"
}

func discoveryPopularTagsKey(limit int) string {
	return fmt.Sprintf("discovery:tags:popular:%d", limit)
}

func discoverySuggestionsKey(query string, limit int) string {
	return fmt.Sprintf("discovery:suggestions:%s:%d", query, limit)
}

func discoveryServerDetailKey(id uuid.UUID) string {
	return fmt.Sprintf("discovery:server:%s", id.String())
}

func discoverySearchKey(query, category, sortBy, sortOrder string, page, limit int) string {
	return fmt.Sprintf("discovery:search:q%s:c%s:s%s:o%s:p%d:l%d", query, category, sortBy, sortOrder, page, limit)
}

func discoveryHomePageKey(userID uuid.UUID, fl, tl, rl int) string {
	return fmt.Sprintf("discovery:home:%s:%d:%d:%d", userID.String(), fl, tl, rl)
}

func discoveryDirectoryKey(query, category, sort, order string, limit, offset int) string {
	return fmt.Sprintf("discovery:directory:q%s:c%s:s%s:o%s:l%d:off%d", query, category, sort, order, limit, offset)
}

// GetDiscoveryServers retrieves cached paginated discovery servers
func (c *RedisCache) GetDiscoveryServers(ctx context.Context, page, limit int, query, category string) (*models.PaginatedDiscoverableServers, error) {
	data, err := c.Get(ctx, discoveryListKey(page, limit, query, category))
	if err != nil {
		return nil, err
	}
	var result models.PaginatedDiscoverableServers
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDiscoveryServers caches paginated discovery servers
func (c *RedisCache) SetDiscoveryServers(ctx context.Context, page, limit int, query, category string, result *models.PaginatedDiscoverableServers) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryListKey(page, limit, query, category), data, DiscoveryCacheTTL)
}

// GetDiscoveryFeatured retrieves cached featured servers
func (c *RedisCache) GetDiscoveryFeatured(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	data, err := c.Get(ctx, discoveryFeaturedKey(limit))
	if err != nil {
		return nil, err
	}
	var result []*models.DiscoverableFeaturedServer
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryFeatured caches featured servers
func (c *RedisCache) SetDiscoveryFeatured(ctx context.Context, limit int, servers []*models.DiscoverableFeaturedServer) error {
	data, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryFeaturedKey(limit), data, DiscoveryFeaturedTTL)
}

// GetDiscoveryTrending retrieves cached trending servers
func (c *RedisCache) GetDiscoveryTrending(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	data, err := c.Get(ctx, discoveryTrendingKey(limit))
	if err != nil {
		return nil, err
	}
	var result []*models.TrendingServerInfo
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryTrending caches trending servers
func (c *RedisCache) SetDiscoveryTrending(ctx context.Context, limit int, servers []*models.TrendingServerInfo) error {
	data, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryTrendingKey(limit), data, DiscoveryTrendingTTL)
}

// GetDiscoveryRecommended retrieves cached recommendations for a user
func (c *RedisCache) GetDiscoveryRecommended(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	data, err := c.Get(ctx, discoveryRecommendedKey(userID, limit))
	if err != nil {
		return nil, err
	}
	var result []*models.ServerRecommendation
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryRecommended caches recommendations for a user
func (c *RedisCache) SetDiscoveryRecommended(ctx context.Context, userID uuid.UUID, limit int, servers []*models.ServerRecommendation) error {
	data, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryRecommendedKey(userID, limit), data, DiscoveryRecommendedTTL)
}

// GetDiscoveryCategories retrieves cached categories
func (c *RedisCache) GetDiscoveryCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	data, err := c.Get(ctx, discoveryCategoriesKey())
	if err != nil {
		return nil, err
	}
	var result []*models.CategoryInfo
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryCategories caches categories
func (c *RedisCache) SetDiscoveryCategories(ctx context.Context, categories []*models.CategoryInfo) error {
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryCategoriesKey(), data, DiscoveryCategoriesTTL)
}

// GetDiscoveryCategoriesWithStats retrieves cached categories with stats
func (c *RedisCache) GetDiscoveryCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	data, err := c.Get(ctx, discoveryCategoriesWithStatsKey())
	if err != nil {
		return nil, err
	}
	var result []*models.CategoryWithStats
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryCategoriesWithStats caches categories with stats
func (c *RedisCache) SetDiscoveryCategoriesWithStats(ctx context.Context, categories []*models.CategoryWithStats) error {
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryCategoriesWithStatsKey(), data, DiscoveryCategoriesTTL)
}

// GetDiscoveryStats retrieves cached discovery stats
func (c *RedisCache) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	data, err := c.Get(ctx, discoveryStatsKey())
	if err != nil {
		return nil, err
	}
	var result models.DiscoveryPageStats
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDiscoveryStats caches discovery stats
func (c *RedisCache) SetDiscoveryStats(ctx context.Context, stats *models.DiscoveryPageStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryStatsKey(), data, DiscoveryStatsTTL)
}

// GetDiscoveryPopularTags retrieves cached popular tags
func (c *RedisCache) GetDiscoveryPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	data, err := c.Get(ctx, discoveryPopularTagsKey(limit))
	if err != nil {
		return nil, err
	}
	var result []*models.DiscoveryTag
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoveryPopularTags caches popular tags
func (c *RedisCache) SetDiscoveryPopularTags(ctx context.Context, limit int, tags []*models.DiscoveryTag) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryPopularTagsKey(limit), data, DiscoveryTagsTTL)
}

// GetDiscoverySuggestions retrieves cached search suggestions
func (c *RedisCache) GetDiscoverySuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	data, err := c.Get(ctx, discoverySuggestionsKey(query, limit))
	if err != nil {
		return nil, err
	}
	var result []*models.SearchSuggestion
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetDiscoverySuggestions caches search suggestions
func (c *RedisCache) SetDiscoverySuggestions(ctx context.Context, query string, limit int, suggestions []*models.SearchSuggestion) error {
	data, err := json.Marshal(suggestions)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoverySuggestionsKey(query, limit), data, DiscoverySuggestionsTTL)
}

// GetDiscoveryServerDetail retrieves cached server detail
func (c *RedisCache) GetDiscoveryServerDetail(ctx context.Context, id uuid.UUID) (*models.DiscoverableServer, error) {
	data, err := c.Get(ctx, discoveryServerDetailKey(id))
	if err != nil {
		return nil, err
	}
	var result models.DiscoverableServer
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDiscoveryServerDetail caches server detail
func (c *RedisCache) SetDiscoveryServerDetail(ctx context.Context, id uuid.UUID, server *models.DiscoverableServer) error {
	data, err := json.Marshal(server)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryServerDetailKey(id), data, DiscoveryServerDetailTTL)
}

// GetDiscoverySearch retrieves cached search results
func (c *RedisCache) GetDiscoverySearch(ctx context.Context, query, category, sortBy, sortOrder string, page, limit int) (*models.DiscoverySearchResponse, error) {
	data, err := c.Get(ctx, discoverySearchKey(query, category, sortBy, sortOrder, page, limit))
	if err != nil {
		return nil, err
	}
	var result models.DiscoverySearchResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDiscoverySearch caches search results
func (c *RedisCache) SetDiscoverySearch(ctx context.Context, query, category, sortBy, sortOrder string, page, limit int, result *models.DiscoverySearchResponse) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoverySearchKey(query, category, sortBy, sortOrder, page, limit), data, DiscoverySearchTTL)
}

// GetDiscoveryHomePage retrieves cached home page data
func (c *RedisCache) GetDiscoveryHomePage(ctx context.Context, userID uuid.UUID, fl, tl, rl int) (*models.DiscoveryHomePage, error) {
	data, err := c.Get(ctx, discoveryHomePageKey(userID, fl, tl, rl))
	if err != nil {
		return nil, err
	}
	var result models.DiscoveryHomePage
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDiscoveryHomePage caches home page data
func (c *RedisCache) SetDiscoveryHomePage(ctx context.Context, userID uuid.UUID, fl, tl, rl int, page *models.DiscoveryHomePage) error {
	data, err := json.Marshal(page)
	if err != nil {
		return err
	}
	return c.Set(ctx, discoveryHomePageKey(userID, fl, tl, rl), data, DiscoveryHomePageTTL)
}

// InvalidateDiscoveryCache clears all discovery-related cache entries using pattern deletion
func (c *RedisCache) InvalidateDiscoveryCache(ctx context.Context) error {
	iter := c.client.Scan(ctx, 0, c.prefix+"discovery:*", 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}
	return iter.Err()
}
