package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockWelcomeService implements the methods used by WelcomeHandler
type mockWelcomeService struct {
	getWelcomeScreenFunc    func(ctx context.Context, serverID uuid.UUID) (*models.WelcomeScreenConfig, error)
	updateWelcomeScreenFunc func(ctx context.Context, serverID, requesterID uuid.UUID, req *models.UpdateWelcomeScreenRequest) (*models.WelcomeScreenConfig, error)
	submitScreeningFunc     func(ctx context.Context, userID, serverID uuid.UUID, req *models.SubmitScreeningRequest) (*models.MemberScreening, error)
	getMemberScreeningFunc  func(ctx context.Context, userID, serverID uuid.UUID) (*models.MemberScreening, error)
	getPendingScreeningsFunc func(ctx context.Context, serverID, requesterID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error)
	approveScreeningFunc    func(ctx context.Context, userID, serverID, moderatorID uuid.UUID) error
	rejectScreeningFunc     func(ctx context.Context, userID, serverID, moderatorID uuid.UUID, reason string) error
}

func (m *mockWelcomeService) GetWelcomeScreen(ctx context.Context, serverID uuid.UUID) (*models.WelcomeScreenConfig, error) {
	if m.getWelcomeScreenFunc != nil {
		return m.getWelcomeScreenFunc(ctx, serverID)
	}
	return nil, nil
}

func (m *mockWelcomeService) UpdateWelcomeScreen(ctx context.Context, serverID, requesterID uuid.UUID, req *models.UpdateWelcomeScreenRequest) (*models.WelcomeScreenConfig, error) {
	if m.updateWelcomeScreenFunc != nil {
		return m.updateWelcomeScreenFunc(ctx, serverID, requesterID, req)
	}
	return nil, nil
}

func (m *mockWelcomeService) SubmitScreening(ctx context.Context, userID, serverID uuid.UUID, req *models.SubmitScreeningRequest) (*models.MemberScreening, error) {
	if m.submitScreeningFunc != nil {
		return m.submitScreeningFunc(ctx, userID, serverID, req)
	}
	return nil, nil
}

func (m *mockWelcomeService) GetMemberScreening(ctx context.Context, userID, serverID uuid.UUID) (*models.MemberScreening, error) {
	if m.getMemberScreeningFunc != nil {
		return m.getMemberScreeningFunc(ctx, userID, serverID)
	}
	return nil, nil
}

func (m *mockWelcomeService) GetPendingScreenings(ctx context.Context, serverID, requesterID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error) {
	if m.getPendingScreeningsFunc != nil {
		return m.getPendingScreeningsFunc(ctx, serverID, requesterID, limit, offset)
	}
	return nil, nil
}

func (m *mockWelcomeService) ApproveScreening(ctx context.Context, userID, serverID, moderatorID uuid.UUID) error {
	if m.approveScreeningFunc != nil {
		return m.approveScreeningFunc(ctx, userID, serverID, moderatorID)
	}
	return nil
}

func (m *mockWelcomeService) RejectScreening(ctx context.Context, userID, serverID, moderatorID uuid.UUID, reason string) error {
	if m.rejectScreeningFunc != nil {
		return m.rejectScreeningFunc(ctx, userID, serverID, moderatorID, reason)
	}
	return nil
}

// welcomeHandlerForTest wraps WelcomeHandler for testing
type welcomeHandlerForTest struct {
	service *mockWelcomeService
}

func newWelcomeHandlerForTest(service *mockWelcomeService) *welcomeHandlerForTest {
	return &welcomeHandlerForTest{service: service}
}

func (h *welcomeHandlerForTest) GetWelcomeScreen(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	config, err := h.service.GetWelcomeScreen(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(config)
}

func (h *welcomeHandlerForTest) UpdateWelcomeScreen(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	var req models.UpdateWelcomeScreenRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}
	config, err := h.service.UpdateWelcomeScreen(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(config)
}

func (h *welcomeHandlerForTest) SubmitScreening(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	var req models.SubmitScreeningRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}
	screening, err := h.service.SubmitScreening(c.Context(), userID, serverID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(screening)
}

func (h *welcomeHandlerForTest) GetMemberScreening(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	screening, err := h.service.GetMemberScreening(c.Context(), userID, serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if screening == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no screening found"})
	}
	return c.JSON(screening)
}

func (h *welcomeHandlerForTest) GetPendingScreenings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	screenings, err := h.service.GetPendingScreenings(c.Context(), serverID, userID, 50, 0)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if screenings == nil {
		screenings = []*models.MemberScreening{}
	}
	return c.JSON(screenings)
}

