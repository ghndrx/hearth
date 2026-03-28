package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStickerToResponse(t *testing.T) {
	now := time.Now()
	serverID := uuid.New()
	userID := uuid.New()

	sticker := &Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "Test Sticker",
		Tags:      []string{"tag1", "tag2"},
		URL:       "https://cdn.example.com/sticker.png",
		Format:    StickerFormatPNG,
		CreatedBy: userID,
		CreatedAt: now,
	}

	resp := sticker.ToResponse()

	if resp.ID != sticker.ID.String() {
		t.Errorf("expected ID %s, got %s", sticker.ID.String(), resp.ID)
	}
	if resp.ServerID == nil || *resp.ServerID != serverID.String() {
		t.Errorf("expected ServerID %s, got %v", serverID.String(), resp.ServerID)
	}
	if resp.Name != sticker.Name {
		t.Errorf("expected Name %s, got %s", sticker.Name, resp.Name)
	}
	if len(resp.Tags) != 2 || resp.Tags[0] != "tag1" || resp.Tags[1] != "tag2" {
		t.Errorf("expected Tags [tag1 tag2], got %v", resp.Tags)
	}
	if resp.URL != sticker.URL {
		t.Errorf("expected URL %s, got %s", sticker.URL, resp.URL)
	}
	if resp.Format != string(sticker.Format) {
		t.Errorf("expected Format %s, got %s", string(sticker.Format), resp.Format)
	}
	if resp.CreatedBy != userID.String() {
		t.Errorf("expected CreatedBy %s, got %s", userID.String(), resp.CreatedBy)
	}
}

func TestStickerToResponseNilServerID(t *testing.T) {
	userID := uuid.New()
	sticker := &Sticker{
		ID:        uuid.New(),
		ServerID:  nil,
		Name:      "Sticker",
		Tags:      []string{},
		URL:       "https://cdn.example.com/sticker.gif",
		Format:    StickerFormatAPNG,
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	resp := sticker.ToResponse()

	if resp.ServerID != nil {
		t.Errorf("expected nil ServerID, got %v", resp.ServerID)
	}
}

func TestDefaultUserSettings(t *testing.T) {
	userID := uuid.New()
	settings := DefaultUserSettings(userID)

	if settings.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, settings.UserID)
	}
	if settings.Theme != "dark" {
		t.Errorf("expected Theme 'dark', got %s", settings.Theme)
	}
	if settings.MessageDisplay != "cozy" {
		t.Errorf("expected MessageDisplay 'cozy', got %s", settings.MessageDisplay)
	}
	if settings.CompactMode != false {
		t.Error("expected CompactMode false")
	}
	if settings.DeveloperMode != false {
		t.Error("expected DeveloperMode false")
	}
	if settings.InlineEmbeds != true {
		t.Error("expected InlineEmbeds true")
	}
	if settings.InlineAttachments != true {
		t.Error("expected InlineAttachments true")
	}
	if settings.RenderReactions != true {
		t.Error("expected RenderReactions true")
	}
	if settings.AnimateEmoji != true {
		t.Error("expected AnimateEmoji true")
	}
	if settings.EnableTTS != true {
		t.Error("expected EnableTTS true")
	}
	if settings.NotificationsEnabled != true {
		t.Error("expected NotificationsEnabled true")
	}
	if settings.NotificationsSound != true {
		t.Error("expected NotificationsSound true")
	}
}

func TestDefaultNotificationPreferences(t *testing.T) {
	userID := uuid.New()
	prefs := DefaultNotificationPreferences(userID)

	if prefs.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, prefs.UserID)
	}
	if prefs.PushEnabled != true {
		t.Error("expected PushEnabled true")
	}
	if prefs.PushMentions != true {
		t.Error("expected PushMentions true")
	}
	if prefs.PushDirectMessages != true {
		t.Error("expected PushDirectMessages true")
	}
	if prefs.PushReplies != true {
		t.Error("expected PushReplies true")
	}
	if prefs.PushFriendRequests != true {
		t.Error("expected PushFriendRequests true")
	}
	if prefs.PushServerInvites != true {
		t.Error("expected PushServerInvites true")
	}
	if prefs.SoundEnabled != true {
		t.Error("expected SoundEnabled true")
	}
	if prefs.SoundMessage != "default" {
		t.Errorf("expected SoundMessage 'default', got %s", prefs.SoundMessage)
	}
	if prefs.SoundMention != "mention" {
		t.Errorf("expected SoundMention 'mention', got %s", prefs.SoundMention)
	}
	if prefs.DesktopEnabled != true {
		t.Error("expected DesktopEnabled true")
	}
	if prefs.DesktopPreviews != true {
		t.Error("expected DesktopPreviews true")
	}
	if prefs.DoNotDisturb != false {
		t.Error("expected DoNotDisturb false")
	}
}
