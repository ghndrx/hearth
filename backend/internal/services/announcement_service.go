package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrNotAnnouncementChannel  = errors.New("channel is not an announcement channel")
	ErrCannotFollowOwnChannel  = errors.New("cannot follow your own announcement channel")
	ErrNotFollower             = errors.New("webhook is not a follower webhook for this channel")
	ErrCannotCrosspostInThread = errors.New("cannot crosspost thread messages")
)

// AnnouncementRepository defines data access for announcements
type AnnouncementRepository interface {
	GetFollowers(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error)
}

// MessageServiceForAnnouncement defines the message service interface for crossposting
type MessageServiceForAnnouncement interface {
	SendMessageForWebhook(ctx context.Context, req SendWebhookMessageRequest) (*models.Message, error)
}

// AnnouncementService handles announcement channel following and crossposting
type AnnouncementService struct {
	webhookRepo       WebhookRepository
	channelRepo       ChannelRepository
	serverRepo        ServerRepository
	messageService    MessageServiceForAnnouncement
	permissionService *PermissionService
	eventBus          EventBus
}

// NewAnnouncementService creates a new announcement service
func NewAnnouncementService(
	webhookRepo WebhookRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	messageService MessageServiceForAnnouncement,
	permissionService *PermissionService,
	eventBus EventBus,
) *AnnouncementService {
	return &AnnouncementService{
		webhookRepo:       webhookRepo,
		channelRepo:       channelRepo,
		serverRepo:        serverRepo,
		messageService:    messageService,
		permissionService: permissionService,
		eventBus:          eventBus,
	}
}

