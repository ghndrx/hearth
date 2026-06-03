package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
	"hearth/internal/services"
)

// mockPremiumServiceForPremiumTests mocks PremiumService methods used by premium handler tests
type mockPremiumServiceForPremiumTests struct {
	mock.Mock
}

func (m *mockPremiumServiceForPremiumTests) GetUserPremiumStatus(ctx context.Context, userID uuid.UUID) (*models.PremiumStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PremiumStatus), args.Error(1)
}

func (m *mockPremiumServiceForPremiumTests) ReactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockPremiumServiceForPremiumTests) BoostServer(ctx context.Context, userID, serverID uuid.UUID) error {
	return m.Called(ctx, userID, serverID).Error(0)
}

func (m *mockPremiumServiceForPremiumTests) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerPerks), args.Error(1)
}

func (m *mockPremiumServiceForPremiumTests) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentMethod), args.Error(1)
}

func (m *mockPremiumServiceForPremiumTests) DeletePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	return m.Called(ctx, userID, paymentMethodID).Error(0)
}

func (m *mockPremiumServiceForPremiumTests) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerBoost), args.Error(1)
}

// mockBillingServiceForPremiumTests mocks BillingService methods used by premium handler tests
type mockBillingServiceForPremiumTests struct {
	mock.Mock
}

func (m *mockBillingServiceForPremiumTests) CreateSubscription(ctx context.Context, userID uuid.UUID, tier models.PremiumTier, paymentMethodID string) (*models.Subscription, error) {
	args := m.Called(ctx, userID, tier, paymentMethodID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

func (m *mockBillingServiceForPremiumTests) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *mockBillingServiceForPremiumTests) HandleWebhook(ctx context.Context, provider string, payload []byte, signature string) error {
	return m.Called(ctx, provider, payload, signature).Error(0)
}

func (m *mockBillingServiceForPremiumTests) CreateBillingPortalSession(ctx context.Context, userID uuid.UUID, returnURL string) (string, error) {
	args := m.Called(ctx, userID, returnURL)
	return args.String(0), args.Error(1)
}

func (m *mockBillingServiceForPremiumTests) GiftSubscription(ctx context.Context, gifterUserID, recipientUserID uuid.UUID, tier models.PremiumTier, months int) (*models.Subscription, error) {
	args := m.Called(ctx, gifterUserID, recipientUserID, tier, months)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), args.Error(1)
}

// premiumHandlerForTest wraps premium handler methods with mock services
type premiumHandlerForTest struct {
	premiumService *mockPremiumServiceForPremiumTests
	billingService *mockBillingServiceForPremiumTests
}

func (h *premiumHandlerForTest) GetSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	status, err := h.premiumService.GetUserPremiumStatus(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(status)
}

func (h *premiumHandlerForTest) CreateSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	var req struct {
		Tier            string `json:"tier"`
		PaymentMethodID string `json:"payment_method_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}
	tier := models.SubscriptionTierFromString(req.Tier)
	if tier == models.TierFree {
		return ValidationError(c, "tier", "must be 'basic' or 'premium'")
	}
	sub, err := h.billingService.CreateSubscription(c.Context(), userID, tier, req.PaymentMethodID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(sub)
}

func (h *premiumHandlerForTest) CancelSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	if err := h.billingService.CancelSubscription(c.Context(), userID); err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"message": "subscription canceled"})
}

func (h *premiumHandlerForTest) ReactivateSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	if err := h.premiumService.ReactivateSubscription(c.Context(), userID); err != nil {
		return HandleServiceError(c, err)
	}
	status, err := h.premiumService.GetUserPremiumStatus(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(status)
}

func (h *premiumHandlerForTest) GetPaymentMethods(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	methods, err := h.premiumService.GetPaymentMethods(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(methods)
}

func (h *premiumHandlerForTest) DeletePaymentMethod(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	paymentMethodID := c.Params("id")
	if paymentMethodID == "" {
		return ValidationError(c, "id", "payment method ID is required")
	}
	if err := h.premiumService.DeletePaymentMethod(c.Context(), userID, paymentMethodID); err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"message": "payment method deleted"})
}

func (h *premiumHandlerForTest) GetBillingPortal(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	returnURL := c.Query("return_url", "https://hearth.example.com")
	portalURL, err := h.billingService.CreateBillingPortalSession(c.Context(), userID, returnURL)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"url": portalURL})
}

func (h *premiumHandlerForTest) HandleBillingWebhook(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return ValidationError(c, "provider", "provider is required")
	}
	payload := c.Body()
	if len(payload) == 0 {
		return ValidationError(c, "body", "webhook payload is required")
	}
	signature := c.Get("Stripe-Signature")
	if err := h.billingService.HandleWebhook(c.Context(), provider, payload, signature); err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"received": true})
}

func (h *premiumHandlerForTest) GiftSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	var req struct {
		RecipientID string `json:"recipient_id"`
		Tier        string `json:"tier"`
		Months      int    `json:"months"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}
	if req.RecipientID == "" {
		return ValidationError(c, "recipient_id", "recipient user ID is required")
	}
	recipientUUID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		return InvalidUUID(c, "recipient_id")
	}
	if req.Months <= 0 {
		req.Months = 1
	}
	tier := models.SubscriptionTierFromString(req.Tier)
	if tier == models.TierFree {
		return ValidationError(c, "tier", "must be 'basic' or 'premium'")
	}
	sub, err := h.billingService.GiftSubscription(c.Context(), userID, recipientUUID, tier, req.Months)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(sub)
}

