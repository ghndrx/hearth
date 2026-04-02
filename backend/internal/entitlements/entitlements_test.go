package entitlements

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
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

func newTestService() (*Service, *mockPremiumRepo) {
	repo := new(mockPremiumRepo)
	premiumSvc := services.NewPremiumService(repo, nil, nil, nil)
	svc := New(premiumSvc)
	return svc, repo
}

func TestCheckSubscriptionTier_Free(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	tier, err := svc.CheckSubscriptionTier(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.TierFree, tier)
}

func TestCheckSubscriptionTier_Premium(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierPremium,
		Status: models.SubStatusActive,
	}
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	tier, err := svc.CheckSubscriptionTier(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.TierPremium, tier)
}

func TestHasFeature_PremiumUser(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierPremium,
		Status:      models.SubStatusActive,
		BoostsTotal: 2,
	}
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	has, err := svc.HasFeature(ctx, userID, "cross_server_emojis")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHasFeature_FreeUser(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	has, err := svc.HasFeature(ctx, userID, "cross_server_emojis")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestGetServerBoostLevel(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	serverID := uuid.New()

	perks := &models.ServerPerks{Level: 2, BoostCount: 15}
	repo.On("GetServerBoostLevel", ctx, serverID).Return(perks, nil)

	level, err := svc.GetServerBoostLevel(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, 2, level)
}

func TestIsSubscribed(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	subscribed, err := svc.IsSubscribed(ctx, userID)
	require.NoError(t, err)
	assert.True(t, subscribed)
}

func TestIsSubscribed_Free(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	repo.On("GetSubscription", ctx, userID).Return(nil, nil)

	subscribed, err := svc.IsSubscribed(ctx, userID)
	require.NoError(t, err)
	assert.False(t, subscribed)
}

func TestGetUploadLimit_BoostedServer(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	serverID := uuid.New()

	repo.On("GetServerBoostCount", ctx, serverID).Return(2, nil)

	limit, err := svc.GetUploadLimit(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, int64(50*1024*1024), limit) // 50MB at level 1
}

func TestGetEmojiLimit_NoBoosted(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	serverID := uuid.New()

	repo.On("GetServerBoostCount", ctx, serverID).Return(0, nil)

	limit, err := svc.GetEmojiLimit(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, 50, limit)
}

func TestIsPremium(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierPremium,
		Status:      models.SubStatusActive,
		BoostsTotal: 2,
		NextBilling: &now,
	}
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	isPremium, err := svc.IsPremium(ctx, userID)
	require.NoError(t, err)
	assert.True(t, isPremium)
}

func TestIsPremium_BasicIsNotPremium(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	userID := uuid.New()

	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierBasic,
		Status: models.SubStatusActive,
	}
	repo.On("GetSubscription", ctx, userID).Return(sub, nil)

	isPremium, err := svc.IsPremium(ctx, userID)
	require.NoError(t, err)
	assert.False(t, isPremium)
}
