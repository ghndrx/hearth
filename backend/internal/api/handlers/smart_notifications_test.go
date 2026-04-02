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

// MockSmartNotificationService mocks SmartNotificationServiceInterface
type MockSmartNotificationService struct {
	mock.Mock
}

func (m *MockSmartNotificationService) ScoreNotification(ctx context.Context, input *models.PriorityScoringInput) (*models.SmartNotification, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SmartNotification), args.Error(1)
}

func (m *MockSmartNotificationService) SnoozeNotifications(ctx context.Context, userID uuid.UUID, req *models.SnoozeRequest) (*models.SnoozeConfig, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SnoozeConfig), args.Error(1)
}

func (m *MockSmartNotificationService) UnsnoozeNotifications(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) error {
	args := m.Called(ctx, userID, serverID, channelID)
	return args.Error(0)
}

func (m *MockSmartNotificationService) IsNotificationSnoozed(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, *time.Time, error) {
	args := m.Called(ctx, userID, serverID, channelID)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*time.Time), args.Error(2)
}

func (m *MockSmartNotificationService) MuteNotifications(ctx context.Context, userID uuid.UUID, req *models.MuteRequest) (*models.MuteConfig, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MuteConfig), args.Error(1)
}

func (m *MockSmartNotificationService) IsNotificationMuted(ctx context.Context, userID uuid.UUID, serverID, channelID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, serverID, channelID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSmartNotificationService) TrackNotificationClick(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) error {
	args := m.Called(ctx, userID, notificationID)
	return args.Error(0)
}

func (m *MockSmartNotificationService) TrackNotificationDismissed(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSmartNotificationService) GetUserEngagement(ctx context.Context, userID uuid.UUID) (*models.UserEngagement, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserEngagement), args.Error(1)
}

func (m *MockSmartNotificationService) GetUserPreferences(ctx context.Context, userID uuid.UUID) *models.SmartNotificationPreferences {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.SmartNotificationPreferences)
}

func (m *MockSmartNotificationService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, prefs *models.SmartNotificationPreferences) error {
	args := m.Called(ctx, userID, prefs)
	return args.Error(0)
}

func (m *MockSmartNotificationService) GetDigest(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) (*models.NotificationDigest, error) {
	args := m.Called(ctx, userID, digestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationDigest), args.Error(1)
}

func (m *MockSmartNotificationService) ListDigests(ctx context.Context, userID uuid.UUID, opts models.DigestListOptions) ([]models.NotificationDigest, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NotificationDigest), args.Error(1)
}

func (m *MockSmartNotificationService) MarkDigestRead(ctx context.Context, userID uuid.UUID, digestID uuid.UUID) error {
	args := m.Called(ctx, userID, digestID)
	return args.Error(0)
}

func (m *MockSmartNotificationService) RouteNotification(ctx context.Context, userID uuid.UUID, smartNotif *models.SmartNotification) (*models.SmartNotification, error) {
	args := m.Called(ctx, userID, smartNotif)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SmartNotification), args.Error(1)
}

// --- Helper ---

func setupSmartNotifTestApp(handler *SmartNotificationHandler) (*fiber.App, uuid.UUID) {
	app := fiber.New()
	userID := uuid.New()

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Register routes
	notif := app.Group("/users/@me/notifications")
	notif.Post("/score", handler.ScoreNotification)
	notif.Post("/snooze", handler.SnoozeNotifications)
	notif.Delete("/snooze", handler.UnsnoozeNotifications)
	notif.Get("/snooze", handler.GetSnoozeStatus)
	notif.Post("/mute", handler.MuteNotifications)
	notif.Get("/mute", handler.GetMuteStatus)
	notif.Post("/:id/click", handler.TrackClick)
	notif.Post("/:id/dismiss", handler.DismissNotification)
	notif.Get("/engagement", handler.GetEngagement)
	notif.Get("/preferences", handler.GetPreferences)
	notif.Patch("/preferences", handler.UpdatePreferences)
	notif.Get("/digests", handler.ListDigests)
	notif.Get("/digests/:id", handler.GetDigest)
	notif.Post("/digests/:id/read", handler.MarkDigestRead)

	return app, userID
}

// --- Tests ---