func (h *premiumHandlerForTest) BoostServer(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}
	if err := h.premiumService.BoostServer(c.Context(), userID, serverID); err != nil {
		return HandleServiceError(c, err)
	}
	perks, err := h.premiumService.GetServerBoostLevel(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{
		"message": "server boosted successfully",
		"perks":   perks,
	})
}

func (h *premiumHandlerForTest) GetUserBoosts(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}
	boosts, err := h.premiumService.GetUserBoosts(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	return c.JSON(boosts)
}

func setupPremiumTestApp(t *testing.T) (*fiber.App, *mockPremiumServiceForPremiumTests, *mockBillingServiceForPremiumTests) {
	t.Helper()
	premiumSvc := new(mockPremiumServiceForPremiumTests)
	billingSvc := new(mockBillingServiceForPremiumTests)
	handler := &premiumHandlerForTest{
		premiumService: premiumSvc,
		billingService: billingSvc,
	}

	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		return c.Next()
	})

	app.Get("/premium/subscription", handler.GetSubscription)
	app.Post("/premium/subscribe", handler.CreateSubscription)
	app.Delete("/premium/subscription", handler.CancelSubscription)
	app.Post("/premium/subscription/reactivate", handler.ReactivateSubscription)
	app.Get("/premium/payment-methods", handler.GetPaymentMethods)
	app.Delete("/premium/payment-methods/:id", handler.DeletePaymentMethod)
	app.Get("/premium/billing-portal", handler.GetBillingPortal)
	app.Post("/billing/webhook/:provider", handler.HandleBillingWebhook)
	app.Post("/premium/gift", handler.GiftSubscription)
	app.Post("/servers/:id/boost", handler.BoostServer)
	app.Get("/premium/boosts", handler.GetUserBoosts)

	return app, premiumSvc, billingSvc
}

// GetPremiumStatus / GetSubscription Tests

func TestPremiumHandler_GetSubscription_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	status := &models.PremiumStatus{
		UserID:          userID,
		Tier:            models.TierPremium,
		Status:          models.SubStatusActive,
		BoostsUsed:      1,
		BoostsTotal:     2,
		BoostsAvailable: 1,
	}
	premiumSvc.On("GetUserPremiumStatus", mock.Anything, userID).Return(status, nil)

	req := httptest.NewRequest("GET", "/premium/subscription", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.PremiumStatus
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.TierPremium, result.Tier)
	assert.Equal(t, models.SubStatusActive, result.Status)
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_GetSubscription_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("GetUserPremiumStatus", mock.Anything, userID).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/premium/subscription", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_GetSubscription_NotFound(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	status := &models.PremiumStatus{
		UserID:          userID,
		Tier:            models.TierFree,
		Status:          models.SubStatusActive,
		BoostsUsed:      0,
		BoostsTotal:     0,
		BoostsAvailable: 0,
	}
	premiumSvc.On("GetUserPremiumStatus", mock.Anything, userID).Return(status, nil)

	req := httptest.NewRequest("GET", "/premium/subscription", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.PremiumStatus
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.TierFree, result.Tier)
	premiumSvc.AssertExpectations(t)
}

