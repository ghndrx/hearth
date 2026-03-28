package websocket

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/pubsub"
)

func TestHubImplementsInterface(t *testing.T) {
	hub := NewHub()
	var _ HubInterface = hub
}

func TestDistributedHubImplementsInterface(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "test-interface")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)
	var _ HubInterface = dh
	var _ DistributedHubInterface = dh
}

func TestGatewayWithHub(t *testing.T) {
	hub := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Gateway should accept Hub
	gateway := NewGateway(hub, nil, nil)
	require.NotNil(t, gateway)
}

func TestGatewayWithDistributedHub(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "test-gateway-dh")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dh.Run(ctx)

	// Gateway should accept DistributedHub
	gateway := NewGateway(dh, nil, nil)
	require.NotNil(t, gateway)
}

func TestRegisterUnregisterChannels(t *testing.T) {
	hub := NewHub()

	// Test that RegisterClient and UnregisterClient return channels
	regCh := hub.RegisterClient()
	unregCh := hub.UnregisterClient()

	assert.NotNil(t, regCh)
	assert.NotNil(t, unregCh)
}

func TestDistributedHubRegisterUnregisterChannels(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := pubsub.New(getRedisURL(), "test-reg-unreg")
	require.NoError(t, err)
	defer ps.Close()

	dh := NewDistributedHub(ps)

	regCh := dh.RegisterClient()
	unregCh := dh.UnregisterClient()

	assert.NotNil(t, regCh)
	assert.NotNil(t, unregCh)
}

// TestHubInterfacePolymorphism verifies both implementations work the same way
func TestHubInterfacePolymorphism(t *testing.T) {
	testCases := []struct {
		name      string
		createHub func(t *testing.T) (HubInterface, func())
	}{
		{
			name: "LocalHub",
			createHub: func(t *testing.T) (HubInterface, func()) {
				hub := NewHub()
				ctx, cancel := context.WithCancel(context.Background())
				go hub.Run(ctx)
				return hub, cancel
			},
		},
		{
			name: "DistributedHub",
			createHub: func(t *testing.T) (HubInterface, func()) {
				url := os.Getenv("REDIS_URL")
				if url == "" {
					url = "redis://localhost:6379/3"
				}
				ps, err := pubsub.New(url, "test-polymorphism-"+uuid.New().String()[:8])
				if err != nil {
					t.Skip("Redis not available")
				}
				dh := NewDistributedHub(ps)
				ctx, cancel := context.WithCancel(context.Background())
				go dh.Run(ctx)
				return dh, func() {
					cancel()
					ps.Close()
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hub, cleanup := tc.createHub(t)
			defer cleanup()

			time.Sleep(50 * time.Millisecond)

			// Test client registration
			userID := uuid.New()
			client := &Client{
				ID:            uuid.New().String(),
				UserID:        userID,
				Username:      "testuser",
				send:          make(chan []byte, 256),
				servers:       make(map[uuid.UUID]bool),
				channels:      make(map[uuid.UUID]bool),
				lastHeartbeat: time.Now(),
			}

			// Get underlying Hub for client
			var baseHub *Hub
			switch h := hub.(type) {
			case *Hub:
				baseHub = h
			case *DistributedHub:
				baseHub = h.Hub
			}
			client.hub = baseHub

			// Register client
			hub.RegisterClient() <- client
			time.Sleep(50 * time.Millisecond)

			// Subscribe to channel
			channelID := uuid.New()
			hub.SubscribeChannel(client, channelID)

			// Send event to channel
			hub.SendToChannel(channelID, &Event{
				Op:   0,
				Type: "TEST_EVENT",
				Data: map[string]string{"test": "data"},
			})

			select {
			case msg := <-client.send:
				assert.Contains(t, string(msg), "TEST_EVENT")
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for event")
			}

			// Unregister
			hub.UnregisterClient() <- client
		})
	}
}
