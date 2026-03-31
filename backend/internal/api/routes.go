package api

import (
	"fmt"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, h *handlers.Handlers, m *middleware.Middleware) {
	// Health check endpoints for Kubernetes/load balancers
	// /health - Returns 503 when draining (for graceful shutdown)
	app.Get("/health", h.Gateway.Health)
	// /healthz - Kubernetes-style liveness probe (always returns 200 if process is alive)
	app.Get("/healthz", h.Gateway.LivenessCheck)
	// /readyz - Kubernetes-style readiness probe (returns 503 when draining)
	app.Get("/readyz", h.Gateway.ReadinessCheck)

	// Per-endpoint rate limiting (applied selectively below)
	// Auth: 300+50=350 req/s per IP (supports load test target of 100+ req/s)
	authRateLimit := m.RateLimitWithConfig(middleware.RateLimitConfig{
		Limit:  350,
		Window: time.Second,
	})
	// Messages: 500+100=600 req/s per user
	messageRateLimit := m.RateLimitWithConfig(middleware.RateLimitConfig{
		Limit:          600,
		Window:         time.Second,
		AuthMultiplier: 1.0,
	})
	// Invites: 5 per hour per user per channel (prevent invite spam / abuse)
	inviteRateLimit := m.RateLimitWithConfig(middleware.RateLimitConfig{
		Limit:  5,
		Window: time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			channelID := c.Params("id")
			if userID, ok := c.Locals("userID").(uuid.UUID); ok {
				return fmt.Sprintf("invite:user:%s:channel:%s", userID.String(), channelID)
			}
			return fmt.Sprintf("invite:ip:%s:channel:%s", c.IP(), channelID)
		},
	})

	// API v1
	v1 := app.Group("/api/v1")

	// Auth routes (public, with auth-specific rate limit)
	auth := v1.Group("/auth", authRateLimit)
	auth.Post("/register", h.Auth.Register)
	auth.Post("/login", h.Auth.Login)
	auth.Post("/login/mfa", h.Auth.LoginWithMFA)
	auth.Post("/refresh", h.Auth.Refresh)
	auth.Post("/logout", h.Auth.Logout)

	// MFA routes (protected)
	mfa := v1.Group("/auth/mfa", m.RequireAuth)
	mfa.Post("/enable", h.Auth.EnableMFA)
	mfa.Post("/verify", h.Auth.VerifyMFASetup)
	mfa.Post("/disable", h.Auth.DisableMFA)

	// OAuth routes (public)
	auth.Get("/oauth/providers", h.Auth.GetEnabledProviders)
	auth.Get("/oauth/:provider", h.Auth.OAuthRedirect)
	auth.Get("/oauth/:provider/callback", h.Auth.OAuthCallback)

	// OAuth account linking routes (protected)
	oauthProtected := v1.Group("/auth/oauth", m.RequireAuth)
	oauthProtected.Get("/linked", h.Auth.GetLinkedProviders)
	oauthProtected.Get("/link/:provider", h.Auth.OAuthLinkRedirect)
	oauthProtected.Delete("/link/:provider", h.Auth.OAuthUnlink)

	// Session management (protected)
	if h.Sessions != nil {
		authProtected := v1.Group("/auth", m.RequireAuth)
		authProtected.Get("/sessions", h.Sessions.GetSessions)
		authProtected.Delete("/sessions", h.Sessions.RevokeAllSessions)
		authProtected.Delete("/sessions/:id", h.Sessions.RevokeSession)
	}

	// Public Server Directory (no auth required)
	if h.Discovery != nil {
		publicServers := v1.Group("/servers")
		publicServers.Get("/", h.Discovery.GetPublicServers)
		publicServers.Get("/categories", h.Discovery.GetPublicCategories)
		publicServers.Get("/:id", h.Discovery.GetPublicServer)
	}

	// Enhanced Public Server Discovery (new endpoints)
	if h.DiscoverableServer != nil {
		discoverServers := v1.Group("/servers")
		discoverServers.Get("/discover", h.DiscoverableServer.GetDiscoverableServers)
		discoverServers.Get("/discover/featured", h.DiscoverableServer.GetFeaturedServers)
		discoverServers.Get("/discover/trending", h.DiscoverableServer.GetTrendingServers)
		discoverServers.Get("/discover/recommended", h.DiscoverableServer.GetRecommendedServers)
		discoverServers.Get("/discover/search", h.DiscoverableServer.SearchServersEnhanced)
		discoverServers.Get("/discover/home", h.DiscoverableServer.GetDiscoveryHomePage)
		discoverServers.Get("/discover/categories/stats", h.DiscoverableServer.GetCategoriesWithStats)
		discoverServers.Get("/discover/tags", h.DiscoverableServer.GetPopularTags)
		discoverServers.Get("/discover/stats", h.DiscoverableServer.GetDiscoveryStats)
		discoverServers.Get("/discover/suggestions", h.DiscoverableServer.GetSearchSuggestions)
		discoverServers.Get("/categories", h.DiscoverableServer.GetCategories)
		discoverServers.Get("/:id", h.DiscoverableServer.GetServerDetail)

		// Public Discovery API (GET /discovery/servers, GET /discovery/categories)
		discoveryPublic := v1.Group("/discovery")
		discoveryPublic.Get("/servers", h.DiscoverableServer.GetDiscoverableServers)
		discoveryPublic.Get("/categories", h.DiscoverableServer.GetCategories)
		discoveryPublic.Get("/servers/search", h.DiscoverableServer.SearchServersEnhanced)
		discoveryPublic.Get("/servers/featured", h.DiscoverableServer.GetFeaturedServers)
		discoveryPublic.Get("/servers/trending", h.DiscoverableServer.GetTrendingServers)
		discoveryPublic.Get("/servers/tags", h.DiscoverableServer.GetPopularTags)
		discoveryPublic.Get("/servers/:id", h.DiscoverableServer.GetServerDetail)
	}

	// Protected routes
	api := v1.Group("", m.RequireAuth)

	// Users
	users := api.Group("/users")
	users.Get("/@me", h.Users.GetMe)
	users.Patch("/@me", h.Users.UpdateMe)
	users.Post("/@me/avatar", h.Users.UpdateAvatar)
	users.Delete("/@me/avatar", h.Users.DeleteAvatar)
	users.Post("/@me/banner", h.Users.UpdateBanner)
	users.Delete("/@me/banner", h.Users.DeleteBanner)
	users.Get("/@me/status", h.Users.GetMyStatus)
	users.Put("/@me/status", h.Users.UpdateMyStatus)
	users.Delete("/@me/status", h.Users.DeleteMyStatus)
	users.Get("/@me/servers", h.Users.GetMyServers)
	users.Get("/@me/channels", h.Users.GetMyDMs)
	users.Post("/@me/channels", h.Users.CreateDM)
	users.Post("/@me/channels/group", h.Users.CreateGroupDM)
	users.Get("/:id", h.Users.GetUser)
	users.Get("/:id/profile", h.Users.GetUserProfile)

	// User Settings
	if h.Settings != nil {
		users.Get("/@me/settings", h.Settings.GetSettings)
		users.Patch("/@me/settings", h.Settings.UpdateSettings)
		users.Delete("/@me/settings", h.Settings.ResetSettings)
	}

	// Read State / Unread
	if h.ReadState != nil {
		users.Get("/@me/unread", h.ReadState.GetUnreadSummary)
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

	// Mentions
	if h.Mentions != nil {
		mentions := api.Group("/mentions")
		mentions.Get("/", h.Mentions.GetMentions)
		mentions.Get("/unread/count", h.Mentions.GetUnreadCount)
		mentions.Get("/stats", h.Mentions.GetStats)
		mentions.Get("/search", h.Mentions.Search)
		mentions.Post("/read-all", h.Mentions.MarkAllAsRead)
		mentions.Patch("/:id/read", h.Mentions.MarkAsRead)
		mentions.Post("/channel/:channelId/read-all", h.Mentions.MarkChannelMentionsAsRead)
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

	// DMs (dedicated routes)
	if h.DMs != nil {
		dms := api.Group("/dms")
		dms.Get("/", h.DMs.GetUserDMs)
		dms.Post("/", h.DMs.CreateDM)
		dms.Post("/group", h.DMs.CreateGroupDM)
		dms.Get("/:channelId/messages", h.DMs.GetDMMessages)
		dms.Post("/:channelId/messages", messageRateLimit, h.DMs.SendDMMessage)
		dms.Put("/:channelId/participants", h.DMs.AddParticipant)
		dms.Delete("/:channelId/participants", h.DMs.RemoveParticipant)
		dms.Delete("/:channelId/leave", h.DMs.LeaveDM)

		// Convenience route: create DM with a specific user
		users.Post("/:id/dm", h.DMs.CreateDMWithUser)
	}

	// Servers
	servers := api.Group("/servers")
	servers.Post("/", h.Servers.Create)
	servers.Get("/:id", h.Servers.Get)
	servers.Patch("/:id", h.Servers.Update)
	servers.Delete("/:id", h.Servers.Delete)
	servers.Post("/:id/transfer-ownership", h.Servers.TransferOwnership)
	servers.Post("/:id/join", h.DiscoverableServer.JoinServer)

	// Server discovery registration (auth-protected, server owner only)
	if h.DiscoverableServer != nil {
		servers.Post("/:serverId/discover", h.DiscoverableServer.RegisterServer)
		servers.Patch("/discover/:id", h.DiscoverableServer.UpdateRegisteredServer)
		servers.Delete("/discover/:id", h.DiscoverableServer.DeleteRegisteredServer)
	}

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
	servers.Put("/:id/vanity-url", h.Servers.SetVanityURL)
	servers.Get("/:id/invites/analytics", h.Servers.GetInviteAnalytics)

	// Server roles
	servers.Get("/:id/roles", h.Servers.GetRoles)
	servers.Post("/:id/roles", h.Servers.CreateRole)
	servers.Patch("/:id/roles", h.Servers.BatchUpdateRolesPositions)
	servers.Patch("/:id/roles/:roleId", h.Servers.UpdateRole)
	servers.Delete("/:id/roles/:roleId", h.Servers.DeleteRole)

	// Server audit logs
	if h.AuditLog != nil {
		servers.Get("/:id/audit-logs", h.AuditLog.GetAuditLogs)
		servers.Get("/:id/audit-logs/action-types", h.AuditLog.GetActionTypes)
		servers.Get("/:id/audit-logs/:entryId", h.AuditLog.GetAuditLogEntry)
	}

	// Welcome screens and onboarding
	if h.Welcome != nil {
		servers.Get("/:id/welcome", h.Welcome.GetWelcomeScreen)
		servers.Put("/:id/welcome", h.Welcome.UpdateWelcomeScreen)
		servers.Post("/:id/screening", h.Welcome.SubmitScreening)
		servers.Get("/:id/screening/me", h.Welcome.GetMemberScreening)
		servers.Get("/:id/screening/pending", h.Welcome.GetPendingScreenings)
		servers.Post("/:id/screening/:userId/approve", h.Welcome.ApproveScreening)
		servers.Post("/:id/screening/:userId/reject", h.Welcome.RejectScreening)
	}

	// Server read state / Ack
	if h.ReadState != nil {
		servers.Get("/:id/unread", h.ReadState.GetServerUnread)
		servers.Post("/:id/ack", h.ReadState.MarkServerAsRead)
	}

	// Channels
	channels := api.Group("/channels")
	channels.Get("/:id", h.Channels.Get)
	channels.Patch("/:id", h.Channels.Update)
	channels.Delete("/:id", h.Channels.Delete)

	// Channel messages (send/edit have per-user rate limits)
	channels.Get("/:id/messages", h.Channels.GetMessages)
	channels.Post("/:id/messages", messageRateLimit, h.Channels.SendMessage)
	channels.Get("/:id/messages/:messageId", h.Channels.GetMessage)
	channels.Patch("/:id/messages/:messageId", messageRateLimit, h.Channels.EditMessage)
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

	// Announcement channel following (if handler is configured)
	if h.Announcements != nil {
		channels.Get("/:id/followers", h.Announcements.GetFollowers)
		channels.Post("/:id/followers", h.Announcements.FollowChannel)
		channels.Delete("/:id/followers/:webhookID", h.Announcements.UnfollowChannel)
		channels.Post("/:id/messages/:messageId/crosspost", h.Announcements.CrosspostMessage)
	}

	// Typing indicator
	channels.Post("/:id/typing", h.Channels.TriggerTyping)
	channels.Get("/:id/typing", h.Channels.GetTypingUsers)

	// Read state / Ack
	if h.ReadState != nil {
		channels.Post("/:id/ack", h.ReadState.MarkChannelAsRead)
		channels.Get("/:id/unread", h.ReadState.GetChannelUnread)
	}

	// Channel threads
	channels.Get("/:id/threads", h.Threads.GetChannelThreads)
	channels.Post("/:id/threads", h.Threads.CreateThread)

	// Threads
	threads := api.Group("/threads")
	threads.Get("/:id", h.Threads.GetThread)
	threads.Patch("/:id", h.Threads.UpdateThread)
	threads.Delete("/:id", h.Threads.DeleteThread)
	threads.Get("/:id/messages", h.Threads.GetThreadMessages)
	threads.Post("/:id/messages", h.Threads.SendThreadMessage)
	threads.Post("/:id/archive", h.Threads.ArchiveThread)
	threads.Post("/:id/unarchive", h.Threads.UnarchiveThread)
	threads.Post("/:id/join", h.Threads.JoinThread)
	threads.Delete("/:id/members/@me", h.Threads.LeaveThread)
	// Thread notification preferences
	threads.Get("/:id/notifications", h.Threads.GetNotificationPreference)
	threads.Put("/:id/notifications", h.Threads.SetNotificationPreference)
	// Thread presence (active viewers)
	threads.Get("/:id/presence", h.Threads.GetActiveViewers)
	threads.Post("/:id/presence", h.Threads.EnterThread)
	threads.Patch("/:id/presence", h.Threads.HeartbeatPresence)
	threads.Delete("/:id/presence", h.Threads.ExitThreadPresence)

	// Forum tags (thread tags for forum posts)
	threads.Put("/:id/tags", h.ForumTags.ApplyTags)
	threads.Get("/:id/tags", h.ForumTags.GetThreadTags)
	threads.Put("/:id/pin", h.ForumTags.PinThread)

	// Thread auto-archive
	if h.ThreadAutoArchive != nil {
		// Thread-level auto-archive status
		threads.Get("/:id/auto-archive", h.ThreadAutoArchive.GetThreadAutoArchiveStatus)

		// Channel-level auto-archive override
		channels.Get("/:id/auto-archive", h.ThreadAutoArchive.GetChannelAutoArchiveOverride)
		channels.Put("/:id/auto-archive", h.ThreadAutoArchive.SetChannelAutoArchiveOverride)
		channels.Delete("/:id/auto-archive", h.ThreadAutoArchive.DeleteChannelAutoArchiveOverride)

		// Server-level auto-archive settings
		servers.Get("/:id/auto-archive", h.ThreadAutoArchive.GetServerAutoArchiveSettings)
		servers.Patch("/:id/auto-archive", h.ThreadAutoArchive.UpdateServerAutoArchiveSettings)
		servers.Get("/:id/auto-archive/stats", h.ThreadAutoArchive.GetServerAutoArchiveStats)
	}

	// Forum channel tags
	channels.Get("/:id/tags", h.ForumTags.ListTags)
	channels.Post("/:id/tags", h.ForumTags.CreateTag)
	channels.Get("/:id/posts", h.ForumTags.ListPosts)

	// Global tag management
	api.Patch("/forum-tags/:tagId", h.ForumTags.UpdateTag)
	api.Delete("/forum-tags/:tagId", h.ForumTags.DeleteTag)

	// Message components
	if h.Components != nil {
		// Component interactions
		api.Post("/interactions/components", h.Components.HandleComponentInteractionV2)

		// Message components CRUD (nested under channels)
		channels.Get("/:id/messages/:messageId/components", h.Components.GetMessageComponents)
		channels.Patch("/:id/messages/:messageId/components", h.Components.UpdateMessageComponents)
		channels.Delete("/:id/messages/:messageId/components", h.Components.RemoveAllComponents)
	}

	// Slash commands (application commands) - authenticated
	if h.SlashCommands != nil {
		// Application command management
		api.Post("/applications/:appId/commands", h.SlashCommands.RegisterCommand)
		api.Get("/applications/:appId/commands", h.SlashCommands.GetCommands)
		api.Get("/applications/:appId/commands/:commandId", h.SlashCommands.GetCommand)
		api.Put("/applications/:appId/commands/:commandId", h.SlashCommands.UpdateCommand)
		api.Delete("/applications/:appId/commands/:commandId", h.SlashCommands.DeleteCommand)
		api.Post("/applications/:appId/commands/bulk", h.SlashCommands.BulkRegisterCommands)
	}

	// Interactions - public endpoint (token-based)
	if h.Interactions != nil {
		api.Post("/interactions", h.Interactions.HandleInteraction)
		api.Post("/interactions/:token/callback", h.Interactions.RespondToInteraction)
		api.Patch("/interactions/:token/messages/:messageId", h.Interactions.EditInteractionResponse)
		api.Delete("/interactions/:token/messages/:messageId", h.Interactions.DeleteInteractionResponse)
	}

	// Server channels
	servers.Get("/:id/channels", h.Servers.GetChannels)
	servers.Post("/:id/channels", h.Servers.CreateChannel)

	// Invites
	invites := api.Group("/invites")
	invites.Get("/:code", h.Invites.Get)
	invites.Post("/:code", h.Invites.Accept)
	invites.Delete("/:code", h.Invites.Delete)
	invites.Get("/:code/analytics", h.Invites.GetAnalytics)

	// Channel webhooks
	if h.Webhooks != nil {
		channels.Post("/:id/webhooks", h.Webhooks.CreateWebhook)
		channels.Get("/:id/webhooks", h.Webhooks.GetChannelWebhooks)
	}

	// Channel invites
	channels.Post("/:id/invites", inviteRateLimit, h.Channels.CreateInvite)

	// Permission overrides
	channels.Get("/:id/permission-overwrites", h.Channels.GetPermissionOverrides)
	channels.Put("/:id/permission-overwrites", h.Channels.SetPermissionOverride)
	channels.Delete("/:id/permission-overwrites/:targetType/:targetId", h.Channels.DeletePermissionOverride)

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

	// Stickers (if handler is configured)
	if h.Stickers != nil {
		// Global stickers
		api.Get("/stickers", h.Stickers.ListGlobalStickers)

		// Server-specific stickers
		servers.Get("/:id/stickers", h.Stickers.ListServerStickers)
		servers.Post("/:id/stickers", h.Stickers.CreateSticker)
		servers.Get("/:id/stickers/:stickerId", h.Stickers.GetSticker)
		servers.Patch("/:id/stickers/:stickerId", h.Stickers.ModifySticker)
		servers.Delete("/:id/stickers/:stickerId", h.Stickers.DeleteSticker)
	}

	// Server Templates (if handler is configured)
	if h.Templates != nil {
		// List public templates
		api.Get("/templates", h.Templates.ListPublicTemplates)

		// User's own templates
		api.Get("/users/me/templates", h.Templates.ListMyTemplates)

		// Server templates (create from server)
		servers.Post("/:id/templates", h.Templates.CreateTemplate)

		// Template operations by code
		api.Get("/templates/:code", h.Templates.GetTemplate)
		api.Post("/templates/:code/use", h.Templates.UseTemplate)

		// Template operations by ID (update/delete)
		api.Patch("/templates/:templateId", h.Templates.UpdateTemplate)
		api.Delete("/templates/:templateId", h.Templates.DeleteTemplate)
	}

	// Scheduled Events (if handler is configured)
	if h.Events != nil {
		// Server events
		servers.Get("/:id/events", h.Events.ListServerEvents)
		servers.Post("/:id/events", h.Events.CreateEvent)

		// Event operations
		api.Get("/events/:id", h.Events.GetEvent)
		api.Patch("/events/:id", h.Events.UpdateEvent)
		api.Delete("/events/:id", h.Events.DeleteEvent)

		// Event RSVPs
		api.Post("/events/:id/rsvp", h.Events.RSVP)
		api.Delete("/events/:id/rsvp", h.Events.RemoveRSVP)
		api.Get("/events/:id/users", h.Events.ListEventUsers)

		// Event actions
		api.Post("/events/:id/start", h.Events.StartEvent)
	}

	// AutoMod Rules (if handler is configured)
	if h.AutoMod != nil {
		// Server-scoped automod rules
		servers.Get("/:id/automod/rules", h.AutoMod.ListRules)
		servers.Post("/:id/automod/rules", h.AutoMod.CreateRule)

		// Global automod rule operations
		api.Get("/automod/rules/:id", h.AutoMod.GetRule)
		api.Patch("/automod/rules/:id", h.AutoMod.UpdateRule)
		api.Delete("/automod/rules/:id", h.AutoMod.DeleteRule)
		api.Get("/automod/rules/:id/stats", h.AutoMod.GetRuleStats)

		// Server-scoped automod alerts
		servers.Get("/:id/automod/alerts", h.AutoMod.ListAlerts)

		// Global automod alert operations
		api.Get("/automod/alerts/:id", h.AutoMod.GetAlert)
		api.Post("/automod/alerts/:id/resolve", h.AutoMod.ResolveAlert)

		// AutoMod testing
		api.Post("/automod/test", h.AutoMod.TestContent)
	}

	// Webhooks (authenticated CRUD)
	if h.Webhooks != nil {
		webhooks := api.Group("/webhooks")
		webhooks.Get("/:webhookID", h.Webhooks.GetWebhook)
		webhooks.Patch("/:webhookID", h.Webhooks.UpdateWebhook)
		webhooks.Delete("/:webhookID", h.Webhooks.DeleteWebhook)

		// Server webhooks
		servers.Get("/:id/webhooks", h.Webhooks.GetServerWebhooks)
	}

	// Webhook execution (public, token-based auth - no RequireAuth)
	if h.Webhooks != nil {
		v1.Post("/webhooks/:webhookID/:token", h.Webhooks.ExecuteWebhook)
	}

	// Search
	search := api.Group("/search")
	search.Get("/", h.Search.SearchAll)
	search.Get("/messages", h.Search.SearchMessages)
	search.Get("/users", h.Search.SearchUsers)
	search.Get("/channels", h.Search.SearchChannels)
	search.Get("/suggestions", h.Search.GetSuggestions)

	// Server Discovery & Browse
	if h.Discovery != nil {
		discovery := api.Group("/discovery")
		discovery.Get("/featured", h.Discovery.GetFeaturedServers)
		discovery.Get("/categories", h.Discovery.GetCategories)
		discovery.Get("/search", h.Discovery.SearchServersEnhanced)
		discovery.Get("/recommendations", h.Discovery.GetRecommendations)
		discovery.Get("/categories/:slug", h.Discovery.GetServersByCategory)
		discovery.Get("/servers/:serverId", h.Discovery.GetServerListing)
		discovery.Post("/report", h.Discovery.ReportServer)
		discovery.Get("/trending", h.Discovery.GetTrendingServers)
		discovery.Get("/stats", h.Discovery.GetDiscoveryStats)
		discovery.Get("/tags", h.Discovery.GetPopularTags)
		discovery.Get("/tags/servers", h.Discovery.GetServersByTags)
		discovery.Get("/page", h.Discovery.GetDiscoveryPage)

		// Server discovery listing management
		servers.Post("/:id/listing", h.Discovery.SubmitForDiscovery)
		servers.Patch("/:id/listing", h.Discovery.UpdateListing)

		// Admin discovery management (if user has admin role - handled in middleware)
		adminDiscovery := api.Group("/admin/discovery")
		adminDiscovery.Post("/:listingId/approve", h.Discovery.ApproveListing)
		adminDiscovery.Post("/:listingId/reject", h.Discovery.RejectListing)
		adminDiscovery.Post("/:listingId/featured", h.Discovery.SetFeatured)
	}

	// App Directory / Bot Marketplace
	if h.AppDirectory != nil {
		apps := api.Group("/apps")
		apps.Get("/", h.AppDirectory.ListApps)
		apps.Get("/categories", h.AppDirectory.ListCategories)
		apps.Get("/:id", h.AppDirectory.GetApp)
		apps.Post("/", h.AppDirectory.CreateApp)
		apps.Patch("/:id", h.AppDirectory.UpdateApp)
		apps.Delete("/:id", h.AppDirectory.DeleteApp)

		// App installations
		apps.Post("/:id/install/:serverId", h.AppDirectory.InstallApp)
		apps.Delete("/:id/install/:serverId", h.AppDirectory.UninstallApp)

		// App reviews
		apps.Get("/:id/reviews", h.AppDirectory.ListAppReviews)
		apps.Get("/:id/reviews/@me", h.AppDirectory.GetMyReviewForApp)
		apps.Post("/:id/reviews", h.AppDirectory.CreateReview)
		apps.Patch("/:id/reviews/:reviewId", h.AppDirectory.UpdateReview)
		apps.Delete("/:id/reviews/:reviewId", h.AppDirectory.DeleteReview)

		// Developer dashboard
		developer := api.Group("/developer")
		developer.Get("/apps", h.AppDirectory.ListDeveloperApps)
		developer.Get("/analytics", h.AppDirectory.GetDeveloperAnalytics)

		// Admin app management
		adminApps := api.Group("/admin/apps")
		adminApps.Post("/:appId/approve", h.AppDirectory.ApproveApp)
		adminApps.Post("/:appId/reject", h.AppDirectory.RejectApp)
		adminApps.Post("/:appId/suspend", h.AppDirectory.SuspendApp)
	}

	// Voice (WebRTC signaling)
	voice := api.Group("/voice")
	voice.Get("/regions", h.Voice.GetRegions)
	voice.Get("/channels/:channelId/states", h.Voice.GetChannelVoiceStates)

	// Voice (LiveKit)
	if h.LiveKitVoice != nil {
		voice.Post("/token", h.LiveKitVoice.GenerateToken)
		voice.Get("/participants/:channelId", h.LiveKitVoice.GetParticipants)
		voice.Delete("/participants/:channelId/:userId", h.LiveKitVoice.DisconnectParticipant)
		voice.Post("/participants/:channelId/:userId/mute", h.LiveKitVoice.MuteParticipant)
	}

	// Screen Share / Streams (if handler is configured)
	if h.ScreenShare != nil {
		// Channel streams
		channels.Post("/:id/streams", h.ScreenShare.StartStream)
		channels.Get("/:id/streams", h.ScreenShare.GetActiveStreamForChannel)

		// Stream operations
		streams := api.Group("/streams")
		streams.Get("/:streamId", h.ScreenShare.GetStreamInfo)
		streams.Patch("/:streamId", h.ScreenShare.UpdateStream)
		streams.Delete("/:streamId", h.ScreenShare.EndStream)
		streams.Post("/:streamId/join", h.ScreenShare.JoinStream)
		streams.Delete("/:streamId/leave", h.ScreenShare.LeaveStream)
	}

	// Live Streaming to Channels (if handler is configured)
	if h.Stream != nil {
		// Channel live streams
		channels.Post("/:id/stream/start", h.Stream.StartStream)
		channels.Post("/:id/stream/stop", h.Stream.StopStream)
		channels.Get("/:id/stream", h.Stream.GetActiveStream)

		// Live stream operations
		api.Get("/streams/:streamId", h.Stream.GetStream)
		api.Patch("/streams/:streamId", h.Stream.UpdateStream)
		api.Post("/streams/:streamId/join", h.Stream.JoinStream)
		api.Post("/streams/:streamId/leave", h.Stream.LeaveStream)
		api.Get("/streams/:streamId/viewers", h.Stream.GetStreamViewers)
	}

	// Server audio settings (per-server audio device preferences)
	if h.ServerAudioSettings != nil {
		users.Get("/@me/audio-settings", h.ServerAudioSettings.GetAllSettings)
		servers.Get("/:id/audio-settings", h.ServerAudioSettings.GetSettings)
		servers.Patch("/:id/audio-settings", h.ServerAudioSettings.UpdateSettings)
		servers.Delete("/:id/audio-settings", h.ServerAudioSettings.DeleteSettings)
	}

	// Premium & Server Boosts (if handler is configured)
	if h.Premium != nil {
		premium := api.Group("/premium")
		premium.Get("/subscription", h.Premium.GetSubscription)
		premium.Post("/subscribe", h.Premium.CreateSubscription)
		premium.Put("/subscription", h.Premium.UpdateSubscription)
		premium.Delete("/subscription", h.Premium.CancelSubscription)
		premium.Post("/subscription/reactivate", h.Premium.ReactivateSubscription)
		premium.Get("/boosts", h.Premium.GetUserBoosts)
		premium.Get("/features/:feature/check", h.Premium.CheckFeatureAccess)
		premium.Get("/invoices", h.Premium.GetBillingInvoices)
		premium.Get("/payment-methods", h.Premium.GetPaymentMethods)

		// Server boost routes
		servers.Post("/:id/boost", h.Premium.BoostServer)
		servers.Delete("/:id/boost", h.Premium.UnboostServer)
		servers.Get("/:id/boosts", h.Premium.GetServerBoosts)
		servers.Get("/:id/perks", h.Premium.GetServerPerks)

		// Billing webhook (public, no auth required)
		v1.Post("/billing/webhook/:provider", h.Premium.HandleBillingWebhook)
	}

	// Server voice states
	servers.Get("/:id/voice-states", h.Voice.GetServerVoiceStates)

	// Gateway stats (admin)
	api.Get("/gateway/stats", h.Gateway.GetStats)

	// E2EE key management (if handler is configured)
	if h.E2EE != nil {
		keys := api.Group("/keys")
		keys.Post("/upload", h.E2EE.UploadKeys)
		keys.Get("/devices", h.E2EE.GetMyDevices)
		keys.Get("/devices/:deviceId/count", h.E2EE.GetPreKeyCount)
		keys.Post("/devices/:deviceId/prekeys", h.E2EE.UploadPreKeys)
		keys.Delete("/devices/:deviceId", h.E2EE.DeleteDevice)
		keys.Get("/:userId/devices", h.E2EE.GetUserDevices)
		keys.Get("/:userId/devices/:deviceId/bundle", h.E2EE.GetPreKeyBundle)
		keys.Get("/:userId/bundles", h.E2EE.GetAllPreKeyBundles)
		keys.Get("/:userId/capabilities", h.E2EE.GetCapabilities)
		keys.Post("/claim", h.E2EE.ClaimKeys)
	}

	// AI endpoints (if handler is configured)
	if h.AI != nil {
		aiGroup := api.Group("/ai")

		// Provider info (public to authenticated users)
		aiGroup.Get("/provider-types", h.AI.GetProviderTypes)
		aiGroup.Get("/feature-types", h.AI.GetFeatureTypes)
		aiGroup.Get("/models", h.AI.GetAvailableModels)
		aiGroup.Get("/health", h.AI.HealthCheck)

		// Provider management (admin routes)
		aiGroup.Get("/providers", h.AI.GetProviders)
		aiGroup.Get("/providers/:id", h.AI.GetProvider)
		aiGroup.Get("/providers/:id/models", h.AI.GetProviderModels)
		aiGroup.Post("/providers", h.AI.CreateProvider)
		aiGroup.Patch("/providers/:id", h.AI.UpdateProvider)
		aiGroup.Delete("/providers/:id", h.AI.DeleteProvider)

		// User AI settings (consolidated credentials + preferences)
		aiGroup.Get("/settings", h.AI.GetUserSettings)
		aiGroup.Put("/settings", h.AI.UpdateUserSettings)

		// User credentials (BYOK - legacy/direct access)
		aiGroup.Get("/credentials", h.AI.GetUserCredentials)
		aiGroup.Post("/credentials", h.AI.SetUserCredentials)
		aiGroup.Delete("/credentials/:providerId", h.AI.DeleteUserCredential)

		// Model routing
		aiGroup.Get("/routing", h.AI.GetModelRoutings)
		aiGroup.Post("/routing", h.AI.SetModelRouting)
		aiGroup.Delete("/routing/:id", h.AI.DeleteModelRouting)

		// Admin defaults (server-wide AI configuration)
		aiGroup.Get("/admin/defaults", h.AI.GetAdminDefaults)
		aiGroup.Post("/admin/defaults", h.AI.SetAdminDefaults)

		// Chat & generation
		aiGroup.Post("/chat", h.AI.ChatCompletion)
		aiGroup.Post("/embeddings", h.AI.GenerateEmbeddings)

		// AI Chat conversations (if handler is configured)
		if h.AIChat != nil {
			// Conversations
			aiGroup.Get("/conversations", h.AIChat.ListConversations)
			aiGroup.Post("/conversations", h.AIChat.CreateConversation)
			aiGroup.Get("/conversations/:id", h.AIChat.GetConversation)
			aiGroup.Patch("/conversations/:id", h.AIChat.UpdateConversation)
			aiGroup.Delete("/conversations/:id", h.AIChat.DeleteConversation)

			// Messages
			aiGroup.Get("/conversations/:id/messages", h.AIChat.GetMessages)
			aiGroup.Post("/conversations/:id/messages", h.AIChat.SendMessage)
			aiGroup.Post("/conversations/:id/messages/:messageId/regenerate", h.AIChat.RegenerateMessage)
			aiGroup.Delete("/conversations/:id/messages/:messageId", h.AIChat.DeleteMessage)

			// Templates
			aiGroup.Get("/templates", h.AIChat.ListTemplates)
			aiGroup.Post("/templates", h.AIChat.CreateTemplate)
			aiGroup.Get("/templates/:id", h.AIChat.GetTemplate)
			aiGroup.Patch("/templates/:id", h.AIChat.UpdateTemplate)
			aiGroup.Delete("/templates/:id", h.AIChat.DeleteTemplate)

			// Shares
			aiGroup.Post("/conversations/:id/share", h.AIChat.ShareConversation)
			aiGroup.Get("/shared/:code", h.AIChat.GetSharedConversation)
		}
	}

	// WebSocket gateway
	app.Get("/gateway", m.WebSocketUpgrade, websocket.New(h.Gateway.Connect))

	// Static files (for self-hosted frontend)
	app.Static("/", "./public")

	// SPA fallback
	app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})
}
