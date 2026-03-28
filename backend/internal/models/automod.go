package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AutoModEventType represents what triggers a rule check
type AutoModEventType int

const (
	EventTypeMessageSend AutoModEventType = 1
	EventTypeMemberJoin  AutoModEventType = 2
)

// AutoModTriggerType represents how a rule is triggered
type AutoModTriggerType int

const (
	TriggerKeyword      AutoModTriggerType = 1
	TriggerSpam         AutoModTriggerType = 2
	TriggerMentionSpam  AutoModTriggerType = 3
	TriggerMLClassified AutoModTriggerType = 4
	TriggerCustomRegex  AutoModTriggerType = 5
)

// ActionType represents what happens when a rule matches
type ActionType int

const (
	ActionBlockMessage     ActionType = 1
	ActionFlagToModerators ActionType = 2
	ActionTimeoutMember    ActionType = 3
	ActionSendAlert        ActionType = 4
	ActionLogToChannel     ActionType = 5
)

// AutoModTrigger contains the conditions that trigger a rule
type AutoModTrigger struct {
	Keywords      []string `json:"keywords,omitempty"`
	RegexPatterns []string `json:"regex_patterns,omitempty"`
	MentionLimit  int      `json:"mention_limit,omitempty"`
	MLCategories  []string `json:"ml_categories,omitempty"`
	Whitelist     []string `json:"whitelist,omitempty"`
}

// Value implements driver.Valuer for database serialization
func (t AutoModTrigger) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Scan implements sql.Scanner for database deserialization
func (t *AutoModTrigger) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

// AutoModActionMetadata contains additional data for an action
type AutoModActionMetadata struct {
	ChannelID     *uuid.UUID `json:"channel_id,omitempty"`
	AlertMessage  *string    `json:"alert_message,omitempty"`
	CustomMessage *string    `json:"custom_message,omitempty"`
}

// Value implements driver.Valuer
func (m AutoModActionMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner
func (m *AutoModActionMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// AutoModAction represents an action to take when a rule matches
type AutoModAction struct {
	Type     ActionType            `json:"type"`
	Duration *time.Duration        `json:"duration,omitempty"`
	Reason   *string               `json:"reason,omitempty"`
	Metadata AutoModActionMetadata `json:"metadata,omitempty"`
}

// AutoModActions is a slice of AutoModAction for JSON handling
type AutoModActions []AutoModAction

// Value implements driver.Valuer
func (a AutoModActions) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner
func (a *AutoModActions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, a)
}

// ModerationRule represents an auto-moderation rule
type ModerationRule struct {
	ID          uuid.UUID          `json:"id" db:"id"`
	ServerID    uuid.UUID          `json:"server_id" db:"server_id"`
	Name        string             `json:"name" db:"name"`
	EventType   AutoModEventType   `json:"event_type" db:"event_type"`
	TriggerType AutoModTriggerType `json:"trigger_type" db:"trigger_type"`
	Trigger     AutoModTrigger     `json:"trigger" db:"trigger"`
	Actions     AutoModActions     `json:"actions" db:"actions"`
	Enabled     bool               `json:"enabled" db:"enabled"`
	CreatedBy   uuid.UUID          `json:"created_by" db:"created_by"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`
}

// CreateAutoModRuleRequest is the input for creating a rule
type CreateAutoModRuleRequest struct {
	Name        string             `json:"name" validate:"required,min=1,max=100"`
	EventType   AutoModEventType   `json:"event_type" validate:"required"`
	TriggerType AutoModTriggerType `json:"trigger_type" validate:"required"`
	Trigger     AutoModTrigger     `json:"trigger" validate:"required"`
	Actions     AutoModActions     `json:"actions" validate:"required,min=1"`
	Enabled     *bool              `json:"enabled,omitempty"`
}

// UpdateAutoModRuleRequest is the input for updating a rule
type UpdateAutoModRuleRequest struct {
	Name        *string             `json:"name,omitempty"`
	EventType   *AutoModEventType   `json:"event_type,omitempty"`
	TriggerType *AutoModTriggerType `json:"trigger_type,omitempty"`
	Trigger     *AutoModTrigger     `json:"trigger,omitempty"`
	Actions     *AutoModActions     `json:"actions,omitempty"`
	Enabled     *bool               `json:"enabled,omitempty"`
}

// AutoModAlert represents a triggered rule instance
type AutoModAlert struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	RuleID         uuid.UUID  `json:"rule_id" db:"rule_id"`
	ServerID       uuid.UUID  `json:"server_id" db:"server_id"`
	MemberID       uuid.UUID  `json:"member_id" db:"member_id"`
	ChannelID      *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"`
	MessageID      *uuid.UUID `json:"message_id,omitempty" db:"message_id"`
	Content        string     `json:"content" db:"content"`
	ActionTaken    ActionType `json:"action_taken" db:"action_taken"`
	Resolved       bool       `json:"resolved" db:"resolved"`
	ResolvedBy     *uuid.UUID `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	MatchedKeyword *string    `json:"matched_keyword,omitempty" db:"matched_keyword"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	Rule   *ModerationRule `json:"rule,omitempty"`
	Member *Member         `json:"member,omitempty"`
}

// AutoModAlertSummary is a lightweight alert for listing
type AutoModAlertSummary struct {
	ID          uuid.UUID  `json:"id"`
	RuleID      uuid.UUID  `json:"rule_id"`
	RuleName    string     `json:"rule_name"`
	ServerID    uuid.UUID  `json:"server_id"`
	MemberID    uuid.UUID  `json:"member_id"`
	MemberName  string     `json:"member_name"`
	ChannelID   *uuid.UUID `json:"channel_id,omitempty"`
	MessageID   *uuid.UUID `json:"message_id,omitempty"`
	Content     string     `json:"content"`
	ActionTaken ActionType `json:"action_taken"`
	Resolved    bool       `json:"resolved"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ResolveAlertRequest is the input for resolving an alert
type ResolveAlertRequest struct {
	Resolved bool `json:"resolved"`
}

// AutoModTestRequest is the input for testing content against rules
type AutoModTestRequest struct {
	ServerID  uuid.UUID  `json:"server_id" validate:"required"`
	Content   string     `json:"content" validate:"required"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	MemberID  *uuid.UUID `json:"member_id,omitempty"`
}

// AutoModTestResult represents the result of testing content
type AutoModTestResult struct {
	Matched  bool            `json:"matched"`
	RuleID   *uuid.UUID      `json:"rule_id,omitempty"`
	RuleName *string         `json:"rule_name,omitempty"`
	Actions  []AutoModAction `json:"actions,omitempty"`
	Keyword  *string         `json:"keyword,omitempty"`
	Pattern  *string         `json:"pattern,omitempty"`
	AlertID  *uuid.UUID      `json:"alert_id,omitempty"`
}

// AutoModRuleTriggerCount tracks trigger usage for analytics
type AutoModRuleTriggerCount struct {
	RuleID       uuid.UUID `db:"rule_id"`
	TriggerCount int       `db:"trigger_count"`
	BlockCount   int       `db:"block_count"`
	FlagCount    int       `db:"flag_count"`
}
