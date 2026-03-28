package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DigestFrequency represents how often digests are sent
type DigestFrequency string

const (
	DigestFrequencyHourly DigestFrequency = "hourly"
	DigestFrequencyDaily  DigestFrequency = "daily"
	DigestFrequencyWeekly DigestFrequency = "weekly"
)

// DigestAggregationMode represents how messages are grouped in digests
type DigestAggregationMode string

const (
	DigestAggregationChannel DigestAggregationMode = "channel"
	DigestAggregationServer  DigestAggregationMode = "server"
)

// DigestMode represents channel/server specific digest behavior
type DigestMode string

const (
	DigestModeInherit   DigestMode = "inherit"   // Use global settings
	DigestModeInclude   DigestMode = "include"   // Always include in digests
	DigestModeExclude   DigestMode = "exclude"   // Never include in digests
	DigestModeImmediate DigestMode = "immediate" // Send immediately, don't batch (channel only)
)

// DigestStatus represents the status of a digest delivery
type DigestStatus string

const (
	DigestStatusPending DigestStatus = "pending"
	DigestStatusSent    DigestStatus = "sent"
	DigestStatusFailed  DigestStatus = "failed"
	DigestStatusSkipped DigestStatus = "skipped"
)

// DigestPreferences represents a user's global digest preferences
type DigestPreferences struct {
	ID                   uuid.UUID             `json:"id" db:"id"`
	UserID               uuid.UUID             `json:"user_id" db:"user_id"`
	Enabled              bool                  `json:"enabled" db:"enabled"`
	Frequency            DigestFrequency       `json:"frequency" db:"frequency"`
	PreferredHour        int                   `json:"preferred_hour" db:"preferred_hour"`
	PreferredDay         int                   `json:"preferred_day" db:"preferred_day"`
	AggregationMode      DigestAggregationMode `json:"aggregation_mode" db:"aggregation_mode"`
	MaxMessagesPerSource int                   `json:"max_messages_per_source" db:"max_messages_per_source"`
	MutedChannelsOnly    bool                  `json:"muted_channels_only" db:"muted_channels_only"`
	Timezone             string                `json:"timezone" db:"timezone"`
	CreatedAt            time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at" db:"updated_at"`
}

// DigestChannelPreference represents channel-specific digest settings
type DigestChannelPreference struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	ChannelID  uuid.UUID  `json:"channel_id" db:"channel_id"`
	DigestMode DigestMode `json:"digest_mode" db:"digest_mode"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// DigestServerPreference represents server-specific digest settings
type DigestServerPreference struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID   uuid.UUID  `json:"server_id" db:"server_id"`
	DigestMode DigestMode `json:"digest_mode" db:"digest_mode"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// DigestQueueItem represents a message queued for digest
type DigestQueueItem struct {
	ID                uuid.UUID        `json:"id" db:"id"`
	UserID            uuid.UUID        `json:"user_id" db:"user_id"`
	ServerID          *uuid.UUID       `json:"server_id,omitempty" db:"server_id"`
	ChannelID         *uuid.UUID       `json:"channel_id,omitempty" db:"channel_id"`
	MessageID         *uuid.UUID       `json:"message_id,omitempty" db:"message_id"`
	MessageContent    string           `json:"message_content" db:"message_content"`
	MessageAuthorID   *uuid.UUID       `json:"message_author_id,omitempty" db:"message_author_id"`
	MessageAuthorName string           `json:"message_author_name" db:"message_author_name"`
	MessageCreatedAt  time.Time        `json:"message_created_at" db:"message_created_at"`
	IsMention         bool             `json:"is_mention" db:"is_mention"`
	NotificationType  NotificationType `json:"notification_type" db:"notification_type"`
	QueuedAt          time.Time        `json:"queued_at" db:"queued_at"`
	DigestPeriod      time.Time        `json:"digest_period" db:"digest_period"`
}

// DigestHistory represents a sent digest
type DigestHistory struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           uuid.UUID       `json:"user_id" db:"user_id"`
	SentAt           time.Time       `json:"sent_at" db:"sent_at"`
	PeriodStart      time.Time       `json:"period_start" db:"period_start"`
	PeriodEnd        time.Time       `json:"period_end" db:"period_end"`
	Frequency        DigestFrequency `json:"frequency" db:"frequency"`
	TotalMessages    int             `json:"total_messages" db:"total_messages"`
	TotalMentions    int             `json:"total_mentions" db:"total_mentions"`
	ServersIncluded  int             `json:"servers_included" db:"servers_included"`
	ChannelsIncluded int             `json:"channels_included" db:"channels_included"`
	ContentJSON      string          `json:"content_json" db:"content_json"`
	Status           DigestStatus    `json:"status" db:"status"`
	ErrorMessage     *string         `json:"error_message,omitempty" db:"error_message"`
	RetryCount       int             `json:"retry_count" db:"retry_count"`
}

