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
	// code := c.Params("code")
	// TODO: Get invite
	return c.JSON(fiber.Map{})
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
	// TODO: Implement
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
