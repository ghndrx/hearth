package pubsub

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRedisURL() string {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1" // Use DB 1 for tests
	}
	return url
}

func skipIfNoRedis(t *testing.T) {
	ps, err := New(getRedisURL(), "test-skip")
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	ps.Close()
}

func TestNew(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "test-node-1")
	require.NoError(t, err)
	require.NotNil(t, ps)
	
	defer ps.Close()

	assert.Equal(t, "test-node-1", ps.nodeID)
	assert.Equal(t, "hearth:pubsub:", ps.prefix)
}

func TestNewInvalidURL(t *testing.T) {
	_, err := New("invalid://url", "test-node")
	assert.Error(t, err)
}

func TestPublishAndReceive(t *testing.T) {
	skipIfNoRedis(t)

	// Create two separate pub/sub instances to simulate different nodes
	ps1, err := New(getRedisURL(), "node-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := New(getRedisURL(), "node-2")
	require.NoError(t, err)
	defer ps2.Close()

	channelID := uuid.New()
	
	// Setup receiver
	received := make(chan *BroadcastMessage, 1)
	ps2.OnMessage(func(msg *BroadcastMessage) {
		received <- msg
	})

	// Subscribe to channel on node 2
	err = ps2.SubscribeChannel(channelID)
	require.NoError(t, err)

	// Give subscription time to establish
	time.Sleep(100 * time.Millisecond)

	// Publish from node 1
	testData := map[string]string{"content": "Hello, World!"}
	err = ps1.PublishToChannel(context.Background(), channelID, TypeMessageCreate, testData)
	require.NoError(t, err)

	// Wait for message
	select {
	case msg := <-received:
		assert.Equal(t, TypeMessageCreate, msg.Type)
		assert.Equal(t, channelID, *msg.ChannelID)
		assert.Equal(t, "node-1", msg.OriginNode)
		
		var data map[string]string
		err := json.Unmarshal(msg.Data, &data)
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", data["content"])
		
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSelfMessageFiltering(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "same-node")
	require.NoError(t, err)
	defer ps.Close()

	channelID := uuid.New()
	
	received := make(chan *BroadcastMessage, 1)
	ps.OnMessage(func(msg *BroadcastMessage) {
		received <- msg
	})

	err = ps.SubscribeChannel(channelID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Publish from same node
	err = ps.PublishToChannel(context.Background(), channelID, TypeMessageCreate, "test")
	require.NoError(t, err)

	// Should NOT receive our own message
	select {
	case <-received:
		t.Fatal("Should not receive own message")
	case <-time.After(500 * time.Millisecond):
		// Expected - no message received
	}
}

func TestPublishToServer(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := New(getRedisURL(), "server-node-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := New(getRedisURL(), "server-node-2")
	require.NoError(t, err)
	defer ps2.Close()

	serverID := uuid.New()

	received := make(chan *BroadcastMessage, 1)
	ps2.OnMessage(func(msg *BroadcastMessage) {
		received <- msg
	})

	err = ps2.SubscribeServer(serverID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = ps1.PublishToServer(context.Background(), serverID, TypeMemberJoin, map[string]string{
		"user_id": uuid.New().String(),
	})
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Equal(t, TypeMemberJoin, msg.Type)
		assert.Equal(t, serverID, *msg.ServerID)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for server message")
	}
}

func TestPublishToUser(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := New(getRedisURL(), "user-node-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := New(getRedisURL(), "user-node-2")
	require.NoError(t, err)
	defer ps2.Close()

	userID := uuid.New()

	received := make(chan *BroadcastMessage, 1)
	ps2.OnMessage(func(msg *BroadcastMessage) {
		received <- msg
	})

	err = ps2.SubscribeUser(userID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = ps1.PublishToUser(context.Background(), userID, TypePresenceUpdate, map[string]string{
		"status": "online",
	})
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Equal(t, TypePresenceUpdate, msg.Type)
		assert.Equal(t, userID, *msg.UserID)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for user message")
	}
}

func TestMultipleHandlers(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := New(getRedisURL(), "multi-node-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := New(getRedisURL(), "multi-node-2")
	require.NoError(t, err)
	defer ps2.Close()

	channelID := uuid.New()

	var mu sync.Mutex
	handlersCalled := 0

	ps2.OnMessage(func(msg *BroadcastMessage) {
		mu.Lock()
		handlersCalled++
		mu.Unlock()
	})

	ps2.OnMessage(func(msg *BroadcastMessage) {
		mu.Lock()
		handlersCalled++
		mu.Unlock()
	})

	err = ps2.SubscribeChannel(channelID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = ps1.PublishToChannel(context.Background(), channelID, TypeMessageCreate, "test")
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 2, handlersCalled)
	mu.Unlock()
}

func TestUnsubscribe(t *testing.T) {
	skipIfNoRedis(t)

	ps1, err := New(getRedisURL(), "unsub-node-1")
	require.NoError(t, err)
	defer ps1.Close()

	ps2, err := New(getRedisURL(), "unsub-node-2")
	require.NoError(t, err)
	defer ps2.Close()

	channelID := uuid.New()

	received := make(chan *BroadcastMessage, 10)
	ps2.OnMessage(func(msg *BroadcastMessage) {
		received <- msg
	})

	// Subscribe
	err = ps2.SubscribeChannel(channelID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Send first message
	err = ps1.PublishToChannel(context.Background(), channelID, TypeMessageCreate, "msg1")
	require.NoError(t, err)

	select {
	case <-received:
		// Got first message
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for first message")
	}

	// Unsubscribe
	err = ps2.UnsubscribeChannel(channelID)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Send second message
	err = ps1.PublishToChannel(context.Background(), channelID, TypeMessageCreate, "msg2")
	require.NoError(t, err)

	// Should NOT receive second message
	select {
	case <-received:
		t.Fatal("Should not receive message after unsubscribe")
	case <-time.After(500 * time.Millisecond):
		// Expected
	}
}

func TestStats(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "stats-node")
	require.NoError(t, err)
	defer ps.Close()

	// Initial stats
	stats := ps.Stats()
	assert.Equal(t, "stats-node", stats["node_id"])
	assert.Equal(t, 0, stats["subscription_count"])

	// Subscribe to some channels
	err = ps.SubscribeChannel(uuid.New())
	require.NoError(t, err)
	err = ps.SubscribeServer(uuid.New())
	require.NoError(t, err)

	stats = ps.Stats()
	assert.Equal(t, 2, stats["subscription_count"])
}

func TestGracefulClose(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "close-node")
	require.NoError(t, err)

	// Subscribe to multiple channels
	for i := 0; i < 5; i++ {
		err = ps.SubscribeChannel(uuid.New())
		require.NoError(t, err)
	}

	// Close should not block or error
	err = ps.Close()
	assert.NoError(t, err)
}

func TestDoubleSubscribe(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "double-sub-node")
	require.NoError(t, err)
	defer ps.Close()

	channelID := uuid.New()

	// Subscribe twice
	err = ps.SubscribeChannel(channelID)
	require.NoError(t, err)

	err = ps.SubscribeChannel(channelID)
	require.NoError(t, err) // Should not error

	stats := ps.Stats()
	assert.Equal(t, 1, stats["subscription_count"]) // Should only have one subscription
}

func TestConcurrentPublish(t *testing.T) {
	skipIfNoRedis(t)

	ps, err := New(getRedisURL(), "concurrent-node")
	require.NoError(t, err)
	defer ps.Close()

	ctx := context.Background()
	channelID := uuid.New()

	var wg sync.WaitGroup
	numGoroutines := 10
	msgsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < msgsPerGoroutine; j++ {
				err := ps.PublishToChannel(ctx, channelID, TypeMessageCreate, map[string]int{
					"goroutine": id,
					"message":   j,
				})
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()
}
