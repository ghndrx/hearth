package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// RoleService handles role-related business logic
type RoleService struct {
	roleRepo    RoleRepository
	serverRepo  ServerRepository
	cache       CacheService
	eventBus    EventBus
	permService *PermissionService
}

// NewRoleService creates a new role service
func NewRoleService(
	roleRepo RoleRepository,
	serverRepo ServerRepository,
	cache CacheService,
	eventBus EventBus,
	permService *PermissionService,
) *RoleService {
	return &RoleService{
		roleRepo:    roleRepo,
		serverRepo:  serverRepo,
		cache:       cache,
		eventBus:    eventBus,
		permService: permService,
	}
}

// checkManageRolesPermission verifies the requester has MANAGE_ROLES permission
func (s *RoleService) checkManageRolesPermission(ctx context.Context, serverID, requesterID uuid.UUID) error {
	// Use permService if available for consistent permission checking
	if s.permService != nil {
		return s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageRoles)
	}
	// Fallback to local permission check
	perms, err := s.ComputeMemberPermissions(ctx, serverID, requesterID)
	if err != nil {
		return err
	}
	if perms&models.PermManageRoles == 0 && perms&models.PermAdministrator == 0 {
		return ErrMissingManageRoles
	}
	return nil
}

// getHighestRolePosition returns the highest (lowest number = highest priority) role position for a member
// Returns -1 for server owner (highest possible)
func (s *RoleService) getHighestRolePosition(ctx context.Context, serverID, userID uuid.UUID) (int, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server != nil && server.OwnerID == userID {
		return -1, nil // Owner is above all roles
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}
	if len(roles) == 0 {
		return 999999, nil // No roles = lowest position
	}

	highest := roles[0].Position
	for _, role := range roles[1:] {
		if role.Position < highest {
			highest = role.Position
		}
	}
	return highest, nil
}

// checkRoleHierarchy ensures requester's highest role is above the target role
func (s *RoleService) checkRoleHierarchy(ctx context.Context, serverID, requesterID uuid.UUID, targetRole *models.Role) error {
	// Use permService if available
	if s.permService != nil {
		canManage, err := s.permService.CanManageRole(ctx, serverID, requesterID, targetRole)
		if err != nil {
			return err
		}
		if !canManage {
			return ErrRoleHierarchy
		}
		return nil
	}
	// Fallback to local hierarchy check
	requesterPos, err := s.getHighestRolePosition(ctx, serverID, requesterID)
	if err != nil {
		return err
	}
	// Lower position number = higher in hierarchy
	if requesterPos >= targetRole.Position {
		return ErrRoleHierarchy
	}
	return nil
}

// CreateRole creates a new role in a server
func (s *RoleService) CreateRole(
	ctx context.Context,
	serverID uuid.UUID,
	creatorID uuid.UUID,
	name string,
	color int,
	permissions int64,
) (*models.Role, error) {
	// Verify creator is a member with permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, creatorID); err != nil {
		return nil, err
	}

	// Get existing roles to determine position
	roles, err := s.roleRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	role := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        name,
		Color:       color,
		Position:    len(roles), // Add at end
		Permissions: permissions,
		CreatedAt:   time.Now(),
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	s.eventBus.Publish("role.created", &RoleCreatedEvent{
		Role:     role,
		ServerID: serverID,
	})

	return role, nil
}

// UpdateRole updates a role
func (s *RoleService) UpdateRole(
	ctx context.Context,
	roleID uuid.UUID,
	requesterID uuid.UUID,
	updates *models.RoleUpdate,
) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, role.ServerID, requesterID); err != nil {
		return nil, err
	}
	if err := s.checkRoleHierarchy(ctx, role.ServerID, requesterID, role); err != nil {
		return nil, err
	}

	// Apply updates
	if updates.Name != nil {
		role.Name = *updates.Name
	}
	if updates.Color != nil {
		role.Color = *updates.Color
	}
	if updates.Permissions != nil {
		role.Permissions = *updates.Permissions
	}
	if updates.Hoist != nil {
		role.Hoist = *updates.Hoist
	}
	if updates.Mentionable != nil {
		role.Mentionable = *updates.Mentionable
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	s.eventBus.Publish("role.updated", &RoleUpdatedEvent{
		Role: role,
	})

	return role, nil
}

