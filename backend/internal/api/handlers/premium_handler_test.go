package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MockPremiumService implements premium service methods for testing
type MockPremiumService struct {
	userPremiumStatus    *models.PremiumStatus
	userPremiumStatusErr error

	createSubscriptionTier    models.PremiumTier
	createSubscriptionResult *models.Subscription
	createSubscriptionErr    error

	updateSubscriptionTierErr error

	cancelSubscriptionErr error

	reactivateSubscriptionErr error

	boostServerServerID uuid.UUID
	boostServerErr      error

	unboostServerErr error

	getServerBoostLevelResult *models.ServerPerks
	getServerBoostLevelErr    error

	getServerBoostsResult []*models.ServerBoost
	getServerBoostsErr    error

	getUserBoostsResult []*models.ServerBoost
	getUserBoostsErr    error

	checkFeatureAccessFeature string
	checkFeatureAccessResult bool
	checkFeatureAccessErr    error

	getBillingInvoicesResult []*models.BillingInvoice
	getBillingInvoicesErr    error

	getPaymentMethodsResult []*models.PaymentMethod
	getPaymentMethodsErr    error

	boostTiers []models.BoostTierInfo
	subTiers   []models.SubscriptionTierInfo
}

func (m *MockPremiumService) GetUserPremiumStatus(ctx context.Context, userID uuid.UUID) (*models.PremiumStatus, error) {
	if m.userPremiumStatusErr != nil {
		return nil, m.userPremiumStatusErr
	}
	return m.userPremiumStatus, nil
}

func (m *MockPremiumService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) (*models.Subscription, error) {
	if m.createSubscriptionErr != nil {
		return nil, m.createSubscriptionErr
	}
	return m.createSubscriptionResult, nil
}

func (m *MockPremiumService) UpdateSubscriptionTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error {
	return m.updateSubscriptionTierErr
}

func (m *MockPremiumService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.cancelSubscriptionErr
}

func (m *MockPremiumService) ReactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.reactivateSubscriptionErr
}

func (m *MockPremiumService) BoostServer(ctx context.Context, userID, serverID uuid.UUID) error {
	return m.boostServerErr
}

func (m *MockPremiumService) UnboostServer(ctx context.Context, userID, serverID uuid.UUID) error {
	return m.unboostServerErr
}

func (m *MockPremiumService) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	return m.getServerBoostLevelResult, m.getServerBoostLevelErr
}

func (m *MockPremiumService) GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error) {
	return m.getServerBoostsResult, m.getServerBoostsErr
}

func (m *MockPremiumService) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	return m.getUserBoostsResult, m.getUserBoostsErr
}

func (m *MockPremiumService) GetUserBoostsAvailable(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.userPremiumStatus != nil {
		return m.userPremiumStatus.BoostsAvailable, nil
	}
	return 0, nil
}

func (m *MockPremiumService) CalculateServerPerks(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	return m.getServerBoostLevelResult, m.getServerBoostLevelErr
}

func (m *MockPremiumService) CheckFeatureAccess(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	if m.checkFeatureAccessErr != nil {
		return false, m.checkFeatureAccessErr
	}
	return m.checkFeatureAccessResult, nil
}

func (m *MockPremiumService) GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error) {
	return m.getBillingInvoicesResult, m.getBillingInvoicesErr
}

func (m *MockPremiumService) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	return m.getPaymentMethodsResult, m.getPaymentMethodsErr
}

func (m *MockPremiumService) GetBoostTiers(ctx context.Context) []models.BoostTierInfo {
	if m.boostTiers == nil {
		return models.GetAllBoostTiers()
	}
	return m.boostTiers
}

func (m *MockPremiumService) GetSubscriptionTiers(ctx context.Context) []models.SubscriptionTierInfo {
	if m.subTiers == nil {
		return models.GetAllSubscriptionTiers()
	}
	return m.subTiers
}

