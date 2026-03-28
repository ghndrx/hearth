package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.channels)
	assert.NotNil(t, hub.servers)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_RegisterClient(t *testing.T) {
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

	hub.clientsMux.RLock()
	defer hub.clientsMux.RUnlock()

	assert.NotNil(t, hub.clients[userID])
	assert.True(t, hub.clients[userID][client])
}

func TestHub_RegisterMultipleClientsForSameUser(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	client1 := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client2 := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	hub.clientsMux.RLock()
	defer hub.clientsMux.RUnlock()

	assert.Len(t, hub.clients[userID], 2)
	assert.True(t, hub.clients[userID][client1])
	assert.True(t, hub.clients[userID][client2])
}

func TestHub_UnregisterClient(t *testing.T) {
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
	hub.unregisterClient(client)

	hub.clientsMux.RLock()
	defer hub.clientsMux.RUnlock()

	_, exists := hub.clients[userID]
	assert.False(t, exists)
}

func TestHub_UnregisterClientWithChannelSubscriptions(t *testing.T) {
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

	hub.registerClient(client)
	hub.SubscribeChannel(client, channelID)

	hub.channelsMux.RLock()
	assert.True(t, hub.channels[channelID][client])
	hub.channelsMux.RUnlock()

	hub.unregisterClient(client)

	hub.channelsMux.RLock()
	defer hub.channelsMux.RUnlock()

	_, exists := hub.channels[channelID]
	assert.False(t, exists)
}

func TestHub_UnregisterClientWithServerSubscriptions(t *testing.T) {
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

	hub.registerClient(client)
	hub.SubscribeServer(client, serverID)

	hub.serversMux.RLock()
	assert.True(t, hub.servers[serverID][client])
	hub.serversMux.RUnlock()

	hub.unregisterClient(client)

	hub.serversMux.RLock()
	defer hub.serversMux.RUnlock()

	_, exists := hub.servers[serverID]
	assert.False(t, exists)
}

func TestHub_SubscribeChannel(t *testing.T) {
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

	hub.SubscribeChannel(client, channelID)

	hub.channelsMux.RLock()
	defer hub.channelsMux.RUnlock()

	assert.NotNil(t, hub.channels[channelID])
	assert.True(t, hub.channels[channelID][client])
}

func TestHub_UnsubscribeChannel(t *testing.T) {
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

	hub.SubscribeChannel(client, channelID)
	hub.UnsubscribeChannel(client, channelID)

	hub.channelsMux.RLock()
	defer hub.channelsMux.RUnlock()

	_, exists := hub.channels[channelID]
	assert.False(t, exists)
}

func TestHub_SubscribeServer(t *testing.T) {
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

	hub.SubscribeServer(client, serverID)

	hub.serversMux.RLock()
	defer hub.serversMux.RUnlock()

	assert.NotNil(t, hub.servers[serverID])
	assert.True(t, hub.servers[serverID][client])
}

func TestHub_GetOnlineUsers(t *testing.T) {
	hub := NewHub()

	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	client1 := &Client{
		ID:       uuid.New().String(),
		UserID:   user1,
		Username: "user1",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	client2 := &Client{
		ID:       uuid.New().String(),
		UserID:   user2,
		Username: "user2",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	online := hub.GetOnlineUsers([]uuid.UUID{user1, user2, user3})

	assert.Len(t, online, 2)
	assert.Contains(t, online, user1)
	assert.Contains(t, online, user2)
	assert.NotContains(t, online, user3)
}

func TestHub_GetOnlineUsersEmpty(t *testing.T) {
	hub := NewHub()

	user1 := uuid.New()

	online := hub.GetOnlineUsers([]uuid.UUID{user1})

	assert.Len(t, online, 0)
}

func TestHub_RunContextCancellation(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Success - Run exited
	case <-time.After(time.Second):
		t.Fatal("Hub.Run did not exit after context cancellation")
	}
}

func TestHub_RunRegisterUnregister(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

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

	// Register via channel
	hub.register <- client

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	hub.clientsMux.RLock()
	registered := hub.clients[userID][client]
	hub.clientsMux.RUnlock()

	assert.True(t, registered)

	// Unregister via channel
	hub.unregister <- client

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	hub.clientsMux.RLock()
	_, exists := hub.clients[userID]
	hub.clientsMux.RUnlock()

	assert.False(t, exists)
}

func TestHub_SendToUser(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

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

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	event := &Event{
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "hello"},
	}

	hub.SendToUser(userID, event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive message")
	}
}

