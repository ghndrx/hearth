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

func TestSoundboardSoundPackToResponse(t *testing.T) {
	now := time.Now()
	serverID := uuid.New()
	sound1ID := uuid.New()
	sound2ID := uuid.New()
	creatorID := uuid.New()

	sound1 := &SoundboardSound{
		ID:         sound1ID,
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
	sound2 := &SoundboardSound{
		ID:         sound2ID,
		ServerID:   &serverID,
		Name:       "sad-trombone",
		EmojiName:  "trombone",
		Volume:     0.5,
		AudioURL:   "https://cdn.example.com/sad.mp3",
		DurationMs: 5000,
		Available:  true,
		CreatorID:  creatorID,
		CreatedAt:  now,
	}

	pack := &SoundboardSoundPack{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "Fun Sounds",
		EmojiName: "laughing",
		IsDefault: false,
		Position:  1,
		CreatedAt: now,
		UpdatedAt: now,
		Sounds:    []*SoundboardSound{sound1, sound2},
	}

	resp := pack.ToResponse()

	if resp.ID != pack.ID.String() {
		t.Errorf("ID = %s; want %s", resp.ID, pack.ID.String())
	}
	if resp.Name != "Fun Sounds" {
		t.Errorf("Name = %s; want 'Fun Sounds'", resp.Name)
	}
	if resp.EmojiName != "laughing" {
		t.Errorf("EmojiName = %s; want 'laughing'", resp.EmojiName)
	}
	if resp.IsDefault {
		t.Error("IsDefault = true; want false")
	}
	if resp.Position != 1 {
		t.Errorf("Position = %d; want 1", resp.Position)
	}
	if resp.ServerID == nil || *resp.ServerID != serverID.String() {
		t.Errorf("ServerID = %v; want %s", resp.ServerID, serverID.String())
	}
	if len(resp.Sounds) != 2 {
		t.Fatalf("len(Sounds) = %d; want 2", len(resp.Sounds))
	}
	if resp.Sounds[0].ID != sound1ID.String() {
		t.Errorf("Sounds[0].ID = %s; want %s", resp.Sounds[0].ID, sound1ID.String())
	}
	if resp.Sounds[1].ID != sound2ID.String() {
		t.Errorf("Sounds[1].ID = %s; want %s", resp.Sounds[1].ID, sound2ID.String())
	}
}

func TestSoundboardSoundPackToResponseNilServerID(t *testing.T) {
	now := time.Now()
	pack := &SoundboardSoundPack{
		ID:        uuid.New(),
		ServerID:  nil,
		Name:      "Default Pack",
		EmojiName: "sound",
		IsDefault: true,
		Position:  0,
		CreatedAt: now,
		UpdatedAt: now,
		Sounds:    nil,
	}

	resp := pack.ToResponse()

	if resp.ID != pack.ID.String() {
		t.Errorf("ID = %s; want %s", resp.ID, pack.ID.String())
	}
	if resp.ServerID != nil {
		t.Errorf("ServerID = %v; want nil", resp.ServerID)
	}
	if !resp.IsDefault {
		t.Error("IsDefault = false; want true")
	}
	if len(resp.Sounds) != 0 {
		t.Errorf("len(Sounds) = %d; want 0", len(resp.Sounds))
	}
}
