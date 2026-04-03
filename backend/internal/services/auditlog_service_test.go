package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

// MockAuditLogRepository implements AuditLogRepositoryInterface for testing
type MockAuditLogRepository struct {
	logs     []models.AuditLogEntry
	err      error
	id       uuid.UUID
}

func NewMockAuditLogRepository() *MockAuditLogRepository {
	return &MockAuditLogRepository{
		logs: make([]models.AuditLogEntry, 0),
	}
}

func (m *MockAuditLogRepository) Create(ctx context.Context, entry *models.AuditLogEntry) error {
	if m.err != nil {
		return m.err
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	m.logs = append(m.logs, *entry)
	return nil
}

func (m *MockAuditLogRepository) CreateBatch(ctx context.Context, entries []models.AuditLogEntry) error {
	if m.err != nil {
		return m.err
	}
	for i := range entries {
		if entries[i].ID == uuid.Nil {
			entries[i].ID = uuid.New()
		}
		if entries[i].CreatedAt.IsZero() {
			entries[i].CreatedAt = time.Now()
		}
	}
	m.logs = append(m.logs, entries...)
	return nil
}

func (m *MockAuditLogRepository) GetByServer(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}

	var result []models.AuditLogEntry
	for _, log := range m.logs {
		if log.ServerID != serverID {
			continue
		}
		if filter.ActionType != "" && log.ActionType != filter.ActionType {
			continue
		}
		if filter.ActionCategory > 0 && log.ActionCategory != filter.ActionCategory {
			continue
		}
		if filter.ActorID != nil && log.ActorID != *filter.ActorID {
			continue
		}
		if filter.TargetID != nil && (log.TargetID == nil || *log.TargetID != *filter.TargetID) {
			continue
		}
		if filter.Before != nil && log.CreatedAt.After(*filter.Before) {
			continue
		}
		if filter.After != nil && log.CreatedAt.Before(*filter.After) {
			continue
		}
		result = append(result, log)
	}

	// Apply pagination
	total := len(result)
	start := filter.Offset
	end := filter.Offset + filter.Limit
	if start > len(result) {
		start = len(result)
	}
	if end > len(result) {
		end = len(result)
	}
	result = result[start:end]

	return result, total, nil
}

func (m *MockAuditLogRepository) GetByID(ctx context.Context, serverID, logID uuid.UUID) (*models.AuditLogEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	for i := range m.logs {
		if m.logs[i].ID == logID && m.logs[i].ServerID == serverID {
			return &m.logs[i], nil
		}
	}
	return nil, ErrAuditLogNotFound
}

func (m *MockAuditLogRepository) GetActionCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[string]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	counts := make(map[string]int)
	for _, log := range m.logs {
		if log.ServerID == serverID && log.CreatedAt.After(since) {
			counts[log.ActionType]++
		}
	}
	return counts, nil
}

func (m *MockAuditLogRepository) GetActionCategoryCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[int]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	counts := make(map[int]int)
	for _, log := range m.logs {
		if log.ServerID == serverID && log.CreatedAt.After(since) && log.ActionCategory > 0 {
			counts[log.ActionCategory]++
		}
	}
	return counts, nil
}

func (m *MockAuditLogRepository) GetModeratorActivity(ctx context.Context, serverID uuid.UUID, since time.Time) ([]ModeratorActivity, error) {
	if m.err != nil {
		return nil, m.err
	}
	activityMap := make(map[uuid.UUID]*ModeratorActivity)
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) {
			continue
		}
		if _, ok := activityMap[log.ActorID]; !ok {
			activityMap[log.ActorID] = &ModeratorActivity{ActorID: log.ActorID}
		}
		activityMap[log.ActorID].TotalActions++
		switch log.ActionType {
		case models.AuditLogMemberBan:
			activityMap[log.ActorID].Bans++
		case models.AuditLogMemberUnban:
			activityMap[log.ActorID].Unbans++
		case models.AuditLogMemberKick:
			activityMap[log.ActorID].Kicks++
		}
	}
	var result []ModeratorActivity
	for _, v := range activityMap {
		result = append(result, *v)
	}
	return result, nil
}

