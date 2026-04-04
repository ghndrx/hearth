package models

import (
	"testing"
)

func TestGetActionCategory(t *testing.T) {
	tests := []struct {
		actionType string
		expected   int
	}{
		// Member actions
		{AuditLogMemberJoin, 10},
		{AuditLogMemberLeave, 10},
		{AuditLogMemberKick, 10},
		{AuditLogMemberBan, 10},
		{AuditLogMemberUnban, 10},
		{AuditLogMemberUpdate, 10},
		{AuditLogMemberRoleUpdate, 10},
		{AuditLogMemberNicknameUpdate, 10},
		{AuditLogMemberTimeout, 10},
		{AuditLogMemberTimeoutRemove, 10},
		{AuditLogMemberVoiceMove, 10},
		{AuditLogMemberVoiceKick, 10},
		{AuditLogMemberDisconnect, 10},
		{AuditLogMemberAdd, 10},
		{AuditLogMemberRemove, 10},
		{AuditLogMemberVerify, 10},
		{AuditLogMemberLinkAccount, 10},

		// Channel actions
		{AuditLogChannelCreate, 20},
		{AuditLogChannelUpdate, 20},
		{AuditLogChannelDelete, 20},
		{AuditLogChannelPinsUpdate, 20},
		{AuditLogChannelPermissionUpdate, 20},
		{AuditLogChannelOverrideCreate, 20},
		{AuditLogChannelOverrideUpdate, 20},
		{AuditLogChannelOverrideDelete, 20},
		{AuditLogChannelFollowNews, 20},
		{AuditLogChannelGroupJoin, 20},
		{AuditLogChannelGroupLeave, 20},

		// Server actions
		{AuditLogServerUpdate, 30},
		{AuditLogServerNameUpdate, 30},
		{AuditLogServerIconUpdate, 30},
		{AuditLogServerBannerUpdate, 30},
		{AuditLogServerSplashUpdate, 30},
		{AuditLogServerDescriptionUpdate, 30},
		{AuditLogServerRegionUpdate, 30},
		{AuditLogServerVerificationUpdate, 30},
		{AuditLogServerAFKChannelUpdate, 30},
		{AuditLogServerAFKTimeoutUpdate, 30},
		{AuditLogServerDefaultChannelUpdate, 30},
		{AuditLogServerWidgetUpdate, 30},
		{AuditLogServerModerationUpdate, 30},
		{AuditLogServerMFALevelUpdate, 30},
		{AuditLogServerExplicitFilterUpdate, 30},
		{AuditLogServerVanityURLUpdate, 30},
		{AuditLogServerWelcomeScreenUpdate, 30},
		{AuditLogServerNSFWUpdate, 30},

		// Message actions
		{AuditLogMessageDelete, 40},
		{AuditLogMessageBulkDelete, 40},
		{AuditLogMessagePin, 40},
		{AuditLogMessageUnpin, 40},
		{AuditLogMessageReactionAdd, 40},
		{AuditLogMessageReactionRemove, 40},
		{AuditLogMessageReactionRemoveAll, 40},
		{AuditLogMessageEdit, 40},
		{AuditLogMessageAck, 40},
		{AuditLogMessagePublish, 40},

		// Thread actions
		{AuditLogThreadCreate, 45},
		{AuditLogThreadUpdate, 45},
		{AuditLogThreadDelete, 45},
		{AuditLogThreadMemberUpdate, 45},

		// Role actions
		{AuditLogRoleCreate, 50},
		{AuditLogRoleUpdate, 50},
		{AuditLogRoleDelete, 50},
		{AuditLogRolePermissionUpdate, 50},

		// Integration actions
		{AuditLogWebhookCreate, 60},
		{AuditLogWebhookUpdate, 60},
		{AuditLogWebhookDelete, 60},
		{AuditLogWebhookChannelMove, 60},
		{AuditLogEmojiCreate, 60},
		{AuditLogEmojiUpdate, 60},
		{AuditLogEmojiDelete, 60},
		{AuditLogStickerCreate, 60},
		{AuditLogStickerUpdate, 60},
		{AuditLogStickerDelete, 60},
		{AuditLogInviteCreate, 60},
		{AuditLogInviteUpdate, 60},
		{AuditLogInviteDelete, 60},
		{AuditLogSlashCommandCreate, 60},
		{AuditLogSlashCommandUpdate, 60},
		{AuditLogSlashCommandDelete, 60},
		{AuditLogSoundboardSoundCreate, 60},
		{AuditLogSoundboardSoundUpdate, 60},
		{AuditLogSoundboardSoundDelete, 60},

		// Application/Bot actions
		{AuditLogBotAdd, 65},
		{AuditLogApplicationUpdate, 65},
		{AuditLogApplicationCommandPermissionUpdate, 65},

		// Voice actions
		{AuditLogVoiceChannelJoin, 70},
		{AuditLogVoiceChannelLeave, 70},
		{AuditLogVoiceChannelMove, 70},
		{AuditLogVoiceChannelKick, 70},
		{AuditLogVoiceChannelMute, 70},
		{AuditLogVoiceChannelDeafen, 70},
		{AuditLogVoiceChannelSelfMute, 70},
		{AuditLogVoiceChannelSelfDeafen, 70},
		{AuditLogVoiceStreamStart, 70},
		{AuditLogVoiceStreamStop, 70},
		{AuditLogVoiceVideoStart, 70},
		{AuditLogVoiceVideoStop, 70},
		{AuditLogStageInstanceCreate, 70},
		{AuditLogStageInstanceUpdate, 70},
		{AuditLogStageInstanceDelete, 70},

		// Auto-mod actions
		{AuditLogAutoModFlag, 80},
		{AuditLogAutoModBlock, 80},
		{AuditLogAutoModWarn, 80},
		{AuditLogAutoModTimeout, 80},
		{AuditLogAutoModKick, 80},
		{AuditLogAutoModBan, 80},
		{AuditLogAutoModMessageDelete, 80},
		{AuditLogAutoModSpanBlock, 80},
		{AuditLogAutoModMentionSpam, 80},
		{AuditLogAutoModWordFilter, 80},
		{AuditLogAutoModLinkFilter, 80},
		{AuditLogAutoModAttachmentBlock, 80},

		// Unknown
		{"UNKNOWN_ACTION", 0},
		{"", 0},
	}

	for _, tc := range tests {
		t.Run(tc.actionType, func(t *testing.T) {
			result := GetActionCategory(tc.actionType)
			if result != tc.expected {
				t.Errorf("GetActionCategory(%q) = %d; want %d", tc.actionType, result, tc.expected)
			}
		})
	}
}

