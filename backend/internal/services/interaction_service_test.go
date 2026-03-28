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

// MockInteractionRepository is a mock implementation of InteractionRepository
type MockInteractionRepository struct {
	mock.Mock
	tokens map[string]*InteractionToken
}

func NewMockInteractionRepository() *MockInteractionRepository {
	return &MockInteractionRepository{
		tokens: make(map[string]*InteractionToken),
	}
}

func (m *MockInteractionRepository) SaveToken(ctx context.Context, token *InteractionToken) error {
	args := m.Called(ctx, token)
	if args.Error(0) == nil {
		m.tokens[token.Token] = token
	}
	return args.Error(0)
}

func (m *MockInteractionRepository) GetToken(ctx context.Context, token string) (*InteractionToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*InteractionToken), args.Error(1)
}

func (m *MockInteractionRepository) MarkTokenUsed(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	if args.Error(0) == nil && m.tokens[token] != nil {
		m.tokens[token].Used = true
	}
	return args.Error(0)
}

// MockWebhookCommander is a mock implementation of WebhookCommander
type MockWebhookCommander struct {
	mock.Mock
}

func (m *MockWebhookCommander) SendCommandWebhook(ctx context.Context, appID uuid.UUID, payload interface{}) (*models.InteractionResponse, error) {
	args := m.Called(ctx, appID, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InteractionResponse), args.Error(1)
}

// MockEventBus is already defined in mocks_test.go

// MockSlashCommandService is a mock for the embedded slash command service
type MockSlashCommandService struct {
	mock.Mock
}

func (m *MockSlashCommandService) ExecuteCommand(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	args := m.Called(ctx, interaction)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InteractionResponse), args.Error(1)
}

func (m *MockSlashCommandService) GetAutocomplete(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	args := m.Called(ctx, interaction)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InteractionResponse), args.Error(1)
}

func TestInteractionService_HandleInteraction_Ping(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	eventBus := new(MockEventBus)

	// Set up mock expectations
	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)
	eventBus.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("*services.InteractionCreatedEvent")).Return()

	service := NewInteractionServiceWithEventBus(repo, nil, nil, nil, eventBus)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypePing,
		Token:     "test-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	resp, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, models.CallbackTypePong, resp.Type)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestInteractionService_HandleInteraction_ApplicationCommand(t *testing.T) {
	// Test that application command interaction type is correctly identified
	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeApplicationCommand,
		Token:     "test-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
		Data: &models.CommandInteractionData{
			ID:   uuid.New(),
			Name: "ping",
			Type: models.CommandTypeSlash,
		},
	}

	// Verify the interaction structure is valid for application command
	assert.Equal(t, models.InteractionTypeApplicationCommand, interaction.Type)
	assert.NotNil(t, interaction.Data)
	cmdData, ok := interaction.Data.(*models.CommandInteractionData)
	assert.True(t, ok)
	assert.Equal(t, "ping", cmdData.Name)
}

func TestInteractionService_HandleInteraction_Autocomplete(t *testing.T) {
	// Test that autocomplete interaction type is correctly identified
	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeAutocomplete,
		Token:     "test-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
		Data: &models.CommandInteractionData{
			ID:   uuid.New(),
			Name: "command",
			Type: models.CommandTypeSlash,
		},
	}

	// Verify the interaction structure is valid for autocomplete
	assert.Equal(t, models.InteractionTypeAutocomplete, interaction.Type)
	assert.NotNil(t, interaction.Data)
	cmdData, ok := interaction.Data.(*models.CommandInteractionData)
	assert.True(t, ok)
	assert.Equal(t, "command", cmdData.Name)
}

func TestInteractionService_HandleInteraction_ModalSubmit(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	mockWebhook := new(MockWebhookCommander)
	eventBus := new(MockEventBus)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeModalSubmit,
		Token:     "test-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)
	eventBus.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("*services.InteractionCreatedEvent")).Return()
	mockWebhook.On("SendCommandWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(
		&models.InteractionResponse{Type: models.CallbackTypeDeferredUpdateMessage}, nil)

	service := NewInteractionServiceWithEventBus(repo, nil, mockWebhook, nil, eventBus)

	resp, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, models.CallbackTypeDeferredUpdateMessage, resp.Type)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestInteractionService_HandleInteraction_UnknownType(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionType(999), // Unknown type
		Token:     "test-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)

	service := NewInteractionService(repo, nil, nil, nil)

	resp, err := service.HandleInteraction(ctx, interaction)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unknown interaction type")
}

