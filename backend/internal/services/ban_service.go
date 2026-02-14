package services

import (
	"context"
	"errors"
)

// Repository defines the data access contract for banning operations.
type BanRepository interface {
	InsertBan(ctx context.Context, userID string, guildID string, reason string) error
	DeleteBan(ctx context.Context, userID string, guildID string) error
	GetBan(ctx context.Context, userID string, guildID string) (*Ban, error)
	ListBans(ctx context.Context, guildID string) ([]Ban, error)
}

// Ban represents a user ban within Hearth.
type Ban struct {
	UserID   string `json:"user_id"`
	GuildID  string `json:"guild_id"`
	Reason   string `json:"reason"`
	CreatedAt int64  `json:"created_at"`
	ModeratorID string `json:"moderator_id"`
}

// BanService handles business logic related to user bans.
type BanService struct {
	repo BanRepository
}

// NewBanService creates a new BanService with the given repository.
func NewBanService(repo BanRepository) *BanService {
	return &BanService{
		repo: repo,
	}
}

var (
	ErrUserAlreadyBanned = errors.New("user is already banned")
	ErrUserNotBanned     = errors.New("user is not currently banned")
)

// BanUser bans a user from a specific guild.
// Returns ErrUserAlreadyBanned if the user is already banned.
func (s *BanService) BanUser(ctx context.Context, userID, guildID, reason, moderatorID string) error {
	// Check if user is already banned
	existingBan, err := s.repo.GetBan(ctx, userID, guildID)
	if err == nil && existingBan != nil {
		return ErrUserAlreadyBanned
	}
	if err != nil && !errors.Is(err, ErrNotFound) { // Assuming ErrNotFound might be used by repo
		return err
	}

	// Note: In a real system, you might need to delete user permissions/roles
	// before or after adding the ban record.

	return s.repo.InsertBan(ctx, userID, guildID, reason)
}

// UnbanUser removes a ban from a user.
// Returns ErrUserNotBanned if the user is not banned.
func (s *BanService) UnbanUser(ctx context.Context, userID, guildID string) error {
	ban, err := s.repo.GetBan(ctx, userID, guildID)
	if err != nil {
		return err
	}
	if ban == nil {
		return ErrUserNotBanned
	}

	// Note: In a real system, you would re-enable user permissions/roles here.

	return s.repo.DeleteBan(ctx, userID, guildID)
}

// IsBanned checks if a user is banned in a guild.
// Returns true if the user is banned, false if not.
func (s *BanService) IsBanned(ctx context.Context, userID, guildID string) (bool, error) {
	_, err := s.repo.GetBan(ctx, userID, guildID)
	if err != nil {
		// If we get an error that indicates "Not Found", we treat it as false
		if errors.Is(err, ErrUserNotBanned) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetBans retrieves all bans for a specific guild.
func (s *BanService) GetBans(ctx context.Context, guildID string) ([]Ban, error) {
	return s.repo.ListBans(ctx, guildID)
}