package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

// Hub manages all WebSocket connections
type Hub struct {
	// Registered clients by user ID
	clients    map[uuid.UUID]map[*Client]bool
	clientsMux sync.RWMutex
	
	// Channel subscriptions
	channels    map[uuid.UUID]map[*Client]bool
	channelsMux sync.RWMutex
	
	// Server subscriptions (for presence, typing, etc.)
	servers    map[uuid.UUID]map[*Client]bool
	serversMux sync.RWMutex
	
	// Incoming events
	broadcast  chan *Event
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		channels:   make(map[uuid.UUID]map[*Client]bool),
		servers:    make(map[uuid.UUID]map[*Client]bool),
		broadcast:  make(chan *Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
			
		case client := <-h.register:
			h.registerClient(client)
			
		case client := <-h.unregister:
			h.unregisterClient(client)
			
		case event := <-h.broadcast:
			h.handleBroadcast(event)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

func (h *Hub) unregisterClient(client *Client) {
	h.clientsMux.Lock()
	if clients, ok := h.clients[client.UserID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
	h.clientsMux.Unlock()
	
	// Unsubscribe from all channels
	h.channelsMux.Lock()
	for channelID, clients := range h.channels {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.channels, channelID)
			}
		}
	}
	h.channelsMux.Unlock()
	
	// Unsubscribe from all servers
	h.serversMux.Lock()
	for serverID, clients := range h.servers {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.servers, serverID)
			}
		}
	}
	h.serversMux.Unlock()
	
	close(client.send)
}

func (h *Hub) handleBroadcast(event *Event) {
	switch {
	case event.ChannelID != nil:
		// Send to all clients subscribed to channel
		h.channelsMux.RLock()
		clients := h.channels[*event.ChannelID]
		h.channelsMux.RUnlock()
		
		for client := range clients {
			select {
			case client.send <- event:
			default:
				// Client buffer full, skip
			}
		}
		
	case event.ServerID != nil:
		// Send to all clients subscribed to server
		h.serversMux.RLock()
		clients := h.servers[*event.ServerID]
		h.serversMux.RUnlock()
		
		for client := range clients {
			select {
			case client.send <- event:
			default:
			}
		}
		
	case event.UserID != nil:
		// Send to specific user (all their connections)
		h.clientsMux.RLock()
		clients := h.clients[*event.UserID]
		h.clientsMux.RUnlock()
		
		for client := range clients {
			select {
			case client.send <- event:
			default:
			}
		}
	}
}

// SubscribeChannel subscribes a client to a channel
func (h *Hub) SubscribeChannel(client *Client, channelID uuid.UUID) {
	h.channelsMux.Lock()
	defer h.channelsMux.Unlock()
	
	if h.channels[channelID] == nil {
		h.channels[channelID] = make(map[*Client]bool)
	}
	h.channels[channelID][client] = true
}

// UnsubscribeChannel unsubscribes a client from a channel
func (h *Hub) UnsubscribeChannel(client *Client, channelID uuid.UUID) {
	h.channelsMux.Lock()
	defer h.channelsMux.Unlock()
	
	if clients, ok := h.channels[channelID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channelID)
		}
	}
}

// SubscribeServer subscribes a client to a server
func (h *Hub) SubscribeServer(client *Client, serverID uuid.UUID) {
	h.serversMux.Lock()
	defer h.serversMux.Unlock()
	
	if h.servers[serverID] == nil {
		h.servers[serverID] = make(map[*Client]bool)
	}
	h.servers[serverID][client] = true
}

// Broadcast sends an event to the appropriate recipients
func (h *Hub) Broadcast(event *Event) {
	h.broadcast <- event
}

// SendToUser sends an event to a specific user
func (h *Hub) SendToUser(userID uuid.UUID, event *Event) {
	event.UserID = &userID
	h.broadcast <- event
}

// SendToChannel sends an event to all users in a channel
func (h *Hub) SendToChannel(channelID uuid.UUID, event *Event) {
	event.ChannelID = &channelID
	h.broadcast <- event
}

// SendToServer sends an event to all users in a server
func (h *Hub) SendToServer(serverID uuid.UUID, event *Event) {
	event.ServerID = &serverID
	h.broadcast <- event
}

// GetOnlineUsers returns IDs of users who have active connections
func (h *Hub) GetOnlineUsers(userIDs []uuid.UUID) []uuid.UUID {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()
	
	online := make([]uuid.UUID, 0)
	for _, id := range userIDs {
		if _, ok := h.clients[id]; ok {
			online = append(online, id)
		}
	}
	return online
}

// Client represents a WebSocket connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan *Event
	UserID uuid.UUID
	
	// Subscriptions
	Channels []uuid.UUID
	Servers  []uuid.UUID
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan *Event, 256),
		UserID:   userID,
		Channels: make([]uuid.UUID, 0),
		Servers:  make([]uuid.UUID, 0),
	}
}

// ReadPump handles incoming messages
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		
		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}
		
		c.handleEvent(&event)
	}
}

// WritePump handles outgoing messages
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case event, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
			
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleEvent(event *Event) {
	switch event.Type {
	case EventTypeTypingStart:
		// Broadcast typing indicator
		c.hub.SendToChannel(event.Data.(TypingData).ChannelID, &Event{
			Type: EventTypeTypingStart,
			Data: TypingData{
				ChannelID: event.Data.(TypingData).ChannelID,
				UserID:    c.UserID,
			},
		})
		
	case EventTypeSubscribe:
		// Subscribe to channel/server
		data := event.Data.(SubscribeData)
		if data.ChannelID != nil {
			c.hub.SubscribeChannel(c, *data.ChannelID)
		}
		if data.ServerID != nil {
			c.hub.SubscribeServer(c, *data.ServerID)
		}
	}
}

// Event types
const (
	EventTypeReady         = "READY"
	EventTypeMessageCreate = "MESSAGE_CREATE"
	EventTypeMessageUpdate = "MESSAGE_UPDATE"
	EventTypeMessageDelete = "MESSAGE_DELETE"
	EventTypeTypingStart   = "TYPING_START"
	EventTypePresenceUpdate = "PRESENCE_UPDATE"
	EventTypeServerCreate  = "SERVER_CREATE"
	EventTypeServerUpdate  = "SERVER_UPDATE"
	EventTypeServerDelete  = "SERVER_DELETE"
	EventTypeMemberJoin    = "MEMBER_JOIN"
	EventTypeMemberLeave   = "MEMBER_LEAVE"
	EventTypeMemberUpdate  = "MEMBER_UPDATE"
	EventTypeChannelCreate = "CHANNEL_CREATE"
	EventTypeChannelUpdate = "CHANNEL_UPDATE"
	EventTypeChannelDelete = "CHANNEL_DELETE"
	EventTypeReactionAdd   = "REACTION_ADD"
	EventTypeReactionRemove = "REACTION_REMOVE"
	EventTypeSubscribe     = "SUBSCRIBE"
	EventTypeUnsubscribe   = "UNSUBSCRIBE"
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"t"`
	Data      interface{} `json:"d"`
	Sequence  int64       `json:"s,omitempty"`
	
	// Routing (not serialized)
	UserID    *uuid.UUID  `json:"-"`
	ChannelID *uuid.UUID  `json:"-"`
	ServerID  *uuid.UUID  `json:"-"`
}

// Event data types
type TypingData struct {
	ChannelID uuid.UUID `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
}

type SubscribeData struct {
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	ServerID  *uuid.UUID `json:"server_id,omitempty"`
}
