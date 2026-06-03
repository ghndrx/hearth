package handlers

import (
	"hearth/internal/ai"
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth                         *AuthHandler
	Sessions                     *SessionHandler
	Users                        *UserHandler
	Settings                     *SettingsHandler
	SavedMessages                *SavedMessagesHandler
	Notifications                *NotificationHandler
	Servers                      *ServerHandler
	Channels                     *ChannelHandler
	DMs                          *DMHandler
	Threads                      *ThreadHandler
	Invites                      *InviteHandler
	Voice                        *VoiceHandler
	Gateway                      *GatewayHandler
	Search                       *SearchHandler
	Attachments                  *AttachmentHandler
	Polls                        *PollHandler
	AuditLog                     *AuditLogHandler
	AI                           *AIHandler
	AIChat                       *AIChatHandler
	Webhooks                     *WebhookHandlers
	E2EE                         *E2EEHandler
	Stickers                     *StickerHandler
	Announcements                *AnnouncementHandler
	Components                   *ComponentHandler
	Events                       *EventHandler
	AutoMod                      *AutoModHandler
	Discovery                    *DiscoveryHandler
	Templates                    *TemplateHandler
	SlashCommands                *SlashCommandHandler
	Interactions                 *InteractionHandler
	ServerAudioSettings          *ServerAudioSettingsHandler
	AppDirectory                 *AppDirectoryHandler
	Welcome                      *WelcomeHandler
	Soundboard                   *SoundboardHandler
	Premium                      *PremiumHandler
	NotificationPreferences  *NotificationPreferenceHandler
	ContentSafety            *ContentSafetyHandler
	ServerFolders                *ServerFolderHandler
	SmartModeration              *SmartModerationHandler
	Embed                        *EmbedHandler

	// Matrix Federation (optional)
	MatrixProfile   interface{} // *matrixfederation.ProfileHandler
	MatrixWellKnown interface{} // *matrixfederation.WellKnownHandler
	MatrixDirectory interface{} // *matrixfederation.RoomDirectoryHandler
	MatrixKeyServer interface{} // *matrixfederation.KeyServerHandler
	MatrixVersions  interface{} // *matrixfederation.VersionsHandler
}

// SetE2EEHandler sets the E2EE handler (optional, not all deployments need E2EE)
func (h *Handlers) SetE2EEHandler(e2eeService *services.E2EEServiceImpl) {
	h.E2EE = NewE2EEHandler(e2eeService)
}

// SetAIHandler sets the AI handler (optional, not all deployments need AI)
func (h *Handlers) SetAIHandler(aiService *ai.AIService) {
	h.AI = NewAIHandler(aiService)
}

// SetAIChatHandler sets the AI Chat handler (optional, requires AI service)
func (h *Handlers) SetAIChatHandler(chatService *ai.ChatService) {
	h.AIChat = NewAIChatHandler(chatService)
}

// SetWebhookHandler sets the webhook handler
func (h *Handlers) SetWebhookHandler(webhookService *services.WebhookService) {
	h.Webhooks = NewWebhookHandlers(webhookService)
}

// SetPollHandler sets the poll handler
func (h *Handlers) SetPollHandler(pollService *services.PollService) {
	h.Polls = NewPollHandler(pollService)
}

// SetStickerHandler sets the sticker handler
func (h *Handlers) SetStickerHandler(stickerService *services.StickerService, serverService *services.ServerService, permService *services.PermissionService, premiumService *services.PremiumService) {
	h.Stickers = NewStickerHandler(stickerService, serverService, permService, premiumService)
}

// SetAnnouncementHandler sets the announcement handler
func (h *Handlers) SetAnnouncementHandler(announcementService *services.AnnouncementService) {
	h.Announcements = NewAnnouncementHandler(announcementService)
}

// SetComponentHandler sets the component handler
func (h *Handlers) SetComponentHandler(componentService *services.ComponentService) {
	h.Components = NewComponentHandler(componentService, nil, nil, nil)
}

