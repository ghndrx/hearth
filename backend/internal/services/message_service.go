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
	CountRepliesTo(ctx context.Context, messageID uuid.UUID) (int, error)

	// Reactions
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	GetReactions(ctx context.Context, messageID uuid.UUID) ([]*models.Reaction, error)
	GetReactionUsers(ctx context.Context, messageID uuid.UUID, emoji string, limit int) ([]*models.ReactionUser, error)
	GetUserReactions(ctx context.Context, messageID, userID uuid.UUID) ([]string, error)

	RemoveAllReactions(ctx context.Context, messageID uuid.UUID) error
	GetTopReactions(ctx context.Context, limit int) ([]*models.Reaction, error)

	// Bulk operations
	DeleteByChannel(ctx context.Context, channelID uuid.UUID) error
	DeleteByAuthor(ctx context.Context, channelID, authorID uuid.UUID, since time.Time) (int, error)
	BulkDeleteMessages(ctx context.Context, messageIDs []uuid.UUID) error
}

// ForwardedMessageRepository defines the interface for forwarded message data access
type ForwardedMessageRepository interface {
	Create(ctx context.Context, message *models.ForwardedMessage) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ForwardedMessage, error)
	GetByOriginalMessageID(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error)
	GetByDestinationChannelID(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error)
}

// AutoThreadThreshold is the number of replies to a message before
// a thread is automatically created.
const AutoThreadThreshold = 3

// MessageService handles message-related business logic
type MessageService struct {
	repo           MessageRepository
	channelRepo    ChannelRepository
	serverRepo     ServerRepository
	roleRepo       RoleRepository
	userRepo       UserRepository
	quotaService   *QuotaService
	rateLimiter    RateLimiter
	e2eeService    E2EEService
	cache          CacheService
	eventBus       EventBus
	permService    *PermissionService
	threadService  *ThreadService
	mentionService *MentionService
}

// NewMessageService creates a new message service
func NewMessageService(
	repo MessageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	roleRepo RoleRepository,
	userRepo UserRepository,
	quotaService *QuotaService,
	rateLimiter RateLimiter,
	e2eeService E2EEService,
	cache CacheService,
	eventBus EventBus,
	permService *PermissionService,
) *MessageService {
	return &MessageService{
		repo:         repo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		roleRepo:     roleRepo,
		userRepo:     userRepo,
		quotaService: quotaService,
		rateLimiter:  rateLimiter,
		e2eeService:  e2eeService,
		cache:        cache,
		eventBus:     eventBus,
		permService:  permService,
	}
}

// SetThreadService sets the thread service for auto-thread creation
func (s *MessageService) SetThreadService(threadService *ThreadService) {
	s.threadService = threadService
}

// SetMentionService sets the mention service for processing mentions
func (s *MessageService) SetMentionService(mentionService *MentionService) {
	s.mentionService = mentionService
}

// getMemberPermissions computes effective permissions for a member in a server.
// Returns PermissionAll for server owners.
func (s *MessageService) getMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	// Get server to check ownership
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server == nil {
		return 0, ErrServerNotFound
	}

	// Owner has all permissions
	if server.OwnerID == userID {
		return models.PermissionAll | models.PermAdministrator, nil
	}

	// If we have a role repo, use it to get permissions
	if s.roleRepo != nil {
		perms, err := s.roleRepo.GetMemberPermissions(ctx, serverID, userID)
		if err != nil {
			return 0, err
		}

		// Get default (@everyone) role permissions
		defaultRole, err := s.roleRepo.GetDefaultRole(ctx, serverID)
		if err == nil && defaultRole != nil {
			perms |= defaultRole.Permissions
		}

		// Administrator grants all permissions
		if perms&models.PermAdministrator != 0 {
			return models.PermissionAll | models.PermAdministrator, nil
		}

		return perms, nil
	}

	// Fallback: if no roleRepo, assume default permissions
	return models.DefaultPermissions, nil
}

// hasPermission checks if a user has a specific permission in a server channel.
// For DM channels, this always returns true (permissions don't apply).
func (s *MessageService) hasPermission(ctx context.Context, channel *models.Channel, userID uuid.UUID, permission int64) (bool, error) {
	// DM channels don't have permission restrictions (beyond being a participant)
	if channel.ServerID == nil {
		return true, nil
	}

	perms, err := s.getMemberPermissions(ctx, *channel.ServerID, userID)
	if err != nil {
		return false, err
	}

	return models.HasPermission(perms, permission), nil
}

