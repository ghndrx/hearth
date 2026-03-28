package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_GetClientCount(t *testing.T) {
	hub := NewHub()

	assert.Equal(t, 0, hub.GetClientCount())

	// Register 3 clients (2 for same user, 1 for different user)
	user1 := uuid.New()
	user2 := uuid.New()

	c1 := &Client{ID: uuid.New().String(), UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c2 := &Client{ID: uuid.New().String(), UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c3 := &Client{ID: uuid.New().String(), UserID: user2, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.registerClient(c3)

	assert.Equal(t, 3, hub.GetClientCount())
}

func TestHub_GetAllClients(t *testing.T) {
	hub := NewHub()

	clients := hub.getAllClients()
	assert.Empty(t, clients)

	user1 := uuid.New()
	user2 := uuid.New()

	c1 := &Client{ID: "c1", UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c2 := &Client{ID: "c2", UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c3 := &Client{ID: "c3", UserID: user2, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.registerClient(c3)

	clients = hub.getAllClients()
	assert.Len(t, clients, 3)
}

func TestHub_SetDrainConfig(t *testing.T) {
	hub := NewHub()

	// Verify default drain state
	assert.True(t, hub.IsHealthy())

	// Set custom config
	cfg := &DrainConfig{
		DrainTimeout: 10 * time.Second,
		GracePeriod:  2 * time.Second,
	}
	hub.SetDrainConfig(cfg)

	assert.True(t, hub.IsHealthy())
	assert.Equal(t, DrainStateHealthy, hub.DrainState())
}

func TestHub_BroadcastChannelFullBuffer(t *testing.T) {
	hub := NewHub()
	channelID := uuid.New()

	// Client with tiny buffer (already full)
	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 1),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)
	hub.SubscribeChannel(client, channelID)

	// Fill the buffer
	client.send <- []byte("filler")

	// Broadcast should skip the full-buffer client without blocking
	event := &Event{
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data:      map[string]string{"content": "should be skipped"},
	}
	hub.handleBroadcast(event)

	// Only the filler message should be in the buffer
	msg := <-client.send
	assert.Equal(t, "filler", string(msg))
}

func TestHub_BroadcastServerFullBuffer(t *testing.T) {
	hub := NewHub()
	serverID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 1),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)
	hub.SubscribeServer(client, serverID)

	// Fill buffer
	client.send <- []byte("filler")

	event := &Event{
		Type:     EventTypeServerUpdate,
		ServerID: &serverID,
		Data:     map[string]string{"name": "test"},
	}
	hub.handleBroadcast(event)

	msg := <-client.send
	assert.Equal(t, "filler", string(msg))
}

func TestHub_BroadcastUserFullBuffer(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		hub:      hub,
		send:     make(chan []byte, 1),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)

	// Fill buffer
	client.send <- []byte("filler")

	event := &Event{
		Type:   EventTypePresenceUpdate,
		UserID: &userID,
		Data:   map[string]string{"status": "online"},
	}
	hub.handleBroadcast(event)

	msg := <-client.send
	assert.Equal(t, "filler", string(msg))
}

func TestHub_BroadcastMarshalError(t *testing.T) {
	hub := NewHub()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)
	hub.SubscribeChannel(client, channelID)

	// Use unmarshalable data (channel type)
	event := &Event{
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data:      make(chan int),
	}
	hub.handleBroadcast(event)

	// Nothing should be sent due to marshal error
	select {
	case <-client.send:
		t.Fatal("Should not receive message on marshal error")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestHub_UnregisterOneOfMultipleClients(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	c1 := &Client{ID: "c1", UserID: userID, hub: hub, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c2 := &Client{ID: "c2", UserID: userID, hub: hub, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.SubscribeChannel(c1, channelID)
	hub.SubscribeChannel(c2, channelID)
	hub.SubscribeServer(c1, serverID)
	hub.SubscribeServer(c2, serverID)

	// Unregister c1 - c2 should remain
	hub.unregisterClient(c1)

	hub.clientsMux.RLock()
	assert.Len(t, hub.clients[userID], 1)
	assert.True(t, hub.clients[userID][c2])
	hub.clientsMux.RUnlock()

	hub.channelsMux.RLock()
	assert.Len(t, hub.channels[channelID], 1)
	hub.channelsMux.RUnlock()

	hub.serversMux.RLock()
	assert.Len(t, hub.servers[serverID], 1)
	hub.serversMux.RUnlock()
}

func TestHub_RegisterUnregisterClient_Channels(t *testing.T) {
	hub := NewHub()

	regCh := hub.RegisterClient()
	unregCh := hub.UnregisterClient()

	require.NotNil(t, regCh)
	require.NotNil(t, unregCh)
}

func TestHub_ConnectDisconnectReconnect(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	// Connect
	client1 := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client1
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeChannel(client1, channelID)
	hub.SubscribeServer(client1, serverID)

	assert.Equal(t, 1, hub.GetClientCount())

	// Verify can receive messages
	hub.SendToChannel(channelID, &Event{Type: EventTypeMessageCreate, Data: "test"})
	select {
	case <-client1.send:
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}

	// Disconnect
	hub.unregister <- client1
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())

	// Reconnect with new client
	client2 := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeChannel(client2, channelID)

	assert.Equal(t, 1, hub.GetClientCount())

	// Verify new client receives messages
	hub.SendToChannel(channelID, &Event{Type: EventTypeMessageCreate, Data: "reconnected"})
	select {
	case msg := <-client2.send:
		assert.Contains(t, string(msg), "MESSAGE_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Reconnected client did not receive message")
	}
}

func TestHub_MultipleSessionsSameUser(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	userID := uuid.New()

	// Desktop session
	desktop := &Client{
		ID:         uuid.New().String(),
		UserID:     userID,
		Username:   "user1",
		ClientType: "desktop",
		hub:        hub,
		send:       make(chan []byte, 256),
		servers:    make(map[uuid.UUID]bool),
		channels:   make(map[uuid.UUID]bool),
	}

	// Web session
	web := &Client{
		ID:         uuid.New().String(),
		UserID:     userID,
		Username:   "user1",
		ClientType: "web",
		hub:        hub,
		send:       make(chan []byte, 256),
		servers:    make(map[uuid.UUID]bool),
		channels:   make(map[uuid.UUID]bool),
	}

	hub.register <- desktop
	hub.register <- web
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 2, hub.GetClientCount())

	// Send to user - both sessions should receive
	hub.SendToUser(userID, &Event{Type: EventTypePresenceUpdate, Data: "status"})

	for _, client := range []*Client{desktop, web} {
		select {
		case <-client.send:
		case <-time.After(time.Second):
			t.Fatalf("Client %s did not receive message", client.ClientType)
		}
	}

	// Disconnect desktop - web should still work
	hub.unregister <- desktop
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.GetClientCount())

	hub.SendToUser(userID, &Event{Type: EventTypePresenceUpdate, Data: "still online"})

	select {
	case <-web.send:
	case <-time.After(time.Second):
		t.Fatal("Web client did not receive message after desktop disconnected")
	}
}

func TestHub_ConcurrentRegisterUnregister(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup

	// Register and unregister clients concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client := &Client{
				ID:       uuid.New().String(),
				UserID:   uuid.New(),
				hub:      hub,
				send:     make(chan []byte, 256),
				servers:  make(map[uuid.UUID]bool),
				channels: make(map[uuid.UUID]bool),
			}

			hub.register <- client
			time.Sleep(10 * time.Millisecond)
			hub.unregister <- client
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())
}

