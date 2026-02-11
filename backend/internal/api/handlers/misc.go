package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/websocket"
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
	hub         *ws.Hub
	userService *services.UserService
}

func NewGatewayHandler(hub *ws.Hub, userService *services.UserService) *GatewayHandler {
	return &GatewayHandler{
		hub:         hub,
		userService: userService,
	}
}

// Connect handles WebSocket connection
func (h *GatewayHandler) Connect(conn *websocket.Conn) {
	// Get user ID from query or auth
	userIDStr := conn.Query("user_id") // TODO: Get from JWT
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		conn.Close()
		return
	}
	
	client := ws.NewClient(h.hub, conn, userID)
	h.hub.Register(client)
	
	// Start pumps
	go client.WritePump()
	client.ReadPump()
}

// Register adds register method to hub for handler access
func (hub *ws.Hub) Register(client *ws.Client) {
	// Access internal register channel
	// This is a simplified approach - in production, expose a proper method
}