// SetComponentHandlerWithDeps sets the component handler with dependencies
func (h *Handlers) SetComponentHandlerWithDeps(
	componentService *services.ComponentService,
	messageService *services.MessageService,
	channelService *services.ChannelService,
	permissionService *services.PermissionService,
) {
	h.Components = NewComponentHandler(componentService, messageService, channelService, permissionService)
}

// SetEventHandler sets the event handler
func (h *Handlers) SetEventHandler(
	eventService *services.EventService,
	serverService *services.ServerService,
	permService *services.PermissionService,
) {
	h.Events = NewEventHandler(eventService, serverService, permService)
}

// SetAutoModHandler sets the auto-mod handler
func (h *Handlers) SetAutoModHandler(
	automodService *services.AutoModService,
	serverService *services.ServerService,
) {
	h.AutoMod = NewAutoModHandler(automodService, serverService)
}

// SetTemplateHandler sets the template handler
func (h *Handlers) SetTemplateHandler(
	templateService *services.TemplateService,
	serverService *services.ServerService,
) {
	h.Templates = NewTemplateHandler(templateService, serverService)
}

// SetDiscoveryHandler sets the discovery handler
func (h *Handlers) SetDiscoveryHandler(
	discoveryService *services.DiscoveryService,
	discoverableServerService *services.DiscoverableServerService,
	serverService *services.ServerService,
) {
	h.Discovery = NewDiscoveryHandler(discoveryService, discoverableServerService, serverService)
}

// SetServerAudioSettingsHandler sets the server audio settings handler
func (h *Handlers) SetServerAudioSettingsHandler(service ServerAudioSettingsServiceInterface) {
	h.ServerAudioSettings = NewServerAudioSettingsHandler(service)
}

// SetWelcomeHandler sets the welcome/onboarding handler
func (h *Handlers) SetWelcomeHandler(
	welcomeService *services.WelcomeService,
	userService *services.UserService,
) {
	h.Welcome = NewWelcomeHandler(welcomeService, userService)
}

// NewHandlers creates all handlers with dependencies
func NewHandlers(
	authService services.AuthService,
	userService *services.UserService,
	serverService *services.ServerService,
	channelService *services.ChannelService,
	messageService *services.MessageService,
	roleService *services.RoleService,
	searchService *services.SearchService,
	threadService *services.ThreadService,
	webhookService *services.WebhookService,
	gateway *websocket.Gateway,
	voiceService *websocket.VoiceSignalingService,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService, serverService, channelService),
		Servers:  NewServerHandler(serverService, channelService, roleService),
		Channels: NewChannelHandler(channelService, messageService),
		Threads:  NewThreadHandler(threadService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(voiceService),
		Gateway:  NewGatewayHandler(gateway),
		Search:   NewSearchHandler(searchService),
		Webhooks: NewWebhookHandlers(webhookService),
	}
}

// NewHandlersWithAttachments creates all handlers including attachments
func NewHandlersWithAttachments(
	authService services.AuthService,
	userService *services.UserService,
	serverService *services.ServerService,
	channelService *services.ChannelService,
	messageService *services.MessageService,
	roleService *services.RoleService,
	searchService *services.SearchService,
	threadService *services.ThreadService,
	attachmentService *services.AttachmentService,
	webhookService *services.WebhookService,
	gateway *websocket.Gateway,
	voiceService *websocket.VoiceSignalingService,
) *Handlers {
	return &Handlers{
		Auth:        NewAuthHandler(authService),
		Users:       NewUserHandler(userService, serverService, channelService),
		Servers:     NewServerHandler(serverService, channelService, roleService),
		Channels:    NewChannelHandler(channelService, messageService),
		Threads:     NewThreadHandler(threadService),
		Invites:     NewInviteHandler(serverService),
		Voice:       NewVoiceHandler(voiceService),
		Gateway:     NewGatewayHandler(gateway),
		Search:      NewSearchHandler(searchService),
		Attachments: NewAttachmentHandler(attachmentService, channelService),
		Webhooks:    NewWebhookHandlers(webhookService),
	}
}

