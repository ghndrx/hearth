package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrUnknownEvent     = errors.New("unknown webhook event type")
	ErrMissingConfig    = errors.New("stripe configuration is missing")
)

// StripeConfig holds Stripe-related configuration.
type StripeConfig struct {
	SecretKey      string
	WebhookSecret  string
	BasicPriceID   string
	PremiumPriceID string
}

// StripeClient handles Stripe API interactions.
// In production, this would use the stripe-go SDK.
// Currently implements the interface with local logic for development.
type StripeClient struct {
	config  StripeConfig
	repo    services.PremiumRepository
	premium *services.PremiumService
}

// NewStripeClient creates a new Stripe client.
func NewStripeClient(config StripeConfig, repo services.PremiumRepository, premium *services.PremiumService) *StripeClient {
	return &StripeClient{
		config:  config,
		repo:    repo,
		premium: premium,
	}
}

// PriceIDForTier returns the Stripe Price ID for a subscription tier.
func (s *StripeClient) PriceIDForTier(tier models.PremiumTier) string {
	switch tier {
	case models.TierBasic:
		return s.config.BasicPriceID
	case models.TierPremium:
		return s.config.PremiumPriceID
	default:
		return ""
	}
}

// CreateSubscription creates a Stripe subscription for a user.
// In production, this would call stripe.Subscription.New().
func (s *StripeClient) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, customerID string) (*models.Subscription, error) {
	priceID := s.PriceIDForTier(tier)
	if priceID == "" {
		return nil, services.ErrInvalidTier
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	stripeSubID := "sub_" + uuid.New().String()[:24]

	sub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 tier,
		Status:               models.SubStatusActive,
		BoostsUsed:           0,
		BoostsTotal:          models.TierBoostsTotal[tier],
		NextBilling:          &periodEnd,
		CurrentPeriodStart:   &now,
		CurrentPeriodEnd:     &periodEnd,
		StripeSubscriptionID: stripeSubID,
		ExternalID:           customerID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	if err := s.repo.UpdateUserPremiumTier(ctx, userID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// CancelSubscription cancels a Stripe subscription.
// In production, this would call stripe.Subscription.Cancel().
func (s *StripeClient) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return services.ErrNoSubscription
	}

	now := time.Now()
	sub.Status = models.SubStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, userID, models.TierFree)
}

// UpdateSubscription changes the tier of a Stripe subscription.
// In production, this would call stripe.Subscription.Update().
func (s *StripeClient) UpdateSubscription(ctx context.Context, userID uuid.UUID, newTier models.PremiumTier) error {
	priceID := s.PriceIDForTier(newTier)
	if priceID == "" {
		return services.ErrInvalidTier
	}

	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return services.ErrNoSubscription
	}

	sub.Tier = newTier
	sub.BoostsTotal = models.TierBoostsTotal[newTier]
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, userID, newTier)
}

// WebhookEvent represents a parsed Stripe webhook event.
type WebhookEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// InvoiceData represents the relevant fields from a Stripe invoice event.
type InvoiceData struct {
	Object struct {
		ID             string `json:"id"`
		CustomerID     string `json:"customer"`
		SubscriptionID string `json:"subscription"`
		AmountPaid     int    `json:"amount_paid"`
		Currency       string `json:"currency"`
		Status         string `json:"status"`
		PeriodStart    int64  `json:"period_start"`
		PeriodEnd      int64  `json:"period_end"`
	} `json:"object"`
}

