package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockSmartModerationService mocks SmartModerationServiceInterface
type MockSmartModerationService struct {
	mock.Mock
}

func (m *MockSmartModerationService) GetOrCreateSettings(ctx context.Context, serverID uuid.UUID) (*models.ModerationSettings, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ModerationSettings), args.Error(1)
}

func (m *MockSmartModerationService) UpdateSettings(ctx context.Context, serverID uuid.UUID, req *models.UpdateModerationSettingsRequest) (*models.ModerationSettings, error) {
	args := m.Called(ctx, serverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ModerationSettings), args.Error(1)
}

func (m *MockSmartModerationService) GetKeywordRules(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.KeywordRule), args.Error(1)
}

func (m *MockSmartModerationService) CreateKeywordRule(ctx context.Context, serverID, createdBy uuid.UUID, req *models.CreateKeywordRuleRequest) (*models.KeywordRule, error) {
	args := m.Called(ctx, serverID, createdBy, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KeywordRule), args.Error(1)
}

func (m *MockSmartModerationService) UpdateKeywordRule(ctx context.Context, ruleID uuid.UUID, req *models.UpdateKeywordRuleRequest) (*models.KeywordRule, error) {
	args := m.Called(ctx, ruleID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KeywordRule), args.Error(1)
}

func (m *MockSmartModerationService) DeleteKeywordRule(ctx context.Context, ruleID uuid.UUID) error {
	args := m.Called(ctx, ruleID)
	return args.Error(0)
}

func (m *MockSmartModerationService) AnalyzeContent(ctx context.Context, req *models.AnalyzeContentRequest) (*models.AnalyzeContentResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AnalyzeContentResult), args.Error(1)
}

func (m *MockSmartModerationService) GetModerationLogs(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.ModerationLogSummary, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ModerationLogSummary), args.Error(1)
}

func (m *MockSmartModerationService) GetMemberModerationHistory(ctx context.Context, serverID, memberID uuid.UUID, limit, offset int) ([]*models.ModerationLog, error) {
	args := m.Called(ctx, serverID, memberID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ModerationLog), args.Error(1)
}

func (m *MockSmartModerationService) TakeModerationAction(ctx context.Context, serverID, moderatorID uuid.UUID, req *models.ModerationActionRequest) (*models.ModerationLog, error) {
	args := m.Called(ctx, serverID, moderatorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ModerationLog), args.Error(1)
}

func (m *MockSmartModerationService) ResolveModerationLog(ctx context.Context, logID, resolvedBy uuid.UUID) error {
	args := m.Called(ctx, logID, resolvedBy)
	return args.Error(0)
}

func (m *MockSmartModerationService) GetDashboardStats(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardStats, error) {
	args := m.Called(ctx, serverID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ModerationDashboardStats), args.Error(1)
}

func (m *MockSmartModerationService) GetUserViolationSummary(ctx context.Context, serverID, userID uuid.UUID) (*models.UserViolationSummary, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserViolationSummary), args.Error(1)
}

func (m *MockSmartModerationService) ResetMemberViolations(ctx context.Context, serverID, memberID uuid.UUID) error {
	args := m.Called(ctx, serverID, memberID)
	return args.Error(0)
}

// MockSmartModerationServerService mocks SmartModerationServerService
type MockSmartModerationServerService struct {
	mock.Mock
}

func (m *MockSmartModerationServerService) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

// setupSmartModerationTest creates a test environment for smart moderation handler tests
func setupSmartModerationTest(tb testing.TB) (*SmartModerationHandler, *MockSmartModerationService, *MockSmartModerationServerService, *fiber.App, uuid.UUID) {
	modMock := new(MockSmartModerationService)
	serverMock := new(MockSmartModerationServerService)
	handler := NewSmartModerationHandler(modMock, serverMock)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			var httpErr *HTTPError
			if errors.As(err, &httpErr) {
				return c.Status(httpErr.Status).JSON(ErrorResponse{
					Error:   httpErr.ErrorType,
					Message: httpErr.Message,
					Code:    httpErr.Code,
				})
			}
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		},
	})
	tb.Cleanup(func() { _ = app.Shutdown() })
	userID := uuid.New()

	// Inject userID middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Register routes matching actual API routes
	servers := app.Group("/servers/:id")
	servers.Get("/moderation/settings", handler.GetSettings)
	servers.Patch("/moderation/settings", handler.UpdateSettings)
	servers.Get("/moderation/rules", handler.ListKeywordRules)
	servers.Post("/moderation/rules", handler.CreateKeywordRule)
	servers.Get("/moderation/logs", handler.ListModerationLogs)

	app.Patch("/moderation/rules/:id", handler.UpdateKeywordRule)
	app.Delete("/moderation/rules/:id", handler.DeleteKeywordRule)
	app.Post("/moderation/analyze", handler.AnalyzeContent)

	return handler, modMock, serverMock, app, userID
}

// --- GetSmartModerationConfig ---

