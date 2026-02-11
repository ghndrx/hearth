package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrCannotDeleteDefault = errors.New("cannot delete the default role")
	ErrRoleHierarchy     = errors.New("cannot modify role higher than your highest role")
)

// RoleRepository defines role data operations
type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error
	
	// Member role operations
	AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error
	RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error
	GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error)
}

// RoleService handles role-related business logic
type RoleService struct {
	roleRepo   RoleRepository
	serverRepo ServerRepository
	cache      CacheService
	eventBus   EventBus
}

// NewRoleService creates a new role service
func NewRoleService(
	roleRepo RoleRepository,
	serverRepo ServerRepository,
	cache CacheService,
	eventBus EventBus,
) *RoleService {
	return &RoleService{
		roleRepo:   roleRepo,
		serverRepo: serverRepo,
		cache:      cache,
		eventBus:   eventBus,
	}
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
	// TODO: Check MANAGE_ROLES permission

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
	// TODO: Check MANAGE_ROLES permission and role hierarchy

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
	// TODO: Check MANAGE_ROLES permission and role hierarchy

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
	// TODO: Check MANAGE_ROLES permission

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
	// TODO: Check MANAGE_ROLES permission and role hierarchy

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
	// TODO: Check MANAGE_ROLES permission and role hierarchy

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
	if permissions&models.PermissionAdministrator != 0 {
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
