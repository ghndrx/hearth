package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of scheduled event
type EventType int

const (
	EventTypeStage    EventType = 1
	EventTypeVoice    EventType = 2
	EventTypeExternal EventType = 3
)

// EventStatus represents the status of a scheduled event
type EventStatus int

const (
	EventStatusScheduled EventStatus = 1
	EventStatusActive    EventStatus = 2
	EventStatusCompleted EventStatus = 3
	EventStatusCancelled EventStatus = 4
)

// RSVPStatus represents a user's RSVP status for an event
type RSVPStatus int

const (
	RSVPStatusInterested RSVPStatus = 1
	RSVPStatusGoing      RSVPStatus = 2
)

// Event represents a scheduled event
type Event struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	ServerID       uuid.UUID       `json:"server_id" db:"server_id"`
	ChannelID      *uuid.UUID      `json:"channel_id,omitempty" db:"channel_id"`
	CreatorID      uuid.UUID       `json:"creator_id" db:"creator_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	ImageURL       *string         `json:"image_url,omitempty" db:"image_url"`
	ScheduledStart time.Time       `json:"scheduled_start" db:"scheduled_start"`
	ScheduledEnd   *time.Time      `json:"scheduled_end,omitempty" db:"scheduled_end"`
	EntityType     EventType       `json:"entity_type" db:"entity_type"`
	Location       string          `json:"location" db:"location"`
	Status         EventStatus     `json:"status" db:"status"`
	UserCount      int             `json:"user_count" db:"user_count"`
	RecurrenceRule json.RawMessage `json:"recurrence_rule,omitempty" db:"recurrence_rule"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// EventRSVP represents a user's RSVP to an event
type EventRSVP struct {
	EventID   uuid.UUID  `json:"event_id" db:"event_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Status    RSVPStatus `json:"status" db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Populated from joins
	User *PublicUser `json:"user,omitempty"`
}

// CreateEventRequest is the input for creating an event
type CreateEventRequest struct {
	Name           string          `json:"name" validate:"required,min=1,max=100"`
	Description    string          `json:"description,omitempty" validate:"omitempty,max=1000"`
	ImageURL       *string         `json:"image_url,omitempty"`
	ScheduledStart time.Time       `json:"scheduled_start" validate:"required"`
	ScheduledEnd   *time.Time      `json:"scheduled_end,omitempty"`
	EntityType     EventType       `json:"entity_type" validate:"required"`
	ChannelID      *uuid.UUID      `json:"channel_id,omitempty"`
	Location       string          `json:"location,omitempty" validate:"omitempty,max=100"`
	RecurrenceRule json.RawMessage `json:"recurrence_rule,omitempty"`
}

// UpdateEventRequest is the input for updating an event
type UpdateEventRequest struct {
	Name           *string         `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description    *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
	ImageURL       *string         `json:"image_url,omitempty"`
	ScheduledStart *time.Time      `json:"scheduled_start,omitempty"`
	ScheduledEnd   *time.Time      `json:"scheduled_end,omitempty"`
	EntityType     *EventType      `json:"entity_type,omitempty"`
	ChannelID      *uuid.UUID      `json:"channel_id,omitempty"`
	Location       *string         `json:"location,omitempty" validate:"omitempty,max=100"`
	Status         *EventStatus    `json:"status,omitempty"`
	RecurrenceRule json.RawMessage `json:"recurrence_rule,omitempty"`
}

// RSVPRequest is the input for RSVPing to an event
type RSVPRequest struct {
	Status RSVPStatus `json:"status" validate:"required"`
}
