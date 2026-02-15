// Package metrics provides Prometheus metrics collectors for Hearth.
package metrics

import (
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "hearth"
	subsystem = "websocket"
)

var (
	// Instance/pod identifier for distributed metrics
	instanceLabel string
	once          sync.Once
)

// GetInstanceLabel returns the instance label (pod name or hostname)
func GetInstanceLabel() string {
	once.Do(func() {
		// Try POD_NAME first (Kubernetes), then HOSTNAME, then fallback to os.Hostname()
		instanceLabel = os.Getenv("POD_NAME")
		if instanceLabel == "" {
			instanceLabel = os.Getenv("HOSTNAME")
		}
		if instanceLabel == "" {
			if hostname, err := os.Hostname(); err == nil {
				instanceLabel = hostname
			} else {
				instanceLabel = "unknown"
			}
		}
	})
	return instanceLabel
}

// WebSocketMetrics holds all WebSocket-related Prometheus metrics
type WebSocketMetrics struct {
	// ConnectionsActive tracks currently active WebSocket connections per instance
	ConnectionsActive *prometheus.GaugeVec

	// ConnectionsTotal tracks total WebSocket connections ever made
	ConnectionsTotal *prometheus.CounterVec

	// MessagesSentTotal tracks total messages sent by type
	MessagesSentTotal *prometheus.CounterVec

	// MessagesReceivedTotal tracks total messages received by opcode
	MessagesReceivedTotal *prometheus.CounterVec

	// MessageLatencySeconds tracks message processing latency
	MessageLatencySeconds *prometheus.HistogramVec

	// SessionsActive tracks active sessions
	SessionsActive *prometheus.GaugeVec

	// ChannelSubscriptionsActive tracks active channel subscriptions
	ChannelSubscriptionsActive *prometheus.GaugeVec

	// ServerSubscriptionsActive tracks active server subscriptions
	ServerSubscriptionsActive *prometheus.GaugeVec

	// HeartbeatsTotal tracks heartbeat messages
	HeartbeatsTotal *prometheus.CounterVec

	// ConnectionDuration tracks how long connections stay open
	ConnectionDuration *prometheus.HistogramVec

	// instance is the pod/instance name for labeling
	instance string
}

// globalMetrics is the singleton instance
var globalMetrics *WebSocketMetrics

// NewWebSocketMetrics creates and registers WebSocket metrics
func NewWebSocketMetrics() *WebSocketMetrics {
	instance := GetInstanceLabel()

	m := &WebSocketMetrics{
		instance: instance,

		ConnectionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "connections_active",
				Help:      "Number of currently active WebSocket connections",
			},
			[]string{"instance", "client_type"},
		),

		ConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "connections_total",
				Help:      "Total number of WebSocket connections established",
			},
			[]string{"instance", "client_type"},
		),

		MessagesSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "messages_sent_total",
				Help:      "Total number of messages sent to clients",
			},
			[]string{"instance", "event_type"},
		),

		MessagesReceivedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "messages_received_total",
				Help:      "Total number of messages received from clients",
			},
			[]string{"instance", "opcode"},
		),

		MessageLatencySeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "message_latency_seconds",
				Help:      "Message processing latency in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"instance", "event_type"},
		),

		SessionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "sessions_active",
				Help:      "Number of active sessions",
			},
			[]string{"instance"},
		),

		ChannelSubscriptionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "channel_subscriptions_active",
				Help:      "Number of active channel subscriptions",
			},
			[]string{"instance"},
		),

		ServerSubscriptionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "server_subscriptions_active",
				Help:      "Number of active server subscriptions",
			},
			[]string{"instance"},
		),

		HeartbeatsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "heartbeats_total",
				Help:      "Total number of heartbeat messages processed",
			},
			[]string{"instance"},
		),

		ConnectionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "connection_duration_seconds",
				Help:      "Duration of WebSocket connections in seconds",
				Buckets:   []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600, 7200},
			},
			[]string{"instance", "client_type"},
		),
	}

	globalMetrics = m
	return m
}

// GetMetrics returns the global metrics instance
func GetMetrics() *WebSocketMetrics {
	if globalMetrics == nil {
		return NewWebSocketMetrics()
	}
	return globalMetrics
}

// Connection tracking methods

