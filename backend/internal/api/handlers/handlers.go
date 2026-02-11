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
	userService *services.UserService,
	serverService *services.ServerService,
	messageService *services.MessageService,
	hub *websocket.Hub,
) *Handlers {
	return &Handlers{
		Auth:     NewAuthHandler(userService),
		Users:    NewUserHandler(userService),
		Servers:  NewServerHandler(serverService),
		Channels: NewChannelHandler(messageService),
		Invites:  NewInviteHandler(serverService),
		Voice:    NewVoiceHandler(),
		Gateway:  NewGatewayHandler(hub, userService),
	}
}
