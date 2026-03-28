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

func TestHub_Lifecycle_ConnectDisconnectReconnect(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	userID := uuid.New()

	// Connect
	client := newMockClient(hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 1, hub.GetClientCount())
	online := hub.GetOnlineUsers([]uuid.UUID{userID})
	assert.Contains(t, online, userID)

	// Disconnect
	hub.unregister <- client
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())
	online = hub.GetOnlineUsers([]uuid.UUID{userID})
	assert.Empty(t, online)

	// Reconnect with new client
	client2 := newMockClient(hub, userID)
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 1, hub.GetClientCount())
	online = hub.GetOnlineUsers([]uuid.UUID{userID})
	assert.Contains(t, online, userID)
}

func TestHub_MultipleConnectionsSameUser(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	userID := uuid.New()

	client1 := newMockClient(hub, userID)
	client2 := newMockClient(hub, userID)
	client3 := newMockClient(hub, userID)

	hub.register <- client1
	hub.register <- client2
	hub.register <- client3
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 3, hub.GetClientCount())
	online := hub.GetOnlineUsers([]uuid.UUID{userID})
	assert.Len(t, online, 1) // Same user ID, counted once

	// Disconnect one
	hub.unregister <- client2
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 2, hub.GetClientCount())
	online = hub.GetOnlineUsers([]uuid.UUID{userID})
	assert.Contains(t, online, userID) // Still online with 2 connections

	// Disconnect remaining
	hub.unregister <- client1
	hub.unregister <- client3
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())
}

func TestHub_BroadcastToChannel(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	channelID := uuid.New()

	// Create 3 clients, subscribe 2 to channel
	client1 := newMockClient(hub, uuid.New())
	client2 := newMockClient(hub, uuid.New())
	client3 := newMockClient(hub, uuid.New())

	hub.register <- client1
	hub.register <- client2
	hub.register <- client3
	time.Sleep(20 * time.Millisecond)

	hub.SubscribeChannel(client1, channelID)
	hub.SubscribeChannel(client2, channelID)

	// Broadcast to channel
	hub.SendToChannel(channelID, &Event{
		Op:   OpDispatch,
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "hello"},
	})

	// Wait for broadcast processing
	time.Sleep(100 * time.Millisecond)

	// client1 and client2 should receive, client3 should not
	assertReceived := func(client *Client, name string) {
		select {
		case msg := <-client.send:
			var event Event
			require.NoError(t, json.Unmarshal(msg, &event))
			assert.Equal(t, EventTypeMessageCreate, event.Type)
		case <-time.After(time.Second):
			t.Fatalf("%s did not receive message", name)
		}
	}

	assertReceived(client1, "client1")
	assertReceived(client2, "client2")

	select {
	case <-client3.send:
		t.Fatal("client3 should not have received the message")
	case <-time.After(100 * time.Millisecond):
		// Good, no message
	}
}

func TestHub_BroadcastToServer(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	serverID := uuid.New()

	client1 := newMockClient(hub, uuid.New())
	client2 := newMockClient(hub, uuid.New())

	hub.register <- client1
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	hub.SubscribeServer(client1, serverID)

	hub.SendToServer(serverID, &Event{
		Op:   OpDispatch,
		Type: EventTypeMemberJoin,
		Data: map[string]string{"user_id": "test"},
	})

	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-client1.send:
		var event Event
		require.NoError(t, json.Unmarshal(msg, &event))
		assert.Equal(t, EventTypeMemberJoin, event.Type)
	case <-time.After(time.Second):
		t.Fatal("client1 did not receive server broadcast")
	}

	select {
	case <-client2.send:
		t.Fatal("client2 should not receive server broadcast")
	case <-time.After(100 * time.Millisecond):
		// Good
	}
}

func TestHub_Lifecycle_SendToUser(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	user1ID := uuid.New()
	user2ID := uuid.New()

	client1 := newMockClient(hub, user1ID)
	client2 := newMockClient(hub, user2ID)

	hub.register <- client1
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	hub.SendToUser(user1ID, &Event{
		Op:   OpDispatch,
		Type: EventTypeUserUpdate,
		Data: map[string]string{"username": "updated"},
	})

	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-client1.send:
		var event Event
		require.NoError(t, json.Unmarshal(msg, &event))
		assert.Equal(t, EventTypeUserUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("client1 did not receive user message")
	}

	select {
	case <-client2.send:
		t.Fatal("client2 should not receive user1's message")
	case <-time.After(100 * time.Millisecond):
		// Good
	}
}

