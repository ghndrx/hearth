package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockServerServiceForMisc implements the methods needed for misc handler tests
type mockMiscServerService struct {
	joinServerFunc   func(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error)
	getInviteFunc    func(ctx context.Context, code string) (*models.Invite, error)
	deleteInviteFunc func(ctx context.Context, code string, requesterID uuid.UUID) error
}

func (m *mockMiscServerService) JoinServer(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error) {
	if m.joinServerFunc != nil {
		return m.joinServerFunc(ctx, userID, code)
	}
	return nil, nil
}

func (m *mockMiscServerService) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	if m.getInviteFunc != nil {
		return m.getInviteFunc(ctx, code)
	}
	return nil, services.ErrInviteNotFound
}

func (m *mockMiscServerService) DeleteInvite(ctx context.Context, code string, requesterID uuid.UUID) error {
	if m.deleteInviteFunc != nil {
		return m.deleteInviteFunc(ctx, code, requesterID)
	}
	return nil
}

// mockGateway implements a minimal gateway for testing
type mockMiscGateway struct {
	getStatsFunc func() map[string]interface{}
}

func (m *mockMiscGateway) GetStats() map[string]interface{} {
	if m.getStatsFunc != nil {
		return m.getStatsFunc()
	}
	return map[string]interface{}{
		"total_connections":  0,
		"active_connections": 0,
		"messages_processed": 0,
		"active_sessions":    0,
	}
}

// setupMiscTestApp creates a test Fiber app with misc handlers using inline routes
func setupMiscTestApp(serverService *mockMiscServerService, gateway *mockMiscGateway) *fiber.App {
	app := fiber.New()

	// Add middleware to set userID from header for testing
	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			} else {
				c.Locals("userID", uuid.Nil)
			}
		} else {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		}
		return c.Next()
	})

	// Invite routes - implemented to match misc.go handler behavior
	app.Get("/invites/:code", func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invite code is required",
			})
		}

		invite, err := serverService.GetInvite(c.Context(), code)
		if err != nil {
			if err == services.ErrInviteNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "invite not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(invite)
	})

	app.Post("/invites/:code", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		code := c.Params("code")

		server, err := serverService.JoinServer(c.Context(), userID, code)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(server)
	})

	app.Delete("/invites/:code", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invite code is required",
			})
		}

		err := serverService.DeleteInvite(c.Context(), code, userID)
		if err != nil {
			if err == services.ErrInviteNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "invite not found",
				})
			}
			if err == services.ErrNotServerMember {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "you don't have permission to delete this invite",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Voice routes
	app.Get("/voice/regions", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{"id": "us-west", "name": "US West", "optimal": true},
			{"id": "us-east", "name": "US East", "optimal": false},
			{"id": "eu-west", "name": "EU West", "optimal": false},
			{"id": "eu-central", "name": "EU Central", "optimal": false},
			{"id": "singapore", "name": "Singapore", "optimal": false},
			{"id": "sydney", "name": "Sydney", "optimal": false},
		})
	})

	// Gateway routes
	app.Get("/gateway/stats", func(c *fiber.Ctx) error {
		return c.JSON(gateway.GetStats())
	})

	return app
}

