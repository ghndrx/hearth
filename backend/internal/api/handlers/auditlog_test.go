package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockAuditLogService implements AuditLogServiceInterface for testing
type MockAuditLogService struct {
	logs           []models.AuditLogEntry
	actionTypes    []string
	categories     []models.AuditLogCategoryInfo
	dashboard      *models.ModerationDashboardSummary
	trendData      []models.DailyModerationTrend
	moderators     []models.ModeratorStats
	offenders      []models.RepeatOffenderStats
	autoModStats   *models.AutoModStats
	err            error
	exportData     []byte
	exportFormat   string
}

func NewMockAuditLogService() *MockAuditLogService {
	return &MockAuditLogService{
		logs: make([]models.AuditLogEntry, 0),
		actionTypes: []string{
			models.AuditLogMemberBan,
			models.AuditLogMemberKick,
			models.AuditLogChannelCreate,
		},
		categories: []models.AuditLogCategoryInfo{
			{Category: 10, Name: "Member", Description: "Member events"},
			{Category: 20, Name: "Channel", Description: "Channel events"},
		},
		dashboard: &models.ModerationDashboardSummary{
			ServerID:         uuid.Nil,
			TotalActions:    100,
			TopAction:       "MEMBER_BAN",
			TopActionCount:  25,
			UniqueModerators: 5,
			UniqueTargets:   30,
			ActionBreakdown: map[string]int{"MEMBER_BAN": 25, "MEMBER_KICK": 15},
			TrendDirection:  "up",
			TrendPercent:    10.5,
		},
		trendData: []models.DailyModerationTrend{
			{Date: time.Now().AddDate(0, 0, -1), TotalActions: 10, Bans: 2, Kicks: 1},
			{Date: time.Now(), TotalActions: 15, Bans: 3, Kicks: 2},
		},
		moderators: []models.ModeratorStats{
			{ModeratorID: uuid.New(), TotalActions: 50, Bans: 20, Kicks: 10},
		},
		offenders: []models.RepeatOffenderStats{
			{UserID: uuid.New(), ModerationCount: 5, DifferentModerators: 3, Bans: 2, Warns: 2, Mutes: 1},
		},
		autoModStats: &models.AutoModStats{
			TotalTriggers: 100, Blocks: 30, Warns: 40, Timeouts: 15, Kicks: 5, Bans: 2,
		},
	}
}

func (m *MockAuditLogService) GetLogs(ctx context.Context, serverID uuid.UUID, filter services.AuditLogFilter) ([]models.AuditLogEntry, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.logs, len(m.logs), nil
}

func (m *MockAuditLogService) GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	for i := range m.logs {
		if m.logs[i].ID == entryID && m.logs[i].ServerID == serverID {
			return &m.logs[i], nil
		}
	}
	return nil, services.ErrAuditLogNotFound
}

func (m *MockAuditLogService) GetActionTypes() []string {
	return m.actionTypes
}

func (m *MockAuditLogService) GetCategories() []models.AuditLogCategoryInfo {
	return m.categories
}

func (m *MockAuditLogService) GetDashboardSummary(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	dash := *m.dashboard
	dash.ServerID = serverID
	return &dash, nil
}

func (m *MockAuditLogService) GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]models.DailyModerationTrend, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.trendData, nil
}

func (m *MockAuditLogService) GetModeratorActivity(ctx context.Context, serverID uuid.UUID, days int) ([]models.ModeratorStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.moderators, nil
}

func (m *MockAuditLogService) GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, days, minCount int) ([]models.RepeatOffenderStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.offenders, nil
}

func (m *MockAuditLogService) GetAutoModStats(ctx context.Context, serverID uuid.UUID, days int) (*models.AutoModStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.autoModStats, nil
}

func (m *MockAuditLogService) ExportLogs(ctx context.Context, serverID uuid.UUID, format string, filter services.AuditLogFilter) ([]byte, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.exportData, m.exportFormat, nil
}

func (m *MockAuditLogService) AddLog(entry models.AuditLogEntry) {
	m.logs = append(m.logs, entry)
}

// MockServerServiceForAuditLog implements ServerServiceForAuditLog for testing
type MockServerServiceForAuditLog struct {
	members     map[string]*models.Member
	permissions map[string]int64
	err         error
}

func NewMockServerServiceForAuditLog() *MockServerServiceForAuditLog {
	return &MockServerServiceForAuditLog{
		members:     make(map[string]*models.Member),
		permissions: make(map[string]int64),
	}
}

func (m *MockServerServiceForAuditLog) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := serverID.String() + ":" + userID.String()
	member, ok := m.members[key]
	if !ok {
		return nil, services.ErrNotServerMember
	}
	return member, nil
}

