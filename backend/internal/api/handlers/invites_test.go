package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// mockInviteService implements the invite operations needed for testing
type mockInviteService struct {
	createInviteFunc     func(ctx context.Context, serverID, channelID, creatorID uuid.UUID, maxUses int, maxAge time.Duration, temporary bool) (*models.Invite, error)
	getInviteFunc        func(ctx context.Context, code string) (*models.Invite, error)
	useInviteFunc        func(ctx context.Context, code string, userID uuid.UUID) (*models.Server, error)
	deleteInviteFunc     func(ctx context.Context, code string, userID uuid.UUID) error
	getServerInvitesFunc func(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Invite, error)
}

// Mock the InviteService methods that InviteHandlers uses
func (m *mockInviteService) CreateInvite(ctx context.Context, req interface{}) (*models.Invite, error) {
	// Type assertion for the request
	if r, ok := req.(struct {
		ServerID  uuid.UUID
		ChannelID uuid.UUID
		CreatorID uuid.UUID
		MaxUses   int
		MaxAge    time.Duration
		Temporary bool
	}); ok && m.createInviteFunc != nil {
		return m.createInviteFunc(ctx, r.ServerID, r.ChannelID, r.CreatorID, r.MaxUses, r.MaxAge, r.Temporary)
	}
	return nil, nil
}

func (m *mockInviteService) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	if m.getInviteFunc != nil {
		return m.getInviteFunc(ctx, code)
	}
	return nil, nil
}

func (m *mockInviteService) UseInvite(ctx context.Context, code string, userID uuid.UUID) (*models.Server, error) {
	if m.useInviteFunc != nil {
		return m.useInviteFunc(ctx, code, userID)
	}
	return nil, nil
}

func (m *mockInviteService) DeleteInvite(ctx context.Context, code string, userID uuid.UUID) error {
	if m.deleteInviteFunc != nil {
		return m.deleteInviteFunc(ctx, code, userID)
	}
	return nil
}

func (m *mockInviteService) GetServerInvites(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Invite, error) {
	if m.getServerInvitesFunc != nil {
		return m.getServerInvitesFunc(ctx, serverID, userID)
	}
	return nil, nil
}

// Helper function to create a test Fiber app with invite routes
func setupInviteTestApp(mockService *mockInviteService) *fiber.App {
	app := fiber.New()

	// Add middleware to set userID from header for testing (must be before routes)
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
			// Provide a default user ID for routes that don't require specific user
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		}
		return c.Next()
	})

	// Routes using a closure that injects mock service through context
	app.Get("/invites/:code", func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid invite code",
			})
		}

		invite, err := mockService.GetInvite(c.Context(), code)
		if err != nil || invite == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invite not found",
			})
		}

		return c.JSON(toInviteResponse(invite))
	})

	app.Post("/invites/:code/use", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		code := c.Params("code")
		if code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid invite code",
			})
		}

		server, err := mockService.UseInvite(c.Context(), code, userID)
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
				"error": "Invalid invite code",
			})
		}

		err := mockService.DeleteInvite(c.Context(), code, userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

func TestGetInvite_Success(t *testing.T) {
	mockService := &mockInviteService{
		getInviteFunc: func(ctx context.Context, code string) (*models.Invite, error) {
			assert.Equal(t, "ABC123", code)
			return &models.Invite{
				Code:      "ABC123",
				ServerID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				ChannelID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				CreatorID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
				MaxUses:   10,
				Uses:      2,
				Temporary: false,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("GET", "/invites/ABC123", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result InviteResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "ABC123", result.Code)
	assert.Equal(t, 10, result.MaxUses)
	assert.Equal(t, 2, result.Uses)
}

func TestGetInvite_NotFound(t *testing.T) {
	mockService := &mockInviteService{
		getInviteFunc: func(ctx context.Context, code string) (*models.Invite, error) {
			return nil, nil // Simulate not found
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("GET", "/invites/INVALID", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Invite not found", result["error"])
}

func TestGetInvite_MissingCode(t *testing.T) {
	mockService := &mockInviteService{}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("GET", "/invites//", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	// The route won't match with empty code, so we expect a 404 (not found route)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestUseInvite_Success(t *testing.T) {
	testUserID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	
	mockService := &mockInviteService{
		useInviteFunc: func(ctx context.Context, code string, userID uuid.UUID) (*models.Server, error) {
			assert.Equal(t, "JOINME", code)
			assert.Equal(t, testUserID, userID)
		desc := "A test server"
			return &models.Server{
				ID:          uuid.MustParse("55555555-5555-5555-5555-555555555555"),
				Name:        "Test Server",
				Description: &desc,
				OwnerID:     uuid.MustParse("66666666-6666-6666-6666-666666666666"),
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("POST", "/invites/JOINME/use", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.Server
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Test Server", result.Name)
	assert.NotNil(t, result.Description)
	assert.Equal(t, "A test server", *result.Description)
}

func TestUseInvite_InvalidCode(t *testing.T) {
	testUserID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	
	mockService := &mockInviteService{
		useInviteFunc: func(ctx context.Context, code string, userID uuid.UUID) (*models.Server, error) {
			assert.Equal(t, "EXPIRED", code)
			return nil, assert.AnError // Simulate error (expired or invalid)
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("POST", "/invites/EXPIRED/use", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDeleteInvite_Success(t *testing.T) {
	testUserID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	
	mockService := &mockInviteService{
		deleteInviteFunc: func(ctx context.Context, code string, userID uuid.UUID) error {
			assert.Equal(t, "DELETE123", code)
			assert.Equal(t, testUserID, userID)
			return nil
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/invites/DELETE123", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestDeleteInvite_Forbidden(t *testing.T) {
	testUserID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	
	mockService := &mockInviteService{
		deleteInviteFunc: func(ctx context.Context, code string, userID uuid.UUID) error {
			return fiber.NewError(fiber.StatusForbidden, "not authorized to delete this invite")
		},
	}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/invites/SOMEOTHER", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "not authorized to delete this invite", result["error"])
}

func TestDeleteInvite_MissingCode(t *testing.T) {
	testUserID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	mockService := &mockInviteService{}

	app := setupInviteTestApp(mockService)

	req := httptest.NewRequest("DELETE", "/invites//", nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	// Empty code won't match the route
	assert.Equal(t, 404, resp.StatusCode)
}
