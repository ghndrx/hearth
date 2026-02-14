package services

import (
	"time"
)

// MessageType defines how the service distinguishes between different events
type MessageType string

const (
	MsgJoin      MessageType = "JOIN"
	MsgLeave     MessageType = "LEAVE"
	MsgSendMessage MessageType = "MSG"
)

// Message represents the data payload sent between the Svelte client and the Go backend.
type Message struct {
	ID        string                 `json:"id"`
	Type      MessageType            `json:"type"`
	Content   string                 `json:"content,omitempty"`
	Username  string                 `json:"username,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EditableMessage is returned by the service to handle message updates
type EditableMessage struct {
	Type      MessageType            `json:"type"`
	OldContent string                 `json:"old_content,omitempty"`
	NewContent string                 `json:"new_content,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChannelService defines the behaviors expected by the frontend client.
// It abstracts the Go connection logic from the Svelte JavaScript logic.
type ChannelService interface {
	// Connect establishes a bidirectional communication channel.
	// In a real architecture, this would return a WebSocket connection.
	Connect(username string) (chan Message, error)

	// Close terminates the connection to the backend.
	Close() error
}

// DefaultChannelService is the standard implementation connecting to the backend.
type DefaultChannelService struct {
	host     string
	username string
	conn     chan Message
}

// NewChannelService creates a client that can talk to the main event loop.
func NewChannelService(host string, username string) ChannelService {
	// Note: In a real implementation, here we would initialize a net.Dialer
	// to the host passed in and start a goroutine to read from the websocket/event bus.
	
	return &DefaultChannelService{
		host:     host,
		username: username,
		conn:     make(chan Message, 100), // Buffered channel for performance
	}
}

// Connect mimics establishing the connection
func (cs *DefaultChannelService) Connect(username string) (chan Message, error) {
	if cs == nil {
		return nil, ErrServiceUnavailable
	}
	return cs.conn, nil
}

// Close mimics shutting down the connection
func (cs *DefaultChannelService) Close() error {
	close(cs.conn)
	return nil
}

// ErrServiceUnavailable is returned if initialization fails
var ErrServiceUnavailable = &ServiceError{
	Code:    "503",
	Message: "Hearth Node is unavailable",
}

// ServiceError captures API failures
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}