func (m *MockAuditLogRepository) GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, since time.Time, minCount int) ([]RepeatOffender, error) {
	if m.err != nil {
		return nil, m.err
	}
	offenderMap := make(map[uuid.UUID]*RepeatOffender)
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) || log.TargetID == nil {
			continue
		}
		if _, ok := offenderMap[*log.TargetID]; !ok {
			offenderMap[*log.TargetID] = &RepeatOffender{TargetID: *log.TargetID}
		}
		offenderMap[*log.TargetID].ModerationCount++
	}
	var result []RepeatOffender
	for _, v := range offenderMap {
		if v.ModerationCount >= minCount {
			result = append(result, *v)
		}
	}
	return result, nil
}

func (m *MockAuditLogRepository) GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]DailyTrendPoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	since := time.Now().AddDate(0, 0, -days)
	trendMap := make(map[string]*DailyTrendPoint)
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) {
			continue
		}
		dateKey := log.CreatedAt.Format("2006-01-02")
		if _, ok := trendMap[dateKey]; !ok {
			trendMap[dateKey] = &DailyTrendPoint{Date: log.CreatedAt}
		}
		trendMap[dateKey].TotalActions++
		switch log.ActionType {
		case models.AuditLogMemberBan:
			trendMap[dateKey].Bans++
		case models.AuditLogMemberKick:
			trendMap[dateKey].Kicks++
		}
	}
	var result []DailyTrendPoint
	for _, v := range trendMap {
		result = append(result, *v)
	}
	return result, nil
}

func (m *MockAuditLogRepository) GetModerationRatios(ctx context.Context, serverID uuid.UUID, since time.Time) (*ModerationRatios, error) {
	if m.err != nil {
		return nil, m.err
	}
	var ratios ModerationRatios
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) {
			continue
		}
		ratios.Total++
		switch log.ActionType {
		case models.AuditLogMemberBan:
			ratios.Bans++
		case models.AuditLogMemberKick:
			ratios.Kicks++
		}
	}
	return &ratios, nil
}

func (m *MockAuditLogRepository) GetDashboardSummary(ctx context.Context, serverID uuid.UUID, since time.Time) (*DashboardSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	var summary DashboardSummary
	summary.ServerID = serverID
	moderators := make(map[uuid.UUID]bool)
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) {
			continue
		}
		summary.TotalActions++
		moderators[log.ActorID] = true
		if log.TargetID != nil {
			summary.UniqueTargets++
		}
	}
	summary.UniqueModerators = len(moderators)
	return &summary, nil
}

func (m *MockAuditLogRepository) GetAutoModStats(ctx context.Context, serverID uuid.UUID, since time.Time) (*AutoModStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	var stats AutoModStats
	for _, log := range m.logs {
		if log.ServerID != serverID || log.CreatedAt.Before(since) {
			continue
		}
		if strings.HasPrefix(log.ActionType, "AUTOMOD_") {
			stats.TotalTriggers++
			switch log.ActionType {
			case models.AuditLogAutoModBlock:
				stats.Blocks++
			case models.AuditLogAutoModWarn:
				stats.Warns++
			case models.AuditLogAutoModTimeout:
				stats.Timeouts++
			case models.AuditLogAutoModKick:
				stats.Kicks++
			case models.AuditLogAutoModBan:
				stats.Bans++
			}
		}
	}
	return &stats, nil
}

func (m *MockAuditLogRepository) ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]AuditLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []AuditLog
	for _, log := range m.logs {
		if log.ActorID == userID || (log.TargetID != nil && *log.TargetID == userID) {
			result = append(result, AuditLog{
				ID:             log.ID,
				ServerID:       log.ServerID,
				ActorID:        log.ActorID,
				ActionType:     log.ActionType,
				ActionCategory: log.ActionCategory,
				TargetID:       log.TargetID,
				CreatedAt:      log.CreatedAt,
			})
		}
	}
	return result, nil
}

func (m *MockAuditLogRepository) ExportToCSV(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) (*strings.Builder, error) {
	if m.err != nil {
		return nil, m.err
	}
	logs, _, _ := m.GetByServer(ctx, serverID, filter)
	var sb strings.Builder
	for _, log := range logs {
		sb.WriteString(log.ID.String())
		sb.WriteString(",")
		sb.WriteString(log.ActionType)
		sb.WriteString("\n")
	}
	return &sb, nil
}

func (m *MockAuditLogRepository) DeleteOlderThan(ctx context.Context, serverID uuid.UUID, before time.Time) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	var newLogs []models.AuditLogEntry
	var deleted int64
	for _, log := range m.logs {
		if log.ServerID == serverID && log.CreatedAt.Before(before) {
			deleted++
		} else {
			newLogs = append(newLogs, log)
		}
	}
	m.logs = newLogs
	return deleted, nil
}

