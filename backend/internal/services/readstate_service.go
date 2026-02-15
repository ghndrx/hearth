package services

import (
	"context"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// ReadStateRepository defines the interface for read state data access
type ReadStateRepository interface {
	GetReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error)
	SetReadState(ctx context.Context, userID, channelID uuid.UUID, lastMessageID *uuid.UUID) error
	IncrementMentionCount(ctx context.Context, userID, channelID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID, channelID uuid.UUID) (int, error)
	GetUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error)
	GetUserUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error)
	GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error)
	MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error
	DeleteByChannel(ctx context.Context, channelID uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}

// ReadStateService handles read state operations
type ReadStateService struct {
	repo        ReadStateRepository
	channelRepo ChannelRepository
}

// NewReadStateService creates a new read state service
func NewReadStateService(repo ReadStateRepository, channelRepo ChannelRepository) *ReadStateService {
	return &ReadStateService{
		repo:        repo,
		channelRepo: channelRepo,
	}
}

// MarkChannelAsRead marks a channel as read for a user
func (s *ReadStateService) MarkChannelAsRead(ctx context.Context, userID, channelID uuid.UUID, messageID *uuid.UUID) (*models.AckResponse, error) {
	// Get the channel to check it exists and get last message
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// If no specific message provided, use the channel's last message
	lastMsgID := messageID
	if lastMsgID == nil {
		lastMsgID = channel.LastMessageID
	}

	// Update read state
	err = s.repo.SetReadState(ctx, userID, channelID, lastMsgID)
	if err != nil {
		return nil, err
	}

	return &models.AckResponse{
		ChannelID:     channelID,
		LastMessageID: lastMsgID,
		MentionCount:  0,
	}, nil
}

// GetChannelReadState gets the read state for a user in a channel
func (s *ReadStateService) GetChannelReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error) {
	return s.repo.GetReadState(ctx, userID, channelID)
}

// GetChannelUnreadInfo gets the unread information for a user in a channel
func (s *ReadStateService) GetChannelUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	return s.repo.GetUnreadInfo(ctx, userID, channelID)
}

// GetUnreadSummary gets unread information for all channels a user has access to
func (s *ReadStateService) GetUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
	return s.repo.GetUserUnreadSummary(ctx, userID)
}

// GetServerUnreadSummary gets unread information for a specific server
func (s *ReadStateService) GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error) {
	return s.repo.GetServerUnreadSummary(ctx, userID, serverID)
}

// MarkServerAsRead marks all channels in a server as read
func (s *ReadStateService) MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error {
	return s.repo.MarkServerAsRead(ctx, userID, serverID)
}

// IncrementMention increments the mention count for a user in a channel
func (s *ReadStateService) IncrementMention(ctx context.Context, userID, channelID uuid.UUID) error {
	return s.repo.IncrementMentionCount(ctx, userID, channelID)
}

// OnChannelDeleted cleans up read states when a channel is deleted
func (s *ReadStateService) OnChannelDeleted(ctx context.Context, channelID uuid.UUID) error {
	return s.repo.DeleteByChannel(ctx, channelID)
}

// OnUserDeleted cleans up read states when a user is deleted
func (s *ReadStateService) OnUserDeleted(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteByUser(ctx, userID)
}
