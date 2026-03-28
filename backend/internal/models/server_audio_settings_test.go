package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultServerAudioSettings(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	settings := DefaultServerAudioSettings(userID, serverID)

	if settings.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, settings.UserID)
	}
	if settings.ServerID != serverID {
		t.Errorf("expected ServerID %v, got %v", serverID, settings.ServerID)
	}
	if settings.InputDeviceID != "" {
		t.Errorf("expected empty InputDeviceID, got %s", settings.InputDeviceID)
	}
	if settings.OutputDeviceID != "" {
		t.Errorf("expected empty OutputDeviceID, got %s", settings.OutputDeviceID)
	}
	if settings.InputVolume != 100 {
		t.Errorf("expected InputVolume 100, got %d", settings.InputVolume)
	}
	if settings.OutputVolume != 100 {
		t.Errorf("expected OutputVolume 100, got %d", settings.OutputVolume)
	}
	if settings.PushToTalkEnabled {
		t.Error("expected PushToTalkEnabled false")
	}
	if settings.PushToTalkKey != "" {
		t.Errorf("expected empty PushToTalkKey, got %s", settings.PushToTalkKey)
	}
	if settings.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}