func TestSmartNotificationHandler_ScoreNotification(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, _ := setupSmartNotifTestApp(handler)

	mockService.On("ScoreNotification", mock.Anything, mock.AnythingOfType("*models.PriorityScoringInput")).Return(&models.SmartNotification{
		PriorityScore: 80,
		Priority:      models.NotificationPriorityUrgent,
		DeliveryMode:  models.DeliveryImmediate,
		Category:      models.NotifCategoryMention,
	}, nil)

	body, _ := json.Marshal(models.PriorityScoringInput{
		NotificationType: models.NotificationTypeMention,
		HasMention:       true,
	})

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/score", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_ScoreNotification_BadRequest(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, _ := setupSmartNotifTestApp(handler)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/score", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSmartNotificationHandler_SnoozeNotifications(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	until := time.Now().Add(30 * time.Minute)
	mockService.On("SnoozeNotifications", mock.Anything, userID, mock.AnythingOfType("*models.SnoozeRequest")).Return(&models.SnoozeConfig{
		UserID: userID,
		Active: true,
		Until:  until,
	}, nil)

	body, _ := json.Marshal(models.SnoozeRequest{DurationMins: 30})
	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/snooze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_SnoozeNotifications_InvalidDuration(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, _ := setupSmartNotifTestApp(handler)

	body, _ := json.Marshal(models.SnoozeRequest{DurationMins: 0})
	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/snooze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSmartNotificationHandler_UnsnoozeNotifications(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("UnsnoozeNotifications", mock.Anything, userID, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/@me/notifications/snooze", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestSmartNotificationHandler_GetSnoozeStatus(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	until := time.Now().Add(30 * time.Minute)
	mockService.On("IsNotificationSnoozed", mock.Anything, userID, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(true, &until, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/snooze", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["snoozed"])
}

func TestSmartNotificationHandler_MuteNotifications(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("MuteNotifications", mock.Anything, userID, mock.AnythingOfType("*models.MuteRequest")).Return(&models.MuteConfig{
		UserID: userID,
		Muted:  true,
	}, nil)

	body, _ := json.Marshal(models.MuteRequest{Muted: true})
	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/mute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_GetMuteStatus(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("IsNotificationMuted", mock.Anything, userID, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(false, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/mute", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, false, result["muted"])
}

func TestSmartNotificationHandler_TrackClick(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	notifID := uuid.New()
	mockService.On("TrackNotificationClick", mock.Anything, userID, notifID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/"+notifID.String()+"/click", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestSmartNotificationHandler_TrackClick_BadID(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, _ := setupSmartNotifTestApp(handler)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/invalid-id/click", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSmartNotificationHandler_DismissNotification(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("TrackNotificationDismissed", mock.Anything, userID).Return(nil)

	notifID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/"+notifID.String()+"/dismiss", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestSmartNotificationHandler_GetEngagement(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("GetUserEngagement", mock.Anything, userID).Return(&models.UserEngagement{
		UserID:         userID,
		TotalReceived:  100,
		TotalClicked:   30,
		TotalDismissed: 10,
		ClickRate:      0.3,
		UpdatedAt:      time.Now(),
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/engagement", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_GetPreferences(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("GetUserPreferences", mock.Anything, userID).Return(&models.SmartNotificationPreferences{
		UserID:             userID,
		Enabled:            true,
		DigestEnabled:      true,
		DigestIntervalMins: 30,
	})

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/preferences", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_UpdatePreferences(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("UpdateUserPreferences", mock.Anything, userID, mock.AnythingOfType("*models.SmartNotificationPreferences")).Return(nil)

	body, _ := json.Marshal(models.SmartNotificationPreferences{
		Enabled:            true,
		DigestEnabled:      false,
		DigestIntervalMins: 60,
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_UpdatePreferences_ClampsInterval(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("UpdateUserPreferences", mock.Anything, userID, mock.MatchedBy(func(p *models.SmartNotificationPreferences) bool {
		return p.DigestIntervalMins >= 5
	})).Return(nil)

	body, _ := json.Marshal(models.SmartNotificationPreferences{
		DigestIntervalMins: 1, // too low, should be clamped to 5
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/@me/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_ListDigests(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	digests := []models.NotificationDigest{
		{ID: uuid.New(), UserID: userID, Title: "Digest 1", Count: 3, CreatedAt: time.Now()},
		{ID: uuid.New(), UserID: userID, Title: "Digest 2", Count: 5, CreatedAt: time.Now()},
	}

	mockService.On("ListDigests", mock.Anything, userID, models.DigestListOptions{Limit: 20, Offset: 0}).Return(digests, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/digests", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	digestList := result["digests"].([]interface{})
	assert.Len(t, digestList, 2)
}

func TestSmartNotificationHandler_ListDigests_Empty(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	mockService.On("ListDigests", mock.Anything, userID, models.DigestListOptions{Limit: 20, Offset: 0}).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/digests", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	digestList := result["digests"].([]interface{})
	assert.Len(t, digestList, 0)
}

func TestSmartNotificationHandler_GetDigest(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	digestID := uuid.New()
	mockService.On("GetDigest", mock.Anything, userID, digestID).Return(&models.NotificationDigest{
		ID:     digestID,
		UserID: userID,
		Title:  "Test digest",
		Count:  3,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/digests/"+digestID.String(), nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmartNotificationHandler_GetDigest_NotFound(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	digestID := uuid.New()
	mockService.On("GetDigest", mock.Anything, userID, digestID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/users/@me/notifications/digests/"+digestID.String(), nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestSmartNotificationHandler_MarkDigestRead(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, userID := setupSmartNotifTestApp(handler)

	digestID := uuid.New()
	mockService.On("MarkDigestRead", mock.Anything, userID, digestID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/digests/"+digestID.String()+"/read", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestSmartNotificationHandler_MarkDigestRead_BadID(t *testing.T) {
	mockService := new(MockSmartNotificationService)
	handler := NewSmartNotificationHandler(mockService)
	app, _ := setupSmartNotifTestApp(handler)

	req := httptest.NewRequest(http.MethodPost, "/users/@me/notifications/digests/bad-id/read", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
