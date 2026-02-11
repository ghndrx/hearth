package models

import (
	"time"

	"github.com/google/uuid"
)

// ChannelType represents the type of channel
type ChannelType string

const (
	ChannelTypeText         ChannelType = "text"
	ChannelTypeVoice        ChannelType = "voice"
	ChannelTypeCategory     ChannelType = "category"
	ChannelTypeAnnouncement ChannelType = "announcement"
	ChannelTypeForum        ChannelType = "forum"
	ChannelTypeStage        ChannelType = "stage"
	ChannelTypeDM           ChannelType = "dm"
	ChannelTypeGroupDM      ChannelType = "group_dm"
)

// Channel represents a communication channel
type Channel struct {
	ID                 uuid.UUID   `json:"id" db:"id"`
	ServerID           *uuid.UUID  `json:"server_id,omitempty" db:"server_id"`
	CategoryID         *uuid.UUID  `json:"category_id,omitempty" db:"category_id"`
	Type               ChannelType `json:"type" db:"type"`
	Name               *string     `json:"name,omitempty" db:"name"`
	Topic              *string     `json:"topic,omitempty" db:"topic"`
	Position           int         `json:"position" db:"position"`
	NSFW               bool        `json:"nsfw" db:"nsfw"`
	SlowmodeSeconds    int         `json:"slowmode_seconds" db:"slowmode_seconds"`
	Bitrate            int         `json:"bitrate" db:"bitrate"`
	UserLimit          int         `json:"user_limit" db:"user_limit"`
	RTCRegion          *string     `json:"rtc_region,omitempty" db:"rtc_region"`
	DefaultAutoArchive int         `json:"default_auto_archive" db:"default_auto_archive"`
	LastMessageID      *uuid.UUID  `json:"last_message_id,omitempty" db:"last_message_id"`
	CreatedAt          time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at" db:"updated_at"`

	// Populated from joins
	PermissionOverrides []PermissionOverride `json:"permission_overrides,omitempty"`
	Recipients          []PublicUser         `json:"recipients,omitempty"` // For DMs
}

// AutoArchiveDuration constants (in minutes)
const (
	AutoArchive1Hour  = 60
	AutoArchive24Hour = 1440
	AutoArchive3Day   = 4320
	AutoArchive1Week  = 10080
)

// CreateChannelRequest is the input for creating a channel
type CreateChannelRequest struct {
	Name       string      `json:"name" validate:"required,min=1,max=100"`
	Type       ChannelType `json:"type" validate:"required"`
	Topic      *string     `json:"topic,omitempty" validate:"omitempty,max=1024"`
	CategoryID *string     `json:"category_id,omitempty"`
	Position   *int        `json:"position,omitempty"`
	NSFW       *bool       `json:"nsfw,omitempty"`
	Bitrate    *int        `json:"bitrate,omitempty" validate:"omitempty,min=8000,max=384000"`
	UserLimit  *int        `json:"user_limit,omitempty" validate:"omitempty,min=0,max=99"`
}

// UpdateChannelRequest is the input for updating a channel
type UpdateChannelRequest struct {
	Name            *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Topic           *string `json:"topic,omitempty" validate:"omitempty,max=1024"`
	Position        *int    `json:"position,omitempty"`
	CategoryID      *string `json:"category_id,omitempty"`
	NSFW            *bool   `json:"nsfw,omitempty"`
	SlowmodeSeconds *int    `json:"slowmode_seconds,omitempty" validate:"omitempty,min=0,max=21600"`
	Bitrate         *int    `json:"bitrate,omitempty" validate:"omitempty,min=8000,max=384000"`
	UserLimit       *int    `json:"user_limit,omitempty" validate:"omitempty,min=0,max=99"`
}

// PermissionOverride represents channel-specific permission overrides
type PermissionOverride struct {
	ChannelID  uuid.UUID `json:"channel_id" db:"channel_id"`
	TargetType string    `json:"target_type" db:"target_type"` // "role" or "user"
	TargetID   uuid.UUID `json:"target_id" db:"target_id"`
	Allow      int64     `json:"allow" db:"allow"`
	Deny       int64     `json:"deny" db:"deny"`
}

// Thread represents a message thread
type Thread struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	ParentChannelID  uuid.UUID  `json:"parent_channel_id" db:"parent_channel_id"`
	OwnerID          uuid.UUID  `json:"owner_id" db:"owner_id"`
	Name             string     `json:"name" db:"name"`
	MessageCount     int        `json:"message_count" db:"message_count"`
	MemberCount      int        `json:"member_count" db:"member_count"`
	Archived         bool       `json:"archived" db:"archived"`
	AutoArchive      int        `json:"auto_archive" db:"auto_archive"` // minutes
	Locked           bool       `json:"locked" db:"locked"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	ArchiveTimestamp *time.Time `json:"archive_timestamp,omitempty" db:"archive_timestamp"`

	// Populated from joins
	ParentChannel *Channel `json:"parent_channel,omitempty"`
}

// CreateThreadRequest is the input for creating a thread
type CreateThreadRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	AutoArchive *int   `json:"auto_archive,omitempty"` // 60, 1440, 4320, 10080
}

// DMChannelRecipient links users to DM channels
type DMChannelRecipient struct {
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
}

// CreateDMRequest is the input for creating/getting a DM channel
type CreateDMRequest struct {
	RecipientID string `json:"recipient_id" validate:"required"`
}

// CreateGroupDMRequest is the input for creating a group DM
type CreateGroupDMRequest struct {
	RecipientIDs []string `json:"recipient_ids" validate:"required,min=1,max=9"`
	Name         *string  `json:"name,omitempty" validate:"omitempty,max=100"`
}
