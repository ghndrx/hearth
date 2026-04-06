package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"

	"hearth/internal/auth"
	"hearth/internal/metrics"
)

// GatewayConfig holds gateway configuration
type GatewayConfig struct {
	HeartbeatInterval time.Duration
	SessionTimeout    time.Duration
	ConnectionLimiter *ConnectionLimiter
}

// DefaultGatewayConfig returns default configuration
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		HeartbeatInterval: 41250 * time.Millisecond, // ~41 seconds
		SessionTimeout:    5 * time.Minute,
		ConnectionLimiter: nil, // No connection limiting by default (opt-in)
	}
}

// Gateway manages WebSocket connections and message routing
type Gateway struct {
	hub        HubInterface
	jwtService *auth.JWTService
	config     *GatewayConfig

	// Voice signaling
	voiceService *VoiceSignalingService

	// Video signaling
	videoService *VideoSignalingService

	// Soundboard signaling
	soundboardService *SoundboardSignalingService

	// Activity signaling
	activityService *ActivitySignalingService

	// Session management
	sessions   map[string]*Session
	sessionsMu sync.RWMutex

	// Metrics (legacy counters for Stats())
	totalConnections  int64
	activeConnections int64
	messagesProcessed int64
	connectionsMu     sync.RWMutex

	// Prometheus metrics
	wsMetrics *metrics.WebSocketMetrics

	// Connection limiter (may be nil)
	connLimiter *ConnectionLimiter

	// Graceful shutdown state
	draining atomic.Bool
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
	ResumeKey    string
	ResumeEvents [][]byte
	resumeMu     sync.Mutex
}

// NewGateway creates a new WebSocket gateway
func NewGateway(hub HubInterface, jwtService *auth.JWTService, config *GatewayConfig) *Gateway {
	if config == nil {
		config = DefaultGatewayConfig()
	}

	return &Gateway{
		hub:         hub,
		jwtService:  jwtService,
		config:      config,
		sessions:    make(map[string]*Session),
		wsMetrics:   metrics.GetMetrics(),
		connLimiter: config.ConnectionLimiter,
	}
}

// Hub returns the underlying HubInterface for broadcasting events
func (g *Gateway) Hub() HubInterface {
	return g.hub
}

// HandleConnection handles a new WebSocket connection
func (g *Gateway) HandleConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Extract token from query params or header
	// First try query parameter (common for WebSocket clients like k6)
	token := conn.Query("token")

	// If no token in query, try Authorization header
	if token == "" {
		auth := conn.Headers("Authorization")
		if auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				token = auth[7:] // Remove "Bearer " prefix
			} else {
				// Use raw header value if no Bearer prefix (e.g., API key format)
				token = auth
			}
		}
	}

	// Log token extraction for debugging
	if token == "" {
		// Debug: log what headers ARE available (without exposing sensitive values)
		log.Printf("[Gateway] WebSocket auth failed: no token in query params or Authorization header, auth header present: %v",
			conn.Headers("Authorization") != "")
		g.sendClose(conn, 4001, "authentication failed")
		return
	}

	// Validate token
	claims, err := g.jwtService.ValidateAccessToken(token)
	if err != nil {
		// Log the specific error for debugging (but don't expose internal details)
		log.Printf("[Gateway] WebSocket auth failed: token validation error for user token: %v", err)
		g.sendClose(conn, 4001, "authentication failed")
		return
	}

	// Extract client IP for connection limiting
	clientIP := ExtractClientIP(conn)

	// Check per-IP and per-user connection limits (if limiter is configured)
	if g.connLimiter != nil {
		result := g.connLimiter.check(context.Background(), clientIP, claims.UserID)
		if !result.Allowed {
			log.Printf("[Gateway] Connection rejected for user %s from IP %s: %s (ip_count=%d, user_count=%d)",
				claims.UserID, clientIP, result.Reason, result.CurrentIPCount, result.CurrentUserCount)
			g.sendClose(conn, result.Code, result.Reason)
			return
		}
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

	// Record Prometheus metrics for connection opened
	g.wsMetrics.ConnectionOpened(clientType)
	g.wsMetrics.SessionCreated()
	connectionStart := time.Now()

	// Increment connection counters in Redis (after successful connection setup)
	if g.connLimiter != nil {
		g.connLimiter.Increment(context.Background(), clientIP, claims.UserID)
	}

	defer func() {
		g.connectionsMu.Lock()
		g.activeConnections--
		g.connectionsMu.Unlock()

		// Record Prometheus metrics for connection closed
		duration := time.Since(connectionStart).Seconds()
		g.wsMetrics.ConnectionClosed(clientType, duration)
		g.wsMetrics.SessionDestroyed()

		// Decrement connection counters in Redis
		if g.connLimiter != nil {
			g.connLimiter.Decrement(context.Background(), clientIP, claims.UserID)
		}
	}()

	// Create client for hub
	client := g.createHubClient(conn, session)

	// Register with hub
	g.hub.RegisterClient() <- client

	defer func() {
		// Cleanup voice state before unregistering
		g.CleanupSession(session.ID)
		g.hub.UnregisterClient() <- client
	}()

	// Send HELLO
	g.sendHello(conn)

	// Start read/write pumps
	go g.writePump(conn, client, session)
	g.readPump(conn, client, session)
}

