package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// MessageServiceForWebhook defines the interface for message service operations used by WebhookService
type MessageServiceForWebhook interface {
	SendMessageForWebhook(ctx context.Context, req SendWebhookMessageRequest) (*models.Message, error)
}

// CacheServiceForWebhook defines the cache operations needed for webhook rate limiting
type CacheServiceForWebhook interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// WebhookService handles webhook-related business logic
type WebhookService struct {
	webhookRepo         WebhookRepository
	webhookDeliveryRepo WebhookDeliveryRepository
	channelRepo         ChannelRepository
	serverRepo          ServerRepository
	permService         *PermissionService
	messageService      MessageServiceForWebhook
	eventBus            EventBus
	cache               CacheServiceForWebhook
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	webhookRepo WebhookRepository,
	webhookDeliveryRepo WebhookDeliveryRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	messageService MessageServiceForWebhook,
	eventBus EventBus,
	cache CacheServiceForWebhook,
) *WebhookService {
	return &WebhookService{
		webhookRepo:         webhookRepo,
		webhookDeliveryRepo: webhookDeliveryRepo,
		channelRepo:         channelRepo,
		serverRepo:          serverRepo,
		permService:         permService,
		messageService:      messageService,
		eventBus:            eventBus,
		cache:               cache,
	}
}

// Rate limit constants for webhooks
const (
	// MaxRequestsPerMinute is the maximum webhook execution requests per minute per webhook
	MaxRequestsPerMinute = 30
	// RateLimitWindow is the rate limit window duration
	RateLimitWindow = time.Minute
	// MaxRetries is the maximum number of retry attempts for failed webhook deliveries
	MaxRetries = 3
)

// CreateWebhookRequest represents a webhook creation request
type CreateWebhookRequest struct {
	ChannelID uuid.UUID
	ServerID  uuid.UUID
	CreatorID uuid.UUID
	Name      string
	Avatar    *string
}

// CreateWebhook creates a new webhook for a channel
func (s *WebhookService) CreateWebhook(ctx context.Context, req *CreateWebhookRequest) (*models.Webhook, error) {
	// Validate channel exists and user has access
	channel, err := s.channelRepo.GetByID(ctx, req.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// Verify creator is a member of the server (for server channels)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, req.CreatorID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		req.ServerID = *channel.ServerID

		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, req.CreatorID, models.PermManageWebhooks); err != nil {
				return nil, err
			}
		}
	}

	// Validate name
	if req.Name == "" || len(req.Name) > 80 {
		return nil, ErrWebhookNameTooLong
	}

	// Check webhook limit per channel (max 10)
	count, err := s.webhookRepo.CountByChannelID(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, ErrTooManyWebhooks
	}

	// Generate unique token
	token, err := generateWebhookToken()
	if err != nil {
		return nil, err
	}

	webhook := &models.Webhook{
		ID:        uuid.New(),
		Type:      models.WebhookTypeIncoming,
		ServerID:  &req.ServerID,
		ChannelID: req.ChannelID,
		CreatorID: &req.CreatorID,
		Name:      req.Name,
		Avatar:    req.Avatar,
		Token:     token,
		CreatedAt: time.Now(),
	}

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, err
	}

	s.eventBus.Publish("webhook.created", &WebhookCreatedEvent{
		WebhookID: webhook.ID,
		ChannelID: webhook.ChannelID,
		ServerID:  req.ServerID,
		CreatorID: req.CreatorID,
	})

	return webhook, nil
}

// GetWebhook retrieves a webhook by ID
func (s *WebhookService) GetWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Webhook, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Verify requester has access to the channel/server
	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	return webhook, nil
}

// GetChannelWebhooks retrieves all webhooks for a channel
func (s *WebhookService) GetChannelWebhooks(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	// Validate channel exists and user has access
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	return s.webhookRepo.GetByChannelID(ctx, channelID)
}

// GetServerWebhooks retrieves all webhooks for a server
func (s *WebhookService) GetServerWebhooks(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Webhook, error) {
	// Verify requester is a member
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Require MANAGE_WEBHOOKS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageWebhooks); err != nil {
			return nil, err
		}
	}

	return s.webhookRepo.GetByServerID(ctx, serverID)
}

// UpdateWebhookRequest represents a webhook update request
type UpdateWebhookRequest struct {
	Name      *string
	Avatar    *string
	ChannelID *uuid.UUID
}

