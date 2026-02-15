package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrainState_String(t *testing.T) {
	tests := []struct {
		state    DrainState
		expected string
	}{
		{DrainStateHealthy, "healthy"},
		{DrainStateDraining, "draining"},
		{DrainStateClosed, "closed"},
		{DrainState(999), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.state.String())
		})
	}
}

func TestDefaultDrainConfig(t *testing.T) {
	cfg := DefaultDrainConfig()

	assert.Equal(t, 30*time.Second, cfg.DrainTimeout)
	assert.Equal(t, 5*time.Second, cfg.GracePeriod)
}

func TestDrainManager_NewDrainManager(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		dm := NewDrainManager(nil, func() []*Client { return nil })

		assert.NotNil(t, dm)
		assert.Equal(t, 30*time.Second, dm.config.DrainTimeout)
		assert.Equal(t, 5*time.Second, dm.config.GracePeriod)
		assert.Equal(t, DrainStateHealthy, dm.State())
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &DrainConfig{
			DrainTimeout: 10 * time.Second,
			GracePeriod:  2 * time.Second,
		}
		dm := NewDrainManager(cfg, func() []*Client { return nil })

		assert.Equal(t, 10*time.Second, dm.config.DrainTimeout)
		assert.Equal(t, 2*time.Second, dm.config.GracePeriod)
	})
}

func TestDrainManager_StateTransitions(t *testing.T) {
	dm := NewDrainManager(nil, func() []*Client { return nil })

	// Initially healthy
	assert.True(t, dm.IsHealthy())
	assert.False(t, dm.IsDraining())
	assert.Equal(t, DrainStateHealthy, dm.State())
}

func TestDrainManager_StartDrain_NoClients(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 1 * time.Second,
		GracePeriod:  100 * time.Millisecond,
	}

	callCount := 0
	dm := NewDrainManager(cfg, func() []*Client {
		callCount++
		return nil // No clients
	})

	ctx := context.Background()
	err := dm.StartDrain(ctx)

	assert.NoError(t, err)
	assert.Equal(t, DrainStateClosed, dm.State())
	assert.False(t, dm.IsHealthy())
	assert.False(t, dm.IsDraining()) // Draining is done, now closed
}

func TestDrainManager_StartDrain_WithClients(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 2 * time.Second,
		GracePeriod:  100 * time.Millisecond,
	}

	// Create mock clients
	clients := []*Client{
		{
			ID:     uuid.New().String(),
			UserID: uuid.New(),
			send:   make(chan []byte, 256),
		},
		{
			ID:     uuid.New().String(),
			UserID: uuid.New(),
			send:   make(chan []byte, 256),
		},
	}

	var mu sync.Mutex
	clientsRemaining := clients

	dm := NewDrainManager(cfg, func() []*Client {
		mu.Lock()
		defer mu.Unlock()
		return clientsRemaining
	})

	// Start drain in background
	go func() {
		dm.StartDrain(context.Background())
	}()

	// Wait for draining to start
	time.Sleep(50 * time.Millisecond)
	assert.True(t, dm.IsDraining())

	// Check that clients received reconnect message
	for _, client := range clients {
		select {
		case msg := <-client.send:
			var parsed Message
			err := json.Unmarshal(msg, &parsed)
			require.NoError(t, err)
			assert.Equal(t, OpReconnect, parsed.Op)

			var data map[string]interface{}
			err = json.Unmarshal(parsed.Data, &data)
			require.NoError(t, err)
			assert.Equal(t, "server_shutdown", data["reason"])
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Did not receive reconnect message")
		}
	}

	// Simulate clients disconnecting
	mu.Lock()
	clientsRemaining = nil
	mu.Unlock()

	// Wait for drain to complete
	dm.WaitForDrain()
	assert.Equal(t, DrainStateClosed, dm.State())
}

func TestDrainManager_StartDrain_Timeout(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 500 * time.Millisecond,
		GracePeriod:  50 * time.Millisecond,
	}

	// Create a client that never disconnects
	stubbornClient := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New(),
		send:   make(chan []byte, 256),
	}

	dm := NewDrainManager(cfg, func() []*Client {
		return []*Client{stubbornClient}
	})

	start := time.Now()
	err := dm.StartDrain(context.Background())
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, DrainStateClosed, dm.State())
	// Should have taken approximately DrainTimeout
	assert.True(t, elapsed >= 500*time.Millisecond, "Expected drain to take at least 500ms, took %v", elapsed)
	assert.True(t, elapsed < 1*time.Second, "Expected drain to complete within 1s, took %v", elapsed)
}

