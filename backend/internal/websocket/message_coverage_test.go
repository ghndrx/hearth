package websocket

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchEvent_MarshalError(t *testing.T) {
	// Unmarshalable data
	msg, err := DispatchEvent(EventMessageCreate, make(chan int), 1)
	assert.Error(t, err)
	assert.Nil(t, msg)
}

func TestDispatchEvent_AllFields(t *testing.T) {
	data := map[string]interface{}{
		"id":      uuid.New().String(),
		"content": "hello world",
		"list":    []int{1, 2, 3},
	}

	msg, err := DispatchEvent(EventTypingStart, data, 99)
	require.NoError(t, err)

	assert.Equal(t, OpDispatch, msg.Op)
	assert.Equal(t, EventTypingStart, msg.Type)
	assert.Equal(t, int64(99), msg.Sequence)

	var decoded map[string]interface{}
	err = json.Unmarshal(msg.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "hello world", decoded["content"])
}

func TestGuildMemberRemoveData(t *testing.T) {
	guildID := uuid.New().String()
	userID := uuid.New().String()

	data := GuildMemberRemoveData{
		GuildID: guildID,
		User: map[string]interface{}{
			"id":       userID,
			"username": "removeduser",
		},
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, guildID, decoded["guild_id"])
	user := decoded["user"].(map[string]interface{})
	assert.Equal(t, "removeduser", user["username"])
}

func TestNotificationCreateData(t *testing.T) {
	actorID := uuid.New().String()
	serverID := uuid.New().String()
	channelID := uuid.New().String()
	messageID := uuid.New().String()
	actorUsername := "actor"
	actorAvatar := "https://example.com/avatar.png"
	serverName := "Test Server"
	channelName := "general"

	data := NotificationCreateData{
		ID:            uuid.New().String(),
		UserID:        uuid.New().String(),
		Type:          "mention",
		Title:         "You were mentioned",
		Body:          "@user mentioned you in #general",
		Read:          false,
		ActorID:       &actorID,
		ActorUsername: &actorUsername,
		ActorAvatar:   &actorAvatar,
		ServerID:      &serverID,
		ServerName:    &serverName,
		ChannelID:     &channelID,
		ChannelName:   &channelName,
		MessageID:     &messageID,
		CreatedAt:     "2025-01-01T00:00:00Z",
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "mention", decoded["type"])
	assert.Equal(t, "You were mentioned", decoded["title"])
	assert.Equal(t, false, decoded["read"])
	assert.Equal(t, actorID, decoded["actor_id"])
	assert.Equal(t, serverName, decoded["server_name"])
}

func TestNotificationCreateData_NilOptionals(t *testing.T) {
	data := NotificationCreateData{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		Type:      "system",
		Title:     "System notification",
		Body:      "System update",
		Read:      true,
		CreatedAt: "2025-01-01T00:00:00Z",
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded["actor_id"])
	assert.Nil(t, decoded["server_id"])
	assert.Nil(t, decoded["channel_id"])
	assert.Nil(t, decoded["message_id"])
}

func TestReadStateUpdateData(t *testing.T) {
	lastMsgID := uuid.New().String()

	data := ReadStateUpdateData{
		ChannelID:     uuid.New().String(),
		LastMessageID: &lastMsgID,
		MentionCount:  3,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, lastMsgID, decoded["last_message_id"])
	assert.Equal(t, float64(3), decoded["mention_count"])
}

func TestReadStateUpdateData_NilLastMessage(t *testing.T) {
	data := ReadStateUpdateData{
		ChannelID:    uuid.New().String(),
		MentionCount: 0,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded["last_message_id"])
	assert.Equal(t, float64(0), decoded["mention_count"])
}

func TestMessage_Omitempty(t *testing.T) {
	// Zero-value sequence and type should be omitted
	msg := Message{Op: OpHeartbeat}

	bytes, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	_, hasSeq := decoded["s"]
	_, hasType := decoded["t"]
	assert.False(t, hasSeq, "zero sequence should be omitted")
	assert.False(t, hasType, "empty type should be omitted")
}

