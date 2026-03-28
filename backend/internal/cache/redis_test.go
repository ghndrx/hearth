package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func getRedisURL(t *testing.T) string {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set, skipping Redis tests")
	}
	return url
}

func newTestRedisCache(t *testing.T) *RedisCache {
	url := getRedisURL(t)
	cache, err := NewRedisCache(url)
	if err != nil {
		t.Skipf("Failed to connect to Redis at %s: %v", url, err)
	}
	return cache
}

func TestRedisCache_SetGetPresence(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

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

func TestRedisCache_GetPresenceMiss(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Get presence for non-existent user (cache miss)
	result, err := cache.GetPresence(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, models.StatusOffline, result.Status)
}

func TestRedisCache_SetGetSession(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

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
}

func TestRedisCache_GetSessionMiss(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()
	sessionID := uuid.New()

	// Get non-existent session should return error
	_, err := cache.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

func TestRedisCache_DeletePresence(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

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

func TestRedisCache_DeleteSession(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()
	sessionID := uuid.New()
	userID := uuid.New()
	ttl := 1 * time.Hour

	session := &models.Session{
		ID:         sessionID,
		UserID:     userID,
		DeviceType: models.DeviceTypeMobile,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}

	// Set session
	err := cache.SetSession(ctx, session, ttl)
	require.NoError(t, err)

	// Delete session
	err = cache.DeleteSession(ctx, sessionID)
	require.NoError(t, err)

	// Get should return error
	_, err = cache.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

// Helper for string pointer
func strPtr(s string) *string {
	return &s
}
