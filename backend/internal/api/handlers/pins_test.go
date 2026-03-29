package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// mockPinDB implements HandlerDB for pins testing
type mockPinDB struct {
	getPinnedMessagesFunc func(ctx context.Context, channelID string) ([]*models.Message, error)
	createPinFunc         func(ctx context.Context, pin *models.Pin) error
	deletePinFunc         func(ctx context.Context, messageID string) error
}

func (m *mockPinDB) GetPinnedMessages(ctx context.Context, channelID string) ([]*models.Message, error) {
	if m.getPinnedMessagesFunc != nil {
		return m.getPinnedMessagesFunc(ctx, channelID)
	}
	return nil, nil
}

func (m *mockPinDB) CreatePin(ctx context.Context, pin *models.Pin) error {
	if m.createPinFunc != nil {
		return m.createPinFunc(ctx, pin)
	}
	return nil
}

func (m *mockPinDB) DeletePin(ctx context.Context, messageID string) error {
	if m.deletePinFunc != nil {
		return m.deletePinFunc(ctx, messageID)
	}
	return nil
}

// Unused mock methods to satisfy HandlerDB interface
func (m *mockPinDB) CreateFollowedChannel(ctx context.Context, follow *models.FollowedChannel) error {
	return nil
}
func (m *mockPinDB) DeleteFollowedChannel(ctx context.Context, channelID, followerChannelID string) error {
	return nil
}
func (m *mockPinDB) GetFollowedChannels(ctx context.Context, channelID string) ([]models.FollowedChannel, error) {
	return nil, nil
}
func (m *mockPinDB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	return nil
}
func (m *mockPinDB) GetUserReminders(ctx context.Context, userID string) ([]models.Reminder, error) {
	return nil, nil
}
func (m *mockPinDB) DeleteReminder(ctx context.Context, reminderID, userID string) error {
	return nil
}
func (m *mockPinDB) GetSounds(ctx context.Context, serverID string) ([]models.SoundboardSound, error) {
	return nil, nil
}
func (m *mockPinDB) CreateSound(ctx context.Context, sound *models.Sound) error {
	return nil
}
func (m *mockPinDB) DeleteSound(ctx context.Context, soundID string) error {
	return nil
}
func (m *mockPinDB) GetSound(ctx context.Context, soundID string) (*models.Sound, error) {
	return nil, nil
}

// testRouter creates a ServeMux with proper routing for pins handler tests
func testRouter(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// GET /channels/{channel_id}/pins
	mux.HandleFunc("GET /channels/{channel_id}/pins", func(w http.ResponseWriter, r *http.Request) {
		handler.GetPinnedMessages(w, r)
	})

	// POST /channels/{channel_id}/messages/{message_id}/pin
	mux.HandleFunc("POST /channels/{channel_id}/messages/{message_id}/pin", func(w http.ResponseWriter, r *http.Request) {
		handler.PinMessage(w, r)
	})

	// DELETE /channels/{channel_id}/messages/{message_id}/unpin
	mux.HandleFunc("DELETE /channels/{channel_id}/messages/{message_id}/unpin", func(w http.ResponseWriter, r *http.Request) {
		handler.UnpinMessage(w, r)
	})

	return mux
}

// =============================================================================
// Tests for GetPinnedMessages (pins handler)
// =============================================================================

func TestPinsHandler_GetPinnedMessages_Success(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := "test-channel"
	messages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Content:   "Pinned message 1",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			ChannelID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Content:   "Pinned message 2",
			CreatedAt: time.Now(),
		},
	}

	db.getPinnedMessagesFunc = func(ctx context.Context, chID string) ([]*models.Message, error) {
		return messages, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID+"/pins", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []*models.Message
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, messages[0].Content, result[0].Content)
	assert.Equal(t, messages[1].Content, result[1].Content)
}

