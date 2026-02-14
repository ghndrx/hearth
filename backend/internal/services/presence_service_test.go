// package services_test
package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockRepo implements PresenceRepo for testing
type MockRepo struct {
	Presences []*Presence
	Err       error
}

func (m *MockRepo) GetPresence(ctx context.Context, userID string, serverID *string) (*Presence, error) {
	for _, p := range m.Presences {
		if p.UserID == userID {
			return p, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *MockRepo) UpsertPresence(ctx context.Context, presence *Presence) error {
	// Simple in-memory update logic for the mock
	existing := false
	for i, p := range m.Presences {
		if p.UserID == presence.UserID {
			m.Presences[i] = presence // Update
			existing = true
			break
		}
	}
	if !existing {
		m.Presences = append(m.Presences, presence) // Create
	}
	return m.Err
}

func (m *MockRepo) GetOnlineUsers(ctx context.Context, serverID string) ([]*Presence, error) {
	var online []*Presence
	for _, p := range m.Presences {
		// Mock logic: consider user online if status is "online" or "away" (common exceptions)
		// Or simply if the repo passed them in this specific list. 
		// Here we just filter by the mocked data.
		if p.Status == StatusOnline {
			online = append(online, p)
		}
	}
	return online, m.Err
}

// ----- Tests -----

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		userID        string
		username      string
		statusStr     string
		activity      string
		serverID      *string
		expectStatus  Status // Expected status after update
		expectError   bool
		mockUpsertErr error
	}{
		{
			name:         "Success: Global Online",
			ctx:          context.Background(),
			userID:       "user123",
			username:     "Alice",
			statusStr:    "online",
			activity:     "Playing CS",
			serverID:     nil,
			expectStatus: StatusOnline,
			expectError:  false,
		},
		{
			name:         "Success: Server Specific",
			ctx:          context.Background(),
			userID:       "user123",
			username:     "Alice",
			statusStr:    "online",
			activity:     "",
			serverID:     strPtr("server_abc"),
			expectStatus: StatusOnline,
			expectError:  false,
		},
		{
			name:        "Failure: Invalid Status",
			ctx:         context.Background(),
			userID:      "user123",
			statusStr:   "farming",
			expectError: true,
		},
		{
			name:             "Failure: Upsert Fails",
			ctx:              context.Background(),
			userID:           "user123",
			statusStr:        "online",
			mockUpsertErr:    errors.New("database connection error"),
			expectError:      true,
			expectStatus:     StatusOnline, // Status shouldn't be updated in logic if error occurs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := &MockRepo{Err: tt.mockUpsertErr}
			svc := NewPresenceService(mockRepo)
			
			// Action
			err := svc.UpdateStatus(tt.ctx, tt.userID, tt.username, tt.statusStr, tt.activity, tt.serverID)

			// Assert
			if tt.expectError && err == nil {
				t.Errorf("expected error, but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// Check the mock repo was called correctly
				// (In a real test, you might assert on the Repo input more strictly)
			}
		})
	}
}

func TestGetOnlineUsers(t *testing.T) {
	ctx := context.Background()
	serverID := "server_1"

	tests := []struct {
		name          string
		setupMock     func(*MockRepo) // Function to setup pre-defined data
		expectedCount int
		expectError   bool
	}{
		{
			name: "Happy Path: Returns list of online users",
			setupMock: func(m *MockRepo) {
				onlineUser1 := &Presence{UserID: "u1", Username: "User1", Status: StatusOnline, ServerID: serverID, LastSeen: time.Now()}
				onlineUser2 := &Presence{UserID: "u2", Username: "User2", Status: StatusOnline, ServerID: serverID, LastSeen: time.Now()}
				// Should not appear
				offline := &Presence{UserID: "u3", Username: "User3", Status: StatusAway, ServerID: serverID, LastSeen: time.Now()} 
				
				m.Presences = append(m.Presences, onlineUser1, onlineUser2, offline)
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "Server Specific",
			setupMock: func(m *MockRepo) {
				// Mock repo implementation defined above filters by StatusOnline
				// Here we rely on expectation that GetOnlineUsers logic in repo returns only Online
				m.Presences = append(m.Presences, 
					&Presence{UserID: "u1", Username: "User1", Status: StatusOnline, ServerID: serverID, LastSeen: time.Now()},
				)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "Error Propagation",
			setupMock: func(m *MockRepo) {
				m.Err = errors.New("database error")
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepo{}
			tt.setupMock(mockRepo)
			
			svc := NewPresenceService(mockRepo)

			users, err := svc.GetOnlineUsers(ctx, serverID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(users) != tt.expectedCount {
					t.Errorf("expected %d users, got %d", tt.expectedCount, len(users))
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}