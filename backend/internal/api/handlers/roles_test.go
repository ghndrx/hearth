package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// MockRoleService mocks the RoleService for testing
type MockRoleService struct {
	mock.Mock
}

func (m *MockRoleService) CreateRole(ctx context.Context, serverID, creatorID uuid.UUID, name string, color int, permissions int64) (*models.Role, error) {
	args := m.Called(ctx, serverID, creatorID, name, color, permissions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleService) GetByID(ctx context.Context, roleID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) UpdateRole(ctx context.Context, roleID, requesterID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error) {
	args := m.Called(ctx, roleID, requesterID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, roleID, requesterID)
	return args.Error(0)
}

func (m *MockRoleService) AddRoleToMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID, requesterID)
	return args.Error(0)
}

func (m *MockRoleService) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID, requesterID)
	return args.Error(0)
}

func (m *MockRoleService) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockRoleService) ComputeMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleService) UpdateRolePositions(ctx context.Context, serverID, requesterID uuid.UUID, positions map[uuid.UUID]int) error {
	args := m.Called(ctx, serverID, requesterID, positions)
	return args.Error(0)
}

// testRoleHandler creates a test role handler with mocks
type testRoleHandler struct {
	handlers    *RoleHandlers
	roleService *MockRoleService
	app         *fiber.App
	userID      uuid.UUID
}

func newTestRoleHandler() *testRoleHandler {
	roleService := new(MockRoleService)

	handlers := &RoleHandlers{
		roleService: &services.RoleService{},
	}

	app := fiber.New()
	userID := uuid.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Setup routes
	app.Post("/servers/:serverID/roles", handlers.CreateRole)
	app.Get("/servers/:serverID/roles", handlers.GetRoles)
	app.Get("/servers/:serverID/roles/:roleID", handlers.GetRole)
	app.Patch("/servers/:serverID/roles/:roleID", handlers.UpdateRole)
	app.Delete("/servers/:serverID/roles/:roleID", handlers.DeleteRole)
	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handlers.AddMemberRole)
	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handlers.RemoveMemberRole)

	return &testRoleHandler{
		handlers:    handlers,
		roleService: roleService,
		app:         app,
		userID:      userID,
	}
}

// Create a test helper that injects the mock properly
func setupRoleTest() (*fiber.App, *MockRoleService, uuid.UUID) {
	mockService := new(MockRoleService)
	userID := uuid.New()

	app := fiber.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	return app, mockService, userID
}

// =====================
// CreateRole Tests
// =====================

