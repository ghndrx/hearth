package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

// Helper function to create a test RoleService with mocks
func newTestRoleService() (*RoleService, *MockRoleRepository, *MockServerRepository, *MockCacheService, *MockEventBus) {
	roleRepo := new(MockRoleRepository)
	serverRepo := new(MockServerRepository)
	cache := new(MockCacheService)
	eventBus := new(MockEventBus)

	service := NewRoleService(roleRepo, serverRepo, cache, eventBus)
	return service, roleRepo, serverRepo, cache, eventBus
}

// ============================================
// CreateRole Tests
// ============================================

func TestCreateRole_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
		JoinedAt: time.Now(),
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{}, nil)
	roleRepo.On("Create", ctx, mock.AnythingOfType("*models.Role")).Return(nil)
	eventBus.On("Publish", "role.created", mock.Anything).Return()

	role, err := service.CreateRole(ctx, serverID, creatorID, "Moderator", 0x00FF00, models.PermManageMessages)

	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Moderator", role.Name)
	assert.Equal(t, 0x00FF00, role.Color)
	assert.Equal(t, models.PermManageMessages, role.Permissions)
	assert.Equal(t, 0, role.Position)
	assert.Equal(t, serverID, role.ServerID)

	serverRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestCreateRole_WithExistingRoles(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{
		UserID:   creatorID,
		ServerID: serverID,
	}

	existingRoles := []*models.Role{
		{ID: uuid.New(), Name: "Admin", Position: 0},
		{ID: uuid.New(), Name: "Member", Position: 1},
	}

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return(existingRoles, nil)
	roleRepo.On("Create", ctx, mock.MatchedBy(func(r *models.Role) bool {
		return r.Position == 2 // New role at end
	})).Return(nil)
	eventBus.On("Publish", "role.created", mock.Anything).Return()

	role, err := service.CreateRole(ctx, serverID, creatorID, "VIP", 0xFFD700, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, role.Position)

	roleRepo.AssertExpectations(t)
}

func TestCreateRole_NotServerMember(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, nil)

	role, err := service.CreateRole(ctx, serverID, creatorID, "Test", 0, 0)

	assert.Nil(t, role)
	assert.Equal(t, ErrNotServerMember, err)

	serverRepo.AssertExpectations(t)
	roleRepo.AssertNotCalled(t, "Create")
}

func TestCreateRole_GetMemberError(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(nil, errors.New("db error"))

	role, err := service.CreateRole(ctx, serverID, creatorID, "Test", 0, 0)

	assert.Nil(t, role)
	assert.Equal(t, ErrNotServerMember, err)
}

func TestCreateRole_GetRolesError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{UserID: creatorID, ServerID: serverID}
	dbErr := errors.New("db error")

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return(nil, dbErr)

	role, err := service.CreateRole(ctx, serverID, creatorID, "Test", 0, 0)

	assert.Nil(t, role)
	assert.Equal(t, dbErr, err)
}

func TestCreateRole_CreateError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	creatorID := uuid.New()

	member := &models.Member{UserID: creatorID, ServerID: serverID}
	dbErr := errors.New("db error")

	serverRepo.On("GetMember", ctx, serverID, creatorID).Return(member, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return([]*models.Role{}, nil)
	roleRepo.On("Create", ctx, mock.Anything).Return(dbErr)

	role, err := service.CreateRole(ctx, serverID, creatorID, "Test", 0, 0)

	assert.Nil(t, role)
	assert.Equal(t, dbErr, err)
}

// ============================================
// UpdateRole Tests
// ============================================

