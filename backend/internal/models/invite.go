package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry represents an action in the audit log
type AuditLogEntry struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ServerID        uuid.UUID       `json:"server_id" db:"server_id"`
	ActorID         uuid.UUID       `json:"actor_id" db:"actor_id"`
	ActionType      string          `json:"action_type" db:"action_type"`
	ActionCategory  int             `json:"action_category" db:"action_category"`
	TargetID        *uuid.UUID      `json:"target_id,omitempty" db:"target_id"`
	TargetType      string          `json:"target_type,omitempty" db:"target_type"`
	Reason          string          `json:"reason,omitempty" db:"reason"`
	Changes         json.RawMessage `json:"changes,omitempty" db:"changes"`
	Metadata        json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	IPAddress       *string         `json:"ip_address,omitempty" db:"ip_address"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`

	// Populated on fetch
	Actor   *PublicUser `json:"actor,omitempty"`
	Target  *PublicUser `json:"target,omitempty"`
}

// Change represents a single change in an audit log entry
type Change struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// AuditLogActionCategory represents the numeric category for audit log actions
type AuditLogActionCategory int

const (
	// Category 10-19: Member events
	AuditLogCategoryMember AuditLogActionCategory = 10

	// Category 20-29: Channel events
	AuditLogCategoryChannel AuditLogActionCategory = 20

	// Category 30-39: Server events
	AuditLogCategoryServer AuditLogActionCategory = 30

	// Category 40-49: Message events
	AuditLogCategoryMessage AuditLogActionCategory = 40

	// Category 50-59: Permission events
	AuditLogCategoryPermission AuditLogActionCategory = 50

	// Category 60-69: Integration events
	AuditLogCategoryIntegration AuditLogActionCategory = 60

	// Category 70-79: Voice/stage events
	AuditLogCategoryVoice AuditLogActionCategory = 70

	// Category 80-89: Auto-moderation events
	AuditLogCategoryAutoMod AuditLogActionCategory = 80
)

