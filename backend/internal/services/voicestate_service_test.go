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
	err := svc.SetMuted(ctx, userID, true)
	assert.NoError(t, err)

	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Muted)
}

func TestVoiceStateService_Mute_NotInVoice(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	err := svc.SetMuted(ctx, uuid.New(), true)
	assert.ErrorIs(t, err, ErrUserNotInVoice)
}

func TestVoiceStateService_Deafen(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID := uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, uuid.New())
	err := svc.SetDeafened(ctx, userID, true)
	assert.NoError(t, err)

	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Deafened)
}

func TestVoiceStateService_Deafen_NotInVoice(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	err := svc.SetDeafened(ctx, uuid.New(), true)
	assert.ErrorIs(t, err, ErrUserNotInVoice)
}

func TestVoiceStateService_SetVideo(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, serverID)

	err := svc.SetVideo(ctx, userID, true)
	assert.NoError(t, err)

	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Video)
}

func TestVoiceStateService_SetVideo_NotInVoice(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	err := svc.SetVideo(ctx, uuid.New(), true)
	assert.ErrorIs(t, err, ErrUserNotInVoice)
}

func TestVoiceStateService_SetStreaming(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, serverID)

	err := svc.SetStreaming(ctx, userID, true)
	assert.NoError(t, err)

	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Streaming)
}

func TestVoiceStateService_SetStreaming_NotInVoice(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	err := svc.SetStreaming(ctx, uuid.New(), true)
	assert.ErrorIs(t, err, ErrUserNotInVoice)
}

func TestVoiceStateService_VoiceStateHasVideoAndStreaming(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, serverID)

	// Initially, video and streaming should be false
	users, _ := svc.GetChannelUsers(ctx, channelID)
	assert.False(t, users[0].Video)
	assert.False(t, users[0].Streaming)

	// Set video
	err := svc.SetVideo(ctx, userID, true)
	assert.NoError(t, err)

	// Set streaming
	err = svc.SetStreaming(ctx, userID, true)
	assert.NoError(t, err)

	users, _ = svc.GetChannelUsers(ctx, channelID)
	assert.True(t, users[0].Video)
	assert.True(t, users[0].Streaming)

	// Turn off video
	err = svc.SetVideo(ctx, userID, false)
	assert.NoError(t, err)

	users, _ = svc.GetChannelUsers(ctx, channelID)
	assert.False(t, users[0].Video)
	assert.True(t, users[0].Streaming)
}
