package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModerationSensitivityLevel represents the sensitivity level for moderation
type ModerationSensitivityLevel int

const (
	SensitivityLow    ModerationSensitivityLevel = 1
	SensitivityMedium ModerationSensitivityLevel = 2
	SensitivityHigh   ModerationSensitivityLevel = 3
)

// ModerationActionType represents automatic moderation actions
type ModerationActionType int

const (
	ModActionWarn         ModerationActionType = 1
	ModActionMute         ModerationActionType = 2
	ModActionDelete       ModerationActionType = 3
	ModActionTempBan      ModerationActionType = 4
	ModActionBlock        ModerationActionType = 5
	ModActionFlagForReview ModerationActionType = 6
)

// ToxicityCategory represents categories of toxic content
type ToxicityCategory int

const (
	ToxicitySpam            ToxicityCategory = 1
	ToxicityProfanity       ToxicityCategory = 2
	ToxicityHarassment      ToxicityCategory = 3
	ToxicityThreat          ToxicityCategory = 4
	ToxicitySexual          ToxicityCategory = 5
	ToxicityHateSpeech      ToxicityCategory = 6
	ToxicitySelfHarm        ToxicityCategory = 7
	ToxicityMisinformation  ToxicityCategory = 8
	ToxicityPersonalInfo    ToxicityCategory = 9
)

// ToxicityScore represents the scored result of content analysis
type ToxicityScore struct {
	Overall      float64                  `json:"overall"`
	Categories   map[ToxicityCategory]float64 `json:"categories"`
	MatchedKeywords []string               `json:"matched_keywords,omitempty"`
	MatchedPatterns []string               `json:"matched_patterns,omitempty"`
	IsMLClassified bool                   `json:"is_ml_classified,omitempty"`
}

// Value implements driver.Valuer for database serialization
func (t ToxicityScore) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Scan implements sql.Scanner for database deserialization
func (t *ToxicityScore) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

