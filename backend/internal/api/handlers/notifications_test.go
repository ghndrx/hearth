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

// MockNotificationService mocks the NotificationService for testing
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) GetNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.NotificationWithActor, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationWithActor), args.Error(1)
}

func (m *MockNotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NotificationWithActor), args.Error(1)
}

func (m *MockNotificationService) GetNotificationStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationStats), args.Error(1)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockNotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationService) DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockNotificationService) DeleteAllReadNotifications(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// testNotificationHandler creates a test notification handler with mocks
type testNotificationHandler struct {
	handler             *NotificationHandler
	notificationService *MockNotificationService
	app                 *fiber.App
	userID              uuid.UUID
}

func newTestNotificationHandler() *testNotificationHandler {
	notificationService := new(MockNotificationService)
	handler := NewNotificationHandler(notificationService)

	app := fiber.New()
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Get("/notifications", handler.GetNotifications)
	app.Get("/notifications/stats", handler.GetNotificationStats)
	app.Post("/notifications/read-all", handler.MarkAllAsRead)
	app.Delete("/notifications/read", handler.DeleteAllRead)
	app.Get("/notifications/:id", handler.GetNotification)
	app.Post("/notifications/:id/read", handler.MarkAsRead)
	app.Delete("/notifications/:id", handler.DeleteNotification)

	return &testNotificationHandler{
		handler:             handler,
		notificationService: notificationService,
		app:                 app,
		userID:              userID,
	}
}

func TestNotificationHandler_GetNotifications(t *testing.T) {
	th := newTestNotificationHandler()

	actorUsername := "testuser"
	notifications := []models.NotificationWithActor{
		{
			Notification: models.Notification{
				ID:        uuid.New(),
				UserID:    th.userID,
				Type:      models.NotificationTypeMention,
				Title:     "You were mentioned",
				Body:      "testuser mentioned you in #general",
				Read:      false,
				CreatedAt: time.Now(),
			},
			ActorUsername: &actorUsername,
		},
	}

	stats := &models.NotificationStats{
		Total:  1,
		Unread: 1,
	}

	th.notificationService.On("ListNotifications", mock.Anything, th.userID, mock.Anything).Return(notifications, nil)
	th.notificationService.On("GetNotificationStats", mock.Anything, th.userID).Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.NotNil(t, result["notifications"])
	assert.Equal(t, float64(1), result["total"])
	assert.Equal(t, float64(1), result["unread"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotifications_WithFilters(t *testing.T) {
	th := newTestNotificationHandler()

	notifications := []models.NotificationWithActor{}
	stats := &models.NotificationStats{Total: 0, Unread: 0}

	th.notificationService.On("ListNotifications", mock.Anything, th.userID, mock.MatchedBy(func(opts models.NotificationListOptions) bool {
		return opts.Limit == 10 && opts.Offset == 5 && opts.Unread != nil && *opts.Unread == true
	})).Return(notifications, nil)
	th.notificationService.On("GetNotificationStats", mock.Anything, th.userID).Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=10&offset=5&unread=true", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotifications_Error(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("ListNotifications", mock.Anything, th.userID, mock.Anything).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get notifications", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotification(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()
	actorUsername := "testuser"
	notification := &models.NotificationWithActor{
		Notification: models.Notification{
			ID:        notificationID,
			UserID:    th.userID,
			Type:      models.NotificationTypeMention,
			Title:     "You were mentioned",
			Body:      "testuser mentioned you",
			Read:      false,
			CreatedAt: time.Now(),
		},
		ActorUsername: &actorUsername,
	}

	th.notificationService.On("GetNotification", mock.Anything, notificationID, th.userID).Return(notification, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.NotificationWithActor
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, notificationID, result.ID)

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotification_NotFound(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("GetNotification", mock.Anything, notificationID, th.userID).Return(nil, services.ErrNotificationNotFound)

	req := httptest.NewRequest(http.MethodGet, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "notification not found", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotification_InvalidID(t *testing.T) {
	th := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodGet, "/notifications/invalid-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid notification id", result["error"])
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("MarkAsRead", mock.Anything, notificationID, th.userID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/notifications/"+notificationID.String()+"/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_MarkAsRead_NotFound(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("MarkAsRead", mock.Anything, notificationID, th.userID).Return(services.ErrNotificationNotFound)

	req := httptest.NewRequest(http.MethodPost, "/notifications/"+notificationID.String()+"/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "notification not found", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_MarkAsRead_InvalidID(t *testing.T) {
	th := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodPost, "/notifications/invalid-uuid/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid notification id", result["error"])
}

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("MarkAllAsRead", mock.Anything, th.userID).Return(int64(5), nil)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, float64(5), result["marked"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_MarkAllAsRead_Error(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("MarkAllAsRead", mock.Anything, th.userID).Return(int64(0), assert.AnError)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to mark all notifications as read", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_DeleteNotification(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("DeleteNotification", mock.Anything, notificationID, th.userID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_DeleteNotification_NotFound(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("DeleteNotification", mock.Anything, notificationID, th.userID).Return(services.ErrNotificationNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "notification not found", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_DeleteNotification_InvalidID(t *testing.T) {
	th := newTestNotificationHandler()

	req := httptest.NewRequest(http.MethodDelete, "/notifications/invalid-uuid", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid notification id", result["error"])
}

func TestNotificationHandler_DeleteAllRead(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("DeleteAllReadNotifications", mock.Anything, th.userID).Return(int64(3), nil)

	req := httptest.NewRequest(http.MethodDelete, "/notifications/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, float64(3), result["deleted"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_DeleteAllRead_Error(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("DeleteAllReadNotifications", mock.Anything, th.userID).Return(int64(0), assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/notifications/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to delete read notifications", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotificationStats(t *testing.T) {
	th := newTestNotificationHandler()

	stats := &models.NotificationStats{
		Total:  10,
		Unread: 3,
	}

	th.notificationService.On("GetNotificationStats", mock.Anything, th.userID).Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications/stats", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.NotificationStats
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, 10, result.Total)
	assert.Equal(t, 3, result.Unread)

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotificationStats_Error(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("GetNotificationStats", mock.Anything, th.userID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/notifications/stats", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get notification stats", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotifications_EmptyList(t *testing.T) {
	th := newTestNotificationHandler()

	notifications := []models.NotificationWithActor{}
	stats := &models.NotificationStats{Total: 0, Unread: 0}

	th.notificationService.On("ListNotifications", mock.Anything, th.userID, mock.Anything).Return(notifications, nil)
	th.notificationService.On("GetNotificationStats", mock.Anything, th.userID).Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	notifications_ := result["notifications"].([]interface{})
	assert.Len(t, notifications_, 0)
	assert.Equal(t, float64(0), result["total"])
	assert.Equal(t, float64(0), result["unread"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_MarkAllAsRead_ZeroUpdated(t *testing.T) {
	th := newTestNotificationHandler()

	th.notificationService.On("MarkAllAsRead", mock.Anything, th.userID).Return(int64(0), nil)

	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, float64(0), result["marked"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_GetNotification_ServiceError(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("GetNotification", mock.Anything, notificationID, th.userID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to get notification", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_MarkAsRead_ServiceError(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("MarkAsRead", mock.Anything, notificationID, th.userID).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodPost, "/notifications/"+notificationID.String()+"/read", nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to mark notification as read", result["error"])

	th.notificationService.AssertExpectations(t)
}

func TestNotificationHandler_DeleteNotification_ServiceError(t *testing.T) {
	th := newTestNotificationHandler()

	notificationID := uuid.New()

	th.notificationService.On("DeleteNotification", mock.Anything, notificationID, th.userID).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/notifications/"+notificationID.String(), nil)
	resp, err := th.app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "failed to delete notification", result["error"])

	th.notificationService.AssertExpectations(t)
}

// Suppress unused import warning
var _ = bytes.Buffer{}
