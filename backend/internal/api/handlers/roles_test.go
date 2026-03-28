package handlers

import (
	"bytes"
	"context"
	"encoding/json"
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

type roleServiceMock struct {
	mock.Mock
}

func (m *roleServiceMock) CreateRole(ctx context.Context, serverID, creatorID uuid.UUID, name string, color int, permissions int64) (*models.Role, error) {
	args := m.Called(ctx, serverID, creatorID, name, color, permissions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *roleServiceMock) GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *roleServiceMock) GetRole(ctx context.Context, roleID, requesterID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, roleID, requesterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *roleServiceMock) UpdateRole(ctx context.Context, roleID, requesterID uuid.UUID, updates *models.RoleUpdate) (*models.Role, error) {
	args := m.Called(ctx, roleID, requesterID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *roleServiceMock) DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, roleID, requesterID)
	return args.Error(0)
}

func (m *roleServiceMock) UpdateRolePositions(ctx context.Context, serverID, requesterID uuid.UUID, positions map[uuid.UUID]int) error {
	args := m.Called(ctx, serverID, requesterID, positions)
	return args.Error(0)
}

func (m *roleServiceMock) AddRoleToMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID, requesterID)
	return args.Error(0)
}

func (m *roleServiceMock) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID, requesterID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID, roleID, requesterID)
	return args.Error(0)
}

func setupRoleTestApp(tb testing.TB, mockService *roleServiceMock) (*fiber.App, uuid.UUID) {
	app := fiber.New()
	tb.Cleanup(func() { app.Shutdown() })
	userID := uuid.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handlers := NewRoleHandlers(mockService)

	app.Post("/servers/:serverID/roles", handlers.CreateRole)
	app.Get("/servers/:serverID/roles", handlers.GetRoles)
	app.Get("/servers/:serverID/roles/:roleID", handlers.GetRole)
	app.Patch("/servers/:serverID/roles/:roleID", handlers.UpdateRole)
	app.Delete("/servers/:serverID/roles/:roleID", handlers.DeleteRole)
	app.Put("/servers/:serverID/members/:memberID/roles/:roleID", handlers.AddMemberRole)
	app.Delete("/servers/:serverID/members/:memberID/roles/:roleID", handlers.RemoveMemberRole)

	return app, userID
}

func TestCreateRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
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
	app, _ := setupRoleTestApp(t, new(roleServiceMock))

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
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("CreateRole", mock.Anything, serverID, userID, "Admin", 0, int64(0)).
		Return(nil, services.ErrNotServerMember)

	body, _ := json.Marshal(map[string]interface{}{"name": "Admin"})
	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRoles_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
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
	app, _ := setupRoleTestApp(t, new(roleServiceMock))

	req := httptest.NewRequest(http.MethodGet, "/servers/invalid/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetRoles_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("GetServerRoles", mock.Anything, serverID, userID).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRoles_EmptyList(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("GetServerRoles", mock.Anything, serverID, userID).Return([]*models.Role{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 0)

	mockService.AssertExpectations(t)
}

func TestGetRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	expectedRole := &models.Role{
		ID:       roleID,
		ServerID: serverID,
		Name:     "Moderator",
		Color:    0x3498db,
		Position: 1,
	}

	mockService.On("GetRole", mock.Anything, roleID, userID).Return(expectedRole, nil)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result models.Role
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, expectedRole.Name, result.Name)

	mockService.AssertExpectations(t)
}

