// Package auth provides authentication primitives including bcrypt password
// hashing with bounded concurrency.
//
// # Bcrypt Worker Pool
//
// The bcrypt worker pool (BcryptPool) solves the CPU saturation problem that
// occurs when many goroutines perform bcrypt operations simultaneously.
// Without bounds, under load (e.g., 100 concurrent registrations), goroutines
// pile up competing for CPU, causing:
//   - p99 latency degradation (220ms → 3.3s observed)
//   - Context switching overhead
//   - Memory pressure from goroutine stacks
//   - Starvation of other services
//
// The worker pool bounds concurrent bcrypt operations to NumCPU workers,
// queuing excess requests. This provides:
//   - Predictable, bounded latency
//   - Efficient CPU utilization without thrashing
//   - Protection for other services sharing the CPU
//
// # Configuration
//
// Configure via environment variables:
//   - BCRYPT_POOL_WORKERS: Number of workers (default: NumCPU)
//   - BCRYPT_POOL_QUEUE: Queue size (default: Workers * 10)
//   - BCRYPT_POOL_TIMEOUT: Operation timeout (default: 5s)
//
// # Usage
//
// Use the pooled functions instead of direct bcrypt calls:
//
//	hash, err := auth.HashPasswordPooled(ctx, password)
//	err := auth.CheckPasswordPooled(ctx, password, hash)
//
// The pool is initialized at application startup in main.go.
package auth

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrPoolClosed is returned when operations are attempted on a closed pool
	ErrPoolClosed = errors.New("bcrypt pool is closed")
	// ErrPoolTimeout is returned when a job times out waiting for a worker
	ErrPoolTimeout = errors.New("bcrypt operation timed out")
	// ErrQueueFull is returned when the job queue is at capacity
	ErrQueueFull = errors.New("bcrypt job queue is full")
)

// PoolConfig configures the bcrypt worker pool
type PoolConfig struct {
	// Workers is the number of concurrent bcrypt workers.
	// Default: runtime.NumCPU() (recommended for CPU-bound work)
	Workers int

	// QueueSize is the maximum number of pending jobs.
	// Default: Workers * 10
	QueueSize int

	// DefaultTimeout is the default timeout for operations.
	// Default: 5 seconds
	DefaultTimeout time.Duration

	// Cost is the bcrypt cost factor.
	// Default: 12 (same as bcryptCost constant)
	Cost int
}

// DefaultPoolConfig returns sensible defaults for the pool
func DefaultPoolConfig() PoolConfig {
	numCPU := runtime.NumCPU()
	return PoolConfig{
		Workers:        numCPU,
		QueueSize:      numCPU * 10,
		DefaultTimeout: 5 * time.Second,
		Cost:           bcryptCost,
	}
}

// hashJob represents a password hashing request
type hashJob struct {
	password string
	result   chan hashResult
}

// hashResult contains the result of a hash operation
type hashResult struct {
	hash string
	err  error
}

// checkJob represents a password verification request
type checkJob struct {
	password string
	hash     string
	result   chan error
}

// BcryptPool manages bounded concurrency for bcrypt operations
type BcryptPool struct {
	config      PoolConfig
	hashJobs    chan hashJob
	checkJobs   chan checkJob
	wg          sync.WaitGroup
	closed      atomic.Bool
	shutdownCtx context.Context
	shutdown    context.CancelFunc

	// Metrics (atomic for thread-safety)
	hashCount    atomic.Int64
	checkCount   atomic.Int64
	timeoutCount atomic.Int64
	queuedCount  atomic.Int64
}

// NewBcryptPool creates a new bounded bcrypt worker pool
func NewBcryptPool(config PoolConfig) *BcryptPool {
	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = config.Workers * 10
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 5 * time.Second
	}
	if config.Cost <= 0 {
		config.Cost = bcryptCost
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &BcryptPool{
		config:      config,
		hashJobs:    make(chan hashJob, config.QueueSize),
		checkJobs:   make(chan checkJob, config.QueueSize),
		shutdownCtx: ctx,
		shutdown:    cancel,
	}

	// Start workers
	for i := 0; i < config.Workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker processes bcrypt jobs from both hash and check queues
func (p *BcryptPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.shutdownCtx.Done():
			return

		case job, ok := <-p.hashJobs:
			if !ok {
				return
			}
			p.queuedCount.Add(-1)
			hash, err := bcrypt.GenerateFromPassword([]byte(job.password), p.config.Cost)
			select {
			case job.result <- hashResult{hash: string(hash), err: err}:
			default:
				// Result channel closed or blocked - job was likely cancelled
			}
			p.hashCount.Add(1)

		case job, ok := <-p.checkJobs:
			if !ok {
				return
			}
			p.queuedCount.Add(-1)
			err := bcrypt.CompareHashAndPassword([]byte(job.hash), []byte(job.password))
			select {
			case job.result <- err:
			default:
				// Result channel closed or blocked - job was likely cancelled
			}
			p.checkCount.Add(1)
		}
	}
}

