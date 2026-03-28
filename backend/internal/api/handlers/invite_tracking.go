package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// InviteTrackingAnalytics represents detailed invite analytics with fake account detection
type InviteTrackingAnalytics struct {
	Code      string     `json:"code"`
	Uses      int        `json:"uses"`
	RealUsers int        `json:"real_users"`
	Fakes     int        `json:"fakes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// accountAgeThreshold is the minimum account age (in days) to consider a user "real".
// Accounts younger than this are flagged as potential fakes.
const accountAgeThreshold = 7

// InviteTrackingHandler handles invite tracking analytics
type InviteTrackingHandler struct {
	serverService *services.ServerService
}

// NewInviteTrackingHandler creates a new invite tracking handler
func NewInviteTrackingHandler(serverService *services.ServerService) *InviteTrackingHandler {
	return &InviteTrackingHandler{
		serverService: serverService,
	}
}

// GetInviteTrackingAnalytics returns per-invite analytics with real vs fake user breakdown
// GET /api/v1/servers/:id/analytics/invites
func (h *InviteTrackingHandler) GetInviteTrackingAnalytics(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Fetch per-invite analytics (includes permission check for MANAGE_SERVER)
	allAnalytics, err := h.serverService.GetServerInviteAnalytics(c.Context(), serverID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	// Fetch invites to get created_at / expires_at metadata
	invites, err := h.serverService.GetInvites(c.Context(), serverID)
	if err != nil {
		return h.handleError(c, err)
	}

	// Build invite metadata lookup
	type inviteMeta struct {
		CreatedAt time.Time
		ExpiresAt *time.Time
	}
	metaMap := make(map[string]inviteMeta, len(invites))
	for _, inv := range invites {
		metaMap[inv.Code] = inviteMeta{CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt}
	}

	// Classify each invite's users as real or fake based on account age
	results := make([]InviteTrackingAnalytics, 0, len(allAnalytics))
	for _, a := range allAnalytics {
		realUsers := 0
		fakes := 0
		for _, log := range a.UseLogs {
			if log.AccountAgeDays >= accountAgeThreshold {
				realUsers++
			} else {
				fakes++
			}
		}

		entry := InviteTrackingAnalytics{
			Code:      a.Code,
			Uses:      a.TotalUses,
			RealUsers: realUsers,
			Fakes:     fakes,
		}
		if meta, ok := metaMap[a.Code]; ok {
			entry.CreatedAt = meta.CreatedAt
			entry.ExpiresAt = meta.ExpiresAt
		}
		results = append(results, entry)
	}

	return c.JSON(fiber.Map{
		"server_id": serverID.String(),
		"invites":   results,
	})
}

// handleError converts service errors to HTTP responses
func (h *InviteTrackingHandler) handleError(c *fiber.Ctx, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "server not found",
		})
	case services.ErrNotServerMember:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "not a member of this server",
		})
	case services.ErrMissingPermission:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "missing MANAGE_SERVER permission",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
}