func TestPinsHandler_GetPinnedMessages_Error(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := "test-channel"

	db.getPinnedMessagesFunc = func(ctx context.Context, chID string) ([]*models.Message, error) {
		return nil, errors.New("database connection failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID+"/pins", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// http.Error writes plain text, not JSON
	assert.Equal(t, "database connection failed\n", w.Body.String())
}

func TestPinsHandler_GetPinnedMessages_EmptyList(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := "test-channel"

	db.getPinnedMessagesFunc = func(ctx context.Context, chID string) ([]*models.Message, error) {
		return []*models.Message{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID+"/pins", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []*models.Message
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 0)
}

// =============================================================================
// Tests for PinMessage (pins handler)
// =============================================================================

func TestPinsHandler_PinMessage_Success(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := uuid.New()
	messageID := uuid.New()
	userID := uuid.New()

	var capturedPin *models.Pin
	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		capturedPin = pin
		return nil
	}

	// Create request with userID in context
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	assert.NotNil(t, capturedPin)
	assert.Equal(t, channelID, capturedPin.ChannelID)
	assert.Equal(t, messageID, capturedPin.MessageID)
	assert.Equal(t, userID, capturedPin.PinnedBy)
}

func TestPinsHandler_PinMessage_InvalidChannelID(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	messageID := uuid.New()
	userID := uuid.New()

	// Note: the handler ignores uuid.Parse errors, so invalid UUIDs result in zero UUIDs
	// The mock captures whatever was passed
	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/channels/invalid-uuid/messages/"+messageID.String()+"/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Handler doesn't validate UUIDs - it just proceeds with zero UUID
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPinsHandler_PinMessage_InvalidMessageID(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := uuid.New()
	userID := uuid.New()

	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/invalid-uuid/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Handler doesn't validate UUIDs - it just proceeds with zero UUID
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPinsHandler_PinMessage_DBError(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := uuid.New()
	messageID := uuid.New()
	userID := uuid.New()

	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		return errors.New("failed to create pin")
	}

	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// http.Error writes plain text, not JSON
	assert.Equal(t, "failed to create pin\n", w.Body.String())
}

// =============================================================================
// Tests for UnpinMessage (pins handler)
// =============================================================================

func TestPinsHandler_UnpinMessage_Success(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	messageID := "test-message-id"

	var capturedMessageID string
	db.deletePinFunc = func(ctx context.Context, msgID string) error {
		capturedMessageID = msgID
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/channels/some-channel/messages/"+messageID+"/unpin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, messageID, capturedMessageID)
}

func TestPinsHandler_UnpinMessage_DBError(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	messageID := "test-message-id"

	db.deletePinFunc = func(ctx context.Context, msgID string) error {
		return errors.New("failed to delete pin")
	}

	req := httptest.NewRequest(http.MethodDelete, "/channels/some-channel/messages/"+messageID+"/unpin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to delete pin\n", w.Body.String())
}

func TestPinsHandler_UnpinMessage_NotFound(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	messageID := "nonexistent-message"

	db.deletePinFunc = func(ctx context.Context, msgID string) error {
		return errors.New("pin not found")
	}

	req := httptest.NewRequest(http.MethodDelete, "/channels/some-channel/messages/"+messageID+"/unpin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "pin not found\n", w.Body.String())
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestPinsHandler_GetPinnedMessages_NilDB(t *testing.T) {
	handler := &Handler{db: nil}
	mux := testRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/channels/test-channel/pins", nil)
	w := httptest.NewRecorder()

	assert.Panics(t, func() {
		mux.ServeHTTP(w, req)
	})
}

func TestPinsHandler_PinMessage_NilDB(t *testing.T) {
	handler := &Handler{db: nil}
	mux := testRouter(handler)

	channelID := uuid.New()
	messageID := uuid.New()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", userID.String())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	assert.Panics(t, func() {
		mux.ServeHTTP(w, req)
	})
}

func TestPinsHandler_UnpinMessage_NilDB(t *testing.T) {
	handler := &Handler{db: nil}
	mux := testRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/channels/some-channel/messages/some-message/unpin", nil)
	w := httptest.NewRecorder()

	assert.Panics(t, func() {
		mux.ServeHTTP(w, req)
	})
}

// =============================================================================
// Additional edge case tests
// =============================================================================

func TestPinsHandler_GetPinnedMessages_NoMock(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	// Don't set getPinnedMessagesFunc - will use default (returns nil)
	req := httptest.NewRequest(http.MethodGet, "/channels/test-channel/pins", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// With no mock set, returns nil results
	assert.Equal(t, http.StatusOK, w.Code)
	var result []*models.Message
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 0)
}

func TestPinsHandler_PinMessage_NoUserInContext(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	channelID := uuid.New()
	messageID := uuid.New()

	var capturedPin *models.Pin
	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		capturedPin = pin
		return nil
	}

	// Request WITHOUT userID in context
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/messages/"+messageID.String()+"/pin", nil)
	// Don't add userID to context

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	// User ID will be empty string, uuid.Parse("") returns zero UUID
	assert.NotNil(t, capturedPin)
	assert.Equal(t, uuid.Nil, capturedPin.PinnedBy)
}

func TestPinsHandler_PinMessage_BothInvalidUUIDs(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	db.createPinFunc = func(ctx context.Context, pin *models.Pin) error {
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/channels/invalid-channel/messages/invalid-message/pin", nil)
	ctx := context.WithValue(req.Context(), "userID", "invalid-user")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Handler proceeds with zero UUIDs
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPinsHandler_UnpinMessage_NoMock(t *testing.T) {
	db := &mockPinDB{}
	handler := &Handler{db: db}
	mux := testRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/channels/some-channel/messages/some-message/unpin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// With no mock set, returns nil (no error)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// Helper to suppress unused import warning
var _ = strings.TrimSpace
