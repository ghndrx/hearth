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
	"hearth/internal/pubsub"
	"hearth/internal/services"
)

func TestDistributedEventBridgeCreation(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "test-bridge-create")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	bus := events.NewBus()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bridge := NewDistributedEventBridge(ctx, dh, bus)
	require.NotNil(t, bridge)
}

func TestDistributedBridgeMessageCreated(t *testing.T) {
	skipIfNoRedis(t)

	// Create two instances to verify cross-instance delivery
	ps1, err := pubsub.New(getRedisURL(), "bridge-msg-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := pubsub.New(getRedisURL(), "bridge-msg-2")
	require.NoError(t, err)
	defer ps2.Close()

	dh1 := NewDistributedHub(ps1)
	dh2 := NewDistributedHub(ps2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	go dh1.Run(ctx1)
	go dh2.Run(ctx2)

	time.Sleep(200 * time.Millisecond)

	// Create event bus on instance 1
	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx1, dh1, bus)

	// Create client on instance 2
	channelID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh2.Hub, userID)

	dh2.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh2.SubscribeChannel(client, channelID)
	time.Sleep(200 * time.Millisecond)

	// Publish message created event on instance 1
	now := time.Now()
	bus.Publish(events.MessageCreated, &services.MessageCreatedEvent{
		Message: &models.Message{
			ID:        uuid.New(),
			ChannelID: channelID,
			AuthorID:  uuid.New(),
			Content:   "Hello from distributed test!",
			CreatedAt: now,
		},
		ChannelID: channelID,
	})

	// Verify client on instance 2 receives the message
	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMessageCreate, event.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for cross-instance message")
	}
}

func TestDistributedBridgeTypingStarted(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := pubsub.New(getRedisURL(), "bridge-typing-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := pubsub.New(getRedisURL(), "bridge-typing-2")
	require.NoError(t, err)
	defer ps2.Close()

	dh1 := NewDistributedHub(ps1)
	dh2 := NewDistributedHub(ps2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	go dh1.Run(ctx1)
	go dh2.Run(ctx2)

	time.Sleep(200 * time.Millisecond)

	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx1, dh1, bus)

	channelID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh2.Hub, userID)

	dh2.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh2.SubscribeChannel(client, channelID)
	time.Sleep(200 * time.Millisecond)

	// Publish typing event
	bus.Publish(events.TypingStarted, &TypingEventData{
		ChannelID: channelID,
		UserID:    uuid.New(),
	})

	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeTypingStart, event.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for typing event")
	}
}

func TestDistributedBridgeMemberJoined(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := pubsub.New(getRedisURL(), "bridge-member-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := pubsub.New(getRedisURL(), "bridge-member-2")
	require.NoError(t, err)
	defer ps2.Close()

	dh1 := NewDistributedHub(ps1)
	dh2 := NewDistributedHub(ps2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	go dh1.Run(ctx1)
	go dh2.Run(ctx2)

	time.Sleep(200 * time.Millisecond)

	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx1, dh1, bus)

	serverID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh2.Hub, userID)

	dh2.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh2.SubscribeServer(client, serverID)
	time.Sleep(200 * time.Millisecond)

	// Publish member joined event
	newMemberID := uuid.New()
	bus.Publish(events.MemberJoined, &MemberEventData{
		ServerID: serverID,
		UserID:   newMemberID,
		User: &models.User{
			ID:       newMemberID,
			Username: "newuser",
		},
		Member: &models.Member{
			UserID:   newMemberID,
			ServerID: serverID,
			JoinedAt: time.Now(),
		},
	})

	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeMemberJoin, event.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for member join event")
	}
}

func TestDistributedBridgeServerUpdate(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "bridge-server-update")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dh.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx, dh, bus)

	serverID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh.Hub, userID)

	dh.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	// Publish server update
	bus.Publish(events.ServerUpdated, &ServerEventData{
		Server: &models.Server{
			ID:      serverID,
			Name:    "Updated Server Name",
			OwnerID: userID,
		},
	})

	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeServerUpdate, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for server update event")
	}
}

func TestDistributedBridgeChannelCreate(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "bridge-channel-create")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dh.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx, dh, bus)

	serverID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh.Hub, userID)

	dh.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	// Publish channel create
	channelID := uuid.New()
	bus.Publish(events.ChannelCreated, &ChannelEventData{
		Channel: &models.Channel{
			ID:       channelID,
			Name:     "new-channel",
			ServerID: &serverID,
			Type:     models.ChannelTypeText,
		},
		ServerID: serverID,
	})

	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
		assert.Equal(t, EventTypeChannelCreate, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for channel create event")
	}
}

// TestMessageDeliveryLatency measures the latency of cross-instance message delivery
func TestMessageDeliveryLatency(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := pubsub.New(getRedisURL(), "latency-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := pubsub.New(getRedisURL(), "latency-2")
	require.NoError(t, err)
	defer ps2.Close()

	dh1 := NewDistributedHub(ps1)
	dh2 := NewDistributedHub(ps2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	go dh1.Run(ctx1)
	go dh2.Run(ctx2)

	time.Sleep(200 * time.Millisecond)

	bus := events.NewBus()
	_ = NewDistributedEventBridge(ctx1, dh1, bus)

	channelID := uuid.New()
	userID := uuid.New()
	client := newMockClient(dh2.Hub, userID)

	dh2.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	dh2.SubscribeChannel(client, channelID)
	time.Sleep(200 * time.Millisecond)

	// Measure latency
	iterations := 10
	var totalLatency time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		bus.Publish(events.MessageCreated, &services.MessageCreatedEvent{
			Message: &models.Message{
				ID:        uuid.New(),
				ChannelID: channelID,
				AuthorID:  uuid.New(),
				Content:   "Latency test message",
				CreatedAt: time.Now(),
			},
			ChannelID: channelID,
		})

		select {
		case <-client.send:
			latency := time.Since(start)
			totalLatency += latency
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout on iteration %d", i)
		}
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("Average cross-instance latency: %v", avgLatency)

	// Target is <100ms
	assert.Less(t, avgLatency, 100*time.Millisecond,
		"Average latency should be less than 100ms, got %v", avgLatency)
}
