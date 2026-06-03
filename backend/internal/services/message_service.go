package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
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
	repo             MessageRepository
	channelRepo      ChannelRepository
	serverRepo       ServerRepository
	roleRepo         RoleRepository
	userRepo         UserRepository
	quotaService     *QuotaService
	rateLimiter      RateLimiter
	e2eeService      E2EEService
	cache            CacheService
	eventBus         EventBus
	permService      *PermissionService
	threadService    *ThreadService
	mentionService   *MentionService
	federationBridge FederationBridgeSender
	federationServer string // e.g., "hearth.example.com" - needed to construct MXID
}

// FederationBridgeSender is the interface for sending messages to federated servers.
// Implemented by matrixfederation.FederationBridge.
type FederationBridgeSender interface {
	OnHearthMessage(ctx context.Context, messageID, channelID uuid.UUID, senderMXID, content string) error
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

// SetFederationBridge sets the federation bridge for outgoing message federation.
// The serverName should be the canonical homeserver name (e.g., "hearth.example.com").
func (s *MessageService) SetFederationBridge(bridge FederationBridgeSender, serverName string) {
	s.federationBridge = bridge
	s.federationServer = serverName
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
		var err error
		author, err = s.userRepo.GetByID(ctx, authorID)
		if err != nil {
			// Log but don't fail message send - mention processing is best-effort
			log.Printf("Warning: failed to fetch author %s for mention processing: %v", authorID, err)
		}
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

	// HRT-2: Federation bridge - send to remote servers if federated.
	// This is best-effort; failure to enqueue should not fail the user-facing send.
	if s.federationBridge != nil && s.federationServer != "" && channel.ServerID != nil {
		senderName := authorID.String()
		if author != nil {
			senderName = author.Username
		}
		go func(msgID, chanID uuid.UUID, senderID uuid.UUID, content, username string) {
			// Construct MXID from username and federation server name
			senderMXID := "@" + username + ":" + s.federationServer
			if err := s.federationBridge.OnHearthMessage(context.Background(), msgID, chanID, senderMXID, content); err != nil {
				// Log but don't fail - federation is best-effort
				// In production, use proper logging; here we use the standard logger
				log.Printf("⚠️  federation bridge: failed to send message %s: %v", msgID, err)
			}
		}(message.ID, channelID, authorID, content, senderName)
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

var ErrReactionExists = errors.New("reaction already exists")
var ErrReactionNotFound = errors.New("reaction not found")

type Reaction struct {
	MessageID string
	UserID    string
	Emoji     string
}

// ReactionService handles message reactions
type ReactionService struct {
	mu        sync.RWMutex
	reactions map[string][]Reaction // messageID -> reactions
}

// NewReactionService creates a new reaction service
func NewReactionService() *ReactionService {
	return &ReactionService{
		reactions: make(map[string][]Reaction),
	}
}

func (s *ReactionService) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.reactions[messageID] {
		if r.UserID == userID && r.Emoji == emoji {
			return ErrReactionExists
		}
	}
	s.reactions[messageID] = append(s.reactions[messageID], Reaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	})
	return nil
}

func (s *ReactionService) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reactions := s.reactions[messageID]
	for i, r := range reactions {
		if r.UserID == userID && r.Emoji == emoji {
			s.reactions[messageID] = append(reactions[:i], reactions[i+1:]...)
			return nil
		}
	}
	return ErrReactionNotFound
}

func (s *ReactionService) GetReactions(ctx context.Context, messageID string) ([]Reaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reactions[messageID], nil
}

// Common errors
var (
	ErrCannotForwardToSameChannel = errors.New("cannot forward message to the same channel")
)

