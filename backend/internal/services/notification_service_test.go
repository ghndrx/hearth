package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNotificationService_CreateAndGetUnread(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	userID := uuid.New()

	n, err := svc.Create(ctx, userID, "mention", "New mention", "Someone mentioned you")
	assert.NoError(t, err)
	assert.Equal(t, "mention", n.Type)

	unread, _ := svc.GetUnread(ctx, userID)
	assert.Len(t, unread, 1)
}

func TestNotificationService_MarkRead(t *testing.T) {
	svc := NewNotificationService()
	ctx := context.Background()
	userID := uuid.New()

	n, _ := svc.Create(ctx, userID, "mention", "Test", "Body")
	svc.MarkRead(ctx, n.ID)

	unread, _ := svc.GetUnread(ctx, userID)
	assert.Len(t, unread, 0)
}
