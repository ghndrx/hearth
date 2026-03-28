package services

import (
	"context"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// PermissionService handles permission checking across the application.
// It centralizes permission logic to ensure consistent authorization.
type PermissionService struct {
	serverRepo  ServerRepository
	roleRepo    RoleRepository
	channelRepo ChannelRepository
	cache       CacheService
}

// NewPermissionService creates a new permission service
func NewPermissionService(
	serverRepo ServerRepository,
	roleRepo RoleRepository,
	channelRepo ChannelRepository,
	cache CacheService,
) *PermissionService {
	return &PermissionService{
		serverRepo:  serverRepo,
		roleRepo:    roleRepo,
		channelRepo: channelRepo,
		cache:       cache,
	}
}

// GetMemberPermissions computes effective permissions for a member in a server.
// Returns PermissionAll | PermAdministrator for server owners.
func (s *PermissionService) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
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
		return models.PermissionAll | models.PermAdministrator, nil
	}

	// Get permissions from member's roles
	if s.roleRepo != nil {
		perms, err := s.roleRepo.GetMemberPermissions(ctx, serverID, userID)
		if err != nil {
			return 0, err
		}

		// Get default (@everyone) role permissions
		defaultRole, err := s.roleRepo.GetDefaultRole(ctx, serverID)
		if err == nil && defaultRole != nil {
			perms |= defaultRole.Permissions
		}

		// Administrator grants all permissions
		if perms&models.PermAdministrator != 0 {
			return models.PermissionAll | models.PermAdministrator, nil
		}

		return perms, nil
	}

	// Fallback: if no roleRepo, assume default permissions
	return models.DefaultPermissions, nil
}

// HasPermission checks if a user has a specific permission in a server.
func (s *PermissionService) HasPermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) (bool, error) {
	perms, err := s.GetMemberPermissions(ctx, serverID, userID)
	if err != nil {
		return false, err
	}
	return models.HasPermission(perms, permission), nil
}

// HasChannelPermission checks if a user has a specific permission in a channel.
// For DM channels, this always returns true (permissions don't apply).
// This method considers channel-level permission overrides.
func (s *PermissionService) HasChannelPermission(ctx context.Context, channel *models.Channel, userID uuid.UUID, permission int64) (bool, error) {
	// DM channels don't have permission restrictions (beyond being a participant)
	if channel.ServerID == nil {
		return true, nil
	}

	// Get effective permissions for this channel
	perms, err := s.GetChannelPermissions(ctx, channel, userID)
	if err != nil {
		return false, err
	}

	return models.HasPermission(perms, permission), nil
}

// GetChannelPermissions computes effective permissions for a user in a specific channel.
// This takes into account role permissions and channel-specific permission overrides.
func (s *PermissionService) GetChannelPermissions(ctx context.Context, channel *models.Channel, userID uuid.UUID) (int64, error) {
	// DM channels don't have permissions
	if channel.ServerID == nil {
		return models.PermissionAll, nil
	}

	serverID := *channel.ServerID

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
		return models.PermissionAll | models.PermAdministrator, nil
	}

	// Get member to check roles
	member, err := s.serverRepo.GetMember(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}
	if member == nil {
		return 0, ErrNotServerMember
	}

	// Get all server roles
	roles, err := s.roleRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return 0, err
	}

	// Get channel permission overrides
	var overrides []models.PermissionOverride
	if s.channelRepo != nil {
		overrides, err = s.channelRepo.GetPermissionOverrides(ctx, channel.ID)
		if err != nil {
			// Non-fatal: continue without overrides
			overrides = nil
		}
	}

	// Use the model's CalculatePermissions function which handles:
	// 1. Server owner bypass
	// 2. Role permission aggregation
	// 3. Administrator bypass
	// 4. Channel-level overrides (everyone, roles, user-specific)
	return models.CalculatePermissions(member, roles, server, channel, overrides), nil
}

// RequirePermission checks if a user has a permission and returns an error if not.
func (s *PermissionService) RequirePermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) error {
	has, err := s.HasPermission(ctx, serverID, userID, permission)
	if err != nil {
		return err
	}
	if !has {
		return permissionError(permission)
	}
	return nil
}

// RequireChannelPermission checks if a user has a channel permission and returns an error if not.
func (s *PermissionService) RequireChannelPermission(ctx context.Context, channel *models.Channel, userID uuid.UUID, permission int64) error {
	has, err := s.HasChannelPermission(ctx, channel, userID, permission)
	if err != nil {
		return err
	}
	if !has {
		return permissionError(permission)
	}
	return nil
}

