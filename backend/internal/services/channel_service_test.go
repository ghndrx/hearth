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

// MockChannelRepository is a mock implementation of ChannelRepository
type MockChannelRepository struct {
	mock.Mock
}

func (m *MockChannelRepository) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepository) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepository) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepository) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepository) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func (m *MockChannelRepository) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PermissionOverride), args.Error(1)
}

func (m *MockChannelRepository) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockChannelRepository) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	args := m.Called(ctx, channelID, targetID, targetType)
	return args.Error(0)
}

func (m *MockChannelRepository) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepository) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepository) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockChannelRepository) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

// MockServerRepository for channel tests
type MockServerRepository struct {
	mock.Mock
}

func (m *MockServerRepository) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepository) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepository) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepository) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepository) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepository) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepository) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepository) BanMember(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepository) UnbanMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepository) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepository) IsBanned(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockServerRepository) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepository) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepository) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepository) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepository) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepository) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepository) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepository) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepository) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepository) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	args := m.Called(ctx, vanityCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepository) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepository) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepository) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepository) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockServerRepository) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, inviteCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

func (m *MockServerRepository) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

func (m *MockServerRepository) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepository) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func setupChannelService() (*ChannelService, *MockChannelRepository, *MockServerRepository, *MockCacheService, *MockEventBus) {
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)
	// Note: permService is nil for basic tests; permission tests use setupChannelServiceWithPermissions
	service := NewChannelService(channelRepo, serverRepo, nil, cache, eventBus)
	return service, channelRepo, serverRepo, cache, eventBus
}

// Note: MockRoleRepository is defined in server_service_test.go

func setupChannelServiceWithPermissions() (*ChannelService, *MockChannelRepository, *MockServerRepository, *MockRoleRepository, *MockCacheService, *MockEventBus) {
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	roleRepo := new(MockRoleRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)
	permService := NewPermissionService(serverRepo, roleRepo, channelRepo, nil)
	service := NewChannelService(channelRepo, serverRepo, permService, cache, eventBus)
	return service, channelRepo, serverRepo, roleRepo, cache, eventBus
}

func TestGetChannel_Success(t *testing.T) {
	service, channelRepo, _, cache, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()

	expectedChannel := &models.Channel{
		ID:        channelID,
		Name:      "general",
		Type:      models.ChannelTypeText,
		CreatedAt: time.Now(),
	}

	cache.On("GetChannel", ctx, channelID).Return(nil, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(expectedChannel, nil)
	cache.On("SetChannel", ctx, expectedChannel, 5*time.Minute).Return(nil)

	channel, err := service.GetChannel(ctx, channelID)

	assert.NoError(t, err)
	assert.Equal(t, "general", channel.Name)
	channelRepo.AssertExpectations(t)
}

func TestGetChannel_FromCache(t *testing.T) {
	service, channelRepo, _, cache, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()

	cachedChannel := &models.Channel{
		ID:   channelID,
		Name: "cached-channel",
	}

	cache.On("GetChannel", ctx, channelID).Return(cachedChannel, nil)

	channel, err := service.GetChannel(ctx, channelID)

	assert.NoError(t, err)
	assert.Equal(t, "cached-channel", channel.Name)
	channelRepo.AssertNotCalled(t, "GetByID")
}

func TestGetChannel_NotFound(t *testing.T) {
	service, channelRepo, _, cache, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()

	cache.On("GetChannel", ctx, channelID).Return(nil, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	channel, err := service.GetChannel(ctx, channelID)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, channel)
}

func TestCreateChannel_Success(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "new-channel", models.ChannelTypeText, nil)

	assert.NoError(t, err)
	assert.Equal(t, "new-channel", channel.Name)
	assert.Equal(t, models.ChannelTypeText, channel.Type)
	channelRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
}

func TestCreateChannel_NotMember(t *testing.T) {
	service, _, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, nil)

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "new-channel", models.ChannelTypeText, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, channel)
}

