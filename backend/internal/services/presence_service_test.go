package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockServerRepositoryForPresence is a mock for ServerRepository used in presence tests
type MockServerRepositoryForPresence struct {
	mock.Mock
}

func (m *MockServerRepositoryForPresence) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForPresence) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForPresence) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForPresence) BanMember(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) UnbanMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForPresence) IsBanned(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForPresence) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForPresence) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForPresence) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForPresence) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

// ========== UpdatePresence Tests ==========

func TestPresenceService_UpdatePresence_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// Setup expectations
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{
		{ID: serverID, Name: "Test Server"},
	}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	// Execute
	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	// Assert
	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
	mockServerRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_UpdatePresence_WithCustomStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	customStatus := "Coding in Go 🚀"

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusDND), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.UpdatePresence(ctx, userID, models.StatusDND, &customStatus, "web")

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_UpdatePresence_CacheError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(errors.New("cache error"))

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "mobile")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache error")
}

func TestPresenceService_UpdatePresence_AllStatuses(t *testing.T) {
	testCases := []struct {
		name   string
		status models.PresenceStatus
	}{
		{"Online", models.StatusOnline},
		{"Idle", models.StatusIdle},
		{"DND", models.StatusDND},
		{"Invisible", models.StatusInvisible},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()

			mockCache := new(MockCacheService)
			mockEventBus := new(MockEventBus)
			mockServerRepo := new(MockServerRepositoryForPresence)

			service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

			mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(tc.status), presenceTTL).Return(nil)
			mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

			err := service.UpdatePresence(ctx, userID, tc.status, nil, "desktop")

			assert.NoError(t, err)
		})
	}
}

// ========== GetPresence Tests ==========

func TestPresenceService_GetPresence_Online(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)

	presence, err := service.GetPresence(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, presence)
	assert.Equal(t, userID, presence.UserID)
	assert.Equal(t, models.StatusOnline, presence.Status)
}

func TestPresenceService_GetPresence_Offline_CacheMiss(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return(nil, errors.New("cache miss"))

	presence, err := service.GetPresence(ctx, userID)

	assert.NoError(t, err) // Should not error, just return offline
	assert.NotNil(t, presence)
	assert.Equal(t, userID, presence.UserID)
	assert.Equal(t, models.StatusOffline, presence.Status)
}

func TestPresenceService_GetPresence_AllStatuses(t *testing.T) {
	testCases := []models.PresenceStatus{
		models.StatusOnline,
		models.StatusIdle,
		models.StatusDND,
		models.StatusInvisible,
	}

	for _, status := range testCases {
		t.Run(string(status), func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()

			mockCache := new(MockCacheService)
			mockEventBus := new(MockEventBus)
			mockServerRepo := new(MockServerRepositoryForPresence)

			service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

			mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(status), nil)

			presence, err := service.GetPresence(ctx, userID)

			assert.NoError(t, err)
			assert.Equal(t, status, presence.Status)
		})
	}
}

// ========== GetBulkPresence Tests ==========

func TestPresenceService_GetBulkPresence_MultipleUsers(t *testing.T) {
	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)
	mockCache.On("Get", ctx, "presence:"+user3.String()).Return(nil, errors.New("not found"))

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{user1, user2, user3})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
	assert.Equal(t, models.StatusOffline, result[user3].Status)
}

func TestPresenceService_GetBulkPresence_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestPresenceService_GetBulkPresence_SingleUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusDND), nil)

	result, err := service.GetBulkPresence(ctx, []uuid.UUID{userID})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, models.StatusDND, result[userID].Status)
}

// ========== Heartbeat Tests ==========

func TestPresenceService_Heartbeat_ExtendsOnlineStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_DefaultsToOnlineIfMissing(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return(nil, errors.New("not found"))
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_PreservesExistingStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusDND), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusDND), presenceTTL).Return(nil)

	err := service.Heartbeat(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestPresenceService_Heartbeat_CacheError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(errors.New("cache error"))

	err := service.Heartbeat(ctx, userID)

	assert.Error(t, err)
}

// ========== SetOffline Tests ==========

func TestPresenceService_SetOffline_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{
		{ID: serverID, Name: "Test Server"},
	}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_SetOffline_DeleteError_StillBroadcasts(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// Even if delete fails, we should still broadcast
	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(errors.New("delete failed"))
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err) // SetOffline doesn't propagate delete errors
}

func TestPresenceService_SetOffline_NoServers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{}, nil)

	err := service.SetOffline(ctx, userID)

	assert.NoError(t, err)
	mockEventBus.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// ========== TypingStart Tests ==========

func TestPresenceService_TypingStart_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockEventBus.On("Publish", "typing.started", mock.MatchedBy(func(indicator *models.TypingIndicator) bool {
		return indicator.UserID == userID && indicator.ChannelID == channelID
	})).Return()

	err := service.TypingStart(ctx, userID, channelID)

	assert.NoError(t, err)
	mockEventBus.AssertExpectations(t)
}

func TestPresenceService_TypingStart_SetsTimestamp(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()
	beforeCall := time.Now()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	var capturedIndicator *models.TypingIndicator
	mockEventBus.On("Publish", "typing.started", mock.MatchedBy(func(indicator *models.TypingIndicator) bool {
		capturedIndicator = indicator
		return true
	})).Return()

	err := service.TypingStart(ctx, userID, channelID)

	assert.NoError(t, err)
	assert.NotNil(t, capturedIndicator)
	assert.False(t, capturedIndicator.Timestamp.Before(beforeCall))
	assert.False(t, capturedIndicator.Timestamp.After(time.Now()))
}