// SendMessage sends a message to a channel
func (s *MessageService) SendMessage(ctx context.Context, authorID uuid.UUID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error) {
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

		// Check SEND_MESSAGES permission via permService if available
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, authorID, models.PermSendMessages); err != nil {
				return nil, err
			}
		} else {
			// Fallback to local permission check
			hasPerm, err := s.hasPermission(ctx, channel, authorID, models.PermSendMessages)
			if err != nil {
				return nil, err
			}
			if !hasPerm {
				return nil, ErrMissingSendMessages
			}
		}
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

	// Check if content, attachments, or sticker exists
	if len(content) == 0 && len(attachments) == 0 && stickerID == nil {
		return nil, ErrEmptyMessage
	}

	// Check rate limit (skip for DMs in some cases)
	if s.rateLimiter != nil && channel.Type != models.ChannelTypeDM {
		if err := s.rateLimiter.Check(ctx, authorID, channelID); err != nil {
			return nil, ErrRateLimited
		}
	}

	// Check slowmode
	if s.rateLimiter != nil && channel.Slowmode > 0 {
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
		AuthorID:    authorID,
		Content:     content,
		Type:        models.MessageTypeDefault,
		Attachments: msgAttachments,
		ReplyToID:   replyTo,
		StickerID:   stickerID,
		CreatedAt:   time.Now(),
	}

	// Set message type based on content
	if stickerID != nil {
		message.Type = models.MessageTypeSticker
	} else if replyTo != nil {
		message.Type = models.MessageTypeReply
	}

	// Handle E2EE for DM channels
	isEncrypted := false
	if channel.Type == models.ChannelTypeDM || channel.Type == models.ChannelTypeGroupDM {
		if channel.E2EEEnabled && s.e2eeService != nil {
			// Content should already be encrypted by client
			// We just verify the format
			if !s.e2eeService.ValidateEncryptedPayload(content) {
				return nil, ErrNoPermission
			}
			message.EncryptedContent = content
			message.Content = "" // Clear plaintext
			isEncrypted = true
		}
	} else if channel.E2EEEnabled && s.e2eeService != nil {
		// Server channel with E2EE enabled
		if !s.e2eeService.ValidateEncryptedPayload(content) {
			return nil, ErrNoPermission
		}
		message.EncryptedContent = content
		message.Content = ""
		isEncrypted = true
	}

	// Fetch author first so we can process mentions with proper server membership validation
	var author *models.User
	if s.userRepo != nil {
		author, _ = s.userRepo.GetByID(ctx, authorID)
	}

	// Process mentions with security validation (if mentionService is available)
	// This validates that mentioned users are actually server members before sending notifications
	if !isEncrypted && s.mentionService != nil && author != nil {
		if err := s.mentionService.ProcessMessageMentions(ctx, message, author, serverID); err != nil {
			// Log but don't fail message send - mention processing is best-effort
			// Fall back to simple mention parsing for the Mentions field
			message.Mentions = parseMentions(content)
		}
	} else if !isEncrypted {
		// Fallback: simple parsing without notification processing
		message.Mentions = parseMentions(content)
	}

	if err := s.repo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Populate author for the response and WebSocket event
	if s.userRepo != nil && author != nil {
		pubUser := author.ToPublic()
		message.Author = &pubUser
	}

	// Update channel's last message
	_ = s.channelRepo.UpdateLastMessage(ctx, channelID, message.ID, message.CreatedAt)

	// Emit event
	s.eventBus.Publish("message.created", &MessageCreatedEvent{
		Message:   message,
		ChannelID: channelID,
		ServerID:  channel.ServerID,
	})

	// Auto-create thread when a message gets 3+ replies
	if replyTo != nil && s.threadService != nil {
		go func() {
			// G118 fix: Use a detached context with timeout instead of request context.
			// Request context will be cancelled when HTTP request completes, but this
			// background operation should continue independently with bounded execution time.
			threadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.maybeAutoCreateThread(threadCtx, channelID, *replyTo, authorID)
		}()
	}

	return message, nil
}

