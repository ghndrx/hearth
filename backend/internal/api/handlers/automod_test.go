package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

type mockAutoModService struct {
	getServerRulesFunc  func(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error)
	createRuleFunc      func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateAutoModRuleRequest) (*models.ModerationRule, error)
	getRuleFunc         func(ctx context.Context, ruleID uuid.UUID) (*models.ModerationRule, error)
	updateRuleFunc      func(ctx context.Context, ruleID uuid.UUID, req *models.UpdateAutoModRuleRequest) (*models.ModerationRule, error)
	deleteRuleFunc      func(ctx context.Context, ruleID uuid.UUID) error
	getServerAlertsFunc func(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error)
	testContentFunc     func(ctx context.Context, req *models.AutoModTestRequest) (*models.AutoModTestResult, error)
	getRuleStatsFunc    func(ctx context.Context, ruleID uuid.UUID) (*models.AutoModRuleTriggerCount, error)
	resolveAlertFunc    func(ctx context.Context, alertID, userID uuid.UUID) error
}

type mockAutoModServerService struct {
	getMemberFunc func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
}

func setupAutoModTestApp(automodMock *mockAutoModService, serverMock *mockAutoModServerService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	// GET /servers/:id/automod/rules
	app.Get("/servers/:id/automod/rules", func(c *fiber.Ctx) error {
		serverIDStr := c.Params("id")
		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		_, err = serverMock.getMemberFunc(c.Context(), serverID, userID)
		if err != nil {
			if errors.Is(err, services.ErrNotServerMember) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		rules, err := automodMock.getServerRulesFunc(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.Status(fiber.StatusOK).JSON(rules)
	})

	// POST /servers/:id/automod/rules
	app.Post("/servers/:id/automod/rules", func(c *fiber.Ctx) error {
		serverIDStr := c.Params("id")
		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		_, err = serverMock.getMemberFunc(c.Context(), serverID, userID)
		if err != nil {
			if errors.Is(err, services.ErrNotServerMember) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		var req models.CreateAutoModRuleRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}

		rule, err := automodMock.createRuleFunc(c.Context(), serverID, userID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.Status(fiber.StatusCreated).JSON(rule)
	})

	// GET /automod/rules/:id
	app.Get("/automod/rules/:id", func(c *fiber.Ctx) error {
		ruleIDStr := c.Params("id")
		ruleID, err := uuid.Parse(ruleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rule ID"})
		}

		rule, err := automodMock.getRuleFunc(c.Context(), ruleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		if rule == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rule not found"})
		}

		return c.Status(fiber.StatusOK).JSON(rule)
	})

	// DELETE /automod/rules/:id
	app.Delete("/automod/rules/:id", func(c *fiber.Ctx) error {
		ruleIDStr := c.Params("id")
		ruleID, err := uuid.Parse(ruleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rule ID"})
		}

		err = automodMock.deleteRuleFunc(c.Context(), ruleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// GET /servers/:id/automod/alerts
	app.Get("/servers/:id/automod/alerts", func(c *fiber.Ctx) error {
		serverIDStr := c.Params("id")
		serverID, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		_, err = serverMock.getMemberFunc(c.Context(), serverID, userID)
		if err != nil {
			if errors.Is(err, services.ErrNotServerMember) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a server member"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		limit := 50
		offset := 0
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}
		if o := c.Query("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil {
				offset = parsed
			}
		}

		alerts, err := automodMock.getServerAlertsFunc(c.Context(), serverID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.Status(fiber.StatusOK).JSON(alerts)
	})

	// POST /automod/test
	app.Post("/automod/test", func(c *fiber.Ctx) error {
		var req models.AutoModTestRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		if req.ServerID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "server_id is required"})
		}

		if req.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "content is required"})
		}

		result, err := automodMock.testContentFunc(c.Context(), &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.Status(fiber.StatusOK).JSON(result)
	})

	// GET /automod/rules/:id/stats
	app.Get("/automod/rules/:id/stats", func(c *fiber.Ctx) error {
		ruleIDStr := c.Params("id")
		ruleID, err := uuid.Parse(ruleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rule ID"})
		}

		stats, err := automodMock.getRuleStatsFunc(c.Context(), ruleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		}

		return c.Status(fiber.StatusOK).JSON(stats)
	})

	return app
}

func TestListRules_Success(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	automodMock := &mockAutoModService{
		getServerRulesFunc: func(ctx context.Context, sID uuid.UUID) ([]*models.ModerationRule, error) {
			return []*models.ModerationRule{
				{ID: ruleID, ServerID: serverID, Name: "No Spam"},
			}, nil
		},
	}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{}, nil
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/automod/rules", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListRules_InvalidServerID(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/servers/invalid-id/automod/rules", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestListRules_NotAMember(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/servers/"+uuid.New().String()+"/automod/rules", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestCreateRule_Success(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	automodMock := &mockAutoModService{
		createRuleFunc: func(ctx context.Context, sID, uID uuid.UUID, r *models.CreateAutoModRuleRequest) (*models.ModerationRule, error) {
			return &models.ModerationRule{ID: ruleID, ServerID: sID, Name: r.Name}, nil
		},
	}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{}, nil
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"No Spam Rule"}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/automod/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "No Spam Rule", result["name"])
}

