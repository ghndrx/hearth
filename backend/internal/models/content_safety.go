package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NSFWDetectionThreshold represents sensitivity levels for NSFW detection
type NSFWDetectionThreshold int

const (
	NSFWThresholdNone      NSFWDetectionThreshold = 0 // No filtering
	NSFWThresholdLow      NSFWDetectionThreshold = 1 // Only explicit content
	NSFWThresholdMedium   NSFWDetectionThreshold = 2 // Moderate filtering
	NSFWThresholdHigh     NSFWDetectionThreshold = 3 // All questionable content
)

// ContentFilterType represents types of content filtering
type ContentFilterType int

const (
	FilterTypeNSFW            ContentFilterType = 1
	FilterTypeViolence        ContentFilterType = 2
	FilterTypeHateSpeech      ContentFilterType = 3
	FilterTypeHarassment      ContentFilterType = 4
	FilterTypeSpam            ContentFilterType = 5
	FilterTypeCustomKeyword   ContentFilterType = 6
)

// ContentFilter represents a server or channel content filter configuration
type ContentFilter struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	ServerID    uuid.UUID              `json:"server_id" db:"server_id"`
	ChannelID   *uuid.UUID             `json:"channel_id,omitempty" db:"channel_id"` // nil means server-wide
	Type        ContentFilterType      `json:"type" db:"type"`
	Name        string                 `json:"name" db:"name"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Threshold   NSFWDetectionThreshold `json:"threshold" db:"threshold"`
	Action      ContentFilterAction    `json:"action" db:"action"`
	FilterData  ContentFilterData      `json:"filter_data" db:"filter_data"`
	ExemptRoles []uuid.UUID            `json:"exempt_roles,omitempty" db:"exempt_roles"`
	CreatedBy   uuid.UUID             `json:"created_by" db:"created_by"`
	CreatedAt   time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at" db:"updated_at"`
}

// ContentFilterAction represents what happens when content is flagged
type ContentFilterAction int

const (
	FilterActionAllow     ContentFilterAction = 0 // Log only
	FilterActionWarn      ContentFilterAction = 1 // Warn user
	FilterActionBlock     ContentFilterAction = 2 // Block message
	FilterActionDelete    ContentFilterAction = 3 // Delete and warn
	FilterActionTimeout   ContentFilterAction = 4 // Timeout user
	FilterActionKick      ContentFilterAction = 5 // Kick user
	FilterActionBan       ContentFilterAction = 6 // Ban user
)

// ContentFilterData contains filter-specific configuration
type ContentFilterData struct {
	Keywords        []string `json:"keywords,omitempty"`
	RegexPatterns   []string `json:"regex_patterns,omitempty"`
	Whitelist       []string `json:"whitelist,omitempty"`
	ThresholdValue  float64  `json:"threshold_value,omitempty"`  // For ML-based detection
	AlertChannelID  *string  `json:"alert_channel_id,omitempty"` // Where to send alerts
}

// Value implements driver.Valuer for database serialization
func (f ContentFilterData) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements sql.Scanner for database deserialization
func (f *ContentFilterData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, f)
}

// AgeVerificationSetting represents age verification requirements for a server/channel
type AgeVerificationSetting struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ServerID        uuid.UUID  `json:"server_id" db:"server_id"`
	ChannelID       *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"` // nil means server-wide
	Enabled         bool       `json:"enabled" db:"enabled"`
	RequiredAge     int        `json:"required_age" db:"required_age"`         // Minimum age required (e.g., 18)
	VerificationType string    `json:"verification_type" db:"verification_type"` // "manual", "automatic", "id_verification"
	CreatedBy       uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// UserContentPreference represents user-level content filtering preferences
