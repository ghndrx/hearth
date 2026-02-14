// package services defined externally; assuming package services

package services

import (
	"context"
	"testing"
	"time"
)

// mockRepo for testing
type mockRepo struct {
	createThreadFunc  func(ctx context.Context, thread *Thread) error
	getThreadFunc     func(ctx context.Context, id string) (*Thread, error)
	listThreadsFunc   func(ctx context.Context, channelID string) ([]*Thread, error)
	archiveThreadFunc func(ctx context.Context, id string) error
	
	createdThread *Thread // To test returns
}

func (m *mockRepo) CreateThread(ctx context.Context, thread *Thread) error {
	if m.createThreadFunc != nil {
		return m.createThreadFunc(ctx, thread)
	}
	m.createdThread = thread
	thread.ID = "mock_thread_123"
	return nil
}

func (m *mockRepo) GetThread(ctx context.Context, id string) (*Thread, error) {
	if m.getThreadFunc != nil {
		return m.getThreadFunc(ctx, id)
	}
	if id == "error_id" {
		return nil, errors.New("not found")
	}
	return &Thread{
		ID:        id,
		Title:     "Test Title",
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockRepo) ListThreads(ctx context.Context, channelID string) ([]*Thread, error) {
	if m.listThreadsFunc != nil {
		return m.listThreadsFunc(ctx, channelID)
	}
	return []*Thread{
		{
			ID:        "thread_1",
			ChannelID: channelID,
			Title:     "Thread 1",
		},
		{
			ID:        "thread_2",
			ChannelID: channelID,
			Title:     "Thread 2",
		},
	}, nil
}

func (m *mockRepo) ArchiveThread(ctx context.Context, id string) error {
	if m.archiveThreadFunc != nil {
		return m.archiveThreadFunc(ctx, id)
	}
	return nil
}

func TestCreateThread(t *testing.T) {
	mockRepo := &mockRepo{}
	service := NewThreadService(mockRepo)
	ctx := context.Background()

	authorName := "TestUser"
	title := "Welcome to the thread"

	// Act
	thread, err := service.CreateThread(ctx, "ch_1", "u_1", authorName, title)

	// Assert
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	if thread == nil {
		t.Fatal("Expected thread to be created, got nil")
	}

	if thread.Title != title {
		t.Errorf("Expected title %q, got %q", title, thread.Title)
	}

	if thread.AuthorName != authorName {
		t.Errorf("Expected author %q, got %q", authorName, thread.AuthorName)
	}

	// Verify mock was called
	if mockRepo.createdThread == nil {
		t.Fatal("Repository CreateThread was not called")
	}
}

func TestGetThread(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo := &mockRepo{}
		service := NewThreadService(mockRepo)
		
		_, err := service.GetThread(ctx, "existing_id")
		if err != nil {
			t.Fatalf("GetThread failed: %v", err)
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		service := NewThreadService(&mockRepo{})
		_, err := service.GetThread(ctx, "")
		if err == nil {
			t.Error("Expected error for empty ID, got nil")
		}
	})

	t.Run("NonExistent", func(t *testing.T) {
		service := NewThreadService(&mockRepo{getThreadFunc: func(ctx context.Context, id string) (*Thread, error) {
			return nil, errors.New("not found")
		}})
		_, err := service.GetThread(ctx, "error_id")
		if err == nil {
			t.Error("Expected error for non-existent ID, got nil")
		}
	})
}

func TestListThreads(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepo{listThreadsFunc: func(ctx context.Context, channelID string) ([]*Thread, error) {
		return []*Thread{
			{ID: "t1"},
			{ID: "t2"},
		}, nil
	}}
	service := NewThreadService(mockRepo)

	threads, err := service.ListThreads(ctx, "some_channel")

	if err != nil {
		t.Fatalf("ListThreads failed: %v", err)
	}

	if len(threads) != 2 {
		t.Errorf("Expected 2 threads, got %d", len(threads))
	}
}

func TestArchiveThread(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepo{}
	service := NewThreadService(mockRepo)

	err := service.ArchiveThread(ctx, "thread_id_xyz")
	if err != nil {
		t.Fatalf("ArchiveThread failed: %v", err)
	}

	if mockRepo.archiveThreadFunc == nil {
		t.Error("ArchiveThread should call repository method")
	}
}

func TestArchiveThread_EmptyID(t *testing.T) {
	service := NewThreadService(&mockRepo{})
	err := service.ArchiveThread(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty ID, got nil")
	}
}