// DeleteRole deletes a role
func (s *RoleService) DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	// Can't delete @everyone role
	if role.IsDefault {
		return ErrCannotDeleteDefault
	}

	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, role.ServerID, requesterID); err != nil {
		return err
	}
	if err := s.checkRoleHierarchy(ctx, role.ServerID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.Delete(ctx, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("role.deleted", &RoleDeletedEvent{
		RoleID:   roleID,
		ServerID: role.ServerID,
	})

	return nil
}

// GetServerRoles gets all roles in a server
func (s *RoleService) GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error) {
	// Verify requester is a member
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	return s.roleRepo.GetByServerID(ctx, serverID)
}

// GetRole retrieves a single role by ID
func (s *RoleService) GetRole(ctx context.Context, roleID, requesterID uuid.UUID) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	// Verify requester is a member of the server this role belongs to
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	return role, nil
}

// UpdateRolePositions updates the positions of multiple roles
func (s *RoleService) UpdateRolePositions(
	ctx context.Context,
	serverID uuid.UUID,
	requesterID uuid.UUID,
	positions map[uuid.UUID]int,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	return s.roleRepo.UpdatePositions(ctx, serverID, positions)
}

// AddRoleToMember assigns a role to a member
func (s *RoleService) AddRoleToMember(
	ctx context.Context,
	serverID, userID, roleID uuid.UUID,
	requesterID uuid.UUID,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	// Verify target is a member
	target, err := s.serverRepo.GetMember(ctx, serverID, userID)
	if err != nil || target == nil {
		return ErrNotServerMember
	}

	// Verify role exists
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return ErrRoleNotFound
	}
	if role.ServerID != serverID {
		return ErrRoleNotFound
	}

	// Verify requester can manage this role (hierarchy check)
	if err := s.checkRoleHierarchy(ctx, serverID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.AddRoleToMember(ctx, serverID, userID, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("member.role_added", &MemberRoleAddedEvent{
		ServerID: serverID,
		UserID:   userID,
		RoleID:   roleID,
	})

	return nil
}

// RemoveRoleFromMember removes a role from a member
func (s *RoleService) RemoveRoleFromMember(
	ctx context.Context,
	serverID, userID, roleID uuid.UUID,
	requesterID uuid.UUID,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	// Fetch role to check hierarchy
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil || role.ServerID != serverID {
		return ErrRoleNotFound
	}
	if err := s.checkRoleHierarchy(ctx, serverID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.RemoveRoleFromMember(ctx, serverID, userID, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("member.role_removed", &MemberRoleRemovedEvent{
		ServerID: serverID,
		UserID:   userID,
		RoleID:   roleID,
	})

	return nil
}

// GetMemberRoles gets all roles for a member
func (s *RoleService) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	return s.roleRepo.GetMemberRoles(ctx, serverID, userID)
}

// ComputeMemberPermissions computes effective permissions for a member
func (s *RoleService) ComputeMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	// Get server to check ownership
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server == nil {
		return 0, ErrServerNotFound
	}

	// Owner has all permissions
	if server.OwnerID == userID {
		return models.PermissionAll, nil
	}

	// Get member's roles
	roles, err := s.roleRepo.GetMemberRoles(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}

	// Combine permissions from all roles
	var permissions int64 = 0
	for _, role := range roles {
		permissions |= role.Permissions
	}

	// Administrator grants all permissions
	if permissions&models.PermAdministrator != 0 {
		return models.PermissionAll, nil
	}

	return permissions, nil
}

// Events

type RoleCreatedEvent struct {
	Role     *models.Role
	ServerID uuid.UUID
}

type RoleUpdatedEvent struct {
	Role *models.Role
}

type RoleDeletedEvent struct {
	RoleID   uuid.UUID
	ServerID uuid.UUID
}

type MemberRoleAddedEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	RoleID   uuid.UUID
}

type MemberRoleRemovedEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	RoleID   uuid.UUID
}
