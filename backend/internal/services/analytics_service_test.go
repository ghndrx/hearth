package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockAnalyticsRepository is a mock implementation of AnalyticsRepository
type MockAnalyticsRepository struct {
	mock.Mock
}

func (m *MockAnalyticsRepository) GetMemberGrowthHistory(ctx context.Context, serverID uuid.UUID, days int) ([]*models.MemberGrowthPoint, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.MemberGrowthPoint), args.Error(1)
}

func (m *MockAnalyticsRepository) GetMessageActivityStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ActivityHourStat, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActivityHourStat), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTopChannels(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.TopChannelStat, error) {
	args := m.Called(ctx, serverID, days, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TopChannelStat), args.Error(1)
}

func (m *MockAnalyticsRepository) GetRetentionMetrics(ctx context.Context, serverID uuid.UUID, days int) (*models.RetentionMetrics, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RetentionMetrics), args.Error(1)
}

func (m *MockAnalyticsRepository) GetServerAnalyticsSummary(ctx context.Context, serverID uuid.UUID) (*models.AnalyticsSummary, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnalyticsSummary), args.Error(1)
}

func (m *MockAnalyticsRepository) GetPeakActivityHours(ctx context.Context, serverID uuid.UUID, days int) ([]*models.PeakHour, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PeakHour), args.Error(1)
}

func (m *MockAnalyticsRepository) GetMostActiveUsers(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.ActiveUserStat, error) {
	args := m.Called(ctx, serverID, days, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveUserStat), args.Error(1)
}

func (m *MockAnalyticsRepository) TakeMemberSnapshot(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) TakeMemberSnapshotForServer(ctx context.Context, serverID uuid.UUID) error {
	args := m.Called(ctx, serverID)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) CleanupOldAnalyticsData(ctx context.Context, retentionDays int) error {
	args := m.Called(ctx, retentionDays)
	return args.Error(0)
}

// MockServerRepoForAnalytics is a minimal mock for server repository
type MockServerRepoForAnalytics struct {
	mock.Mock
}

func (m *MockServerRepoForAnalytics) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

// Implement other required methods as no-ops for testing
func (m *MockServerRepoForAnalytics) Create(ctx context.Context, server *models.Server) error {
	return nil
}
func (m *MockServerRepoForAnalytics) Update(ctx context.Context, server *models.Server) error {
	return nil
}
func (m *MockServerRepoForAnalytics) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockServerRepoForAnalytics) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetAllMembers(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetMembersWithRolePaginated(ctx context.Context, serverID, roleID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetAllMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) AddMember(ctx context.Context, member *models.Member) error {
	return nil
}
func (m *MockServerRepoForAnalytics) UpdateMember(ctx context.Context, member *models.Member) error {
	return nil
}
func (m *MockServerRepoForAnalytics) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *MockServerRepoForAnalytics) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *MockServerRepoForAnalytics) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) AddBan(ctx context.Context, ban *models.Ban) error { return nil }
func (m *MockServerRepoForAnalytics) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) CreateInvite(ctx context.Context, invite *models.Invite) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) DeleteInvite(ctx context.Context, code string) error { return nil }
func (m *MockServerRepoForAnalytics) IncrementInviteUses(ctx context.Context, code string) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *MockServerRepoForAnalytics) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *MockServerRepoForAnalytics) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

func TestAnalyticsService_GetMemberGrowth(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	// Set up server mock
	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	// Set up growth data mock
	now := time.Now()
	growthData := []*models.MemberGrowthPoint{
		{Date: now.AddDate(0, 0, -2), Count: 100, Change: 0},
		{Date: now.AddDate(0, 0, -1), Count: 105, Change: 5},
		{Date: now, Count: 110, Change: 5},
	}
	mockRepo.On("GetMemberGrowthHistory", ctx, serverID, 7).Return(growthData, nil)

	// Test as owner
	result, err := service.GetMemberGrowth(ctx, serverID, ownerID, 7)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, serverID.String(), result.ServerID)
	assert.Equal(t, "7d", result.Period)
	assert.Len(t, result.Data, 3)

	mockRepo.AssertExpectations(t)
	mockServerRepo.AssertExpectations(t)
}

func TestAnalyticsService_GetMemberGrowth_ServerNotFound(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	mockServerRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	result, err := service.GetMemberGrowth(ctx, serverID, requesterID, 7)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrServerNotFound, err)
}