func TestUpdateChannel_Success(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "old-name",
		Topic:    "old topic",
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	newName := "new-name"
	newTopic := "new topic"
	updates := &models.ChannelUpdate{
		Name:  &newName,
		Topic: &newTopic,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("Update", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteChannel", ctx, channelID).Return(nil)
	eventBus.On("Publish", "channel.updated", mock.AnythingOfType("*services.ChannelUpdatedEvent")).Return()

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.NoError(t, err)
	assert.Equal(t, "new-name", channel.Name)
	assert.Equal(t, "new topic", channel.Topic)
}

func TestDeleteChannel_Success(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("Delete", ctx, channelID).Return(nil)
	cache.On("DeleteChannel", ctx, channelID).Return(nil)
	eventBus.On("Publish", "channel.deleted", mock.AnythingOfType("*services.ChannelDeletedEvent")).Return()

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.NoError(t, err)
	channelRepo.AssertExpectations(t)
}

func TestDeleteChannel_CannotDeleteDM(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	requesterID := uuid.New()

	dmChannel := &models.Channel{
		ID:   channelID,
		Type: models.ChannelTypeDM,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(dmChannel, nil)

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrCannotDeleteDM, err)
}

func TestGetOrCreateDM_Existing(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	user1ID := uuid.New()
	user2ID := uuid.New()

	existingDM := &models.Channel{
		ID:         uuid.New(),
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{user1ID, user2ID},
	}

	channelRepo.On("GetDMChannel", ctx, user1ID, user2ID).Return(existingDM, nil)

	channel, err := service.GetOrCreateDM(ctx, user1ID, user2ID)

	assert.NoError(t, err)
	assert.Equal(t, models.ChannelTypeDM, channel.Type)
	channelRepo.AssertNotCalled(t, "Create")
}

func TestGetOrCreateDM_CreateNew(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	user1ID := uuid.New()
	user2ID := uuid.New()

	channelRepo.On("GetDMChannel", ctx, user1ID, user2ID).Return(nil, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)

	channel, err := service.GetOrCreateDM(ctx, user1ID, user2ID)

	assert.NoError(t, err)
	assert.Equal(t, models.ChannelTypeDM, channel.Type)
	assert.True(t, channel.E2EEEnabled) // DMs should be E2EE
	channelRepo.AssertExpectations(t)
}

func TestCreateGroupDM_Success(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	ownerID := uuid.New()
	recipientIDs := []uuid.UUID{uuid.New(), uuid.New()}

	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)

	channel, err := service.CreateGroupDM(ctx, ownerID, "Friend Group", recipientIDs)

	assert.NoError(t, err)
	assert.Equal(t, "Friend Group", channel.Name)
	assert.Equal(t, models.ChannelTypeGroupDM, channel.Type)
	assert.Equal(t, &ownerID, channel.OwnerID)
	assert.Len(t, channel.Recipients, 3) // owner + 2 recipients
}

func TestGetServerChannels_Success(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedChannels := []*models.Channel{
		{ID: uuid.New(), Name: "general"},
		{ID: uuid.New(), Name: "random"},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(expectedChannels, nil)

	channels, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, channels, 2)
}

func TestGetServerChannels_NotMember(t *testing.T) {
	service, _, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	channels, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, channels)
}

func TestGetUserDMs_Success(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID := uuid.New()

	expectedChannels := []*models.Channel{
		{
			ID:         uuid.New(),
			Type:       models.ChannelTypeDM,
			Recipients: []uuid.UUID{userID, uuid.New()},
		},
		{
			ID:         uuid.New(),
			Type:       models.ChannelTypeGroupDM,
			Recipients: []uuid.UUID{userID, uuid.New(), uuid.New()},
		},
	}

	channelRepo.On("GetUserDMs", ctx, userID).Return(expectedChannels, nil)

	channels, err := service.GetUserDMs(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, channels, 2)
	assert.Equal(t, expectedChannels[0].ID, channels[0].ID)
	assert.Equal(t, expectedChannels[1].ID, channels[1].ID)
	channelRepo.AssertExpectations(t)
}

func TestGetUserDMs_Empty(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID := uuid.New()

	channelRepo.On("GetUserDMs", ctx, userID).Return([]*models.Channel{}, nil)

	channels, err := service.GetUserDMs(ctx, userID)

	assert.NoError(t, err)
	assert.Empty(t, channels)
	assert.NotNil(t, channels)
	channelRepo.AssertExpectations(t)
}

