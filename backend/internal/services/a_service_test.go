package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHeartRepository implements HeartRepository for testing purposes.
type MockHeartRepository struct {
	mock.Mock
}

func (m *MockHeartRepository) Create(ctx context.Context, heart *Heart) error {
	args := m.Called(ctx, heart)
	return args.Error(0)
}

func (m *MockHeartRepository) GetByID(ctx context.Context, id uuid.UUID) (*Heart, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Heart), args.Error(1)
}

func (m *MockHeartRepository) GetByTargetID(ctx context.Context, targetUserID uuid.UUID) (int64, error) {
	args := m.Called(ctx, targetUserID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockHeartRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func TestHeartService_CreateHeart(t *testing.T) {
	ctx := context.Background()

	authorID := uuid.New()
	targetID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		userID    uuid.UUID
		target    uuid.UUID
		expectErr bool
	}{
		{
			name:   "Success",
			userID: authorID,
			target: targetID,
		},
		{
			name:      "Self heart rejected",
			userID:    authorID,
			target:    authorID,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockHeartRepository)
			service := NewHeartService(mockRepo)

			if tt.name == "Success" {
				// Setup expectations
				expectedHeart := &Heart{
					ID:        uuid.New(),
					AuthorID:  tt.userID,
					TargetID:  tt.target,
					Active:    true,
					CreatedAt: now,
				}
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*services.Heart")).Return(nil).Run(func(args mock.Arguments) {
					h := args.Get(1).(*Heart)
					h.ID = expectedHeart.ID
					h.CreatedAt = now
				}).Once()

				// Test execution
				heart, err := service.CreateHeart(ctx, tt.userID, tt.target)

				// Assertions
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, heart.AuthorID)
				assert.Equal(t, tt.target, heart.TargetID)
				mockRepo.AssertExpectations(t)
			} else if tt.name == "Self heart rejected" {
				// Test execution
				_, err := service.CreateHeart(ctx, tt.userID, tt.target)
				assert.Error(t, err)
				assert.Equal(t, ErrSelfAction, err)
				// Ensure repo.Create was NOT called
				mockRepo.AssertNotCalled(t, "Create", ctx, mock.Anything)
			}
		})
	}
}

func TestHeartService_GetHeartSummary(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockHeartRepository)
	service := NewHeartService(mockRepo)

	// Mock Data
	targetUser := uuid.New()
	fromUser := uuid.New()
	metricsToSend := int64(42)
	metricsToReceive := int64(1337)

	// Setup Expectations
	mockRepo.On("GetByUserID", ctx, fromUser).Return(metricsToSend, nil).Once()
	mockRepo.On("GetByTargetID", ctx, targetUser).Return(metricsToReceive, nil).Once()

	// Test Execution
	result, err := service.GetHeartSummary(ctx, targetUser, fromUser)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, metricsToReceive, result.LikeCount)
	assert.Equal(t, metricsToSend, result.RepostCount)
	assert.Equal(t, int64(0), result.QuoteCount)

	// Verify calls
	mockRepo.AssertExpectations(t)
}
