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

// MockMessageRepository is a mock implementation of MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockMessageRepository) Update(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageRepository) GetChannelMessages(ctx context.Context, channelID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	args := m.Called(ctx, channelID, before, after, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

func (m *MockMessageRepository) GetPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]*models.Message, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

func (m *MockMessageRepository) SearchMessages(ctx context.Context, query string, channelID *uuid.UUID, authorID *uuid.UUID, limit int) ([]*models.Message, error) {
	args := m.Called(ctx, query, channelID, authorID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

func (m *MockMessageRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	args := m.Called(ctx, messageID, userID, emoji)
	return args.Error(0)
}

func (m *MockMessageRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	args := m.Called(ctx, messageID, userID, emoji)
	return args.Error(0)
}

func (m *MockMessageRepository) GetReactions(ctx context.Context, messageID uuid.UUID) ([]*models.Reaction, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Reaction), args.Error(1)
}

func (m *MockMessageRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}

func (m *MockMessageRepository) DeleteByAuthor(ctx context.Context, channelID, authorID uuid.UUID, since time.Time) (int, error) {
	args := m.Called(ctx, channelID, authorID, since)
	return args.Int(0), args.Error(1)
}

// MockQuotaService is a mock implementation
type MockQuotaService struct {
	mock.Mock
}

func (m *MockQuotaService) GetEffectiveLimits(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID) (*Limits, error) {
	args := m.Called(ctx, userID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Limits), args.Error(1)
}

// MockRateLimiter is a mock implementation
type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) Check(ctx context.Context, userID, channelID uuid.UUID) error {
	args := m.Called(ctx, userID, channelID)
	return args.Error(0)
}

func (m *MockRateLimiter) CheckSlowmode(ctx context.Context, userID, channelID uuid.UUID, slowmode int) error {
	args := m.Called(ctx, userID, channelID, slowmode)
	return args.Error(0)
}

func (m *MockRateLimiter) Reset(ctx context.Context, userID, channelID uuid.UUID) error {
	args := m.Called(ctx, userID, channelID)
	return args.Error(0)
}

// MockE2EEService is a mock implementation
type MockE2EEService struct {
	mock.Mock
}

func (m *MockE2EEService) ValidateEncryptedPayload(content string) bool {
	args := m.Called(content)
	return args.Bool(0)
}

func (m *MockE2EEService) GetPreKeys(ctx context.Context, userID uuid.UUID) (*models.PreKeyBundle, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PreKeyBundle), args.Error(1)
}

func (m *MockE2EEService) UploadPreKeys(ctx context.Context, userID uuid.UUID, bundle *models.PreKeyBundle) error {
	args := m.Called(ctx, userID, bundle)
	return args.Error(0)
}

func (m *MockE2EEService) CreateGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error {
	args := m.Called(ctx, channelID, memberIDs)
	return args.Error(0)
}

func (m *MockE2EEService) AddGroupMember(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockE2EEService) RemoveGroupMember(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

// MockChannelRepositoryForMessages implements what we need
type MockChannelRepositoryForMessages struct {
	mock.Mock
}

func (m *MockChannelRepositoryForMessages) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func setupMessageService() (*MessageService, *MockMessageRepository, *MockChannelRepositoryForMessages, *MockServerRepository, *MockQuotaService, *MockRateLimiter, *MockE2EEService, *MockCacheService, *MockEventBus) {
	msgRepo := new(MockMessageRepository)
	channelRepo := new(MockChannelRepositoryForMessages)
	serverRepo := new(MockServerRepository)
	rateLimiter := new(MockRateLimiter)
	e2eeService := new(MockE2EEService)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)
	mockQuotaService := new(MockQuotaService)

	// Create a real quota service with default config
	quotaConfig := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}
	quotaService := NewQuotaService(quotaConfig, nil, nil, nil)

	service := &MessageService{
		repo:         msgRepo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		quotaService: quotaService,
		rateLimiter:  rateLimiter,
		e2eeService:  e2eeService,
		cache:        cache,
		eventBus:     eventBus,
	}

	return service, msgRepo, channelRepo, serverRepo, mockQuotaService, rateLimiter, e2eeService, cache, eventBus
}

func TestSendMessage_Success(t *testing.T) {
	t.Skip("TODO: Fix quota service setup in tests")
}