func TestGetUserDMs_RepositoryError(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID := uuid.New()

	channelRepo.On("GetUserDMs", ctx, userID).Return(nil, errors.New("database error"))

	channels, err := service.GetUserDMs(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, channels)
	assert.Equal(t, "database error", err.Error())
	channelRepo.AssertExpectations(t)
}

// Permission tests

func TestCreateChannel_PermissionDenied(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, cache, eventBus := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(), // Different owner
	}

	// Default role with no MANAGE_CHANNELS permission
	defaultRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        "@everyone",
		IsDefault:   true,
		Permissions: models.DefaultPermissions, // Does not include PermManageChannels
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, creatorID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(defaultRole, nil)

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "new-channel", models.ChannelTypeText, nil)

	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "permission")
	serverRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)

	// These should not be called since permission check fails first
	channelRepo.AssertNotCalled(t, "GetByServerID")
	channelRepo.AssertNotCalled(t, "Create")
	cache.AssertNotCalled(t, "DeleteServer")
	eventBus.AssertNotCalled(t, "Publish")
}

func TestCreateChannel_WithManageChannelsPermission(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, cache, eventBus := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	// User has MANAGE_CHANNELS permission from their role
	roleRepo.On("GetMemberPermissions", ctx, serverID, creatorID).Return(models.PermManageChannels|models.PermViewChannels, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(nil, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "new-channel", models.ChannelTypeText, nil)

	assert.NoError(t, err)
	assert.NotNil(t, channel)
	assert.Equal(t, "new-channel", channel.Name)
	serverRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	channelRepo.AssertExpectations(t)
}

func TestCreateChannel_ServerOwnerBypassesPermissionCheck(t *testing.T) {
	service, channelRepo, serverRepo, _, cache, eventBus := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	member := &models.Member{
		UserID:   ownerID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	// Owner should bypass all permission checks
	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetMember", ctx, serverID, ownerID).Return(member, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	// Owner gets all permissions automatically in PermissionService
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, ownerID, "owner-channel", models.ChannelTypeText, nil)

	assert.NoError(t, err)
	assert.NotNil(t, channel)
	assert.Equal(t, "owner-channel", channel.Name)
}

func TestDeleteChannel_PermissionDenied(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, cache, eventBus := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "to-delete",
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	defaultRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        "@everyone",
		IsDefault:   true,
		Permissions: models.DefaultPermissions,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(defaultRole, nil)

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
	roleRepo.AssertExpectations(t)

	// Delete should not be called
	channelRepo.AssertNotCalled(t, "Delete")
	cache.AssertNotCalled(t, "DeleteChannel")
	eventBus.AssertNotCalled(t, "Publish")
}

// =============================================================================
// GetServerChannels Permission Filtering Tests
// =============================================================================

func TestGetServerChannels_WithPermissionFiltering_SomeVisible(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(), // Different from requester
	}

	// Create channels with permission overrides
	visibleChannelID := uuid.New()
	hiddenChannelID := uuid.New()

	channels := []*models.Channel{
		{
			ID:       visibleChannelID,
			ServerID: &serverID,
			Name:     "visible-channel",
			Type:     models.ChannelTypeText,
		},
		{
			ID:       hiddenChannelID,
			ServerID: &serverID,
			Name:     "hidden-channel",
			Type:     models.ChannelTypeText,
		},
	}

	// Override that denies VIEW_CHANNELS for the hidden channel (using @everyone role)
	// @everyone role ID is the same as server ID
	hiddenOverrides := []models.PermissionOverride{
		{
			ChannelID:  hiddenChannelID,
			TargetType: "role",
			TargetID:   serverID, // @everyone role ID equals server ID
			Deny:       models.PermViewChannels,
			Allow:      0,
		},
	}
	visibleOverrides := []models.PermissionOverride{}

	// @everyone role with VIEW_CHANNELS permission
	everyoneRole := &models.Role{
		ID:          serverID, // @everyone role ID equals server ID
		ServerID:    serverID,
		Name:        "@everyone",
		Permissions: models.PermViewChannels, // Grant VIEW_CHANNELS by default
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(channels, nil)

	// Setup permission checking mocks - GetByID called for each channel to check ownership
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil).Times(2)

	// Setup permission checking mocks - return @everyone role
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{everyoneRole}, nil).Times(2)

	// For visible channel - no overrides that deny
	channelRepo.On("GetPermissionOverrides", ctx, visibleChannelID).Return(visibleOverrides, nil).Once()

	// For hidden channel - has deny override
	channelRepo.On("GetPermissionOverrides", ctx, hiddenChannelID).Return(hiddenOverrides, nil).Once()

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "visible-channel", result[0].Name)
}