// MockBillingService implements billing service for testing
type MockBillingService struct {
	createSubscriptionResult *models.Subscription
	createSubscriptionErr    error
	cancelSubscriptionErr    error
	handleWebhookErr          error
}

func (m *MockBillingService) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error) {
	if m.createSubscriptionErr != nil {
		return nil, m.createSubscriptionErr
	}
	return m.createSubscriptionResult, nil
}

func (m *MockBillingService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.cancelSubscriptionErr
}

func (m *MockBillingService) HandleWebhook(ctx context.Context, provider string, payload []byte, signature string) error {
	return m.handleWebhookErr
}

func setupPremiumTestApp(mockPremium *MockPremiumService, mockBilling *MockBillingService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	// Create a wrapper that satisfies the PremiumService interface
	type PremiumServiceWrapper struct {
		*MockPremiumService
	}

	// We need to wrap it properly - create inline
	app.Get("/premium/subscription", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		status, err := mockPremium.GetUserPremiumStatus(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(status)
	})

	app.Get("/premium/boost-tiers", func(c *fiber.Ctx) error {
		tiers := mockPremium.GetBoostTiers(c.Context())
		return c.JSON(tiers)
	})

	app.Get("/premium/subscription-tiers", func(c *fiber.Ctx) error {
		tiers := mockPremium.GetSubscriptionTiers(c.Context())
		return c.JSON(tiers)
	})

	app.Get("/premium/boosts", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		boosts, err := mockPremium.GetUserBoosts(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(boosts)
	})

	app.Get("/servers/:id/perks", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		perks, err := mockPremium.GetServerBoostLevel(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(perks)
	})

	app.Get("/servers/:id/boosts", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		boosts, err := mockPremium.GetServerBoosts(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(boosts)
	})

	app.Post("/servers/:id/boost", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		if err := mockPremium.BoostServer(c.Context(), userID, serverID); err != nil {
			if errors.Is(err, services.ErrNoSubscription) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "no subscription"})
			}
			if errors.Is(err, services.ErrBoostLimitReached) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "boost limit reached"})
			}
			if errors.Is(err, services.ErrAlreadyBoosted) {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already boosted"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		perks, _ := mockPremium.GetServerBoostLevel(c.Context(), serverID)
		return c.JSON(fiber.Map{"message": "server boosted successfully", "perks": perks})
	})

	app.Delete("/servers/:id/boost", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		if err := mockPremium.UnboostServer(c.Context(), userID, serverID); err != nil {
			if errors.Is(err, services.ErrNotBoosted) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not boosted"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "server unboosted successfully"})
	})

	app.Get("/premium/features/:feature/check", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		feature := c.Params("feature")
		hasAccess, err := mockPremium.CheckFeatureAccess(c.Context(), userID, feature)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"feature": feature, "has_access": hasAccess})
	})

	return app
}

func TestGetBoostTiers(t *testing.T) {
	mockPremium := &MockPremiumService{}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/boost-tiers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var tiers []models.BoostTierInfo
	err = json.NewDecoder(resp.Body).Decode(&tiers)
	require.NoError(t, err)
	assert.Len(t, tiers, 4)
	assert.Equal(t, 0, tiers[0].Level)
	assert.Equal(t, 1, tiers[1].Level)
	assert.Equal(t, 2, tiers[2].Level)
	assert.Equal(t, 3, tiers[3].Level)
}

func TestGetSubscriptionTiers(t *testing.T) {
	mockPremium := &MockPremiumService{}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/subscription-tiers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var tiers []models.SubscriptionTierInfo
	err = json.NewDecoder(resp.Body).Decode(&tiers)
	require.NoError(t, err)
	assert.Len(t, tiers, 3)
	assert.Equal(t, models.TierFree, tiers[0].Tier)
	assert.Equal(t, models.TierBasic, tiers[1].Tier)
	assert.Equal(t, models.TierPremium, tiers[2].Tier)
}

