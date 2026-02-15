package services

import (
	"context"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Latency   time.Duration `json:"latency"`
}

// HealthReport represents the overall health report
type HealthReport struct {
	Status    HealthStatus  `json:"status"`
	Checks    []HealthCheck `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker defines a health check function
type HealthChecker func(ctx context.Context) HealthCheck

// HealthService manages health checks for the application
type HealthService struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{
		checkers: make(map[string]HealthChecker),
	}
}

// RegisterChecker registers a health check with a name
func (s *HealthService) RegisterChecker(name string, checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

// UnregisterChecker removes a registered health check
func (s *HealthService) UnregisterChecker(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.checkers, name)
}

// CheckHealth runs all registered health checks and returns a report
func (s *HealthService) CheckHealth(ctx context.Context) (*HealthReport, error) {
	s.mu.RLock()
	checkers := make(map[string]HealthChecker, len(s.checkers))
	for k, v := range s.checkers {
		checkers[k] = v
	}
	s.mu.RUnlock()

	report := &HealthReport{
		Status:    HealthStatusHealthy,
		Checks:    make([]HealthCheck, 0, len(checkers)),
		Timestamp: time.Now(),
	}

	for name, checker := range checkers {
		start := time.Now()
		check := checker(ctx)
		check.Latency = time.Since(start)
		check.Name = name
		check.Timestamp = time.Now()

		report.Checks = append(report.Checks, check)

		if check.Status == HealthStatusUnhealthy {
			report.Status = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && report.Status != HealthStatusUnhealthy {
			report.Status = HealthStatusDegraded
		}
	}

	return report, nil
}

// GetCheckers returns the number of registered checkers
func (s *HealthService) GetCheckers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.checkers)
}