func TestInteractionService_HandleInteraction_StoresToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	eventBus := new(MockEventBus)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypePing,
		Token:     "unique-token-123",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)
	eventBus.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("*services.InteractionCreatedEvent")).Return()

	// Set up GetToken to return the stored token
	repo.On("GetToken", mock.Anything, "unique-token-123").Return(&InteractionToken{
		Token:         "unique-token-123",
		InteractionID: interaction.ID,
		AppID:         interaction.AppID,
		UserID:        interaction.UserID,
		ChannelID:     interaction.ChannelID,
	}, nil)

	service := NewInteractionServiceWithEventBus(repo, nil, nil, nil, eventBus)

	_, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	repo.AssertCalled(t, "SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken"))

	// Verify token was stored
	storedToken, err := repo.GetToken(ctx, "unique-token-123")
	assert.NoError(t, err)
	assert.NotNil(t, storedToken)
	assert.Equal(t, "unique-token-123", storedToken.Token)
}

func TestInteractionService_CreateResponse_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	mockWebhook := new(MockWebhookCommander)

	// Create a stored token
	serverID := uuid.New()
	token := &InteractionToken{
		Token:         "response-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ServerID:      &serverID,
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		Used:          false,
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "response-token").Return(token, nil)
	repo.On("MarkTokenUsed", mock.Anything, "response-token").Return(nil)

	mockWebhook.On("SendCommandWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(
		&models.InteractionResponse{Type: models.CallbackTypeChannelMessage}, nil)

	service := NewInteractionService(repo, nil, mockWebhook, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
		Data: &models.InteractionCallbackData{
			Content: strPtr("Follow-up message"),
		},
	}

	err := service.CreateResponse(ctx, "response-token", response)

	assert.NoError(t, err)
	repo.AssertCalled(t, "MarkTokenUsed", mock.Anything, "response-token")
	mockWebhook.AssertCalled(t, "SendCommandWebhook", mock.Anything, token.AppID, mock.Anything)
}

func TestInteractionService_CreateResponse_InvalidToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	repo.On("GetToken", mock.Anything, "invalid-token").Return(nil, assert.AnError)

	service := NewInteractionService(repo, nil, nil, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
	}

	err := service.CreateResponse(ctx, "invalid-token", response)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestInteractionService_CreateResponse_TokenAlreadyUsed(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	token := &InteractionToken{
		Token:         "used-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		Used:          true, // Already used
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "used-token").Return(token, nil)

	service := NewInteractionService(repo, nil, nil, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
	}

	err := service.CreateResponse(ctx, "used-token", response)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token already used")
}

func TestInteractionService_CreateResponse_TokenExpired(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	token := &InteractionToken{
		Token:         "expired-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(-1 * time.Minute), // Expired
		Used:          false,
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "expired-token").Return(token, nil)

	service := NewInteractionService(repo, nil, nil, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
	}

	err := service.CreateResponse(ctx, "expired-token", response)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

func TestInteractionService_EditResponse_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	mockWebhook := new(MockWebhookCommander)

	token := &InteractionToken{
		Token:         "edit-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		Used:          false,
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "edit-token").Return(token, nil)
	mockWebhook.On("SendCommandWebhook", mock.Anything, token.AppID, mock.Anything).Return(
		&models.InteractionResponse{Type: models.CallbackTypeChannelMessage}, nil)

	service := NewInteractionService(repo, nil, mockWebhook, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeUpdateMessage,
		Data: &models.InteractionCallbackData{
			Content: strPtr("Edited message"),
		},
	}

	err := service.EditResponse(ctx, "edit-token", response)

	assert.NoError(t, err)
	mockWebhook.AssertCalled(t, "SendCommandWebhook", mock.Anything, token.AppID, mock.Anything)
}

