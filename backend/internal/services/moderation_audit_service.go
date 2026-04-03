package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hearth/internal/events"
	"hearth/internal/models"
)

// ModerationEventType defines the types of moderation events
type ModerationEventType string

const (
	// Member moderation events
	ModerationEventMemberBan    ModerationEventType = "member_ban"
	ModerationEventMemberKick   ModerationEventType = "member_kick"
	ModerationEventMemberMute   ModerationEventType = "member_mute"
	ModerationEventMemberWarn   ModerationEventType = "member_warn"
	ModerationEventMemberUnmute ModerationEventType = "member_unmute"
	ModerationEventMemberUnban  ModerationEventType = "member_unban"

	// Message moderation events
	ModerationEventMessageDelete ModerationEventType = "message_delete"
	ModerationEventMessageEdit   ModerationEventType = "message_edit"

	// Channel/Role CRUD events
	ModerationEventChannelCreate ModerationEventType = "channel_create"
	ModerationEventChannelUpdate ModerationEventType = "channel_update"
	ModerationEventChannelDelete ModerationEventType = "channel_delete"
	ModerationEventRoleCreate    ModerationEventType = "role_create"
	ModerationEventRoleUpdate    ModerationEventType = "role_update"
	ModerationEventRoleDelete    ModerationEventType = "role_delete"

	// WebSocket event type for audit logs
	WebSocketEventAuditLogCreate = "AUDIT_LOG_CREATE"
)