type UserContentPreference struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	UserID          uuid.UUID   `json:"user_id" db:"user_id"`
	NSFWFilterLevel NSFWDetectionThreshold `json:"nsfw_filter_level" db:"nsfw_filter_level"`
	HideNSFWContent bool        `json:"hide_nsfw_content" db:"hide_nsfw_content"`
	HideExplicitContent bool    `json:"hide_explicit_content" db:"hide_explicit_content"`
	AutoCollapseNSFW bool       `json:"auto_collapse_nsfw" db:"auto_collapse_nsfw"`
	AllowAgeVerifiedChannels bool `json:"allow_age_verified_channels" db:"allow_age_verified_channels"`
	TrustedServers   []uuid.UUID `json:"trusted_servers,omitempty" db:"trusted_servers"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

// CreateContentFilterRequest is the input for creating a content filter
type CreateContentFilterRequest struct {
	ChannelID  *string                `json:"channel_id,omitempty"`
	Type       ContentFilterType      `json:"type" validate:"required"`
	Name       string                 `json:"name" validate:"required,min=1,max=100"`
	Enabled    *bool                   `json:"enabled,omitempty"`
	Threshold  NSFWDetectionThreshold `json:"threshold,omitempty"`
	Action     ContentFilterAction    `json:"action" validate:"required"`
	FilterData ContentFilterData      `json:"filter_data,omitempty"`
	ExemptRoles []string              `json:"exempt_roles,omitempty"`
}

// UpdateContentFilterRequest is the input for updating a content filter
type UpdateContentFilterRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	Threshold   *NSFWDetectionThreshold `json:"threshold,omitempty"`
	Action      *ContentFilterAction    `json:"action,omitempty"`
	FilterData  *ContentFilterData      `json:"filter_data,omitempty"`
	ExemptRoles *[]string               `json:"exempt_roles,omitempty"`
}

// CreateAgeVerificationRequest is the input for creating age verification settings
type CreateAgeVerificationRequest struct {
	ChannelID        *string `json:"channel_id,omitempty"`
	Enabled          bool    `json:"enabled"`
	RequiredAge      int     `json:"required_age" validate:"required,min=13,max=100"`
	VerificationType string  `json:"verification_type" validate:"required,oneof=manual automatic id_verification"`
}

// UpdateAgeVerificationRequest is the input for updating age verification
type UpdateAgeVerificationRequest struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	RequiredAge      *int    `json:"required_age,omitempty"`
	VerificationType *string `json:"verification_type,omitempty"`
}

// UpdateUserContentPreferenceRequest is the input for updating user content preferences
type UpdateUserContentPreferenceRequest struct {
	NSFWFilterLevel        *NSFWDetectionThreshold `json:"nsfw_filter_level,omitempty"`
	HideNSFWContent        *bool                  `json:"hide_nsfw_content,omitempty"`
	HideExplicitContent    *bool                  `json:"hide_explicit_content,omitempty"`
	AutoCollapseNSFW       *bool                  `json:"auto_collapse_nsfw,omitempty"`
	AllowAgeVerifiedChannels *bool                `json:"allow_age_verified_channels,omitempty"`
	TrustedServers         *[]string              `json:"trusted_servers,omitempty"`
}

// ContentScanResult represents the result of scanning content
type ContentScanResult struct {
	Passed       bool                  `json:"passed"`
	Flags        []ContentFlag         `json:"flags,omitempty"`
	ActionTaken ContentFilterAction    `json:"action_taken,omitempty"`
	FilterName  string                `json:"filter_name,omitempty"`
	MatchedRule *uuid.UUID            `json:"matched_rule,omitempty"`
	Message     string                `json:"message,omitempty"`
}

// ContentFlag represents a detected content flag
type ContentFlag struct {
	Type     ContentFilterType `json:"type"`
	Severity int               `json:"severity"` // 1-10
	Detail   string            `json:"detail,omitempty"`
	Keyword  *string           `json:"keyword,omitempty"`
}

// ContentFilterSummary is a lightweight representation for listing
type ContentFilterSummary struct {
	ID        uuid.UUID              `json:"id"`
	ServerID  uuid.UUID              `json:"server_id"`
	ChannelID *uuid.UUID             `json:"channel_id,omitempty"`
	Type      ContentFilterType      `json:"type"`
	Name      string                 `json:"name"`
	Enabled   bool                   `json:"enabled"`
	Threshold NSFWDetectionThreshold `json:"threshold"`
}

// ContentSafetySettings is a comprehensive view of all safety settings for a server
type ContentSafetySettings struct {
	ServerID               uuid.UUID                   `json:"server_id"`
	Filters                []*ContentFilter            `json:"filters"`
	AgeVerification        *AgeVerificationSetting     `json:"age_verification,omitempty"`
	ServerDefaultThreshold NSFWDetectionThreshold     `json:"server_default_threshold"`
}