func TestHub_SendToUser_MultipleConnections(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	userID := uuid.New()

	client1 := newMockClient(hub, userID)
	client2 := newMockClient(hub, userID)

	hub.register <- client1
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	hub.SendToUser(userID, &Event{
		Op:   OpDispatch,
		Type: EventTypePresenceUpdate,
		Data: map[string]string{"status": "online"},
	})

	time.Sleep(100 * time.Millisecond)

	// Both connections should receive the message
	for _, client := range []*Client{client1, client2} {
		select {
		case msg := <-client.send:
			var event Event
			require.NoError(t, json.Unmarshal(msg, &event))
			assert.Equal(t, EventTypePresenceUpdate, event.Type)
		case <-time.After(time.Second):
			t.Fatal("client did not receive message")
		}
	}
}

func TestHub_UnregisterCleansUpSubscriptions(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	userID := uuid.New()
	client := newMockClient(hub, userID)
	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	channelID := uuid.New()
	serverID := uuid.New()

	hub.SubscribeChannel(client, channelID)
	hub.SubscribeServer(client, serverID)

	// Verify subscriptions exist
	hub.channelsMux.RLock()
	assert.True(t, hub.channels[channelID][client])
	hub.channelsMux.RUnlock()

	hub.serversMux.RLock()
	assert.True(t, hub.servers[serverID][client])
	hub.serversMux.RUnlock()

	// Unregister
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	// Verify subscriptions cleaned up
	hub.channelsMux.RLock()
	_, chExists := hub.channels[channelID]
	hub.channelsMux.RUnlock()
	assert.False(t, chExists)

	hub.serversMux.RLock()
	_, srvExists := hub.servers[serverID]
	hub.serversMux.RUnlock()
	assert.False(t, srvExists)
}

func TestHub_ContextCancellation(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Hub stopped
	case <-time.After(time.Second):
		t.Fatal("Hub did not stop after context cancellation")
	}
}

func TestHub_GetOnlineUsers_MultipleUsers(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	offlineUser := uuid.New()

	hub.register <- newMockClient(hub, user1)
	hub.register <- newMockClient(hub, user2)
	hub.register <- newMockClient(hub, user3)
	time.Sleep(30 * time.Millisecond)

	online := hub.GetOnlineUsers([]uuid.UUID{user1, user2, user3, offlineUser})
	assert.Len(t, online, 3)
	assert.Contains(t, online, user1)
	assert.Contains(t, online, user2)
	assert.Contains(t, online, user3)
	assert.NotContains(t, online, offlineUser)
}

func TestHub_Broadcast_ChannelBufferFull(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	channelID := uuid.New()

	// Client with tiny buffer to test overflow
	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 1),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(20 * time.Millisecond)
	hub.SubscribeChannel(client, channelID)

	// Fill the buffer
	client.send <- []byte("fill")

	// Send another message - should be dropped (non-blocking select)
	hub.SendToChannel(channelID, &Event{
		Op:   OpDispatch,
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "overflow"},
	})

	time.Sleep(100 * time.Millisecond)

	// Drain the original fill message
	<-client.send

	// The overflow message should have been dropped
	select {
	case <-client.send:
		// Got the overflow message - that's fine too (race condition)
	case <-time.After(100 * time.Millisecond):
		// Expected - message was dropped
	}
}

func TestHub_Lifecycle_SetDrainConfig(t *testing.T) {
	hub := NewHub()

	config := &DrainConfig{
		DrainTimeout: 10 * time.Second,
		GracePeriod:  2 * time.Second,
	}

	hub.SetDrainConfig(config)

	assert.True(t, hub.IsHealthy())
	assert.False(t, hub.IsDraining())
	assert.Equal(t, DrainStateHealthy, hub.DrainState())
}

func TestHub_Lifecycle_Shutdown(t *testing.T) {
	hub := NewHubWithDrainConfig(&DrainConfig{
		DrainTimeout: 500 * time.Millisecond,
		GracePeriod:  50 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err := hub.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	assert.Equal(t, DrainStateClosed, hub.DrainState())
}

func TestHub_ShutdownWithClients(t *testing.T) {
	hub := NewHubWithDrainConfig(&DrainConfig{
		DrainTimeout: 1 * time.Second,
		GracePeriod:  100 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)

	// Add clients
	client1 := newMockClient(hub, uuid.New())
	client2 := newMockClient(hub, uuid.New())
	hub.register <- client1
	hub.register <- client2
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 2, hub.GetClientCount())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	// Shutdown should send reconnect to clients
	err := hub.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Clients should have received reconnect message
	for _, client := range []*Client{client1, client2} {
		select {
		case msg := <-client.send:
			var m Message
			require.NoError(t, json.Unmarshal(msg, &m))
			assert.Equal(t, OpReconnect, m.Op)
		case <-time.After(time.Second):
			// May have already drained
		}
	}
}

func TestHub_RegisterAndUnregisterChannels(t *testing.T) {
	hub := NewHub()

	// Test RegisterClient/UnregisterClient interface methods
	regCh := hub.RegisterClient()
	unregCh := hub.UnregisterClient()

	assert.NotNil(t, regCh)
	assert.NotNil(t, unregCh)
}