// ModerationAuditMetadata contains additional context for moderation events
type ModerationAuditMetadata struct {
	Reason         string                 `json:"reason,omitempty"`
	Duration       time.Duration          `json:"duration,omitempty"`        // For mutes
	OldValue       interface{}            `json:"old_value,omitempty"`       // For updates
	NewValue       interface{}            `json:"new_value,omitempty"`       // For updates
	Changes        []models.Change        `json:"changes,omitempty"`         // Detailed changes
	ChannelID      *uuid.UUID             `json:"channel_id,omitempty"`      // For message events
	ChannelName    string                 `json:"channel_name,omitempty"`    // For channel events
	RoleID         *uuid.UUID             `json:"role_id,omitempty"`        // For role events
	RoleName       string                 `json:"role_name,omitempty"`       // For role events
	MessageID      *uuid.UUID             `json:"message_id,omitempty"`      // For message events
	MessageContent string                 `json:"message_content,omitempty"` // For message delete
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// ModerationAuditEntry represents a moderation audit log entry for the service layer
type ModerationAuditEntry struct {
	ID         uuid.UUID                `json:"id"`
	ServerID   uuid.UUID                `json:"server_id"`
	ActorID    uuid.UUID                `json:"actor_id"`
	EventType  ModerationEventType     `json:"event_type"`
	TargetID   *uuid.UUID               `json:"target_id,omitempty"`
	TargetType string                   `json:"target_type,omitempty"`
	Metadata   *ModerationAuditMetadata `json:"metadata,omitempty"`
	IPAddress  string                   `json:"ip_address,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
}

// ModerationAnalytics contains comprehensive moderation analytics
type ModerationAnalytics struct {
	ServerID          uuid.UUID                    `json:"server_id"`
	Period            string                       `json:"period"` // e.g., "7d", "30d", "90d"
	Ratios            *models.ModerationRatiosStats `json:"ratios"`
	TrendData         []models.DailyModerationTrend `json:"trend_data"`
	ModeratorActivity []models.ModeratorStats       `json:"moderator_activity"`
	RepeatOffenders   []models.RepeatOffenderStats  `json:"repeat_offenders"`
}

// ModerationAuditRepository interface for persistence
type ModerationAuditRepository interface {
	Create(ctx context.Context, entry *ModerationAuditEntry) error
	GetByServer(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]ModerationAuditEntry, int, error)
	GetModerationAnalytics(ctx context.Context, serverID uuid.UUID, since time.Time) (*ModerationAnalytics, error)
	ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]ModerationAuditEntry, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// ModerationAuditService handles moderation audit logging with event emission
type ModerationAuditService struct {
	repo     ModerationAuditRepository
	eventBus EventBusInterface
	wsHub    WebSocketHubInterface
}

// EventBusInterface defines the interface for event bus operations
type EventBusInterface interface {
	Publish(eventType string, data interface{})
}

// WebSocketEvent represents a WebSocket event to be sent
type WebSocketEvent struct {
	Op       int         `json:"op"`
	Type     string      `json:"t"`
	Data     interface{} `json:"d"`
	ServerID *uuid.UUID  `json:"-"`
}

// WebSocketHubInterface defines the interface for WebSocket hub operations
type WebSocketHubInterface interface {
	SendToServer(serverID uuid.UUID, event *WebSocketEvent)
}

// NewModerationAuditService creates a new moderation audit service
func NewModerationAuditService(repo ModerationAuditRepository, eventBus EventBusInterface, wsHub WebSocketHubInterface) *ModerationAuditService {
	return &ModerationAuditService{
		repo:     repo,
		eventBus: eventBus,
		wsHub:    wsHub,
	}
}

// LogModerationAction logs a moderation action and emits events
func (s *ModerationAuditService) LogModerationAction(
	ctx context.Context,
	serverID, actorID uuid.UUID,
	eventType ModerationEventType,
	targetID *uuid.UUID,
	targetType string,
	metadata *ModerationAuditMetadata,
	ipAddress string,
) error {
	entry := &ModerationAuditEntry{
		ID:         uuid.New(),
		ServerID:   serverID,
		ActorID:    actorID,
		EventType:  eventType,
		TargetID:   targetID,
		TargetType: targetType,
		Metadata:   metadata,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
	}

	// Persist to database
	if err := s.repo.Create(ctx, entry); err != nil {
		return fmt.Errorf("failed to persist moderation audit: %w", err)
	}

	// Emit event to internal event bus
	s.emitEvent(entry)

	// Broadcast to WebSocket
	s.broadcastToWebSocket(entry)

	return nil
}

// emitEvent publishes the moderation event to the internal event bus
func (s *ModerationAuditService) emitEvent(entry *ModerationAuditEntry) {
	if s.eventBus == nil {
		return
	}

	eventData := map[string]interface{}{
		"id":          entry.ID,
		"server_id":   entry.ServerID,
		"actor_id":    entry.ActorID,
		"event_type":  entry.EventType,
		"target_id":   entry.TargetID,
		"target_type": entry.TargetType,
		"metadata":    entry.Metadata,
		"created_at":  entry.CreatedAt,
	}

	s.eventBus.Publish(fmt.Sprintf("moderation.%s", entry.EventType), eventData)
	s.eventBus.Publish(events.ModerationEventAny, eventData) // Wildcard for all moderation events
}

// broadcastToWebSocket sends the audit event to connected WebSocket clients
func (s *ModerationAuditService) broadcastToWebSocket(entry *ModerationAuditEntry) {
	if s.wsHub == nil {
		return
	}

	auditData := map[string]interface{}{
		"id":          entry.ID,
		"server_id":   entry.ServerID,
		"actor_id":    entry.ActorID,
		"event_type":  entry.EventType,
		"target_id":   entry.TargetID,
		"target_type": entry.TargetType,
		"metadata":    entry.Metadata,
		"created_at":  entry.CreatedAt,
	}

	// Create WebSocket event with OpDispatch (0 = Discord dispatch)
	event := &WebSocketEvent{
		Op:       0, // OpDispatch
		Type:     WebSocketEventAuditLogCreate,
		Data:     auditData,
		ServerID: &entry.ServerID,
	}

	// Send to all users in the server
	s.wsHub.SendToServer(entry.ServerID, event)
}

// GetAuditLogs retrieves audit logs with filtering
func (s *ModerationAuditService) GetAuditLogs(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]ModerationAuditEntry, int, error) {
	return s.repo.GetByServer(ctx, serverID, filter)
}

// GetModerationAnalytics retrieves comprehensive moderation analytics
func (s *ModerationAuditService) GetModerationAnalytics(ctx context.Context, serverID uuid.UUID, days int) (*ModerationAnalytics, error) {
	since := time.Now().AddDate(0, 0, -days)
	return s.repo.GetModerationAnalytics(ctx, serverID, since)
}

// ExportForGDPR exports all audit logs related to a user for GDPR compliance
func (s *ModerationAuditService) ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]ModerationAuditEntry, error) {
	return s.repo.ExportForGDPR(ctx, userID)
}

// CleanupOldLogs removes audit logs older than the retention period
func (s *ModerationAuditService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	return s.repo.DeleteOlderThan(ctx, before)
}

// SerializeMetadata converts ModerationAuditMetadata to JSON for storage
func SerializeMetadata(metadata *ModerationAuditMetadata) ([]byte, error) {
	if metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(metadata)
}

// DeserializeMetadata converts JSON back to ModerationAuditMetadata
func DeserializeMetadata(data []byte) (*ModerationAuditMetadata, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var metadata ModerationAuditMetadata
	err := json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}
