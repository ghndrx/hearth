package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestModerationService_Warn(t *testing.T) {
	svc := NewModerationService()
	ctx := context.Background()

	action, err := svc.Warn(ctx, uuid.New(), uuid.New(), uuid.New(), "Spam")
	assert.NoError(t, err)
	assert.Equal(t, "warn", action.Action)
}

func TestModerationService_Timeout(t *testing.T) {
	svc := NewModerationService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	action, _ := svc.Timeout(ctx, serverID, userID, uuid.New(), time.Hour, "Harassment")
	assert.Equal(t, "timeout", action.Action)
	assert.NotNil(t, action.ExpiresAt)

	history, _ := svc.GetHistory(ctx, serverID, userID)
	assert.Len(t, history, 1)
}
