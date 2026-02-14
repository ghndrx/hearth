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

// MockWebhookRepository is a mock implementation of WebhookRepository
type MockWebhookRepository struct {
	mock.Mock
}

func (m *MockWebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockWebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockWebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWebhookRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).(int), args.Error(1)
}

// MockChannelRepositoryForWebhook is a mock implementation of ChannelRepository for webhook tests
type MockChannelRepositoryForWebhook struct {
	mock.Mock
}

func (m *MockChannelRepositoryForWebhook) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForWebhook) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForWebhook) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForWebhook) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepositoryForWebhook) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForWebhook) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForWebhook) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForWebhook) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

// MockServerRepoForWebhook is a mock implementation of ServerRepository for webhook tests
type MockServerRepoForWebhook struct {
	mock.Mock
}

func (m *MockServerRepoForWebhook) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepoForWebhook) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepoForWebhook) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepoForWebhook) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepoForWebhook) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepoForWebhook) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepoForWebhook) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockServerRepoForWebhook) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockServerRepoForWebhook) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepoForWebhook) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepoForWebhook) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepoForWebhook) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

// Helper function to create a WebhookService with mocks
func newTestWebhookService() (*WebhookService, *MockWebhookRepository, *MockChannelRepositoryForWebhook, *MockServerRepoForWebhook, *MockEventBus) {
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepositoryForWebhook)
	serverRepo := new(MockServerRepoForWebhook)
	eventBus := new(MockEventBus)

	service := NewWebhookService(webhookRepo, channelRepo, serverRepo, eventBus)

	return service, webhookRepo, channelRepo, serverRepo, eventBus
}

// ============================================================================
// CreateWebhook Tests
// ============================================================================

