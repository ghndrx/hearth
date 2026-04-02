package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrNoSubscription     = errors.New("no active subscription")
	ErrBoostLimitReached  = errors.New("boost limit reached")
	ErrAlreadyBoosted     = errors.New("server already boosted by this user")
	ErrNotBoosted         = errors.New("server not boosted by this user")
	ErrServerLimitReached = errors.New("server boost limit reached")
	ErrInvalidTier        = errors.New("invalid subscription tier")
	ErrPaymentFailed      = errors.New("payment processing failed")
)

// ServerBoostEvent is published when a server boost is added or removed
type ServerBoostEvent struct {
	ServerID    uuid.UUID `json:"server_id"`
	UserID      uuid.UUID `json:"user_id"`
	BoostCount  int       `json:"boost_count"`
	LevelBefore int       `json:"level_before"`
	LevelAfter  int       `json:"level_after"`
	Action      string    `json:"action"` // "added" or "removed"
}

// PremiumRepository defines the interface for premium data access
type PremiumRepository interface {
	// Subscription operations
	CreateSubscription(ctx context.Context, sub *models.Subscription) error
	GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error)
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, sub *models.Subscription) error
	DeleteSubscription(ctx context.Context, userID uuid.UUID) error

	// Server boost operations
	CreateServerBoost(ctx context.Context, boost *models.ServerBoost) error
	GetServerBoost(ctx context.Context, serverID, userID uuid.UUID) (*models.ServerBoost, error)
	GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error)
	GetServerBoostCount(ctx context.Context, serverID uuid.UUID) (int, error)
	DeactivateServerBoost(ctx context.Context, serverID, userID uuid.UUID) error
	GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error)
	GetUserActiveBoostCount(ctx context.Context, userID uuid.UUID) (int, error)
	UpdateServerBoostLevel(ctx context.Context, serverID uuid.UUID, boostCount int) error
	GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error)

	// Billing operations
	CreateBillingCustomer(ctx context.Context, customer *models.Customer) error
	GetBillingCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error)
	GetBillingCustomerByExternalID(ctx context.Context, externalID string) (*models.Customer, error)
	CreateBillingInvoice(ctx context.Context, invoice *models.BillingInvoice) error
	GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error)
	CreatePaymentMethod(ctx context.Context, userID uuid.UUID, pm *models.PaymentMethod) error
	GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error)
	GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error)
	DeletePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error

	// User premium tier
	UpdateUserPremiumTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error
	GetUserPremiumTier(ctx context.Context, userID uuid.UUID) (models.PremiumTier, error)
}

// PremiumService handles premium subscription and server boost business logic
type PremiumService struct {
	repo       PremiumRepository
	userRepo   UserRepository
	serverRepo ServerRepository
	billing    *BillingService
	eventBus   EventBus
}

// NewPremiumService creates a new premium service
func NewPremiumService(
	repo PremiumRepository,
	userRepo UserRepository,
	serverRepo ServerRepository,
	billing *BillingService,
) *PremiumService {
	return &PremiumService{
		repo:       repo,
		userRepo:   userRepo,
		serverRepo: serverRepo,
		billing:    billing,
	}
}

// SetEventBus sets the event bus for publishing boost events
func (s *PremiumService) SetEventBus(eventBus EventBus) {
	s.eventBus = eventBus
}

// GetUserPremiumStatus returns the full premium status for a user
func (s *PremiumService) GetUserPremiumStatus(ctx context.Context, userID uuid.UUID) (*models.PremiumStatus, error) {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	status := &models.PremiumStatus{
		UserID:          userID,
		BoostsUsed:      0,
		BoostsTotal:     0,
		BoostsAvailable: 0,
	}

	if sub != nil {
		status.Tier = sub.Tier
		status.Status = sub.Status
		status.BoostsUsed = sub.BoostsUsed
		status.BoostsTotal = sub.BoostsTotal
		status.BoostsAvailable = sub.BoostsTotal - sub.BoostsUsed
		status.Subscription = sub
		status.ExpiresAt = sub.NextBilling
	} else {
		status.Tier = models.TierFree
		status.Status = models.SubStatusActive
	}

	features := models.GetPremiumFeatures(status.Tier)
	status.Features = features

	return status, nil
}

// CreateSubscription creates a new subscription for a user
func (s *PremiumService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) (*models.Subscription, error) {
	if tier == models.TierFree {
		return nil, ErrInvalidTier
	}

	// Check if subscription already exists
	existing, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update existing subscription
		existing.Tier = tier
		existing.Status = models.SubStatusActive
		existing.BoostsTotal = models.TierBoostsTotal[tier]
		existing.UpdatedAt = time.Now()
		if err := s.repo.UpdateSubscription(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new subscription
	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        tier,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: models.TierBoostsTotal[tier],
		NextBilling: calculateNextBilling(tier),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Update user's premium tier
	if err := s.repo.UpdateUserPremiumTier(ctx, userID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// UpdateSubscriptionTier updates the tier of an existing subscription
func (s *PremiumService) UpdateSubscriptionTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoSubscription
	}

	sub.Tier = tier
	sub.BoostsTotal = models.TierBoostsTotal[tier]
	sub.NextBilling = calculateNextBilling(tier)
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, userID, tier)
}

// CancelSubscription cancels a user's subscription
func (s *PremiumService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoSubscription
	}

	now := time.Now()
	sub.Status = models.SubStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	// Downgrade user to free tier
	return s.repo.UpdateUserPremiumTier(ctx, userID, models.TierFree)
}

