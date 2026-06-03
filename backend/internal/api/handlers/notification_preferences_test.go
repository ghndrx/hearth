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
)

// MockNotificationCoordinator mocks the NotificationCoordinatorInterface
type MockNotificationCoordinator struct {
	mock.Mock
}

func (m *MockNotificationCoordinator) GetChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID) (*models.ChannelNotificationPreference, error) {
	args := m.Called(ctx, userID, channelID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChannelNotificationPreference), args.Error(1)
}

func (m *MockNotificationCoordinator) UpdateChannelPreference(ctx context.Context, userID, channelID, serverID uuid.UUID, req *models.UpdateChannelNotificationPreferenceRequest) (*models.ChannelNotificationPreference, error) {
	args := m.Called(ctx, userID, channelID, serverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChannelNotificationPreference), args.Error(1)
}

func (m *MockNotificationCoordinator) GetServerPreference(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerNotificationPreference, error) {
	args := m.Called(ctx, userID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerNotificationPreference), args.Error(1)
}

func (m *MockNotificationCoordinator) UpdateServerPreference(ctx context.Context, userID, serverID uuid.UUID, req *models.UpdateServerNotificationPreferenceRequest) (*models.ServerNotificationPreference, error) {
	args := m.Called(ctx, userID, serverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerNotificationPreference), args.Error(1)
}

// testNotificationPrefsHandler creates a test notification preferences handler with mocks
type testNotificationPrefsHandler struct {
	channelHandler *NotificationPreferenceHandler
	serverHandler  *NotificationPreferenceHandler
	coordinator    *MockNotificationCoordinator
	app            *fiber.App
	userID         uuid.UUID
}

func newTestNotificationPrefsHandler(tb testing.TB) *testNotificationPrefsHandler {
	coordinator := new(MockNotificationCoordinator)
	channelHandler := &NotificationPreferenceHandler{coordinator: coordinator}
	serverHandler := &NotificationPreferenceHandler{coordinator: coordinator}

	app := fiber.New()
	tb.Cleanup(func() { _ = app.Shutdown() })
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes for channel preferences
	app.Get("/users/@me/channels/:channelId/notifications", channelHandler.GetChannelNotificationPreference)
	app.Patch("/users/@me/channels/:channelId/notifications", channelHandler.UpdateChannelNotificationPreference)

	// Setup routes for server preferences
	app.Get("/users/@me/servers/:serverId/notifications", serverHandler.GetServerNotificationPreference)
	app.Patch("/users/@me/servers/:serverId/notifications", serverHandler.UpdateServerNotificationPreference)

	return &testNotificationPrefsHandler{
		channelHandler: channelHandler,
		serverHandler:  serverHandler,
		coordinator:    coordinator,
		app:            app,
		userID:         userID,
	}
}

