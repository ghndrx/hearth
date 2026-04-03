package models

import (
	"time"

	"github.com/google/uuid"
)

// ThreadAutoArchiveSettings represents server-level auto-archive configuration
type ThreadAutoArchiveSettings struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	ServerID               uuid.UUID `json:"server_id" db:"server_id"`
	DefaultDuration        int       `json:"default_duration" db:"default_duration"` // minutes
	AllowOverride          bool      `json:"allow_override" db:"allow_override"`
	ArchiveDurationOptions []int     `json:"archive_duration_options" db:"archive_duration_options"` // available options in minutes
	RequirePostAuthor      bool      `json:"require_post_author" db:"require_post_author"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// ChannelAutoArchiveOverride represents channel-level auto-archive override
type ChannelAutoArchiveOverride struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	ChannelID           uuid.UUID `json:"channel_id" db:"channel_id"`
	AutoArchiveDuration int       `json:"auto_archive_duration" db:"auto_archive_duration"` // minutes
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`

	// Populated from joins
	Channel *Channel `json:"channel,omitempty"`
}

// ThreadAutoArchiveMeta tracks auto-archive state for a thread
type ThreadAutoArchiveMeta struct {
	ThreadID              uuid.UUID  `json:"thread_id" db:"thread_id"`
	LastActivityAt        time.Time  `json:"last_activity_at" db:"last_activity_at"`
	LastActivityMessageID *uuid.UUID `json:"last_activity_message_id,omitempty" db:"last_activity_message_id"`
	LastActivityUserID    *uuid.UUID `json:"last_activity_user_id,omitempty" db:"last_activity_user_id"`
	NextArchiveAt         *time.Time `json:"next_archive_at,omitempty" db:"next_archive_at"`
	ArchiveEligible       bool       `json:"archive_eligible" db:"archive_eligible"`
	BumpedByOwner         bool       `json:"bumped_by_owner" db:"bumped_by_owner"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`

	// Populated from joins
	Thread *Thread `json:"thread,omitempty"`
}

// CreateThreadAutoArchiveSettingsRequest is the request body for creating server auto-archive settings
type CreateThreadAutoArchiveSettingsRequest struct {
	DefaultDuration        int   `json:"default_duration" validate:"required,oneof=60 1440 4320 10080"`
	AllowOverride          *bool `json:"allow_override,omitempty"`
	ArchiveDurationOptions []int `json:"archive_duration_options,omitempty" validate:"dive,oneof=60 1440 4320 10080"`
	RequirePostAuthor      *bool `json:"require_post_author,omitempty"`
}

// UpdateThreadAutoArchiveSettingsRequest is the request body for updating server auto-archive settings
type UpdateThreadAutoArchiveSettingsRequest struct {
	DefaultDuration        *int   `json:"default_duration,omitempty" validate:"omitempty,oneof=60 1440 4320 10080"`
	AllowOverride          *bool  `json:"allow_override,omitempty"`
	ArchiveDurationOptions *[]int `json:"archive_duration_options,omitempty" validate:"dive,oneof=60 1440 4320 10080"`
	RequirePostAuthor      *bool  `json:"require_post_author,omitempty"`
}

// SetChannelAutoArchiveOverrideRequest is the request body for setting channel override
type SetChannelAutoArchiveOverrideRequest struct {
	AutoArchiveDuration int `json:"auto_archive_duration" validate:"required,oneof=60 1440 4320 10080"`
}

// ThreadAutoArchiveResponse represents the response for auto-archive queries
type ThreadAutoArchiveResponse struct {
	ThreadID      uuid.UUID  `json:"thread_id"`
	NextArchiveAt *time.Time `json:"next_archive_at,omitempty"`
	Eligible      bool       `json:"archive_eligible"`
	Status        string     `json:"status"` // "active", "scheduled", "ready", "archived"
}

// ThreadAutoArchiveStats represents statistics about auto-archive for a server
type ThreadAutoArchiveStats struct {
	ServerID              uuid.UUID `json:"server_id"`
	TotalThreads          int       `json:"total_threads"`
	ArchivedThreads       int       `json:"archived_threads"`
	ScheduledThreads      int       `json:"scheduled_threads"`
	ReadyToArchiveThreads int       `json:"ready_to_archive_threads"`
}
