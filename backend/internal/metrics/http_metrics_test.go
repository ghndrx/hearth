package metrics

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal path", "/api/v1/users", "/api/v1/users"},
		{"path with param", "/api/v1/users/:id", "/api/v1/users/:id"},
		{"empty path", "", "unknown"},
		{"health path", "/health", "/health"},
		{"nested param", "/api/v1/servers/:id/channels", "/api/v1/servers/:id/channels"},
		{"UUID segment replaced", "/api/v1/users/550e8400-e29b-41d4-a716-446655440000/messages", "/api/v1/users/:id/messages"},
		{"multiple UUIDs", "/api/550e8400-e29b-41d4-a716-446655440000/items/660e8400-e29b-41d4-a716-446655440001", "/api/:id/items/:id"},
		{"root path", "/", "/"},
		{"non-UUID 36 char", "/api/v1/abcdefghijklmnopqrstuvwxyz1234567890", "/api/v1/abcdefghijklmnopqrstuvwxyz1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHTTPMetrics_Singleton(t *testing.T) {
	m1 := GetHTTPMetrics()
	require.NotNil(t, m1)
	m2 := GetHTTPMetrics()
	assert.Same(t, m1, m2, "GetHTTPMetrics should return the same instance")
}

func TestHTTPMetrics_AllFieldsInitialized(t *testing.T) {
	m := GetHTTPMetrics()
	assert.NotNil(t, m.RequestsTotal)
	assert.NotNil(t, m.RequestDurationSeconds)
	assert.NotNil(t, m.RequestsInFlight)
	assert.NotEmpty(t, m.instance)
}

func TestHTTPMetrics_RequestsInFlight(t *testing.T) {
	m := GetHTTPMetrics()

	before := testutil.ToFloat64(m.RequestsInFlight)
	m.RequestsInFlight.Inc()
	assert.Equal(t, before+1, testutil.ToFloat64(m.RequestsInFlight))

	m.RequestsInFlight.Dec()
	assert.Equal(t, before, testutil.ToFloat64(m.RequestsInFlight))
}

func TestHTTPMetrics_RequestsTotal(t *testing.T) {
	m := GetHTTPMetrics()
	method := "GET"
	path := "/test-counter"
	status := "200"

	before := testutil.ToFloat64(m.RequestsTotal.WithLabelValues(method, path, status))
	m.RequestsTotal.WithLabelValues(method, path, status).Inc()
	after := testutil.ToFloat64(m.RequestsTotal.WithLabelValues(method, path, status))

	assert.Equal(t, before+1, after)
}

func TestHTTPMetrics_RequestDurationSeconds(t *testing.T) {
	m := GetHTTPMetrics()

	require.NotPanics(t, func() {
		m.RequestDurationSeconds.WithLabelValues("POST", "/test-duration", "201").Observe(0.05)
		m.RequestDurationSeconds.WithLabelValues("GET", "/test-duration", "200").Observe(0.001)
		m.RequestDurationSeconds.WithLabelValues("GET", "/test-duration", "500").Observe(5.0)
	})
}

func TestHTTPMetrics_Middleware_NotNil(t *testing.T) {
	m := GetHTTPMetrics()
	handler := m.Middleware()
	assert.NotNil(t, handler, "Middleware should return a non-nil handler")
}

func TestHTTPMetrics_Middleware_RecordsMetrics(t *testing.T) {
	m := GetHTTPMetrics()

	app := fiber.New()
	app.Use(m.Middleware())
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	beforeTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	req, _ := http.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))

	afterTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/api/test", "200"))
	assert.Equal(t, beforeTotal+1, afterTotal)
}

func TestHTTPMetrics_Middleware_SkipsMetricsEndpoint(t *testing.T) {
	m := GetHTTPMetrics()

	app := fiber.New()
	app.Use(m.Middleware())
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendString("metrics")
	})

	beforeTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/metrics", "200"))

	req, _ := http.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	afterTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/metrics", "200"))
	assert.Equal(t, beforeTotal, afterTotal, "metrics endpoint should be skipped")
}

func TestHTTPMetrics_Middleware_Records500(t *testing.T) {
	m := GetHTTPMetrics()

	app := fiber.New()
	app.Use(m.Middleware())
	app.Get("/api/error", func(c *fiber.Ctx) error {
		return c.Status(500).SendString("error")
	})

	beforeTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/api/error", "500"))

	req, _ := http.NewRequest("GET", "/api/error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	afterTotal := testutil.ToFloat64(m.RequestsTotal.WithLabelValues("GET", "/api/error", "500"))
	assert.Equal(t, beforeTotal+1, afterTotal)
}