// NewHandlersWithTyping creates all handlers including typing service
func NewHandlersWithTyping(
	authService services.AuthService,
	userService *services.UserService,
	serverService *services.ServerService,
	channelService *services.ChannelService,
	messageService *services.MessageService,
	roleService *services.RoleService,
	searchService *services.SearchService,
	threadService *services.ThreadService,
	typingService *services.TypingService,
	webhookService *services.WebhookService,
	gateway *websocket.Gateway,
	voiceService *websocket.VoiceSignalingService,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService, serverService, channelService),
		Servers:  NewServerHandler(serverService, channelService, roleService),
		Channels: NewChannelHandlerWithTyping(channelService, messageService, typingService),
		Threads:  NewThreadHandler(threadService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(voiceService),
		Gateway:  NewGatewayHandler(gateway),
		Search:   NewSearchHandler(searchService),
		Webhooks: NewWebhookHandlers(webhookService),
	}
}

// SetDMHandler sets the DM handler
func (h *Handlers) SetDMHandler(
	dmService *services.DMService,
	channelService ChannelServiceForUsersInterface,
	userService UserServiceInterface,
	messageService MessageServiceInterface,
) {
	h.DMs = NewDMHandler(dmService, channelService, userService, messageService)
}

// SetSlashCommandHandler sets the slash command handler
func (h *Handlers) SetSlashCommandHandler(
	slashCmdService *services.SlashCommandService,
	permService *services.PermissionService,
) {
	h.SlashCommands = NewSlashCommandHandler(slashCmdService, permService)
}

// SetInteractionHandler sets the interaction handler
func (h *Handlers) SetInteractionHandler(
	interactionService *services.InteractionService,
) {
	h.Interactions = NewInteractionHandler(interactionService)
}

// SetPremiumHandler sets the premium handler
func (h *Handlers) SetPremiumHandler(
	premiumService *services.PremiumService,
	billingService *services.BillingService,
) {
	h.Premium = NewPremiumHandler(premiumService, billingService)
}

// SetSoundboardHandler sets the soundboard handler
func (h *Handlers) SetSoundboardHandler(
	soundboardService *services.SoundboardService,
	serverService *services.ServerService,
	permService *services.PermissionService,
) {
	h.Soundboard = NewSoundboardHandler(soundboardService, serverService, permService)
}

// SetNotificationPreferenceHandler sets notification preference handlers
func (h *Handlers) SetNotificationPreferenceHandler(
	coordinator *services.NotificationCoordinator,
	digestService *services.DigestService,
) {
	h.NotificationPreferences = NewNotificationPreferenceHandler(coordinator, digestService)
}

// SetContentSafetyHandler sets the content safety handler
func (h *Handlers) SetContentSafetyHandler(
	contentSafetyService *services.ContentSafetyService,
	serverService *services.ServerService,
) {
	h.ContentSafety = NewContentSafetyHandler(contentSafetyService, serverService)
}

// SetServerFolderHandler sets the server folder handler
func (h *Handlers) SetServerFolderHandler(
	serverFolderService *services.ServerFolderService,
) {
	h.ServerFolders = NewServerFolderHandler(serverFolderService)
}

// SetSmartModerationHandler sets the smart moderation handler
func (h *Handlers) SetSmartModerationHandler(
	smartModService SmartModerationServiceInterface,
	serverService SmartModerationServerService,
) {
	h.SmartModeration = NewSmartModerationHandler(smartModService, serverService)
}

// SetEmbedHandler sets the embed handler
func (h *Handlers) SetEmbedHandler(embedService *services.EmbedService) {
	h.Embed = NewEmbedHandler(embedService)
}
