package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientConstants(t *testing.T) {
	assert.Equal(t, 10*time.Second, writeWait)
	assert.Equal(t, 60*time.Second, pongWait)
	assert.Equal(t, (pongWait*9)/10, pingPeriod)
	assert.Equal(t, int64(65536), int64(maxMessageSize))
}

func TestNewClient(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	sessionID := uuid.New().String()

	client := NewClient(hub, nil, userID, "testuser", sessionID, "web")

	require.NotNil(t, client)
	assert.NotEmpty(t, client.ID)
	assert.Equal(t, userID, client.UserID)
	assert.Equal(t, "testuser", client.Username)
	assert.Equal(t, sessionID, client.SessionID)
	assert.Equal(t, "web", client.ClientType)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.servers)
	assert.NotNil(t, client.channels)
	assert.Equal(t, int64(0), client.sequence)
}

func TestClient_SubscribeServer(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	serverID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.SubscribeServer(serverID)

	assert.True(t, client.servers[serverID])

	hub.serversMux.RLock()
	assert.True(t, hub.servers[serverID][client])
	hub.serversMux.RUnlock()
}

func TestClient_UnsubscribeServer(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	serverID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.SubscribeServer(serverID)
	client.UnsubscribeServer(serverID)

	assert.False(t, client.servers[serverID])
}

func TestClient_SubscribeChannel(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.SubscribeChannel(channelID)

	assert.True(t, client.channels[channelID])

	hub.channelsMux.RLock()
	assert.True(t, hub.channels[channelID][client])
	hub.channelsMux.RUnlock()
}

func TestClient_UnsubscribeChannel(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.SubscribeChannel(channelID)
	client.UnsubscribeChannel(channelID)

	assert.False(t, client.channels[channelID])

	hub.channelsMux.RLock()
	_, exists := hub.channels[channelID]
	hub.channelsMux.RUnlock()
	assert.False(t, exists)
}

func TestClient_IsSubscribedToServer(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	serverID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	assert.False(t, client.IsSubscribedToServer(serverID))

	client.SubscribeServer(serverID)

	assert.True(t, client.IsSubscribedToServer(serverID))
}

func TestClient_IsSubscribedToChannel(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	assert.False(t, client.IsSubscribedToChannel(channelID))

	client.SubscribeChannel(channelID)

	assert.True(t, client.IsSubscribedToChannel(channelID))
}

func TestClient_HandleHeartbeat(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:            uuid.New().String(),
		UserID:        userID,
		Username:      "testuser",
		hub:           hub,
		send:          make(chan []byte, 256),
		servers:       make(map[uuid.UUID]bool),
		channels:      make(map[uuid.UUID]bool),
		lastHeartbeat: time.Now().Add(-time.Hour), // Old heartbeat
	}

	oldHeartbeat := client.lastHeartbeat

	msg := &Message{Op: OpHeartbeat}
	client.handleHeartbeat(msg)

	assert.True(t, client.lastHeartbeat.After(oldHeartbeat))

	// Should receive HeartbeatAck
	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, OpHeartbeatAck, resp.Op)
	case <-time.After(time.Second):
		t.Fatal("Did not receive heartbeat ack")
	}
}

func TestClient_HandleIdentify(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	sessionID := uuid.New().String()

	client := &Client{
		ID:        uuid.New().String(),
		UserID:   userID,
		Username:  "testuser",
		SessionID: sessionID,
		hub:       hub,
		send:      make(chan []byte, 256),
		servers:   make(map[uuid.UUID]bool),
		channels:  make(map[uuid.UUID]bool),
	}

	msg := &Message{Op: OpIdentify}
	client.handleIdentify(msg)

	// Should receive READY event
	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, OpDispatch, resp.Op)
		assert.Equal(t, "READY", resp.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive READY event")
	}
}

func TestClient_HandleResume(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	msg := &Message{Op: OpResume}
	client.handleResume(msg)

	// Should receive RESUMED event
	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, OpDispatch, resp.Op)
		assert.Equal(t, "RESUMED", resp.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive RESUMED event")
	}
}

