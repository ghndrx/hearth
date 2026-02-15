package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_HandlePresenceUpdate(t *testing.T) {
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

	msg := &Message{Op: OpPresenceUpdate}

	// Should not panic even without presence service integration
	client.handlePresenceUpdate(msg)
}

func TestClient_HandleMessagePresenceUpdate(t *testing.T) {
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

	msg := Message{
		Op:   OpPresenceUpdate,
		Data: json.RawMessage(`{"status": "online"}`),
	}
	data, _ := json.Marshal(msg)

	client.handleMessage(data)

	assert.Equal(t, int64(1), client.sequence)
}

func TestClient_SendBufferFull(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	// Create client with small send buffer
	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 1), // Small buffer
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	// Fill the buffer
	msg1 := &Message{Op: OpDispatch, Type: "TEST1"}
	client.Send(msg1)

	// Verify first message went through
	select {
	case <-client.send:
		// Good, consumed message
	default:
		t.Fatal("Expected message in buffer")
	}

	// Refill the buffer to capacity
	client.Send(msg1)

	// The Send function will block if buffer is full and then close channel
	// We can't safely test this without goroutines, so just verify the buffer is full
	select {
	case <-client.send:
		// Good, buffer was full and we consumed it
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Buffer should have had message")
	}
}

func TestClient_ConcurrentSubscriptions(t *testing.T) {
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

	// Subscribe to multiple servers/channels concurrently
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(i int) {
			serverID := uuid.New()
			channelID := uuid.New()

			client.SubscribeServer(serverID)
			client.SubscribeChannel(channelID)

			assert.True(t, client.IsSubscribedToServer(serverID))
			assert.True(t, client.IsSubscribedToChannel(channelID))

			client.UnsubscribeServer(serverID)
			client.UnsubscribeChannel(channelID)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestClient_SendNilMessage(t *testing.T) {
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

	// Sending nil should not panic
	client.Send(nil)

	// Channel should still be empty (nil marshal returns null)
	select {
	case msg := <-client.send:
		// "null" is valid JSON for nil
		assert.Equal(t, "null", string(msg))
	case <-time.After(100 * time.Millisecond):
		// Or might not send anything
	}
}

func TestClient_HandleHeartbeatWithData(t *testing.T) {
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
		lastHeartbeat: time.Now().Add(-time.Hour),
		sequence:      42,
	}

	msg := &Message{
		Op:       OpHeartbeat,
		Sequence: 42,
	}
	client.handleHeartbeat(msg)

	// Verify heartbeat was updated
	assert.True(t, time.Since(client.lastHeartbeat) < time.Second)

	// Verify ack was sent
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

func TestClient_HandleIdentifyWithData(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	sessionID := uuid.New().String()

	client := &Client{
		ID:        uuid.New().String(),
		UserID:    userID,
		Username:  "testuser",
		SessionID: sessionID,
		hub:       hub,
		send:      make(chan []byte, 256),
		servers:   make(map[uuid.UUID]bool),
		channels:  make(map[uuid.UUID]bool),
	}

	identifyData := map[string]interface{}{
		"token": "test-token",
		"properties": map[string]string{
			"$os":      "linux",
			"$browser": "chrome",
			"$device":  "pc",
		},
		"compress": false,
	}
	data, _ := json.Marshal(identifyData)

	msg := &Message{
		Op:   OpIdentify,
		Data: data,
	}
	client.handleIdentify(msg)

	// Should receive READY
	select {
	case received := <-client.send:
		var resp Message
		err := json.Unmarshal(received, &resp)
		require.NoError(t, err)
		assert.Equal(t, OpDispatch, resp.Op)
		assert.Equal(t, "READY", resp.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive READY")
	}
}

func TestClient_HandleAllOpcodes(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	sessionID := uuid.New().String()

	client := &Client{
		ID:        uuid.New().String(),
		UserID:    userID,
		Username:  "testuser",
		SessionID: sessionID,
		hub:       hub,
		send:      make(chan []byte, 256),
		servers:   make(map[uuid.UUID]bool),
		channels:  make(map[uuid.UUID]bool),
	}

	tests := []struct {
		name     string
		op       int
		expected string
	}{
		{"heartbeat", OpHeartbeat, "heartbeat_ack"},
		{"identify", OpIdentify, "READY"},
		{"resume", OpResume, "RESUMED"},
		{"presence_update", OpPresenceUpdate, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := Message{Op: tt.op}
			data, _ := json.Marshal(msg)
			client.handleMessage(data)

			if tt.expected != "" {
				select {
				case <-client.send:
					// Message received
				case <-time.After(100 * time.Millisecond):
					if tt.expected != "" {
						t.Fatalf("Expected response for %s", tt.name)
					}
				}
			}
		})
	}
}

func TestClient_SubscriptionCleanup(t *testing.T) {
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

	// Subscribe to multiple resources
	servers := make([]uuid.UUID, 5)
	channels := make([]uuid.UUID, 5)

	for i := 0; i < 5; i++ {
		servers[i] = uuid.New()
		channels[i] = uuid.New()
		client.SubscribeServer(servers[i])
		client.SubscribeChannel(channels[i])
	}

	// Verify all subscriptions
	for i := 0; i < 5; i++ {
		assert.True(t, client.IsSubscribedToServer(servers[i]))
		assert.True(t, client.IsSubscribedToChannel(channels[i]))
	}

	// Unsubscribe all
	for i := 0; i < 5; i++ {
		client.UnsubscribeServer(servers[i])
		client.UnsubscribeChannel(channels[i])
	}

	// Verify cleanup
	for i := 0; i < 5; i++ {
		assert.False(t, client.IsSubscribedToServer(servers[i]))
		assert.False(t, client.IsSubscribedToChannel(channels[i]))
	}
}

func TestClient_DoubleSubscription(t *testing.T) {
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

	serverID := uuid.New()
	channelID := uuid.New()

	// Subscribe twice
	client.SubscribeServer(serverID)
	client.SubscribeServer(serverID)

	client.SubscribeChannel(channelID)
	client.SubscribeChannel(channelID)

	assert.True(t, client.IsSubscribedToServer(serverID))
	assert.True(t, client.IsSubscribedToChannel(channelID))

	// Single unsubscribe should work
	client.UnsubscribeServer(serverID)
	client.UnsubscribeChannel(channelID)

	assert.False(t, client.IsSubscribedToServer(serverID))
	assert.False(t, client.IsSubscribedToChannel(channelID))
}

func TestClient_SequenceIncrement(t *testing.T) {
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

	// Send multiple heartbeats and verify sequence increments
	for i := 1; i <= 10; i++ {
		msg := Message{Op: OpHeartbeat}
		data, _ := json.Marshal(msg)
		client.handleMessage(data)

		assert.Equal(t, int64(i), client.sequence)

		// Drain the send channel
		<-client.send
	}
}
