package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// AnalyticsRepository defines the interface for analytics data access
type AnalyticsRepository interface {
	GetMemberGrowthHistory(ctx context.Context, serverID uuid.UUID, days int) ([]*models.MemberGrowthPoint, error)
	GetMessageActivityStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ActivityHourStat, error)
	GetTopChannels(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.TopChannelStat, error)
	GetRetentionMetrics(ctx context.Context, serverID uuid.UUID, days int) (*models.RetentionMetrics, error)
	GetServerAnalyticsSummary(ctx context.Context, serverID uuid.UUID) (*models.AnalyticsSummary, error)
	GetPeakActivityHours(ctx context.Context, serverID uuid.UUID, days int) ([]*models.PeakHour, error)
	GetMostActiveUsers(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.ActiveUserStat, error)
	TakeMemberSnapshot(ctx context.Context) error
	TakeMemberSnapshotForServer(ctx context.Context, serverID uuid.UUID) error
	CleanupOldAnalyticsData(ctx context.Context, retentionDays int) error
}

// AnalyticsService handles server analytics business logic
type AnalyticsService struct {
	repo         AnalyticsRepository
	serverRepo   ServerRepository
	permService  *PermissionService
	cache        CacheService
	cacheEnabled bool
	cacheTTL     time.Duration
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	repo AnalyticsRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	cache CacheService,
) *AnalyticsService {
	return &AnalyticsService{
		repo:         repo,
		serverRepo:   serverRepo,
		permService:  permService,
		cache:        cache,
		cacheEnabled: cache != nil,
		cacheTTL:     5 * time.Minute, // Cache analytics for 5 minutes
	}
}

// cacheKey generates a cache key for analytics data
func (s *AnalyticsService) cacheKey(serverID uuid.UUID, metric string, days int) string {
	return fmt.Sprintf("analytics:%s:%s:%d", serverID.String(), metric, days)
}

// GetMemberGrowth returns member growth history for a server
func (s *AnalyticsService) GetMemberGrowth(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.MemberGrowthResponse, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Normalize days
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	// Try cache first
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "growth", days)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var response models.MemberGrowthResponse
			if err := json.Unmarshal(cached, &response); err == nil {
				return &response, nil
			}
		}
	}

	// Get data from repository
	data, err := s.repo.GetMemberGrowthHistory(ctx, serverID, days)
	if err != nil {
		return nil, err
	}

	response := &models.MemberGrowthResponse{
		ServerID: serverID.String(),
		Period:   fmt.Sprintf("%dd", days),
		Data:     data,
	}

	// Cache the result
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "growth", days)
		if cached, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, cached, s.cacheTTL)
		}
	}

	return response, nil
}

// GetMessageActivity returns message activity heatmap data
func (s *AnalyticsService) GetMessageActivity(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.ActivityHeatmapResponse, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Normalize days
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	// Try cache first
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "activity", days)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var response models.ActivityHeatmapResponse
			if err := json.Unmarshal(cached, &response); err == nil {
				return &response, nil
			}
		}
	}

	// Get activity data
	data, err := s.repo.GetMessageActivityStats(ctx, serverID, days)
	if err != nil {
		return nil, err
	}

	// Get peak hours
	peakHours, err := s.repo.GetPeakActivityHours(ctx, serverID, days)
	if err != nil {
		// Non-critical, continue without peak hours
		peakHours = nil
	}

	// Calculate total stats
	totalMessages := 0
	for _, stat := range data {
		totalMessages += stat.MessageCount
	}

	response := &models.ActivityHeatmapResponse{
		ServerID:  serverID.String(),
		Period:    fmt.Sprintf("%dd", days),
		Data:      data,
		PeakHours: peakHours,
	}
	response.TotalStats.TotalMessages = totalMessages
	if len(data) > 0 {
		response.TotalStats.AvgPerHour = float64(totalMessages) / float64(len(data))
	}

	// Cache the result
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "activity", days)
		if cached, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, cached, s.cacheTTL)
		}
	}

	return response, nil
}

