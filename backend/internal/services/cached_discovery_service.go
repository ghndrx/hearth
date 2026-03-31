package services

import (
	"context"

	"github.com/google/uuid"

	"hearth/internal/cache"
	"hearth/internal/models"
)

// CachedDiscoverableServerService wraps DiscoverableServerService with Redis caching
type CachedDiscoverableServerService struct {
	*DiscoverableServerService
	cache *cache.RedisCache
}

// NewCachedDiscoverableServerService creates a cached discoverable server service
func NewCachedDiscoverableServerService(
	svc *DiscoverableServerService,
	redisCache *cache.RedisCache,
) *CachedDiscoverableServerService {
	return &CachedDiscoverableServerService{
		DiscoverableServerService: svc,
		cache:                     redisCache,
	}
}

// GetDiscoverableServers returns paginated discoverable servers with caching
func (s *CachedDiscoverableServerService) GetDiscoverableServers(ctx context.Context, filters *models.DiscoverFilters) (*models.PaginatedDiscoverableServers, error) {
	models.NormalizeDiscoverFilters(filters)

	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryServers(ctx, filters.Page, filters.Limit, filters.Query, string(filters.Category))
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetDiscoverableServers(ctx, filters)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryServers(ctx, filters.Page, filters.Limit, filters.Query, string(filters.Category), result)
	}

	return result, nil
}

// GetFeaturedServers returns featured servers with caching
func (s *CachedDiscoverableServerService) GetFeaturedServers(ctx context.Context, limit int) ([]*models.DiscoverableFeaturedServer, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryFeatured(ctx, limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetFeaturedServers(ctx, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryFeatured(ctx, limit, result)
	}

	return result, nil
}

// GetTrendingServers returns trending servers with caching
func (s *CachedDiscoverableServerService) GetTrendingServers(ctx context.Context, limit int) ([]*models.TrendingServerInfo, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryTrending(ctx, limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetTrendingServers(ctx, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryTrending(ctx, limit, result)
	}

	return result, nil
}

// GetRecommendedServers returns personalized recommendations with caching
func (s *CachedDiscoverableServerService) GetRecommendedServers(ctx context.Context, userID uuid.UUID, limit int) ([]*models.ServerRecommendation, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryRecommended(ctx, userID, limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetRecommendedServers(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryRecommended(ctx, userID, limit, result)
	}

	return result, nil
}

// GetCategories returns categories with caching
func (s *CachedDiscoverableServerService) GetCategories(ctx context.Context) ([]*models.CategoryInfo, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryCategories(ctx)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryCategories(ctx, result)
	}

	return result, nil
}

// GetCategoriesWithStats returns categories with stats and caching
func (s *CachedDiscoverableServerService) GetCategoriesWithStats(ctx context.Context) ([]*models.CategoryWithStats, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryCategoriesWithStats(ctx)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetCategoriesWithStats(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryCategoriesWithStats(ctx, result)
	}

	return result, nil
}

// GetDiscoveryStats returns discovery stats with caching
func (s *CachedDiscoverableServerService) GetDiscoveryStats(ctx context.Context) (*models.DiscoveryPageStats, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryStats(ctx)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetDiscoveryStats(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryStats(ctx, result)
	}

	return result, nil
}

// GetPopularTags returns popular tags with caching
func (s *CachedDiscoverableServerService) GetPopularTags(ctx context.Context, limit int) ([]*models.DiscoveryTag, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryPopularTags(ctx, limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetPopularTags(ctx, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryPopularTags(ctx, limit, result)
	}

	return result, nil
}

// GetSearchSuggestions returns search suggestions with caching
func (s *CachedDiscoverableServerService) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]*models.SearchSuggestion, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoverySuggestions(ctx, query, limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetSearchSuggestions(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoverySuggestions(ctx, query, limit, result)
	}

	return result, nil
}

// SearchServersEnhanced performs enhanced search with caching
func (s *CachedDiscoverableServerService) SearchServersEnhanced(ctx context.Context, req *models.DiscoverySearchRequest) (*models.DiscoverySearchResponse, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoverySearch(ctx, req.Query, string(req.Category), req.SortBy, req.SortOrder, req.Page, req.Limit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.SearchServersEnhanced(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoverySearch(ctx, req.Query, string(req.Category), req.SortBy, req.SortOrder, req.Page, req.Limit, result)
	}

	return result, nil
}

// GetDiscoveryHomePage returns the full discovery home page with caching
func (s *CachedDiscoverableServerService) GetDiscoveryHomePage(ctx context.Context, userID uuid.UUID, featuredLimit, trendingLimit, recommendedLimit int) (*models.DiscoveryHomePage, error) {
	if s.cache != nil {
		cached, err := s.cache.GetDiscoveryHomePage(ctx, userID, featuredLimit, trendingLimit, recommendedLimit)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	result, err := s.DiscoverableServerService.GetDiscoveryHomePage(ctx, userID, featuredLimit, trendingLimit, recommendedLimit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetDiscoveryHomePage(ctx, userID, featuredLimit, trendingLimit, recommendedLimit, result)
	}

	return result, nil
}

// RegisterServer registers and invalidates cache
func (s *CachedDiscoverableServerService) RegisterServer(ctx context.Context, serverID, ownerID uuid.UUID, req *models.RegisterServerRequest) (*models.DiscoverableServer, error) {
	result, err := s.DiscoverableServerService.RegisterServer(ctx, serverID, ownerID, req)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateDiscoveryCache(ctx)
	}

	return result, nil
}

// UpdateRegisteredServer updates and invalidates cache
func (s *CachedDiscoverableServerService) UpdateRegisteredServer(ctx context.Context, id, ownerID uuid.UUID, req *models.UpdateDiscoverableServerRequest) (*models.DiscoverableServer, error) {
	result, err := s.DiscoverableServerService.UpdateRegisteredServer(ctx, id, ownerID, req)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateDiscoveryCache(ctx)
	}

	return result, nil
}

// DeleteRegisteredServer deletes and invalidates cache
func (s *CachedDiscoverableServerService) DeleteRegisteredServer(ctx context.Context, id, ownerID uuid.UUID) error {
	err := s.DiscoverableServerService.DeleteRegisteredServer(ctx, id, ownerID)
	if err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateDiscoveryCache(ctx)
	}

	return nil
}
