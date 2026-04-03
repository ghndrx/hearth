package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// AuditLogFilter contains filtering options for audit log queries
type AuditLogFilter struct {
	ActionType     string     // Filter by specific action type (e.g., "MEMBER_BAN", "CHANNEL_CREATE")
	ActionCategory int       // Filter by action category
	ActorID        *uuid.UUID // Filter by the user who performed the action
	TargetID       *uuid.UUID // Filter by the target of the action
	TargetType     string     // Filter by target type (member, message, channel, role)
	Before         *time.Time // Filter entries before this time
	After          *time.Time // Filter entries after this time
	ReasonKeyword  string     // Filter by keyword in reason
	Limit          int        // Maximum number of entries to return (default 50, max 100)
	Offset         int        // Offset for pagination
}

// AuditLogServiceInterface defines the audit log service methods
type AuditLogServiceInterface interface {
	Log(ctx context.Context, entry *models.AuditLogEntry) error
	LogBatch(ctx context.Context, entries []models.AuditLogEntry) error
	GetLogs(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error)
	GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error)
	GetActionTypes() []string
	GetCategories() []models.AuditLogCategoryInfo
	GetDashboardSummary(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardSummary, error)
	GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]models.DailyModerationTrend, error)
	GetModeratorActivity(ctx context.Context, serverID uuid.UUID, days int) ([]models.ModeratorStats, error)
	GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, days, minCount int) ([]models.RepeatOffenderStats, error)
	GetAutoModStats(ctx context.Context, serverID uuid.UUID, days int) (*models.AutoModStats, error)
	ExportLogs(ctx context.Context, serverID uuid.UUID, format string, filter AuditLogFilter) ([]byte, string, error)
	ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]models.AuditLogEntry, error)
	CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error)
}

// AuditLogRepositoryInterface defines the repository methods needed by the service
type AuditLogRepositoryInterface interface {
	Create(ctx context.Context, log *models.AuditLogEntry) error
	CreateBatch(ctx context.Context, logs []models.AuditLogEntry) error
	GetByServer(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error)
	GetByID(ctx context.Context, serverID, logID uuid.UUID) (*models.AuditLogEntry, error)
	GetActionCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[string]int, error)
	GetActionCategoryCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[int]int, error)
	GetModeratorActivity(ctx context.Context, serverID uuid.UUID, since time.Time) ([]ModeratorActivity, error)
	GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, since time.Time, minCount int) ([]RepeatOffender, error)
	GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]DailyTrendPoint, error)
	GetModerationRatios(ctx context.Context, serverID uuid.UUID, since time.Time) (*ModerationRatios, error)
	GetDashboardSummary(ctx context.Context, serverID uuid.UUID, since time.Time) (*DashboardSummary, error)
	GetAutoModStats(ctx context.Context, serverID uuid.UUID, since time.Time) (*AutoModStats, error)
	ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]AuditLog, error)
	ExportToCSV(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) (*strings.Builder, error)
	DeleteOlderThan(ctx context.Context, serverID uuid.UUID, before time.Time) (int64, error)
	DeleteAllOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// ModeratorActivity from postgres repo
type ModeratorActivity struct {
	ActorID        uuid.UUID `db:"actor_id" json:"actor_id"`
	TotalActions   int       `db:"total_actions" json:"total_actions"`
	Bans           int       `db:"bans" json:"bans"`
	Unbans         int       `db:"unbans" json:"unbans"`
	Kicks          int       `db:"kicks" json:"kicks"`
	Mutes          int       `db:"mutes" json:"mutes"`
	Unmutes        int       `db:"unmutes" json:"unmutes"`
	Warns          int       `db:"warns" json:"warns"`
	MessageDeletes int       `db:"message_deletes" json:"message_deletes"`
}

// RepeatOffender from postgres repo
type RepeatOffender struct {
	TargetID            uuid.UUID `db:"target_id" json:"target_id"`
	ModerationCount     int       `db:"moderation_count" json:"moderation_count"`
	DifferentModerators int       `db:"different_moderators" json:"different_moderators"`
	Bans                int       `db:"bans" json:"bans"`
	Warns               int       `db:"warns" json:"warns"`
	Mutes               int       `db:"mutes" json:"mutes"`
}

