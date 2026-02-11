package models

import (
	"time"

	"github.com/google/uuid"
)

// Invite represents a server invite
type Invite struct {
	Code      string     `json:"code" db:"code"`
	ServerID  uuid.UUID  `json:"server_id" db:"server_id"`
	ChannelID uuid.UUID  `json:"channel_id" db:"channel_id"`
	CreatorID uuid.UUID  `json:"creator_id" db:"creator_id"`
	MaxUses   int        `json:"max_uses" db:"max_uses"`
	Uses      int        `json:"uses" db:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Temporary bool       `json:"temporary" db:"temporary"` // Kick when they go offline
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Populated on fetch
	Server  *Server  `json:"server,omitempty"`
	Channel *Channel `json:"channel,omitempty"`
	Creator *User    `json:"creator,omitempty"`
}

// IsExpired checks if the invite has expired
func (i *Invite) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// IsMaxUsesReached checks if the invite has reached max uses
func (i *Invite) IsMaxUsesReached() bool {
	if i.MaxUses == 0 {
		return false
	}
	return i.Uses >= i.MaxUses
}

// IsValid checks if the invite can be used
func (i *Invite) IsValid() bool {
	return !i.IsExpired() && !i.IsMaxUsesReached()
}

// Ban represents a server ban
type Ban struct {
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Reason    string    `json:"reason,omitempty" db:"reason"`
	BannedBy  uuid.UUID `json:"banned_by" db:"banned_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Populated on fetch
	User     *User `json:"user,omitempty"`
	Moderator *User `json:"moderator,omitempty"`
}

// AuditLogEntry represents an action in the audit log
type AuditLogEntry struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ServerID   uuid.UUID `json:"server_id" db:"server_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	TargetID   *uuid.UUID `json:"target_id,omitempty" db:"target_id"`
	ActionType string    `json:"action_type" db:"action_type"`
	Changes    []Change  `json:"changes,omitempty"`
	Reason     string    `json:"reason,omitempty" db:"reason"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`

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
	AuditLogServerUpdate       = "SERVER_UPDATE"
	AuditLogChannelCreate      = "CHANNEL_CREATE"
	AuditLogChannelUpdate      = "CHANNEL_UPDATE"
	AuditLogChannelDelete      = "CHANNEL_DELETE"
	AuditLogMemberKick         = "MEMBER_KICK"
	AuditLogMemberBan          = "MEMBER_BAN"
	AuditLogMemberUnban        = "MEMBER_UNBAN"
	AuditLogMemberUpdate       = "MEMBER_UPDATE"
	AuditLogRoleCreate         = "ROLE_CREATE"
	AuditLogRoleUpdate         = "ROLE_UPDATE"
	AuditLogRoleDelete         = "ROLE_DELETE"
	AuditLogInviteCreate       = "INVITE_CREATE"
	AuditLogInviteDelete       = "INVITE_DELETE"
	AuditLogWebhookCreate      = "WEBHOOK_CREATE"
	AuditLogWebhookUpdate      = "WEBHOOK_UPDATE"
	AuditLogWebhookDelete      = "WEBHOOK_DELETE"
	AuditLogEmojiCreate        = "EMOJI_CREATE"
	AuditLogEmojiUpdate        = "EMOJI_UPDATE"
	AuditLogEmojiDelete        = "EMOJI_DELETE"
	AuditLogMessageDelete      = "MESSAGE_DELETE"
	AuditLogMessageBulkDelete  = "MESSAGE_BULK_DELETE"
	AuditLogMessagePin         = "MESSAGE_PIN"
	AuditLogMessageUnpin       = "MESSAGE_UNPIN"
)