// Audit log action types - Discord parity (80+ event types)
const (
	// === Member Actions (10-19) ===
	AuditLogMemberJoin              = "MEMBER_JOIN"
	AuditLogMemberLeave             = "MEMBER_LEAVE"
	AuditLogMemberKick              = "MEMBER_KICK"
	AuditLogMemberBan               = "MEMBER_BAN"
	AuditLogMemberUnban             = "MEMBER_UNBAN"
	AuditLogMemberUpdate            = "MEMBER_UPDATE"
	AuditLogMemberRoleUpdate        = "MEMBER_ROLE_UPDATE"
	AuditLogMemberNicknameUpdate    = "MEMBER_NICKNAME_UPDATE"
	AuditLogMemberTimeout           = "MEMBER_TIMEOUT"
	AuditLogMemberTimeoutRemove     = "MEMBER_TIMEOUT_REMOVE"
	AuditLogMemberVoiceMove         = "MEMBER_VOICE_MOVE"
	AuditLogMemberVoiceKick         = "MEMBER_VOICE_KICK"
	AuditLogMemberDisconnect       = "MEMBER_DISCONNECT"
	AuditLogMemberAdd               = "MEMBER_ADD" // Added via invite
	AuditLogMemberRemove            = "MEMBER_REMOVE"
	AuditLogMemberVerify           = "MEMBER_VERIFY"
	AuditLogMemberLinkAccount       = "MEMBER_LINK_ACCOUNT"

	// === Channel Actions (20-29) ===
	AuditLogChannelCreate           = "CHANNEL_CREATE"
	AuditLogChannelUpdate           = "CHANNEL_UPDATE"
	AuditLogChannelDelete           = "CHANNEL_DELETE"
	AuditLogChannelPinsUpdate       = "CHANNEL_PINS_UPDATE"
	AuditLogChannelPermissionUpdate = "CHANNEL_PERMISSION_UPDATE"
	AuditLogChannelOverrideCreate   = "CHANNEL_OVERRIDE_CREATE"
	AuditLogChannelOverrideUpdate   = "CHANNEL_OVERRIDE_UPDATE"
	AuditLogChannelOverrideDelete   = "CHANNEL_OVERRIDE_DELETE"
	AuditLogChannelFollowNews       = "CHANNEL_FOLLOW_NEWS"
	AuditLogChannelGroupJoin        = "CHANNEL_GROUP_JOIN"
	AuditLogChannelGroupLeave       = "CHANNEL_GROUP_LEAVE"

	// === Server Settings Actions (30-39) ===
	AuditLogServerUpdate            = "SERVER_UPDATE"
	AuditLogServerNameUpdate        = "SERVER_NAME_UPDATE"
	AuditLogServerIconUpdate        = "SERVER_ICON_UPDATE"
	AuditLogServerBannerUpdate      = "SERVER_BANNER_UPDATE"
	AuditLogServerSplashUpdate      = "SERVER_SPLASH_UPDATE"
	AuditLogServerDescriptionUpdate = "SERVER_DESCRIPTION_UPDATE"
	AuditLogServerRegionUpdate      = "SERVER_REGION_UPDATE"
	AuditLogServerVerificationUpdate = "SERVER_VERIFICATION_UPDATE"
	AuditLogServerAFKChannelUpdate  = "SERVER_AFK_CHANNEL_UPDATE"
	AuditLogServerAFKTimeoutUpdate   = "SERVER_AFK_TIMEOUT_UPDATE"
	AuditLogServerDefaultChannelUpdate = "SERVER_DEFAULT_CHANNEL_UPDATE"
	AuditLogServerWidgetUpdate      = "SERVER_WIDGET_UPDATE"
	AuditLogServerModerationUpdate  = "SERVER_MODERATION_UPDATE"
	AuditLogServerMFALevelUpdate    = "SERVER_MFA_LEVEL_UPDATE"
	AuditLogServerExplicitFilterUpdate = "SERVER_EXPLICIT_FILTER_UPDATE"
	AuditLogServerVanityURLUpdate   = "SERVER_VANITY_URL_UPDATE"
	AuditLogServerWelcomeScreenUpdate = "SERVER_WELCOME_SCREEN_UPDATE"
	AuditLogServerNSFWUpdate        = "SERVER_NSFW_UPDATE"

	// === Role Actions (50-59) ===
	AuditLogRoleCreate             = "ROLE_CREATE"
	AuditLogRoleUpdate             = "ROLE_UPDATE"
	AuditLogRoleDelete             = "ROLE_DELETE"
	AuditLogRolePermissionUpdate   = "ROLE_PERMISSION_UPDATE"

	// === Message Actions (40-49) ===
	AuditLogMessageDelete          = "MESSAGE_DELETE"
	AuditLogMessageBulkDelete      = "MESSAGE_BULK_DELETE"
	AuditLogMessagePin             = "MESSAGE_PIN"
	AuditLogMessageUnpin           = "MESSAGE_UNPIN"
	AuditLogMessageReactionAdd     = "MESSAGE_REACTION_ADD"
	AuditLogMessageReactionRemove   = "MESSAGE_REACTION_REMOVE"
	AuditLogMessageReactionRemoveAll = "MESSAGE_REACTION_REMOVE_ALL"
	AuditLogMessageEdit           = "MESSAGE_EDIT"
	AuditLogMessageAck            = "MESSAGE_ACK"
	AuditLogMessagePublish        = "MESSAGE_PUBLISH"

	// === Integration Actions (60-69) ===
	AuditLogWebhookCreate          = "WEBHOOK_CREATE"
	AuditLogWebhookUpdate          = "WEBHOOK_UPDATE"
	AuditLogWebhookDelete          = "WEBHOOK_DELETE"
	AuditLogWebhookChannelMove     = "WEBHOOK_CHANNEL_MOVE"
	AuditLogEmojiCreate           = "EMOJI_CREATE"
	AuditLogEmojiUpdate           = "EMOJI_UPDATE"
	AuditLogEmojiDelete           = "EMOJI_DELETE"
	AuditLogStickerCreate         = "STICKER_CREATE"
	AuditLogStickerUpdate         = "STICKER_UPDATE"
	AuditLogStickerDelete         = "STICKER_DELETE"
	AuditLogInviteCreate          = "INVITE_CREATE"
	AuditLogInviteUpdate          = "INVITE_UPDATE"
	AuditLogInviteDelete          = "INVITE_DELETE"
	AuditLogSlashCommandCreate    = "SLASH_COMMAND_CREATE"
	AuditLogSlashCommandUpdate     = "SLASH_COMMAND_UPDATE"
	AuditLogSlashCommandDelete     = "SLASH_COMMAND_DELETE"
	AuditLogSoundboardSoundCreate  = "SOUNDBOARD_SOUND_CREATE"
	AuditLogSoundboardSoundUpdate  = "SOUNDBOARD_SOUND_UPDATE"
	AuditLogSoundboardSoundDelete  = "SOUNDBOARD_SOUND_DELETE"

	// === Voice Actions (70-79) ===
	AuditLogVoiceChannelJoin       = "VOICE_CHANNEL_JOIN"
	AuditLogVoiceChannelLeave      = "VOICE_CHANNEL_LEAVE"
	AuditLogVoiceChannelMove       = "VOICE_CHANNEL_MOVE"
	AuditLogVoiceChannelKick       = "VOICE_CHANNEL_KICK"
	AuditLogVoiceChannelMute       = "VOICE_CHANNEL_MUTE"
	AuditLogVoiceChannelDeafen     = "VOICE_CHANNEL_DEAFEN"
	AuditLogVoiceChannelSelfMute   = "VOICE_CHANNEL_SELF_MUTE"
	AuditLogVoiceChannelSelfDeafen = "VOICE_CHANNEL_SELF_DEAFEN"
	AuditLogVoiceStreamStart       = "VOICE_STREAM_START"
	AuditLogVoiceStreamStop        = "VOICE_STREAM_STOP"
	AuditLogVoiceVideoStart        = "VOICE_VIDEO_START"
	AuditLogVoiceVideoStop         = "VOICE_VIDEO_STOP"
	AuditLogStageInstanceCreate    = "STAGE_INSTANCE_CREATE"
	AuditLogStageInstanceUpdate    = "STAGE_INSTANCE_UPDATE"
	AuditLogStageInstanceDelete     = "STAGE_INSTANCE_DELETE"

	// === Auto-Moderation Actions (80-89) ===
	AuditLogAutoModFlag            = "AUTOMOD_FLAG"
	AuditLogAutoModBlock           = "AUTOMOD_BLOCK"
	AuditLogAutoModWarn            = "AUTOMOD_WARN"
	AuditLogAutoModTimeout         = "AUTOMOD_TIMEOUT"
	AuditLogAutoModKick            = "AUTOMOD_KICK"
	AuditLogAutoModBan             = "AUTOMOD_BAN"
	AuditLogAutoModMessageDelete   = "AUTOMOD_MESSAGE_DELETE"
	AuditLogAutoModSpanBlock       = "AUTOMOD_SPAM_BLOCK"
	AuditLogAutoModMentionSpam     = "AUTOMOD_MENTION_SPAM"
	AuditLogAutoModWordFilter      = "AUTOMOD_WORD_FILTER"
	AuditLogAutoModLinkFilter      = "AUTOMOD_LINK_FILTER"
	AuditLogAutoModAttachmentBlock = "AUTOMOD_ATTACHMENT_BLOCK"

	// === Thread Actions (45-49) ===
	AuditLogThreadCreate           = "THREAD_CREATE"
	AuditLogThreadUpdate           = "THREAD_UPDATE"
	AuditLogThreadDelete           = "THREAD_DELETE"
	AuditLogThreadMemberUpdate     = "THREAD_MEMBER_UPDATE"

	// === Application/Bot Actions (65-69) ===
	AuditLogBotAdd                 = "BOT_ADD"
	AuditLogApplicationUpdate      = "APPLICATION_UPDATE"
	AuditLogApplicationCommandPermissionUpdate = "APPLICATION_COMMAND_PERMISSION_UPDATE"
)

