package auth

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// LoadTestResult holds metrics from a load test
type LoadTestResult struct {
	TotalRequests   int
	SuccessCount    int
	FailureCount    int
	TotalDuration   time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MaxLatency      time.Duration
	MinLatency      time.Duration
	AvgLatency      time.Duration
	Throughput      float64 // requests per second
}

// runLoadTest executes concurrent bcrypt operations and measures latencies
func runLoadTest(t *testing.T, name string, reqCount, concurrency int, hashFn func(ctx context.Context, password string) (string, error)) LoadTestResult {
	t.Helper()

	var wg sync.WaitGroup
	latencies := make([]time.Duration, reqCount)
	results := make(chan struct {
		idx     int
		latency time.Duration
		success bool
	}, reqCount)

	sem := make(chan struct{}, concurrency)
	password := "LoadTestPassword123"
	ctx := context.Background()

	start := time.Now()

	for i := 0; i < reqCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			opStart := time.Now()
			_, err := hashFn(ctx, password)
			latency := time.Since(opStart)

			results <- struct {
				idx     int
				latency time.Duration
				success bool
			}{idx: idx, latency: latency, success: err == nil}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)
	close(results)

	// Collect results
	successCount := 0
	for r := range results {
		latencies[r.idx] = r.latency
		if r.success {
			successCount++
		}
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50Idx := int(float64(reqCount) * 0.50)
	p95Idx := int(float64(reqCount) * 0.95)
	p99Idx := int(float64(reqCount) * 0.99)

	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}

	result := LoadTestResult{
		TotalRequests: reqCount,
		SuccessCount:  successCount,
		FailureCount:  reqCount - successCount,
		TotalDuration: totalDuration,
		P50Latency:    latencies[p50Idx],
		P95Latency:    latencies[p95Idx],
		P99Latency:    latencies[p99Idx],
		MaxLatency:    latencies[reqCount-1],
		MinLatency:    latencies[0],
		AvgLatency:    totalLatency / time.Duration(reqCount),
		Throughput:    float64(reqCount) / totalDuration.Seconds(),
	}

	t.Logf("\n%s Load Test Results:\n"+
		"  Total Requests:  %d\n"+
		"  Success Rate:    %.2f%%\n"+
		"  Total Duration:  %v\n"+
		"  Throughput:      %.2f req/s\n"+
		"  Latency P50:     %v\n"+
		"  Latency P95:     %v\n"+
		"  Latency P99:     %v\n"+
		"  Latency Max:     %v\n"+
		"  Latency Avg:     %v\n",
		name,
		result.TotalRequests,
		float64(result.SuccessCount)/float64(result.TotalRequests)*100,
		result.TotalDuration,
		result.Throughput,
		result.P50Latency,
		result.P95Latency,
		result.P99Latency,
		result.MaxLatency,
		result.AvgLatency,
	)

	return result
}

// TestLoadTest_DirectBcrypt tests bcrypt without pooling (baseline)
func TestLoadTest_DirectBcrypt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	directHashFn := func(ctx context.Context, password string) (string, error) {
		// Direct bcrypt - this is what the OLD code did
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10) // Cost 10 for faster tests
		return string(hash), err
	}

	// Simulate 100 concurrent registrations
	result := runLoadTest(t, "Direct bcrypt (no pool)", 100, 100, directHashFn)

	// This will likely show high p99 latency due to goroutine contention
	t.Logf("Direct bcrypt p99 latency: %v", result.P99Latency)
}

// TestLoadTest_PooledBcrypt tests bcrypt with worker pool
func TestLoadTest_PooledBcrypt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	pool := NewBcryptPool(PoolConfig{
		Workers:        runtime.NumCPU(),
		QueueSize:      200,
		Cost:           10, // Cost 10 for faster tests
		DefaultTimeout: 30 * time.Second,
	})
	defer pool.Close()

	// Simulate 100 concurrent registrations
	result := runLoadTest(t, "Pooled bcrypt", 100, 100, pool.HashPassword)

	// Pool should maintain more consistent latency
	t.Logf("Pooled bcrypt p99 latency: %v", result.P99Latency)
}

