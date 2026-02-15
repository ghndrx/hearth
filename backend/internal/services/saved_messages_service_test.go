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

// MockSavedMessagesRepository mocks the SavedMessagesRepository
type MockSavedMessagesRepository struct {
	mock.Mock
}

func (m *MockSavedMessagesRepository) Save(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error) {
	args := m.Called(ctx, userID, messageID, note)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SavedMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesRepository) GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*models.SavedMessage, error) {
	args := m.Called(ctx, userID, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesRepository) GetByUser(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesRepository) GetByUserWithMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesRepository) UpdateNote(ctx context.Context, id uuid.UUID, note *string) error {
	args := m.Called(ctx, id, note)
	return args.Error(0)
}

func (m *MockSavedMessagesRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSavedMessagesRepository) DeleteByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) error {
	args := m.Called(ctx, userID, messageID)
	return args.Error(0)
}

func (m *MockSavedMessagesRepository) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockSavedMessagesRepository) IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, messageID)
	return args.Bool(0), args.Error(1)
}

// MockMessageChecker mocks the MessageExistsChecker
type MockMessageChecker struct {
	mock.Mock
}

func (m *MockMessageChecker) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func TestSavedMessagesService_SaveMessage(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()
	note := "Important message"

	message := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		Content:   "Test content",
		CreatedAt: time.Now(),
	}

	savedMsg := &models.SavedMessage{
		ID:        uuid.New(),
		UserID:    userID,
		MessageID: messageID,
		Note:      &note,
		CreatedAt: time.Now(),
	}

	msgChecker.On("GetByID", ctx, messageID).Return(message, nil)
	repo.On("Save", ctx, userID, messageID, &note).Return(savedMsg, nil)
	eventBus.On("Publish", "user.message_saved", mock.Anything).Return()

	result, err := service.SaveMessage(ctx, userID, messageID, &note)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, savedMsg.ID, result.ID)
	assert.Equal(t, message, result.Message)

	repo.AssertExpectations(t)
	msgChecker.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSavedMessagesService_SaveMessage_MessageNotFound(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	msgChecker.On("GetByID", ctx, messageID).Return(nil, nil)

	result, err := service.SaveMessage(ctx, userID, messageID, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
	assert.Nil(t, result)

	msgChecker.AssertExpectations(t)
}

func TestSavedMessagesService_SaveMessage_RepoError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	message := &models.Message{
		ID:      messageID,
		Content: "Test content",
	}

	msgChecker.On("GetByID", ctx, messageID).Return(message, nil)
	repo.On("Save", ctx, userID, messageID, (*string)(nil)).Return(nil, assert.AnError)

	result, err := service.SaveMessage(ctx, userID, messageID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)

	msgChecker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessages(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	opts := &models.SavedMessagesQueryOptions{Limit: 50}

	savedMessages := []*models.SavedMessage{
		{
			ID:        uuid.New(),
			UserID:    userID,
			MessageID: uuid.New(),
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			MessageID: uuid.New(),
			CreatedAt: time.Now(),
		},
	}

	repo.On("GetByUserWithMessages", ctx, userID, opts).Return(savedMessages, nil)

	result, err := service.GetSavedMessages(ctx, userID, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessage(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	message := &models.Message{
		ID:      messageID,
		Content: "Test content",
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)
	msgChecker.On("GetByID", ctx, messageID).Return(message, nil)

	result, err := service.GetSavedMessage(ctx, userID, savedID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, savedID, result.ID)
	assert.Equal(t, message, result.Message)

	repo.AssertExpectations(t)
	msgChecker.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessage_NotFound(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()

	repo.On("GetByID", ctx, savedID).Return(nil, nil)

	result, err := service.GetSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)
	assert.Equal(t, ErrSavedMessageNotFound, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessage_Unauthorized(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    otherUserID, // Different user
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)

	result, err := service.GetSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)
	assert.Equal(t, ErrUnauthorized, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_UpdateSavedMessageNote(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()
	newNote := "Updated note"

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	message := &models.Message{
		ID:      messageID,
		Content: "Test content",
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)
	repo.On("UpdateNote", ctx, savedID, &newNote).Return(nil)
	msgChecker.On("GetByID", ctx, messageID).Return(message, nil)

	result, err := service.UpdateSavedMessageNote(ctx, userID, savedID, &newNote)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &newNote, result.Note)

	repo.AssertExpectations(t)
	msgChecker.AssertExpectations(t)
}

func TestSavedMessagesService_UpdateSavedMessageNote_NotFound(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	newNote := "Updated note"

	repo.On("GetByID", ctx, savedID).Return(nil, nil)

	result, err := service.UpdateSavedMessageNote(ctx, userID, savedID, &newNote)

	assert.Error(t, err)
	assert.Equal(t, ErrSavedMessageNotFound, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_UpdateSavedMessageNote_Unauthorized(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()
	newNote := "Updated note"

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    otherUserID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)

	result, err := service.UpdateSavedMessageNote(ctx, userID, savedID, &newNote)

	assert.Error(t, err)
	assert.Equal(t, ErrUnauthorized, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessage(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)
	repo.On("Delete", ctx, savedID).Return(nil)
	eventBus.On("Publish", "user.message_unsaved", mock.Anything).Return()

	err := service.RemoveSavedMessage(ctx, userID, savedID)

	assert.NoError(t, err)

	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessage_NotFound(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()

	repo.On("GetByID", ctx, savedID).Return(nil, nil)

	err := service.RemoveSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)
	assert.Equal(t, ErrSavedMessageNotFound, err)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessage_Unauthorized(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    otherUserID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)

	err := service.RemoveSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)
	assert.Equal(t, ErrUnauthorized, err)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessageByMessageID(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByUserAndMessage", ctx, userID, messageID).Return(savedMsg, nil)
	repo.On("DeleteByUserAndMessage", ctx, userID, messageID).Return(nil)
	eventBus.On("Publish", "user.message_unsaved", mock.Anything).Return()

	err := service.RemoveSavedMessageByMessageID(ctx, userID, messageID)

	assert.NoError(t, err)

	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessageByMessageID_NotFound(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	repo.On("GetByUserAndMessage", ctx, userID, messageID).Return(nil, nil)

	err := service.RemoveSavedMessageByMessageID(ctx, userID, messageID)

	assert.Error(t, err)
	assert.Equal(t, ErrSavedMessageNotFound, err)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_IsSaved(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	repo.On("IsSaved", ctx, userID, messageID).Return(true, nil)

	result, err := service.IsSaved(ctx, userID, messageID)

	assert.NoError(t, err)
	assert.True(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_IsSaved_NotSaved(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	repo.On("IsSaved", ctx, userID, messageID).Return(false, nil)

	result, err := service.IsSaved(ctx, userID, messageID)

	assert.NoError(t, err)
	assert.False(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedCount(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Count", ctx, userID).Return(42, nil)

	result, err := service.GetSavedCount(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, 42, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedCount_Error(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("Count", ctx, userID).Return(0, assert.AnError)

	result, err := service.GetSavedCount(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, 0, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_SaveMessage_MsgRepoError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	msgChecker.On("GetByID", ctx, messageID).Return(nil, assert.AnError)

	result, err := service.SaveMessage(ctx, userID, messageID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)

	msgChecker.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessages_Error(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetByUserWithMessages", ctx, userID, (*models.SavedMessagesQueryOptions)(nil)).Return(nil, assert.AnError)

	result, err := service.GetSavedMessages(ctx, userID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_GetSavedMessage_RepoError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()

	repo.On("GetByID", ctx, savedID).Return(nil, assert.AnError)

	result, err := service.GetSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_UpdateSavedMessageNote_UpdateError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()
	newNote := "Updated note"

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)
	repo.On("UpdateNote", ctx, savedID, &newNote).Return(assert.AnError)

	result, err := service.UpdateSavedMessageNote(ctx, userID, savedID, &newNote)

	assert.Error(t, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessage_DeleteError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByID", ctx, savedID).Return(savedMsg, nil)
	repo.On("Delete", ctx, savedID).Return(assert.AnError)

	err := service.RemoveSavedMessage(ctx, userID, savedID)

	assert.Error(t, err)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_RemoveSavedMessageByMessageID_DeleteError(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	savedID := uuid.New()
	messageID := uuid.New()

	savedMsg := &models.SavedMessage{
		ID:        savedID,
		UserID:    userID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	repo.On("GetByUserAndMessage", ctx, userID, messageID).Return(savedMsg, nil)
	repo.On("DeleteByUserAndMessage", ctx, userID, messageID).Return(assert.AnError)

	err := service.RemoveSavedMessageByMessageID(ctx, userID, messageID)

	assert.Error(t, err)

	repo.AssertExpectations(t)
}

func TestSavedMessagesService_IsSaved_Error(t *testing.T) {
	repo := new(MockSavedMessagesRepository)
	msgChecker := new(MockMessageChecker)
	eventBus := new(MockEventBus)

	service := NewSavedMessagesService(repo, msgChecker, eventBus)

	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	repo.On("IsSaved", ctx, userID, messageID).Return(false, assert.AnError)

	result, err := service.IsSaved(ctx, userID, messageID)

	assert.Error(t, err)
	assert.False(t, result)

	repo.AssertExpectations(t)
}