// GetActionCategory returns the numeric category for an action type
func GetActionCategory(actionType string) int {
	switch actionType {
	// Member actions (10-19)
	case AuditLogMemberJoin, AuditLogMemberLeave, AuditLogMemberKick, AuditLogMemberBan,
		AuditLogMemberUnban, AuditLogMemberUpdate, AuditLogMemberRoleUpdate,
		AuditLogMemberNicknameUpdate, AuditLogMemberTimeout, AuditLogMemberTimeoutRemove,
		AuditLogMemberVoiceMove, AuditLogMemberVoiceKick, AuditLogMemberDisconnect,
		AuditLogMemberAdd, AuditLogMemberRemove, AuditLogMemberVerify, AuditLogMemberLinkAccount:
		return int(AuditLogCategoryMember)

	// Channel actions (20-29)
	case AuditLogChannelCreate, AuditLogChannelUpdate, AuditLogChannelDelete,
		AuditLogChannelPinsUpdate, AuditLogChannelPermissionUpdate,
		AuditLogChannelOverrideCreate, AuditLogChannelOverrideUpdate,
		AuditLogChannelOverrideDelete, AuditLogChannelFollowNews,
		AuditLogChannelGroupJoin, AuditLogChannelGroupLeave:
		return int(AuditLogCategoryChannel)

	// Message actions (40-49)
	case AuditLogMessageDelete, AuditLogMessageBulkDelete, AuditLogMessagePin,
		AuditLogMessageUnpin, AuditLogMessageReactionAdd, AuditLogMessageReactionRemove,
		AuditLogMessageReactionRemoveAll, AuditLogMessageEdit, AuditLogMessageAck,
		AuditLogMessagePublish:
		return int(AuditLogCategoryMessage)

	// Thread actions (45-49)
	case AuditLogThreadCreate, AuditLogThreadUpdate, AuditLogThreadDelete,
		AuditLogThreadMemberUpdate:
		return 45

	// Server actions (30-39)
	case AuditLogServerUpdate, AuditLogServerNameUpdate, AuditLogServerIconUpdate,
		AuditLogServerBannerUpdate, AuditLogServerSplashUpdate, AuditLogServerDescriptionUpdate,
		AuditLogServerRegionUpdate, AuditLogServerVerificationUpdate, AuditLogServerAFKChannelUpdate,
		AuditLogServerAFKTimeoutUpdate, AuditLogServerDefaultChannelUpdate, AuditLogServerWidgetUpdate,
		AuditLogServerModerationUpdate, AuditLogServerMFALevelUpdate, AuditLogServerExplicitFilterUpdate,
		AuditLogServerVanityURLUpdate, AuditLogServerWelcomeScreenUpdate, AuditLogServerNSFWUpdate:
		return int(AuditLogCategoryServer)

	// Role actions (50-59)
	case AuditLogRoleCreate, AuditLogRoleUpdate, AuditLogRoleDelete, AuditLogRolePermissionUpdate:
		return int(AuditLogCategoryPermission)

	// Integration actions (60-69)
	case AuditLogWebhookCreate, AuditLogWebhookUpdate, AuditLogWebhookDelete,
		AuditLogWebhookChannelMove, AuditLogEmojiCreate, AuditLogEmojiUpdate,
		AuditLogEmojiDelete, AuditLogStickerCreate, AuditLogStickerUpdate,
		AuditLogStickerDelete, AuditLogInviteCreate, AuditLogInviteUpdate,
		AuditLogInviteDelete, AuditLogSlashCommandCreate, AuditLogSlashCommandUpdate,
		AuditLogSlashCommandDelete, AuditLogSoundboardSoundCreate,
		AuditLogSoundboardSoundUpdate, AuditLogSoundboardSoundDelete:
		return int(AuditLogCategoryIntegration)

	// Application/Bot actions (65-69)
	case AuditLogBotAdd, AuditLogApplicationUpdate, AuditLogApplicationCommandPermissionUpdate:
		return 65

	// Voice actions (70-79)
	case AuditLogVoiceChannelJoin, AuditLogVoiceChannelLeave, AuditLogVoiceChannelMove,
		AuditLogVoiceChannelKick, AuditLogVoiceChannelMute, AuditLogVoiceChannelDeafen,
		AuditLogVoiceChannelSelfMute, AuditLogVoiceChannelSelfDeafen, AuditLogVoiceStreamStart,
		AuditLogVoiceStreamStop, AuditLogVoiceVideoStart, AuditLogVoiceVideoStop,
		AuditLogStageInstanceCreate, AuditLogStageInstanceUpdate, AuditLogStageInstanceDelete:
		return int(AuditLogCategoryVoice)

	// Auto-mod actions (80-89)
	case AuditLogAutoModFlag, AuditLogAutoModBlock, AuditLogAutoModWarn,
		AuditLogAutoModTimeout, AuditLogAutoModKick, AuditLogAutoModBan,
		AuditLogAutoModMessageDelete, AuditLogAutoModSpanBlock, AuditLogAutoModMentionSpam,
		AuditLogAutoModWordFilter, AuditLogAutoModLinkFilter, AuditLogAutoModAttachmentBlock:
		return int(AuditLogCategoryAutoMod)

	default:
		return 0
	}
}

