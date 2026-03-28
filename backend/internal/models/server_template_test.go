package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServerTemplateToResponse(t *testing.T) {
	sourceServerID := uuid.New()
	template := &ServerTemplate{
		ID:             uuid.New(),
		Code:           "tmpl-abc123",
		Name:           "Gaming Server",
		Description:    "A template for gaming communities",
		SourceServerID: &sourceServerID,
		CreatorID:      uuid.New(),
		SerializedData: json.RawMessage(`{"channels":[],"roles":[],"settings":{}}`),
		UsageCount:     42,
		IsPublic:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	resp := template.ToResponse()

	if resp.ID != template.ID {
		t.Errorf("expected ID %v, got %v", template.ID, resp.ID)
	}
	if resp.Code != "tmpl-abc123" {
		t.Errorf("expected Code 'tmpl-abc123', got %s", resp.Code)
	}
	if resp.Name != "Gaming Server" {
		t.Errorf("expected Name 'Gaming Server', got %s", resp.Name)
	}
	if resp.Description != "A template for gaming communities" {
		t.Errorf("expected Description set, got %s", resp.Description)
	}
	if resp.SourceServerID == nil || *resp.SourceServerID != sourceServerID {
		t.Error("expected SourceServerID to be set")
	}
	if resp.CreatorID != template.CreatorID {
		t.Error("expected CreatorID to match")
	}
	if resp.UsageCount != 42 {
		t.Errorf("expected UsageCount 42, got %d", resp.UsageCount)
	}
	if !resp.IsPublic {
		t.Error("expected IsPublic true")
	}
}

func TestServerTemplateToResponseNilSourceServer(t *testing.T) {
	template := &ServerTemplate{
		ID:             uuid.New(),
		Code:           "tmpl-xyz",
		Name:           "Minimal",
		SourceServerID: nil,
		CreatorID:      uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	resp := template.ToResponse()

	if resp.SourceServerID != nil {
		t.Error("expected nil SourceServerID")
	}
}

func TestChannelTypeConstants(t *testing.T) {
	types := map[ChannelType]string{
		ChannelTypeText:         "text",
		ChannelTypeVoice:        "voice",
		ChannelTypeCategory:     "category",
		ChannelTypeAnnouncement: "announcement",
		ChannelTypeForum:        "forum",
		ChannelTypeStage:        "stage",
		ChannelTypeDM:           "dm",
		ChannelTypeGroupDM:      "group_dm",
	}

	for ct, expected := range types {
		if string(ct) != expected {
			t.Errorf("expected %s, got %s", expected, ct)
		}
	}
}

func TestVoiceChannelLimits(t *testing.T) {
	if MinVoiceBitrate != 8000 {
		t.Errorf("expected MinVoiceBitrate 8000, got %d", MinVoiceBitrate)
	}
	if MaxVoiceBitrate != 384000 {
		t.Errorf("expected MaxVoiceBitrate 384000, got %d", MaxVoiceBitrate)
	}
	if MaxVoiceUserLimit != 99 {
		t.Errorf("expected MaxVoiceUserLimit 99, got %d", MaxVoiceUserLimit)
	}
	if MaxSlowmodeSeconds != 21600 {
		t.Errorf("expected MaxSlowmodeSeconds 21600, got %d", MaxSlowmodeSeconds)
	}
}

func TestMessageTypeConstants(t *testing.T) {
	if MessageTypeDefault != 0 {
		t.Errorf("expected MessageTypeDefault 0, got %d", MessageTypeDefault)
	}
	if MessageTypeReply != 1 {
		t.Errorf("expected MessageTypeReply 1, got %d", MessageTypeReply)
	}
	if MessageTypePinned != 7 {
		t.Errorf("expected MessageTypePinned 7, got %d", MessageTypePinned)
	}
	if MessageTypeSticker != 11 {
		t.Errorf("expected MessageTypeSticker 11, got %d", MessageTypeSticker)
	}
}

func TestMessageFlagConstants(t *testing.T) {
	if MessageFlagCrossposted != 1<<0 {
		t.Error("MessageFlagCrossposted should be 1<<0")
	}
	if MessageFlagSuppressEmbeds != 1<<2 {
		t.Error("MessageFlagSuppressEmbeds should be 1<<2")
	}
	if MessageFlagEphemeral != 1<<6 {
		t.Error("MessageFlagEphemeral should be 1<<6")
	}
	if MessageFlagHasThread != 1<<5 {
		t.Error("MessageFlagHasThread should be 1<<5")
	}

	// Flags should be unique
	flags := []int{
		MessageFlagCrossposted, MessageFlagIsCrosspost, MessageFlagSuppressEmbeds,
		MessageFlagSourceMsgDeleted, MessageFlagUrgent, MessageFlagHasThread,
		MessageFlagEphemeral, MessageFlagLoading, MessageFlagFailedToMention,
	}
	seen := make(map[int]bool)
	for _, f := range flags {
		if seen[f] {
			t.Errorf("duplicate message flag value: %d", f)
		}
		seen[f] = true
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	types := []NotificationType{
		NotificationTypeMention, NotificationTypeReply,
		NotificationTypeDirectMessage, NotificationTypeFriendRequest,
		NotificationTypeFriendAccept, NotificationTypeServerInvite,
		NotificationTypeServerJoin, NotificationTypeReaction,
		NotificationTypeSystem,
	}

	seen := make(map[NotificationType]bool)
	for _, nt := range types {
		if nt == "" {
			t.Error("NotificationType should not be empty")
		}
		if seen[nt] {
			t.Errorf("duplicate NotificationType: %s", nt)
		}
		seen[nt] = true
	}
}

func TestFriendshipStatusConstants(t *testing.T) {
	if FriendshipStatusPending != "pending" {
		t.Errorf("expected 'pending', got %s", FriendshipStatusPending)
	}
	if FriendshipStatusAccepted != "accepted" {
		t.Errorf("expected 'accepted', got %s", FriendshipStatusAccepted)
	}
	if FriendshipStatusBlocked != "blocked" {
		t.Errorf("expected 'blocked', got %s", FriendshipStatusBlocked)
	}
}

func TestErrRecordNotFound(t *testing.T) {
	if ErrRecordNotFound == nil {
		t.Fatal("ErrRecordNotFound should not be nil")
	}
	if ErrRecordNotFound.Error() != "record not found" {
		t.Errorf("expected 'record not found', got %s", ErrRecordNotFound.Error())
	}
}

func TestWebhookTypeConstants(t *testing.T) {
	if WebhookTypeIncoming != 1 {
		t.Errorf("expected WebhookTypeIncoming 1, got %d", WebhookTypeIncoming)
	}
	if WebhookTypeChannelFollower != 2 {
		t.Errorf("expected WebhookTypeChannelFollower 2, got %d", WebhookTypeChannelFollower)
	}
	if WebhookTypeApplication != 3 {
		t.Errorf("expected WebhookTypeApplication 3, got %d", WebhookTypeApplication)
	}
}

func TestEventTypeConstants(t *testing.T) {
	if EventTypeStage != 1 {
		t.Errorf("expected EventTypeStage 1, got %d", EventTypeStage)
	}
	if EventTypeVoice != 2 {
		t.Errorf("expected EventTypeVoice 2, got %d", EventTypeVoice)
	}
	if EventTypeExternal != 3 {
		t.Errorf("expected EventTypeExternal 3, got %d", EventTypeExternal)
	}
}

func TestEventStatusConstants(t *testing.T) {
	if EventStatusScheduled != 1 {
		t.Errorf("expected EventStatusScheduled 1, got %d", EventStatusScheduled)
	}
	if EventStatusActive != 2 {
		t.Errorf("expected EventStatusActive 2, got %d", EventStatusActive)
	}
	if EventStatusCompleted != 3 {
		t.Errorf("expected EventStatusCompleted 3, got %d", EventStatusCompleted)
	}
	if EventStatusCancelled != 4 {
		t.Errorf("expected EventStatusCancelled 4, got %d", EventStatusCancelled)
	}
}

func TestRSVPStatusConstants(t *testing.T) {
	if RSVPStatusInterested != 1 {
		t.Errorf("expected RSVPStatusInterested 1, got %d", RSVPStatusInterested)
	}
	if RSVPStatusGoing != 2 {
		t.Errorf("expected RSVPStatusGoing 2, got %d", RSVPStatusGoing)
	}
}

func TestLiveStreamTypeConstants(t *testing.T) {
	if LiveStreamTypeScreen != 1 {
		t.Errorf("expected LiveStreamTypeScreen 1, got %d", LiveStreamTypeScreen)
	}
	if LiveStreamTypeApplication != 2 {
		t.Errorf("expected LiveStreamTypeApplication 2, got %d", LiveStreamTypeApplication)
	}
	if LiveStreamTypeCamera != 3 {
		t.Errorf("expected LiveStreamTypeCamera 3, got %d", LiveStreamTypeCamera)
	}
}

func TestLiveStreamQualityConstants(t *testing.T) {
	if LiveStreamQuality480p != 1 {
		t.Errorf("expected LiveStreamQuality480p 1, got %d", LiveStreamQuality480p)
	}
	if LiveStreamQuality720p != 2 {
		t.Errorf("expected LiveStreamQuality720p 2, got %d", LiveStreamQuality720p)
	}
	if LiveStreamQuality1080p != 3 {
		t.Errorf("expected LiveStreamQuality1080p 3, got %d", LiveStreamQuality1080p)
	}
}

func TestLiveStreamStatusConstants(t *testing.T) {
	if LiveStreamStatusActive != 1 {
		t.Errorf("expected LiveStreamStatusActive 1, got %d", LiveStreamStatusActive)
	}
	if LiveStreamStatusEnded != 2 {
		t.Errorf("expected LiveStreamStatusEnded 2, got %d", LiveStreamStatusEnded)
	}
}

func TestActivityTypeConstants(t *testing.T) {
	if ActivityTypePlaying != 0 {
		t.Errorf("expected ActivityTypePlaying 0, got %d", ActivityTypePlaying)
	}
	if ActivityTypeStreaming != 1 {
		t.Errorf("expected ActivityTypeStreaming 1, got %d", ActivityTypeStreaming)
	}
	if ActivityTypeListening != 2 {
		t.Errorf("expected ActivityTypeListening 2, got %d", ActivityTypeListening)
	}
	if ActivityTypeWatching != 3 {
		t.Errorf("expected ActivityTypeWatching 3, got %d", ActivityTypeWatching)
	}
	if ActivityTypeCustom != 4 {
		t.Errorf("expected ActivityTypeCustom 4, got %d", ActivityTypeCustom)
	}
	if ActivityTypeCompeting != 5 {
		t.Errorf("expected ActivityTypeCompeting 5, got %d", ActivityTypeCompeting)
	}
}

func TestDeviceTypeConstants(t *testing.T) {
	if DeviceTypeDesktop != "desktop" {
		t.Errorf("expected 'desktop', got %s", DeviceTypeDesktop)
	}
	if DeviceTypeMobile != "mobile" {
		t.Errorf("expected 'mobile', got %s", DeviceTypeMobile)
	}
	if DeviceTypeTablet != "tablet" {
		t.Errorf("expected 'tablet', got %s", DeviceTypeTablet)
	}
	if DeviceTypeUnknown != "unknown" {
		t.Errorf("expected 'unknown', got %s", DeviceTypeUnknown)
	}
}

func TestWSEventConstants(t *testing.T) {
	// Verify a sampling of important WS event constants
	if WSEventHeartbeat != "HEARTBEAT" {
		t.Errorf("expected 'HEARTBEAT', got %s", WSEventHeartbeat)
	}
	if WSEventIdentify != "IDENTIFY" {
		t.Errorf("expected 'IDENTIFY', got %s", WSEventIdentify)
	}
	if WSEventReady != "READY" {
		t.Errorf("expected 'READY', got %s", WSEventReady)
	}
	if WSEventMessageCreate != "MESSAGE_CREATE" {
		t.Errorf("expected 'MESSAGE_CREATE', got %s", WSEventMessageCreate)
	}
	if WSEventPresenceUpdate != "PRESENCE_UPDATE" {
		t.Errorf("expected 'PRESENCE_UPDATE', got %s", WSEventPresenceUpdate)
	}
	if WSEventServerCreate != "SERVER_CREATE" {
		t.Errorf("expected 'SERVER_CREATE', got %s", WSEventServerCreate)
	}
	if WSEventThreadCreate != "THREAD_CREATE" {
		t.Errorf("expected 'THREAD_CREATE', got %s", WSEventThreadCreate)
	}
}

func TestAuditLogActionTypes(t *testing.T) {
	actions := []string{
		AuditLogServerUpdate, AuditLogChannelCreate, AuditLogChannelUpdate,
		AuditLogChannelDelete, AuditLogMemberKick, AuditLogMemberBan,
		AuditLogMemberUnban, AuditLogMemberUpdate, AuditLogRoleCreate,
		AuditLogRoleUpdate, AuditLogRoleDelete, AuditLogInviteCreate,
		AuditLogInviteDelete, AuditLogWebhookCreate, AuditLogWebhookUpdate,
		AuditLogWebhookDelete, AuditLogEmojiCreate, AuditLogEmojiUpdate,
		AuditLogEmojiDelete, AuditLogMessageDelete, AuditLogMessageBulkDelete,
		AuditLogMessagePin, AuditLogMessageUnpin,
	}

	seen := make(map[string]bool)
	for _, action := range actions {
		if action == "" {
			t.Error("audit log action type should not be empty")
		}
		if seen[action] {
			t.Errorf("duplicate audit log action type: %s", action)
		}
		seen[action] = true
	}
}

func TestStickerFormatConstants(t *testing.T) {
	if StickerFormatPNG != "PNG" {
		t.Errorf("expected 'PNG', got %s", StickerFormatPNG)
	}
	if StickerFormatAPNG != "APNG" {
		t.Errorf("expected 'APNG', got %s", StickerFormatAPNG)
	}
	if StickerFormatGIF != "GIF" {
		t.Errorf("expected 'GIF', got %s", StickerFormatGIF)
	}
}

func TestConnectedAccountTypes(t *testing.T) {
	types := []ConnectedAccountType{
		ConnectedAccountGitHub, ConnectedAccountTwitter, ConnectedAccountSpotify,
		ConnectedAccountSteam, ConnectedAccountTwitch, ConnectedAccountYouTube,
		ConnectedAccountReddit, ConnectedAccountPlayStation, ConnectedAccountXbox,
	}

	seen := make(map[ConnectedAccountType]bool)
	for _, ct := range types {
		if ct == "" {
			t.Error("ConnectedAccountType should not be empty")
		}
		if seen[ct] {
			t.Errorf("duplicate ConnectedAccountType: %s", ct)
		}
		seen[ct] = true
	}
}

func TestConnectedAccountVisibility(t *testing.T) {
	if VisibilityPrivate != 0 {
		t.Errorf("expected VisibilityPrivate 0, got %d", VisibilityPrivate)
	}
	if VisibilityFriendsOnly != 1 {
		t.Errorf("expected VisibilityFriendsOnly 1, got %d", VisibilityFriendsOnly)
	}
	if VisibilityEveryone != 2 {
		t.Errorf("expected VisibilityEveryone 2, got %d", VisibilityEveryone)
	}
}

func TestBadgeTypeConstants(t *testing.T) {
	badges := []string{
		BadgeEarlySupporter, BadgeVerifiedBot, BadgeBugHunter,
		BadgePremium, BadgeStaff, BadgePartner,
		BadgeHypeSquad, BadgeNitro,
	}

	seen := make(map[string]bool)
	for _, b := range badges {
		if b == "" {
			t.Error("badge type should not be empty")
		}
		if seen[b] {
			t.Errorf("duplicate badge type: %s", b)
		}
		seen[b] = true
	}
}

func TestDigestStatusConstants(t *testing.T) {
	statuses := []DigestStatus{
		DigestStatusPending, DigestStatusSent,
		DigestStatusFailed, DigestStatusSkipped,
	}
	seen := make(map[DigestStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("DigestStatus should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate DigestStatus: %s", s)
		}
		seen[s] = true
	}
}

func TestAutoArchiveConstants(t *testing.T) {
	if AutoArchive1Hour != 60 {
		t.Errorf("expected 60, got %d", AutoArchive1Hour)
	}
	if AutoArchive24Hour != 1440 {
		t.Errorf("expected 1440, got %d", AutoArchive24Hour)
	}
	if AutoArchive3Day != 4320 {
		t.Errorf("expected 4320, got %d", AutoArchive3Day)
	}
	if AutoArchive1Week != 10080 {
		t.Errorf("expected 10080, got %d", AutoArchive1Week)
	}
}
