package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/services"
)

func TestNewHandlers(t *testing.T) {
	// Create mock services for testing
	authService := &services.AuthService{}
	userService := &services.UserService{}
	serverService := &services.ServerService{}
	channelService := &services.ChannelService{}
	messageService := &services.MessageService{}
	roleService := &services.RoleService{}

	handlers := NewHandlers(
		authService,
		userService,
		serverService,
		channelService,
		messageService,
		roleService,
		nil, // gateway can be nil for this test
	)

	require.NotNil(t, handlers)
	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.Users)
	assert.NotNil(t, handlers.Servers)
	assert.NotNil(t, handlers.Channels)
	assert.NotNil(t, handlers.Invites)
	assert.NotNil(t, handlers.Voice)
	assert.NotNil(t, handlers.Gateway)
}

func TestHandlersStruct(t *testing.T) {
	h := &Handlers{}

	// Test that all fields can be assigned
	h.Auth = &AuthHandler{}
	h.Users = &UserHandler{}
	h.Servers = &ServerHandler{}
	h.Channels = &ChannelHandler{}
	h.Invites = &InviteHandler{}
	h.Voice = &VoiceHandler{}
	h.Gateway = &GatewayHandler{}

	assert.NotNil(t, h.Auth)
	assert.NotNil(t, h.Users)
	assert.NotNil(t, h.Servers)
	assert.NotNil(t, h.Channels)
	assert.NotNil(t, h.Invites)
	assert.NotNil(t, h.Voice)
	assert.NotNil(t, h.Gateway)
}
