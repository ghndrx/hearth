package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAnalyticsRepoMock(t *testing.T) (*AnalyticsRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewAnalyticsRepository(sqlxDB)
	return repo, mock
}

// --- GetMemberGrowthHistory Tests ---

func TestAnalyticsRepo_GetMemberGrowthHistory_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	now := time.Now()
	mock.ExpectQuery("WITH date_series AS").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"date", "count", "change"}).
			AddRow(now.AddDate(0, 0, -1).Truncate(24*time.Hour), 100, 5).
			AddRow(now.Truncate(24*time.Hour), 105, 5))

	result, err := repo.GetMemberGrowthHistory(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 100, result[0].Count)
	assert.Equal(t, 5, result[0].Change)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMemberGrowthHistory_EmptyResult(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	mock.ExpectQuery("WITH date_series AS").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"date", "count", "change"}))

	result, err := repo.GetMemberGrowthHistory(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMemberGrowthHistory_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	mock.ExpectQuery("WITH date_series AS").
		WithArgs(serverID, days).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetMemberGrowthHistory(ctx, serverID, days)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetMessageActivityStats Tests ---

func TestAnalyticsRepo_GetMessageActivityStats_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"day_of_week", "hour", "message_count", "unique_users"}).
			AddRow(1, 14, 50, 10).
			AddRow(2, 15, 75, 15))

	result, err := repo.GetMessageActivityStats(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].DayOfWeek)
	assert.Equal(t, 14, result[0].Hour)
	assert.Equal(t, 50, result[0].MessageCount)
	assert.Equal(t, 10, result[0].UniqueUsers)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMessageActivityStats_EmptyResult(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"day_of_week", "hour", "message_count", "unique_users"}))

	result, err := repo.GetMessageActivityStats(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMessageActivityStats_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetMessageActivityStats(ctx, serverID, days)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetTopChannels Tests ---

func TestAnalyticsRepo_GetTopChannels_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7
	limit := 10

	lastActivity := time.Now()
	mock.ExpectQuery("WITH channel_stats AS").
		WithArgs(serverID, days, limit).
		WillReturnRows(sqlmock.NewRows([]string{
			"channel_id", "channel_name", "channel_type", "message_count", "unique_authors", "last_activity",
		}).
			AddRow(uuid.New(), "general", "text", 500, 45, lastActivity).
			AddRow(uuid.New(), "random", "text", 300, 25, lastActivity))

	result, err := repo.GetTopChannels(ctx, serverID, days, limit)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "general", result[0].ChannelName)
	assert.Equal(t, "text", result[0].ChannelType)
	assert.Equal(t, 500, result[0].MessageCount)
	assert.Equal(t, 45, result[0].UniqueAuthors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetTopChannels_EmptyResult(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30
	limit := 5

	mock.ExpectQuery("WITH channel_stats AS").
		WithArgs(serverID, days, limit).
		WillReturnRows(sqlmock.NewRows([]string{
			"channel_id", "channel_name", "channel_type", "message_count", "unique_authors", "last_activity",
		}))

	result, err := repo.GetTopChannels(ctx, serverID, days, limit)
	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetTopChannels_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7
	limit := 10

	mock.ExpectQuery("WITH channel_stats AS").
		WithArgs(serverID, days, limit).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetTopChannels(ctx, serverID, days, limit)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetRetentionMetrics Tests ---

func TestAnalyticsRepo_GetRetentionMetrics_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	now := time.Now()
	// Mock DAU query
	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"date", "active_users"}).
			AddRow(now.AddDate(0, 0, -2).Truncate(24*time.Hour), 50).
			AddRow(now.AddDate(0, 0, -1).Truncate(24*time.Hour), 60))

	// Mock MAU query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{"mau"}).AddRow(150))

	// Mock total members query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	result, err := repo.GetRetentionMetrics(ctx, serverID, days)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.DailyActiveUsers, 2)
	assert.Equal(t, 150, result.MAU)
	assert.Equal(t, 500, result.TotalMembers)
	assert.Equal(t, 55.0, result.AverageDAU) // (50+60)/2
	assert.InDelta(t, 0.367, result.Stickiness, 0.01) // 55/150
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetRetentionMetrics_ZeroMAU(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	// Mock DAU query
	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"date", "active_users"}))

	// Mock MAU query - returns 0
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{"mau"}).AddRow(0))

	// Mock total members query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, err := repo.GetRetentionMetrics(ctx, serverID, days)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.MAU)
	assert.Equal(t, 0.0, result.Stickiness) // stickiness should be 0 when MAU is 0
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetRetentionMetrics_DAUQueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetRetentionMetrics(ctx, serverID, days)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetRetentionMetrics_MAUQueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	now := time.Now()
	// Mock DAU query - succeeds
	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"date", "active_users"}).
			AddRow(now.AddDate(0, 0, -1).Truncate(24*time.Hour), 50))

	// Mock MAU query - fails
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetRetentionMetrics(ctx, serverID, days)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetServerAnalyticsSummary Tests ---