// ForwardService handles message forwarding business logic
type ForwardService struct {
	messageRepo      MessageRepository
	forwardedMsgRepo ForwardedMessageRepository
	channelRepo      ChannelRepository
	serverRepo       ServerRepository
	permService      *PermissionService
	eventBus         EventBus
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
		OriginalMessageID:    originalMessageID,
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
		OriginalMessageID:    originalMessageID,
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
	OriginalMessageID    uuid.UUID `json:"original_message_id"`
	ForwardedByID        uuid.UUID `json:"forwarded_by_id"`
	DestinationChannelID uuid.UUID `json:"destination_channel_id"`
}

var (
	ErrEmbedNotFound    = errors.New("embed not found")
	ErrInvalidEmbedData = errors.New("invalid embed data")
)

// EmbedRepositoryInterface defines methods needed from EmbedRepository
type EmbedRepositoryInterface interface {
	CreateTemplate(ctx context.Context, template *models.EmbedTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*models.EmbedTemplate, error)
	GetTemplatesByUserID(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.EmbedTemplate) error
	DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error
}

// EmbedService handles embed operations
type EmbedService struct {
	repo       EmbedRepositoryInterface
	httpClient *http.Client
}

// NewEmbedService creates a new embed service
func NewEmbedService(repo EmbedRepositoryInterface) *EmbedService {
	return &EmbedService{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// CreateTemplate creates a new embed template
func (s *EmbedService) CreateTemplate(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	if req.Name == "" {
		return nil, ErrInvalidEmbedData
	}

	template := &models.EmbedTemplate{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		Title:        req.Title,
		Description:  req.Description,
		URL:          req.URL,
		Color:        req.Color,
		AuthorName:   req.AuthorName,
		AuthorURL:    req.AuthorURL,
		AuthorIcon:   req.AuthorIcon,
		FooterText:   req.FooterText,
		FooterIcon:   req.FooterIcon,
		ImageURL:     req.ImageURL,
		ThumbnailURL: req.ThumbnailURL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplates retrieves all embed templates for a user
func (s *EmbedService) GetTemplates(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
	templates, err := s.repo.GetTemplatesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}
	if templates == nil {
		return []models.EmbedTemplate{}, nil
	}
	return templates, nil
}

// GetTemplate retrieves a specific template
func (s *EmbedService) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error) {
	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, ErrTemplateNotFound
	}
	if template.UserID != userID {
		return nil, ErrTemplateNotFound
	}
	return template, nil
}

// UpdateTemplate updates an embed template
func (s *EmbedService) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, ErrTemplateNotFound
	}
	if template.UserID != userID {
		return nil, ErrTemplateNotFound
	}

	// Update fields if provided
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Title != nil {
		template.Title = req.Title
	}
	if req.Description != nil {
		template.Description = req.Description
	}
	if req.URL != nil {
		template.URL = req.URL
	}
	if req.Color != nil {
		template.Color = req.Color
	}
	if req.AuthorName != nil {
		template.AuthorName = req.AuthorName
	}
	if req.AuthorURL != nil {
		template.AuthorURL = req.AuthorURL
	}
	if req.AuthorIcon != nil {
		template.AuthorIcon = req.AuthorIcon
	}
	if req.FooterText != nil {
		template.FooterText = req.FooterText
	}
	if req.FooterIcon != nil {
		template.FooterIcon = req.FooterIcon
	}
	if req.ImageURL != nil {
		template.ImageURL = req.ImageURL
	}
	if req.ThumbnailURL != nil {
		template.ThumbnailURL = req.ThumbnailURL
	}
	template.UpdatedAt = time.Now()

	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// DeleteTemplate deletes an embed template
func (s *EmbedService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	if err := s.repo.DeleteTemplate(ctx, templateID, userID); err != nil {
		return ErrTemplateNotFound
	}
	return nil
}

