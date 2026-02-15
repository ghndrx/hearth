package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/contrib/websocket"
)

// DrainConfig holds configuration for graceful connection draining
type DrainConfig struct {
	// DrainTimeout is the maximum time to wait for connections to close gracefully
	DrainTimeout time.Duration

	// GracePeriod is the time between sending reconnect signal and force-closing connections
	GracePeriod time.Duration
}

// DefaultDrainConfig returns sensible defaults for connection draining
func DefaultDrainConfig() *DrainConfig {
	return &DrainConfig{
		DrainTimeout: 30 * time.Second,
		GracePeriod:  5 * time.Second,
	}
}

// DrainState represents the current draining state
type DrainState int32

const (
	// DrainStateHealthy indicates normal operation
	DrainStateHealthy DrainState = iota
	// DrainStateDraining indicates the server is draining connections
	DrainStateDraining
	// DrainStateClosed indicates the server has closed all connections
	DrainStateClosed
)

// String returns a string representation of the drain state
func (s DrainState) String() string {
	switch s {
	case DrainStateHealthy:
		return "healthy"
	case DrainStateDraining:
		return "draining"
	case DrainStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// DrainManager manages graceful connection draining
type DrainManager struct {
	config *DrainConfig
	state  atomic.Int32

	// Callback to get all active clients
	getClients func() []*Client

	// Callback when draining is complete
	onDrainComplete func()

	// For coordinating shutdown
	drainOnce sync.Once
	drainDone chan struct{}
}

// NewDrainManager creates a new drain manager
func NewDrainManager(config *DrainConfig, getClients func() []*Client) *DrainManager {
	if config == nil {
		config = DefaultDrainConfig()
	}

	dm := &DrainManager{
		config:     config,
		getClients: getClients,
		drainDone:  make(chan struct{}),
	}
	dm.state.Store(int32(DrainStateHealthy))

	return dm
}

// State returns the current drain state
func (dm *DrainManager) State() DrainState {
	return DrainState(dm.state.Load())
}

// IsHealthy returns true if the server is accepting new connections
func (dm *DrainManager) IsHealthy() bool {
	return dm.State() == DrainStateHealthy
}

// IsDraining returns true if the server is in draining mode
func (dm *DrainManager) IsDraining() bool {
	return dm.State() == DrainStateDraining
}

// SetOnDrainComplete sets a callback to be invoked when draining is complete
func (dm *DrainManager) SetOnDrainComplete(fn func()) {
	dm.onDrainComplete = fn
}

// StartDrain initiates graceful connection draining
// It sends a reconnect signal to all clients, waits for them to disconnect,
// then returns when all connections are closed or timeout is reached
func (dm *DrainManager) StartDrain(ctx context.Context) error {
	var drainErr error

	dm.drainOnce.Do(func() {
		log.Printf("[Drain] Starting graceful connection draining (timeout: %v, grace: %v)",
			dm.config.DrainTimeout, dm.config.GracePeriod)

		// Transition to draining state
		dm.state.Store(int32(DrainStateDraining))

		// Get all active clients
		clients := dm.getClients()
		clientCount := len(clients)
		log.Printf("[Drain] Broadcasting reconnect to %d clients", clientCount)

		// Send reconnect signal to all clients
		dm.broadcastReconnect(clients)

		// Create a context with drain timeout
		drainCtx, cancel := context.WithTimeout(ctx, dm.config.DrainTimeout)
		defer cancel()

		// Wait for grace period before checking connection counts
		graceTicker := time.NewTimer(dm.config.GracePeriod)
		defer graceTicker.Stop()

		select {
		case <-graceTicker.C:
			// Grace period over, start polling for connection drain
		case <-drainCtx.Done():
			log.Printf("[Drain] Context cancelled during grace period")
			drainErr = drainCtx.Err()
			dm.state.Store(int32(DrainStateClosed))
			close(dm.drainDone)
			return
		}

		// Poll for connections to close
		pollTicker := time.NewTicker(500 * time.Millisecond)
		defer pollTicker.Stop()

		for {
			select {
			case <-pollTicker.C:
				remaining := len(dm.getClients())
				if remaining == 0 {
					log.Printf("[Drain] All connections drained successfully")
					dm.state.Store(int32(DrainStateClosed))
					close(dm.drainDone)
					if dm.onDrainComplete != nil {
						dm.onDrainComplete()
					}
					return
				}
				log.Printf("[Drain] Waiting for %d connections to close...", remaining)

			case <-drainCtx.Done():
				remainingClients := dm.getClients()
				remaining := len(remainingClients)
				if remaining > 0 {
					log.Printf("[Drain] Drain timeout reached, force-closing %d connections", remaining)
					closed := dm.ForceCloseClients(remainingClients, CloseGoingAway, "server shutdown")
					log.Printf("[Drain] Force-closed %d connections", closed)
				} else {
					log.Printf("[Drain] All connections drained before timeout")
				}
				dm.state.Store(int32(DrainStateClosed))
				close(dm.drainDone)
				if dm.onDrainComplete != nil {
					dm.onDrainComplete()
				}
				return
			}
		}
	})

	return drainErr
}

// WaitForDrain blocks until draining is complete
func (dm *DrainManager) WaitForDrain() {
	<-dm.drainDone
}

// WebSocket close codes for shutdown
const (
	// CloseGoingAway indicates the server is going down (standard code 1001)
	CloseGoingAway = websocket.CloseGoingAway // 1001
	// CloseServiceRestart indicates the server is restarting
	CloseServiceRestart = 1012
)

// broadcastReconnect sends a reconnect opcode to all connected clients
func (dm *DrainManager) broadcastReconnect(clients []*Client) {
	// Create the reconnect message
	// OpReconnect (7) tells clients to reconnect to a different gateway
	reconnectData, _ := json.Marshal(map[string]interface{}{
		"reason": "server_shutdown",
	})

	msg := &Message{
		Op:   OpReconnect,
		Data: reconnectData,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Drain] Failed to marshal reconnect message: %v", err)
		return
	}

	// Send to all clients (non-blocking)
	var sent, failed int
	for _, client := range clients {
		select {
		case client.send <- msgBytes:
			sent++
		default:
			// Client buffer full, skip
			failed++
		}
	}

	log.Printf("[Drain] Sent reconnect to %d clients (%d failed due to full buffer)", sent, failed)
}

// ForceCloseClients forcefully closes remaining client connections with a close code
func (dm *DrainManager) ForceCloseClients(clients []*Client, closeCode int, reason string) int {
	closed := 0
	for _, client := range clients {
		if client.conn != nil {
			// Send WebSocket close frame with code and reason
			closeMsg := websocket.FormatCloseMessage(closeCode, reason)
			if err := client.conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
				log.Printf("[Drain] Failed to send close to client %s: %v", client.ID, err)
			}
			if err := client.conn.Close(); err != nil {
				log.Printf("[Drain] Failed to close client %s: %v", client.ID, err)
			} else {
				closed++
			}
		}
		// Close the send channel to stop writePump
		select {
		case <-client.send:
			// Already closed
		default:
			close(client.send)
		}
	}
	return closed
}

// ReconnectEvent is the event type for reconnect signals
const EventReconnect = "RECONNECT"

// ReconnectData contains information about why the client should reconnect
type ReconnectData struct {
	Reason    string `json:"reason"`
	ResumeURL string `json:"resume_gateway_url,omitempty"`
}
