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

// =============================================================================
// Mock implementations for follow and reminders testing
// =============================================================================

// mockFollowRemindersDB implements HandlerDB for follow and reminders testing
type mockFollowRemindersDB struct {
	createFollowedChannelFunc func(ctx context.Context, follow *models.FollowedChannel) error
	deleteFollowedChannelFunc func(ctx context.Context, channelID, followerChannelID string) error
	getFollowedChannelsFunc    func(ctx context.Context, channelID string) ([]models.FollowedChannel, error)
	createReminderFunc         func(ctx context.Context, reminder *models.Reminder) error
	getUserRemindersFunc       func(ctx context.Context, userID string) ([]models.Reminder, error)
	deleteReminderFunc         func(ctx context.Context, reminderID, userID string) error
}

func (m *mockFollowRemindersDB) CreateFollowedChannel(ctx context.Context, follow *models.FollowedChannel) error {
	if m.createFollowedChannelFunc != nil {
		return m.createFollowedChannelFunc(ctx, follow)
	}
	return nil
}

func (m *mockFollowRemindersDB) DeleteFollowedChannel(ctx context.Context, channelID, followerChannelID string) error {
	if m.deleteFollowedChannelFunc != nil {
		return m.deleteFollowedChannelFunc(ctx, channelID, followerChannelID)
	}
	return nil
}

func (m *mockFollowRemindersDB) GetFollowedChannels(ctx context.Context, channelID string) ([]models.FollowedChannel, error) {
	if m.getFollowedChannelsFunc != nil {
		return m.getFollowedChannelsFunc(ctx, channelID)
	}
	return nil, nil
}

func (m *mockFollowRemindersDB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	if m.createReminderFunc != nil {
		return m.createReminderFunc(ctx, reminder)
	}
	return nil
}

func (m *mockFollowRemindersDB) GetUserReminders(ctx context.Context, userID string) ([]models.Reminder, error) {
	if m.getUserRemindersFunc != nil {
		return m.getUserRemindersFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockFollowRemindersDB) DeleteReminder(ctx context.Context, reminderID, userID string) error {
	if m.deleteReminderFunc != nil {
		return m.deleteReminderFunc(ctx, reminderID, userID)
	}
	return nil
}

// Unused mock methods to satisfy HandlerDB interface
func (m *mockFollowRemindersDB) GetPinnedMessages(ctx context.Context, channelID string) ([]*models.Message, error) {
	return nil, nil
}
func (m *mockFollowRemindersDB) CreatePin(ctx context.Context, pin *models.Pin) error {
	return nil
}
func (m *mockFollowRemindersDB) DeletePin(ctx context.Context, messageID string) error {
	return nil
}
func (m *mockFollowRemindersDB) GetSounds(ctx context.Context, serverID string) ([]models.SoundboardSound, error) {
	return nil, nil
}
func (m *mockFollowRemindersDB) CreateSound(ctx context.Context, sound *models.Sound) error {
	return nil
}
func (m *mockFollowRemindersDB) DeleteSound(ctx context.Context, soundID string) error {
	return nil
}
func (m *mockFollowRemindersDB) GetSound(ctx context.Context, soundID string) (*models.Sound, error) {
	return nil, nil
}

// followRemindersRouter creates a ServeMux with proper routing for follow and reminders handler tests
func followRemindersRouter(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// POST /channels/{channel_id}/follow
	mux.HandleFunc("POST /channels/{channel_id}/follow", func(w http.ResponseWriter, r *http.Request) {
		handler.FollowChannel(w, r)
	})

	// DELETE /channels/{channel_id}/follow
	mux.HandleFunc("DELETE /channels/{channel_id}/follow", func(w http.ResponseWriter, r *http.Request) {
		handler.UnfollowChannel(w, r)
	})

	// GET /channels/{channel_id}/followers
	mux.HandleFunc("GET /channels/{channel_id}/followers", func(w http.ResponseWriter, r *http.Request) {
		handler.GetFollowers(w, r)
	})

	// POST /reminders
	mux.HandleFunc("POST /reminders", func(w http.ResponseWriter, r *http.Request) {
		handler.CreateReminder(w, r)
	})

	// GET /reminders
	mux.HandleFunc("GET /reminders", func(w http.ResponseWriter, r *http.Request) {
		handler.GetReminders(w, r)
	})

	// DELETE /reminders/{id}
	mux.HandleFunc("DELETE /reminders/{id}", func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteReminder(w, r)
	})

	return mux
}

