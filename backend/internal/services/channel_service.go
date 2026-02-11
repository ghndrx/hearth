package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrChannelNotFound  = errors.New("channel not found")
	ErrNotChannelMember = errors.New("not a member of this channel")
	ErrCannotDeleteDM   = errors.New("cannot delete DM channel")
)

// ChannelService handles channel-related business logic
type ChannelService struct {
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	cache       CacheService
	eventBus    EventBus
}

// NewChannelService creates a new channel service
func NewChannelService(
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	cache CacheService,
	eventBus EventBus,
) *ChannelService {
	return &ChannelService{
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		cache:       cache,
		eventBus:    eventBus,
	}
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	// Try cache first
	if cached, err := s.cache.GetChannel(ctx, id); err == nil && cached != nil {
		return cached, nil
	}

	channel, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Cache for next time
	_ = s.cache.SetChannel(ctx, channel, 5*time.Minute)

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

	// TODO: Check MANAGE_CHANNELS permission

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
	_ = s.cache.DeleteServer(ctx, serverID)

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
		// TODO: Check MANAGE_CHANNELS permission
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
	_ = s.cache.DeleteChannel(ctx, id)

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
		// TODO: Check MANAGE_CHANNELS permission
	}

	if err := s.channelRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.DeleteChannel(ctx, id)

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

	// TODO: Filter channels by requester's permissions

	return channels, nil
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
