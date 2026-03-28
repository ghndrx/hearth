package models

import (
	"time"

	"github.com/google/uuid"
)

// Poll represents a poll in a channel
type Poll struct {
	ID         uuid.UUID    `json:"id" db:"id"`
	ChannelID  uuid.UUID    `json:"channel_id" db:"channel_id"`
	CreatorID  uuid.UUID    `json:"creator_id" db:"creator_id"`
	Question   string       `json:"question" db:"question"`
	Options    []PollOption `json:"options,omitempty"`
	IsMultiple bool         `json:"is_multiple" db:"is_multiple"`
	EndTime    *time.Time   `json:"end_time,omitempty" db:"end_time"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at"`
}

// PollOption represents an option in a poll
type PollOption struct {
	ID     uuid.UUID `json:"id" db:"id"`
	PollID uuid.UUID `json:"poll_id" db:"poll_id"`
	Text   string    `json:"text" db:"text"`
	Votes  int       `json:"votes" db:"votes"`
}

// PollOptionVote represents a vote on a poll option
type PollOptionVote struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PollID    uuid.UUID `json:"poll_id" db:"poll_id"`
	OptionID  uuid.UUID `json:"option_id" db:"option_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