// KeywordRule represents a configurable keyword/regex rule for moderation
type KeywordRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	Name        string    `json:"name" db:"name"`               // Rule name for display
	Pattern     string    `json:"pattern" db:"pattern"`         // Keyword or regex pattern
	IsRegex     bool     `json:"is_regex" db:"is_regex"`       // Whether pattern is regex
	Sensitivity int      `json:"sensitivity" db:"sensitivity"` // 1-3 scale
	Category    ToxicityCategory `json:"category" db:"category"` // Toxicity category
	Action      ModerationActionType `json:"action" db:"action"` // Default action
	Weight      float64   `json:"weight" db:"weight"`          // Weight for scoring (0.0-1.0)
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ModerationSettings represents server-level moderation configuration
type ModerationSettings struct {
	ID                       uuid.UUID                   `json:"id" db:"id"`
	ServerID                 uuid.UUID                   `json:"server_id" db:"server_id"`
	Enabled                  bool                        `json:"enabled" db:"enabled"`
	SensitivityLevel         ModerationSensitivityLevel  `json:"sensitivity_level" db:"sensitivity_level"`
	MLClassificationEnabled  bool                        `json:"ml_classification_enabled" db:"ml_classification_enabled"` // Placeholder for future ML
	AutoModerationEnabled    bool                        `json:"auto_moderation_enabled" db:"auto_moderation_enabled"`
	ViolationThreshold       int                         `json:"violation_threshold" db:"violation_threshold"` // Score threshold for auto-action
	WarningThreshold         int                         `json:"warning_threshold" db:"warning_threshold"`     // Violations before warn
	MuteThreshold            int                         `json:"mute_threshold" db:"mute_threshold"`           // Violations before mute
	TempBanThreshold         int                         `json:"temp_ban_threshold" db:"temp_ban_threshold"`   // Violations before temp ban
	TempBanDuration          time.Duration               `json:"temp_ban_duration" db:"temp_ban_duration"`
	MuteDuration             time.Duration               `json:"mute_duration" db:"mute_duration"`
	LogChannelID             *uuid.UUID                  `json:"log_channel_id,omitempty" db:"log_channel_id"`
	ExemptRoles              []uuid.UUID                 `json:"exempt_roles,omitempty" db:"exempt_roles"`
	ExemptChannels           []uuid.UUID                 `json:"exempt_channels,omitempty" db:"exempt_channels"`
	AuditRetentionDays       int                         `json:"audit_retention_days" db:"audit_retention_days"`
	CreatedAt                time.Time                   `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time                   `json:"updated_at" db:"updated_at"`
}

// Value implements driver.Valuer
func (m ModerationSettings) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner
func (m *ModerationSettings) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// ModerationLog represents a logged moderation action or violation
type ModerationLog struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	ServerID       uuid.UUID              `json:"server_id" db:"server_id"`
	MemberID       uuid.UUID              `json:"member_id" db:"member_id"`
	ModeratorID    *uuid.UUID             `json:"moderator_id,omitempty" db:"moderator_id"` // nil for auto-mod actions
	ActionType     ModerationActionType   `json:"action_type" db:"action_type"`
	ViolationScore *ToxicityScore         `json:"violation_score,omitempty" db:"violation_score"`
	Reason         string                 `json:"reason" db:"reason"`
	ChannelID      *uuid.UUID             `json:"channel_id,omitempty" db:"channel_id"`
	MessageID      *uuid.UUID             `json:"message_id,omitempty" db:"message_id"`
	RuleID        *uuid.UUID             `json:"rule_id,omitempty" db:"rule_id"`
	RuleName      *string                `json:"rule_name,omitempty" db:"rule_name"`
	Duration       *time.Duration         `json:"duration,omitempty" db:"duration"`
	Resolved       bool                   `json:"resolved" db:"resolved"`
	ResolvedBy     *uuid.UUID             `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt     *time.Time            `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// Value implements driver.Valuer
func (m ModerationLog) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner
func (m *ModerationLog) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// ModerationLogSummary is a lightweight log entry for listing
type ModerationLogSummary struct {
	ID           uuid.UUID             `json:"id"`
	ServerID     uuid.UUID             `json:"server_id"`
	MemberID     uuid.UUID             `json:"member_id"`
	MemberName   string                `json:"member_name"`
	ModeratorID  *uuid.UUID            `json:"moderator_id,omitempty"`
	ActionType   ModerationActionType  `json:"action_type"`
	Reason       string                `json:"reason"`
	ChannelID    *uuid.UUID            `json:"channel_id,omitempty"`
	MessageID    *uuid.UUID            `json:"message_id,omitempty"`
	Resolved     bool                  `json:"resolved"`
	CreatedAt    time.Time             `json:"created_at"`
}

// UserViolationSummary tracks a user's violations for threshold-based actions
type UserViolationSummary struct {
	UserID          uuid.UUID `json:"user_id"`
	ServerID        uuid.UUID `json:"server_id"`
	ViolationCount  int       `json:"violation_count"`
	LastViolationAt *time.Time `json:"last_violation_at,omitempty"`
	WarnCount       int       `json:"warn_count"`
	MuteCount       int       `json:"mute_count"`
	TempBanCount    int       `json:"temp_ban_count"`
	TotalScore      float64   `json:"total_score"`
}

// CreateModerationSettingsRequest is the input for creating moderation settings
type CreateModerationSettingsRequest struct {
	Enabled               *bool                        `json:"enabled,omitempty"`
	SensitivityLevel      *ModerationSensitivityLevel `json:"sensitivity_level,omitempty"`
	MLClassificationEnabled *bool                     `json:"ml_classification_enabled,omitempty"`
	AutoModerationEnabled *bool                       `json:"auto_moderation_enabled,omitempty"`
	ViolationThreshold    *int                        `json:"violation_threshold,omitempty"`
	WarningThreshold      *int                        `json:"warning_threshold,omitempty"`
	MuteThreshold         *int                        `json:"mute_threshold,omitempty"`
	TempBanThreshold      *int                        `json:"temp_ban_threshold,omitempty"`
	TempBanDuration       *time.Duration               `json:"temp_ban_duration,omitempty"`
	MuteDuration          *time.Duration               `json:"mute_duration,omitempty"`
	LogChannelID          *uuid.UUID                   `json:"log_channel_id,omitempty"`
	ExemptRoles           []uuid.UUID                  `json:"exempt_roles,omitempty"`
	ExemptChannels        []uuid.UUID                  `json:"exempt_channels,omitempty"`
	AuditRetentionDays    *int                        `json:"audit_retention_days,omitempty"`
}

// UpdateModerationSettingsRequest is the input for updating moderation settings
type UpdateModerationSettingsRequest struct {
	Enabled               *bool                        `json:"enabled,omitempty"`
	SensitivityLevel      *ModerationSensitivityLevel `json:"sensitivity_level,omitempty"`
	MLClassificationEnabled *bool                     `json:"ml_classification_enabled,omitempty"`
	AutoModerationEnabled *bool                       `json:"auto_moderation_enabled,omitempty"`
	ViolationThreshold    *int                        `json:"violation_threshold,omitempty"`
	WarningThreshold      *int                        `json:"warning_threshold,omitempty"`
	MuteThreshold         *int                        `json:"mute_threshold,omitempty"`
	TempBanThreshold      *int                        `json:"temp_ban_threshold,omitempty"`
	TempBanDuration       *time.Duration               `json:"temp_ban_duration,omitempty"`
	MuteDuration          *time.Duration               `json:"mute_duration,omitempty"`
	LogChannelID          *uuid.UUID                   `json:"log_channel_id,omitempty"`
	ExemptRoles           *[]uuid.UUID                 `json:"exempt_roles,omitempty"`
	ExemptChannels        *[]uuid.UUID                 `json:"exempt_channels,omitempty"`
	AuditRetentionDays    *int                        `json:"audit_retention_days,omitempty"`
}

// CreateKeywordRuleRequest is the input for creating a keyword rule
type CreateKeywordRuleRequest struct {
	Name        string                `json:"name" validate:"required,min=1,max=100"`
	Pattern     string                `json:"pattern" validate:"required"`
	IsRegex     bool                  `json:"is_regex"`
	Sensitivity int                   `json:"sensitivity" validate:"min=1,max=3"`
	Category    ToxicityCategory      `json:"category" validate:"required"`
	Action      ModerationActionType `json:"action" validate:"required"`
	Weight      *float64              `json:"weight,omitempty"`
	Enabled     *bool                 `json:"enabled,omitempty"`
}

// UpdateKeywordRuleRequest is the input for updating a keyword rule
type UpdateKeywordRuleRequest struct {
	Name        *string                `json:"name,omitempty"`
	Pattern     *string                `json:"pattern,omitempty"`
	IsRegex     *bool                  `json:"is_regex,omitempty"`
	Sensitivity *int                   `json:"sensitivity,omitempty"`
	Category    *ToxicityCategory      `json:"category,omitempty"`
	Action      *ModerationActionType `json:"action,omitempty"`
	Weight      *float64               `json:"weight,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
}

