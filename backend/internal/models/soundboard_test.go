package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSoundboardSoundToResponse(t *testing.T) {
	serverID := uuid.New()
	creatorID := uuid.New()
	now := time.Now()

	sound := &SoundboardSound{
		ID:         uuid.New(),
		ServerID:   &serverID,
		Name:       "airhorn",
		EmojiName:  "horn",
		Volume:     0.75,
		AudioURL:   "https://cdn.example.com/airhorn.mp3",
		DurationMs: 3000,
		Available:  true,
		CreatorID:  creatorID,
		CreatedAt:  now,
	}

	resp := sound.ToResponse()

	if resp.ID != sound.ID.String() {
		t.Errorf("expected ID %s, got %s", sound.ID.String(), resp.ID)
	}
	if resp.ServerID == nil || *resp.ServerID != serverID.String() {
		t.Errorf("expected ServerID %s, got %v", serverID.String(), resp.ServerID)
	}
	if resp.Name != "airhorn" {
		t.Errorf("expected Name 'airhorn', got %s", resp.Name)
	}
	if resp.EmojiName != "horn" {
		t.Errorf("expected EmojiName 'horn', got %s", resp.EmojiName)
	}
	if resp.Volume != 0.75 {
		t.Errorf("expected Volume 0.75, got %f", resp.Volume)
	}
	if resp.AudioURL != sound.AudioURL {
		t.Errorf("expected AudioURL %s, got %s", sound.AudioURL, resp.AudioURL)
	}
	if resp.DurationMs != 3000 {
		t.Errorf("expected DurationMs 3000, got %d", resp.DurationMs)
	}
	if !resp.Available {
		t.Error("expected Available true")
	}
	if resp.CreatorID != creatorID.String() {
		t.Errorf("expected CreatorID %s, got %s", creatorID.String(), resp.CreatorID)
	}
	if resp.CreatedAt != now.Format(time.RFC3339) {
		t.Errorf("expected CreatedAt %s, got %s", now.Format(time.RFC3339), resp.CreatedAt)
	}
}

func TestSoundboardSoundToResponseNilServerID(t *testing.T) {
	sound := &SoundboardSound{
		ID:        uuid.New(),
		ServerID:  nil,
		Name:      "default-sound",
		CreatorID: uuid.New(),
		CreatedAt: time.Now(),
	}

	resp := sound.ToResponse()
	if resp.ServerID != nil {
		t.Errorf("expected nil ServerID, got %v", resp.ServerID)
	}
}
