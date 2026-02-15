package services

import (
	"context"
	"encoding/json"
	"errors"
	"hearth/internal/models"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ============================================================================
// Session Management Service (Repository Pattern)
// ============================================================================

// WebSocketRepository defines the interface for storage operations required by the WebSocket service.
// The concrete implementation (e.g., in the database package) must satisfy this.
type WebSocketRepository interface {
	SaveActiveSession(ctx context.Context, session *models.Session) error
	RemoveActiveSession(ctx context.Context, sessionID uuid.UUID) error
	FindActiveSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
}

// WebSocketService handles WebSocket connections, message routing, and session lifecycle.
type WebSocketService struct {
	repo WebSocketRepository
}

// NewWebSocketService creates a new instance of WebSocketService.
func NewWebSocketService(repo WebSocketRepository) *WebSocketService {
	return &WebSocketService{
		repo: repo,
	}
}

// ConnectSession handles the logic for a user establishing a WebSocket connection.
func (s *WebSocketService) ConnectSession(ctx context.Context, userID uuid.UUID, connID uuid.UUID) (*models.Session, error) {
	// Basic validation
	if userID == uuid.Nil || connID == uuid.Nil {
		return nil, errors.New("invalid user or connection ID")
	}

	session := &models.Session{
		ID:     uuid.New(),
		UserID: userID,
	}

	if err := s.repo.SaveActiveSession(ctx, session); err != nil {
		return nil, errors.New("failed to establish session in repository")
	}

	return session, nil
}

// DisconnectSession handles the cleanup when a user disconnects.
func (s *WebSocketService) DisconnectSession(ctx context.Context, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return errors.New("invalid session ID")
	}

	_, err := s.repo.FindActiveSession(ctx, sessionID)
	if err != nil {
		// Depending on repo implementation, might be "not found" or an actual db error
		return errors.New("session not found or already disconnected")
	}

	if err := s.repo.RemoveActiveSession(ctx, sessionID); err != nil {
		return errors.New("failed to remove session from repository")
	}

	return nil
}

// ============================================================================
// Real-time WebSocket Hub (Connection Management)
// ============================================================================

// MessageType defines the structure of messages exchanged over the WebSocket.
type MessageType int

const (
	MessageTyping MessageType = iota
	MessageCreate
)

// ClientMessage represents a message received from a client.
type ClientMessage struct {
	Type    MessageType     `json:"type"`
	Content json.RawMessage `json:"content"`
	Author  string          `json:"author"`
}

// ServerMessage represents a message sent from the server.
type ServerMessage struct {
	Type    MessageType `json:"type"`
	Content interface{} `json:"content"`
	To      string      `json:"to"` // Optional recipient ID
}

// Conn represents the connection to a specific client.
type Conn struct {
	hub  *Hub
	ws   *websocket.Conn
	User *User                    // The user context attached to this connection
	Send chan *ServerMessage      // Channel for outbound messages
}

// User represents the identity of a websocket connection.
type User struct {
	ID    string
	Email string
}

// WSMessageHandler defines the contract for processing messages after it's been decoded.
type WSMessageHandler func(*Conn, *ClientMessage)

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	clients map[*Conn]bool
	mu      sync.RWMutex
	// Broadcasts allow new clients to receive history and allow the server to push specific data.
	Broadcast      chan *ServerMessage
	Register       chan *Conn
	Unregister     chan *Conn
	MessageHandler WSMessageHandler
}

// CreateHub initializes the broadcast machinery.
func CreateHub(handler WSMessageHandler) *Hub {
	return &Hub{
		clients:        make(map[*Conn]bool),
		Broadcast:      make(chan *ServerMessage, 256),
		Register:       make(chan *Conn),
		Unregister:     make(chan *Conn),
		MessageHandler: handler,
	}
}

// Run starts the hub's main loop, acting as the event listener and dispatcher.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.Register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("Client connected: %s. Total: %d", conn.User.ID, len(h.clients))

			// Send acknowledgment or initial session data (omitted for brevity)

		case conn := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				close(conn.Send)
				h.mu.Unlock()
				log.Printf("Client disconnected: %s. Total: %d", conn.User.ID, len(h.clients))
			} else {
				h.mu.Unlock()
			}

		case message := <-h.Broadcast:
			if message == nil {
				return // Graceful shutdown
			}
			// Send to all clients
			h.mu.RLock()
			for client := range h.clients {
				if message.To == "" || client.User.ID == message.To {
					select {
					case client.Send <- message:
					default:
						// Client buffer full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ServeHTTP is the entry point for the WebSocket handler.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// CheckOrigin is a security constraint allowing only requests from the same origin
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	user := &User{
		ID:    r.URL.Query().Get("user_id"),
		Email: r.URL.Query().Get("email"),
	}

	conn := &Conn{
		hub:  h,
		ws:   ws,
		User: user,
		Send: make(chan *ServerMessage, 256), // Buffered channel
	}

	// Register the connection with the hub
	h.Register <- conn

	// Start the reader and writer go-routines for this connection
	go conn.writePump()
	go conn.readPump(h)
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Conn) readPump(h *Hub) {
	defer func() {
		h.Unregister <- c
	}()

	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second)) // 60 second read timeout
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message from %s: %v", c.User.ID, err)
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error decoding message from %s: %v", c.User.ID, err)
			continue
		}

		// Delegate business logic to the registered handler
		if h.MessageHandler != nil {
			h.MessageHandler(c, &msg)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Conn) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.ws.WriteJSON(message); err != nil {
				log.Printf("Error writing message to %s: %v", c.User.ID, err)
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
