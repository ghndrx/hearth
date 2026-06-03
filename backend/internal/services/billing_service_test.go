package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v74"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func newTestBillingService(repo *MockPremiumRepo, stripeKey string) *BillingService {
	return NewBillingService(repo, stripeKey, "", "price_basic_123", "price_premium_456")
}

func TestNewBillingService(t *testing.T) {
	t.Run("mock mode when stripe key is empty", func(t *testing.T) {
		repo := new(MockPremiumRepo)
		svc := newTestBillingService(repo, "")
		assert.True(t, svc.isMock)
		assert.Nil(t, svc.stripe)
	})

	t.Run("stripe mode when stripe key is provided", func(t *testing.T) {
		repo := new(MockPremiumRepo)
		svc := newTestBillingService(repo, "sk_test_dummy")
		assert.False(t, svc.isMock)
		assert.NotNil(t, svc.stripe)
	})
}

// ---------------------------------------------------------------------------
// CreateCustomer
// ---------------------------------------------------------------------------

func TestBillingService_CreateCustomer_MockMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()
	email := "test@example.com"

	repo.On("CreateBillingCustomer", ctx, mock.AnythingOfType("*models.Customer")).Return(nil).Once()

	customer, err := svc.CreateCustomer(ctx, userID, email)

	require.NoError(t, err)
	assert.NotNil(t, customer)
	assert.Equal(t, userID, customer.UserID)
	assert.Equal(t, email, customer.Email)
	assert.NotEmpty(t, customer.ID)
	assert.Contains(t, customer.ExternalID, "mock_cus_")
	repo.AssertExpectations(t)
}

func TestBillingService_CreateCustomer_MockMode_RepoError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateBillingCustomer", ctx, mock.AnythingOfType("*models.Customer")).Return(errors.New("db error")).Once()

	customer, err := svc.CreateCustomer(ctx, userID, "test@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save customer")
	assert.Nil(t, customer)
	repo.AssertExpectations(t)
}

func TestBillingService_CreateCustomer_StripeMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	// Stripe API call will fail before repo is ever touched.
	_, err := svc.CreateCustomer(ctx, userID, "test@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create stripe customer")
}

// ---------------------------------------------------------------------------
// GetOrCreateCustomer
// ---------------------------------------------------------------------------

func TestBillingService_GetOrCreateCustomer_Existing(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	existing := &models.Customer{
		ID:         "cust_123",
		UserID:     userID,
		Email:      "test@example.com",
		ExternalID: "mock_cus_123",
	}

	repo.On("GetBillingCustomer", ctx, userID).Return(existing, nil).Once()

	customer, err := svc.GetOrCreateCustomer(ctx, userID, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, existing, customer)
	repo.AssertExpectations(t)
}

func TestBillingService_GetOrCreateCustomer_New(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetBillingCustomer", ctx, userID).Return(nil, nil).Once()
	repo.On("CreateBillingCustomer", ctx, mock.AnythingOfType("*models.Customer")).Return(nil).Once()

	customer, err := svc.GetOrCreateCustomer(ctx, userID, "test@example.com")

	require.NoError(t, err)
	assert.NotNil(t, customer)
	assert.Equal(t, userID, customer.UserID)
	repo.AssertExpectations(t)
}

func TestBillingService_GetOrCreateCustomer_GetError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetBillingCustomer", ctx, userID).Return(nil, errors.New("db error")).Once()

	customer, err := svc.GetOrCreateCustomer(ctx, userID, "test@example.com")

	assert.Error(t, err)
	assert.Nil(t, customer)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateSubscription
// ---------------------------------------------------------------------------

func TestBillingService_CreateSubscription_MockMode_Basic(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(nil).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierBasic, "")

	require.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, userID, sub.UserID)
	assert.Equal(t, models.TierBasic, sub.Tier)
	assert.Equal(t, models.SubStatusActive, sub.Status)
	assert.Equal(t, models.TierBoostsTotal[models.TierBasic], sub.BoostsTotal)
	assert.Equal(t, 0, sub.BoostsUsed)
	assert.NotNil(t, sub.NextBilling)
	assert.Contains(t, sub.StripeSubscriptionID, "mock_sub_")
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_MockMode_Premium(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierPremium).Return(nil).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierPremium, "pm_123")

	require.NoError(t, err)
	assert.Equal(t, models.TierPremium, sub.Tier)
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_MockMode_Nitro(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierNitro).Return(nil).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierNitro, "")

	require.NoError(t, err)
	assert.Equal(t, models.TierNitro, sub.Tier)
	assert.Contains(t, sub.StripeSubscriptionID, "mock_sub_")
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_InvalidTier_NonMock(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	// TierNitro has no configured price ID and we're not in mock mode.
	sub, err := svc.CreateSubscription(ctx, userID, models.TierNitro, "")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTier, err)
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_MockMode_RepoError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(errors.New("db error")).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierBasic, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save subscription")
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_MockMode_UpdateTierError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(errors.New("tier update failed")).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierBasic, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tier update failed")
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_CreateSubscription_MockMode_FreeTier(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierFree).Return(nil).Once()

	sub, err := svc.CreateSubscription(ctx, userID, models.TierFree, "")

	require.NoError(t, err)
	assert.Equal(t, models.TierFree, sub.Tier)
	assert.Equal(t, 0, sub.BoostsTotal)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CancelSubscription
