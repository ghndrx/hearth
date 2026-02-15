package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChannelTypes(t *testing.T) {
	// Test all channel type constants
	expectedTypes := map[ChannelType]string{
		ChannelTypeText:         "text",
		ChannelTypeVoice:        "voice",
		ChannelTypeCategory:     "category",
		ChannelTypeAnnouncement: "announcement",
		ChannelTypeForum:        "forum",
		ChannelTypeStage:        "stage",
		ChannelTypeDM:           "dm",
		ChannelTypeGroupDM:      "group_dm",
	}

	for channelType, expected := range expectedTypes {
		if string(channelType) != expected {
			t.Errorf("expected channel type %s, got %s", expected, string(channelType))
		}
	}
}

func TestAutoArchiveDurations(t *testing.T) {
	if AutoArchive1Hour != 60 {
		t.Errorf("expected AutoArchive1Hour to be 60, got %d", AutoArchive1Hour)
	}
	if AutoArchive24Hour != 1440 {
		t.Errorf("expected AutoArchive24Hour to be 1440, got %d", AutoArchive24Hour)
	}
	if AutoArchive3Day != 4320 {
		t.Errorf("expected AutoArchive3Day to be 4320, got %d", AutoArchive3Day)
	}
	if AutoArchive1Week != 10080 {
		t.Errorf("expected AutoArchive1Week to be 10080, got %d", AutoArchive1Week)
	}
}

func TestChannelStruct(t *testing.T) {
	serverID := uuid.New()
	parentID := uuid.New()
	ownerID := uuid.New()
	lastMsgID := uuid.New()
	rtcRegion := "us-west"

	channel := Channel{
		ID:                 uuid.New(),
		ServerID:           &serverID,
		ParentID:           &parentID,
		OwnerID:            &ownerID,
		Type:               ChannelTypeText,
		Name:               "general",
		Topic:              "General discussion",
		Position:           0,
		NSFW:               false,
		Slowmode:           5,
		E2EEEnabled:        true,
		Bitrate:            64000,
		UserLimit:          10,
		RTCRegion:          &rtcRegion,
		DefaultAutoArchive: AutoArchive24Hour,
		LastMessageID:      &lastMsgID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		PermissionOverrides: []PermissionOverride{
			{
				ChannelID:  uuid.New(),
				TargetType: "role",
				TargetID:   uuid.New(),
				Allow:      1024,
				Deny:       0,
			},
		},
		Recipients: []uuid.UUID{uuid.New()},
	}

	// Verify fields
	if channel.Name != "general" {
		t.Errorf("expected name 'general', got %s", channel.Name)
	}
	if channel.Type != ChannelTypeText {
		t.Errorf("expected type 'text', got %s", channel.Type)
	}
	if channel.Slowmode != 5 {
		t.Errorf("expected slowmode 5, got %d", channel.Slowmode)
	}
	if !channel.E2EEEnabled {
		t.Error("expected E2EEEnabled to be true")
	}
	if len(channel.PermissionOverrides) != 1 {
		t.Errorf("expected 1 permission override, got %d", len(channel.PermissionOverrides))
	}
	if len(channel.Recipients) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(channel.Recipients))
	}
}

func TestCreateChannelRequest(t *testing.T) {
	topic := "Test topic"
	categoryID := "category-123"
	position := 5
	nsfw := true
	bitrate := 128000
	userLimit := 50

	req := CreateChannelRequest{
		Name:       "test-channel",
		Type:       ChannelTypeVoice,
		Topic:      &topic,
		CategoryID: &categoryID,
		Position:   &position,
		NSFW:       &nsfw,
		Bitrate:    &bitrate,
		UserLimit:  &userLimit,
	}

	if req.Name != "test-channel" {
		t.Errorf("expected name 'test-channel', got %s", req.Name)
	}
	if req.Type != ChannelTypeVoice {
		t.Errorf("expected type 'voice', got %s", req.Type)
	}
	if *req.Topic != "Test topic" {
		t.Errorf("expected topic 'Test topic', got %s", *req.Topic)
	}
	if *req.Bitrate != 128000 {
		t.Errorf("expected bitrate 128000, got %d", *req.Bitrate)
	}
}

