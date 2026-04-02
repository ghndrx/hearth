package websocket

import "encoding/json"

// Opcodes
const (
	OpDispatch            = 0  // Receive: Dispatched event
	OpHeartbeat           = 1  // Send/Receive: Heartbeat
	OpIdentify            = 2  // Send: Identify
	OpPresenceUpdate      = 3  // Send: Presence update
	OpVoiceStateUpdate    = 4  // Send: Voice state update
	OpResume              = 6  // Send: Resume session
	OpReconnect           = 7  // Receive: Reconnect
	OpRequestGuildMembers = 8  // Send: Request guild members
	OpInvalidSession      = 9  // Receive: Invalid session
	OpHello               = 10 // Receive: Hello
	OpHeartbeatAck        = 11 // Receive: Heartbeat ack
)

// Message represents a WebSocket message
type Message struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d,omitempty"`
	Sequence int64           `json:"s,omitempty"`
	Type     string          `json:"t,omitempty"`
}

// Event types
const (
	EventReady                    = "READY"
	EventResumed                  = "RESUMED"
	EventMessageCreate            = "MESSAGE_CREATE"
	EventMessageUpdate            = "MESSAGE_UPDATE"
	EventMessageDelete            = "MESSAGE_DELETE"
	EventMessageDeleteBulk        = "MESSAGE_DELETE_BULK"
	EventMessageReactionAdd       = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove    = "MESSAGE_REACTION_REMOVE"
	EventMessageReactionRemoveAll = "MESSAGE_REACTION_REMOVE_ALL"
	EventTypingStart              = "TYPING_START"
	EventChannelCreate            = "CHANNEL_CREATE"
	EventChannelUpdate            = "CHANNEL_UPDATE"
	EventChannelDelete            = "CHANNEL_DELETE"
	EventChannelPinsUpdate        = "CHANNEL_PINS_UPDATE"
	EventGuildCreate              = "GUILD_CREATE"
	EventGuildUpdate              = "GUILD_UPDATE"
	EventGuildDelete              = "GUILD_DELETE"
	EventGuildMemberAdd           = "GUILD_MEMBER_ADD"
	EventGuildMemberUpdate        = "GUILD_MEMBER_UPDATE"
	EventGuildMemberRemove        = "GUILD_MEMBER_REMOVE"
	EventGuildMembersChunk        = "GUILD_MEMBERS_CHUNK"
	EventGuildRoleCreate          = "GUILD_ROLE_CREATE"
	EventGuildRoleUpdate          = "GUILD_ROLE_UPDATE"
	EventGuildRoleDelete          = "GUILD_ROLE_DELETE"
	EventGuildBanAdd              = "GUILD_BAN_ADD"
	EventGuildBanRemove           = "GUILD_BAN_REMOVE"
	EventGuildEmojisUpdate        = "GUILD_EMOJIS_UPDATE"
	EventPresenceUpdate           = "PRESENCE_UPDATE"
	EventVoiceStateUpdate         = "VOICE_STATE_UPDATE"
	EventVoiceServerUpdate        = "VOICE_SERVER_UPDATE"
	EventUserUpdate               = "USER_UPDATE"
	EventInviteCreate             = "INVITE_CREATE"
	EventInviteDelete             = "INVITE_DELETE"
	EventNotificationCreate       = "NOTIFICATION_CREATE"
	EventNotificationUpdate       = "NOTIFICATION_UPDATE"
	EventReadStateUpdate          = "READ_STATE_UPDATE"

	// User status events
	EventUserStatusUpdate = "USER_STATUS_UPDATE"

	// DM events
	EventDMChannelCreate   = "DM_CHANNEL_CREATE"
	EventDMRecipientAdd    = "DM_RECIPIENT_ADD"
	EventDMRecipientRemove = "DM_RECIPIENT_REMOVE"
	EventGroupDMUpdate     = "GROUP_DM_UPDATE"

	// Stream events
	EventStreamStart       = "STREAM_START"
	EventStreamEnd         = "STREAM_END"
	EventStreamViewerJoin  = "STREAM_VIEWER_JOIN"
	EventStreamViewerLeave = "STREAM_VIEWER_LEAVE"

	// Interaction events
	EventInteractionCreate = "INTERACTION_CREATE"
	EventInteractionUpdate = "INTERACTION_UPDATE"
	EventAutocomplete      = "AUTOCOMPLETE"
	EventCommandExecute    = "COMMAND_EXECUTE"
	EventCommandResponse   = "COMMAND_RESPONSE"

	// Thread auto-archive events
	EventThreadAutoArchiveUpdate = "THREAD_AUTO_ARCHIVE_UPDATE"
	EventThreadAutoArchive      = "THREAD_AUTO_ARCHIVE"
)

