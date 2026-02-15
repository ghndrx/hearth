package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"

	"hearth/internal/auth"
)

// GatewayConfig holds gateway configuration
type GatewayConfig struct {
	HeartbeatInterval time.Duration
	SessionTimeout    time.Duration
}

// DefaultGatewayConfig returns default configuration
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		HeartbeatInterval: 41250 * time.Millisecond, // ~41 seconds
		SessionTimeout:    5 * time.Minute,
	}
}

// Gateway manages WebSocket connections and message routing
type Gateway struct {
	hub        *Hub
	jwtService *auth.JWTService
	config     *GatewayConfig

	// Session management
	sessions   map[string]*Session
	sessionsMu sync.RWMutex

	// Metrics
	totalConnections   int64
	activeConnections  int64
	messagesProcessed  int64
	connectionsMu      sync.RWMutex
}

// Session represents a WebSocket session
type Session struct {
	ID            string
	UserID        uuid.UUID
	Username      string
	ClientType    string
	CreatedAt     time.Time
	LastHeartbeat time.Time
	Sequence      int64
	
	// Resume support
	ResumeKey     string
	ResumeEvents  [][]byte
	resumeMu      sync.Mutex
}

// NewGateway creates a new WebSocket gateway
func NewGateway(hub *Hub, jwtService *auth.JWTService, config *GatewayConfig) *Gateway {
	if config == nil {
		config = DefaultGatewayConfig()
	}
	
	return &Gateway{
		hub:        hub,
		jwtService: jwtService,
		config:     config,
		sessions:   make(map[string]*Session),
	}
}

// HandleConnection handles a new WebSocket connection
func (g *Gateway) HandleConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Extract token from query params or header
	token := conn.Query("token")
	if token == "" {
		// Try Authorization header (passed through Fiber)
		if auth := conn.Headers("Authorization"); len(auth) > 7 {
			token = auth[7:] // Remove "Bearer "
		}
	}

	// Validate token
	claims, err := g.jwtService.ValidateAccessToken(token)
	if err != nil {
		g.sendClose(conn, 4001, "authentication failed")
		return
	}

	// Get connection metadata
	sessionID := conn.Query("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	
	clientType := conn.Query("client_type")
	if clientType == "" {
		clientType = "web"
	}

	// Check for session resume
	resumeKey := conn.Query("resume")
	if resumeKey != "" {
		if g.handleResume(conn, resumeKey, claims.UserID) {
			return // Resume handled separately
		}
	}

	// Create session
	session := &Session{
		ID:            sessionID,
		UserID:        claims.UserID,
		Username:      claims.Username,
		ClientType:    clientType,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Sequence:      0,
		ResumeKey:     uuid.New().String(),
		ResumeEvents:  make([][]byte, 0, 100),
	}

	g.sessionsMu.Lock()
	g.sessions[session.ResumeKey] = session
	g.sessionsMu.Unlock()

	// Track connection
	g.connectionsMu.Lock()
	g.totalConnections++
	g.activeConnections++
	g.connectionsMu.Unlock()

	defer func() {
		g.connectionsMu.Lock()
		g.activeConnections--
		g.connectionsMu.Unlock()
	}()

	// Create client for hub
	client := g.createHubClient(conn, session)

	// Register with hub
	g.hub.register <- client

	defer func() {
		g.hub.unregister <- client
	}()

	// Send HELLO
	g.sendHello(conn)

	// Start read/write pumps
	go g.writePump(conn, client, session)
	g.readPump(conn, client, session)
}

func (g *Gateway) createHubClient(conn *websocket.Conn, session *Session) *Client {
	// Create a wrapper that adapts fiber websocket to our Client
	return &Client{
		ID:            uuid.New().String(),
		UserID:        session.UserID,
		Username:      session.Username,
		hub:           g.hub,
		conn:          nil, // Will use fiber conn directly
		send:          make(chan []byte, 256),
		servers:       make(map[uuid.UUID]bool),
		channels:      make(map[uuid.UUID]bool),
		SessionID:     session.ID,
		ClientType:    session.ClientType,
		lastHeartbeat: time.Now(),
		sequence:      0,
	}
}

func (g *Gateway) readPump(conn *websocket.Conn, client *Client, session *Session) {
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		session.LastHeartbeat = time.Now()
		return nil
	})

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error if needed
			}
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		g.handleMessage(conn, client, session, data)
	}
}

