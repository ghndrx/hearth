package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrAlreadyFollowing = errors.New("already following this channel")
	ErrNotFollowing     = errors.New("not following this channel")
	ErrCannotFollowSelf = errors.New("a channel cannot follow itself")
)

// FollowRepositoryInterface defines follow data access
type FollowRepositoryInterface interface {
	Create(ctx context.Context, follow *models.FollowedChannel) error
	Delete(ctx context.Context, channelID, followerChannelID uuid.UUID) error
	GetByChannelAndFollower(ctx context.Context, channelID, followerChannelID uuid.UUID) (*models.FollowedChannel, error)
	GetFollowers(ctx context.Context, channelID uuid.UUID) ([]models.FollowedChannel, error)
}

// FollowService handles channel follow business logic
type FollowService struct {
	followRepo  FollowRepositoryInterface
	channelRepo ChannelRepository
}

// NewFollowService creates a new follow service
func NewFollowService(followRepo FollowRepositoryInterface, channelRepo ChannelRepository) *FollowService {
	return &FollowService{
		followRepo:  followRepo,
		channelRepo: channelRepo,
	}
}

// FollowChannel creates a follow relationship between channels
func (s *FollowService) FollowChannel(ctx context.Context, channelID, followerChannelID uuid.UUID) (*models.FollowedChannel, error) {
	if channelID == followerChannelID {
		return nil, ErrCannotFollowSelf
	}

	// Verify target channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Verify follower channel exists
	follower, err := s.channelRepo.GetByID(ctx, followerChannelID)
	if err != nil {
		return nil, err
	}
	if follower == nil {
		return nil, ErrChannelNotFound
	}

	// Check if already following
	existing, err := s.followRepo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyFollowing
	}

	follow := &models.FollowedChannel{
		ChannelID:         channelID,
		FollowerChannelID: followerChannelID,
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.followRepo.Create(ctx, follow); err != nil {
		return nil, err
	}

	return follow, nil
}

// UnfollowChannel removes a follow relationship
func (s *FollowService) UnfollowChannel(ctx context.Context, channelID, followerChannelID uuid.UUID) error {
	existing, err := s.followRepo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFollowing
	}

	return s.followRepo.Delete(ctx, channelID, followerChannelID)
}

// GetFollowers retrieves all followers of a channel
func (s *FollowService) GetFollowers(ctx context.Context, channelID uuid.UUID) ([]models.FollowedChannel, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	return s.followRepo.GetFollowers(ctx, channelID)
}
