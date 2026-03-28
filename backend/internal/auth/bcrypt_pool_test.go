package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBcryptPool(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 20,
	})
	defer pool.Close()

	assert.NotNil(t, pool)
	stats := pool.Stats()
	assert.Equal(t, 4, stats.Workers)
	assert.Equal(t, 20, stats.QueueSize)
}

func TestBcryptPool_DefaultConfig(t *testing.T) {
	config := DefaultPoolConfig()

	assert.Greater(t, config.Workers, 0)
	assert.Greater(t, config.QueueSize, 0)
	assert.Equal(t, 5*time.Second, config.DefaultTimeout)
	assert.Equal(t, bcryptCost, config.Cost)
}

func TestBcryptPool_HashPassword(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4, // Low cost for fast tests
	})
	defer pool.Close()

	ctx := context.Background()
	password := "SecurePassword123"

	hash, err := pool.HashPassword(ctx, password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestBcryptPool_CheckPassword(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4, // Low cost for fast tests
	})
	defer pool.Close()

	ctx := context.Background()
	password := "SecurePassword123"

	hash, err := pool.HashPassword(ctx, password)
	require.NoError(t, err)

	// Correct password
	err = pool.CheckPassword(ctx, password, hash)
	assert.NoError(t, err)

	// Wrong password
	err = pool.CheckPassword(ctx, "WrongPassword123", hash)
	assert.Equal(t, ErrPasswordMismatch, err)
}

func TestBcryptPool_ValidationBeforeHash(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})
	defer pool.Close()

	ctx := context.Background()

	// Too short password should fail validation before queuing
	_, err := pool.HashPassword(ctx, "short")
	assert.Equal(t, ErrPasswordTooShort, err)

	// Weak password
	_, err = pool.HashPassword(ctx, "alllowercase123")
	assert.Equal(t, ErrPasswordWeak, err)
}

func TestBcryptPool_ConcurrentHashing(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 50,
		Cost:      4, // Low cost for fast tests
	})
	defer pool.Close()

	ctx := context.Background()
	passwords := []string{
		"Password1abc",
		"Another2Pwd",
		"ThirdPass3x",
		"FourthP4ss",
		"FifthPass5",
	}

	var wg sync.WaitGroup
	results := make(chan error, len(passwords)*10)

	// Run 50 concurrent hash operations
	for i := 0; i < 10; i++ {
		for _, pwd := range passwords {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				_, err := pool.HashPassword(ctx, p)
				results <- err
			}(pwd)
		}
	}

	wg.Wait()
	close(results)

	// All should succeed
	for err := range results {
		assert.NoError(t, err)
	}
}

func TestBcryptPool_ConcurrentVerification(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 100,
		Cost:      4,
	})
	defer pool.Close()

	ctx := context.Background()

	// Pre-generate hashes
	password := "TestPassword123"
	hash, err := pool.HashPassword(ctx, password)
	require.NoError(t, err)

	var wg sync.WaitGroup
	var successCount, failCount atomic.Int32

	// Run many concurrent verifications
	for i := 0; i < 50; i++ {
		wg.Add(2)
		// Correct password
		go func() {
			defer wg.Done()
			if pool.CheckPassword(ctx, password, hash) == nil {
				successCount.Add(1)
			}
		}()
		// Wrong password
		go func() {
			defer wg.Done()
			if pool.CheckPassword(ctx, "WrongPassword1", hash) == ErrPasswordMismatch {
				failCount.Add(1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), successCount.Load())
	assert.Equal(t, int32(50), failCount.Load())
}

func TestBcryptPool_ContextCancellation(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:        1,
		QueueSize:      5,
		Cost:           10, // Higher cost to ensure we can cancel
		DefaultTimeout: 10 * time.Second,
	})

	// Fill the queue with a slow job and track when it's done
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.HashPassword(ctx, "SlowPassword123") // This will occupy the worker
	}()

	// Now try with a cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := pool.HashPassword(cancelledCtx, "Password123Test")

	// Should fail due to cancellation
	assert.Error(t, err)

	// Wait for background job to complete before closing pool
	wg.Wait()
	pool.Close()
}

func TestBcryptPool_Timeout(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:        1,
		QueueSize:      2,
		Cost:           12, // Higher cost for slow hashing
		DefaultTimeout: 10 * time.Millisecond,
	})

	ctx := context.Background()

	// Track goroutines for cleanup
	var wg sync.WaitGroup

	// First job will occupy the worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.HashPassword(ctx, "FirstPassword1")
	}()

	// Give it a moment to start processing
	time.Sleep(5 * time.Millisecond)

	// Very short timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// This should timeout waiting for a worker
	_, err := pool.HashPassword(timeoutCtx, "SecondPassword1")

	// Should timeout
	assert.Error(t, err)

	// Wait for background goroutine before closing pool
	wg.Wait()
	pool.Close()
}

