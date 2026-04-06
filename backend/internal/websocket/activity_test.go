package websocket

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivitySignalConstants(t *testing.T) {
	// Client -> Server signal types
	assert.Equal(t, "ACTIVITY_START", ActivitySignalStart)
	assert.Equal(t, "ACTIVITY_JOIN", ActivitySignalJoin)
	assert.Equal(t, "ACTIVITY_LEAVE", ActivitySignalLeave)
	assert.Equal(t, "ACTIVITY_END", ActivitySignalEnd)
	assert.Equal(t, "ACTIVITY_GAME_MOVE", ActivitySignalGameMove)
	assert.Equal(t, "ACTIVITY_SYNC", ActivitySignalSync)

	// Server -> Client event types
	assert.Equal(t, "ACTIVITY_START", EventActivityStart)
	assert.Equal(t, "ACTIVITY_PARTICIPANT_JOIN", EventActivityJoin)
	assert.Equal(t, "ACTIVITY_PARTICIPANT_LEAVE", EventActivityLeave)
	assert.Equal(t, "ACTIVITY_END", EventActivityEnd)
	assert.Equal(t, "ACTIVITY_STATE_SYNC", EventActivityStateSync)
	assert.Equal(t, "ACTIVITY_GAME_MOVE", EventActivityGameMove)
}

func TestActivityStartData_Serialization(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()

	data := ActivityStartData{
		ChannelID:       channelID,
		ServerID:       serverID,
		ActivityType:   "chess",
		MaxParticipants: 2,
		Metadata:       json.RawMessage(`{"time_control":"blitz"}`),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivityStartData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, channelID, decoded.ChannelID)
	assert.Equal(t, serverID, decoded.ServerID)
	assert.Equal(t, "chess", decoded.ActivityType)
	assert.Equal(t, 2, decoded.MaxParticipants)
	assert.Equal(t, `{"time_control":"blitz"}`, string(decoded.Metadata))
}

func TestActivityJoinData_Serialization(t *testing.T) {
	activityID := uuid.New()

	data := ActivityJoinData{
		ActivityID: activityID,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivityJoinData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, activityID, decoded.ActivityID)
}

func TestActivityLeaveData_Serialization(t *testing.T) {
	activityID := uuid.New()

	data := ActivityLeaveData{
		ActivityID: activityID,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivityLeaveData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, activityID, decoded.ActivityID)
}

func TestActivityEndData_Serialization(t *testing.T) {
	activityID := uuid.New()

	data := ActivityEndData{
		ActivityID: activityID,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivityEndData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, activityID, decoded.ActivityID)
}

func TestActivityGameMoveData_Serialization(t *testing.T) {
	activityID := uuid.New()

	data := ActivityGameMoveData{
		ActivityID: activityID,
		Action:     "move",
		Data:       json.RawMessage(`{"from":"e2","to":"e4"}`),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivityGameMoveData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, activityID, decoded.ActivityID)
	assert.Equal(t, "move", decoded.Action)
	assert.Equal(t, `{"from":"e2","to":"e4"}`, string(decoded.Data))
}

func TestActivitySyncData_Serialization(t *testing.T) {
	activityID := uuid.New()

	data := ActivitySyncData{
		ActivityID: activityID,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded ActivitySyncData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, activityID, decoded.ActivityID)
}

func TestNewActivitySignalingService(t *testing.T) {
	hub := NewHub()
	activitySvc := NewActivitySignalingService(hub, nil)

	require.NotNil(t, activitySvc)
	assert.NotNil(t, activitySvc.activeActivities)
	assert.Equal(t, hub, activitySvc.hub)
	assert.Nil(t, activitySvc.activityService)
}

// Note: Handler integration tests (handleStart, handleJoin, handleLeave,
// handleEnd, handleGameMove, handleSync) require a non-nil activityService
// or a mock VoiceActivityService. These are tested via integration tests
// in a separate test suite with proper dependency injection.

func TestHandleActivityMessage_Routing(t *testing.T) {
	hub := NewHub()
	as := NewActivitySignalingService(hub, nil)

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      nil, // not used in routing tests
		send:     make(chan []byte, 256),
	}

	t.Run("unknown message type returns nil (no error)", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", "UNKNOWN_TYPE", nil)
		assert.NoError(t, err)
	})

	t.Run("start with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalStart, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("join with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalJoin, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("leave with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalLeave, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("end with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalEnd, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("game move with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalGameMove, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("sync with invalid JSON returns error", func(t *testing.T) {
		err := as.HandleActivityMessage(nil, client, "session-1", ActivitySignalSync, json.RawMessage("invalid"))
		assert.Error(t, err)
	})
}

func TestActivitySignalingService_ActiveActivitiesTracking(t *testing.T) {
	hub := NewHub()
	as := NewActivitySignalingService(hub, nil)

	channelID1 := uuid.New()
	channelID2 := uuid.New()
	activityID1 := uuid.New()
	activityID2 := uuid.New()

	// Test direct map manipulation (simulates what handlers do)
	as.activeActivitiesMu.Lock()
	as.activeActivities[channelID1] = activityID1
	as.activeActivities[channelID2] = activityID2
	as.activeActivitiesMu.Unlock()

	as.activeActivitiesMu.RLock()
	assert.Equal(t, 2, len(as.activeActivities))
	as.activeActivitiesMu.RUnlock()

	// Remove one
	as.activeActivitiesMu.Lock()
	delete(as.activeActivities, channelID1)
	as.activeActivitiesMu.Unlock()

	as.activeActivitiesMu.RLock()
	assert.Equal(t, 1, len(as.activeActivities))
	trackedID, ok := as.activeActivities[channelID2]
	assert.True(t, ok)
	assert.Equal(t, activityID2, trackedID)
	as.activeActivitiesMu.RUnlock()
}
