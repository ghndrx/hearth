package api

import (
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"hearth/internal/api/handlers"
	"hearth/internal/api/middleware"
	"hearth/internal/config"
	"hearth/internal/matrix"
	"hearth/internal/matrixfederation"
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
	// Webhooks: 30 req/min per webhook
	webhookRateLimit := m.RateLimitWithConfig(middleware.RateLimitConfig{
		Limit:  30,
		Window: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "webhook:" + c.Params("webhookID")
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
		// Note: GET /:id is registered below in discoverServers group
	}

	// Enhanced Public Server Discovery (new endpoints)
	if h.Discovery != nil {
		discoverServers := v1.Group("/servers")
		discoverServers.Get("/discover", h.Discovery.GetDiscoverableServers)
		discoverServers.Get("/discover/featured", h.Discovery.GetFeaturedServersDS)
		discoverServers.Get("/discover/trending", h.Discovery.GetTrendingServersDS)

		discoverServers.Get("/discover/search", h.Discovery.SearchServersEnhancedDS)
		discoverServers.Get("/discover/home", h.Discovery.GetDiscoveryHomePage)
		discoverServers.Get("/discover/categories/stats", h.Discovery.GetCategoriesWithStats)
		discoverServers.Get("/discover/tags", h.Discovery.GetPopularTagsDS)
		discoverServers.Get("/discover/stats", h.Discovery.GetDiscoveryStatsDS)
		discoverServers.Get("/discover/suggestions", h.Discovery.GetSearchSuggestions)
		discoverServers.Get("/categories", h.Discovery.GetCategoriesDS)
		discoverServers.Get("/:id", h.Discovery.GetServerDetail)

		// Main public server directory endpoint - GET /api/v1/discovery
		// Lists servers that have opted into being discoverable with featured, trending, recommendations
		v1.Get("/discovery", h.Discovery.GetDiscovery)

		// Public server directory endpoint - GET /api/v1/directory
		// Public server directory with search, categories, and pagination
		v1.Get("/directory", h.Discovery.GetDirectory)

		// Discovery activity tracking (no auth required, user optional)
		v1.Post("/directory/:id/track", h.Discovery.TrackDiscoveryActivity)
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

	// Server Folders
	if h.ServerFolders != nil {
		serverFolders := users.Group("/@me/server-folders")
		serverFolders.Get("/", h.ServerFolders.GetAll)
		serverFolders.Post("/", h.ServerFolders.Create)
		serverFolders.Post("/move", h.ServerFolders.MoveServer)
		serverFolders.Post("/move-batch", h.ServerFolders.MoveServers)
		serverFolders.Post("/reorder", h.ServerFolders.ReorderServers)
		serverFolders.Get("/:id", h.ServerFolders.Get)
		serverFolders.Patch("/:id", h.ServerFolders.Update)
		serverFolders.Delete("/:id", h.ServerFolders.Delete)
	}

	// User Settings
	if h.Settings != nil {
		users.Get("/@me/settings", h.Settings.GetSettings)
		users.Patch("/@me/settings", h.Settings.UpdateSettings)
		users.Delete("/@me/settings", h.Settings.ResetSettings)
	}

	// Read State / Unread
	if h.Notifications != nil {
		users.Get("/@me/unread", h.Notifications.GetUnreadSummary)
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

	// Smart Notifications
	if h.Notifications != nil {
		smartNotifs := users.Group("/@me/notifications")
		smartNotifs.Post("/score", h.Notifications.ScoreNotification)
		smartNotifs.Post("/snooze", h.Notifications.SnoozeNotifications)
		smartNotifs.Get("/snooze", h.Notifications.GetSnoozeStatus)
		smartNotifs.Delete("/snooze", h.Notifications.UnsnoozeNotifications)
		smartNotifs.Post("/mute", h.Notifications.MuteNotifications)
		smartNotifs.Get("/mute", h.Notifications.GetMuteStatus)
		smartNotifs.Get("/engagement", h.Notifications.GetEngagement)
		smartNotifs.Get("/preferences", h.Notifications.GetSmartNotificationPreferences)
		smartNotifs.Patch("/preferences", h.Notifications.UpdateSmartNotificationPreferences)
		smartNotifs.Get("/digests", h.Notifications.ListDigests)
		smartNotifs.Get("/digests/:id", h.Notifications.GetDigest)
		smartNotifs.Post("/digests/:id/read", h.Notifications.MarkDigestRead)
		smartNotifs.Post("/:id/click", h.Notifications.TrackClick)
		smartNotifs.Post("/:id/dismiss", h.Notifications.DismissNotification)
	}

	// Push Notifications
	if h.Notifications != nil {
		push := api.Group("/push")
		push.Post("/subscription", h.Notifications.RegisterSubscription)
		push.Delete("/subscription", h.Notifications.UnregisterSubscription)
		push.Get("/preferences", h.Notifications.GetPushPreferences)
		push.Patch("/preferences", h.Notifications.UpdatePushPreferences)
	}

	// Digest Notifications
	if h.NotificationPreferences != nil {
		digest := users.Group("/@me/digest")
		digest.Get("/preferences", h.NotificationPreferences.GetPreferences)
		digest.Patch("/preferences", h.NotificationPreferences.UpdatePreferences)
		digest.Get("/channels", h.NotificationPreferences.GetChannelPreferences)
		digest.Get("/channels/:channelId", h.NotificationPreferences.GetChannelPreference)
		digest.Patch("/channels/:channelId", h.NotificationPreferences.UpdateChannelPreference)
		digest.Get("/servers", h.NotificationPreferences.GetServerPreferences)
		digest.Get("/servers/:serverId", h.NotificationPreferences.GetServerPreference)
		digest.Patch("/servers/:serverId", h.NotificationPreferences.UpdateServerPreference)
		digest.Get("/preview", h.NotificationPreferences.GetDigestPreview)
		digest.Delete("/queue", h.NotificationPreferences.ClearDigestQueue)
		digest.Get("/history", h.NotificationPreferences.GetDigestHistory)
		digest.Get("/history/:digestId", h.NotificationPreferences.GetDigest)
		digest.Post("/generate", h.NotificationPreferences.GenerateDigestNow)
	}

	// Channel Notification Preferences
	if h.NotificationPreferences != nil {
		channelNotifPrefs := users.Group("/@me/channels/:channelId/notifications")
		channelNotifPrefs.Get("/", h.NotificationPreferences.GetChannelNotificationPreference)
		channelNotifPrefs.Patch("/", h.NotificationPreferences.UpdateChannelNotificationPreference)
	}

	// Channel Notification Overrides (simplified level-based system)
	channelOverrides := users.Group("/@me/notification-overrides")
	channelOverrides.Get("/", h.Channels.ListChannelOverrides)
	channelOverrides.Get("/:channel_id", h.Channels.GetChannelOverride)
	channelOverrides.Put("/:channel_id", h.Channels.SetChannelOverride)
	channelOverrides.Delete("/:channel_id", h.Channels.ClearChannelOverride)

	// Server Notification Preferences
	if h.NotificationPreferences != nil {
		serverNotifPrefs := users.Group("/@me/servers/:serverId/notifications")
		serverNotifPrefs.Get("/", h.NotificationPreferences.GetServerNotificationPreference)
		serverNotifPrefs.Patch("/", h.NotificationPreferences.UpdateServerNotificationPreference)
	}

	// Mentions
	if h.Notifications != nil {
		mentions := api.Group("/mentions")
		mentions.Get("/", h.Notifications.GetMentions)
		mentions.Get("/unread/count", h.Notifications.GetUnreadCount)
		mentions.Get("/stats", h.Notifications.GetStats)
		mentions.Get("/search", h.Notifications.Search)
		mentions.Post("/read-all", h.Notifications.MarkAllMentionsAsRead)
		mentions.Patch("/:id/read", h.Notifications.MarkMentionAsRead)
		mentions.Post("/channel/:channelId/read-all", h.Notifications.MarkChannelMentionsAsRead)
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
		dms.Patch("/:channelId", h.DMs.UpdateGroupDM)
		dms.Put("/:channelId/participants", h.DMs.AddParticipant)
		dms.Delete("/:channelId/participants", h.DMs.RemoveParticipant)
		dms.Delete("/:channelId/leave", h.DMs.LeaveDM)
		dms.Patch("/:channelId/owner", h.DMs.TransferOwnership)

		// Convenience route: create DM with a specific user
		users.Post("/:id/dm", h.DMs.CreateDMWithUser)
	}

	// Servers
	servers := api.Group("/servers")
	servers.Get("/discover/recommended", h.Discovery.GetRecommendedServers)
	servers.Post("/", h.Servers.Create)
	servers.Get("/:id", h.Servers.Get)
	if h.Discovery != nil {
		servers.Get("/discover/recommended", h.Discovery.GetRecommendedServers)
	}
	if h.Discovery != nil {
		servers.Get("/discover/recommended", h.Discovery.GetRecommendedServers)
	}
	servers.Patch("/:id", h.Servers.Update)
	servers.Delete("/:id", h.Servers.Delete)
	servers.Post("/:id/transfer-ownership", h.Servers.TransferOwnership)
	servers.Post("/:id/join", h.Discovery.JoinServer)

	// Server discovery registration (auth-protected, server owner only)
	if h.Discovery != nil {
		servers.Post("/:serverId/discover", h.Discovery.RegisterServer)
		servers.Patch("/discover/:id", h.Discovery.UpdateRegisteredServer)
		servers.Delete("/discover/:id", h.Discovery.DeleteRegisteredServer)
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
	if h.Notifications != nil {
		servers.Get("/:id/unread", h.Notifications.GetServerUnread)
		servers.Post("/:id/ack", h.Notifications.MarkServerAsRead)
	}

	// Channels
	channels := api.Group("/channels")
	channels.Get("/:id", h.Channels.Get)
	channels.Patch("/:id", h.Channels.Update)
	channels.Delete("/:id", h.Channels.Delete)
	channels.Post("/:id/federate", h.Channels.Federate)

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
	if h.Notifications != nil {
		channels.Post("/:id/ack", h.Notifications.MarkChannelAsRead)
		channels.Get("/:id/unread", h.Notifications.GetChannelUnread)
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
	threads.Put("/:id/tags", h.Threads.ApplyTags)
	threads.Get("/:id/tags", h.Threads.GetThreadTags)
	threads.Put("/:id/pin", h.Threads.PinThread)
	threads.Put("/:id/solved", h.Threads.MarkSolved)

	// Forum channel tags
	channels.Get("/:id/tags", h.Threads.ListTags)
	channels.Post("/:id/tags", h.Threads.CreateTag)
	channels.Get("/:id/posts", h.Threads.ListPosts)
	channels.Get("/:id/forum-config", h.Threads.GetForumConfig)
	channels.Patch("/:id/forum-config", h.Threads.UpdateForumConfig)
	channels.Patch("/:id/tags/:tagId", h.Threads.UpdateTag)
	channels.Delete("/:id/tags/:tagId", h.Threads.DeleteTag)

	// Global tag management
	api.Patch("/forum-tags/:tagId", h.Threads.UpdateTag)
	api.Delete("/forum-tags/:tagId", h.Threads.DeleteTag)

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

	// Interactions - public endpoint (token-based and interaction_id-based)
	if h.Interactions != nil {
		v1.Post("/interactions", h.Interactions.HandleInteraction)
		// POST /interactions/{interaction_id}/callback - accepts both interaction_id and token
		v1.Post("/interactions/:interaction_id/callback", h.Interactions.RespondToInteraction)
		v1.Post("/interactions/:interaction_id/callback/:token", h.Interactions.RespondToInteractionWithToken)
		v1.Patch("/interactions/:interaction_id/messages/:messageId", h.Interactions.EditInteractionResponse)
		v1.Delete("/interactions/:interaction_id/messages/:messageId", h.Interactions.DeleteInteractionResponse)

		// Modal submit endpoint
		api.Post("/interactions/modals/submit", h.Interactions.HandleModalSubmit)
	}

	// Server channels
	servers.Get("/:id/channels", h.Servers.GetChannels)
	servers.Post("/:id/channels", h.Servers.CreateChannel)

	// Invites
	v1.Get("/invites/:code", h.Invites.Get)
	invites := api.Group("/invites")
	invites.Post("/:code", h.Invites.Accept)
	invites.Delete("/:code", h.Invites.Delete)
	invites.Get("/:code/analytics", h.Invites.GetAnalytics)

	// Channel webhooks
	if h.Webhooks != nil {
		channels.Post("/:id/webhooks", h.Webhooks.CreateWebhook)
		channels.Get("/:id/webhooks", h.Webhooks.GetChannelWebhooks)
	}

	// Channel invites
	channels.Post("/:id/invites", h.Channels.CreateInvite)

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
		v1.Get("/attachments/:id", h.Attachments.Get)
		v1.Get("/attachments/:id/download", h.Attachments.Download)
		attachments := api.Group("/attachments")
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

	// Soundboard (if handler is configured)
	if h.Soundboard != nil {
		// Global soundboard sounds
		api.Get("/soundboard/defaults", h.Soundboard.ListDefaultSounds)

		// Server-specific soundboard sounds and packs
		servers.Get("/:id/soundboard", h.Soundboard.ListServerSounds)
		servers.Post("/:id/soundboard", h.Soundboard.CreateSound)
		servers.Get("/:id/soundboard/sounds/:soundId", h.Soundboard.GetSound)
		servers.Patch("/:id/soundboard/sounds/:soundId", h.Soundboard.ModifySound)
		servers.Delete("/:id/soundboard/sounds/:soundId", h.Soundboard.DeleteSound)

		// Soundboard packs
		servers.Get("/:id/soundboard/packs", h.Soundboard.ListServerPacks)
		servers.Post("/:id/soundboard/packs", h.Soundboard.CreatePack)
		servers.Get("/:id/soundboard/packs/:packId", h.Soundboard.GetPack)
		servers.Patch("/:id/soundboard/packs/:packId", h.Soundboard.ModifyPack)
		servers.Delete("/:id/soundboard/packs/:packId", h.Soundboard.DeletePack)
		servers.Put("/:id/soundboard/packs/:packId/sounds/:soundId", h.Soundboard.AddSoundToPack)
		servers.Delete("/:id/soundboard/packs/:packId/sounds/:soundId", h.Soundboard.RemoveSoundFromPack)
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

	// Embeds (if handler is configured)
	if h.Embed != nil {
		// URL metadata fetching for link previews
		api.Get("/embeds/fetch", h.Embed.FetchURLMetadata)

		// Embed Templates
		api.Get("/embeds/templates", h.Embed.ListTemplates)
		api.Post("/embeds/templates", h.Embed.CreateTemplate)
		api.Get("/embeds/templates/:id", h.Embed.GetTemplate)
		api.Put("/embeds/templates/:id", h.Embed.UpdateTemplate)
		api.Delete("/embeds/templates/:id", h.Embed.DeleteTemplate)
	}

	// Scheduled Events (if handler is configured)
	if h.Events != nil {
		// Server events
		servers.Get("/:id/events", h.Events.ListServerEvents)
		servers.Post("/:id/events", h.Events.CreateEvent)
		servers.Get("/:id/events/ical", h.Events.ExportServerEventsICal)

		// Event operations
		api.Get("/events/:id", h.Events.GetEvent)
		api.Patch("/events/:id", h.Events.UpdateEvent)
		api.Delete("/events/:id", h.Events.DeleteEvent)
		api.Get("/events/:id/ical", h.Events.ExportEventICal)

		// Event RSVPs
		api.Post("/events/:id/rsvp", h.Events.RSVP)
		api.Delete("/events/:id/rsvp", h.Events.RemoveRSVP)
		api.Get("/events/:id/users", h.Events.ListEventUsers)

		// Event actions
		api.Post("/events/:id/start", h.Events.StartEvent)

		// User events (iCal export for user's RSVPed events)
		api.Get("/users/me/events/ical", h.Events.ExportUserEventsICal)
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

	// Smart Moderation (AI-powered moderation suite)
	if h.SmartModeration != nil {
		// Server moderation settings
		servers.Get("/:id/moderation/settings", h.SmartModeration.GetSettings)
		servers.Patch("/:id/moderation/settings", h.SmartModeration.UpdateSettings)

		// Keyword/regex rules
		servers.Get("/:id/moderation/rules", h.SmartModeration.ListKeywordRules)
		servers.Post("/:id/moderation/rules", h.SmartModeration.CreateKeywordRule)

		// Global keyword rule operations
		api.Patch("/moderation/rules/:id", h.SmartModeration.UpdateKeywordRule)
		api.Delete("/moderation/rules/:id", h.SmartModeration.DeleteKeywordRule)

		// Moderation logs
		servers.Get("/:id/moderation/logs", h.SmartModeration.ListModerationLogs)
		servers.Get("/:id/moderation/members/:memberId/history", h.SmartModeration.GetMemberModerationHistory)
		servers.Get("/:id/moderation/members/:memberId/summary", h.SmartModeration.GetUserViolationSummary)
		servers.Post("/:id/moderation/members/:memberId/reset", h.SmartModeration.ResetMemberViolations)

		// Moderation actions
		servers.Post("/:id/moderation/actions", h.SmartModeration.TakeModerationAction)

		// Dashboard
		servers.Get("/:id/moderation/stats", h.SmartModeration.GetDashboardStats)

		// Global operations
		api.Post("/moderation/analyze", h.SmartModeration.AnalyzeContent)
		api.Post("/moderation/logs/:id/resolve", h.SmartModeration.ResolveModerationLog)
	}

	// Content Safety (NSFW filters, age verification, user preferences)
	if h.ContentSafety != nil {
		// Server content filters
		servers.Get("/:id/content-filters", h.ContentSafety.ListContentFilters)
		servers.Post("/:id/content-filters", h.ContentSafety.CreateContentFilter)

		// Global content filter operations
		api.Get("/content-filters/:id", h.ContentSafety.GetContentFilter)
		api.Patch("/content-filters/:id", h.ContentSafety.UpdateContentFilter)
		api.Delete("/content-filters/:id", h.ContentSafety.DeleteContentFilter)

		// Age verification settings
		servers.Get("/:id/age-verification", h.ContentSafety.GetAgeVerification)
		servers.Put("/:id/age-verification", h.ContentSafety.CreateAgeVerification)
		servers.Patch("/:id/age-verification", h.ContentSafety.UpdateAgeVerification)
		servers.Delete("/:id/age-verification", h.ContentSafety.DeleteAgeVerification)

		// Server content safety settings (comprehensive)
		servers.Get("/:id/content-safety", h.ContentSafety.GetServerSafetySettings)

		// Content safety testing
		servers.Post("/:id/content-safety/test", h.ContentSafety.TestContentScan)

		// User content preferences
		users.Get("/@me/content-preferences", h.ContentSafety.GetUserContentPreferences)
		users.Put("/@me/content-preferences", h.ContentSafety.UpdateUserContentPreferences)
	}

	// Webhooks (authenticated CRUD)
	if h.Webhooks != nil {
		webhooks := api.Group("/webhooks")
		webhooks.Get("/:webhookID", h.Webhooks.GetWebhook)
		webhooks.Patch("/:webhookID", h.Webhooks.UpdateWebhook)
		webhooks.Delete("/:webhookID", h.Webhooks.DeleteWebhook)
		webhooks.Get("/:webhookID/stats", h.Webhooks.GetWebhookStats)
		webhooks.Get("/:webhookID/deliveries", h.Webhooks.GetWebhookDeliveries)
		webhooks.Post("/:webhookID/test", h.Webhooks.TestWebhook)

		// Server webhooks
		servers.Get("/:id/webhooks", h.Webhooks.GetServerWebhooks)
	}

	// Webhook execution (public, token-based auth - no RequireAuth)
	if h.Webhooks != nil {
		v1.Post("/webhooks/:webhookID/:token", webhookRateLimit, h.Webhooks.ExecuteWebhook)
	}

	// Search
	search := api.Group("/search")
	search.Get("/", h.Search.SearchAll)
	search.Get("/messages", h.Search.SearchMessages)
	search.Get("/users", h.Search.SearchUsers)
	search.Get("/channels", h.Search.SearchChannels)
	search.Get("/suggestions", h.Search.GetSuggestions)
	// Global cross-server search
	search.Get("/global/messages", h.Search.GlobalSearchMessages)

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

		// Admin discovery management (requires admin role)
		adminDiscovery := api.Group("/admin/discovery", m.RequireAdmin)
		adminDiscovery.Post("/:listingId/approve", h.Discovery.ApproveListing)
		adminDiscovery.Post("/:listingId/reject", h.Discovery.RejectListing)
		adminDiscovery.Post("/:listingId/featured", h.Discovery.SetFeatured)
	}

	// Admin directory management (approve/feature servers in the directory)
	if h.Discovery != nil {
		adminDirectory := api.Group("/admin/directory", m.RequireAdmin)
		adminDirectory.Post("/:id/approve", h.Discovery.AdminApproveServer)
		adminDirectory.Post("/:id/reject", h.Discovery.AdminRejectServer)
		adminDirectory.Post("/:id/feature", h.Discovery.AdminFeatureServer)
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
		adminApps := api.Group("/admin/apps", m.RequireAdmin)
		adminApps.Post("/:appId/approve", h.AppDirectory.ApproveApp)
		adminApps.Post("/:appId/reject", h.AppDirectory.RejectApp)
		adminApps.Post("/:appId/suspend", h.AppDirectory.SuspendApp)
	}

	// Voice (WebRTC signaling)
	v1.Get("/voice/regions", h.Voice.GetRegions)
	voice := api.Group("/voice")
	voice.Get("/channels/:channelId/states", h.Voice.GetChannelVoiceStates)

	// Voice (LiveKit)
	voice.Post("/token", h.Voice.GenerateToken)
	voice.Get("/participants/:channelId", h.Voice.GetParticipants)
	voice.Delete("/participants/:channelId/:userId", h.Voice.DisconnectParticipant)
	voice.Post("/participants/:channelId/:userId/mute", h.Voice.MuteParticipant)

	// Voice Activities (Poker, Chess, Watch Together)
	channels.Post("/:id/activities", h.Voice.StartActivity)
	channels.Get("/:id/activities", h.Voice.GetChannelActivity)

	activities := api.Group("/activities")
	activities.Get("/:activityId", h.Voice.GetActivity)
	activities.Post("/:activityId/join", h.Voice.JoinActivity)
	activities.Delete("/:activityId/participants/@me", h.Voice.LeaveActivity)
	activities.Delete("/:activityId", h.Voice.EndActivity)
	activities.Get("/:activityId/state", h.Voice.GetGameState)
	activities.Post("/:activityId/moves", h.Voice.GameMove)

	// Calls (video/audio)
	calls := api.Group("/calls")
	calls.Post("/", h.Voice.Create)
	calls.Get("/:id", h.Voice.Get)
	calls.Post("/:id/join", h.Voice.Join)
	calls.Post("/:id/leave", h.Voice.Leave)
	calls.Post("/:id/signal", h.Voice.Signal)

	// Screen Share / Streams
	// Channel streams
	channels.Post("/:id/streams", h.Voice.StartStream)
	channels.Get("/:id/streams", h.Voice.GetActiveStreamForChannel)

	// Stream operations
	streams := api.Group("/streams")
	streams.Get("/:streamId", h.Voice.GetStreamInfo)
	streams.Patch("/:streamId", h.Voice.UpdateStream)
	streams.Delete("/:streamId", h.Voice.EndStream)
	streams.Post("/:streamId/join", h.Voice.JoinStream)
	streams.Delete("/:streamId/leave", h.Voice.LeaveStream)

	// Live Streaming to Channels
	// Channel live streams
	channels.Post("/:id/stream/start", h.Voice.StartLiveStream)
	channels.Post("/:id/stream/stop", h.Voice.StopLiveStream)
	channels.Get("/:id/stream", h.Voice.GetActiveLiveStream)

	// Live stream operations
	api.Get("/streams/:streamId", h.Voice.GetLiveStream)
	api.Patch("/streams/:streamId", h.Voice.UpdateLiveStream)
	api.Post("/streams/:streamId/join", h.Voice.JoinLiveStream)
	api.Post("/streams/:streamId/leave", h.Voice.LeaveLiveStream)
	api.Get("/streams/:streamId/viewers", h.Voice.GetLiveStreamViewers)

	// Server audio settings (per-server audio device preferences)
	if h.ServerAudioSettings != nil {
		users.Get("/@me/audio-settings", h.ServerAudioSettings.GetAllSettings)
		servers.Get("/:id/audio-settings", h.ServerAudioSettings.GetSettings)
		servers.Patch("/:id/audio-settings", h.ServerAudioSettings.UpdateSettings)
		servers.Delete("/:id/audio-settings", h.ServerAudioSettings.DeleteSettings)
	}

	// Premium & Server Boosts (if handler is configured)
	if h.Premium != nil {
		// Plans endpoint is public (no auth required)
		v1.Get("/premium/plans", h.Premium.GetPlans)

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
		premium.Delete("/payment-methods/:id", h.Premium.DeletePaymentMethod)
		premium.Post("/gift", h.Premium.GiftSubscription)
		premium.Get("/billing-portal", h.Premium.GetBillingPortal)

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
		aiAdmin := aiGroup.Group("", m.RequireAdmin)
		aiAdmin.Get("/providers", h.AI.GetProviders)
		aiAdmin.Get("/providers/:id", h.AI.GetProvider)
		aiAdmin.Get("/providers/:id/models", h.AI.GetProviderModels)
		aiAdmin.Post("/providers", h.AI.CreateProvider)
		aiAdmin.Patch("/providers/:id", h.AI.UpdateProvider)
		aiAdmin.Delete("/providers/:id", h.AI.DeleteProvider)

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
		aiAdmin.Get("/admin/defaults", h.AI.GetAdminDefaults)
		aiAdmin.Post("/admin/defaults", h.AI.SetAdminDefaults)

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

// MatrixFederationDeps holds the dependencies needed for Matrix federation route setup.
type MatrixFederationDeps struct {
	App             *fiber.App
	Config          *config.Config
	UserService     matrixfederation.UserGetter
	SigningKeyStore *matrixfederation.KeyStore

	// Phase 3 (Hearth-03..Hearth-06) federation dependencies.
	// Optional: when nil, the federation event/state/join/backfill/transaction
	// endpoints will not be mounted.
	EventStore     matrixfederation.FederationEventStore
	StateStore     matrixfederation.StateStore
	AuthChecker    *matrixfederation.AuthChecker
	RoomAliasStore matrixfederation.RoomAliasStore
}

// SetupMatrixFederationRoutes configures Matrix protocol endpoints.
// These are mounted at the root and are not under /api/v1.
func SetupMatrixFederationRoutes(deps *MatrixFederationDeps) {
	if !deps.Config.FederationEnabled {
		return
	}

	app := deps.App
	cfg := deps.Config

	// Build homeserver config from environment
	hsCfg := &matrix.HomeserverConfig{
		ServerName:         cfg.FederationServerName,
		BaseURL:            cfg.PublicURL,
		FederationURL:      cfg.FederationURL,
		DefaultIdentityURL: cfg.FederationIdentityURL,
		Version:            "1.12.0",
		Name:               "Hearth",
	}

	if hsCfg.ServerName == "" {
		// Derive from public URL if not explicitly set
		hsCfg.ServerName = cfg.PublicURL
	}

	// Validate the homeserver config
	if err := hsCfg.Validate(); err != nil {
		log.Printf("⚠️  Matrix federation config invalid: %v", err)
		return
	}

	// Well-known endpoints
	wellKnownOpts := &matrixfederation.WellKnownOptions{
		IdentityServerURL: cfg.FederationIdentityURL,
	}
	wellKnownHandler := matrixfederation.NewWellKnownHandler(hsCfg, wellKnownOpts)
	matrixfederation.SetupWellKnownRoutes(app, wellKnownHandler)

	// Version endpoints
	versionsHandler := matrixfederation.NewVersionsHandler("Hearth", "1.0.0")
	matrixfederation.SetupVersionRoutes(app, versionsHandler)

	// Profile API (Phase 1 - identity layer)
	// Adapts Hearth user service to Matrix Profile API
	if deps.UserService != nil {
		profileAdapter := matrixfederation.NewUserServiceAdapter(deps.UserService, hsCfg)
		profileHandler := matrixfederation.NewProfileHandler(profileAdapter)
		matrixfederation.SetupProfileRoutes(app, profileHandler, "/_matrix/client/v3")
		log.Printf("✅ Matrix Profile API configured")
	}

	// Server signing keys
	if deps.SigningKeyStore != nil {
		keyServerHandler := matrixfederation.NewKeyServerHandler(deps.SigningKeyStore, 0)
		matrixfederation.SetupKeyServerRoutes(app, keyServerHandler)
		log.Printf("✅ Matrix Key Server API configured")
	}

	// Room directory (Phase 2)
	roomStore := deps.RoomAliasStore
	if roomStore == nil {
		roomStore = matrixfederation.NewInMemoryRoomAliasStore()
	}
	directoryHandler := matrixfederation.NewRoomDirectoryHandler(roomStore, hsCfg)
	matrixfederation.SetupDirectoryRoutes(app, directoryHandler, "/_matrix/client/v3", "/_matrix/federation/v1")

	// Public rooms
	publicRoomsHandler := matrixfederation.NewPublicRoomsHandler(roomStore, hsCfg)
	matrixfederation.SetupPublicRoomsRoutes(app, publicRoomsHandler, "/_matrix/client/v3")

	// Phase 3 (Hearth-03..Hearth-06) federation server-server endpoints.
	// These are mounted only when the necessary dependencies are wired in.
	if deps.EventStore != nil && deps.StateStore != nil && deps.SigningKeyStore != nil {
		// X-Matrix authentication middleware for incoming federation requests.
		fedMiddleware := matrixfederation.NewFederationMiddleware(deps.SigningKeyStore)
		app.Use(fedMiddleware.VerifyXMatrix())

		// Transaction handler: PUT /_matrix/federation/v1/send/{txnId}
		txProcessor := matrixfederation.NewTransactionProcessor(
			hsCfg.ServerName,
			deps.EventStore,
			deps.StateStore,
			deps.AuthChecker,
		)
		txHandler := matrixfederation.NewTransactionHandler(txProcessor)
		matrixfederation.SetupTransactionRoutes(app, txHandler)

		// Event query handlers
		eventHandler := matrixfederation.NewFederationEventHandler(
			hsCfg.ServerName, deps.EventStore, deps.StateStore,
		)
		matrixfederation.SetupFederationEventRoutes(app, eventHandler)

		// State query handlers
		stateHandler := matrixfederation.NewFederationStateHandler(
			hsCfg.ServerName, deps.StateStore, deps.EventStore,
		)
		matrixfederation.SetupFederationStateRoutes(app, stateHandler)

		// Join handshake (make_join / send_join)
		joinHandler := matrixfederation.NewJoinHandler(
			hsCfg.ServerName, deps.StateStore, deps.EventStore, roomStore, deps.AuthChecker,
		)
		matrixfederation.SetupJoinRoutes(app, joinHandler)

		// Backfill
		backfillHandler := matrixfederation.NewBackfillHandler(
			hsCfg.ServerName, deps.EventStore, deps.StateStore,
		)
		matrixfederation.SetupBackfillRoutes(app, backfillHandler)

		log.Printf("✅ Matrix federation Phase 3 (server-server API) configured")
	}

	log.Printf("✅ Matrix federation routes configured for %s", hsCfg.ServerName)
}