// UpdateWebhook updates a webhook
func (s *WebhookService) UpdateWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, req *UpdateWebhookRequest) (*models.Webhook, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Verify requester has permission
	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageWebhooks); err != nil {
				return nil, err
			}
		}
	}

	// Validate and update name
	if req.Name != nil {
		if *req.Name == "" || len(*req.Name) > 80 {
			return nil, ErrWebhookNameTooLong
		}
		webhook.Name = *req.Name
	}

	// Update avatar
	if req.Avatar != nil {
		webhook.Avatar = req.Avatar
	}

	// Update channel (if moving to another channel in same server)
	if req.ChannelID != nil && *req.ChannelID != webhook.ChannelID {
		newChannel, err := s.channelRepo.GetByID(ctx, *req.ChannelID)
		if err != nil {
			return nil, ErrChannelNotFound
		}

		// Ensure new channel is in the same server
		if channel.ServerID != nil && (newChannel.ServerID == nil || *newChannel.ServerID != *channel.ServerID) {
			return nil, ErrNoPermission
		}

		webhook.ChannelID = *req.ChannelID
	}

	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	s.eventBus.Publish("webhook.updated", &WebhookUpdatedEvent{
		WebhookID: webhook.ID,
		ChannelID: webhook.ChannelID,
		UpdaterID: requesterID,
	})

	return webhook, nil
}

// DeleteWebhook deletes a webhook
func (s *WebhookService) DeleteWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) error {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return ErrWebhookNotFound
	}

	// Verify requester has permission
	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageWebhooks); err != nil {
				return err
			}
		}
	}

	if err := s.webhookRepo.Delete(ctx, webhookID); err != nil {
		return err
	}

	s.eventBus.Publish("webhook.deleted", &WebhookDeletedEvent{
		WebhookID: webhook.ID,
		ChannelID: webhook.ChannelID,
		ServerID:  webhook.ServerID,
		DeleterID: requesterID,
	})

	return nil
}

// ExecuteWebhookRequest represents a request to execute a webhook
type ExecuteWebhookRequest struct {
	Content         string
	Username        *string
	AvatarURL       *string
	TTS             bool
	Embeds          []models.Embed
	AllowedMentions *models.WebhookMessage `json:"allowed_mentions,omitempty"`
	ThreadName      string
}

// CheckRateLimit checks if the webhook is rate limited
func (s *WebhookService) CheckRateLimit(ctx context.Context, webhookID uuid.UUID) error {
	if s.cache == nil {
		return nil // No cache configured, skip rate limiting
	}

	key := fmt.Sprintf("webhook:ratelimit:%s", webhookID.String())
	count, err := s.cache.IncrementWithExpiry(ctx, key, RateLimitWindow)
	if err != nil {
		// Log error but don't block the request on cache failure
		return nil
	}

	if count > int64(MaxRequestsPerMinute) {
		return ErrWebhookRateLimited
	}

	return nil
}

// ExecuteWebhook executes a webhook by sending a message
func (s *WebhookService) ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *ExecuteWebhookRequest) (*models.Message, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Verify token using constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(webhook.Token), []byte(token)) != 1 {
		return nil, ErrInvalidWebhookToken
	}

	// Check rate limit
	if err := s.CheckRateLimit(ctx, webhookID); err != nil {
		return nil, err
	}

	// Validate message (must have content, embeds, or files)
	if req.Content == "" && len(req.Embeds) == 0 {
		return nil, ErrEmptyMessage
	}

	// Validate embeds limit
	if len(req.Embeds) > 10 {
		return nil, ErrTooManyEmbeds
	}

	// Log delivery attempt
	delivery := &models.WebhookDelivery{
		ID:            uuid.New(),
		WebhookID:     webhookID,
		AttemptNumber: 1,
		CreatedAt:     time.Now(),
	}

	// Store request payload for auditing
	requestPayload := map[string]interface{}{
		"content":     req.Content,
		"tts":         req.TTS,
		"embeds_count": len(req.Embeds),
		"thread_name": req.ThreadName,
	}
	delivery.RequestPayload = &requestPayload

	// Create actual message via MessageService using webhook-specific method
	// This skips permission checks and uses webhook ID as AuthorID
	message, err := s.messageService.SendMessageForWebhook(ctx, SendWebhookMessageRequest{
		WebhookID:       webhook.ID,
		ChannelID:       webhook.ChannelID,
		Content:         req.Content,
		Username:        req.Username,
		AvatarURL:       req.AvatarURL,
		TTS:             req.TTS,
		Embeds:          req.Embeds,
		AllowedMentions: req.AllowedMentions,
		ThreadName:      req.ThreadName,
	})

	now := time.Now()
	delivery.DeliveredAt = &now

	if err != nil {
		// Log failed delivery
		errorMsg := err.Error()
		delivery.ErrorMessage = &errorMsg
		zero := 0
		delivery.StatusCode = &zero
		if s.webhookDeliveryRepo != nil {
			s.webhookDeliveryRepo.Create(ctx, delivery)
		}
		return nil, err
	}

	// Log successful delivery
	statusCode := http.StatusOK
	delivery.StatusCode = &statusCode
	responseBody := "Message created successfully"
	delivery.ResponseBody = &responseBody
	zeroDuration := 0
	delivery.DurationMs = &zeroDuration // Could track actual duration if needed

	if s.webhookDeliveryRepo != nil {
		s.webhookDeliveryRepo.Create(ctx, delivery)
	}

	s.eventBus.Publish("webhook.executed", &WebhookExecutedEvent{
		WebhookID: webhook.ID,
		ChannelID: webhook.ChannelID,
		MessageID: message.ID,
	})

	return message, nil
}

