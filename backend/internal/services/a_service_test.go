package services

import (
	"context"
	"testing"
)

// MockContext implements a simple mock for context.Context.
// We do not need the full context implementation to test logic.
type MockContext struct{}

func (m *MockContext) Deadline() (deadline time.Time, ok bool) { return }
func (m *MockContext) Done() <-chan struct{}                 { return nil }
func (m *MockContext) Err() error                             { return nil }
func (m *MockContext) Value(key interface{}) interface{}      { return nil }
func (m *MockContext) Type() string                           { return "mock" }

// TestUserService_CreateUser tests the happy path and error cases for CreateUser.
func TestUserService_CreateUser(t *testing.T) {
	ctx := context.Background()
	service := NewUserService()

	t.Run("Happy Path", func(t *testing.T) {
		name := "Alice"
		email := "alice@example.com"

		user, err := service.CreateUser(ctx, name, email)

		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		// Verify User fields
		if user.Name != name {
			t.Errorf("Expected name %s, got %s", name, user.Name)
		}
		if user.Email != email {
			t.Errorf("Expected email %s, got %s", email, user.Email)
		}
		if user.ID != "user_"+name {
			t.Errorf("Expected auto-generated ID, got %s", user.ID)
		}

		// Verify user exists in the store
		if _, exists := service.(*userService).users[user.ID]; !exists {
			t.Error("Expected user to be stored in map")
		}
	})

	t.Run("Empty Name", func(t *testing.T) {
		_, err := service.CreateUser(ctx, "", "test@test.com")
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
	})
}

// TestUserService_GetUser tests the retrieval logic.
func TestUserService_GetUser(t *testing.T) {
	ctx := context.Background()
	service := NewUserService()

	// Setup: Create a user in the map first (simulating persistence)
	svc := service.(*userService)
	svc.users["user_Existing"] = &User{ID: "user_Existing", Name: "Bob"}

	t.Run("Getting Existing User", func(t *testing.T) {
		user, err := service.GetUser(ctx, "user_Existing")

		if err != nil {
			t.Fatalf("Could not fetch existing user: %v", err)
		}

		if user.Name != "Bob" {
			t.Errorf("Expected Bob, got %s", user.Name)
		}
	})

	t.Run("Getting Non-Existent User", func(t *testing.T) {
		_, err := service.GetUser(ctx, "user_NonExistent")

		if err == nil {
			t.Error("Expected error for non-existent user, got nil")
		}
	})

	t.Run("GetUser with Empty ID", func(t *testing.T) {
		_, err := service.GetUser(ctx, "")
		if err == nil {
			t.Error("Expected error for empty ID, got nil")
		}
	})
}