func TestGetAllAuditLogActionTypes(t *testing.T) {
	actions := GetAllAuditLogActionTypes()
	if len(actions) == 0 {
		t.Fatal("GetAllAuditLogActionTypes() returned empty slice")
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, action := range actions {
		if seen[action] {
			t.Errorf("Duplicate action type: %q", action)
		}
		seen[action] = true
	}

	// Verify key known actions are present
	keyActions := []string{
		AuditLogMemberJoin,
		AuditLogChannelCreate,
		AuditLogMessageDelete,
		AuditLogRoleUpdate,
		AuditLogWebhookCreate,
		AuditLogBotAdd,
	}
	for _, action := range keyActions {
		found := false
		for _, a := range actions {
			if a == action {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected action %q not found in GetAllAuditLogActionTypes()", action)
		}
	}
}

func TestGetAuditLogCategories(t *testing.T) {
	categories := GetAuditLogCategories()
	if len(categories) == 0 {
		t.Fatal("GetAuditLogCategories() returned empty slice")
	}

	// Verify all categories have required fields
	for i, cat := range categories {
		if cat.Name == "" {
			t.Errorf("Category[%d] has empty Name", i)
		}
		if cat.Description == "" {
			t.Errorf("Category[%d] has empty Description", i)
		}
		if cat.Category == 0 && cat.Name != "" {
			// Category 0 is only valid for unknown/default cases
			// But our categories should have proper category numbers
		}
	}

	// Verify we have expected categories
	categoryNames := make(map[string]bool)
	for _, cat := range categories {
		categoryNames[cat.Name] = true
	}

	expected := []string{"Member", "Channel", "Server", "Message", "Thread", "Role", "Integration", "Application", "Voice", "AutoMod"}
	for _, name := range expected {
		if !categoryNames[name] {
			t.Errorf("Expected category %q not found", name)
		}
	}
}

func TestAuditLogFilterParamsNormalize(t *testing.T) {
	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		expectedLimit  int
		expectedOffset int
	}{
		{"zero values", 0, 0, 50, 0},
		{"negative limit", -1, 0, 50, 0},
		{"negative offset", 10, -5, 10, 0},
		{"limit over 100", 0, 0, 50, 0},
		{"limit 200 capped to 100", 200, 0, 100, 0},
		{"valid values", 25, 10, 25, 10},
		{"limit exactly 100", 100, 0, 100, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &AuditLogFilterParams{Limit: tc.inputLimit, Offset: tc.inputOffset}
			p.Normalize()
			if p.Limit != tc.expectedLimit {
				t.Errorf("Limit = %d; want %d", p.Limit, tc.expectedLimit)
			}
			if p.Offset != tc.expectedOffset {
				t.Errorf("Offset = %d; want %d", p.Offset, tc.expectedOffset)
			}
		})
	}
}
