package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestVoiceStateService_JoinAndLeave(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, serverID)
	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.Len(t, users, 1)

	svc.Leave(ctx, userID)
	users, _ = svc.GetChannelUsers(ctx, channelID)
	assert.Len(t, users, 0)
}

func TestVoiceStateService_Mute(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID := uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, uuid.New())
	svc.SetMuted(ctx, userID, true)

	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Muted)
}
