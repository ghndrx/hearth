package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockStatusRepository is a test double for StatusRepository.
type MockStatusRepository struct {
	mock.Mock
}

func (m *MockStatusRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Status, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Status), args.Error(1)
}

func (m *MockStatusRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Status), args.Error(1)
}

func (m *MockStatusRepository) Update(ctx context.Context, status *models.Status) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockStatusRepository) Create(ctx context.Context, status *models.Status) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func TestStatusService_UpdateOrCreateStatus_Success(t *testing.T) {
	// Arrange
	userID := uuid.New()
	gameID := "valheim"
	activityDetails := "Entering the woods..."
	statusInput := models.Status{
		Status:          "Playing Valheim",
		GameID:          &gameID,
		ActivityDetails: &activityDetails,
	}

	// Mock existing status
	expectedStatus := &models.Status{
		ID:              uuid.New(),
		UserID:          userID,
		Status:          "Playing Valheim",
		GameID:          &gameID,
		ActivityDetails: &activityDetails,
	}

	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expectedStatus, nil)
	mockRepo.On("Update", mock.Anything, expectedStatus).Return(nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateOrCreateStatus_New(t *testing.T) {
	// Arrange
	userID := uuid.New()
	emptyGameID := ""
	emptyActivityDetails := ""
	statusInput := models.Status{
		Status:          "Online",
		GameID:          &emptyGameID,
		ActivityDetails: &emptyActivityDetails,
	}

	mockRepo := new(MockStatusRepository)
	// Mock not found (GetByUserID returns nil, nil)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, nil)
	// Mock creation
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *models.Status) bool {
		return s.UserID == userID && s.Status == "Online"
	})).Return(nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, statusInput.Status, result.Status)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateOrCreateStatus_DBError(t *testing.T) {
	// Arrange
	userID := uuid.New()
	statusInput := models.Status{Status: "Offline"}

	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("database connection lost"))

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.UpdateOrCreateStatus(context.Background(), userID, statusInput)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetUserStatus(t *testing.T) {
	// Arrange
	userID := uuid.New()
	expectedStatus := &models.Status{
		ID:     uuid.New(),
		UserID: userID,
		Status: "Playing Minecraft",
	}
	mockRepo := new(MockStatusRepository)
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expectedStatus, nil)

	service := NewStatusService(mockRepo)

	// Act
	result, err := service.GetUserStatus(context.Background(), userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, result)
	mockRepo.AssertExpectations(t)
}
