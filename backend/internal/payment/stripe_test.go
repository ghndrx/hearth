package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// mockPremiumRepo implements services.PremiumRepository for testing
type mockPremiumRepo struct {
	mock.Mock
}

func (m *mockPremiumRepo) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	return m.Called(ctx, sub).Error(0)
}
func (m *mockPremiumRepo) GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}
func (m *mockPremiumRepo) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*models.Subscription, error) {
	args := m.Called(ctx, stripeSubID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}
func (m *mockPremiumRepo) UpdateSubscription(ctx context.Context, sub *models.Subscription) error {
	return m.Called(ctx, sub).Error(0)
}
func (m *mockPremiumRepo) DeleteSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockPremiumRepo) CreateServerBoost(ctx context.Context, boost *models.ServerBoost) error {
	return m.Called(ctx, boost).Error(0)
}
func (m *mockPremiumRepo) GetServerBoost(ctx context.Context, serverID, userID uuid.UUID) (*models.ServerBoost, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerBoost), args.Error(1)
}
func (m *mockPremiumRepo) GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerBoost), args.Error(1)
}
func (m *mockPremiumRepo) GetServerBoostCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}
func (m *mockPremiumRepo) DeactivateServerBoost(ctx context.Context, serverID, userID uuid.UUID) error {
	return m.Called(ctx, serverID, userID).Error(0)
}
func (m *mockPremiumRepo) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerBoost), args.Error(1)
}
func (m *mockPremiumRepo) GetUserActiveBoostCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
func (m *mockPremiumRepo) UpdateServerBoostLevel(ctx context.Context, serverID uuid.UUID, boostCount int) error {
	return m.Called(ctx, serverID, boostCount).Error(0)
}
func (m *mockPremiumRepo) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerPerks), args.Error(1)
}
func (m *mockPremiumRepo) CreateBillingCustomer(ctx context.Context, customer *models.Customer) error {
	return m.Called(ctx, customer).Error(0)
}
func (m *mockPremiumRepo) GetBillingCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}
func (m *mockPremiumRepo) GetBillingCustomerByExternalID(ctx context.Context, externalID string) (*models.Customer, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}
func (m *mockPremiumRepo) CreateBillingInvoice(ctx context.Context, invoice *models.BillingInvoice) error {
	return m.Called(ctx, invoice).Error(0)
}
func (m *mockPremiumRepo) GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BillingInvoice), args.Error(1)
}
func (m *mockPremiumRepo) CreatePaymentMethod(ctx context.Context, userID uuid.UUID, pm *models.PaymentMethod) error {
	return m.Called(ctx, userID, pm).Error(0)
}
func (m *mockPremiumRepo) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentMethod), args.Error(1)
}
func (m *mockPremiumRepo) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaymentMethod), args.Error(1)
}
func (m *mockPremiumRepo) DeletePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	return m.Called(ctx, userID, paymentMethodID).Error(0)
}
func (m *mockPremiumRepo) UpdateUserPremiumTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error {
	return m.Called(ctx, userID, tier).Error(0)
}
func (m *mockPremiumRepo) GetUserPremiumTier(ctx context.Context, userID uuid.UUID) (models.PremiumTier, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(models.PremiumTier), args.Error(1)
}

func newTestClient() (*StripeClient, *mockPremiumRepo) {
	repo := new(mockPremiumRepo)
	config := StripeConfig{
		SecretKey:      "sk_test_123",
		WebhookSecret:  "whsec_test_123",
		BasicPriceID:   "price_basic_299",
		PremiumPriceID: "price_premium_999",
	}
	client := NewStripeClient(config, repo, nil)
	return client, repo
}

func TestCreateSubscription_Basic(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierBasic).Return(nil)

	sub, err := client.CreateSubscription(ctx, userID, models.TierBasic, "cus_test123")

	require.NoError(t, err)
	assert.Equal(t, models.TierBasic, sub.Tier)
	assert.Equal(t, models.SubStatusActive, sub.Status)
	assert.Equal(t, 2, sub.BoostsTotal)
	assert.NotEmpty(t, sub.StripeSubscriptionID)
	assert.NotNil(t, sub.CurrentPeriodStart)
	assert.NotNil(t, sub.CurrentPeriodEnd)
	repo.AssertExpectations(t)
}

func TestCreateSubscription_Premium(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierPremium).Return(nil)

	sub, err := client.CreateSubscription(ctx, userID, models.TierPremium, "cus_test123")

	require.NoError(t, err)
	assert.Equal(t, models.TierPremium, sub.Tier)
	assert.Equal(t, 2, sub.BoostsTotal)
	repo.AssertExpectations(t)
}

func TestCreateSubscription_FreeTierFails(t *testing.T) {
	client, _ := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	_, err := client.CreateSubscription(ctx, userID, models.TierFree, "cus_test123")
	assert.Error(t, err)
}