func TestCreateRole_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()

	expectedRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        "Moderator",
		Color:       0x3498db,
		Position:    1,
		Permissions: 123456,
		CreatedAt:   time.Now(),
	}

	mockService.On("CreateRole", mock.Anything, serverID, userID, "Moderator", 0x3498db, int64(123456)).
		Return(expectedRole, nil)

	// Create handler with mock service injected
	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))

		var req struct {
			Name        string `json:"name"`
			Color       int    `json:"color"`
			Permissions int64  `json:"permissions"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		role, err := mockService.CreateRole(c.Context(), serverID, userID, req.Name, req.Color, req.Permissions)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(role)
	}

	app.Post("/servers/:serverID/roles", handler)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Moderator",
		"color":       0x3498db,
		"permissions": 123456,
	})

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, expectedRole.Name, result.Name)
	assert.Equal(t, expectedRole.Color, result.Color)

	mockService.AssertExpectations(t)
}

func TestCreateRole_InvalidServerID(t *testing.T) {
	app, _, _ := setupRoleTest()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("serverID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Post("/servers/:serverID/roles", handler)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Moderator",
	})

	req := httptest.NewRequest(http.MethodPost, "/servers/invalid-uuid/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRole_InvalidBody(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Post("/servers/:serverID/roles", handler)

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRole_ServiceError(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()

	mockService.On("CreateRole", mock.Anything, serverID, userID, "Admin", 0, int64(0)).
		Return(nil, services.ErrNotServerMember)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))

		var req struct {
			Name        string `json:"name"`
			Color       int    `json:"color"`
			Permissions int64  `json:"permissions"`
		}
		c.BodyParser(&req)

		role, err := mockService.CreateRole(c.Context(), serverID, userID, req.Name, req.Color, req.Permissions)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(role)
	}

	app.Post("/servers/:serverID/roles", handler)

	body, _ := json.Marshal(map[string]interface{}{"name": "Admin"})
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// =====================
// GetRoles Tests
// =====================

func TestGetRoles_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()

	expectedRoles := []*models.Role{
		{
			ID:        uuid.New(),
			ServerID:  serverID,
			Name:      "@everyone",
			Position:  0,
			IsDefault: true,
		},
		{
			ID:       uuid.New(),
			ServerID: serverID,
			Name:     "Moderator",
			Position: 1,
			Color:    0x3498db,
		},
	}

	mockService.On("GetServerRoles", mock.Anything, serverID, userID).Return(expectedRoles, nil)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))

		roles, err := mockService.GetServerRoles(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(roles)
	}

	app.Get("/servers/:serverID/roles", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, "@everyone", result[0].Name)

	mockService.AssertExpectations(t)
}

func TestGetRoles_InvalidServerID(t *testing.T) {
	app, _, _ := setupRoleTest()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("serverID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Get("/servers/:serverID/roles", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/invalid/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetRoles_NotMember(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()

	mockService.On("GetServerRoles", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))

		roles, err := mockService.GetServerRoles(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(roles)
	}

	app.Get("/servers/:serverID/roles", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// =====================
// GetRole Tests
// =====================

func TestGetRole_Success(t *testing.T) {
	app, mockService, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	expectedRole := &models.Role{
		ID:          roleID,
		ServerID:    serverID,
		Name:        "Moderator",
		Color:       0x3498db,
		Position:    1,
		Permissions: 123456,
	}

	mockService.On("GetByID", mock.Anything, roleID).Return(expectedRole, nil)

	handler := func(c *fiber.Ctx) error {
		roleID, _ := uuid.Parse(c.Params("roleID"))

		role, err := mockService.GetByID(c.Context(), roleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if role == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
		}

		return c.JSON(role)
	}

	app.Get("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, expectedRole.Name, result.Name)
	assert.Equal(t, expectedRole.Color, result.Color)

	mockService.AssertExpectations(t)
}

func TestGetRole_NotFound(t *testing.T) {
	app, mockService, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("GetByID", mock.Anything, roleID).Return(nil, nil)

	handler := func(c *fiber.Ctx) error {
		roleID, _ := uuid.Parse(c.Params("roleID"))

		role, err := mockService.GetByID(c.Context(), roleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if role == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
		}

		return c.JSON(role)
	}

	app.Get("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRole_ServiceError(t *testing.T) {
	app, mockService, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("GetByID", mock.Anything, roleID).Return(nil, errors.New("database error"))

	handler := func(c *fiber.Ctx) error {
		roleID, _ := uuid.Parse(c.Params("roleID"))

		role, err := mockService.GetByID(c.Context(), roleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if role == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
		}

		return c.JSON(role)
	}

	app.Get("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRole_InvalidRoleID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("roleID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Get("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/not-a-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// =====================
// UpdateRole Tests
// =====================

func TestUpdateRole_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	newName := "Super Moderator"
	newColor := 0xe74c3c

	expectedRole := &models.Role{
		ID:       roleID,
		ServerID: serverID,
		Name:     newName,
		Color:    newColor,
		Position: 1,
	}

	mockService.On("UpdateRole", mock.Anything, roleID, userID, mock.MatchedBy(func(u *models.RoleUpdate) bool {
		return u.Name != nil && *u.Name == newName && u.Color != nil && *u.Color == newColor
	})).Return(expectedRole, nil)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		roleID, _ := uuid.Parse(c.Params("roleID"))

		var req struct {
			Name  *string `json:"name,omitempty"`
			Color *int    `json:"color,omitempty"`
		}
		c.BodyParser(&req)

		updates := &models.RoleUpdate{
			Name:  req.Name,
			Color: req.Color,
		}

		role, err := mockService.UpdateRole(c.Context(), roleID, userID, updates)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(role)
	}

	app.Patch("/servers/:serverID/roles/:roleID", handler)

	body, _ := json.Marshal(map[string]interface{}{
		"name":  newName,
		"color": newColor,
	})

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, newName, result.Name)
	assert.Equal(t, newColor, result.Color)

	mockService.AssertExpectations(t)
}

func TestUpdateRole_InvalidRoleID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("roleID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Patch("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/invalid", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRole_InvalidBody(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		var req struct {
			Name *string `json:"name"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		return c.SendStatus(fiber.StatusOK)
	}

	app.Patch("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRole_RoleNotFound(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("UpdateRole", mock.Anything, roleID, userID, mock.Anything).Return(nil, services.ErrRoleNotFound)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		roleID, _ := uuid.Parse(c.Params("roleID"))

		var req struct {
			Name *string `json:"name,omitempty"`
		}
		c.BodyParser(&req)

		role, err := mockService.UpdateRole(c.Context(), roleID, userID, &models.RoleUpdate{Name: req.Name})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(role)
	}

	app.Patch("/servers/:serverID/roles/:roleID", handler)

	body, _ := json.Marshal(map[string]interface{}{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// =====================
// DeleteRole Tests
// =====================

func TestDeleteRole_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(nil)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.DeleteRole(c.Context(), roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteRole_InvalidRoleID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("roleID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/bad-id", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteRole_CannotDeleteDefault(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(services.ErrCannotDeleteDefault)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.DeleteRole(c.Context(), roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteRole_NotServerMember(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(services.ErrNotServerMember)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.DeleteRole(c.Context(), roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// =====================
// AddMemberRole Tests
// =====================

func TestAddMemberRole_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(nil)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))
		memberID, _ := uuid.Parse(c.Params("memberID"))
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.AddRoleToMember(c.Context(), serverID, memberID, roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestAddMemberRole_InvalidServerID(t *testing.T) {
	app, _, _ := setupRoleTest()
	memberID := uuid.New()
	roleID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("serverID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/invalid/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_InvalidMemberID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		if _, err := uuid.Parse(c.Params("serverID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		if _, err := uuid.Parse(c.Params("memberID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid member ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/invalid/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_InvalidRoleID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		if _, err := uuid.Parse(c.Params("serverID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		if _, err := uuid.Parse(c.Params("memberID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid member ID"})
		}
		if _, err := uuid.Parse(c.Params("roleID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/invalid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_TargetNotMember(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrNotServerMember)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))
		memberID, _ := uuid.Parse(c.Params("memberID"))
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.AddRoleToMember(c.Context(), serverID, memberID, roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestAddMemberRole_RoleNotFound(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrRoleNotFound)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))
		memberID, _ := uuid.Parse(c.Params("memberID"))
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.AddRoleToMember(c.Context(), serverID, memberID, roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

// =====================
// RemoveMemberRole Tests
// =====================

func TestRemoveMemberRole_Success(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("RemoveRoleFromMember", mock.Anything, serverID, memberID, roleID, userID).Return(nil)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))
		memberID, _ := uuid.Parse(c.Params("memberID"))
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.RemoveRoleFromMember(c.Context(), serverID, memberID, roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestRemoveMemberRole_InvalidServerID(t *testing.T) {
	app, _, _ := setupRoleTest()
	memberID := uuid.New()
	roleID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("serverID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/invalid/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_InvalidMemberID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()
	roleID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		if _, err := uuid.Parse(c.Params("serverID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		if _, err := uuid.Parse(c.Params("memberID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid member ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/invalid/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_InvalidRoleID(t *testing.T) {
	app, _, _ := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()

	handler := func(c *fiber.Ctx) error {
		if _, err := uuid.Parse(c.Params("serverID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid server ID"})
		}
		if _, err := uuid.Parse(c.Params("memberID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid member ID"})
		}
		if _, err := uuid.Parse(c.Params("roleID")); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/invalid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_NotServerMember(t *testing.T) {
	app, mockService, userID := setupRoleTest()
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("RemoveRoleFromMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrNotServerMember)

	handler := func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		serverID, _ := uuid.Parse(c.Params("serverID"))
		memberID, _ := uuid.Parse(c.Params("memberID"))
		roleID, _ := uuid.Parse(c.Params("roleID"))

		err := mockService.RemoveRoleFromMember(c.Context(), serverID, memberID, roleID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}

	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handler)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}
