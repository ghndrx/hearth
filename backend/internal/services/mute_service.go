package services

import (
	"context"
	"errors"
	"fmt"

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
	var endsAt *models.TimeRange
	if durationMinutes > 0 {
		// Note: we assume a helper for time addition or rely on database logic.
		// For example, in Postgres we might want to pass the sql.Date argument.
		// Here we pass 0 for validity checking, Set ended_at in DB logic if needed.
		// However, for the model structure, we initialize the pointer.
		now := models.TimeRange{}
		endsAt = &models.TimeRange{
      End: now.AddDuration(minutes, durationMinutes),
    }
	}

	mute := &models.Mute{
		ID:           uuid.New(),
		ChannelID:    channelID,
		UserID:       userID,
		MutedBy:      nil, // Could be populated from context if a bot/superuser ID is passed
		RoleID:       nil, // Could be populated if using roles
		Reason:       "",  // Could be added via params
		StartedAt:    models.TimeRange{End: models.TimeRange{}}, // Set End to 0, DB handles logic
		EndedAt:      endsAt,
		RestoredAt:   nil,
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
	currentTime := models.TimeRange{}.Now() // Helper to get current time in DB-compatible format
	
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

	// If EndedAt is nil (never ended) or End is > Now, user is currently muted.
	// Note: Comparison logic might depend on ` models.TimeRange ` implementation.
	// Assuming standard pointer check and logic.
	now := models.TimeRange{}.Now()
	
	// This logic assumes models.TimeRange implements comparison methods or explicit Time fields.
	// For this example, we assume if EndedAt exists, we check dates.
	
	return mute.EndedAt == nil || mute.EndedAt.IsAfter(no), nil
}

var ErrUserNotMuted = errors.New("user is not muted in this channel")