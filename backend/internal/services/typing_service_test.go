package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTypingService_StartAndGet(t *testing.T) {
	svc := NewTypingService()
	ctx := context.Background()

	err := svc.StartTyping(ctx, "channel-1", "user-1")
	assert.NoError(t, err)

	users, err := svc.GetTypingUsers(ctx, "channel-1")
	assert.NoError(t, err)
	assert.Contains(t, users, "user-1")
}

func TestTypingService_StopTyping(t *testing.T) {
	svc := NewTypingService()
	ctx := context.Background()

	svc.StartTyping(ctx, "channel-1", "user-1")
	svc.StopTyping(ctx, "channel-1", "user-1")

	users, _ := svc.GetTypingUsers(ctx, "channel-1")
	assert.NotContains(t, users, "user-1")
}

func TestTypingService_CleanupExpiredIndicators(t *testing.T) {
	svc := NewTypingService()
	svc.ttl = 50 * time.Millisecond // Short TTL for testing
	ctx := context.Background()

	// Start typing
	svc.StartTyping(ctx, "channel-1", "user-1")

	// Verify user is typing
	users, _ := svc.GetTypingUsers(ctx, "channel-1")
	assert.Contains(t, users, "user-1")

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Verify user is no longer typing (expired and cleaned up)
	users, _ = svc.GetTypingUsers(ctx, "channel-1")
	assert.NotContains(t, users, "user-1")
}

func TestTypingService_CleanupEmptyChannel(t *testing.T) {
	svc := NewTypingService()
	svc.ttl = 50 * time.Millisecond // Short TTL for testing
	ctx := context.Background()

	// Start typing for single user
	svc.StartTyping(ctx, "channel-1", "user-1")

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Call GetTypingUsers to trigger cleanup
	users, _ := svc.GetTypingUsers(ctx, "channel-1")
	assert.Empty(t, users)

	// Verify channel map is cleaned up by starting another typing session
	// This ensures the old map was deleted
	svc.StartTyping(ctx, "channel-1", "user-2")
	users, _ = svc.GetTypingUsers(ctx, "channel-1")
	assert.Contains(t, users, "user-2")
}

func TestTypingService_MultipleUsers(t *testing.T) {
	svc := NewTypingService()
	ctx := context.Background()

	// Multiple users typing in same channel
	svc.StartTyping(ctx, "channel-1", "user-1")
	svc.StartTyping(ctx, "channel-1", "user-2")
	svc.StartTyping(ctx, "channel-1", "user-3")

	users, _ := svc.GetTypingUsers(ctx, "channel-1")
	assert.Len(t, users, 3)
	assert.Contains(t, users, "user-1")
	assert.Contains(t, users, "user-2")
	assert.Contains(t, users, "user-3")

	// Stop one user
	svc.StopTyping(ctx, "channel-1", "user-2")

	users, _ = svc.GetTypingUsers(ctx, "channel-1")
	assert.Len(t, users, 2)
	assert.Contains(t, users, "user-1")
	assert.NotContains(t, users, "user-2")
	assert.Contains(t, users, "user-3")
}

func TestTypingService_MultipleChannels(t *testing.T) {
	svc := NewTypingService()
	ctx := context.Background()

	// Same user typing in multiple channels
	svc.StartTyping(ctx, "channel-1", "user-1")
	svc.StartTyping(ctx, "channel-2", "user-1")
	svc.StartTyping(ctx, "channel-3", "user-1")

	users1, _ := svc.GetTypingUsers(ctx, "channel-1")
	users2, _ := svc.GetTypingUsers(ctx, "channel-2")
	users3, _ := svc.GetTypingUsers(ctx, "channel-3")

	assert.Contains(t, users1, "user-1")
	assert.Contains(t, users2, "user-1")
	assert.Contains(t, users3, "user-1")

	// Stop typing in one channel
	svc.StopTyping(ctx, "channel-2", "user-1")

	users1, _ = svc.GetTypingUsers(ctx, "channel-1")
	users2, _ = svc.GetTypingUsers(ctx, "channel-2")
	users3, _ = svc.GetTypingUsers(ctx, "channel-3")

	assert.Contains(t, users1, "user-1")
	assert.NotContains(t, users2, "user-1")
	assert.Contains(t, users3, "user-1")
}