func TestUpdateRole_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingRole := &models.Role{
		ID:          roleID,
		ServerID:    serverID,
		Name:        "Old Name",
		Color:       0xFF0000,
		Permissions: 0,
	}

	member := &models.Member{UserID: requesterID, ServerID: serverID}
	newName := "New Name"
	newColor := 0x00FF00

	roleRepo.On("GetByID", ctx, roleID).Return(existingRole, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("Update", ctx, mock.MatchedBy(func(r *models.Role) bool {
		return r.Name == newName && r.Color == newColor
	})).Return(nil)
	eventBus.On("Publish", "role.updated", mock.Anything).Return()

	updates := &models.RoleUpdate{
		Name:  &newName,
		Color: &newColor,
	}

	role, err := service.UpdateRole(ctx, roleID, requesterID, updates)

	require.NoError(t, err)
	assert.Equal(t, newName, role.Name)
	assert.Equal(t, newColor, role.Color)

	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUpdateRole_AllFields(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingRole := &models.Role{
		ID:          roleID,
		ServerID:    serverID,
		Name:        "Old",
		Color:       0,
		Permissions: 0,
		Hoist:       false,
		Mentionable: false,
	}

	member := &models.Member{UserID: requesterID, ServerID: serverID}
	name := "New"
	color := 0xFFFFFF
	perms := models.PermManageChannels | models.PermManageRoles
	hoist := true
	mentionable := true

	roleRepo.On("GetByID", ctx, roleID).Return(existingRole, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("Update", ctx, mock.Anything).Return(nil)
	eventBus.On("Publish", "role.updated", mock.Anything).Return()

	updates := &models.RoleUpdate{
		Name:        &name,
		Color:       &color,
		Permissions: &perms,
		Hoist:       &hoist,
		Mentionable: &mentionable,
	}

	role, err := service.UpdateRole(ctx, roleID, requesterID, updates)

	require.NoError(t, err)
	assert.Equal(t, name, role.Name)
	assert.Equal(t, color, role.Color)
	assert.Equal(t, perms, role.Permissions)
	assert.True(t, role.Hoist)
	assert.True(t, role.Mentionable)
}

func TestUpdateRole_NotFound(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	requesterID := uuid.New()

	roleRepo.On("GetByID", ctx, roleID).Return(nil, nil)

	role, err := service.UpdateRole(ctx, roleID, requesterID, &models.RoleUpdate{})

	assert.Nil(t, role)
	assert.Equal(t, ErrRoleNotFound, err)
}

func TestUpdateRole_GetRoleError(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	roleRepo.On("GetByID", ctx, roleID).Return(nil, dbErr)

	role, err := service.UpdateRole(ctx, roleID, requesterID, &models.RoleUpdate{})

	assert.Nil(t, role)
	assert.Equal(t, dbErr, err)
}

func TestUpdateRole_NotServerMember(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	existingRole := &models.Role{ID: roleID, ServerID: serverID}

	roleRepo.On("GetByID", ctx, roleID).Return(existingRole, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	role, err := service.UpdateRole(ctx, roleID, requesterID, &models.RoleUpdate{})

	assert.Nil(t, role)
	assert.Equal(t, ErrNotServerMember, err)
}

func TestUpdateRole_UpdateError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	existingRole := &models.Role{ID: roleID, ServerID: serverID}
	member := &models.Member{UserID: requesterID, ServerID: serverID}
	name := "New"

	roleRepo.On("GetByID", ctx, roleID).Return(existingRole, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("Update", ctx, mock.Anything).Return(dbErr)

	role, err := service.UpdateRole(ctx, roleID, requesterID, &models.RoleUpdate{Name: &name})

	assert.Nil(t, role)
	assert.Equal(t, dbErr, err)
}

// ============================================
// DeleteRole Tests
// ============================================

func TestDeleteRole_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	role := &models.Role{
		ID:        roleID,
		ServerID:  serverID,
		IsDefault: false,
	}
	member := &models.Member{UserID: requesterID, ServerID: serverID}

	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("Delete", ctx, roleID).Return(nil)
	eventBus.On("Publish", "role.deleted", mock.Anything).Return()

	err := service.DeleteRole(ctx, roleID, requesterID)

	require.NoError(t, err)
	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestDeleteRole_NotFound(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	requesterID := uuid.New()

	roleRepo.On("GetByID", ctx, roleID).Return(nil, nil)

	err := service.DeleteRole(ctx, roleID, requesterID)

	assert.Equal(t, ErrRoleNotFound, err)
}

func TestDeleteRole_GetRoleError(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	roleRepo.On("GetByID", ctx, roleID).Return(nil, dbErr)

	err := service.DeleteRole(ctx, roleID, requesterID)

	assert.Equal(t, dbErr, err)
}

func TestDeleteRole_CannotDeleteDefaultRole(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	role := &models.Role{
		ID:        roleID,
		ServerID:  serverID,
		IsDefault: true, // @everyone role
	}

	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)

	err := service.DeleteRole(ctx, roleID, requesterID)

	assert.Equal(t, ErrCannotDeleteDefault, err)
}

func TestDeleteRole_NotServerMember(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()

	role := &models.Role{ID: roleID, ServerID: serverID, IsDefault: false}

	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.DeleteRole(ctx, roleID, requesterID)

	assert.Equal(t, ErrNotServerMember, err)
}

func TestDeleteRole_DeleteError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	role := &models.Role{ID: roleID, ServerID: serverID, IsDefault: false}
	member := &models.Member{UserID: requesterID, ServerID: serverID}

	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("Delete", ctx, roleID).Return(dbErr)

	err := service.DeleteRole(ctx, roleID, requesterID)

	assert.Equal(t, dbErr, err)
}

// ============================================
// GetServerRoles Tests
// ============================================

func TestGetServerRoles_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{UserID: requesterID, ServerID: serverID}
	roles := []*models.Role{
		{ID: uuid.New(), Name: "Admin", Position: 0},
		{ID: uuid.New(), Name: "Mod", Position: 1},
		{ID: uuid.New(), Name: "Member", Position: 2},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("GetByServerID", ctx, serverID).Return(roles, nil)

	result, err := service.GetServerRoles(ctx, serverID, requesterID)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "Admin", result[0].Name)
}

