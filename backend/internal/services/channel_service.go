package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"hearth/internal/events"
	"hearth/internal/models"
)

// ChannelService handles channel-related business logic
type ChannelService struct {
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	cache       CacheService
	eventBus    EventBus
}

// NewChannelService creates a new channel service
func NewChannelService(
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	cache CacheService,
	eventBus EventBus,
) *ChannelService {
	return &ChannelService{
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		cache:       cache,
		eventBus:    eventBus,
	}
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetChannel(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	channel, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Cache for next time
	if s.cache != nil {
		_ = s.cache.SetChannel(ctx, channel, 5*time.Minute)
	}

	return channel, nil
}

// CreateChannel creates a new channel in a server
func (s *ChannelService) CreateChannel(
	ctx context.Context,
	serverID uuid.UUID,
	creatorID uuid.UUID,
	name string,
	channelType models.ChannelType,
	parentID *uuid.UUID,
) (*models.Channel, error) {
	// Verify creator is a member with permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Check MANAGE_CHANNELS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, creatorID, models.PermManageChannels); err != nil {
			return nil, err
		}
	}

	// Get position (append to end)
	channels, err := s.channelRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	position := len(channels)

	channel := &models.Channel{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      name,
		Type:      channelType,
		Position:  position,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteServer(ctx, serverID)
	}

	s.eventBus.Publish("channel.created", &ChannelCreatedEvent{
		Channel:  channel,
		ServerID: serverID,
	})

	return channel, nil
}

// UpdateChannel updates a channel
func (s *ChannelService) UpdateChannel(
	ctx context.Context,
	id uuid.UUID,
	requesterID uuid.UUID,
	updates *models.ChannelUpdate,
) (*models.Channel, error) {
	channel, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// For server channels, check permissions
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Check MANAGE_CHANNELS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageChannels); err != nil {
				return nil, err
			}
		}
	}

	// Apply updates
	if updates.Name != nil {
		channel.Name = *updates.Name
	}
	if updates.Topic != nil {
		channel.Topic = *updates.Topic
	}
	if updates.Position != nil {
		channel.Position = *updates.Position
	}
	if updates.ParentID != nil {
		channel.ParentID = updates.ParentID
	}
	if updates.Slowmode != nil {
		channel.Slowmode = *updates.Slowmode
	}
	if updates.NSFW != nil {
		channel.NSFW = *updates.NSFW
	}
	if updates.E2EEEnabled != nil {
		channel.E2EEEnabled = *updates.E2EEEnabled
	}

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, id)
	}

	s.eventBus.Publish("channel.updated", &ChannelUpdatedEvent{
		Channel: channel,
	})

	return channel, nil
}

// DeleteChannel deletes a channel
func (s *ChannelService) DeleteChannel(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	// Can't delete DM channels
	if channel.Type == models.ChannelTypeDM || channel.Type == models.ChannelTypeGroupDM {
		return ErrCannotDeleteDM
	}

	// Check permissions
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
		// Check MANAGE_CHANNELS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageChannels); err != nil {
				return err
			}
		}
	}

	if err := s.channelRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, id)
	}

	s.eventBus.Publish("channel.deleted", &ChannelDeletedEvent{
		ChannelID: id,
		ServerID:  channel.ServerID,
	})

	return nil
}