func (m *MockAuditLogRepository) DeleteAllOlderThan(ctx context.Context, before time.Time) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	var newLogs []models.AuditLogEntry
	var deleted int64
	for _, log := range m.logs {
		if log.CreatedAt.Before(before) {
			deleted++
		} else {
			newLogs = append(newLogs, log)
		}
	}
	m.logs = newLogs
	return deleted, nil
}

func TestAuditLogService_Log(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()

	changes, _ := json.Marshal([]models.Change{
		{Key: "name", OldValue: "old", NewValue: "new"},
	})

	entry := &models.AuditLogEntry{
		ServerID:       serverID,
		ActorID:        userID,
		ActionType:     models.AuditLogMemberBan,
		TargetID:       &targetID,
		Changes:       changes,
		Reason:         "spam",
	}

	err := svc.Log(ctx, entry)
	require.NoError(t, err)

	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.AuditLogMemberBan, logs[0].ActionType)
	assert.Equal(t, userID, logs[0].ActorID)
	assert.Equal(t, &targetID, logs[0].TargetID)
	assert.Equal(t, "spam", logs[0].Reason)
}

func TestAuditLogService_GetLogs_FilterByActionType(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create entries with different action types
	for _, actionType := range []string{models.AuditLogMemberBan, models.AuditLogChannelCreate, models.AuditLogMemberBan} {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: actionType,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Filter by action type
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		ActionType: models.AuditLogMemberBan,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, models.AuditLogMemberBan, log.ActionType)
	}
}

func TestAuditLogService_GetLogs_FilterByActorID(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create entries from different users
	for i, userID := range []uuid.UUID{userID1, userID2, userID1} {
		actionType := models.AuditLogMemberBan
		if i == 1 {
			actionType = models.AuditLogChannelCreate
		}
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: actionType,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Filter by user ID
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		ActorID: &userID1,
		Limit:   10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, userID1, log.ActorID)
	}
}

func TestAuditLogService_GetLogs_FilterByTargetID(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	targetID1 := uuid.New()
	targetID2 := uuid.New()

	// Create entries with different targets
	targets := []uuid.UUID{targetID1, targetID2, targetID1, uuid.Nil}
	actionTypes := []string{models.AuditLogMemberBan, models.AuditLogMemberBan, models.AuditLogMemberBan, models.AuditLogChannelCreate}
	for i := range targets {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: actionTypes[i],
			TargetID:   &targets[i],
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Filter by target ID
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		TargetID: &targetID1,
		Limit:    10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, &targetID1, log.TargetID)
	}
}

func TestAuditLogService_GetLogs_FilterByDateRange(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create entries
	actionTypes := []string{models.AuditLogMemberBan, models.AuditLogChannelCreate, models.AuditLogRoleCreate, models.AuditLogMemberKick}
	for _, actionType := range actionTypes {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: actionType,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	afterTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	entry := &models.AuditLogEntry{
		ServerID:   serverID,
		ActorID:    userID,
		ActionType: models.AuditLogMemberKick,
	}
	err := svc.Log(ctx, entry)
	require.NoError(t, err)

	beforeTime := time.Now()

	// Filter by date range
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		After:  &afterTime,
		Before: &beforeTime,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
}

func TestAuditLogService_GetLogs_Pagination(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 15 entries
	for i := 0; i < 15; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Get first page
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, 15, total)

	// Get last page
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, 15, total)

	// Get beyond entries
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 15,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 0)
	assert.Equal(t, 15, total)
}

func TestAuditLogService_GetLogs_DefaultLimit(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 60 entries
	for i := 0; i < 60; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Get with default limit
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{})
	require.NoError(t, err)
	assert.Len(t, logs, 50) // Default limit is 50
	assert.Equal(t, 60, total)
}

func TestAuditLogService_GetLogs_MaxLimit(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 150 entries
	for i := 0; i < 150; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Get with limit > 100
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit: 200,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 100) // Max limit is 100
	assert.Equal(t, 150, total)
}

func TestAuditLogService_GetLogByID(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create an entry
	entry := &models.AuditLogEntry{
		ServerID:   serverID,
		ActorID:    userID,
		ActionType: models.AuditLogMemberBan,
		Reason:     "spam",
	}
	err := svc.Log(ctx, entry)
	require.NoError(t, err)

	// Get all logs to find the entry ID
	logs, _, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, logs, 1)

	// Get by ID
	retrieved, err := svc.GetLogByID(ctx, serverID, logs[0].ID)
	require.NoError(t, err)
	assert.Equal(t, logs[0].ID, retrieved.ID)
	assert.Equal(t, models.AuditLogMemberBan, retrieved.ActionType)
	assert.Equal(t, "spam", retrieved.Reason)
}

