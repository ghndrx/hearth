package handlers

import (
	"context"
	"encoding/json"
	"io"
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

// MockAnalyticsService mocks the AnalyticsService for handler tests
type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) GetSummary(ctx context.Context, serverID, requesterID uuid.UUID) (*models.ServerInsightsResponse, error) {
	args := m.Called(ctx, serverID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerInsightsResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetMemberGrowth(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.MemberGrowthResponse, error) {
	args := m.Called(ctx, serverID, requesterID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MemberGrowthResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetMessageActivity(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.ActivityHeatmapResponse, error) {
	args := m.Called(ctx, serverID, requesterID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActivityHeatmapResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetTopChannels(ctx context.Context, serverID, requesterID uuid.UUID, days, limit int) (*models.TopChannelsResponse, error) {
	args := m.Called(ctx, serverID, requesterID, days, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TopChannelsResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetRetention(ctx context.Context, serverID, requesterID uuid.UUID, days int) (*models.RetentionResponse, error) {
	args := m.Called(ctx, serverID, requesterID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RetentionResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetMostActiveUsers(ctx context.Context, serverID, requesterID uuid.UUID, days, limit int) ([]*models.ActiveUserStat, error) {
	args := m.Called(ctx, serverID, requesterID, days, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveUserStat), args.Error(1)
}

func (m *MockAnalyticsService) InvalidateCache(ctx context.Context, serverID uuid.UUID) error {
	args := m.Called(ctx, serverID)
	return args.Error(0)
}

// setupAnalyticsTestApp creates a Fiber app with the analytics handler and auth middleware
func setupAnalyticsTestApp(handler *AnalyticsHandler, userID uuid.UUID) *fiber.App {
	app := fiber.New()
	api := app.Group("/api/v1")
	servers := api.Group("/servers")

	// Auth middleware
	app.Use(func(c *fiber.Ctx) error {
		if userID != uuid.Nil {
			c.Locals("userID", userID)
		}
		return c.Next()
	})

	// Register routes matching actual API structure
	servers.Get("/:id/insights", handler.GetSummary)
	servers.Get("/:id/insights/growth", handler.GetMemberGrowth)
	servers.Get("/:id/insights/activity", handler.GetMessageActivity)
	servers.Get("/:id/insights/channels", handler.GetTopChannels)
	servers.Get("/:id/insights/retention", handler.GetRetention)
	servers.Get("/:id/insights/users", handler.GetMostActiveUsers)
	servers.Post("/:id/insights/invalidate", handler.InvalidateCache)

	return app
}

// --- GetSummary tests ---

func TestAnalyticsHandler_GetSummary_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.ServerInsightsResponse{
		ServerID: serverID,
		Period:   "7d",
		Summary: &models.AnalyticsSummary{
			MessagesToday:    100,
			ActiveUsersToday: 20,
			TotalMembers:     500,
		},
	}

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result models.ServerInsightsResponse
	json.Unmarshal(body, &result)
	assert.Equal(t, serverID, result.ServerID)
	assert.Equal(t, "7d", result.Period)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetSummary_Unauthorized(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)

	// No userID set (uuid.Nil)
	app := setupAnalyticsTestApp(handler, uuid.Nil)
	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAnalyticsHandler_GetSummary_InvalidServerID(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/not-a-uuid/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestAnalyticsHandler_GetSummary_ServerNotFound(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(nil, services.ErrServerNotFound)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestAnalyticsHandler_GetSummary_NotServerMember(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAnalyticsHandler_GetSummary_MissingPermission(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(nil, services.ErrMissingPermission)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAnalyticsHandler_GetSummary_InternalError(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(nil, assert.AnError)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- GetMemberGrowth tests ---

func TestAnalyticsHandler_GetMemberGrowth_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	now := time.Now()
	expectedResp := &models.MemberGrowthResponse{
		ServerID: serverID.String(),
		Period:   "30d",
		Data: []*models.MemberGrowthPoint{
			{Date: now, Count: 100, Change: 5},
		},
	}

	mockSvc.On("GetMemberGrowth", mock.Anything, serverID, userID, 30).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/growth?days=30", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result models.MemberGrowthResponse
	json.Unmarshal(body, &result)
	assert.Equal(t, "30d", result.Period)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetMemberGrowth_DefaultDays(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.MemberGrowthResponse{
		ServerID: serverID.String(),
		Period:   "7d",
		Data:     []*models.MemberGrowthPoint{},
	}

	// Default days=7 when not specified
	mockSvc.On("GetMemberGrowth", mock.Anything, serverID, userID, 7).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/growth", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetMemberGrowth_DaysCapped(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.MemberGrowthResponse{
		ServerID: serverID.String(),
		Period:   "90d",
		Data:     []*models.MemberGrowthPoint{},
	}

	// Days capped at 90
	mockSvc.On("GetMemberGrowth", mock.Anything, serverID, userID, 90).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/growth?days=200", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetMemberGrowth_Unauthorized(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)

	app := setupAnalyticsTestApp(handler, uuid.Nil)
	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/growth", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// --- GetMessageActivity tests ---

func TestAnalyticsHandler_GetMessageActivity_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.ActivityHeatmapResponse{
		ServerID: serverID.String(),
		Period:   "7d",
		Data:     []*models.ActivityHourStat{},
		PeakHours: []*models.PeakHour{
			{Hour: 14, MessageCount: 100},
		},
	}

	mockSvc.On("GetMessageActivity", mock.Anything, serverID, userID, 7).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/activity", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetMessageActivity_ServiceError(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetMessageActivity", mock.Anything, serverID, userID, 7).Return(nil, services.ErrServerNotFound)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/activity", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// --- GetTopChannels tests ---

func TestAnalyticsHandler_GetTopChannels_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	now := time.Now()
	expectedResp := &models.TopChannelsResponse{
		ServerID: serverID.String(),
		Period:   "7d",
		Data: []*models.TopChannelStat{
			{
				ChannelID:     uuid.New(),
				ChannelName:   "general",
				ChannelType:   "text",
				MessageCount:  500,
				UniqueAuthors: 45,
				LastActivity:  &now,
			},
		},
	}

	mockSvc.On("GetTopChannels", mock.Anything, serverID, userID, 7, 10).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/channels", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetTopChannels_WithDaysAndLimit(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.TopChannelsResponse{
		ServerID: serverID.String(),
		Period:   "14d",
		Data:     []*models.TopChannelStat{},
	}

	mockSvc.On("GetTopChannels", mock.Anything, serverID, userID, 14, 25).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/channels?days=14&limit=25", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetTopChannels_LimitCapped(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.TopChannelsResponse{
		ServerID: serverID.String(),
		Period:   "7d",
		Data:     []*models.TopChannelStat{},
	}

	// limit capped at 50
	mockSvc.On("GetTopChannels", mock.Anything, serverID, userID, 7, 50).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/channels?limit=200", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

// --- GetRetention tests ---

func TestAnalyticsHandler_GetRetention_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	now := time.Now()
	expectedResp := &models.RetentionResponse{
		ServerID: serverID.String(),
		Period:   "30d",
		Data: &models.RetentionMetrics{
			MAU:          150,
			TotalMembers: 500,
			AverageDAU:   48.5,
			Stickiness:   0.323,
			DailyActiveUsers: []*models.DailyActiveUserPoint{
				{Date: now, Count: 50},
			},
		},
	}

	mockSvc.On("GetRetention", mock.Anything, serverID, userID, 30).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/retention?days=30", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetRetention_DefaultDays(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedResp := &models.RetentionResponse{
		ServerID: serverID.String(),
		Period:   "30d",
		Data:     &models.RetentionMetrics{},
	}

	// Default days=30 for retention endpoint
	mockSvc.On("GetRetention", mock.Anything, serverID, userID, 30).Return(expectedResp, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/retention", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

// --- GetMostActiveUsers tests ---

func TestAnalyticsHandler_GetMostActiveUsers_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	expectedUsers := []*models.ActiveUserStat{
		{UserID: uuid.New(), MessageCount: 500, DaysActive: 5},
	}

	mockSvc.On("GetMostActiveUsers", mock.Anything, serverID, userID, 7, 10).Return(expectedUsers, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/users", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, serverID.String(), result["server_id"])
	assert.Equal(t, "7d", result["period"])
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_GetMostActiveUsers_WithDaysAndLimit(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetMostActiveUsers", mock.Anything, serverID, userID, 14, 25).Return([]*models.ActiveUserStat{}, nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("GET", "/api/v1/servers/"+serverID.String()+"/insights/users?days=14&limit=25", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

// --- InvalidateCache tests ---

func TestAnalyticsHandler_InvalidateCache_Success(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	// GetSummary succeeds (permission check passes)
	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(&models.ServerInsightsResponse{
		ServerID: serverID,
		Period:   "7d",
	}, nil)
	mockSvc.On("InvalidateCache", mock.Anything, serverID).Return(nil)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("POST", "/api/v1/servers/"+serverID.String()+"/insights/invalidate", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, true, result["success"])
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_InvalidateCache_Unauthorized(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)

	app := setupAnalyticsTestApp(handler, uuid.Nil)
	serverID := uuid.New()
	req := httptest.NewRequest("POST", "/api/v1/servers/"+serverID.String()+"/insights/invalidate", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAnalyticsHandler_InvalidateCache_GetSummaryFails(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("POST", "/api/v1/servers/"+serverID.String()+"/insights/invalidate", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

func TestAnalyticsHandler_InvalidateCache_InvalidateFails(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	userID := uuid.New()
	serverID := uuid.New()

	mockSvc.On("GetSummary", mock.Anything, serverID, userID).Return(&models.ServerInsightsResponse{
		ServerID: serverID,
	}, nil)
	mockSvc.On("InvalidateCache", mock.Anything, serverID).Return(assert.AnError)

	app := setupAnalyticsTestApp(handler, userID)
	req := httptest.NewRequest("POST", "/api/v1/servers/"+serverID.String()+"/insights/invalidate", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	mockSvc.AssertExpectations(t)
}

// --- Parsing helper tests ---

func TestAnalyticsHandler_ParseDays(t *testing.T) {
	handler := &AnalyticsHandler{}

	// Default value
	assert.Equal(t, 7, handler.parseDays(""))
	assert.Equal(t, 7, handler.parseDays("invalid"))
	assert.Equal(t, 7, handler.parseDays("-1"))
	assert.Equal(t, 7, handler.parseDays("0"))

	// Valid values
	assert.Equal(t, 7, handler.parseDays("7"))
	assert.Equal(t, 14, handler.parseDays("14"))
	assert.Equal(t, 30, handler.parseDays("30"))
	assert.Equal(t, 90, handler.parseDays("90"))

	// Capped at 90
	assert.Equal(t, 90, handler.parseDays("100"))
	assert.Equal(t, 90, handler.parseDays("365"))
}

func TestAnalyticsHandler_ParseLimit(t *testing.T) {
	handler := &AnalyticsHandler{}

	// Default value
	assert.Equal(t, 10, handler.parseLimit(""))
	assert.Equal(t, 10, handler.parseLimit("invalid"))
	assert.Equal(t, 10, handler.parseLimit("-1"))
	assert.Equal(t, 10, handler.parseLimit("0"))

	// Valid values
	assert.Equal(t, 5, handler.parseLimit("5"))
	assert.Equal(t, 10, handler.parseLimit("10"))
	assert.Equal(t, 25, handler.parseLimit("25"))
	assert.Equal(t, 50, handler.parseLimit("50"))

	// Capped at 50
	assert.Equal(t, 50, handler.parseLimit("100"))
	assert.Equal(t, 50, handler.parseLimit("1000"))
}

// --- handleError tests ---

func TestAnalyticsHandler_HandleError_ServerNotFound(t *testing.T) {
	handler := &AnalyticsHandler{}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.handleError(c, services.ErrServerNotFound)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestAnalyticsHandler_HandleError_NotServerMember(t *testing.T) {
	handler := &AnalyticsHandler{}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.handleError(c, services.ErrNotServerMember)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAnalyticsHandler_HandleError_MissingPermission(t *testing.T) {
	handler := &AnalyticsHandler{}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.handleError(c, services.ErrMissingPermission)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAnalyticsHandler_HandleError_PermissionString(t *testing.T) {
	handler := &AnalyticsHandler{}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.handleError(c, fiber.NewError(fiber.StatusForbidden, "missing permission: MANAGE_SERVER"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAnalyticsHandler_HandleError_InternalError(t *testing.T) {
	handler := &AnalyticsHandler{}

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.handleError(c, assert.AnError)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- NewAnalyticsHandler tests ---

func TestNewAnalyticsHandler(t *testing.T) {
	mockSvc := new(MockAnalyticsService)
	handler := NewAnalyticsHandler(mockSvc)
	assert.NotNil(t, handler)
	assert.Equal(t, mockSvc, handler.analyticsService)
}
