package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// ============================================================================
// Session Management Service Tests (Repository Pattern)
// ============================================================================

// MockWebSocketRepository is a mock implementation of WebSocketRepository for testing.
type MockWebSocketRepository struct {
	mock.Mock
}

func (m *MockWebSocketRepository) SaveActiveSession(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockWebSocketRepository) RemoveActiveSession(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockWebSocketRepository) FindActiveSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func TestNewWebSocketService(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

func TestConnectSession_Success(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	connID := uuid.New()

	// Mock expectations
	mockRepo.On("SaveActiveSession", ctx, mock.AnythingOfType("*models.Session")).Return(nil)

	session, err := service.ConnectSession(ctx, userID, connID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, userID, session.UserID)
	mockRepo.AssertExpectations(t)
}

func TestConnectSession_InvalidInput(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	ctx := context.Background()

	_, err := service.ConnectSession(ctx, uuid.Nil, uuid.New())
	assert.Error(t, err)

	_, err = service.ConnectSession(ctx, uuid.New(), uuid.Nil)
	assert.Error(t, err)
}

func TestConnectSession_RepositoryError(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	ctx := context.Background()
	userID := uuid.New()
	connID := uuid.New()

	dbErr := errors.New("database connection failed")
	mockRepo.On("SaveActiveSession", ctx, mock.AnythingOfType("*models.Session")).Return(dbErr)

	session, err := service.ConnectSession(ctx, userID, connID)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "failed to establish session")
	mockRepo.AssertExpectations(t)
}