func TestCancelSubscription(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierFree).Return(nil)

	err := client.CancelSubscription(ctx, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCancelSubscription_NoSubscription(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	err := client.CancelSubscription(ctx, userID)
	assert.Error(t, err)
}

func TestUpdateSubscription(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}

	repo.On("GetSubscription", ctx, userID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierPremium).Return(nil)

	err := client.UpdateSubscription(ctx, userID, models.TierPremium)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestConstructWebhookEvent_NoSecret(t *testing.T) {
	repo := new(mockPremiumRepo)
	config := StripeConfig{SecretKey: "sk_test_123"}
	client := NewStripeClient(config, repo, nil)

	payload := []byte(`{"id":"evt_123","type":"invoice.paid","data":{"object":{}}}`)
	event, err := client.ConstructWebhookEvent(payload, "")

	require.NoError(t, err)
	assert.Equal(t, "invoice.paid", event.Type)
	assert.Equal(t, "evt_123", event.ID)
}

func TestConstructWebhookEvent_ValidSignature(t *testing.T) {
	client, _ := newTestClient()

	payload := []byte(`{"id":"evt_123","type":"invoice.paid","data":{"object":{}}}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test_123"))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	header := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	event, err := client.ConstructWebhookEvent(payload, header)

	require.NoError(t, err)
	assert.Equal(t, "invoice.paid", event.Type)
}

func TestConstructWebhookEvent_InvalidSignature(t *testing.T) {
	client, _ := newTestClient()

	payload := []byte(`{"id":"evt_123","type":"invoice.paid","data":{"object":{}}}`)
	header := "t=123456789,v1=invalidsignature"

	_, err := client.ConstructWebhookEvent(payload, header)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSignature, err)
}

func TestHandleInvoicePaid(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()
	stripeSubID := "sub_test123"

	sub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierBasic,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	invoiceData := map[string]interface{}{
		"object": map[string]interface{}{
			"id":           "inv_123",
			"customer":     "cus_123",
			"subscription": stripeSubID,
			"amount_paid":  299,
			"currency":     "usd",
			"status":       "paid",
			"period_start": time.Now().Unix(),
			"period_end":   time.Now().AddDate(0, 1, 0).Unix(),
		},
	}
	data, _ := json.Marshal(invoiceData)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("CreateBillingInvoice", ctx, mock.AnythingOfType("*models.BillingInvoice")).Return(nil)

	err := client.HandleInvoicePaid(ctx, data)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleCustomerSubscriptionDeleted(t *testing.T) {
	client, repo := newTestClient()
	ctx := context.Background()
	userID := uuid.New()
	stripeSubID := "sub_test123"

	sub := &models.Subscription{
		ID:                   uuid.New(),
		UserID:               userID,
		Tier:                 models.TierPremium,
		Status:               models.SubStatusActive,
		StripeSubscriptionID: stripeSubID,
	}

	subData := map[string]interface{}{
		"object": map[string]interface{}{
			"id":       stripeSubID,
			"customer": "cus_123",
			"status":   "canceled",
		},
	}
	data, _ := json.Marshal(subData)

	repo.On("GetSubscriptionByStripeID", ctx, stripeSubID).Return(sub, nil)
	repo.On("UpdateSubscription", ctx, mock.AnythingOfType("*models.Subscription")).Return(nil)
	repo.On("UpdateUserPremiumTier", ctx, userID, models.TierFree).Return(nil)

	err := client.HandleCustomerSubscriptionDeleted(ctx, data)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandleWebhookEvent_UnknownType(t *testing.T) {
	client, _ := newTestClient()
	ctx := context.Background()

	event := &WebhookEvent{
		ID:   "evt_123",
		Type: "unknown.event.type",
		Data: json.RawMessage(`{}`),
	}

	err := client.HandleWebhookEvent(ctx, event)
	assert.NoError(t, err) // Unknown events should be silently ignored
}

func TestPriceIDForTier(t *testing.T) {
	client, _ := newTestClient()

	assert.Equal(t, "price_basic_299", client.PriceIDForTier(models.TierBasic))
	assert.Equal(t, "price_premium_999", client.PriceIDForTier(models.TierPremium))
	assert.Equal(t, "", client.PriceIDForTier(models.TierFree))
}

func TestVerifyStripeSignature_ExpiredTimestamp(t *testing.T) {
	payload := []byte(`{"test": true}`)
	// Timestamp from 10 minutes ago (beyond 5-minute tolerance)
	oldTs := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	signedPayload := oldTs + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	header := fmt.Sprintf("t=%s,v1=%s", oldTs, sig)

	result := verifyStripeSignature(payload, header, "whsec_test")
	assert.False(t, result)
}