// =============================================================================
// Tests for FollowChannel
// =============================================================================

func TestFollowHandler_FollowChannel_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New()
	followerChannelID := uuid.New()

	var capturedFollow *models.FollowedChannel
	db.createFollowedChannelFunc = func(ctx context.Context, follow *models.FollowedChannel) error {
		capturedFollow = follow
		return nil
	}

	body := `{"follower_channel_id":"` + followerChannelID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.FollowedChannel
	json.NewDecoder(w.Body).Decode(&result)
	assert.Equal(t, channelID, result.ChannelID)
	assert.Equal(t, followerChannelID, result.FollowerChannelID)
	assert.NotNil(t, capturedFollow)
	assert.Equal(t, channelID, capturedFollow.ChannelID)
	assert.Equal(t, followerChannelID, capturedFollow.FollowerChannelID)
}

func TestFollowHandler_FollowChannel_InvalidJSON(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New().String()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFollowHandler_FollowChannel_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New().String()
	followerChannelID := uuid.New().String()

	db.createFollowedChannelFunc = func(ctx context.Context, follow *models.FollowedChannel) error {
		return errors.New("database connection failed")
	}

	body := `{"follower_channel_id":"` + followerChannelID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Tests for UnfollowChannel
// =============================================================================

func TestFollowHandler_UnfollowChannel_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New().String()
	followerChannelID := uuid.New().String()

	var capturedChannelID, capturedFollowerID string
	db.deleteFollowedChannelFunc = func(ctx context.Context, chID, flwID string) error {
		capturedChannelID = chID
		capturedFollowerID = flwID
		return nil
	}

	body := `{"follower_channel_id":"` + followerChannelID + `"}`
	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, channelID, capturedChannelID)
	assert.Equal(t, followerChannelID, capturedFollowerID)
}

func TestFollowHandler_UnfollowChannel_InvalidJSON(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New().String()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFollowHandler_UnfollowChannel_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New().String()
	followerChannelID := uuid.New().String()

	db.deleteFollowedChannelFunc = func(ctx context.Context, chID, flwID string) error {
		return errors.New("database error")
	}

	body := `{"follower_channel_id":"` + followerChannelID + `"}`
	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID+"/follow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Tests for GetFollowers
// =============================================================================

func TestFollowHandler_GetFollowers_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New()
	followers := []models.FollowedChannel{
		{
			ChannelID:         channelID,
			FollowerChannelID: uuid.New(),
			CreatedAt:         time.Now(),
		},
		{
			ChannelID:         channelID,
			FollowerChannelID: uuid.New(),
			CreatedAt:         time.Now(),
		},
	}

	db.getFollowedChannelsFunc = func(ctx context.Context, chID string) ([]models.FollowedChannel, error) {
		return followers, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/followers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.FollowedChannel
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, followers[0].ChannelID, result[0].ChannelID)
	assert.Equal(t, followers[1].ChannelID, result[1].ChannelID)
}

