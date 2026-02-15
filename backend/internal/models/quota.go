package models

import (
	"time"

	"github.com/google/uuid"
)

// QuotaConfig represents instance-level quota defaults
type QuotaConfig struct {
	Storage  StorageQuotaConfig  `yaml:"storage" json:"storage"`
	Messages MessageQuotaConfig  `yaml:"messages" json:"messages"`
	Servers  ServerQuotaConfig   `yaml:"servers" json:"servers"`
	Voice    VoiceQuotaConfig    `yaml:"voice" json:"voice"`
	API      APIQuotaConfig      `yaml:"api" json:"api"`
}

// StorageQuotaConfig defines storage limits
type StorageQuotaConfig struct {
	Enabled                 bool     `yaml:"enabled" json:"enabled"`
	UserStorageMB           int64    `yaml:"user_storage_mb" json:"user_storage_mb"`                       // 0 = unlimited
	ServerStorageMB         int64    `yaml:"server_storage_mb" json:"server_storage_mb"`                   // 0 = unlimited
	MaxFileSizeMB           int64    `yaml:"max_file_size_mb" json:"max_file_size_mb"`                     // 0 = unlimited
	MaxAvatarSizeMB         int64    `yaml:"max_avatar_size_mb" json:"max_avatar_size_mb"`
	MaxEmojiSizeMB          int64    `yaml:"max_emoji_size_mb" json:"max_emoji_size_mb"`
	MaxAttachmentsPerMsg    int      `yaml:"max_attachments_per_message" json:"max_attachments_per_message"`
	MaxFilesPerUser         int      `yaml:"max_files_per_user" json:"max_files_per_user"`                 // 0 = unlimited
	AllowedExtensions       []string `yaml:"allowed_extensions" json:"allowed_extensions"`                 // empty = all
	BlockedExtensions       []string `yaml:"blocked_extensions" json:"blocked_extensions"`
}

// MessageQuotaConfig defines message limits
type MessageQuotaConfig struct {
	RateLimitMessages      int `yaml:"rate_limit_messages" json:"rate_limit_messages"`           // 0 = unlimited
	RateLimitWindowSeconds int `yaml:"rate_limit_window_seconds" json:"rate_limit_window_seconds"`
	DefaultSlowmodeSeconds int `yaml:"default_slowmode_seconds" json:"default_slowmode_seconds"`
	MaxSlowmodeSeconds     int `yaml:"max_slowmode_seconds" json:"max_slowmode_seconds"`
	MaxMessageLength       int `yaml:"max_message_length" json:"max_message_length"`             // 0 = unlimited
	MaxEmbedCount          int `yaml:"max_embed_count" json:"max_embed_count"`
	MaxMentionsPerMessage  int `yaml:"max_mentions_per_message" json:"max_mentions_per_message"` // 0 = unlimited
	MaxReactionsPerMessage int `yaml:"max_reactions_per_message" json:"max_reactions_per_message"`
}

// ServerQuotaConfig defines server limits
type ServerQuotaConfig struct {
	MaxServersOwned  int `yaml:"max_servers_owned" json:"max_servers_owned"`   // 0 = unlimited
	MaxServersJoined int `yaml:"max_servers_joined" json:"max_servers_joined"` // 0 = unlimited
	MaxChannels      int `yaml:"max_channels" json:"max_channels"`             // 0 = unlimited
	MaxRoles         int `yaml:"max_roles" json:"max_roles"`
	MaxEmoji         int `yaml:"max_emoji" json:"max_emoji"`
	MaxEmojiAnimated int `yaml:"max_emoji_animated" json:"max_emoji_animated"`
	MaxMembers       int `yaml:"max_members" json:"max_members"`
	MaxInvites       int `yaml:"max_invites" json:"max_invites"`
	MaxBans          int `yaml:"max_bans" json:"max_bans"`
	MaxWebhooks      int `yaml:"max_webhooks" json:"max_webhooks"`
}

// VoiceQuotaConfig defines voice/video limits
type VoiceQuotaConfig struct {
	Enabled                bool `yaml:"enabled" json:"enabled"`
	MaxBitrateKbps         int  `yaml:"max_bitrate_kbps" json:"max_bitrate_kbps"`
	MaxVideoHeight         int  `yaml:"max_video_height" json:"max_video_height"`
	MaxScreenShareFPS      int  `yaml:"max_screen_share_fps" json:"max_screen_share_fps"`
	MaxVoiceUsersPerChannel int `yaml:"max_voice_users_per_channel" json:"max_voice_users_per_channel"`
	MaxVideoUsersPerChannel int `yaml:"max_video_users_per_channel" json:"max_video_users_per_channel"`
	MaxCallDurationMinutes int  `yaml:"max_call_duration_minutes" json:"max_call_duration_minutes"` // 0 = unlimited
}

