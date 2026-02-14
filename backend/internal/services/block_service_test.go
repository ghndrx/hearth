package services

import (
	"context"
	"testing"
)

func TestBlockUser(t *testing.T) {
	ctx := context.Background()
	service := NewBlockService()
	userID := "user123"
	targetID := "target456"

	err := service.BlockUser(ctx, userID, targetID)
	if err != nil {
		t.Fatalf("BlockUser failed: %v", err)
	}

	// Verify via IsBlocked
	isBlocked, err := service.IsBlocked(ctx, userID, targetID)
	if err != nil {
		t.Fatalf("IsBlocked check failed: %v", err)
	}
	if !isBlocked {
		t.Error("Expected user to be blocked, but IsBlocked returned false")
	}
}

func TestGetBlockedUsers(t *testing.T) {
	ctx := context.Background()
	service := NewBlockService()
	userID := "mod1"
	targetA := "user_a"
	targetB := "user_b"

	// Block two users
	service.BlockUser(ctx, userID, targetA)
	service.BlockUser(ctx, userID, targetB)

	// Fetch list
	blocked, err := service.GetBlockedUsers(ctx, userID)
	if err != nil {
		t.Fatalf("GetBlockedUsers failed: %v", err)
	}

	if len(blocked) != 2 {
		t.Errorf("Expected 2 blocked users, got %d", len(blocked))
	}

	// Simple check to ensure they match the expected IDs
	foundA := false
	foundB := false
	for _, u := range blocked {
		if u == targetA {
			foundA = true
		}
		if u == targetB {
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Error("Could not find expected users in the blocked list")
	}
}

func TestUnblockUser(t *testing.T) {
	ctx := context.Background()
	service := NewBlockService()
	userID := "alice"
	targetID := "eve"

	// Block first
	service.BlockUser(ctx, userID, targetID)

	// Verify blocked
	isBlocked, _ := service.IsBlocked(ctx, userID, targetID)
	if !isBlocked {
		t.Fatal("Setup failed: user should be blocked")
	}

	// Unblock
	err := service.UnblockUser(ctx, userID, targetID)
	if err != nil {
		t.Fatalf("UnblockUser failed: %v", err)
	}

	// Verify unblocked
	isBlocked, _ = service.IsBlocked(ctx, userID, targetID)
	if isBlocked {
		t.Error("Expected user to be unblocked, but IsBlocked returned true")
	}
}

func TestIsBlocked_MultipleUpdates(t *testing.T) {
	ctx := context.Background()
	service := NewBlockService()
	userID := "bob"
	target1 := "mixer1"
	target2 := "mixer2"

	// Test initial state
	isBlocked, _ := service.IsBlocked(ctx, userID, target1)
	if isBlocked {
		t.Error("Initial state should not be blocked")
	}

	// Block target1
	service.BlockUser(ctx, userID, target1)
	isBlocked, _ = service.IsBlocked(ctx, userID, target1)
	if !isBlocked {
		t.Error("target1 should be blocked after BlockUser")
	}

	// Block target2 (target1 remains blocked)
	service.BlockUser(ctx, userID, target2)
	isBlocked, _ = service.IsBlocked(ctx, userID, target1)
	if !isBlocked {
		t.Error("target1 should still be blocked after blocking target2")
	}
}

func TestErrorMsgs(t *testing.T) {
	ctx := context.Background()
	service := NewBlockService()

	tests := []struct {
		name       string
		operation  func(ctx context.Context, userID, targetID string) error
		userID     string
		targetID   string
		expectErr  bool
		expectMsg  string
	}{
		{
			name:       "Block with empty User ID",
			operation:  service.BlockUser,
			userID:     "",
			targetID:   "123",
			expectErr:  true,
			expectMsg:  "user IDs cannot be empty",
		},
		{
			name:       "Unblock with empty User ID",
			operation:  service.UnblockUser,
			userID:     "",
			targetID:   "123",
			expectErr:  true,
			expectMsg:  "user IDs cannot be empty",
		},
		{
			name:       "GetBlocked with empty User ID",
			operation:  service.GetBlockedUsers,
			userID:     "",
			targetID:   "",
			expectErr:  true,
			expectMsg:  "user ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation(ctx, tt.userID, tt.targetID)
			if (err != nil) != tt.expectErr {
				t.Errorf("operation error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.expectErr && err.Error() != tt.expectMsg {
				t.Errorf("error message = %v, want %v", err.Error(), tt.expectMsg)
			}
		})
	}
}