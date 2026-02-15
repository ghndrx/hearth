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

func (m *MockServerRepository) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func setupChannelService() (*ChannelService, *MockChannelRepository, *MockServerRepository, *MockCacheService, *MockEventBus) {
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)
	service := NewChannelService(channelRepo, serverRepo, cache, eventBus)
	return service, channelRepo, serverRepo, cache, eventBus
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