// APIQuotaConfig defines API rate limits
type APIQuotaConfig struct {
	RequestsPerMinute        int `yaml:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour          int `yaml:"requests_per_hour" json:"requests_per_hour"`
	BurstLimit               int `yaml:"burst_limit" json:"burst_limit"`
	MaxConcurrentConnections int `yaml:"max_concurrent_connections" json:"max_concurrent_connections"`
	MaxGuildsPerConnection   int `yaml:"max_guilds_per_connection" json:"max_guilds_per_connection"`
	BotRequestsPerMinute     int `yaml:"bot_requests_per_minute" json:"bot_requests_per_minute"`
	BotBurstLimit            int `yaml:"bot_burst_limit" json:"bot_burst_limit"`
}

// ServerQuotas represents server-specific quota overrides
type ServerQuotas struct {
	ServerID      uuid.UUID  `json:"server_id" db:"server_id"`
	StorageMB     *int64     `json:"storage_mb,omitempty" db:"storage_mb"`
	MaxFileSizeMB *int       `json:"max_file_size_mb,omitempty" db:"max_file_size_mb"`
	MaxChannels   *int       `json:"max_channels,omitempty" db:"max_channels"`
	MaxRoles      *int       `json:"max_roles,omitempty" db:"max_roles"`
	MaxEmoji      *int       `json:"max_emoji,omitempty" db:"max_emoji"`
	MaxMembers    *int       `json:"max_members,omitempty" db:"max_members"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// RoleQuotas represents role-specific quota overrides
type RoleQuotas struct {
	RoleID          uuid.UUID `json:"role_id" db:"role_id"`
	StorageMB       *int64    `json:"storage_mb,omitempty" db:"storage_mb"`
	MaxFileSizeMB   *int      `json:"max_file_size_mb,omitempty" db:"max_file_size_mb"`
	MessageRateLimit *int     `json:"message_rate_limit,omitempty" db:"message_rate_limit"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// UserQuotas represents user-specific quota overrides
type UserQuotas struct {
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	StorageMB        *int64    `json:"storage_mb,omitempty" db:"storage_mb"`
	MaxFileSizeMB    *int      `json:"max_file_size_mb,omitempty" db:"max_file_size_mb"`
	MaxServersOwned  *int      `json:"max_servers_owned,omitempty" db:"max_servers_owned"`
	MaxServersJoined *int      `json:"max_servers_joined,omitempty" db:"max_servers_joined"`
	MessageRateLimit *int      `json:"message_rate_limit,omitempty" db:"message_rate_limit"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// StorageUsage tracks storage consumption
type StorageUsage struct {
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	UsedBytes   int64     `json:"used_bytes" db:"used_bytes"`
	FileCount   int       `json:"file_count" db:"file_count"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}

// EffectiveLimits represents the calculated limits for a user
type EffectiveLimits struct {
	StorageMB        int64  `json:"storage_mb"`         // -1 = unlimited
	MaxFileSizeMB    int64  `json:"max_file_size_mb"`   // -1 = unlimited
	MessageRateLimit int    `json:"message_rate_limit"` // 0 = unlimited
	SlowmodeSeconds  int    `json:"slowmode_seconds"`
	MaxServersOwned  int    `json:"max_servers_owned"`  // 0 = unlimited
	MaxServersJoined int    `json:"max_servers_joined"` // 0 = unlimited
	Sources          map[string]string `json:"sources"` // field -> source level
}

// StorageInfo provides user storage information
type StorageInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	UsedBytes   int64     `json:"used_bytes"`
	UsedMB      float64   `json:"used_mb"`
	LimitBytes  int64     `json:"limit_bytes"`  // -1 = unlimited
	LimitMB     int64     `json:"limit_mb"`     // -1 = unlimited
	FileCount   int       `json:"file_count"`
	Percentage  float64   `json:"percentage"`   // 0-100, -1 if unlimited
	IsUnlimited bool      `json:"is_unlimited"`
}

// QuotaError represents a quota-related error
type QuotaError struct {
	Type       string                 `json:"error"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RetryAfter int                    `json:"retry_after,omitempty"`
	UpgradeURL string                 `json:"upgrade_url,omitempty"`
}

func (e *QuotaError) Error() string {
	return e.Message
}

// NewStorageQuotaError creates a storage quota exceeded error
func NewStorageQuotaError(usedMB, limitMB, fileSizeMB int64) *QuotaError {
	return &QuotaError{
		Type:    "quota_exceeded",
		Message: "You have exceeded your storage quota",
		Details: map[string]interface{}{
			"used_mb":      usedMB,
			"limit_mb":     limitMB,
			"file_size_mb": fileSizeMB,
			"would_be_mb":  usedMB + fileSizeMB,
		},
		UpgradeURL: "/settings/premium",
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(retryAfter int, limit, windowSeconds, slowmode int) *QuotaError {
	return &QuotaError{
		Type:       "rate_limited",
		Message:    "You are sending messages too quickly",
		RetryAfter: retryAfter,
		Details: map[string]interface{}{
			"limit":            limit,
			"window_seconds":   windowSeconds,
			"slowmode_seconds": slowmode,
		},
	}
}

// NewFileTooLargeError creates a file size error
func NewFileTooLargeError(fileSizeMB, maxSizeMB int64) *QuotaError {
	return &QuotaError{
		Type:    "file_too_large",
		Message: "File exceeds maximum size",
		Details: map[string]interface{}{
			"file_size_mb": fileSizeMB,
			"max_size_mb":  maxSizeMB,
		},
	}
}

// DefaultQuotaConfig returns sensible defaults
func DefaultQuotaConfig() *QuotaConfig {
	return &QuotaConfig{
		Storage: StorageQuotaConfig{
			Enabled:              true,
			UserStorageMB:        500,   // 500 MB per user
			ServerStorageMB:      5000,  // 5 GB per server
			MaxFileSizeMB:        25,    // 25 MB max file
			MaxAvatarSizeMB:      8,
			MaxEmojiSizeMB:       1,
			MaxAttachmentsPerMsg: 10,
			MaxFilesPerUser:      0,     // unlimited
			AllowedExtensions:    []string{},
			BlockedExtensions:    []string{"exe", "bat", "cmd", "sh", "ps1", "msi"},
		},
		Messages: MessageQuotaConfig{
			RateLimitMessages:      5,
			RateLimitWindowSeconds: 5,
			DefaultSlowmodeSeconds: 0,
			MaxSlowmodeSeconds:     21600,
			MaxMessageLength:       2000,
			MaxEmbedCount:          10,
			MaxMentionsPerMessage:  20,
			MaxReactionsPerMessage: 20,
		},
		Servers: ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
			MaxChannels:      500,
			MaxRoles:         250,
			MaxEmoji:         50,
			MaxEmojiAnimated: 50,
			MaxMembers:       500000,
			MaxInvites:       1000,
			MaxBans:          100000,
			MaxWebhooks:      15,
		},
		Voice: VoiceQuotaConfig{
			Enabled:                 true,
			MaxBitrateKbps:          384,
			MaxVideoHeight:          1080,
			MaxScreenShareFPS:       30,
			MaxVoiceUsersPerChannel: 99,
			MaxVideoUsersPerChannel: 25,
			MaxCallDurationMinutes:  0, // unlimited
		},
		API: APIQuotaConfig{
			RequestsPerMinute:        60,
			RequestsPerHour:          1000,
			BurstLimit:               10,
			MaxConcurrentConnections: 5,
			MaxGuildsPerConnection:   100,
			BotRequestsPerMinute:     120,
			BotBurstLimit:            20,
		},
	}
}

// UnlimitedQuotaConfig returns a config with all limits disabled
func UnlimitedQuotaConfig() *QuotaConfig {
	return &QuotaConfig{
		Storage: StorageQuotaConfig{
			Enabled:              true,
			UserStorageMB:        0, // unlimited
			ServerStorageMB:      0,
			MaxFileSizeMB:        0,
			MaxAvatarSizeMB:      0,
			MaxEmojiSizeMB:       0,
			MaxAttachmentsPerMsg: 0,
			MaxFilesPerUser:      0,
			AllowedExtensions:    []string{},
			BlockedExtensions:    []string{},
		},
		Messages: MessageQuotaConfig{
			RateLimitMessages:      0, // unlimited
			RateLimitWindowSeconds: 0,
			DefaultSlowmodeSeconds: 0,
			MaxSlowmodeSeconds:     0,
			MaxMessageLength:       0,
			MaxEmbedCount:          0,
			MaxMentionsPerMessage:  0,
			MaxReactionsPerMessage: 0,
		},
		Servers: ServerQuotaConfig{
			MaxServersOwned:  0,
			MaxServersJoined: 0,
			MaxChannels:      0,
			MaxRoles:         0,
			MaxEmoji:         0,
			MaxEmojiAnimated: 0,
			MaxMembers:       0,
			MaxInvites:       0,
			MaxBans:          0,
			MaxWebhooks:      0,
		},
		Voice: VoiceQuotaConfig{
			Enabled:                 true,
			MaxBitrateKbps:          0,
			MaxVideoHeight:          0,
			MaxScreenShareFPS:       0,
			MaxVoiceUsersPerChannel: 0,
			MaxVideoUsersPerChannel: 0,
			MaxCallDurationMinutes:  0,
		},
		API: APIQuotaConfig{
			RequestsPerMinute:        0,
			RequestsPerHour:          0,
			BurstLimit:               0,
			MaxConcurrentConnections: 0,
			MaxGuildsPerConnection:   0,
			BotRequestsPerMinute:     0,
			BotBurstLimit:            0,
		},
	}
}

// IsUnlimited checks if a limit value represents unlimited
func IsUnlimited(value int64) bool {
	return value <= 0
}