// GetTopChannels returns channels ranked by message volume
func (s *AnalyticsService) GetTopChannels(ctx context.Context, serverID, requesterID uuid.UUID, days, limit int) (*models.TopChannelsResponse, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Normalize params
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Try cache first
	if s.cacheEnabled {
		cacheKey := fmt.Sprintf("%s:%d", s.cacheKey(serverID, "channels", days), limit)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var response models.TopChannelsResponse
			if err := json.Unmarshal(cached, &response); err == nil {
				return &response, nil
			}
		}
	}

	// Get data from repository
	data, err := s.repo.GetTopChannels(ctx, serverID, days, limit)
	if err != nil {
		return nil, err
	}

	response := &models.TopChannelsResponse{
		ServerID: serverID.String(),
		Period:   fmt.Sprintf("%dd", days),
		Data:     data,
	}

	// Cache the result
	if s.cacheEnabled {
		cacheKey := fmt.Sprintf("%s:%d", s.cacheKey(serverID, "channels", days), limit)
		if cached, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, cached, s.cacheTTL)
		}
	}

	return response, nil
}

// GetRetention returns retention and engagement metrics
func (s *AnalyticsService) GetRetention(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.RetentionResponse, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Normalize days
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	// Try cache first
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "retention", days)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var response models.RetentionResponse
			if err := json.Unmarshal(cached, &response); err == nil {
				return &response, nil
			}
		}
	}

	// Get data from repository
	data, err := s.repo.GetRetentionMetrics(ctx, serverID, days)
	if err != nil {
		return nil, err
	}

	response := &models.RetentionResponse{
		ServerID: serverID.String(),
		Period:   fmt.Sprintf("%dd", days),
		Data:     data,
	}

	// Cache the result
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "retention", days)
		if cached, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, cached, s.cacheTTL)
		}
	}

	return response, nil
}

// GetSummary returns a quick overview of key metrics
func (s *AnalyticsService) GetSummary(ctx context.Context, serverID, requesterID uuid.UUID) (*models.ServerInsightsResponse, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Try cache first (shorter TTL for summary)
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "summary", 0)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var response models.ServerInsightsResponse
			if err := json.Unmarshal(cached, &response); err == nil {
				return &response, nil
			}
		}
	}

	// Get data from repository
	summary, err := s.repo.GetServerAnalyticsSummary(ctx, serverID)
	if err != nil {
		return nil, err
	}

	response := &models.ServerInsightsResponse{
		ServerID: serverID,
		Period:   "7d",
		Summary:  summary,
	}

	// Cache for shorter duration (2 minutes for summary)
	if s.cacheEnabled {
		cacheKey := s.cacheKey(serverID, "summary", 0)
		if cached, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, cached, 2*time.Minute)
		}
	}

	return response, nil
}

// GetMostActiveUsers returns the most active users in a server
func (s *AnalyticsService) GetMostActiveUsers(ctx context.Context, serverID, requesterID uuid.UUID, days, limit int) ([]*models.ActiveUserStat, error) {
	// Check permissions
	if err := s.checkPermissions(ctx, serverID, requesterID); err != nil {
		return nil, err
	}

	// Normalize params
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}

	return s.repo.GetMostActiveUsers(ctx, serverID, days, limit)
}

// checkPermissions verifies the requester has MANAGE_SERVER permission
func (s *AnalyticsService) checkPermissions(ctx context.Context, serverID, requesterID uuid.UUID) error {
	// Verify server exists
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	// Owner always has access
	if server.OwnerID == requesterID {
		return nil
	}

	// Check MANAGE_SERVER permission
	if s.permService != nil {
		return s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageServer)
	}

	return nil
}

// InvalidateCache clears cached analytics for a server
func (s *AnalyticsService) InvalidateCache(ctx context.Context, serverID uuid.UUID) error {
	if !s.cacheEnabled {
		return nil
	}

	// Clear all analytics cache keys for this server
	metrics := []string{"growth", "activity", "channels", "retention", "summary"}
	days := []int{0, 7, 14, 30, 90}

	for _, metric := range metrics {
		for _, d := range days {
			key := s.cacheKey(serverID, metric, d)
			_ = s.cache.Delete(ctx, key)
		}
	}

	return nil
}

// TakeDailySnapshots should be called by a scheduler to capture daily member counts
func (s *AnalyticsService) TakeDailySnapshots(ctx context.Context) error {
	return s.repo.TakeMemberSnapshot(ctx)
}

// CleanupOldData removes analytics data older than the retention period
func (s *AnalyticsService) CleanupOldData(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 90 // Default retention
	}
	return s.repo.CleanupOldAnalyticsData(ctx, retentionDays)
}
