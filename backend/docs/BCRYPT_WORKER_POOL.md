# Bcrypt Worker Pool

## Overview

The Hearth backend uses a bounded worker pool for bcrypt password operations to prevent performance degradation under load.

## Problem Statement

Bcrypt is intentionally CPU-intensive (this is a security feature). However, when many users register or login concurrently:

1. **Without pooling**: Every goroutine calls `bcrypt.GenerateFromPassword()` directly
2. All goroutines compete for CPU time simultaneously
3. Context switching overhead increases exponentially
4. **Result**: p99 latency spikes from ~220ms to 3+ seconds under 100 req/s

## Solution: Bounded Worker Pool

The `BcryptPool` limits concurrent bcrypt operations to `NumCPU()` workers:

```go
// Architecture
┌─────────────────┐
│  HTTP Handlers  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  Job Queue      │────▶│  Worker Pool    │
│  (buffered)     │     │  (NumCPU size)  │
└─────────────────┘     └─────────────────┘
         │
         ▼
    Result Channel
```

### Key Benefits

1. **Bounded concurrency**: Only NumCPU workers run bcrypt simultaneously
2. **Predictable latency**: Jobs queue rather than compete for CPU
3. **Backpressure**: Queue limits prevent memory exhaustion
4. **Context support**: Operations respect request timeouts

## Configuration

```go
config := auth.PoolConfig{
    Workers:        runtime.NumCPU(), // Concurrent bcrypt operations
    QueueSize:      runtime.NumCPU() * 10, // Pending job buffer
    DefaultTimeout: 5 * time.Second,  // Per-operation timeout
    Cost:           12,               // bcrypt cost factor
}

pool := auth.NewBcryptPool(config)
```

### Sizing Guidelines

| Scenario | Workers | Queue Size | Notes |
|----------|---------|------------|-------|
| Default | NumCPU | NumCPU*10 | Optimal for CPU-bound work |
| High throughput | NumCPU | NumCPU*50 | Larger queue for burst handling |
| Limited resources | NumCPU/2 | NumCPU*5 | Save CPU for other work |

## Usage

### Global Pool (Recommended)

```go
// In application startup
bcryptPool := auth.NewBcryptPool(auth.DefaultPoolConfig())
auth.SetGlobalPool(bcryptPool)
defer bcryptPool.Close()

// In handlers/services
hash, err := auth.HashPasswordPooled(ctx, password)
err := auth.CheckPasswordPooled(ctx, password, hash)
```

### Direct Pool Usage

```go
pool := auth.NewBcryptPool(config)
defer pool.Close()

hash, err := pool.HashPassword(ctx, password)
err := pool.CheckPassword(ctx, password, hash)
```

## Error Handling

```go
hash, err := pool.HashPassword(ctx, password)
switch {
case errors.Is(err, auth.ErrPoolClosed):
    // Pool was shut down
case errors.Is(err, auth.ErrPoolTimeout):
    // Operation exceeded timeout
case errors.Is(err, auth.ErrQueueFull):
    // Backpressure - too many pending requests
case errors.Is(err, auth.ErrPasswordTooShort):
    // Validation failed (checked before queueing)
}
```

## Monitoring

```go
stats := pool.Stats()
log.Printf("Pool stats: workers=%d queued=%d hashes=%d checks=%d timeouts=%d",
    stats.Workers,
    stats.Queued,
    stats.HashCount,
    stats.CheckCount,
    stats.TimeoutCount,
)
```

### Metrics to Watch

- **Queued count**: High values indicate backpressure
- **Timeout count**: Non-zero suggests pool saturation
- **Hash/Check counts**: Throughput metrics

## Performance Results

### Load Test: 100 concurrent registrations, bcrypt cost=10

| Metric | Direct bcrypt | Pooled bcrypt | Improvement |
|--------|--------------|---------------|-------------|
| p50 latency | ~200ms | ~180ms | 1.1x |
| p95 latency | ~1.5s | ~400ms | 3.75x |
| p99 latency | ~3.3s | ~480ms | 6.9x |
| Max latency | ~5s | ~600ms | 8.3x |

### Why p99 Improves Dramatically

Direct bcrypt with 100 concurrent goroutines:
- 100 goroutines fight for ~4-8 CPU cores
- Context switching destroys cache locality
- Some goroutines starve while others run

Pooled bcrypt with 4-8 workers:
- Only NumCPU operations run simultaneously
- Others wait in queue (cheap)
- Each operation gets dedicated CPU time
- Queue acts as shock absorber

## Integration

### Service Layer

```go
// services/auth_service.go
func (s *authService) Register(ctx context.Context, email, username, password string) (*models.User, *AuthTokens, error) {
    // ...
    
    // Hash password using bounded worker pool
    hashedPassword, err := auth.HashPasswordPooled(ctx, password)
    if err != nil {
        return nil, nil, err
    }
    
    // ...
}

func (s *authService) Login(ctx context.Context, email, password string) (*models.User, *AuthTokens, error) {
    // ...
    
    // Verify password using bounded worker pool
    if err := auth.CheckPasswordPooled(ctx, password, user.PasswordHash); err != nil {
        return nil, nil, ErrInvalidCredentials
    }
    
    // ...
}
```

### Application Startup

```go
// cmd/hearth/main.go
func main() {
    // ... config loading ...

    // Initialize bcrypt worker pool
    bcryptPool := auth.NewBcryptPool(auth.DefaultPoolConfig())
    auth.SetGlobalPool(bcryptPool)
    defer bcryptPool.Close()
    
    // ... rest of initialization ...
}
```

## Testing

Run unit tests:
```bash
go test ./internal/auth/... -v
```

Run load tests:
```bash
go test ./internal/auth/... -v -run "TestLoadTest" -count=1
```

Run benchmarks:
```bash
go test ./internal/auth/... -bench=. -benchmem
```

## Future Improvements

1. **Prometheus metrics**: Export pool stats as Prometheus gauges
2. **Adaptive sizing**: Auto-tune workers based on latency
3. **Priority queues**: Prioritize login over registration
4. **Distributed pool**: Offload to dedicated hashing service

## References

- [bcrypt timing attacks](https://www.usenix.org/legacy/events/usenix99/provos/provos_html/node6.html)
- [Go runtime scheduler](https://go.dev/blog/ismmkeynote)
- [Bounded concurrency patterns](https://go.dev/blog/pipelines)