// DispatchEvent creates a dispatch message
func DispatchEvent(eventType string, data interface{}, seq int64) (*Message, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Message{
		Op:       OpDispatch,
		Type:     eventType,
		Data:     d,
		Sequence: seq,
	}, nil
}

// HelloData is sent on connection
type HelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"` // Milliseconds
}

// ReadyData is sent after identify
type ReadyData struct {
	Version         int           `json:"v"`
	User            interface{}   `json:"user"`
	Guilds          []interface{} `json:"guilds"`
	PrivateChannels []interface{} `json:"private_channels"`
	SessionID       string        `json:"session_id"`
	ResumeURL       string        `json:"resume_gateway_url,omitempty"`
}

// MessageCreateData represents a new message event
type MessageCreateData struct {
	ID                string        `json:"id"`
	ChannelID         string        `json:"channel_id"`
	GuildID           string        `json:"guild_id,omitempty"`
	Author            interface{}   `json:"author"`
	Content           string        `json:"content"`
	Timestamp         string        `json:"timestamp"`
	EditedTimestamp   *string       `json:"edited_timestamp,omitempty"`
	TTS               bool          `json:"tts"`
	MentionEveryone   bool          `json:"mention_everyone"`
	Mentions          []interface{} `json:"mentions"`
	MentionRoles      []string      `json:"mention_roles"`
	Attachments       []interface{} `json:"attachments"`
	Embeds            []interface{} `json:"embeds"`
	Reactions         []interface{} `json:"reactions,omitempty"`
	Pinned            bool          `json:"pinned"`
	Type              int           `json:"type"`
	Flags             int           `json:"flags,omitempty"`
	ReferencedMessage interface{}   `json:"referenced_message,omitempty"`
}

// TypingStartData represents a typing event
type TypingStartData struct {
	ChannelID string      `json:"channel_id"`
	GuildID   string      `json:"guild_id,omitempty"`
	UserID    string      `json:"user_id"`
	Timestamp int64       `json:"timestamp"`
	Member    interface{} `json:"member,omitempty"`
}

// PresenceUpdateData represents a presence update
type PresenceUpdateData struct {
	User         interface{}   `json:"user"`
	GuildID      string        `json:"guild_id,omitempty"`
	Status       string        `json:"status"`
	Activities   []interface{} `json:"activities"`
	ClientStatus interface{}   `json:"client_status"`
}

// GuildMemberAddData represents a member join
type GuildMemberAddData struct {
	GuildID  string      `json:"guild_id"`
	User     interface{} `json:"user"`
	JoinedAt string      `json:"joined_at"`
	Roles    []string    `json:"roles"`
}

// GuildMemberRemoveData represents a member leave
type GuildMemberRemoveData struct {
	GuildID string      `json:"guild_id"`
	User    interface{} `json:"user"`
}

// NotificationCreateData represents a new notification event
type NotificationCreateData struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	Type          string      `json:"type"`
	Title         string      `json:"title"`
	Body          string      `json:"body"`
	Read          bool        `json:"read"`
	Data          interface{} `json:"data,omitempty"`
	ActorID       *string     `json:"actor_id,omitempty"`
	ActorUsername *string     `json:"actor_username,omitempty"`
	ActorAvatar   *string     `json:"actor_avatar,omitempty"`
	ServerID      *string     `json:"server_id,omitempty"`
	ServerName    *string     `json:"server_name,omitempty"`
	ChannelID     *string     `json:"channel_id,omitempty"`
	ChannelName   *string     `json:"channel_name,omitempty"`
	MessageID     *string     `json:"message_id,omitempty"`
	CreatedAt     string      `json:"created_at"`
}

// UserStatusUpdateData represents a user status change
type UserStatusUpdateData struct {
	UserID     string  `json:"user_id"`
	CustomText *string `json:"custom_text,omitempty"`
	Emoji      *string `json:"emoji,omitempty"`
	EmojiID    *string `json:"emoji_id,omitempty"`
	EmojiName  *string `json:"emoji_name,omitempty"`
	ClearAfter *string `json:"clear_after,omitempty"`
}

// ReadStateUpdateData represents a read state change
type ReadStateUpdateData struct {
	ChannelID     string  `json:"channel_id"`
	LastMessageID *string `json:"last_message_id,omitempty"`
	MentionCount  int     `json:"mention_count"`
}