func TestGetServerChannels_WithPermissionFiltering_AllVisible(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	channels := []*models.Channel{
		{ID: uuid.New(), ServerID: &serverID, Name: "general", Type: models.ChannelTypeText},
		{ID: uuid.New(), ServerID: &serverID, Name: "random", Type: models.ChannelTypeText},
		{ID: uuid.New(), ServerID: &serverID, Name: "announcements", Type: models.ChannelTypeText},
	}

	// @everyone role with VIEW_CHANNELS permission
	everyoneRole := &models.Role{
		ID:          serverID, // @everyone role ID equals server ID
		ServerID:    serverID,
		Name:        "@everyone",
		Permissions: models.PermViewChannels,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(channels, nil)

	// All channels visible - GetByID called for each channel to check ownership
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil).Times(3)

	// All channels visible - return @everyone role with VIEW_CHANNELS
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{everyoneRole}, nil).Times(3)
	channelRepo.On("GetPermissionOverrides", ctx, mock.Anything).Return([]models.PermissionOverride{}, nil).Times(3)

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestGetServerChannels_WithPermissionFiltering_NoneVisible(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	channel1ID := uuid.New()
	channel2ID := uuid.New()

	channels := []*models.Channel{
		{ID: channel1ID, ServerID: &serverID, Name: "secret-1", Type: models.ChannelTypeText},
		{ID: channel2ID, ServerID: &serverID, Name: "secret-2", Type: models.ChannelTypeText},
	}

	// Deny VIEW_CHANNELS on all channels via @everyone role
	// @everyone role ID equals server ID
	denyAllOverrides := []models.PermissionOverride{
		{
			ChannelID:  channel1ID,
			TargetType: "role",
			TargetID:   serverID, // @everyone role ID equals server ID
			Deny:       models.PermViewChannels,
			Allow:      0,
		},
	}
	denyAllOverrides2 := []models.PermissionOverride{
		{
			ChannelID:  channel2ID,
			TargetType: "role",
			TargetID:   serverID, // @everyone role ID equals server ID
			Deny:       models.PermViewChannels,
			Allow:      0,
		},
	}

	// @everyone role with VIEW_CHANNELS permission (will be denied by overrides)
	everyoneRole := &models.Role{
		ID:          serverID, // @everyone role ID equals server ID
		ServerID:    serverID,
		Name:        "@everyone",
		Permissions: models.PermViewChannels,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(channels, nil)

	// GetByID called for each channel to check ownership
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil).Times(2)

	// Return @everyone role
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{everyoneRole}, nil).Times(2)
	channelRepo.On("GetPermissionOverrides", ctx, channel1ID).Return(denyAllOverrides, nil).Once()
	channelRepo.On("GetPermissionOverrides", ctx, channel2ID).Return(denyAllOverrides2, nil).Once()

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Empty(t, result)
}