// SendWebhookMessageRequest represents a request to send a message via webhook
type SendWebhookMessageRequest struct {
	WebhookID       uuid.UUID
	ChannelID       uuid.UUID
	Content         string
	Username        *string // Optional webhook name override
	AvatarURL       *string // Optional avatar override
	TTS             bool
	Embeds          []models.Embed
	AllowedMentions *models.WebhookMessage `json:"allowed_mentions,omitempty"`
	ThreadName      string
}

// SendMessageForWebhook sends a message on behalf of a webhook.
// This method skips permission checks since the webhook is already validated.
// It uses the webhook ID as the AuthorID and supports username/avatar overrides.
func (s *MessageService) SendMessageForWebhook(ctx context.Context, req SendWebhookMessageRequest) (*models.Message, error) {
	// Get channel
	channel, err := s.channelRepo.GetByID(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Get quota limits (use webhook ID as the "user" for quota purposes)
	var serverID *uuid.UUID
	if channel.ServerID != nil {
		serverID = channel.ServerID
	}
	limits, err := s.quotaService.GetEffectiveLimits(ctx, req.WebhookID, serverID)
	if err != nil {
		return nil, err
	}

	// Check message length
	if limits.MaxMessageLength > 0 && len(req.Content) > limits.MaxMessageLength {
		return nil, ErrMessageTooLong
	}

	// Check if content or embeds exist
	if req.Content == "" && len(req.Embeds) == 0 {
		return nil, ErrEmptyMessage
	}

	// Create message with webhook ID as author
	message := &models.Message{
		ID:        uuid.New(),
		ChannelID: req.ChannelID,
		AuthorID:  req.WebhookID, // Webhook ID as author
		Content:   req.Content,
		Type:      models.MessageTypeDefault,
		TTS:       req.TTS,
		Embeds:    req.Embeds,
		CreatedAt: time.Now(),
	}

	// Parse mentions from content
	message.Mentions = parseMentions(req.Content)

	// Persist to database
	if err := s.repo.Create(ctx, message); err != nil {
		return nil, err
	}

	// Set author display (webhook name + overrides)
	username := "Webhook"
	if req.Username != nil && *req.Username != "" {
		username = *req.Username
	}

	message.Author = &models.PublicUser{
		ID:        req.WebhookID,
		Username:  username,
		AvatarURL: req.AvatarURL,
	}

	// Update channel's last message
	_ = s.channelRepo.UpdateLastMessage(ctx, req.ChannelID, message.ID, message.CreatedAt)

	// Emit event for WebSocket delivery
	s.eventBus.Publish("message.created", &MessageCreatedEvent{
		Message:   message,
		ChannelID: req.ChannelID,
		ServerID:  channel.ServerID,
	})

	return message, nil
}

// EditMessage edits an existing message
func (s *MessageService) EditMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID, newContent string) (*models.Message, error) {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	// Author can always edit their own messages
	if message.AuthorID != requesterID {
		// For others' messages, need MANAGE_MESSAGES permission
		channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
		if err != nil {
			return nil, err
		}
		if channel == nil {
			return nil, ErrChannelNotFound
		}

		// Only allow editing others' messages in server channels with MANAGE_MESSAGES
		// Note: In most chat systems, you can't actually edit others' messages,
		// but we check the permission anyway for completeness
		if channel.ServerID != nil {
			// Use permService if available, otherwise fall back to local check
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageMessages); err != nil {
					return nil, err
				}
			} else {
				hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
				if err != nil {
					return nil, err
				}
				if !hasPerm {
					return nil, ErrNotMessageAuthor
				}
			}
		} else {
			// DM channel - cannot edit others' messages
			return nil, ErrNotMessageAuthor
		}
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
		channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
		if err != nil {
			return err
		}
		if channel == nil {
			return ErrChannelNotFound
		}

		if channel.ServerID != nil {
			// Server channel - check MANAGE_MESSAGES permission via permService if available
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageMessages); err != nil {
					return err
				}
			} else {
				hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
				if err != nil {
					return err
				}
				if !hasPerm {
					return ErrMissingManageMessages
				}
			}
		} else {
			// DM channel - cannot delete others' messages
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

const (
	MaxBulkDeleteCount = 100
	BulkDeleteAgeLimit = 14 * 24 * time.Hour
)

func (s *MessageService) BulkDeleteMessages(ctx context.Context, channelID uuid.UUID, messageIDs []uuid.UUID, requesterID uuid.UUID) error {
	if len(messageIDs) > MaxBulkDeleteCount {
		return ErrBulkDeleteLimit
	}

	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}

		// Server channel - check MANAGE_MESSAGES permission via permService if available
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageMessages); err != nil {
				return err
			}
		} else {
			hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
			if err != nil {
				return err
			}
			if !hasPerm {
				return ErrMissingManageMessages
			}
		}
	} else {
		return ErrNoPermission
	}

	cutoffTime := time.Now().Add(-BulkDeleteAgeLimit)

	var validIDs []uuid.UUID
	for _, id := range messageIDs {
		msg, err := s.repo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if msg == nil {
			continue
		}
		if msg.CreatedAt.Before(cutoffTime) {
			continue
		}
		if msg.ChannelID != channelID {
			continue
		}
		validIDs = append(validIDs, id)
	}

	if len(validIDs) == 0 {
		return nil
	}

	if err := s.repo.BulkDeleteMessages(ctx, validIDs); err != nil {
		return err
	}

	for _, id := range validIDs {
		s.eventBus.Publish("message.deleted", &MessageDeletedEvent{
			MessageID: id,
			ChannelID: channelID,
		})
	}

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
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, ErrNotServerMember
		}

		// Check READ_MESSAGE_HISTORY permission
		hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermReadMessageHistory)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, ErrMissingReadMessages
		}
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

		// Check READ_MESSAGE_HISTORY permission
		hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermReadMessageHistory)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, ErrMissingReadMessages
		}
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

	// Check access to the channel
	channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}

		// Check MANAGE_MESSAGES permission
		hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
		if err != nil {
			return err
		}
		if !hasPerm {
			return ErrMissingManageMessages
		}
	} else {
		// DM channel - check if requester is participant
		if !isChannelParticipant(channel, requesterID) {
			return ErrNoPermission
		}
	}

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