// GetAllAuditLogActionTypes returns all valid audit log action types organized by category
func GetAllAuditLogActionTypes() []string {
	return []string{
		// Member actions
		AuditLogMemberJoin, AuditLogMemberLeave, AuditLogMemberKick, AuditLogMemberBan,
		AuditLogMemberUnban, AuditLogMemberUpdate, AuditLogMemberRoleUpdate,
		AuditLogMemberNicknameUpdate, AuditLogMemberTimeout, AuditLogMemberTimeoutRemove,
		AuditLogMemberVoiceMove, AuditLogMemberVoiceKick, AuditLogMemberDisconnect,
		AuditLogMemberAdd, AuditLogMemberRemove, AuditLogMemberVerify, AuditLogMemberLinkAccount,
		// Channel actions
		AuditLogChannelCreate, AuditLogChannelUpdate, AuditLogChannelDelete,
		AuditLogChannelPinsUpdate, AuditLogChannelPermissionUpdate,
		AuditLogChannelOverrideCreate, AuditLogChannelOverrideUpdate,
		AuditLogChannelOverrideDelete, AuditLogChannelFollowNews,
		AuditLogChannelGroupJoin, AuditLogChannelGroupLeave,
		// Server actions
		AuditLogServerUpdate, AuditLogServerNameUpdate, AuditLogServerIconUpdate,
		AuditLogServerBannerUpdate, AuditLogServerSplashUpdate, AuditLogServerDescriptionUpdate,
		AuditLogServerRegionUpdate, AuditLogServerVerificationUpdate, AuditLogServerAFKChannelUpdate,
		AuditLogServerAFKTimeoutUpdate, AuditLogServerDefaultChannelUpdate, AuditLogServerWidgetUpdate,
		AuditLogServerModerationUpdate, AuditLogServerMFALevelUpdate, AuditLogServerExplicitFilterUpdate,
		AuditLogServerVanityURLUpdate, AuditLogServerWelcomeScreenUpdate, AuditLogServerNSFWUpdate,
		// Message actions
		AuditLogMessageDelete, AuditLogMessageBulkDelete, AuditLogMessagePin,
		AuditLogMessageUnpin, AuditLogMessageReactionAdd, AuditLogMessageReactionRemove,
		AuditLogMessageReactionRemoveAll, AuditLogMessageEdit, AuditLogMessageAck,
		AuditLogMessagePublish,
		// Thread actions
		AuditLogThreadCreate, AuditLogThreadUpdate, AuditLogThreadDelete,
		AuditLogThreadMemberUpdate,
		// Role actions
		AuditLogRoleCreate, AuditLogRoleUpdate, AuditLogRoleDelete, AuditLogRolePermissionUpdate,
		// Integration actions
		AuditLogWebhookCreate, AuditLogWebhookUpdate, AuditLogWebhookDelete,
		AuditLogWebhookChannelMove, AuditLogEmojiCreate, AuditLogEmojiUpdate,
		AuditLogEmojiDelete, AuditLogStickerCreate, AuditLogStickerUpdate,
		AuditLogStickerDelete, AuditLogInviteCreate, AuditLogInviteUpdate,
		AuditLogInviteDelete, AuditLogSlashCommandCreate, AuditLogSlashCommandUpdate,
		AuditLogSlashCommandDelete, AuditLogSoundboardSoundCreate,
		AuditLogSoundboardSoundUpdate, AuditLogSoundboardSoundDelete,
		// Application/Bot actions
		AuditLogBotAdd, AuditLogApplicationUpdate, AuditLogApplicationCommandPermissionUpdate,
		// Voice actions
		AuditLogVoiceChannelJoin, AuditLogVoiceChannelLeave, AuditLogVoiceChannelMove,
		AuditLogVoiceChannelKick, AuditLogVoiceChannelMute, AuditLogVoiceChannelDeafen,
		AuditLogVoiceChannelSelfMute, AuditLogVoiceChannelSelfDeafen, AuditLogVoiceStreamStart,
		AuditLogVoiceStreamStop, AuditLogVoiceVideoStart, AuditLogVoiceVideoStop,
		AuditLogStageInstanceCreate, AuditLogStageInstanceUpdate, AuditLogStageInstanceDelete,
		// Auto-mod actions
		AuditLogAutoModFlag, AuditLogAutoModBlock, AuditLogAutoModWarn,
		AuditLogAutoModTimeout, AuditLogAutoModKick, AuditLogAutoModBan,
		AuditLogAutoModMessageDelete, AuditLogAutoModSpanBlock, AuditLogAutoModMentionSpam,
		AuditLogAutoModWordFilter, AuditLogAutoModLinkFilter, AuditLogAutoModAttachmentBlock,
	}
}

