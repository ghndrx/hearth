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

// MockThreadRepository mocks the ThreadRepository interface
type MockThreadRepository struct {
	mock.Mock
}

func (m *MockThreadRepository) Create(ctx context.Context, thread *models.Thread) error {
	args := m.Called(ctx, thread)
	return args.Error(0)
}

func (m *MockThreadRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Thread), args.Error(1)
}

func (m *MockThreadRepository) Update(ctx context.Context, thread *models.Thread) error {
	args := m.Called(ctx, thread)
	return args.Error(0)
}

func (m *MockThreadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockThreadRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Thread), args.Error(1)
}

func (m *MockThreadRepository) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Thread), args.Error(1)
}

func (m *MockThreadRepository) Archive(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockThreadRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockThreadRepository) AddMember(ctx context.Context, threadID, userID uuid.UUID) error {
	args := m.Called(ctx, threadID, userID)
	return args.Error(0)
}

func (m *MockThreadRepository) RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error {
	args := m.Called(ctx, threadID, userID)
	return args.Error(0)
}

func (m *MockThreadRepository) IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, threadID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockThreadRepository) GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, threadID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockThreadRepository) CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	args := m.Called(ctx, threadID, authorID, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadMessage), args.Error(1)
}

func (m *MockThreadRepository) GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	args := m.Called(ctx, threadID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ThreadMessage), args.Error(1)
}

func (m *MockThreadRepository) IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error {
	args := m.Called(ctx, threadID)
	return args.Error(0)
}

// MockChannelRepositoryForThread mocks ChannelRepository for thread tests
type MockChannelRepositoryForThread struct {
	mock.Mock
}

func (m *MockChannelRepositoryForThread) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForThread) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForThread) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForThread) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepositoryForThread) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForThread) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForThread) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForThread) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func setupThreadService() (*ThreadService, *MockThreadRepository, *MockChannelRepositoryForThread, *MockServerRepository, *MockEventBus) {
	threadRepo := new(MockThreadRepository)
	channelRepo := new(MockChannelRepositoryForThread)
	serverRepo := new(MockServerRepository)
	eventBus := new(MockEventBus)
	service := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)
	return service, threadRepo, channelRepo, serverRepo, eventBus
}

func TestThreadService_CreateThread_Success(t *testing.T) {
	service, threadRepo, channelRepo, _, eventBus := setupThreadService()
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()

	channel := &models.Channel{
		ID:   channelID,
		Type: models.ChannelTypeText,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	threadRepo.On("Create", ctx, mock.AnythingOfType("*models.Thread")).Return(nil)
	eventBus.On("Publish", "thread.created", mock.Anything).Return()

	thread, err := service.CreateThread(ctx, channelID, ownerID, "Test Thread", nil)

	assert.NoError(t, err)
	assert.NotNil(t, thread)
	assert.Equal(t, "Test Thread", thread.Name)
	threadRepo.AssertExpectations(t)
}

func TestThreadService_GetThread_Success(t *testing.T) {
	service, threadRepo, _, _, _ := setupThreadService()
	ctx := context.Background()
	threadID := uuid.New()

	expectedThread := &models.Thread{
		ID:   threadID,
		Name: "Test Thread",
	}

	threadRepo.On("GetByID", ctx, threadID).Return(expectedThread, nil)

	thread, err := service.GetThread(ctx, threadID)

	assert.NoError(t, err)
	assert.Equal(t, expectedThread.Name, thread.Name)
}

func TestThreadService_GetThread_NotFound(t *testing.T) {
	service, threadRepo, _, _, _ := setupThreadService()
	ctx := context.Background()
	threadID := uuid.New()

	threadRepo.On("GetByID", ctx, threadID).Return(nil, ErrThreadNotFound)

	thread, err := service.GetThread(ctx, threadID)

	assert.Error(t, err)
	assert.Nil(t, thread)
}

func TestThreadService_ArchiveThread_Success(t *testing.T) {
	service, threadRepo, _, _, eventBus := setupThreadService()
	ctx := context.Background()
	threadID := uuid.New()
	userID := uuid.New()

	thread := &models.Thread{
		ID:      threadID,
		OwnerID: userID,
	}

	threadRepo.On("GetByID", ctx, threadID).Return(thread, nil)
	threadRepo.On("Archive", ctx, threadID).Return(nil)
	eventBus.On("Publish", "thread.archived", mock.Anything).Return()

	err := service.ArchiveThread(ctx, threadID, userID)

	assert.NoError(t, err)
	threadRepo.AssertExpectations(t)
}
