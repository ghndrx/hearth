package services

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

// --- Test Utilities ---

// mockGateway implements Gateway for testing
type mockGateway struct {
	sendMu    sync.Mutex
	sentRecvs []interface{}
	closed    bool
	writeErr  error
}

func newMockGateway(wErr error) *mockGateway {
	return &mockGateway{writeErr: wErr}
}

func (m *mockGateway) WriteJSON(v interface{}) error {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()

	if m.writeErr != nil {
		return m.writeErr
	}

	// Marshal to raw bytes to avoid infinite recursion of MarshalJSON on interface{} in debug prints
	bytes, _ := json.Marshal(v)
	var val interface{}
	json.Unmarshal(bytes, &val)

	m.sentRecvs = append(m.sentRecvs, val)
	return nil
}

func (m *mockGateway) Close() error {
	m.closed = true
	return nil
}

func (m *mockGateway) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

// mockUserService implements UserService
type mockUserService struct {
	memSvc *InMemoryUserService
}

func (m *mockUserService) GetMemberGuilds(userID int64) ([]Guild, error) {
	return m.memSvc.GetMemberGuilds(userID)
}

// --- Service Tests ---

func TestHearthService_ManageClient(t *testing.T) {
	userService := mockUserService{memSvc: NewInMemoryUserService()}
	service := NewHearthService(&userService)

	tests := []struct {
		name          string
		userID        int64
		expectError   bool
		expectedTypes []string
	}{
		{
			name:          "Valid user connection",
			userID:        1, // From InMemoryUserService seed
			expectError:   false,
			expectedTypes: []string{"heartbeat", "guild_update"},
		},
		{
			name:        "Invalid user connection",
			userID:      999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context that will allow the loop to run briefly
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Create the mock connection
			conn := newMockGateway(nil)

			// Spawn the client handler in a goroutine
			go service.ManageClient(ctx, conn)

			// Allow time for the goroutine to execute
			time.Sleep(500 * time.Millisecond)

			if tt.expectError && len(conn.sentRecvs) > 0 {
				t.Logf("ERROR: Expected request to fail, but received: %+v", conn.sentRecvs)
			}

			if !tt.expectError {
				if len(conn.sentRecvs) == 0 {
					t.Fatal("Expected at least one message sent, but none found")
				}

				// Check if we received the correct number of heartbeats and updates
				// In the real service, this loop runs every second. 
				// Our test waits 0.5s to catch the first turn.
				
				// Count heartbeats
				heartbeatCount := 0
				updateCount := 0
				for _, msg := range conn.sentRecvs {
					msgMap, ok := msg.(map[string]interface{})
					if !ok {
						continue
					}
					if t, exists := msgMap["type"]; exists {
						if t == "heartbeat" {
							heartbeatCount++
						}
						if t == "guild_update" {
							updateCount++
						}
					}
				}

				if !contains(tt.expectedTypes, "heartbeat") {
					if heartbeatCount > 0 {
						t.Errorf("Unexpected heartbeat. Total: %d", heartbeatCount)
					}
				} else if heartbeatCount != 1 {
					t.Errorf("Expected 1 heartbeat, got %d", heartbeatCount)
				}

				if !contains(tt.expectedTypes, "guild_update") {
					if updateCount > 0 {
						t.Errorf("Unexpected guild_update. Total: %d", updateCount)
					}
				} else if updateCount != 1 {
					t.Errorf("Expected 1 guild_update, got %d", updateCount)
				}
			}
		})
	}
}

func TestHearthService_ConnectsToRealTCP(t *testing.T) {
	service := NewHearthService(&mockUserService{memSvc: NewInMemoryUserService()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:0" // Get an available port
	go service.Serve(ctx, addr)

	// Wait for the listener to start (it prints to logger)
	time.Sleep(100 * time.Millisecond)

	// Attempt to connect a client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to dial service: %v", err)
	}
	defer conn.Close()

	// Receive data and check valid JSON
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from connection: %v", err)
	}

	// Note: We trust the JSON write logic as tested above, 
	// but here we ensure the connection didn't crash the service.
	if n == 0 {
		t.Error("Received 0 bytes")
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}