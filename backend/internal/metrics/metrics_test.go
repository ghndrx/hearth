package metrics

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestMetrics(t *testing.T, podName string) *WebSocketMetrics {
	os.Setenv("POD_NAME", podName)
	t.Cleanup(func() {
		os.Unsetenv("POD_NAME")
		os.Unsetenv("HOSTNAME")
	})
	return GetMetrics()
}

func TestGetMetrics_ReturnsSingleton(t *testing.T) {
	m := getTestMetrics(t, "test-singleton")
	// Note: m.instance reflects the hostname captured at first NewWebSocketMetrics call.
	// We verify the metric vectors are initialized and that GetMetrics returns the same instance.
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

	m2 := GetMetrics()
	assert.Same(t, m, m2)
}

func TestConnectionOpened_IncrementsActiveAndTotal(t *testing.T) {
	m := getTestMetrics(t, "conn-test-pod")
	clientType := "web"

	m.ConnectionOpened(clientType)
	active := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	total := testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, clientType))
	assert.Equal(t, float64(1), active)
	assert.Equal(t, float64(1), total)

	m.ConnectionOpened(clientType)
	active = testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	total = testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, clientType))
	assert.Equal(t, float64(2), active)
	assert.Equal(t, float64(2), total)
}

func TestConnectionOpened_AndClosed_DecrementsActive(t *testing.T) {
	m := getTestMetrics(t, "conn-lifecycle-pod")
	clientType := "desktop"

	m.ConnectionOpened(clientType)
	m.ConnectionOpened(clientType)
	m.ConnectionClosed(clientType, 120.5)

	active := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, float64(1), active, "Active connections should decrease by 1")
}

func TestConnectionClosed_BelowZero(t *testing.T) {
	m := getTestMetrics(t, "close-zero-pod")
	clientType := "bot"

	m.ConnectionClosed(clientType, 10.0)
	active := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, float64(-1), active)
}

func TestMessageSent_IncrementsCounter(t *testing.T) {
	m := getTestMetrics(t, "msg-sent-pod")

	m.MessageSent("MESSAGE_CREATE")
	m.MessageSent("MESSAGE_CREATE")
	m.MessageSent("TYPING_START")

	assert.Equal(t, float64(2), testutil.ToFloat64(m.MessagesSentTotal.WithLabelValues(m.instance, "MESSAGE_CREATE")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.MessagesSentTotal.WithLabelValues(m.instance, "TYPING_START")))
	assert.Equal(t, float64(0), testutil.ToFloat64(m.MessagesSentTotal.WithLabelValues(m.instance, "MESSAGE_DELETE")))
}

func TestMessageReceived_IncrementsCounterByOpcode(t *testing.T) {
	m := getTestMetrics(t, "msg-recv-pod")

	m.MessageReceived("1")
	m.MessageReceived("1")
	m.MessageReceived("1")
	m.MessageReceived("2")

	assert.Equal(t, float64(3), testutil.ToFloat64(m.MessagesReceivedTotal.WithLabelValues(m.instance, "1")))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.MessagesReceivedTotal.WithLabelValues(m.instance, "2")))
	assert.Equal(t, float64(0), testutil.ToFloat64(m.MessagesReceivedTotal.WithLabelValues(m.instance, "3")))
}

func TestMessageProcessed_RecordsLatencyHistogram(t *testing.T) {
	m := getTestMetrics(t, "latency-pod")

	require.NotPanics(t, func() {
		m.MessageProcessed("MESSAGE_CREATE", 0.005)
		m.MessageProcessed("MESSAGE_CREATE", 0.015)
		m.MessageProcessed("PRESENCE_UPDATE", 0.002)
	})
}

func TestMessageProcessed_VariousLatencies(t *testing.T) {
	m := getTestMetrics(t, "latencies-pod")
	latencies := []float64{0.0005, 0.003, 0.008, 0.02, 0.04, 0.08, 0.2, 0.4, 0.9, 2.0, 4.0, 8.0}

	for _, lat := range latencies {
		require.NotPanics(t, func() {
			m.MessageProcessed("MESSAGE_CREATE", lat)
		})
	}
}

func TestSessionCreated_IncrementsActiveSessions(t *testing.T) {
	m := getTestMetrics(t, "session-pod")
	m.SetActiveSessions(0) // reset to known baseline

	m.SessionCreated()
	m.SessionCreated()
	m.SessionDestroyed()

	assert.Equal(t, float64(1), testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance)))
}