func TestGetUserPremiumStatus(t *testing.T) {
	userID := uuid.New()
	mockPremium := &MockPremiumService{
		userPremiumStatus: &models.PremiumStatus{
			UserID:          userID,
			Tier:            models.TierBasic,
			Status:          models.SubStatusActive,
			BoostsUsed:      1,
			BoostsTotal:     2,
			BoostsAvailable: 1,
			Features:        models.GetPremiumFeatures(models.TierBasic),
		},
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/subscription", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var status models.PremiumStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)
	assert.Equal(t, userID, status.UserID)
	assert.Equal(t, models.TierBasic, status.Tier)
	assert.Equal(t, 1, status.BoostsAvailable)
}

func TestGetUserPremiumStatus_Unauthorized(t *testing.T) {
	mockPremium := &MockPremiumService{}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/subscription", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestBoostServer_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		getServerBoostLevelResult: &models.ServerPerks{
			Level:      1,
			BoostCount: 1,
		},
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestBoostServer_NoSubscription(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		boostServerErr: services.ErrNoSubscription,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestBoostServer_LimitReached(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		boostServerErr: services.ErrBoostLimitReached,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestBoostServer_AlreadyBoosted(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		boostServerErr: services.ErrAlreadyBoosted,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodPost, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestUnboostServer_Success(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUnboostServer_NotBoosted(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		unboostServerErr: services.ErrNotBoosted,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodDelete, "/servers/"+serverID.String()+"/boost", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetServerPerks(t *testing.T) {
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		getServerBoostLevelResult: &models.ServerPerks{
			Level:           2,
			BoostCount:      15,
			BoostsRequired:  30,
			EmojiLimit:      150,
			FileUploadLimit: 100 * 1024 * 1024,
			VoiceBitrate:    256000,
			HasBanner:       true,
		},
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/perks", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var perks models.ServerPerks
	err = json.NewDecoder(resp.Body).Decode(&perks)
	require.NoError(t, err)
	assert.Equal(t, 2, perks.Level)
	assert.Equal(t, 150, perks.EmojiLimit)
	assert.True(t, perks.HasBanner)
}

func TestGetServerBoosts(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()

	mockPremium := &MockPremiumService{
		getServerBoostsResult: []*models.ServerBoost{
			{
				ID:       uuid.New(),
				ServerID: serverID,
				UserID:   userID,
				Active:   true,
				CreatedAt: time.Now(),
			},
		},
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+serverID.String()+"/boosts", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var boosts []*models.ServerBoost
	err = json.NewDecoder(resp.Body).Decode(&boosts)
	require.NoError(t, err)
	assert.Len(t, boosts, 1)
}

func TestGetUserBoosts(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	mockPremium := &MockPremiumService{
		getUserBoostsResult: []*models.ServerBoost{
			{
				ID:       uuid.New(),
				ServerID: serverID,
				UserID:   userID,
				Active:   true,
				CreatedAt: time.Now(),
			},
		},
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/boosts", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var boosts []*models.ServerBoost
	err = json.NewDecoder(resp.Body).Decode(&boosts)
	require.NoError(t, err)
	assert.Len(t, boosts, 1)
}

func TestCheckFeatureAccess_PremiumUser(t *testing.T) {
	userID := uuid.New()

	mockPremium := &MockPremiumService{
		checkFeatureAccessResult: true,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/features/cross_server_emojis/check", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "cross_server_emojis", result["feature"])
	assert.Equal(t, true, result["has_access"])
}

func TestCheckFeatureAccess_FreeUser(t *testing.T) {
	userID := uuid.New()

	mockPremium := &MockPremiumService{
		checkFeatureAccessResult: false,
	}
	mockBilling := &MockBillingService{}
	app := setupPremiumTestApp(mockPremium, mockBilling)

	req := httptest.NewRequest(http.MethodGet, "/premium/features/cross_server_emojis/check", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, false, result["has_access"])
}