// CreateSubscription Tests

func TestPremiumHandler_CreateSubscription_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	now := time.Now()
	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: userID,
		Tier:   models.TierPremium,
		Status: models.SubStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	billingSvc.On("CreateSubscription", mock.Anything, userID, models.TierPremium, "").Return(sub, nil)

	body, _ := json.Marshal(map[string]string{"tier": "premium"})
	req := httptest.NewRequest("POST", "/premium/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result models.Subscription
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.TierPremium, result.Tier)
	billingSvc.AssertExpectations(t)
}

func TestPremiumHandler_CreateSubscription_ValidationError(t *testing.T) {
	app, _, _ := setupPremiumTestApp(t)

	body, _ := json.Marshal(map[string]string{"tier": "free"})
	req := httptest.NewRequest("POST", "/premium/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestPremiumHandler_CreateSubscription_BillingError(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("CreateSubscription", mock.Anything, userID, models.TierPremium, "").Return(nil, errors.New("billing failed"))

	body, _ := json.Marshal(map[string]string{"tier": "premium"})
	req := httptest.NewRequest("POST", "/premium/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	billingSvc.AssertExpectations(t)
}

// CancelSubscription Tests

func TestPremiumHandler_CancelSubscription_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("CancelSubscription", mock.Anything, userID).Return(nil)

	req := httptest.NewRequest("DELETE", "/premium/subscription", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "subscription canceled", result["message"])
	billingSvc.AssertExpectations(t)
}

func TestPremiumHandler_CancelSubscription_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("CancelSubscription", mock.Anything, userID).Return(errors.New("cancel failed"))

	req := httptest.NewRequest("DELETE", "/premium/subscription", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	billingSvc.AssertExpectations(t)
}

// ReactivateSubscription Tests

func TestPremiumHandler_ReactivateSubscription_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("ReactivateSubscription", mock.Anything, userID).Return(nil)
	status := &models.PremiumStatus{
		UserID: userID,
		Tier:   models.TierPremium,
		Status: models.SubStatusActive,
	}
	premiumSvc.On("GetUserPremiumStatus", mock.Anything, userID).Return(status, nil)

	req := httptest.NewRequest("POST", "/premium/subscription/reactivate", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.PremiumStatus
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.TierPremium, result.Tier)
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_ReactivateSubscription_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("ReactivateSubscription", mock.Anything, userID).Return(errors.New("reactivate failed"))

	req := httptest.NewRequest("POST", "/premium/subscription/reactivate", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}

// ListPaymentMethods Tests

func TestPremiumHandler_ListPaymentMethods_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	methods := []*models.PaymentMethod{
		{ID: "pm_1", Type: models.PaymentMethodCard, Last4: "4242", Brand: "Visa", IsDefault: true},
		{ID: "pm_2", Type: models.PaymentMethodCard, Last4: "0000", Brand: "Mastercard"},
	}
	premiumSvc.On("GetPaymentMethods", mock.Anything, userID).Return(methods, nil)

	req := httptest.NewRequest("GET", "/premium/payment-methods", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []*models.PaymentMethod
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
	assert.Equal(t, "pm_1", result[0].ID)
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_ListPaymentMethods_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("GetPaymentMethods", mock.Anything, userID).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/premium/payment-methods", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}

// DeletePaymentMethod Tests

func TestPremiumHandler_DeletePaymentMethod_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("DeletePaymentMethod", mock.Anything, userID, "pm_123").Return(nil)

	req := httptest.NewRequest("DELETE", "/premium/payment-methods/pm_123", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "payment method deleted", result["message"])
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_DeletePaymentMethod_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("DeletePaymentMethod", mock.Anything, userID, "pm_123").Return(errors.New("delete failed"))

	req := httptest.NewRequest("DELETE", "/premium/payment-methods/pm_123", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}

// GetBillingPortal Tests

func TestPremiumHandler_GetBillingPortal_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("CreateBillingPortalSession", mock.Anything, userID, "https://hearth.example.com").Return("https://portal.example.com/session/abc", nil)

	req := httptest.NewRequest("GET", "/premium/billing-portal", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "https://portal.example.com/session/abc", result["url"])
	billingSvc.AssertExpectations(t)
}

func TestPremiumHandler_GetBillingPortal_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("CreateBillingPortalSession", mock.Anything, userID, "https://hearth.example.com").Return("", errors.New("portal error"))

	req := httptest.NewRequest("GET", "/premium/billing-portal", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	billingSvc.AssertExpectations(t)
}

// HandleBillingWebhook Tests

func TestPremiumHandler_HandleBillingWebhook_SuccessMockMode(t *testing.T) {
	app, _, billingSvc := setupPremiumTestApp(t)

	payload := []byte(`{"type":"invoice.paid"}`)
	billingSvc.On("HandleWebhook", mock.Anything, "stripe", payload, "").Return(nil)

	req := httptest.NewRequest("POST", "/billing/webhook/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]bool
	json.NewDecoder(resp.Body).Decode(&result)
	assert.True(t, result["received"])
	billingSvc.AssertExpectations(t)
}

func TestPremiumHandler_HandleBillingWebhook_InvalidSignature(t *testing.T) {
	app, _, billingSvc := setupPremiumTestApp(t)

	payload := []byte(`{"type":"invoice.paid"}`)
	billingSvc.On("HandleWebhook", mock.Anything, "stripe", payload, "invalid_signature").Return(services.ErrWebhookInvalid)

	req := httptest.NewRequest("POST", "/billing/webhook/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "invalid_signature")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	billingSvc.AssertExpectations(t)
}

// GiftSubscription Tests

func TestPremiumHandler_GiftSubscription_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	recipientID := uuid.New()
	app, _, billingSvc := setupPremiumTestApp(t)

	now := time.Now()
	sub := &models.Subscription{
		ID:     uuid.New(),
		UserID: recipientID,
		Tier:   models.TierPremium,
		Status: models.SubStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	billingSvc.On("GiftSubscription", mock.Anything, userID, recipientID, models.TierPremium, 3).Return(sub, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"recipient_id": recipientID.String(),
		"tier":         "premium",
		"months":       3,
	})
	req := httptest.NewRequest("POST", "/premium/gift", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result models.Subscription
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, models.TierPremium, result.Tier)
	billingSvc.AssertExpectations(t)
}

func TestPremiumHandler_GiftSubscription_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	recipientID := uuid.New()
	app, _, billingSvc := setupPremiumTestApp(t)

	billingSvc.On("GiftSubscription", mock.Anything, userID, recipientID, models.TierPremium, 1).Return(nil, errors.New("gift failed"))

	body, _ := json.Marshal(map[string]interface{}{
		"recipient_id": recipientID.String(),
		"tier":         "premium",
	})
	req := httptest.NewRequest("POST", "/premium/gift", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	billingSvc.AssertExpectations(t)
}

// BoostServer Tests

func TestPremiumHandler_BoostServer_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	serverID := uuid.New()
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("BoostServer", mock.Anything, userID, serverID).Return(nil)
	perks := &models.ServerPerks{Level: 1, BoostCount: 2}
	premiumSvc.On("GetServerBoostLevel", mock.Anything, serverID).Return(perks, nil)

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/boost", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "server boosted successfully", result["message"])
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_BoostServer_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	serverID := uuid.New()
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("BoostServer", mock.Anything, userID, serverID).Return(errors.New("boost failed"))

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/boost", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}

// ListBoosts Tests

func TestPremiumHandler_ListBoosts_Success(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	boosts := []*models.ServerBoost{
		{ID: uuid.New(), ServerID: uuid.New(), UserID: userID, Active: true},
		{ID: uuid.New(), ServerID: uuid.New(), UserID: userID, Active: true},
	}
	premiumSvc.On("GetUserBoosts", mock.Anything, userID).Return(boosts, nil)

	req := httptest.NewRequest("GET", "/premium/boosts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result []*models.ServerBoost
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
	premiumSvc.AssertExpectations(t)
}

func TestPremiumHandler_ListBoosts_Error(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	app, premiumSvc, _ := setupPremiumTestApp(t)

	premiumSvc.On("GetUserBoosts", mock.Anything, userID).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/premium/boosts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	premiumSvc.AssertExpectations(t)
}
