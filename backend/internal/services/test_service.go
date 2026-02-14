package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ContextKey is a custom type for Request Context keys.
type ContextKey string

// KeySessionID is used to store the session ID in the request context.
const KeySessionID ContextKey = "session_id"

// Session represents a connected user session.
type Session struct {
	UserID    string
	SessionID string
	RTT       time.Duration // Round Trip Time for connection health
}

// Message represents an event payload (e.g., Join, Leave).
type Message struct {
	Type    string `json:"type"`
	UserID  string `json:"user_id"`
	Channel string `json:"channel"` // Optional: channel ID for future extension
}

// Mux provides thread-safe broadcast capabilities for channel events.
type Mux interface {
	Broadcast(channel string, msg Message)
}

// Logger logs service events.
type Logger interface {
	Debug(module, message string)
	Info(module, message string)
	Error(module, message string)
}

// GatewayService manages concurrent WebSocket connections (simulated here with /ws).
type GatewayService struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	channels      map[string]map[string]*Session // channelID -> userID -> Session
	logger        Logger
	eventMux      Mux
}

// NewGatewayService creates a new instance of the GatewayService.
func NewGatewayService(l Logger, m Mux) *GatewayService {
	return &GatewayService{
		sessions:   make(map[string]*Session),
		channels:   make(map[string]map[string]*Session),
		logger:     l,
		eventMux:   m,
	}
}

// HandleWS simulates the WebSocket upgrade handler.
func (s *GatewayService) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Simulate generating a Session ID
	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	
	// In a real implementation, this would upgrade the connection and start a goroutine.
	// Here, we simulate the backend behavior of tracking the session.
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Add User to general sessions
	session := &Session{
		UserID:    sessionID, // Simplified for demo: UserID == SessionID
		SessionID: sessionID,
		RTT:       time.Second,
	}
	s.sessions[sessionID] = session

	// 2. Add User to "general" channel
	if s.channels["general"] == nil {
		s.channels["general"] = make(map[string]*Session)
	}
	s.channels["general"][sessionID] = session

	// 3. Log the event
	s.logger.Info("Gateway", fmt.Sprintf("User [ID:%s] connected to 'general'", sessionID))

	// 4. Broadcast to channel (Notification via Mock Mux)
	s.eventMux.Broadcast("general", Message{
		Type:    "user_joined",
		UserID:  sessionID,
		Channel: "general",
	})

	// Set mock context ID for testing purposes
	ctx := r.Context()
	ctx = context.WithValue(ctx, KeySessionID, sessionID)
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusSwitchingProtocols) // Simulated 101
}

// GetChannelMembers retrieves active sessions for a specific channel.
func (s *GatewayService) GetChannelMembers(ctx context.Context, channelID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, exists := s.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", channelID)
	}

	// Convert map to slice
	result := make([]*Session, 0, len(members))
	for _, sess := range members {
		result = append(result, sess)
	}

	return result, nil
}

// CleanupStaleSessions removes sessions that have timed out.
func (s *GatewayService) CleanupStaleSessions(ctx context.Context, threshold time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	toRemove := make([]string, 0)

	for id, sess := range s.sessions {
		if now.Sub(sess.RTT) > threshold {
			toRemove = append(toRemove, id)
			s.logger.Debug("Gateway", fmt.Sprintf("Removing stale session %s", id))
		}
	}

	for _, id := range toRemove {
		delete(s.sessions, id)
		// Note: This simplified implementation doesn't decrement channel counts,
		// but in a real highly concurrent app, you might use range delete carefully.
	}

	return count
}