func (g *Gateway) createHubClient(conn *websocket.Conn, session *Session) *Client {
	// Get the underlying Hub from the interface
	// For DistributedHub, we need access to its embedded Hub
	var baseHub *Hub
	switch h := g.hub.(type) {
	case *Hub:
		baseHub = h
	case *DistributedHub:
		baseHub = h.Hub
	default:
		// Fallback - try type assertion
		if dh, ok := g.hub.(*DistributedHub); ok {
			baseHub = dh.Hub
		}
	}

	// Create a wrapper that adapts fiber websocket to our Client
	return &Client{
		ID:            uuid.New().String(),
		UserID:        session.UserID,
		Username:      session.Username,
		hub:           baseHub,
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

			// Record message sent metric (try to extract event type)
			eventType := g.extractEventType(message)
			g.wsMetrics.MessageSent(eventType)

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
	startTime := time.Now()

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		g.sendError(conn, "invalid message format")
		return
	}

	// Record message received metric
	g.wsMetrics.MessageReceived(strconv.Itoa(msg.Op))

	g.connectionsMu.Lock()
	g.messagesProcessed++
	g.connectionsMu.Unlock()

	session.Sequence++

	switch msg.Op {
	case OpHeartbeat:
		g.wsMetrics.HeartbeatReceived()
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

	// Record message processing latency
	latency := time.Since(startTime).Seconds()
	g.wsMetrics.MessageProcessed(metrics.OpcodeToString(msg.Op), latency)
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
	// Voice signaling events
	case VoiceSignalJoin, VoiceSignalLeave, VoiceSignalOffer, VoiceSignalAnswer, VoiceSignalICECandidate, VoiceSignalSpeaking:
		g.handleVoiceDispatch(conn, client, session, dispatchData.T, dispatchData.D)
	// Soundboard events
	case SoundboardSignalPlay, SoundboardSignalStop:
		g.handleSoundboardDispatch(conn, client, session, dispatchData.T, dispatchData.D)
	// Activity events
	case ActivitySignalStart, ActivitySignalJoin, ActivitySignalLeave, ActivitySignalEnd, ActivitySignalGameMove, ActivitySignalSync:
		g.handleActivityDispatch(conn, client, session, dispatchData.T, dispatchData.D)
	// Video signaling events
	case VideoSignalRing, VideoSignalRingResponse, VideoSignalLeave, VideoSignalOffer, VideoSignalAnswer, VideoSignalICECandidate, VideoSignalStateUpdate, VideoSignalScreenStart, VideoSignalScreenStop:
		g.handleVideoDispatch(conn, client, session, dispatchData.T, dispatchData.D)
	default:
		log.Printf("[Gateway] Unknown client dispatch type: %s", dispatchData.T)
	}
}

func (g *Gateway) handleVoiceDispatch(conn *websocket.Conn, client *Client, session *Session, eventType string, data json.RawMessage) {
	if g.voiceService == nil {
		g.sendError(conn, "voice not available")
		return
	}

	ctx := context.Background()
	if err := g.voiceService.HandleVoiceMessage(ctx, client, session.ID, eventType, data); err != nil {
		log.Printf("[Gateway] Voice dispatch error: %v", err)
		g.sendError(conn, "voice operation failed")
	}
}

func (g *Gateway) handleSoundboardDispatch(conn *websocket.Conn, client *Client, session *Session, eventType string, data json.RawMessage) {
	if g.soundboardService == nil {
		g.sendError(conn, "soundboard not available")
		return
	}

	ctx := context.Background()
	if err := g.soundboardService.HandleSoundboardMessage(ctx, client, session.ID, eventType, data); err != nil {
		log.Printf("[Gateway] Soundboard dispatch error: %v", err)
		g.sendError(conn, "soundboard operation failed")
	}
}

func (g *Gateway) handleActivityDispatch(conn *websocket.Conn, client *Client, session *Session, eventType string, data json.RawMessage) {
	if g.activityService == nil {
		g.sendError(conn, "activities not available")
		return
	}

	ctx := context.Background()
	if err := g.activityService.HandleActivityMessage(ctx, client, session.ID, eventType, data); err != nil {
		log.Printf("[Gateway] Activity dispatch error: %v", err)
		g.sendError(conn, "activity operation failed")
	}
}