func (m *MockServerServiceForAuditLog) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	key := serverID.String() + ":" + userID.String()
	return m.permissions[key], nil
}

func (m *MockServerServiceForAuditLog) AddMember(serverID, userID uuid.UUID, permissions int64) {
	key := serverID.String() + ":" + userID.String()
	m.members[key] = &models.Member{
		ServerID: serverID,
		UserID:   userID,
	}
	m.permissions[key] = permissions
}

// setupAuditLogTest creates a test Fiber app with the audit log handler
func setupAuditLogTest(t *testing.T) (*fiber.App, *MockAuditLogService, *MockServerServiceForAuditLog, uuid.UUID, uuid.UUID) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	auditLogSvc := NewMockAuditLogService()
	serverSvc := NewMockServerServiceForAuditLog()
	handler := NewAuditLogHandler(auditLogSvc, serverSvc)

	serverID := uuid.New()
	userID := uuid.New()

	// Add user as member with VIEW_AUDIT_LOG permission
	serverSvc.AddMember(serverID, userID, models.PermViewAuditLog)

	// Set up routes with auth middleware
	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handler.RegisterAuditLogRoutes(api.Group("/servers"))

	return app, auditLogSvc, serverSvc, serverID, userID
}

func TestAuditLogHandler_GetAuditLogs_Success(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	// Add some logs
	targetID := uuid.New()
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:             uuid.New(),
		ServerID:       serverID,
		ActorID:        userID,
		ActionType:     models.AuditLogMemberBan,
		ActionCategory: 10,
		TargetID:       &targetID,
		Reason:         "spam",
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	logs := result["audit_logs"].([]interface{})
	assert.Len(t, logs, 1)
	assert.Equal(t, float64(1), result["total"])
}

func TestAuditLogHandler_GetAuditLogs_WithFilters(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	// Add logs
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:             uuid.New(),
		ServerID:       serverID,
		ActorID:        userID,
		ActionType:     models.AuditLogMemberBan,
		ActionCategory: 10,
		CreatedAt:      time.Now(),
	})

	tests := []struct {
		name        string
		queryParams string
		statusCode  int
	}{
		{
			name:        "filter by action_type",
			queryParams: "?action_type=MEMBER_BAN",
			statusCode:  http.StatusOK,
		},
		{
			name:        "filter by action_category",
			queryParams: "?action_category=10",
			statusCode:  http.StatusOK,
		},
		{
			name:        "filter by before",
			queryParams: "?before=" + time.Now().Add(time.Hour).Format(time.RFC3339),
			statusCode:  http.StatusOK,
		},
		{
			name:        "filter by after",
			queryParams: "?after=" + time.Now().Add(-time.Hour).Format(time.RFC3339),
			statusCode:  http.StatusOK,
		},
		{
			name:        "filter by limit",
			queryParams: "?limit=10",
			statusCode:  http.StatusOK,
		},
		{
			name:        "filter by offset",
			queryParams: "?offset=5",
			statusCode:  http.StatusOK,
		},
		{
			name:        "combined filters",
			queryParams: "?action_type=MEMBER_BAN&limit=10&offset=0",
			statusCode:  http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs"+tc.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}

func TestAuditLogHandler_GetAuditLogs_InvalidFilters(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	tests := []struct {
		name        string
		queryParams string
		statusCode  int
		errorMsg    string
	}{
		{
			name:        "invalid action_category",
			queryParams: "?action_category=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid action_category",
		},
		{
			name:        "invalid action_category range",
			queryParams: "?action_category=100",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid action_category",
		},
		{
			name:        "invalid before timestamp",
			queryParams: "?before=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid before timestamp",
		},
		{
			name:        "invalid after timestamp",
			queryParams: "?after=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid after timestamp",
		},
		{
			name:        "invalid limit",
			queryParams: "?limit=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid limit",
		},
		{
			name:        "invalid offset",
			queryParams: "?offset=-1",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid offset",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs"+tc.queryParams, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tc.statusCode, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Contains(t, result["error"].(string), tc.errorMsg)
		})
	}
}

func TestAuditLogHandler_GetAuditLogs_InvalidServerID(t *testing.T) {
	app, _, _, _, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/invalid/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "invalid server id", result["error"])
}

