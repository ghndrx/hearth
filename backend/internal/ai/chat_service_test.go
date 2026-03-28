package ai

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// MockChatRepository is a mock implementation of ChatRepository
type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) CreateConversation(ctx context.Context, conv *models.AIConversation) error {
	args := m.Called(ctx, conv)
	return args.Error(0)
}

func (m *MockChatRepository) GetConversation(ctx context.Context, id uuid.UUID) (*models.AIConversation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversation), args.Error(1)
}

func (m *MockChatRepository) GetConversationWithMessages(ctx context.Context, id uuid.UUID, limit int) (*models.AIConversationWithMessages, error) {
	args := m.Called(ctx, id, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversationWithMessages), args.Error(1)
}

func (m *MockChatRepository) ListConversations(ctx context.Context, params *models.ConversationListParams) ([]*models.AIConversation, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AIConversation), args.Error(1)
}

func (m *MockChatRepository) UpdateConversation(ctx context.Context, conv *models.AIConversation) error {
	args := m.Called(ctx, conv)
	return args.Error(0)
}

func (m *MockChatRepository) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) CreateMessage(ctx context.Context, msg *models.ConversationMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessage(ctx context.Context, id uuid.UUID) (*models.ConversationMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConversationMessage), args.Error(1)
}

func (m *MockChatRepository) GetConversationMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.ConversationMessage, error) {
	args := m.Called(ctx, conversationID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ConversationMessage), args.Error(1)
}

func (m *MockChatRepository) UpdateMessage(ctx context.Context, msg *models.ConversationMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatRepository) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) DeleteMessagesAfter(ctx context.Context, conversationID uuid.UUID, afterID uuid.UUID) error {
	args := m.Called(ctx, conversationID, afterID)
	return args.Error(0)
}

func (m *MockChatRepository) CreateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error {
	args := m.Called(ctx, tmpl)
	return args.Error(0)
}

func (m *MockChatRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIChatTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIChatTemplate), args.Error(1)
}

func (m *MockChatRepository) ListTemplates(ctx context.Context, userID *uuid.UUID, includePublic bool, category string) ([]*models.AIChatTemplate, error) {
	args := m.Called(ctx, userID, includePublic, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AIChatTemplate), args.Error(1)
}

func (m *MockChatRepository) UpdateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error {
	args := m.Called(ctx, tmpl)
	return args.Error(0)
}

func (m *MockChatRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) CreateShare(ctx context.Context, share *models.AIConversationShare) error {
	args := m.Called(ctx, share)
	return args.Error(0)
}

func (m *MockChatRepository) GetShareByCode(ctx context.Context, code string) (*models.AIConversationShare, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AIConversationShare), args.Error(1)
}

func (m *MockChatRepository) GetConversationShares(ctx context.Context, conversationID uuid.UUID) ([]*models.AIConversationShare, error) {
	args := m.Called(ctx, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AIConversationShare), args.Error(1)
}

func (m *MockChatRepository) DeleteShare(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChatRepository) IncrementShareViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCreateConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	req := &models.CreateConversationRequest{
		Title: "Test Conversation",
	}

	mockRepo.On("CreateConversation", mock.Anything, mock.AnythingOfType("*models.AIConversation")).Return(nil)

	conv, err := service.CreateConversation(context.Background(), userID, req)

	require.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "Test Conversation", conv.Title)
	assert.Equal(t, userID, conv.UserID)
	assert.False(t, conv.IsArchived)
	assert.False(t, conv.IsPinned)

	mockRepo.AssertExpectations(t)
}

func TestCreateConversationWithDefaults(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	req := &models.CreateConversationRequest{}

	mockRepo.On("CreateConversation", mock.Anything, mock.AnythingOfType("*models.AIConversation")).Return(nil)

	conv, err := service.CreateConversation(context.Background(), userID, req)

	require.NoError(t, err)
	assert.Equal(t, "New Chat", conv.Title)
	assert.Equal(t, float32(0.7), conv.Temperature)
	assert.Equal(t, 2048, conv.MaxTokens)

	mockRepo.AssertExpectations(t)
}

func TestGetConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	convID := uuid.New()
	expectedConv := &models.AIConversation{
		ID:     convID,
		UserID: userID,
		Title:  "Test",
	}

	mockRepo.On("GetConversation", mock.Anything, convID).Return(expectedConv, nil)

	conv, err := service.GetConversation(context.Background(), userID, convID)

	require.NoError(t, err)
	assert.Equal(t, expectedConv, conv)

	mockRepo.AssertExpectations(t)
}

