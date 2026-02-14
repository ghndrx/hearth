// package services
package services

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ----- Interfaces -----

// Status represents the user's current status.
type Status string

const (
	StatusOnline   Status = "online"
	StatusAway     Status = "away"
	StatusDoNotDisturb Status = "dnd"
	StatusInvisible Status = "invisible"
)

// Presence defines the data structure for a user's presence.
type Presence struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Status    Status    `json:"status"`
	ServerID  string    `json:"server_id"` // nil if global status
	Activity  string    `json:"activity"`  // e.g., "Playing Valorant"
	LastSeen  time.Time `json:"last_seen"`
}

// PresenceRepo defines the contract for storing and retrieving Presence data.
// In a real application, this would interface with a database.
type PresenceRepo interface {
	GetPresence(ctx context.Context, userID string, serverID *string) (*Presence, error)
	UpsertPresence(ctx context.Context, presence *Presence) error
	GetOnlineUsers(ctx context.Context, serverID string) ([]*Presence, error)
}

// ---- Service Definition -----

// PresenceService handles presence logic.
type PresenceService struct {
	repo PresenceRepo
	mu   sync.RWMutex // Protects in-memory cache for write-heavy operations if needed
}

// NewPresenceService creates a new PresenceService instance.
func NewPresenceService(repo PresenceRepo) *PresenceService {
	return &PresenceService{
		repo: repo,
	}
}

// UpdateStatus updates a user's presence state.
// It respects graceful error handling and logic to determine LastSeen.
func (s *PresenceService) UpdateStatus(ctx context.Context, userID, username, statusStr, activity string, serverID *string) error {
	
	// Ensure status string is valid
	var status Status
	switch statusStr {
	case string(StatusOnline), string(StatusAway), string(StatusDoNotDisturb), string(StatusInvisible):
		status = Status(statusStr)
	default:
		return errors.New("invalid status provided")
	}

	presence := &Presence{
		UserID:    userID,
		Username:  username,
		Status:    status,
		ServerID:  serverID,
		Activity:  activity,
		LastSeen:  time.Now(),
	}

	return s.repo.UpsertPresence(ctx, presence)
}

// GetOnlineUsers fetches all users who are currently marked as online or in a specific state.
func (s *PresenceService) GetOnlineUsers(ctx context.Context, serverID string) ([]*Presence, error) {
	// Using the repository to fetch data directly
	users, err := s.repo.GetOnlineUsers(ctx, serverID)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetPresence retrieves a specific user's presence by ID.
// (Optional helper method standard in services)
func (s *PresenceService) GetPresence(ctx context.Context, userID string, serverID *string) (*Presence, error) {
	return s.repo.GetPresence(ctx, userID, serverID)
}