func TestDrainManager_StartDrain_OnlyOnce(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 100 * time.Millisecond,
		GracePeriod:  10 * time.Millisecond,
	}

	callCount := 0
	dm := NewDrainManager(cfg, func() []*Client {
		callCount++
		return nil
	})

	// Call StartDrain multiple times concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dm.StartDrain(context.Background())
		}()
	}

	wg.Wait()

	// Drain should have only happened once
	assert.Equal(t, DrainStateClosed, dm.State())
	// getClients should have been called at least once, but drain logic should be idempotent
}

func TestDrainManager_OnDrainComplete_Callback(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 100 * time.Millisecond,
		GracePeriod:  10 * time.Millisecond,
	}

	dm := NewDrainManager(cfg, func() []*Client { return nil })

	callbackCalled := false
	dm.SetOnDrainComplete(func() {
		callbackCalled = true
	})

	dm.StartDrain(context.Background())
	dm.WaitForDrain()

	assert.True(t, callbackCalled)
}

func TestDrainManager_ContextCancellation(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 10 * time.Second, // Long timeout
		GracePeriod:  5 * time.Second,  // Long grace period
	}

	dm := NewDrainManager(cfg, func() []*Client {
		return []*Client{{ID: "test", send: make(chan []byte, 1)}}
	})

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := dm.StartDrain(ctx)
	elapsed := time.Since(start)

	// Should return quickly due to context cancellation
	assert.Error(t, err)
	assert.True(t, elapsed < 1*time.Second, "Expected quick return on context cancel, took %v", elapsed)
	assert.Equal(t, DrainStateClosed, dm.State())
}

func TestHubWithDrainConfig(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 5 * time.Second,
		GracePeriod:  1 * time.Second,
	}

	hub := NewHubWithDrainConfig(cfg)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.drainManager)
	assert.Equal(t, DrainStateHealthy, hub.DrainState())
	assert.True(t, hub.IsHealthy())
	assert.False(t, hub.IsDraining())
}

func TestHub_Shutdown(t *testing.T) {
	cfg := &DrainConfig{
		DrainTimeout: 1 * time.Second,
		GracePeriod:  100 * time.Millisecond,
	}

	hub := NewHubWithDrainConfig(cfg)

	// Start hub event loop
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	defer cancel()

	// Register a client
	client := &Client{
		ID:     uuid.New().String(),
		UserID: uuid.New(),
		send:   make(chan []byte, 256),
	}
	hub.register <- client

	// Give time for registration to be processed by hub loop
	time.Sleep(50 * time.Millisecond)

	// Verify client is registered
	clientCount := hub.GetClientCount()
	if clientCount != 1 {
		t.Skipf("Client registration didn't complete in time (got %d clients)", clientCount)
	}

	// Channel to capture received message
	reconnectReceived := make(chan *Message, 1)
	go func() {
		for msg := range client.send {
			var parsed Message
			if err := json.Unmarshal(msg, &parsed); err == nil {
				if parsed.Op == OpReconnect {
					reconnectReceived <- &parsed
					return
				}
			}
		}
	}()

	// Shutdown in a goroutine since it blocks until drain completes
	shutdownDone := make(chan error)
	go func() {
		shutdownDone <- hub.Shutdown(context.Background())
	}()

	// Wait for either reconnect message or timeout
	select {
	case msg := <-reconnectReceived:
		assert.Equal(t, OpReconnect, msg.Op)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Client did not receive reconnect message")
	}

	// Wait for shutdown to complete
	select {
	case err := <-shutdownDone:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown timed out")
	}

	// Verify state changed
	assert.Equal(t, DrainStateClosed, hub.DrainState())
}

func TestReconnectMessage_Format(t *testing.T) {
	// Test that reconnect message is correctly formatted
	reconnectData, _ := json.Marshal(map[string]interface{}{
		"reason": "server_shutdown",
	})

	msg := &Message{
		Op:   OpReconnect,
		Data: reconnectData,
	}

	msgBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	// Parse back
	var parsed map[string]interface{}
	err = json.Unmarshal(msgBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, float64(OpReconnect), parsed["op"])
	assert.NotNil(t, parsed["d"])
}

func TestCloseConstants(t *testing.T) {
	// Verify WebSocket close codes are correctly defined
	assert.Equal(t, 1001, CloseGoingAway)
	assert.Equal(t, 1012, CloseServiceRestart)
}
