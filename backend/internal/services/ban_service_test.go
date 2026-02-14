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

// MockBanManagementRepository implements BanManagementRepository for testing.
type MockBanManagementRepository struct {
	mock.Mock
}

func (m *MockBanManagementRepository) FindByUserAndGuild(ctx context.Context, guildID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, guildID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockBanManagementRepository) Create(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockBanManagementRepository) GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockBanManagementRepository) Delete(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func TestBanService_CreateBan_Success(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	guildID := uuid.New()
	userID := uuid.New()
	bannedBy := uuid.New()
	reason := "Spamming"

	mockRepo.On("FindByUserAndGuild", ctx, guildID, userID).Return(nil, ErrBanNotFound)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(b *models.Ban) bool {
		return b.ServerID == guildID && b.UserID == userID && *b.Reason == reason
	})).Return(nil)

	ban, err := service.CreateBan(ctx, guildID, userID, reason, bannedBy)

	assert.NoError(t, err)
	assert.NotNil(t, ban)
	assert.Equal(t, guildID, ban.ServerID)
	assert.Equal(t, userID, ban.UserID)
	mockRepo.AssertExpectations(t)
}

func TestBanService_CreateBan_AlreadyExists(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	guildID := uuid.New()
	userID := uuid.New()
	bannedBy := uuid.New()
	reason := "Spamming"

	existingBan := &models.Ban{
		ServerID: guildID,
		UserID:   userID,
	}

	mockRepo.On("FindByUserAndGuild", ctx, guildID, userID).Return(existingBan, nil)

	ban, err := service.CreateBan(ctx, guildID, userID, reason, bannedBy)

	assert.ErrorIs(t, err, ErrBanAlreadyExist)
	assert.Nil(t, ban)
	mockRepo.AssertExpectations(t)
}

func TestBanService_CreateBan_RepositoryError(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	guildID := uuid.New()
	userID := uuid.New()
	bannedBy := uuid.New()
	reason := "Spamming"

	mockRepo.On("FindByUserAndGuild", ctx, guildID, userID).Return(nil, errors.New("db error"))

	ban, err := service.CreateBan(ctx, guildID, userID, reason, bannedBy)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing bans")
	assert.Nil(t, ban)
	mockRepo.AssertExpectations(t)
}

func TestBanService_Unban_Success(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	existingBan := &models.Ban{
		ServerID: serverID,
		UserID:   userID,
	}

	mockRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(existingBan, nil)
	mockRepo.On("Delete", ctx, serverID, userID).Return(nil)

	err := service.Unban(ctx, serverID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBanService_Unban_NotFound(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(nil, errors.New("not found"))

	err := service.Unban(ctx, serverID, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve ban")
	mockRepo.AssertExpectations(t)
}

func TestBanService_GetBan_Success(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	expectedBan := &models.Ban{
		ServerID: serverID,
		UserID:   userID,
	}

	mockRepo.On("GetByServerAndUser", ctx, serverID, userID).Return(expectedBan, nil)

	ban, err := service.GetBan(ctx, serverID, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedBan, ban)
	mockRepo.AssertExpectations(t)
}

func TestBanService_CheckIfBanned_True(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	guildID := uuid.New()
	userID := uuid.New()

	existingBan := &models.Ban{
		ServerID: guildID,
		UserID:   userID,
	}

	mockRepo.On("FindByUserAndGuild", ctx, guildID, userID).Return(existingBan, nil)

	isBanned, err := service.CheckIfBanned(ctx, guildID, userID)

	assert.NoError(t, err)
	assert.True(t, isBanned)
	mockRepo.AssertExpectations(t)
}

func TestBanService_CheckIfBanned_False(t *testing.T) {
	mockRepo := new(MockBanManagementRepository)
	service := NewBanService(mockRepo)
	ctx := context.Background()

	guildID := uuid.New()
	userID := uuid.New()

	mockRepo.On("FindByUserAndGuild", ctx, guildID, userID).Return(nil, ErrBanNotFound)

	isBanned, err := service.CheckIfBanned(ctx, guildID, userID)

	assert.NoError(t, err)
	assert.False(t, isBanned)
	mockRepo.AssertExpectations(t)
}
