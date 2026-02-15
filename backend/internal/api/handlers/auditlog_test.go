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
	logs        []models.AuditLogEntry
	actionTypes []string
	err         error
}

func NewMockAuditLogService() *MockAuditLogService {
	return &MockAuditLogService{
		logs: make([]models.AuditLogEntry, 0),
		actionTypes: []string{
			models.AuditLogServerUpdate,
			models.AuditLogChannelCreate,
			models.AuditLogMemberBan,
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

	servers := api.Group("/servers")
	servers.Get("/:id/audit-logs", handler.GetAuditLogs)
	servers.Get("/:id/audit-logs/action-types", handler.GetActionTypes)
	servers.Get("/:id/audit-logs/:entryId", handler.GetAuditLogEntry)

	return app, auditLogSvc, serverSvc, serverID, userID
}

func TestAuditLogHandler_GetAuditLogs_Success(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	// Add some logs
	targetID := uuid.New()
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   serverID,
		UserID:     userID,
		ActionType: models.AuditLogMemberBan,
		TargetID:   &targetID,
		Reason:     "spam",
		CreatedAt:  time.Now(),
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
		ID:         uuid.New(),
		ServerID:   serverID,
		UserID:     userID,
		ActionType: models.AuditLogMemberBan,
		CreatedAt:  time.Now(),
	})

	filterUserID := uuid.New()
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
			name:        "filter by user_id",
			queryParams: "?user_id=" + filterUserID.String(),
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
			name:        "invalid user_id",
			queryParams: "?user_id=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid user_id",
		},
		{
			name:        "invalid target_id",
			queryParams: "?target_id=invalid",
			statusCode:  http.StatusBadRequest,
			errorMsg:    "invalid target_id",
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

	servers := api.Group("/servers")
	servers.Get("/:id/audit-logs", handler.GetAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogs_AdminHasPermission(t *testing.T) {
	app := fiber.New()
	
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

	servers := api.Group("/servers")
	servers.Get("/:id/audit-logs", handler.GetAuditLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuditLogHandler_GetAuditLogEntry_Success(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	entryID := uuid.New()
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:         entryID,
		ServerID:   serverID,
		UserID:     userID,
		ActionType: models.AuditLogMemberBan,
		Reason:     "spam",
		CreatedAt:  time.Now(),
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

func TestAuditLogHandler_GetActionTypes_NoPermission(t *testing.T) {
	app := fiber.New()
	
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

	servers := api.Group("/servers")
	servers.Get("/:id/audit-logs/action-types", handler.GetActionTypes)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs/action-types", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
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

func TestAuditLogHandler_ResponseFormat(t *testing.T) {
	app, auditLogSvc, _, serverID, userID := setupAuditLogTest(t)

	targetID := uuid.New()
	now := time.Now()
	auditLogSvc.AddLog(models.AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   serverID,
		UserID:     userID,
		ActionType: models.AuditLogMemberBan,
		TargetID:   &targetID,
		Changes: []models.Change{
			{Key: "nickname", OldValue: "old", NewValue: "new"},
		},
		Reason:    "spam",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/audit-logs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, result, "audit_logs")
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "limit")
	assert.Contains(t, result, "offset")

	logs := result["audit_logs"].([]interface{})
	assert.Len(t, logs, 1)

	log := logs[0].(map[string]interface{})
	assert.Contains(t, log, "id")
	assert.Contains(t, log, "server_id")
	assert.Contains(t, log, "user_id")
	assert.Contains(t, log, "action_type")
	assert.Contains(t, log, "created_at")
	assert.Equal(t, models.AuditLogMemberBan, log["action_type"])
}
