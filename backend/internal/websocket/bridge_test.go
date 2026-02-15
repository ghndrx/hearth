package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/events"
	"hearth/internal/models"
	"hearth/internal/services"
)

func TestNewEventBridge(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()

	bridge := NewEventBridge(hub, bus)

	require.NotNil(t, bridge)
	assert.NotNil(t, bridge.hub)
	assert.NotNil(t, bridge.bus)
}

func TestEventBridge_sendToChannel_MarshalError(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	channelID := uuid.New()
	// Try to marshal a channel that can't be marshaled
	badData := make(chan int)

	// Should not panic
	bridge.sendToChannel(channelID, "TEST", badData)
}

func TestEventBridge_sendToServer_MarshalError(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	serverID := uuid.New()
	badData := make(chan int)

	bridge.sendToServer(serverID, "TEST", badData)
}

func TestEventBridge_sendToUser_MarshalError(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	userID := uuid.New()
	badData := make(chan int)

	bridge.sendToUser(userID, "TEST", badData)
}

func TestEventBridge_onMessageCreated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

	userID := uuid.New()
	channelID := uuid.New()
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
	hub.SubscribeChannel(client, channelID)

	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		ServerID:  &serverID,
		AuthorID:  userID,
		Content:   "Test message",
		CreatedAt: time.Now(),
	}

	bus.Publish(events.MessageCreated, &services.MessageCreatedEvent{
		Message:   msg,
		ChannelID: channelID,
		ServerID:  &serverID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMessageCreate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive message event")
	}
}

func TestEventBridge_onMessageUpdated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   "Updated content",
		CreatedAt: time.Now(),
	}

	bus.Publish(events.MessageUpdated, &services.MessageUpdatedEvent{
		Message:   msg,
		ChannelID: channelID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMessageUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive message update event")
	}
}

func TestEventBridge_onMessageDeleted(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.MessageDeleted, &services.MessageDeletedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
		AuthorID:  userID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMessageDelete, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive message delete event")
	}
}

func TestEventBridge_onMessagePinned(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.MessagePinned, &services.MessagePinnedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
		PinnedBy:  userID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeChannelPinsUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive channel pins update event")
	}
}

func TestEventBridge_onReactionAdded(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.ReactionAdded, &ReactionEventData{
		MessageID: uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		Emoji:     "👍",
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeReactionAdd, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive reaction add event")
	}
}

func TestEventBridge_onReactionRemoved(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.ReactionRemoved, &ReactionEventData{
		MessageID: uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		Emoji:     "👎",
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeReactionRemove, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive reaction remove event")
	}
}

func TestEventBridge_onChannelCreated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	channel := &models.Channel{
		ID:       uuid.New(),
		ServerID: &serverID,
		Name:     "general",
		Type:     "text",
	}

	bus.Publish(events.ChannelCreated, &ChannelEventData{
		Channel:  channel,
		ServerID: serverID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeChannelCreate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive channel create event")
	}
}

func TestEventBridge_onChannelUpdated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	channel := &models.Channel{
		ID:       uuid.New(),
		ServerID: &serverID,
		Name:     "general-updated",
		Type:     "text",
	}

	bus.Publish(events.ChannelUpdated, &ChannelEventData{
		Channel:  channel,
		ServerID: serverID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeChannelUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive channel update event")
	}
}

func TestEventBridge_onChannelDeleted(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	channel := &models.Channel{
		ID:       uuid.New(),
		ServerID: &serverID,
		Name:     "deleted-channel",
		Type:     "text",
	}

	bus.Publish(events.ChannelDeleted, &ChannelEventData{
		Channel:  channel,
		ServerID: serverID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeChannelDelete, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive channel delete event")
	}
}

func TestEventBridge_onServerUpdated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	server := &models.Server{
		ID:      serverID,
		Name:    "Updated Server",
		OwnerID: userID,
	}

	bus.Publish(events.ServerUpdated, &ServerEventData{
		Server: server,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeServerUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive server update event")
	}
}

func TestEventBridge_onServerDeleted(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	server := &models.Server{
		ID:      serverID,
		Name:    "Deleted Server",
		OwnerID: userID,
	}

	bus.Publish(events.ServerDeleted, &ServerEventData{
		Server: server,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeServerDelete, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive server delete event")
	}
}

func TestEventBridge_onMemberJoined(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.MemberJoined, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
		User: &models.User{
			ID:       uuid.New(),
			Username: "newmember",
		},
		Member: &models.Member{
			JoinedAt: time.Now(),
		},
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMemberJoin, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive member join event")
	}
}

func TestEventBridge_onMemberLeft(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.MemberLeft, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
		User: &models.User{
			ID:       uuid.New(),
			Username: "leftmember",
		},
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMemberLeave, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive member leave event")
	}
}