// IsServerOwner checks if a user is the owner of a server.
func (s *PermissionService) IsServerOwner(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return false, err
	}
	if server == nil {
		return false, ErrServerNotFound
	}
	return server.OwnerID == userID, nil
}

// IsServerOwnerOrAdmin checks if a user is the owner or has admin permissions.
func (s *PermissionService) IsServerOwnerOrAdmin(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return false, err
	}
	if server == nil {
		return false, ErrServerNotFound
	}

	// Owner always has admin
	if server.OwnerID == userID {
		return true, nil
	}

	// Check for admin permission
	return s.HasPermission(ctx, serverID, userID, models.PermAdministrator)
}

// GetHighestRolePosition returns the highest role position for a member.
// Used for role hierarchy checks.
func (s *PermissionService) GetHighestRolePosition(ctx context.Context, serverID, userID uuid.UUID) (int, error) {
	// Check if user is owner (owner is always highest)
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server == nil {
		return 0, ErrServerNotFound
	}
	if server.OwnerID == userID {
		return int(^uint(0) >> 1), nil // Max int value for owner
	}

	// Get member's roles
	if s.roleRepo == nil {
		return 0, nil
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}

	highest := 0
	for _, role := range roles {
		if role.Position > highest {
			highest = role.Position
		}
	}

	return highest, nil
}

// CanManageRole checks if a user can manage a specific role based on hierarchy.
// A user can only manage roles below their highest role position.
func (s *PermissionService) CanManageRole(ctx context.Context, serverID, userID uuid.UUID, role *models.Role) (bool, error) {
	// First check if user has MANAGE_ROLES permission
	has, err := s.HasPermission(ctx, serverID, userID, models.PermManageRoles)
	if err != nil {
		return false, err
	}
	if !has {
		return false, nil
	}

	// Server owner can manage all roles
	isOwner, err := s.IsServerOwner(ctx, serverID, userID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// Check hierarchy - user's highest role must be above the target role
	userHighest, err := s.GetHighestRolePosition(ctx, serverID, userID)
	if err != nil {
		return false, err
	}

	return userHighest > role.Position, nil
}

// CanManageMember checks if a user can manage another member based on role hierarchy.
// A user can only manage members whose highest role is below their own.
func (s *PermissionService) CanManageMember(ctx context.Context, serverID, actorID, targetID uuid.UUID) (bool, error) {
	// Can't manage yourself with this check (that's a separate concern)
	if actorID == targetID {
		return false, nil
	}

	// Server owner can manage anyone
	isOwner, err := s.IsServerOwner(ctx, serverID, actorID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// Check if target is the owner (can't manage owner)
	targetIsOwner, err := s.IsServerOwner(ctx, serverID, targetID)
	if err != nil {
		return false, err
	}
	if targetIsOwner {
		return false, nil
	}

	// Check hierarchy
	actorHighest, err := s.GetHighestRolePosition(ctx, serverID, actorID)
	if err != nil {
		return false, err
	}

	targetHighest, err := s.GetHighestRolePosition(ctx, serverID, targetID)
	if err != nil {
		return false, err
	}

	return actorHighest > targetHighest, nil
}

// permissionError returns the appropriate error for a missing permission.
func permissionError(permission int64) error {
	switch permission {
	case models.PermSendMessages:
		return ErrMissingSendMessages
	case models.PermReadMessageHistory:
		return ErrMissingReadMessages
	case models.PermManageMessages:
		return ErrMissingManageMessages
	case models.PermAddReactions:
		return ErrMissingAddReactions
	case models.PermManageRoles:
		return ErrMissingManageRoles
	case models.PermManageChannels:
		return ErrMissingManageChannels
	case models.PermKickMembers:
		return ErrMissingKickMembers
	case models.PermBanMembers:
		return ErrMissingBanMembers
	case models.PermCreateInvite:
		return ErrMissingCreateInvite
	case models.PermManageServer:
		return ErrMissingManageServer
	case models.PermManageWebhooks:
		return ErrMissingManageWebhooks
	case models.PermManageThreads:
		return ErrMissingManageThreads
	case models.PermAdministrator:
		return ErrMissingAdministrator
	case models.PermMoveMembers:
		return ErrMissingMoveMembers
	case models.PermMuteMembers:
		return ErrMissingMuteMembers
	case models.PermManageEmoji:
		return ErrMissingManageEmojis
	case models.PermViewChannels:
		return ErrMissingViewChannels
	default:
		return ErrMissingPermission
	}
}