// ---------------------------------------------------------------------------

func TestBillingService_CancelSubscription_Existing_Mock(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()

	err := svc.CancelSubscription(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, models.SubStatusCanceled, sub.Status)
	assert.NotNil(t, sub.CanceledAt)
	repo.AssertExpectations(t)
}

func TestBillingService_CancelSubscription_NotFound(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil).Once()

	err := svc.CancelSubscription(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrSubscriptionNotFound, err)
	repo.AssertExpectations(t)
}

func TestBillingService_CancelSubscription_GetError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, errors.New("db error")).Once()

	err := svc.CancelSubscription(ctx, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
	repo.AssertExpectations(t)
}

func TestBillingService_CancelSubscription_StripeMode_NoStripeSubID(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: "", // empty -> no stripe API call
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()

	err := svc.CancelSubscription(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, models.SubStatusCanceled, sub.Status)
	repo.AssertExpectations(t)
}

func TestBillingService_CancelSubscription_StripeMode_WithStripeSubID(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: "sub_123",
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil).Once()

	err := svc.CancelSubscription(ctx, userID)

	// Stripe cancel fails because we're using a dummy key.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cancel stripe subscription")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// HandleWebhook
// ---------------------------------------------------------------------------

func TestBillingService_HandleWebhook_MockMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()

	payload := []byte(`{"type":"invoice.paid"}`)
	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
}

func TestBillingService_HandleWebhook_InvalidProvider(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()

	err := svc.HandleWebhook(ctx, "paypal", []byte(`{}`), "")

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookInvalid, err)
}

func TestBillingService_HandleWebhook_EmptyPayload(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()

	err := svc.HandleWebhook(ctx, "stripe", []byte{}, "")

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookInvalid, err)
}

func TestBillingService_HandleWebhook_InvalidJSON_Mock(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()

	err := svc.HandleWebhook(ctx, "stripe", []byte(`{invalid`), "")

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookInvalid, err)
}

func TestBillingService_HandleWebhook_StripeMode_InvalidJSON(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	err := svc.HandleWebhook(ctx, "stripe", []byte(`{invalid json`), "")

	assert.Error(t, err)
	assert.Equal(t, ErrWebhookInvalid, err)
}

