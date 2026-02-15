package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// MessageType represents the type of pub/sub message
type MessageType string

const (
	TypeMessageCreate  MessageType = "MESSAGE_CREATE"
	TypeMessageUpdate  MessageType = "MESSAGE_UPDATE"
	TypeMessageDelete  MessageType = "MESSAGE_DELETE"
	TypeTypingStart    MessageType = "TYPING_START"
	TypePresenceUpdate MessageType = "PRESENCE_UPDATE"
	TypeReactionAdd    MessageType = "REACTION_ADD"
	TypeReactionRemove MessageType = "REACTION_REMOVE"
	TypeChannelUpdate  MessageType = "CHANNEL_UPDATE"
	TypeChannelDelete  MessageType = "CHANNEL_DELETE"
	TypeMemberJoin     MessageType = "MEMBER_JOIN"
	TypeMemberLeave    MessageType = "MEMBER_LEAVE"
	TypeServerUpdate   MessageType = "SERVER_UPDATE"
)

// BroadcastMessage represents a message sent via Redis Pub/Sub
type BroadcastMessage struct {
	Type       MessageType     `json:"type"`
	ChannelID  *uuid.UUID      `json:"channel_id,omitempty"`
	ServerID   *uuid.UUID      `json:"server_id,omitempty"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	Data       json.RawMessage `json:"data"`
	OriginNode string          `json:"origin_node"`
	Timestamp  time.Time       `json:"timestamp"`
}

// Handler is a function that handles incoming pub/sub messages
type Handler func(msg *BroadcastMessage)

// PubSub manages Redis pub/sub connections for real-time message fan-out
type PubSub struct {
	client     *redis.Client
	prefix     string
	nodeID     string
	
	// Subscription management
	subscriptions map[string]*redis.PubSub
	subMux        sync.RWMutex
	
	// Local handlers
	handlers []Handler
	handlerMux sync.RWMutex
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new PubSub manager
func New(redisURL string, nodeID string) (*PubSub, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	psCtx, psCancel := context.WithCancel(context.Background())

	ps := &PubSub{
		client:        client,
		prefix:        "hearth:pubsub:",
		nodeID:        nodeID,
		subscriptions: make(map[string]*redis.PubSub),
		handlers:      make([]Handler, 0),
		ctx:           psCtx,
		cancel:        psCancel,
	}

	return ps, nil
}

// OnMessage registers a handler for incoming pub/sub messages
func (p *PubSub) OnMessage(handler Handler) {
	p.handlerMux.Lock()
	defer p.handlerMux.Unlock()
	p.handlers = append(p.handlers, handler)
}

// Publish sends a message to a Redis channel
func (p *PubSub) Publish(ctx context.Context, msg *BroadcastMessage) error {
	msg.OriginNode = p.nodeID
	msg.Timestamp = time.Now()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	channel := p.resolveChannel(msg)
	return p.client.Publish(ctx, channel, data).Err()
}

// PublishToChannel publishes a message to a specific channel
func (p *PubSub) PublishToChannel(ctx context.Context, channelID uuid.UUID, msgType MessageType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := &BroadcastMessage{
		Type:      msgType,
		ChannelID: &channelID,
		Data:      jsonData,
	}
	return p.Publish(ctx, msg)
}

// PublishToServer publishes a message to all members of a server
func (p *PubSub) PublishToServer(ctx context.Context, serverID uuid.UUID, msgType MessageType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := &BroadcastMessage{
		Type:     msgType,
		ServerID: &serverID,
		Data:     jsonData,
	}
	return p.Publish(ctx, msg)
}

// PublishToUser publishes a message to a specific user
func (p *PubSub) PublishToUser(ctx context.Context, userID uuid.UUID, msgType MessageType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := &BroadcastMessage{
		Type:   msgType,
		UserID: &userID,
		Data:   jsonData,
	}
	return p.Publish(ctx, msg)
}

// SubscribeChannel subscribes to a channel's messages
func (p *PubSub) SubscribeChannel(channelID uuid.UUID) error {
	channel := p.prefix + "channel:" + channelID.String()
	return p.subscribe(channel)
}

// UnsubscribeChannel unsubscribes from a channel
func (p *PubSub) UnsubscribeChannel(channelID uuid.UUID) error {
	channel := p.prefix + "channel:" + channelID.String()
	return p.unsubscribe(channel)
}

// SubscribeServer subscribes to a server's events
func (p *PubSub) SubscribeServer(serverID uuid.UUID) error {
	channel := p.prefix + "server:" + serverID.String()
	return p.subscribe(channel)
}

// UnsubscribeServer unsubscribes from a server
func (p *PubSub) UnsubscribeServer(serverID uuid.UUID) error {
	channel := p.prefix + "server:" + serverID.String()
	return p.unsubscribe(channel)
}

// SubscribeUser subscribes to a user's direct messages
func (p *PubSub) SubscribeUser(userID uuid.UUID) error {
	channel := p.prefix + "user:" + userID.String()
	return p.subscribe(channel)
}

// UnsubscribeUser unsubscribes from a user's messages
func (p *PubSub) UnsubscribeUser(userID uuid.UUID) error {
	channel := p.prefix + "user:" + userID.String()
	return p.unsubscribe(channel)
}

// SubscribeGlobal subscribes to global broadcast events
func (p *PubSub) SubscribeGlobal() error {
	return p.subscribe(p.prefix + "global")
}

func (p *PubSub) subscribe(channel string) error {
	p.subMux.Lock()
	defer p.subMux.Unlock()

	if _, exists := p.subscriptions[channel]; exists {
		return nil // Already subscribed
	}

	sub := p.client.Subscribe(p.ctx, channel)
	
	// Wait for subscription confirmation
	_, err := sub.Receive(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", channel, err)
	}

	p.subscriptions[channel] = sub

	// Start listening in a goroutine
	p.wg.Add(1)
	go p.listen(channel, sub)

	return nil
}

func (p *PubSub) unsubscribe(channel string) error {
	p.subMux.Lock()
	defer p.subMux.Unlock()

	sub, exists := p.subscriptions[channel]
	if !exists {
		return nil
	}

	delete(p.subscriptions, channel)
	return sub.Close()
}

func (p *PubSub) listen(channel string, sub *redis.PubSub) {
	defer p.wg.Done()

	ch := sub.Channel()
	for {
		select {
		case <-p.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			p.handleMessage(msg)
		}
	}
}

func (p *PubSub) handleMessage(redisMsg *redis.Message) {
	var msg BroadcastMessage
	if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
		log.Printf("Failed to unmarshal pub/sub message: %v", err)
		return
	}

	// Skip messages from ourselves
	if msg.OriginNode == p.nodeID {
		return
	}

	// Dispatch to all handlers
	p.handlerMux.RLock()
	handlers := make([]Handler, len(p.handlers))
	copy(handlers, p.handlers)
	p.handlerMux.RUnlock()

	for _, handler := range handlers {
		handler(&msg)
	}
}

func (p *PubSub) resolveChannel(msg *BroadcastMessage) string {
	if msg.ChannelID != nil {
		return p.prefix + "channel:" + msg.ChannelID.String()
	}
	if msg.ServerID != nil {
		return p.prefix + "server:" + msg.ServerID.String()
	}
	if msg.UserID != nil {
		return p.prefix + "user:" + msg.UserID.String()
	}
	return p.prefix + "global"
}

// Close gracefully shuts down the pub/sub manager
func (p *PubSub) Close() error {
	p.cancel()

	p.subMux.Lock()
	for _, sub := range p.subscriptions {
		sub.Close()
	}
	p.subscriptions = make(map[string]*redis.PubSub)
	p.subMux.Unlock()

	// Wait for all listeners to stop
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Println("PubSub shutdown timed out")
	}

	return p.client.Close()
}

// Stats returns current subscription statistics
func (p *PubSub) Stats() map[string]interface{} {
	p.subMux.RLock()
	defer p.subMux.RUnlock()

	channels := make([]string, 0, len(p.subscriptions))
	for ch := range p.subscriptions {
		channels = append(channels, ch)
	}

	return map[string]interface{}{
		"node_id":            p.nodeID,
		"subscription_count": len(p.subscriptions),
		"channels":           channels,
	}
}