func TestAuditLogHandler_GetAuditLogs_NotMember(t *testing.T) {
	app, _, serverSvc, _, _ := setupAuditLogTest(t)

	// Create a different server that the user is not a member of
	otherServerID := uuid.New()
	_ = serverSvc // User is not a member of otherServerID

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+otherServerID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogs_NoPermission(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	auditLogSvc := NewMockAuditLogService()
	serverSvc := NewMockServerServiceForAuditLog()
	handler := NewAuditLogHandler(auditLogSvc, serverSvc)

	serverID := uuid.New()
	userID := uuid.New()

	// Add user as member WITHOUT VIEW_AUDIT_LOG permission
	serverSvc.AddMember(serverID, userID, 0)

	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handler.RegisterAuditLogRoutes(api.Group("/servers"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogs_AdminHasPermission(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	auditLogSvc := NewMockAuditLogService()
	serverSvc := NewMockServerServiceForAuditLog()
	handler := NewAuditLogHandler(auditLogSvc, serverSvc)

	serverID := uuid.New()
	userID := uuid.New()

	// Add user as member with ADMINISTRATOR permission (should grant VIEW_AUDIT_LOG)
	serverSvc.AddMember(serverID, userID, models.PermAdministrator)

	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handler.RegisterAuditLogRoutes(api.Group("/servers"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogEntry_Success(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	entryID := uuid.New()
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:             entryID,
		ServerID:       serverID,
		ActorID:        userID,
		ActionType:     models.AuditLogMemberBan,
		ActionCategory: 10,
		Reason:         "spam",
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/"+entryID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.AuditLogEntry
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, entryID, result.ID)
	assert.Equal(t, models.AuditLogMemberBan, result.ActionType)
	assert.Equal(t, "spam", result.Reason)
}

func TestAuditLogHandler_GetAuditLogEntry_NotFound(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	randomEntryID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/"+randomEntryID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogEntry_InvalidEntryID(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "invalid audit log entry id", result["error"])
}

func TestAuditLogHandler_GetActionTypes_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/action-types", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	actionTypes := result["action_types"].([]interface{})
	assert.Len(t, actionTypes, 3)
}

func TestAuditLogHandler_GetCategories_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/categories", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	categories := result["categories"].([]interface{})
	assert.Len(t, categories, 2)
}

func TestAuditLogHandler_GetModerationDashboard_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/dashboard", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.ModerationDashboardSummary
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 100, result.TotalActions)
	assert.Equal(t, "MEMBER_BAN", result.TopAction)
	assert.Equal(t, 5, result.UniqueModerators)
}

func TestAuditLogHandler_GetModerationDashboard_WithDays(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/dashboard?days=30", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuditLogHandler_GetModerationDashboard_InvalidDays(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/dashboard?days=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuditLogHandler_GetModerationTrend_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/trends", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "7d", result["period"])
	assert.NotNil(t, result["trend_data"])
}

func TestAuditLogHandler_GetModeratorActivity_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/moderators", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "7d", result["period"])
	assert.NotNil(t, result["moderators"])
}

func TestAuditLogHandler_GetRepeatOffenders_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/offenders", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "30d", result["period"])
	assert.Equal(t, float64(2), result["min_count"])
	assert.NotNil(t, result["offenders"])
}

func TestAuditLogHandler_GetAutoModStats_Success(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/moderation/automod", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "7d", result["period"])
	assert.NotNil(t, result["stats"])
}

func TestAuditLogHandler_LimitCapping(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	// Request with limit > 100 should cap at 100
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs?limit=200", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, float64(100), result["limit"])
}

func TestAuditLogHandler_ExportAuditLogs_InvalidFormat(t *testing.T) {
	app, _, _, serverID, _ := setupAuditLogTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/export?format=xml", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuditLogHandler_NoPermission_AllEndpoints(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	auditLogSvc := NewMockAuditLogService()
	serverSvc := NewMockServerServiceForAuditLog()
	handler := NewAuditLogHandler(auditLogSvc, serverSvc)

	serverID := uuid.New()
	userID := uuid.New()

	// Add user as member WITHOUT VIEW_AUDIT_LOG permission
	serverSvc.AddMember(serverID, userID, 0)

	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handler.RegisterAuditLogRoutes(api.Group("/servers"))

	endpoints := []string{
		"/api/v1/servers/" + serverID.String() + "/audit-logs",
		"/api/v1/servers/" + serverID.String() + "/audit-logs/action-types",
		"/api/v1/servers/" + serverID.String() + "/audit-logs/categories",
		"/api/v1/servers/" + serverID.String() + "/audit-logs/export",
		"/api/v1/servers/" + serverID.String() + "/moderation/dashboard",
		"/api/v1/servers/" + serverID.String() + "/moderation/trends",
		"/api/v1/servers/" + serverID.String() + "/moderation/moderators",
		"/api/v1/servers/" + serverID.String() + "/moderation/offenders",
		"/api/v1/servers/" + serverID.String() + "/moderation/automod",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		resp, err := app.Test(req)
		require.NoError(t, err, "endpoint: %s", endpoint)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "endpoint: %s", endpoint)
	}
}
