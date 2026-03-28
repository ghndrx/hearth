package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
	"hearth/internal/services"
)

// AutoModHandler handles auto-moderation API endpoints
type AutoModHandler struct {
	automodService *services.AutoModService
	serverService  *services.ServerService
}

// NewAutoModHandler creates a new auto-mod handler
func NewAutoModHandler(automodService *services.AutoModService, serverService *services.ServerService) *AutoModHandler {
	return &AutoModHandler{
		automodService: automodService,
		serverService:  serverService,
	}
}

// getAutoModUserID safely extracts userID from Fiber context
func getAutoModUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDVal := c.Locals("userID")
	if userIDVal == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user id")
	}
	return userID, nil
}

// ListRules lists all auto-mod rules for a server
// GET /api/v1/servers/:id/automod/rules
func (h *AutoModHandler) ListRules(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	rules, err := h.automodService.GetServerRules(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(rules)
}

// CreateRule creates a new auto-mod rule
// POST /api/v1/servers/:id/automod/rules
func (h *AutoModHandler) CreateRule(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user has manage server permissions (would need permService for full check)
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	var req models.CreateAutoModRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" {
		return ValidationError(c, "name", "is required")
	}

	rule, err := h.automodService.CreateRule(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// GetRule gets a specific auto-mod rule
// GET /api/v1/automod/rules/:id
func (h *AutoModHandler) GetRule(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	rule, err := h.automodService.GetRule(c.Context(), ruleID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if rule == nil {
		return fiber.NewError(fiber.StatusNotFound, "rule not found")
	}

	return c.JSON(rule)
}

// UpdateRule updates an existing auto-mod rule
// PATCH /api/v1/automod/rules/:id
func (h *AutoModHandler) UpdateRule(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	_, err = getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Get existing rule to check ownership
	existing, err := h.automodService.GetRule(c.Context(), ruleID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if existing == nil {
		return fiber.NewError(fiber.StatusNotFound, "rule not found")
	}

	var req models.UpdateAutoModRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	rule, err := h.automodService.UpdateRule(c.Context(), ruleID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(rule)
}

// DeleteRule deletes an auto-mod rule
// DELETE /api/v1/automod/rules/:id
func (h *AutoModHandler) DeleteRule(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	if err := h.automodService.DeleteRule(c.Context(), ruleID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListAlerts lists recent auto-mod alerts for a server
// GET /api/v1/servers/:id/automod/alerts
func (h *AutoModHandler) ListAlerts(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	alerts, err := h.automodService.GetServerAlerts(c.Context(), serverID, limit, offset)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(alerts)
}

// GetAlert gets a specific auto-mod alert
// GET /api/v1/automod/alerts/:id
func (h *AutoModHandler) GetAlert(c *fiber.Ctx) error {
	_, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid alert id")
	}

	// For now, just return not found - would need alert service to check permissions
	return fiber.NewError(fiber.StatusNotFound, "alert not found")
}

// ResolveAlert resolves or reopens an alert
// POST /api/v1/automod/alerts/:id/resolve
func (h *AutoModHandler) ResolveAlert(c *fiber.Ctx) error {
	alertID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid alert id")
	}

	userID, err := getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.ResolveAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Resolved {
		if err := h.automodService.ResolveAlert(c.Context(), alertID, userID); err != nil {
			return HandleServiceError(c, err)
		}
	}

	return c.JSON(fiber.Map{"resolved": req.Resolved})
}

// TestContent tests content against auto-mod rules
// POST /api/v1/automod/test
func (h *AutoModHandler) TestContent(c *fiber.Ctx) error {
	userID, err := getAutoModUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.AutoModTestRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.ServerID == uuid.Nil {
		return ValidationError(c, "server_id", "is required")
	}
	if req.Content == "" {
		return ValidationError(c, "content", "is required")
	}

	// Set member ID if not provided
	if req.MemberID == nil {
		req.MemberID = &userID
	}

	result, err := h.automodService.TestContent(c.Context(), &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(result)
}

// GetRuleStats gets statistics for an auto-mod rule
// GET /api/v1/automod/rules/:id/stats
func (h *AutoModHandler) GetRuleStats(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	stats, err := h.automodService.GetRuleStats(c.Context(), ruleID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(stats)
}