// ReactivateSubscription reactivates a canceled subscription
func (s *PremiumService) ReactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrNoSubscription
	}

	if sub.Status != models.SubStatusCanceled {
		return errors.New("subscription is not canceled")
	}

	sub.Status = models.SubStatusActive
	sub.CanceledAt = nil
	sub.NextBilling = calculateNextBilling(sub.Tier)
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, userID, sub.Tier)
}

// BoostServer applies a user's server boost to a server
func (s *PremiumService) BoostServer(ctx context.Context, userID, serverID uuid.UUID) error {
	// Get user's subscription status
	status, err := s.GetUserPremiumStatus(ctx, userID)
	if err != nil {
		return err
	}

	if status.Tier == models.TierFree {
		return ErrNoSubscription
	}

	if status.BoostsAvailable <= 0 {
		return ErrBoostLimitReached
	}

	// Check if already boosting this server
	existing, err := s.repo.GetServerBoost(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if existing != nil && existing.Active {
		return ErrAlreadyBoosted
	}

	// Create/update the boost
	now := time.Now()
	boost := &models.ServerBoost{
		ID:        uuid.New(),
		ServerID:  serverID,
		UserID:    userID,
		Active:    true,
		CreatedAt: now,
		ExpiresAt: nil, // Boosts last as long as subscription is active
	}

	if err := s.repo.CreateServerBoost(ctx, boost); err != nil {
		return err
	}

	// Update subscription boost usage
	sub := status.Subscription
	sub.BoostsUsed++
	sub.UpdatedAt = time.Now()
	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	// Update server boost count and level
	boostCount, err := s.repo.GetServerBoostCount(ctx, serverID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateServerBoostLevel(ctx, serverID, boostCount); err != nil {
		return err
	}

	// Publish boost added event
	if s.eventBus != nil {
		levelBefore := models.CalculateServerLevel(boostCount - 1)
		levelAfter := models.CalculateServerLevel(boostCount)
		s.eventBus.Publish("server.boost_added", &ServerBoostEvent{
			ServerID:    serverID,
			UserID:      userID,
			BoostCount:  boostCount,
			LevelBefore: levelBefore,
			LevelAfter:  levelAfter,
			Action:      "added",
		})
	}

	return nil
}

// UnboostServer removes a user's boost from a server
func (s *PremiumService) UnboostServer(ctx context.Context, userID, serverID uuid.UUID) error {
	boost, err := s.repo.GetServerBoost(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if boost == nil || !boost.Active {
		return ErrNotBoosted
	}

	// Deactivate the boost
	if err := s.repo.DeactivateServerBoost(ctx, serverID, userID); err != nil {
		return err
	}

	// Update subscription boost usage
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub != nil && sub.BoostsUsed > 0 {
		sub.BoostsUsed--
		sub.UpdatedAt = time.Now()
		if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
			return err
		}
	}

	// Update server boost count and level
	boostCount, err := s.repo.GetServerBoostCount(ctx, serverID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateServerBoostLevel(ctx, serverID, boostCount); err != nil {
		return err
	}

	// Publish boost removed event
	if s.eventBus != nil {
		levelBefore := models.CalculateServerLevel(boostCount + 1)
		levelAfter := models.CalculateServerLevel(boostCount)
		s.eventBus.Publish("server.boost_removed", &ServerBoostEvent{
			ServerID:    serverID,
			UserID:      userID,
			BoostCount:  boostCount,
			LevelBefore: levelBefore,
			LevelAfter:  levelAfter,
			Action:      "removed",
		})
	}

	return nil
}

// GetServerBoostLevel returns the current boost level and perks for a server
func (s *PremiumService) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	return s.repo.GetServerBoostLevel(ctx, serverID)
}

// GetServerBoosts returns all active boosts on a server
func (s *PremiumService) GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error) {
	return s.repo.GetServerBoosts(ctx, serverID)
}

// GetUserBoosts returns all active boosts for a user
func (s *PremiumService) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	return s.repo.GetUserBoosts(ctx, userID)
}

// GetUserBoostsAvailable returns the number of boosts a user has available
func (s *PremiumService) GetUserBoostsAvailable(ctx context.Context, userID uuid.UUID) (int, error) {
	status, err := s.GetUserPremiumStatus(ctx, userID)
	if err != nil {
		return 0, err
	}
	return status.BoostsAvailable, nil
}

// CalculateServerPerks returns the perks for a server based on boost level
func (s *PremiumService) CalculateServerPerks(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	boostCount, err := s.repo.GetServerBoostCount(ctx, serverID)
	if err != nil {
		return nil, err
	}
	perks := models.GetServerPerks(boostCount)
	return &perks, nil
}

// CheckFeatureAccess checks if a user has access to a specific premium feature
func (s *PremiumService) CheckFeatureAccess(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	status, err := s.GetUserPremiumStatus(ctx, userID)
	if err != nil {
		return false, err
	}

	if status.Tier == models.TierFree {
		return false, nil
	}

	features := status.Features
	switch feature {
	case "cross_server_emojis":
		return features.CrossServerEmojis, nil
	case "high_quality_video":
		return features.HighQualityVideo, nil
	case "custom_discriminator":
		return features.CustomDiscriminator, nil
	case "early_access":
		return features.EarlyAccess, nil
	case "premium_badge":
		return features.PremiumBadge, nil
	case "server_boost":
		return features.ServerBoosts > 0, nil
	case "larger_file_uploads":
		return features.FileUploadSize > 8*1024*1024, nil
	default:
		return false, nil
	}
}

// GetBillingInvoices returns billing invoices for a user
func (s *PremiumService) GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.GetBillingInvoices(ctx, userID, limit)
}

// GetPaymentMethods returns payment methods for a user
func (s *PremiumService) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	return s.repo.GetPaymentMethods(ctx, userID)
}
