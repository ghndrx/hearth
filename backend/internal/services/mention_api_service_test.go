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

// MockMentionRepository is a mock implementation of MentionRepository
type MockMentionRepository struct {
	mock.Mock
}

func (m *MockMentionRepository) Create(ctx context.Context, mention *models.Mention) error {
	args := m.Called(ctx, mention)
	return args.Error(0)
}

func (m *MockMentionRepository) CreateBatch(ctx context.Context, mentions []*models.Mention) error {
	args := m.Called(ctx, mentions)
	return args.Error(0)
}

func (m *MockMentionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Mention), args.Error(1)
}

func (m *MockMentionRepository) GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]models.MentionWithContext), args.Int(1), args.Error(2)
}

func (m *MockMentionRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockMentionRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MentionStats), args.Error(1)
}

func (m *MockMentionRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMentionRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockMentionRepository) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockMentionRepository) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockMentionRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}

func (m *MockMentionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockMentionRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	args := m.Called(ctx, userID, query, limit)
	return args.Get(0).([]models.MentionWithContext), args.Error(1)
}

func TestMentionAPIService_GetMentions(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	filter := &models.MentionFilter{
		UserID: userID,
		Limit:  50,
	}

	expectedMentions := []models.MentionWithContext{
		{
			Mention: models.Mention{
				ID:          uuid.New(),
				UserID:      userID,
				MessageID:   uuid.New(),
				MentionedBy: uuid.New(),
				ChannelID:   uuid.New(),
				MentionType: models.MentionKindUser,
				CreatedAt:   time.Now(),
			},
			AuthorName: "testuser",
			Preview:    "Hello @testuser!",
		},
	}

	mockRepo.On("GetMentionsWithContext", ctx, filter).Return(expectedMentions, 1, nil)

	mentions, total, err := service.GetMentions(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, mentions, 1)
	assert.Equal(t, "testuser", mentions[0].AuthorName)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_GetUnreadCount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	expectedCount := 5

	mockRepo.On("GetUnreadCount", ctx, userID).Return(expectedCount, nil)

	count, err := service.GetUnreadCount(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_GetStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	expectedStats := &models.MentionStats{
		TotalCount:  100,
		UnreadCount: 5,
		TodayCount:  3,
	}

	mockRepo.On("GetStats", ctx, userID).Return(expectedStats, nil)

	stats, err := service.GetStats(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, 100, stats.TotalCount)
	assert.Equal(t, 5, stats.UnreadCount)
	assert.Equal(t, 3, stats.TodayCount)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_MarkAsRead(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	mentionID := uuid.New()

	// Mock the GetByID call
	mention := &models.Mention{
		ID:     mentionID,
		UserID: userID,
	}
	mockRepo.On("GetByID", ctx, mentionID).Return(mention, nil)
	mockRepo.On("MarkAsRead", ctx, mentionID, userID).Return(nil)

	err := service.MarkAsRead(ctx, mentionID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_MarkAsRead_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	mentionID := uuid.New()

	// Mock not found
	mockRepo.On("GetByID", ctx, mentionID).Return(nil, nil)

	err := service.MarkAsRead(ctx, mentionID, userID)

	assert.Equal(t, ErrMentionNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_MarkAsRead_WrongUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()
	mentionID := uuid.New()

	// Mock mention belongs to different user
	mention := &models.Mention{
		ID:     mentionID,
		UserID: otherUserID,
	}
	mockRepo.On("GetByID", ctx, mentionID).Return(mention, nil)

	err := service.MarkAsRead(ctx, mentionID, userID)

	assert.Equal(t, ErrMentionNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_MarkAllAsRead(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	expectedCount := 10

	mockRepo.On("MarkAllAsRead", ctx, userID).Return(expectedCount, nil)

	count, err := service.MarkAllAsRead(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_MarkChannelMentionsAsRead(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	channelID := uuid.New()
	expectedCount := 3

	mockRepo.On("MarkChannelMentionsAsRead", ctx, userID, channelID).Return(expectedCount, nil)

	count, err := service.MarkChannelMentionsAsRead(ctx, userID, channelID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_Search(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	userID := uuid.New()
	query := "hello"
	limit := 20

	expectedMentions := []models.MentionWithContext{
		{
			Mention: models.Mention{
				ID:          uuid.New(),
				UserID:      userID,
				MentionType: models.MentionKindUser,
			},
			Preview: "hello world",
		},
	}

	mockRepo.On("Search", ctx, userID, query, limit).Return(expectedMentions, nil)

	results, err := service.Search(ctx, userID, query, limit)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Preview, "hello")
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_CreateMention(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	req := &models.CreateMentionRequest{
		UserID:      uuid.New(),
		MessageID:   uuid.New(),
		MentionedBy: uuid.New(),
		ChannelID:   uuid.New(),
		MentionType: models.MentionKindUser,
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Mention")).Return(nil)

	err := service.CreateMention(ctx, req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_CreateMentionsBatch(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	mentions := []*models.Mention{
		{
			UserID:      uuid.New(),
			MessageID:   uuid.New(),
			MentionedBy: uuid.New(),
			ChannelID:   uuid.New(),
			MentionType: models.MentionKindUser,
		},
		{
			UserID:      uuid.New(),
			MessageID:   uuid.New(),
			MentionedBy: uuid.New(),
			ChannelID:   uuid.New(),
			MentionType: models.MentionKindRole,
		},
	}

	mockRepo.On("CreateBatch", ctx, mentions).Return(nil)

	err := service.CreateMentionsBatch(ctx, mentions)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionAPIService_DeleteByMessage(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMentionRepository)
	service := NewMentionAPIService(mockRepo, nil)

	messageID := uuid.New()

	mockRepo.On("DeleteByMessage", ctx, messageID).Return(nil)

	err := service.DeleteByMessage(ctx, messageID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMentionFilter_SetDefaults(t *testing.T) {
	t.Run("sets default limit", func(t *testing.T) {
		filter := &models.MentionFilter{}
		filter.SetDefaults()
		assert.Equal(t, 50, filter.Limit)
	})

	t.Run("caps limit at 100", func(t *testing.T) {
		filter := &models.MentionFilter{Limit: 200}
		filter.SetDefaults()
		assert.Equal(t, 100, filter.Limit)
	})

	t.Run("preserves valid limit", func(t *testing.T) {
		filter := &models.MentionFilter{Limit: 25}
		filter.SetDefaults()
		assert.Equal(t, 25, filter.Limit)
	})
}
