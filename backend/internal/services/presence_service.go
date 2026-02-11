package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

const (
	presenceTTL     = 2 * time.Minute
	idleTimeout     = 5 * time.Minute
	heartbeatInterval = 30 * time.Second
)

// PresenceService handles user presence tracking
type PresenceService struct {
	cache     CacheService
	eventBus  EventBus
	serverRepo ServerRepository
}

// NewPresenceService creates a new presence service
func NewPresenceService(
	cache CacheService,
	eventBus EventBus,
	serverRepo ServerRepository,
) *PresenceService {
	return &PresenceService{
		cache:     cache,
		eventBus:  eventBus,
		serverRepo: serverRepo,
	}
}

// UpdatePresence updates a user's presence
func (s *PresenceService) UpdatePresence(
	ctx context.Context,
	userID uuid.UUID,
	status models.PresenceStatus,
	customStatus *string,
	clientType string, // "desktop", "mobile", "web"
) error {
	presence := &models.Presence{
		UserID:       userID,
		Status:       status,
		CustomStatus: customStatus,
		UpdatedAt:    time.Now(),
	}

	// Store in cache
	if err := s.cache.SetPresence(ctx, userID, string(status), presenceTTL); err != nil {
		return err
	}

	// Broadcast to relevant servers
	s.broadcastPresenceUpdate(ctx, userID, presence)

	return nil
}

// GetPresence gets a user's presence
func (s *PresenceService) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	status, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		return &models.Presence{
			UserID: userID,
			Status: models.PresenceOffline,
		}, nil
	}

	return &models.Presence{
		UserID: userID,
		Status: models.PresenceStatus(status),
	}, nil
}

// GetBulkPresence gets presence for multiple users
func (s *PresenceService) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	result := make(map[uuid.UUID]*models.Presence)

	for _, userID := range userIDs {
		presence, _ := s.GetPresence(ctx, userID)
		result[userID] = presence
	}

	return result, nil
}

// Heartbeat updates the user's last seen time
func (s *PresenceService) Heartbeat(ctx context.Context, userID uuid.UUID) error {
	// Extend presence TTL
	status, _ := s.cache.GetPresence(ctx, userID)
	if status == "" {
		status = string(models.PresenceOnline)
	}

	return s.cache.SetPresence(ctx, userID, status, presenceTTL)
}

// SetOffline marks a user as offline
func (s *PresenceService) SetOffline(ctx context.Context, userID uuid.UUID) error {
	presence := &models.Presence{
		UserID: userID,
		Status: models.PresenceOffline,
	}

	// Remove from cache (will return offline on next get)
	_ = s.cache.Delete(ctx, "presence:"+userID.String())

	// Broadcast
	s.broadcastPresenceUpdate(ctx, userID, presence)

	return nil
}

// TypingStart indicates a user started typing
func (s *PresenceService) TypingStart(ctx context.Context, userID, channelID uuid.UUID) error {
	typing := &models.TypingIndicator{
		UserID:    userID,
		ChannelID: channelID,
		Timestamp: time.Now(),
	}

	s.eventBus.Publish("typing.started", typing)

	return nil
}

// broadcastPresenceUpdate sends presence updates to relevant users
func (s *PresenceService) broadcastPresenceUpdate(ctx context.Context, userID uuid.UUID, presence *models.Presence) {
	// Get all servers the user is in
	servers, err := s.serverRepo.GetUserServers(ctx, userID)
	if err != nil {
		return
	}

	// Publish to each server's presence channel
	for _, server := range servers {
		s.eventBus.Publish("presence.updated", &PresenceUpdateEvent{
			UserID:   userID,
			ServerID: server.ID,
			Presence: presence,
		})
	}
}

// GetServerPresences gets presence for all members of a server
func (s *PresenceService) GetServerPresences(ctx context.Context, serverID uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	// Get all members
	members, err := s.serverRepo.GetMembers(ctx, serverID, 0, 1000) // TODO: Pagination
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(members))
	for i, member := range members {
		userIDs[i] = member.UserID
	}

	return s.GetBulkPresence(ctx, userIDs)
}

// Events

type PresenceUpdateEvent struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
	Presence *models.Presence
}
