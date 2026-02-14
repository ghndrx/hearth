package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
)

// MockRoleService implements RoleService interface for testing
type MockRoleService struct {
	memberRoles     map[string][]*models.Role
	permissions     map[string]int64
	getRolesErr     error
	computePermsErr error
}

func (m *MockRoleService) GetMemberRoles(ctx interface{}, serverID, userID uuid.UUID) ([]*models.Role, error) {
	if m.getRolesErr != nil {
		return nil, m.getRolesErr
	}
	key := serverID.String() + ":" + userID.String()
	return m.memberRoles[key], nil
}

func (m *MockRoleService) ComputeMemberPermissions(ctx interface{}, serverID, userID uuid.UUID) (int64, error) {
	if m.computePermsErr != nil {
		return 0, m.computePermsErr
	}
	key := serverID.String() + ":" + userID.String()
	return m.permissions[key], nil
}

// MockServerRepository implements ServerRepository interface for testing
type MockServerRepository struct {
	members map[string]*models.Member
	servers map[uuid.UUID]*models.Server
	getErr  error
}

func (m *MockServerRepository) GetMember(ctx interface{}, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := serverID.String() + ":" + userID.String()
	return m.members[key], nil
}

func (m *MockServerRepository) GetByID(ctx interface{}, id uuid.UUID) (*models.Server, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.servers[id], nil
}

func setupTest() (*RoleMiddleware, *MockRoleService, *MockServerRepository) {
	mockRoleService := &MockRoleService{
		memberRoles: make(map[string][]*models.Role),
		permissions: make(map[string]int64),
	}
	mockServerRepo := &MockServerRepository{
		members: make(map[string]*models.Member),
		servers: make(map[uuid.UUID]*models.Server),
	}
	rm := NewRoleMiddleware(mockRoleService, mockServerRepo)
	return rm, mockRoleService, mockServerRepo
}

func TestNewRoleMiddleware(t *testing.T) {
	mockRoleService := &MockRoleService{}
	mockServerRepo := &MockServerRepository{}
	rm := NewRoleMiddleware(mockRoleService, mockServerRepo)

	if rm == nil {
		t.Fatal("NewRoleMiddleware returned nil")
	}
	if rm.roleService != mockRoleService {
		t.Error("roleService not set correctly")
	}
	if rm.serverRepo != mockServerRepo {
		t.Error("serverRepo not set correctly")
	}
}

