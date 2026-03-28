package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/metrics"
	"hearth/internal/pubsub"
)

func newTestGateway(t *testing.T) (*Gateway, *Hub, context.CancelFunc) {
	t.Helper()
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	return gateway, hub, cancel
}

func newTestClient(t *testing.T, hub *Hub, userID uuid.UUID) *Client {
	t.Helper()
	return &Client{
		ID:            uuid.New().String(),
		UserID:        userID,
		Username:      "testuser",
		hub:           hub,
		send:          make(chan []byte, 256),
		servers:       make(map[uuid.UUID]bool),
		channels:      make(map[uuid.UUID]bool),
		lastHeartbeat: time.Now(),
	}
}

func TestGateway_HandleSubscribe_Channel(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{
		ID:     "session-1",
		UserID: userID,
	}

	channelID := uuid.New()
	data, _ := json.Marshal(map[string]string{
		"channel_id": channelID.String(),
	})

	gateway.handleSubscribe(nil, client, session, data)

	assert.True(t, client.IsSubscribedToChannel(channelID))
}

func TestGateway_HandleSubscribe_Server(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{
		ID:     "session-1",
		UserID: userID,
	}

	serverID := uuid.New()
	data, _ := json.Marshal(map[string]string{
		"server_id": serverID.String(),
	})

	gateway.handleSubscribe(nil, client, session, data)

	assert.True(t, client.IsSubscribedToServer(serverID))
}

func TestGateway_HandleSubscribe_Both(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	channelID := uuid.New()
	serverID := uuid.New()
	data, _ := json.Marshal(map[string]string{
		"channel_id": channelID.String(),
		"server_id":  serverID.String(),
	})

	gateway.handleSubscribe(nil, client, session, data)

	assert.True(t, client.IsSubscribedToChannel(channelID))
	assert.True(t, client.IsSubscribedToServer(serverID))
}

func TestGateway_HandleSubscribe_InvalidChannelID(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	data, _ := json.Marshal(map[string]string{
		"channel_id": "not-a-uuid",
	})

	// Should not panic
	gateway.handleSubscribe(nil, client, session, data)
}

func TestGateway_HandleSubscribe_InvalidServerID(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	data, _ := json.Marshal(map[string]string{
		"server_id": "not-a-uuid",
	})

	gateway.handleSubscribe(nil, client, session, data)
}

func TestGateway_HandleSubscribe_InvalidJSON(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	gateway.handleSubscribe(nil, client, session, json.RawMessage("invalid"))
}

func TestGateway_HandleUnsubscribe_Channel(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	channelID := uuid.New()
	client.SubscribeChannel(channelID)
	assert.True(t, client.IsSubscribedToChannel(channelID))

	data, _ := json.Marshal(map[string]string{
		"channel_id": channelID.String(),
	})

	gateway.handleUnsubscribe(nil, client, session, data)

	assert.False(t, client.IsSubscribedToChannel(channelID))
}

func TestGateway_HandleUnsubscribe_Server(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	serverID := uuid.New()
	client.SubscribeServer(serverID)
	assert.True(t, client.IsSubscribedToServer(serverID))

	data, _ := json.Marshal(map[string]string{
		"server_id": serverID.String(),
	})

	gateway.handleUnsubscribe(nil, client, session, data)

	assert.False(t, client.IsSubscribedToServer(serverID))
}

func TestGateway_HandleUnsubscribe_InvalidJSON(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	gateway.handleUnsubscribe(nil, client, session, json.RawMessage("invalid"))
}

func TestGateway_HandleUnsubscribe_InvalidChannelID(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	data, _ := json.Marshal(map[string]string{
		"channel_id": "not-a-uuid",
	})

	gateway.handleUnsubscribe(nil, client, session, data)
}

func TestGateway_HandleUnsubscribe_InvalidServerID(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	data, _ := json.Marshal(map[string]string{
		"server_id": "not-a-uuid",
	})

	gateway.handleUnsubscribe(nil, client, session, data)
}

func TestGateway_HandlePresenceUpdate(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	// Subscribe to a server so presence updates go somewhere
	serverID := uuid.New()
	client.SubscribeServer(serverID)

	// Create another client subscribed to the same server to receive the presence update
	otherUserID := uuid.New()
	otherClient := newTestClient(t, hub, otherUserID)
	hub.register <- otherClient
	time.Sleep(20 * time.Millisecond)
	otherClient.SubscribeServer(serverID)

	presenceData, _ := json.Marshal(map[string]interface{}{
		"status":     "online",
		"activities": []interface{}{},
		"afk":        false,
	})

	msg := &Message{
		Op:   OpPresenceUpdate,
		Data: presenceData,
	}

	gateway.handlePresenceUpdate(nil, client, session, msg)

	// The other client should receive a presence event via server broadcast
	select {
	case data := <-otherClient.send:
		var event Event
		require.NoError(t, json.Unmarshal(data, &event))
		assert.Equal(t, EventTypePresenceUpdate, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for presence update")
	}
}

func TestGateway_HandlePresenceUpdate_NilData(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpPresenceUpdate}

	// Should not panic with nil data
	gateway.handlePresenceUpdate(nil, client, session, msg)
}

