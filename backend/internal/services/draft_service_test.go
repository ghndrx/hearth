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

// MockDraftRepository implements the DraftRepository interface for testing.
type MockDraftRepository struct {
	mock.Mock
}

func (m *MockDraftRepository) GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Draft), args.Error(1)
}

func (m *MockDraftRepository) CreateDraft(ctx context.Context, draft *models.Draft) error {
	args := m.Called(ctx, draft)
	return args.Error(0)
}

func (m *MockDraftRepository) UpdateDraftStatus(ctx context.Context, id uuid.UUID, status models.DraftStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDraftRepository) ListChannels(ctx context.Context, guildID uuid.UUID) ([]models.Channel, error) {
	args := m.Called(ctx, guildID)
	return args.Get(0).([]models.Channel), args.Error(1)
}

func TestNewDraftService(t *testing.T) {
	repo := new(MockDraftRepository)
	service := NewDraftService(repo)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
}

func TestCreateDraft_Success(t *testing.T) {
	// Arrange
	service := NewDraftService(new(MockDraftRepository))
	ctx := context.Background()
	req := models.CreateDraftRequest{
		Title:     "Test Title",
		Content:   "Test Content",
		GuildID:   uuid.New(),
		ChannelID: uuid.New(),
		CreatedBy: uuid.New(),
	}

	// We expect the repo to be called exactly once
	mockRepo := service.repo.(*MockDraftRepository)
	mockRepo.On("CreateDraft", ctx, mock.MatchedBy(func(d *models.Draft) bool {
		return d.Title == req.Title && d.Status == models.DraftStatusDraft
	})).Return(nil)

	// Act
	result, err := service.CreateDraft(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Title, result.Title)
	assert.Equal(t, models.DraftStatusDraft, result.Status)
	mockRepo.AssertExpectations(t)
}

func TestCreateDraft_EmptyTitle(t *testing.T) {
	// Arrange
	service := NewDraftService(new(MockDraftRepository))
	ctx := context.Background()
	req := models.CreateDraftRequest{
		Title:   "",
		Content: "Some content",
		GuildID: uuid.New(),
	}

	// Act
	_, err := service.CreateDraft(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, "draft title cannot be empty", err.Error())
}

func TestPublishDraft_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uuid.New()
	mockRepo := new(MockDraftRepository)
	service := NewDraftService(mockRepo)

	mockRepo.On("UpdateDraftStatus", ctx, draftID, models.DraftStatusPublished).Return(nil)

	// Act
	err := service.PublishDraft(ctx, draftID, uuid.New(), "msg123")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateDraft_UpdateExisting(t *testing.T) {
	// Arrange
	ctx := context.Background()
	draftID := uuid.New()
	newTitle := "Updated Title"
	mockRepo := new(MockDraftRepository)
	service := NewDraftService(mockRepo)

	// Setup repo mock to return existing draft
	existingDraft := &models.Draft{
		ID:        draftID,
		Title:     "Old Title",
		Content:   "Old Content",
		Status:    models.DraftStatusDraft,
		CreatedAt: time.Now(),
	}
	mockRepo.On("GetDraft", ctx, draftID).Return(existingDraft, nil)
	mockRepo.On("CreateDraft", ctx, mock.AnythingOfType("*models.Draft")).Return(nil)

	// Act
	req := models.UpdateDraftRequest{
		Title:   &newTitle,
		Content: nil,
	}
	err := service.UpdateDraft(ctx, draftID, req)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