// DigestContent represents the structured content of a digest
type DigestContent struct {
	Period     DigestPeriodInfo       `json:"period"`
	Servers    []DigestServerSummary  `json:"servers"`
	DMChannels []DigestChannelSummary `json:"dm_channels,omitempty"`
	TotalStats DigestStats            `json:"total_stats"`
}

// DigestPeriodInfo describes the time period covered by a digest
type DigestPeriodInfo struct {
	Start     time.Time       `json:"start"`
	End       time.Time       `json:"end"`
	Frequency DigestFrequency `json:"frequency"`
}

// DigestServerSummary summarizes activity in a server
type DigestServerSummary struct {
	ServerID   uuid.UUID              `json:"server_id"`
	ServerName string                 `json:"server_name"`
	ServerIcon *string                `json:"server_icon,omitempty"`
	Channels   []DigestChannelSummary `json:"channels"`
	Stats      DigestStats            `json:"stats"`
}

// DigestChannelSummary summarizes activity in a channel
type DigestChannelSummary struct {
	ChannelID   uuid.UUID              `json:"channel_id"`
	ChannelName string                 `json:"channel_name"`
	Messages    []DigestMessageSummary `json:"messages"`
	Stats       DigestStats            `json:"stats"`
}

// DigestMessageSummary is a condensed message for digest
type DigestMessageSummary struct {
	MessageID  *uuid.UUID `json:"message_id,omitempty"`
	AuthorID   *uuid.UUID `json:"author_id,omitempty"`
	AuthorName string     `json:"author_name"`
	Content    string     `json:"content"`
	IsMention  bool       `json:"is_mention"`
	CreatedAt  time.Time  `json:"created_at"`
}

// DigestStats contains aggregate statistics
type DigestStats struct {
	MessageCount int `json:"message_count"`
	MentionCount int `json:"mention_count"`
}

// Request/Response types

// UpdateDigestPreferencesRequest represents a request to update digest preferences
type UpdateDigestPreferencesRequest struct {
	Enabled              *bool                  `json:"enabled,omitempty"`
	Frequency            *DigestFrequency       `json:"frequency,omitempty"`
	PreferredHour        *int                   `json:"preferred_hour,omitempty"`
	PreferredDay         *int                   `json:"preferred_day,omitempty"`
	AggregationMode      *DigestAggregationMode `json:"aggregation_mode,omitempty"`
	MaxMessagesPerSource *int                   `json:"max_messages_per_source,omitempty"`
	MutedChannelsOnly    *bool                  `json:"muted_channels_only,omitempty"`
	Timezone             *string                `json:"timezone,omitempty"`
}

// UpdateDigestChannelPreferenceRequest represents a request to update channel digest settings
type UpdateDigestChannelPreferenceRequest struct {
	DigestMode DigestMode `json:"digest_mode" validate:"required"`
}

// UpdateDigestServerPreferenceRequest represents a request to update server digest settings
type UpdateDigestServerPreferenceRequest struct {
	DigestMode DigestMode `json:"digest_mode" validate:"required"`
}

// DigestPreview represents a preview of what the next digest will contain
type DigestPreview struct {
	NextDigestAt    time.Time  `json:"next_digest_at"`
	PendingCount    int        `json:"pending_count"`
	PendingMentions int        `json:"pending_mentions"`
	PendingServers  int        `json:"pending_servers"`
	PendingChannels int        `json:"pending_channels"`
	OldestPending   *time.Time `json:"oldest_pending,omitempty"`
}

// DigestHistoryListOptions represents options for listing digest history
type DigestHistoryListOptions struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Helper methods

// ToJSON serializes DigestContent to JSON string
func (c *DigestContent) ToJSON() (string, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ParseDigestContent parses JSON string to DigestContent
func ParseDigestContent(jsonStr string) (*DigestContent, error) {
	var content DigestContent
	err := json.Unmarshal([]byte(jsonStr), &content)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// DefaultDigestPreferences returns default preferences for a new user
func DefaultDigestPreferences(userID uuid.UUID) *DigestPreferences {
	return &DigestPreferences{
		ID:                   uuid.New(),
		UserID:               userID,
		Enabled:              false,
		Frequency:            DigestFrequencyDaily,
		PreferredHour:        9,
		PreferredDay:         1,
		AggregationMode:      DigestAggregationServer,
		MaxMessagesPerSource: 50,
		MutedChannelsOnly:    true,
		Timezone:             "UTC",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

// ValidateFrequency checks if a frequency value is valid
func ValidateFrequency(f DigestFrequency) bool {
	switch f {
	case DigestFrequencyHourly, DigestFrequencyDaily, DigestFrequencyWeekly:
		return true
	}
	return false
}

// ValidateAggregationMode checks if an aggregation mode is valid
func ValidateAggregationMode(m DigestAggregationMode) bool {
	switch m {
	case DigestAggregationChannel, DigestAggregationServer:
		return true
	}
	return false
}

// ValidateDigestMode checks if a digest mode is valid
func ValidateDigestMode(m DigestMode) bool {
	switch m {
	case DigestModeInherit, DigestModeInclude, DigestModeExclude, DigestModeImmediate:
		return true
	}
	return false
}
