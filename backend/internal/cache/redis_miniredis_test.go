package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// newTestRedisCacheWithMiniredis creates a RedisCache backed by miniredis
func newTestRedisCacheWithMiniredis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Construct proper redis URL from miniredis address
	redisURL := "redis://" + mr.Addr() + "/0"
	cache, err := NewRedisCache(redisURL)
	require.NoError(t, err)

	return cache, mr
}

// TestRedisCache_GenericOperations tests basic Get/Set/Delete operations
func TestRedisCache_GenericOperations(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")
	ttl := 1 * time.Minute

	// Set value
	err := cache.Set(ctx, key, value, ttl)
	require.NoError(t, err)

	// Get value
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)

	// Delete value
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Get should fail after delete
	_, err = cache.Get(ctx, key)
	assert.Error(t, err)
}

// TestRedisCache_URL tests the URL method
func TestRedisCache_URL(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	url := cache.URL()
	assert.Contains(t, url, mr.Addr())
	assert.Contains(t, url, "redis://")
}

// TestRedisCache_Client tests the Client method
func TestRedisCache_Client(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	client := cache.Client()
	assert.NotNil(t, client)
}

// TestRedisCache_UserOperations tests user caching operations
func TestRedisCache_UserOperations(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	user := &models.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
	}
	ttl := 5 * time.Minute

	// Set user
	err := cache.SetUser(ctx, user, ttl)
	require.NoError(t, err)

	// Get user
	result, err := cache.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Username, result.Username)
	assert.Equal(t, user.Email, result.Email)

	// Delete user
	err = cache.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Get should return error
	_, err = cache.GetUser(ctx, user.ID)
	assert.Error(t, err)
}

// TestRedisCache_ServerOperations tests server caching operations
func TestRedisCache_ServerOperations(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	server := &models.Server{
		ID:        uuid.New(),
		Name:      "Test Server",
		CreatedAt: time.Now(),
	}
	ttl := 10 * time.Minute

	// Set server
	err := cache.SetServer(ctx, server, ttl)
	require.NoError(t, err)

	// Get server
	result, err := cache.GetServer(ctx, server.ID)
	require.NoError(t, err)
	assert.Equal(t, server.ID, result.ID)
	assert.Equal(t, server.Name, result.Name)

	// Delete server
	err = cache.DeleteServer(ctx, server.ID)
	require.NoError(t, err)

	// Get should return error
	_, err = cache.GetServer(ctx, server.ID)
	assert.Error(t, err)
}

// TestRedisCache_ChannelOperations tests channel caching operations
func TestRedisCache_ChannelOperations(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	channel := &models.Channel{
		ID:        uuid.New(),
		Name:      "test-channel",
		CreatedAt: time.Now(),
	}
	ttl := 5 * time.Minute

	// Set channel
	err := cache.SetChannel(ctx, channel, ttl)
	require.NoError(t, err)

	// Get channel
	result, err := cache.GetChannel(ctx, channel.ID)
	require.NoError(t, err)
	assert.Equal(t, channel.ID, result.ID)
	assert.Equal(t, channel.Name, result.Name)

	// Delete channel
	err = cache.DeleteChannel(ctx, channel.ID)
	require.NoError(t, err)

	// Get should return error
	_, err = cache.GetChannel(ctx, channel.ID)
	assert.Error(t, err)
}

// TestRedisCache_IncrementWithExpiry tests rate limiting counter
func TestRedisCache_IncrementWithExpiry(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	key := "rate:limit:1"
	ttl := 1 * time.Minute

	// First increment
	count1, err := cache.IncrementWithExpiry(ctx, key, ttl)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)

	// Second increment
	count2, err := cache.IncrementWithExpiry(ctx, key, ttl)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)

	// Third increment
	count3, err := cache.IncrementWithExpiry(ctx, key, ttl)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count3)
}

// TestRedisCache_PresenceCacheMiss tests that cache miss returns offline presence
func TestRedisCache_PresenceCacheMiss(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Get presence for non-existent user (cache miss)
	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, models.StatusOffline, result.Status)
}

// TestRedisCache_PresenceSetGet tests setting and getting presence
func TestRedisCache_PresenceSetGet(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	ttl := 1 * time.Minute

	presence := &models.Presence{
		UserID:       userID,
		Status:       models.StatusOnline,
		CustomStatus: strPtr("Working on stuff"),
		UpdatedAt:    time.Now(),
	}

	// Set presence
	err := cache.SetPresence(ctx, userID, presence, ttl)
	require.NoError(t, err)

	// Get presence
	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, models.StatusOnline, result.Status)
	assert.NotNil(t, result.CustomStatus)
	assert.Equal(t, "Working on stuff", *result.CustomStatus)
}

// TestRedisCache_PresenceDelete tests deleting presence
func TestRedisCache_PresenceDelete(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	ttl := 1 * time.Minute

	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusOnline,
	}

	// Set presence
	err := cache.SetPresence(ctx, userID, presence, ttl)
	require.NoError(t, err)

	// Delete presence
	err = cache.DeletePresence(ctx, userID)
	require.NoError(t, err)

	// Get should return offline presence (cache miss returns offline)
	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusOffline, result.Status)
}

// TestRedisCache_PresenceBulk tests bulk presence retrieval
func TestRedisCache_PresenceBulk(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()
	ttl := 1 * time.Minute

	// Set presence for user1 and user2 only
	presence1 := &models.Presence{UserID: userID1, Status: models.StatusOnline}
	presence2 := &models.Presence{UserID: userID2, Status: models.StatusIdle}

	err := cache.SetPresence(ctx, userID1, presence1, ttl)
	require.NoError(t, err)
	err = cache.SetPresence(ctx, userID2, presence2, ttl)
	require.NoError(t, err)

	// Get bulk presence for all 3 users
	results, err := cache.GetPresenceBulk(ctx, []uuid.UUID{userID1, userID2, userID3})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// User 1 should be online
	assert.Equal(t, models.StatusOnline, results[userID1].Status)

	// User 2 should be idle
	assert.Equal(t, models.StatusIdle, results[userID2].Status)

	// User 3 should be offline (cache miss)
	assert.Equal(t, models.StatusOffline, results[userID3].Status)
}

