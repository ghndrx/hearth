package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVideoServiceHelper(t *testing.T) (*VideoSignalingService, *Hub, context.CancelFunc) {
	t.Helper()
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)
	vs := NewVideoSignalingService(hub)
	return vs, hub, cancel
}

func newVideoRegisteredClientHelper(t *testing.T, hub *Hub, userID uuid.UUID) *Client {
	t.Helper()
	client := newMockClient(hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)
	return client
}

func TestVideoSignaling_HandleVideoMessage_Routing(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	client := newVideoRegisteredClientHelper(t, hub, uuid.New())
	ctx := context.Background()

	t.Run("unknown message type returns nil", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", "UNKNOWN_TYPE", nil)
		assert.NoError(t, err)
	})

	t.Run("ring with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalRing, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("ring response with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalRingResponse, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("offer with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalOffer, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("answer with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalAnswer, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("ice candidate with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalICECandidate, json.RawMessage("invalid"))
		assert.Error(t, err)
	})

	t.Run("state update with invalid JSON returns error", func(t *testing.T) {
		err := vs.HandleVideoMessage(ctx, client, "session-1", VideoSignalStateUpdate, json.RawMessage("invalid"))
		assert.Error(t, err)
	})
}

func TestVideoSignaling_RingInitiation(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	callerID := uuid.New()
	targetID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	callerClient := newVideoRegisteredClientHelper(t, hub, callerID)
	_ = newVideoRegisteredClientHelper(t, hub, targetID) // target

	ringData, _ := json.Marshal(VideoCallData{
		ChannelID: channelID,
		ServerID:  serverID,
		CallType:  VideoCallTypeDirect,
		ToUserID:  targetID,
	})

	err := vs.HandleVideoMessage(context.Background(), callerClient, "session-1", VideoSignalRing, ringData)
	require.NoError(t, err)

	// Verify call was created
	assert.Equal(t, 1, len(vs.calls))

	// Verify pending ring was set for target
	pendingCallID := vs.GetPendingRing(targetID)
	assert.NotEqual(t, uuid.Nil, pendingCallID)

	// Verify call state
	call := vs.GetCall(pendingCallID)
	require.NotNil(t, call)
	assert.Equal(t, VideoStateRingingOut, call.State)
	assert.Equal(t, callerID, call.InitiatorID)
	assert.Contains(t, call.Participants, callerID)
}

