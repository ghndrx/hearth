package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockSavedMessagesService mocks the SavedMessagesService for testing
type MockSavedMessagesService struct {
	mock.Mock
}

func (m *MockSavedMessagesService) SaveMessage(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error) {
	args := m.Called(ctx, userID, messageID, note)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesService) GetSavedMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesService) GetSavedMessage(ctx context.Context, userID, savedID uuid.UUID) (*models.SavedMessage, error) {
	args := m.Called(ctx, userID, savedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesService) UpdateSavedMessageNote(ctx context.Context, userID, savedID uuid.UUID, note *string) (*models.SavedMessage, error) {
	args := m.Called(ctx, userID, savedID, note)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SavedMessage), args.Error(1)
}

func (m *MockSavedMessagesService) RemoveSavedMessage(ctx context.Context, userID, savedID uuid.UUID) error {
	args := m.Called(ctx, userID, savedID)
	return args.Error(0)
}

func (m *MockSavedMessagesService) RemoveSavedMessageByMessageID(ctx context.Context, userID, messageID uuid.UUID) error {
	args := m.Called(ctx, userID, messageID)
	return args.Error(0)
}

func (m *MockSavedMessagesService) IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, messageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSavedMessagesService) GetSavedCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// testSavedMessagesHandler creates a test saved messages handler with mocks
type testSavedMessagesHandler struct {
	handler *SavedMessagesHandler
	service *MockSavedMessagesService
	app     *fiber.App
	userID  uuid.UUID
}

func newTestSavedMessagesHandler() *testSavedMessagesHandler {
	service := new(MockSavedMessagesService)
	handler := NewSavedMessagesHandler(service)

	app := fiber.New()
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Post("/users/@me/saved-messages", handler.SaveMessage)
	app.Get("/users/@me/saved-messages", handler.GetSavedMessages)
	app.Get("/users/@me/saved-messages/count", handler.GetSavedCount)
	app.Get("/users/@me/saved-messages/check/:messageId", handler.IsSaved)
	app.Get("/users/@me/saved-messages/:id", handler.GetSavedMessage)
	app.Patch("/users/@me/saved-messages/:id", handler.UpdateSavedMessage)
	app.Delete("/users/@me/saved-messages/:id", handler.RemoveSavedMessage)
	app.Delete("/users/@me/saved-messages/message/:messageId", handler.RemoveSavedMessageByMessage)

	return &testSavedMessagesHandler{
		handler: handler,
		service: service,
		app:     app,
		userID:  userID,
	}
}

