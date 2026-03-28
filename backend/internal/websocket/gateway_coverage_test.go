package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/metrics"
)

func TestGateway_ExtractEventType(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	t.Run("dispatch event with type", func(t *testing.T) {
		data, _ := json.Marshal(map[string]interface{}{
			"op": 0,
			"t":  "MESSAGE_CREATE",
		})
		assert.Equal(t, "MESSAGE_CREATE", gateway.extractEventType(data))
	})

	t.Run("heartbeat event (no type)", func(t *testing.T) {
		data, _ := json.Marshal(map[string]interface{}{
			"op": 1,
		})
		result := gateway.extractEventType(data)
		// Should return opcode string via metrics helper
		assert.NotEmpty(t, result)
	})

	t.Run("invalid json", func(t *testing.T) {
		assert.Equal(t, "unknown", gateway.extractEventType([]byte("not json")))
	})

	t.Run("empty data", func(t *testing.T) {
		data, _ := json.Marshal(map[string]interface{}{
			"op": 0,
		})
		result := gateway.extractEventType(data)
		// No type, should return opcode string
		assert.NotEmpty(t, result)
	})
}

func TestGateway_CleanupSession(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	sessionID := "session-123"
	resumeKey := "resume-key-1"

	// Add a session
	gateway.sessionsMu.Lock()
	gateway.sessions[resumeKey] = &Session{
		ID:            sessionID,
		UserID:        uuid.New(),
		Username:      "testuser",
		ResumeKey:     resumeKey,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
	}
	gateway.sessionsMu.Unlock()

	gateway.sessionsMu.RLock()
	assert.Len(t, gateway.sessions, 1)
	gateway.sessionsMu.RUnlock()

	// Cleanup
	gateway.CleanupSession(sessionID)

	gateway.sessionsMu.RLock()
	assert.Empty(t, gateway.sessions)
	gateway.sessionsMu.RUnlock()
}

func TestGateway_CleanupSession_NonExistent(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	// Should not panic
	gateway.CleanupSession("nonexistent-session")
}

func TestGateway_SetVoiceService(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	assert.Nil(t, gateway.voiceService)

	// Can't create a real VoiceSignalingService without a DB, but we can test nil
	gateway.SetVoiceService(nil)
	assert.Nil(t, gateway.voiceService)
}

func TestGateway_SubscribeClient(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	client := &Client{
		ID:       uuid.New().String(),
		UserID:   uuid.New(),
		Username: "testuser",
		hub:      hub,
		send:     make(chan []byte, 256),
		servers:  make(map[uuid.UUID]bool),
		channels: make(map[uuid.UUID]bool),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	serverID := uuid.New()
	gateway.SubscribeClient(client, serverID)

	assert.True(t, client.IsSubscribedToServer(serverID))
}

func TestGateway_GetStats_WithSessions(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	// Add sessions
	gateway.sessionsMu.Lock()
	gateway.sessions["key1"] = &Session{ID: "s1", UserID: uuid.New()}
	gateway.sessions["key2"] = &Session{ID: "s2", UserID: uuid.New()}
	gateway.sessionsMu.Unlock()

	// Simulate some stats
	gateway.connectionsMu.Lock()
	gateway.totalConnections = 10
	gateway.activeConnections = 2
	gateway.messagesProcessed = 500
	gateway.connectionsMu.Unlock()

	stats := gateway.GetStats()

	assert.Equal(t, int64(10), stats["total_connections"])
	assert.Equal(t, int64(2), stats["active_connections"])
	assert.Equal(t, int64(500), stats["messages_processed"])
	assert.Equal(t, 2, stats["active_sessions"])
}

func TestGateway_GetActiveConnections_AfterIncrement(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	gateway.connectionsMu.Lock()
	gateway.activeConnections = 42
	gateway.connectionsMu.Unlock()

	assert.Equal(t, int64(42), gateway.GetActiveConnections())
}

func TestGateway_IsHealthy_WithDraining(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	assert.True(t, gateway.IsHealthy())

	// Set draining
	gateway.draining.Store(true)
	assert.False(t, gateway.IsHealthy())
}

func TestGateway_IsDraining_BothFlags(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	assert.False(t, gateway.IsDraining())

	// Gateway-level draining
	gateway.draining.Store(true)
	assert.True(t, gateway.IsDraining())
}

func TestGateway_DrainState_DrainingWithNoHub(t *testing.T) {
	gateway := &Gateway{}
	gateway.draining.Store(true)

	assert.Equal(t, DrainStateDraining, gateway.DrainState())
}

func TestGateway_NewGateway_DefaultConfig(t *testing.T) {
	hub := NewHub()
	gateway := NewGateway(hub, nil, nil)

	require.NotNil(t, gateway)
	assert.Equal(t, 41250*time.Millisecond, gateway.config.HeartbeatInterval)
	assert.Equal(t, 5*time.Minute, gateway.config.SessionTimeout)
}

func TestGateway_NewGateway_CustomConfig(t *testing.T) {
	hub := NewHub()
	cfg := &GatewayConfig{
		HeartbeatInterval: 30 * time.Second,
		SessionTimeout:    10 * time.Minute,
	}

	gateway := NewGateway(hub, nil, cfg)

	require.NotNil(t, gateway)
	assert.Equal(t, 30*time.Second, gateway.config.HeartbeatInterval)
	assert.Equal(t, 10*time.Minute, gateway.config.SessionTimeout)
}

func TestGateway_CleanupMultipleSessions(t *testing.T) {
	hub := NewHub()
	gateway := &Gateway{
		hub:       hub,
		config:    DefaultGatewayConfig(),
		sessions:  make(map[string]*Session),
		wsMetrics: metrics.GetMetrics(),
	}

	// Add multiple sessions
	for i := 0; i < 5; i++ {
		key := uuid.New().String()
		gateway.sessions[key] = &Session{
			ID:        uuid.New().String(),
			UserID:    uuid.New(),
			ResumeKey: key,
		}
	}

	// Add a target session
	targetID := "target-session"
	targetKey := "target-key"
	gateway.sessions[targetKey] = &Session{
		ID:        targetID,
		UserID:    uuid.New(),
		ResumeKey: targetKey,
	}

	assert.Len(t, gateway.sessions, 6)

	gateway.CleanupSession(targetID)

	assert.Len(t, gateway.sessions, 5)
}

func TestSession_ResumeEvents(t *testing.T) {
	session := &Session{
		ID:            uuid.New().String(),
		UserID:        uuid.New(),
		Username:      "testuser",
		ClientType:    "web",
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		ResumeKey:     uuid.New().String(),
		ResumeEvents:  make([][]byte, 0, 100),
	}

	// Add events
	for i := 0; i < 5; i++ {
		session.resumeMu.Lock()
		session.ResumeEvents = append(session.ResumeEvents, []byte(`{"test":"event"}`))
		session.resumeMu.Unlock()
	}

	session.resumeMu.Lock()
	assert.Len(t, session.ResumeEvents, 5)
	session.resumeMu.Unlock()
}

func TestGateway_Shutdown_SetsState(t *testing.T) {
	hub := NewHubWithDrainConfig(&DrainConfig{
		DrainTimeout: 500 * time.Millisecond,
		GracePeriod:  50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	gateway := NewGateway(hub, nil, nil)

	assert.False(t, gateway.IsDraining())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err := gateway.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	assert.True(t, gateway.draining.Load())
	assert.False(t, gateway.IsHealthy())
}
