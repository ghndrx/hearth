package websocket

import (
	"context"
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

func newDistributedBridgeTestSetup(t *testing.T) (*DistributedEventBridge, *DistributedHub, *events.Bus, *Client, context.CancelFunc) {
	t.Helper()
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "bridge-test-"+uuid.New().String()[:8])
	require.NoError(t, err)

	dh := NewDistributedHub(ps)

	ctx, cancel := context.WithCancel(context.Background())
	go dh.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	bus := events.NewBus()
	bridge := NewDistributedEventBridge(ctx, dh, bus)

	// Create and register a client
	userID := uuid.New()
	client := newMockClient(dh.Hub, userID)
	dh.Hub.register <- client
	time.Sleep(50 * time.Millisecond)

	cleanup := func() {
		cancel()
		ps.Close()
	}

	return bridge, dh, bus, client, cleanup
}

func TestDistributedBridge_MessageUpdated(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	now := time.Now()
	bus.Publish(events.MessageUpdated, &services.MessageUpdatedEvent{
		Message: &models.Message{
			ID:        uuid.New(),
			ChannelID: channelID,
			AuthorID:  uuid.New(),
			Content:   "Updated content",
			CreatedAt: now,
		},
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message update event")
	}
}

func TestDistributedBridge_MessageDeleted(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.MessageDeleted, &services.MessageDeletedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_DELETE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message delete event")
	}
}

func TestDistributedBridge_MessagePinned(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.MessagePinned, &services.MessagePinnedEvent{
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "CHANNEL_PINS_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message pinned event")
	}
}

func TestDistributedBridge_ReactionAdded(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.ReactionAdded, &services.ReactionAddedEvent{
		ChannelID: channelID,
		MessageID: uuid.New(),
		UserID:    uuid.New(),
		Emoji:     "👍",
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "REACTION_ADD")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for reaction add event")
	}
}

func TestDistributedBridge_ReactionRemoved(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.ReactionRemoved, &services.ReactionRemovedEvent{
		ChannelID: channelID,
		MessageID: uuid.New(),
		UserID:    uuid.New(),
		Emoji:     "👍",
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "REACTION_REMOVE")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for reaction remove event")
	}
}

func TestDistributedBridge_ChannelUpdated(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.ChannelUpdated, &ChannelEventData{
		Channel: &models.Channel{
			ID:       uuid.New(),
			Name:     "updated-channel",
			ServerID: &serverID,
			Type:     models.ChannelTypeText,
		},
		ServerID: serverID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "CHANNEL_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for channel update event")
	}
}

func TestDistributedBridge_ChannelDeleted(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.ChannelDeleted, &ChannelEventData{
		Channel: &models.Channel{
			ID:       uuid.New(),
			Name:     "deleted-channel",
			ServerID: &serverID,
		},
		ServerID: serverID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "CHANNEL_DELETE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for channel delete event")
	}
}