// ConnectionOpened records a new connection
func (m *WebSocketMetrics) ConnectionOpened(clientType string) {
	m.ConnectionsActive.WithLabelValues(m.instance, clientType).Inc()
	m.ConnectionsTotal.WithLabelValues(m.instance, clientType).Inc()
}

// ConnectionClosed records a closed connection with duration
func (m *WebSocketMetrics) ConnectionClosed(clientType string, durationSeconds float64) {
	m.ConnectionsActive.WithLabelValues(m.instance, clientType).Dec()
	m.ConnectionDuration.WithLabelValues(m.instance, clientType).Observe(durationSeconds)
}

// Message tracking methods

// MessageSent records an outgoing message
func (m *WebSocketMetrics) MessageSent(eventType string) {
	m.MessagesSentTotal.WithLabelValues(m.instance, eventType).Inc()
}

// MessageReceived records an incoming message by opcode
func (m *WebSocketMetrics) MessageReceived(opcode string) {
	m.MessagesReceivedTotal.WithLabelValues(m.instance, opcode).Inc()
}

// MessageProcessed records message processing latency
func (m *WebSocketMetrics) MessageProcessed(eventType string, latencySeconds float64) {
	m.MessageLatencySeconds.WithLabelValues(m.instance, eventType).Observe(latencySeconds)
}

// Session tracking methods

// SessionCreated records a new session
func (m *WebSocketMetrics) SessionCreated() {
	m.SessionsActive.WithLabelValues(m.instance).Inc()
}

// SessionDestroyed records a destroyed session
func (m *WebSocketMetrics) SessionDestroyed() {
	m.SessionsActive.WithLabelValues(m.instance).Dec()
}

// Subscription tracking methods

// ChannelSubscribed records a channel subscription
func (m *WebSocketMetrics) ChannelSubscribed() {
	m.ChannelSubscriptionsActive.WithLabelValues(m.instance).Inc()
}

// ChannelUnsubscribed records a channel unsubscription
func (m *WebSocketMetrics) ChannelUnsubscribed() {
	m.ChannelSubscriptionsActive.WithLabelValues(m.instance).Dec()
}

// ServerSubscribed records a server subscription
func (m *WebSocketMetrics) ServerSubscribed() {
	m.ServerSubscriptionsActive.WithLabelValues(m.instance).Inc()
}

// ServerUnsubscribed records a server unsubscription
func (m *WebSocketMetrics) ServerUnsubscribed() {
	m.ServerSubscriptionsActive.WithLabelValues(m.instance).Dec()
}

// HeartbeatReceived records a heartbeat
func (m *WebSocketMetrics) HeartbeatReceived() {
	m.HeartbeatsTotal.WithLabelValues(m.instance).Inc()
}

// SetActiveConnections sets the gauge directly (for sync with hub stats)
func (m *WebSocketMetrics) SetActiveConnections(clientType string, count float64) {
	m.ConnectionsActive.WithLabelValues(m.instance, clientType).Set(count)
}

// SetChannelSubscriptions sets the channel subscription gauge directly
func (m *WebSocketMetrics) SetChannelSubscriptions(count float64) {
	m.ChannelSubscriptionsActive.WithLabelValues(m.instance).Set(count)
}

// SetServerSubscriptions sets the server subscription gauge directly
func (m *WebSocketMetrics) SetServerSubscriptions(count float64) {
	m.ServerSubscriptionsActive.WithLabelValues(m.instance).Set(count)
}

// SetActiveSessions sets the sessions gauge directly
func (m *WebSocketMetrics) SetActiveSessions(count float64) {
	m.SessionsActive.WithLabelValues(m.instance).Set(count)
}

// OpcodeToString converts an opcode to a string label
func OpcodeToString(op int) string {
	switch op {
	case 0:
		return "dispatch"
	case 1:
		return "heartbeat"
	case 2:
		return "identify"
	case 3:
		return "presence_update"
	case 4:
		return "voice_state_update"
	case 6:
		return "resume"
	case 7:
		return "reconnect"
	case 8:
		return "request_guild_members"
	case 9:
		return "invalid_session"
	case 10:
		return "hello"
	case 11:
		return "heartbeat_ack"
	default:
		return "unknown"
	}
}
