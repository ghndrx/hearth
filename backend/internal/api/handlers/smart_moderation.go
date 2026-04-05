package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
	"hearth/internal/services"
)

// SmartModerationHandler handles smart moderation API endpoints
type SmartModerationHandler struct {
	modService     *services.SmartModerationService
	serverService  *services.ServerService
}

// NewSmartModerationHandler creates a new smart moderation handler
func NewSmartModerationHandler(modService *services.SmartModerationService, serverService *services.ServerService) *SmartModerationHandler {
	return &SmartModerationHandler{
		modService:    modService,
		serverService: serverService,
	}
}

// getModerationUserID safely extracts userID from Fiber context
func getModerationUserID(c *fiber.Ctx) (uuid.UUID, error) {
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

// requireServerMember checks if user is a member of the server
func (h *SmartModerationHandler) requireServerMember(c *fiber.Ctx, serverID, userID uuid.UUID) error {
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}
	return nil
}

// GetSettings gets moderation settings for a server
// GET /api/v1/servers/:id/moderation/settings
func (h *SmartModerationHandler) GetSettings(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	settings, err := h.modService.GetOrCreateSettings(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(settings)
}

// UpdateSettings updates moderation settings for a server
// PATCH /api/v1/servers/:id/moderation/settings
func (h *SmartModerationHandler) UpdateSettings(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	var req models.UpdateModerationSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	settings, err := h.modService.UpdateSettings(c.Context(), serverID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(settings)
}

// ListKeywordRules lists all keyword rules for a server
// GET /api/v1/servers/:id/moderation/rules
func (h *SmartModerationHandler) ListKeywordRules(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	rules, err := h.modService.GetKeywordRules(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(rules)
}

// CreateKeywordRule creates a new keyword rule
// POST /api/v1/servers/:id/moderation/rules
func (h *SmartModerationHandler) CreateKeywordRule(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	var req models.CreateKeywordRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" {
		return ValidationError(c, "name", "is required")
	}
	if req.Pattern == "" {
		return ValidationError(c, "pattern", "is required")
	}

	rule, err := h.modService.CreateKeywordRule(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

// UpdateKeywordRule updates a keyword rule
// PATCH /api/v1/moderation/rules/:id
func (h *SmartModerationHandler) UpdateKeywordRule(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	_, err = getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.UpdateKeywordRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	rule, err := h.modService.UpdateKeywordRule(c.Context(), ruleID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(rule)
}

// DeleteKeywordRule deletes a keyword rule
// DELETE /api/v1/moderation/rules/:id
func (h *SmartModerationHandler) DeleteKeywordRule(c *fiber.Ctx) error {
	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid rule id")
	}

	if err := h.modService.DeleteKeywordRule(c.Context(), ruleID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AnalyzeContent analyzes content for violations
// POST /api/v1/moderation/analyze
func (h *SmartModerationHandler) AnalyzeContent(c *fiber.Ctx) error {
	_, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.AnalyzeContentRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.ServerID == uuid.Nil {
		return ValidationError(c, "server_id", "is required")
	}
	if req.Content == "" {
		return ValidationError(c, "content", "is required")
	}

	result, err := h.modService.AnalyzeContent(c.Context(), &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(result)
}

// ListModerationLogs lists moderation logs for a server
// GET /api/v1/servers/:id/moderation/logs
func (h *SmartModerationHandler) ListModerationLogs(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.modService.GetModerationLogs(c.Context(), serverID, limit, offset)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(logs)
}

// GetMemberModerationHistory gets moderation history for a specific member
// GET /api/v1/servers/:id/moderation/members/:memberId/history
func (h *SmartModerationHandler) GetMemberModerationHistory(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	logs, err := h.modService.GetMemberModerationHistory(c.Context(), serverID, memberID, limit, offset)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(logs)
}

// TakeModerationAction takes a moderation action against a member
// POST /api/v1/servers/:id/moderation/actions
func (h *SmartModerationHandler) TakeModerationAction(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	moderatorID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, moderatorID); err != nil {
		return err
	}

	var req models.ModerationActionRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.MemberID == uuid.Nil {
		return ValidationError(c, "member_id", "is required")
	}

	log, err := h.modService.TakeModerationAction(c.Context(), serverID, moderatorID, &req)
	if err != nil {
		if err == services.ErrModerationRateLimited {
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded for moderation actions")
		}
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(log)
}

// ResolveModerationLog resolves or reopens a moderation log
// POST /api/v1/moderation/logs/:id/resolve
func (h *SmartModerationHandler) ResolveModerationLog(c *fiber.Ctx) error {
	logID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid log id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.ResolveLogRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Resolved {
		if err := h.modService.ResolveModerationLog(c.Context(), logID, userID); err != nil {
			return HandleServiceError(c, err)
		}
	}

	return c.JSON(fiber.Map{"resolved": req.Resolved})
}

// GetDashboardStats gets moderation dashboard statistics
// GET /api/v1/servers/:id/moderation/stats
func (h *SmartModerationHandler) GetDashboardStats(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	days, _ := strconv.Atoi(c.Query("days", "7"))

	stats, err := h.modService.GetDashboardStats(c.Context(), serverID, days)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(stats)
}

// GetUserViolationSummary gets violation summary for a user
// GET /api/v1/servers/:id/moderation/members/:memberId/summary
func (h *SmartModerationHandler) GetUserViolationSummary(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	userID, err := getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.requireServerMember(c, serverID, userID); err != nil {
		return err
	}

	summary, err := h.modService.GetUserViolationSummary(c.Context(), serverID, memberID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if summary == nil {
		return c.JSON(&models.UserViolationSummary{
			UserID:     memberID,
			ServerID:   serverID,
		})
	}

	return c.JSON(summary)
}

// ResetMemberViolations resets all violations for a member
// POST /api/v1/servers/:id/moderation/members/:memberId/reset
func (h *SmartModerationHandler) ResetMemberViolations(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	memberID, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}

	_, err = getModerationUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	if err := h.modService.ResetMemberViolations(c.Context(), serverID, memberID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
