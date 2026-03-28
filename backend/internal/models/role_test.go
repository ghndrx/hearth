package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestCalculatePermissionsChannelOverrides(t *testing.T) {
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	channelID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	everyoneRole := &Role{
		ID:          serverID,
		Permissions: PermViewChannels | PermSendMessages,
	}

	customRole := &Role{
		ID:          roleID,
		Permissions: PermEmbedLinks,
	}

	roles := []*Role{everyoneRole, customRole}

	channel := &Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	t.Run("everyone role override denies permission", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{},
		}
		overrides := []PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   serverID, // @everyone
				Deny:       PermSendMessages,
			},
		}
		perms := CalculatePermissions(member, roles, server, channel, overrides)
		if perms&PermSendMessages != 0 {
			t.Error("expected SendMessages to be denied by channel override")
		}
		if perms&PermViewChannels == 0 {
			t.Error("expected ViewChannels to still be allowed")
		}
	})

	t.Run("everyone role override allows permission", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{},
		}
		overrides := []PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   serverID,
				Allow:      PermManageMessages,
			},
		}
		perms := CalculatePermissions(member, roles, server, channel, overrides)
		if perms&PermManageMessages == 0 {
			t.Error("expected ManageMessages to be allowed by override")
		}
	})

	t.Run("role override applies to member with that role", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{roleID},
		}
		overrides := []PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   roleID,
				Allow:      PermManageChannels,
			},
		}
		perms := CalculatePermissions(member, roles, server, channel, overrides)
		if perms&PermManageChannels == 0 {
			t.Error("expected ManageChannels from role override")
		}
	})

	t.Run("user-specific override has highest priority", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{},
		}
		overrides := []PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "role",
				TargetID:   serverID,
				Deny:       PermSendMessages,
			},
			{
				ChannelID:  channelID,
				TargetType: "user",
				TargetID:   userID,
				Allow:      PermSendMessages,
			},
		}
		perms := CalculatePermissions(member, roles, server, channel, overrides)
		if perms&PermSendMessages == 0 {
			t.Error("expected user override to re-allow SendMessages")
		}
	})

	t.Run("user-specific deny override", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{},
		}
		overrides := []PermissionOverride{
			{
				ChannelID:  channelID,
				TargetType: "user",
				TargetID:   userID,
				Deny:       PermViewChannels | PermSendMessages,
			},
		}
		perms := CalculatePermissions(member, roles, server, channel, overrides)
		if perms&PermViewChannels != 0 {
			t.Error("expected ViewChannels to be denied by user override")
		}
		if perms&PermSendMessages != 0 {
			t.Error("expected SendMessages to be denied by user override")
		}
	})

	t.Run("no overrides with channel still works", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{roleID},
		}
		perms := CalculatePermissions(member, roles, server, channel, nil)
		if perms&PermViewChannels == 0 {
			t.Error("expected ViewChannels from everyone role")
		}
		if perms&PermEmbedLinks == 0 {
			t.Error("expected EmbedLinks from custom role")
		}
	})
}

func TestPermissionAllCoverage(t *testing.T) {
	// Verify PermissionAll includes key permissions
	expectedPerms := []int64{
		PermViewChannels, PermManageChannels, PermManageRoles,
		PermSendMessages, PermManageMessages,
		PermConnect, PermSpeak, PermVideo,
		PermKickMembers, PermBanMembers,
		PermManageServer, PermManageWebhooks,
		PermManageEvents, PermUseSoundboard,
	}
	for _, p := range expectedPerms {
		if PermissionAll&p == 0 {
			t.Errorf("PermissionAll missing permission bit %d", p)
		}
	}

	// PermissionAll should NOT include Administrator
	if PermissionAll&PermAdministrator != 0 {
		t.Error("PermissionAll should not include Administrator")
	}
}

func TestHasPermissionZeroPerms(t *testing.T) {
	if HasPermission(0, PermViewChannels) {
		t.Error("zero perms should not have any permission")
	}
}
