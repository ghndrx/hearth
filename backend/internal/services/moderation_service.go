package services

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ModerationAction struct {
	ID        uuid.UUID
	ServerID  uuid.UUID
	UserID    uuid.UUID
	ModID     uuid.UUID
	Action    string
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type ModerationService struct {
	actions []ModerationAction
}

func NewModerationService() *ModerationService {
	return &ModerationService{}
}

func (s *ModerationService) Warn(ctx context.Context, serverID, userID, modID uuid.UUID, reason string) (*ModerationAction, error) {
	action := ModerationAction{
		ID:        uuid.New(),
		ServerID:  serverID,
		UserID:    userID,
		ModID:     modID,
		Action:    "warn",
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	s.actions = append(s.actions, action)
	return &action, nil
}

func (s *ModerationService) Timeout(ctx context.Context, serverID, userID, modID uuid.UUID, duration time.Duration, reason string) (*ModerationAction, error) {
	expires := time.Now().Add(duration)
	action := ModerationAction{
		ID:        uuid.New(),
		ServerID:  serverID,
		UserID:    userID,
		ModID:     modID,
		Action:    "timeout",
		Reason:    reason,
		ExpiresAt: &expires,
		CreatedAt: time.Now(),
	}
	s.actions = append(s.actions, action)
	return &action, nil
}

func (s *ModerationService) GetHistory(ctx context.Context, serverID, userID uuid.UUID) ([]ModerationAction, error) {
	var history []ModerationAction
	for _, a := range s.actions {
		if a.ServerID == serverID && a.UserID == userID {
			history = append(history, a)
		}
	}
	return history, nil
}
