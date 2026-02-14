package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// WebhookService handles webhook-related business logic
type WebhookService struct {
	webhookRepo WebhookRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	eventBus    EventBus
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	webhookRepo WebhookRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	eventBus EventBus,
) *WebhookService {
	return &WebhookService{
		webhookRepo: webhookRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		eventBus:    eventBus,
	}
}

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
	}

	// TODO: Check MANAGE_WEBHOOKS permission

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

	// TODO: Check MANAGE_WEBHOOKS permission

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
		// TODO: Check MANAGE_WEBHOOKS permission
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
		// TODO: Check MANAGE_WEBHOOKS permission
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
	Content   string
	Username  *string
	AvatarURL *string
	TTS       bool
}

// ExecuteWebhook executes a webhook by sending a message
func (s *WebhookService) ExecuteWebhook(ctx context.Context, webhookID uuid.UUID, token string, req *ExecuteWebhookRequest) (*models.Message, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, ErrWebhookNotFound
	}

	// Verify token
	if webhook.Token != token {
		return nil, ErrInvalidWebhookToken
	}

	// Validate message
	if req.Content == "" {
		return nil, ErrEmptyMessage
	}

	// TODO: Create actual message via MessageService
	// For now, return a mock message
	author := models.PublicUser{
		ID:       webhook.ID,
		Username: webhook.Name,
	}

	// Override username if provided
	if req.Username != nil && *req.Username != "" {
		author.Username = *req.Username
	}

	message := &models.Message{
		ID:        uuid.New(),
		ChannelID: webhook.ChannelID,
		Content:   req.Content,
		Author:    &author,
		CreatedAt: time.Now(),
	}

	s.eventBus.Publish("webhook.executed", &WebhookExecutedEvent{
		WebhookID: webhook.ID,
		ChannelID: webhook.ChannelID,
		MessageID: message.ID,
	})

	return message, nil
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
