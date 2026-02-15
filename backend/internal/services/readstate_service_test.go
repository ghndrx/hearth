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

// MockReadStateRepository is a mock implementation of ReadStateRepository
type MockReadStateRepository struct {
	mock.Mock
}

func (m *MockReadStateRepository) GetReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error) {
	args := m.Called(ctx, userID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReadState), args.Error(1)
}

func (m *MockReadStateRepository) SetReadState(ctx context.Context, userID, channelID uuid.UUID, lastMessageID *uuid.UUID) error {
	args := m.Called(ctx, userID, channelID, lastMessageID)
	return args.Error(0)
}

func (m *MockReadStateRepository) IncrementMentionCount(ctx context.Context, userID, channelID uuid.UUID) error {
	args := m.Called(ctx, userID, channelID)
	return args.Error(0)
}

func (m *MockReadStateRepository) GetUnreadCount(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockReadStateRepository) GetUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error) {
	args := m.Called(ctx, userID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChannelUnreadInfo), args.Error(1)
}

func (m *MockReadStateRepository) GetUserUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UnreadSummary), args.Error(1)
}

func (m *MockReadStateRepository) GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error) {
	args := m.Called(ctx, userID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UnreadSummary), args.Error(1)
}

func (m *MockReadStateRepository) MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error {
	args := m.Called(ctx, userID, serverID)
	return args.Error(0)
}

func (m *MockReadStateRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}

func (m *MockReadStateRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockChannelRepository for testing
type MockChannelRepositoryForReadState struct {
	mock.Mock
}

func (m *MockChannelRepositoryForReadState) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForReadState) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForReadState) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForReadState) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepositoryForReadState) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForReadState) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForReadState) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForReadState) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func TestReadStateService_MarkChannelAsRead(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()
	lastMsgID := uuid.New()

	t.Run("marks channel as read with channel's last message", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		channel := &models.Channel{
			ID:            channelID,
			LastMessageID: &lastMsgID,
		}
		mockChannelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
		mockRepo.On("SetReadState", ctx, userID, channelID, &lastMsgID).Return(nil)

		ack, err := svc.MarkChannelAsRead(ctx, userID, channelID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, ack)
		assert.Equal(t, channelID, ack.ChannelID)
		assert.Equal(t, &lastMsgID, ack.LastMessageID)
		assert.Equal(t, 0, ack.MentionCount)
		mockRepo.AssertExpectations(t)
		mockChannelRepo.AssertExpectations(t)
	})

	t.Run("marks channel as read with specific message", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		specificMsgID := uuid.New()
		channel := &models.Channel{
			ID:            channelID,
			LastMessageID: &lastMsgID,
		}
		mockChannelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
		mockRepo.On("SetReadState", ctx, userID, channelID, &specificMsgID).Return(nil)

		ack, err := svc.MarkChannelAsRead(ctx, userID, channelID, &specificMsgID)

		assert.NoError(t, err)
		assert.NotNil(t, ack)
		assert.Equal(t, &specificMsgID, ack.LastMessageID)
		mockRepo.AssertExpectations(t)
		mockChannelRepo.AssertExpectations(t)
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockChannelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

		ack, err := svc.MarkChannelAsRead(ctx, userID, channelID, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrChannelNotFound, err)
		assert.Nil(t, ack)
		mockChannelRepo.AssertExpectations(t)
	})
}

func TestReadStateService_GetChannelUnreadInfo(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	t.Run("returns unread info for channel", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		channel := &models.Channel{ID: channelID}
		unreadInfo := &models.ChannelUnreadInfo{
			ChannelID:    channelID,
			UnreadCount:  5,
			MentionCount: 2,
			HasUnread:    true,
		}

		mockChannelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
		mockRepo.On("GetUnreadInfo", ctx, userID, channelID).Return(unreadInfo, nil)

		info, err := svc.GetChannelUnreadInfo(ctx, userID, channelID)

		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, 5, info.UnreadCount)
		assert.Equal(t, 2, info.MentionCount)
		assert.True(t, info.HasUnread)
		mockRepo.AssertExpectations(t)
		mockChannelRepo.AssertExpectations(t)
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockChannelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

		info, err := svc.GetChannelUnreadInfo(ctx, userID, channelID)

		assert.Error(t, err)
		assert.Equal(t, ErrChannelNotFound, err)
		assert.Nil(t, info)
		mockChannelRepo.AssertExpectations(t)
	})
}

func TestReadStateService_GetUnreadSummary(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("returns unread summary", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		summary := &models.UnreadSummary{
			TotalUnread:   10,
			TotalMentions: 3,
			Channels: []models.ChannelUnreadInfo{
				{ChannelID: uuid.New(), UnreadCount: 5, MentionCount: 2, HasUnread: true},
				{ChannelID: uuid.New(), UnreadCount: 5, MentionCount: 1, HasUnread: true},
			},
		}

		mockRepo.On("GetUserUnreadSummary", ctx, userID).Return(summary, nil)

		result, err := svc.GetUnreadSummary(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 10, result.TotalUnread)
		assert.Equal(t, 3, result.TotalMentions)
		assert.Len(t, result.Channels, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestReadStateService_GetServerUnreadSummary(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	t.Run("returns server unread summary", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		summary := &models.UnreadSummary{
			TotalUnread:   15,
			TotalMentions: 5,
			Channels: []models.ChannelUnreadInfo{
				{ChannelID: uuid.New(), UnreadCount: 10, MentionCount: 3, HasUnread: true},
				{ChannelID: uuid.New(), UnreadCount: 5, MentionCount: 2, HasUnread: true},
			},
		}

		mockRepo.On("GetServerUnreadSummary", ctx, userID, serverID).Return(summary, nil)

		result, err := svc.GetServerUnreadSummary(ctx, userID, serverID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 15, result.TotalUnread)
		assert.Equal(t, 5, result.TotalMentions)
		assert.Len(t, result.Channels, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestReadStateService_MarkServerAsRead(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	t.Run("marks all server channels as read", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockRepo.On("MarkServerAsRead", ctx, userID, serverID).Return(nil)

		err := svc.MarkServerAsRead(ctx, userID, serverID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestReadStateService_IncrementMention(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	channelID := uuid.New()

	t.Run("increments mention count", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockRepo.On("IncrementMentionCount", ctx, userID, channelID).Return(nil)

		err := svc.IncrementMention(ctx, userID, channelID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestReadStateService_OnChannelDeleted(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()

	t.Run("cleans up read states when channel deleted", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockRepo.On("DeleteByChannel", ctx, channelID).Return(nil)

		err := svc.OnChannelDeleted(ctx, channelID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestReadStateService_OnUserDeleted(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("cleans up read states when user deleted", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockChannelRepo := new(MockChannelRepositoryForReadState)
		svc := NewReadStateService(mockRepo, mockChannelRepo)

		mockRepo.On("DeleteByUser", ctx, userID).Return(nil)

		err := svc.OnUserDeleted(ctx, userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
