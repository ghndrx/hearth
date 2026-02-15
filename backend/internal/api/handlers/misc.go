package handlers

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
	ws "hearth/internal/websocket"
)

// InviteHandler handles invite operations
type InviteHandler struct {
	serverService *services.ServerService
}

func NewInviteHandler(serverService *services.ServerService) *InviteHandler {
	return &InviteHandler{serverService: serverService}
}

// Get returns invite info
func (h *InviteHandler) Get(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invite code is required",
		})
	}

	invite, err := h.serverService.GetInvite(c.Context(), code)
	if err != nil {
		if err == services.ErrInviteNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "invite not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(invite)
}

// Accept accepts an invite
func (h *InviteHandler) Accept(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	code := c.Params("code")

	server, err := h.serverService.JoinServer(c.Context(), userID, code)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server)
}

// Delete deletes an invite
func (h *InviteHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invite code is required",
		})
	}

	err := h.serverService.DeleteInvite(c.Context(), code, userID)
	if err != nil {
		if err == services.ErrInviteNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "invite not found",
			})
		}
		if err == services.ErrNotServerMember {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you don't have permission to delete this invite",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// VoiceHandler handles voice operations
type VoiceHandler struct{}

func NewVoiceHandler() *VoiceHandler {
	return &VoiceHandler{}
}

// GetRegions returns available voice regions
func (h *VoiceHandler) GetRegions(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"id": "us-west", "name": "US West", "optimal": true},
		{"id": "us-east", "name": "US East", "optimal": false},
		{"id": "eu-west", "name": "EU West", "optimal": false},
		{"id": "eu-central", "name": "EU Central", "optimal": false},
		{"id": "singapore", "name": "Singapore", "optimal": false},
		{"id": "sydney", "name": "Sydney", "optimal": false},
	})
}

// GatewayHandler handles WebSocket gateway connections
type GatewayHandler struct {
	gateway *ws.Gateway
}

func NewGatewayHandler(gateway *ws.Gateway) *GatewayHandler {
	return &GatewayHandler{
		gateway: gateway,
	}
}

// Connect handles WebSocket connection upgrade and delegates to Gateway
func (h *GatewayHandler) Connect(conn *websocket.Conn) {
	h.gateway.HandleConnection(conn)
}

// GetStats returns gateway statistics
func (h *GatewayHandler) GetStats(c *fiber.Ctx) error {
	return c.JSON(h.gateway.GetStats())
}

// Health returns health status for load balancer
// Returns 200 OK when healthy, 503 Service Unavailable when draining
// This is the primary health check endpoint for Kubernetes readiness probes
func (h *GatewayHandler) Health(c *fiber.Ctx) error {
	if h.gateway == nil {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"drain_state": "healthy",
		})
	}

	drainState := h.gateway.DrainState()
	isHealthy := h.gateway.IsHealthy()

	response := fiber.Map{
		"status":      "ok",
		"drain_state": drainState.String(),
		"connections": h.gateway.GetActiveConnections(),
	}

	if !isHealthy {
		response["status"] = "draining"
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}

	return c.JSON(response)
}

// ReadinessCheck checks if the server is ready to accept requests
// Returns 200 OK when ready, 503 Service Unavailable when not ready or draining
// Alias for /readyz Kubernetes-style endpoint
func (h *GatewayHandler) ReadinessCheck(c *fiber.Ctx) error {
	if h.gateway == nil {
		return c.JSON(fiber.Map{
			"ready": true,
		})
	}

	if h.gateway.IsDraining() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"ready":       false,
			"reason":      "draining",
			"drain_state": h.gateway.DrainState().String(),
		})
	}

	return c.JSON(fiber.Map{
		"ready":       true,
		"connections": h.gateway.GetActiveConnections(),
	})
}

// LivenessCheck checks if the server is alive (basic health)
// Always returns 200 if the process is running - for Kubernetes liveness probes
func (h *GatewayHandler) LivenessCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"alive": true,
	})
}