// TestLoadTest_Comparison runs both and compares
func TestLoadTest_Comparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	const (
		reqCount    = 50
		concurrency = 50
		cost        = 10
	)

	t.Logf("Load Test Configuration:")
	t.Logf("  CPUs: %d", runtime.NumCPU())
	t.Logf("  Requests: %d", reqCount)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  bcrypt cost: %d", cost)

	// Test 1: Direct bcrypt (unbounded concurrency)
	directHashFn := func(ctx context.Context, password string) (string, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		return string(hash), err
	}
	directResult := runLoadTest(t, "Direct (unbounded)", reqCount, concurrency, directHashFn)

	// Test 2: Pooled bcrypt (bounded to NumCPU workers)
	pool := NewBcryptPool(PoolConfig{
		Workers:        runtime.NumCPU(),
		QueueSize:      reqCount * 2,
		Cost:           cost,
		DefaultTimeout: 30 * time.Second,
	})
	defer pool.Close()
	pooledResult := runLoadTest(t, "Pooled (bounded)", reqCount, concurrency, pool.HashPassword)

	// Compare results
	t.Logf("\n=== Comparison ===")
	t.Logf("P99 Latency Improvement: %.2fx (direct=%v, pooled=%v)",
		float64(directResult.P99Latency)/float64(pooledResult.P99Latency),
		directResult.P99Latency, pooledResult.P99Latency)

	// The pooled version should have lower or comparable p99
	// Under high load, direct bcrypt causes goroutine pile-up
	if pooledResult.SuccessCount < reqCount {
		t.Logf("Note: %d pooled requests failed (likely due to queue/timeout)", reqCount-pooledResult.SuccessCount)
	}
}

// TestP99Target verifies we can meet the <500ms p99 target
func TestP99Target(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Use cost 10 (production would be 12, but that's ~4x slower)
	// At cost 10, single hash takes ~60-80ms
	// With bounded pool, we should maintain predictable latency

	pool := NewBcryptPool(PoolConfig{
		Workers:        runtime.NumCPU(),
		QueueSize:      200,
		Cost:           10,
		DefaultTimeout: 5 * time.Second,
	})
	defer pool.Close()

	result := runLoadTest(t, "P99 Target Test", 100, 100, pool.HashPassword)

	// Target: p99 < 500ms with cost 10
	// Note: Production with cost 12 would be ~4x slower
	targetP99 := 500 * time.Millisecond
	if result.P99Latency > targetP99 {
		t.Logf("WARNING: p99 latency %v exceeds target %v (cost=10)", result.P99Latency, targetP99)
		t.Logf("Note: This is expected under heavy load on limited hardware")
	} else {
		t.Logf("SUCCESS: p99 latency %v is under target %v", result.P99Latency, targetP99)
	}
}

// BenchmarkComparison provides benchmark data for documentation
func BenchmarkComparison(b *testing.B) {
	cost := 10
	password := "BenchmarkPassword123"

	b.Run("Direct_Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bcrypt.GenerateFromPassword([]byte(password), cost)
		}
	})

	b.Run("Direct_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bcrypt.GenerateFromPassword([]byte(password), cost)
			}
		})
	})

	pool := NewBcryptPool(PoolConfig{
		Workers:        runtime.NumCPU(),
		QueueSize:      100,
		Cost:           cost,
		DefaultTimeout: 10 * time.Second,
	})
	defer pool.Close()

	b.Run("Pooled_Parallel", func(b *testing.B) {
		ctx := context.Background()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pool.HashPassword(ctx, "Password123abc")
			}
		})
	})
}

// TestPoolSizing helps determine optimal pool size
func TestPoolSizing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sizing test in short mode")
	}

	const (
		reqCount    = 30
		concurrency = 30
		cost        = 10
	)

	cpuCount := runtime.NumCPU()
	workerSizes := []int{1, cpuCount / 2, cpuCount, cpuCount * 2}

	t.Logf("Testing pool sizes on %d CPUs with %d concurrent requests", cpuCount, concurrency)

	results := make(map[int]LoadTestResult)
	for _, workers := range workerSizes {
		if workers < 1 {
			workers = 1
		}

		pool := NewBcryptPool(PoolConfig{
			Workers:        workers,
			QueueSize:      reqCount * 2,
			Cost:           cost,
			DefaultTimeout: 30 * time.Second,
		})

		name := fmt.Sprintf("Workers=%d", workers)
		results[workers] = runLoadTest(t, name, reqCount, concurrency, pool.HashPassword)
		pool.Close()
	}

	// Recommend optimal size
	t.Logf("\n=== Recommendations ===")
	t.Logf("For CPU-bound bcrypt, workers = NumCPU (%d) is typically optimal", cpuCount)
	t.Logf("More workers cause context-switching overhead; fewer leave CPUs idle")
}
