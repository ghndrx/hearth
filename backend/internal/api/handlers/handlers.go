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
}

// NewHandlers creates all handlers with dependencies
func NewHandlers(
	authService *services.AuthService,
	userService *services.UserService,
	serverService *services.ServerService,
	messageService *services.MessageService,
	gateway *websocket.Gateway,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(authService),
		Users:    NewUserHandler(userService),
		Servers:  NewServerHandler(serverService),
		Channels: NewChannelHandler(messageService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(),
		Gateway:  NewGatewayHandler(gateway),
	}
}
