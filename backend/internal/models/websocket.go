package models

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

// WebSocketConnection represents an active WebSocket connection for a user.
type WebSocketConnection struct {
	// Conn is the underlying WebSocket connection.
	Conn *websocket.Conn

	// Out is the channel for outgoing messages to be written to the client.
	Out chan []byte

	// Done signals when the connection should be terminated.
	Done chan struct{}
}

// Send queues a message to be sent to the client.
func (c *WebSocketConnection) Send(data json.RawMessage) {
	select {
	case c.Out <- []byte(data):
	default:
		// Channel full or closed, drop message
	}
}

// Close terminates the WebSocket connection.
func (c *WebSocketConnection) Close() error {
	close(c.Done)
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// NewWebSocketConnection creates a new WebSocketConnection wrapper.
func NewWebSocketConnection(conn *websocket.Conn) *WebSocketConnection {
	return &WebSocketConnection{
		Conn: conn,
		Out:  make(chan []byte, 256),
		Done: make(chan struct{}),
	}
}

// WebSocketMessage represents a message sent over WebSocket.
type WebSocketMessage struct {
	// Type is the message type identifier (e.g., "HEARTBEAT", "MESSAGE_CREATE").
	Type string `json:"type"`

	// Payload contains the message data, structure depends on Type.
	Payload json.RawMessage `json:"payload,omitempty"`

	// Sequence is the sequence number for ordered delivery.
	Sequence int64 `json:"seq,omitempty"`

	// Timestamp is when the message was created.
	Timestamp int64 `json:"ts,omitempty"`
}

// WebSocketEvent types
const (
	WSEventHeartbeat    = "HEARTBEAT"
	WSEventHeartbeatAck = "HEARTBEAT_ACK"
	WSEventIdentify     = "IDENTIFY"
	WSEventReady        = "READY"
	WSEventResume       = "RESUME"
	WSEventResumed      = "RESUMED"
	WSEventReconnect    = "RECONNECT"
	WSEventInvalidate   = "INVALIDATE_SESSION"
	WSEventHello        = "HELLO"

	// Channel events
	WSEventChannelCreate = "CHANNEL_CREATE"
	WSEventChannelUpdate = "CHANNEL_UPDATE"
	WSEventChannelDelete = "CHANNEL_DELETE"
	WSEventJoinChannel   = "JOIN_CHANNEL"
	WSEventChannelJoined = "CHANNEL_JOINED"

	// Message events
	WSEventMessageCreate = "MESSAGE_CREATE"
	WSEventMessageUpdate = "MESSAGE_UPDATE"
	WSEventMessageDelete = "MESSAGE_DELETE"

	// Presence events
	WSEventPresenceUpdate = "PRESENCE_UPDATE"
	WSEventTypingStart    = "TYPING_START"

	// Server events
	WSEventServerCreate       = "SERVER_CREATE"
	WSEventServerUpdate       = "SERVER_UPDATE"
	WSEventServerDelete       = "SERVER_DELETE"
	WSEventServerMemberAdd    = "SERVER_MEMBER_ADD"
	WSEventServerMemberRemove = "SERVER_MEMBER_REMOVE"
	WSEventServerMemberUpdate = "SERVER_MEMBER_UPDATE"

	// FEAT-001: Thread events
	WSEventThreadCreate          = "THREAD_CREATE"
	WSEventThreadUpdate          = "THREAD_UPDATE"
	WSEventThreadDelete          = "THREAD_DELETE"
	WSEventThreadPresenceUpdate  = "THREAD_PRESENCE_UPDATE"
	WSEventThreadMessageCreate   = "THREAD_MESSAGE_CREATE"
)