func (g *Gateway) writePump(conn *websocket.Conn, client *Client, session *Session) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// Store for potential resume
			session.resumeMu.Lock()
			if len(session.ResumeEvents) < 100 {
				session.ResumeEvents = append(session.ResumeEvents, message)
			} else {
				// Slide window
				session.ResumeEvents = append(session.ResumeEvents[1:], message)
			}
			session.resumeMu.Unlock()

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (g *Gateway) handleMessage(conn *websocket.Conn, client *Client, session *Session, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		g.sendError(conn, "invalid message format")
		return
	}

	g.connectionsMu.Lock()
	g.messagesProcessed++
	g.connectionsMu.Unlock()

	session.Sequence++

	switch msg.Op {
	case OpHeartbeat:
		g.handleHeartbeat(conn, session)
		
	case OpIdentify:
		g.handleIdentify(conn, client, session, &msg)
		
	case OpPresenceUpdate:
		g.handlePresenceUpdate(conn, client, session, &msg)
		
	case OpVoiceStateUpdate:
		g.handleVoiceStateUpdate(conn, client, session, &msg)
		
	case OpResume:
		// Already handled at connection time
		g.sendError(conn, "resume must be initiated on connect")
		
	case OpRequestGuildMembers:
		g.handleRequestMembers(conn, client, session, &msg)
	
	case OpDispatch:
		// Handle client-sent dispatch events (like SUBSCRIBE)
		g.handleClientDispatch(conn, client, session, &msg)
		
	default:
		log.Printf("[Gateway] Unknown opcode: %d from user %s", msg.Op, session.UserID)
		g.sendError(conn, "unknown opcode")
	}
}

func (g *Gateway) handleClientDispatch(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	// Parse the dispatch event
	var dispatchData struct {
		T string          `json:"t"` // Event type
		D json.RawMessage `json:"d"` // Event data
	}
	
	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &dispatchData); err != nil {
			log.Printf("[Gateway] Failed to parse dispatch data: %v", err)
			return
		}
	}
	
	log.Printf("[Gateway] Client dispatch: %s from user %s", dispatchData.T, session.UserID)
	
	switch dispatchData.T {
	case "SUBSCRIBE":
		g.handleSubscribe(conn, client, session, dispatchData.D)
	case "UNSUBSCRIBE":
		g.handleUnsubscribe(conn, client, session, dispatchData.D)
	default:
		log.Printf("[Gateway] Unknown client dispatch type: %s", dispatchData.T)
	}
}

func (g *Gateway) handleSubscribe(conn *websocket.Conn, client *Client, session *Session, data json.RawMessage) {
	var subData struct {
		ChannelID string `json:"channel_id,omitempty"`
		ServerID  string `json:"server_id,omitempty"`
	}
	
	if err := json.Unmarshal(data, &subData); err != nil {
		log.Printf("[Gateway] Failed to parse subscribe data: %v", err)
		return
	}
	
	if subData.ChannelID != "" {
		channelID, err := uuid.Parse(subData.ChannelID)
		if err != nil {
			log.Printf("[Gateway] Invalid channel ID: %s", subData.ChannelID)
			return
		}
		client.SubscribeChannel(channelID)
		log.Printf("[Gateway] User %s subscribed to channel %s", session.UserID, channelID)
	}
	
	if subData.ServerID != "" {
		serverID, err := uuid.Parse(subData.ServerID)
		if err != nil {
			log.Printf("[Gateway] Invalid server ID: %s", subData.ServerID)
			return
		}
		client.SubscribeServer(serverID)
		log.Printf("[Gateway] User %s subscribed to server %s", session.UserID, serverID)
	}
}

func (g *Gateway) handleUnsubscribe(conn *websocket.Conn, client *Client, session *Session, data json.RawMessage) {
	var subData struct {
		ChannelID string `json:"channel_id,omitempty"`
		ServerID  string `json:"server_id,omitempty"`
	}
	
	if err := json.Unmarshal(data, &subData); err != nil {
		log.Printf("[Gateway] Failed to parse unsubscribe data: %v", err)
		return
	}
	
	if subData.ChannelID != "" {
		channelID, err := uuid.Parse(subData.ChannelID)
		if err != nil {
			return
		}
		client.UnsubscribeChannel(channelID)
		log.Printf("[Gateway] User %s unsubscribed from channel %s", session.UserID, channelID)
	}
	
	if subData.ServerID != "" {
		serverID, err := uuid.Parse(subData.ServerID)
		if err != nil {
			return
		}
		client.UnsubscribeServer(serverID)
		log.Printf("[Gateway] User %s unsubscribed from server %s", session.UserID, serverID)
	}
}

func (g *Gateway) handleHeartbeat(conn *websocket.Conn, session *Session) {
	session.LastHeartbeat = time.Now()
	g.sendMessage(conn, &Message{Op: OpHeartbeatAck})
}

func (g *Gateway) handleIdentify(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	var data struct {
		Properties struct {
			OS      string `json:"$os"`
			Browser string `json:"$browser"`
			Device  string `json:"$device"`
		} `json:"properties"`
		Compress bool `json:"compress"`
	}
	
	if msg.Data != nil {
		json.Unmarshal(msg.Data, &data)
	}

	// Send READY event
	ready := ReadyData{
		Version:         10,
		SessionID:       session.ID,
		ResumeURL:       "", // Set if using resume URL
		Guilds:          []interface{}{}, // Will be populated by services
		PrivateChannels: []interface{}{},
		User: map[string]interface{}{
			"id":       session.UserID.String(),
			"username": session.Username,
		},
	}

	readyData, _ := json.Marshal(ready)
	g.sendMessage(conn, &Message{
		Op:       OpDispatch,
		Type:     EventReady,
		Data:     readyData,
		Sequence: session.Sequence,
	})
}

