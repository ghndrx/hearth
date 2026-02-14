package handlers

import (
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Auth     *AuthHandler
	Users    *UserHandler
	Servers  *ServerHandler
	Channels *ChannelHandler
	Invites  *InviteHandler
	Voice    *VoiceHandler
	Gateway  *GatewayHandler
	Search   *SearchHandler
}

// NewHandlers creates all handlers with dependencies
func NewHandlers(
	authService *services.AuthService,
	userService *services.UserService,
	serverService *services.ServerService,
	channelService *services.ChannelService,
	messageService *services.MessageService,
	roleService *services.RoleService,
	searchService *services.SearchService,
	gateway *websocket.Gateway,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService, serverService, channelService),
		Servers:  NewServerHandler(serverService, channelService, roleService),
		Channels: NewChannelHandler(channelService, messageService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(),
		Gateway:  NewGatewayHandler(gateway),
		Search:   NewSearchHandler(searchService),
	}
}