// ModerationActionRequest is the input for taking a moderation action
type ModerationActionRequest struct {
	MemberID   uuid.UUID              `json:"member_id" validate:"required"`
	Action     ModerationActionType   `json:"action" validate:"required"`
	Reason     string                 `json:"reason,omitempty"`
	Duration   *time.Duration         `json:"duration,omitempty"`
	ChannelID  *uuid.UUID             `json:"channel_id,omitempty"`
	MessageID  *uuid.UUID             `json:"message_id,omitempty"`
}

// ResolveLogRequest is the input for resolving a moderation log entry
type ResolveLogRequest struct {
	Resolved bool `json:"resolved"`
}

// ModerationDashboardStats represents statistics for the moderation dashboard
type ModerationDashboardStats struct {
	TotalViolations    int                              `json:"total_violations"`
	TotalWarnings      int                              `json:"total_warnings"`
	TotalMutes         int                              `json:"total_mutes"`
	TotalTempBans      int                              `json:"total_temp_bans"`
	TopCategories      map[ToxicityCategory]int         `json:"top_categories"`
	TopOffenders       []*UserViolationSummary          `json:"top_offenders"`
	RecentActions      []*ModerationLogSummary          `json:"recent_actions"`
	TrendData          []*DailyModerationCount          `json:"trend_data"`
}

// DailyModerationCount represents daily moderation action counts
type DailyModerationCount struct {
	Date         time.Time `json:"date"`
	WarningCount int       `json:"warning_count"`
	MuteCount    int       `json:"mute_count"`
	TempBanCount int       `json:"temp_ban_count"`
	DeleteCount  int       `json:"delete_count"`
	FlagCount    int       `json:"flag_count"`
}

// AnalyzeContentRequest is the input for content analysis
type AnalyzeContentRequest struct {
	ServerID  uuid.UUID  `json:"server_id" validate:"required"`
	Content  string     `json:"content" validate:"required"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	MemberID  *uuid.UUID `json:"member_id,omitempty"`
}

// AnalyzeContentResult is the result of content analysis
type AnalyzeContentResult struct {
	Violations  []ViolationDetail `json:"violations"`
	TotalScore  float64          `json:"total_score"`
	ShouldBlock bool             `json:"should_block"`
	Actions     []ModerationActionType `json:"actions"`
}

// ViolationDetail represents a single violation found in content
type ViolationDetail struct {
	RuleID       uuid.UUID           `json:"rule_id"`
	RuleName     string              `json:"rule_name"`
	Category     ToxicityCategory    `json:"category"`
	Score        float64             `json:"score"`
	MatchedText string              `json:"matched_text"`
	Action       ModerationActionType `json:"action"`
}
