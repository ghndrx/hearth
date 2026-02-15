package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHub_HandleBroadcast_ToChannel(t *testing.T) {
	hub := NewHub()
	channelID := uuid.New()

	// Create and register clients
	client1 := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}
	client2 := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user2",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	hub.SubscribeChannel(client1, channelID)
	hub.SubscribeChannel(client2, channelID)

	// Create broadcast message
	event := &Event{
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data:      map[string]string{"content": "Hello channel!"},
	}

	hub.handleBroadcast(event)

	// Both clients should receive the message
	timeout := time.After(time.Second)
	received := 0

	for received < 2 {
		select {
		case msg := <-client1.send:
			assert.Contains(t, string(msg), "MESSAGE_CREATE")
			received++
		case msg := <-client2.send:
			assert.Contains(t, string(msg), "MESSAGE_CREATE")
			received++
		case <-timeout:
			t.Fatal("Timeout waiting for broadcast")
		}
	}
}

func TestHub_HandleBroadcast_ToServer(t *testing.T) {
	hub := NewHub()
	serverID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)
	hub.SubscribeServer(client, serverID)

	event := &Event{
		Type:     EventTypeServerUpdate,
		ServerID: &serverID,
		Data:     map[string]string{"name": "Updated Server"},
	}

	hub.handleBroadcast(event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive server broadcast")
	}
}

func TestHub_HandleBroadcast_ToUser(t *testing.T) {
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

	hub.registerClient(client)

	event := &Event{
		Type:   EventTypePresenceUpdate,
		UserID: &userID,
		Data:   map[string]string{"status": "online"},
	}

	hub.handleBroadcast(event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "PRESENCE_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive user broadcast")
	}
}

func TestHub_HandleBroadcast_NoTarget(t *testing.T) {
	hub := NewHub()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client)

	// No UserID, ChannelID, or ServerID - should not send to anyone
	// (current implementation doesn't support global broadcasts)
	event := &Event{
		Type: "MAINTENANCE_NOTICE",
		Data: map[string]string{"message": "Server maintenance in 5 minutes"},
	}

	hub.handleBroadcast(event)

	// No message should be received since no target is specified
	select {
	case <-client.send:
		t.Fatal("Should not receive message without target")
	case <-time.After(100 * time.Millisecond):
		// Expected - no target means no delivery
	}
}

func TestHub_Broadcast_Channel(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeChannel(client, channelID)

	// Use broadcast channel directly
	hub.broadcast <- &Event{
		Type:      EventTypeMessageCreate,
		ChannelID: &channelID,
		Data:      json.RawMessage(`{"content":"test"}`),
	}

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive broadcast")
	}
}

func TestHub_UnsubscribeServerCleanup(t *testing.T) {
	hub := NewHub()
	serverID := uuid.New()

	client1 := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}
	client2 := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "user2",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.SubscribeServer(client1, serverID)
	hub.SubscribeServer(client2, serverID)

	hub.serversMux.RLock()
	assert.Len(t, hub.servers[serverID], 2)
	hub.serversMux.RUnlock()

	// Unsubscribe first client
	hub.serversMux.Lock()
	delete(hub.servers[serverID], client1)
	if len(hub.servers[serverID]) == 0 {
		delete(hub.servers, serverID)
	}
	hub.serversMux.Unlock()

	hub.serversMux.RLock()
	assert.Len(t, hub.servers[serverID], 1)
	hub.serversMux.RUnlock()
}

func TestHub_MultipleUsersPerServer(t *testing.T) {
	hub := NewHub()
	serverID := uuid.New()

	// 10 clients in same server
	clients := make([]*Client, 10)
	for i := 0; i < 10; i++ {
		clients[i] = &Client{
			ID:       uuid.New().String(),
			UserID:   uuid.New(),
			Username: "user",
			hub:      hub,
			send:     make(chan []byte, 256),
			servers:  make(map[uuid.UUID]bool),
			channels: make(map[uuid.UUID]bool),
		}
		hub.registerClient(clients[i])
		hub.SubscribeServer(clients[i], serverID)
	}

	event := &Event{
		Type:     EventTypeServerUpdate,
		ServerID: &serverID,
		Data:     map[string]string{"update": "test"},
	}

	hub.handleBroadcast(event)

	// All 10 clients should receive
	timeout := time.After(2 * time.Second)
	received := 0

	for received < 10 {
		for _, c := range clients {
			select {
			case <-c.send:
				received++
			default:
			}
		}
		if received >= 10 {
			break
		}
		select {
		case <-timeout:
			t.Fatalf("Only received %d of 10 broadcasts", received)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestHub_ConcurrentBroadcasts(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	channelID := uuid.New()

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)
	hub.SubscribeChannel(client, channelID)

	// Send multiple broadcasts concurrently
	for i := 0; i < 10; i++ {
		go func(i int) {
			hub.SendToChannel(channelID, &Event{
				Type: EventTypeMessageCreate,
				Data: map[string]int{"number": i},
			})
		}(i)
	}

	// Collect all messages
	received := 0
	timeout := time.After(2 * time.Second)

	for received < 10 {
		select {
		case <-client.send:
			received++
		case <-timeout:
			t.Fatalf("Only received %d of 10 concurrent broadcasts", received)
		}
	}

	assert.Equal(t, 10, received)
}

func TestHub_SendToNonExistentUser(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	nonExistentUser := uuid.New()

	// Should not panic
	hub.SendToUser(nonExistentUser, &Event{
		Type: EventTypePresenceUpdate,
		Data: map[string]string{"status": "online"},
	})

	// Give time to process
	time.Sleep(50 * time.Millisecond)
	// No assertion needed - just verify no panic
}

func TestHub_SendToNonExistentChannel(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	nonExistentChannel := uuid.New()

	// Should not panic
	hub.SendToChannel(nonExistentChannel, &Event{
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "test"},
	})

	time.Sleep(50 * time.Millisecond)
}

func TestHub_SendToNonExistentServer(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	nonExistentServer := uuid.New()

	// Should not panic
	hub.SendToServer(nonExistentServer, &Event{
		Type: EventTypeServerUpdate,
		Data: map[string]string{"name": "test"},
	})

	time.Sleep(50 * time.Millisecond)
}

func TestHub_UnregisterWithPendingMessages(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Fill send buffer
	for i := 0; i < 200; i++ {
		select {
		case client.send <- []byte(`{"test":"message"}`):
		default:
			break
		}
	}

	// Unregister should clean up properly
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.clientsMux.RLock()
	_, exists := hub.clients[client.UserID]
	hub.clientsMux.RUnlock()

	assert.False(t, exists)
}