// DailyTrendPoint from postgres repo
type DailyTrendPoint struct {
	Date           time.Time `db:"date" json:"date"`
	TotalActions   int       `db:"total_actions" json:"total_actions"`
	Bans           int       `db:"bans" json:"bans"`
	Kicks          int       `db:"kicks" json:"kicks"`
	Mutes          int       `db:"mutes" json:"mutes"`
	Warns          int       `db:"warns" json:"warns"`
	MessageDeletes int       `db:"message_deletes" json:"message_deletes"`
}

// ModerationRatios from postgres repo
type ModerationRatios struct {
	Total          int     `db:"total" json:"total"`
	Bans           int     `db:"bans" json:"bans"`
	Unbans         int     `db:"unbans" json:"unbans"`
	Mutes          int     `db:"mutes" json:"mutes"`
	Unmutes        int     `db:"unmutes" json:"unmutes"`
	Warns          int     `db:"warns" json:"warns"`
	Kicks          int     `db:"kicks" json:"kicks"`
	MessageDeletes int     `db:"message_deletes" json:"message_deletes"`
	BulkDeletes    int     `db:"bulk_deletes" json:"bulk_deletes"`
	BanRatio       float64 `json:"ban_ratio"`
	MuteRatio      float64 `json:"mute_ratio"`
	WarnRatio      float64 `json:"warn_ratio"`
	KickRatio      float64 `json:"kick_ratio"`
}

// DashboardSummary from postgres repo
type DashboardSummary struct {
	ServerID         uuid.UUID      `db:"server_id" json:"server_id"`
	TotalActions     int            `db:"total_actions" json:"total_actions"`
	UniqueModerators int            `db:"unique_moderators" json:"unique_moderators"`
	UniqueTargets    int            `db:"unique_targets" json:"unique_targets"`
	TopAction        string         `json:"top_action"`
	TopActionCount   int            `json:"top_action_count"`
	ActionBreakdown  map[string]int `json:"action_breakdown"`
}

// AutoModStats from postgres repo
type AutoModStats struct {
	TotalTriggers   int `json:"total_triggers"`
	Blocks          int `json:"blocks"`
	Warns           int `json:"warns"`
	Timeouts        int `json:"timeouts"`
	Kicks           int `json:"kicks"`
	Bans            int `json:"bans"`
	MessageDeletes  int `json:"message_deletes"`
	MentionSpam     int `json:"mention_spam"`
	WordFilter      int `json:"word_filter"`
	LinkFilter      int `json:"link_filter"`
}