// GetOrCreateDM gets or creates a DM channel between two users
func (s *ChannelService) GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	// Check if DM already exists
	channel, err := s.channelRepo.GetDMChannel(ctx, user1ID, user2ID)
	if err != nil {
		return nil, err
	}
	if channel != nil {
		return channel, nil
	}

	// Create new DM channel
	channel = &models.Channel{
		ID:          uuid.New(),
		Type:        models.ChannelTypeDM,
		E2EEEnabled: true, // DMs are always E2EE
		Recipients:  []uuid.UUID{user1ID, user2ID},
		CreatedAt:   time.Now(),
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// CreateGroupDM creates a group DM
func (s *ChannelService) CreateGroupDM(
	ctx context.Context,
	ownerID uuid.UUID,
	name string,
	recipientIDs []uuid.UUID,
) (*models.Channel, error) {
	// Include owner in recipients
	allRecipients := append([]uuid.UUID{ownerID}, recipientIDs...)

	channel := &models.Channel{
		ID:          uuid.New(),
		Name:        name,
		Type:        models.ChannelTypeGroupDM,
		OwnerID:     &ownerID,
		E2EEEnabled: false, // Group DMs are not E2EE by default
		Recipients:  allRecipients,
		CreatedAt:   time.Now(),
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, err
	}

	// Publish group DM creation event for WebSocket broadcast to recipients
	s.eventBus.Publish(events.GroupDMCreated, &GroupDMCreatedEvent{
		Channel: channel,
	})

	return channel, nil
}

// UpdateGroupDM updates a group DM's name
func (s *ChannelService) UpdateGroupDM(
	ctx context.Context,
	channelID uuid.UUID,
	requesterID uuid.UUID,
	updates *GroupDMUpdate,
) (*models.Channel, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	if channel.Type != models.ChannelTypeGroupDM {
		return nil, ErrNotGroupDM
	}

	// Only the owner can update group DM settings
	if channel.OwnerID == nil || *channel.OwnerID != requesterID {
		return nil, ErrNotGroupDMOwner
	}

	if updates.Name != nil {
		channel.Name = *updates.Name
	}

	if updates.Icon != nil {
		channel.Icon = updates.Icon
	}

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	// Publish group DM specific update event for WebSocket broadcast
	s.eventBus.Publish(events.GroupDMUpdated, &GroupDMUpdatedEvent{
		Channel: channel,
	})

	return channel, nil
}

// GetUserDMs gets all DM channels for a user
func (s *ChannelService) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return s.channelRepo.GetUserDMs(ctx, userID)
}

// GetServerChannels gets all channels in a server
func (s *ChannelService) GetServerChannels(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Channel, error) {
	// Verify requester is a member
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	channels, err := s.channelRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Filter channels by requester's VIEW_CHANNELS permission
	if s.permService != nil {
		var visible []*models.Channel
		for _, ch := range channels {
			canView, err := s.permService.HasChannelPermission(ctx, ch, requesterID, models.PermViewChannels)
			if err == nil && canView {
				visible = append(visible, ch)
			}
		}
		return visible, nil
	}

	return channels, nil
}

// ReorderChannels bulk-updates channel positions and category assignments
func (s *ChannelService) ReorderChannels(
	ctx context.Context,
	requesterID uuid.UUID,
	entries []models.ReorderChannelEntry,
) error {
	if len(entries) == 0 {
		return nil
	}

	// Get the first channel to determine the server
	firstChannel, err := s.channelRepo.GetByID(ctx, entries[0].ID)
	if err != nil {
		return err
	}
	if firstChannel == nil {
		return ErrChannelNotFound
	}

	// Verify requester has permission
	if firstChannel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *firstChannel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *firstChannel.ServerID, requesterID, models.PermManageChannels); err != nil {
				return err
			}
		}
	}

	if err := s.channelRepo.BulkUpdatePositions(ctx, entries); err != nil {
		return err
	}

	// Invalidate cache for all affected channels
	if s.cache != nil {
		for _, entry := range entries {
			_ = s.cache.DeleteChannel(ctx, entry.ID)
		}
		if firstChannel.ServerID != nil {
			_ = s.cache.DeleteServer(ctx, *firstChannel.ServerID)
		}
	}

	// Publish update events for each channel
	for _, entry := range entries {
		ch, _ := s.channelRepo.GetByID(ctx, entry.ID)
		if ch != nil {
			s.eventBus.Publish("channel.updated", &ChannelUpdatedEvent{
				Channel: ch,
			})
		}
	}

	return nil
}

// Events

type ChannelCreatedEvent struct {
	Channel  *models.Channel
	ServerID uuid.UUID
}

type ChannelUpdatedEvent struct {
	Channel *models.Channel
}

type ChannelDeletedEvent struct {
	ChannelID uuid.UUID
	ServerID  *uuid.UUID
}

// SharedChannelInfo represents a shared channel with server details
type SharedChannelInfo struct {
	ID         uuid.UUID
	Name       string
	ServerID   *uuid.UUID
	ServerName string
	ServerIcon *string
}