func TestMessage_WithAllFields(t *testing.T) {
	msg := Message{
		Op:       OpDispatch,
		Type:     EventMessageCreate,
		Sequence: 10,
		Data:     json.RawMessage(`{"key":"value"}`),
	}

	bytes, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, float64(0), decoded["op"])
	assert.Equal(t, "MESSAGE_CREATE", decoded["t"])
	assert.Equal(t, float64(10), decoded["s"])
	assert.NotNil(t, decoded["d"])
}

func TestStreamEventConstants(t *testing.T) {
	assert.Equal(t, "STREAM_START", EventStreamStart)
	assert.Equal(t, "STREAM_END", EventStreamEnd)
	assert.Equal(t, "STREAM_VIEWER_JOIN", EventStreamViewerJoin)
	assert.Equal(t, "STREAM_VIEWER_LEAVE", EventStreamViewerLeave)
}

func TestAdditionalEventConstants(t *testing.T) {
	assert.Equal(t, "MESSAGE_DELETE_BULK", EventMessageDeleteBulk)
	assert.Equal(t, "MESSAGE_REACTION_ADD", EventMessageReactionAdd)
	assert.Equal(t, "MESSAGE_REACTION_REMOVE", EventMessageReactionRemove)
	assert.Equal(t, "MESSAGE_REACTION_REMOVE_ALL", EventMessageReactionRemoveAll)
	assert.Equal(t, "CHANNEL_PINS_UPDATE", EventChannelPinsUpdate)
	assert.Equal(t, "GUILD_MEMBER_UPDATE", EventGuildMemberUpdate)
	assert.Equal(t, "GUILD_MEMBERS_CHUNK", EventGuildMembersChunk)
	assert.Equal(t, "GUILD_ROLE_CREATE", EventGuildRoleCreate)
	assert.Equal(t, "GUILD_ROLE_UPDATE", EventGuildRoleUpdate)
	assert.Equal(t, "GUILD_ROLE_DELETE", EventGuildRoleDelete)
	assert.Equal(t, "GUILD_BAN_ADD", EventGuildBanAdd)
	assert.Equal(t, "GUILD_BAN_REMOVE", EventGuildBanRemove)
	assert.Equal(t, "GUILD_EMOJIS_UPDATE", EventGuildEmojisUpdate)
	assert.Equal(t, "VOICE_STATE_UPDATE", EventVoiceStateUpdate)
	assert.Equal(t, "VOICE_SERVER_UPDATE", EventVoiceServerUpdate)
	assert.Equal(t, "USER_UPDATE", EventUserUpdate)
	assert.Equal(t, "INVITE_CREATE", EventInviteCreate)
	assert.Equal(t, "INVITE_DELETE", EventInviteDelete)
	assert.Equal(t, "NOTIFICATION_CREATE", EventNotificationCreate)
	assert.Equal(t, "NOTIFICATION_UPDATE", EventNotificationUpdate)
	assert.Equal(t, "READ_STATE_UPDATE", EventReadStateUpdate)
}

func TestHelloData_Serialization(t *testing.T) {
	hello := HelloData{HeartbeatInterval: 45000}

	bytes, err := json.Marshal(hello)
	require.NoError(t, err)

	assert.JSONEq(t, `{"heartbeat_interval":45000}`, string(bytes))
}

func TestReconnectData(t *testing.T) {
	data := ReconnectData{
		Reason:    "server_shutdown",
		ResumeURL: "wss://gateway.example.com",
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "server_shutdown", decoded["reason"])
	assert.Equal(t, "wss://gateway.example.com", decoded["resume_gateway_url"])
}

func TestReconnectData_NoResumeURL(t *testing.T) {
	data := ReconnectData{Reason: "maintenance"}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	_, hasURL := decoded["resume_gateway_url"]
	assert.False(t, hasURL, "empty resume URL should be omitted")
}