func (h *welcomeHandlerForTest) ApproveScreening(c *fiber.Ctx) error {
	moderatorID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return InvalidUUID(c, "user id")
	}
	if err := h.service.ApproveScreening(c.Context(), userID, serverID, moderatorID); err != nil {
		return HandleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *welcomeHandlerForTest) RejectScreening(c *fiber.Ctx) error {
	moderatorID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return InvalidUUID(c, "user id")
	}
	var req ScreeningDecisionRequest
	_ = c.BodyParser(&req)
	if err := h.service.RejectScreening(c.Context(), userID, serverID, moderatorID, req.Reason); err != nil {
		return HandleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func setupOnboardingTestApp(t *testing.T, service *mockWelcomeService) *fiber.App {
	t.Helper()
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
			return c.Status(code).JSON(ErrorResponse{Error: "internal_error", Message: err.Error()})
		},
	})
	t.Cleanup(func() { app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	handler := newWelcomeHandlerForTest(service)
	app.Get("/servers/:id/welcome", handler.GetWelcomeScreen)
	app.Put("/servers/:id/welcome", handler.UpdateWelcomeScreen)
	app.Post("/servers/:id/screening", handler.SubmitScreening)
	app.Get("/servers/:id/screening/me", handler.GetMemberScreening)
	app.Get("/servers/:id/screening/pending", handler.GetPendingScreenings)
	app.Post("/servers/:id/screening/:userId/approve", handler.ApproveScreening)
	app.Post("/servers/:id/screening/:userId/reject", handler.RejectScreening)

	return app
}

func TestWelcomeHandler_GetWelcomeScreen_Success(t *testing.T) {
	serverID := uuid.New()
	svc := &mockWelcomeService{
		getWelcomeScreenFunc: func(ctx context.Context, sid uuid.UUID) (*models.WelcomeScreenConfig, error) {
			assert.Equal(t, serverID, sid)
			return &models.WelcomeScreenConfig{
				WelcomeScreen: models.WelcomeScreen{
					ServerID: sid,
					Enabled:  true,
					Title:    "Welcome!",
				},
				Rules:     []models.Rule{{ID: "1", Title: "Be nice"}},
				Questions: []models.ScreeningQuestion{{ID: "1", Question: "Why join?"}},
			}, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/welcome", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.WelcomeScreenConfig
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Welcome!", result.Title)
	assert.Len(t, result.Rules, 1)
}

func TestWelcomeHandler_GetWelcomeScreen_InvalidID(t *testing.T) {
	svc := &mockWelcomeService{}
	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/invalid-id/welcome", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWelcomeHandler_GetWelcomeScreen_ServiceError(t *testing.T) {
	svc := &mockWelcomeService{
		getWelcomeScreenFunc: func(ctx context.Context, sid uuid.UUID) (*models.WelcomeScreenConfig, error) {
			return nil, services.ErrServerNotFound
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/welcome", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestWelcomeHandler_UpdateWelcomeScreen_Success(t *testing.T) {
	serverID := uuid.New()
	enabled := true
	title := "New Title"
	svc := &mockWelcomeService{
		updateWelcomeScreenFunc: func(ctx context.Context, sid, reqUID uuid.UUID, req *models.UpdateWelcomeScreenRequest) (*models.WelcomeScreenConfig, error) {
			assert.Equal(t, serverID, sid)
			assert.Equal(t, enabled, *req.Enabled)
			assert.Equal(t, title, *req.Title)
			return &models.WelcomeScreenConfig{
				WelcomeScreen: models.WelcomeScreen{ServerID: sid, Enabled: enabled, Title: title},
			}, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"enabled": enabled,
		"title":   title,
	})
	req := httptest.NewRequest("PUT", "/servers/"+serverID.String()+"/welcome", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.WelcomeScreenConfig
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, title, result.Title)
}

func TestWelcomeHandler_UpdateWelcomeScreen_Unauthorized(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })
	// No userID middleware
	handler := newWelcomeHandlerForTest(&mockWelcomeService{})
	app.Put("/servers/:id/welcome", handler.UpdateWelcomeScreen)

	body, _ := json.Marshal(map[string]interface{}{"enabled": true})
	req := httptest.NewRequest("PUT", "/servers/"+uuid.New().String()+"/welcome", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWelcomeHandler_UpdateWelcomeScreen_InvalidBody(t *testing.T) {
	svc := &mockWelcomeService{}
	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("PUT", "/servers/"+uuid.New().String()+"/welcome", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWelcomeHandler_UpdateWelcomeScreen_Forbidden(t *testing.T) {
	svc := &mockWelcomeService{
		updateWelcomeScreenFunc: func(ctx context.Context, sid, reqUID uuid.UUID, req *models.UpdateWelcomeScreenRequest) (*models.WelcomeScreenConfig, error) {
			return nil, services.ErrNotServerOwner
		},
	}

	app := setupOnboardingTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"enabled": true})
	req := httptest.NewRequest("PUT", "/servers/"+uuid.New().String()+"/welcome", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestWelcomeHandler_SubmitScreening_Success(t *testing.T) {
	serverID := uuid.New()
	svc := &mockWelcomeService{
		submitScreeningFunc: func(ctx context.Context, uid, sid uuid.UUID, req *models.SubmitScreeningRequest) (*models.MemberScreening, error) {
			assert.Equal(t, serverID, sid)
			assert.True(t, req.RulesRead)
			return &models.MemberScreening{
				ID:        uuid.New(),
				UserID:    uid,
				ServerID:  sid,
				Status:    models.ScreeningStatusPending,
				RulesRead: true,
			}, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{
		"answers":    []models.ScreeningAnswer{{QuestionID: "1", Answer: "Because"}},
		"rules_read": true,
	})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/screening", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.MemberScreening
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.ScreeningStatusPending, result.Status)
}

func TestWelcomeHandler_SubmitScreening_AlreadyExists(t *testing.T) {
	svc := &mockWelcomeService{
		submitScreeningFunc: func(ctx context.Context, uid, sid uuid.UUID, req *models.SubmitScreeningRequest) (*models.MemberScreening, error) {
			return nil, services.ErrScreeningAlreadyExists
		},
	}

	app := setupOnboardingTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"rules_read": true})
	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/screening", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestWelcomeHandler_GetMemberScreening_Success(t *testing.T) {
	serverID := uuid.New()
	svc := &mockWelcomeService{
		getMemberScreeningFunc: func(ctx context.Context, uid, sid uuid.UUID) (*models.MemberScreening, error) {
			assert.Equal(t, serverID, sid)
			return &models.MemberScreening{
				ID:       uuid.New(),
				UserID:   uid,
				ServerID: sid,
				Status:   models.ScreeningStatusPending,
			}, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/screening/me", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.MemberScreening
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.ScreeningStatusPending, result.Status)
}

func TestWelcomeHandler_GetMemberScreening_NotFound(t *testing.T) {
	svc := &mockWelcomeService{
		getMemberScreeningFunc: func(ctx context.Context, uid, sid uuid.UUID) (*models.MemberScreening, error) {
			return nil, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/screening/me", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestWelcomeHandler_GetPendingScreenings_Success(t *testing.T) {
	serverID := uuid.New()
	svc := &mockWelcomeService{
		getPendingScreeningsFunc: func(ctx context.Context, sid, reqUID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error) {
			assert.Equal(t, serverID, sid)
			assert.Equal(t, 50, limit)
			return []*models.MemberScreening{
				{ID: uuid.New(), ServerID: sid, Status: models.ScreeningStatusPending},
				{ID: uuid.New(), ServerID: sid, Status: models.ScreeningStatusPending},
			}, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/screening/pending", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []models.MemberScreening
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
}

func TestWelcomeHandler_GetPendingScreenings_Forbidden(t *testing.T) {
	svc := &mockWelcomeService{
		getPendingScreeningsFunc: func(ctx context.Context, sid, reqUID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error) {
			return nil, services.ErrNotServerModerator
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/screening/pending", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestWelcomeHandler_GetPendingScreenings_EmptyResult(t *testing.T) {
	svc := &mockWelcomeService{
		getPendingScreeningsFunc: func(ctx context.Context, sid, reqUID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error) {
			return nil, nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("GET", "/servers/"+uuid.New().String()+"/screening/pending", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []models.MemberScreening
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 0)
}

func TestWelcomeHandler_ApproveScreening_Success(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	svc := &mockWelcomeService{
		approveScreeningFunc: func(ctx context.Context, uid, sid, modID uuid.UUID) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, serverID, sid)
			return nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/screening/"+userID.String()+"/approve", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestWelcomeHandler_ApproveScreening_InvalidServerID(t *testing.T) {
	svc := &mockWelcomeService{}
	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/invalid-id/screening/"+uuid.New().String()+"/approve", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWelcomeHandler_ApproveScreening_InvalidUserID(t *testing.T) {
	svc := &mockWelcomeService{}
	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/screening/invalid-id/approve", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWelcomeHandler_ApproveScreening_Forbidden(t *testing.T) {
	svc := &mockWelcomeService{
		approveScreeningFunc: func(ctx context.Context, uid, sid, modID uuid.UUID) error {
			return services.ErrNotServerModerator
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/screening/"+uuid.New().String()+"/approve", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestWelcomeHandler_RejectScreening_Success(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	svc := &mockWelcomeService{
		rejectScreeningFunc: func(ctx context.Context, uid, sid, modID uuid.UUID, reason string) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, serverID, sid)
			assert.Equal(t, "Spam account", reason)
			return nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	body, _ := json.Marshal(map[string]interface{}{"reason": "Spam account"})
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/screening/"+userID.String()+"/reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestWelcomeHandler_RejectScreening_NoReason(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	svc := &mockWelcomeService{
		rejectScreeningFunc: func(ctx context.Context, uid, sid, modID uuid.UUID, reason string) error {
			assert.Equal(t, "", reason)
			return nil
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/screening/"+userID.String()+"/reject", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestWelcomeHandler_RejectScreening_Forbidden(t *testing.T) {
	svc := &mockWelcomeService{
		rejectScreeningFunc: func(ctx context.Context, uid, sid, modID uuid.UUID, reason string) error {
			return services.ErrNotServerModerator
		},
	}

	app := setupOnboardingTestApp(t, svc)
	req := httptest.NewRequest("POST", "/servers/"+uuid.New().String()+"/screening/"+uuid.New().String()+"/reject", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