// GetSharedChannelsWithServerNames returns channels that both users have access to
func (s *ChannelService) GetSharedChannelsWithServerNames(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]SharedChannelInfo, int, error) {
	// Check if repo has the method via type assertion
	if repo, ok := s.channelRepo.(interface {
		GetSharedChannelsWithServerNames(ctx context.Context, userID1, userID2 uuid.UUID, limit int) (interface{}, int, error)
	}); ok {
		channels, total, err := repo.GetSharedChannelsWithServerNames(ctx, userID1, userID2, limit)
		if err != nil {
			return nil, 0, err
		}

		// Convert from repo type to service type
		result := []SharedChannelInfo{}
		if chSlice, ok := channels.([]*struct {
			ID         uuid.UUID  `db:"id"`
			Name       string     `db:"name"`
			ServerID   *uuid.UUID `db:"server_id"`
			ServerName string     `db:"server_name"`
			ServerIcon *string    `db:"server_icon"`
		}); ok {
			for _, ch := range chSlice {
				result = append(result, SharedChannelInfo{
					ID:         ch.ID,
					Name:       ch.Name,
					ServerID:   ch.ServerID,
					ServerName: ch.ServerName,
					ServerIcon: ch.ServerIcon,
				})
			}
		}
		return result, total, nil
	}

	// Fallback: return empty
	return []SharedChannelInfo{}, 0, nil
}

// GetPermissionOverrides returns all permission overrides for a channel
func (s *ChannelService) GetPermissionOverrides(ctx context.Context, channelID, requesterID uuid.UUID) ([]models.PermissionOverride, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// For server channels, check permissions
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageChannels); err != nil {
				return nil, err
			}
		}
	}

	return s.channelRepo.GetPermissionOverrides(ctx, channelID)
}

// SetPermissionOverride creates or updates a permission override for a channel
func (s *ChannelService) SetPermissionOverride(
	ctx context.Context,
	channelID, targetID uuid.UUID,
	targetType string,
	allow, deny int64,
	requesterID uuid.UUID,
) (*models.PermissionOverride, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Can't set permissions on DM channels
	if channel.ServerID == nil {
		return nil, ErrCannotDeleteDM
	}

	// Check MANAGE_CHANNELS permission
	member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageChannels); err != nil {
			return nil, err
		}
	}

	override := &models.PermissionOverride{
		ChannelID:  channelID,
		TargetType: targetType,
		TargetID:   targetID,
		Allow:      allow,
		Deny:       deny,
	}

	if err := s.channelRepo.UpsertPermissionOverride(ctx, override); err != nil {
		return nil, err
	}

	// Invalidate cache for the channel
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	// Publish event
	s.eventBus.Publish("channel.permission_override_updated", &ChannelPermissionOverrideEvent{
		ChannelID:  channelID,
		TargetType: targetType,
		TargetID:   targetID,
		Allow:      allow,
		Deny:       deny,
	})

	return override, nil
}

// DeletePermissionOverride removes a permission override from a channel
func (s *ChannelService) DeletePermissionOverride(
	ctx context.Context,
	channelID, targetID uuid.UUID,
	targetType string,
	requesterID uuid.UUID,
) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	// Can't delete permissions on DM channels
	if channel.ServerID == nil {
		return ErrCannotDeleteDM
	}

	// Check MANAGE_CHANNELS permission
	member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageChannels); err != nil {
			return err
		}
	}

	if err := s.channelRepo.DeletePermissionOverride(ctx, channelID, targetID, targetType); err != nil {
		return err
	}

	// Invalidate cache for the channel
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	// Publish event
	s.eventBus.Publish("channel.permission_override_deleted", &ChannelPermissionOverrideDeletedEvent{
		ChannelID:  channelID,
		TargetType: targetType,
		TargetID:   targetID,
	})

	return nil
}

// ChannelPermissionOverrideEvent is published when a permission override is updated
type ChannelPermissionOverrideEvent struct {
	ChannelID  uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Allow      int64
	Deny       int64
}

// ChannelPermissionOverrideDeletedEvent is published when a permission override is deleted
type ChannelPermissionOverrideDeletedEvent struct {
	ChannelID  uuid.UUID
	TargetType string
	TargetID   uuid.UUID
}

// GroupDMCreatedEvent is published when a group DM is created
type GroupDMCreatedEvent struct {
	Channel *models.Channel
}

// GroupDMUpdatedEvent is published when a group DM is updated
type GroupDMUpdatedEvent struct {
	Channel *models.Channel
}

// GroupDMUpdate describes fields that can be updated on a group DM
type GroupDMUpdate struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Icon *string `json:"icon,omitempty" validate:"omitempty,max=100"`
}