func TestHub_SendToChannel(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

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

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeChannel(client, channelID)

	event := &Event{
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "channel message"},
	}

	hub.SendToChannel(channelID, event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive channel message")
	}
}

func TestHub_SendToServer(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

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

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeServer(client, serverID)

	event := &Event{
		Type: EventTypeServerUpdate,
		Data: map[string]string{"name": "Test Server"},
	}

	hub.SendToServer(serverID, event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Did not receive server message")
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	channelID := uuid.New()

	// Create two clients in same channel
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

	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	hub.SubscribeChannel(client1, channelID)
	hub.SubscribeChannel(client2, channelID)

	event := &Event{
		Type: EventTypeMessageCreate,
		Data: map[string]string{"content": "broadcast"},
	}

	hub.SendToChannel(channelID, event)

	// Both should receive
	for _, client := range []*Client{client1, client2} {
		select {
		case msg := <-client.send:
			assert.Contains(t, string(msg), "broadcast")
		case <-time.After(time.Second):
			t.Fatal("Client did not receive broadcast")
		}
	}
}

func TestHubEventTypes(t *testing.T) {
	assert.Equal(t, "READY", EventTypeReady)
	assert.Equal(t, "MESSAGE_CREATE", EventTypeMessageCreate)
	assert.Equal(t, "MESSAGE_UPDATE", EventTypeMessageUpdate)
	assert.Equal(t, "MESSAGE_DELETE", EventTypeMessageDelete)
	assert.Equal(t, "TYPING_START", EventTypeTypingStart)
	assert.Equal(t, "PRESENCE_UPDATE", EventTypePresenceUpdate)
	assert.Equal(t, "SERVER_CREATE", EventTypeServerCreate)
	assert.Equal(t, "SERVER_UPDATE", EventTypeServerUpdate)
	assert.Equal(t, "SERVER_DELETE", EventTypeServerDelete)
	assert.Equal(t, "MEMBER_JOIN", EventTypeMemberJoin)
	assert.Equal(t, "MEMBER_LEAVE", EventTypeMemberLeave)
	assert.Equal(t, "MEMBER_UPDATE", EventTypeMemberUpdate)
	assert.Equal(t, "CHANNEL_CREATE", EventTypeChannelCreate)
	assert.Equal(t, "CHANNEL_UPDATE", EventTypeChannelUpdate)
	assert.Equal(t, "CHANNEL_DELETE", EventTypeChannelDelete)
	assert.Equal(t, "REACTION_ADD", EventTypeReactionAdd)
	assert.Equal(t, "REACTION_REMOVE", EventTypeReactionRemove)
	assert.Equal(t, "SUBSCRIBE", EventTypeSubscribe)
	assert.Equal(t, "UNSUBSCRIBE", EventTypeUnsubscribe)
}

func TestTypingData(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	data := TypingData{
		ChannelID: channelID,
		UserID:    userID,
	}

	assert.Equal(t, channelID, data.ChannelID)
	assert.Equal(t, userID, data.UserID)
}

func TestSubscribeData(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()

	data := SubscribeData{
		ChannelID: &channelID,
		ServerID:  &serverID,
	}

	assert.Equal(t, &channelID, data.ChannelID)
	assert.Equal(t, &serverID, data.ServerID)
}
