package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuditLogActionTypes(t *testing.T) {
	// Test all audit log action type constants
	expectedActions := map[string]string{
		AuditLogServerUpdate:      "SERVER_UPDATE",
		AuditLogChannelCreate:     "CHANNEL_CREATE",
		AuditLogChannelUpdate:     "CHANNEL_UPDATE",
		AuditLogChannelDelete:     "CHANNEL_DELETE",
		AuditLogMemberKick:        "MEMBER_KICK",
		AuditLogMemberBan:         "MEMBER_BAN",
		AuditLogMemberUnban:       "MEMBER_UNBAN",
		AuditLogMemberUpdate:      "MEMBER_UPDATE",
		AuditLogRoleCreate:        "ROLE_CREATE",
		AuditLogRoleUpdate:        "ROLE_UPDATE",
		AuditLogRoleDelete:        "ROLE_DELETE",
		AuditLogInviteCreate:      "INVITE_CREATE",
		AuditLogInviteDelete:      "INVITE_DELETE",
		AuditLogWebhookCreate:     "WEBHOOK_CREATE",
		AuditLogWebhookUpdate:     "WEBHOOK_UPDATE",
		AuditLogWebhookDelete:     "WEBHOOK_DELETE",
		AuditLogEmojiCreate:       "EMOJI_CREATE",
		AuditLogEmojiUpdate:       "EMOJI_UPDATE",
		AuditLogEmojiDelete:       "EMOJI_DELETE",
		AuditLogMessageDelete:     "MESSAGE_DELETE",
		AuditLogMessageBulkDelete: "MESSAGE_BULK_DELETE",
		AuditLogMessagePin:        "MESSAGE_PIN",
		AuditLogMessageUnpin:      "MESSAGE_UNPIN",
	}

	for action, expected := range expectedActions {
		if action != expected {
			t.Errorf("expected action %s, got %s", expected, action)
		}
	}
}

func TestAuditLogEntry(t *testing.T) {
	targetID := uuid.New()

	entry := AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   uuid.New(),
		UserID:     uuid.New(),
		TargetID:   &targetID,
		ActionType: AuditLogMemberBan,
		Changes: []Change{
			{Key: "banned", OldValue: false, NewValue: true},
		},
		Reason:    "Violation of rules",
		CreatedAt: time.Now(),
		User:      nil,
		Target:    nil,
	}

	if entry.ActionType != AuditLogMemberBan {
		t.Errorf("expected action type %s, got %s", AuditLogMemberBan, entry.ActionType)
	}
	if entry.Reason != "Violation of rules" {
		t.Errorf("expected reason 'Violation of rules', got %s", entry.Reason)
	}
	if len(entry.Changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(entry.Changes))
	}
}

func TestAuditLogEntryWithoutTarget(t *testing.T) {
	entry := AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   uuid.New(),
		UserID:     uuid.New(),
		TargetID:   nil,
		ActionType: AuditLogServerUpdate,
		Changes: []Change{
			{Key: "name", OldValue: "Old Server", NewValue: "New Server"},
			{Key: "icon", OldValue: nil, NewValue: "https://example.com/icon.png"},
		},
		Reason:    "",
		CreatedAt: time.Now(),
	}

	if entry.TargetID != nil {
		t.Error("expected nil target ID")
	}
	if len(entry.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(entry.Changes))
	}
}

func TestChange(t *testing.T) {
	change := Change{
		Key:      "name",
		OldValue: "Old Name",
		NewValue: "New Name",
	}

	if change.Key != "name" {
		t.Errorf("expected key 'name', got %s", change.Key)
	}
	if change.OldValue != "Old Name" {
		t.Errorf("expected old value 'Old Name', got %v", change.OldValue)
	}
	if change.NewValue != "New Name" {
		t.Errorf("expected new value 'New Name', got %v", change.NewValue)
	}
}

func TestChangeWithNilValues(t *testing.T) {
	change := Change{
		Key:      "avatar",
		OldValue: nil,
		NewValue: "https://example.com/avatar.png",
	}

	if change.OldValue != nil {
		t.Error("expected nil old value")
	}
	if change.NewValue == nil {
		t.Error("expected non-nil new value")
	}
}

func TestChangeWithIntValues(t *testing.T) {
	change := Change{
		Key:      "position",
		OldValue: 5,
		NewValue: 10,
	}

	if change.Key != "position" {
		t.Errorf("expected key 'position', got %s", change.Key)
	}
	if change.OldValue.(int) != 5 {
		t.Errorf("expected old value 5, got %v", change.OldValue)
	}
	if change.NewValue.(int) != 10 {
		t.Errorf("expected new value 10, got %v", change.NewValue)
	}
}

func TestChangeWithBoolValues(t *testing.T) {
	change := Change{
		Key:      "nsfw",
		OldValue: false,
		NewValue: true,
	}

	if change.OldValue.(bool) != false {
		t.Errorf("expected old value false, got %v", change.OldValue)
	}
	if change.NewValue.(bool) != true {
		t.Errorf("expected new value true, got %v", change.NewValue)
	}
}

func TestAuditLogEntryWithUser(t *testing.T) {
	user := &User{
		ID:       uuid.New(),
		Username: "admin",
	}

	target := &User{
		ID:       uuid.New(),
		Username: "baduser",
	}

	entry := AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   uuid.New(),
		UserID:     user.ID,
		TargetID:   &target.ID,
		ActionType: AuditLogMemberKick,
		Changes:    []Change{},
		Reason:     "Spamming",
		CreatedAt:  time.Now(),
		User:       user,
		Target:     target,
	}

	if entry.User == nil {
		t.Error("expected non-nil user")
	}
	if entry.User.Username != "admin" {
		t.Errorf("expected user username 'admin', got %s", entry.User.Username)
	}
	if entry.Target == nil {
		t.Error("expected non-nil target")
	}
	if entry.Target.Username != "baduser" {
		t.Errorf("expected target username 'baduser', got %s", entry.Target.Username)
	}
}

func TestMultipleChanges(t *testing.T) {
	changes := []Change{
		{Key: "name", OldValue: "Old", NewValue: "New"},
		{Key: "topic", OldValue: "", NewValue: "New topic"},
		{Key: "nsfw", OldValue: false, NewValue: true},
		{Key: "slowmode", OldValue: 0, NewValue: 5},
	}

	if len(changes) != 4 {
		t.Errorf("expected 4 changes, got %d", len(changes))
	}

	// Check first change
	if changes[0].Key != "name" {
		t.Error("first change should be 'name'")
	}

	// Check nsfw change
	if changes[2].Key != "nsfw" {
		t.Error("third change should be 'nsfw'")
	}
}