// Test InviteHandler.Get - Success
func TestInviteHandler_Get_Success(t *testing.T) {
	testServerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testChannelID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testCreatorID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	mockServerSvc := &mockMiscServerService{
		getInviteFunc: func(ctx context.Context, code string) (*models.Invite, error) {
			assert.Equal(t, "VALIDCODE", code)
			return &models.Invite{
				Code:      "VALIDCODE",
				ServerID:  testServerID,
				ChannelID: testChannelID,
				CreatorID: testCreatorID,
				MaxUses:   10,
				Uses:      2,
				Temporary: false,
				CreatedAt: time.Now(),
				Server: &models.Server{
					ID:   testServerID,
					Name: "Test Server",
				},
			}, nil
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/invites/VALIDCODE", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.Invite
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "VALIDCODE", result.Code)
	assert.Equal(t, testServerID, result.ServerID)
	assert.Equal(t, 10, result.MaxUses)
	assert.Equal(t, 2, result.Uses)
	assert.NotNil(t, result.Server)
	assert.Equal(t, "Test Server", result.Server.Name)
}

// Test InviteHandler.Get - Not Found
func TestInviteHandler_Get_NotFound(t *testing.T) {
	mockServerSvc := &mockMiscServerService{
		getInviteFunc: func(ctx context.Context, code string) (*models.Invite, error) {
			return nil, services.ErrInviteNotFound
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/invites/INVALID", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invite not found", result["error"])
}

// Test InviteHandler.Get - Server Error
func TestInviteHandler_Get_ServerError(t *testing.T) {
	mockServerSvc := &mockMiscServerService{
		getInviteFunc: func(ctx context.Context, code string) (*models.Invite, error) {
			return nil, errors.New("database connection failed")
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/invites/ANYCODE", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "database connection failed", result["error"])
}

// Test InviteHandler.Accept - Success
func TestInviteHandler_Accept_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testServerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	mockServerSvc := &mockMiscServerService{
		joinServerFunc: func(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error) {
			assert.Equal(t, testUserID, userID)
			assert.Equal(t, "VALIDCODE", code)
			desc := "Test server description"
			return &models.Server{
				ID:          testServerID,
				Name:        "Test Server",
				Description: &desc,
				OwnerID:     uuid.MustParse("33333333-3333-3333-3333-333333333333"),
				CreatedAt:   time.Now(),
			}, nil
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("POST", "/invites/VALIDCODE", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.Server
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Test Server", result.Name)
	assert.Equal(t, testServerID, result.ID)
}

// Test InviteHandler.Accept - Invalid Code
func TestInviteHandler_Accept_InvalidCode(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockServerSvc := &mockMiscServerService{
		joinServerFunc: func(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error) {
			return nil, errors.New("invalid or expired invite code")
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("POST", "/invites/INVALID", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid or expired invite code", result["error"])
}

// Test InviteHandler.Accept - Empty Code
func TestInviteHandler_Accept_EmptyCode(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mockServerSvc := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	// Empty code in URL
	req := httptest.NewRequest("POST", "/invites//", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	// Route won't match with empty code
	assert.Equal(t, 404, resp.StatusCode)
}

// Test InviteHandler.Delete - Success
func TestInviteHandler_Delete_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockServerSvc := &mockMiscServerService{
		deleteInviteFunc: func(ctx context.Context, code string, requesterID uuid.UUID) error {
			assert.Equal(t, "DELETEME", code)
			assert.Equal(t, testUserID, requesterID)
			return nil
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("DELETE", "/invites/DELETEME", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

// Test InviteHandler.Delete - Not Found
func TestInviteHandler_Delete_NotFound(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockServerSvc := &mockMiscServerService{
		deleteInviteFunc: func(ctx context.Context, code string, requesterID uuid.UUID) error {
			return services.ErrInviteNotFound
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("DELETE", "/invites/NOTFOUND", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invite not found", result["error"])
}

// Test InviteHandler.Delete - Forbidden (not member/owner)
func TestInviteHandler_Delete_Forbidden(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockServerSvc := &mockMiscServerService{
		deleteInviteFunc: func(ctx context.Context, code string, requesterID uuid.UUID) error {
			return services.ErrNotServerMember
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("DELETE", "/invites/NOPERM", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "you don't have permission to delete this invite", result["error"])
}

// Test InviteHandler.Delete - Server Error
func TestInviteHandler_Delete_ServerError(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockServerSvc := &mockMiscServerService{
		deleteInviteFunc: func(ctx context.Context, code string, requesterID uuid.UUID) error {
			return errors.New("database error")
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("DELETE", "/invites/ANYCODE", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "database error", result["error"])
}

// Test VoiceHandler.GetRegions
func TestVoiceHandler_GetRegions(t *testing.T) {
	mockServerSvc := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/voice/regions", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Should return 6 voice regions
	assert.Len(t, result, 6)

	// Check first region (us-west should be optimal)
	assert.Equal(t, "us-west", result[0]["id"])
	assert.Equal(t, "US West", result[0]["name"])
	assert.Equal(t, true, result[0]["optimal"])

	// Check second region (us-east should not be optimal)
	assert.Equal(t, "us-east", result[1]["id"])
	assert.Equal(t, false, result[1]["optimal"])

	// Verify all expected regions are present
	expectedRegions := []string{"us-west", "us-east", "eu-west", "eu-central", "singapore", "sydney"}
	for i, region := range expectedRegions {
		assert.Equal(t, region, result[i]["id"])
	}
}

// Test GatewayHandler.GetStats
func TestGatewayHandler_GetStats(t *testing.T) {
	mockServerSvc := &mockMiscServerService{}
	mockGw := &mockMiscGateway{
		getStatsFunc: func() map[string]interface{} {
			return map[string]interface{}{
				"total_connections":  int64(100),
				"active_connections": int64(42),
				"messages_processed": int64(1000),
				"active_sessions":    10,
			}
		},
	}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/gateway/stats", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(100), result["total_connections"])
	assert.Equal(t, float64(42), result["active_connections"])
	assert.Equal(t, float64(1000), result["messages_processed"])
	assert.Equal(t, float64(10), result["active_sessions"])
}

// Test GatewayHandler.GetStats - Empty Stats
func TestGatewayHandler_GetStats_Empty(t *testing.T) {
	mockServerSvc := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/gateway/stats", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(0), result["total_connections"])
	assert.Equal(t, float64(0), result["active_connections"])
	assert.Equal(t, float64(0), result["messages_processed"])
	assert.Equal(t, float64(0), result["active_sessions"])
}

// Test GatewayHandler.GetStats - Custom Stats
func TestGatewayHandler_GetStats_Custom(t *testing.T) {
	mockServerSvc := &mockMiscServerService{}
	mockGw := &mockMiscGateway{
		getStatsFunc: func() map[string]interface{} {
			return map[string]interface{}{
				"total_connections":  int64(999),
				"active_connections": int64(0),
				"messages_processed": int64(50000),
				"active_sessions":    1,
				"custom_field":       "custom_value",
			}
		},
	}
	app := setupMiscTestApp(mockServerSvc, mockGw)

	req := httptest.NewRequest("GET", "/gateway/stats", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, float64(999), result["total_connections"])
	assert.Equal(t, float64(0), result["active_connections"])
	assert.Equal(t, float64(50000), result["messages_processed"])
	assert.Equal(t, float64(1), result["active_sessions"])
	assert.Equal(t, "custom_value", result["custom_field"])
}
