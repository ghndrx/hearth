package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/websocket"
	
	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, h *handlers.Handlers, m *middleware.Middleware) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	
	// API v1
	v1 := app.Group("/api/v1")
	
	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register", h.Auth.Register)
	auth.Post("/login", h.Auth.Login)
	auth.Post("/refresh", h.Auth.Refresh)
	auth.Post("/logout", h.Auth.Logout)
	auth.Get("/oauth/:provider", h.Auth.OAuthRedirect)
	auth.Get("/oauth/:provider/callback", h.Auth.OAuthCallback)
	
	// Protected routes
	api := v1.Group("", m.RequireAuth)
	
	// Users
	users := api.Group("/users")
	users.Get("/@me", h.Users.GetMe)
	users.Patch("/@me", h.Users.UpdateMe)
	users.Get("/@me/servers", h.Users.GetMyServers)
	users.Get("/@me/channels", h.Users.GetMyDMs)
	users.Get("/:id", h.Users.GetUser)
	
	// Relationships
	users.Get("/@me/relationships", h.Users.GetRelationships)
	users.Post("/@me/relationships", h.Users.CreateRelationship)
	users.Delete("/@me/relationships/:id", h.Users.DeleteRelationship)
	
	// Servers
	servers := api.Group("/servers")
	servers.Post("/", h.Servers.Create)
	servers.Get("/:id", h.Servers.Get)
	servers.Patch("/:id", h.Servers.Update)
	servers.Delete("/:id", h.Servers.Delete)
	
	// Server members
	servers.Get("/:id/members", h.Servers.GetMembers)
	servers.Get("/:id/members/:userId", h.Servers.GetMember)
	servers.Patch("/:id/members/:userId", h.Servers.UpdateMember)
	servers.Delete("/:id/members/:userId", h.Servers.RemoveMember)
	servers.Delete("/:id/members/@me", h.Servers.Leave)
	
	// Server bans
	servers.Get("/:id/bans", h.Servers.GetBans)
	servers.Put("/:id/bans/:userId", h.Servers.CreateBan)
	servers.Delete("/:id/bans/:userId", h.Servers.RemoveBan)
	
	// Server invites
	servers.Get("/:id/invites", h.Servers.GetInvites)
	
	// Server roles
	servers.Get("/:id/roles", h.Servers.GetRoles)
	servers.Post("/:id/roles", h.Servers.CreateRole)
	servers.Patch("/:id/roles/:roleId", h.Servers.UpdateRole)
	servers.Delete("/:id/roles/:roleId", h.Servers.DeleteRole)
	
	// Channels
	channels := api.Group("/channels")
	channels.Get("/:id", h.Channels.Get)
	channels.Patch("/:id", h.Channels.Update)
	channels.Delete("/:id", h.Channels.Delete)
	
	// Channel messages
	channels.Get("/:id/messages", h.Channels.GetMessages)
	channels.Post("/:id/messages", h.Channels.SendMessage)
	channels.Get("/:id/messages/:messageId", h.Channels.GetMessage)
	channels.Patch("/:id/messages/:messageId", h.Channels.EditMessage)
	channels.Delete("/:id/messages/:messageId", h.Channels.DeleteMessage)
	
	// Reactions
	channels.Put("/:id/messages/:messageId/reactions/:emoji/@me", h.Channels.AddReaction)
	channels.Delete("/:id/messages/:messageId/reactions/:emoji/@me", h.Channels.RemoveReaction)
	
	// Pins
	channels.Get("/:id/pins", h.Channels.GetPins)
	channels.Put("/:id/pins/:messageId", h.Channels.PinMessage)
	channels.Delete("/:id/pins/:messageId", h.Channels.UnpinMessage)
	
	// Typing indicator
	channels.Post("/:id/typing", h.Channels.TriggerTyping)
	
	// Server channels
	servers.Get("/:id/channels", h.Servers.GetChannels)
	servers.Post("/:id/channels", h.Servers.CreateChannel)
	
	// Invites
	invites := api.Group("/invites")
	invites.Get("/:code", h.Invites.Get)
	invites.Post("/:code", h.Invites.Accept)
	invites.Delete("/:code", h.Invites.Delete)
	
	// Channel invites
	channels.Post("/:id/invites", h.Channels.CreateInvite)
	
	// Voice
	voice := api.Group("/voice")
	voice.Get("/regions", h.Voice.GetRegions)
	voice.Post("/join", h.Voice.JoinVoice)
	voice.Post("/leave", h.Voice.LeaveVoice)
	voice.Patch("/state", h.Voice.UpdateVoiceState)
	voice.Get("/state/@me", h.Voice.GetMyVoiceState)
	
	// Voice states for channels
	channels.Get("/:id/voice-states", h.Voice.GetChannelVoiceStates)
	
	// Gateway stats (admin)
	api.Get("/gateway/stats", h.Gateway.GetStats)
	
	// WebSocket gateway
	app.Get("/gateway", m.WebSocketUpgrade, websocket.New(h.Gateway.Connect))
	
	// Static files (for self-hosted frontend)
	app.Static("/", "./public")
	
	// SPA fallback
	app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})
}