// TestRedisCache_PresenceBulkEmpty tests bulk presence with empty slice
func TestRedisCache_PresenceBulkEmpty(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	results, err := cache.GetPresenceBulk(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

// TestRedisCache_SessionOperations tests session caching operations
func TestRedisCache_SessionOperations(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	sessionID := uuid.New()
	userID := uuid.New()
	ttl := 1 * time.Hour

	session := &models.Session{
		ID:         sessionID,
		UserID:     userID,
		DeviceType: models.DeviceTypeDesktop,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}

	// Set session
	err := cache.SetSession(ctx, session, ttl)
	require.NoError(t, err)

	// Get session
	result, err := cache.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, models.DeviceTypeDesktop, result.DeviceType)

	// Delete session
	err = cache.DeleteSession(ctx, sessionID)
	require.NoError(t, err)

	// Get should return error
	_, err = cache.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

// TestRedisCache_SessionCacheMiss tests session cache miss
func TestRedisCache_SessionCacheMiss(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	sessionID := uuid.New()

	// Get non-existent session should return error
	_, err := cache.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

// TestRedisCache_PublishSubscribe tests pub/sub operations
func TestRedisCache_PublishSubscribe(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	channelName := "test-channel"
	message := map[string]interface{}{"type": "test", "data": "hello"}

	// Subscribe to channel
	pubsub := cache.Subscribe(ctx, channelName)
	require.NotNil(t, pubsub)

	// Give subscription time to set up
	time.Sleep(10 * time.Millisecond)

	// Publish message
	err := cache.Publish(ctx, channelName, message)
	require.NoError(t, err)

	// Receive message
	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Contains(t, msg.Payload, "test")
	assert.Contains(t, msg.Payload, "hello")

	pubsub.Close()
}

// TestRedisCache_Close tests closing the cache
func TestRedisCache_Close(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)

	// Close should not panic
	err := cache.Close()
	assert.NoError(t, err)

	// Clean up miniredis after cache is closed
	mr.Close()
}

// TestRedisCache_GetInvalidJSON tests handling of invalid JSON in cache
func TestRedisCache_GetInvalidJSON(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Directly set invalid JSON in Redis
	mr.Set("hearth:user:"+userID.String(), "not valid json")

	// GetUser should return error on invalid JSON
	_, err := cache.GetUser(ctx, userID)
	assert.Error(t, err)
}

// TestRedisCache_PresenceInvalidJSON tests handling of invalid JSON for presence
func TestRedisCache_PresenceInvalidJSON(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Directly set invalid JSON in Redis
	mr.Set("hearth:presence:"+userID.String(), "not valid json")

	// GetPresence should return offline presence on invalid JSON
	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusOffline, result.Status)
}

// TestRedisCache_IncrementMultipleKeys tests increment with multiple keys
func TestRedisCache_IncrementMultipleKeys(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	ttl := 1 * time.Minute

	// Increment different keys
	_, err := cache.IncrementWithExpiry(ctx, "key1", ttl)
	require.NoError(t, err)
	_, err = cache.IncrementWithExpiry(ctx, "key2", ttl)
	require.NoError(t, err)
	_, err = cache.IncrementWithExpiry(ctx, "key3", ttl)
	require.NoError(t, err)

	// Verify they are independent
	count1, err := cache.IncrementWithExpiry(ctx, "key1", ttl)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count1)

	count2, err := cache.IncrementWithExpiry(ctx, "key2", ttl)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)
}

// TestRedisCache_PresenceDNDStatus tests presence with DND status
func TestRedisCache_PresenceDNDStatus(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	ttl := 1 * time.Minute

	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusDND,
	}

	err := cache.SetPresence(ctx, userID, presence, ttl)
	require.NoError(t, err)

	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusDND, result.Status)
}

// TestRedisCache_PresenceStreamingStatus tests presence with streaming activity
func TestRedisCache_PresenceStreamingStatus(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	ttl := 1 * time.Minute
	now := time.Now()

	presence := &models.Presence{
		UserID: userID,
		Status: models.StatusOnline,
		Activities: []models.Activity{
			{
				Name:      "Hearth",
				Type:      models.ActivityTypeStreaming,
				URL:       "https://twitch.tv/hearth",
				CreatedAt: now,
			},
		},
	}

	err := cache.SetPresence(ctx, userID, presence, ttl)
	require.NoError(t, err)

	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusOnline, result.Status)
	assert.Len(t, result.Activities, 1)
	assert.Equal(t, "Hearth", result.Activities[0].Name)
	assert.Equal(t, models.ActivityTypeStreaming, result.Activities[0].Type)
}

// TestRedisCache_NewRedisCacheInvalidURL tests error handling for invalid URL
func TestRedisCache_NewRedisCacheInvalidURL(t *testing.T) {
	_, err := NewRedisCache("invalid-url")
	assert.Error(t, err)
}

// TestRedisCache_GenericCacheMiss tests generic cache miss
func TestRedisCache_GenericCacheMiss(t *testing.T) {
	cache, mr := newTestRedisCacheWithMiniredis(t)
	defer cache.Close()
	defer mr.Close()

	ctx := context.Background()

	// Get non-existent key
	_, err := cache.Get(ctx, "nonexistent")
	assert.Error(t, err)
}