// GetAuditLogCategories returns audit log categories with metadata
func GetAuditLogCategories() []AuditLogCategoryInfo {
	return []AuditLogCategoryInfo{
		{Category: int(AuditLogCategoryMember), Name: "Member", Description: "Member join, leave, kick, ban, timeout, role changes"},
		{Category: int(AuditLogCategoryChannel), Name: "Channel", Description: "Channel create, update, delete, permissions"},
		{Category: int(AuditLogCategoryServer), Name: "Server", Description: "Server settings, name, icon, region changes"},
		{Category: int(AuditLogCategoryMessage), Name: "Message", Description: "Message delete, pin, reactions, edits"},
		{Category: 45, Name: "Thread", Description: "Thread create, update, delete, member updates"},
		{Category: int(AuditLogCategoryPermission), Name: "Role", Description: "Role create, update, delete, permissions"},
		{Category: int(AuditLogCategoryIntegration), Name: "Integration", Description: "Webhooks, emojis, stickers, invites, slash commands"},
		{Category: 65, Name: "Application", Description: "Bot additions, application updates"},
		{Category: int(AuditLogCategoryVoice), Name: "Voice", Description: "Voice channel join, leave, move, stage events"},
		{Category: int(AuditLogCategoryAutoMod), Name: "AutoMod", Description: "Auto-moderation triggers and actions"},
	}
}

