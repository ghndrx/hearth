package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpcodes(t *testing.T) {
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

func TestEventConstants(t *testing.T) {
	assert.Equal(t, "READY", EventReady)
	assert.Equal(t, "RESUMED", EventResumed)
	assert.Equal(t, "MESSAGE_CREATE", EventMessageCreate)
	assert.Equal(t, "MESSAGE_UPDATE", EventMessageUpdate)
	assert.Equal(t, "MESSAGE_DELETE", EventMessageDelete)
	assert.Equal(t, "MESSAGE_DELETE_BULK", EventMessageDeleteBulk)
	assert.Equal(t, "MESSAGE_REACTION_ADD", EventMessageReactionAdd)
	assert.Equal(t, "MESSAGE_REACTION_REMOVE", EventMessageReactionRemove)
	assert.Equal(t, "MESSAGE_REACTION_REMOVE_ALL", EventMessageReactionRemoveAll)
	assert.Equal(t, "TYPING_START", EventTypingStart)
	assert.Equal(t, "CHANNEL_CREATE", EventChannelCreate)
	assert.Equal(t, "CHANNEL_UPDATE", EventChannelUpdate)
	assert.Equal(t, "CHANNEL_DELETE", EventChannelDelete)
	assert.Equal(t, "CHANNEL_PINS_UPDATE", EventChannelPinsUpdate)
	assert.Equal(t, "GUILD_CREATE", EventGuildCreate)
	assert.Equal(t, "GUILD_UPDATE", EventGuildUpdate)
	assert.Equal(t, "GUILD_DELETE", EventGuildDelete)
	assert.Equal(t, "GUILD_MEMBER_ADD", EventGuildMemberAdd)
	assert.Equal(t, "GUILD_MEMBER_UPDATE", EventGuildMemberUpdate)
	assert.Equal(t, "GUILD_MEMBER_REMOVE", EventGuildMemberRemove)
	assert.Equal(t, "GUILD_MEMBERS_CHUNK", EventGuildMembersChunk)
	assert.Equal(t, "GUILD_ROLE_CREATE", EventGuildRoleCreate)
	assert.Equal(t, "GUILD_ROLE_UPDATE", EventGuildRoleUpdate)
	assert.Equal(t, "GUILD_ROLE_DELETE", EventGuildRoleDelete)
	assert.Equal(t, "GUILD_BAN_ADD", EventGuildBanAdd)
	assert.Equal(t, "GUILD_BAN_REMOVE", EventGuildBanRemove)
	assert.Equal(t, "GUILD_EMOJIS_UPDATE", EventGuildEmojisUpdate)
	assert.Equal(t, "PRESENCE_UPDATE", EventPresenceUpdate)
	assert.Equal(t, "VOICE_STATE_UPDATE", EventVoiceStateUpdate)
	assert.Equal(t, "VOICE_SERVER_UPDATE", EventVoiceServerUpdate)
	assert.Equal(t, "USER_UPDATE", EventUserUpdate)
	assert.Equal(t, "INVITE_CREATE", EventInviteCreate)
	assert.Equal(t, "INVITE_DELETE", EventInviteDelete)
}

func TestMessageSerializationFull(t *testing.T) {
	msg := Message{
		Op:       OpDispatch,
		Type:     EventMessageCreate,
		Sequence: 42,
		Data:     json.RawMessage(`{"content":"hello"}`),
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Op, decoded.Op)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Sequence, decoded.Sequence)
	assert.JSONEq(t, `{"content":"hello"}`, string(decoded.Data))
}

func TestMessageWithEmptyData(t *testing.T) {
	msg := Message{
		Op:   OpHeartbeat,
		Type: "",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, OpHeartbeat, decoded.Op)
	assert.Empty(t, decoded.Type)
}

func TestDispatchEventBasic(t *testing.T) {
	testData := map[string]interface{}{
		"content": "hello world",
		"id":      "12345",
	}

	msg, err := DispatchEvent(EventMessageCreate, testData, 100)
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.Equal(t, OpDispatch, msg.Op)
	assert.Equal(t, EventMessageCreate, msg.Type)
	assert.Equal(t, int64(100), msg.Sequence)

	// Verify data is properly marshaled
	var decoded map[string]interface{}
	err = json.Unmarshal(msg.Data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "hello world", decoded["content"])
	assert.Equal(t, "12345", decoded["id"])
}

func TestDispatchEventWithStruct(t *testing.T) {
	data := TypingStartData{
		ChannelID: "channel-123",
		GuildID:   "guild-456",
		UserID:    "user-789",
		Timestamp: time.Now().Unix(),
	}

	msg, err := DispatchEvent(EventTypingStart, data, 50)
	require.NoError(t, err)

	var decoded TypingStartData
	err = json.Unmarshal(msg.Data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "channel-123", decoded.ChannelID)
	assert.Equal(t, "guild-456", decoded.GuildID)
	assert.Equal(t, "user-789", decoded.UserID)
}

func TestHelloDataSerialization(t *testing.T) {
	hello := HelloData{
		HeartbeatInterval: 45000,
	}

	data, err := json.Marshal(hello)
	require.NoError(t, err)

	var decoded HelloData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 45000, decoded.HeartbeatInterval)
}

