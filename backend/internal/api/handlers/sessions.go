package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// SessionHandler handles session/device management endpoints
type SessionHandler struct {
	sessionService services.SessionService
}

// NewSessionHandler creates a new SessionHandler
func NewSessionHandler(sessionService services.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// SessionResponse represents a session in API responses
type SessionResponse struct {
	ID           string    `json:"id"`
	DeviceName   string    `json:"device_name,omitempty"`
	DeviceType   string    `json:"device_type,omitempty"`
	IP           string    `json:"ip,omitempty"`
	Location     string    `json:"location,omitempty"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
	IsCurrent    bool      `json:"is_current,omitempty"`
}

// GetSessions returns all active sessions for the current user
// @Summary Get user sessions
// @Description Returns all active sessions for the current user, including device information
// @Tags Sessions
// @Produce json
// @Success 200 {object} fiber.Map "List of sessions"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /auth/sessions [get]
func (h *SessionHandler) GetSessions(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	// Get current session ID from claims if available
	var currentSessionID *uuid.UUID
	if sessionIDStr, ok := c.Locals("sessionID").(string); ok {
		if sid, err := uuid.Parse(sessionIDStr); err == nil {
			currentSessionID = &sid
		}
	}

	sessions, err := h.sessionService.GetUserSessions(c.Context(), userID, currentSessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve sessions",
		})
	}

	return c.JSON(fiber.Map{
		"sessions": sessions,
	})
}

// RevokeSession revokes a specific session
// @Summary Revoke a session
// @Description Revokes a specific session by ID. Cannot revoke the current session (use logout instead).
// @Tags Sessions
// @Param id path string true "Session ID"
// @Success 204 "Session revoked successfully"
// @Failure 400 {object} fiber.Map "Invalid session ID or cannot revoke current session"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - cannot revoke this session"
// @Failure 404 {object} fiber.Map "Session not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /auth/sessions/{id} [delete]
func (h *SessionHandler) RevokeSession(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	sessionIDParam := c.Params("id")
	if sessionIDParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Session ID is required",
		})
	}

	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid session ID format",
		})
	}

	// Check if user is trying to revoke their current session
	if currentSessionID, ok := c.Locals("sessionID").(string); ok {
		if currentSID, err := uuid.Parse(currentSessionID); err == nil && currentSID == sessionID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Cannot revoke current session. Use logout instead.",
			})
		}
	}

	err = h.sessionService.RevokeSession(c.Context(), userID, sessionID)
	if err != nil {
		switch err {
		case services.ErrSessionNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "session_not_found",
				"message": "Session not found",
			})
		case services.ErrUnauthorized:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Cannot revoke this session",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "Failed to revoke session",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RevokeAllSessions revokes all sessions except the current one
// @Summary Revoke all other sessions
// @Description Revokes all sessions for the current user except the current one
// @Tags Sessions
// @Produce json
// @Success 200 {object} fiber.Map "Success message"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /auth/sessions [delete]
func (h *SessionHandler) RevokeAllSessions(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	// Get current session ID to exclude
	var exceptSessionID *uuid.UUID
	if currentSessionID, ok := c.Locals("sessionID").(string); ok {
		if currentSID, err := uuid.Parse(currentSessionID); err == nil {
			exceptSessionID = &currentSID
		}
	}

	err := h.sessionService.RevokeAllSessions(c.Context(), userID, exceptSessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to revoke sessions",
		})
	}

	return c.JSON(fiber.Map{
		"message": "All other sessions have been revoked",
	})
}
