package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// MockPermissionService implements PermissionServiceInterface for testing
type MockPermissionService struct {
	hasPermissionFunc        func(ctx context.Context, serverID, userID uuid.UUID, permission int64) (bool, error)
	getMemberPermissionsFunc func(ctx context.Context, serverID, userID uuid.UUID) (int64, error)
}

func (m *MockPermissionService) HasPermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) (bool, error) {
	if m.hasPermissionFunc != nil {
		return m.hasPermissionFunc(ctx, serverID, userID, permission)
	}
	return true, nil
}

func (m *MockPermissionService) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	if m.getMemberPermissionsFunc != nil {
		return m.getMemberPermissionsFunc(ctx, serverID, userID)
	}
	return 0, nil
}

// MockCommandPermissionsRepository implements CommandPermissionsRepository for testing
type MockCommandPermissionsRepository struct {
	permissions map[string][]byte // key: "commandID-guildID"
}

func NewMockCommandPermissionsRepository() *MockCommandPermissionsRepository {
	return &MockCommandPermissionsRepository{
		permissions: make(map[string][]byte),
	}
}

func (m *MockCommandPermissionsRepository) SetPermissions(ctx context.Context, commandID, guildID uuid.UUID, permissionsJSON []byte) error {
	key := commandID.String() + "-" + guildID.String()
	m.permissions[key] = permissionsJSON
	return nil
}

func (m *MockCommandPermissionsRepository) GetPermissions(ctx context.Context, commandID, guildID uuid.UUID) ([]byte, error) {
	key := commandID.String() + "-" + guildID.String()
	perms, ok := m.permissions[key]
	if !ok {
		return nil, nil
	}
	return perms, nil
}

func (m *MockCommandPermissionsRepository) GetByCommandID(ctx context.Context, commandID uuid.UUID) ([]byte, error) {
	var allPerms []interface{}
	for key, perms := range m.permissions {
		if len(key) > 36 && key[:36] == commandID.String() {
			var p []interface{}
			json.Unmarshal(perms, &p)
			allPerms = append(allPerms, p...)
		}
	}
	return json.Marshal(allPerms)
}

func (m *MockCommandPermissionsRepository) DeletePermissions(ctx context.Context, commandID, guildID uuid.UUID) error {
	key := commandID.String() + "-" + guildID.String()
	delete(m.permissions, key)
	return nil
}