func TestWebhookService_CreateWebhook_Success(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	webhookRepo.On("CountByChannelID", ctx, channelID).Return(2, nil)
	webhookRepo.On("Create", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)
	eventBus.On("Publish", "webhook.created", mock.AnythingOfType("*services.WebhookCreatedEvent")).Return()

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
		Avatar:    nil,
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, "Test Webhook", webhook.Name)
	assert.Equal(t, channelID, webhook.ChannelID)
	assert.Equal(t, serverID, *webhook.ServerID)
	assert.Equal(t, creatorID, *webhook.CreatorID)
	assert.NotEmpty(t, webhook.Token)
	assert.Equal(t, models.WebhookTypeIncoming, webhook.Type)

	channelRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
	webhookRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestWebhookService_CreateWebhook_ChannelNotFound(t *testing.T) {
	service, _, channelRepo, _, _ := newTestWebhookService()
	ctx := context.Background()

	channelID := uuid.New()
	creatorID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, errors.New("not found"))

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_NotServerMember(t *testing.T) {
	service, _, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, nil)

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_NameTooLong(t *testing.T) {
	service, _, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      string(make([]byte, 81)), // 81 characters
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNameTooLong, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_EmptyName(t *testing.T) {
	service, _, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "",
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNameTooLong, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_TooManyWebhooks(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	webhookRepo.On("CountByChannelID", ctx, channelID).Return(10, nil) // Max limit reached

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, ErrTooManyWebhooks, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_RepositoryError(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	webhookRepo.On("CountByChannelID", ctx, channelID).Return(2, nil)
	webhookRepo.On("Create", ctx, mock.AnythingOfType("*models.Webhook")).Return(errors.New("database error"))

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, webhook)
}

func TestWebhookService_CreateWebhook_WithAvatar(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()
	avatarURL := "https://example.com/avatar.png"

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   creatorID,
		JoinedAt: time.Now(),
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	webhookRepo.On("CountByChannelID", ctx, channelID).Return(2, nil)
	webhookRepo.On("Create", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)
	eventBus.On("Publish", "webhook.created", mock.AnythingOfType("*services.WebhookCreatedEvent")).Return()

	req := &CreateWebhookRequest{
		ChannelID: channelID,
		CreatorID: creatorID,
		Name:      "Test Webhook",
		Avatar:    &avatarURL,
	}

	webhook, err := service.CreateWebhook(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.NotNil(t, webhook.Avatar)
	assert.Equal(t, avatarURL, *webhook.Avatar)
}

// ============================================================================
// GetWebhook Tests
// ============================================================================

func TestWebhookService_GetWebhook_Success(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)

	result, err := service.GetWebhook(ctx, webhookID, requesterID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, webhookID, result.ID)
	assert.Equal(t, "Test Webhook", result.Name)
}

func TestWebhookService_GetWebhook_NotFound(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	webhookID := uuid.New()
	requesterID := uuid.New()

	webhookRepo.On("GetByID", ctx, webhookID).Return(nil, errors.New("not found"))

	result, err := service.GetWebhook(ctx, webhookID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNotFound, err)
	assert.Nil(t, result)
}

func TestWebhookService_GetWebhook_NotServerMember(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	result, err := service.GetWebhook(ctx, webhookID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}

// ============================================================================
// GetChannelWebhooks Tests
// ============================================================================

func TestWebhookService_GetChannelWebhooks_Success(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	requesterID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhooks := []*models.Webhook{
		{ID: uuid.New(), ChannelID: channelID, Name: "Webhook 1"},
		{ID: uuid.New(), ChannelID: channelID, Name: "Webhook 2"},
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	webhookRepo.On("GetByChannelID", ctx, channelID).Return(webhooks, nil)

	result, err := service.GetChannelWebhooks(ctx, channelID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestWebhookService_GetChannelWebhooks_ChannelNotFound(t *testing.T) {
	service, _, channelRepo, _, _ := newTestWebhookService()
	ctx := context.Background()

	channelID := uuid.New()
	requesterID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, errors.New("not found"))

	result, err := service.GetChannelWebhooks(ctx, channelID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, result)
}

// ============================================================================
// GetServerWebhooks Tests
// ============================================================================

func TestWebhookService_GetServerWebhooks_Success(t *testing.T) {
	service, webhookRepo, _, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhooks := []*models.Webhook{
		{ID: uuid.New(), ServerID: &serverID, Name: "Webhook 1"},
		{ID: uuid.New(), ServerID: &serverID, Name: "Webhook 2"},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	webhookRepo.On("GetByServerID", ctx, serverID).Return(webhooks, nil)

	result, err := service.GetServerWebhooks(ctx, serverID, requesterID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestWebhookService_GetServerWebhooks_NotMember(t *testing.T) {
	service, _, _, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	result, err := service.GetServerWebhooks(ctx, serverID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}

// ============================================================================
// UpdateWebhook Tests
// ============================================================================

func TestWebhookService_UpdateWebhook_Success(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()
	newName := "Updated Webhook"

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	webhookRepo.On("Update", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)
	eventBus.On("Publish", "webhook.updated", mock.AnythingOfType("*services.WebhookUpdatedEvent")).Return()

	req := &UpdateWebhookRequest{
		Name: &newName,
	}

	result, err := service.UpdateWebhook(ctx, webhookID, requesterID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
}

func TestWebhookService_UpdateWebhook_NotFound(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	webhookID := uuid.New()
	requesterID := uuid.New()
	newName := "Updated Webhook"

	webhookRepo.On("GetByID", ctx, webhookID).Return(nil, errors.New("not found"))

	req := &UpdateWebhookRequest{
		Name: &newName,
	}

	result, err := service.UpdateWebhook(ctx, webhookID, requesterID, req)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNotFound, err)
	assert.Nil(t, result)
}

func TestWebhookService_UpdateWebhook_InvalidName(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()
	invalidName := string(make([]byte, 81))

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)

	req := &UpdateWebhookRequest{
		Name: &invalidName,
	}

	result, err := service.UpdateWebhook(ctx, webhookID, requesterID, req)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNameTooLong, err)
	assert.Nil(t, result)
}

func TestWebhookService_UpdateWebhook_MoveToDifferentChannel(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	newChannelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	newChannel := &models.Channel{
		ID:       newChannelID,
		ServerID: &serverID,
		Name:     "other-channel",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByID", ctx, newChannelID).Return(newChannel, nil)
	webhookRepo.On("Update", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)
	eventBus.On("Publish", "webhook.updated", mock.AnythingOfType("*services.WebhookUpdatedEvent")).Return()

	req := &UpdateWebhookRequest{
		ChannelID: &newChannelID,
	}

	result, err := service.UpdateWebhook(ctx, webhookID, requesterID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newChannelID, result.ChannelID)
}

// ============================================================================
// DeleteWebhook Tests
// ============================================================================

func TestWebhookService_DeleteWebhook_Success(t *testing.T) {
	service, webhookRepo, channelRepo, serverRepo, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	requesterID := uuid.New()

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Name:     "general",
	}

	member := &models.Member{
		ServerID: serverID,
		UserID:   requesterID,
		JoinedAt: time.Now(),
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	webhookRepo.On("Delete", ctx, webhookID).Return(nil)
	eventBus.On("Publish", "webhook.deleted", mock.AnythingOfType("*services.WebhookDeletedEvent")).Return()

	err := service.DeleteWebhook(ctx, webhookID, requesterID)

	assert.NoError(t, err)
	webhookRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestWebhookService_DeleteWebhook_NotFound(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	webhookID := uuid.New()
	requesterID := uuid.New()

	webhookRepo.On("GetByID", ctx, webhookID).Return(nil, errors.New("not found"))

	err := service.DeleteWebhook(ctx, webhookID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNotFound, err)
}

// ============================================================================
// ExecuteWebhook Tests
// ============================================================================

func TestWebhookService_ExecuteWebhook_Success(t *testing.T) {
	service, webhookRepo, _, _, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()

	token, _ := generateWebhookToken()
	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
		Token:     token,
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	eventBus.On("Publish", "webhook.executed", mock.AnythingOfType("*services.WebhookExecutedEvent")).Return()

	req := &ExecuteWebhookRequest{
		Content: "Hello from webhook!",
	}

	message, err := service.ExecuteWebhook(ctx, webhookID, token, req)

	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, "Hello from webhook!", message.Content)
	assert.Equal(t, channelID, message.ChannelID)
	assert.Equal(t, "Test Webhook", message.Author.Username)
}

func TestWebhookService_ExecuteWebhook_NotFound(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	webhookID := uuid.New()
	token := "some-token"

	webhookRepo.On("GetByID", ctx, webhookID).Return(nil, errors.New("not found"))

	req := &ExecuteWebhookRequest{
		Content: "Hello from webhook!",
	}

	message, err := service.ExecuteWebhook(ctx, webhookID, token, req)

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookNotFound, err)
	assert.Nil(t, message)
}

func TestWebhookService_ExecuteWebhook_InvalidToken(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
		Token:     "valid-token",
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)

	req := &ExecuteWebhookRequest{
		Content: "Hello from webhook!",
	}

	message, err := service.ExecuteWebhook(ctx, webhookID, "invalid-token", req)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidWebhookToken, err)
	assert.Nil(t, message)
}

func TestWebhookService_ExecuteWebhook_EmptyContent(t *testing.T) {
	service, webhookRepo, _, _, _ := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()

	token, _ := generateWebhookToken()
	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
		Token:     token,
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)

	req := &ExecuteWebhookRequest{
		Content: "",
	}

	message, err := service.ExecuteWebhook(ctx, webhookID, token, req)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyMessage, err)
	assert.Nil(t, message)
}

func TestWebhookService_ExecuteWebhook_WithCustomUsername(t *testing.T) {
	service, webhookRepo, _, _, eventBus := newTestWebhookService()
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	customUsername := "Custom Bot"

	token, _ := generateWebhookToken()
	webhook := &models.Webhook{
		ID:        webhookID,
		ChannelID: channelID,
		ServerID:  &serverID,
		Name:      "Test Webhook",
		Token:     token,
	}

	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	eventBus.On("Publish", "webhook.executed", mock.AnythingOfType("*services.WebhookExecutedEvent")).Return()

	req := &ExecuteWebhookRequest{
		Content:  "Hello from webhook!",
		Username: &customUsername,
	}

	message, err := service.ExecuteWebhook(ctx, webhookID, token, req)

	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, customUsername, message.Author.Username)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestGenerateWebhookToken(t *testing.T) {
	tokens := make(map[string]bool)

	// Generate multiple tokens and verify uniqueness
	for i := 0; i < 100; i++ {
		token, err := generateWebhookToken()
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Len(t, token, 64) // hex encoded 32 bytes = 64 characters

		// Check uniqueness
		assert.False(t, tokens[token], "Generated duplicate token")
		tokens[token] = true
	}
}