func TestGateway_HandlePresenceUpdate_NoServers(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	presenceData, _ := json.Marshal(map[string]interface{}{
		"status": "idle",
	})

	msg := &Message{Op: OpPresenceUpdate, Data: presenceData}

	// Should not panic without server subscriptions
	gateway.handlePresenceUpdate(nil, client, session, msg)
}

func TestGateway_HandlePresenceUpdate_WithCustomStatus(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	serverID := uuid.New()
	client.SubscribeServer(serverID)

	customText := "Working"
	emoji := "💻"
	presenceData, _ := json.Marshal(map[string]interface{}{
		"status":     "dnd",
		"activities": []interface{}{},
		"custom_status": map[string]interface{}{
			"custom_text": customText,
			"emoji":       emoji,
		},
	})

	msg := &Message{Op: OpPresenceUpdate, Data: presenceData}

	gateway.handlePresenceUpdate(nil, client, session, msg)

	// Just verify it doesn't panic - the event is broadcast
}

func TestGateway_HandleVoiceDispatch_NoVoiceService(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	// voiceService is nil by default
	gateway.handleVoiceDispatch(nil, client, session, VoiceSignalJoin, nil)

	// The sendError would try to write to nil conn, which we can't test here
	// But we verify no panic occurs and the code path is exercised
}

func TestGateway_HandleVoiceStateUpdate_NoVoiceService(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpVoiceStateUpdate}

	gateway.handleVoiceStateUpdate(nil, client, session, msg)
}

func TestGateway_HandleVoiceStateUpdate_InvalidServerID(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	// Set up a voice service
	vs := NewVoiceSignalingService(hub, nil)
	gateway.SetVoiceService(vs)

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	// Invalid server_id
	data, _ := json.Marshal(map[string]interface{}{
		"server_id": "not-a-uuid",
	})

	msg := &Message{Op: OpVoiceStateUpdate, Data: data}

	gateway.handleVoiceStateUpdate(nil, client, session, msg)
}

func TestGateway_HandleVoiceStateUpdate_NilData(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	vs := NewVoiceSignalingService(hub, nil)
	gateway.SetVoiceService(vs)

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpVoiceStateUpdate}

	// nil data => empty server_id => parse error
	gateway.handleVoiceStateUpdate(nil, client, session, msg)
}

func TestGateway_HandleVoiceStateUpdate_InvalidJSON(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	vs := NewVoiceSignalingService(hub, nil)
	gateway.SetVoiceService(vs)

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpVoiceStateUpdate, Data: json.RawMessage("invalid")}

	gateway.handleVoiceStateUpdate(nil, client, session, msg)
}

func TestGateway_HandleClientDispatch_Subscribe(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	channelID := uuid.New()
	dispatchData, _ := json.Marshal(map[string]interface{}{
		"t": "SUBSCRIBE",
		"d": map[string]string{
			"channel_id": channelID.String(),
		},
	})

	msg := &Message{Op: OpDispatch, Data: dispatchData}

	gateway.handleClientDispatch(nil, client, session, msg)

	assert.True(t, client.IsSubscribedToChannel(channelID))
}

func TestGateway_HandleClientDispatch_Unsubscribe(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	channelID := uuid.New()
	client.SubscribeChannel(channelID)

	dispatchData, _ := json.Marshal(map[string]interface{}{
		"t": "UNSUBSCRIBE",
		"d": map[string]string{
			"channel_id": channelID.String(),
		},
	})

	msg := &Message{Op: OpDispatch, Data: dispatchData}

	gateway.handleClientDispatch(nil, client, session, msg)

	assert.False(t, client.IsSubscribedToChannel(channelID))
}

func TestGateway_HandleClientDispatch_UnknownType(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	dispatchData, _ := json.Marshal(map[string]interface{}{
		"t": "UNKNOWN_EVENT",
		"d": map[string]string{},
	})

	msg := &Message{Op: OpDispatch, Data: dispatchData}

	// Should not panic
	gateway.handleClientDispatch(nil, client, session, msg)
}

func TestGateway_HandleClientDispatch_InvalidJSON(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpDispatch, Data: json.RawMessage("invalid json")}

	gateway.handleClientDispatch(nil, client, session, msg)
}

