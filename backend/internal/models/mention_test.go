package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestMentionFilterSetDefaults(t *testing.T) {
	tests := []struct {
		name          string
		initialLimit  int
		expectedLimit int
	}{
		{"zero defaults to 50", 0, 50},
		{"negative defaults to 50", -1, 50},
		{"within range unchanged", 50, 50},
		{"above max capped to 100", 200, 100},
		{"exactly 100 unchanged", 100, 100},
		{"exactly 1 unchanged", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &MentionFilter{
				UserID: uuid.New(),
				Limit:  tt.initialLimit,
			}
			filter.SetDefaults()
			if filter.Limit != tt.expectedLimit {
				t.Errorf("SetDefaults() Limit = %d, want %d", filter.Limit, tt.expectedLimit)
			}
		})
	}
}

func TestMentionKindConstants(t *testing.T) {
	kinds := []MentionKind{
		MentionKindUser, MentionKindRole, MentionKindChannel,
		MentionKindEveryone, MentionKindHere,
	}

	for _, kind := range kinds {
		if kind == "" {
			t.Error("MentionKind constant should not be empty")
		}
	}

	if MentionKindUser != "user" {
		t.Errorf("expected 'user', got %s", MentionKindUser)
	}
	if MentionKindRole != "role" {
		t.Errorf("expected 'role', got %s", MentionKindRole)
	}
	if MentionKindChannel != "channel" {
		t.Errorf("expected 'channel', got %s", MentionKindChannel)
	}
	if MentionKindEveryone != "everyone" {
		t.Errorf("expected 'everyone', got %s", MentionKindEveryone)
	}
	if MentionKindHere != "here" {
		t.Errorf("expected 'here', got %s", MentionKindHere)
	}
}
