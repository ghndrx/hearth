package models

import (
	"time"

	"github.com/google/uuid"
)

// Mute represents a mute record for a user in a channel
type Mute struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ChannelID  uuid.UUID  `json:"channel_id" db:"channel_id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	MutedBy    *uuid.UUID `json:"muted_by,omitempty" db:"muted_by"`
	RoleID     *uuid.UUID `json:"role_id,omitempty" db:"role_id"`
	Reason     string     `json:"reason,omitempty" db:"reason"`
	StartedAt  time.Time  `json:"started_at" db:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	RestoredAt *time.Time `json:"restored_at,omitempty" db:"restored_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// TimeRange represents a time range with Start and End
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AddDuration adds minutes to the End time
func (t TimeRange) AddDuration(unit string, amount int) time.Time {
	if unit == "minutes" {
		return t.End.Add(time.Duration(amount) * time.Minute)
	}
	return t.End
}

// Now returns a TimeRange with current time as End
func (t TimeRange) Now() TimeRange {
	return TimeRange{End: time.Now()}
}

// IsAfter checks if the End time is after the given time
func (t TimeRange) IsAfter(other TimeRange) bool {
	return t.End.After(other.End)
}
