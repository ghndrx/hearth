package metrics

import (
	"os"
	"sync"
	"testing"

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

	// Test with os.Hostname fallback
	t.Run("with os.Hostname fallback", func(t *testing.T) {
		once = sync.Once{}
		instanceLabel = ""
		os.Unsetenv("POD_NAME")
		os.Unsetenv("HOSTNAME")

		label := GetInstanceLabel()
		assert.NotEmpty(t, label)
		hostname, err := os.Hostname()
		if err == nil {
			assert.Equal(t, hostname, label)
		}
	})

	// Test idempotency (sync.Once)
	t.Run("idempotent", func(t *testing.T) {
		once = sync.Once{}
		instanceLabel = ""
		os.Setenv("POD_NAME", "pod-a")
		defer os.Unsetenv("POD_NAME")

		label1 := GetInstanceLabel()
		os.Setenv("POD_NAME", "pod-b")
		label2 := GetInstanceLabel()
		assert.Equal(t, label1, label2, "second call should return cached value")
	})
}

func TestGetMetrics_Singleton(t *testing.T) {
	m1 := GetMetrics()
	require.NotNil(t, m1)
	m2 := GetMetrics()
	assert.Same(t, m1, m2, "GetMetrics should return the same instance")
}

func TestGetMetrics_AllFieldsInitialized(t *testing.T) {
	m := GetMetrics()
	assert.NotNil(t, m.ConnectionsActive)
	assert.NotNil(t, m.ConnectionsTotal)
	assert.NotNil(t, m.MessagesSentTotal)
	assert.NotNil(t, m.MessagesReceivedTotal)
	assert.NotNil(t, m.MessageLatencySeconds)
	assert.NotNil(t, m.SessionsActive)
	assert.NotNil(t, m.ChannelSubscriptionsActive)
	assert.NotNil(t, m.ServerSubscriptionsActive)
	assert.NotNil(t, m.HeartbeatsTotal)
	assert.NotNil(t, m.ConnectionDuration)
	assert.NotEmpty(t, m.instance)
}

func TestWebSocketMetrics_ConnectionOpened(t *testing.T) {
	m := GetMetrics()
	clientType := "test-conn-opened"

	before := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	beforeTotal := testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, clientType))

	m.ConnectionOpened(clientType)

	after := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	afterTotal := testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, clientType))

	assert.Equal(t, before+1, after)
	assert.Equal(t, beforeTotal+1, afterTotal)
}

func TestWebSocketMetrics_ConnectionClosed(t *testing.T) {
	m := GetMetrics()
	clientType := "test-conn-closed"

	// Open first so gauge is positive
	m.ConnectionOpened(clientType)
	before := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))

	m.ConnectionClosed(clientType, 120.5)

	after := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, before-1, after)
}

func TestWebSocketMetrics_MessageSent(t *testing.T) {
	m := GetMetrics()
	eventType := "TEST_EVENT_SENT"

	before := testutil.ToFloat64(m.MessagesSentTotal.WithLabelValues(m.instance, eventType))
	m.MessageSent(eventType)
	after := testutil.ToFloat64(m.MessagesSentTotal.WithLabelValues(m.instance, eventType))

	assert.Equal(t, before+1, after)
}

func TestWebSocketMetrics_MessageReceived(t *testing.T) {
	m := GetMetrics()
	opcode := "test_opcode"

	before := testutil.ToFloat64(m.MessagesReceivedTotal.WithLabelValues(m.instance, opcode))
	m.MessageReceived(opcode)
	after := testutil.ToFloat64(m.MessagesReceivedTotal.WithLabelValues(m.instance, opcode))

	assert.Equal(t, before+1, after)
}

func TestWebSocketMetrics_MessageProcessed(t *testing.T) {
	m := GetMetrics()
	eventType := "TEST_LATENCY"

	// Should not panic and should record observation
	require.NotPanics(t, func() {
		m.MessageProcessed(eventType, 0.005)
		m.MessageProcessed(eventType, 0.1)
		m.MessageProcessed(eventType, 0.0)
	})
}

func TestWebSocketMetrics_SessionCreatedAndDestroyed(t *testing.T) {
	m := GetMetrics()

	before := testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance))

	m.SessionCreated()
	m.SessionCreated()
	assert.Equal(t, before+2, testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance)))

	m.SessionDestroyed()
	assert.Equal(t, before+1, testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance)))
}

func TestWebSocketMetrics_ChannelSubscribeUnsubscribe(t *testing.T) {
	m := GetMetrics()

	before := testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance))

	m.ChannelSubscribed()
	m.ChannelSubscribed()
	m.ChannelSubscribed()
	assert.Equal(t, before+3, testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance)))

	m.ChannelUnsubscribed()
	assert.Equal(t, before+2, testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestWebSocketMetrics_ServerSubscribeUnsubscribe(t *testing.T) {
	m := GetMetrics()

	before := testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance))

	m.ServerSubscribed()
	m.ServerSubscribed()
	assert.Equal(t, before+2, testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance)))

	m.ServerUnsubscribed()
	assert.Equal(t, before+1, testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestWebSocketMetrics_HeartbeatReceived(t *testing.T) {
	m := GetMetrics()

	before := testutil.ToFloat64(m.HeartbeatsTotal.WithLabelValues(m.instance))
	m.HeartbeatReceived()
	m.HeartbeatReceived()
	after := testutil.ToFloat64(m.HeartbeatsTotal.WithLabelValues(m.instance))

	assert.Equal(t, before+2, after)
}

func TestWebSocketMetrics_SetActiveConnections(t *testing.T) {
	m := GetMetrics()
	clientType := "test-set-active"

	m.SetActiveConnections(clientType, 42.0)
	val := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, 42.0, val)

	m.SetActiveConnections(clientType, 0.0)
	val = testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, 0.0, val)
}

func TestWebSocketMetrics_SetChannelSubscriptions(t *testing.T) {
	m := GetMetrics()

	m.SetChannelSubscriptions(100.0)
	val := testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance))
	assert.Equal(t, 100.0, val)
}

func TestWebSocketMetrics_SetServerSubscriptions(t *testing.T) {
	m := GetMetrics()

	m.SetServerSubscriptions(50.0)
	val := testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance))
	assert.Equal(t, 50.0, val)
}

func TestWebSocketMetrics_SetActiveSessions(t *testing.T) {
	m := GetMetrics()

	m.SetActiveSessions(25.0)
	val := testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance))
	assert.Equal(t, 25.0, val)
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
		{5, "unknown"},
		{12, "unknown"},
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

func TestWebSocketMetrics_ConcurrentMethodCalls(t *testing.T) {
	m := GetMetrics()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.ConnectionOpened("concurrent")
			m.MessageSent("TEST")
			m.MessageReceived("1")
			m.HeartbeatReceived()
			m.SessionCreated()
			m.ChannelSubscribed()
			m.ServerSubscribed()
			m.MessageProcessed("TEST", 0.01)
			m.ConnectionClosed("concurrent", 1.0)
			m.SessionDestroyed()
			m.ChannelUnsubscribed()
			m.ServerUnsubscribed()
		}()
	}

	wg.Wait()
}
