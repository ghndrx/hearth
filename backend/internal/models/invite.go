package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry represents an action in the audit log
type AuditLogEntry struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ServerID   uuid.UUID  `json:"server_id" db:"server_id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	TargetID   *uuid.UUID `json:"target_id,omitempty" db:"target_id"`
	ActionType string     `json:"action_type" db:"action_type"`
	Changes    []Change   `json:"changes,omitempty"`
	Reason     string     `json:"reason,omitempty" db:"reason"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`

	// Populated on fetch
	User   *User `json:"user,omitempty"`
	Target *User `json:"target,omitempty"`
}

// Change represents a single change in an audit log entry
type Change struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// Audit log action types
const (
	AuditLogServerUpdate      = "SERVER_UPDATE"
	AuditLogChannelCreate     = "CHANNEL_CREATE"
	AuditLogChannelUpdate     = "CHANNEL_UPDATE"
	AuditLogChannelDelete     = "CHANNEL_DELETE"
	AuditLogMemberKick        = "MEMBER_KICK"
	AuditLogMemberBan         = "MEMBER_BAN"
	AuditLogMemberUnban       = "MEMBER_UNBAN"
	AuditLogMemberUpdate      = "MEMBER_UPDATE"
	AuditLogRoleCreate        = "ROLE_CREATE"
	AuditLogRoleUpdate        = "ROLE_UPDATE"
	AuditLogRoleDelete        = "ROLE_DELETE"
	AuditLogInviteCreate      = "INVITE_CREATE"
	AuditLogInviteDelete      = "INVITE_DELETE"
	AuditLogWebhookCreate     = "WEBHOOK_CREATE"
	AuditLogWebhookUpdate     = "WEBHOOK_UPDATE"
	AuditLogWebhookDelete     = "WEBHOOK_DELETE"
	AuditLogEmojiCreate       = "EMOJI_CREATE"
	AuditLogEmojiUpdate       = "EMOJI_UPDATE"
	AuditLogEmojiDelete       = "EMOJI_DELETE"
	AuditLogMessageDelete     = "MESSAGE_DELETE"
	AuditLogMessageBulkDelete = "MESSAGE_BULK_DELETE"
	AuditLogMessagePin        = "MESSAGE_PIN"
	AuditLogMessageUnpin      = "MESSAGE_UNPIN"
)