func TestAnalyticsService_GetMessageActivity(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	activityData := []*models.ActivityHourStat{
		{DayOfWeek: 1, Hour: 14, MessageCount: 50, UniqueUsers: 10},
		{DayOfWeek: 1, Hour: 15, MessageCount: 75, UniqueUsers: 15},
	}
	mockRepo.On("GetMessageActivityStats", ctx, serverID, 7).Return(activityData, nil)

	peakHours := []*models.PeakHour{
		{Hour: 15, MessageCount: 75},
		{Hour: 14, MessageCount: 50},
	}
	mockRepo.On("GetPeakActivityHours", ctx, serverID, 7).Return(peakHours, nil)

	result, err := service.GetMessageActivity(ctx, serverID, ownerID, 7)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 2)
	assert.Len(t, result.PeakHours, 2)
	assert.Equal(t, 125, result.TotalStats.TotalMessages)
}

func TestAnalyticsService_GetTopChannels(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	channelData := []*models.TopChannelStat{
		{ChannelID: uuid.New(), ChannelName: "general", ChannelType: "text", MessageCount: 500},
		{ChannelID: uuid.New(), ChannelName: "random", ChannelType: "text", MessageCount: 200},
	}
	mockRepo.On("GetTopChannels", ctx, serverID, 7, 10).Return(channelData, nil)

	result, err := service.GetTopChannels(ctx, serverID, ownerID, 7, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, "general", result.Data[0].ChannelName)
}

func TestAnalyticsService_GetRetention(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	retentionData := &models.RetentionMetrics{
		MAU:          150,
		TotalMembers: 500,
		AverageDAU:   48.5,
		Stickiness:   0.323,
		DailyActiveUsers: []*models.DailyActiveUserPoint{
			{Date: time.Now(), Count: 50},
		},
	}
	mockRepo.On("GetRetentionMetrics", ctx, serverID, 30).Return(retentionData, nil)

	result, err := service.GetRetention(ctx, serverID, ownerID, 30)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 150, result.Data.MAU)
	assert.Equal(t, 500, result.Data.TotalMembers)
}

func TestAnalyticsService_GetSummary(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	summaryData := &models.AnalyticsSummary{
		MessagesToday:    150,
		ActiveUsersToday: 25,
		MessagesWeek:     1200,
		ActiveUsersWeek:  75,
		TotalMembers:     500,
		NewMembersWeek:   12,
	}
	mockRepo.On("GetServerAnalyticsSummary", ctx, serverID).Return(summaryData, nil)

	result, err := service.GetSummary(ctx, serverID, ownerID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, serverID, result.ServerID)
	assert.Equal(t, 150, result.Summary.MessagesToday)
	assert.Equal(t, 500, result.Summary.TotalMembers)
}

func TestAnalyticsService_NormalizeDays(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	mockRepo := new(MockAnalyticsRepository)
	mockServerRepo := new(MockServerRepoForAnalytics)

	service := NewAnalyticsService(mockRepo, mockServerRepo, nil, nil)

	server := &models.Server{
		ID:      serverID,
		Name:    "Test Server",
		OwnerID: ownerID,
	}
	mockServerRepo.On("GetByID", ctx, serverID).Return(server, nil)

	// Test that days > 90 gets normalized to 90
	mockRepo.On("GetMemberGrowthHistory", ctx, serverID, 90).Return([]*models.MemberGrowthPoint{}, nil)

	result, err := service.GetMemberGrowth(ctx, serverID, ownerID, 100)
	assert.NoError(t, err)
	assert.Equal(t, "90d", result.Period)

	// Test that days <= 0 gets normalized to 7
	mockRepo.On("GetMemberGrowthHistory", ctx, serverID, 7).Return([]*models.MemberGrowthPoint{}, nil)

	result, err = service.GetMemberGrowth(ctx, serverID, ownerID, 0)
	assert.NoError(t, err)
	assert.Equal(t, "7d", result.Period)
}

func TestAnalyticsService_TakeDailySnapshots(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo, nil, nil, nil)

	mockRepo.On("TakeMemberSnapshot", ctx).Return(nil)

	err := service.TakeDailySnapshots(ctx)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_CleanupOldData(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo, nil, nil, nil)

	mockRepo.On("CleanupOldAnalyticsData", ctx, 90).Return(nil)

	err := service.CleanupOldData(ctx, 90)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_CleanupOldData_DefaultRetention(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo, nil, nil, nil)

	// When retention is 0, default to 90
	mockRepo.On("CleanupOldAnalyticsData", ctx, 90).Return(nil)

	err := service.CleanupOldData(ctx, 0)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