// MuteRepository defines the interface for Mute data access operations.
type MuteRepository interface {
	// Create creates a new Mute record in the database.
	Create(ctx context.Context, mute *models.Mute) error

	// GetByID retrieves a mute by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Mute, error)

	// GetByChannelAndUser retrieves mute status for a specific channel and user.
	GetByChannelAndUser(ctx context.Context, channelID, userID uuid.UUID) (*models.Mute, error)

	// Update modifies an existing mute record (e.g., ending the mute).
	Update(ctx context.Context, mute *models.Mute) error

	// SoftDelete marks a mute as ended logic-wise (optional but good for data integrity)
	// or relies on the `EndedAt` timestamp field.
}

// MuteService handles business logic related to muting users in voice/text channels.
type MuteService struct {
	repo MuteRepository
}

// NewMuteService creates a new MuteService instance.
func NewMuteService(repo MuteRepository) *MuteService {
	return &MuteService{
		repo: repo,
	}
}

// MuteUser creates a new mute record for a user in a specific channel.
func (s *MuteService) MuteUser(ctx context.Context, channelID, userID uuid.UUID, durationMinutes int) (*models.Mute, error) {
	if durationMinutes < 0 {
		return nil, errors.New("duration minutes cannot be negative")
	}

	// Calculate end time
	var endsAt *time.Time
	if durationMinutes > 0 {
		endTime := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
		endsAt = &endTime
	}

	mute := &models.Mute{
		ID:         uuid.New(),
		ChannelID:  channelID,
		UserID:     userID,
		MutedBy:    nil, // Could be populated from context if a bot/superuser ID is passed
		RoleID:     nil, // Could be populated if using roles
		Reason:     "",  // Could be added via params
		StartedAt:  time.Now(),
		EndedAt:    endsAt,
		RestoredAt: nil,
	}

	if err := s.repo.Create(ctx, mute); err != nil {
		return nil, fmt.Errorf("failed to create mute: %w", err)
	}

	return mute, nil
}

// UnmuteUser removes a mute for a user in a specific channel.
func (s *MuteService) UnmuteUser(ctx context.Context, channelID, userID uuid.UUID) error {
	// 1. Find the active mute
	mute, err := s.repo.GetByChannelAndUser(ctx, channelID, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve mute: %w", err)
	}

	if mute == nil {
		return ErrUserNotMuted
	}

	// 2. Update the mute record
	currentTime := time.Now()

	mute.EndedAt = &currentTime
	mute.RestoredAt = &currentTime

	if err := s.repo.Update(ctx, mute); err != nil {
		return fmt.Errorf("failed to update mute: %w", err)
	}

	return nil
}

// IsUserMuted checks if a user is currently muted in a specific channel.
func (s *MuteService) IsUserMuted(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	mute, err := s.repo.GetByChannelAndUser(ctx, channelID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check mute status: %w", err)
	}

	// If mute is nil, user is not muted.
	if mute == nil {
		return false, nil
	}

	// If EndedAt is nil (never ended) or EndedAt is in the future, user is currently muted.
	now := time.Now()

	return mute.EndedAt == nil || mute.EndedAt.After(now), nil
}

var ErrUserNotMuted = errors.New("user is not muted in this channel")

const (
	// TypingTTL is how long a typing indicator lasts
	TypingTTL = 10 * time.Second
)

// TypingService manages typing indicators
type TypingService struct {
	mu       sync.RWMutex
	typing   map[uuid.UUID]map[uuid.UUID]time.Time // channelID -> userID -> timestamp
	eventBus EventBus
}

// NewTypingService creates a new typing service
func NewTypingService(eventBus EventBus) *TypingService {
	svc := &TypingService{
		typing:   make(map[uuid.UUID]map[uuid.UUID]time.Time),
		eventBus: eventBus,
	}

	// Start background cleanup goroutine
	go svc.cleanupLoop()

	return svc
}

// StartTyping records that a user started typing in a channel
func (s *TypingService) StartTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] == nil {
		s.typing[channelID] = make(map[uuid.UUID]time.Time)
	}

	now := time.Now()
	s.typing[channelID][userID] = now

	// Publish typing event for WebSocket broadcast
	if s.eventBus != nil {
		indicator := &models.TypingIndicator{
			ChannelID: channelID,
			UserID:    userID,
			Timestamp: now,
		}
		s.eventBus.Publish(events.TypingStarted, indicator)
	}

	return nil
}

// StopTyping removes a user's typing indicator (e.g., when they send a message)
func (s *TypingService) StopTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.typing[channelID] != nil {
		delete(s.typing[channelID], userID)
		if len(s.typing[channelID]) == 0 {
			delete(s.typing, channelID)
		}
	}

	return nil
}

