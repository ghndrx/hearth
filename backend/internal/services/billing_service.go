package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrCustomerNotFound     = errors.New("billing customer not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrWebhookInvalid       = errors.New("invalid webhook payload")
)

// BillingService handles payment processing and billing operations
// This is a STUB implementation - integrate with real payment provider (Stripe/Paddle) before production
type BillingService struct {
	repo      PremiumRepository
	stripeKey string
	isProd    bool
}

// NewBillingService creates a new billing service
// stripeKey can be empty for mock mode (development only)
func NewBillingService(repo PremiumRepository, stripeKey string, isProd bool) *BillingService {
	return &BillingService{
		repo:      repo,
		stripeKey: stripeKey,
		isProd:    isProd,
	}
}

// CreateCustomer creates a billing customer for a user
func (s *BillingService) CreateCustomer(ctx context.Context, userID uuid.UUID, email string) (*models.Customer, error) {
	// STUB: Create a mock customer record
	// In production, integrate with Stripe: customer.New(params)
	customer := &models.Customer{
		ID:         uuid.New().String(),
		UserID:     userID,
		Email:      email,
		ExternalID: "mock_" + uuid.New().String(),
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateBillingCustomer(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	return customer, nil
}

// GetOrCreateCustomer gets or creates a billing customer
func (s *BillingService) GetOrCreateCustomer(ctx context.Context, userID uuid.UUID, email string) (*models.Customer, error) {
	cust, err := s.repo.GetBillingCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cust != nil {
		return cust, nil
	}
	return s.CreateCustomer(ctx, userID, email)
}

// CreateSubscription creates a paid subscription
// STUB: This creates a mock subscription - integrate with real payment provider before production
func (s *BillingService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error) {
	// Validate tier
	price := models.PremiumTierPricing[tier]
	if price == 0 && tier != models.TierFree {
		return nil, ErrInvalidTier
	}

	// Create mock subscription
	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        tier,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: models.TierBoostsTotal[tier],
		NextBilling: calculateNextBilling(tier),
		ExternalID:  "mock_sub_" + uuid.New().String(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Update user premium tier
	if err := s.repo.UpdateUserPremiumTier(ctx, userID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// CancelSubscription cancels a subscription
// STUB: This cancels a mock subscription - integrate with real payment provider before production
func (s *BillingService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	now := time.Now()
	sub.Status = models.SubStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	// User stays on premium tier until period ends
	return nil
}

// HandleWebhook processes a billing provider webhook
// STUB: This is a no-op stub - implement real webhook handling before production
func (s *BillingService) HandleWebhook(ctx context.Context, provider string, payload []byte, signature string) error {
	if len(payload) == 0 {
		return ErrWebhookInvalid
	}

	// STUB: Parse webhook event
	// In production:
	// - Verify webhook signature (Stripe-Signature header)
	// - Parse event based on provider
	// - Handle relevant event types

	var event struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return ErrWebhookInvalid
	}

	// Log webhook receipt (in production, process events)
	fmt.Printf("Received webhook event: %s\n", event.Type)
	return nil
}

// ProcessPayment processes a one-time payment
// STUB: Not implemented - integrate with real payment provider before production
func (s *BillingService) ProcessPayment(ctx context.Context, userID uuid.UUID, amount int, currency string) error {
	// STUB: Implement one-time payment processing
	return errors.New("payment processing not implemented - integrate with Stripe/Paddle before production")
}

// UpdatePaymentMethod updates the default payment method
// STUB: Not implemented - integrate with real payment provider before production
func (s *BillingService) UpdatePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	// STUB: Implement payment method update
	return errors.New("payment method update not implemented - integrate with Stripe/Paddle before production")
}

// CreateBillingPortalSession creates a billing portal session URL
// STUB: Not implemented - integrate with real payment provider before production
func (s *BillingService) CreateBillingPortalSession(ctx context.Context, userID uuid.UUID, returnURL string) (string, error) {
	// STUB: Return mock URL
	return "https://mock-billing-portal.example.com/session/" + uuid.New().String(), nil
}

// GiftSubscription gifts a subscription to a recipient user
// STUB: Not implemented - integrate with real payment provider before production
func (s *BillingService) GiftSubscription(ctx context.Context, gifterUserID, recipientUserID uuid.UUID, tier models.PremiumTier, months int) (*models.Subscription, error) {
	if tier == models.TierFree {
		return nil, ErrInvalidTier
	}

	// Create a gifted subscription for the recipient
	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      recipientUserID,
		Tier:        tier,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: models.TierBoostsTotal[tier],
		NextBilling: calculateNextBillingForMonths(tier, months),
		ExternalID:  "mock_gift_" + uuid.New().String(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create gift subscription: %w", err)
	}

	// Update recipient's premium tier
	if err := s.repo.UpdateUserPremiumTier(ctx, recipientUserID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// calculateNextBillingForMonths returns the billing date after a given number of months
func calculateNextBillingForMonths(tier models.PremiumTier, months int) *time.Time {
	if tier == models.TierFree || months <= 0 {
		return nil
	}
	next := time.Now().AddDate(0, months, 0)
	return &next
}

// GetSubscriptionByExternalID looks up a subscription by its external (Stripe) ID
// STUB: Not implemented - implement for real webhook handling
func (s *BillingService) GetSubscriptionByExternalID(ctx context.Context, externalID string) (*models.Subscription, error) {
	// STUB: This would query by external_id in production
	return nil, ErrSubscriptionNotFound
}

// calculateNextBilling returns the next billing date based on tier
func calculateNextBilling(tier models.PremiumTier) *time.Time {
	if tier == models.TierFree {
		return nil
	}
	// Monthly billing
	next := time.Now().AddDate(0, 1, 0)
	return &next
}
