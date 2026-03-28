package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemberDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		nickname *string
		user     *User
		expected string
	}{
		{
			name:     "returns nickname when set",
			nickname: strPtr("NickName"),
			user:     &User{Username: "username"},
			expected: "NickName",
		},
		{
			name:     "returns username when no nickname",
			nickname: nil,
			user:     &User{Username: "username"},
			expected: "username",
		},
		{
			name:     "returns username when nickname is empty",
			nickname: strPtr(""),
			user:     &User{Username: "username"},
			expected: "username",
		},
		{
			name:     "returns empty when no nickname and no user",
			nickname: nil,
			user:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member := &Member{
				Nickname: tt.nickname,
			}
			result := member.DisplayName(tt.user)
			if result != tt.expected {
				t.Errorf("DisplayName() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestInviteIsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{
			name:      "not expired when expiresAt is nil",
			expiresAt: nil,
			expected:  false,
		},
		{
			name:      "expired when in the past",
			expiresAt: &past,
			expected:  true,
		},
		{
			name:      "not expired when in the future",
			expiresAt: &future,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invite := &Invite{
				ExpiresAt: tt.expiresAt,
			}
			result := invite.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInviteIsMaxUsesReached(t *testing.T) {
	tests := []struct {
		name     string
		maxUses  int
		uses     int
		expected bool
	}{
		{
			name:     "unlimited uses (0)",
			maxUses:  0,
			uses:     100,
			expected: false,
		},
		{
			name:     "not reached",
			maxUses:  10,
			uses:     5,
			expected: false,
		},
		{
			name:     "reached exactly",
			maxUses:  10,
			uses:     10,
			expected: true,
		},
		{
			name:     "exceeded",
			maxUses:  10,
			uses:     15,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invite := &Invite{
				MaxUses: tt.maxUses,
				Uses:    tt.uses,
			}
			result := invite.IsMaxUsesReached()
			if result != tt.expected {
				t.Errorf("IsMaxUsesReached() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInviteIsValid(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		maxUses   int
		uses      int
		expected  bool
	}{
		{
			name:      "valid: no expiry, no max uses",
			expiresAt: nil,
			maxUses:   0,
			uses:      0,
			expected:  true,
		},
		{
			name:      "valid: future expiry, under max",
			expiresAt: &future,
			maxUses:   10,
			uses:      5,
			expected:  true,
		},
		{
			name:      "invalid: expired",
			expiresAt: &past,
			maxUses:   10,
			uses:      5,
			expected:  false,
		},
		{
			name:      "invalid: max uses reached",
			expiresAt: &future,
			maxUses:   10,
			uses:      10,
			expected:  false,
		},
		{
			name:      "invalid: both expired and max uses reached",
			expiresAt: &past,
			maxUses:   10,
			uses:      10,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invite := &Invite{
				ExpiresAt: tt.expiresAt,
				MaxUses:   tt.maxUses,
				Uses:      tt.uses,
			}
			result := invite.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVerificationLevelConstants(t *testing.T) {
	if VerificationNone != 0 {
		t.Errorf("expected VerificationNone=0, got %d", VerificationNone)
	}
	if VerificationLow != 1 {
		t.Errorf("expected VerificationLow=1, got %d", VerificationLow)
	}
	if VerificationMedium != 2 {
		t.Errorf("expected VerificationMedium=2, got %d", VerificationMedium)
	}
	if VerificationHigh != 3 {
		t.Errorf("expected VerificationHigh=3, got %d", VerificationHigh)
	}
	if VerificationVeryHigh != 4 {
		t.Errorf("expected VerificationVeryHigh=4, got %d", VerificationVeryHigh)
	}
}

func TestExplicitContentFilterConstants(t *testing.T) {
	if ExplicitFilterDisabled != 0 {
		t.Errorf("expected ExplicitFilterDisabled=0, got %d", ExplicitFilterDisabled)
	}
	if ExplicitFilterNoRole != 1 {
		t.Errorf("expected ExplicitFilterNoRole=1, got %d", ExplicitFilterNoRole)
	}
	if ExplicitFilterAllMembers != 2 {
		t.Errorf("expected ExplicitFilterAllMembers=2, got %d", ExplicitFilterAllMembers)
	}
}

func TestDefaultNotificationConstants(t *testing.T) {
	if NotifyAllMessages != 0 {
		t.Errorf("expected NotifyAllMessages=0, got %d", NotifyAllMessages)
	}
	if NotifyMentionsOnly != 1 {
		t.Errorf("expected NotifyMentionsOnly=1, got %d", NotifyMentionsOnly)
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		perms    int64
		perm     int64
		expected bool
	}{
		{
			name:     "has specific permission",
			perms:    PermViewChannels | PermSendMessages,
			perm:     PermViewChannels,
			expected: true,
		},
		{
			name:     "does not have permission",
			perms:    PermViewChannels,
			perm:     PermManageChannels,
			expected: false,
		},
		{
			name:     "administrator has all permissions",
			perms:    PermAdministrator,
			perm:     PermManageServer,
			expected: true,
		},
		{
			name:     "administrator bypasses missing perms",
			perms:    PermAdministrator,
			perm:     PermBanMembers,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermission(tt.perms, tt.perm)
			if result != tt.expected {
				t.Errorf("HasPermission(%d, %d) = %v, expected %v", tt.perms, tt.perm, result, tt.expected)
			}
		})
	}
}

func TestPermissionConstants(t *testing.T) {
	// Verify some key permission bits
	if PermViewChannels != 1<<0 {
		t.Errorf("expected PermViewChannels = 1<<0")
	}
	if PermManageChannels != 1<<1 {
		t.Errorf("expected PermManageChannels = 1<<1")
	}
	if PermAdministrator != 1<<62 {
		t.Errorf("expected PermAdministrator = 1<<62")
	}
}

func TestCalculatePermissions(t *testing.T) {
	serverID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

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
		Permissions: PermManageMessages | PermEmbedLinks,
	}

	roles := []*Role{everyoneRole, customRole}

	t.Run("server owner gets all permissions", func(t *testing.T) {
		member := &Member{
			UserID: ownerID,
		}
		perms := CalculatePermissions(member, roles, server, nil, nil)
		if perms&PermAdministrator == 0 {
			t.Error("owner should have Administrator")
		}
		if perms != (PermissionAll | PermAdministrator) {
			t.Error("owner should have all permissions")
		}
	})

	t.Run("member gets everyone role permissions", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{},
		}
		perms := CalculatePermissions(member, roles, server, nil, nil)
		if perms&PermViewChannels == 0 {
			t.Error("member should have ViewChannels from @everyone")
		}
		if perms&PermSendMessages == 0 {
			t.Error("member should have SendMessages from @everyone")
		}
	})

	t.Run("member gets additional role permissions", func(t *testing.T) {
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{roleID},
		}
		perms := CalculatePermissions(member, roles, server, nil, nil)
		if perms&PermManageMessages == 0 {
			t.Error("member should have ManageMessages from custom role")
		}
		if perms&PermEmbedLinks == 0 {
			t.Error("member should have EmbedLinks from custom role")
		}
	})

	t.Run("admin role grants all permissions", func(t *testing.T) {
		adminRoleID := uuid.New()
		adminRole := &Role{
			ID:          adminRoleID,
			Permissions: PermAdministrator,
		}
		member := &Member{
			UserID: userID,
			Roles:  []uuid.UUID{adminRoleID},
		}
		perms := CalculatePermissions(member, append(roles, adminRole), server, nil, nil)
		if perms != (PermissionAll | PermAdministrator) {
			t.Error("admin role should grant all permissions")
		}
	})
}

func TestDefaultPermissions(t *testing.T) {
	// Default permissions should include common user permissions
	if DefaultPermissions&PermViewChannels == 0 {
		t.Error("DefaultPermissions should include ViewChannels")
	}
	if DefaultPermissions&PermSendMessages == 0 {
		t.Error("DefaultPermissions should include SendMessages")
	}
	if DefaultPermissions&PermConnect == 0 {
		t.Error("DefaultPermissions should include Connect")
	}
	if DefaultPermissions&PermSpeak == 0 {
		t.Error("DefaultPermissions should include Speak")
	}

	// Should not include admin permissions
	if DefaultPermissions&PermAdministrator != 0 {
		t.Error("DefaultPermissions should not include Administrator")
	}
	if DefaultPermissions&PermManageServer != 0 {
		t.Error("DefaultPermissions should not include ManageServer")
	}
	if DefaultPermissions&PermBanMembers != 0 {
		t.Error("DefaultPermissions should not include BanMembers")
	}
}
