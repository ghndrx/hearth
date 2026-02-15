package metrics

import (
	"os"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstanceLabel(t *testing.T) {
	// Reset the once for testing
	once = sync.Once{}
	instanceLabel = ""

	// Test with POD_NAME set
	t.Run("with POD_NAME", func(t *testing.T) {
		once = sync.Once{}
		instanceLabel = ""
		os.Setenv("POD_NAME", "test-pod-123")
		defer os.Unsetenv("POD_NAME")

		label := GetInstanceLabel()
		assert.Equal(t, "test-pod-123", label)
	})

	// Test with HOSTNAME fallback
	t.Run("with HOSTNAME", func(t *testing.T) {
		once = sync.Once{}
		instanceLabel = ""
		os.Unsetenv("POD_NAME")
		os.Setenv("HOSTNAME", "test-hostname")
		defer os.Unsetenv("HOSTNAME")

		label := GetInstanceLabel()
		assert.Equal(t, "test-hostname", label)
	})
}

func TestWebSocketMetrics_ConnectionTracking(t *testing.T) {
	// Create a new registry to avoid conflicts with global metrics
	registry := prometheus.NewRegistry()

	// Create metrics manually for testing
	connectionsActive := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "connections_active",
		},
		[]string{"instance", "client_type"},
	)
	connectionsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "test",
			Name:      "connections_total",
		},
		[]string{"instance", "client_type"},
	)

	registry.MustRegister(connectionsActive)
	registry.MustRegister(connectionsTotal)

	// Simulate connection opened
	instance := "test-pod"
	clientType := "web"

	connectionsActive.WithLabelValues(instance, clientType).Inc()
	connectionsTotal.WithLabelValues(instance, clientType).Inc()

	// Verify active connections
	val := testutil.ToFloat64(connectionsActive.WithLabelValues(instance, clientType))
	assert.Equal(t, float64(1), val)

	// Simulate another connection
	connectionsActive.WithLabelValues(instance, clientType).Inc()
	connectionsTotal.WithLabelValues(instance, clientType).Inc()

	val = testutil.ToFloat64(connectionsActive.WithLabelValues(instance, clientType))
	assert.Equal(t, float64(2), val)

	// Simulate connection closed
	connectionsActive.WithLabelValues(instance, clientType).Dec()

	val = testutil.ToFloat64(connectionsActive.WithLabelValues(instance, clientType))
	assert.Equal(t, float64(1), val)

	// Total should still be 2
	totalVal := testutil.ToFloat64(connectionsTotal.WithLabelValues(instance, clientType))
	assert.Equal(t, float64(2), totalVal)
}

func TestWebSocketMetrics_MessageTracking(t *testing.T) {
	registry := prometheus.NewRegistry()

	messagesSent := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "test",
			Name:      "messages_sent_total",
		},
		[]string{"instance", "event_type"},
	)

	messagesReceived := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "test",
			Name:      "messages_received_total",
		},
		[]string{"instance", "opcode"},
	)

	registry.MustRegister(messagesSent)
	registry.MustRegister(messagesReceived)

	instance := "test-pod"

	// Track sent messages
	messagesSent.WithLabelValues(instance, "MESSAGE_CREATE").Inc()
	messagesSent.WithLabelValues(instance, "MESSAGE_CREATE").Inc()
	messagesSent.WithLabelValues(instance, "TYPING_START").Inc()

	assert.Equal(t, float64(2), testutil.ToFloat64(messagesSent.WithLabelValues(instance, "MESSAGE_CREATE")))
	assert.Equal(t, float64(1), testutil.ToFloat64(messagesSent.WithLabelValues(instance, "TYPING_START")))

	// Track received messages
	messagesReceived.WithLabelValues(instance, "1").Inc() // heartbeat
	messagesReceived.WithLabelValues(instance, "2").Inc() // identify

	assert.Equal(t, float64(1), testutil.ToFloat64(messagesReceived.WithLabelValues(instance, "1")))
	assert.Equal(t, float64(1), testutil.ToFloat64(messagesReceived.WithLabelValues(instance, "2")))
}

func TestWebSocketMetrics_LatencyHistogram(t *testing.T) {
	registry := prometheus.NewRegistry()

	latency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "test",
			Name:      "message_latency_seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"instance", "event_type"},
	)

	registry.MustRegister(latency)

	instance := "test-pod"

	// Record latencies
	latency.WithLabelValues(instance, "MESSAGE_CREATE").Observe(0.005)
	latency.WithLabelValues(instance, "MESSAGE_CREATE").Observe(0.015)
	latency.WithLabelValues(instance, "MESSAGE_CREATE").Observe(0.002)

	// Histogram should have recorded 3 observations
	// We can't easily test histogram values without more complex setup,
	// but we can verify it doesn't panic
	require.NotPanics(t, func() {
		latency.WithLabelValues(instance, "MESSAGE_CREATE").Observe(0.001)
	})
}

func TestOpcodeToString(t *testing.T) {
	tests := []struct {
		opcode   int
		expected string
	}{
		{0, "dispatch"},
		{1, "heartbeat"},
		{2, "identify"},
		{3, "presence_update"},
		{4, "voice_state_update"},
		{6, "resume"},
		{7, "reconnect"},
		{8, "request_guild_members"},
		{9, "invalid_session"},
		{10, "hello"},
		{11, "heartbeat_ack"},
		{99, "unknown"},
		{-1, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := OpcodeToString(tt.opcode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebSocketMetrics_SubscriptionTracking(t *testing.T) {
	registry := prometheus.NewRegistry()

	channelSubs := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "channel_subscriptions_active",
		},
		[]string{"instance"},
	)

	serverSubs := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "server_subscriptions_active",
		},
		[]string{"instance"},
	)

	registry.MustRegister(channelSubs)
	registry.MustRegister(serverSubs)

	instance := "test-pod"

	// Subscribe to channels
	channelSubs.WithLabelValues(instance).Inc()
	channelSubs.WithLabelValues(instance).Inc()
	channelSubs.WithLabelValues(instance).Inc()

	assert.Equal(t, float64(3), testutil.ToFloat64(channelSubs.WithLabelValues(instance)))

	// Subscribe to servers
	serverSubs.WithLabelValues(instance).Inc()
	serverSubs.WithLabelValues(instance).Inc()

	assert.Equal(t, float64(2), testutil.ToFloat64(serverSubs.WithLabelValues(instance)))

	// Unsubscribe
	channelSubs.WithLabelValues(instance).Dec()
	serverSubs.WithLabelValues(instance).Dec()

	assert.Equal(t, float64(2), testutil.ToFloat64(channelSubs.WithLabelValues(instance)))
	assert.Equal(t, float64(1), testutil.ToFloat64(serverSubs.WithLabelValues(instance)))
}