func TestClient_HandleMessageInvalidFormat(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.handleMessage([]byte("not json"))

	// Should receive error
	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, "ERROR", resp.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive error")
	}
}

func TestClient_HandleMessageUnknownOpcode(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	msg := Message{Op: 99} // Unknown opcode
	data, _ := json.Marshal(msg)
	client.handleMessage(data)

	// Should receive error
	select {
	case received := <-client.send:
		var resp Message
		err := json.Unmarshal(received, &resp)
		require.NoError(t, err)
		assert.Equal(t, "ERROR", resp.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive error")
	}
}

func TestClient_HandleMessageSequence(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
		sequence: 0,
	}

	// Send multiple messages and verify sequence increments
	msg := Message{Op: OpHeartbeat}
	data, _ := json.Marshal(msg)

	client.handleMessage(data)
	assert.Equal(t, int64(1), client.sequence)

	client.handleMessage(data)
	assert.Equal(t, int64(2), client.sequence)

	client.handleMessage(data)
	assert.Equal(t, int64(3), client.sequence)
}

func TestClient_SendReady(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	sessionID := uuid.New().String()

	client := &Client{
		ID:        uuid.New().String(),
		UserID:   userID,
		Username:  "testuser",
		SessionID: sessionID,
		hub:       hub,
		send:      make(chan []byte, 256),
		servers:   make(map[uuid.UUID]bool),
		channels:  make(map[uuid.UUID]bool),
		sequence:  5,
	}

	client.sendReady()

	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, OpDispatch, resp.Op)
		assert.Equal(t, "READY", resp.Type)
		assert.Equal(t, int64(5), resp.Sequence)

		// Verify ready data
		var readyData map[string]interface{}
		err = json.Unmarshal(resp.Data, &readyData)
		require.NoError(t, err)
		assert.Equal(t, float64(10), readyData["v"])
		assert.Equal(t, sessionID, readyData["session_id"])

		user := readyData["user"].(map[string]interface{})
		assert.Equal(t, userID.String(), user["id"])
		assert.Equal(t, "testuser", user["username"])
	case <-time.After(time.Second):
		t.Fatal("Did not receive READY")
	}
}

func TestClient_SendError(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client.sendError("Test error message")

	select {
	case data := <-client.send:
		var resp Message
		err := json.Unmarshal(data, &resp)
		require.NoError(t, err)
		assert.Equal(t, "ERROR", resp.Type)

		var errorData map[string]string
		err = json.Unmarshal(resp.Data, &errorData)
		require.NoError(t, err)
		assert.Equal(t, "Test error message", errorData["message"])
	case <-time.After(time.Second):
		t.Fatal("Did not receive error")
	}
}

func TestClient_Send(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	msg := &Message{
		Op:   OpDispatch,
		Type: EventMessageCreate,
		Data: json.RawMessage(`{"content":"test"}`),
	}

	client.Send(msg)

	select {
	case data := <-client.send:
		var received Message
		err := json.Unmarshal(data, &received)
		require.NoError(t, err)
		assert.Equal(t, OpDispatch, received.Op)
		assert.Equal(t, EventMessageCreate, received.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive message")
	}
}

func TestClient_MultipleSubscriptions(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	server1 := uuid.New()
	server2 := uuid.New()
	channel1 := uuid.New()
	channel2 := uuid.New()

	client.SubscribeServer(server1)
	client.SubscribeServer(server2)
	client.SubscribeChannel(channel1)
	client.SubscribeChannel(channel2)

	assert.True(t, client.IsSubscribedToServer(server1))
	assert.True(t, client.IsSubscribedToServer(server2))
	assert.True(t, client.IsSubscribedToChannel(channel1))
	assert.True(t, client.IsSubscribedToChannel(channel2))

	client.UnsubscribeServer(server1)
	client.UnsubscribeChannel(channel1)

	assert.False(t, client.IsSubscribedToServer(server1))
	assert.True(t, client.IsSubscribedToServer(server2))
	assert.False(t, client.IsSubscribedToChannel(channel1))
	assert.True(t, client.IsSubscribedToChannel(channel2))
}
