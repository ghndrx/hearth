package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	h.Search = &SearchHandler{}

	assert.NotNil(t, h.Auth)
	assert.NotNil(t, h.Users)
	assert.NotNil(t, h.Servers)
	assert.NotNil(t, h.Channels)
	assert.NotNil(t, h.Invites)
	assert.NotNil(t, h.Voice)
	assert.NotNil(t, h.Gateway)
	assert.NotNil(t, h.Search)
}
