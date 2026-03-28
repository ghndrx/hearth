package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	httpSubsystem = "http"
)

// HTTPMetrics holds all HTTP-related Prometheus metrics
type HTTPMetrics struct {
	// RequestsTotal counts HTTP requests by method, path, and status code
	RequestsTotal *prometheus.CounterVec

	// RequestDurationSeconds tracks HTTP request latency
	RequestDurationSeconds *prometheus.HistogramVec

	// RequestsInFlight tracks currently in-flight requests
	RequestsInFlight prometheus.Gauge

	instance string
}

var globalHTTPMetrics *HTTPMetrics

// NewHTTPMetrics creates and registers HTTP metrics
func NewHTTPMetrics() *HTTPMetrics {
	instance := GetInstanceLabel()

	m := &HTTPMetrics{
		instance: instance,

		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: httpSubsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests by method, path, and status code",
			},
			[]string{"method", "path", "status"},
		),

		RequestDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: httpSubsystem,
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),

		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: httpSubsystem,
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),
	}

	globalHTTPMetrics = m
	return m
}

// GetHTTPMetrics returns the global HTTP metrics instance
func GetHTTPMetrics() *HTTPMetrics {
	if globalHTTPMetrics == nil {
		return NewHTTPMetrics()
	}
	return globalHTTPMetrics
}

// Middleware returns a Fiber middleware that records HTTP metrics
func (m *HTTPMetrics) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip metrics endpoint to avoid recursion
		if c.Path() == "/metrics" {
			return c.Next()
		}

		m.RequestsInFlight.Inc()
		start := time.Now()

		err := c.Next()

		m.RequestsInFlight.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		path := normalizePath(c.Route().Path)

		m.RequestsTotal.WithLabelValues(c.Method(), path, status).Inc()
		m.RequestDurationSeconds.WithLabelValues(c.Method(), path, status).Observe(duration)

		return err
	}
}

// normalizePath reduces cardinality by replacing dynamic segments with placeholders.
func normalizePath(routePath string) string {
	if routePath == "" {
		return "unknown"
	}
	// Fiber route paths already use :param syntax, so they are naturally grouped.
	// Collapse any UUID-like segments that leak through.
	parts := strings.Split(routePath, "/")
	for i, part := range parts {
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}