func (g *Gateway) handlePresenceUpdate(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	var data struct {
		Status     string        `json:"status"`
		Activities []interface{} `json:"activities"`
		Since      *int64        `json:"since"`
		AFK        bool          `json:"afk"`
	}
	
	if msg.Data != nil {
		json.Unmarshal(msg.Data, &data)
	}

	// Broadcast presence to subscribed servers
	presence := PresenceUpdateData{
		Status:     data.Status,
		Activities: data.Activities,
		User: map[string]interface{}{
			"id": session.UserID.String(),
		},
	}

	presenceData, _ := json.Marshal(presence)
	
	// Broadcast to all servers the user is in
	client.mu.RLock()
	servers := make([]uuid.UUID, 0, len(client.servers))
	for serverID := range client.servers {
		servers = append(servers, serverID)
	}
	client.mu.RUnlock()

	for _, serverID := range servers {
		g.hub.SendToServer(serverID, &Event{
			Type: EventTypePresenceUpdate,
			Data: json.RawMessage(presenceData),
		})
	}
}

func (g *Gateway) handleVoiceStateUpdate(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	// Voice implementation placeholder
	// Will be implemented with WebRTC integration
}

func (g *Gateway) handleRequestMembers(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	var data struct {
		GuildID   string   `json:"guild_id"`
		Query     string   `json:"query"`
		Limit     int      `json:"limit"`
		Presences bool     `json:"presences"`
		UserIDs   []string `json:"user_ids"`
		Nonce     string   `json:"nonce"`
	}
	
	if msg.Data != nil {
		json.Unmarshal(msg.Data, &data)
	}

	// This would query the server service for members
	// For now, send an empty chunk
	chunk := map[string]interface{}{
		"guild_id":    data.GuildID,
		"members":     []interface{}{},
		"chunk_index": 0,
		"chunk_count": 1,
		"nonce":       data.Nonce,
	}

	chunkData, _ := json.Marshal(chunk)
	g.sendMessage(conn, &Message{
		Op:       OpDispatch,
		Type:     EventGuildMembersChunk,
		Data:     chunkData,
		Sequence: session.Sequence,
	})
}

func (g *Gateway) handleResume(conn *websocket.Conn, resumeKey string, userID uuid.UUID) bool {
	g.sessionsMu.RLock()
	session, ok := g.sessions[resumeKey]
	g.sessionsMu.RUnlock()

	if !ok || session.UserID != userID {
		g.sendClose(conn, 4006, "invalid session")
		return true
	}

	// Check if session is still valid
	if time.Since(session.LastHeartbeat) > g.config.SessionTimeout {
		g.sessionsMu.Lock()
		delete(g.sessions, resumeKey)
		g.sessionsMu.Unlock()
		g.sendClose(conn, 4009, "session timed out")
		return true
	}

	// Replay missed events
	session.resumeMu.Lock()
	events := session.ResumeEvents
	session.ResumeEvents = make([][]byte, 0, 100)
	session.resumeMu.Unlock()

	for _, event := range events {
		conn.WriteMessage(websocket.TextMessage, event)
	}

	// Send RESUMED
	g.sendMessage(conn, &Message{
		Op:   OpDispatch,
		Type: EventResumed,
	})

	return false // Continue with normal handling
}

func (g *Gateway) sendHello(conn *websocket.Conn) {
	hello := HelloData{
		HeartbeatInterval: int(g.config.HeartbeatInterval.Milliseconds()),
	}
	
	helloData, _ := json.Marshal(hello)
	g.sendMessage(conn, &Message{
		Op:   OpHello,
		Data: helloData,
	})
}

func (g *Gateway) sendMessage(conn *websocket.Conn, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

func (g *Gateway) sendError(conn *websocket.Conn, message string) {
	errorData, _ := json.Marshal(map[string]string{"message": message})
	g.sendMessage(conn, &Message{
		Op:   OpDispatch,
		Type: "ERROR",
		Data: errorData,
	})
}

func (g *Gateway) sendClose(conn *websocket.Conn, code int, reason string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"code":   code,
		"reason": reason,
	})
	conn.WriteMessage(websocket.CloseMessage, payload)
	conn.Close()
}

// SubscribeClient subscribes a client to a server's events
func (g *Gateway) SubscribeClient(client *Client, serverID uuid.UUID) {
	client.SubscribeServer(serverID)
}

// GetStats returns gateway statistics
func (g *Gateway) GetStats() map[string]interface{} {
	g.connectionsMu.RLock()
	defer g.connectionsMu.RUnlock()

	g.sessionsMu.RLock()
	sessionCount := len(g.sessions)
	g.sessionsMu.RUnlock()

	return map[string]interface{}{
		"total_connections":   g.totalConnections,
		"active_connections":  g.activeConnections,
		"messages_processed":  g.messagesProcessed,
		"active_sessions":     sessionCount,
	}
}
