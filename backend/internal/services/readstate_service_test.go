package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadStateService_MarkAsRead(t *testing.T) {
	svc := NewReadStateService()
	ctx := context.Background()

	before := time.Now()
	err := svc.MarkAsRead(ctx, "channel-1", "user-1")
	assert.NoError(t, err)

	lastRead, err := svc.GetLastRead(ctx, "channel-1", "user-1")
	assert.NoError(t, err)
	assert.True(t, lastRead.After(before) || lastRead.Equal(before))
}