// ========== GetServerPresences Tests ==========

func TestPresenceService_GetServerPresences_Success(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	members := []*models.Member{
		{UserID: user1, ServerID: serverID},
		{UserID: user2, ServerID: serverID},
	}

	mockServerRepo.On("GetMembers", ctx, serverID, 0, 1000).Return(members, nil)
	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
}

func TestPresenceService_GetServerPresences_EmptyServer(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockServerRepo.On("GetMembers", ctx, serverID, 0, 1000).Return([]*models.Member{}, nil)

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestPresenceService_GetServerPresences_RepoError(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockServerRepo.On("GetMembers", ctx, serverID, 0, 1000).Return(nil, errors.New("database error"))

	result, err := service.GetServerPresences(ctx, serverID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestPresenceService_GetServerPresences_MixedPresence(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	user1 := uuid.New() // online
	user2 := uuid.New() // idle
	user3 := uuid.New() // dnd
	user4 := uuid.New() // offline (cache miss)

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	members := []*models.Member{
		{UserID: user1, ServerID: serverID},
		{UserID: user2, ServerID: serverID},
		{UserID: user3, ServerID: serverID},
		{UserID: user4, ServerID: serverID},
	}

	mockServerRepo.On("GetMembers", ctx, serverID, 0, 1000).Return(members, nil)
	mockCache.On("Get", ctx, "presence:"+user1.String()).Return([]byte(models.StatusOnline), nil)
	mockCache.On("Get", ctx, "presence:"+user2.String()).Return([]byte(models.StatusIdle), nil)
	mockCache.On("Get", ctx, "presence:"+user3.String()).Return([]byte(models.StatusDND), nil)
	mockCache.On("Get", ctx, "presence:"+user4.String()).Return(nil, errors.New("not found"))

	result, err := service.GetServerPresences(ctx, serverID)

	assert.NoError(t, err)
	assert.Len(t, result, 4)
	assert.Equal(t, models.StatusOnline, result[user1].Status)
	assert.Equal(t, models.StatusIdle, result[user2].Status)
	assert.Equal(t, models.StatusDND, result[user3].Status)
	assert.Equal(t, models.StatusOffline, result[user4].Status)
}

// ========== BroadcastPresenceUpdate Tests ==========

func TestPresenceService_BroadcastPresenceUpdate_MultipleServers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	server1 := uuid.New()
	server2 := uuid.New()
	server3 := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	servers := []*models.Server{
		{ID: server1, Name: "Server 1"},
		{ID: server2, Name: "Server 2"},
		{ID: server3, Name: "Server 3"},
	}

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return(servers, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return().Times(3)

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	assert.NoError(t, err)
	mockEventBus.AssertNumberOfCalls(t, "Publish", 3)
}

func TestPresenceService_BroadcastPresenceUpdate_ServerRepoError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return(nil, errors.New("db error"))

	// Even with server repo error, the main operation succeeds
	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")

	assert.NoError(t, err)
	// No events should be published when server lookup fails
	mockEventBus.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// ========== PresenceUpdateEvent Tests ==========

func TestPresenceUpdateEvent_Structure(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusOnline,
	}

	event := &PresenceUpdateEvent{
		UserID:   userID,
		ServerID: serverID,
		Presence: presence,
	}

	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, serverID, event.ServerID)
	assert.NotNil(t, event.Presence)
	assert.Equal(t, models.StatusOnline, event.Presence.Status)
}

// ========== Constants Tests ==========

func TestPresenceService_Constants(t *testing.T) {
	// Verify constants are reasonable
	assert.Equal(t, 2*time.Minute, presenceTTL)
	assert.Equal(t, 5*time.Minute, idleTimeout)
	assert.Equal(t, 30*time.Second, heartbeatInterval)
}

// ========== Integration-style Tests ==========

func TestPresenceService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	mockCache := new(MockCacheService)
	mockEventBus := new(MockEventBus)
	mockServerRepo := new(MockServerRepositoryForPresence)

	service := NewPresenceService(mockCache, mockEventBus, mockServerRepo)

	// 1. User comes online
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	mockServerRepo.On("GetUserServers", ctx, userID).Return([]*models.Server{{ID: serverID}}, nil)
	mockEventBus.On("Publish", "presence.updated", mock.AnythingOfType("*services.PresenceUpdateEvent")).Return()

	err := service.UpdatePresence(ctx, userID, models.StatusOnline, nil, "desktop")
	assert.NoError(t, err)

	// 2. Check presence
	mockCache.On("Get", ctx, "presence:"+userID.String()).Return([]byte(models.StatusOnline), nil)
	presence, err := service.GetPresence(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusOnline, presence.Status)

	// 3. Heartbeat
	mockCache.On("Set", ctx, "presence:"+userID.String(), []byte(models.StatusOnline), presenceTTL).Return(nil)
	err = service.Heartbeat(ctx, userID)
	assert.NoError(t, err)

	// 4. User goes offline
	mockCache.On("Delete", ctx, "presence:"+userID.String()).Return(nil)
	err = service.SetOffline(ctx, userID)
	assert.NoError(t, err)
}