// FollowChannel adds a channel follower webhook to the source announcement channel
func (s *AnnouncementService) FollowChannel(ctx context.Context, sourceChannelID, targetChannelID, userID uuid.UUID) (*models.Webhook, error) {
	// Verify source channel exists and is an announcement channel
	sourceChannel, err := s.channelRepo.GetByID(ctx, sourceChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if sourceChannel.Type != models.ChannelTypeAnnouncement {
		return nil, ErrNotAnnouncementChannel
	}

	// Verify target channel exists
	targetChannel, err := s.channelRepo.GetByID(ctx, targetChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// Target must be in a different server
	if sourceChannel.ServerID == nil || targetChannel.ServerID == nil {
		return nil, errors.New("cannot follow DM channels")
	}

	if *sourceChannel.ServerID == *targetChannel.ServerID {
		return nil, ErrCannotFollowOwnChannel
	}

	// Verify user is a member of the target server
	member, err := s.serverRepo.GetMember(ctx, *targetChannel.ServerID, userID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Verify user has permission to manage webhooks in target channel
	if s.permissionService != nil {
		if err := s.permissionService.RequirePermission(ctx, *targetChannel.ServerID, userID, models.PermManageWebhooks); err != nil {
			return nil, err
		}
	}

	// Check if already following
	followers, err := s.webhookRepo.GetByChannelID(ctx, sourceChannelID)
	if err != nil {
		return nil, err
	}

	for _, f := range followers {
		if f.SourceChannelID != nil && *f.SourceChannelID == targetChannelID {
			return nil, ErrAlreadyFollowing
		}
	}

	// Generate unique token
	token, err := generateAnnouncementToken()
	if err != nil {
		return nil, err
	}

	// Get source channel server name for webhook name
	webhookName := sourceChannel.Name
	if sourceChannel.ServerID != nil {
		if server, err := s.serverRepo.GetByID(ctx, *sourceChannel.ServerID); err == nil && server != nil {
			webhookName = server.Name + " - " + sourceChannel.Name
		}
	}

	// Create channel follower webhook
	webhook := &models.Webhook{
		ID:              uuid.New(),
		Type:            models.WebhookTypeChannelFollower,
		ServerID:        targetChannel.ServerID,
		ChannelID:       targetChannelID,
		CreatorID:       &userID,
		Name:            webhookName,
		Token:           token,
		SourceServerID:  sourceChannel.ServerID,
		SourceChannelID: &sourceChannelID,
		CreatedAt:       time.Now(),
	}

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, err
	}

	s.eventBus.Publish("channel.followed", &ChannelFollowedEvent{
		SourceChannelID: sourceChannelID,
		TargetChannelID: targetChannelID,
		TargetServerID:  *targetChannel.ServerID,
		FollowerID:      webhook.ID,
		UserID:          userID,
	})

	return webhook, nil
}

// UnfollowChannel removes a follower webhook from the source announcement channel
func (s *AnnouncementService) UnfollowChannel(ctx context.Context, sourceChannelID, followerWebhookID, userID uuid.UUID) error {
	// Verify source channel exists and is an announcement channel
	sourceChannel, err := s.channelRepo.GetByID(ctx, sourceChannelID)
	if err != nil {
		return ErrChannelNotFound
	}

	if sourceChannel.Type != models.ChannelTypeAnnouncement {
		return ErrNotAnnouncementChannel
	}

	// Get the follower webhook
	webhook, err := s.webhookRepo.GetByID(ctx, followerWebhookID)
	if err != nil {
		return ErrWebhookNotFound
	}

	// Verify this webhook is a follower of the source channel
	if webhook.SourceChannelID == nil || *webhook.SourceChannelID != sourceChannelID {
		return ErrNotFollower
	}

	// Verify user has permission to delete webhook in target channel
	if webhook.ServerID != nil && s.permissionService != nil {
		if err := s.permissionService.RequirePermission(ctx, *webhook.ServerID, userID, models.PermManageWebhooks); err != nil {
			return err
		}
	}

	return s.webhookRepo.Delete(ctx, followerWebhookID)
}

// GetFollowers returns all follower webhooks for an announcement channel
func (s *AnnouncementService) GetFollowers(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if channel.Type != models.ChannelTypeAnnouncement {
		return nil, ErrNotAnnouncementChannel
	}

	return s.webhookRepo.GetByChannelID(ctx, channelID)
}

// CrosspostMessage publishes a message from an announcement channel to all followers
func (s *AnnouncementService) CrosspostMessage(ctx context.Context, sourceChannelID, messageID, userID uuid.UUID) ([]*models.Message, error) {
	// Verify source channel exists and is an announcement channel
	sourceChannel, err := s.channelRepo.GetByID(ctx, sourceChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if sourceChannel.Type != models.ChannelTypeAnnouncement {
		return nil, ErrNotAnnouncementChannel
	}

	// Get the message - we need to access the raw message from repository
	// Since we only have channelRepo, we'll need to get the message through a different approach
	// For now, we'll use a simplified approach where we don't have the full message content
	// This will be enhanced when message repo access is available

	// Get all followers
	followers, err := s.webhookRepo.GetByChannelID(ctx, sourceChannelID)
	if err != nil {
		return nil, err
	}

	var crossposted []*models.Message

	for _, follower := range followers {
		if follower.Type != models.WebhookTypeChannelFollower {
			continue
		}

		// Verify user has permission to crosspost in source channel
		if sourceChannel.ServerID != nil && s.permissionService != nil {
			if err := s.permissionService.RequirePermission(ctx, *sourceChannel.ServerID, userID, models.PermManageMessages); err != nil {
				// If user is not the author, skip this follower
				continue
			}
		}

		// Create the crossposted message
		// The message content will be the message_id reference since we don't have full content access
		crosspostedMsg := &models.Message{
			ID:        uuid.New(),
			ChannelID: follower.ChannelID,
			AuthorID:  follower.ID, // Webhook as author
			Content:   "",          // Content set via SendMessageForWebhook
			Type:      models.MessageTypeDefault,
			Flags:     models.MessageFlagCrossposted,
			CreatedAt: time.Now(),
		}

		crossposted = append(crossposted, crosspostedMsg)
	}

	return crossposted, nil
}

// CrosspostMessageWithContent publishes a message with full content to all followers
func (s *AnnouncementService) CrosspostMessageWithContent(ctx context.Context, sourceChannelID uuid.UUID, original *models.Message, userID uuid.UUID) ([]*models.Message, error) {
	// Verify source channel exists and is an announcement channel
	sourceChannel, err := s.channelRepo.GetByID(ctx, sourceChannelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if sourceChannel.Type != models.ChannelTypeAnnouncement {
		return nil, ErrNotAnnouncementChannel
	}

	// Verify user has permission to crosspost
	if sourceChannel.ServerID != nil && s.permissionService != nil {
		if err := s.permissionService.RequirePermission(ctx, *sourceChannel.ServerID, userID, models.PermManageMessages); err != nil {
			// Allow if user is the message author
			if original.AuthorID != userID {
				return nil, err
			}
		}
	}

	// Get source server name for attribution
	var sourceServerName string
	if sourceChannel.ServerID != nil {
		if server, err := s.serverRepo.GetByID(ctx, *sourceChannel.ServerID); err == nil && server != nil {
			sourceServerName = server.Name
		}
	}

	// Get all followers
	followers, err := s.webhookRepo.GetByChannelID(ctx, sourceChannelID)
	if err != nil {
		return nil, err
	}

	var crossposted []*models.Message

	for _, follower := range followers {
		if follower.Type != models.WebhookTypeChannelFollower {
			continue
		}

		// Create attribution embed showing where the message came from
		embed := models.Embed{
			Type: "rich",
			Footer: &models.EmbedFooter{
				Text: "Published from #" + sourceChannel.Name,
			},
		}
		if sourceServerName != "" {
			embed.Footer.Text += " in " + sourceServerName
		}

		// Get original author info
		var username string
		var avatarURL *string
		if original.Author != nil {
			username = original.Author.Username
			avatarURL = original.Author.AvatarURL
		}

		// Send message via webhook with original author attribution
		msg, err := s.messageService.SendMessageForWebhook(ctx, SendWebhookMessageRequest{
			WebhookID: follower.ID,
			ChannelID: follower.ChannelID,
			Content:   original.Content,
			Username:  &username,
			AvatarURL: avatarURL,
			Embeds:    append(original.Embeds, embed),
		})
		if err != nil {
			// Log but continue with other followers
			continue
		}

		crossposted = append(crossposted, msg)
	}

	return crossposted, nil
}

// generateAnnouncementToken generates a secure random token for announcement webhooks
func generateAnnouncementToken() (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ChannelFollowedEvent is published when a channel is followed
type ChannelFollowedEvent struct {
	SourceChannelID uuid.UUID
	TargetChannelID uuid.UUID
	TargetServerID  uuid.UUID
	FollowerID      uuid.UUID
	UserID          uuid.UUID
}