func TestVideoSignaling_RingAccept(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	callerID := uuid.New()
	targetID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	callerClient := newVideoRegisteredClientHelper(t, hub, callerID)
	targetClient := newVideoRegisteredClientHelper(t, hub, targetID)

	// Create a call first
	now := time.Now()
	callID := uuid.New()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   channelID,
		ServerID:    serverID,
		CallType:    VideoCallTypeDirect,
		State:       VideoStateRingingOut,
		InitiatorID: callerID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			callerID: {
				UserID:   callerID,
				Username: "caller",
				JoinedAt: now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.pendingRings[targetID] = callID
	vs.userCalls[callerID] = callID

	responseData, _ := json.Marshal(VideoRingResponseData{
		CallID:   callID,
		Accept:   true,
		ToUserID: callerID,
	})

	err := vs.HandleVideoMessage(context.Background(), targetClient, "session-2", VideoSignalRingResponse, responseData)
	require.NoError(t, err)

	// Verify call state changed to connecting
	updatedCall := vs.GetCall(callID)
	require.NotNil(t, updatedCall)
	assert.Equal(t, VideoStateConnecting, updatedCall.State)

	// Verify target was added as participant
	assert.Contains(t, updatedCall.Participants, targetID)

	// Verify pending ring was cleared
	assert.Equal(t, uuid.Nil, vs.GetPendingRing(targetID))

	// Verify user's call mapping
	assert.Equal(t, callID, vs.GetUserCall(targetID))

	_ = callerClient // used in setup
}

func TestVideoSignaling_RingDecline(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	callerID := uuid.New()
	targetID := uuid.New()

	_ = newVideoRegisteredClientHelper(t, hub, callerID)
	targetClient := newVideoRegisteredClientHelper(t, hub, targetID)

	// Create a call first
	now := time.Now()
	callID := uuid.New()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   uuid.New(),
		ServerID:    uuid.New(),
		CallType:    VideoCallTypeDirect,
		State:       VideoStateRingingOut,
		InitiatorID: callerID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			callerID: {
				UserID:   callerID,
				Username: "caller",
				JoinedAt: now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.pendingRings[targetID] = callID

	responseData, _ := json.Marshal(VideoRingResponseData{
		CallID:   callID,
		Accept:   false,
		ToUserID: callerID,
	})

	err := vs.HandleVideoMessage(context.Background(), targetClient, "session-2", VideoSignalRingResponse, responseData)
	require.NoError(t, err)

	// Verify call was cleaned up
	assert.Equal(t, 0, len(vs.calls))

	// Verify pending ring was cleared
	assert.Equal(t, uuid.Nil, vs.GetPendingRing(targetID))
}

func TestVideoSignaling_Leave(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	user1ID := uuid.New()
	user2ID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	callID := uuid.New()

	user1Client := newVideoRegisteredClientHelper(t, hub, user1ID)
	user2Client := newVideoRegisteredClientHelper(t, hub, user2ID)
	_ = user2Client

	now := time.Now()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   channelID,
		ServerID:    serverID,
		CallType:    VideoCallTypeDirect,
		State:       VideoStateConnected,
		InitiatorID: user1ID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			user1ID: {
				UserID:   user1ID,
				Username: "user1",
				JoinedAt: now,
			},
			user2ID: {
				UserID:   user2ID,
				Username: "user2",
				JoinedAt: now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.userCalls[user1ID] = callID
	vs.userCalls[user2ID] = callID

	leaveData, _ := json.Marshal(VideoLeaveData{
		CallID:    callID,
		ChannelID: channelID,
		ServerID:  serverID,
	})

	err := vs.HandleVideoMessage(context.Background(), user1Client, "session-1", VideoSignalLeave, leaveData)
	require.NoError(t, err)

	// Verify user1 was removed from call
	updatedCall := vs.GetCall(callID)
	require.NotNil(t, updatedCall)
	assert.NotContains(t, updatedCall.Participants, user1ID)

	// Verify user1's call mapping was cleared
	assert.Equal(t, uuid.Nil, vs.GetUserCall(user1ID))

	// User2 should still be in call
	assert.Contains(t, updatedCall.Participants, user2ID)
}

func TestVideoSignaling_StateUpdate(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	userID := uuid.New()
	callID := uuid.New()

	userClient := newVideoRegisteredClientHelper(t, hub, userID)

	now := time.Now()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   uuid.New(),
		ServerID:    uuid.New(),
		CallType:    VideoCallTypeDirect,
		State:       VideoStateConnected,
		InitiatorID: userID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			userID: {
				UserID:        userID,
				Username:      "user",
				IsCameraOn:    false,
				IsMuted:       true,
				IsScreenShare: false,
				JoinedAt:      now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.userCalls[userID] = callID

	stateData, _ := json.Marshal(VideoStateUpdateData{
		CallID:        callID,
		IsCameraOn:    true,
		IsMuted:       false,
		IsScreenShare: false,
	})

	err := vs.HandleVideoMessage(context.Background(), userClient, "session-1", VideoSignalStateUpdate, stateData)
	require.NoError(t, err)

	// Verify state was updated
	updatedCall := vs.GetCall(callID)
	require.NotNil(t, updatedCall)
	participant := updatedCall.Participants[userID]
	assert.True(t, participant.IsCameraOn)
	assert.False(t, participant.IsMuted)
}

func TestVideoSignaling_ScreenShare(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	userID := uuid.New()
	callID := uuid.New()

	userClient := newVideoRegisteredClientHelper(t, hub, userID)

	now := time.Now()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   uuid.New(),
		ServerID:    uuid.New(),
		CallType:    VideoCallTypeDirect,
		State:       VideoStateConnected,
		InitiatorID: userID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			userID: {
				UserID:        userID,
				Username:      "user",
				IsScreenShare: false,
				JoinedAt:      now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.userCalls[userID] = callID

	// Start screen share
	startData, _ := json.Marshal(VideoStateUpdateData{
		CallID:        callID,
		IsScreenShare: true,
	})

	err := vs.HandleVideoMessage(context.Background(), userClient, "session-1", VideoSignalScreenStart, startData)
	require.NoError(t, err)

	// Verify screen share started
	updatedCall := vs.GetCall(callID)
	require.NotNil(t, updatedCall)
	assert.True(t, updatedCall.Participants[userID].IsScreenShare)

	// Stop screen share
	stopData, _ := json.Marshal(VideoStateUpdateData{
		CallID:        callID,
		IsScreenShare: false,
	})

	err = vs.HandleVideoMessage(context.Background(), userClient, "session-1", VideoSignalScreenStop, stopData)
	require.NoError(t, err)

	// Verify screen share stopped
	updatedCall = vs.GetCall(callID)
	require.NotNil(t, updatedCall)
	assert.False(t, updatedCall.Participants[userID].IsScreenShare)
}

func TestVideoSignaling_OfferAnswer(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	callerID := uuid.New()
	targetID := uuid.New()
	callID := uuid.New()

	callerClient := newVideoRegisteredClientHelper(t, hub, callerID)
	targetClient := newVideoRegisteredClientHelper(t, hub, targetID)

	// Simulate offer from caller to target
	offerData, _ := json.Marshal(VideoOfferData{
		CallID:   callID,
		SDP:      "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n",
		ToUserID: targetID,
	})

	err := vs.HandleVideoMessage(context.Background(), callerClient, "session-1", VideoSignalOffer, offerData)
	require.NoError(t, err)

	// Verify offer was received by target
	select {
	case msg := <-targetClient.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeVideoOffer, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for offer message")
	}

	// Simulate answer from target to caller
	answerData, _ := json.Marshal(VideoAnswerData{
		CallID:   callID,
		SDP:      "v=0\r\no=- 1 1 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n",
		ToUserID: callerID,
	})

	err = vs.HandleVideoMessage(context.Background(), targetClient, "session-2", VideoSignalAnswer, answerData)
	require.NoError(t, err)

	// Verify answer was received by caller
	select {
	case msg := <-callerClient.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeVideoAnswer, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for answer message")
	}
}

func TestVideoSignaling_ICECandidate(t *testing.T) {
	vs, hub, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	callerID := uuid.New()
	targetID := uuid.New()
	callID := uuid.New()

	callerClient := newVideoRegisteredClientHelper(t, hub, callerID)
	targetClient := newVideoRegisteredClientHelper(t, hub, targetID)

	candidateData, _ := json.Marshal(VideoICECandidateData{
		CallID:        callID,
		Candidate:     "candidate:1 1 UDP 2113661178 192.168.1.1 12345 typ host",
		SDPMid:        "0",
		SDPMLineIndex: 0,
		ToUserID:      targetID,
	})

	err := vs.HandleVideoMessage(context.Background(), callerClient, "session-1", VideoSignalICECandidate, candidateData)
	require.NoError(t, err)

	// Verify candidate was received by target
	select {
	case msg := <-targetClient.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeVideoICECandidate, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ICE candidate message")
	}
}

func TestVideoSignaling_CleanupCall(t *testing.T) {
	vs, _, cancel := newTestVideoServiceHelper(t)
	defer cancel()

	userID := uuid.New()
	callID := uuid.New()

	now := time.Now()
	call := &VideoCallInfo{
		CallID:      callID,
		ChannelID:   uuid.New(),
		ServerID:    uuid.New(),
		CallType:    VideoCallTypeDirect,
		State:       VideoStateConnected,
		InitiatorID: userID,
		StartedAt:   now,
		Participants: map[uuid.UUID]*VideoParticipantData{
			userID: {
				UserID:   userID,
				Username: "user",
				JoinedAt: now,
			},
		},
		peers: make(map[uuid.UUID]*Client),
	}
	vs.calls[callID] = call
	vs.userCalls[userID] = callID

	vs.cleanupCall(callID)

	// Verify call was removed
	assert.Equal(t, 0, len(vs.calls))

	// Verify user call mapping was cleared
	assert.Equal(t, uuid.Nil, vs.GetUserCall(userID))
}