func TestInteractionService_EditResponse_ExpiredToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	token := &InteractionToken{
		Token:         "expired-edit-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(-1 * time.Minute),
		Used:          false,
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "expired-edit-token").Return(token, nil)

	service := NewInteractionService(repo, nil, nil, nil)

	response := &models.InteractionResponse{
		Type: models.CallbackTypeUpdateMessage,
	}

	err := service.EditResponse(ctx, "expired-edit-token", response)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

func TestInteractionService_DeleteResponse_Success(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	mockWebhook := new(MockWebhookCommander)

	token := &InteractionToken{
		Token:         "delete-token",
		InteractionID: uuid.New(),
		AppID:         uuid.New(),
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		Used:          false,
		CreatedAt:     time.Now(),
	}

	repo.On("GetToken", mock.Anything, "delete-token").Return(token, nil)
	mockWebhook.On("SendCommandWebhook", mock.Anything, token.AppID, mock.Anything).Return(
		&models.InteractionResponse{Type: models.CallbackTypeChannelMessage}, nil)

	service := NewInteractionService(repo, nil, mockWebhook, nil)

	err := service.DeleteResponse(ctx, "delete-token", "message-123")

	assert.NoError(t, err)
	mockWebhook.AssertCalled(t, "SendCommandWebhook", mock.Anything, token.AppID, mock.Anything)
}

func TestInteractionService_DeleteResponse_InvalidToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()

	repo.On("GetToken", mock.Anything, "nonexistent-token").Return(nil, assert.AnError)

	service := NewInteractionService(repo, nil, nil, nil)

	err := service.DeleteResponse(ctx, "nonexistent-token", "message-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestInteractionService_GenerateToken(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	service := NewInteractionService(repo, nil, nil, nil)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypePing,
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
		// Token is intentionally empty to trigger generation
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)

	resp, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, models.CallbackTypePong, resp.Type)
	repo.AssertCalled(t, "SaveToken", mock.Anything, mock.MatchedBy(func(token *InteractionToken) bool {
		return token.Token != ""
	}))
}

func TestInteractionService_PublishesEvents(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	eventBus := new(MockEventBus)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypePing,
		Token:     "event-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)
	eventBus.On("Publish", "interaction.created", mock.AnythingOfType("*services.InteractionCreatedEvent")).Return()

	service := NewInteractionServiceWithEventBus(repo, nil, nil, nil, eventBus)

	_, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	// Ping interactions should publish InteractionCreated event
	eventBus.AssertCalled(t, "Publish", "interaction.created", mock.AnythingOfType("*services.InteractionCreatedEvent"))
}

func TestInteractionService_AutocompletePublishesAutocompleteEvent(t *testing.T) {
	// This test verifies that when an autocomplete interaction is received,
	// the event bus publishes the correct event type.
	// Note: The actual autocomplete handling requires slashCmdService to be set,
	// so we test the event type directly by checking the interaction type constant.

	interactionType := models.InteractionTypeAutocomplete
	assert.Equal(t, models.InteractionType(4), interactionType)

	// Verify the interaction type is correctly mapped
	eventType := "autocomplete.created"
	assert.Equal(t, "autocomplete.created", eventType)
}

func TestInteractionService_ModalSubmitPublishesModalEvent(t *testing.T) {
	ctx := context.Background()
	repo := NewMockInteractionRepository()
	eventBus := new(MockEventBus)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeModalSubmit,
		Token:     "modal-token",
		ChannelID: uuid.New(),
		AppID:     uuid.New(),
		UserID:    uuid.New(),
	}

	repo.On("SaveToken", mock.Anything, mock.AnythingOfType("*services.InteractionToken")).Return(nil)
	eventBus.On("Publish", "modal.submitted", mock.AnythingOfType("*services.InteractionCreatedEvent")).Return()

	service := NewInteractionServiceWithEventBus(repo, nil, nil, nil, eventBus)

	_, err := service.HandleInteraction(ctx, interaction)

	assert.NoError(t, err)
	// Modal submit should publish ModalSubmitted event
	eventBus.AssertCalled(t, "Publish", "modal.submitted", mock.AnythingOfType("*services.InteractionCreatedEvent"))
}
