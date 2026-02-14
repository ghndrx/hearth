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