// AuditLog from postgres repo
type AuditLog struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	ServerID       uuid.UUID       `db:"server_id" json:"server_id"`
	ActorID        uuid.UUID       `db:"actor_id" json:"actor_id"`
	ActionType     string          `db:"action_type" json:"action_type"`
	ActionCategory int             `db:"action_category" json:"action_category"`
	TargetID       *uuid.UUID      `db:"target_id" json:"target_id,omitempty"`
	TargetType     *string         `db:"target_type" json:"target_type,omitempty"`
	Reason         *string         `db:"reason" json:"reason,omitempty"`
	Changes        json.RawMessage `db:"changes" json:"changes,omitempty"`
	Metadata       json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	IPAddress      *string         `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// AuditLogService manages audit log entries
type AuditLogService struct {
	repo   AuditLogRepositoryInterface
	mu     sync.RWMutex
	events map[uuid.UUID][]chan<- *models.AuditLogEntry // Server subscribers
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService(repo AuditLogRepositoryInterface) *AuditLogService {
	return &AuditLogService{
		repo:   repo,
		events: make(map[uuid.UUID][]chan<- *models.AuditLogEntry),
	}
}

// Log creates a new audit log entry
func (s *AuditLogService) Log(ctx context.Context, entry *models.AuditLogEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if entry.ActionCategory == 0 {
		entry.ActionCategory = models.GetActionCategory(entry.ActionType)
	}

	// Serialize changes and metadata if they're structs
	if entry.Changes != nil {
		if changes, ok := any(entry.Changes).([]models.Change); ok {
			data, err := json.Marshal(changes)
			if err != nil {
				return fmt.Errorf("failed to marshal changes: %w", err)
			}
			entry.Changes = data
		}
	}

	return s.repo.Create(ctx, entry)
}

// LogBatch creates multiple audit log entries in a single transaction
func (s *AuditLogService) LogBatch(ctx context.Context, entries []models.AuditLogEntry) error {
	for i := range entries {
		if entries[i].ID == uuid.Nil {
			entries[i].ID = uuid.New()
		}
		if entries[i].CreatedAt.IsZero() {
			entries[i].CreatedAt = time.Now()
		}
		if entries[i].ActionCategory == 0 {
			entries[i].ActionCategory = models.GetActionCategory(entries[i].ActionType)
		}
	}

	return s.repo.CreateBatch(ctx, entries)
}

// GetLogs retrieves audit log entries with filtering and pagination
func (s *AuditLogService) GetLogs(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error) {
	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.GetByServer(ctx, serverID, filter)
}

// GetLogByID retrieves a specific audit log entry
func (s *AuditLogService) GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error) {
	return s.repo.GetByID(ctx, serverID, entryID)
}

// GetActionTypes returns all valid audit log action types
func (s *AuditLogService) GetActionTypes() []string {
	return models.GetAllAuditLogActionTypes()
}

// GetCategories returns all audit log categories with metadata
func (s *AuditLogService) GetCategories() []models.AuditLogCategoryInfo {
	return models.GetAuditLogCategories()
}

// GetDashboardSummary returns a quick summary for the moderation dashboard
func (s *AuditLogService) GetDashboardSummary(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardSummary, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	since := time.Now().AddDate(0, 0, -days)

	summary, err := s.repo.GetDashboardSummary(ctx, serverID, since)
	if err != nil {
		return nil, err
	}

	// Convert to model
	result := &models.ModerationDashboardSummary{
		ServerID:         summary.ServerID,
		TotalActions:     summary.TotalActions,
		TopAction:        summary.TopAction,
		TopActionCount:   summary.TopActionCount,
		UniqueModerators: summary.UniqueModerators,
		UniqueTargets:    summary.UniqueTargets,
		ActionBreakdown:  summary.ActionBreakdown,
		TrendDirection:   "stable",
		TrendPercent:     0,
	}

	// Calculate trend direction by comparing to previous period
	prevSince := since.AddDate(0, 0, -days)
	prevSummary, err := s.repo.GetDashboardSummary(ctx, serverID, prevSince)
	if err == nil && prevSummary.TotalActions > 0 {
		change := float64(summary.TotalActions-prevSummary.TotalActions) / float64(prevSummary.TotalActions) * 100
		result.TrendPercent = change
		if change > 5 {
			result.TrendDirection = "up"
		} else if change < -5 {
			result.TrendDirection = "down"
		}
	}

	return result, nil
}

// GetTrendData returns daily moderation trend data
func (s *AuditLogService) GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]models.DailyModerationTrend, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	data, err := s.repo.GetTrendData(ctx, serverID, days)
	if err != nil {
		return nil, err
	}

	result := make([]models.DailyModerationTrend, len(data))
	for i := range data {
		result[i] = models.DailyModerationTrend{
			Date:           data[i].Date,
			TotalActions:   data[i].TotalActions,
			Bans:           data[i].Bans,
			Kicks:          data[i].Kicks,
			Mutes:          data[i].Mutes,
			Warns:          data[i].Warns,
			MessageDeletes: data[i].MessageDeletes,
		}
	}

	return result, nil
}

// GetModeratorActivity returns moderation activity metrics per moderator
func (s *AuditLogService) GetModeratorActivity(ctx context.Context, serverID uuid.UUID, days int) ([]models.ModeratorStats, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	since := time.Now().AddDate(0, 0, -days)

	activity, err := s.repo.GetModeratorActivity(ctx, serverID, since)
	if err != nil {
		return nil, err
	}

	result := make([]models.ModeratorStats, len(activity))
	for i := range activity {
		result[i] = models.ModeratorStats{
			ModeratorID:      activity[i].ActorID,
			TotalActions:     activity[i].TotalActions,
			Bans:             activity[i].Bans,
			Unbans:           activity[i].Unbans,
			Kicks:           activity[i].Kicks,
			Mutes:           activity[i].Mutes,
			Unmutes:         activity[i].Unmutes,
			Warns:           activity[i].Warns,
			MessageDeletes:   activity[i].MessageDeletes,
		}
	}

	return result, nil
}

// GetRepeatOffenders returns users who have been moderated multiple times
func (s *AuditLogService) GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, days, minCount int) ([]models.RepeatOffenderStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	if minCount <= 0 {
		minCount = 2
	}

	since := time.Now().AddDate(0, 0, -days)

	offenders, err := s.repo.GetRepeatOffenders(ctx, serverID, since, minCount)
	if err != nil {
		return nil, err
	}

	result := make([]models.RepeatOffenderStats, len(offenders))
	for i := range offenders {
		result[i] = models.RepeatOffenderStats{
			UserID:              offenders[i].TargetID,
			ModerationCount:     offenders[i].ModerationCount,
			DifferentModerators: offenders[i].DifferentModerators,
			Bans:                offenders[i].Bans,
			Warns:               offenders[i].Warns,
			Mutes:               offenders[i].Mutes,
		}
	}

	return result, nil
}

// GetAutoModStats returns auto-moderation statistics
func (s *AuditLogService) GetAutoModStats(ctx context.Context, serverID uuid.UUID, days int) (*models.AutoModStats, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	since := time.Now().AddDate(0, 0, -days)

	stats, err := s.repo.GetAutoModStats(ctx, serverID, since)
	if err != nil {
		return nil, err
	}

	return &models.AutoModStats{
		TotalTriggers:  stats.TotalTriggers,
		Blocks:        stats.Blocks,
		Warns:         stats.Warns,
		Timeouts:      stats.Timeouts,
		Kicks:         stats.Kicks,
		Bans:          stats.Bans,
		MessageDeletes: stats.MessageDeletes,
		MentionSpam:   stats.MentionSpam,
		WordFilter:    stats.WordFilter,
		LinkFilter:    stats.LinkFilter,
	}, nil
}

// ExportLogs exports audit logs in the specified format
func (s *AuditLogService) ExportLogs(ctx context.Context, serverID uuid.UUID, format string, filter AuditLogFilter) ([]byte, string, error) {
	switch format {
	case "csv":
		sb, err := s.repo.ExportToCSV(ctx, serverID, filter)
		if err != nil {
			return nil, "", err
		}
		return []byte(sb.String()), "text/csv", nil
	case "json":
		fallthrough
	default:
		logs, _, err := s.GetLogs(ctx, serverID, filter)
		if err != nil {
			return nil, "", err
		}
		data, err := json.Marshal(logs)
		if err != nil {
			return nil, "", err
		}
		return data, "application/json", nil
	}
}

// ExportForGDPR exports all audit logs related to a user for GDPR compliance
func (s *AuditLogService) ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]models.AuditLogEntry, error) {
	logs, err := s.repo.ExportForGDPR(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert from repo model to service model
	result := make([]models.AuditLogEntry, len(logs))
	for i := range logs {
		result[i] = models.AuditLogEntry{
			ID:             logs[i].ID,
			ServerID:       logs[i].ServerID,
			ActorID:        logs[i].ActorID,
			ActionType:     logs[i].ActionType,
			ActionCategory: logs[i].ActionCategory,
			TargetID:       logs[i].TargetID,
			CreatedAt:      logs[i].CreatedAt,
		}
		// Copy pointer fields
		if logs[i].TargetType != nil {
			result[i].TargetType = *logs[i].TargetType
		}
		if logs[i].Reason != nil {
			result[i].Reason = *logs[i].Reason
		}
		result[i].Changes = logs[i].Changes
		result[i].Metadata = logs[i].Metadata
		if logs[i].IPAddress != nil {
			result[i].IPAddress = logs[i].IPAddress
		}
	}

	return result, nil
}

// CleanupOldLogs removes audit logs older than the retention period
func (s *AuditLogService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // Default 90 days retention
	}
	before := time.Now().AddDate(0, 0, -retentionDays)
	return s.repo.DeleteAllOlderThan(ctx, before)
}