func TestGetServerChannels_OwnerSeesAllChannels(t *testing.T) {
	service, channelRepo, serverRepo, _, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	member := &models.Member{
		UserID:   ownerID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	channels := []*models.Channel{
		{ID: uuid.New(), ServerID: &serverID, Name: "general", Type: models.ChannelTypeText},
		{ID: uuid.New(), ServerID: &serverID, Name: "admin-only", Type: models.ChannelTypeText},
		{ID: uuid.New(), ServerID: &serverID, Name: "mods-only", Type: models.ChannelTypeText},
	}

	serverRepo.On("GetMember", ctx, serverID, ownerID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(channels, nil)

	// Owner bypass - GetByID is called to check ownership
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil).Times(3)

	result, err := service.GetServerChannels(ctx, serverID, ownerID)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestGetServerChannels_PermissionCheckError(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	channels := []*models.Channel{
		{ID: uuid.New(), ServerID: &serverID, Name: "general", Type: models.ChannelTypeText},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(channels, nil)

	// GetByID called first to check ownership
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil).Once()

	// Simulate error when getting roles - this results in channel being filtered out, not an error
	roleRepo.On("GetByServerID", ctx, serverID).Return(nil, errors.New("database error")).Once()

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	// Permission errors are swallowed - channel is just not visible
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetServerChannels_ChannelRepoError(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(nil, errors.New("database connection failed"))

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestGetServerChannels_EmptyServer(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)

	result, err := service.GetServerChannels(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

// =============================================================================
// GetSharedChannelsWithServerNames Tests
// =============================================================================

func TestGetSharedChannelsWithServerNames_WithRepoMethod(t *testing.T) {
	_, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create a mock repo that implements the extended interface
	mockExtendedRepo := &MockExtendedChannelRepository{MockChannelRepository: channelRepo}

	serviceWithExtendedRepo := NewChannelService(
		mockExtendedRepo,
		nil,
		nil,
		nil,
		nil,
	)

	// The repo returns a different type that gets converted
	channelID := uuid.New()
	serverID := uuid.New()
	serverName := "Test Server"
	repoResult := []*struct {
		ID         uuid.UUID  `db:"id"`
		Name       string     `db:"name"`
		ServerID   *uuid.UUID `db:"server_id"`
		ServerName string     `db:"server_name"`
		ServerIcon *string    `db:"server_icon"`
	}{
		{
			ID:         channelID,
			Name:       "general",
			ServerID:   &serverID,
			ServerName: serverName,
			ServerIcon: nil,
		},
	}

	mockExtendedRepo.On("GetSharedChannelsWithServerNames", ctx, userID1, userID2, 10).Return(repoResult, 1, nil)

	result, total, err := serviceWithExtendedRepo.GetSharedChannelsWithServerNames(ctx, userID1, userID2, 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, result, 1)
	assert.Equal(t, "general", result[0].Name)
	assert.Equal(t, "Test Server", result[0].ServerName)
}

func TestGetSharedChannelsWithServerNames_WithoutRepoMethod(t *testing.T) {
	service, _, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Use standard repo (doesn't have the extended method)
	result, total, err := service.GetSharedChannelsWithServerNames(ctx, userID1, userID2, 10)

	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, result)
}

func TestGetSharedChannelsWithServerNames_RepoError(t *testing.T) {
	_, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()

	mockExtendedRepo := &MockExtendedChannelRepository{MockChannelRepository: channelRepo}

	serviceWithExtendedRepo := NewChannelService(
		mockExtendedRepo,
		nil,
		nil,
		nil,
		nil,
	)

	mockExtendedRepo.On("GetSharedChannelsWithServerNames", ctx, userID1, userID2, 10).Return(nil, 0, errors.New("query failed"))

	result, total, err := serviceWithExtendedRepo.GetSharedChannelsWithServerNames(ctx, userID1, userID2, 10)

	assert.Error(t, err)
	assert.Equal(t, 0, total)
	assert.Nil(t, result)
}

// MockExtendedChannelRepository extends MockChannelRepository with the additional method
type MockExtendedChannelRepository struct {
	*MockChannelRepository
}

func (m *MockExtendedChannelRepository) GetSharedChannelsWithServerNames(ctx context.Context, userID1, userID2 uuid.UUID, limit int) (interface{}, int, error) {
	args := m.Called(ctx, userID1, userID2, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0), args.Int(1), args.Error(2)
}

func (m *MockExtendedChannelRepository) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockExtendedChannelRepository) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockExtendedChannelRepository) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockExtendedChannelRepository) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

// =============================================================================
// Additional Channel Service Tests
// =============================================================================

func TestGetChannel_CacheErrorFallback(t *testing.T) {
	service, channelRepo, _, cache, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()

	expectedChannel := &models.Channel{
		ID:   channelID,
		Name: "test-channel",
	}

	// Cache error should fall back to repository
	cache.On("GetChannel", ctx, channelID).Return(nil, errors.New("cache unavailable"))
	channelRepo.On("GetByID", ctx, channelID).Return(expectedChannel, nil)
	cache.On("SetChannel", ctx, expectedChannel, 5*time.Minute).Return(nil)

	channel, err := service.GetChannel(ctx, channelID)

	assert.NoError(t, err)
	assert.Equal(t, expectedChannel.Name, channel.Name)
}

func TestGetChannel_RepositoryError(t *testing.T) {
	service, channelRepo, _, cache, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()

	cache.On("GetChannel", ctx, channelID).Return(nil, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(nil, errors.New("connection timeout"))

	channel, err := service.GetChannel(ctx, channelID)

	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "connection timeout")
}

func TestCreateChannel_WithParent(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()
	parentID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "child-channel", models.ChannelTypeText, &parentID)

	assert.NoError(t, err)
	assert.NotNil(t, channel.ParentID)
	assert.Equal(t, parentID, *channel.ParentID)
}

func TestCreateChannel_ExistingChannelsPositioning(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	existingChannels := []*models.Channel{
		{ID: uuid.New(), Position: 0},
		{ID: uuid.New(), Position: 1},
		{ID: uuid.New(), Position: 2},
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return(existingChannels, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "new-channel", models.ChannelTypeText, nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, channel.Position)
}

func TestCreateChannel_RepoError(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(errors.New("unique constraint violation"))

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "duplicate-channel", models.ChannelTypeText, nil)

	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "unique constraint violation")
}

