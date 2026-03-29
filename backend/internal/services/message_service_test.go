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

func (m *MockMessageRepository) GetReactionUsers(ctx context.Context, messageID uuid.UUID, emoji string, limit int) ([]*models.ReactionUser, error) {
	args := m.Called(ctx, messageID, emoji, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ReactionUser), args.Error(1)
}

func (m *MockMessageRepository) GetUserReactions(ctx context.Context, messageID, userID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, messageID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockMessageRepository) RemoveAllReactions(ctx context.Context, messageID uuid.UUID) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockMessageRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}

func (m *MockMessageRepository) DeleteByAuthor(ctx context.Context, channelID, authorID uuid.UUID, since time.Time) (int, error) {
	args := m.Called(ctx, channelID, authorID, since)
	return args.Int(0), args.Error(1)
}

func (m *MockMessageRepository) BulkDeleteMessages(ctx context.Context, messageIDs []uuid.UUID) error {
	args := m.Called(ctx, messageIDs)
	return args.Error(0)
}

func (m *MockMessageRepository) CountRepliesTo(ctx context.Context, messageID uuid.UUID) (int, error) {
	args := m.Called(ctx, messageID)
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

func (m *MockChannelRepositoryForMessages) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PermissionOverride), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	args := m.Called(ctx, channelID, targetID, targetType)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepositoryForMessages) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockChannelRepositoryForMessages) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

// MockRoleRepositoryForMessages implements RoleRepository for message tests
type MockRoleRepositoryForMessages struct {
	mock.Mock
}

func (m *MockRoleRepositoryForMessages) Create(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForMessages) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForMessages) Update(ctx context.Context, role *models.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error {
	args := m.Called(ctx, serverID, positions)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepositoryForMessages) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleRepositoryForMessages) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleRepositoryForMessages) GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func setupMessageService() (*MessageService, *MockMessageRepository, *MockChannelRepositoryForMessages, *MockServerRepository, *MockRoleRepositoryForMessages, *MockQuotaService, *MockRateLimiter, *MockE2EEService, *MockCacheService, *MockEventBus) {
	msgRepo := new(MockMessageRepository)
	channelRepo := new(MockChannelRepositoryForMessages)
	serverRepo := new(MockServerRepository)
	roleRepo := new(MockRoleRepositoryForMessages)
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
	quotaService := NewQuotaService(quotaConfig, serverRepo, nil, roleRepo, nil)

	service := &MessageService{
		repo:         msgRepo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		roleRepo:     roleRepo,
		quotaService: quotaService,
		rateLimiter:  rateLimiter,
		e2eeService:  e2eeService,
		cache:        cache,
		eventBus:     eventBus,
	}

	return service, msgRepo, channelRepo, serverRepo, roleRepo, mockQuotaService, rateLimiter, e2eeService, cache, eventBus
}

// setupPermissionMocks sets up default mocks for permission checking in server channels.
// Use this for tests that need permission checking to pass with default (permissive) settings.
func setupPermissionMocks(serverRepo *MockServerRepository, roleRepo *MockRoleRepositoryForMessages, serverID uuid.UUID, userID uuid.UUID) {
	// Create a server owned by a different user (so we actually check permissions)
	server := &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(), // Different owner so we check permissions
	}
	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil).Maybe()

	// Return default permissions (which include all basic permissions)
	roleRepo.On("GetMemberPermissions", mock.Anything, serverID, userID).Return(models.DefaultPermissions, nil).Maybe()

	// Return a default role with default permissions
	defaultRole := &models.Role{
		ID:          serverID, // @everyone role has same ID as server
		ServerID:    serverID,
		Permissions: models.DefaultPermissions,
		IsDefault:   true,
	}
	roleRepo.On("GetDefaultRole", mock.Anything, serverID).Return(defaultRole, nil).Maybe()

	// Return member roles (just the default role for simple tests)
	memberRoles := []*models.Role{defaultRole}
	roleRepo.On("GetMemberRoles", mock.Anything, serverID, userID).Return(memberRoles, nil).Maybe()
}