// SubscriptionData represents the relevant fields from a Stripe subscription event.
type SubscriptionData struct {
	Object struct {
		ID                 string `json:"id"`
		CustomerID         string `json:"customer"`
		Status             string `json:"status"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
	} `json:"object"`
}

// ConstructWebhookEvent verifies the webhook signature and parses the event.
func (s *StripeClient) ConstructWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	if s.config.WebhookSecret == "" {
		// In development mode without webhook secret, just parse the event
		var event WebhookEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse webhook: %w", err)
		}
		return &event, nil
	}

	// Verify signature: Stripe uses t=timestamp,v1=signature format
	if !verifyStripeSignature(payload, signature, s.config.WebhookSecret) {
		return nil, ErrInvalidSignature
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	return &event, nil
}

// HandleInvoicePaid processes a successful invoice payment from Stripe.
func (s *StripeClient) HandleInvoicePaid(ctx context.Context, data json.RawMessage) error {
	var invoice InvoiceData
	if err := json.Unmarshal(data, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice data: %w", err)
	}

	// Look up subscription by Stripe subscription ID
	sub, err := s.repo.GetSubscriptionByStripeID(ctx, invoice.Object.SubscriptionID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Subscription not found locally, skip
	}

	// Update subscription period
	periodStart := time.Unix(invoice.Object.PeriodStart, 0)
	periodEnd := time.Unix(invoice.Object.PeriodEnd, 0)
	sub.CurrentPeriodStart = &periodStart
	sub.CurrentPeriodEnd = &periodEnd
	sub.NextBilling = &periodEnd
	sub.Status = models.SubStatusActive
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	// Record the invoice
	billingInvoice := &models.BillingInvoice{
		ID:          uuid.New().String(),
		UserID:      sub.UserID,
		ExternalID:  invoice.Object.ID,
		Amount:      invoice.Object.AmountPaid,
		Currency:    strings.ToUpper(invoice.Object.Currency),
		Status:      "paid",
		Description: fmt.Sprintf("%s subscription", sub.Tier),
		PaidAt:      timePtr(time.Now()),
		CreatedAt:   time.Now(),
	}

	return s.repo.CreateBillingInvoice(ctx, billingInvoice)
}

// HandleCustomerSubscriptionDeleted processes a subscription cancellation from Stripe.
func (s *StripeClient) HandleCustomerSubscriptionDeleted(ctx context.Context, data json.RawMessage) error {
	var subData SubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := s.repo.GetSubscriptionByStripeID(ctx, subData.Object.ID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Not found locally, skip
	}

	now := time.Now()
	sub.Status = models.SubStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, sub.UserID, models.TierFree)
}

// HandleWebhookEvent routes a webhook event to the appropriate handler.
func (s *StripeClient) HandleWebhookEvent(ctx context.Context, event *WebhookEvent) error {
	switch event.Type {
	case "invoice.paid":
		return s.HandleInvoicePaid(ctx, event.Data)
	case "customer.subscription.deleted":
		return s.HandleCustomerSubscriptionDeleted(ctx, event.Data)
	case "customer.subscription.updated":
		// Handle subscription updates (e.g., plan changes)
		return s.handleSubscriptionUpdated(ctx, event.Data)
	default:
		// Ignore unhandled event types
		return nil
	}
}

func (s *StripeClient) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var subData SubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return fmt.Errorf("failed to parse subscription data: %w", err)
	}

	sub, err := s.repo.GetSubscriptionByStripeID(ctx, subData.Object.ID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	periodStart := time.Unix(subData.Object.CurrentPeriodStart, 0)
	periodEnd := time.Unix(subData.Object.CurrentPeriodEnd, 0)
	sub.CurrentPeriodStart = &periodStart
	sub.CurrentPeriodEnd = &periodEnd
	sub.UpdatedAt = time.Now()

	if subData.Object.Status == "past_due" {
		sub.Status = models.SubStatusPastDue
	} else if subData.Object.Status == "active" {
		sub.Status = models.SubStatusActive
	}

	return s.repo.UpdateSubscription(ctx, sub)
}

// verifyStripeSignature verifies a Stripe webhook signature.
func verifyStripeSignature(payload []byte, header, secret string) bool {
	if header == "" || secret == "" {
		return false
	}

	// Parse "t=timestamp,v1=signature" format
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return false
	}

	// Check timestamp tolerance (5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return true
		}
	}
	return false
}

func timePtr(t time.Time) *time.Time {
	return &t
}
