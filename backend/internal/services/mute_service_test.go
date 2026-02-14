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

// MockMuteRepository implements the MuteRepository interface for testing.
type MockMuteRepository struct {
	mock.Mock
}

func (m *MockMuteRepository) Create(ctx context.Context, mute *models.Mute) error {
	args := m.Called(ctx, mute)
	return args.Error(0)
}

func (m *MockMuteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Mute, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Mute), args.Error(1)
}

func (m *MockMuteRepository) GetByChannelAndUser(ctx context.Context, channelID, userID uuid.UUID) (*models.Mute, error) {
	args := m.Called(ctx, channelID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Mute), args.Error(1)
}

func (m *MockMuteRepository) Update(ctx context.Context, mute *models.Mute) error {
	args := m.Called(ctx, mute)
	return args.Error(0)
}

func TestMuteService_MuteUser(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := new(MockMuteRepository)
	service := NewMuteService(repo)

	channelID := uuid.New()
	userID := uuid.New()
	duration := 60 // minutes

	expectedMute := &models.Mute{
		ID:        uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		StartedAt: models.TimeRange{End: models.TimeRange{}},
		// EndsAt would be calculated by service, mock expects final state or intermediate
	}

	repo.On("Create", ctx, mock.AnythingOfType("*models.Mute")).Return(nil)

	// Act
	result, err := service.MuteUser(ctx, channelID, userID, duration)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	repo.AssertExpectations(t)
}

func TestMuteService_UnmuteUser_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := new(MockMuteRepository)
	service := NewMuteService(repo)

	channelID := uuid.New()
	userID := uuid.New()

	now := time.Now()
	activeMute := &models.Mute{
		ID:        uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		StartedAt: models.TimeRange{End: models.TimeRange{}},
	}

	repo.On("GetByChannelAndUser", ctx, channelID, userID).Return(activeMute, nil)
	repo.On("Update", ctx, activeMute).Return(nil)

	// Act
	err := service.UnmuteUser(ctx, channelID, userID)

	// Assert
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteService_IsUserMuted_Muted(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := new(MockMuteRepository)
	service := NewMuteService(repo)

	channelID := uuid.New()
	userID := uuid.New()

	// Mute that has NOT ended yet (EndedAt is nil implies indefinite or effectively active)
	mutedUser := &models.Mute{
		ID:        uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		StartedAt: models.TimeRange{End: models.TimeRange{}},
		EndedAt:   nil, // Still active
	}

	repo.On("GetByChannelAndUser", ctx, channelID, userID).Return(mutedUser, nil)

	// Act
	isMuted, err := service.IsUserMuted(ctx, channelID, userID)

	// Assert
	assert.True(t, isMuted)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMuteService_UnmuteUser_NotMuted(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := new(MockMuteRepository)
	service := NewMuteService(repo)

	channelID := uuid.New()
	userID := uuid.New()

	repo.On("GetByChannelAndUser", ctx, channelID, userID).Return(nil, nil) // Not muted

	// Act
	err := service.UnmuteUser(ctx, channelID, userID)

	// Assert
	assert.Equal(t, ErrUserNotMuted, err)
	repo.AssertExpectations(t)
}