func TestFollowHandler_GetFollowers_EmptyList(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New()

	db.getFollowedChannelsFunc = func(ctx context.Context, chID string) ([]models.FollowedChannel, error) {
		return []models.FollowedChannel{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/followers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.FollowedChannel
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 0)
}

func TestFollowHandler_GetFollowers_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	channelID := uuid.New()

	db.getFollowedChannelsFunc = func(ctx context.Context, chID string) ([]models.FollowedChannel, error) {
		return nil, errors.New("database connection failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/followers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Tests for CreateReminder
// =============================================================================

func TestReminderHandler_CreateReminder_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()
	remindAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	messageID := uuid.New().String()
	channelID := uuid.New().String()

	var capturedReminder *models.Reminder
	db.createReminderFunc = func(ctx context.Context, reminder *models.Reminder) error {
		capturedReminder = reminder
		return nil
	}

	body := `{"remind_at":"` + remindAt + `","message_id":"` + messageID + `","channel_id":"` + channelID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/reminders", strings.NewReader(body))
	req = requestWithUserID(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.Reminder
	json.NewDecoder(w.Body).Decode(&result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, messageID, result.MessageID)
	assert.Equal(t, channelID, result.ChannelID)
	assert.NotNil(t, capturedReminder)
	assert.Equal(t, userID, capturedReminder.UserID)
}

func TestReminderHandler_CreateReminder_InvalidJSON(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/reminders", strings.NewReader(body))
	req = requestWithUserID(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReminderHandler_CreateReminder_InvalidRemindAtFormat(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()

	body := `{"remind_at":"invalid-date","message_id":"123","channel_id":"456"}`
	req := httptest.NewRequest(http.MethodPost, "/reminders", strings.NewReader(body))
	req = requestWithUserID(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid remind_at format")
}

func TestReminderHandler_CreateReminder_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()
	remindAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	db.createReminderFunc = func(ctx context.Context, reminder *models.Reminder) error {
		return errors.New("database connection failed")
	}

	body := `{"remind_at":"` + remindAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/reminders", strings.NewReader(body))
	req = requestWithUserID(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Tests for GetReminders
// =============================================================================

func TestReminderHandler_GetReminders_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()
	reminders := []models.Reminder{
		{
			ID:        uuid.New().String(),
			UserID:    userID,
			MessageID: uuid.New().String(),
			ChannelID: uuid.New().String(),
			RemindAt:  time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New().String(),
			UserID:    userID,
			MessageID: uuid.New().String(),
			ChannelID: uuid.New().String(),
			RemindAt:  time.Now().Add(48 * time.Hour),
			CreatedAt: time.Now(),
		},
	}

	db.getUserRemindersFunc = func(ctx context.Context, uID string) ([]models.Reminder, error) {
		return reminders, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/reminders", nil)
	req = requestWithUserID(req, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.Reminder
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, reminders[0].UserID, result[0].UserID)
	assert.Equal(t, reminders[1].UserID, result[1].UserID)
}

func TestReminderHandler_GetReminders_EmptyList(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()

	db.getUserRemindersFunc = func(ctx context.Context, uID string) ([]models.Reminder, error) {
		return []models.Reminder{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/reminders", nil)
	req = requestWithUserID(req, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.Reminder
	json.NewDecoder(w.Body).Decode(&result)
	assert.Len(t, result, 0)
}

func TestReminderHandler_GetReminders_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	userID := uuid.New().String()

	db.getUserRemindersFunc = func(ctx context.Context, uID string) ([]models.Reminder, error) {
		return nil, errors.New("database connection failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/reminders", nil)
	req = requestWithUserID(req, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Tests for DeleteReminder
// =============================================================================

func TestReminderHandler_DeleteReminder_Success(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	reminderID := uuid.New().String()
	userID := uuid.New().String()

	var capturedReminderID, capturedUserID string
	db.deleteReminderFunc = func(ctx context.Context, rID, uID string) error {
		capturedReminderID = rID
		capturedUserID = uID
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/reminders/"+reminderID, nil)
	req = requestWithUserID(req, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, reminderID, capturedReminderID)
	assert.Equal(t, userID, capturedUserID)
}

func TestReminderHandler_DeleteReminder_DBError(t *testing.T) {
	db := &mockFollowRemindersDB{}
	handler := &Handler{db: db}
	mux := followRemindersRouter(handler)

	reminderID := uuid.New().String()
	userID := uuid.New().String()

	db.deleteReminderFunc = func(ctx context.Context, rID, uID string) error {
		return errors.New("database error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/reminders/"+reminderID, nil)
	req = requestWithUserID(req, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Helper functions
// =============================================================================

// requestWithUserID creates a request with userID in context
func requestWithUserID(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), "userID", userID)
	return req.WithContext(ctx)
}
