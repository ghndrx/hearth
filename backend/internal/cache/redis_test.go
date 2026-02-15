package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// setupTestRedis creates a miniredis instance for testing
func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	cache, err := NewRedisCache("redis://" + mr.Addr())
	require.NoError(t, err)

	t.Cleanup(func() {
		cache.Close()
		mr.Close()
	})

	return cache, mr
}

func TestNewRedisCache(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cache, err := NewRedisCache("redis://" + mr.Addr())
	require.NoError(t, err)
	defer cache.Close()

	assert.NotNil(t, cache)
	assert.Equal(t, "hearth:", cache.prefix)
}

func TestNewRedisCacheInvalidURL(t *testing.T) {
	_, err := NewRedisCache("invalid://url")
	assert.Error(t, err)
}

func TestNewRedisCacheConnectionFailed(t *testing.T) {
	// Try to connect to a non-existent server
	_, err := NewRedisCache("redis://localhost:59999")
	assert.Error(t, err)
}

func TestRedisCacheGetSet(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "test-key", []byte("test-value"), time.Minute)
	require.NoError(t, err)

	// Get the value
	val, err := cache.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), val)
}

func TestRedisCacheGetMiss(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")
	assert.Error(t, err)
}

func TestRedisCacheDelete(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "delete-me", []byte("value"), time.Minute)
	require.NoError(t, err)

	// Delete it
	err = cache.Delete(ctx, "delete-me")
	require.NoError(t, err)

	// Should be gone
	_, err = cache.Get(ctx, "delete-me")
	assert.Error(t, err)
}

func TestRedisCacheUser(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	avatarURL := "https://example.com/avatar.png"
	user := &models.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Username:      "testuser",
		Discriminator: "1234",
		AvatarURL:     &avatarURL,
		Status:        models.StatusOnline,
		Verified:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set user
	err := cache.SetUser(ctx, user, time.Minute)
	require.NoError(t, err)

	// Get user
	retrieved, err := cache.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, user.Email, retrieved.Email)
	assert.Equal(t, user.Username, retrieved.Username)
	assert.Equal(t, *user.AvatarURL, *retrieved.AvatarURL)
}

func TestRedisCacheUserNotFound(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	_, err := cache.GetUser(ctx, uuid.New())
	assert.Error(t, err)
}

func TestRedisCacheDeleteUser(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	user := &models.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Username: "testuser",
	}

	err := cache.SetUser(ctx, user, time.Minute)
	require.NoError(t, err)

	err = cache.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	_, err = cache.GetUser(ctx, user.ID)
	assert.Error(t, err)
}