func TestCreateChannel_ChannelTypeVoice(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{}, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteServer", ctx, serverID).Return(nil)
	eventBus.On("Publish", "channel.created", mock.AnythingOfType("*services.ChannelCreatedEvent")).Return()

	channel, err := service.CreateChannel(ctx, serverID, creatorID, "voice-room", models.ChannelTypeVoice, nil)

	assert.NoError(t, err)
	assert.Equal(t, models.ChannelTypeVoice, channel.Type)
}

func TestUpdateChannel_NotFound(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	requesterID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	newName := "updated-name"
	updates := &models.ChannelUpdate{Name: &newName}

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, channel)
}

func TestUpdateChannel_DMChannel(t *testing.T) {
	service, channelRepo, _, cache, eventBus := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	requesterID := uuid.New()

	dmChannel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{requesterID, uuid.New()},
	}

	channelRepo.On("GetByID", ctx, channelID).Return(dmChannel, nil)
	channelRepo.On("Update", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteChannel", ctx, channelID).Return(nil)
	eventBus.On("Publish", "channel.updated", mock.AnythingOfType("*services.ChannelUpdatedEvent")).Return()

	newName := "dm-name"
	updates := &models.ChannelUpdate{Name: &newName}

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.NoError(t, err)
	assert.Equal(t, "dm-name", channel.Name)
}

