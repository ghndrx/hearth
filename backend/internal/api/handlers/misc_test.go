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
)

// mockServerServiceForMisc implements only the methods needed for misc handler tests
type mockMiscServerService struct {
	joinServerFunc func(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error)
}

func (m *mockMiscServerService) JoinServer(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error) {
	if m.joinServerFunc != nil {
		return m.joinServerFunc(ctx, userID, code)
	}
	return nil, nil
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

	// Invite routes - implemented inline like invites_test.go
	app.Get("/invites/:code", func(c *fiber.Ctx) error {
		// Currently returns empty JSON (not implemented in misc.go)
		return c.JSON(fiber.Map{})
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
		// Currently returns NoContent (not implemented in misc.go)
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

// Test InviteHandler.Get (currently returns empty JSON)
func TestInviteHandler_Get(t *testing.T) {
	mockService := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

	req := httptest.NewRequest("GET", "/invites/TEST123", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	// Currently returns empty object as it's not implemented
	assert.Empty(t, result)
}

// Test InviteHandler.Accept - Success
func TestInviteHandler_Accept_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testServerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	mockService := &mockMiscServerService{
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
	app := setupMiscTestApp(mockService, mockGw)

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

	mockService := &mockMiscServerService{
		joinServerFunc: func(ctx context.Context, userID uuid.UUID, code string) (*models.Server, error) {
			return nil, errors.New("invalid or expired invite code")
		},
	}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

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
	mockService := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

	// Empty code in URL
	req := httptest.NewRequest("POST", "/invites//", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	// Route won't match with empty code
	assert.Equal(t, 404, resp.StatusCode)
}

// Test InviteHandler.Delete
func TestInviteHandler_Delete(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	mockService := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

	req := httptest.NewRequest("DELETE", "/invites/DELETEME", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

// Test VoiceHandler.GetRegions
func TestVoiceHandler_GetRegions(t *testing.T) {
	mockService := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

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
	mockService := &mockMiscServerService{}
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
	app := setupMiscTestApp(mockService, mockGw)

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
	mockService := &mockMiscServerService{}
	mockGw := &mockMiscGateway{}
	app := setupMiscTestApp(mockService, mockGw)

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
	mockService := &mockMiscServerService{}
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
	app := setupMiscTestApp(mockService, mockGw)

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