// UnpinMessage unpins a message
func (s *MessageService) UnpinMessage(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	// Check access to the channel
	channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}

		// Check MANAGE_MESSAGES permission
		hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
		if err != nil {
			return err
		}
		if !hasPerm {
			return ErrMissingManageMessages
		}
	} else {
		// DM channel - check if requester is participant
		if !isChannelParticipant(channel, requesterID) {
			return ErrNoPermission
		}
	}

	if !message.Pinned {
		return nil // Already unpinned, no-op
	}

	message.Pinned = false

	if err := s.repo.Update(ctx, message); err != nil {
		return err
	}

	s.eventBus.Publish("message.unpinned", &MessageUnpinnedEvent{
		MessageID:  messageID,
		ChannelID:  message.ChannelID,
		UnpinnedBy: requesterID,
	})

	return nil
}

// GetPinnedMessages retrieves all pinned messages in a channel
func (s *MessageService) GetPinnedMessages(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Message, error) {
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

		// Check READ_MESSAGE_HISTORY permission
		hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermReadMessageHistory)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, ErrMissingReadMessages
		}
	} else {
		// DM channel - check if requester is participant
		if !isChannelParticipant(channel, requesterID) {
			return nil, ErrNoPermission
		}
	}

	return s.repo.GetPinnedMessages(ctx, channelID)
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

	// Check access to the channel
	channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}

		// Check ADD_REACTIONS permission via permService if available
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermAddReactions); err != nil {
				return err
			}
		} else {
			hasPerm, err := s.hasPermission(ctx, channel, userID, models.PermAddReactions)
			if err != nil {
				return err
			}
			if !hasPerm {
				return ErrMissingAddReactions
			}
		}
	} else {
		// DM channel - check if user is participant
		if !isChannelParticipant(channel, userID) {
			return ErrNoPermission
		}
	}

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
func (s *MessageService) RemoveReaction(ctx context.Context, messageID, requesterID uuid.UUID, emoji string, targetUserID *uuid.UUID) error {
	// Determine whose reaction we're removing
	reactionOwnerID := requesterID
	if targetUserID != nil {
		reactionOwnerID = *targetUserID
	}

	// Always fetch the message to get ChannelID for the event
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	// If removing someone else's reaction, need to check permissions
	if reactionOwnerID != requesterID {
		channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
		if err != nil {
			return err
		}
		if channel == nil {
			return ErrChannelNotFound
		}

		if channel.ServerID != nil {
			// Check MANAGE_MESSAGES permission for removing others' reactions via permService if available
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageMessages); err != nil {
					return err
				}
			} else {
				hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
				if err != nil {
					return err
				}
				if !hasPerm {
					return ErrMissingManageMessages
				}
			}
		} else {
			// DM channel - cannot remove others' reactions
			return ErrNoPermission
		}
	}

	if err := s.repo.RemoveReaction(ctx, messageID, reactionOwnerID, emoji); err != nil {
		return err
	}

	s.eventBus.Publish("reaction.removed", &ReactionRemovedEvent{
		MessageID: messageID,
		ChannelID: message.ChannelID,
		UserID:    reactionOwnerID,
		Emoji:     emoji,
	})

	return nil
}

