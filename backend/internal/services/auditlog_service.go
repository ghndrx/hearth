package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// AuditLogFilter contains filtering options for audit log queries
type AuditLogFilter struct {
	ActionType string     // Filter by specific action type (e.g., "MEMBER_BAN", "CHANNEL_CREATE")
	UserID     *uuid.UUID // Filter by the user who performed the action
	TargetID   *uuid.UUID // Filter by the target of the action
	Before     *time.Time // Filter entries before this time
	After      *time.Time // Filter entries after this time
	Limit      int        // Maximum number of entries to return (default 50, max 100)
	Offset     int        // Offset for pagination
}

// AuditLogServiceInterface defines the audit log service methods
type AuditLogServiceInterface interface {
	Log(ctx context.Context, serverID, userID uuid.UUID, action string, targetID *uuid.UUID, changes []models.Change, reason string) error
	GetLogs(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error)
	GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error)
	GetActionTypes() []string
}

// AuditLogService manages audit log entries
type AuditLogService struct {
	mu      sync.RWMutex
	entries []models.AuditLogEntry
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService() *AuditLogService {
	return &AuditLogService{
		entries: make([]models.AuditLogEntry, 0),
	}
}

// Log creates a new audit log entry
func (s *AuditLogService) Log(ctx context.Context, serverID, userID uuid.UUID, action string, targetID *uuid.UUID, changes []models.Change, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := models.AuditLogEntry{
		ID:         uuid.New(),
		ServerID:   serverID,
		UserID:     userID,
		TargetID:   targetID,
		ActionType: action,
		Changes:    changes,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}

	s.entries = append(s.entries, entry)
	return nil
}

// GetLogs retrieves audit log entries with filtering and pagination
func (s *AuditLogService) GetLogs(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]models.AuditLogEntry, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Set default limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Filter entries
	var filtered []models.AuditLogEntry
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := s.entries[i]

		// Must match server
		if entry.ServerID != serverID {
			continue
		}

		// Apply action type filter
		if filter.ActionType != "" && entry.ActionType != filter.ActionType {
			continue
		}

		// Apply user filter
		if filter.UserID != nil && entry.UserID != *filter.UserID {
			continue
		}

		// Apply target filter
		if filter.TargetID != nil {
			if entry.TargetID == nil || *entry.TargetID != *filter.TargetID {
				continue
			}
		}

		// Apply date range filters
		if filter.Before != nil && !entry.CreatedAt.Before(*filter.Before) {
			continue
		}
		if filter.After != nil && !entry.CreatedAt.After(*filter.After) {
			continue
		}

		filtered = append(filtered, entry)
	}

	total := len(filtered)

	// Apply pagination
	start := filter.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

// GetLogByID retrieves a specific audit log entry
func (s *AuditLogService) GetLogByID(ctx context.Context, serverID, entryID uuid.UUID) (*models.AuditLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.entries {
		if s.entries[i].ID == entryID && s.entries[i].ServerID == serverID {
			entry := s.entries[i]
			return &entry, nil
		}
	}

	return nil, ErrAuditLogNotFound
}

// GetActionTypes returns all valid audit log action types
func (s *AuditLogService) GetActionTypes() []string {
	return []string{
		models.AuditLogServerUpdate,
		models.AuditLogChannelCreate,
		models.AuditLogChannelUpdate,
		models.AuditLogChannelDelete,
		models.AuditLogMemberKick,
		models.AuditLogMemberBan,
		models.AuditLogMemberUnban,
		models.AuditLogMemberUpdate,
		models.AuditLogRoleCreate,
		models.AuditLogRoleUpdate,
		models.AuditLogRoleDelete,
		models.AuditLogInviteCreate,
		models.AuditLogInviteDelete,
		models.AuditLogWebhookCreate,
		models.AuditLogWebhookUpdate,
		models.AuditLogWebhookDelete,
		models.AuditLogEmojiCreate,
		models.AuditLogEmojiUpdate,
		models.AuditLogEmojiDelete,
		models.AuditLogMessageDelete,
		models.AuditLogMessageBulkDelete,
		models.AuditLogMessagePin,
		models.AuditLogMessageUnpin,
	}
}
