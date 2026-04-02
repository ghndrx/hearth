package services

import (
	"context"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// DMService handles DM-specific business logic beyond basic CRUD
type DMService struct {
	channelRepo ChannelRepository
	eventBus    EventBus
	cache       CacheService
}

// NewDMService creates a new DM service
func NewDMService(
	channelRepo ChannelRepository,
	eventBus EventBus,
	cache CacheService,
) *DMService {
	return &DMService{
		channelRepo: channelRepo,
		eventBus:    eventBus,
		cache:       cache,
	}
}

// AddUserToGroupDM adds a user to a group DM channel
func (s *DMService) AddUserToGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) (*models.Channel, error) {
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

	// Only the owner can add users
	if channel.OwnerID == nil || *channel.OwnerID != requesterID {
		return nil, ErrNotGroupDMOwner
	}

	// Check if user is already a recipient
	for _, r := range channel.Recipients {
		if r == userID {
			return nil, ErrAlreadyDMRecipient
		}
	}

	// Check group DM size limit (MaxGroupDMUsers max)
	if len(channel.Recipients) >= models.MaxGroupDMUsers {
		return nil, ErrGroupDMFull
	}

	if err := s.channelRepo.AddRecipient(ctx, channelID, userID); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	// Reload channel with updated recipients
	channel, err = s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	s.eventBus.Publish("dm.recipient_add", &DMRecipientEvent{
		ChannelID: channelID,
		UserID:    userID,
		Channel:   channel,
	})

	return channel, nil
}

// RemoveUserFromGroupDM removes a user from a group DM channel
func (s *DMService) RemoveUserFromGroupDM(ctx context.Context, channelID, requesterID, userID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.Type != models.ChannelTypeGroupDM {
		return ErrNotGroupDM
	}

	// Only the owner can remove users (unless removing self = leave)
	if requesterID != userID {
		if channel.OwnerID == nil || *channel.OwnerID != requesterID {
			return ErrNotGroupDMOwner
		}
	}

	// Check if user is a recipient
	found := false
	for _, r := range channel.Recipients {
		if r == userID {
			found = true
			break
		}
	}
	if !found {
		return ErrNotDMRecipient
	}

	if err := s.channelRepo.RemoveRecipient(ctx, channelID, userID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	s.eventBus.Publish("dm.recipient_remove", &DMRecipientEvent{
		ChannelID: channelID,
		UserID:    userID,
	})

	return nil
}

// LeaveDM removes the requester from a DM or group DM channel
func (s *DMService) LeaveDM(ctx context.Context, channelID, userID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.Type != models.ChannelTypeDM && channel.Type != models.ChannelTypeGroupDM {
		return ErrNotDMChannel
	}

	// Check if user is a recipient
	found := false
	for _, r := range channel.Recipients {
		if r == userID {
			found = true
			break
		}
	}
	if !found {
		return ErrNotDMRecipient
	}

	if err := s.channelRepo.RemoveRecipient(ctx, channelID, userID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	s.eventBus.Publish("dm.recipient_remove", &DMRecipientEvent{
		ChannelID: channelID,
		UserID:    userID,
	})

	return nil
}

// TransferGroupDMOwnership transfers ownership of a group DM to another member
func (s *DMService) TransferGroupDMOwnership(ctx context.Context, channelID, currentOwnerID, newOwnerID uuid.UUID) (*models.Channel, error) {
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

	// Only the current owner can transfer ownership
	if channel.OwnerID == nil || *channel.OwnerID != currentOwnerID {
		return nil, ErrNotGroupDMOwner
	}

	// New owner must be a recipient
	found := false
	for _, r := range channel.Recipients {
		if r == newOwnerID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrCannotTransferToNonMember
	}

	// Update ownership
	channel.OwnerID = &newOwnerID
	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteChannel(ctx, channelID)
	}

	// Publish ownership transfer event
	s.eventBus.Publish("dm.ownership_transfer", &DMRecipientEvent{
		ChannelID: channelID,
		UserID:    newOwnerID,
		Channel:   channel,
	})

	return channel, nil
}

// Events

// DMRecipientEvent is published when a user is added/removed from a DM
type DMRecipientEvent struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Channel   *models.Channel // populated on add
}