func TestUpdateChannelRequest(t *testing.T) {
	name := "updated-channel"
	topic := "Updated topic"
	position := 10
	categoryID := "new-category"
	nsfw := false
	slowmode := 30
	bitrate := 256000
	userLimit := 25

	req := UpdateChannelRequest{
		Name:            &name,
		Topic:           &topic,
		Position:        &position,
		CategoryID:      &categoryID,
		NSFW:            &nsfw,
		SlowmodeSeconds: &slowmode,
		Bitrate:         &bitrate,
		UserLimit:       &userLimit,
	}

	if *req.Name != "updated-channel" {
		t.Errorf("expected name 'updated-channel', got %s", *req.Name)
	}
	if *req.SlowmodeSeconds != 30 {
		t.Errorf("expected slowmode 30, got %d", *req.SlowmodeSeconds)
	}
}

func TestChannelUpdate(t *testing.T) {
	name := "partial-update"
	topic := "New topic"
	position := 3
	slowmode := 10
	nsfw := true
	e2ee := false

	update := ChannelUpdate{
		Name:        &name,
		Topic:       &topic,
		Position:    &position,
		Slowmode:    &slowmode,
		NSFW:        &nsfw,
		E2EEEnabled: &e2ee,
	}

	if *update.Name != "partial-update" {
		t.Errorf("expected name 'partial-update', got %s", *update.Name)
	}
	if *update.Slowmode != 10 {
		t.Errorf("expected slowmode 10, got %d", *update.Slowmode)
	}
}

func TestPermissionOverride(t *testing.T) {
	channelID := uuid.New()
	targetID := uuid.New()

	override := PermissionOverride{
		ChannelID:  channelID,
		TargetType: "user",
		TargetID:   targetID,
		Allow:      0x10, // VIEW_CHANNEL
		Deny:       0x08, // SEND_MESSAGES
	}

	if override.TargetType != "user" {
		t.Errorf("expected target type 'user', got %s", override.TargetType)
	}
	if override.Allow != 0x10 {
		t.Errorf("expected allow 0x10, got %d", override.Allow)
	}
	if override.Deny != 0x08 {
		t.Errorf("expected deny 0x08, got %d", override.Deny)
	}
}

func TestThread(t *testing.T) {
	now := time.Now()
	archiveTime := now.Add(-time.Hour)

	thread := Thread{
		ID:               uuid.New(),
		ParentChannelID:  uuid.New(),
		OwnerID:          uuid.New(),
		Name:             "Test Thread",
		MessageCount:     42,
		MemberCount:      5,
		Archived:         true,
		AutoArchive:      AutoArchive1Hour,
		Locked:           false,
		CreatedAt:        now,
		ArchiveTimestamp: &archiveTime,
		ParentChannel:    nil,
	}

	if thread.Name != "Test Thread" {
		t.Errorf("expected name 'Test Thread', got %s", thread.Name)
	}
	if thread.MessageCount != 42 {
		t.Errorf("expected message count 42, got %d", thread.MessageCount)
	}
	if !thread.Archived {
		t.Error("expected thread to be archived")
	}
	if thread.AutoArchive != AutoArchive1Hour {
		t.Errorf("expected auto archive %d, got %d", AutoArchive1Hour, thread.AutoArchive)
	}
}

func TestCreateThreadRequest(t *testing.T) {
	autoArchive := AutoArchive3Day

	req := CreateThreadRequest{
		Name:        "New Thread",
		AutoArchive: &autoArchive,
	}

	if req.Name != "New Thread" {
		t.Errorf("expected name 'New Thread', got %s", req.Name)
	}
	if *req.AutoArchive != AutoArchive3Day {
		t.Errorf("expected auto archive %d, got %d", AutoArchive3Day, *req.AutoArchive)
	}
}

func TestDMChannelRecipient(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	recipient := DMChannelRecipient{
		ChannelID: channelID,
		UserID:    userID,
	}

	if recipient.ChannelID != channelID {
		t.Errorf("expected channel ID %v, got %v", channelID, recipient.ChannelID)
	}
	if recipient.UserID != userID {
		t.Errorf("expected user ID %v, got %v", userID, recipient.UserID)
	}
}

func TestCreateDMRequest(t *testing.T) {
	req := CreateDMRequest{
		RecipientID: "user-123",
	}

	if req.RecipientID != "user-123" {
		t.Errorf("expected recipient ID 'user-123', got %s", req.RecipientID)
	}
}

func TestCreateGroupDMRequest(t *testing.T) {
	name := "Friend Group"

	req := CreateGroupDMRequest{
		RecipientIDs: []string{"user-1", "user-2", "user-3"},
		Name:         &name,
	}

	if len(req.RecipientIDs) != 3 {
		t.Errorf("expected 3 recipients, got %d", len(req.RecipientIDs))
	}
	if *req.Name != "Friend Group" {
		t.Errorf("expected name 'Friend Group', got %s", *req.Name)
	}
}
