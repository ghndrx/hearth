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

func TestVoiceStateService_GetUserState(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	// User not in voice
	state, err := svc.GetUserState(ctx, userID)
	assert.ErrorIs(t, err, ErrUserNotInVoice)
	assert.Nil(t, state)

	// Join and get state
	svc.Join(ctx, userID, channelID, serverID)
	state, err = svc.GetUserState(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, userID, state.UserID)
	assert.Equal(t, channelID, state.ChannelID)
	assert.Equal(t, serverID, state.ServerID)
	assert.False(t, state.Muted)
	assert.False(t, state.Deafened)
}

func TestVoiceStateService_GetUserState_WithMuteDeafen(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	userID, channelID, serverID := uuid.New(), uuid.New(), uuid.New()

	svc.Join(ctx, userID, channelID, serverID)
	svc.SetMuted(ctx, userID, true)
	svc.SetDeafened(ctx, userID, true)

	state, err := svc.GetUserState(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, state.Muted)
	assert.True(t, state.Deafened)
}

func TestVoiceStateService_GetServerUsers(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()
	serverID := uuid.New()
	channelID1 := uuid.New()
	channelID2 := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	otherServer := uuid.New()

	// Join users to different channels in same server
	svc.Join(ctx, user1, channelID1, serverID)
	svc.Join(ctx, user2, channelID1, serverID)
	svc.Join(ctx, user3, channelID2, serverID)

	// Add user to different server
	user4 := uuid.New()
	svc.Join(ctx, user4, uuid.New(), otherServer)

	// Get all users in server
	users, err := svc.GetServerUsers(ctx, serverID)
	assert.NoError(t, err)
	assert.Len(t, users, 3)

	// Verify user IDs
	userIDs := make(map[uuid.UUID]bool)
	for _, u := range users {
		userIDs[u.UserID] = true
	}
	assert.True(t, userIDs[user1])
	assert.True(t, userIDs[user2])
	assert.True(t, userIDs[user3])
	assert.False(t, userIDs[user4]) // Different server
}

func TestVoiceStateService_GetServerUsers_Empty(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	users, err := svc.GetServerUsers(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Empty(t, users)
}

func TestVoiceStateService_GetChannelUsers_Empty(t *testing.T) {
	svc := NewVoiceStateService()
	ctx := context.Background()

	users, err := svc.GetChannelUsers(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Empty(t, users)
}