func TestAnalyticsRepo_GetServerAnalyticsSummary_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	// Mock main summary query
	mock.ExpectQuery("WITH recent_stats AS").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{
			"messages_today", "active_users_today", "messages_week", "active_users_week",
			"total_members", "new_members_week", "member_change_week",
		}).AddRow(150, 25, 1200, 75, 500, 12, 10))

	// Mock previous week messages query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))

	result, err := repo.GetServerAnalyticsSummary(ctx, serverID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 150, result.MessagesToday)
	assert.Equal(t, 25, result.ActiveUsersToday)
	assert.Equal(t, 1200, result.MessagesWeek)
	assert.Equal(t, 75, result.ActiveUsersWeek)
	assert.Equal(t, 500, result.TotalMembers)
	assert.Equal(t, 12, result.NewMembersWeek)
	assert.Equal(t, 10, result.MemberChangeWeek)
	// MessageChangePercent: (1200-1000)/1000 * 100 = 20
	assert.Equal(t, 20.0, result.MessageChangePercent)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetServerAnalyticsSummary_NoPrevWeekMessages(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	// Mock main summary query
	mock.ExpectQuery("WITH recent_stats AS").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{
			"messages_today", "active_users_today", "messages_week", "active_users_week",
			"total_members", "new_members_week", "member_change_week",
		}).AddRow(150, 25, 0, 75, 500, 12, 10))

	result, err := repo.GetServerAnalyticsSummary(ctx, serverID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.MessageChangePercent) // No prev week messages, stays 0
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetServerAnalyticsSummary_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	mock.ExpectQuery("WITH recent_stats AS").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetServerAnalyticsSummary(ctx, serverID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetServerAnalyticsSummary_PrevWeekQueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	// Mock main summary query
	mock.ExpectQuery("WITH recent_stats AS").
		WithArgs(serverID).
		WillReturnRows(sqlmock.NewRows([]string{
			"messages_today", "active_users_today", "messages_week", "active_users_week",
			"total_members", "new_members_week", "member_change_week",
		}).AddRow(150, 25, 1200, 75, 500, 12, 10))

	// Mock previous week messages query - error
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	// Should still succeed - prev week error is ignored
	result, err := repo.GetServerAnalyticsSummary(ctx, serverID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.MessageChangePercent) // Error in prev week calc, stays 0
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- TakeMemberSnapshot Tests ---

func TestAnalyticsRepo_TakeMemberSnapshot_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT take_member_snapshots").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.TakeMemberSnapshot(ctx)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_TakeMemberSnapshot_Error(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("SELECT take_member_snapshots").
		WillReturnError(sql.ErrConnDone)

	err := repo.TakeMemberSnapshot(ctx)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- TakeMemberSnapshotForServer Tests ---

func TestAnalyticsRepo_TakeMemberSnapshotForServer_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	mock.ExpectExec("INSERT INTO server_member_snapshots").
		WithArgs(serverID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.TakeMemberSnapshotForServer(ctx, serverID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_TakeMemberSnapshotForServer_Error(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	mock.ExpectExec("INSERT INTO server_member_snapshots").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	err := repo.TakeMemberSnapshotForServer(ctx, serverID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetPeakActivityHours Tests ---

func TestAnalyticsRepo_GetPeakActivityHours_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"hour", "total_messages"}).
			AddRow(14, 500).
			AddRow(15, 450).
			AddRow(19, 400))

	result, err := repo.GetPeakActivityHours(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 14, result[0].Hour)
	assert.Equal(t, 500, result[0].MessageCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetPeakActivityHours_EmptyResult(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnRows(sqlmock.NewRows([]string{"hour", "total_messages"}))

	result, err := repo.GetPeakActivityHours(ctx, serverID, days)
	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetPeakActivityHours_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetPeakActivityHours(ctx, serverID, days)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetMostActiveUsers Tests ---

func TestAnalyticsRepo_GetMostActiveUsers_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7
	limit := 10

	displayName := "Test User"
	avatarURL := "https://example.com/avatar.png"
	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days, limit).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "username", "display_name", "avatar_url", "message_count", "days_active",
		}).
			AddRow(uuid.New(), "user1", displayName, avatarURL, 500, 7).
			AddRow(uuid.New(), "user2", nil, nil, 300, 5))

	result, err := repo.GetMostActiveUsers(ctx, serverID, days, limit)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "user1", result[0].Username)
	assert.Equal(t, &displayName, result[0].DisplayName)
	assert.Equal(t, &avatarURL, result[0].AvatarURL)
	assert.Equal(t, 500, result[0].MessageCount)
	assert.Equal(t, 7, result[0].DaysActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMostActiveUsers_EmptyResult(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 30
	limit := 5

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days, limit).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "username", "display_name", "avatar_url", "message_count", "days_active",
		}))

	result, err := repo.GetMostActiveUsers(ctx, serverID, days, limit)
	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_GetMostActiveUsers_QueryError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	days := 7
	limit := 10

	mock.ExpectQuery("SELECT").
		WithArgs(serverID, days, limit).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetMostActiveUsers(ctx, serverID, days, limit)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- CleanupOldAnalyticsData Tests ---

func TestAnalyticsRepo_CleanupOldAnalyticsData_Success(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	retentionDays := 30

	// Mock hourly activity cleanup
	mock.ExpectExec("DELETE FROM server_activity_hourly").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 100))

	// Mock daily active users cleanup
	mock.ExpectExec("DELETE FROM server_daily_active_users").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 50))

	// Mock member snapshots cleanup
	mock.ExpectExec("DELETE FROM server_member_snapshots").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 10))

	err := repo.CleanupOldAnalyticsData(ctx, retentionDays)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_CleanupOldAnalyticsData_HourlyActivityError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	retentionDays := 30

	mock.ExpectExec("DELETE FROM server_activity_hourly").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	err := repo.CleanupOldAnalyticsData(ctx, retentionDays)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAnalyticsRepo_CleanupOldAnalyticsData_DailyActiveUsersError(t *testing.T) {
	repo, mock := setupAnalyticsRepoMock(t)
	ctx := context.Background()
	retentionDays := 30

	// Mock hourly activity cleanup - succeeds
	mock.ExpectExec("DELETE FROM server_activity_hourly").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 100))

	// Mock daily active users cleanup - fails
	mock.ExpectExec("DELETE FROM server_daily_active_users").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	err := repo.CleanupOldAnalyticsData(ctx, retentionDays)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