func TestHub_ConcurrentSubscribeAndBroadcast(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	serverID := uuid.New()

	// Register clients
	clients := make([]*Client, 5)
	for i := range clients {
		clients[i] = &Client{
			ID:       uuid.New().String(),
			UserID:   uuid.New(),
			hub:      hub,
			send:     make(chan []byte, 256),
			servers:  make(map[uuid.UUID]bool),
			channels: make(map[uuid.UUID]bool),
		}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	// Concurrently subscribe and broadcast
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(c *Client) {
			defer wg.Done()
			hub.SubscribeChannel(c, channelID)
			hub.SubscribeServer(c, serverID)
		}(clients[i])

		go func() {
			defer wg.Done()
			hub.SendToChannel(channelID, &Event{Type: EventTypeMessageCreate, Data: "test"})
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}

func TestHub_UnsubscribeChannelNonExistent(t *testing.T) {
	hub := NewHub()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	// Unsubscribe from channel never subscribed to - should not panic
	hub.UnsubscribeChannel(client, channelID)

	hub.channelsMux.RLock()
	_, exists := hub.channels[channelID]
	hub.channelsMux.RUnlock()
	assert.False(t, exists)
}

func TestHub_SendToUserMultipleConnections(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	userID := uuid.New()

	// Register 3 clients for the same user
	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = &Client{
			ID:       uuid.New().String(),
			UserID:   userID,
			hub:      hub,
			send:     make(chan []byte, 256),
			servers:  make(map[uuid.UUID]bool),
			channels: make(map[uuid.UUID]bool),
		}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	// All 3 should receive user-targeted message
	hub.SendToUser(userID, &Event{Type: "DM_RECEIVED", Data: "hello"})

	for i, c := range clients {
		select {
		case msg := <-c.send:
			assert.Contains(t, string(msg), "DM_RECEIVED")
		case <-time.After(time.Second):
			t.Fatalf("Client %d did not receive user message", i)
		}
	}
}

func TestHub_BroadcastViaPublicMethod(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()
	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)
	hub.SubscribeChannel(client, channelID)

	// Use the Broadcast method (generic)
	hub.Broadcast(&Event{
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data:      map[string]string{"via": "broadcast"},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "broadcast")
	case <-time.After(time.Second):
		t.Fatal("Did not receive broadcast message")
	}
}

func TestHub_GetOnlineUsersWithMultipleSessions(t *testing.T) {
	hub := NewHub()

	user1 := uuid.New()
	user2 := uuid.New()

	// User1 has 2 sessions
	c1a := &Client{ID: "c1a", UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c1b := &Client{ID: "c1b", UserID: user1, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}
	c2 := &Client{ID: "c2", UserID: user2, send: make(chan []byte, 256), servers: make(map[uuid.UUID]bool), channels: make(map[uuid.UUID]bool)}

	hub.registerClient(c1a)
	hub.registerClient(c1b)
	hub.registerClient(c2)

	online := hub.GetOnlineUsers([]uuid.UUID{user1, user2})
	assert.Len(t, online, 2)

	// Unregister one of user1's sessions - should still be online
	hub.unregisterClient(c1a)

	online = hub.GetOnlineUsers([]uuid.UUID{user1, user2})
	assert.Len(t, online, 2)
	assert.Contains(t, online, user1)

	// Unregister user1's last session
	hub.unregisterClient(c1b)

	online = hub.GetOnlineUsers([]uuid.UUID{user1, user2})
	assert.Len(t, online, 1)
	assert.Contains(t, online, user2)
}

func TestHub_EventSerialization(t *testing.T) {
	hub := NewHub()
	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)
	hub.SubscribeChannel(client, channelID)

	event := &Event{
		Op:        0,
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data: map[string]interface{}{
			"id":         uuid.New().String(),
			"channel_id": channelID.String(),
			"content":    "Test message content",
			"author": map[string]string{
				"id":       uuid.New().String(),
				"username": "testuser",
			},
		},
		Sequence: 42,
	}

	hub.handleBroadcast(event)

	select {
	case msg := <-client.send:
		var received Event
		err := json.Unmarshal(msg, &received)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMessageCreate, received.Type)
		assert.Equal(t, 0, received.Op)
		assert.Equal(t, int64(42), received.Sequence)
	case <-time.After(time.Second):
		t.Fatal("Did not receive event")
	}
}

func TestHub_SubscribeMultipleChannels(t *testing.T) {
	hub := NewHub()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)

	channels := make([]uuid.UUID, 5)
	for i := range channels {
		channels[i] = uuid.New()
		hub.SubscribeChannel(client, channels[i])
	}

	hub.channelsMux.RLock()
	for _, ch := range channels {
		assert.True(t, hub.channels[ch][client])
	}
	hub.channelsMux.RUnlock()

	// Unsubscribe from all
	for _, ch := range channels {
		hub.UnsubscribeChannel(client, ch)
	}

	hub.channelsMux.RLock()
	for _, ch := range channels {
		_, exists := hub.channels[ch]
		assert.False(t, exists)
	}
	hub.channelsMux.RUnlock()
}
