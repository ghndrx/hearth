package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEmojiService_Create(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	e, err := svc.Create(ctx, serverID, "pepe", "/emojis/pepe.png", false)
	assert.NoError(t, err)
	assert.Equal(t, "pepe", e.Name)

	emojis, _ := svc.GetServerEmojis(ctx, serverID)
	assert.Len(t, emojis, 1)
}

func TestEmojiService_Delete(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	e, _ := svc.Create(ctx, serverID, "kek", "/emojis/kek.png", false)
	err := svc.Delete(ctx, e.ID)
	assert.NoError(t, err)

	emojis, _ := svc.GetServerEmojis(ctx, serverID)
	assert.Len(t, emojis, 0)
}

func TestEmojiService_Delete_NotFound(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()

	err := svc.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrEmojiNotFound)
}