func TestDisconnectSession_Success(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	ctx := context.Background()
	sessionID := uuid.New()
	dummySession := &models.Session{ID: sessionID}

	// Mock expectations
	mockRepo.On("FindActiveSession", ctx, sessionID).Return(dummySession, nil)
	mockRepo.On("RemoveActiveSession", ctx, sessionID).Return(nil)

	err := service.DisconnectSession(ctx, sessionID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDisconnectSession_SessionNotFound(t *testing.T) {
	mockRepo := new(MockWebSocketRepository)
	service := NewWebSocketService(mockRepo)

	ctx := context.Background()
	sessionID := uuid.New()

	// Mock expectations
	mockRepo.On("FindActiveSession", ctx, sessionID).Return(nil, errors.New("not found"))

	err := service.DisconnectSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Real-time WebSocket Hub Tests
// ============================================================================

// MockHandler implements WSMessageHandler to verify logic is called correctly.
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Handler(conn *Conn, msg *ClientMessage) {
	m.Called(conn, msg)
}

func TestCreateHub(t *testing.T) {
	mockHandler := new(MockHandler)
	hub := CreateHub(mockHandler.Handler)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.Broadcast)
	assert.NotNil(t, hub.Register)
	assert.NotNil(t, hub.Unregister)
}

func TestHub_RegisterAndUnregister(t *testing.T) {
	mockHandler := new(MockHandler)
	hub := CreateHub(mockHandler.Handler)

	go hub.Run()
	defer func() { hub.Broadcast <- nil }() // Graceful shutdown

	user := &User{ID: "test-user", Email: "test@example.com"}
	conn := &Conn{
		hub:  hub,
		User: user,
		Send: make(chan *ServerMessage, 256),
	}

	// Register
	hub.Register <- conn
	time.Sleep(10 * time.Millisecond) // Allow goroutine to process

	hub.mu.RLock()
	_, exists := hub.clients[conn]
	hub.mu.RUnlock()
	assert.True(t, exists, "Connection should be registered")

	// Unregister
	hub.Unregister <- conn
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	_, exists = hub.clients[conn]
	hub.mu.RUnlock()
	assert.False(t, exists, "Connection should be unregistered")
}

func TestHub_Broadcast(t *testing.T) {
	mockHandler := new(MockHandler)
	hub := CreateHub(mockHandler.Handler)

	go hub.Run()
	defer func() { hub.Broadcast <- nil }()

	user1 := &User{ID: "user1", Email: "user1@example.com"}
	conn1 := &Conn{
		hub:  hub,
		User: user1,
		Send: make(chan *ServerMessage, 256),
	}

	user2 := &User{ID: "user2", Email: "user2@example.com"}
	conn2 := &Conn{
		hub:  hub,
		User: user2,
		Send: make(chan *ServerMessage, 256),
	}

	hub.Register <- conn1
	hub.Register <- conn2
	time.Sleep(10 * time.Millisecond)

	// Broadcast to all
	msg := &ServerMessage{
		Type:    MessageCreate,
		Content: "Hello everyone",
		To:      "",
	}
	hub.Broadcast <- msg
	time.Sleep(10 * time.Millisecond)

	// Both should receive
	select {
	case received := <-conn1.Send:
		assert.Equal(t, "Hello everyone", received.Content)
	default:
		t.Error("conn1 should have received the message")
	}

	select {
	case received := <-conn2.Send:
		assert.Equal(t, "Hello everyone", received.Content)
	default:
		t.Error("conn2 should have received the message")
	}
}

func TestHub_DirectMessage(t *testing.T) {
	mockHandler := new(MockHandler)
	hub := CreateHub(mockHandler.Handler)

	go hub.Run()
	defer func() { hub.Broadcast <- nil }()

	user1 := &User{ID: "user1", Email: "user1@example.com"}
	conn1 := &Conn{
		hub:  hub,
		User: user1,
		Send: make(chan *ServerMessage, 256),
	}

	user2 := &User{ID: "user2", Email: "user2@example.com"}
	conn2 := &Conn{
		hub:  hub,
		User: user2,
		Send: make(chan *ServerMessage, 256),
	}

	hub.Register <- conn1
	hub.Register <- conn2
	time.Sleep(10 * time.Millisecond)

	// Send direct message to user2 only
	msg := &ServerMessage{
		Type:    MessageCreate,
		Content: "Private message",
		To:      "user2",
	}
	hub.Broadcast <- msg
	time.Sleep(10 * time.Millisecond)

	// Only user2 should receive
	select {
	case <-conn1.Send:
		t.Error("conn1 should NOT have received the direct message")
	default:
		// Expected
	}

	select {
	case received := <-conn2.Send:
		assert.Equal(t, "Private message", received.Content)
	default:
		t.Error("conn2 should have received the direct message")
	}
}

func TestWebSocket_ConnectionLifecycle(t *testing.T) {
	mockHandler := new(MockHandler)
	hub := CreateHub(mockHandler.Handler)

	go hub.Run()
	defer func() { hub.Broadcast <- nil }()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=test123&email=test@example.com"

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	// Give hub time to register
	time.Sleep(50 * time.Millisecond)

	// Verify client is registered
	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()
	assert.Equal(t, 1, clientCount, "One client should be connected")

	// Send a message from client
	clientMsg := ClientMessage{
		Type:    MessageCreate,
		Content: json.RawMessage(`{"text": "Hello World"}`),
		Author:  "test123",
	}

	mockHandler.On("Handler", mock.Anything, mock.Anything).Return()

	err = ws.WriteJSON(clientMsg)
	assert.NoError(t, err)

	// Give time for message processing
	time.Sleep(50 * time.Millisecond)

	// Close connection
	ws.Close()
	time.Sleep(50 * time.Millisecond)

	// Verify client is unregistered
	hub.mu.RLock()
	clientCount = len(hub.clients)
	hub.mu.RUnlock()
	assert.Equal(t, 0, clientCount, "No clients should be connected after close")
}

func TestClientMessage_JSONMarshaling(t *testing.T) {
	msg := ClientMessage{
		Type:    MessageCreate,
		Content: json.RawMessage(`{"text": "test"}`),
		Author:  "user123",
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded ClientMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Author, decoded.Author)
}

func TestServerMessage_JSONMarshaling(t *testing.T) {
	msg := ServerMessage{
		Type:    MessageTyping,
		Content: map[string]string{"user": "typing..."},
		To:      "recipient123",
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded ServerMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.To, decoded.To)
}