func TestEventBridge_onMemberBanned(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.MemberBanned, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
		User: &models.User{
			ID:       uuid.New(),
			Username: "bannedmember",
		},
		Reason: "Spam",
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeBanAdd, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive ban add event")
	}
}

func TestEventBridge_onUserUpdated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.UserUpdated, &UserEventData{
		User: &models.User{
			ID:       userID,
			Username: "updateduser",
		},
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeUserUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive user update event")
	}
}

func TestEventBridge_onPresenceUpdate(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	bus.Publish(events.PresenceUpdate, &PresenceEventData{
		UserID:     userID,
		Status:     "online",
		Activities: []string{},
		ServerIDs:  []uuid.UUID{serverID},
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypePresenceUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive presence update event")
	}
}

func TestEventBridge_onTypingStarted(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

	userID := uuid.New()
	channelID := uuid.New()
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
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.TypingStarted, &TypingEventData{
		ChannelID: channelID,
		ServerID:  &serverID,
		UserID:    userID,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeTypingStart, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive typing start event")
	}
}

func TestEventBridge_onMemberUpdated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	nickname := "NewNick"
	bus.Publish(events.MemberUpdated, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
		User: &models.User{
			ID:       uuid.New(),
			Username: "member",
		},
		Member: &models.Member{
			Nickname: &nickname,
		},
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMemberUpdate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive member update event")
	}
}

func TestEventBridge_onServerCreated(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

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

	serverID := uuid.New()
	server := &models.Server{
		ID:      serverID,
		Name:    "New Server",
		OwnerID: userID,
	}

	bus.Publish(events.ServerCreated, &ServerEventData{
		Server: server,
	})

	select {
	case data := <-client.send:
		var event Event
		err := json.Unmarshal(data, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeServerCreate, event.Type)
	case <-time.After(time.Second):
		t.Fatal("Did not receive server create event")
	}
}