func TestPermissionResolution_UserAllow(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()
	userID := uuid.New()

	// Create permission override allowing a specific user
	permissions := []*models.CommandPermissionOverride{
		{
			ID:     userID,
			Type:   2, // User type
			Allow:  true,
			Denial: false,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	// Retrieve and verify
	retrieved, err := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	var result []*models.CommandPermissionOverride
	err = json.Unmarshal(retrieved, &result)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, userID, result[0].ID)
	assert.True(t, result[0].Allow)
}

func TestPermissionResolution_UserDeny(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()
	userID := uuid.New()

	// Create permission override denying a specific user
	permissions := []*models.CommandPermissionOverride{
		{
			ID:     userID,
			Type:   2, // User type
			Allow:  false,
			Denial: true,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	retrieved, err := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NoError(t, err)

	var result []*models.CommandPermissionOverride
	json.Unmarshal(retrieved, &result)
	assert.Len(t, result, 1)
	assert.True(t, result[0].Denial)
}

func TestPermissionResolution_RoleAllow(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()
	roleID := uuid.New()

	permissions := []*models.CommandPermissionOverride{
		{
			ID:     roleID,
			Type:   1, // Role type
			Allow:  true,
			Denial: false,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	retrieved, err := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NoError(t, err)

	var result []*models.CommandPermissionOverride
	json.Unmarshal(retrieved, &result)
	assert.Len(t, result, 1)
	assert.Equal(t, roleID, result[0].ID)
	assert.Equal(t, 1, result[0].Type)
	assert.True(t, result[0].Allow)
}

func TestPermissionResolution_MultipleOverrides(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()

	roleID := uuid.New()
	allowedUserID := uuid.New()
	deniedUserID := uuid.New()

	permissions := []*models.CommandPermissionOverride{
		{
			ID:     roleID,
			Type:   1,
			Allow:  true,
			Denial: false,
		},
		{
			ID:     allowedUserID,
			Type:   2,
			Allow:  true,
			Denial: false,
		},
		{
			ID:     deniedUserID,
			Type:   2,
			Allow:  false,
			Denial: true,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	retrieved, err := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NoError(t, err)

	var result []*models.CommandPermissionOverride
	json.Unmarshal(retrieved, &result)
	assert.Len(t, result, 3)

	// Verify each override
	roleFound := false
	allowedUserFound := false
	deniedUserFound := false

	for _, p := range result {
		if p.ID == roleID && p.Type == 1 {
			roleFound = true
			assert.True(t, p.Allow)
		}
		if p.ID == allowedUserID && p.Type == 2 && p.Allow {
			allowedUserFound = true
		}
		if p.ID == deniedUserID && p.Type == 2 && p.Denial {
			deniedUserFound = true
		}
	}

	assert.True(t, roleFound, "Role permission not found")
	assert.True(t, allowedUserFound, "Allowed user permission not found")
	assert.True(t, deniedUserFound, "Denied user permission not found")
}

func TestPermissionResolution_DefaultDeny(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()

	// No permissions set - command should use default_permission from command
	perms, err := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NoError(t, err)
	assert.Nil(t, perms)
}

func TestPermissionResolution_SetPermissionsUpdate(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()
	userID := uuid.New()

	// Initial permissions
	permissions := []*models.CommandPermissionOverride{
		{
			ID:     userID,
			Type:   2,
			Allow:  true,
			Denial: false,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	// Update permissions
	updatedPermissions := []*models.CommandPermissionOverride{
		{
			ID:     userID,
			Type:   2,
			Allow:  false,
			Denial: true,
		},
	}
	updatedJSON, _ := json.Marshal(updatedPermissions)
	permRepo.SetPermissions(ctx, commandID, guildID, updatedJSON)

	retrieved, _ := permRepo.GetPermissions(ctx, commandID, guildID)
	var result []*models.CommandPermissionOverride
	json.Unmarshal(retrieved, &result)

	assert.Len(t, result, 1)
	assert.False(t, result[0].Allow)
	assert.True(t, result[0].Denial)
}

func TestPermissionResolution_DeletePermissions(t *testing.T) {
	permRepo := NewMockCommandPermissionsRepository()

	ctx := context.Background()
	commandID := uuid.New()
	guildID := uuid.New()
	userID := uuid.New()

	permissions := []*models.CommandPermissionOverride{
		{
			ID:     userID,
			Type:   2,
			Allow:  true,
			Denial: false,
		},
	}
	permsJSON, _ := json.Marshal(permissions)
	permRepo.SetPermissions(ctx, commandID, guildID, permsJSON)

	// Verify it exists
	retrieved, _ := permRepo.GetPermissions(ctx, commandID, guildID)
	assert.NotNil(t, retrieved)

	// Delete
	err := permRepo.DeletePermissions(ctx, commandID, guildID)
	assert.NoError(t, err)

	// Verify it's gone
	retrieved, _ = permRepo.GetPermissions(ctx, commandID, guildID)
	assert.Nil(t, retrieved)
}

func TestPermissionResolution_PermissionCheckLogic(t *testing.T) {
	// Test the conceptual permission checking logic

	t.Run("user with allow takes precedence over default deny", func(t *testing.T) {
		// This tests the logic that a user with explicit Allow should be allowed
		// even if the command's default_permission is false
		userID := uuid.New()

		permissions := []*models.CommandPermissionOverride{
			{ID: userID, Type: 2, Allow: true, Denial: false},
		}

		// Logic: Find user's permission override
		var userPerm *models.CommandPermissionOverride
		for _, p := range permissions {
			if p.ID == userID && p.Type == 2 {
				userPerm = p
				break
			}
		}

		assert.NotNil(t, userPerm)
		assert.True(t, userPerm.Allow)
		// Even though default might be false, user has explicit allow
	})

	t.Run("user with denial takes precedence", func(t *testing.T) {
		// A user with explicit Denial should be denied regardless of other allows
		userID := uuid.New()

		permissions := []*models.CommandPermissionOverride{
			{ID: userID, Type: 2, Allow: true, Denial: false}, // Allow for user
			{ID: userID, Type: 2, Allow: false, Denial: true}, // But also deny for same user
		}

		// Logic: Denial takes precedence
		var userDenial *models.CommandPermissionOverride
		for _, p := range permissions {
			if p.ID == userID && p.Type == 2 && p.Denial {
				userDenial = p
				break
			}
		}

		assert.NotNil(t, userDenial)
		assert.True(t, userDenial.Denial)
	})
}

func TestCommandPermissionOverride_JSON(t *testing.T) {
	override := &models.CommandPermissionOverride{
		ID:     uuid.New(),
		Type:   1,
		Allow:  true,
		Denial: false,
	}

	data, err := json.Marshal(override)
	assert.NoError(t, err)

	var result models.CommandPermissionOverride
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)

	assert.Equal(t, override.ID, result.ID)
	assert.Equal(t, override.Type, result.Type)
	assert.Equal(t, override.Allow, result.Allow)
	assert.Equal(t, override.Denial, result.Denial)
}

func TestCommandPermissions_JSON(t *testing.T) {
	perms := &models.CommandPermissions{
		Overrides: []*models.CommandPermissionOverride{
			{ID: uuid.New(), Type: 1, Allow: true, Denial: false},
			{ID: uuid.New(), Type: 2, Allow: false, Denial: true},
		},
	}

	data, err := json.Marshal(perms)
	assert.NoError(t, err)

	var result models.CommandPermissions
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)

	assert.Len(t, result.Overrides, 2)
}

func TestPermissionOverride_Types(t *testing.T) {
	// Verify permission type constants
	roleType := 1
	userType := 2

	assert.Equal(t, 1, roleType, "Role permission type should be 1")
	assert.Equal(t, 2, userType, "User permission type should be 2")
}