func TestDistributedBridge_ServerCreated(t *testing.T) {
	_, _, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	// Server create is sent to the owner
	ownerID := client.UserID

	bus.Publish(events.ServerCreated, &ServerEventData{
		Server: &models.Server{
			ID:      uuid.New(),
			Name:    "New Server",
			OwnerID: ownerID,
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_CREATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for server create event")
	}
}

func TestDistributedBridge_ServerDeleted(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.ServerDeleted, &ServerEventData{
		Server: &models.Server{
			ID:      serverID,
			Name:    "Deleted Server",
			OwnerID: uuid.New(),
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_DELETE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for server delete event")
	}
}

func TestDistributedBridge_MemberLeft(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	memberID := uuid.New()
	bus.Publish(events.MemberLeft, &MemberEventData{
		ServerID: serverID,
		UserID:   memberID,
		User: &models.User{
			ID:       memberID,
			Username: "leavinguser",
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_LEAVE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for member left event")
	}
}

func TestDistributedBridge_MemberUpdated(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	memberID := uuid.New()
	nickname := "NewNick"
	bus.Publish(events.MemberUpdated, &MemberEventData{
		ServerID: serverID,
		UserID:   memberID,
		User: &models.User{
			ID:       memberID,
			Username: "updateduser",
		},
		Member: &models.Member{
			UserID:   memberID,
			ServerID: serverID,
			Nickname: &nickname,
			JoinedAt: time.Now(),
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for member update event")
	}
}

func TestDistributedBridge_MemberKicked(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	memberID := uuid.New()
	bus.Publish(events.MemberKicked, &MemberEventData{
		ServerID: serverID,
		UserID:   memberID,
		User: &models.User{
			ID:       memberID,
			Username: "kickeduser",
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_LEAVE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for member kicked event")
	}
}

func TestDistributedBridge_MemberBanned(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	memberID := uuid.New()
	bus.Publish(events.MemberBanned, &MemberEventData{
		ServerID: serverID,
		UserID:   memberID,
		User: &models.User{
			ID:       memberID,
			Username: "banneduser",
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "GUILD_BAN_ADD")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for member banned event")
	}
}

func TestDistributedBridge_UserUpdated(t *testing.T) {
	_, _, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	bus.Publish(events.UserUpdated, &UserEventData{
		User: &models.User{
			ID:       client.UserID,
			Username: "updatedusername",
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "USER_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for user update event")
	}
}

func TestDistributedBridge_PresenceUpdate(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	serverID := uuid.New()
	dh.SubscribeServer(client, serverID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.PresenceUpdate, &PresenceEventData{
		UserID:     uuid.New(),
		Status:     "online",
		Activities: []string{},
		ServerIDs:  []uuid.UUID{serverID},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "PRESENCE_UPDATE")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for presence update event")
	}
}

func TestDistributedBridge_WrongEventType(t *testing.T) {
	_, _, bus, _, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	// Publish with wrong data type - should not panic
	bus.Publish(events.MessageCreated, "wrong data type")
	bus.Publish(events.MessageUpdated, "wrong data type")
	bus.Publish(events.MessageDeleted, "wrong data type")
	bus.Publish(events.MessagePinned, "wrong data type")
	bus.Publish(events.ReactionAdded, "wrong data type")
	bus.Publish(events.ReactionRemoved, "wrong data type")
	bus.Publish(events.ChannelCreated, "wrong data type")
	bus.Publish(events.ChannelUpdated, "wrong data type")
	bus.Publish(events.ChannelDeleted, "wrong data type")
	bus.Publish(events.ServerCreated, "wrong data type")
	bus.Publish(events.ServerUpdated, "wrong data type")
	bus.Publish(events.ServerDeleted, "wrong data type")
	bus.Publish(events.MemberJoined, "wrong data type")
	bus.Publish(events.MemberLeft, "wrong data type")
	bus.Publish(events.MemberUpdated, "wrong data type")
	bus.Publish(events.MemberKicked, "wrong data type")
	bus.Publish(events.MemberBanned, "wrong data type")
	bus.Publish(events.UserUpdated, "wrong data type")
	bus.Publish(events.PresenceUpdate, "wrong data type")
	bus.Publish(events.TypingStarted, "wrong data type")

	time.Sleep(100 * time.Millisecond)
}

func TestDistributedBridge_ConversionHelpers(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "bridge-helpers-"+uuid.New().String()[:8])
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := events.NewBus()
	bridge := NewDistributedEventBridge(ctx, dh, bus)

	t.Run("messageToWS with nil", func(t *testing.T) {
		result := bridge.messageToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("messageToWS with author", func(t *testing.T) {
		now := time.Now()
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			AuthorID:  uuid.New(),
			Content:   "hello",
			CreatedAt: now,
			Pinned:    true,
			Author: &models.PublicUser{
				ID:       uuid.New(),
				Username: "author",
			},
		}
		result := bridge.messageToWS(msg)
		assert.Equal(t, "hello", result["content"])
		assert.NotNil(t, result["author"])
	})

	t.Run("messageToWS with edited timestamp", func(t *testing.T) {
		now := time.Now()
		edited := now.Add(time.Hour)
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			AuthorID:  uuid.New(),
			Content:   "edited",
			CreatedAt: now,
			EditedAt:  &edited,
		}
		result := bridge.messageToWS(msg)
		assert.NotNil(t, result["edited_timestamp"])
	})

	t.Run("channelToWS with nil", func(t *testing.T) {
		result := bridge.channelToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("channelToWS with topic and parent", func(t *testing.T) {
		serverID := uuid.New()
		parentID := uuid.New()
		ch := &models.Channel{
			ID:       uuid.New(),
			Name:     "test-channel",
			ServerID: &serverID,
			Topic:    "Test topic",
			ParentID: &parentID,
			Type:     models.ChannelTypeText,
		}
		result := bridge.channelToWS(ch)
		assert.Equal(t, "test-channel", result["name"])
		assert.Equal(t, "Test topic", result["topic"])
		assert.Equal(t, parentID.String(), result["parent_id"])
		assert.Equal(t, serverID.String(), result["guild_id"])
	})

	t.Run("serverToWS with nil", func(t *testing.T) {
		result := bridge.serverToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("serverToWS with optional fields", func(t *testing.T) {
		icon := "icon.png"
		banner := "banner.png"
		desc := "A server"
		srv := &models.Server{
			ID:          uuid.New(),
			Name:        "Test Server",
			OwnerID:     uuid.New(),
			IconURL:     &icon,
			BannerURL:   &banner,
			Description: &desc,
		}
		result := bridge.serverToWS(srv)
		assert.Equal(t, "Test Server", result["name"])
		assert.Equal(t, "icon.png", result["icon"])
		assert.Equal(t, "banner.png", result["banner"])
		assert.Equal(t, "A server", result["description"])
	})

	t.Run("userToWS with nil", func(t *testing.T) {
		result := bridge.userToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("userToWS with avatar", func(t *testing.T) {
		avatar := "avatar.png"
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
			AvatarURL:     &avatar,
		}
		result := bridge.userToWS(user)
		assert.Equal(t, "testuser", result["username"])
		assert.Equal(t, "avatar.png", result["avatar"])
	})

	t.Run("publicUserToWS with nil", func(t *testing.T) {
		result := bridge.publicUserToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("publicUserToWS with avatar", func(t *testing.T) {
		avatar := "pub-avatar.png"
		user := &models.PublicUser{
			ID:            uuid.New(),
			Username:      "pubuser",
			Discriminator: "0002",
			AvatarURL:     &avatar,
		}
		result := bridge.publicUserToWS(user)
		assert.Equal(t, "pubuser", result["username"])
		assert.Equal(t, "pub-avatar.png", result["avatar"])
	})

	t.Run("buildMemberData with user", func(t *testing.T) {
		data := &MemberEventData{
			ServerID: uuid.New(),
			UserID:   uuid.New(),
			User: &models.User{
				ID:       uuid.New(),
				Username: "member",
			},
		}
		result := bridge.buildMemberData(data, false)
		assert.NotNil(t, result["user"])
		assert.NotNil(t, result["guild_id"])
	})

	t.Run("buildMemberData with joinedAt", func(t *testing.T) {
		data := &MemberEventData{
			ServerID: uuid.New(),
			UserID:   uuid.New(),
			Member: &models.Member{
				JoinedAt: time.Now(),
			},
		}
		result := bridge.buildMemberData(data, true)
		assert.NotNil(t, result["joined_at"])
		assert.NotNil(t, result["roles"])
	})

	t.Run("buildMemberData without user", func(t *testing.T) {
		userID := uuid.New()
		data := &MemberEventData{
			ServerID: uuid.New(),
			UserID:   userID,
		}
		result := bridge.buildMemberData(data, false)
		user := result["user"].(map[string]interface{})
		assert.Equal(t, userID.String(), user["id"])
	})
}

func TestDistributedBridge_TypingWithServerID(t *testing.T) {
	_, dh, bus, client, cleanup := newDistributedBridgeTestSetup(t)
	defer cleanup()

	channelID := uuid.New()
	serverID := uuid.New()
	dh.SubscribeChannel(client, channelID)
	time.Sleep(100 * time.Millisecond)

	bus.Publish(events.TypingStarted, &TypingEventData{
		ChannelID: channelID,
		UserID:    uuid.New(),
		ServerID:  &serverID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "TYPING_START")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for typing event")
	}
}

func TestDistributedHub_WithDrainConfig(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "drain-config-test")
	require.NoError(t, err)
	defer ps.Close()

	config := &DrainConfig{
		DrainTimeout: 5 * time.Second,
		GracePeriod:  1 * time.Second,
	}

	dh := NewDistributedHubWithDrainConfig(ps, config)
	require.NotNil(t, dh)
	require.NotNil(t, dh.Hub)

	assert.True(t, dh.IsHealthy())
	assert.False(t, dh.IsDraining())
}

func TestDistributedHub_HandlePubSubMessage(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "pubsub-msg-test")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dh.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	channelID := uuid.New()
	serverID := uuid.New()
	userID := uuid.New()

	client := newMockClient(dh.Hub, uuid.New())
	dh.Hub.register <- client
	time.Sleep(50 * time.Millisecond)
	dh.SubscribeChannel(client, channelID)

	t.Run("channel routing", func(t *testing.T) {
		dh.handlePubSubMessage(&pubsub.BroadcastMessage{
			Type:      pubsub.MessageType(EventTypeMessageCreate),
			Data:      []byte(`{"content":"hello"}`),
			ChannelID: &channelID,
		})

		select {
		case msg := <-client.send:
			assert.Contains(t, string(msg), "MESSAGE_CREATE")
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		}
	})

	t.Run("server routing", func(t *testing.T) {
		dh.SubscribeServer(client, serverID)

		dh.handlePubSubMessage(&pubsub.BroadcastMessage{
			Type:     pubsub.MessageType(EventTypeMemberJoin),
			Data:     []byte(`{"user_id":"test"}`),
			ServerID: &serverID,
		})

		select {
		case msg := <-client.send:
			assert.Contains(t, string(msg), "MEMBER_JOIN")
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		}
	})

	t.Run("user routing", func(t *testing.T) {
		// Register client with specific userID
		userClient := newMockClient(dh.Hub, userID)
		dh.Hub.register <- userClient
		time.Sleep(50 * time.Millisecond)

		dh.handlePubSubMessage(&pubsub.BroadcastMessage{
			Type:   pubsub.MessageType(EventTypeUserUpdate),
			Data:   []byte(`{"username":"updated"}`),
			UserID: &userID,
		})

		select {
		case msg := <-userClient.send:
			assert.Contains(t, string(msg), "USER_UPDATE")
		case <-time.After(time.Second):
			t.Fatal("Timeout")
		}
	})
}

func TestDistributedHub_BroadcastDistributed_MarshalError(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "broadcast-err-test")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dh.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	// Event with unmarshalable data
	event := &Event{
		Op:   OpDispatch,
		Type: "TEST",
		Data: make(chan int), // channels can't be marshaled
	}

	err = dh.BroadcastDistributed(context.Background(), event)
	assert.Error(t, err)
}
