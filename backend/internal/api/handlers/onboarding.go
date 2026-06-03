package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// WelcomeHandler handles welcome screen HTTP endpoints
type WelcomeHandler struct {
	welcomeService *services.WelcomeService
	userService    UserServiceInterface
}

// NewWelcomeHandler creates a new welcome handler
func NewWelcomeHandler(welcomeService *services.WelcomeService, userService UserServiceInterface) *WelcomeHandler {
	return &WelcomeHandler{
		welcomeService: welcomeService,
		userService:    userService,
	}
}

// GetWelcomeScreen returns the welcome screen configuration for a server
// @Summary Get welcome screen
// @Description Returns the welcome screen configuration including rules and screening questions
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} models.WelcomeScreenConfig
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/welcome [get]
func (h *WelcomeHandler) GetWelcomeScreen(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	config, err := h.welcomeService.GetWelcomeScreen(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(config)
}

// UpdateWelcomeScreen updates the welcome screen configuration
// @Summary Update welcome screen
// @Description Updates the welcome screen configuration including rules and screening questions
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body models.UpdateWelcomeScreenRequest true "Welcome screen update data"
// @Success 200 {object} models.WelcomeScreenConfig
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/welcome [put]
func (h *WelcomeHandler) UpdateWelcomeScreen(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	var req models.UpdateWelcomeScreenRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	config, err := h.welcomeService.UpdateWelcomeScreen(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(config)
}

// SubmitScreening submits a member's screening answers
// @Summary Submit screening
// @Description New member submits screening answers for server approval
// @Tags Servers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body models.SubmitScreeningRequest true "Screening answers"
// @Success 200 {object} models.MemberScreening
// @Failure 400 {object} fiber.Map "Invalid server ID or request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 409 {object} fiber.Map "Screening already submitted"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/screening [post]
func (h *WelcomeHandler) SubmitScreening(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	var req models.SubmitScreeningRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	screening, err := h.welcomeService.SubmitScreening(c.Context(), userID, serverID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(screening)
}

// GetMemberScreening returns the current user's screening status for a server
// @Summary Get my screening status
// @Description Returns the current user's screening status for a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {object} models.MemberScreening
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "No screening found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/screening/me [get]
func (h *WelcomeHandler) GetMemberScreening(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	screening, err := h.welcomeService.GetMemberScreening(c.Context(), userID, serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if screening == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "no screening found",
		})
	}

	return c.JSON(screening)
}

// GetPendingScreenings returns pending screenings for a server (moderators only)
// @Summary Get pending screenings
// @Description Returns a list of pending member screenings for a server
// @Tags Servers
// @Produce json
// @Param id path string true "Server ID"
// @Param limit query int false "Number of screenings to return (default 50, max 100)"
// @Param offset query int false "Offset for pagination (default 0)"
// @Success 200 {array} models.MemberScreening
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/screening/pending [get]
func (h *WelcomeHandler) GetPendingScreenings(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	screenings, err := h.welcomeService.GetPendingScreenings(c.Context(), serverID, userID, limit, offset)
	if err != nil {
		return HandleServiceError(c, err)
	}

	if screenings == nil {
		screenings = []*models.MemberScreening{}
	}

	return c.JSON(screenings)
}

// ScreeningDecisionRequest is the input for screening decisions
type ScreeningDecisionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ApproveScreening approves a member's screening (moderators only)
// @Summary Approve screening
// @Description Approves a member's screening application
// @Tags Servers
// @Param id path string true "Server ID"
// @Param userId path string true "User ID whose screening to approve"
// @Success 204 "Screening approved successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/screening/{userId}/approve [post]
func (h *WelcomeHandler) ApproveScreening(c *fiber.Ctx) error {
	moderatorID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	if err := h.welcomeService.ApproveScreening(c.Context(), userID, serverID, moderatorID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RejectScreening rejects a member's screening (moderators only)
// @Summary Reject screening
// @Description Rejects a member's screening and removes them from the server
// @Tags Servers
// @Accept json
// @Param id path string true "Server ID"
// @Param userId path string true "User ID whose screening to reject"
// @Param body body ScreeningDecisionRequest false "Rejection reason"
// @Success 204 "Screening rejected successfully"
// @Failure 400 {object} fiber.Map "Invalid server ID or user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Insufficient permissions"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/screening/{userId}/reject [post]
func (h *WelcomeHandler) RejectScreening(c *fiber.Ctx) error {
	moderatorID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "server id")
	}

	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	var req ScreeningDecisionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.welcomeService.RejectScreening(c.Context(), userID, serverID, moderatorID, req.Reason); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