func TestGateway_HandleClientDispatch_NilData(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpDispatch}

	gateway.handleClientDispatch(nil, client, session, msg)
}

func TestGateway_HandleClientDispatch_VoiceSignalTypes(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	voiceTypes := []string{
		VoiceSignalJoin, VoiceSignalLeave, VoiceSignalOffer,
		VoiceSignalAnswer, VoiceSignalICECandidate, VoiceSignalSpeaking,
	}

	for _, vt := range voiceTypes {
		t.Run(vt, func(t *testing.T) {
			dispatchData, _ := json.Marshal(map[string]interface{}{
				"t": vt,
				"d": map[string]string{},
			})

			msg := &Message{Op: OpDispatch, Data: dispatchData}

			// voiceService is nil, so handleVoiceDispatch will log "voice not available"
			gateway.handleClientDispatch(nil, client, session, msg)
		})
	}
}

func TestGateway_HandleRequestMembers(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{
		ID:       "session-1",
		UserID:   userID,
		Sequence: 5,
	}

	guildID := uuid.New().String()
	requestData, _ := json.Marshal(map[string]interface{}{
		"guild_id": guildID,
		"query":    "",
		"limit":    100,
		"nonce":    "test-nonce",
	})

	msg := &Message{Op: OpRequestGuildMembers, Data: requestData}

	// Can't test sendMessage with nil conn, but we test the data flow
	// The method will try to write to nil conn and log the error
	gateway.handleRequestMembers(nil, client, session, msg)
}

func TestGateway_HandleRequestMembers_NilData(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg := &Message{Op: OpRequestGuildMembers}

	gateway.handleRequestMembers(nil, client, session, msg)
}

func TestGateway_HandleIdentify(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{
		ID:       "session-1",
		UserID:   userID,
		Username: "testuser",
		Sequence: 1,
	}

	identifyData, _ := json.Marshal(map[string]interface{}{
		"properties": map[string]string{
			"$os":      "linux",
			"$browser": "test",
		},
	})

	msg := &Message{Op: OpIdentify, Data: identifyData}

	// Will fail on sendMessage (nil conn) but exercises the logic
	gateway.handleIdentify(nil, client, session, msg)
}

func TestGateway_HandleIdentify_NilData(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{
		ID:       "session-1",
		UserID:   userID,
		Username: "testuser",
	}

	msg := &Message{Op: OpIdentify}

	gateway.handleIdentify(nil, client, session, msg)
}

func TestGateway_HandleHeartbeat(t *testing.T) {
	gateway, _, cancel := newTestGateway(t)
	defer cancel()

	session := &Session{
		ID:            "session-1",
		UserID:        uuid.New(),
		LastHeartbeat: time.Now().Add(-time.Hour),
	}

	oldHeartbeat := session.LastHeartbeat

	// sendMessage will fail on nil conn, but heartbeat timestamp should update
	gateway.handleHeartbeat(nil, session)

	assert.True(t, session.LastHeartbeat.After(oldHeartbeat))
}

func TestGateway_HandleResume_NoSession(t *testing.T) {
	gateway, _, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()

	// handleResume with non-existent session
	result := gateway.handleResume(nil, "non-existent-key", userID)

	// Should return true (handled - invalid session)
	assert.True(t, result)
}

func TestGateway_HandleResume_WrongUser(t *testing.T) {
	gateway, _, cancel := newTestGateway(t)
	defer cancel()

	resumeKey := "test-resume-key"
	sessionUserID := uuid.New()
	wrongUserID := uuid.New()

	gateway.sessionsMu.Lock()
	gateway.sessions[resumeKey] = &Session{
		ID:            "session-1",
		UserID:        sessionUserID,
		Username:      "testuser",
		ResumeKey:     resumeKey,
		LastHeartbeat: time.Now(),
		ResumeEvents:  make([][]byte, 0, 100),
	}
	gateway.sessionsMu.Unlock()

	result := gateway.handleResume(nil, resumeKey, wrongUserID)

	// Should return true (invalid session - wrong user)
	assert.True(t, result)
}

func TestGateway_HandleResume_TimedOut(t *testing.T) {
	gateway, _, cancel := newTestGateway(t)
	defer cancel()

	resumeKey := "test-resume-key"
	userID := uuid.New()

	gateway.sessionsMu.Lock()
	gateway.sessions[resumeKey] = &Session{
		ID:            "session-1",
		UserID:        userID,
		Username:      "testuser",
		ResumeKey:     resumeKey,
		LastHeartbeat: time.Now().Add(-10 * time.Minute), // Expired
		ResumeEvents:  make([][]byte, 0, 100),
	}
	gateway.sessionsMu.Unlock()

	result := gateway.handleResume(nil, resumeKey, userID)

	// Should return true (session timed out)
	assert.True(t, result)

	// Session should be removed
	gateway.sessionsMu.RLock()
	_, exists := gateway.sessions[resumeKey]
	gateway.sessionsMu.RUnlock()
	assert.False(t, exists)
}

