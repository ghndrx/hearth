package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hearth/internal/models"
)

var (
	ErrBanNotFound     = errors.New("ban not found")
	ErrBanAlreadyExist = errors.New("user is already banned in this guild")
	ErrBanAlreadyActive = errors.New("ban is already active")
)

// BanRepository defines the contract for ban persistence.
type BanRepository interface {
	FindByUserAndGuild(ctx context.Context, guildID, userID string) (*models.Ban, error)
	Insert(ctx context.Context, ban *models.Ban) error
	Get(ctx context.Context, id string) (*models.Ban, error)
	Delete(ctx context.Context, id string) error
}

// BanService handles business logic related to banning users.
type BanService struct {
	repo BanRepository
}

// NewBanService creates a new BanService instance.
func NewBanService(repo BanRepository) *BanService {
	return &BanService{
		repo: repo,
	}
}

// CreateBan creates a new ban record.
// It checks if the user is already banned before inserting.
func (s *BanService) CreateBan(ctx context.Context, guildID, userID, moderatorID, reason string, expiresAt *time.Time) (*models.Ban, error) {
	// Check if user is currently banned
	existingBan, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)
	
	// Handle database specific errors (should be wrapped externally, but checked here)
	if err != nil && !errors.Is(err, ErrBanNotFound) {
		return nil, fmt.Errorf("failed to check existing bans: %w", err)
	}

	if existingBan != nil {
		// Use specific error if active, else generic conflict
		if isBanActive(existingBan) {
			return nil, ErrBanAlreadyActive
		}
		
		// If found but inactive (expired), we might want to update it, but for simplicity 
		// we treat existing existence as conflict in this Create operation, 
		// or we could return that it was updated.
		// Let's follow strict Create semantics: fail if exists.
		return nil, ErrBanAlreadyExist
	}

	newBan := &models.Ban{
		ID:          uuid.New().String(),
		GuildID:     guildID,
		UserID:      userID,
		ModeratorID: moderatorID,
		Reason:      reason,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Insert(ctx, newBan); err != nil {
		return nil, fmt.Errorf("failed to insert ban: %w", err)
	}

	return newBan, nil
}

// Unban removes a ban by ID.
func (s *BanService) Unban(ctx context.Context, bannedID string) error {
	// Verify it exists
	ban, err := s.repo.Get(ctx, bannedID)
	if err != nil {
		return fmt.Errorf("failed to retrieve ban: %w", err)
	}

	if err := s.repo.Delete(ctx, bannedID); err != nil {
		return fmt.Errorf("failed to delete ban: %w", err)
	}

	return nil
}

// GetBan retrieves a ban by ID.
func (s *BanService) GetBan(ctx context.Context, id string) (*models.Ban, error) {
	return s.repo.Get(ctx, id)
}

// CheckIfBanned checks if a user is currently banned in a guild.
// It returns true if banned. It is the caller's responsibility to check context cancellation.
func (s *BanService) CheckIfBanned(ctx context.Context, guildID, userID string) (bool, error) {
	ban, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)
	if err != nil {
		// If it's an "Not Found" error, the user is not banned.
		if errors.Is(err, ErrBanNotFound) {
			return false, nil
		}
		return false, err
	}

	return isBanActive(ban), nil
}

// Helper to determine if a ban record represents a currently active punishment
func isBanActive(ban *models.Ban) bool {
	if ban.ExpiresAt == nil {
		return true
	}
	return ban.ExpiresAt.After(time.Now())
}