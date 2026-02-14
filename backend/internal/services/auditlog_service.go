package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID        uuid.UUID
	ServerID  uuid.UUID
	UserID    uuid.UUID
	Action    string
	TargetID  *uuid.UUID
	Changes   map[string]interface{}
	CreatedAt time.Time
}

type AuditLogService struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

func NewAuditLogService() *AuditLogService {
	return &AuditLogService{}
}

func (s *AuditLogService) Log(ctx context.Context, serverID, userID uuid.UUID, action string, targetID *uuid.UUID, changes map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, AuditEntry{
		ID:        uuid.New(),
		ServerID:  serverID,
		UserID:    userID,
		Action:    action,
		TargetID:  targetID,
		Changes:   changes,
		CreatedAt: time.Now(),
	})
	return nil
}

func (s *AuditLogService) GetLogs(ctx context.Context, serverID uuid.UUID, limit int) ([]AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var logs []AuditEntry
	for i := len(s.entries) - 1; i >= 0 && len(logs) < limit; i-- {
		if s.entries[i].ServerID == serverID {
			logs = append(logs, s.entries[i])
		}
	}
	return logs, nil
}