func TestGetRole_InvalidRoleID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/invalid-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetRole_RoleNotFound(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("GetRole", mock.Anything, roleID, userID).Return(nil, services.ErrRoleNotFound)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRole_NotServerMember(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("GetRole", mock.Anything, roleID, userID).Return(nil, services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestGetRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("GetRole", mock.Anything, roleID, userID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
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
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/invalid-uuid", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRole_InvalidBody(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()
	roleID := uuid.New()

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("UpdateRole", mock.Anything, roleID, userID, mock.Anything).Return(nil, services.ErrRoleNotFound)

	body, _ := json.Marshal(map[string]interface{}{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateRole_WithAllFields(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	newName := "Admin"
	newColor := 0xff0000
	newPerms := int64(8)
	newHoist := true
	newMentionable := true

	expectedRole := &models.Role{
		ID:          roleID,
		ServerID:    serverID,
		Name:        newName,
		Color:       newColor,
		Permissions: newPerms,
		Hoist:       newHoist,
		Mentionable: newMentionable,
		Position:    1,
	}

	mockService.On("UpdateRole", mock.Anything, roleID, userID, mock.MatchedBy(func(u *models.RoleUpdate) bool {
		return u.Name != nil && *u.Name == newName &&
			u.Color != nil && *u.Color == newColor &&
			u.Permissions != nil && *u.Permissions == newPerms &&
			u.Hoist != nil && *u.Hoist == newHoist &&
			u.Mentionable != nil && *u.Mentionable == newMentionable
	})).Return(expectedRole, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        newName,
		"color":       newColor,
		"permissions": newPerms,
		"hoist":       newHoist,
		"mentionable": newMentionable,
	})

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestUpdateRole_WithPosition(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	newPosition := 5

	expectedRole := &models.Role{
		ID:       roleID,
		ServerID: serverID,
		Name:     "Role",
		Position: newPosition,
	}

	mockService.On("UpdateRole", mock.Anything, roleID, userID, mock.MatchedBy(func(u *models.RoleUpdate) bool {
		return u.Position != nil && *u.Position == newPosition
	})).Return(expectedRole, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"position": newPosition,
	})

	req := httptest.NewRequest(http.MethodPatch, "/servers/"+serverID.String()+"/roles/"+roleID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteRole_InvalidRoleID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/invalid-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteRole_CannotDeleteDefault(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(services.ErrCannotDeleteDefault)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestDeleteRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	roleID := uuid.New()

	mockService.On("DeleteRole", mock.Anything, roleID, userID).Return(services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestAddMemberRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestAddMemberRole_InvalidServerID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	memberID := uuid.New()
	roleID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/servers/invalid-uuid/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_InvalidMemberID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()
	roleID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/invalid-uuid/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_InvalidRoleID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()
	memberID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/invalid-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddMemberRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestAddMemberRole_RoleNotFound(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("AddRoleToMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrRoleNotFound)

	req := httptest.NewRequest(http.MethodPut, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestRemoveMemberRole_Success(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("RemoveRoleFromMember", mock.Anything, serverID, memberID, roleID, userID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestRemoveMemberRole_InvalidServerID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	memberID := uuid.New()
	roleID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/servers/invalid-uuid/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_InvalidMemberID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()
	roleID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/invalid-uuid/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_InvalidRoleID(t *testing.T) {
	app, _ := setupRoleTestApp(t, new(roleServiceMock))
	serverID := uuid.New()
	memberID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/invalid-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRemoveMemberRole_ServiceError(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()

	mockService.On("RemoveRoleFromMember", mock.Anything, serverID, memberID, roleID, userID).Return(services.ErrNotServerMember)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/members/"+memberID.String()+"/roles/"+roleID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateRole_WithAllFields(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()

	expectedRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        "Admin",
		Color:       0xff0000,
		Position:    1,
		Permissions: 8,
		Hoist:       true,
		Mentionable: true,
		CreatedAt:   time.Now(),
	}

	mockService.On("CreateRole", mock.Anything, serverID, userID, "Admin", 0xff0000, int64(8)).
		Return(expectedRole, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Admin",
		"color":       0xff0000,
		"permissions": 8,
		"hoist":       true,
		"mentionable": true,
	})

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	mockService.AssertExpectations(t)
}

func TestCreateRole_MissingName(t *testing.T) {
	mockService := new(roleServiceMock)
	app, userID := setupRoleTestApp(t, mockService)
	serverID := uuid.New()

	mockService.On("CreateRole", mock.Anything, serverID, userID, "", 0xff0000, int64(0)).
		Return(nil, services.ErrNotServerMember)

	body, _ := json.Marshal(map[string]interface{}{
		"color": 0xff0000,
	})

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockService.AssertExpectations(t)
}
