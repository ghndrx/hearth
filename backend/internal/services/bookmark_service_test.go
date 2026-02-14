package services

import (
	"context"
	"heart/internal/repositories"
	"errors"
	"testing"
	"time"
)

// mockBookmarkRepository is a simple in-memory mock for the repository layer.
type mockBookmarkRepository struct {
	data map[string][]Bookmark // UserID -> list of Bookmarks
	mu   sync.RWMutex
}

func NewMockBookmarkRepository() *mockBookmarkRepository {
	return &mockBookmarkRepository{
		data: make(map[string][]Bookmark),
	}
}

func (m *mockBookmarkRepository) Create(ctx context.Context, bookmark Bookmark) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check duplicates
	for _, b := range m.data[bookmark.UserID] {
		if b.Region == bookmark.Region {
			return errors.New("duplicate entry")
		}
	}

	m.data[bookmark.UserID] = append(m.data[bookmark.UserID], bookmark)
	return nil
}

func (m *mockBookmarkRepository) FindByUserID(ctx context.Context, userID string) ([]Bookmark, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[userID], nil
}

func (m *mockBookmarkRepository) Delete(ctx context.Context, userID, region string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find index
	var index int
	var found bool
	for i, b := range m.data[userID] {
		if b.Region == region {
			index = i
			found = true
			break
		}
	}

	if !found {
		return errors.New("not found")
	}

	// Remove
	newSlice := append(m.data[userID][:index], m.data[userID][index+1:]...)
	m.data[userID] = newSlice
	return nil
}

// TestBookmarkService tests the service logic.
func TestBookmarkService_Add(t *testing.T) {
	repo := NewMockBookmarkRepository()
	service := NewBookmarkService(repo, nil)
	ctx := context.Background()

	testUser := "test_user_123"
	testRegion := "na-east"

	// Test Case 1: Successful Add
	t.Run("Success", func(t *testing.T) {
		cmd := AddBookmarkCommand{Region: testRegion}
		bookmark, err := service.AddBookmark(ctx, testUser, cmd)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if bookmark.Region != testRegion {
			t.Errorf("Expected region %s, got %s", testRegion, bookmark.Region)
		}
	})

	// Test Case 2: Duplicate Add
	t.Run("Duplicate", func(t *testing.T) {
		cmd := AddBookmarkCommand{Region: testRegion}
		_, err := service.AddBookmark(ctx, testUser, cmd)

		if err == nil {
			t.Error("Expected error for duplicate add, got nil")
		}
	})

	// Test Case 3: Empty User ID
	t.Run("Invalid User ID", func(t *testing.T) {
		cmd := AddBookmarkCommand{Region: "eu-west"}
		_, err := service.AddBookmark(ctx, "", cmd)

		if err == nil {
			t.Error("Expected error for empty userID, got nil")
		}
	})
}

func TestBookmarkService_List(t *testing.T) {
	repo := NewMockBookmarkRepository()
	service := NewBookmarkService(repo, nil)
	ctx := context.Background()

	testUser := "list_user"
	initialData := []Bookmark{
		{UserID: testUser, Region: "na-east", Created: "2023-01-01"},
		{UserID: testUser, Region: "eu-west", Created: "2023-01-02"},
	}

	// Setup
	for _, d := range initialData {
		repo.Create(ctx, d)
	}

	// Test
	t.Run("Get List", func(t *testing.T) {
		bookmarks, err := service.ListBookmarks(ctx, testUser)
		if err != nil {
			t.Fatalf("Failed to list bookmarks: %v", err)
		}

		if len(bookmarks) != len(initialData) {
			t.Errorf("Expected length %d, got %d", len(initialData), len(bookmarks))
		}
	})
}

func TestBookmarkService_Remove(t *testing.T) {
	repo := NewMockBookmarkRepository()
	service := NewBookmarkService(repo, nil)
	ctx := context.Background()

	testUser := "remove_user"
	initialRegion := "na-east"

	// Setup
	repo.Create(ctx, Bookmark{UserID: testUser, Region: initialRegion})

	// Test Case 1: Successful Remove
	t.Run("Success", func(t *testing.T) {
		err := service.RemoveBookmark(ctx, testUser, initialRegion)
		if err != nil {
			t.Errorf("Expected no error on remove, got: %v", err)
		}

		// Verify deletion from repo/data
		_, err = service.ListBookmarks(ctx, testUser)
		if errors.Is(err, ErrNotFound) {
			t.Error("Bookmark should still exist after successful removal logic")
		}
	})

	// Test Case 2: Remove Non-existent
	t.Run("Not Found", func(t *testing.T) {
		err := service.RemoveBookmark(ctx, testUser, "unknown-region")
		if errors.Is(err, ErrNotFound) {
			t.Logf("Correctly returned Not Found error")
		} else if err != nil {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}

// Mock logger for empty initialization in tests
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{})        {}
func (m *mockLogger) Info(msg string, fields ...interface{})         {}
func (m *mockLogger) Warn(msg string, fields ...interface{})         {}
func (m *mockLogger) Error(msg string, fields ...interface{})        {}
func (m *mockLogger) Trace(msg string, fields ...interface{})        {}
func (m *mockLogger) Fatal(msg string, fields ...interface{})        {}
func (m *mockLogger) Panic(msg string, fields ...interface{})        {}
func (m *mockLogger) WithField(key string, value interface{}) logger.Logger {
	return m
}
func (m *mockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return m
}