func TestChannelNotificationPreferenceHandler_Get_Success(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	now := time.Now()
	pref := &models.ChannelNotificationPreference{
		ID:                 uuid.New(),
		UserID:             th.userID,
		ChannelID:          channelID,
		ServerID:           serverID,
		EnableMentions:     true,
		EnableMessages:     true,
		EnableReactions:    true,
		EnableThreads:      true,
		EnablePins:         true,
		EnableVoiceActivity: true,
		DeliveryMode:       models.DeliveryBatched,
		Muted:              false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	th.coordinator.On("GetChannelPreference", mock.Anything, th.userID, channelID, serverID).Return(pref, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ChannelNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, th.userID, result.UserID)
	assert.Equal(t, channelID, result.ChannelID)
	assert.Equal(t, serverID, result.ServerID)
	assert.True(t, result.EnableMentions)

	th.coordinator.AssertExpectations(t)
}

func TestChannelNotificationPreferenceHandler_Get_InvalidChannelID(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/users/@me/channels/invalid-uuid/notifications?server_id="+serverID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChannelNotificationPreferenceHandler_Get_InvalidServerID(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/users/@me/channels/"+channelID.String()+"/notifications?server_id=invalid-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChannelNotificationPreferenceHandler_Update_Success(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	enableMentions := false
	updateReq := &models.UpdateChannelNotificationPreferenceRequest{
		EnableMentions: &enableMentions,
	}

	now := time.Now()
	updatedPref := &models.ChannelNotificationPreference{
		ID:             uuid.New(),
		UserID:         th.userID,
		ChannelID:      channelID,
		ServerID:       serverID,
		EnableMentions: false,
		EnableMessages: true,
		DeliveryMode:   models.DeliveryBatched,
		Muted:          false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	th.coordinator.On("UpdateChannelPreference", mock.Anything, th.userID, channelID, serverID, updateReq).Return(updatedPref, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ChannelNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.False(t, result.EnableMentions)

	th.coordinator.AssertExpectations(t)
}

func TestChannelNotificationPreferenceHandler_Update_InvalidChannelID(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	enableMentions := false
	body, _ := json.Marshal(&models.UpdateChannelNotificationPreferenceRequest{EnableMentions: &enableMentions})
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/invalid-uuid/notifications?server_id="+serverID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChannelNotificationPreferenceHandler_Update_InvalidBody(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServerNotificationPreferenceHandler_Get_Success(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	now := time.Now()
	pref := &models.ServerNotificationPreference{
		ID:               uuid.New(),
		UserID:           th.userID,
		ServerID:         serverID,
		EnableMentions:   true,
		EnableMessages:   true,
		EnableReactions:  true,
		EnableThreads:    true,
		NotifyRoles:      nil,
		Muted:            false,
		MutedUntil:       nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	th.coordinator.On("GetServerPreference", mock.Anything, th.userID, serverID).Return(pref, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/servers/"+serverID.String()+"/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ServerNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, th.userID, result.UserID)
	assert.Equal(t, serverID, result.ServerID)
	assert.True(t, result.EnableMentions)

	th.coordinator.AssertExpectations(t)
}

func TestServerNotificationPreferenceHandler_Get_InvalidServerID(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/servers/invalid-uuid/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServerNotificationPreferenceHandler_Update_Success(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	enableMentions := false
	notifyRoles := []uuid.UUID{uuid.New()}
	updateReq := &models.UpdateServerNotificationPreferenceRequest{
		EnableMentions: &enableMentions,
		NotifyRoles:    notifyRoles,
	}

	now := time.Now()
	updatedPref := &models.ServerNotificationPreference{
		ID:             uuid.New(),
		UserID:         th.userID,
		ServerID:       serverID,
		EnableMentions: false,
		EnableMessages: true,
		EnableReactions: true,
		EnableThreads:  true,
		NotifyRoles:    notifyRoles,
		Muted:          false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	th.coordinator.On("UpdateServerPreference", mock.Anything, th.userID, serverID, updateReq).Return(updatedPref, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/servers/"+serverID.String()+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ServerNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.False(t, result.EnableMentions)
	assert.Equal(t, notifyRoles, result.NotifyRoles)

	th.coordinator.AssertExpectations(t)
}

func TestServerNotificationPreferenceHandler_Update_InvalidServerID(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	enableMentions := false
	body, _ := json.Marshal(&models.UpdateServerNotificationPreferenceRequest{EnableMentions: &enableMentions})
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/servers/invalid-uuid/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServerNotificationPreferenceHandler_Update_InvalidBody(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/servers/"+serverID.String()+"/notifications", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChannelNotificationPreferenceHandler_Get_CoordinatorError(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()

	th.coordinator.On("GetChannelPreference", mock.Anything, th.userID, channelID, serverID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.coordinator.AssertExpectations(t)
}

func TestChannelNotificationPreferenceHandler_Update_CoordinatorError(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	enableMentions := false
	updateReq := &models.UpdateChannelNotificationPreferenceRequest{
		EnableMentions: &enableMentions,
	}

	th.coordinator.On("UpdateChannelPreference", mock.Anything, th.userID, channelID, serverID, updateReq).Return(nil, assert.AnError)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.coordinator.AssertExpectations(t)
}

func TestServerNotificationPreferenceHandler_Get_CoordinatorError(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()

	th.coordinator.On("GetServerPreference", mock.Anything, th.userID, serverID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/servers/"+serverID.String()+"/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.coordinator.AssertExpectations(t)
}

func TestServerNotificationPreferenceHandler_Update_CoordinatorError(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	enableMentions := false
	updateReq := &models.UpdateServerNotificationPreferenceRequest{
		EnableMentions: &enableMentions,
	}

	th.coordinator.On("UpdateServerPreference", mock.Anything, th.userID, serverID, updateReq).Return(nil, assert.AnError)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/servers/"+serverID.String()+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	th.coordinator.AssertExpectations(t)
}

func TestChannelNotificationPreferenceHandler_Update_WithDeliveryMode(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	deliveryMode := models.DeliveryImmediate
	updateReq := &models.UpdateChannelNotificationPreferenceRequest{
		DeliveryMode: &deliveryMode,
	}

	now := time.Now()
	updatedPref := &models.ChannelNotificationPreference{
		ID:           uuid.New(),
		UserID:       th.userID,
		ChannelID:    channelID,
		ServerID:     serverID,
		DeliveryMode: models.DeliveryImmediate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	th.coordinator.On("UpdateChannelPreference", mock.Anything, th.userID, channelID, serverID, updateReq).Return(updatedPref, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ChannelNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, models.DeliveryImmediate, result.DeliveryMode)

	th.coordinator.AssertExpectations(t)
}

func TestChannelNotificationPreferenceHandler_Update_WithMute(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	channelID := uuid.New()
	serverID := uuid.New()
	muted := true
	updateReq := &models.UpdateChannelNotificationPreferenceRequest{
		Muted: &muted,
	}

	now := time.Now()
	updatedPref := &models.ChannelNotificationPreference{
		ID:        uuid.New(),
		UserID:    th.userID,
		ChannelID: channelID,
		ServerID:  serverID,
		Muted:     true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	th.coordinator.On("UpdateChannelPreference", mock.Anything, th.userID, channelID, serverID, updateReq).Return(updatedPref, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/channels/"+channelID.String()+"/notifications?server_id="+serverID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ChannelNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result.Muted)

	th.coordinator.AssertExpectations(t)
}

func TestServerNotificationPreferenceHandler_Update_WithMute(t *testing.T) {
	th := newTestNotificationPrefsHandler(t)

	serverID := uuid.New()
	muted := true
	updateReq := &models.UpdateServerNotificationPreferenceRequest{
		Muted: &muted,
	}

	now := time.Now()
	updatedPref := &models.ServerNotificationPreference{
		ID:        uuid.New(),
		UserID:    th.userID,
		ServerID:  serverID,
		Muted:     true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	th.coordinator.On("UpdateServerPreference", mock.Anything, th.userID, serverID, updateReq).Return(updatedPref, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/servers/"+serverID.String()+"/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ServerNotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result.Muted)

	th.coordinator.AssertExpectations(t)
}
