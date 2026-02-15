package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPermissionBits(t *testing.T) {
	// Test general permissions
	if PermViewChannels != 1<<0 {
		t.Error("PermViewChannels should be 1<<0")
	}
	if PermManageChannels != 1<<1 {
		t.Error("PermManageChannels should be 1<<1")
	}
	if PermManageRoles != 1<<2 {
		t.Error("PermManageRoles should be 1<<2")
	}
	if PermManageEmoji != 1<<3 {
		t.Error("PermManageEmoji should be 1<<3")
	}
	if PermViewAuditLog != 1<<4 {
		t.Error("PermViewAuditLog should be 1<<4")
	}
	if PermManageWebhooks != 1<<5 {
		t.Error("PermManageWebhooks should be 1<<5")
	}
	if PermManageServer != 1<<6 {
		t.Error("PermManageServer should be 1<<6")
	}

	// Test membership permissions
	if PermCreateInvite != 1<<10 {
		t.Error("PermCreateInvite should be 1<<10")
	}
	if PermKickMembers != 1<<13 {
		t.Error("PermKickMembers should be 1<<13")
	}
	if PermBanMembers != 1<<14 {
		t.Error("PermBanMembers should be 1<<14")
	}

	// Test text permissions
	if PermSendMessages != 1<<20 {
		t.Error("PermSendMessages should be 1<<20")
	}
	if PermManageMessages != 1<<25 {
		t.Error("PermManageMessages should be 1<<25")
	}
	if PermAddReactions != 1<<33 {
		t.Error("PermAddReactions should be 1<<33")
	}

	// Test voice permissions
	if PermConnect != 1<<40 {
		t.Error("PermConnect should be 1<<40")
	}
	if PermSpeak != 1<<41 {
		t.Error("PermSpeak should be 1<<41")
	}

	// Test admin
	if PermAdministrator != 1<<62 {
		t.Error("PermAdministrator should be 1<<62")
	}
}

func TestHasPermissionBasic(t *testing.T) {
	// Basic permission check
	perms := PermViewChannels | PermSendMessages
	if !HasPermission(perms, PermViewChannels) {
		t.Error("should have PermViewChannels")
	}
	if !HasPermission(perms, PermSendMessages) {
		t.Error("should have PermSendMessages")
	}
	if HasPermission(perms, PermManageServer) {
		t.Error("should not have PermManageServer")
	}

	// Administrator bypasses everything
	adminPerms := PermAdministrator
	if !HasPermission(adminPerms, PermManageServer) {
		t.Error("administrator should have all permissions")
	}
	if !HasPermission(adminPerms, PermBanMembers) {
		t.Error("administrator should have ban permission")
	}
}

func TestHasPermissionWithZeroPerms(t *testing.T) {
	var perms int64 = 0
	if HasPermission(perms, PermViewChannels) {
		t.Error("zero perms should not have any permission")
	}
}

func TestPermissionAll(t *testing.T) {
	// PermissionAll should include basic permissions but not Administrator
	if PermissionAll&PermViewChannels == 0 {
		t.Error("PermissionAll should include PermViewChannels")
	}
	if PermissionAll&PermSendMessages == 0 {
		t.Error("PermissionAll should include PermSendMessages")
	}
	if PermissionAll&PermAdministrator != 0 {
		t.Error("PermissionAll should not include PermAdministrator")
	}
}

func TestDefaultPermissionsBasic(t *testing.T) {
	// DefaultPermissions should include basic user permissions
	if DefaultPermissions&PermViewChannels == 0 {
		t.Error("DefaultPermissions should include PermViewChannels")
	}
	if DefaultPermissions&PermSendMessages == 0 {
		t.Error("DefaultPermissions should include PermSendMessages")
	}
	if DefaultPermissions&PermConnect == 0 {
		t.Error("DefaultPermissions should include PermConnect")
	}
	if DefaultPermissions&PermSpeak == 0 {
		t.Error("DefaultPermissions should include PermSpeak")
	}
	// Should not include dangerous perms
	if DefaultPermissions&PermBanMembers != 0 {
		t.Error("DefaultPermissions should not include PermBanMembers")
	}
	if DefaultPermissions&PermManageServer != 0 {
		t.Error("DefaultPermissions should not include PermManageServer")
	}
}

