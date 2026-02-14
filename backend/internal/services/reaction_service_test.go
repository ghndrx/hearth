package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReactionService_AddAndGet(t *testing.T) {
	svc := NewReactionService()
	ctx := context.Background()

	err := svc.AddReaction(ctx, "msg-1", "user-1", "👍")
	assert.NoError(t, err)

	reactions, _ := svc.GetReactions(ctx, "msg-1")
	assert.Len(t, reactions, 1)
	assert.Equal(t, "👍", reactions[0].Emoji)
}

func TestReactionService_Remove(t *testing.T) {
	svc := NewReactionService()
	ctx := context.Background()

	svc.AddReaction(ctx, "msg-1", "user-1", "👍")
	svc.RemoveReaction(ctx, "msg-1", "user-1", "👍")

	reactions, _ := svc.GetReactions(ctx, "msg-1")
	assert.Len(t, reactions, 0)
}
