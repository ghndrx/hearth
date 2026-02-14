package services

import (
	"context"
	"testing"

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
