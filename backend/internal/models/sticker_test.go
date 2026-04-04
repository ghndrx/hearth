package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStickerTierFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected StickerPackTier
	}{
		{"basic", StickerPackTierBasic},
		{"premium", StickerPackTierPremium},
		{"free", StickerPackTierFree},
		{"unknown", StickerPackTierFree},
		{"", StickerPackTierFree},
		{"PREMIUM", StickerPackTierFree}, // case-sensitive
		{"Basic", StickerPackTierFree},   // case-sensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := StickerTierFromString(tc.input)
			if result != tc.expected {
				t.Errorf("StickerTierFromString(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTierMeetsRequirement(t *testing.T) {
	tests := []struct {
		name         string
		userTier     StickerPackTier
		requiredTier StickerPackTier
		expected     bool
	}{
		{"free user, free tier", StickerPackTierFree, StickerPackTierFree, true},
		{"free user, basic tier", StickerPackTierFree, StickerPackTierBasic, false},
		{"free user, premium tier", StickerPackTierFree, StickerPackTierPremium, false},
		{"basic user, free tier", StickerPackTierBasic, StickerPackTierFree, true},
		{"basic user, basic tier", StickerPackTierBasic, StickerPackTierBasic, true},
		{"basic user, premium tier", StickerPackTierBasic, StickerPackTierPremium, false},
		{"premium user, free tier", StickerPackTierPremium, StickerPackTierFree, true},
		{"premium user, basic tier", StickerPackTierPremium, StickerPackTierBasic, true},
		{"premium user, premium tier", StickerPackTierPremium, StickerPackTierPremium, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TierMeetsRequirement(tc.userTier, tc.requiredTier)
			if result != tc.expected {
				t.Errorf("TierMeetsRequirement(%v, %v) = %v; want %v", tc.userTier, tc.requiredTier, result, tc.expected)
			}
		})
	}
}

func TestStickerPackToPackResponse(t *testing.T) {
	now := time.Now()
	serverID := uuid.New()
	creatorID := uuid.New()
	desc := "A test pack"

	pack := &StickerPack{
		ID:           uuid.New(),
		Name:         "Test Pack",
		Description:  &desc,
		IconURL:      nil,
		Tier:         StickerPackTierBasic,
		StickerCount: 2,
		IsActive:     true,
		IsGlobal:     false,
		ServerID:     &serverID,
		CreatedBy:    &creatorID,
		CreatedAt:    now,
		UpdatedAt:    now,
		Stickers:     nil,
	}

	resp := pack.ToPackResponse()

	if resp.ID != pack.ID.String() {
		t.Errorf("expected ID %s, got %s", pack.ID.String(), resp.ID)
	}
	if resp.Name != "Test Pack" {
		t.Errorf("expected Name 'Test Pack', got %s", resp.Name)
	}
	if resp.Description == nil || *resp.Description != desc {
		t.Errorf("expected Description %q, got %v", desc, resp.Description)
	}
	if resp.Tier != string(StickerPackTierBasic) {
		t.Errorf("expected Tier 'basic', got %s", resp.Tier)
	}
	if resp.StickerCount != 2 {
		t.Errorf("expected StickerCount 2, got %d", resp.StickerCount)
	}
	if !resp.IsActive {
		t.Error("expected IsActive true")
	}
	if resp.IsGlobal {
		t.Error("expected IsGlobal false")
	}
	if resp.ServerID == nil || *resp.ServerID != serverID.String() {
		t.Errorf("expected ServerID %s, got %v", serverID.String(), resp.ServerID)
	}
	if resp.CreatedBy == nil || *resp.CreatedBy != creatorID.String() {
		t.Errorf("expected CreatedBy %s, got %v", creatorID.String(), resp.CreatedBy)
	}
	if len(resp.Stickers) != 0 {
		t.Errorf("expected empty Stickers, got %d", len(resp.Stickers))
	}
}

func TestStickerPackToPackResponseWithStickers(t *testing.T) {
	now := time.Now()
	serverID := uuid.New()
	creatorID := uuid.New()

	packID := uuid.New()
	sticker1 := &Sticker{
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "Sticker 1",
		Tags:         []string{"tag1"},
		URL:          "https://cdn.example.com/s1.png",
		Format:       StickerFormatPNG,
		RequiredTier: StickerPackTierFree,
		CreatedBy:    creatorID,
		CreatedAt:    now,
	}
	sticker2 := &Sticker{
		ID:           uuid.New(),
		ServerID:     nil,
		Name:         "Sticker 2",
		Tags:         []string{"tag2", "tag3"},
		URL:          "https://cdn.example.com/s2.gif",
		Format:       StickerFormatGIF,
		RequiredTier: StickerPackTierPremium,
		CreatedBy:    creatorID,
		CreatedAt:    now,
	}

	pack := &StickerPack{
		ID:           packID,
		Name:         "Pack With Stickers",
		Description:  nil,
		IconURL:      nil,
		Tier:         StickerPackTierPremium,
		StickerCount: 2,
		IsActive:     true,
		IsGlobal:     true,
		ServerID:     nil,
		CreatedBy:    nil,
		CreatedAt:    now,
		UpdatedAt:    now,
		Stickers:     []*Sticker{sticker1, sticker2},
	}

	resp := pack.ToPackResponse()

	if len(resp.Stickers) != 2 {
		t.Fatalf("expected 2 stickers, got %d", len(resp.Stickers))
	}

	// Check first sticker
	if resp.Stickers[0].ID != sticker1.ID.String() {
		t.Errorf("expected sticker1 ID %s, got %s", sticker1.ID.String(), resp.Stickers[0].ID)
	}
	if resp.Stickers[0].Name != "Sticker 1" {
		t.Errorf("expected sticker1 Name 'Sticker 1', got %s", resp.Stickers[0].Name)
	}
	if resp.Stickers[0].ServerID == nil || *resp.Stickers[0].ServerID != serverID.String() {
		t.Errorf("expected sticker1 ServerID %s, got %v", serverID.String(), resp.Stickers[0].ServerID)
	}
	if resp.Stickers[0].RequiredTier != string(StickerPackTierFree) {
		t.Errorf("expected sticker1 RequiredTier 'free', got %s", resp.Stickers[0].RequiredTier)
	}

	// Check second sticker (global)
	if resp.Stickers[1].ID != sticker2.ID.String() {
		t.Errorf("expected sticker2 ID %s, got %s", sticker2.ID.String(), resp.Stickers[1].ID)
	}
	if resp.Stickers[1].ServerID != nil {
		t.Errorf("expected sticker2 ServerID nil, got %v", resp.Stickers[1].ServerID)
	}
	if resp.Stickers[1].RequiredTier != string(StickerPackTierPremium) {
		t.Errorf("expected sticker2 RequiredTier 'premium', got %s", resp.Stickers[1].RequiredTier)
	}

	// Global pack should have nil server and creator
	if resp.ServerID != nil {
		t.Errorf("expected nil ServerID for global pack, got %v", resp.ServerID)
	}
	if resp.CreatedBy != nil {
		t.Errorf("expected nil CreatedBy for global pack, got %v", resp.CreatedBy)
	}
}

func TestStickerPackToPackResponseIconURL(t *testing.T) {
	now := time.Now()
	iconURL := "https://cdn.example.com/icon.png"

	pack := &StickerPack{
		ID:          uuid.New(),
		Name:        "Pack With Icon",
		Description: nil,
		IconURL:     &iconURL,
		Tier:        StickerPackTierFree,
		StickerCount: 0,
		IsActive:    true,
		IsGlobal:    false,
		ServerID:    nil,
		CreatedBy:   nil,
		CreatedAt:   now,
		UpdatedAt:   now,
		Stickers:    nil,
	}

	resp := pack.ToPackResponse()

	if resp.IconURL == nil {
		t.Fatal("expected IconURL, got nil")
	}
	if *resp.IconURL != iconURL {
		t.Errorf("expected IconURL %s, got %s", iconURL, *resp.IconURL)
	}
}
