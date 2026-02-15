package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"

	"hearth/internal/pubsub"
)

// DistributedHub wraps Hub with Redis pub/sub for cross-instance communication
type DistributedHub struct {
	*Hub
	pubsub *pubsub.PubSub

	// Track which channels/servers are locally subscribed to
	// so we can manage Redis subscriptions
	localChannelSubs map[uuid.UUID]int // channel -> subscriber count
	localServerSubs  map[uuid.UUID]int
	localUserSubs    map[uuid.UUID]int
	localSubsMux     sync.RWMutex
}

// NewDistributedHub creates a hub with Redis pub/sub support
func NewDistributedHub(ps *pubsub.PubSub) *DistributedHub {
	dh := &DistributedHub{
		Hub:              NewHub(),
		pubsub:           ps,
		localChannelSubs: make(map[uuid.UUID]int),
		localServerSubs:  make(map[uuid.UUID]int),
		localUserSubs:    make(map[uuid.UUID]int),
	}

	// Register handler for incoming pub/sub messages
	ps.OnMessage(dh.handlePubSubMessage)

	return dh
}

// NewDistributedHubWithDrainConfig creates a hub with Redis pub/sub and custom drain config
func NewDistributedHubWithDrainConfig(ps *pubsub.PubSub, drainConfig *DrainConfig) *DistributedHub {
	dh := &DistributedHub{
		Hub:              NewHubWithDrainConfig(drainConfig),
		pubsub:           ps,
		localChannelSubs: make(map[uuid.UUID]int),
		localServerSubs:  make(map[uuid.UUID]int),
		localUserSubs:    make(map[uuid.UUID]int),
	}

	// Register handler for incoming pub/sub messages
	ps.OnMessage(dh.handlePubSubMessage)

	return dh
}

// Run starts the hub's event loop with Redis pub/sub integration
func (dh *DistributedHub) Run(ctx context.Context) {
	// Subscribe to global events
	if err := dh.pubsub.SubscribeGlobal(); err != nil {
		log.Printf("Failed to subscribe to global pub/sub: %v", err)
	}

	// Run the base hub
	dh.Hub.Run(ctx)
}

// handlePubSubMessage processes messages from other instances
func (dh *DistributedHub) handlePubSubMessage(msg *pubsub.BroadcastMessage) {
	// Convert pub/sub message to WebSocket event
	event := &Event{
		Op:   0,
		Type: string(msg.Type),
		Data: msg.Data,
	}

	// Route based on target
	if msg.ChannelID != nil {
		event.ChannelID = msg.ChannelID
	}
	if msg.ServerID != nil {
		event.ServerID = msg.ServerID
	}
	if msg.UserID != nil {
		event.UserID = msg.UserID
	}

	// Use base hub's local broadcast (don't re-publish to Redis)
	dh.Hub.handleBroadcast(event)
}

