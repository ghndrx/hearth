package services

import (
	"context"
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

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	s.eventBus.Publish("channel.updated", &ChannelUpdatedEvent{
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

// GroupDMUpdate describes fields that can be updated on a group DM
type GroupDMUpdate struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
}
