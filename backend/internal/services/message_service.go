package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// MessageRepository defines the interface for message data access
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error)
	Update(ctx context.Context, message *models.Message) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Queries
	GetChannelMessages(ctx context.Context, channelID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error)
	GetPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]*models.Message, error)
	SearchMessages(ctx context.Context, query string, channelID *uuid.UUID, authorID *uuid.UUID, limit int) ([]*models.Message, error)

	// Reactions
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	GetReactions(ctx context.Context, messageID uuid.UUID) ([]*models.Reaction, error)

	// Bulk operations
	DeleteByChannel(ctx context.Context, channelID uuid.UUID) error
	DeleteByAuthor(ctx context.Context, channelID, authorID uuid.UUID, since time.Time) (int, error)
}

// MessageService handles message-related business logic
type MessageService struct {
	repo         MessageRepository
	channelRepo  ChannelRepository
	serverRepo   ServerRepository
	quotaService *QuotaService
	rateLimiter  RateLimiter
	e2eeService  E2EEService
	cache        CacheService
	eventBus     EventBus
}

// NewMessageService creates a new message service
func NewMessageService(
	repo MessageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	quotaService *QuotaService,
	rateLimiter RateLimiter,
	e2eeService E2EEService,
	cache CacheService,
	eventBus EventBus,
) *MessageService {
	return &MessageService{
		repo:         repo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		quotaService: quotaService,
		rateLimiter:  rateLimiter,
		e2eeService:  e2eeService,
		cache:        cache,
		eventBus:     eventBus,
	}
}

// SendMessage sends a message to a channel
func (s *MessageService) SendMessage(ctx context.Context, authorID uuid.UUID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID) (*models.Message, error) {
	// Get channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Check permissions for server channels
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, authorID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// TODO: Check SEND_MESSAGES permission
	}

	// Get quota limits
	var serverID *uuid.UUID
	if channel.ServerID != nil {
		serverID = channel.ServerID
	}
	limits, err := s.quotaService.GetEffectiveLimits(ctx, authorID, serverID)
	if err != nil {
		return nil, err
	}

	// Check message length
	if limits.MaxMessageLength > 0 && len(content) > limits.MaxMessageLength {
		return nil, ErrMessageTooLong
	}

	// Check if content or attachments exist
	if len(content) == 0 && len(attachments) == 0 {
		return nil, ErrEmptyMessage
	}

	// Check rate limit (skip for DMs in some cases)
	if channel.Type != models.ChannelTypeDM {
		if err := s.rateLimiter.Check(ctx, authorID, channelID); err != nil {
			return nil, ErrRateLimited
		}
	}

	// Check slowmode
	if channel.Slowmode > 0 {
		if err := s.rateLimiter.CheckSlowmode(ctx, authorID, channelID, channel.Slowmode); err != nil {
			return nil, ErrRateLimited
		}
	}

	// Convert attachments
	var msgAttachments []models.Attachment
	for _, att := range attachments {
		if att != nil {
			msgAttachments = append(msgAttachments, *att)
		}
	}

	// Create message
	message := &models.Message{
		ID:          uuid.New(),
		ChannelID:   channelID,
		ServerID:    channel.ServerID,
		AuthorID:    authorID,
		Content:     content,
		Type:        models.MessageTypeDefault,
		Attachments: msgAttachments,
		ReplyToID:   replyTo,
		CreatedAt:   time.Now(),
	}

	// Set reply type if replying
	if replyTo != nil {
		message.Type = models.MessageTypeReply
	}

	// Handle E2EE for DM channels
	isEncrypted := false
	if channel.Type == models.ChannelTypeDM || channel.Type == models.ChannelTypeGroupDM {
		if channel.E2EEEnabled {
			// Content should already be encrypted by client
			// We just verify the format
			if !s.e2eeService.ValidateEncryptedPayload(content) {
				return nil, ErrNoPermission
			}
			message.EncryptedContent = content
			message.Content = "" // Clear plaintext
			isEncrypted = true
		}
	} else if channel.E2EEEnabled {
		// Server channel with E2EE enabled
		if !s.e2eeService.ValidateEncryptedPayload(content) {
			return nil, ErrNoPermission
		}
		message.EncryptedContent = content
		message.Content = ""
		isEncrypted = true
	}

	// Parse mentions from content (if not encrypted)
	if !isEncrypted {
		message.Mentions = parseMentions(content)
	}

	if err := s.repo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Update channel's last message
	_ = s.channelRepo.UpdateLastMessage(ctx, channelID, message.ID, message.CreatedAt)

	// Emit event
	s.eventBus.Publish("message.created", &MessageCreatedEvent{
		Message:   message,
		ChannelID: channelID,
		ServerID:  channel.ServerID,
	})

	return message, nil
}