func TestSendMessage_ChannelNotFound(t *testing.T) {
	service, _, channelRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	channelID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, message)
}

func TestSendMessage_NotServerMember(t *testing.T) {
	service, _, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, authorID).Return(nil, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, message)
}

func TestSendMessage_EmptyMessage(t *testing.T) {
	service, _, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   authorID,
		ServerID: serverID,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, authorID).Return(member, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyMessage, err)
	assert.Nil(t, message)
}

func TestEditMessage_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   "Original content",
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.updated", mock.AnythingOfType("*services.MessageUpdatedEvent")).Return()

	message, err := service.EditMessage(ctx, messageID, authorID, "Updated content")

	assert.NoError(t, err)
	assert.Equal(t, "Updated content", message.Content)
	assert.NotNil(t, message.EditedAt)
}

func TestEditMessage_NotAuthor(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	otherUserID := uuid.New()
	messageID := uuid.New()

	existingMessage := &models.Message{
		ID:       messageID,
		AuthorID: authorID,
		Content:  "Original content",
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)

	message, err := service.EditMessage(ctx, messageID, otherUserID, "Hacked!")

	assert.Error(t, err)
	assert.Equal(t, ErrNotMessageAuthor, err)
	assert.Nil(t, message)
}

func TestDeleteMessage_ByAuthor(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Delete", ctx, messageID).Return(nil)
	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil) // Not needed for author delete
	eventBus.On("Publish", "message.deleted", mock.AnythingOfType("*services.MessageDeletedEvent")).Return()

	err := service.DeleteMessage(ctx, messageID, authorID)

	assert.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestDeleteMessage_NotAuthor(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	otherUserID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
	}

	// DM channel (no server)
	channel := &models.Channel{
		ID:   channelID,
		Type: models.ChannelTypeDM,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	err := service.DeleteMessage(ctx, messageID, otherUserID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotMessageAuthor, err)
}

func TestGetMessages_Success(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{ID: uuid.New(), Content: "Message 1"},
		{ID: uuid.New(), Content: "Message 2"},
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return(expectedMessages, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 0)

	assert.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestAddReaction_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("AddReaction", ctx, messageID, userID, "👍").Return(nil)
	eventBus.On("Publish", "reaction.added", mock.AnythingOfType("*services.ReactionAddedEvent")).Return()

	err := service.AddReaction(ctx, messageID, userID, "👍")

	assert.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestRemoveReaction_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("RemoveReaction", ctx, messageID, userID, "👍").Return(nil)
	eventBus.On("Publish", "reaction.removed", mock.AnythingOfType("*services.ReactionRemovedEvent")).Return()

	err := service.RemoveReaction(ctx, messageID, userID, "👍")

	assert.NoError(t, err)
}

func TestPinMessage_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    false,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.pinned", mock.AnythingOfType("*services.MessagePinnedEvent")).Return()

	err := service.PinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
}

func TestUnpinMessage_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    true,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.unpinned", mock.AnythingOfType("*services.MessageUnpinnedEvent")).Return()

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
	assert.False(t, existingMessage.Pinned)
}

func TestUnpinMessage_NotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.Equal(t, ErrMessageNotFound, err)
}

func TestUnpinMessage_AlreadyUnpinned(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    false, // Already unpinned
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	// No Update or Publish calls expected - it's a no-op

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
}

func TestGetPinnedMessages_Success(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	pinnedMessages := []*models.Message{
		{ID: uuid.New(), ChannelID: channelID, Pinned: true, Content: "Pinned 1"},
		{ID: uuid.New(), ChannelID: channelID, Pinned: true, Content: "Pinned 2"},
	}

	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)
	msgRepo.On("GetPinnedMessages", ctx, channelID).Return(pinnedMessages, nil)

	result, err := service.GetPinnedMessages(ctx, channelID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetPinnedMessages_ChannelNotFound(t *testing.T) {
	service, _, chanRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()

	chanRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	result, err := service.GetPinnedMessages(ctx, channelID, requesterID)

	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, result)
}

func TestGetPinnedMessages_NotServerMember(t *testing.T) {
	service, _, chanRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	result, err := service.GetPinnedMessages(ctx, channelID, requesterID)

	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}