// AuditLogCategoryInfo provides metadata about an audit log category
type AuditLogCategoryInfo struct {
	Category    int    `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AuditLogFilterParams holds filtering options for audit log queries
type AuditLogFilterParams struct {
	ActionType   string     `query:"action_type"`
	ActionCategory int      `query:"action_category"`
	ActorID      *uuid.UUID `query:"-"`
	TargetID      *uuid.UUID `query:"-"`
	TargetType   string     `query:"target_type"`
	Before       *time.Time `query:"-"`
	After        *time.Time `query:"-"`
	ReasonKeyword string    `query:"reason_keyword"`
	Limit        int        `query:"limit"`
	Offset       int        `query:"offset"`
}

// Normalize applies defaults and limits to filter params
func (p *AuditLogFilterParams) Normalize() {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// AuditLogExportFormat represents supported export formats
type AuditLogExportFormat string

const (
	AuditLogExportJSON AuditLogExportFormat = "json"
	AuditLogExportCSV  AuditLogExportFormat = "csv"
)

// AuditLogExportRequest represents an audit log export request
type AuditLogExportRequest struct {
	Format     AuditLogExportFormat `json:"format"`
	ActionType string              `json:"action_type,omitempty"`
	Before     *time.Time           `json:"before,omitempty"`
	After      *time.Time           `json:"after,omitempty"`
}

// ModerationDashboardSummary provides a quick overview for the moderation dashboard
type ModerationDashboardSummary struct {
	ServerID         uuid.UUID `json:"server_id"`
	TotalActions     int       `json:"total_actions"`
	TopAction        string    `json:"top_action"`
	TopActionCount   int       `json:"top_action_count"`
	UniqueModerators int       `json:"unique_moderators"`
	UniqueTargets    int       `json:"unique_targets"`
	ActionBreakdown  map[string]int `json:"action_breakdown"`
	TrendDirection   string    `json:"trend_direction"` // "up", "down", "stable"
	TrendPercent     float64   `json:"trend_percent"`
}

// ModerationTimeSeriesPoint represents a single point in moderation time series
type ModerationTimeSeriesPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	TotalActions int      `json:"total_actions"`
	ByCategory  map[int]int `json:"by_category"`
	ByType      map[string]int `json:"by_type"`
}

// AutoModStats contains auto-moderation statistics
type AutoModStats struct {
	TotalTriggers  int `json:"total_triggers"`
	Blocks        int `json:"blocks"`
	Warns         int `json:"warns"`
	Timeouts      int `json:"timeouts"`
	Kicks         int `json:"kicks"`
	Bans          int `json:"bans"`
	MessageDeletes int `json:"message_deletes"`
	MentionSpam   int `json:"mention_spam"`
	WordFilter    int `json:"word_filter"`
	LinkFilter    int `json:"link_filter"`
}

// ModerationAnalytics contains comprehensive moderation analytics
type ModerationAnalytics struct {
	ServerID          uuid.UUID               `json:"server_id"`
	Period            string                  `json:"period"`
	Ratios            *ModerationRatiosStats   `json:"ratios"`
	TrendData         []DailyModerationTrend   `json:"trend_data"`
	ModeratorActivity []ModeratorStats         `json:"moderator_activity"`
	RepeatOffenders   []RepeatOffenderStats    `json:"repeat_offenders"`
}

// ModerationRatiosStats contains warn/ban/mute ratios
type ModerationRatiosStats struct {
	Total          int     `json:"total"`
	Bans           int     `json:"bans"`
	Unbans         int     `json:"unbans"`
	Mutes          int     `json:"mutes"`
	Unmutes        int     `json:"unmutes"`
	Warns          int     `json:"warns"`
	Kicks          int     `json:"kicks"`
	MessageDeletes int    `json:"message_deletes"`
	BulkDeletes    int     `json:"bulk_deletes"`
	BanRatio       float64 `json:"ban_ratio_percent"`
	MuteRatio      float64 `json:"mute_ratio_percent"`
	WarnRatio      float64 `json:"warn_ratio_percent"`
	KickRatio      float64 `json:"kick_ratio_percent"`
}

// DailyModerationTrend represents moderation actions per day
type DailyModerationTrend struct {
	Date           time.Time `json:"date"`
	TotalActions   int       `json:"total_actions"`
	Bans           int       `json:"bans"`
	Kicks          int       `json:"kicks"`
	Mutes          int       `json:"mutes"`
	Warns          int       `json:"warns"`
	MessageDeletes int       `json:"message_deletes"`
}

// ModeratorStats represents a moderator's activity
type ModeratorStats struct {
	ModeratorID      uuid.UUID `json:"moderator_id"`
	TotalActions     int       `json:"total_actions"`
	Bans             int       `json:"bans"`
	Unbans           int       `json:"unbans"`
	Kicks            int       `json:"kicks"`
	Mutes            int       `json:"mutes"`
	Unmutes          int       `json:"unmutes"`
	Warns            int       `json:"warns"`
	MessageDeletes   int       `json:"message_deletes"`
	UniqueTargets    int       `json:"unique_targets"`
}

// RepeatOffenderStats represents a repeat offender's stats
type RepeatOffenderStats struct {
	UserID              uuid.UUID `json:"user_id"`
	ModerationCount     int       `json:"moderation_count"`
	DifferentModerators  int       `json:"different_moderators"`
	Bans                int       `json:"bans"`
	Warns               int       `json:"warns"`
	Mutes               int       `json:"mutes"`
}
