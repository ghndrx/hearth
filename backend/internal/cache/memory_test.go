package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func TestMemoryCache_GetSet(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get value
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestMemoryCache_GetMiss(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	_, err := cache.Get(ctx, "nonexistent")
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set and delete
	cache.Set(ctx, key, value, 1*time.Minute)
	err := cache.Delete(ctx, key)
	require.NoError(t, err)

	// Should be gone
	_, err = cache.Get(ctx, key)
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_Expiry(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set with very short TTL
	err := cache.Set(ctx, key, value, 1*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, err = cache.Get(ctx, key)
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_Overwrite(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value1 := []byte("value1")
	value2 := []byte("value2")

	// Set first value
	cache.Set(ctx, key, value1, 1*time.Minute)

	// Overwrite with second value
	cache.Set(ctx, key, value2, 1*time.Minute)

	// Should get second value
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value2, result)
}

func TestMemoryCache_MultipleKeys(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Set multiple keys
	for i := 0; i < 100; i++ {
		key := string(rune('a' + i%26))
		value := []byte{byte(i)}
		cache.Set(ctx, key, value, 1*time.Minute)
	}

	// Verify we can get them back
	for i := 0; i < 26; i++ {
		key := string(rune('a' + i))
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		// Last value for this key should be present
		assert.NotEmpty(t, result)
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(n int) {
			key := "concurrent-key"
			value := []byte{byte(n)}
			for j := 0; j < 100; j++ {
				cache.Set(ctx, key, value, 1*time.Minute)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			key := "concurrent-key"
			for j := 0; j < 100; j++ {
				cache.Get(ctx, key)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMemoryCache_GetUser(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	_, err := cache.GetUser(ctx, uuid.New())
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_SetUser(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	user := &models.User{ID: uuid.New(), Username: "test"}
	err := cache.SetUser(ctx, user, 1*time.Minute)
	assert.NoError(t, err)
}

func TestMemoryCache_DeleteUser(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	err := cache.DeleteUser(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestMemoryCache_GetServer(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	_, err := cache.GetServer(ctx, uuid.New())
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_SetServer(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	server := &models.Server{ID: uuid.New(), Name: "Test Server"}
	err := cache.SetServer(ctx, server, 1*time.Minute)
	assert.NoError(t, err)
}

func TestMemoryCache_DeleteServer(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	err := cache.DeleteServer(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestMemoryCache_GetChannel(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	_, err := cache.GetChannel(ctx, uuid.New())
	assert.Equal(t, ErrCacheMiss, err)
}

func TestMemoryCache_SetChannel(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	channel := &models.Channel{ID: uuid.New(), Name: "test-channel"}
	err := cache.SetChannel(ctx, channel, 1*time.Minute)
	assert.NoError(t, err)
}

func TestMemoryCache_DeleteChannel(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	err := cache.DeleteChannel(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Add items with different expiries
	cache.Set(ctx, "short-lived", []byte("value1"), 50*time.Millisecond)
	cache.Set(ctx, "long-lived", []byte("value2"), 1*time.Minute)

	// Verify both items exist initially
	_, err := cache.Get(ctx, "short-lived")
	require.NoError(t, err)
	_, err = cache.Get(ctx, "long-lived")
	require.NoError(t, err)

	// Wait for the short-lived item to expire
	time.Sleep(100 * time.Millisecond)

	// The Get method should return cache miss for expired items
	_, err = cache.Get(ctx, "short-lived")
	assert.Equal(t, ErrCacheMiss, err)

	// Long-lived should still be there
	_, err = cache.Get(ctx, "long-lived")
	require.NoError(t, err)

	// Verify that we can still set and get new items (cache is functional)
	cache.Set(ctx, "new-item", []byte("new-value"), 1*time.Minute)
	result, err := cache.Get(ctx, "new-item")
	require.NoError(t, err)
	assert.Equal(t, []byte("new-value"), result)
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache()

	// Should not error on close
	err := cache.Close()
	assert.NoError(t, err)

	// Should be able to close again without issue
	err = cache.Close()
	assert.NoError(t, err)
}

func TestMemoryCache_EmptyValue(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "empty-key"
	value := []byte{}

	// Set empty value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get empty value
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
	assert.Empty(t, result)
}

func TestMemoryCache_NilValue(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "nil-key"
	var value []byte

	// Set nil value
	err := cache.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Get nil value
	result, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, result)
}