func TestRequireRole(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()
	requiredRoleID := uuid.New()
	otherRoleID := uuid.New()

	// Setup mock data
	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(), // Different user
	}
	mockRoleService.memberRoles[serverID.String()+":"+userID.String()] = []*models.Role{
		{ID: requiredRoleID, Name: "Admin"},
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		serverID       string
		path           string
		requiredRoles  []uuid.UUID
		setupRoles     []*models.Role
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "unauthorized - no user ID",
			userID:         uuid.Nil,
			serverID:       serverID.String(),
			path:           "/test",
			requiredRoles:  []uuid.UUID{requiredRoleID},
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "unauthorized",
		},
		{
			name:           "missing server ID",
			userID:         userID,
			serverID:       "",
			path:           "/test",
			requiredRoles:  []uuid.UUID{requiredRoleID},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "server ID required",
		},
		{
			name:           "invalid server ID",
			userID:         userID,
			serverID:       "invalid-uuid",
			requiredRoles:  []uuid.UUID{requiredRoleID},
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid server ID",
		},
		{
			name:           "not a member",
			userID:         uuid.New(), // Different user
			serverID:       serverID.String(),
			path:           "/test",
			requiredRoles:  []uuid.UUID{requiredRoleID},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not a member",
		},
		{
			name:           "missing required role",
			userID:         userID,
			serverID:       serverID.String(),
			path:           "/test",
			requiredRoles:  []uuid.UUID{otherRoleID},
			setupRoles:     []*models.Role{{ID: requiredRoleID, Name: "Admin"}},
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "missing required role",
		},
		{
			name:           "has required role",
			userID:         userID,
			serverID:       serverID.String(),
			path:           "/test",
			requiredRoles:  []uuid.UUID{requiredRoleID},
			setupRoles:     []*models.Role{{ID: requiredRoleID, Name: "Admin"}},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "has one of multiple required roles",
			userID:         userID,
			serverID:       serverID.String(),
			path:           "/test",
			requiredRoles:  []uuid.UUID{otherRoleID, requiredRoleID},
			setupRoles:     []*models.Role{{ID: requiredRoleID, Name: "Admin"}},
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update mock roles for this test if specified
			if tt.setupRoles != nil && tt.userID != uuid.Nil {
				mockRoleService.memberRoles[serverID.String()+":"+tt.userID.String()] = tt.setupRoles
			}

			app := fiber.New()

			// Handle edge cases differently
			if tt.userID == uuid.Nil {
				// No userID set - test unauthorized case
				app.Get("/servers/:serverID/test", rm.RequireRole(tt.requiredRoles...), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			} else if tt.serverID == "" {
				// Empty serverID - test missing server ID case using query param
				app.Get("/test", func(c *fiber.Ctx) error {
					c.Locals("userID", tt.userID)
					c.Params("serverID", "")
					return rm.RequireRole(tt.requiredRoles...)(c)
				}, func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				req := httptest.NewRequest("GET", "/test", nil)
				resp, err := app.Test(req)
				if err != nil {
					t.Fatalf("app.Test failed: %v", err)
				}
				if resp.StatusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				}
				return
			} else {
				app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
					c.Locals("userID", tt.userID)
					return c.Next()
				}, rm.RequireRole(tt.requiredRoles...), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			}

			req := httptest.NewRequest("GET", "/servers/"+tt.serverID+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()
	ownerID := uuid.New()

	// Setup mock data
	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		serverID       string
		permission     int64
		permissions    int64
		isOwner        bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "unauthorized - no user ID",
			userID:         uuid.Nil,
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			expectedStatus: fiber.StatusUnauthorized,
			expectedError:  "unauthorized",
		},
		{
			name:           "missing server ID",
			userID:         userID,
			serverID:       "",
			permission:     models.PermManageChannels,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "server ID required",
		},
		{
			name:           "invalid server ID",
			userID:         userID,
			serverID:       "invalid-uuid",
			permission:     models.PermManageChannels,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "invalid server ID",
		},
		{
			name:           "not a member",
			userID:         uuid.New(),
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "not a member",
		},
		{
			name:           "missing permission",
			userID:         userID,
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			permissions:    models.PermSendMessages, // Different permission
			expectedStatus: fiber.StatusForbidden,
			expectedError:  "missing required permission",
		},
		{
			name:           "has permission",
			userID:         userID,
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			permissions:    models.PermManageChannels,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "server owner bypass",
			userID:         ownerID,
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			permissions:    0, // No explicit permissions needed
			isOwner:        true,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "administrator permission grants all",
			userID:         userID,
			serverID:       serverID.String(),
			permission:     models.PermManageChannels,
			permissions:    models.PermAdministrator,
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup permissions
			testUserID := tt.userID
			if tt.isOwner {
				testUserID = ownerID
				mockServerRepo.members[serverID.String()+":"+ownerID.String()] = &models.Member{
					UserID:   ownerID,
					ServerID: serverID,
				}
			}
			if testUserID != uuid.Nil {
				mockRoleService.permissions[serverID.String()+":"+testUserID.String()] = tt.permissions
			}

			app := fiber.New()

			// Handle edge cases differently
			if tt.userID == uuid.Nil {
				// No userID set - test unauthorized case
				app.Get("/servers/:serverID/test", rm.RequirePermission(tt.permission), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			} else if tt.serverID == "" {
				// Empty serverID - test missing server ID case using query param
				app.Get("/test", func(c *fiber.Ctx) error {
					c.Locals("userID", tt.userID)
					c.Params("serverID", "")
					return rm.RequirePermission(tt.permission)(c)
				}, func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				req := httptest.NewRequest("GET", "/test", nil)
				resp, err := app.Test(req)
				if err != nil {
					t.Fatalf("app.Test failed: %v", err)
				}
				if resp.StatusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				}
				return
			} else {
				app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
					c.Locals("userID", tt.userID)
					return c.Next()
				}, rm.RequirePermission(tt.permission), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			}

			req := httptest.NewRequest("GET", "/servers/"+tt.serverID+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRequirePermissionOrRole(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()
	adminRoleID := uuid.New()
	modRoleID := uuid.New()

	// Setup mock data
	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	tests := []struct {
		name           string
		permission     int64
		roleIDs        []uuid.UUID
		userRoles      []*models.Role
		permissions    int64
		expectedStatus int
	}{
		{
			name:           "has required permission",
			permission:     models.PermManageChannels,
			roleIDs:        []uuid.UUID{modRoleID},
			userRoles:      []*models.Role{},
			permissions:    models.PermManageChannels,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "has required role",
			permission:     models.PermManageChannels,
			roleIDs:        []uuid.UUID{adminRoleID, modRoleID},
			userRoles:      []*models.Role{{ID: adminRoleID, Name: "Admin"}},
			permissions:    0,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "has neither permission nor role",
			permission:     models.PermManageChannels,
			roleIDs:        []uuid.UUID{adminRoleID},
			userRoles:      []*models.Role{{ID: modRoleID, Name: "Moderator"}},
			permissions:    models.PermSendMessages,
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "no roles to check, missing permission",
			permission:     models.PermManageChannels,
			roleIDs:        []uuid.UUID{},
			userRoles:      []*models.Role{},
			permissions:    models.PermSendMessages,
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoleService.memberRoles[serverID.String()+":"+userID.String()] = tt.userRoles
			mockRoleService.permissions[serverID.String()+":"+userID.String()] = tt.permissions

			app := fiber.New()
			app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return c.Next()
			}, rm.RequirePermissionOrRole(tt.permission, tt.roleIDs...), func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()

	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	tests := []struct {
		name           string
		permissions    int64
		expectedStatus int
	}{
		{
			name:           "has administrator permission",
			permissions:    models.PermAdministrator,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "missing administrator permission",
			permissions:    models.PermManageChannels,
			expectedStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoleService.permissions[serverID.String()+":"+userID.String()] = tt.permissions

			app := fiber.New()
			app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return c.Next()
			}, rm.RequireAdmin(), func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRequireServerOwner(t *testing.T) {
	rm, _, mockServerRepo := setupTest()

	ownerID := uuid.New()
	otherUserID := uuid.New()
	serverID := uuid.New()

	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		expectedStatus int
	}{
		{
			name:           "is owner",
			userID:         ownerID,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "not owner",
			userID:         otherUserID,
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "no user ID",
			userID:         uuid.Nil,
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			if tt.userID == uuid.Nil {
				// No userID set - test unauthorized case
				app.Get("/servers/:serverID/test", rm.RequireServerOwner(), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			} else {
				app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
					c.Locals("userID", tt.userID)
					return c.Next()
				}, rm.RequireServerOwner(), func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
			}

			req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRoleMiddlewareErrors(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()

	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}

	tests := []struct {
		name           string
		setupMock      func()
		middleware     fiber.Handler
		expectedStatus int
	}{
		{
			name: "GetMemberRoles error",
			setupMock: func() {
				mockRoleService.getRolesErr = errors.New("database error")
				mockServerRepo.servers[serverID] = &models.Server{
					ID:      serverID,
					OwnerID: uuid.New(),
				}
			},
			middleware:     rm.RequireRole(uuid.New()),
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name: "ComputeMemberPermissions error",
			setupMock: func() {
				mockRoleService.getRolesErr = nil
				mockRoleService.computePermsErr = errors.New("database error")
				mockServerRepo.servers[serverID] = &models.Server{
					ID:      serverID,
					OwnerID: uuid.New(),
				}
			},
			middleware:     rm.RequirePermission(models.PermManageChannels),
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name: "server not found",
			setupMock: func() {
				mockRoleService.computePermsErr = nil
				mockServerRepo.servers = make(map[uuid.UUID]*models.Server) // Clear servers
			},
			middleware:     rm.RequirePermission(models.PermManageChannels),
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name: "GetByID error",
			setupMock: func() {
				// Clear the getErr for GetMember but set it for GetByID specifically
				// This test simulates GetByID failing after GetMember succeeds
				mockServerRepo.servers[serverID] = nil // This will cause GetByID to return nil server
			},
			middleware:     rm.RequirePermission(models.PermManageChannels),
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRoleService.getRolesErr = nil
			mockRoleService.computePermsErr = nil
			mockServerRepo.getErr = nil
			mockServerRepo.servers = make(map[uuid.UUID]*models.Server)
			mockServerRepo.servers[serverID] = &models.Server{
				ID:      serverID,
				OwnerID: uuid.New(),
			}

			tt.setupMock()

			app := fiber.New()
			app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
				c.Locals("userID", userID)
				return c.Next()
			}, tt.middleware, func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRequirePermissionMultiplePermissions(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()

	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	// Test with combined permissions
	mockRoleService.permissions[serverID.String()+":"+userID.String()] =
		models.PermManageChannels | models.PermManageRoles | models.PermKickMembers

	app := fiber.New()
	app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	}, rm.RequirePermission(models.PermManageRoles), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestRequireRoleMultipleRoles(t *testing.T) {
	rm, mockRoleService, mockServerRepo := setupTest()

	userID := uuid.New()
	serverID := uuid.New()
	role1ID := uuid.New()
	role2ID := uuid.New()
	role3ID := uuid.New()

	mockServerRepo.members[serverID.String()+":"+userID.String()] = &models.Member{
		UserID:   userID,
		ServerID: serverID,
	}
	mockServerRepo.servers[serverID] = &models.Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	// User has role1 and role3
	mockRoleService.memberRoles[serverID.String()+":"+userID.String()] = []*models.Role{
		{ID: role1ID, Name: "Role1"},
		{ID: role3ID, Name: "Role3"},
	}

	// Should pass with role1 (user has it)
	app := fiber.New()
	app.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	}, rm.RequireRole(role1ID, role2ID), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// Should fail when only requiring role2 (user doesn't have it)
	app2 := fiber.New()
	app2.Get("/servers/:serverID/test", func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	}, rm.RequireRole(role2ID), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	resp2, err := app2.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status %d, got %d", fiber.StatusForbidden, resp2.StatusCode)
	}
}
