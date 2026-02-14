// services/attachment_service_test.go
package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

// mockStorage implements StorageProvider in-memory for testing.
type mockStorage struct {
	objects map[string][]byte
	meta    map[string]map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		objects: make(map[string][]byte),
		meta:    make(map[string]map[string]string),
	}
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	// Read entire content into memory (not recommended for production for huge files)
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	m.objects[key] = content
	m.meta[key] = make(map[string]string)
	m.meta[key]["source"] = "test_mock"

	// Return a fake URL or store key
	return "http://test.storage/" + key, nil
}

func (m *mockStorage) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	content, exists := m.objects[key]
	if !exists {
		return nil, errors.New("key not found")
	}
	return io.NopCloser(strings.NewReader(string(content))), nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	delete(m.objects, key)
	delete(m.meta, key)
	return nil
}

func (m *mockStorage) GetMetadata(ctx context.Context, key string) (map[string]string, error) {
	if meta, exists := m.meta[key]; exists {
		return meta, nil
	}
	return nil, errors.New("metadata not found")
}

func TestUploadAttachment(t *testing.T) {
	testData := []byte("test file content")
	
	storage := newMockStorage()
	service := NewAttachmentService(storage, "http://localhost:8080", "attachments")

	attachment, err := service.UploadAttachment(
		context.Background(),
		"document.pdf",
		strings.NewReader(string(testData)),
		"application/pdf",
		int64(len(testData)),
		map[string]string{"author": "test_user_123"},
	)

	if err != nil {
		t.Fatalf("UploadAttachment failed: %v", err)
	}

	if attachment.Filename != "document.pdf" {
		t.Errorf("Expected filename 'document.pdf', got '%s'", attachment.Filename)
	}

	if attachment.MimeType != "application/pdf" {
		t.Errorf("Expected mime 'application/pdf', got '%s'", attachment.MimeType)
	}

	// Verify storage persistence
	if _, exists := storage.objects["attachments-abc-123-def.pdf"]; !exists {
		t.Error("File was not uploaded to storage mock")
	}

	if storage.meta["attachments-abc-123-def.pdf"]["author"] != "test_user_123" {
		t.Error("Metadata was not saved correctly")
	}
}

func TestGetAttachmentMetadata(t *testing.T) {
	storage := newMockStorage()
	service := NewAttachmentService(storage, "http://localhost:8080", "test")

	// Pre-seed storage
	key := "test-image.jpg"
	storage.objects[key] = []byte("fake image")
	storage.meta[key] = map[string]string{"author": "eve", "size": "medium"}

	metadata, err := service.GetAttachmentMetadata(context.Background(), key)
	if err != nil {
		t.Fatalf("GetAttachmentMetadata failed: %v", err)
	}

	if metadata["author"] != "eve" {
		t.Errorf("Expected metadata author 'eve', got '%s'", metadata["author"])
	}
}

func TestDownloadAttachment(t *testing.T) {
	storage := newMockStorage()
	service := NewAttachmentService(storage, "http://localhost:8080", "downloads")

	key := "file.txt"
	expectedContent := "Hello, World!"
	storage.objects[key] = []byte(expectedContent)

	reader, err := service.DownloadAttachment(context.Background(), key)
	if err != nil {
		t.Fatalf("DownloadAttachment failed: %v", err)
	}
	defer reader.Close() // Ensure stream is closed

	actualContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	if string(actualContent) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(actualContent))
	}
}

func TestInvalidUpload(t *testing.T) {
	storage := newMockStorage()
	service := NewAttachmentService(storage, "http://localhost:8080", "attachments")

	// Test nil reader
	_, err := service.UploadAttachment(context.Background(), "file.txt", nil, "text/plain", 10, nil)
	if err == nil {
		t.Error("Expected error for nil reader, got nil")
	}

	// Test empty filename
	_, err = service.UploadAttachment(context.Background(), "", strings.NewReader("data"), "text/plain", 4, nil)
	if err == nil {
		t.Error("Expected error for empty filename, got nil")
	}
}

// Copy of error definitions for testing (In a real setup, you might import services.errors)
var (
	ErrNotFound = errors.New("attachment not found")
)