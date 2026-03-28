package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// PremiumHandler handles premium subscription and server boost endpoints
type PremiumHandler struct {
	premiumService *services.PremiumService
	billingService *services.BillingService
}

// NewPremiumHandler creates a new premium handler
func NewPremiumHandler(premiumService *services.PremiumService, billingService *services.BillingService) *PremiumHandler {
	return &PremiumHandler{
		premiumService: premiumService,
		billingService: billingService,
	}
}

// GetSubscription returns the current user's subscription
// @Summary Get user subscription
// @Description Returns the current user's premium subscription status
// @Tags Premium
// @Produce json
// @Success 200 {object} models.PremiumStatus
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/subscription [get]
func (h *PremiumHandler) GetSubscription(c *fiber.Ctx) error {
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

// CreateSubscription creates a new subscription for the user
// @Summary Create subscription
// @Description Creates or updates a premium subscription
// @Tags Premium
// @Accept json
// @Produce json
// @Param body body struct{Tier string `json:"tier"`} true "Subscription tier"
// @Success 201 {object} models.Subscription
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/subscribe [post]
func (h *PremiumHandler) CreateSubscription(c *fiber.Ctx) error {
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

// UpdateSubscription updates the user's subscription tier
// @Summary Update subscription
// @Description Updates the subscription tier
// @Tags Premium
// @Accept json
// @Produce json
// @Param body body struct{Tier string `json:"tier"`} true "New subscription tier"
// @Success 200 {object} models.Subscription
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/subscription [put]
func (h *PremiumHandler) UpdateSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		Tier string `json:"tier"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	tier := models.SubscriptionTierFromString(req.Tier)
	if tier == models.TierFree {
		return ValidationError(c, "tier", "must be 'basic' or 'premium'")
	}

	if err := h.premiumService.UpdateSubscriptionTier(c.Context(), userID, tier); err != nil {
		return HandleServiceError(c, err)
	}

	status, err := h.premiumService.GetUserPremiumStatus(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(status)
}

// CancelSubscription cancels the user's subscription
// @Summary Cancel subscription
// @Description Cancels the premium subscription
// @Tags Premium
// @Produce json
// @Success 200 {object} fiber.Map
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/subscription [delete]
func (h *PremiumHandler) CancelSubscription(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.billingService.CancelSubscription(c.Context(), userID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"message": "subscription canceled"})
}

// ReactivateSubscription reactivates a canceled subscription
// @Summary Reactivate subscription
// @Description Reactivates a canceled subscription
// @Tags Premium
// @Produce json
// @Success 200 {object} models.Subscription
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/subscription/reactivate [post]
func (h *PremiumHandler) ReactivateSubscription(c *fiber.Ctx) error {
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

// BoostServer applies a user's boost to a server
// @Summary Boost server
// @Description Applies one of the user's server boosts to a server
// @Tags Server Boosts
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Boost limit reached"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/boost [post]
func (h *PremiumHandler) BoostServer(c *fiber.Ctx) error {
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

	// Get updated server perks
	perks, err := h.premiumService.GetServerBoostLevel(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "server boosted successfully",
		"perks":   perks,
	})
}

// UnboostServer removes a user's boost from a server
// @Summary Unboost server
// @Description Removes one of the user's server boosts from a server
// @Tags Server Boosts
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Server not boosted"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/boost [delete]
func (h *PremiumHandler) UnboostServer(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	if err := h.premiumService.UnboostServer(c.Context(), userID, serverID); err != nil {
		return HandleServiceError(c, err)
	}

	perks, err := h.premiumService.GetServerBoostLevel(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "server unboosted successfully",
		"perks":   perks,
	})
}

// GetServerBoosts returns all boosts on a server
// @Summary Get server boosts
// @Description Returns all active boosts on a server
// @Tags Server Boosts
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.ServerBoost
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/boosts [get]
func (h *PremiumHandler) GetServerBoosts(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	boosts, err := h.premiumService.GetServerBoosts(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(boosts)
}

// GetServerPerks returns the perks for a server based on boost level
// @Summary Get server perks
// @Description Returns the perks available for a server based on its boost level
// @Tags Server Boosts
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} models.ServerPerks
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/perks [get]
func (h *PremiumHandler) GetServerPerks(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	perks, err := h.premiumService.CalculateServerPerks(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(perks)
}

// GetUserBoosts returns all boosts for the current user
// @Summary Get user boosts
// @Description Returns all server boosts owned by the current user
// @Tags Server Boosts
// @Produce json
// @Success 200 {array} models.ServerBoost
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/boosts [get]
func (h *PremiumHandler) GetUserBoosts(c *fiber.Ctx) error {
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

// CheckFeatureAccess checks if the user has access to a premium feature
// @Summary Check feature access
// @Description Checks if the current user has access to a specific premium feature
// @Tags Premium
// @Produce json
// @Param feature path string true "Feature name"
// @Success 200 {object} fiber.Map
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/features/{feature}/check [get]
func (h *PremiumHandler) CheckFeatureAccess(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	feature := c.Params("feature")
	if feature == "" {
		return ValidationError(c, "feature", "feature name is required")
	}

	hasAccess, err := h.premiumService.CheckFeatureAccess(c.Context(), userID, feature)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"feature":    feature,
		"has_access": hasAccess,
	})
}

// GetBillingInvoices returns billing invoices for the current user
// @Summary Get billing invoices
// @Description Returns billing invoices for the current user
// @Tags Premium
// @Produce json
// @Param limit query int false "Number of invoices to return"
// @Success 200 {array} models.BillingInvoice
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/invoices [get]
func (h *PremiumHandler) GetBillingInvoices(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	limit := c.QueryInt("limit", 10)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	invoices, err := h.premiumService.GetBillingInvoices(c.Context(), userID, limit)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(invoices)
}

// GetPaymentMethods returns payment methods for the current user
// @Summary Get payment methods
// @Description Returns stored payment methods for the current user
// @Tags Premium
// @Produce json
// @Success 200 {array} models.PaymentMethod
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /premium/payment-methods [get]
func (h *PremiumHandler) GetPaymentMethods(c *fiber.Ctx) error {
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

// HandleBillingWebhook handles billing provider webhooks
// @Summary Handle billing webhook
// @Description Processes webhooks from billing providers (Stripe, Paddle)
// @Tags Premium
// @Accept json
// @Param provider path string true "Billing provider"
// @Param body body []byte true "Webhook payload"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map "Invalid webhook"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /billing/webhook/{provider} [post]
func (h *PremiumHandler) HandleBillingWebhook(c *fiber.Ctx) error {
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
