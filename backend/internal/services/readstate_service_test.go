package services

import (
	"context"
	"testing"
	"time"

	"hearth/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReadStateRepository is a struct that implements IReadStateRepository for testing.
type MockReadStateRepository struct {
	mock.Mock
}

func (m *MockReadStateRepository) GetByUserIDAndChannelID(ctx context.Context, userID, channelID int64) (*models.ReadState, error) {
	args := m.Called(ctx, userID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReadState), args.Error(1)
}

func (m *MockReadStateRepository) MarkRead(ctx context.Context, readState *models.ReadState) error {
	args := m.Called(ctx, readState)
	return args.Error(0)
}

func (m *MockReadStateRepository) GetUnreadCount(ctx context.Context, userID, channelID int64) (int, error) {
	args := m.Called(ctx, userID, channelID)
	return args.Get(0).(int), args.Error(1)
}

func TestReadStateService_MarkRead(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		channelID      int64
		lastMessageID  int64
		initialState   *models.ReadState
		setupMock      func(*MockReadStateRepository)
		expectedLastID int64
		expectErr      bool
	}{
		{
			name:        "Create new read state if none exists",
			userID:      100,
			channelID:   200,
			lastMessageID: 50,
			initialState: nil,
			setupMock: func(m *MockReadStateRepository) {
			 Args := []interface{}{
				 context.Background(),
				 int64(100), int64(200),
			 }
			 m.On("GetByUserIDAndChannelID", Args...).Return(nil, nil)
			 m.On("MarkRead", mock.Anything, mock.MatchedBy(func(rs *models.ReadState) bool {
				 return rs.UserID == 100 && rs.ChannelID == 200 && rs.LastMessageID == 50
			 })).Return(nil)
			},
			expectedLastID: 50,
			expectErr:      false,
		},
		{
			name:        "Update existing read state",
			userID:      100,
			channelID:   200,
			lastMessageID: 55,
			initialState: &models.ReadState{
				ID:          1,
				UserID:      100,
				ChannelID:   200,
				LastMessageID: 50,
				UpdatedAt:   time.Now(),
			},
			setupMock: func(m *MockReadStateRepository) {
			 m.On("GetByUserIDAndChannelID", context.Background(), int64(100), int64(200)).Return(initialState(t), nil)
			 m.On("MarkRead", mock.Anything, mock.MatchedBy(func(rs *models.ReadState) bool {
				 // Ensure the new state matches the input ID and updated message ID
				 return rs.ID == 1 && rs.LastMessageID == 55
			 })).Return(nil)
			},
			expectedLastID: 55,
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockReadStateRepository)
			tt.setupMock(mockRepo)

			repo := mockRepo // reassign for readability if desired, or use mockRepo directly
			service := NewReadStateService(repo)

			// Act
			err := service.MarkRead(context.Background(), tt.userID, tt.channelID, tt.lastMessageID)

			// Assert
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// Helper to create test data
func initialState(t *testing.T) *models.ReadState {
	t.Helper()
	return &models.ReadState{
		ID:          1,
		UserID:      100,
		ChannelID:   200,
		LastMessageID: 50,
		UpdatedAt:   time.Now(),
	}
}

func TestReadStateService_GetUnreadCount(t *testing.T) {
	t.Run("Return 0 when no read state exists", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		mockRepo.On("GetByUserIDAndChannelID", context.Background(), int64(1), int64(1)).Return(nil, nil)
		mockRepo.On("GetUnreadCount", context.Background(), int64(1), int64(1)).Return(0, nil)
		
		service := NewReadStateService(mockRepo)
		
		count, err := service.GetUnreadCount(context.Background(), 1, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Return count from repository", func(t *testing.T) {
		mockRepo := new(MockReadStateRepository)
		expectedCount := 5
		mockRepo.On("GetByUserIDAndChannelID", context.Background(), int64(1), int64(1)).Return(initialState(t), nil)
		mockRepo.On("GetUnreadCount", context.Background(), int64(1), int64(1)).Return(expectedCount, nil)
		
		service := NewReadStateService(mockRepo)
		
		count, err := service.GetUnreadCount(context.Background(), 1, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})
}