func TestSendMessage_Success(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, rateLimiter, _, _, eventBus := setupMessageService()
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

	// Setup permission mocks for successful message sending
	setupPermissionMocks(serverRepo, roleRepo, serverID, authorID)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, authorID).Return(member, nil)
	rateLimiter.On("Check", ctx, authorID, channelID).Return(nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	channelRepo.On("UpdateLastMessage", ctx, channelID, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("time.Time")).Return(nil)
	eventBus.On("Publish", "message.created", mock.AnythingOfType("*services.MessageCreatedEvent")).Return()

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, "Hello!", message.Content)
	assert.Equal(t, authorID, message.AuthorID)
	assert.Equal(t, channelID, message.ChannelID)
	msgRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSendMessage_ChannelNotFound(t *testing.T) {
	service, _, channelRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	channelID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, message)
}

func TestSendMessage_NotServerMember(t *testing.T) {
	service, _, channelRepo, serverRepo, _, _, _, _, _, _ := setupMessageService()
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

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, message)
}

func TestSendMessage_EmptyMessage(t *testing.T) {
	service, _, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
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

	// Setup permission mocks
	setupPermissionMocks(serverRepo, roleRepo, serverID, authorID)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, authorID).Return(member, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "", nil, nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyMessage, err)
	assert.Nil(t, message)
}

func TestEditMessage_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _, eventBus := setupMessageService()
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
	service, msgRepo, channelRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	otherUserID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   "Original content",
	}

	// DM channel - cannot edit others' messages
	channel := &models.Channel{
		ID:   channelID,
		Type: models.ChannelTypeDM,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	message, err := service.EditMessage(ctx, messageID, otherUserID, "Hacked!")

	assert.Error(t, err)
	assert.Equal(t, ErrNotMessageAuthor, err)
	assert.Nil(t, message)
}

func TestDeleteMessage_ByAuthor(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, _, eventBus := setupMessageService()
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
	service, msgRepo, channelRepo, _, _, _, _, _, _, _ := setupMessageService()
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
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
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

	// Setup permission mocks
	setupPermissionMocks(serverRepo, roleRepo, serverID, requesterID)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return(expectedMessages, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 0)

	assert.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestAddReaction_Success(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
	}

	// DM channel - no permission checks needed beyond being a participant
	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{userID, uuid.New()},
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	msgRepo.On("AddReaction", ctx, messageID, userID, "👍").Return(nil)
	eventBus.On("Publish", "reaction.added", mock.AnythingOfType("*services.ReactionAddedEvent")).Return()

	err := service.AddReaction(ctx, messageID, userID, "👍")

	assert.NoError(t, err)
	msgRepo.AssertExpectations(t)
}

func TestRemoveReaction_Success(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(&models.Message{ID: messageID, ChannelID: channelID}, nil)
	msgRepo.On("RemoveReaction", ctx, messageID, userID, "👍").Return(nil)
	eventBus.On("Publish", "reaction.removed", mock.AnythingOfType("*services.ReactionRemovedEvent")).Return()

	// Pass nil for targetUserID to remove own reaction
	err := service.RemoveReaction(ctx, messageID, userID, "👍", nil)

	assert.NoError(t, err)
}

