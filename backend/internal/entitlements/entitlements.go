package entitlements

import (
	"context"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Service provides entitlement checks for premium features and server boosts.
// It wraps PremiumService to provide a clean, purpose-specific API for feature gating.
type Service struct {
	premium *services.PremiumService
}

// New creates a new entitlements service.
func New(premium *services.PremiumService) *Service {
	return &Service{premium: premium}
}

// CheckSubscriptionTier returns the user's current subscription tier.
func (s *Service) CheckSubscriptionTier(ctx context.Context, userID uuid.UUID) (models.PremiumTier, error) {
	status, err := s.premium.GetUserPremiumStatus(ctx, userID)
	if err != nil {
		return models.TierFree, err
	}
	return status.Tier, nil
}

// HasFeature checks whether a user has access to a specific premium feature.
func (s *Service) HasFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	return s.premium.CheckFeatureAccess(ctx, userID, feature)
}

// GetServerBoostLevel returns the current boost level for a server.
func (s *Service) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (int, error) {
	perks, err := s.premium.GetServerBoostLevel(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if perks == nil {
		return 0, nil
	}
	return perks.Level, nil
}

// GetServerPerks returns the full perks struct for a server.
func (s *Service) GetServerPerks(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	return s.premium.CalculateServerPerks(ctx, serverID)
}

// IsSubscribed returns true if the user has any paid subscription (basic or premium).
func (s *Service) IsSubscribed(ctx context.Context, userID uuid.UUID) (bool, error) {
	tier, err := s.CheckSubscriptionTier(ctx, userID)
	if err != nil {
		return false, err
	}
	return tier != models.TierFree, nil
}

// IsPremium returns true if the user has the premium tier specifically.
func (s *Service) IsPremium(ctx context.Context, userID uuid.UUID) (bool, error) {
	tier, err := s.CheckSubscriptionTier(ctx, userID)
	if err != nil {
		return false, err
	}
	return tier == models.TierPremium, nil
}

// GetUploadLimit returns the max file upload size in bytes for a server.
func (s *Service) GetUploadLimit(ctx context.Context, serverID uuid.UUID) (int64, error) {
	perks, err := s.premium.CalculateServerPerks(ctx, serverID)
	if err != nil {
		return 8 * 1024 * 1024, err // default 8MB on error
	}
	return perks.FileUploadLimit, nil
}

// GetEmojiLimit returns the max emoji count for a server.
func (s *Service) GetEmojiLimit(ctx context.Context, serverID uuid.UUID) (int, error) {
	perks, err := s.premium.CalculateServerPerks(ctx, serverID)
	if err != nil {
		return 50, err // default 50 on error
	}
	return perks.EmojiLimit, nil
}
