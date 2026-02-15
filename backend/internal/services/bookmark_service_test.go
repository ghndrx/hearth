package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBookmarkService_AddAndList(t *testing.T) {
	svc := NewBookmarkService()
	ctx := context.Background()
	userID := uuid.New()

	b, err := svc.Add(ctx, userID, uuid.New(), "Important message")
	assert.NoError(t, err)
	assert.Equal(t, "Important message", b.Note)

	list, _ := svc.List(ctx, userID)
	assert.Len(t, list, 1)
}

func TestBookmarkService_Remove(t *testing.T) {
	svc := NewBookmarkService()
	ctx := context.Background()
	userID := uuid.New()

	b, _ := svc.Add(ctx, userID, uuid.New(), "Test")
	svc.Remove(ctx, userID, b.ID)

	list, _ := svc.List(ctx, userID)
	assert.Len(t, list, 0)
}