func TestGetConversationNotAuthorized(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()
	convID := uuid.New()
	conv := &models.AIConversation{
		ID:     convID,
		UserID: otherUserID, // Different user
		Title:  "Test",
	}

	mockRepo.On("GetConversation", mock.Anything, convID).Return(conv, nil)

	_, err := service.GetConversation(context.Background(), userID, convID)

	assert.Equal(t, ErrNotAuthorized, err)

	mockRepo.AssertExpectations(t)
}

func TestUpdateConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	convID := uuid.New()
	existingConv := &models.AIConversation{
		ID:     convID,
		UserID: userID,
		Title:  "Old Title",
	}

	newTitle := "New Title"
	req := &models.UpdateConversationRequest{
		Title: &newTitle,
	}

	mockRepo.On("GetConversation", mock.Anything, convID).Return(existingConv, nil)
	mockRepo.On("UpdateConversation", mock.Anything, mock.AnythingOfType("*models.AIConversation")).Return(nil)

	conv, err := service.UpdateConversation(context.Background(), userID, convID, req)

	require.NoError(t, err)
	assert.Equal(t, "New Title", conv.Title)

	mockRepo.AssertExpectations(t)
}

func TestDeleteConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	convID := uuid.New()
	conv := &models.AIConversation{
		ID:     convID,
		UserID: userID,
	}

	mockRepo.On("GetConversation", mock.Anything, convID).Return(conv, nil)
	mockRepo.On("DeleteConversation", mock.Anything, convID).Return(nil)

	err := service.DeleteConversation(context.Background(), userID, convID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestListConversations(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	params := &models.ConversationListParams{
		UserID: userID,
		Limit:  10,
	}

	expectedConvs := []*models.AIConversation{
		{ID: uuid.New(), UserID: userID, Title: "Conv 1"},
		{ID: uuid.New(), UserID: userID, Title: "Conv 2"},
	}

	mockRepo.On("ListConversations", mock.Anything, params).Return(expectedConvs, nil)

	convs, err := service.ListConversations(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, convs, 2)

	mockRepo.AssertExpectations(t)
}

func TestGetTemplates(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	expectedTemplates := []*models.AIChatTemplate{
		{ID: uuid.New(), Name: "Template 1", IsPublic: true},
		{ID: uuid.New(), Name: "Template 2", UserID: &userID},
	}

	mockRepo.On("ListTemplates", mock.Anything, &userID, true, "").Return(expectedTemplates, nil)

	templates, err := service.GetTemplates(context.Background(), &userID, "")

	require.NoError(t, err)
	assert.Len(t, templates, 2)

	mockRepo.AssertExpectations(t)
}

func TestShareConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	convID := uuid.New()
	conv := &models.AIConversation{
		ID:     convID,
		UserID: userID,
	}

	mockRepo.On("GetConversation", mock.Anything, convID).Return(conv, nil)
	mockRepo.On("CreateShare", mock.Anything, mock.AnythingOfType("*models.AIConversationShare")).Return(nil)

	share, err := service.ShareConversation(context.Background(), userID, convID, true, false, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, share.ShareCode)
	assert.True(t, share.IsPublic)
	assert.False(t, share.CanContinue)

	mockRepo.AssertExpectations(t)
}

func TestGetSharedConversation(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	userID := uuid.New()
	convID := uuid.New()
	shareID := uuid.New()
	shareCode := "abc123"

	share := &models.AIConversationShare{
		ID:             shareID,
		ConversationID: convID,
		SharedBy:       userID,
		ShareCode:      shareCode,
		IsPublic:       true,
	}

	conv := &models.AIConversationWithMessages{
		AIConversation: models.AIConversation{
			ID:     convID,
			UserID: userID,
			Title:  "Shared Conv",
		},
		Messages: []models.ConversationMessage{},
	}

	mockRepo.On("GetShareByCode", mock.Anything, shareCode).Return(share, nil)
	mockRepo.On("IncrementShareViewCount", mock.Anything, shareID).Return(nil)
	mockRepo.On("GetConversationWithMessages", mock.Anything, convID, 100).Return(conv, nil)

	result, err := service.GetSharedConversation(context.Background(), shareCode)

	require.NoError(t, err)
	assert.Equal(t, "Shared Conv", result.Title)

	mockRepo.AssertExpectations(t)
}

func TestGetSharedConversationExpired(t *testing.T) {
	mockRepo := new(MockChatRepository)
	service := NewChatService(mockRepo, nil)

	shareCode := "expired123"
	expiredTime := time.Now().Add(-24 * time.Hour)

	share := &models.AIConversationShare{
		ID:        uuid.New(),
		ShareCode: shareCode,
		ExpiresAt: &expiredTime,
	}

	mockRepo.On("GetShareByCode", mock.Anything, shareCode).Return(share, nil)

	_, err := service.GetSharedConversation(context.Background(), shareCode)

	assert.Equal(t, ErrShareExpired, err)

	mockRepo.AssertExpectations(t)
}
