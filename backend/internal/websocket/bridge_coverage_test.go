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

// Helper to create a hub+bridge test environment
func setupBridgeTest(t *testing.T) (*Hub, *events.Bus, *EventBridge, *Client, func()) {
	t.Helper()

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

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

	return hub, bus, bridge, client, cancel
}

func TestBridge_MessageToWS(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("nil message", func(t *testing.T) {
		result := bridge.messageToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("basic message", func(t *testing.T) {
		now := time.Now()
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			AuthorID:  uuid.New(),
			Content:   "Hello world",
			CreatedAt: now,
			Pinned:    true,
			TTS:       true,
		}

		result := bridge.messageToWS(msg)
		assert.NotNil(t, result)
		assert.Equal(t, msg.ID.String(), result["id"])
		assert.Equal(t, msg.ChannelID.String(), result["channel_id"])
		assert.Equal(t, "Hello world", result["content"])
		assert.Equal(t, true, result["pinned"])
		assert.Equal(t, true, result["tts"])
		assert.Equal(t, 0, result["type"])
		assert.NotNil(t, result["timestamp"])

		// No author set, should fall back to AuthorID
		author := result["author"].(map[string]interface{})
		assert.Equal(t, msg.AuthorID.String(), author["id"])
	})

	t.Run("message with author", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.png"
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			AuthorID:  uuid.New(),
			Content:   "With author",
			CreatedAt: time.Now(),
			Author: &models.PublicUser{
				ID:            uuid.New(),
				Username:      "authoruser",
				Discriminator: "1234",
				AvatarURL:     &avatarURL,
			},
		}

		result := bridge.messageToWS(msg)
		author := result["author"].(map[string]interface{})
		assert.Equal(t, "authoruser", author["username"])
		assert.Equal(t, avatarURL, author["avatar"])
	})

	t.Run("message with edited timestamp", func(t *testing.T) {
		editedAt := time.Now()
		msg := &models.Message{
			ID:        uuid.New(),
			ChannelID: uuid.New(),
			AuthorID:  uuid.New(),
			Content:   "Edited",
			CreatedAt: time.Now(),
			EditedAt:  &editedAt,
		}

		result := bridge.messageToWS(msg)
		assert.NotNil(t, result["edited_timestamp"])
	})
}

