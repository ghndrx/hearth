package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// WelcomeScreen represents a server's welcome/onboarding configuration
type WelcomeScreen struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	ServerID        uuid.UUID      `json:"server_id" db:"server_id"`
	Enabled         bool           `json:"enabled" db:"enabled"`
	Title           string         `json:"title" db:"title"`
	Description     string         `json:"description" db:"description"`
	WelcomeChannels pq.StringArray `json:"welcome_channels" db:"welcome_channels"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
}

// Rule represents a server rule in the welcome screen
type Rule struct {
	ID          string `json:"id"`
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ScreeningQuestion represents a membership screening question
type ScreeningQuestion struct {
	ID       string   `json:"id"`
	Order    int      `json:"order"`
	Question string   `json:"question"`
	Required bool     `json:"required"`
	Type     string   `json:"type"` // text, select, agree
	Options  []string `json:"options,omitempty"`
}

// WelcomeScreenConfig contains the full welcome screen configuration
type WelcomeScreenConfig struct {
	WelcomeScreen
	Rules     []Rule              `json:"rules"`
	Questions []ScreeningQuestion `json:"questions"`
}

// ScreeningAnswer represents a user's answer to a screening question
type ScreeningAnswer struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

// MemberScreening represents a member's onboarding/screening state
type MemberScreening struct {
	ID        uuid.UUID         `json:"id" db:"id"`
	UserID    uuid.UUID         `json:"user_id" db:"user_id"`
	ServerID  uuid.UUID         `json:"server_id" db:"server_id"`
	Answers   []ScreeningAnswer `json:"answers"`
	RulesRead bool              `json:"rules_read" db:"rules_read"`
	Status    string            `json:"status" db:"status"` // pending, approved, rejected
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

// Screening status constants
const (
	ScreeningStatusPending  = "pending"
	ScreeningStatusApproved = "approved"
	ScreeningStatusRejected = "rejected"
)

// UpdateWelcomeScreenRequest is the input for updating welcome screen
type UpdateWelcomeScreenRequest struct {
	Enabled         *bool               `json:"enabled,omitempty"`
	Title           *string             `json:"title,omitempty"`
	Description     *string             `json:"description,omitempty"`
	WelcomeChannels []string            `json:"welcome_channels,omitempty"`
	Rules           []Rule              `json:"rules,omitempty"`
	Questions       []ScreeningQuestion `json:"questions,omitempty"`
}

// SubmitScreeningRequest is the input for submitting screening answers
type SubmitScreeningRequest struct {
	Answers   []ScreeningAnswer `json:"answers"`
	RulesRead bool              `json:"rules_read"`
}

// ScreeningDecisionRequest is the input for approving/rejecting screening
type ScreeningDecisionRequest struct {
	Reason string `json:"reason,omitempty"`
}
