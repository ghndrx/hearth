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
	users.Post("/@me/channels", h.Users.CreateDM)
	users.Post("/@me/channels/group", h.Users.CreateGroupDM)
	users.Get("/:id", h.Users.GetUser)
	
	// User Settings
	if h.Settings != nil {
		users.Get("/@me/settings", h.Settings.GetSettings)
		users.Patch("/@me/settings", h.Settings.UpdateSettings)
		users.Delete("/@me/settings", h.Settings.ResetSettings)
	}
	
	// Notifications
	if h.Notifications != nil {
		notifications := api.Group("/notifications")
		notifications.Get("/", h.Notifications.GetNotifications)
		notifications.Get("/stats", h.Notifications.GetNotificationStats)
		notifications.Post("/read-all", h.Notifications.MarkAllAsRead)
		notifications.Delete("/read", h.Notifications.DeleteAllRead)
		notifications.Get("/:id", h.Notifications.GetNotification)
		notifications.Post("/:id/read", h.Notifications.MarkAsRead)
		notifications.Delete("/:id", h.Notifications.DeleteNotification)
	}
	
	// Saved Messages (Bookmarks)
	if h.SavedMessages != nil {
		savedMessages := users.Group("/@me/saved-messages")
		savedMessages.Post("/", h.SavedMessages.SaveMessage)
		savedMessages.Get("/", h.SavedMessages.GetSavedMessages)
		savedMessages.Get("/count", h.SavedMessages.GetSavedCount)
		savedMessages.Get("/check/:messageId", h.SavedMessages.IsSaved)
		savedMessages.Get("/:id", h.SavedMessages.GetSavedMessage)
		savedMessages.Patch("/:id", h.SavedMessages.UpdateSavedMessage)
		savedMessages.Delete("/:id", h.SavedMessages.RemoveSavedMessage)
		savedMessages.Delete("/message/:messageId", h.SavedMessages.RemoveSavedMessageByMessage)
	}
	
	// Relationships
	users.Get("/@me/relationships", h.Users.GetRelationships)
	users.Post("/@me/relationships", h.Users.CreateRelationship)
	users.Delete("/@me/relationships/:id", h.Users.DeleteRelationship)
	
	// Friends
	users.Get("/@me/friends", h.Users.GetFriends)
	users.Get("/@me/friends/pending", h.Users.GetPendingFriendRequests)
	users.Put("/@me/friends/:id", h.Users.AcceptFriendRequest)
	users.Delete("/@me/friends/:id/request", h.Users.DeclineFriendRequest)
	
	// Servers
	servers := api.Group("/servers")
	servers.Post("/", h.Servers.Create)
	servers.Get("/:id", h.Servers.Get)
	servers.Patch("/:id", h.Servers.Update)
	servers.Delete("/:id", h.Servers.Delete)
	servers.Post("/:id/transfer-ownership", h.Servers.TransferOwnership)
	
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
	
	// Server audit logs
	if h.AuditLog != nil {
		servers.Get("/:id/audit-logs", h.AuditLog.GetAuditLogs)
		servers.Get("/:id/audit-logs/action-types", h.AuditLog.GetActionTypes)
		servers.Get("/:id/audit-logs/:entryId", h.AuditLog.GetAuditLogEntry)
	}
	
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
	channels.Get("/:id/messages/:messageId/reactions", h.Channels.GetReactions)
	channels.Get("/:id/messages/:messageId/reactions/:emoji", h.Channels.GetReactionUsers)
	channels.Put("/:id/messages/:messageId/reactions/:emoji/@me", h.Channels.AddReaction)
	channels.Delete("/:id/messages/:messageId/reactions/:emoji/@me", h.Channels.RemoveReaction)
	
	// Pins
	channels.Get("/:id/pins", h.Channels.GetPins)
	channels.Put("/:id/pins/:messageId", h.Channels.PinMessage)
	channels.Delete("/:id/pins/:messageId", h.Channels.UnpinMessage)
	
	// Typing indicator
	channels.Post("/:id/typing", h.Channels.TriggerTyping)
	channels.Get("/:id/typing", h.Channels.GetTypingUsers)
	
	// Channel threads
	channels.Get("/:id/threads", h.Threads.GetChannelThreads)
	channels.Post("/:id/threads", h.Threads.CreateThread)
	
	// Threads
	threads := api.Group("/threads")
	threads.Get("/:id", h.Threads.GetThread)
	threads.Delete("/:id", h.Threads.DeleteThread)
	threads.Get("/:id/messages", h.Threads.GetThreadMessages)
	threads.Post("/:id/messages", h.Threads.SendThreadMessage)
	threads.Post("/:id/archive", h.Threads.ArchiveThread)
	threads.Post("/:id/unarchive", h.Threads.UnarchiveThread)
	threads.Post("/:id/join", h.Threads.JoinThread)
	threads.Delete("/:id/members/@me", h.Threads.LeaveThread)
	
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
	
	// Channel polls
	if h.Polls != nil {
		channels.Get("/:id/polls", h.Polls.GetChannelPolls)
		channels.Post("/:id/polls", h.Polls.CreatePoll)
	}
	
	// Polls
	if h.Polls != nil {
		polls := api.Group("/polls")
		polls.Get("/:id", h.Polls.GetPoll)
		polls.Get("/:id/results", h.Polls.GetResults)
		polls.Post("/:id/vote", h.Polls.Vote)
		polls.Post("/:id/close", h.Polls.ClosePoll)
		polls.Delete("/:id", h.Polls.DeletePoll)
	}
	
	// Attachments (if handler is configured)
	if h.Attachments != nil {
		// Channel attachments
		channels.Post("/:id/attachments", h.Attachments.Upload)
		channels.Get("/:id/attachments", h.Attachments.GetChannelAttachments)
		
		// Attachments
		attachments := api.Group("/attachments")
		attachments.Get("/:id", h.Attachments.Get)
		attachments.Get("/:id/download", h.Attachments.Download)
		attachments.Get("/:id/signed-url", h.Attachments.GetSignedURL)
		attachments.Delete("/:id", h.Attachments.Delete)
	}
	
	// Search
	search := api.Group("/search")
	search.Get("/", h.Search.SearchAll)
	search.Get("/messages", h.Search.SearchMessages)
	search.Get("/users", h.Search.SearchUsers)
	search.Get("/channels", h.Search.SearchChannels)
	
	// Voice
	voice := api.Group("/voice")
	voice.Get("/regions", h.Voice.GetRegions)
	
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