// GetTypingUsers returns a list of users currently typing in a channel
func (s *TypingService) GetTypingUsers(ctx context.Context, channelID uuid.UUID) ([]models.TypingIndicator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var indicators []models.TypingIndicator
	now := time.Now()

	if channelUsers, ok := s.typing[channelID]; ok {
		for userID, ts := range channelUsers {
			if now.Sub(ts) < TypingTTL {
				indicators = append(indicators, models.TypingIndicator{
					ChannelID: channelID,
					UserID:    userID,
					Timestamp: ts,
				})
			}
		}
	}

	return indicators, nil
}

// IsTyping checks if a specific user is currently typing in a channel
func (s *TypingService) IsTyping(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if channelUsers, ok := s.typing[channelID]; ok {
		if ts, exists := channelUsers[userID]; exists {
			return time.Since(ts) < TypingTTL, nil
		}
	}

	return false, nil
}

// cleanupLoop periodically cleans up expired typing indicators
func (s *TypingService) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired typing indicators
func (s *TypingService) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for channelID, channelUsers := range s.typing {
		for userID, ts := range channelUsers {
			if now.Sub(ts) >= TypingTTL {
				delete(channelUsers, userID)
			}
		}
		if len(channelUsers) == 0 {
			delete(s.typing, channelID)
		}
	}
}

// GetTypingUserIDs returns just the user IDs currently typing (convenience method)
func (s *TypingService) GetTypingUserIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	indicators, err := s.GetTypingUsers(ctx, channelID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(indicators))
	for i, ind := range indicators {
		userIDs[i] = ind.UserID
	}

	return userIDs, nil
}

// ClearChannel removes all typing indicators for a channel (e.g., when channel is deleted)
func (s *TypingService) ClearChannel(ctx context.Context, channelID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.typing, channelID)
	return nil
}

var (
	ErrAlreadyFollowing = errors.New("already following this channel")
	ErrNotFollowing     = errors.New("not following this channel")
	ErrCannotFollowSelf = errors.New("a channel cannot follow itself")
)

// FollowRepositoryInterface defines follow data access
type FollowRepositoryInterface interface {
	Create(ctx context.Context, follow *models.FollowedChannel) error
	Delete(ctx context.Context, channelID, followerChannelID uuid.UUID) error
	GetByChannelAndFollower(ctx context.Context, channelID, followerChannelID uuid.UUID) (*models.FollowedChannel, error)
	GetFollowers(ctx context.Context, channelID uuid.UUID) ([]models.FollowedChannel, error)
}

// FollowService handles channel follow business logic
type FollowService struct {
	followRepo  FollowRepositoryInterface
	channelRepo ChannelRepository
}

// NewFollowService creates a new follow service
func NewFollowService(followRepo FollowRepositoryInterface, channelRepo ChannelRepository) *FollowService {
	return &FollowService{
		followRepo:  followRepo,
		channelRepo: channelRepo,
	}
}

// FollowChannel creates a follow relationship between channels
func (s *FollowService) FollowChannel(ctx context.Context, channelID, followerChannelID uuid.UUID) (*models.FollowedChannel, error) {
	if channelID == followerChannelID {
		return nil, ErrCannotFollowSelf
	}

	// Verify target channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Verify follower channel exists
	follower, err := s.channelRepo.GetByID(ctx, followerChannelID)
	if err != nil {
		return nil, err
	}
	if follower == nil {
		return nil, ErrChannelNotFound
	}

	// Check if already following
	existing, err := s.followRepo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyFollowing
	}

	follow := &models.FollowedChannel{
		ChannelID:         channelID,
		FollowerChannelID: followerChannelID,
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.followRepo.Create(ctx, follow); err != nil {
		return nil, err
	}

	return follow, nil
}

// UnfollowChannel removes a follow relationship
func (s *FollowService) UnfollowChannel(ctx context.Context, channelID, followerChannelID uuid.UUID) error {
	existing, err := s.followRepo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFollowing
	}

	return s.followRepo.Delete(ctx, channelID, followerChannelID)
}

// GetFollowers retrieves all followers of a channel
func (s *FollowService) GetFollowers(ctx context.Context, channelID uuid.UUID) ([]models.FollowedChannel, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	return s.followRepo.GetFollowers(ctx, channelID)
}