// SubscribeChannel subscribes a client to a channel with Redis tracking
func (dh *DistributedHub) SubscribeChannel(client *Client, channelID uuid.UUID) {
	dh.Hub.SubscribeChannel(client, channelID)

	// Track local subscription and subscribe to Redis if first
	dh.localSubsMux.Lock()
	dh.localChannelSubs[channelID]++
	if dh.localChannelSubs[channelID] == 1 {
		if err := dh.pubsub.SubscribeChannel(channelID); err != nil {
			log.Printf("Failed to subscribe to Redis channel %s: %v", channelID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// UnsubscribeChannel unsubscribes a client from a channel with Redis tracking
func (dh *DistributedHub) UnsubscribeChannel(client *Client, channelID uuid.UUID) {
	dh.Hub.UnsubscribeChannel(client, channelID)

	// Decrement local subscription count and unsubscribe from Redis if last
	dh.localSubsMux.Lock()
	dh.localChannelSubs[channelID]--
	if dh.localChannelSubs[channelID] <= 0 {
		delete(dh.localChannelSubs, channelID)
		if err := dh.pubsub.UnsubscribeChannel(channelID); err != nil {
			log.Printf("Failed to unsubscribe from Redis channel %s: %v", channelID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// SubscribeServer subscribes a client to a server with Redis tracking
func (dh *DistributedHub) SubscribeServer(client *Client, serverID uuid.UUID) {
	dh.Hub.SubscribeServer(client, serverID)

	dh.localSubsMux.Lock()
	dh.localServerSubs[serverID]++
	if dh.localServerSubs[serverID] == 1 {
		if err := dh.pubsub.SubscribeServer(serverID); err != nil {
			log.Printf("Failed to subscribe to Redis server %s: %v", serverID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// UnsubscribeServer unsubscribes a client from a server with Redis tracking
func (dh *DistributedHub) UnsubscribeServer(client *Client, serverID uuid.UUID) {
	// Note: Base hub doesn't have UnsubscribeServer, so just manage Redis
	dh.localSubsMux.Lock()
	dh.localServerSubs[serverID]--
	if dh.localServerSubs[serverID] <= 0 {
		delete(dh.localServerSubs, serverID)
		if err := dh.pubsub.UnsubscribeServer(serverID); err != nil {
			log.Printf("Failed to unsubscribe from Redis server %s: %v", serverID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// SubscribeUser subscribes to a user's events (for DMs, presence)
func (dh *DistributedHub) SubscribeUser(userID uuid.UUID) {
	dh.localSubsMux.Lock()
	dh.localUserSubs[userID]++
	if dh.localUserSubs[userID] == 1 {
		if err := dh.pubsub.SubscribeUser(userID); err != nil {
			log.Printf("Failed to subscribe to Redis user %s: %v", userID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// UnsubscribeUser unsubscribes from a user's events
func (dh *DistributedHub) UnsubscribeUser(userID uuid.UUID) {
	dh.localSubsMux.Lock()
	dh.localUserSubs[userID]--
	if dh.localUserSubs[userID] <= 0 {
		delete(dh.localUserSubs, userID)
		if err := dh.pubsub.UnsubscribeUser(userID); err != nil {
			log.Printf("Failed to unsubscribe from Redis user %s: %v", userID, err)
		}
	}
	dh.localSubsMux.Unlock()
}

// BroadcastDistributed sends an event to local clients AND publishes to Redis
func (dh *DistributedHub) BroadcastDistributed(ctx context.Context, event *Event) error {
	// Send to local clients immediately
	dh.Hub.Broadcast(event)

	// Publish to Redis for other instances
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	msg := &pubsub.BroadcastMessage{
		Type: pubsub.MessageType(event.Type),
		Data: data,
	}

	if event.ChannelID != nil {
		msg.ChannelID = event.ChannelID
	}
	if event.ServerID != nil {
		msg.ServerID = event.ServerID
	}
	if event.UserID != nil {
		msg.UserID = event.UserID
	}

	return dh.pubsub.Publish(ctx, msg)
}

// SendToChannelDistributed sends to channel locally and via Redis
func (dh *DistributedHub) SendToChannelDistributed(ctx context.Context, channelID uuid.UUID, eventType string, data interface{}) error {
	event := &Event{
		Op:        0,
		Type:      eventType,
		Data:      data,
		ChannelID: &channelID,
	}
	return dh.BroadcastDistributed(ctx, event)
}

// SendToServerDistributed sends to server locally and via Redis
func (dh *DistributedHub) SendToServerDistributed(ctx context.Context, serverID uuid.UUID, eventType string, data interface{}) error {
	event := &Event{
		Op:       0,
		Type:     eventType,
		Data:     data,
		ServerID: &serverID,
	}
	return dh.BroadcastDistributed(ctx, event)
}

// SendToUserDistributed sends to user locally and via Redis
func (dh *DistributedHub) SendToUserDistributed(ctx context.Context, userID uuid.UUID, eventType string, data interface{}) error {
	event := &Event{
		Op:     0,
		Type:   eventType,
		Data:   data,
		UserID: &userID,
	}
	return dh.BroadcastDistributed(ctx, event)
}

// Stats returns hub statistics including pub/sub info
func (dh *DistributedHub) Stats() map[string]interface{} {
	dh.localSubsMux.RLock()
	defer dh.localSubsMux.RUnlock()

	dh.clientsMux.RLock()
	clientCount := 0
	for _, clients := range dh.clients {
		clientCount += len(clients)
	}
	userCount := len(dh.clients)
	dh.clientsMux.RUnlock()

	dh.channelsMux.RLock()
	channelSubCount := len(dh.channels)
	dh.channelsMux.RUnlock()

	dh.serversMux.RLock()
	serverSubCount := len(dh.servers)
	dh.serversMux.RUnlock()

	return map[string]interface{}{
		"clients":              clientCount,
		"users":                userCount,
		"local_channel_subs":   channelSubCount,
		"local_server_subs":    serverSubCount,
		"redis_channel_subs":   len(dh.localChannelSubs),
		"redis_server_subs":    len(dh.localServerSubs),
		"redis_user_subs":      len(dh.localUserSubs),
		"pubsub_stats":         dh.pubsub.Stats(),
	}
}