func TestPinMessage_Success(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, roleRepo, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    false,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Setup permission mocks (user has MANAGE_MESSAGES)
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", mock.Anything, serverID, requesterID).Return(models.PermManageMessages, nil)
	roleRepo.On("GetDefaultRole", mock.Anything, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.pinned", mock.AnythingOfType("*services.MessagePinnedEvent")).Return()

	err := service.PinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
}

func TestUnpinMessage_Success(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, roleRepo, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    true,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Setup permission mocks (user has MANAGE_MESSAGES)
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", mock.Anything, serverID, requesterID).Return(models.PermManageMessages, nil)
	roleRepo.On("GetDefaultRole", mock.Anything, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.unpinned", mock.AnythingOfType("*services.MessageUnpinnedEvent")).Return()

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
	assert.False(t, existingMessage.Pinned)
}

func TestUnpinMessage_NotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.Equal(t, ErrMessageNotFound, err)
}

func TestUnpinMessage_AlreadyUnpinned(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    false, // Already unpinned
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Setup permission mocks (user has MANAGE_MESSAGES)
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", mock.Anything, serverID, requesterID).Return(models.PermManageMessages, nil)
	roleRepo.On("GetDefaultRole", mock.Anything, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)
	// No Update or Publish calls expected - it's a no-op

	err := service.UnpinMessage(ctx, messageID, requesterID)

	assert.NoError(t, err)
}

func TestGetPinnedMessages_Success(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
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

	// Setup permission mocks
	setupPermissionMocks(serverRepo, roleRepo, serverID, requesterID)

	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)
	msgRepo.On("GetPinnedMessages", ctx, channelID).Return(pinnedMessages, nil)

	result, err := service.GetPinnedMessages(ctx, channelID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetPinnedMessages_ChannelNotFound(t *testing.T) {
	service, _, chanRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()

	chanRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	result, err := service.GetPinnedMessages(ctx, channelID, requesterID)

	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, result)
}

func TestGetPinnedMessages_NotServerMember(t *testing.T) {
	service, _, chanRepo, serverRepo, _, _, _, _, _, _ := setupMessageService()
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

// ============================================
// Permission Tests
// ============================================

func TestSendMessage_MissingSendMessagesPermission(t *testing.T) {
	service, _, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
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

	// Setup: user has no SEND_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, authorID).Return(int64(0), nil) // No permissions
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, authorID).Return(member, nil)

	message, err := service.SendMessage(ctx, authorID, channelID, "Hello!", nil, nil, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingSendMessages, err)
	assert.Nil(t, message)
}

func TestSendMessage_ServerOwnerBypassesPermissions(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, rateLimiter, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	ownerID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   ownerID,
		ServerID: serverID,
	}

	// Setup: user is server owner (even with no explicit permissions)
	server := &models.Server{ID: serverID, OwnerID: ownerID}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	// Owner bypasses permission checks, but quota service still calls GetMemberRoles
	roleRepo.On("GetMemberRoles", ctx, serverID, ownerID).Return([]*models.Role{}, nil)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, ownerID).Return(member, nil)
	rateLimiter.On("Check", ctx, ownerID, channelID).Return(nil)
	msgRepo.On("Create", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	channelRepo.On("UpdateLastMessage", ctx, channelID, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("time.Time")).Return(nil)
	eventBus.On("Publish", "message.created", mock.AnythingOfType("*services.MessageCreatedEvent")).Return()

	message, err := service.SendMessage(ctx, ownerID, channelID, "Hello!", nil, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, message)
}

func TestDeleteMessage_MissingManageMessagesPermission(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	requesterID := uuid.New() // Different user
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	// Setup: requester has no MANAGE_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	err := service.DeleteMessage(ctx, messageID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingManageMessages, err)
}

func TestAddReaction_MissingAddReactionsPermission(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}

	// Setup: user has no ADD_REACTIONS permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, userID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)

	err := service.AddReaction(ctx, messageID, userID, "👍")

	assert.Error(t, err)
	assert.Equal(t, ErrMissingAddReactions, err)
}

func TestGetMessages_MissingReadMessageHistoryPermission(t *testing.T) {
	service, _, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
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

	// Setup: user has no READ_MESSAGE_HISTORY permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 50)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingReadMessages, err)
	assert.Nil(t, messages)
}