func TestReadyDataSerialization(t *testing.T) {
	ready := ReadyData{
		Version:         9,
		User:            map[string]string{"id": "user-123", "username": "testuser"},
		Guilds:          []interface{}{map[string]string{"id": "guild-1"}},
		PrivateChannels: []interface{}{},
		SessionID:       "session-abc",
		ResumeURL:       "wss://resume.example.com",
	}

	data, err := json.Marshal(ready)
	require.NoError(t, err)

	var decoded ReadyData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 9, decoded.Version)
	assert.Equal(t, "session-abc", decoded.SessionID)
	assert.Equal(t, "wss://resume.example.com", decoded.ResumeURL)
	assert.Len(t, decoded.Guilds, 1)
}

func TestMessageCreateDataSerialization(t *testing.T) {
	edited := "2024-01-15T10:30:00Z"
	msg := MessageCreateData{
		ID:              "msg-123",
		ChannelID:       "channel-456",
		GuildID:         "guild-789",
		Author:          map[string]string{"id": "user-1", "username": "test"},
		Content:         "Hello, World!",
		Timestamp:       "2024-01-15T10:00:00Z",
		EditedTimestamp: &edited,
		TTS:             false,
		MentionEveryone: true,
		Mentions:        []interface{}{},
		MentionRoles:    []string{"role-1"},
		Attachments:     []interface{}{},
		Embeds:          []interface{}{},
		Reactions:       []interface{}{map[string]interface{}{"emoji": "👍", "count": 5}},
		Pinned:          true,
		Type:            0,
		Flags:           4,
		ReferencedMessage: map[string]string{"id": "ref-msg"},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded MessageCreateData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "msg-123", decoded.ID)
	assert.Equal(t, "Hello, World!", decoded.Content)
	assert.True(t, decoded.MentionEveryone)
	assert.True(t, decoded.Pinned)
	assert.Equal(t, 4, decoded.Flags)
	assert.Len(t, decoded.MentionRoles, 1)
}

func TestTypingStartDataSerialization(t *testing.T) {
	typing := TypingStartData{
		ChannelID: "channel-123",
		GuildID:   "guild-456",
		UserID:    "user-789",
		Timestamp: 1705312800,
		Member:    map[string]string{"nick": "testnick"},
	}

	data, err := json.Marshal(typing)
	require.NoError(t, err)

	var decoded TypingStartData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "channel-123", decoded.ChannelID)
	assert.Equal(t, int64(1705312800), decoded.Timestamp)
}

func TestPresenceUpdateDataSerialization(t *testing.T) {
	presence := PresenceUpdateData{
		User:         map[string]string{"id": "user-123"},
		GuildID:      "guild-456",
		Status:       "online",
		Activities:   []interface{}{map[string]string{"name": "Playing Game"}},
		ClientStatus: map[string]string{"desktop": "online"},
	}

	data, err := json.Marshal(presence)
	require.NoError(t, err)

	var decoded PresenceUpdateData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "online", decoded.Status)
	assert.Equal(t, "guild-456", decoded.GuildID)
	assert.Len(t, decoded.Activities, 1)
}

func TestGuildMemberAddDataSerialization(t *testing.T) {
	member := GuildMemberAddData{
		GuildID:  "guild-123",
		User:     map[string]string{"id": "user-456", "username": "newmember"},
		JoinedAt: "2024-01-15T10:00:00Z",
		Roles:    []string{"role-1", "role-2"},
	}

	data, err := json.Marshal(member)
	require.NoError(t, err)

	var decoded GuildMemberAddData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "guild-123", decoded.GuildID)
	assert.Len(t, decoded.Roles, 2)
}

func TestGuildMemberRemoveData(t *testing.T) {
	member := GuildMemberRemoveData{
		GuildID: "guild-123",
		User:    map[string]string{"id": "user-456", "username": "leavingmember"},
	}

	data, err := json.Marshal(member)
	require.NoError(t, err)

	var decoded GuildMemberRemoveData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "guild-123", decoded.GuildID)
}

func TestMessageJSONOmitsEmptyFields(t *testing.T) {
	msg := Message{
		Op: OpHeartbeat,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Should not include empty fields with omitempty
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded, "op")
	// Sequence should be omitted when 0 due to omitempty
	// Type should be omitted when empty due to omitempty
}

func TestDispatchEventMarshalError(t *testing.T) {
	// Create something that can't be marshaled
	badData := make(chan int) // channels can't be marshaled

	_, err := DispatchEvent(EventMessageCreate, badData, 1)
	assert.Error(t, err)
}

func TestCompleteMessageFlow(t *testing.T) {
	// Simulate a complete message flow
	
	// 1. Server sends HELLO
	hello := HelloData{HeartbeatInterval: 45000}
	helloMsg, err := DispatchEvent("", hello, 0)
	require.NoError(t, err)
	helloMsg.Op = OpHello
	helloMsg.Type = ""

	// 2. Server sends READY after identify
	ready := ReadyData{
		Version:   9,
		SessionID: "session-123",
		User:      map[string]string{"id": "user-1"},
		Guilds:    []interface{}{},
	}
	readyMsg, err := DispatchEvent(EventReady, ready, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), readyMsg.Sequence)

	// 3. Server dispatches message
	msgData := MessageCreateData{
		ID:        "msg-1",
		ChannelID: "channel-1",
		Content:   "Test message",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	createMsg, err := DispatchEvent(EventMessageCreate, msgData, 2)
	require.NoError(t, err)
	assert.Equal(t, EventMessageCreate, createMsg.Type)
	assert.Equal(t, int64(2), createMsg.Sequence)
}
