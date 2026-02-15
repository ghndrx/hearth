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

func TestEmojiService_Create_MultipleEmojis(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	// Create multiple emojis
	e1, err := svc.Create(ctx, serverID, "pepe", "/emojis/pepe.png", false)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, e1.ID)

	e2, err := svc.Create(ctx, serverID, "sadge", "/emojis/sadge.png", false)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, e2.ID)

	e3, err := svc.Create(ctx, serverID, "monka", "/emojis/monka.gif", true)
	assert.NoError(t, err)
	assert.True(t, e3.Animated)

	emojis, err := svc.GetServerEmojis(ctx, serverID)
	assert.NoError(t, err)
	assert.Len(t, emojis, 3)
}

func TestEmojiService_Create_DifferentServers(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	server1 := uuid.New()
	server2 := uuid.New()

	// Create emojis for different servers
	_, err := svc.Create(ctx, server1, "server1emoji", "/emojis/s1.png", false)
	assert.NoError(t, err)

	_, err = svc.Create(ctx, server2, "server2emoji", "/emojis/s2.png", false)
	assert.NoError(t, err)

	// Verify separation
	server1Emojis, _ := svc.GetServerEmojis(ctx, server1)
	assert.Len(t, server1Emojis, 1)
	assert.Equal(t, "server1emoji", server1Emojis[0].Name)

	server2Emojis, _ := svc.GetServerEmojis(ctx, server2)
	assert.Len(t, server2Emojis, 1)
	assert.Equal(t, "server2emoji", server2Emojis[0].Name)
}

func TestEmojiService_GetServerEmojis_Empty(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	emojis, err := svc.GetServerEmojis(ctx, serverID)
	assert.NoError(t, err)
	assert.Empty(t, emojis)
}

func TestEmojiService_GetServerEmojis_NonExistentServer(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()

	emojis, err := svc.GetServerEmojis(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Empty(t, emojis)
}

func TestEmojiService_Delete(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	// Create emoji
	e, _ := svc.Create(ctx, serverID, "toDelete", "/emojis/delete.png", false)

	// Verify it exists
	emojis, _ := svc.GetServerEmojis(ctx, serverID)
	assert.Len(t, emojis, 1)

	// Delete it
	err := svc.Delete(ctx, e.ID)
	assert.NoError(t, err)

	// Verify it's gone
	emojis, _ = svc.GetServerEmojis(ctx, serverID)
	assert.Len(t, emojis, 0)
}

func TestEmojiService_Delete_NonExistent(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()

	// Deleting non-existent emoji should not error
	err := svc.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestEmojiService_Delete_OnlyTargetEmoji(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	// Create multiple emojis
	e1, _ := svc.Create(ctx, serverID, "keep", "/emojis/keep.png", false)
	e2, _ := svc.Create(ctx, serverID, "delete", "/emojis/delete.png", false)

	// Delete only one
	err := svc.Delete(ctx, e2.ID)
	assert.NoError(t, err)

	// Verify only target is deleted
	emojis, _ := svc.GetServerEmojis(ctx, serverID)
	assert.Len(t, emojis, 1)
	assert.Equal(t, e1.ID, emojis[0].ID)
	assert.Equal(t, "keep", emojis[0].Name)
}

func TestEmojiService_ConcurrentAccess(t *testing.T) {
	svc := NewEmojiService()
	ctx := context.Background()
	serverID := uuid.New()

	// Create emoji
	e, _ := svc.Create(ctx, serverID, "concurrent", "/emojis/concurrent.png", false)

	// Concurrent reads and delete
	done := make(chan bool, 3)

	// Reader 1
	go func() {
		svc.GetServerEmojis(ctx, serverID)
		done <- true
	}()

	// Reader 2
	go func() {
		svc.GetServerEmojis(ctx, serverID)
		done <- true
	}()

	// Deleter
	go func() {
		svc.Delete(ctx, e.ID)
		done <- true
	}()

	// Wait for all
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not panic and should have consistent state
	emojis, _ := svc.GetServerEmojis(ctx, serverID)
	// After delete, should be 0 or the delete happened after reads
	assert.True(t, len(emojis) == 0 || len(emojis) == 1)
}

func TestCustomEmoji_Fields(t *testing.T) {
	serverID := uuid.New()
	emojiID := uuid.New()

	emoji := &CustomEmoji{
		ID:       emojiID,
		ServerID: serverID,
		Name:     "testemoji",
		URL:      "/emojis/test.png",
		Animated: true,
	}

	assert.Equal(t, emojiID, emoji.ID)
	assert.Equal(t, serverID, emoji.ServerID)
	assert.Equal(t, "testemoji", emoji.Name)
	assert.Equal(t, "/emojis/test.png", emoji.URL)
	assert.True(t, emoji.Animated)
}

func TestErrEmojiNotFound(t *testing.T) {
	assert.Equal(t, "emoji not found", ErrEmojiNotFound.Error())
}