func (g *Gateway) handleVideoDispatch(conn *websocket.Conn, client *Client, session *Session, eventType string, data json.RawMessage) {
	if g.videoService == nil {
		g.sendError(conn, "video calls not available")
		return
	}

	ctx := context.Background()
	if err := g.videoService.HandleVideoMessage(ctx, client, session.ID, eventType, data); err != nil {
		log.Printf("[Gateway] Video dispatch error: %v", err)
		g.sendError(conn, "video operation failed")
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
		g.wsMetrics.ChannelSubscribed()
		log.Printf("[Gateway] User %s subscribed to channel %s", session.UserID, channelID)
	}

	if subData.ServerID != "" {
		serverID, err := uuid.Parse(subData.ServerID)
		if err != nil {
			log.Printf("[Gateway] Invalid server ID: %s", subData.ServerID)
			return
		}
		client.SubscribeServer(serverID)
		g.wsMetrics.ServerSubscribed()
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
		g.wsMetrics.ChannelUnsubscribed()
		log.Printf("[Gateway] User %s unsubscribed from channel %s", session.UserID, channelID)
	}

	if subData.ServerID != "" {
		serverID, err := uuid.Parse(subData.ServerID)
		if err != nil {
			return
		}
		client.UnsubscribeServer(serverID)
		g.wsMetrics.ServerUnsubscribed()
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
		ResumeURL:       "",              // Set if using resume URL
		Guilds:          []interface{}{}, // Will be populated by services
		PrivateChannels: []interface{}{},
		User: map[string]interface{}{
			"id":       session.UserID.String(),
			"username": session.Username,
		},
	}

	readyData, err := json.Marshal(ready)
	if err != nil {
		log.Printf("Failed to marshal ready data: %v", err)
		g.sendError(conn, "Internal server error")
		return
	}
	g.sendMessage(conn, &Message{
		Op:       OpDispatch,
		Type:     EventReady,
		Data:     readyData,
		Sequence: session.Sequence,
	})
}

func (g *Gateway) handlePresenceUpdate(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	var data struct {
		Status       string        `json:"status"`
		Activities   []interface{} `json:"activities"`
		Since        *int64        `json:"since"`
		AFK          bool          `json:"afk"`
		CustomStatus *struct {
			CustomText *string `json:"custom_text"`
			Emoji      *string `json:"emoji"`
		} `json:"custom_status,omitempty"`
	}

	if msg.Data != nil {
		json.Unmarshal(msg.Data, &data)
	}

	// Build user object with custom status info
	userObj := map[string]interface{}{
		"id": session.UserID.String(),
	}

	// Broadcast presence to subscribed servers
	presence := PresenceUpdateData{
		Status:     data.Status,
		Activities: data.Activities,
		User:       userObj,
	}

	presenceData, err := json.Marshal(presence)
	if err != nil {
		log.Printf("Failed to marshal presence data: %v", err)
		return
	}

	// Broadcast to all servers the user is in
	client.mu.RLock()
	servers := make([]uuid.UUID, 0, len(client.servers))
	for serverID := range client.servers {
		servers = append(servers, serverID)
	}
	client.mu.RUnlock()

	for _, serverID := range servers {
		g.hub.SendToServer(serverID, &Event{
			Op:   OpDispatch,
			Type: EventTypePresenceUpdate,
			Data: json.RawMessage(presenceData),
		})
	}
}

func (g *Gateway) handleVoiceStateUpdate(conn *websocket.Conn, client *Client, session *Session, msg *Message) {
	if g.voiceService == nil {
		g.sendError(conn, "voice not available")
		return
	}

	var data struct {
		ChannelID    *string `json:"channel_id"`
		ServerID     string  `json:"server_id"`
		SelfMuted    bool    `json:"self_muted"`
		SelfDeafened bool    `json:"self_deafened"`
		SelfVideo    bool    `json:"self_video"`
		SelfStream   bool    `json:"self_stream"`
	}

	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			g.sendError(conn, "invalid voice state data")
			return
		}
	}

	serverID, err := uuid.Parse(data.ServerID)
	if err != nil {
		g.sendError(conn, "invalid server id")
		return
	}

	// Handle state update
	ctx := context.Background()
	if err := g.voiceService.UpdateSelfState(ctx, client, serverID,
		data.SelfMuted, data.SelfDeafened, data.SelfVideo, data.SelfStream); err != nil {
		log.Printf("[Gateway] Voice state update error: %v", err)
		g.sendError(conn, "voice state update failed")
	}
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

	chunkData, err := json.Marshal(chunk)
	if err != nil {
		log.Printf("Failed to marshal guild members chunk data: %v", err)
		g.sendError(conn, "Failed to retrieve guild members")
		return
	}
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
		if err := conn.WriteMessage(websocket.TextMessage, event); err != nil {
			log.Printf("Failed to send buffered event during resume: %v", err)
			// Continue sending other events
		}
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

	helloData, err := json.Marshal(hello)
	if err != nil {
		log.Printf("Failed to marshal hello data: %v", err)
		conn.Close()
		return
	}
	g.sendMessage(conn, &Message{
		Op:   OpHello,
		Data: helloData,
	})
}