// FetchURLMetadata fetches OpenGraph/metadata for a URL and returns a link preview response
func (s *EmbedService) FetchURLMetadata(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, ErrInvalidURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, ErrInvalidURL
	}
	req.Header.Set("User-Agent", "HearthEmbedPreview/1.0 (+https://hearth.chat)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ErrUnreachableURL
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, ErrUnreachableURL
	}

	// Limit body read
	limited := io.LimitReader(resp.Body, 5*1024*1024) // 5MB
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, ErrUnreachableURL
	}

	htmlContent := string(body)

	preview := &models.LinkPreviewResponse{
		ID:   uuid.New(),
		URL:  rawURL,
		Type: "website",
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "video") {
		preview.Type = "video"
	} else if strings.Contains(contentType, "audio") {
		preview.Type = "audio"
	} else if strings.Contains(contentType, "image") {
		preview.Type = "image"
	}

	// Extract title from <title> tag as fallback
	if preview.Title == nil {
		if titleMatch := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(htmlContent); len(titleMatch) > 1 {
			title := titleMatch[1]
			preview.Title = &title
		}
	}

	return preview, nil
}

// TemplateToEmbedData converts a template to EmbedData for the frontend
func (s *EmbedService) TemplateToEmbedData(template *models.EmbedTemplate) map[string]interface{} {
	data := make(map[string]interface{})

	if template.Title != nil {
		data["title"] = *template.Title
	}
	if template.Description != nil {
		data["description"] = *template.Description
	}
	if template.URL != nil {
		data["url"] = *template.URL
	}
	if template.Color != nil {
		data["color"] = *template.Color
	}
	if template.AuthorName != nil || template.AuthorURL != nil || template.AuthorIcon != nil {
		author := make(map[string]interface{})
		if template.AuthorName != nil {
			author["name"] = *template.AuthorName
		}
		if template.AuthorURL != nil {
			author["url"] = *template.AuthorURL
		}
		if template.AuthorIcon != nil {
			author["icon"] = *template.AuthorIcon
		}
		data["author"] = author
	}
	if template.FooterText != nil || template.FooterIcon != nil {
		footer := make(map[string]interface{})
		if template.FooterText != nil {
			footer["text"] = *template.FooterText
		}
		if template.FooterIcon != nil {
			footer["icon"] = *template.FooterIcon
		}
		data["footer"] = footer
	}
	if template.ImageURL != nil {
		data["image_url"] = *template.ImageURL
	}
	if template.ThumbnailURL != nil {
		data["thumbnail_url"] = *template.ThumbnailURL
	}

	return data
}

// MentionType represents different types of mentions
type MentionType string

const (
	MentionTypeUser     MentionType = "user"
	MentionTypeRole     MentionType = "role"
	MentionTypeEveryone MentionType = "everyone"
	MentionTypeHere     MentionType = "here"
)

// ParsedMention represents a parsed mention from message content
type ParsedMention struct {
	Type     MentionType
	ID       *uuid.UUID // nil for @everyone/@here
	Username string     // For display purposes
	Raw      string     // The original matched text
	Start    int        // Position in string
	End      int        // End position in string
}

// MentionParseResult contains all parsed mentions from a message
type MentionParseResult struct {
	UserMentions    []uuid.UUID
	RoleMentions    []uuid.UUID
	MentionEveryone bool
	MentionHere     bool
	AllMentions     []ParsedMention
}

// MentionService handles mention parsing and notification creation
type MentionService struct {
	userRepo         UserRepository
	roleRepo         RoleRepository
	memberRepo       ServerRepository
	notificationRepo NotificationRepository
	readStateRepo    ReadStateRepository
	mentionRepo      MentionRepository
	eventBus         EventBus
}

// NewMentionService creates a new mention service
func NewMentionService(
	userRepo UserRepository,
	roleRepo RoleRepository,
	memberRepo ServerRepository,
	notificationRepo NotificationRepository,
	readStateRepo ReadStateRepository,
	eventBus EventBus,
) *MentionService {
	return &MentionService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		memberRepo:       memberRepo,
		notificationRepo: notificationRepo,
		readStateRepo:    readStateRepo,
		eventBus:         eventBus,
	}
}