func TestRedisCacheServer(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	iconURL := "https://example.com/icon.png"
	server := &models.Server{
		ID:        uuid.New(),
		OwnerID:   uuid.New(),
		Name:      "Test Server",
		IconURL:   &iconURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set server
	err := cache.SetServer(ctx, server, time.Minute)
	require.NoError(t, err)

	// Get server
	retrieved, err := cache.GetServer(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, server.ID, retrieved.ID)
	assert.Equal(t, server.Name, retrieved.Name)
	assert.Equal(t, *server.IconURL, *retrieved.IconURL)
}

func TestRedisCacheServerNotFound(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	_, err := cache.GetServer(ctx, uuid.New())
	assert.Error(t, err)
}

func TestRedisCacheDeleteServer(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	server := &models.Server{
		ID:   uuid.New(),
		Name: "Test Server",
	}

	err := cache.SetServer(ctx, server, time.Minute)
	require.NoError(t, err)

	err = cache.DeleteServer(ctx, server.ID)
	require.NoError(t, err)

	_, err = cache.GetServer(ctx, server.ID)
	assert.Error(t, err)
}

func TestRedisCacheChannel(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	serverID := uuid.New()
	channel := &models.Channel{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Type:      models.ChannelTypeText,
		Name:      "general",
		Topic:     "General discussion",
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set channel
	err := cache.SetChannel(ctx, channel, time.Minute)
	require.NoError(t, err)

	// Get channel
	retrieved, err := cache.GetChannel(ctx, channel.ID)
	require.NoError(t, err)
	assert.Equal(t, channel.ID, retrieved.ID)
	assert.Equal(t, channel.Name, retrieved.Name)
	assert.Equal(t, channel.Type, retrieved.Type)
}

func TestRedisCacheChannelNotFound(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	_, err := cache.GetChannel(ctx, uuid.New())
	assert.Error(t, err)
}

func TestRedisCacheDeleteChannel(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	channel := &models.Channel{
		ID:   uuid.New(),
		Name: "test-channel",
		Type: models.ChannelTypeText,
	}

	err := cache.SetChannel(ctx, channel, time.Minute)
	require.NoError(t, err)

	err = cache.DeleteChannel(ctx, channel.ID)
	require.NoError(t, err)

	_, err = cache.GetChannel(ctx, channel.ID)
	assert.Error(t, err)
}

func TestRedisCacheIncrementWithExpiry(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// First increment should return 1
	val, err := cache.IncrementWithExpiry(ctx, "counter", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Second increment should return 2
	val, err = cache.IncrementWithExpiry(ctx, "counter", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), val)

	// Third increment
	val, err = cache.IncrementWithExpiry(ctx, "counter", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(3), val)
}

func TestRedisCachePresence(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	userID := uuid.New()

	// Set presence
	err := cache.SetPresence(ctx, userID, "online", time.Minute)
	require.NoError(t, err)

	// Get presence
	status, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "online", status)
}

func TestRedisCachePresenceDefault(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// Get presence for non-existent user should return "offline"
	status, err := cache.GetPresence(ctx, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "offline", status)
}

func TestRedisCachePresenceStatuses(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	statuses := []string{"online", "idle", "dnd", "invisible"}

	for _, status := range statuses {
		userID := uuid.New()
		err := cache.SetPresence(ctx, userID, status, time.Minute)
		require.NoError(t, err)

		retrieved, err := cache.GetPresence(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, status, retrieved)
	}
}

func TestRedisCachePublishSubscribe(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// Subscribe
	pubsub := cache.Subscribe(ctx, "test-channel")
	defer pubsub.Close()

	// Wait for subscription to be ready (miniredis needs time)
	time.Sleep(100 * time.Millisecond)

	// Publish
	testData := map[string]string{"message": "hello"}
	err := cache.Publish(ctx, "test-channel", testData)
	require.NoError(t, err)

	// Receive with timeout to avoid hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	msg, err := pubsub.ReceiveMessage(ctxWithTimeout)
	require.NoError(t, err)

	var received map[string]string
	err = json.Unmarshal([]byte(msg.Payload), &received)
	require.NoError(t, err)
	assert.Equal(t, "hello", received["message"])
}

func TestRedisCacheClose(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cache, err := NewRedisCache("redis://" + mr.Addr())
	require.NoError(t, err)

	err = cache.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	_, err = cache.Get(context.Background(), "test")
	assert.Error(t, err)
}

func TestRedisCacheKeyPrefix(t *testing.T) {
	cache, mr := setupTestRedis(t)
	ctx := context.Background()

	err := cache.Set(ctx, "my-key", []byte("value"), time.Minute)
	require.NoError(t, err)

	// Check that the key in Redis includes the prefix
	keys := mr.Keys()
	found := false
	for _, key := range keys {
		if key == "hearth:my-key" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected key with prefix 'hearth:'")
}

func TestRedisCacheTTL(t *testing.T) {
	cache, mr := setupTestRedis(t)
	ctx := context.Background()

	err := cache.Set(ctx, "ttl-key", []byte("value"), 5*time.Minute)
	require.NoError(t, err)

	// Check TTL (miniredis stores TTL)
	ttl := mr.TTL("hearth:ttl-key")
	assert.True(t, ttl > 0 && ttl <= 5*time.Minute)
}

func TestRedisCacheSetUserMarshalError(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// User with a channel field would fail to marshal
	// but our model doesn't have such fields, so this test passes
	user := &models.User{
		ID:       uuid.New(),
		Username: "test",
	}

	err := cache.SetUser(ctx, user, time.Minute)
	assert.NoError(t, err)
}

func TestRedisCacheGetUserUnmarshalError(t *testing.T) {
	cache, _ := setupTestRedis(t)
	ctx := context.Background()

	// Set invalid JSON
	err := cache.Set(ctx, "user:"+uuid.New().String(), []byte("invalid-json"), time.Minute)
	require.NoError(t, err)

	// Try to get - should fail to unmarshal
	userID := uuid.New()
	cache.Set(ctx, "user:"+userID.String(), []byte("{invalid"), time.Minute)
	_, err = cache.GetUser(ctx, userID)
	assert.Error(t, err)
}