func TestBillingService_HandleWebhook_InvoicePaid(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()
	stripeSubID := "sub_123"
	now := time.Now()

	localSub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	invoice := stripe.Invoice{
		ID: "inv_123",
		Subscription: &stripe.Subscription{
			ID: stripeSubID,
		},
		PeriodStart: now.Unix(),
		PeriodEnd:   now.AddDate(0, 1, 0).Unix(),
		AmountPaid:  299,
		Currency:    "usd",
	}
	eventData, _ := json.Marshal(invoice)
	event := stripe.Event{
		Type: "invoice.paid",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(localSub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("CreateBillingInvoice", ctx, mock.AnythingOfType("*models.BillingInvoice")).Return(nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_InvoicePaid_NoSubscription(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID: "inv_123",
		Subscription: &stripe.Subscription{
			ID: "sub_unknown",
		},
	}
	eventData, _ := json.Marshal(invoice)
	event := stripe.Event{
		Type: "invoice.paid",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, "sub_unknown").Return(nil, nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_InvoicePaid_MissingSubInInvoice(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID:           "inv_123",
		Subscription: nil,
	}
	eventData, _ := json.Marshal(invoice)
	event := stripe.Event{
		Type: "invoice.paid",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
}

func TestBillingService_HandleWebhook_SubscriptionDeleted(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()
	stripeSubID := "sub_456"

	localSub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierPremium,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	stripeSub := stripe.Subscription{ID: stripeSubID}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(localSub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierFree).Return(nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_SubscriptionDeleted_NotFound(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	stripeSub := stripe.Subscription{ID: "sub_unknown"}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, "sub_unknown").Return(nil, nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_SubscriptionUpdated(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()
	stripeSubID := "sub_789"

	localSub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	now := time.Now()
	stripeSub := stripe.Subscription{
		ID:                 stripeSubID,
		Status:             stripe.SubscriptionStatusPastDue,
		CurrentPeriodStart: now.Unix(),
		CurrentPeriodEnd:   now.AddDate(0, 1, 0).Unix(),
	}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(localSub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	assert.Equal(t, models.SubStatusPastDue, localSub.Status)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_SubscriptionUpdated_Active(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	stripeSubID := "sub_abc"

	localSub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               uuid.New(),
		Tier:                 models.TierPremium,
		Status:               models.SubStatusPastDue,
		StripeSubscriptionID: stripeSubID,
	}

	now := time.Now()
	stripeSub := stripe.Subscription{
		ID:                 stripeSubID,
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: now.Unix(),
		CurrentPeriodEnd:   now.AddDate(0, 1, 0).Unix(),
	}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(localSub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	assert.Equal(t, models.SubStatusActive, localSub.Status)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_SubscriptionUpdated_Canceled(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	stripeSubID := "sub_def"

	localSub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               uuid.New(),
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	now := time.Now()
	stripeSub := stripe.Subscription{
		ID:                 stripeSubID,
		Status:             stripe.SubscriptionStatusCanceled,
		CurrentPeriodStart: now.Unix(),
		CurrentPeriodEnd:   now.AddDate(0, 1, 0).Unix(),
	}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(localSub, nil).Once()
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	assert.Equal(t, models.SubStatusCanceled, localSub.Status)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_SubscriptionUpdated_NotFound(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	stripeSub := stripe.Subscription{
		ID:     "sub_unknown",
		Status: stripe.SubscriptionStatusActive,
	}
	eventData, _ := json.Marshal(stripeSub)
	event := stripe.Event{
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{Raw: eventData},
	}
	payload, _ := json.Marshal(event)

	repo.On("GetSubscriptionByStripeID", ctx, "sub_unknown").Return(nil, nil).Once()

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBillingService_HandleWebhook_UnknownEvent(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	event := stripe.Event{
		Type: "charge.succeeded",
		Data: &stripe.EventData{Raw: []byte(`{}`)},
	}
	payload, _ := json.Marshal(event)

	err := svc.HandleWebhook(ctx, "stripe", payload, "")

	require.NoError(t, err) // unknown events are silently ignored
}

// ---------------------------------------------------------------------------
// ProcessPayment
// ---------------------------------------------------------------------------

func TestBillingService_ProcessPayment_MockMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	err := svc.ProcessPayment(ctx, userID, 1000, "usd")

	require.NoError(t, err)
}

func TestBillingService_ProcessPayment_StripeMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	// Return an existing customer so the code reaches the Stripe PaymentIntents call.
	repo.On("GetBillingCustomer", ctx, userID).Return(&models.Customer{
		UserID:     userID,
		ExternalID: "cus_123",
	}, nil).Once()

	err := svc.ProcessPayment(ctx, userID, 1000, "usd")

	// Fails when trying to create stripe payment intent.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create payment intent")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdatePaymentMethod
// ---------------------------------------------------------------------------

func TestBillingService_UpdatePaymentMethod_MockMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	err := svc.UpdatePaymentMethod(ctx, userID, "pm_123")

	require.NoError(t, err)
}

func TestBillingService_UpdatePaymentMethod_StripeMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	// Return an existing customer so the code reaches the Stripe PaymentMethods call.
	repo.On("GetBillingCustomer", ctx, userID).Return(&models.Customer{
		UserID:     userID,
		ExternalID: "cus_123",
	}, nil).Once()

	err := svc.UpdatePaymentMethod(ctx, userID, "pm_123")

	// Fails when trying to attach payment method via Stripe.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to attach payment method")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateBillingPortalSession
// ---------------------------------------------------------------------------

func TestBillingService_CreateBillingPortalSession_MockMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	userID := uuid.New()

	url, err := svc.CreateBillingPortalSession(ctx, userID, "https://example.com/return")

	require.NoError(t, err)
	assert.Contains(t, url, "https://mock-billing-portal.example.com/session/")
}

func TestBillingService_CreateBillingPortalSession_StripeMode(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()
	userID := uuid.New()

	// Return an existing customer so the code reaches the Stripe BillingPortalSessions call.
	repo.On("GetBillingCustomer", ctx, userID).Return(&models.Customer{
		UserID:     userID,
		ExternalID: "cus_123",
	}, nil).Once()

	url, err := svc.CreateBillingPortalSession(ctx, userID, "https://example.com/return")

	// Fails when trying to create billing portal session via Stripe.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create billing portal session")
	assert.Empty(t, url)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GiftSubscription
// ---------------------------------------------------------------------------

func TestBillingService_GiftSubscription_MockMode_FreeTier(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	gifterID := uuid.New()
	recipientID := uuid.New()

	sub, err := svc.GiftSubscription(ctx, gifterID, recipientID, models.TierFree, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTier, err)
	assert.Nil(t, sub)
}

func TestBillingService_GiftSubscription_MockMode_Success(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	gifterID := uuid.New()
	recipientID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil).Once()
	repo.On("UpdateUserPremiumTier", ctx, recipientID, models.TierPremium).Return(nil).Once()

	sub, err := svc.GiftSubscription(ctx, gifterID, recipientID, models.TierPremium, 3)

	require.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, recipientID, sub.UserID)
	assert.Equal(t, models.TierPremium, sub.Tier)
	assert.Equal(t, models.SubStatusActive, sub.Status)
	assert.Equal(t, models.TierBoostsTotal[models.TierPremium], sub.BoostsTotal)
	assert.Contains(t, sub.StripeSubscriptionID, "mock_gift_")
	repo.AssertExpectations(t)
}

func TestBillingService_GiftSubscription_MockMode_RepoError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	gifterID := uuid.New()
	recipientID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(errors.New("db error")).Once()

	sub, err := svc.GiftSubscription(ctx, gifterID, recipientID, models.TierBasic, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gift subscription")
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_GiftSubscription_NonMock_InvalidTier(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "sk_test_dummy")
	ctx := context.Background()

	sub, err := svc.GiftSubscription(ctx, uuid.New(), uuid.New(), models.TierNitro, 1)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidTier, err)
	assert.Nil(t, sub)
}

// ---------------------------------------------------------------------------
// GetSubscriptionByExternalID
// ---------------------------------------------------------------------------

func TestBillingService_GetSubscriptionByExternalID_Found(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	externalID := "sub_123"

	expected := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               uuid.New(),
		StripeSubscriptionID: externalID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
	}

	repo.On("GetSubscriptionByStripeID", ctx, externalID).Return(expected, nil).Once()

	sub, err := svc.GetSubscriptionByExternalID(ctx, externalID)

	require.NoError(t, err)
	assert.Equal(t, expected, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_GetSubscriptionByExternalID_NotFound(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	externalID := "sub_123"

	repo.On("GetSubscriptionByStripeID", ctx, externalID).Return(nil, nil).Once()

	sub, err := svc.GetSubscriptionByExternalID(ctx, externalID)

	require.NoError(t, err)
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

func TestBillingService_GetSubscriptionByExternalID_RepoError(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")
	ctx := context.Background()
	externalID := "sub_123"

	repo.On("GetSubscriptionByStripeID", ctx, externalID).Return(nil, errors.New("db error")).Once()

	sub, err := svc.GetSubscriptionByExternalID(ctx, externalID)

	assert.Error(t, err)
	assert.Nil(t, sub)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// priceIDForTier
// ---------------------------------------------------------------------------

func TestBillingService_priceIDForTier(t *testing.T) {
	repo := new(MockPremiumRepo)
	svc := newTestBillingService(repo, "")

	assert.Equal(t, "price_basic_123", svc.priceIDForTier(models.TierBasic))
	assert.Equal(t, "price_premium_456", svc.priceIDForTier(models.TierPremium))
	assert.Equal(t, "", svc.priceIDForTier(models.TierFree))
	assert.Equal(t, "", svc.priceIDForTier(models.TierNitro))
}

// ---------------------------------------------------------------------------
// calculateNextBilling
// ---------------------------------------------------------------------------

func TestCalculateNextBilling(t *testing.T) {
	t.Run("free tier returns nil", func(t *testing.T) {
		next := calculateNextBilling(models.TierFree)
		assert.Nil(t, next)
	})

	t.Run("paid tier returns future date", func(t *testing.T) {
		next := calculateNextBilling(models.TierBasic)
		assert.NotNil(t, next)
		assert.True(t, next.After(time.Now()))
	})
}