func TestPinMessage_MissingManageMessagesPermission(t *testing.T) {
	service, msgRepo, chanRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Pinned:    false,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Setup: user has no MANAGE_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", mock.Anything, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", mock.Anything, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", mock.Anything, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	chanRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{}, nil)

	err := service.PinMessage(ctx, messageID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingManageMessages, err)
}

func TestRemoveOthersReaction_MissingManageMessagesPermission(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	targetUserID := uuid.New() // Different user's reaction
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	// Setup: requester has no MANAGE_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	err := service.RemoveReaction(ctx, messageID, requesterID, "👍", &targetUserID)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingManageMessages, err)
}

func TestRemoveOthersReaction_WithManageMessagesPermission(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	targetUserID := uuid.New() // Different user's reaction
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	// Setup: requester HAS MANAGE_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(models.PermManageMessages, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	msgRepo.On("RemoveReaction", ctx, messageID, targetUserID, "👍").Return(nil)
	eventBus.On("Publish", "reaction.removed", mock.AnythingOfType("*services.ReactionRemovedEvent")).Return()

	err := service.RemoveReaction(ctx, messageID, requesterID, "👍", &targetUserID)

	assert.NoError(t, err)
}

func TestBulkDeleteMessages_Success(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, roleRepo, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	messageIDs := []uuid.UUID{uuid.New(), uuid.New()}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	// Setup: requester has MANAGE_MESSAGES permission
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(models.PermManageMessages, nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{UserID: requesterID, ServerID: serverID}, nil)

	// Each message is valid (recent and in correct channel)
	for _, msgID := range messageIDs {
		msgRepo.On("GetByID", ctx, msgID).Return(&models.Message{
			ID:        msgID,
			ChannelID: channelID,
			CreatedAt: time.Now().Add(-1 * time.Hour), // Recent message
		}, nil)
	}

	msgRepo.On("BulkDeleteMessages", ctx, messageIDs).Return(nil)
	eventBus.On("Publish", "message.deleted", mock.AnythingOfType("*services.MessageDeletedEvent")).Return().Times(2)

	err := service.BulkDeleteMessages(ctx, channelID, messageIDs, requesterID)

	assert.NoError(t, err)
	msgRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestBulkDeleteMessages_NotServerMember(t *testing.T) {
	service, _, channelRepo, serverRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	messageIDs := []uuid.UUID{uuid.New()}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.BulkDeleteMessages(ctx, channelID, messageIDs, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
}

func TestBulkDeleteMessages_MissingPermission(t *testing.T) {
	service, _, channelRepo, serverRepo, roleRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	messageIDs := []uuid.UUID{uuid.New()}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	// Setup: requester has NO permissions
	server := &models.Server{ID: serverID, OwnerID: uuid.New()}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberPermissions", ctx, serverID, requesterID).Return(int64(0), nil)
	roleRepo.On("GetDefaultRole", ctx, serverID).Return(&models.Role{Permissions: 0}, nil)

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(&models.Member{UserID: requesterID, ServerID: serverID}, nil)

	err := service.BulkDeleteMessages(ctx, channelID, messageIDs, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingManageMessages, err)
}

func TestBulkDeleteMessages_ExceedsLimit(t *testing.T) {
	service, _, _, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()

	// Create 101 message IDs (exceeds limit of 100)
	messageIDs := make([]uuid.UUID, 101)
	for i := range messageIDs {
		messageIDs[i] = uuid.New()
	}

	err := service.BulkDeleteMessages(ctx, channelID, messageIDs, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrBulkDeleteLimit, err)
}

func TestBulkDeleteMessages_DMChannel(t *testing.T) {
	service, _, channelRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()

	messageIDs := []uuid.UUID{uuid.New()}

	// DM channel has no ServerID
	channel := &models.Channel{
		ID:       channelID,
		ServerID: nil,
		Type:     models.ChannelTypeDM,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	err := service.BulkDeleteMessages(ctx, channelID, messageIDs, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNoPermission, err)
}
