package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536 // 64KB
)

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   uuid.UUID
	Username string

	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Subscriptions
	servers  map[uuid.UUID]bool
	channels map[uuid.UUID]bool
	mu       sync.RWMutex

	// Session info
	SessionID  string
	ClientType string // "desktop", "web", "mobile"

	// Heartbeat
	lastHeartbeat time.Time
	sequence      int64
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, username, sessionID, clientType string) *Client {
	return &Client{
		ID:            uuid.New().String(),
		UserID:        userID,
		Username:      username,
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		servers:       make(map[uuid.UUID]bool),
		channels:      make(map[uuid.UUID]bool),
		SessionID:     sessionID,
		ClientType:    clientType,
		lastHeartbeat: time.Now(),
		sequence:      0,
	}
}

// ReadPump pumps messages from the WebSocket to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastHeartbeat = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Parse and handle message
		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the WebSocket
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to the client
func (c *Client) Send(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
		// Client buffer full, close connection
		close(c.send)
		c.hub.unregister <- c
	}
}

// handleMessage processes an incoming WebSocket message
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	c.sequence++

	switch msg.Op {
	case OpHeartbeat:
		c.handleHeartbeat(&msg)
	case OpIdentify:
		c.handleIdentify(&msg)
	case OpResume:
		c.handleResume(&msg)
	case OpPresenceUpdate:
		c.handlePresenceUpdate(&msg)
	default:
		c.sendError("Unknown opcode")
	}
}

func (c *Client) handleHeartbeat(msg *Message) {
	c.lastHeartbeat = time.Now()
	c.Send(&Message{
		Op: OpHeartbeatAck,
	})
}

func (c *Client) handleIdentify(msg *Message) {
	// Send READY event
	c.sendReady()
}

func (c *Client) handleResume(msg *Message) {
	// Resume session - replay missed events
	c.Send(&Message{
		Op:   OpDispatch,
		Type: "RESUMED",
	})
}

func (c *Client) handlePresenceUpdate(msg *Message) {
	// Handle presence update
	// Will be integrated with presence service
}

func (c *Client) sendReady() {
	readyData := map[string]interface{}{
		"v":          10,
		"session_id": c.SessionID,
		"user": map[string]interface{}{
			"id":       c.UserID.String(),
			"username": c.Username,
		},
	}

	data, _ := json.Marshal(readyData)
	c.Send(&Message{
		Op:       OpDispatch,
		Type:     "READY",
		Data:     data,
		Sequence: c.sequence,
	})
}

func (c *Client) sendError(message string) {
	errorData := map[string]string{"message": message}
	data, _ := json.Marshal(errorData)
	c.Send(&Message{
		Op:   OpDispatch,
		Type: "ERROR",
		Data: data,
	})
}

// Subscription management

func (c *Client) SubscribeServer(serverID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.servers[serverID] = true
	c.hub.SubscribeServer(c, serverID)
}

func (c *Client) UnsubscribeServer(serverID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.servers, serverID)
}

func (c *Client) SubscribeChannel(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channelID] = true
	c.hub.SubscribeChannel(c, channelID)
}

func (c *Client) UnsubscribeChannel(channelID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channelID)
	c.hub.UnsubscribeChannel(c, channelID)
}

func (c *Client) IsSubscribedToServer(serverID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.servers[serverID]
}

func (c *Client) IsSubscribedToChannel(channelID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channels[channelID]
}
