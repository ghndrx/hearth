package models

import "time"

type Ban struct {
	ID          string     `json:"id" db:"id"`
	GuildID     string     `json:"guild_id" db:"guild_id"`
	UserID      string     `json:"user_id" db:"user_id"`
	ModeratorID string     `json:"moderator_id" db:"moderator_id"`
	Reason      string     `json:"reason" db:"reason"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}