func TestAuditLogService_GetLogByID_NotFound(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	randomID := uuid.New()

	// Try to get non-existent entry
	entry, err := svc.GetLogByID(ctx, serverID, randomID)
	assert.ErrorIs(t, err, ErrAuditLogNotFound)
	assert.Nil(t, entry)
}

func TestAuditLogService_GetActionTypes(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	types := svc.GetActionTypes()

	// Verify all expected action types are present
	expectedTypes := []string{
		models.AuditLogServerUpdate,
		models.AuditLogChannelCreate,
		models.AuditLogMemberKick,
		models.AuditLogMemberBan,
		models.AuditLogMemberUnban,
		models.AuditLogRoleCreate,
		models.AuditLogWebhookCreate,
		models.AuditLogEmojiCreate,
		models.AuditLogMessageDelete,
		models.AuditLogMessageBulkDelete,
	}

	for _, expected := range expectedTypes {
		assert.Contains(t, types, expected)
	}
}

func TestAuditLogService_GetCategories(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	categories := svc.GetCategories()

	// Verify categories are returned
	assert.NotEmpty(t, categories)

	// Check that expected categories exist
	categoryMap := make(map[int]bool)
	for _, c := range categories {
		categoryMap[c.Category] = true
	}

	assert.True(t, categoryMap[10]) // Member
	assert.True(t, categoryMap[20]) // Channel
	assert.True(t, categoryMap[30]) // Server
	assert.True(t, categoryMap[40]) // Message
	assert.True(t, categoryMap[80]) // AutoMod
}

func TestAuditLogService_GetDashboardSummary(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 5; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	summary, err := svc.GetDashboardSummary(ctx, serverID, 7)
	require.NoError(t, err)
	assert.Equal(t, 5, summary.TotalActions)
}

func TestAuditLogService_GetTrendData(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	trend, err := svc.GetTrendData(ctx, serverID, 7)
	require.NoError(t, err)
	assert.NotEmpty(t, trend)
}

func TestAuditLogService_GetModeratorActivity(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	activity, err := svc.GetModeratorActivity(ctx, serverID, 7)
	require.NoError(t, err)
	assert.Len(t, activity, 1)
	assert.Equal(t, 3, activity[0].TotalActions)
}

func TestAuditLogService_GetRepeatOffenders(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()

	// Create multiple entries for the same target
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
			TargetID:   &targetID,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	offenders, err := svc.GetRepeatOffenders(ctx, serverID, 30, 2)
	require.NoError(t, err)
	assert.Len(t, offenders, 1)
	assert.Equal(t, 3, offenders[0].ModerationCount)
}

func TestAuditLogService_GetAutoModStats(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogAutoModBlock,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	stats, err := svc.GetAutoModStats(ctx, serverID, 7)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalTriggers)
	assert.Equal(t, 3, stats.Blocks)
}

func TestAuditLogService_ExportLogs_JSON(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	data, contentType, err := svc.ExportLogs(ctx, serverID, "json", AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, "application/json", contentType)
	assert.NotEmpty(t, data)
}

func TestAuditLogService_ExportLogs_CSV(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &models.AuditLogEntry{
			ServerID:   serverID,
			ActorID:    userID,
			ActionType: models.AuditLogMemberBan,
		}
		err := svc.Log(ctx, entry)
		require.NoError(t, err)
	}

	data, contentType, err := svc.ExportLogs(ctx, serverID, "csv", AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, "text/csv", contentType)
	assert.NotEmpty(t, data)
}

func TestAuditLogService_CleanupOldLogs(t *testing.T) {
	repo := NewMockAuditLogRepository()
	svc := NewAuditLogService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create an entry
	entry := &models.AuditLogEntry{
		ServerID:   serverID,
		ActorID:    userID,
		ActionType: models.AuditLogMemberBan,
	}
	err := svc.Log(ctx, entry)
	require.NoError(t, err)

	// Cleanup with 90 days retention
	deleted, err := svc.CleanupOldLogs(ctx, 90)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted) // No old entries

	// Verify entry still exists
	logs, _, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
}