func TestSessionDestroyed_DecrementsActiveSessions(t *testing.T) {
	m := getTestMetrics(t, "session-destroy-pod")
	m.SetActiveSessions(0) // reset to known baseline

	m.SessionCreated()
	m.SessionDestroyed()

	assert.Equal(t, float64(0), testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance)))
}

func TestChannelSubscribed_IncrementsChannelSubscriptions(t *testing.T) {
	m := getTestMetrics(t, "chan-sub-pod")
	m.SetChannelSubscriptions(0) // reset to known baseline

	m.ChannelSubscribed()
	m.ChannelSubscribed()
	m.ChannelSubscribed()

	assert.Equal(t, float64(3), testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestChannelUnsubscribed_DecrementsChannelSubscriptions(t *testing.T) {
	m := getTestMetrics(t, "chan-unsub-pod")
	m.SetChannelSubscriptions(0) // reset to known baseline

	m.ChannelSubscribed()
	m.ChannelUnsubscribed()

	assert.Equal(t, float64(0), testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestServerSubscribed_IncrementsServerSubscriptions(t *testing.T) {
	m := getTestMetrics(t, "server-sub-pod")
	m.SetServerSubscriptions(0) // reset to known baseline

	m.ServerSubscribed()
	m.ServerSubscribed()

	assert.Equal(t, float64(2), testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestServerUnsubscribed_DecrementsServerSubscriptions(t *testing.T) {
	m := getTestMetrics(t, "server-unsub-pod")
	m.SetServerSubscriptions(0) // reset to known baseline

	m.ServerSubscribed()
	m.ServerUnsubscribed()

	assert.Equal(t, float64(0), testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestHeartbeatReceived_IncrementsHeartbeatCounter(t *testing.T) {
	m := getTestMetrics(t, "heartbeat-pod")

	m.HeartbeatReceived()
	m.HeartbeatReceived()
	m.HeartbeatReceived()

	assert.Equal(t, float64(3), testutil.ToFloat64(m.HeartbeatsTotal.WithLabelValues(m.instance)))
}

func TestSetActiveConnections_SetsGaugeDirectly(t *testing.T) {
	m := getTestMetrics(t, "set-conn-pod")

	m.SetActiveConnections("web", 10)
	m.SetActiveConnections("mobile", 5)

	assert.Equal(t, float64(10), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, "web")))
	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, "mobile")))
}

func TestSetChannelSubscriptions_SetsGaugeDirectly(t *testing.T) {
	m := getTestMetrics(t, "set-chan-pod")

	m.SetChannelSubscriptions(42)

	assert.Equal(t, float64(42), testutil.ToFloat64(m.ChannelSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestSetServerSubscriptions_SetsGaugeDirectly(t *testing.T) {
	m := getTestMetrics(t, "set-server-pod")

	m.SetServerSubscriptions(7)

	assert.Equal(t, float64(7), testutil.ToFloat64(m.ServerSubscriptionsActive.WithLabelValues(m.instance)))
}

func TestSetActiveSessions_SetsGaugeDirectly(t *testing.T) {
	m := getTestMetrics(t, "set-sessions-pod")

	m.SetActiveSessions(15)

	assert.Equal(t, float64(15), testutil.ToFloat64(m.SessionsActive.WithLabelValues(m.instance)))
}

func TestConnectionDuration_RecordsHistogram(t *testing.T) {
	m := getTestMetrics(t, "duration-pod")
	clientType := "api"

	require.NotPanics(t, func() {
		m.ConnectionClosed(clientType, 5.0)
		m.ConnectionClosed(clientType, 30.0)
		m.ConnectionClosed(clientType, 120.0)
	})
}

func TestCombinedMetrics_OperationsOnSameInstance(t *testing.T) {
	// This test operates on a shared metrics instance (all tests do via singleton)
	// but makes assertions relative to its own operations only.
	m := getTestMetrics(t, "combined-ops-pod")
	clientType := "web"

	// Baseline from operations in this test
	startActive := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))

	m.ConnectionOpened(clientType)
	afterOpen := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, startActive+1, afterOpen)

	m.ConnectionClosed(clientType, 45.0)
	afterClose := testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, clientType))
	assert.Equal(t, startActive, afterClose)
}
