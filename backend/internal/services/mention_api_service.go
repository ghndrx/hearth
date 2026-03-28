package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// MentionRepository defines the interface for mention data access
type MentionRepository interface {
	Create(ctx context.Context, mention *models.Mention) error
	CreateBatch(ctx context.Context, mentions []*models.Mention) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error)
	GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error)
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error)
	MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error)
	DeleteByMessage(ctx context.Context, messageID uuid.UUID) error
	DeleteByChannel(ctx context.Context, channelID uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error)
}

// Errors
var (
	ErrMentionNotFound = errors.New("mention not found")
)

// MentionAPIService handles mention-related business logic for the API
type MentionAPIService struct {
	mentionRepo MentionRepository
	eventBus    EventBus
}

// NewMentionAPIService creates a new mention API service
func NewMentionAPIService(mentionRepo MentionRepository, eventBus EventBus) *MentionAPIService {
	return &MentionAPIService{
		mentionRepo: mentionRepo,
		eventBus:    eventBus,
	}
}

// GetMentions retrieves mentions for a user with the given filter
func (s *MentionAPIService) GetMentions(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	return s.mentionRepo.GetMentionsWithContext(ctx, filter)
}

// GetUnreadCount returns the count of unread mentions for a user
func (s *MentionAPIService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.mentionRepo.GetUnreadCount(ctx, userID)
}

// GetStats returns mention statistics for a user
func (s *MentionAPIService) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	return s.mentionRepo.GetStats(ctx, userID)
}

// MarkAsRead marks a mention as read
func (s *MentionAPIService) MarkAsRead(ctx context.Context, mentionID, userID uuid.UUID) error {
	// Verify the mention exists and belongs to the user
	mention, err := s.mentionRepo.GetByID(ctx, mentionID)
	if err != nil {
		return err
	}
	if mention == nil || mention.UserID != userID {
		return ErrMentionNotFound
	}

	if err := s.mentionRepo.MarkAsRead(ctx, mentionID, userID); err != nil {
		return err
	}

	// Emit event
	if s.eventBus != nil {
		s.eventBus.Publish("mentions.read", &MentionReadEvent{
			UserID:    userID,
			MentionID: mentionID,
		})
	}

	return nil
}

// MarkAllAsRead marks all mentions as read for a user
func (s *MentionAPIService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.mentionRepo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Emit event
	if s.eventBus != nil && count > 0 {
		s.eventBus.Publish("mentions.read_all", &MentionsReadAllEvent{
			UserID: userID,
			Count:  count,
		})
	}

	return count, nil
}

// MarkChannelMentionsAsRead marks all mentions in a channel as read
func (s *MentionAPIService) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	count, err := s.mentionRepo.MarkChannelMentionsAsRead(ctx, userID, channelID)
	if err != nil {
		return 0, err
	}

	// Emit event
	if s.eventBus != nil && count > 0 {
		s.eventBus.Publish("mentions.channel_read", &MentionsChannelReadEvent{
			UserID:    userID,
			ChannelID: channelID,
			Count:     count,
		})
	}

	return count, nil
}

// Search searches mentions by query
func (s *MentionAPIService) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	return s.mentionRepo.Search(ctx, userID, query, limit)
}

// CreateMention creates a mention record (called from message service)
func (s *MentionAPIService) CreateMention(ctx context.Context, req *models.CreateMentionRequest) error {
	mention := &models.Mention{
		UserID:             req.UserID,
		MessageID:          req.MessageID,
		MentionedBy:        req.MentionedBy,
		ChannelID:          req.ChannelID,
		GuildID:            req.GuildID,
		MentionType:        req.MentionType,
		MentionedRoleID:    req.MentionedRoleID,
		MentionedChannelID: req.MentionedChannelID,
	}

	if err := s.mentionRepo.Create(ctx, mention); err != nil {
		return err
	}

	// Emit event for real-time delivery
	if s.eventBus != nil {
		s.eventBus.Publish("mention.created", &MentionCreatedEvent{
			Mention: mention,
		})
	}

	return nil
}

// CreateMentionsBatch creates multiple mention records
func (s *MentionAPIService) CreateMentionsBatch(ctx context.Context, mentions []*models.Mention) error {
	if len(mentions) == 0 {
		return nil
	}

	if err := s.mentionRepo.CreateBatch(ctx, mentions); err != nil {
		return err
	}

	// Emit events for each mention
	if s.eventBus != nil {
		for _, m := range mentions {
			s.eventBus.Publish("mention.created", &MentionCreatedEvent{
				Mention: m,
			})
		}
	}

	return nil
}

// DeleteByMessage deletes mentions when a message is deleted
func (s *MentionAPIService) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	return s.mentionRepo.DeleteByMessage(ctx, messageID)
}

// Events

// MentionCreatedEvent is emitted when a mention is created
type MentionCreatedEvent struct {
	Mention *models.Mention
}

// MentionReadEvent is emitted when a mention is marked as read
type MentionReadEvent struct {
	UserID    uuid.UUID
	MentionID uuid.UUID
}

// MentionsReadAllEvent is emitted when all mentions are marked as read
type MentionsReadAllEvent struct {
	UserID uuid.UUID
	Count  int
}

// MentionsChannelReadEvent is emitted when channel mentions are marked as read
type MentionsChannelReadEvent struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
	Count     int
}
