package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// FriendshipStatus represents the status of a friendship
type FriendshipStatus string

const (
	FriendshipStatusPending  FriendshipStatus = "pending"
	FriendshipStatusAccepted FriendshipStatus = "accepted"
	FriendshipStatusBlocked  FriendshipStatus = "blocked"
)

// Friendship represents a relationship between two users
type Friendship struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID1   uuid.UUID        `json:"user_id_1" db:"user_id_1"`
	UserID2   uuid.UUID        `json:"user_id_2" db:"user_id_2"`
	Status    FriendshipStatus `json:"status" db:"status"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`

	// Populated from joins
	User1 *PublicUser `json:"user_1,omitempty"`
	User2 *PublicUser `json:"user_2,omitempty"`
}

// ErrRecordNotFound is returned when a record is not found in the database
var ErrRecordNotFound = errors.New("record not found")
