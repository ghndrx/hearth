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

func TestEmojiService_Get(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	e, _ := svc.Create(ctx, serverID, "pepe", "/emojis/pepe.png", false)

	got, err := svc.Get(ctx, e.ID)
	assert.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, "pepe", got.Name)
}

func TestEmojiService_Get_NotFound(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()

	_, err := svc.Get(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrEmojiNotFound)
}

func TestEmojiService_Update(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	e, _ := svc.Create(ctx, serverID, "pepe", "/emojis/pepe.png", false)

	updated, err := svc.Update(ctx, e.ID, "newpepe")
	assert.NoError(t, err)
	assert.Equal(t, "newpepe", updated.Name)
	assert.Equal(t, e.ID, updated.ID)
}

func TestEmojiService_Update_NotFound(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()

	_, err := svc.Update(ctx, uuid.New(), "nonexistent")
	assert.ErrorIs(t, err, ErrEmojiNotFound)
}

func TestEmojiService_GetServerEmojis(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()
	otherServerID := uuid.New()

	svc.Create(ctx, serverID, "pepe", "/emojis/pepe.png", false)
	svc.Create(ctx, serverID, "kek", "/emojis/kek.png", false)
	svc.Create(ctx, otherServerID, "other", "/emojis/other.png", false)

	emojis, err := svc.GetServerEmojis(ctx, serverID)
	assert.NoError(t, err)
	assert.Len(t, emojis, 2)
}
