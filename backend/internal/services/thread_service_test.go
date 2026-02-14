package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestThreadService_CreateAndGet(t *testing.T) {
	svc := NewThreadService()
	ctx := context.Background()

	thread, err := svc.CreateThread(ctx, uuid.New(), uuid.New(), "Test Thread")
	assert.NoError(t, err)
	assert.Equal(t, "Test Thread", thread.Name)

	found, err := svc.GetThread(ctx, thread.ID)
	assert.NoError(t, err)
	assert.Equal(t, thread.ID, found.ID)
}

func TestThreadService_Archive(t *testing.T) {
	svc := NewThreadService()
	ctx := context.Background()

	thread, _ := svc.CreateThread(ctx, uuid.New(), uuid.New(), "Test")
	err := svc.ArchiveThread(ctx, thread.ID)
	assert.NoError(t, err)

	found, _ := svc.GetThread(ctx, thread.ID)
	assert.True(t, found.Archived)
}
