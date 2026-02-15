package websocket

import (
	"context"

	"github.com/google/uuid"
)

// HubInterface defines the interface for WebSocket hubs
// This allows both Hub and DistributedHub to be used interchangeably
type HubInterface interface {
	// Run starts the hub's event loop
	Run(ctx context.Context)

	// Client subscription management
	SubscribeChannel(client *Client, channelID uuid.UUID)
	UnsubscribeChannel(client *Client, channelID uuid.UUID)
	SubscribeServer(client *Client, serverID uuid.UUID)

	// Event broadcasting
	Broadcast(event *Event)
	SendToUser(userID uuid.UUID, event *Event)
	SendToChannel(channelID uuid.UUID, event *Event)
	SendToServer(serverID uuid.UUID, event *Event)

	// Queries
	GetOnlineUsers(userIDs []uuid.UUID) []uuid.UUID
	GetClientCount() int

	// Registration channels (for Gateway)
	RegisterClient() chan<- *Client
	UnregisterClient() chan<- *Client

	// Graceful shutdown
	Shutdown(ctx context.Context) error
	IsHealthy() bool
	IsDraining() bool
	DrainState() DrainState
}

// RegisterClient returns the registration channel
func (h *Hub) RegisterClient() chan<- *Client {
	return h.register
}

// UnregisterClient returns the unregistration channel
func (h *Hub) UnregisterClient() chan<- *Client {
	return h.unregister
}

// Ensure Hub implements HubInterface
var _ HubInterface = (*Hub)(nil)

// DistributedHubInterface extends HubInterface with distributed capabilities
type DistributedHubInterface interface {
	HubInterface

	// Distributed event broadcasting (publishes to Redis)
	BroadcastDistributed(ctx context.Context, event *Event) error
	SendToChannelDistributed(ctx context.Context, channelID uuid.UUID, eventType string, data interface{}) error
	SendToServerDistributed(ctx context.Context, serverID uuid.UUID, eventType string, data interface{}) error
	SendToUserDistributed(ctx context.Context, userID uuid.UUID, eventType string, data interface{}) error

	// User subscription for DMs
	SubscribeUser(userID uuid.UUID)
	UnsubscribeUser(userID uuid.UUID)

	// Statistics
	Stats() map[string]interface{}
}

// RegisterClient returns the registration channel for DistributedHub
func (dh *DistributedHub) RegisterClient() chan<- *Client {
	return dh.Hub.register
}

// UnregisterClient returns the unregistration channel for DistributedHub
func (dh *DistributedHub) UnregisterClient() chan<- *Client {
	return dh.Hub.unregister
}

// Ensure DistributedHub implements DistributedHubInterface
var _ DistributedHubInterface = (*DistributedHub)(nil)