func TestSavedMessagesHandler_SaveMessage(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()
	note := "Important message"
	saved := &models.SavedMessage{
		ID:        uuid.New(),
		UserID:    th.userID,
		MessageID: messageID,
		Note:      &note,
		CreatedAt: time.Now(),
		Message: &models.Message{
			ID:      messageID,
			Content: "Test message content",
		},
	}

	th.service.On("SaveMessage", mock.Anything, th.userID, messageID, &note).Return(saved, nil)

	body := map[string]interface{}{
		"message_id": messageID.String(),
		"note":       note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result models.SavedMessage
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, saved.ID, result.ID)
	assert.Equal(t, *saved.Note, *result.Note)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_SaveMessage_WithoutNote(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()
	saved := &models.SavedMessage{
		ID:        uuid.New(),
		UserID:    th.userID,
		MessageID: messageID,
		Note:      nil,
		CreatedAt: time.Now(),
	}

	th.service.On("SaveMessage", mock.Anything, th.userID, messageID, (*string)(nil)).Return(saved, nil)

	body := map[string]interface{}{
		"message_id": messageID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_SaveMessage_InvalidBody(t *testing.T) {
	th := newTestSavedMessagesHandler()

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid request body", result["error"])
}

func TestSavedMessagesHandler_SaveMessage_MissingMessageID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	body := map[string]interface{}{
		"note": "Some note",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "message_id is required", result["error"])
}

func TestSavedMessagesHandler_SaveMessage_InvalidMessageID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	body := map[string]interface{}{
		"message_id": "not-a-uuid",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid message_id format", result["error"])
}

func TestSavedMessagesHandler_SaveMessage_NoteTooLong(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()
	longNote := make([]byte, 501)
	for i := range longNote {
		longNote[i] = 'a'
	}
	note := string(longNote)

	body := map[string]interface{}{
		"message_id": messageID.String(),
		"note":       note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "note must be 500 characters or less", result["error"])
}

func TestSavedMessagesHandler_SaveMessage_MessageNotFound(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("SaveMessage", mock.Anything, th.userID, messageID, (*string)(nil)).Return(nil, services.ErrMessageNotFound)

	body := map[string]interface{}{
		"message_id": messageID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "message not found", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessages(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID1 := uuid.New()
	messageID2 := uuid.New()
	savedMessages := []*models.SavedMessage{
		{
			ID:        uuid.New(),
			UserID:    th.userID,
			MessageID: messageID1,
			CreatedAt: time.Now(),
			Message: &models.Message{
				ID:      messageID1,
				Content: "First message",
			},
		},
		{
			ID:        uuid.New(),
			UserID:    th.userID,
			MessageID: messageID2,
			CreatedAt: time.Now(),
			Message: &models.Message{
				ID:      messageID2,
				Content: "Second message",
			},
		},
	}

	th.service.On("GetSavedMessages", mock.Anything, th.userID, mock.Anything).Return(savedMessages, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*models.SavedMessage
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Len(t, result, 2)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessages_WithPagination(t *testing.T) {
	th := newTestSavedMessagesHandler()

	beforeID := uuid.New()
	savedMessages := []*models.SavedMessage{}

	th.service.On("GetSavedMessages", mock.Anything, th.userID, mock.MatchedBy(func(opts *models.SavedMessagesQueryOptions) bool {
		return opts.Before != nil && *opts.Before == beforeID && opts.Limit == 25
	})).Return(savedMessages, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages?before="+beforeID.String()+"&limit=25", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessages_Error(t *testing.T) {
	th := newTestSavedMessagesHandler()

	th.service.On("GetSavedMessages", mock.Anything, th.userID, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get saved messages", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessage(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	messageID := uuid.New()
	note := "My note"
	saved := &models.SavedMessage{
		ID:        savedID,
		UserID:    th.userID,
		MessageID: messageID,
		Note:      &note,
		CreatedAt: time.Now(),
		Message: &models.Message{
			ID:      messageID,
			Content: "Test content",
		},
	}

	th.service.On("GetSavedMessage", mock.Anything, th.userID, savedID).Return(saved, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.SavedMessage
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, savedID, result.ID)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessage_NotFound(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("GetSavedMessage", mock.Anything, th.userID, savedID).Return(nil, services.ErrSavedMessageNotFound)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "saved message not found", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessage_Unauthorized(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("GetSavedMessage", mock.Anything, th.userID, savedID).Return(nil, services.ErrUnauthorized)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "access denied", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_UpdateSavedMessage(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	messageID := uuid.New()
	newNote := "Updated note"
	saved := &models.SavedMessage{
		ID:        savedID,
		UserID:    th.userID,
		MessageID: messageID,
		Note:      &newNote,
		CreatedAt: time.Now(),
	}

	th.service.On("UpdateSavedMessageNote", mock.Anything, th.userID, savedID, &newNote).Return(saved, nil)

	body := map[string]interface{}{
		"note": newNote,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.SavedMessage
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, newNote, *result.Note)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_UpdateSavedMessage_NoteTooLong(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	longNote := make([]byte, 501)
	for i := range longNote {
		longNote[i] = 'a'
	}
	note := string(longNote)

	body := map[string]interface{}{
		"note": note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "note must be 500 characters or less", result["error"])
}

func TestSavedMessagesHandler_UpdateSavedMessage_NotFound(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	note := "Some note"

	th.service.On("UpdateSavedMessageNote", mock.Anything, th.userID, savedID, &note).Return(nil, services.ErrSavedMessageNotFound)

	body := map[string]interface{}{
		"note": note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessage(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("RemoveSavedMessage", mock.Anything, th.userID, savedID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessage_NotFound(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("RemoveSavedMessage", mock.Anything, th.userID, savedID).Return(services.ErrSavedMessageNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "saved message not found", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessage_Unauthorized(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("RemoveSavedMessage", mock.Anything, th.userID, savedID).Return(services.ErrUnauthorized)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "access denied", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessageByMessage(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("RemoveSavedMessageByMessageID", mock.Anything, th.userID, messageID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/message/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessageByMessage_NotFound(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("RemoveSavedMessageByMessageID", mock.Anything, th.userID, messageID).Return(services.ErrSavedMessageNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/message/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_IsSaved(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("IsSaved", mock.Anything, th.userID, messageID).Return(true, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/check/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["saved"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_IsSaved_NotSaved(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("IsSaved", mock.Anything, th.userID, messageID).Return(false, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/check/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, false, result["saved"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_IsSaved_InvalidMessageID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/check/not-a-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid message id", result["error"])
}

func TestSavedMessagesHandler_GetSavedCount(t *testing.T) {
	th := newTestSavedMessagesHandler()

	th.service.On("GetSavedCount", mock.Anything, th.userID).Return(42, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/count", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, float64(42), result["count"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedCount_Error(t *testing.T) {
	th := newTestSavedMessagesHandler()

	th.service.On("GetSavedCount", mock.Anything, th.userID).Return(0, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/count", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get saved count", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessage_InvalidID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/not-a-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid saved message id", result["error"])
}

func TestSavedMessagesHandler_UpdateSavedMessage_InvalidBody(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid request body", result["error"])
}

func TestSavedMessagesHandler_RemoveSavedMessage_InvalidID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/not-a-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid saved message id", result["error"])
}

func TestSavedMessagesHandler_RemoveSavedMessageByMessage_InvalidID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/message/not-a-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid message id", result["error"])
}

func TestSavedMessagesHandler_SaveMessage_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("SaveMessage", mock.Anything, th.userID, messageID, (*string)(nil)).Return(nil, assert.AnError)

	body := map[string]interface{}{
		"message_id": messageID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/saved-messages", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to save message", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_GetSavedMessage_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("GetSavedMessage", mock.Anything, th.userID, savedID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get saved message", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_UpdateSavedMessage_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	note := "Some note"

	th.service.On("UpdateSavedMessageNote", mock.Anything, th.userID, savedID, &note).Return(nil, assert.AnError)

	body := map[string]interface{}{
		"note": note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to update saved message", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessage_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()

	th.service.On("RemoveSavedMessage", mock.Anything, th.userID, savedID).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/"+savedID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to remove saved message", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_RemoveSavedMessageByMessage_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("RemoveSavedMessageByMessageID", mock.Anything, th.userID, messageID).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/saved-messages/message/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to remove saved message", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_IsSaved_ServiceError(t *testing.T) {
	th := newTestSavedMessagesHandler()

	messageID := uuid.New()

	th.service.On("IsSaved", mock.Anything, th.userID, messageID).Return(false, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/saved-messages/check/"+messageID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to check saved status", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_UpdateSavedMessage_Unauthorized(t *testing.T) {
	th := newTestSavedMessagesHandler()

	savedID := uuid.New()
	note := "Some note"

	th.service.On("UpdateSavedMessageNote", mock.Anything, th.userID, savedID, &note).Return(nil, services.ErrUnauthorized)

	body := map[string]interface{}{
		"note": note,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/"+savedID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "access denied", result["error"])

	th.service.AssertExpectations(t)
}

func TestSavedMessagesHandler_UpdateSavedMessage_InvalidID(t *testing.T) {
	th := newTestSavedMessagesHandler()

	body := map[string]interface{}{
		"note": "Some note",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/users/@me/saved-messages/not-a-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid saved message id", result["error"])
}
