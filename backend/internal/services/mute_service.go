package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// MuteRepository defines the interface for Mute data access operations.
type MuteRepository interface {
	// Create creates a new Mute record in the database.
	Create(ctx context.Context, mute *models.Mute) error

	// GetByID retrieves a mute by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Mute, error)

	// GetByChannelAndUser retrieves mute status for a specific channel and user.
	GetByChannelAndUser(ctx context.Context, channelID, userID uuid.UUID) (*models.Mute, error)

	// Update modifies an existing mute record (e.g., ending the mute).
	Update(ctx context.Context, mute *models.Mute) error

	// SoftDelete marks a mute as ended logic-wise (optional but good for data integrity)
	// or relies on the `EndedAt` timestamp field.
}

// MuteService handles business logic related to muting users in voice/text channels.
type MuteService struct {
	repo MuteRepository
}

// NewMuteService creates a new MuteService instance.
func NewMuteService(repo MuteRepository) *MuteService {
	return &MuteService{
		repo: repo,
	}
}

// MuteUser creates a new mute record for a user in a specific channel.
func (s *MuteService) MuteUser(ctx context.Context, channelID, userID uuid.UUID, durationMinutes int) (*models.Mute, error) {
	if durationMinutes < 0 {
		return nil, errors.New("duration minutes cannot be negative")
	}

	// Calculate end time
	var endsAt *time.Time
	if durationMinutes > 0 {
		endTime := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
		endsAt = &endTime
	}

	mute := &models.Mute{
		ID:         uuid.New(),
		ChannelID:  channelID,
		UserID:     userID,
		MutedBy:    nil, // Could be populated from context if a bot/superuser ID is passed
		RoleID:     nil, // Could be populated if using roles
		Reason:     "",  // Could be added via params
		StartedAt:  time.Now(),
		EndedAt:    endsAt,
		RestoredAt: nil,
	}

	if err := s.repo.Create(ctx, mute); err != nil {
		return nil, fmt.Errorf("failed to create mute: %w", err)
	}

	return mute, nil
}

// UnmuteUser removes a mute for a user in a specific channel.
func (s *MuteService) UnmuteUser(ctx context.Context, channelID, userID uuid.UUID) error {
	// 1. Find the active mute
	mute, err := s.repo.GetByChannelAndUser(ctx, channelID, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve mute: %w", err)
	}

	if mute == nil {
		return ErrUserNotMuted
	}

	// 2. Update the mute record
	currentTime := time.Now()

	mute.EndedAt = &currentTime
	mute.RestoredAt = &currentTime

	if err := s.repo.Update(ctx, mute); err != nil {
		return fmt.Errorf("failed to update mute: %w", err)
	}

	return nil
}

// IsUserMuted checks if a user is currently muted in a specific channel.
func (s *MuteService) IsUserMuted(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	mute, err := s.repo.GetByChannelAndUser(ctx, channelID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check mute status: %w", err)
	}

	// If mute is nil, user is not muted.
	if mute == nil {
		return false, nil
	}

	// If EndedAt is nil (never ended) or EndedAt is in the future, user is currently muted.
	now := time.Now()

	return mute.EndedAt == nil || mute.EndedAt.After(now), nil
}

var ErrUserNotMuted = errors.New("user is not muted in this channel")
