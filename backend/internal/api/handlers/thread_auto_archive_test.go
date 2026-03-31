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

// MockThreadAutoArchiveService is a mock implementation of ThreadAutoArchiveServiceInterface
type MockThreadAutoArchiveService struct {
	mock.Mock
}

func (m *MockThreadAutoArchiveService) GetOrCreateServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadAutoArchiveSettings), args.Error(1)
}

func (m *MockThreadAutoArchiveService) UpdateServerSettings(ctx context.Context, serverID, requesterID uuid.UUID, req models.UpdateThreadAutoArchiveSettingsRequest) (*models.ThreadAutoArchiveSettings, error) {
	args := m.Called(ctx, serverID, requesterID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadAutoArchiveSettings), args.Error(1)
}

func (m *MockThreadAutoArchiveService) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadAutoArchiveSettings), args.Error(1)
}

func (m *MockThreadAutoArchiveService) SetChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID, req models.SetChannelAutoArchiveOverrideRequest) (*models.ChannelAutoArchiveOverride, error) {
	args := m.Called(ctx, channelID, requesterID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChannelAutoArchiveOverride), args.Error(1)
}

func (m *MockThreadAutoArchiveService) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChannelAutoArchiveOverride), args.Error(1)
}

func (m *MockThreadAutoArchiveService) DeleteChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID) error {
	args := m.Called(ctx, channelID, requesterID)
	return args.Error(0)
}

func (m *MockThreadAutoArchiveService) GetThreadAutoArchiveStatus(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveResponse, error) {
	args := m.Called(ctx, threadID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadAutoArchiveResponse), args.Error(1)
}

func (m *MockThreadAutoArchiveService) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ThreadAutoArchiveStats), args.Error(1)
}

func (m *MockThreadAutoArchiveService) ArchiveThread(ctx context.Context, threadID uuid.UUID) error {
	args := m.Called(ctx, threadID)
	return args.Error(0)
}

func setupThreadAutoArchiveTestApp(handler *ThreadAutoArchiveHandler) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.New())
		return c.Next()
	})

	// Register routes matching the production route structure
	threads := app.Group("/threads")
	threads.Get("/:thread_id/auto-archive", handler.GetThreadAutoArchiveStatus)

	channels := app.Group("/channels")
	channels.Get("/:channel_id/auto-archive", handler.GetChannelAutoArchiveOverride)
	channels.Put("/:channel_id/auto-archive", handler.SetChannelAutoArchiveOverride)
	channels.Delete("/:channel_id/auto-archive", handler.DeleteChannelAutoArchiveOverride)

	servers := app.Group("/servers")
	servers.Get("/:server_id/auto-archive", handler.GetServerAutoArchiveSettings)
	servers.Patch("/:server_id/auto-archive", handler.UpdateServerAutoArchiveSettings)
	servers.Get("/:server_id/auto-archive/stats", handler.GetServerAutoArchiveStats)

	return app
}

func TestThreadAutoArchiveHandler_GetServerAutoArchiveSettings(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	serverID := uuid.New()
	settings := &models.ThreadAutoArchiveSettings{
		ID:              uuid.New(),
		ServerID:        serverID,
		DefaultDuration: 1440,
		AllowOverride:   true,
	}

	mockService.On("GetServerSettings", mock.Anything, serverID).Return(settings, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ThreadAutoArchiveSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, settings.ServerID, result.ServerID)
	assert.Equal(t, settings.DefaultDuration, result.DefaultDuration)
}

func TestThreadAutoArchiveHandler_GetServerAutoArchiveSettings_InvalidServerID(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/invalid-uuid/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_UpdateServerAutoArchiveSettings(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	serverID := uuid.New()
	newDuration := 4320

	settings := &models.ThreadAutoArchiveSettings{
		ID:              uuid.New(),
		ServerID:        serverID,
		DefaultDuration: newDuration,
		AllowOverride:   true,
	}

	mockService.On("UpdateServerSettings", mock.Anything, serverID, mock.Anything, mock.AnythingOfType("models.UpdateThreadAutoArchiveSettingsRequest")).Return(settings, nil)

	body, _ := json.Marshal(models.UpdateThreadAutoArchiveSettingsRequest{
		DefaultDuration: &newDuration,
	})
	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/auto-archive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ThreadAutoArchiveSettings
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, newDuration, result.DefaultDuration)
}

func TestThreadAutoArchiveHandler_GetChannelAutoArchiveOverride(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	channelID := uuid.New()
	override := &models.ChannelAutoArchiveOverride{
		ID:                   uuid.New(),
		ChannelID:           channelID,
		AutoArchiveDuration: 4320,
	}

	mockService.On("GetChannelOverride", mock.Anything, channelID).Return(override, nil)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_GetChannelAutoArchiveOverride_NoOverride(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	channelID := uuid.New()

	mockService.On("GetChannelOverride", mock.Anything, channelID).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_SetChannelAutoArchiveOverride(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	channelID := uuid.New()
	duration := 4320

	override := &models.ChannelAutoArchiveOverride{
		ID:                   uuid.New(),
		ChannelID:           channelID,
		AutoArchiveDuration: duration,
	}

	mockService.On("SetChannelOverride", mock.Anything, channelID, mock.Anything, mock.AnythingOfType("models.SetChannelAutoArchiveOverrideRequest")).Return(override, nil)

	body, _ := json.Marshal(models.SetChannelAutoArchiveOverrideRequest{
		AutoArchiveDuration: duration,
	})
	req := httptest.NewRequest(http.MethodPut, "/channels/"+channelID.String()+"/auto-archive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_DeleteChannelAutoArchiveOverride(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	channelID := uuid.New()

	mockService.On("DeleteChannelOverride", mock.Anything, channelID, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/channels/"+channelID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_GetThreadAutoArchiveStatus(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	threadID := uuid.New()
	nextArchive := time.Now().Add(1 * time.Hour)

	status := &models.ThreadAutoArchiveResponse{
		ThreadID:      threadID,
		NextArchiveAt: &nextArchive,
		Eligible:      true,
		Status:        "scheduled",
	}

	mockService.On("GetThreadAutoArchiveStatus", mock.Anything, threadID).Return(status, nil)

	req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_GetThreadAutoArchiveStatus_NotFound(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	threadID := uuid.New()

	mockService.On("GetThreadAutoArchiveStatus", mock.Anything, threadID).Return(nil, services.ErrThreadNotFound)

	req := httptest.NewRequest(http.MethodGet, "/threads/"+threadID.String()+"/auto-archive", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestThreadAutoArchiveHandler_GetServerAutoArchiveStats(t *testing.T) {
	mockService := new(MockThreadAutoArchiveService)
	handler := NewThreadAutoArchiveHandler(mockService)
	app := setupThreadAutoArchiveTestApp(handler)

	serverID := uuid.New()
	stats := &models.ThreadAutoArchiveStats{
		ServerID:              serverID,
		TotalThreads:          100,
		ArchivedThreads:       20,
		ScheduledThreads:      50,
		ReadyToArchiveThreads: 5,
	}

	mockService.On("GetServerStats", mock.Anything, serverID).Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/auto-archive/stats", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ThreadAutoArchiveStats
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, serverID, result.ServerID)
	assert.Equal(t, 100, result.TotalThreads)
	assert.Equal(t, 5, result.ReadyToArchiveThreads)
}
