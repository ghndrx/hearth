package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v74"
	stripeclient "github.com/stripe/stripe-go/v74/client"
	"github.com/stripe/stripe-go/v74/webhook"

	"hearth/internal/models"
)

var (
	ErrCustomerNotFound     = errors.New("billing customer not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrWebhookInvalid       = errors.New("invalid webhook payload")
	ErrStripeNotConfigured  = errors.New("stripe is not configured")
)

// BillingService handles payment processing and billing operations.
// When stripeKey is empty, it operates in mock mode (development only).
type BillingService struct {
	repo          PremiumRepository
	stripe        *stripeclient.API
	priceBasic    string
	pricePremium  string
	webhookSecret string
	isMock        bool
}

// NewBillingService creates a new billing service.
// If stripeKey is empty, the service runs in mock mode.
func NewBillingService(repo PremiumRepository, stripeKey, webhookSecret, priceBasic, pricePremium string) *BillingService {
	isMock := stripeKey == ""
	var sc *stripeclient.API
	if !isMock {
		sc = &stripeclient.API{}
		sc.Init(stripeKey, nil)
	}
	return &BillingService{
		repo:          repo,
		stripe:        sc,
		priceBasic:    priceBasic,
		pricePremium:  pricePremium,
		webhookSecret: webhookSecret,
		isMock:        isMock,
	}
}

func (s *BillingService) priceIDForTier(tier models.PremiumTier) string {
	switch tier {
	case models.TierBasic:
		return s.priceBasic
	case models.TierPremium:
		return s.pricePremium
	default:
		return ""
	}
}

// CreateCustomer creates a billing customer for a user.
func (s *BillingService) CreateCustomer(ctx context.Context, userID uuid.UUID, email string) (*models.Customer, error) {
	customer := &models.Customer{
		ID:        uuid.New().String(),
		UserID:    userID,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if !s.isMock {
		params := &stripe.CustomerParams{
			Email: stripe.String(email),
		}
		params.AddMetadata("user_id", userID.String())
		c, err := s.stripe.Customers.New(params)
		if err != nil {
			return nil, fmt.Errorf("failed to create stripe customer: %w", err)
		}
		customer.ExternalID = c.ID
	} else {
		customer.ExternalID = "mock_cus_" + uuid.New().String()
	}

	if err := s.repo.CreateBillingCustomer(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	return customer, nil
}

// GetOrCreateCustomer gets or creates a billing customer.
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

// CreateSubscription creates a paid subscription.
func (s *BillingService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error) {
	price := s.priceIDForTier(tier)
	if price == "" && tier != models.TierFree {
		if s.isMock {
			price = "mock_price_" + string(tier)
		} else {
			return nil, ErrInvalidTier
		}
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        tier,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: models.TierBoostsTotal[tier],
		NextBilling: &periodEnd,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if !s.isMock {
		cust, err := s.GetOrCreateCustomer(ctx, userID, "") // email fetched from user repo in real flow
		if err != nil {
			return nil, fmt.Errorf("failed to get customer: %w", err)
		}

		// Attach payment method if provided
		if paymentMethodID != "" {
			_, err := s.stripe.PaymentMethods.Attach(paymentMethodID, &stripe.PaymentMethodAttachParams{
				Customer: stripe.String(cust.ExternalID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to attach payment method: %w", err)
			}
			// Set as default payment method
			_, err = s.stripe.Customers.Update(cust.ExternalID, &stripe.CustomerParams{
				InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
					DefaultPaymentMethod: stripe.String(paymentMethodID),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to set default payment method: %w", err)
			}
		}

		stripeSub, err := s.stripe.Subscriptions.New(&stripe.SubscriptionParams{
			Customer: stripe.String(cust.ExternalID),
			Items: []*stripe.SubscriptionItemsParams{
				{Price: stripe.String(price)},
			},
			PaymentBehavior: stripe.String("default_incomplete"),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create stripe subscription: %w", err)
		}

		sub.StripeSubscriptionID = stripeSub.ID
		sub.ExternalID = cust.ExternalID
		if stripeSub.CurrentPeriodStart > 0 {
			start := time.Unix(stripeSub.CurrentPeriodStart, 0)
			sub.CurrentPeriodStart = &start
		}
		if stripeSub.CurrentPeriodEnd > 0 {
			end := time.Unix(stripeSub.CurrentPeriodEnd, 0)
			sub.CurrentPeriodEnd = &end
			sub.NextBilling = &end
		}
	} else {
		sub.StripeSubscriptionID = "mock_sub_" + uuid.New().String()[:24]
		sub.ExternalID = "mock_cus_" + uuid.New().String()
		sub.CurrentPeriodStart = &now
		sub.CurrentPeriodEnd = &periodEnd
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	if err := s.repo.UpdateUserPremiumTier(ctx, userID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// CancelSubscription cancels a subscription.
func (s *BillingService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionNotFound
	}

	if !s.isMock && sub.StripeSubscriptionID != "" {
		_, err := s.stripe.Subscriptions.Cancel(sub.StripeSubscriptionID, &stripe.SubscriptionCancelParams{})
		if err != nil {
			// If already canceled on Stripe side, continue
			var stripeErr *stripe.Error
			if !errors.As(err, &stripeErr) || stripeErr.Code != stripe.ErrorCodeResourceMissing {
				return fmt.Errorf("failed to cancel stripe subscription: %w", err)
			}
		}
	}

	now := time.Now()
	sub.Status = models.SubStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	return nil
}

// HandleWebhook processes a billing provider webhook.
func (s *BillingService) HandleWebhook(ctx context.Context, provider string, payload []byte, signature string) error {
	if provider != "stripe" {
		return ErrWebhookInvalid
	}
	if len(payload) == 0 {
		return ErrWebhookInvalid
	}

	if s.isMock {
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payload, &event); err != nil {
			return ErrWebhookInvalid
		}
		log.Printf("[MOCK] Received webhook event: %s", event.Type)
		return nil
	}

	// Verify signature and parse event
	var event stripe.Event
	var err error
	if s.webhookSecret != "" {
		event, err = webhook.ConstructEvent(payload, signature, s.webhookSecret)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrWebhookInvalid, err)
		}
	} else {
		if err := json.Unmarshal(payload, &event); err != nil {
			return ErrWebhookInvalid
		}
	}

	switch event.Type {
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	default:
		return nil
	}
}

func (s *BillingService) handleInvoicePaid(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	if invoice.Subscription == nil {
		return nil
	}

	sub, err := s.repo.GetSubscriptionByStripeID(ctx, invoice.Subscription.ID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	periodStart := time.Unix(invoice.PeriodStart, 0)
	periodEnd := time.Unix(invoice.PeriodEnd, 0)
	sub.CurrentPeriodStart = &periodStart
	sub.CurrentPeriodEnd = &periodEnd
	sub.NextBilling = &periodEnd
	sub.Status = models.SubStatusActive
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	billingInvoice := &models.BillingInvoice{
		ID:          uuid.New().String(),
		UserID:      sub.UserID,
		ExternalID:  invoice.ID,
		Amount:      int(invoice.AmountPaid),
		Currency:    string(invoice.Currency),
		Status:      "paid",
		Description: fmt.Sprintf("%s subscription", sub.Tier),
		PaidAt:      timePtr(time.Now()),
		CreatedAt:   time.Now(),
	}

	return s.repo.CreateBillingInvoice(ctx, billingInvoice)
}

func (s *BillingService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	localSub, err := s.repo.GetSubscriptionByStripeID(ctx, sub.ID)
	if err != nil {
		return err
	}
	if localSub == nil {
		return nil
	}

	now := time.Now()
	localSub.Status = models.SubStatusCanceled
	localSub.CanceledAt = &now
	localSub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, localSub); err != nil {
		return err
	}

	return s.repo.UpdateUserPremiumTier(ctx, localSub.UserID, models.TierFree)
}

func (s *BillingService) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	localSub, err := s.repo.GetSubscriptionByStripeID(ctx, sub.ID)
	if err != nil {
		return err
	}
	if localSub == nil {
		return nil
	}

	periodStart := time.Unix(sub.CurrentPeriodStart, 0)
	periodEnd := time.Unix(sub.CurrentPeriodEnd, 0)
	localSub.CurrentPeriodStart = &periodStart
	localSub.CurrentPeriodEnd = &periodEnd
	localSub.UpdatedAt = time.Now()

	switch sub.Status {
	case stripe.SubscriptionStatusPastDue:
		localSub.Status = models.SubStatusPastDue
	case stripe.SubscriptionStatusActive:
		localSub.Status = models.SubStatusActive
	case stripe.SubscriptionStatusCanceled:
		localSub.Status = models.SubStatusCanceled
	}

	return s.repo.UpdateSubscription(ctx, localSub)
}

// ProcessPayment processes a one-time payment.
func (s *BillingService) ProcessPayment(ctx context.Context, userID uuid.UUID, amount int, currency string) error {
	if s.isMock {
		return nil
	}

	cust, err := s.GetOrCreateCustomer(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
		Customer: stripe.String(cust.ExternalID),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	_, err = s.stripe.PaymentIntents.New(params)
	if err != nil {
		return fmt.Errorf("failed to create payment intent: %w", err)
	}

	return nil
}

// UpdatePaymentMethod updates the default payment method.
func (s *BillingService) UpdatePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	if s.isMock {
		return nil
	}

	cust, err := s.GetOrCreateCustomer(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	_, err = s.stripe.PaymentMethods.Attach(paymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(cust.ExternalID),
	})
	if err != nil {
		return fmt.Errorf("failed to attach payment method: %w", err)
	}

	_, err = s.stripe.Customers.Update(cust.ExternalID, &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update customer default payment method: %w", err)
	}

	return nil
}

// CreateBillingPortalSession creates a billing portal session URL.
func (s *BillingService) CreateBillingPortalSession(ctx context.Context, userID uuid.UUID, returnURL string) (string, error) {
	if s.isMock {
		return "https://mock-billing-portal.example.com/session/" + uuid.New().String(), nil
	}

	cust, err := s.GetOrCreateCustomer(ctx, userID, "")
	if err != nil {
		return "", fmt.Errorf("failed to get customer: %w", err)
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(cust.ExternalID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := s.stripe.BillingPortalSessions.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create billing portal session: %w", err)
	}

	return session.URL, nil
}

// GiftSubscription gifts a subscription to a recipient user.
func (s *BillingService) GiftSubscription(ctx context.Context, gifterUserID, recipientUserID uuid.UUID, tier models.PremiumTier, months int) (*models.Subscription, error) {
	if tier == models.TierFree {
		return nil, ErrInvalidTier
	}

	price := s.priceIDForTier(tier)
	if price == "" && !s.isMock {
		return nil, ErrInvalidTier
	}

	now := time.Now()
	periodEnd := now.AddDate(0, months, 0)

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      recipientUserID,
		Tier:        tier,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: models.TierBoostsTotal[tier],
		NextBilling: &periodEnd,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if !s.isMock {
		cust, err := s.GetOrCreateCustomer(ctx, recipientUserID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get customer: %w", err)
		}

		stripeSub, err := s.stripe.Subscriptions.New(&stripe.SubscriptionParams{
			Customer: stripe.String(cust.ExternalID),
			Items: []*stripe.SubscriptionItemsParams{
				{Price: stripe.String(price)},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create stripe subscription: %w", err)
		}

		sub.StripeSubscriptionID = stripeSub.ID
		sub.ExternalID = cust.ExternalID
		if stripeSub.CurrentPeriodStart > 0 {
			start := time.Unix(stripeSub.CurrentPeriodStart, 0)
			sub.CurrentPeriodStart = &start
		}
		if stripeSub.CurrentPeriodEnd > 0 {
			end := time.Unix(stripeSub.CurrentPeriodEnd, 0)
			sub.CurrentPeriodEnd = &end
		}
	} else {
		sub.StripeSubscriptionID = "mock_gift_" + uuid.New().String()[:24]
		sub.ExternalID = "mock_cus_" + uuid.New().String()
		sub.CurrentPeriodStart = &now
		sub.CurrentPeriodEnd = &periodEnd
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create gift subscription: %w", err)
	}

	if err := s.repo.UpdateUserPremiumTier(ctx, recipientUserID, tier); err != nil {
		return nil, err
	}

	return sub, nil
}

// GetSubscriptionByExternalID looks up a subscription by its external (Stripe) ID.
func (s *BillingService) GetSubscriptionByExternalID(ctx context.Context, externalID string) (*models.Subscription, error) {
	return s.repo.GetSubscriptionByStripeID(ctx, externalID)
}

func calculateNextBilling(tier models.PremiumTier) *time.Time {
	if tier == models.TierFree {
		return nil
	}
	next := time.Now().AddDate(0, 1, 0)
	return &next
}