// EditMessage edits an existing message
func (s *MessageService) EditMessage(ctx context.Context, messageID uuid.UUID, authorID uuid.UUID, newContent string) (*models.Message, error) {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	if message.AuthorID != authorID {
		return nil, ErrNotMessageAuthor
	}

	message.Content = newContent
	message.EditedAt = timePtr(time.Now())

	// Re-parse mentions if not encrypted (EncryptedContent is empty for non-encrypted)
	if message.EncryptedContent == "" {
		message.Mentions = parseMentions(newContent)
	}

	if err := s.repo.Update(ctx, message); err != nil {
		return nil, err
	}

	s.eventBus.Publish("message.updated", &MessageUpdatedEvent{
		Message:   message,
		ChannelID: message.ChannelID,
	})

	return message, nil
}

// DeleteMessage deletes a message
func (s *MessageService) DeleteMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	// Author can always delete their own messages
	if message.AuthorID != requesterID {
		// Check if requester has MANAGE_MESSAGES permission
		channel, _ := s.channelRepo.GetByID(ctx, message.ChannelID)
		if channel != nil && channel.ServerID != nil {
			// TODO: Check MANAGE_MESSAGES permission
		} else {
			return ErrNotMessageAuthor
		}
	}

	if err := s.repo.Delete(ctx, messageID); err != nil {
		return err
	}

	s.eventBus.Publish("message.deleted", &MessageDeletedEvent{
		MessageID: messageID,
		ChannelID: message.ChannelID,
		AuthorID:  message.AuthorID,
	})

	return nil
}

// GetMessages retrieves messages from a channel with pagination
func (s *MessageService) GetMessages(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Check access
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// TODO: Check READ_MESSAGES permission
	} else {
		// DM channel - check if requester is participant
		if !isChannelParticipant(channel, requesterID) {
			return nil, ErrNoPermission
		}
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetChannelMessages(ctx, channelID, before, after, limit)
}

// GetMessage retrieves a specific message by ID
func (s *MessageService) GetMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) (*models.Message, error) {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	// Check access to the channel
	channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// TODO: Check READ_MESSAGES permission
	} else {
		// DM channel - check if requester is participant
		if !isChannelParticipant(channel, requesterID) {
			return nil, ErrNoPermission
		}
	}

	return message, nil
}

// PinMessage pins a message
func (s *MessageService) PinMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	// TODO: Check MANAGE_MESSAGES permission

	message.Pinned = true

	if err := s.repo.Update(ctx, message); err != nil {
		return err
	}

	s.eventBus.Publish("message.pinned", &MessagePinnedEvent{
		MessageID: messageID,
		ChannelID: message.ChannelID,
		PinnedBy:  requesterID,
	})

	return nil
}

// AddReaction adds a reaction to a message
func (s *MessageService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	// TODO: Check ADD_REACTIONS permission

	if err := s.repo.AddReaction(ctx, messageID, userID, emoji); err != nil {
		return err
	}

	s.eventBus.Publish("reaction.added", &ReactionAddedEvent{
		MessageID: messageID,
		ChannelID: message.ChannelID,
		UserID:    userID,
		Emoji:     emoji,
	})

	return nil
}

// RemoveReaction removes a reaction from a message
func (s *MessageService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	if err := s.repo.RemoveReaction(ctx, messageID, userID, emoji); err != nil {
		return err
	}

	s.eventBus.Publish("reaction.removed", &ReactionRemovedEvent{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	})

	return nil
}

// Helpers

func parseMentions(content string) []uuid.UUID {
	// TODO: Parse @mentions and return user IDs
	return nil
}

func isChannelParticipant(channel *models.Channel, userID uuid.UUID) bool {
	for _, p := range channel.Recipients {
		if p == userID {
			return true
		}
	}
	return false
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Events

type MessageCreatedEvent struct {
	Message   *models.Message
	ChannelID uuid.UUID
	ServerID  *uuid.UUID
}

type MessageUpdatedEvent struct {
	Message   *models.Message
	ChannelID uuid.UUID
}

type MessageDeletedEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	AuthorID  uuid.UUID
}

type MessagePinnedEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	PinnedBy  uuid.UUID
}

type ReactionAddedEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}

type ReactionRemovedEvent struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}