func TestGetSmartModerationConfig_Success(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetOrCreateSettings", mock.Anything, serverID).Return(&models.ModerationSettings{
		ID:       uuid.New(),
		ServerID: serverID,
		Enabled:  true,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/settings", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

func TestGetSmartModerationConfig_Error(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetOrCreateSettings", mock.Anything, serverID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/settings", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

// --- UpdateSmartModerationConfig ---

func TestUpdateSmartModerationConfig_Success(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("UpdateSettings", mock.Anything, serverID, mock.AnythingOfType("*models.UpdateModerationSettingsRequest")).Return(&models.ModerationSettings{
		ID:       uuid.New(),
		ServerID: serverID,
		Enabled:  false,
	}, nil)

	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/moderation/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

func TestUpdateSmartModerationConfig_ValidationError(t *testing.T) {
	_, _, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/moderation/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	serverMock.AssertExpectations(t)
}

// --- GetAutoModRules ---

func TestGetAutoModRules_Success(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetKeywordRules", mock.Anything, serverID).Return([]*models.KeywordRule{
		{ID: uuid.New(), ServerID: serverID, Name: "No Spam", Pattern: "spam"},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/rules", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 1)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

func TestGetAutoModRules_Error(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetKeywordRules", mock.Anything, serverID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/rules", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

// --- CreateAutoModRule ---

func TestCreateAutoModRule_Success(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()
	ruleID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("CreateKeywordRule", mock.Anything, serverID, userID, mock.AnythingOfType("*models.CreateKeywordRuleRequest")).Return(&models.KeywordRule{
		ID:       ruleID,
		ServerID: serverID,
		Name:     "No Spam",
		Pattern:  "spam",
	}, nil)

	body, _ := json.Marshal(models.CreateKeywordRuleRequest{
		Name:    "No Spam",
		Pattern: "spam",
	})
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/moderation/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

func TestCreateAutoModRule_ValidationError(t *testing.T) {
	_, _, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)

	body, _ := json.Marshal(map[string]interface{}{"name": "", "pattern": ""})
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/moderation/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	serverMock.AssertExpectations(t)
}

// --- UpdateAutoModRule ---

func TestUpdateAutoModRule_Success(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	ruleID := uuid.New()

	modMock.On("UpdateKeywordRule", mock.Anything, ruleID, mock.AnythingOfType("*models.UpdateKeywordRuleRequest")).Return(&models.KeywordRule{
		ID:      ruleID,
		Name:    "Updated Rule",
		Pattern: "badword",
	}, nil)

	body, _ := json.Marshal(map[string]interface{}{"name": "Updated Rule"})
	req := httptest.NewRequest(http.MethodPatch, "/moderation/rules/"+ruleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	modMock.AssertExpectations(t)
}

func TestUpdateAutoModRule_NotFound(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	ruleID := uuid.New()

	modMock.On("UpdateKeywordRule", mock.Anything, ruleID, mock.AnythingOfType("*models.UpdateKeywordRuleRequest")).Return(nil, services.ErrModerationRuleNotFound)

	body, _ := json.Marshal(map[string]interface{}{"name": "Updated Rule"})
	req := httptest.NewRequest(http.MethodPatch, "/moderation/rules/"+ruleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	modMock.AssertExpectations(t)
}

func TestUpdateAutoModRule_ValidationError(t *testing.T) {
	_, _, _, app, _ := setupSmartModerationTest(t)
	ruleID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/moderation/rules/"+ruleID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- DeleteAutoModRule ---

func TestDeleteAutoModRule_Success(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	ruleID := uuid.New()

	modMock.On("DeleteKeywordRule", mock.Anything, ruleID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/moderation/rules/"+ruleID.String(), nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	modMock.AssertExpectations(t)
}

func TestDeleteAutoModRule_NotFound(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	ruleID := uuid.New()

	modMock.On("DeleteKeywordRule", mock.Anything, ruleID).Return(services.ErrModerationRuleNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/moderation/rules/"+ruleID.String(), nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	modMock.AssertExpectations(t)
}

// --- GetSmartModerationLogs ---

func TestGetSmartModerationLogs_Success(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetModerationLogs", mock.Anything, serverID, 50, 0).Return([]*models.ModerationLogSummary{
		{ID: uuid.New(), ServerID: serverID, MemberID: uuid.New(), Reason: "spam"},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/logs", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

func TestGetSmartModerationLogs_Error(t *testing.T) {
	_, modMock, serverMock, app, userID := setupSmartModerationTest(t)
	serverID := uuid.New()

	serverMock.On("GetMember", mock.Anything, serverID, userID).Return(&models.Member{}, nil)
	modMock.On("GetModerationLogs", mock.Anything, serverID, 50, 0).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/moderation/logs", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	modMock.AssertExpectations(t)
	serverMock.AssertExpectations(t)
}

// --- TestAutoModRule ---

func TestTestAutoModRule_Success(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	serverID := uuid.New()

	modMock.On("AnalyzeContent", mock.Anything, mock.AnythingOfType("*models.AnalyzeContentRequest")).Return(&models.AnalyzeContentResult{
		Violations:  []models.ViolationDetail{},
		TotalScore:  0,
		ShouldBlock: false,
		Actions:     []models.ModerationActionType{},
	}, nil)

	body, _ := json.Marshal(models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "hello world",
	})
	req := httptest.NewRequest(http.MethodPost, "/moderation/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	modMock.AssertExpectations(t)
}

func TestTestAutoModRule_Error(t *testing.T) {
	_, modMock, _, app, _ := setupSmartModerationTest(t)
	serverID := uuid.New()

	modMock.On("AnalyzeContent", mock.Anything, mock.AnythingOfType("*models.AnalyzeContentRequest")).Return(nil, assert.AnError)

	body, _ := json.Marshal(models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "hello world",
	})
	req := httptest.NewRequest(http.MethodPost, "/moderation/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	modMock.AssertExpectations(t)
}