func TestGetServerRoles_NotServerMember(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	result, err := service.GetServerRoles(ctx, serverID, requesterID)

	assert.Nil(t, result)
	assert.Equal(t, ErrNotServerMember, err)
}

func TestGetServerRoles_GetMemberError(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, errors.New("db error"))

	result, err := service.GetServerRoles(ctx, serverID, requesterID)

	assert.Nil(t, result)
	assert.Equal(t, ErrNotServerMember, err)
}

// ============================================
// GetRole Tests
// ============================================

func TestGetRole_Success(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()
	serverID := uuid.New()

	expectedRole := &models.Role{
		ID:          roleID,
		ServerID:    serverID,
		Name:        "Moderator",
		Color:       0x3498db,
		Position:    1,
		Permissions: models.PermManageMessages,
	}

	roleRepo.On("GetByID", ctx, roleID).Return(expectedRole, nil)

	result, err := service.GetRole(ctx, roleID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedRole.ID, result.ID)
	assert.Equal(t, expectedRole.Name, result.Name)
	assert.Equal(t, expectedRole.Color, result.Color)
	roleRepo.AssertExpectations(t)
}

func TestGetRole_NotFound(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()

	roleRepo.On("GetByID", ctx, roleID).Return(nil, nil)

	result, err := service.GetRole(ctx, roleID)

	require.NoError(t, err)
	assert.Nil(t, result)
	roleRepo.AssertExpectations(t)
}

func TestGetRole_RepositoryError(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	roleID := uuid.New()

	roleRepo.On("GetByID", ctx, roleID).Return(nil, errors.New("database connection failed"))

	result, err := service.GetRole(ctx, roleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "database connection failed", err.Error())
	roleRepo.AssertExpectations(t)
}

// ============================================
// UpdateRolePositions Tests
// ============================================

func TestUpdateRolePositions_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{UserID: requesterID, ServerID: serverID}
	positions := map[uuid.UUID]int{
		uuid.New(): 0,
		uuid.New(): 1,
		uuid.New(): 2,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("UpdatePositions", ctx, serverID, positions).Return(nil)

	err := service.UpdateRolePositions(ctx, serverID, requesterID, positions)

	require.NoError(t, err)
	roleRepo.AssertExpectations(t)
}

func TestUpdateRolePositions_NotServerMember(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.UpdateRolePositions(ctx, serverID, requesterID, map[uuid.UUID]int{})

	assert.Equal(t, ErrNotServerMember, err)
}

func TestUpdateRolePositions_UpdateError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	member := &models.Member{UserID: requesterID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("UpdatePositions", ctx, serverID, mock.Anything).Return(dbErr)

	err := service.UpdateRolePositions(ctx, serverID, requesterID, map[uuid.UUID]int{})

	assert.Equal(t, dbErr, err)
}

// ============================================
// AddRoleToMember Tests
// ============================================

func TestAddRoleToMember_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	requesterMember := &models.Member{UserID: requesterID, ServerID: serverID}
	targetMember := &models.Member{UserID: userID, ServerID: serverID}
	role := &models.Role{ID: roleID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(requesterMember, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(targetMember, nil)
	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)
	roleRepo.On("AddRoleToMember", ctx, serverID, userID, roleID).Return(nil)
	eventBus.On("Publish", "member.role_added", mock.Anything).Return()

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	require.NoError(t, err)
	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestAddRoleToMember_RequesterNotMember(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, ErrNotServerMember, err)
}

func TestAddRoleToMember_TargetNotMember(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	requesterMember := &models.Member{UserID: requesterID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(requesterMember, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, nil)

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, ErrNotServerMember, err)
}