func TestEventBridge_ConversionHelpers(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("messageToWS with nil", func(t *testing.T) {
		result := bridge.messageToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("channelToWS with nil", func(t *testing.T) {
		result := bridge.channelToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("serverToWS with nil", func(t *testing.T) {
		result := bridge.serverToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("userToWS with nil", func(t *testing.T) {
		result := bridge.userToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("publicUserToWS with nil", func(t *testing.T) {
		result := bridge.publicUserToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("messageToWS with valid message", func(t *testing.T) {
		serverID := uuid.New()
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			ServerID:  &serverID,
			AuthorID:  uuid.New(),
			Content:   "Test content",
			CreatedAt: time.Now(),
			EditedAt:  &[]time.Time{time.Now()}[0],
			Pinned:    true,
			TTS:       false,
			Author: &models.PublicUser{
				ID:       uuid.New(),
				Username: "author",
			},
		}

		result := bridge.messageToWS(msg)
		require.NotNil(t, result)
		assert.Equal(t, msg.ID.String(), result["id"])
		assert.Equal(t, msg.Content, result["content"])
		assert.Equal(t, true, result["pinned"])
		assert.NotNil(t, result["edited_timestamp"])
	})

	t.Run("channelToWS with valid channel", func(t *testing.T) {
		serverID := uuid.New()
		parentID := uuid.New()
		channel := &models.Channel{
			ID:       uuid.New(),
			ServerID: &serverID,
			Name:     "general",
			Type:     "text",
			Topic:    "Test topic",
			Position: 1,
			ParentID: &parentID,
		}

		result := bridge.channelToWS(channel)
		require.NotNil(t, result)
		assert.Equal(t, channel.ID.String(), result["id"])
		assert.Equal(t, channel.Name, result["name"])
		assert.Equal(t, channel.Topic, result["topic"])
		assert.Equal(t, channel.Position, result["position"])
	})

	t.Run("serverToWS with valid server", func(t *testing.T) {
		ownerID := uuid.New()
		iconURL := "https://example.com/icon.png"
		bannerURL := "https://example.com/banner.png"
		description := "Test server"

		server := &models.Server{
			ID:          uuid.New(),
			Name:        "Test Server",
			OwnerID:     ownerID,
			IconURL:     &iconURL,
			BannerURL:   &bannerURL,
			Description: &description,
		}

		result := bridge.serverToWS(server)
		require.NotNil(t, result)
		assert.Equal(t, server.ID.String(), result["id"])
		assert.Equal(t, server.Name, result["name"])
		assert.Equal(t, iconURL, result["icon"])
		assert.Equal(t, bannerURL, result["banner"])
		assert.Equal(t, description, result["description"])
	})

	t.Run("userToWS with valid user", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.png"
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
			AvatarURL:     &avatarURL,
		}

		result := bridge.userToWS(user)
		require.NotNil(t, result)
		assert.Equal(t, user.ID.String(), result["id"])
		assert.Equal(t, user.Username, result["username"])
		assert.Equal(t, user.Discriminator, result["discriminator"])
		assert.Equal(t, avatarURL, result["avatar"])
	})

	t.Run("publicUserToWS with valid public user", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.png"
		user := &models.PublicUser{
			ID:            uuid.New(),
			Username:      "publicuser",
			Discriminator: "1234",
			AvatarURL:     &avatarURL,
		}

		result := bridge.publicUserToWS(user)
		require.NotNil(t, result)
		assert.Equal(t, user.ID.String(), result["id"])
		assert.Equal(t, user.Username, result["username"])
	})
}

func TestEventBridge_EventDataStructures(t *testing.T) {
	t.Run("MessageCreatedEvent", func(t *testing.T) {
		msgID := uuid.New()
		channelID := uuid.New()
		serverID := uuid.New()

		data := services.MessageCreatedEvent{
			Message: &models.Message{
				ID: msgID,
			},
			ChannelID: channelID,
			ServerID:  &serverID,
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded services.MessageCreatedEvent
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, channelID, decoded.ChannelID)
	})

	t.Run("ReactionEventData", func(t *testing.T) {
		data := ReactionEventData{
			MessageID: uuid.New(),
			ChannelID: uuid.New(),
			UserID:    uuid.New(),
			Emoji:     "🔥",
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded ReactionEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, data.Emoji, decoded.Emoji)
	})

	t.Run("ChannelEventData", func(t *testing.T) {
		serverID := uuid.New()
		data := ChannelEventData{
			Channel: &models.Channel{
				ID:   uuid.New(),
				Name: "general",
			},
			ServerID: serverID,
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded ChannelEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, serverID, decoded.ServerID)
	})

	t.Run("MemberEventData", func(t *testing.T) {
		data := MemberEventData{
			ServerID: uuid.New(),
			UserID:   uuid.New(),
			Reason:   "Test",
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded MemberEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, data.Reason, decoded.Reason)
	})

	t.Run("UserEventData", func(t *testing.T) {
		data := UserEventData{
			User: &models.User{
				ID:       uuid.New(),
				Username: "test",
			},
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded UserEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, data.User.Username, decoded.User.Username)
	})

	t.Run("PresenceEventData", func(t *testing.T) {
		data := PresenceEventData{
			UserID:     uuid.New(),
			Status:     "online",
			Activities: []string{"gaming"},
			ServerIDs:  []uuid.UUID{uuid.New()},
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded PresenceEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, data.Status, decoded.Status)
	})

	t.Run("TypingEventData", func(t *testing.T) {
		channelID := uuid.New()
		serverID := uuid.New()
		data := TypingEventData{
			ChannelID: channelID,
			ServerID:  &serverID,
			UserID:    uuid.New(),
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded TypingEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, channelID, decoded.ChannelID)
	})

	t.Run("ServerEventData", func(t *testing.T) {
		data := ServerEventData{
			Server: &models.Server{
				ID:   uuid.New(),
				Name: "Test",
			},
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)

		var decoded ServerEventData
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		assert.Equal(t, data.Server.Name, decoded.Server.Name)
	})
}

func TestEventBridge_InvalidEventData(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	// Test with wrong data type for each handler
	t.Run("onMessageCreated with invalid data", func(t *testing.T) {
		// Should not panic with wrong data type
		bridge.onMessageCreated(events.Event{
			Type: events.MessageCreated,
			Data: "invalid data",
		})
	})

	t.Run("onMessageUpdated with invalid data", func(t *testing.T) {
		bridge.onMessageUpdated(events.Event{
			Type: events.MessageUpdated,
			Data: 123,
		})
	})

	t.Run("onMessageDeleted with invalid data", func(t *testing.T) {
		bridge.onMessageDeleted(events.Event{
			Type: events.MessageDeleted,
			Data: nil,
		})
	})

	t.Run("onMessagePinned with invalid data", func(t *testing.T) {
		bridge.onMessagePinned(events.Event{
			Type: events.MessagePinned,
			Data: map[string]string{},
		})
	})

	t.Run("onReactionAdded with invalid data", func(t *testing.T) {
		bridge.onReactionAdded(events.Event{
			Type: events.ReactionAdded,
			Data: "invalid",
		})
	})

	t.Run("onReactionRemoved with invalid data", func(t *testing.T) {
		bridge.onReactionRemoved(events.Event{
			Type: events.ReactionRemoved,
			Data: 456,
		})
	})

	t.Run("onChannelCreated with invalid data", func(t *testing.T) {
		bridge.onChannelCreated(events.Event{
			Type: events.ChannelCreated,
			Data: nil,
		})
	})

	t.Run("onChannelUpdated with invalid data", func(t *testing.T) {
		bridge.onChannelUpdated(events.Event{
			Type: events.ChannelUpdated,
			Data: "invalid",
		})
	})

	t.Run("onChannelDeleted with invalid data", func(t *testing.T) {
		bridge.onChannelDeleted(events.Event{
			Type: events.ChannelDeleted,
			Data: 789,
		})
	})

	t.Run("onServerCreated with invalid data", func(t *testing.T) {
		bridge.onServerCreated(events.Event{
			Type: events.ServerCreated,
			Data: nil,
		})
	})

	t.Run("onServerUpdated with invalid data", func(t *testing.T) {
		bridge.onServerUpdated(events.Event{
			Type: events.ServerUpdated,
			Data: "invalid",
		})
	})

	t.Run("onServerDeleted with invalid data", func(t *testing.T) {
		bridge.onServerDeleted(events.Event{
			Type: events.ServerDeleted,
			Data: []int{1, 2, 3},
		})
	})

	t.Run("onMemberJoined with invalid data", func(t *testing.T) {
		bridge.onMemberJoined(events.Event{
			Type: events.MemberJoined,
			Data: nil,
		})
	})

	t.Run("onMemberLeft with invalid data", func(t *testing.T) {
		bridge.onMemberLeft(events.Event{
			Type: events.MemberLeft,
			Data: "invalid",
		})
	})

	t.Run("onMemberUpdated with invalid data", func(t *testing.T) {
		bridge.onMemberUpdated(events.Event{
			Type: events.MemberUpdated,
			Data: 12345,
		})
	})

	t.Run("onMemberKicked with invalid data", func(t *testing.T) {
		bridge.onMemberKicked(events.Event{
			Type: events.MemberKicked,
			Data: nil,
		})
	})

	t.Run("onMemberBanned with invalid data", func(t *testing.T) {
		bridge.onMemberBanned(events.Event{
			Type: events.MemberBanned,
			Data: "invalid",
		})
	})

	t.Run("onUserUpdated with invalid data", func(t *testing.T) {
		bridge.onUserUpdated(events.Event{
			Type: events.UserUpdated,
			Data: 999,
		})
	})

	t.Run("onPresenceUpdate with invalid data", func(t *testing.T) {
		bridge.onPresenceUpdate(events.Event{
			Type: events.PresenceUpdate,
			Data: nil,
		})
	})

	t.Run("onTypingStarted with invalid data", func(t *testing.T) {
		bridge.onTypingStarted(events.Event{
			Type: events.TypingStarted,
			Data: "invalid",
		})
	})
}

func TestEventBridge_buildMemberData(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	serverID := uuid.New()
	userID := uuid.New()
	joinedAt := time.Now()

	t.Run("with full member data including joined at", func(t *testing.T) {
		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
			User: &models.User{
				ID:       userID,
				Username: "testuser",
			},
			Member: &models.Member{
				JoinedAt: joinedAt,
			},
		}

		result := bridge.buildMemberData(data, true)

		assert.Equal(t, serverID.String(), result["guild_id"])
		assert.NotNil(t, result["user"])
		assert.NotNil(t, result["joined_at"])
	})

	t.Run("without joined at", func(t *testing.T) {
		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
			User: &models.User{
				ID:       userID,
				Username: "testuser",
			},
		}

		result := bridge.buildMemberData(data, false)

		assert.Equal(t, serverID.String(), result["guild_id"])
		assert.NotNil(t, result["user"])
		assert.Nil(t, result["joined_at"])
	})

	t.Run("with nickname", func(t *testing.T) {
		nickname := "CoolNick"
		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
			Member: &models.Member{
				Nickname: &nickname,
			},
		}

		result := bridge.buildMemberData(data, false)
		// Note: nickname is added separately in onMemberUpdated, not in buildMemberData
		// buildMemberData returns base data, handlers add specific fields
		assert.Equal(t, serverID.String(), result["guild_id"])
		assert.Nil(t, result["nick"]) // nick is not added by buildMemberData
	})

	t.Run("without user, only userID", func(t *testing.T) {
		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
		}

		result := bridge.buildMemberData(data, false)

		userData, ok := result["user"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, userID.String(), userData["id"])
	})
}

func TestEventBridge_Constants(t *testing.T) {
	assert.Equal(t, "CHANNEL_PINS_UPDATE", EventTypeChannelPinsUpdate)
	assert.Equal(t, "GUILD_BAN_ADD", EventTypeBanAdd)
	assert.Equal(t, "USER_UPDATE", EventTypeUserUpdate)
}
