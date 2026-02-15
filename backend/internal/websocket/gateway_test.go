package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGatewayConfig(t *testing.T) {
	cfg := DefaultGatewayConfig()

	assert.Equal(t, 41250*time.Millisecond, cfg.HeartbeatInterval)
	assert.Equal(t, 5*time.Minute, cfg.SessionTimeout)
}

func TestSession(t *testing.T) {
	session := &Session{
		ID:            uuid.New().String(),
		UserID:        uuid.New(),
		Username:      "testuser",
		ClientType:    "web",
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Sequence:      0,
		ResumeKey:     uuid.New().String(),
		ResumeEvents:  make([][]byte, 0, 100),
	}

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "testuser", session.Username)
	assert.Equal(t, "web", session.ClientType)
	assert.NotEmpty(t, session.ResumeKey)
}

func TestHelloData(t *testing.T) {
	hello := HelloData{
		HeartbeatInterval: 41250,
	}

	data, err := json.Marshal(hello)
	require.NoError(t, err)

	var decoded HelloData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 41250, decoded.HeartbeatInterval)
}

func TestReadyData(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New().String()

	ready := ReadyData{
		Version:         10,
		SessionID:       sessionID,
		Guilds:          []interface{}{},
		PrivateChannels: []interface{}{},
		User: map[string]interface{}{
			"id":       userID.String(),
			"username": "testuser",
		},
	}

	data, err := json.Marshal(ready)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, float64(10), decoded["v"])
	assert.Equal(t, sessionID, decoded["session_id"])

	user := decoded["user"].(map[string]interface{})
	assert.Equal(t, userID.String(), user["id"])
	assert.Equal(t, "testuser", user["username"])
}

func TestMessageCreateData(t *testing.T) {
	msgData := MessageCreateData{
		ID:              uuid.New().String(),
		ChannelID:       uuid.New().String(),
		GuildID:         uuid.New().String(),
		Content:         "Hello, world!",
		Timestamp:       time.Now().Format(time.RFC3339),
		TTS:             false,
		MentionEveryone: false,
		Mentions:        []interface{}{},
		MentionRoles:    []string{},
		Attachments:     []interface{}{},
		Embeds:          []interface{}{},
		Pinned:          false,
		Type:            0,
		Author: map[string]interface{}{
			"id":       uuid.New().String(),
			"username": "testuser",
		},
	}

	data, err := json.Marshal(msgData)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "Hello, world!", decoded["content"])
	assert.Equal(t, false, decoded["tts"])
	assert.Equal(t, false, decoded["pinned"])
}

func TestTypingStartData(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	typing := TypingStartData{
		ChannelID: channelID.String(),
		UserID:    userID.String(),
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(typing)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, channelID.String(), decoded["channel_id"])
	assert.Equal(t, userID.String(), decoded["user_id"])
}

func TestPresenceUpdateData(t *testing.T) {
	presence := PresenceUpdateData{
		Status:       "online",
		Activities:   []interface{}{},
		ClientStatus: map[string]string{"web": "online"},
		User: map[string]interface{}{
			"id": uuid.New().String(),
		},
	}

	data, err := json.Marshal(presence)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "online", decoded["status"])
}

func TestGuildMemberAddData(t *testing.T) {
	guildID := uuid.New().String()
	userID := uuid.New().String()

	member := GuildMemberAddData{
		GuildID:  guildID,
		JoinedAt: time.Now().Format(time.RFC3339),
		Roles:    []string{},
		User: map[string]interface{}{
			"id":       userID,
			"username": "newmember",
		},
	}

	data, err := json.Marshal(member)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, guildID, decoded["guild_id"])

	user := decoded["user"].(map[string]interface{})
	assert.Equal(t, "newmember", user["username"])
}

func TestDispatchEvent(t *testing.T) {
	msgData := map[string]interface{}{
		"id":         uuid.New().String(),
		"channel_id": uuid.New().String(),
		"content":    "Test message",
	}

	msg, err := DispatchEvent(EventMessageCreate, msgData, 42)
	require.NoError(t, err)

	assert.Equal(t, OpDispatch, msg.Op)
	assert.Equal(t, EventMessageCreate, msg.Type)
	assert.Equal(t, int64(42), msg.Sequence)
	assert.NotNil(t, msg.Data)

	// Verify data can be decoded
	var decoded map[string]interface{}
	err = json.Unmarshal(msg.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "Test message", decoded["content"])
}

