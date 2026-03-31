package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// PollRepository defines the interface for data persistence.
// This ensures services are decoupled from the specific database implementation.
type PollRepository interface {
	Create(ctx context.Context, poll *models.Poll) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Poll, error)
	Update(ctx context.Context, poll *models.Poll) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Poll, error)
	GetByGuildID(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error)
	VoteForOption(ctx context.Context, pollID, optionID, userID uuid.UUID) error
	CheckUserVote(ctx context.Context, pollID, userID uuid.UUID) (*models.PollOptionVote, error)
}

// PollService handles business logic for polls.
type PollService struct {
	repo PollRepository
}

// NewPollService creates a new PollService instance.
func NewPollService(repo PollRepository) *PollService {
	return &PollService{
		repo: repo,
	}
}

// CreatePoll creates a new poll and saves it to the repository.
func (s *PollService) CreatePoll(ctx context.Context, poll *models.Poll) error {
	if poll.ID == uuid.Nil {
		poll.ID = uuid.New()
	}
	return s.repo.Create(ctx, poll)
}

// GetPoll retrieves a poll by its ID.
func (s *PollService) GetPoll(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid poll ID")
	}
	return s.repo.GetByID(ctx, id)
}

// UpdatePoll updates an existing poll.
func (s *PollService) UpdatePoll(ctx context.Context, poll *models.Poll) error {
	if poll.ID == uuid.Nil {
		return errors.New("cannot update poll with nil ID")
	}
	return s.repo.Update(ctx, poll)
}

// DeletePoll deletes a poll by its ID.
func (s *PollService) DeletePoll(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("cannot delete poll with nil ID")
	}
	return s.repo.Delete(ctx, id)
}

// GetGuildPolls retrieves all active polls for a specific guild.
func (s *PollService) GetGuildPolls(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error) {
	if guildID == uuid.Nil {
		return nil, errors.New("invalid guild ID")
	}
	return s.repo.GetByGuildID(ctx, guildID)
}

// GetChannelPolls retrieves all polls for a specific channel.
func (s *PollService) GetChannelPolls(ctx context.Context, channelID uuid.UUID) ([]*models.Poll, error) {
	if channelID == uuid.Nil {
		return nil, errors.New("invalid channel ID")
	}
	return s.repo.GetByChannelID(ctx, channelID)
}

// Vote adds a vote to a specific option within a poll.
func (s *PollService) Vote(ctx context.Context, pollID, optionID, userID uuid.UUID) error {
	// Check if poll exists and is still open
	poll, err := s.repo.GetByID(ctx, pollID)
	if err != nil {
		return fmt.Errorf("error fetching poll: %w", err)
	}
	if poll == nil {
		return ErrPollNotFound
	}
	if poll.EndTime != nil && poll.EndTime.Before(time.Now()) {
		return ErrPollClosed
	}

	// Check if user already voted
	userVote, err := s.repo.CheckUserVote(ctx, pollID, userID)
	if err != nil {
		return fmt.Errorf("error checking previous vote: %w", err)
	}
	if userVote != nil {
		return errors.New("user has already voted on this poll")
	}

	// VoteForOption records the vote in a transaction
	err = s.repo.VoteForOption(ctx, pollID, optionID, userID)
	if err != nil {
		return fmt.Errorf("failed to cast vote: %w", err)
	}

	return nil
}
