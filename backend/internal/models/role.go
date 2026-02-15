package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a server role with permissions
type Role struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	Name         string    `json:"name" db:"name"`
	Color        int       `json:"color" db:"color"` // RGB integer
	Permissions  int64     `json:"permissions" db:"permissions"`
	Position     int       `json:"position" db:"position"`
	Hoist        bool      `json:"hoist" db:"hoist"`             // Show separately in member list
	Managed      bool      `json:"managed" db:"managed"`         // Managed by integration
	Mentionable  bool      `json:"mentionable" db:"mentionable"`
	IsDefault    bool      `json:"is_default" db:"is_default"`   // @everyone role
	IconURL      *string   `json:"icon_url,omitempty" db:"icon_url"`
	UnicodeEmoji *string   `json:"unicode_emoji,omitempty" db:"unicode_emoji"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Permission bits
const (
	// General
	PermViewChannels       int64 = 1 << 0
	PermManageChannels     int64 = 1 << 1
	PermManageRoles        int64 = 1 << 2
	PermManageEmoji        int64 = 1 << 3
	PermViewAuditLog       int64 = 1 << 4
	PermManageWebhooks     int64 = 1 << 5
	PermManageServer       int64 = 1 << 6

	// Membership
	PermCreateInvite       int64 = 1 << 10
	PermChangeNickname     int64 = 1 << 11
	PermManageNicknames    int64 = 1 << 12
	PermKickMembers        int64 = 1 << 13
	PermBanMembers         int64 = 1 << 14
	PermTimeoutMembers     int64 = 1 << 15

	// Text
	PermSendMessages       int64 = 1 << 20
	PermSendMessagesInThreads int64 = 1 << 21
	PermCreatePublicThreads int64 = 1 << 22
	PermCreatePrivateThreads int64 = 1 << 23
	PermSendTTS            int64 = 1 << 24
	PermManageMessages     int64 = 1 << 25
	PermManageThreads      int64 = 1 << 26
	PermEmbedLinks         int64 = 1 << 27
	PermAttachFiles        int64 = 1 << 28
	PermReadMessageHistory int64 = 1 << 29
	PermMentionEveryone    int64 = 1 << 30
	PermUseExternalEmoji   int64 = 1 << 31
	PermUseExternalStickers int64 = 1 << 32
	PermAddReactions       int64 = 1 << 33
	PermUseSlashCommands   int64 = 1 << 34

	// Voice
	PermConnect            int64 = 1 << 40
	PermSpeak              int64 = 1 << 41
	PermVideo              int64 = 1 << 42
	PermUseVoiceActivity   int64 = 1 << 43
	PermPrioritySpeaker    int64 = 1 << 44
	PermMuteMembers        int64 = 1 << 45
	PermDeafenMembers      int64 = 1 << 46
	PermMoveMembers        int64 = 1 << 47
	PermUseSoundboard      int64 = 1 << 48

	// Admin (bit 62 is max safe for int64)
	PermAdministrator      int64 = 1 << 62
)

// PermissionAll is all permissions combined (except Administrator)
const PermissionAll int64 = PermViewChannels | PermManageChannels | PermManageRoles |
	PermManageEmoji | PermViewAuditLog | PermManageWebhooks | PermManageServer |
	PermCreateInvite | PermChangeNickname | PermManageNicknames |
	PermKickMembers | PermBanMembers | PermTimeoutMembers |
	PermSendMessages | PermSendMessagesInThreads | PermCreatePublicThreads |
	PermCreatePrivateThreads | PermSendTTS | PermManageMessages |
	PermManageThreads | PermEmbedLinks | PermAttachFiles |
	PermReadMessageHistory | PermMentionEveryone | PermUseExternalEmoji |
	PermUseExternalStickers | PermAddReactions | PermUseSlashCommands |
	PermConnect | PermSpeak | PermVideo | PermUseVoiceActivity |
	PermPrioritySpeaker | PermMuteMembers | PermDeafenMembers |
	PermMoveMembers | PermUseSoundboard

// DefaultPermissions for @everyone role
const DefaultPermissions int64 = PermViewChannels | PermCreateInvite |
	PermChangeNickname | PermSendMessages | PermSendMessagesInThreads |
	PermCreatePublicThreads | PermEmbedLinks | PermAttachFiles |
	PermReadMessageHistory | PermAddReactions | PermUseExternalEmoji |
	PermUseSlashCommands | PermConnect | PermSpeak | PermVideo |
	PermUseVoiceActivity

// HasPermission checks if a permission set includes a specific permission
func HasPermission(perms, perm int64) bool {
	if perms&PermAdministrator != 0 {
		return true
	}
	return perms&perm != 0
}

// CreateRoleRequest is the input for creating a role
type CreateRoleRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Color       *int    `json:"color,omitempty"`
	Permissions *int64  `json:"permissions,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
}

// UpdateRoleRequest is the input for updating a role
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Color       *int    `json:"color,omitempty"`
	Permissions *int64  `json:"permissions,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
	Position    *int    `json:"position,omitempty"`
}

// MemberRole links members to roles
type MemberRole struct {
	MemberUserID   uuid.UUID `json:"member_user_id" db:"member_user_id"`
	MemberServerID uuid.UUID `json:"member_server_id" db:"member_server_id"`
	RoleID         uuid.UUID `json:"role_id" db:"role_id"`
}

// CalculatePermissions computes effective permissions for a member
func CalculatePermissions(member *Member, roles []*Role, server *Server, channel *Channel, overrides []PermissionOverride) int64 {
	// Server owner has all permissions
	if member.UserID == server.OwnerID {
		return PermissionAll | PermAdministrator
	}

	// Start with @everyone role (assumed to be first/lowest)
	var permissions int64 = 0
	
	// Find @everyone role and add its permissions
	for _, role := range roles {
		// @everyone role has same ID as server
		if role.ID == server.ID {
			permissions = role.Permissions
			break
		}
	}

	// Add permissions from member's roles
	for _, role := range roles {
		for _, memberRoleID := range member.Roles {
			if role.ID == memberRoleID {
				permissions |= role.Permissions
			}
		}
	}

	// Administrator bypasses everything
	if permissions&PermAdministrator != 0 {
		return PermissionAll | PermAdministrator
	}

	// Apply channel overrides if channel is provided
	if channel != nil && len(overrides) > 0 {
		// First, apply @everyone overrides
		for _, override := range overrides {
			if override.TargetType == "role" && override.TargetID == server.ID {
				permissions &= ^override.Deny
				permissions |= override.Allow
			}
		}

		// Then apply role overrides (in position order)
		for _, role := range roles {
			for _, memberRoleID := range member.Roles {
				if role.ID == memberRoleID {
					for _, override := range overrides {
						if override.TargetType == "role" && override.TargetID == role.ID {
							permissions &= ^override.Deny
							permissions |= override.Allow
						}
					}
				}
			}
		}

		// Finally, apply user-specific overrides (highest priority)
		for _, override := range overrides {
			if override.TargetType == "user" && override.TargetID == member.UserID {
				permissions &= ^override.Deny
				permissions |= override.Allow
			}
		}
	}

	return permissions
}