// HashPassword hashes a password using the worker pool
func (p *BcryptPool) HashPassword(ctx context.Context, password string) (string, error) {
	if p.closed.Load() {
		return "", ErrPoolClosed
	}

	// Validate password first (fast path)
	if err := ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	job := hashJob{
		password: password,
		result:   make(chan hashResult, 1),
	}

	// Apply timeout from context or default
	timeout := p.config.DefaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to submit job - check shutdown to avoid sending to closed channel
	select {
	case <-p.shutdownCtx.Done():
		return "", ErrPoolClosed
	case p.hashJobs <- job:
		p.queuedCount.Add(1)
	default:
		return "", ErrQueueFull
	}

	// Wait for result or timeout
	select {
	case result := <-job.result:
		return result.hash, result.err
	case <-timeoutCtx.Done():
		p.timeoutCount.Add(1)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrPoolTimeout
	}
}

// CheckPassword verifies a password against a hash using the worker pool
func (p *BcryptPool) CheckPassword(ctx context.Context, password, hash string) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	job := checkJob{
		password: password,
		hash:     hash,
		result:   make(chan error, 1),
	}

	// Apply timeout from context or default
	timeout := p.config.DefaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to submit job - check shutdown to avoid sending to closed channel
	select {
	case <-p.shutdownCtx.Done():
		return ErrPoolClosed
	case p.checkJobs <- job:
		p.queuedCount.Add(1)
	default:
		return ErrQueueFull
	}

	// Wait for result or timeout
	select {
	case err := <-job.result:
		if err != nil {
			return ErrPasswordMismatch
		}
		return nil
	case <-timeoutCtx.Done():
		p.timeoutCount.Add(1)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrPoolTimeout
	}
}

// Close shuts down the worker pool gracefully
func (p *BcryptPool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	p.shutdown()
	close(p.hashJobs)
	close(p.checkJobs)
	p.wg.Wait()

	return nil
}

// Stats returns current pool statistics
type PoolStats struct {
	Workers      int
	QueueSize    int
	Queued       int64
	HashCount    int64
	CheckCount   int64
	TimeoutCount int64
}

// Stats returns current pool statistics
func (p *BcryptPool) Stats() PoolStats {
	return PoolStats{
		Workers:      p.config.Workers,
		QueueSize:    p.config.QueueSize,
		Queued:       p.queuedCount.Load(),
		HashCount:    p.hashCount.Load(),
		CheckCount:   p.checkCount.Load(),
		TimeoutCount: p.timeoutCount.Load(),
	}
}

// Global pool instance (lazily initialized)
var (
	globalPool   *BcryptPool
	globalPoolMu sync.RWMutex
)

// GlobalPool returns the global bcrypt worker pool (creates if needed)
func GlobalPool() *BcryptPool {
	globalPoolMu.RLock()
	pool := globalPool
	globalPoolMu.RUnlock()

	// Fast path: pool exists and is not closed
	if pool != nil && !pool.closed.Load() {
		return pool
	}

	// Slow path: need to create or recreate pool
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	// Double-check after acquiring write lock
	if globalPool != nil && !globalPool.closed.Load() {
		return globalPool
	}

	// Close old pool if it exists (might be closed already, that's ok)
	if globalPool != nil {
		globalPool.Close()
	}

	// Create new pool
	globalPool = NewBcryptPool(DefaultPoolConfig())
	return globalPool
}

// SetGlobalPool replaces the global pool (useful for testing)
// Pass nil to clear the pool without replacement.
func SetGlobalPool(pool *BcryptPool) {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()
	if globalPool != nil && globalPool != pool {
		globalPool.Close()
	}
	globalPool = pool
}

// HashPasswordPooled hashes a password using the global pool
func HashPasswordPooled(ctx context.Context, password string) (string, error) {
	return GlobalPool().HashPassword(ctx, password)
}

// CheckPasswordPooled verifies a password using the global pool
func CheckPasswordPooled(ctx context.Context, password, hash string) error {
	return GlobalPool().CheckPassword(ctx, password, hash)
}
