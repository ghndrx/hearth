package handlers

import (
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth          *AuthHandler
	Users         *UserHandler
	Settings      *SettingsHandler
	SavedMessages *SavedMessagesHandler
	Notifications *NotificationHandler
	Servers       *ServerHandler
	Channels      *ChannelHandler
	Threads       *ThreadHandler
	Invites       *InviteHandler
	Voice         *VoiceHandler
	Gateway       *GatewayHandler
	Search        *SearchHandler
	Attachments   *AttachmentHandler
	Polls         *PollHandler
	AuditLog      *AuditLogHandler
	ReadState     *ReadStateHandler
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
	gateway *websocket.Gateway,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService, serverService, channelService),
		Servers:  NewServerHandler(serverService, channelService, roleService),
		Channels: NewChannelHandler(channelService, messageService),
		Threads:  NewThreadHandler(threadService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(),
		Gateway:  NewGatewayHandler(gateway),
		Search:   NewSearchHandler(searchService),
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
	gateway *websocket.Gateway,
) *Handlers {
	return &Handlers{
		Auth:        NewAuthHandler(authService),
		Users:       NewUserHandler(userService, serverService, channelService),
		Servers:     NewServerHandler(serverService, channelService, roleService),
		Channels:    NewChannelHandler(channelService, messageService),
		Threads:     NewThreadHandler(threadService),
		Invites:     NewInviteHandler(serverService),
		Voice:       NewVoiceHandler(),
		Gateway:     NewGatewayHandler(gateway),
		Search:      NewSearchHandler(searchService),
		Attachments: NewAttachmentHandler(attachmentService, channelService),
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
	gateway *websocket.Gateway,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService, serverService, channelService),
		Servers:  NewServerHandler(serverService, channelService, roleService),
		Channels: NewChannelHandlerWithTyping(channelService, messageService, typingService),
		Threads:  NewThreadHandler(threadService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(),
		Gateway:  NewGatewayHandler(gateway),
		Search:   NewSearchHandler(searchService),
	}
}