func TestCreateRule_MissingName(t *testing.T) {
	serverID := uuid.New()

	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{}, nil
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/automod/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateRule_NotAMember(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"Test Rule"}`
	req := httptest.NewRequest(http.MethodPost, "/servers/"+uuid.New().String()+"/automod/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestGetRule_Success(t *testing.T) {
	ruleID := uuid.New()
	serverID := uuid.New()

	automodMock := &mockAutoModService{
		getRuleFunc: func(ctx context.Context, rID uuid.UUID) (*models.ModerationRule, error) {
			return &models.ModerationRule{ID: ruleID, ServerID: serverID, Name: "Test Rule"}, nil
		},
	}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/automod/rules/"+ruleID.String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "Test Rule", result["name"])
}

func TestGetRule_NotFound(t *testing.T) {
	automodMock := &mockAutoModService{
		getRuleFunc: func(ctx context.Context, rID uuid.UUID) (*models.ModerationRule, error) {
			return nil, nil
		},
	}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/automod/rules/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetRule_InvalidID(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/automod/rules/invalid-id", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteRule_Success(t *testing.T) {
	automodMock := &mockAutoModService{
		deleteRuleFunc: func(ctx context.Context, rID uuid.UUID) error {
			return nil
		},
	}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/automod/rules/"+uuid.New().String(), nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteRule_InvalidID(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodDelete, "/automod/rules/invalid-id", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestListAlerts_SuccessWithPagination(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()

	automodMock := &mockAutoModService{
		getServerAlertsFunc: func(ctx context.Context, sID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 20, offset)
			return []*models.AutoModAlertSummary{}, nil
		},
	}
	serverMock := &mockAutoModServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{}, nil
		},
	}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/automod/alerts?limit=10&offset=20", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTestContent_Success(t *testing.T) {
	serverID := uuid.New()

	automodMock := &mockAutoModService{
		testContentFunc: func(ctx context.Context, r *models.AutoModTestRequest) (*models.AutoModTestResult, error) {
			return &models.AutoModTestResult{}, nil
		},
	}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"server_id":"` + serverID.String() + `","content":"test message"}`
	req := httptest.NewRequest(http.MethodPost, "/automod/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTestContent_MissingContent(t *testing.T) {
	serverID := uuid.New()

	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"server_id":"` + serverID.String() + `","content":""}`
	req := httptest.NewRequest(http.MethodPost, "/automod/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestTestContent_MissingServerID(t *testing.T) {
	automodMock := &mockAutoModService{}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"content":"test message"}`
	req := httptest.NewRequest(http.MethodPost, "/automod/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetRuleStats_Success(t *testing.T) {
	ruleID := uuid.New()

	automodMock := &mockAutoModService{
		getRuleStatsFunc: func(ctx context.Context, rID uuid.UUID) (*models.AutoModRuleTriggerCount, error) {
			return &models.AutoModRuleTriggerCount{}, nil
		},
	}
	serverMock := &mockAutoModServerService{}

	app := setupAutoModTestApp(automodMock, serverMock)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/automod/rules/"+ruleID.String()+"/stats", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
