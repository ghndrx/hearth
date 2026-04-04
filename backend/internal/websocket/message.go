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
	EventThreadAutoArchive       = "THREAD_AUTO_ARCHIVE"

	// Premium & Server Boost events
	EventServerBoostUpdate         = "SERVER_BOOST_UPDATE"
	EventServerBoostLevelUp        = "SERVER_BOOST_LEVEL_UP"
	EventServerBoostLevelDown      = "SERVER_BOOST_LEVEL_DOWN"
	EventPremiumSubscriptionUpdate = "PREMIUM_SUBSCRIPTION_UPDATE"

	// Forum Channel events
	EventForumPostCreate    = "FORUM_POST_CREATE"
	EventForumPostUpdate    = "FORUM_POST_UPDATE"
	EventForumPostDelete    = "FORUM_POST_DELETE"
	EventForumTagCreate     = "FORUM_TAG_CREATE"
	EventForumTagUpdate     = "FORUM_TAG_UPDATE"
	EventForumTagDelete     = "FORUM_TAG_DELETE"
	EventForumPostPin       = "FORUM_POST_PIN"
	EventForumPostUnpin     = "FORUM_POST_UNPIN"
	EventForumPostArchive   = "FORUM_POST_ARCHIVE"
	EventForumPostUnarchive = "FORUM_POST_UNARCHIVE"

	// Server Folder events
	EventServerFolderCreate   = "SERVER_FOLDER_CREATE"
	EventServerFolderUpdate   = "SERVER_FOLDER_UPDATE"
	EventServerFolderDelete   = "SERVER_FOLDER_DELETE"
	EventServerFolderMove     = "SERVER_FOLDER_MOVE"
	EventServerFolderReorder  = "SERVER_FOLDER_REORDER"

	// Soundboard events
	EventSoundboardPlay = "SOUNDBOARD_PLAY"
	EventSoundboardStop = "SOUNDBOARD_STOP"
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

// ServerBoostUpdateData represents a server boost being added or removed
type ServerBoostUpdateData struct {
	ServerID   string `json:"server_id"`
	UserID     string `json:"user_id"`
	BoosterTag string `json:"booster_tag,omitempty"`
	Action     string `json:"action"` // "added" or "removed"
}

// ServerBoostLevelUpdateData represents a server's boost level change
type ServerBoostLevelUpdateData struct {
	ServerID       string      `json:"server_id"`
	Level          int         `json:"level"`
	BoostCount     int         `json:"boost_count"`
	BoostsRequired int         `json:"boosts_required,omitempty"` // for next level
	Perks          interface{} `json:"perks,omitempty"`
}

// PremiumSubscriptionUpdateData represents a user's subscription change
type PremiumSubscriptionUpdateData struct {
	UserID    string `json:"user_id"`
	Tier      string `json:"tier"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ForumPostCreateData represents a new forum post (thread) event
type ForumPostCreateData struct {
	ChannelID     string   `json:"channel_id"`
	GuildID       string   `json:"guild_id,omitempty"`
	Post         interface{} `json:"post"`
	AppliedTags   []string `json:"applied_tags,omitempty"`
	Author        interface{} `json:"author"`
}

// ForumPostUpdateData represents a forum post update event
type ForumPostUpdateData struct {
	ChannelID    string   `json:"channel_id"`
	GuildID      string   `json:"guild_id,omitempty"`
	PostID       string   `json:"post_id"`
	Name         *string  `json:"name,omitempty"`
	AppliedTags  []string `json:"applied_tags,omitempty"`
	Archived     *bool    `json:"archived,omitempty"`
	Locked       *bool    `json:"locked,omitempty"`
	AutoArchive  *int     `json:"auto_archive,omitempty"`
	IsPinned     *bool    `json:"is_pinned,omitempty"`
}

// ForumPostDeleteData represents a forum post deletion event
type ForumPostDeleteData struct {
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id,omitempty"`
	PostID    string `json:"post_id"`
}

// ForumTagEventData represents a forum tag event
type ForumTagEventData struct {
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id,omitempty"`
	Tag       interface{} `json:"tag"`
}

// ForumPostPinData represents a forum post pin event
type ForumPostPinData struct {
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id,omitempty"`
	PostID    string `json:"post_id"`
	Pinned    bool   `json:"pinned"`
}

// ServerFolderCreateData represents a server folder creation event
type ServerFolderCreateData struct {
	Folder interface{} `json:"folder"`
}

// ServerFolderUpdateData represents a server folder update event
type ServerFolderUpdateData struct {
	Folder interface{} `json:"folder"`
}

// ServerFolderDeleteData represents a server folder deletion event
type ServerFolderDeleteData struct {
	FolderID string `json:"folder_id"`
}

// ServerFolderMoveData represents a server being moved to a folder
type ServerFolderMoveData struct {
	ServerID string  `json:"server_id"`
	FolderID *string `json:"folder_id,omitempty"` // nil means unassigned
}

// ServerFolderReorderData represents servers being reordered
type ServerFolderReorderData struct {
	FolderID    *string `json:"folder_id,omitempty"`
	ServerIDs   []string `json:"server_ids"`
}