// RemoveOwnReaction removes the requester's own reaction (backward compatibility)
func (s *MessageService) RemoveOwnReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	return s.RemoveReaction(ctx, messageID, userID, emoji, nil)
}

// RemoveAllReactions removes all reactions from a message (requires MANAGE_MESSAGES)
func (s *MessageService) RemoveAllReactions(ctx context.Context, messageID, requesterID uuid.UUID) error {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, message.ChannelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageMessages); err != nil {
				return err
			}
		} else {
			hasPerm, err := s.hasPermission(ctx, channel, requesterID, models.PermManageMessages)
			if err != nil {
				return err
			}
			if !hasPerm {
				return ErrMissingManageMessages
			}
		}
	} else {
		return ErrNoPermission
	}

	if err := s.repo.RemoveAllReactions(ctx, messageID); err != nil {
		return err
	}

	s.eventBus.Publish("reaction.removed_all", &ReactionRemovedAllEvent{
		MessageID: messageID,
		ChannelID: message.ChannelID,
	})

	return nil
}

// GetReactions returns aggregated reactions for a message
func (s *MessageService) GetReactions(ctx context.Context, messageID uuid.UUID, requesterID uuid.UUID) ([]*models.Reaction, error) {
	// First get the message to check access
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
	} else {
		if !isChannelParticipant(channel, requesterID) {
			return nil, ErrNoPermission
		}
	}

	reactions, err := s.repo.GetReactions(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Check which reactions the requester has made
	userEmojis, _ := s.repo.GetUserReactions(ctx, messageID, requesterID)
	userReacted := make(map[string]bool)
	for _, emoji := range userEmojis {
		userReacted[emoji] = true
	}

	// Set the Me flag
	for _, r := range reactions {
		r.Me = userReacted[r.Emoji]
	}

	return reactions, nil
}

// GetReactionUsers returns users who reacted with a specific emoji
func (s *MessageService) GetReactionUsers(ctx context.Context, messageID uuid.UUID, emoji string, requesterID uuid.UUID, limit int) ([]*models.ReactionUser, error) {
	// First get the message to check access
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
	} else {
		if !isChannelParticipant(channel, requesterID) {
			return nil, ErrNoPermission
		}
	}

	if limit <= 0 || limit > 100 {
		limit = 25
	}

	return s.repo.GetReactionUsers(ctx, messageID, emoji, limit)
}

// GetTopReactions returns the most frequently used reactions across all messages
func (s *MessageService) GetTopReactions(ctx context.Context, limit int) ([]*models.Reaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	return s.repo.GetTopReactions(ctx, limit)
}

// maybeAutoCreateThread creates a thread when a message receives enough replies
func (s *MessageService) maybeAutoCreateThread(ctx context.Context, channelID, parentMessageID, authorID uuid.UUID) {
	count, err := s.repo.CountRepliesTo(ctx, parentMessageID)
	if err != nil || count < AutoThreadThreshold {
		return
	}

	parentMsg, err := s.repo.GetByID(ctx, parentMessageID)
	if err != nil || parentMsg == nil {
		return
	}

	// Generate thread name from parent message content
	name := parentMsg.Content
	if len(name) > 50 {
		name = name[:50] + "..."
	}
	if name == "" {
		name = "Thread"
	}

	_, _ = s.threadService.CreateThread(ctx, channelID, parentMsg.AuthorID, name, nil, &parentMessageID, nil)
}

// Helpers

func parseMentions(content string) []uuid.UUID {
	return ParseMentionsSimple(content)
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

type MessageUnpinnedEvent struct {
	MessageID  uuid.UUID
	ChannelID  uuid.UUID
	UnpinnedBy uuid.UUID
}

type ReactionAddedEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}

type ReactionRemovedEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
}

type ReactionRemovedAllEvent struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
}
