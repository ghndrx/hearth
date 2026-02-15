package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGateway_DrainState(t *testing.T) {
	hub := NewHub()
	gateway := NewGateway(hub, nil, nil)

	// Initial state should be healthy
	assert.Equal(t, DrainStateHealthy, gateway.DrainState())
	assert.True(t, gateway.IsHealthy())
	assert.False(t, gateway.IsDraining())
}

func TestGateway_Shutdown(t *testing.T) {
	hub := NewHubWithDrainConfig(&DrainConfig{
		DrainTimeout: 1 * time.Second,
		GracePeriod:  100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	gateway := NewGateway(hub, nil, nil)

	// Initial state
	assert.True(t, gateway.IsHealthy())
	assert.False(t, gateway.IsDraining())

	// Start shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err := gateway.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// After shutdown
	assert.False(t, gateway.IsHealthy())
	assert.True(t, gateway.IsDraining() || gateway.DrainState() == DrainStateClosed)
}

func TestGateway_GetStats_IncludesDrainState(t *testing.T) {
	hub := NewHub()
	gateway := NewGateway(hub, nil, nil)

	stats := gateway.GetStats()

	assert.Contains(t, stats, "draining")
	assert.Contains(t, stats, "drain_state")
	assert.Equal(t, false, stats["draining"])
	assert.Equal(t, "healthy", stats["drain_state"])
}

func TestGateway_GetActiveConnections(t *testing.T) {
	hub := NewHub()
	gateway := NewGateway(hub, nil, nil)

	// Should be 0 initially
	assert.Equal(t, int64(0), gateway.GetActiveConnections())
}

func TestGateway_NilHub(t *testing.T) {
	gateway := &Gateway{}

	// Should handle nil hub gracefully
	assert.True(t, gateway.IsHealthy())
	assert.False(t, gateway.IsDraining())
	assert.Equal(t, DrainStateHealthy, gateway.DrainState())

	// Shutdown with nil hub should not panic
	err := gateway.Shutdown(context.Background())
	assert.NoError(t, err)
}