// NewMentionServiceWithRepo creates a new mention service with mention repository
func NewMentionServiceWithRepo(
	userRepo UserRepository,
	roleRepo RoleRepository,
	memberRepo ServerRepository,
	notificationRepo NotificationRepository,
	readStateRepo ReadStateRepository,
	mentionRepo MentionRepository,
	eventBus EventBus,
) *MentionService {
	return &MentionService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		memberRepo:       memberRepo,
		notificationRepo: notificationRepo,
		readStateRepo:    readStateRepo,
		mentionRepo:      mentionRepo,
		eventBus:         eventBus,
	}
}

// SetMentionRepo sets the mention repository (allows adding after creation)
func (s *MentionService) SetMentionRepo(repo MentionRepository) {
	s.mentionRepo = repo
}

var (
	// Matches @username (alphanumeric + underscore, 2-32 chars)
	userMentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]{2,32})`)
	// Matches <@user_id> format (Discord-style)
	userIDMentionRegex = regexp.MustCompile(`<@!?([a-f0-9-]{36})>`)
	// Matches <@&role_id> format (Discord-style role mention)
	roleMentionRegex = regexp.MustCompile(`<@&([a-f0-9-]{36})>`)
	// Matches @everyone
	everyoneMentionRegex = regexp.MustCompile(`@everyone\b`)
	// Matches @here
	hereMentionRegex = regexp.MustCompile(`@here\b`)
)

// ParseMentions extracts all mentions from message content
func (s *MentionService) ParseMentions(ctx context.Context, content string, serverID *uuid.UUID) (*MentionParseResult, error) {
	result := &MentionParseResult{
		UserMentions: make([]uuid.UUID, 0),
		RoleMentions: make([]uuid.UUID, 0),
		AllMentions:  make([]ParsedMention, 0),
	}

	seenUsers := make(map[uuid.UUID]bool)
	seenRoles := make(map[uuid.UUID]bool)

	// Parse @everyone
	if everyoneMentionRegex.MatchString(content) {
		result.MentionEveryone = true
		matches := everyoneMentionRegex.FindAllStringIndex(content, -1)
		for _, match := range matches {
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeEveryone,
				Raw:   "@everyone",
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse @here
	if hereMentionRegex.MatchString(content) {
		result.MentionHere = true
		matches := hereMentionRegex.FindAllStringIndex(content, -1)
		for _, match := range matches {
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeHere,
				Raw:   "@here",
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse <@user_id> format
	userIDMatches := userIDMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range userIDMatches {
		idStr := content[match[2]:match[3]]
		if id, err := uuid.Parse(idStr); err == nil {
			if !seenUsers[id] {
				seenUsers[id] = true
				result.UserMentions = append(result.UserMentions, id)
			}
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeUser,
				ID:    &id,
				Raw:   content[match[0]:match[1]],
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse <@&role_id> format
	roleMatches := roleMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range roleMatches {
		idStr := content[match[2]:match[3]]
		if id, err := uuid.Parse(idStr); err == nil {
			if !seenRoles[id] {
				seenRoles[id] = true
				result.RoleMentions = append(result.RoleMentions, id)
			}
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeRole,
				ID:    &id,
				Raw:   content[match[0]:match[1]],
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse @username format (resolve to user IDs)
	userMatches := userMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range userMatches {
		username := content[match[2]:match[3]]
		rawMention := content[match[0]:match[1]]

		// Skip if it's @everyone or @here
		if username == "everyone" || username == "here" {
			continue
		}

		// Try to resolve username to user
		user, err := s.userRepo.GetByUsername(ctx, username)
		if err == nil && user != nil {
			if !seenUsers[user.ID] {
				seenUsers[user.ID] = true
				result.UserMentions = append(result.UserMentions, user.ID)
			}
			id := user.ID
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:     MentionTypeUser,
				ID:       &id,
				Username: username,
				Raw:      rawMention,
				Start:    match[0],
				End:      match[1],
			})
		}
	}

	return result, nil
}

// ParseMentionsSimple is a simpler version that just extracts user IDs
// Used by the message service for the Mentions field
func ParseMentionsSimple(content string) []uuid.UUID {
	mentions := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)

	// Parse <@user_id> format
	matches := userIDMentionRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if id, err := uuid.Parse(match[1]); err == nil {
				if !seen[id] {
					seen[id] = true
					mentions = append(mentions, id)
				}
			}
		}
	}

	return mentions
}

// ProcessMessageMentions processes mentions in a message and creates notifications
func (s *MentionService) ProcessMessageMentions(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	serverID *uuid.UUID,
) error {
	if message.EncryptedContent != "" {
		// Skip encrypted messages - can't parse mentions
		return nil
	}

	result, err := s.ParseMentions(ctx, message.Content, serverID)
	if err != nil {
		return err
	}

	// Update message with parsed mentions
	message.Mentions = result.UserMentions
	message.MentionRoles = result.RoleMentions
	message.MentionEveryone = result.MentionEveryone

	// Create notifications and mention records for user mentions
	for _, userID := range result.UserMentions {
		if userID == message.AuthorID {
			continue // Don't notify yourself
		}
		// SECURITY: Validate user is a server member before sending mention notification
		// This prevents users from mentioning arbitrary people who aren't in the server
		if serverID != nil && s.memberRepo != nil {
			member, err := s.memberRepo.GetMember(ctx, *serverID, userID)
			if err != nil || member == nil {
				continue // User is not a server member, skip notification
			}
		}
		if err := s.createMentionNotification(ctx, message, author, userID, serverID); err != nil {
			continue // Log but don't fail
		}
		// Create mention record
		if s.mentionRepo != nil {
			mention := &models.Mention{
				UserID:      userID,
				MessageID:   message.ID,
				MentionedBy: message.AuthorID,
				ChannelID:   message.ChannelID,
				GuildID:     serverID,
				MentionType: models.MentionKindUser,
			}
			_ = s.mentionRepo.Create(ctx, mention)
		}
		// Increment mention count for read state
		if err := s.readStateRepo.IncrementMentionCount(ctx, userID, message.ChannelID); err != nil {
			continue
		}
	}

	// Handle @everyone/@here mentions in servers
	if serverID != nil && (result.MentionEveryone || result.MentionHere) {
		mentionType := models.MentionKindEveryone
		if result.MentionHere {
			mentionType = models.MentionKindHere
		}
		members, err := s.getAllMembersWithPagination(ctx, *serverID)
		if err == nil {
			for _, member := range members {
				if member.UserID == message.AuthorID {
					continue
				}
				// For @here, we'd check online status - simplified for now
				if err := s.createMentionNotification(ctx, message, author, member.UserID, serverID); err != nil {
					continue
				}
				// Create mention record
				if s.mentionRepo != nil {
					mention := &models.Mention{
						UserID:      member.UserID,
						MessageID:   message.ID,
						MentionedBy: message.AuthorID,
						ChannelID:   message.ChannelID,
						GuildID:     serverID,
						MentionType: mentionType,
					}
					_ = s.mentionRepo.Create(ctx, mention)
				}
				if err := s.readStateRepo.IncrementMentionCount(ctx, member.UserID, message.ChannelID); err != nil {
					continue
				}
			}
		}
	}

	// Handle role mentions
	if serverID != nil && len(result.RoleMentions) > 0 {
		for _, roleID := range result.RoleMentions {
			// Get members with this role
			membersWithRole, err := s.memberRepo.GetMembersWithRole(ctx, *serverID, roleID)
			if err != nil {
				continue
			}
			for _, member := range membersWithRole {
				if member.UserID == message.AuthorID {
					continue
				}
				if err := s.createMentionNotification(ctx, message, author, member.UserID, serverID); err != nil {
					continue
				}
				// Create mention record for role mention
				if s.mentionRepo != nil {
					mention := &models.Mention{
						UserID:          member.UserID,
						MessageID:       message.ID,
						MentionedBy:     message.AuthorID,
						ChannelID:       message.ChannelID,
						GuildID:         serverID,
						MentionType:     models.MentionKindRole,
						MentionedRoleID: &roleID,
					}
					_ = s.mentionRepo.Create(ctx, mention)
				}
				if err := s.readStateRepo.IncrementMentionCount(ctx, member.UserID, message.ChannelID); err != nil {
					continue
				}
			}
		}
	}

	return nil
}

// createMentionNotification creates a notification for a mention
func (s *MentionService) createMentionNotification(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	recipientID uuid.UUID,
	serverID *uuid.UUID,
) error {
	// Truncate content for notification body
	body := message.Content
	if len(body) > 200 {
		body = body[:197] + "..."
	}

	notification := &models.Notification{
		UserID:    recipientID,
		Type:      models.NotificationTypeMention,
		Title:     author.Username + " mentioned you",
		Body:      body,
		ActorID:   &message.AuthorID,
		ServerID:  serverID,
		ChannelID: &message.ChannelID,
		MessageID: &message.ID,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return err
	}

	// Emit event for real-time delivery
	s.eventBus.Publish("notification.created", &NotificationCreatedEvent{
		Notification: notification,
	})

	return nil
}

// ProcessReplyMention creates a notification when someone replies to a message
func (s *MentionService) ProcessReplyMention(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	replyToAuthorID uuid.UUID,
	serverID *uuid.UUID,
) error {
	if replyToAuthorID == message.AuthorID {
		return nil // Don't notify yourself
	}

	body := message.Content
	if len(body) > 200 {
		body = body[:197] + "..."
	}

	notification := &models.Notification{
		UserID:    replyToAuthorID,
		Type:      models.NotificationTypeReply,
		Title:     author.Username + " replied to you",
		Body:      body,
		ActorID:   &message.AuthorID,
		ServerID:  serverID,
		ChannelID: &message.ChannelID,
		MessageID: &message.ID,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return err
	}

	// Emit event for real-time delivery
	s.eventBus.Publish("notification.created", &NotificationCreatedEvent{
		Notification: notification,
	})

	// Increment mention count
	return s.readStateRepo.IncrementMentionCount(ctx, replyToAuthorID, message.ChannelID)
}

// FormatMentionContent converts @username mentions to <@user_id> format for storage
func (s *MentionService) FormatMentionContent(ctx context.Context, content string) string {
	// Find all @username mentions and replace with <@user_id>
	result := userMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		username := strings.TrimPrefix(match, "@")
		if username == "everyone" || username == "here" {
			return match // Keep as-is
		}
		user, err := s.userRepo.GetByUsername(ctx, username)
		if err == nil && user != nil {
			return "<@" + user.ID.String() + ">"
		}
		return match // Keep original if user not found
	})
	return result
}

// RenderMentionContent converts <@user_id> format back to displayable format
func RenderMentionContent(content string, usernames map[uuid.UUID]string) string {
	result := userIDMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract ID from <@id> or <@!id>
		idStr := strings.TrimPrefix(match, "<@")
		idStr = strings.TrimPrefix(idStr, "!")
		idStr = strings.TrimSuffix(idStr, ">")

		if id, err := uuid.Parse(idStr); err == nil {
			if username, ok := usernames[id]; ok {
				return "@" + username
			}
		}
		return match
	})
	return result
}

func (s *MentionService) getAllMembersWithPagination(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	const batchSize = 100
	var allMembers []*models.Member
	var cursor *models.MemberCursor

	for {
		result, err := s.memberRepo.GetMembersPaginated(ctx, serverID, cursor, batchSize)
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, result.Members...)

		if !result.HasMore {
			break
		}

		nextCursor, err := models.DecodeMemberCursor(result.NextCursor)
		if err != nil {
			return nil, err
		}
		cursor = nextCursor
	}

	return allMembers, nil
}