func TestUpdateChannel_AllFields(t *testing.T) {
	service, channelRepo, serverRepo, cache, eventBus := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:          channelID,
		ServerID:    &serverID,
		Name:        "old-name",
		Topic:       "old-topic",
		Position:    0,
		Slowmode:    0,
		NSFW:        false,
		E2EEEnabled: false,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	newName := "new-name"
	newTopic := "new-topic"
	newPosition := 5
	newSlowmode := 30
	newNSFW := true
	newE2EE := true

	updates := &models.ChannelUpdate{
		Name:        &newName,
		Topic:       &newTopic,
		Position:    &newPosition,
		Slowmode:    &newSlowmode,
		NSFW:        &newNSFW,
		E2EEEnabled: &newE2EE,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("Update", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)
	cache.On("DeleteChannel", ctx, channelID).Return(nil)
	eventBus.On("Publish", "channel.updated", mock.AnythingOfType("*services.ChannelUpdatedEvent")).Return()

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.NoError(t, err)
	assert.Equal(t, "new-name", channel.Name)
	assert.Equal(t, "new-topic", channel.Topic)
	assert.Equal(t, 5, channel.Position)
	assert.Equal(t, 30, channel.Slowmode)
	assert.True(t, channel.NSFW)
	assert.True(t, channel.E2EEEnabled)
}

func TestUpdateChannel_RepoError(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "channel",
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("Update", ctx, mock.AnythingOfType("*models.Channel")).Return(errors.New("update failed"))

	newName := "updated"
	updates := &models.ChannelUpdate{Name: &newName}

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.Error(t, err)
	assert.Nil(t, channel)
}

func TestDeleteChannel_NotFound(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	requesterID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
}

func TestDeleteChannel_GroupDM(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	requesterID := uuid.New()

	groupDMChannel := &models.Channel{
		ID:   channelID,
		Type: models.ChannelTypeGroupDM,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(groupDMChannel, nil)

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrCannotDeleteDM, err)
}

func TestDeleteChannel_NotServerMember(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
}

func TestDeleteChannel_RepoError(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("Delete", ctx, channelID).Return(errors.New("delete failed"))

	err := service.DeleteChannel(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

func TestGetOrCreateDM_RepoErrorOnGet(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	user1ID := uuid.New()
	user2ID := uuid.New()

	channelRepo.On("GetDMChannel", ctx, user1ID, user2ID).Return(nil, errors.New("query error"))

	channel, err := service.GetOrCreateDM(ctx, user1ID, user2ID)

	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "query error")
}

func TestGetOrCreateDM_CreateError(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	user1ID := uuid.New()
	user2ID := uuid.New()

	channelRepo.On("GetDMChannel", ctx, user1ID, user2ID).Return(nil, nil)
	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(errors.New("insert failed"))

	channel, err := service.GetOrCreateDM(ctx, user1ID, user2ID)

	assert.Error(t, err)
	assert.Nil(t, channel)
}

func TestCreateGroupDM_EmptyRecipients(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	ownerID := uuid.New()

	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(nil)

	channel, err := service.CreateGroupDM(ctx, ownerID, "Solo Group", []uuid.UUID{})

	assert.NoError(t, err)
	assert.Len(t, channel.Recipients, 1) // Just the owner
	assert.Equal(t, &ownerID, channel.OwnerID)
}

func TestCreateGroupDM_RepoError(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	ownerID := uuid.New()
	recipientIDs := []uuid.UUID{uuid.New()}

	channelRepo.On("Create", ctx, mock.AnythingOfType("*models.Channel")).Return(errors.New("create error"))

	channel, err := service.CreateGroupDM(ctx, ownerID, "Test Group", recipientIDs)

	assert.Error(t, err)
	assert.Nil(t, channel)
}

func TestGetUserDMs_NilResult(t *testing.T) {
	service, channelRepo, _, _, _ := setupChannelService()
	ctx := context.Background()
	userID := uuid.New()

	channelRepo.On("GetUserDMs", ctx, userID).Return(nil, nil)

	channels, err := service.GetUserDMs(ctx, userID)

	assert.NoError(t, err)
	assert.Nil(t, channels)
}

func TestGetChannel_NilCache(t *testing.T) {
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	eventBus := new(MockEventBus)

	// Create service without cache
	service := NewChannelService(channelRepo, serverRepo, nil, nil, eventBus)

	ctx := context.Background()
	channelID := uuid.New()

	expectedChannel := &models.Channel{
		ID:   channelID,
		Name: "no-cache-channel",
	}

	channelRepo.On("GetByID", ctx, channelID).Return(expectedChannel, nil)

	channel, err := service.GetChannel(ctx, channelID)

	assert.NoError(t, err)
	assert.Equal(t, "no-cache-channel", channel.Name)
}

func TestUpdateChannel_NotServerMember(t *testing.T) {
	service, channelRepo, serverRepo, _, _ := setupChannelService()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "channel",
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	newName := "updated"
	updates := &models.ChannelUpdate{Name: &newName}

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, channel)
}

func TestUpdateChannel_PermissionDenied(t *testing.T) {
	service, channelRepo, serverRepo, roleRepo, _, _ := setupChannelServiceWithPermissions()
	ctx := context.Background()
	channelID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingChannel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "channel",
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	defaultRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        "@everyone",
		IsDefault:   true,
		Permissions: models.DefaultPermissions,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(existingChannel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(defaultRole, nil)

	newName := "updated"
	updates := &models.ChannelUpdate{Name: &newName}

	channel, err := service.UpdateChannel(ctx, channelID, requesterID, updates)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
	assert.Nil(t, channel)
}