func TestAddRoleToMember_RoleNotFound(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	requesterMember := &models.Member{UserID: requesterID, ServerID: serverID}
	targetMember := &models.Member{UserID: userID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(requesterMember, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(targetMember, nil)
	roleRepo.On("GetByID", ctx, roleID).Return(nil, nil)

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, ErrRoleNotFound, err)
}

func TestAddRoleToMember_RoleFromDifferentServer(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	otherServerID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	requesterMember := &models.Member{UserID: requesterID, ServerID: serverID}
	targetMember := &models.Member{UserID: userID, ServerID: serverID}
	role := &models.Role{ID: roleID, ServerID: otherServerID} // Different server

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(requesterMember, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(targetMember, nil)
	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, ErrRoleNotFound, err)
}

func TestAddRoleToMember_AddError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	requesterMember := &models.Member{UserID: requesterID, ServerID: serverID}
	targetMember := &models.Member{UserID: userID, ServerID: serverID}
	role := &models.Role{ID: roleID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(requesterMember, nil)
	serverRepo.On("GetMember", ctx, serverID, userID).Return(targetMember, nil)
	roleRepo.On("GetByID", ctx, roleID).Return(role, nil)
	roleRepo.On("AddRoleToMember", ctx, serverID, userID, roleID).Return(dbErr)

	err := service.AddRoleToMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, dbErr, err)
}

// ============================================
// RemoveRoleFromMember Tests
// ============================================

func TestRemoveRoleFromMember_Success(t *testing.T) {
	service, roleRepo, serverRepo, _, eventBus := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	member := &models.Member{UserID: requesterID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("RemoveRoleFromMember", ctx, serverID, userID, roleID).Return(nil)
	eventBus.On("Publish", "member.role_removed", mock.Anything).Return()

	err := service.RemoveRoleFromMember(ctx, serverID, userID, roleID, requesterID)

	require.NoError(t, err)
	roleRepo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestRemoveRoleFromMember_NotServerMember(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	err := service.RemoveRoleFromMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, ErrNotServerMember, err)
}

func TestRemoveRoleFromMember_RemoveError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requesterID := uuid.New()
	dbErr := errors.New("db error")

	member := &models.Member{UserID: requesterID, ServerID: serverID}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	roleRepo.On("RemoveRoleFromMember", ctx, serverID, userID, roleID).Return(dbErr)

	err := service.RemoveRoleFromMember(ctx, serverID, userID, roleID, requesterID)

	assert.Equal(t, dbErr, err)
}

// ============================================
// GetMemberRoles Tests
// ============================================

func TestGetMemberRoles_Success(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	roles := []*models.Role{
		{ID: uuid.New(), Name: "Admin"},
		{ID: uuid.New(), Name: "Mod"},
	}

	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return(roles, nil)

	result, err := service.GetMemberRoles(ctx, serverID, userID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetMemberRoles_Error(t *testing.T) {
	service, roleRepo, _, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	dbErr := errors.New("db error")

	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return(nil, dbErr)

	result, err := service.GetMemberRoles(ctx, serverID, userID)

	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

// ============================================
// ComputeMemberPermissions Tests
// ============================================

func TestComputeMemberPermissions_Owner(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, ownerID)

	require.NoError(t, err)
	assert.Equal(t, models.PermissionAll, perms)
}

func TestComputeMemberPermissions_ServerNotFound(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	serverRepo.On("GetByID", ctx, serverID).Return(nil, nil)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	assert.Zero(t, perms)
	assert.Equal(t, ErrServerNotFound, err)
}

func TestComputeMemberPermissions_GetServerError(t *testing.T) {
	service, _, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	dbErr := errors.New("db error")

	serverRepo.On("GetByID", ctx, serverID).Return(nil, dbErr)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	assert.Zero(t, perms)
	assert.Equal(t, dbErr, err)
}

func TestComputeMemberPermissions_CombineRolePermissions(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	roles := []*models.Role{
		{ID: uuid.New(), Permissions: models.PermSendMessages},
		{ID: uuid.New(), Permissions: models.PermManageMessages},
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return(roles, nil)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	require.NoError(t, err)
	assert.True(t, perms&models.PermSendMessages != 0)
	assert.True(t, perms&models.PermManageMessages != 0)
}

func TestComputeMemberPermissions_Administrator(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	roles := []*models.Role{
		{ID: uuid.New(), Permissions: models.PermAdministrator},
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return(roles, nil)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	require.NoError(t, err)
	assert.Equal(t, models.PermissionAll, perms)
}

func TestComputeMemberPermissions_GetRolesError(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	dbErr := errors.New("db error")

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return(nil, dbErr)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	assert.Zero(t, perms)
	assert.Equal(t, dbErr, err)
}

func TestComputeMemberPermissions_NoRoles(t *testing.T) {
	service, roleRepo, serverRepo, _, _ := newTestRoleService()
	ctx := context.Background()
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	server := &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)
	roleRepo.On("GetMemberRoles", ctx, serverID, userID).Return([]*models.Role{}, nil)

	perms, err := service.ComputeMemberPermissions(ctx, serverID, userID)

	require.NoError(t, err)
	assert.Zero(t, perms)
}