func TestRoleStruct(t *testing.T) {
	iconURL := "https://example.com/icon.png"
	emoji := "🎮"

	role := Role{
		ID:           uuid.New(),
		ServerID:     uuid.New(),
		Name:         "Moderators",
		Color:        0xFF5733,
		Permissions:  PermKickMembers | PermBanMembers | PermManageMessages,
		Position:     5,
		Hoist:        true,
		Managed:      false,
		Mentionable:  true,
		IsDefault:    false,
		IconURL:      &iconURL,
		UnicodeEmoji: &emoji,
		CreatedAt:    time.Now(),
	}

	if role.Name != "Moderators" {
		t.Errorf("expected name 'Moderators', got %s", role.Name)
	}
	if role.Color != 0xFF5733 {
		t.Errorf("expected color 0xFF5733, got %d", role.Color)
	}
	if !role.Hoist {
		t.Error("expected role to be hoisted")
	}
	if !role.Mentionable {
		t.Error("expected role to be mentionable")
	}
	if !HasPermission(role.Permissions, PermKickMembers) {
		t.Error("expected role to have kick permission")
	}
}

func TestCreateRoleRequest(t *testing.T) {
	color := 0x00FF00
	perms := PermSendMessages | PermAddReactions
	hoist := true
	mentionable := false

	req := CreateRoleRequest{
		Name:        "New Role",
		Color:       &color,
		Permissions: &perms,
		Hoist:       &hoist,
		Mentionable: &mentionable,
	}

	if req.Name != "New Role" {
		t.Errorf("expected name 'New Role', got %s", req.Name)
	}
	if *req.Color != 0x00FF00 {
		t.Errorf("expected color 0x00FF00, got %d", *req.Color)
	}
}

func TestUpdateRoleRequest(t *testing.T) {
	name := "Updated Role"
	color := 0x0000FF
	perms := PermManageChannels
	hoist := false
	mentionable := true
	position := 10

	req := UpdateRoleRequest{
		Name:        &name,
		Color:       &color,
		Permissions: &perms,
		Hoist:       &hoist,
		Mentionable: &mentionable,
		Position:    &position,
	}

	if *req.Name != "Updated Role" {
		t.Errorf("expected name 'Updated Role', got %s", *req.Name)
	}
	if *req.Position != 10 {
		t.Errorf("expected position 10, got %d", *req.Position)
	}
}

func TestMemberRole(t *testing.T) {
	mr := MemberRole{
		MemberUserID:   uuid.New(),
		MemberServerID: uuid.New(),
		RoleID:         uuid.New(),
	}

	if mr.MemberUserID == uuid.Nil {
		t.Error("expected non-nil member user ID")
	}
	if mr.MemberServerID == uuid.Nil {
		t.Error("expected non-nil member server ID")
	}
	if mr.RoleID == uuid.Nil {
		t.Error("expected non-nil role ID")
	}
}

func TestCalculatePermissionsServerOwner(t *testing.T) {
	serverID := uuid.New()
	ownerID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: ownerID,
	}

	member := &Member{
		UserID:   ownerID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	perms := CalculatePermissions(member, []*Role{}, server, nil, nil)

	// Owner should have all permissions
	if perms&PermAdministrator == 0 {
		t.Error("owner should have administrator permission")
	}
	if perms&PermManageServer == 0 {
		t.Error("owner should have manage server permission")
	}
}