func TestBridge_ChannelToWS(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("nil channel", func(t *testing.T) {
		result := bridge.channelToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("basic channel", func(t *testing.T) {
		ch := &models.Channel{
			ID:       uuid.New(),
			Name:     "general",
			Type:     models.ChannelTypeText,
			Position: 3,
		}

		result := bridge.channelToWS(ch)
		assert.Equal(t, ch.ID.String(), result["id"])
		assert.Equal(t, "general", result["name"])
		assert.Equal(t, 3, result["position"])
		_, hasGuild := result["guild_id"]
		assert.False(t, hasGuild)
	})

	t.Run("channel with server and topic", func(t *testing.T) {
		serverID := uuid.New()
		parentID := uuid.New()
		ch := &models.Channel{
			ID:       uuid.New(),
			Name:     "dev-chat",
			Type:     models.ChannelTypeText,
			Topic:    "Development discussion",
			ServerID: &serverID,
			ParentID: &parentID,
			Position: 1,
		}

		result := bridge.channelToWS(ch)
		assert.Equal(t, serverID.String(), result["guild_id"])
		assert.Equal(t, "Development discussion", result["topic"])
		assert.Equal(t, parentID.String(), result["parent_id"])
	})
}

func TestBridge_ServerToWS(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("nil server", func(t *testing.T) {
		result := bridge.serverToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("basic server", func(t *testing.T) {
		srv := &models.Server{
			ID:      uuid.New(),
			Name:    "My Server",
			OwnerID: uuid.New(),
		}

		result := bridge.serverToWS(srv)
		assert.Equal(t, srv.ID.String(), result["id"])
		assert.Equal(t, "My Server", result["name"])
		assert.Equal(t, srv.OwnerID.String(), result["owner_id"])
	})

	t.Run("server with all optional fields", func(t *testing.T) {
		iconURL := "https://example.com/icon.png"
		bannerURL := "https://example.com/banner.png"
		desc := "A test server"

		srv := &models.Server{
			ID:          uuid.New(),
			Name:        "Full Server",
			OwnerID:     uuid.New(),
			IconURL:     &iconURL,
			BannerURL:   &bannerURL,
			Description: &desc,
		}

		result := bridge.serverToWS(srv)
		assert.Equal(t, iconURL, result["icon"])
		assert.Equal(t, bannerURL, result["banner"])
		assert.Equal(t, desc, result["description"])
	})
}

func TestBridge_UserToWS(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("nil user", func(t *testing.T) {
		result := bridge.userToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("user without avatar", func(t *testing.T) {
		user := &models.User{
			ID:            uuid.New(),
			Username:      "testuser",
			Discriminator: "0001",
		}

		result := bridge.userToWS(user)
		assert.Equal(t, "testuser", result["username"])
		assert.Equal(t, "0001", result["discriminator"])
		assert.Equal(t, false, result["bot"])
		_, hasAvatar := result["avatar"]
		assert.False(t, hasAvatar)
	})

	t.Run("user with avatar", func(t *testing.T) {
		avatarURL := "https://example.com/avatar.png"
		user := &models.User{
			ID:            uuid.New(),
			Username:      "avataruser",
			Discriminator: "5678",
			AvatarURL:     &avatarURL,
		}

		result := bridge.userToWS(user)
		assert.Equal(t, avatarURL, result["avatar"])
	})
}

func TestBridge_PublicUserToWS(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("nil public user", func(t *testing.T) {
		result := bridge.publicUserToWS(nil)
		assert.Nil(t, result)
	})

	t.Run("public user", func(t *testing.T) {
		avatarURL := "https://cdn.example.com/avatar.png"
		user := &models.PublicUser{
			ID:            uuid.New(),
			Username:      "pubuser",
			Discriminator: "9999",
			AvatarURL:     &avatarURL,
		}

		result := bridge.publicUserToWS(user)
		assert.Equal(t, "pubuser", result["username"])
		assert.Equal(t, avatarURL, result["avatar"])
		assert.Equal(t, false, result["bot"])
	})
}

func TestBridge_BuildMemberData(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	bridge := NewEventBridge(hub, bus)

	t.Run("with user object and joinedAt", func(t *testing.T) {
		serverID := uuid.New()
		userID := uuid.New()
		now := time.Now()

		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
			User: &models.User{
				ID:            userID,
				Username:      "testmember",
				Discriminator: "0001",
			},
			Member: &models.Member{
				UserID:   userID,
				ServerID: serverID,
				JoinedAt: now,
			},
		}

		result := bridge.buildMemberData(data, true)
		assert.Equal(t, serverID.String(), result["guild_id"])
		assert.NotNil(t, result["joined_at"])
		assert.NotNil(t, result["roles"])

		user := result["user"].(map[string]interface{})
		assert.Equal(t, "testmember", user["username"])
	})

	t.Run("without user object", func(t *testing.T) {
		serverID := uuid.New()
		userID := uuid.New()

		data := &MemberEventData{
			ServerID: serverID,
			UserID:   userID,
		}

		result := bridge.buildMemberData(data, false)
		user := result["user"].(map[string]interface{})
		assert.Equal(t, userID.String(), user["id"])
		_, hasJoinedAt := result["joined_at"]
		assert.False(t, hasJoinedAt)
	})
}

func TestBridge_OnMessageUpdated(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	editedAt := time.Now()
	bus.Publish(events.MessageUpdated, &services.MessageUpdatedEvent{
		Message: &models.Message{
			ID:        uuid.New(),
			ChannelID: channelID,
			AuthorID:  uuid.New(),
			Content:   "Updated content",
			CreatedAt: time.Now(),
			EditedAt:  &editedAt,
		},
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message update")
	}
}

func TestBridge_OnMessageDeleted(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.MessageDeleted, &services.MessageDeletedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MESSAGE_DELETE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMessagePinned(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.MessagePinned, &services.MessagePinnedEvent{
		ChannelID: channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "CHANNEL_PINS_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnReactionAdded(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.ReactionAdded, &services.ReactionAddedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
		UserID:    uuid.New(),
		Emoji:     "👍",
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "REACTION_ADD")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnReactionRemoved(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.ReactionRemoved, &services.ReactionRemovedEvent{
		MessageID: uuid.New(),
		ChannelID: channelID,
		UserID:    uuid.New(),
		Emoji:     "👍",
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "REACTION_REMOVE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnChannelCreated(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.ChannelCreated, &ChannelEventData{
		Channel: &models.Channel{
			ID:       uuid.New(),
			Name:     "new-channel",
			ServerID: &serverID,
			Type:     models.ChannelTypeText,
		},
		ServerID: serverID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "CHANNEL_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnChannelUpdated(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

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
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnChannelDeleted(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

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
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnServerCreated(t *testing.T) {
	_, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	bus.Publish(events.ServerCreated, &ServerEventData{
		Server: &models.Server{
			ID:      uuid.New(),
			Name:    "New Server",
			OwnerID: client.UserID,
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_CREATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnServerUpdated(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.ServerUpdated, &ServerEventData{
		Server: &models.Server{
			ID:      serverID,
			Name:    "Updated Server",
			OwnerID: uuid.New(),
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "SERVER_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnServerDeleted(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

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
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMemberJoined(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	newMemberID := uuid.New()
	bus.Publish(events.MemberJoined, &MemberEventData{
		ServerID: serverID,
		UserID:   newMemberID,
		User: &models.User{
			ID:       newMemberID,
			Username: "newmember",
		},
		Member: &models.Member{
			UserID:   newMemberID,
			ServerID: serverID,
			JoinedAt: time.Now(),
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_JOIN")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMemberLeft(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.MemberLeft, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_LEAVE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMemberUpdated(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	nickname := "newnick"
	bus.Publish(events.MemberUpdated, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
		Member: &models.Member{
			UserID:   uuid.New(),
			ServerID: serverID,
			JoinedAt: time.Now(),
			Nickname: &nickname,
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_UPDATE")
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMemberKicked(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.MemberKicked, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "MEMBER_LEAVE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnMemberBanned(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.MemberBanned, &MemberEventData{
		ServerID: serverID,
		UserID:   uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "GUILD_BAN_ADD")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnUserUpdated(t *testing.T) {
	_, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	bus.Publish(events.UserUpdated, &UserEventData{
		User: &models.User{
			ID:            client.UserID,
			Username:      "updateduser",
			Discriminator: "0001",
		},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "USER_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnPresenceUpdate(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	serverID := uuid.New()
	hub.SubscribeServer(client, serverID)

	bus.Publish(events.PresenceUpdate, &PresenceEventData{
		UserID:     uuid.New(),
		Status:     "online",
		Activities: []string{},
		ServerIDs:  []uuid.UUID{serverID},
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "PRESENCE_UPDATE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnTypingStarted(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.TypingStarted, &TypingEventData{
		ChannelID: channelID,
		UserID:    uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "TYPING_START")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnTypingStartedWithServer(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	serverID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish(events.TypingStarted, &TypingEventData{
		ChannelID: channelID,
		ServerID:  &serverID,
		UserID:    uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "TYPING_START")
		var event Event
		err := json.Unmarshal(msg, &event)
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

// Test wrong type assertions in event handlers
func TestBridge_WrongTypeAssertions(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

	// These should not panic, just log and return
	bus.Publish(events.MessageCreated, "wrong type")
	bus.Publish(events.MessageUpdated, 42)
	bus.Publish(events.MessageDeleted, nil)
	bus.Publish(events.MessagePinned, struct{}{})
	bus.Publish(events.ReactionAdded, "wrong")
	bus.Publish(events.ReactionRemoved, 123)
	bus.Publish(events.ChannelCreated, false)
	bus.Publish(events.ChannelUpdated, []int{})
	bus.Publish(events.ChannelDeleted, map[string]int{})
	bus.Publish(events.ServerCreated, "not a server")
	bus.Publish(events.ServerUpdated, 0)
	bus.Publish(events.ServerDeleted, true)
	bus.Publish(events.MemberJoined, "nope")
	bus.Publish(events.MemberLeft, 1)
	bus.Publish(events.MemberUpdated, nil)
	bus.Publish(events.MemberKicked, "wrong")
	bus.Publish(events.MemberBanned, false)
	bus.Publish(events.UserUpdated, 42)
	bus.Publish(events.PresenceUpdate, "not presence")
	bus.Publish(events.TypingStarted, []byte("nope"))

	// Give time for async event processing
	time.Sleep(100 * time.Millisecond)
}

func TestBridge_OnStreamStarted(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish("stream.started", &services.StreamStartedEvent{
		ChannelID: channelID,
		ServerID:  uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "STREAM_START")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnStreamEnded(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish("stream.ended", &services.StreamEndedEvent{
		StreamID:  uuid.New(),
		ChannelID: channelID,
		ServerID:  uuid.New(),
		UserID:    uuid.New(),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "STREAM_END")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnStreamViewerJoined(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish("stream.viewer_joined", &services.StreamViewerJoinedEvent{
		StreamID:    uuid.New(),
		UserID:      uuid.New(),
		ViewerCount: 5,
		ServerID:    uuid.New(),
		ChannelID:   channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "STREAM_VIEWER_JOIN")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_OnStreamViewerLeft(t *testing.T) {
	hub, bus, _, client, cancel := setupBridgeTest(t)
	defer cancel()

	channelID := uuid.New()
	hub.SubscribeChannel(client, channelID)

	bus.Publish("stream.viewer_left", &services.StreamViewerLeftEvent{
		StreamID:    uuid.New(),
		UserID:      uuid.New(),
		ViewerCount: 4,
		ServerID:    uuid.New(),
		ChannelID:   channelID,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "STREAM_VIEWER_LEAVE")
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	}
}

func TestBridge_StreamWrongTypes(t *testing.T) {
	hub := NewHub()
	bus := events.NewBus()
	_ = NewEventBridge(hub, bus)

	// Should not panic
	bus.Publish("stream.started", "wrong")
	bus.Publish("stream.ended", 42)
	bus.Publish("stream.viewer_joined", nil)
	bus.Publish("stream.viewer_left", false)

	time.Sleep(100 * time.Millisecond)
}
