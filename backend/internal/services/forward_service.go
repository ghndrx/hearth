package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// Common errors
var (
	ErrCannotForwardToSameChannel = errors.New("cannot forward message to the same channel")
)

// ForwardService handles message forwarding business logic
type ForwardService struct {
	messageRepo        MessageRepository
	forwardedMsgRepo   ForwardedMessageRepository
	channelRepo        ChannelRepository
	serverRepo         ServerRepository
	permService        *PermissionService
	eventBus           EventBus
}

// NewForwardService creates a new forward service
func NewForwardService(
	messageRepo MessageRepository,
	forwardedMsgRepo ForwardedMessageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *ForwardService {
	return &ForwardService{
		messageRepo:      messageRepo,
		forwardedMsgRepo: forwardedMsgRepo,
		channelRepo:      channelRepo,
		serverRepo:       serverRepo,
		permService:      permService,
		eventBus:         eventBus,
	}
}

// ForwardMessage forwards a message to a destination channel
func (s *ForwardService) ForwardMessage(
	ctx context.Context,
	originalMessageID uuid.UUID,
	forwarderID uuid.UUID,
	destinationChannelID uuid.UUID,
	comment string,
) (*models.ForwardedMessage, error) {
	// Get the original message
	originalMsg, err := s.messageRepo.GetByID(ctx, originalMessageID)
	if err != nil {
		return nil, err
	}
	if originalMsg == nil {
		return nil, ErrMessageNotFound
	}

	// Get the destination channel
	destChannel, err := s.channelRepo.GetByID(ctx, destinationChannelID)
	if err != nil {
		return nil, err
	}
	if destChannel == nil {
		return nil, ErrChannelNotFound
	}

	// Check that we're not forwarding to the same channel
	if originalMsg.ChannelID == destinationChannelID {
		return nil, ErrCannotForwardToSameChannel
	}

	// Check that the forwarder can read the original message
	sourceChannel, err := s.channelRepo.GetByID(ctx, originalMsg.ChannelID)
	if err != nil {
		return nil, err
	}
	if sourceChannel == nil {
		return nil, ErrChannelNotFound
	}

	// Check read permission on source channel
	if err := s.checkReadPermission(ctx, sourceChannel, forwarderID); err != nil {
		return nil, err
	}

	// Check send permission on destination channel
	if err := s.checkSendPermission(ctx, destChannel, forwarderID); err != nil {
		return nil, err
	}

	// Create the forwarded message record
	forwardedMsg := &models.ForwardedMessage{
		ID:                   uuid.New(),
		OriginalMessageID:     originalMessageID,
		ForwardedByID:        forwarderID,
		DestinationChannelID: destinationChannelID,
		Comment:              comment,
		CreatedAt:            time.Now(),
	}

	if err := s.forwardedMsgRepo.Create(ctx, forwardedMsg); err != nil {
		return nil, err
	}

	// Publish event for the forward
	s.eventBus.Publish("message.forwarded", &MessageForwardedEvent{
		OriginalMessageID:     originalMessageID,
		ForwardedByID:        forwarderID,
		DestinationChannelID: destinationChannelID,
	})

	return forwardedMsg, nil
}

// GetForwardsByOriginalMessage returns all forwards of a message
func (s *ForwardService) GetForwardsByOriginalMessage(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error) {
	return s.forwardedMsgRepo.GetByOriginalMessageID(ctx, originalMessageID)
}

// GetForwardsByDestinationChannel returns all forwards sent to a channel
func (s *ForwardService) GetForwardsByDestinationChannel(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error) {
	return s.forwardedMsgRepo.GetByDestinationChannelID(ctx, channelID, limit, offset)
}

// checkReadPermission checks if a user can read messages in a channel
func (s *ForwardService) checkReadPermission(ctx context.Context, channel *models.Channel, userID uuid.UUID) error {
	if s.permService != nil && channel.ServerID != nil {
		// Check both view channels and read message history permissions
		if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermViewChannels); err != nil {
			return err
		}
		return s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermReadMessageHistory)
	}
	// Fallback: check channel membership
	if channel.Type == models.ChannelTypeDM {
		// For DMs, check if user is a recipient
		count, err := s.channelRepo.CountRecipients(ctx, channel.ID)
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrNotChannelMember
		}
		return nil
	}
	// For server channels, check membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return ErrNotChannelMember
		}
	}
	return nil
}

// checkSendPermission checks if a user can send messages in a channel
func (s *ForwardService) checkSendPermission(ctx context.Context, channel *models.Channel, userID uuid.UUID) error {
	if s.permService != nil && channel.ServerID != nil {
		return s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermSendMessages)
	}
	// Fallback: check channel membership
	if channel.Type == models.ChannelTypeDM {
		// For DMs, any recipient can send
		count, err := s.channelRepo.CountRecipients(ctx, channel.ID)
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrNotChannelMember
		}
		return nil
	}
	// For server channels, check membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return ErrNotChannelMember
		}
	}
	return nil
}

// MessageForwardedEvent is published when a message is forwarded
type MessageForwardedEvent struct {
	OriginalMessageID     uuid.UUID `json:"original_message_id"`
	ForwardedByID        uuid.UUID `json:"forwarded_by_id"`
	DestinationChannelID uuid.UUID `json:"destination_channel_id"`
}