func TestCalculatePermissionsBasicMember(t *testing.T) {
	serverID := uuid.New()
	everyoneRoleID := serverID // @everyone role has same ID as server

	server := &Server{
		ID:      serverID,
		OwnerID: uuid.New(), // Different owner
	}

	everyoneRole := &Role{
		ID:          everyoneRoleID,
		ServerID:    serverID,
		Name:        "@everyone",
		Permissions: DefaultPermissions,
		IsDefault:   true,
	}

	member := &Member{
		UserID:   uuid.New(),
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	perms := CalculatePermissions(member, []*Role{everyoneRole}, server, nil, nil)

	// Should have default permissions
	if perms&PermViewChannels == 0 {
		t.Error("member should have view channels permission")
	}
	if perms&PermSendMessages == 0 {
		t.Error("member should have send messages permission")
	}
	// Should not have admin permissions
	if perms&PermAdministrator != 0 {
		t.Error("basic member should not have administrator")
	}
	if perms&PermBanMembers != 0 {
		t.Error("basic member should not have ban members")
	}
}

func TestCalculatePermissionsWithRoles(t *testing.T) {
	serverID := uuid.New()
	modRoleID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	everyoneRole := &Role{
		ID:          serverID,
		ServerID:    serverID,
		Permissions: PermViewChannels | PermSendMessages,
	}

	modRole := &Role{
		ID:          modRoleID,
		ServerID:    serverID,
		Name:        "Moderator",
		Permissions: PermKickMembers | PermBanMembers | PermManageMessages,
	}

	member := &Member{
		UserID:   uuid.New(),
		ServerID: serverID,
		Roles:    []uuid.UUID{modRoleID},
	}

	perms := CalculatePermissions(member, []*Role{everyoneRole, modRole}, server, nil, nil)

	// Should have base + mod permissions
	if perms&PermViewChannels == 0 {
		t.Error("should have view channels from @everyone")
	}
	if perms&PermKickMembers == 0 {
		t.Error("should have kick members from mod role")
	}
	if perms&PermBanMembers == 0 {
		t.Error("should have ban members from mod role")
	}
}

func TestCalculatePermissionsAdminBypassesAll(t *testing.T) {
	serverID := uuid.New()
	adminRoleID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	adminRole := &Role{
		ID:          adminRoleID,
		ServerID:    serverID,
		Name:        "Admin",
		Permissions: PermAdministrator,
	}

	member := &Member{
		UserID:   uuid.New(),
		ServerID: serverID,
		Roles:    []uuid.UUID{adminRoleID},
	}

	perms := CalculatePermissions(member, []*Role{adminRole}, server, nil, nil)

	// Admin should have everything
	if perms&PermAdministrator == 0 {
		t.Error("admin should have administrator permission")
	}
	if perms&PermManageServer == 0 {
		t.Error("admin should have manage server permission")
	}
}

func TestCalculatePermissionsWithChannelOverrides(t *testing.T) {
	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	everyoneRole := &Role{
		ID:          serverID,
		ServerID:    serverID,
		Permissions: PermViewChannels | PermSendMessages,
	}

	channel := &Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Deny send messages for @everyone in this channel
	overrides := []PermissionOverride{
		{
			ChannelID:  channelID,
			TargetType: "role",
			TargetID:   serverID, // @everyone
			Allow:      0,
			Deny:       PermSendMessages,
		},
	}

	member := &Member{
		UserID:   userID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	perms := CalculatePermissions(member, []*Role{everyoneRole}, server, channel, overrides)

	// Should have view but not send
	if perms&PermViewChannels == 0 {
		t.Error("should still have view channels")
	}
	if perms&PermSendMessages != 0 {
		t.Error("should not have send messages (denied by override)")
	}
}

func TestCalculatePermissionsUserOverrideHighestPriority(t *testing.T) {
	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	server := &Server{
		ID:      serverID,
		OwnerID: uuid.New(),
	}

	everyoneRole := &Role{
		ID:          serverID,
		ServerID:    serverID,
		Permissions: PermViewChannels,
	}

	channel := &Channel{
		ID:       channelID,
		ServerID: &serverID,
	}

	// Deny view for @everyone, but allow for specific user
	overrides := []PermissionOverride{
		{
			ChannelID:  channelID,
			TargetType: "role",
			TargetID:   serverID,
			Allow:      0,
			Deny:       PermViewChannels,
		},
		{
			ChannelID:  channelID,
			TargetType: "user",
			TargetID:   userID,
			Allow:      PermViewChannels | PermSendMessages,
			Deny:       0,
		},
	}

	member := &Member{
		UserID:   userID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	perms := CalculatePermissions(member, []*Role{everyoneRole}, server, channel, overrides)

	// User override should grant permissions back
	if perms&PermViewChannels == 0 {
		t.Error("user override should grant view channels")
	}
	if perms&PermSendMessages == 0 {
		t.Error("user override should grant send messages")
	}
}