func (g *Gateway) sendMessage(conn *websocket.Conn, msg *Message) {
	if conn == nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send websocket message: %v", err)
	}
}

func (g *Gateway) sendError(conn *websocket.Conn, message string) {
	if conn == nil {
		return
	}
	errorData, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		log.Printf("Failed to marshal error data: %v", err)
		conn.Close()
		return
	}
	g.sendMessage(conn, &Message{
		Op:   OpDispatch,
		Type: "ERROR",
		Data: errorData,
	})
}

func (g *Gateway) sendClose(conn *websocket.Conn, code int, reason string) {
	if conn == nil {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"code":   code,
		"reason": reason,
	})
	if err := conn.WriteMessage(websocket.CloseMessage, payload); err != nil {
		// Log but don't fail on write errors during close
		log.Printf("Failed to write close message: %v", err)
	}
	if err := conn.Close(); err != nil {
		log.Printf("Failed to close connection: %v", err)
	}
}

// extractEventType extracts the event type from a message for metrics
func (g *Gateway) extractEventType(data []byte) string {
	var msg struct {
		Op   int    `json:"op"`
		Type string `json:"t"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return "unknown"
	}
	if msg.Type != "" {
		return msg.Type
	}
	return metrics.OpcodeToString(msg.Op)
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

	stats := map[string]interface{}{
		"total_connections":  g.totalConnections,
		"active_connections": g.activeConnections,
		"messages_processed": g.messagesProcessed,
		"active_sessions":    sessionCount,
		"draining":           g.draining.Load(),
	}

	// Include hub drain state if available
	if g.hub != nil {
		stats["drain_state"] = g.hub.DrainState().String()
	}

	return stats
}

// Shutdown initiates graceful shutdown of the gateway
// It broadcasts a reconnect signal to all clients and waits for them to disconnect
func (g *Gateway) Shutdown(ctx context.Context) error {
	log.Printf("[Gateway] Initiating graceful shutdown...")
	g.draining.Store(true)

	// Delegate to hub's shutdown which handles the actual draining
	if g.hub != nil {
		return g.hub.Shutdown(ctx)
	}

	return nil
}

// IsHealthy returns true if the gateway is accepting new connections
func (g *Gateway) IsHealthy() bool {
	if g.draining.Load() {
		return false
	}
	if g.hub != nil {
		return g.hub.IsHealthy()
	}
	return true
}

// IsDraining returns true if the gateway is in draining mode
func (g *Gateway) IsDraining() bool {
	return g.draining.Load() || (g.hub != nil && g.hub.IsDraining())
}

// DrainState returns the current drain state
func (g *Gateway) DrainState() DrainState {
	if g.hub != nil {
		return g.hub.DrainState()
	}
	if g.draining.Load() {
		return DrainStateDraining
	}
	return DrainStateHealthy
}

// GetActiveConnections returns the current number of active connections
func (g *Gateway) GetActiveConnections() int64 {
	g.connectionsMu.RLock()
	defer g.connectionsMu.RUnlock()
	return g.activeConnections
}

// SetVoiceService sets the voice signaling service
func (g *Gateway) SetVoiceService(vs *VoiceSignalingService) {
	g.voiceService = vs
}

// SetSoundboardService sets the soundboard signaling service
func (g *Gateway) SetSoundboardService(ss *SoundboardSignalingService) {
	g.soundboardService = ss
}

// SetActivityService sets the activity signaling service
func (g *Gateway) SetActivityService(as *ActivitySignalingService) {
	g.activityService = as
}

// SetVideoService sets the video signaling service
func (g *Gateway) SetVideoService(vs *VideoSignalingService) {
	g.videoService = vs
}

// CleanupSession cleans up voice state when a session disconnects
func (g *Gateway) CleanupSession(sessionID string) {
	if g.voiceService != nil {
		ctx := context.Background()
		if err := g.voiceService.CleanupSession(ctx, sessionID); err != nil {
			log.Printf("[Gateway] Failed to cleanup voice session %s: %v", sessionID, err)
		}
	}

	// Remove session from map
	g.sessionsMu.Lock()
	for key, s := range g.sessions {
		if s.ID == sessionID {
			delete(g.sessions, key)
			break
		}
	}
	g.sessionsMu.Unlock()
}
