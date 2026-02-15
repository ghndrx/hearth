package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ws "hearth/internal/websocket"
)

// mockGatewayForHealth implements the methods needed for health check tests
type mockGatewayForHealth struct {
	isHealthy            bool
	isDraining           bool
	drainState           ws.DrainState
	activeConnections    int64
}

func (m *mockGatewayForHealth) IsHealthy() bool {
	return m.isHealthy
}

func (m *mockGatewayForHealth) IsDraining() bool {
	return m.isDraining
}

func (m *mockGatewayForHealth) DrainState() ws.DrainState {
	return m.drainState
}

func (m *mockGatewayForHealth) GetActiveConnections() int64 {
	return m.activeConnections
}

func (m *mockGatewayForHealth) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": m.activeConnections,
		"draining":           m.isDraining,
		"drain_state":        m.drainState.String(),
	}
}

// setupHealthTestApp creates a test Fiber app with health endpoints
func setupHealthTestApp(gateway *mockGatewayForHealth) *fiber.App {
	app := fiber.New()

	// /health - Returns 503 when draining (for graceful shutdown)
	app.Get("/health", func(c *fiber.Ctx) error {
		if gateway == nil {
			return c.JSON(fiber.Map{
				"status":      "ok",
				"drain_state": "healthy",
			})
		}

		drainState := gateway.DrainState()
		isHealthy := gateway.IsHealthy()

		response := fiber.Map{
			"status":      "ok",
			"drain_state": drainState.String(),
			"connections": gateway.GetActiveConnections(),
		}

		if !isHealthy {
			response["status"] = "draining"
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}

		return c.JSON(response)
	})

	// /healthz - Kubernetes-style liveness probe (always returns 200 if process is alive)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"alive": true,
		})
	})

	// /readyz - Kubernetes-style readiness probe (returns 503 when draining)
	app.Get("/readyz", func(c *fiber.Ctx) error {
		if gateway == nil {
			return c.JSON(fiber.Map{
				"ready": true,
			})
		}

		if gateway.IsDraining() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"ready":       false,
				"reason":      "draining",
				"drain_state": gateway.DrainState().String(),
			})
		}

		return c.JSON(fiber.Map{
			"ready":       true,
			"connections": gateway.GetActiveConnections(),
		})
	})

	return app
}

func TestHealth_WhenHealthy(t *testing.T) {
	gateway := &mockGatewayForHealth{
		isHealthy:         true,
		isDraining:        false,
		drainState:        ws.DrainStateHealthy,
		activeConnections: 42,
	}
	app := setupHealthTestApp(gateway)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "healthy", result["drain_state"])
	assert.Equal(t, float64(42), result["connections"])
}

func TestHealth_WhenDraining(t *testing.T) {
	gateway := &mockGatewayForHealth{
		isHealthy:         false,
		isDraining:        true,
		drainState:        ws.DrainStateDraining,
		activeConnections: 10,
	}
	app := setupHealthTestApp(gateway)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "draining", result["status"])
	assert.Equal(t, "draining", result["drain_state"])
	assert.Equal(t, float64(10), result["connections"])
}

func TestHealth_WhenClosed(t *testing.T) {
	gateway := &mockGatewayForHealth{
		isHealthy:         false,
		isDraining:        false,
		drainState:        ws.DrainStateClosed,
		activeConnections: 0,
	}
	app := setupHealthTestApp(gateway)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "draining", result["status"])
	assert.Equal(t, "closed", result["drain_state"])
}

func TestHealth_NilGateway(t *testing.T) {
	app := setupHealthTestApp(nil)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "healthy", result["drain_state"])
}

func TestHealthz_Liveness_AlwaysReturns200(t *testing.T) {
	testCases := []struct {
		name    string
		gateway *mockGatewayForHealth
	}{
		{
			name: "healthy",
			gateway: &mockGatewayForHealth{
				isHealthy:  true,
				isDraining: false,
				drainState: ws.DrainStateHealthy,
			},
		},
		{
			name: "draining",
			gateway: &mockGatewayForHealth{
				isHealthy:  false,
				isDraining: true,
				drainState: ws.DrainStateDraining,
			},
		},
		{
			name: "closed",
			gateway: &mockGatewayForHealth{
				isHealthy:  false,
				isDraining: false,
				drainState: ws.DrainStateClosed,
			},
		},
		{
			name:    "nil gateway",
			gateway: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := setupHealthTestApp(tc.gateway)

			req := httptest.NewRequest("GET", "/healthz", nil)
			resp, err := app.Test(req, -1)

			require.NoError(t, err)
			// Liveness probe always returns 200
			assert.Equal(t, 200, resp.StatusCode)

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			assert.Equal(t, true, result["alive"])
		})
	}
}

func TestReadyz_WhenReady(t *testing.T) {
	gateway := &mockGatewayForHealth{
		isHealthy:         true,
		isDraining:        false,
		drainState:        ws.DrainStateHealthy,
		activeConnections: 100,
	}
	app := setupHealthTestApp(gateway)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, true, result["ready"])
	assert.Equal(t, float64(100), result["connections"])
}

func TestReadyz_WhenDraining(t *testing.T) {
	gateway := &mockGatewayForHealth{
		isHealthy:         false,
		isDraining:        true,
		drainState:        ws.DrainStateDraining,
		activeConnections: 5,
	}
	app := setupHealthTestApp(gateway)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, false, result["ready"])
	assert.Equal(t, "draining", result["reason"])
	assert.Equal(t, "draining", result["drain_state"])
}

func TestReadyz_NilGateway(t *testing.T) {
	app := setupHealthTestApp(nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, true, result["ready"])
}

// TestHealthEndpoints_KubernetesIntegration tests the endpoints work correctly
// for Kubernetes deployment scenarios
func TestHealthEndpoints_KubernetesIntegration(t *testing.T) {
	t.Run("rolling update - healthy pod", func(t *testing.T) {
		gateway := &mockGatewayForHealth{
			isHealthy:         true,
			isDraining:        false,
			drainState:        ws.DrainStateHealthy,
			activeConnections: 50,
		}
		app := setupHealthTestApp(gateway)

		// Both probes should succeed
		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, 200, resp.StatusCode, "liveness should pass")

		req = httptest.NewRequest("GET", "/readyz", nil)
		resp, _ = app.Test(req, -1)
		assert.Equal(t, 200, resp.StatusCode, "readiness should pass")

		req = httptest.NewRequest("GET", "/health", nil)
		resp, _ = app.Test(req, -1)
		assert.Equal(t, 200, resp.StatusCode, "health should pass")
	})

	t.Run("rolling update - draining pod", func(t *testing.T) {
		gateway := &mockGatewayForHealth{
			isHealthy:         false,
			isDraining:        true,
			drainState:        ws.DrainStateDraining,
			activeConnections: 10,
		}
		app := setupHealthTestApp(gateway)

		// Liveness should still pass (pod is alive)
		req := httptest.NewRequest("GET", "/healthz", nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, 200, resp.StatusCode, "liveness should pass even when draining")

		// Readiness should fail (stop sending new traffic)
		req = httptest.NewRequest("GET", "/readyz", nil)
		resp, _ = app.Test(req, -1)
		assert.Equal(t, 503, resp.StatusCode, "readiness should fail when draining")

		// Health (load balancer) should fail
		req = httptest.NewRequest("GET", "/health", nil)
		resp, _ = app.Test(req, -1)
		assert.Equal(t, 503, resp.StatusCode, "health should fail when draining")
	})
}