// ExecuteWebhookWithRetry executes a webhook with retry logic for failed deliveries
func (s *WebhookService) ExecuteWebhookWithRetry(ctx context.Context, webhookID uuid.UUID, token string, req *ExecuteWebhookRequest) (*models.Message, error) {
	var lastErr error
	var message *models.Message

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		message, lastErr = s.ExecuteWebhook(ctx, webhookID, token, req)
		if lastErr == nil {
			return message, nil
		}

		// Don't retry on certain errors
		if lastErr == ErrWebhookNotFound || lastErr == ErrInvalidWebhookToken || lastErr == ErrEmptyMessage || lastErr == ErrTooManyEmbeds {
			return nil, lastErr
		}

		// Don't retry on rate limit
		if lastErr == ErrWebhookRateLimited {
			return nil, lastErr
		}

		// Exponential backoff before retry
		if attempt < MaxRetries {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, lastErr
}

// GetWebhookStats retrieves delivery statistics for a webhook
func (s *WebhookService) GetWebhookStats(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.WebhookDeliveryStats, error) {
	// Verify webhook exists and requester has access
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageWebhooks); err != nil {
				return nil, err
			}
		}
	}

	if s.webhookDeliveryRepo == nil {
		return &models.WebhookDeliveryStats{}, nil
	}

	return s.webhookDeliveryRepo.GetStats(ctx, webhookID)
}

// GetWebhookDeliveries retrieves delivery history for a webhook
func (s *WebhookService) GetWebhookDeliveries(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, error) {
	// Verify webhook exists and requester has access
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageWebhooks); err != nil {
				return nil, err
			}
		}
	}

	if s.webhookDeliveryRepo == nil {
		return []*models.WebhookDelivery{}, nil
	}

	return s.webhookDeliveryRepo.GetByWebhookID(ctx, webhookID, limit, offset)
}

// TestWebhook tests a webhook by sending a test message
func (s *WebhookService) TestWebhook(ctx context.Context, webhookID uuid.UUID, requesterID uuid.UUID) (*models.Message, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Verify requester has permission
	channel, err := s.channelRepo.GetByID(ctx, webhook.ChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Require MANAGE_WEBHOOKS permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageWebhooks); err != nil {
				return nil, err
			}
		}
	}

	// Send a test message
	testContent := "🔔 **Webhook Test**\n\nThis is a test message from your webhook **" + webhook.Name + "**.\n\nYour webhook is working correctly! ✅"
	
	return s.ExecuteWebhook(ctx, webhookID, webhook.Token, &ExecuteWebhookRequest{
		Content: testContent,
	})
}

// Helper functions

func generateWebhookToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Events

type WebhookCreatedEvent struct {
	WebhookID uuid.UUID
	ChannelID uuid.UUID
	ServerID  uuid.UUID
	CreatorID uuid.UUID
}

type WebhookUpdatedEvent struct {
	WebhookID uuid.UUID
	ChannelID uuid.UUID
	UpdaterID uuid.UUID
}

type WebhookDeletedEvent struct {
	WebhookID uuid.UUID
	ChannelID uuid.UUID
	ServerID  *uuid.UUID
	DeleterID uuid.UUID
}

type WebhookExecutedEvent struct {
	WebhookID uuid.UUID
	ChannelID uuid.UUID
	MessageID uuid.UUID
}
