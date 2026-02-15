package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrBanNotFound      = errors.New("ban not found")
	ErrBanAlreadyExist  = errors.New("user is already banned in this guild")
	ErrBanAlreadyActive = errors.New("ban is already active")
)

// BanManagementRepository defines the contract for ban persistence operations.
// This is separate from the BanRepository in invite_service.go which has different needs.
type BanManagementRepository interface {
	FindByUserAndGuild(ctx context.Context, guildID, userID uuid.UUID) (*models.Ban, error)
	Create(ctx context.Context, ban *models.Ban) error
	GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error)
	Delete(ctx context.Context, serverID, userID uuid.UUID) error
}

// BanService handles business logic related to banning users.
type BanService struct {
	repo BanManagementRepository
}

// NewBanService creates a new BanService instance.
func NewBanService(repo BanManagementRepository) *BanService {
	return &BanService{
		repo: repo,
	}
}

// CreateBan creates a new ban record.
// It checks if the user is already banned before inserting.
func (s *BanService) CreateBan(ctx context.Context, guildID, userID uuid.UUID, reason string, bannedBy uuid.UUID) (*models.Ban, error) {
	// Check if user is currently banned
	existingBan, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)

	// Handle database specific errors (should be wrapped externally, but checked here)
	if err != nil && !errors.Is(err, ErrBanNotFound) {
		return nil, fmt.Errorf("failed to check existing bans: %w", err)
	}

	if existingBan != nil {
		// User is already banned
		return nil, ErrBanAlreadyExist
	}

	newBan := &models.Ban{
		ServerID: guildID,
		UserID:   userID,
		Reason:   &reason,
		BannedBy: &bannedBy,
	}

	if err := s.repo.Create(ctx, newBan); err != nil {
		return nil, fmt.Errorf("failed to create ban: %w", err)
	}

	return newBan, nil
}

// Unban removes a ban by server and user ID.
func (s *BanService) Unban(ctx context.Context, serverID, userID uuid.UUID) error {
	// Verify it exists
	ban, err := s.repo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve ban: %w", err)
	}

	// Avoid unused variable error
	_ = ban

	if err := s.repo.Delete(ctx, serverID, userID); err != nil {
		return fmt.Errorf("failed to delete ban: %w", err)
	}

	return nil
}

// GetBan retrieves a ban by server and user ID.
func (s *BanService) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return s.repo.GetByServerAndUser(ctx, serverID, userID)
}

// CheckIfBanned checks if a user is currently banned in a guild.
// It returns true if banned. It is the caller's responsibility to check context cancellation.
func (s *BanService) CheckIfBanned(ctx context.Context, guildID, userID uuid.UUID) (bool, error) {
	ban, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)
	if err != nil {
		// If it's an "Not Found" error, the user is not banned.
		if errors.Is(err, ErrBanNotFound) {
			return false, nil
		}
		return false, err
	}

	// Avoid unused variable error
	_ = ban
	return true, nil
}