func TestGateway_CreateHubClient_WithHub(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	session := &Session{
		ID:         "session-1",
		UserID:     uuid.New(),
		Username:   "testuser",
		ClientType: "web",
	}

	client := gateway.createHubClient(nil, session)

	require.NotNil(t, client)
	assert.Equal(t, session.UserID, client.UserID)
	assert.Equal(t, session.Username, client.Username)
	assert.Equal(t, session.ID, client.SessionID)
	assert.Equal(t, session.ClientType, client.ClientType)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.servers)
	assert.NotNil(t, client.channels)
}

func TestGateway_CreateHubClient_WithDistributedHub(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "test-create-client")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	gateway := &Gateway{
		hub:       dh,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	session := &Session{
		ID:         "session-1",
		UserID:     uuid.New(),
		Username:   "testuser",
		ClientType: "desktop",
	}

	client := gateway.createHubClient(nil, session)

	require.NotNil(t, client)
	assert.Equal(t, dh.Hub, client.hub)
	assert.Equal(t, "desktop", client.ClientType)
}

func TestGateway_HandleMessage_InvalidJSON(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	// Can't fully test as sendError uses conn, but exercises the code path
	gateway.handleMessage(nil, client, session, []byte("not json"))

	// messagesProcessed should NOT increment on invalid JSON (returns early)
	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(0), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_Heartbeat(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{
		ID:            "session-1",
		UserID:        userID,
		LastHeartbeat: time.Now().Add(-time.Hour),
	}

	msg, _ := json.Marshal(Message{Op: OpHeartbeat})

	gateway.handleMessage(nil, client, session, msg)

	assert.Equal(t, int64(1), session.Sequence)
	assert.True(t, session.LastHeartbeat.After(time.Now().Add(-time.Second)))

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_Resume(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg, _ := json.Marshal(Message{Op: OpResume})

	// Resume at message level sends error (must be on connect)
	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_UnknownOpcode(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg, _ := json.Marshal(Message{Op: 99})

	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_Dispatch(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	session := &Session{ID: "session-1", UserID: userID}

	channelID := uuid.New()
	dispatchData, _ := json.Marshal(map[string]interface{}{
		"t": "SUBSCRIBE",
		"d": map[string]string{
			"channel_id": channelID.String(),
		},
	})
	msg, _ := json.Marshal(Message{Op: OpDispatch, Data: dispatchData})

	gateway.handleMessage(nil, client, session, msg)

	assert.True(t, client.IsSubscribedToChannel(channelID))
}

func TestGateway_HandleMessage_Identify(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID, Username: "testuser"}

	msg, _ := json.Marshal(Message{Op: OpIdentify})

	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_PresenceUpdate(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	presenceData, _ := json.Marshal(map[string]interface{}{
		"status": "online",
	})
	msg, _ := json.Marshal(Message{Op: OpPresenceUpdate, Data: presenceData})

	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_VoiceStateUpdate(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	msg, _ := json.Marshal(Message{Op: OpVoiceStateUpdate})

	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_HandleMessage_RequestGuildMembers(t *testing.T) {
	gateway, hub, cancel := newTestGateway(t)
	defer cancel()

	userID := uuid.New()
	client := newTestClient(t, hub, userID)
	session := &Session{ID: "session-1", UserID: userID}

	requestData, _ := json.Marshal(map[string]interface{}{
		"guild_id": uuid.New().String(),
		"nonce":    "test",
	})
	msg, _ := json.Marshal(Message{Op: OpRequestGuildMembers, Data: requestData})

	gateway.handleMessage(nil, client, session, msg)

	gateway.connectionsMu.RLock()
	assert.Equal(t, int64(1), gateway.messagesProcessed)
	gateway.connectionsMu.RUnlock()
}

func TestGateway_Shutdown_NilHub(t *testing.T) {
	gateway := &Gateway{
		sessions: make(map[string]*Session),
	}

	err := gateway.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, gateway.draining.Load())
}

func TestGateway_IsHealthy_NilHub(t *testing.T) {
	gateway := &Gateway{}

	assert.True(t, gateway.IsHealthy())

	gateway.draining.Store(true)
	assert.False(t, gateway.IsHealthy())
}

func TestGateway_DrainState_Healthy(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	assert.Equal(t, DrainStateHealthy, gateway.DrainState())
}

func TestGateway_DrainState_NilHub_NotDraining(t *testing.T) {
	gateway := &Gateway{}

	assert.Equal(t, DrainStateHealthy, gateway.DrainState())
}