func TestMessageOpcodes(t *testing.T) {
	// Verify opcode constants match expected values
	assert.Equal(t, 0, OpDispatch)
	assert.Equal(t, 1, OpHeartbeat)
	assert.Equal(t, 2, OpIdentify)
	assert.Equal(t, 3, OpPresenceUpdate)
	assert.Equal(t, 4, OpVoiceStateUpdate)
	assert.Equal(t, 6, OpResume)
	assert.Equal(t, 7, OpReconnect)
	assert.Equal(t, 8, OpRequestGuildMembers)
	assert.Equal(t, 9, OpInvalidSession)
	assert.Equal(t, 10, OpHello)
	assert.Equal(t, 11, OpHeartbeatAck)
}

func TestEventTypes(t *testing.T) {
	// Verify event type constants
	assert.Equal(t, "READY", EventReady)
	assert.Equal(t, "RESUMED", EventResumed)
	assert.Equal(t, "MESSAGE_CREATE", EventMessageCreate)
	assert.Equal(t, "MESSAGE_UPDATE", EventMessageUpdate)
	assert.Equal(t, "MESSAGE_DELETE", EventMessageDelete)
	assert.Equal(t, "TYPING_START", EventTypingStart)
	assert.Equal(t, "PRESENCE_UPDATE", EventPresenceUpdate)
	assert.Equal(t, "GUILD_CREATE", EventGuildCreate)
	assert.Equal(t, "GUILD_UPDATE", EventGuildUpdate)
	assert.Equal(t, "GUILD_DELETE", EventGuildDelete)
	assert.Equal(t, "GUILD_MEMBER_ADD", EventGuildMemberAdd)
	assert.Equal(t, "GUILD_MEMBER_REMOVE", EventGuildMemberRemove)
	assert.Equal(t, "CHANNEL_CREATE", EventChannelCreate)
	assert.Equal(t, "CHANNEL_UPDATE", EventChannelUpdate)
	assert.Equal(t, "CHANNEL_DELETE", EventChannelDelete)
}

func TestMessageSerialization(t *testing.T) {
	// Test full message roundtrip
	original := Message{
		Op:       OpDispatch,
		Type:     EventMessageCreate,
		Sequence: 123,
		Data:     json.RawMessage(`{"content":"hello"}`),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Op, decoded.Op)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Sequence, decoded.Sequence)
	assert.JSONEq(t, string(original.Data), string(decoded.Data))
}

func TestHeartbeatMessage(t *testing.T) {
	// Heartbeat message
	hb := Message{
		Op: OpHeartbeat,
	}

	data, err := json.Marshal(hb)
	require.NoError(t, err)

	// Should serialize to {"op":1}
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, float64(OpHeartbeat), decoded["op"])
}

func TestHeartbeatAckMessage(t *testing.T) {
	hbAck := Message{
		Op: OpHeartbeatAck,
	}

	data, err := json.Marshal(hbAck)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, float64(OpHeartbeatAck), decoded["op"])
}

// TestGatewayStats tests statistics collection
func TestGatewayStats(t *testing.T) {
	// Create a gateway without JWT service for testing stats
	hub := NewHub()
	gateway := &Gateway{
		hub:      hub,
		config:   DefaultGatewayConfig(),
		sessions: make(map[string]*Session),
	}

	stats := gateway.GetStats()

	assert.Contains(t, stats, "total_connections")
	assert.Contains(t, stats, "active_connections")
	assert.Contains(t, stats, "messages_processed")
	assert.Contains(t, stats, "active_sessions")

	// Initial stats should be zero
	assert.Equal(t, int64(0), stats["total_connections"])
	assert.Equal(t, int64(0), stats["active_connections"])
	assert.Equal(t, int64(0), stats["messages_processed"])
	assert.Equal(t, 0, stats["active_sessions"])
}