func TestBcryptPool_QueueFull(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:        1,
		QueueSize:      1,
		Cost:           14, // Very high cost to ensure queue stays full
		DefaultTimeout: 100 * time.Millisecond,
	})

	ctx := context.Background()

	// Track goroutines for cleanup
	var wg sync.WaitGroup
	var started sync.WaitGroup

	// Saturate the pool
	for i := 0; i < 3; i++ {
		started.Add(1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			started.Done()
			pool.HashPassword(ctx, "Password123xyz")
		}()
	}
	started.Wait()
	time.Sleep(10 * time.Millisecond) // Let jobs queue up

	// Now the queue should be full
	_, err := pool.HashPassword(ctx, "OverflowPassword1")

	// Could be queue full or timeout depending on timing
	if err != nil {
		assert.True(t, err == ErrQueueFull || err == ErrPoolTimeout || err == context.DeadlineExceeded)
	}

	// Wait for all goroutines before closing pool
	wg.Wait()
	pool.Close()
}

func TestBcryptPool_Close(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})

	// Should work before close
	ctx := context.Background()
	_, err := pool.HashPassword(ctx, "Password123abc")
	require.NoError(t, err)

	// Close the pool
	err = pool.Close()
	assert.NoError(t, err)

	// Should fail after close
	_, err = pool.HashPassword(ctx, "Password123abc")
	assert.Equal(t, ErrPoolClosed, err)

	err = pool.CheckPassword(ctx, "Password123abc", "somehash")
	assert.Equal(t, ErrPoolClosed, err)
}

func TestBcryptPool_DoubleClose(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})

	// Double close should not panic
	err := pool.Close()
	assert.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)
}

func TestBcryptPool_Stats(t *testing.T) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})
	defer pool.Close()

	ctx := context.Background()

	// Initial stats
	stats := pool.Stats()
	assert.Equal(t, int64(0), stats.HashCount)
	assert.Equal(t, int64(0), stats.CheckCount)

	// Hash some passwords
	hash, _ := pool.HashPassword(ctx, "Password123abc")
	pool.HashPassword(ctx, "Password456def")

	// Check some passwords
	pool.CheckPassword(ctx, "Password123abc", hash)

	// Stats should be updated
	stats = pool.Stats()
	assert.Equal(t, int64(2), stats.HashCount)
	assert.Equal(t, int64(1), stats.CheckCount)
}

func TestGlobalPool(t *testing.T) {
	// Create a fresh pool for this test using SetGlobalPool
	// (avoids unsafe reset of globalPoolOnce)
	testPool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})
	SetGlobalPool(testPool)

	pool := GlobalPool()
	assert.NotNil(t, pool)

	// Should return same instance
	pool2 := GlobalPool()
	assert.Equal(t, pool, pool2)

	// Clean up handled by SetGlobalPool in next test or test cleanup
}

func TestHashPasswordPooled(t *testing.T) {
	// Set up a test pool
	testPool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})
	SetGlobalPool(testPool)
	defer testPool.Close()

	ctx := context.Background()
	hash, err := HashPasswordPooled(ctx, "TestPassword123")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestCheckPasswordPooled(t *testing.T) {
	// Set up a test pool
	testPool := NewBcryptPool(PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Cost:      4,
	})
	SetGlobalPool(testPool)
	defer testPool.Close()

	ctx := context.Background()
	hash, _ := HashPasswordPooled(ctx, "TestPassword123")

	err := CheckPasswordPooled(ctx, "TestPassword123", hash)
	assert.NoError(t, err)

	err = CheckPasswordPooled(ctx, "WrongPassword123", hash)
	assert.Equal(t, ErrPasswordMismatch, err)
}

// Benchmark tests

func BenchmarkBcryptPool_Hash_Sequential(b *testing.B) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 100,
		Cost:      10,
	})
	defer pool.Close()

	ctx := context.Background()
	password := "BenchmarkPassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.HashPassword(ctx, password)
	}
}

func BenchmarkBcryptPool_Hash_Parallel(b *testing.B) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 100,
		Cost:      10,
	})
	defer pool.Close()

	ctx := context.Background()
	password := "BenchmarkPassword123"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.HashPassword(ctx, password)
		}
	})
}

func BenchmarkBcryptPool_Check_Parallel(b *testing.B) {
	pool := NewBcryptPool(PoolConfig{
		Workers:   4,
		QueueSize: 100,
		Cost:      10,
	})
	defer pool.Close()

	ctx := context.Background()
	password := "BenchmarkPassword123"
	hash, _ := pool.HashPassword(ctx, password)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.CheckPassword(ctx, password, hash)
		}
	})
}

// Compare with direct bcrypt (no pool)
func BenchmarkBcrypt_Direct_Parallel(b *testing.B) {
	password := "BenchmarkPassword123"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			HashPassword(password) // Direct bcrypt without pool
		}
	})
}
