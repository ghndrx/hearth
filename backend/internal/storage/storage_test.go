package storage

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockBackend is a mock storage backend for testing
type MockBackend struct {
	files      map[string][]byte
	publicURL  string
	uploadErr  error
	downloadErr error
	deleteErr  error
}

func NewMockBackend(publicURL string) *MockBackend {
	return &MockBackend{
		files:     make(map[string][]byte),
		publicURL: publicURL,
	}
}

func (m *MockBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	m.files[path] = data
	return m.GetURL(path), nil
}

func (m *MockBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockBackend) Delete(ctx context.Context, path string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.files, path)
	return nil
}

func (m *MockBackend) GetURL(path string) string {
	return m.publicURL + "/" + path
}

func (m *MockBackend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return m.GetURL(path) + "?signed=true", nil
}

func TestNewService(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, []string{"exe", "bat", "cmd"})

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.backend != backend {
		t.Error("backend not set correctly")
	}
	if service.maxFileSize != 10*1024*1024 {
		t.Errorf("expected max file size %d, got %d", 10*1024*1024, service.maxFileSize)
	}
	if !service.blockedExts["exe"] {
		t.Error("expected exe to be blocked")
	}
	if !service.blockedExts["bat"] {
		t.Error("expected bat to be blocked")
	}
}

func TestNewServiceBlockedExtsCaseInsensitive(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, []string{"EXE", "BAT"})

	if !service.blockedExts["exe"] {
		t.Error("expected exe to be blocked (case insensitive)")
	}
	if !service.blockedExts["bat"] {
		t.Error("expected bat to be blocked (case insensitive)")
	}
}

// Helper to create a mock multipart file header
func createMockFileHeader(filename, contentType string, content []byte) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, _ := writer.CreatePart(h)
	part.Write(content)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(10 << 20)

	return form.File["file"][0]
}

func TestUploadFile(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, []string{"exe"})

	content := []byte("test file content")
	fileHeader := createMockFileHeader("test.txt", "text/plain", content)
	uploaderID := uuid.New()

	info, err := service.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil file info")
	}
	if info.Filename != "test.txt" {
		t.Errorf("expected filename 'test.txt', got %s", info.Filename)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %s", info.ContentType)
	}
	if info.UploadedBy != uploaderID {
		t.Errorf("expected uploader ID %v, got %v", uploaderID, info.UploadedBy)
	}
	if !strings.HasPrefix(info.Path, "attachments/") {
		t.Errorf("expected path to start with 'attachments/', got %s", info.Path)
	}
}

func TestUploadFileBlockedExtension(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, []string{"exe"})

	content := []byte("malware")
	fileHeader := createMockFileHeader("virus.exe", "application/octet-stream", content)
	uploaderID := uuid.New()

	_, err := service.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")

	if err == nil {
		t.Fatal("expected error for blocked extension")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestUploadFileBlockedContentType(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, nil)

	content := []byte("executable")
	fileHeader := createMockFileHeader("program.bin", "application/x-msdownload", content)
	uploaderID := uuid.New()

	_, err := service.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")

	if err == nil {
		t.Fatal("expected error for blocked content type")
	}
	if !strings.Contains(err.Error(), "content type not allowed") {
		t.Errorf("expected 'content type not allowed' error, got: %v", err)
	}
}

func TestUploadFileTooLarge(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	// 1 byte max file size
	service := NewService(backend, 0, nil)
	service.maxFileSize = 10 // 10 bytes max

	content := []byte("this is way more than 10 bytes of content")
	fileHeader := createMockFileHeader("large.txt", "text/plain", content)
	uploaderID := uuid.New()

	_, err := service.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")

	if err == nil {
		t.Fatal("expected error for file too large")
	}
	if !strings.Contains(err.Error(), "file too large") {
		t.Errorf("expected 'file too large' error, got: %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, nil)

	// Upload a file first
	content := []byte("test content")
	fileHeader := createMockFileHeader("test.txt", "text/plain", content)
	uploaderID := uuid.New()

	info, _ := service.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")

	// Delete the file
	err := service.DeleteFile(context.Background(), info.Path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	if _, exists := backend.files[info.Path]; exists {
		t.Error("file should be deleted")
	}
}

func TestGetURL(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, nil)

	url := service.GetURL("attachments/file.txt")
	expected := "https://cdn.example.com/attachments/file.txt"

	if url != expected {
		t.Errorf("expected URL %s, got %s", expected, url)
	}
}

func TestGetSignedURL(t *testing.T) {
	backend := NewMockBackend("https://cdn.example.com")
	service := NewService(backend, 10, nil)

	url, err := service.GetSignedURL(context.Background(), "attachments/file.txt", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(url, "signed=true") {
		t.Errorf("expected signed URL, got %s", url)
	}
}

func TestFileInfo(t *testing.T) {
	info := FileInfo{
		ID:          uuid.New(),
		Path:        "attachments/user123/2024/01/file.txt",
		URL:         "https://cdn.example.com/attachments/user123/2024/01/file.txt",
		Filename:    "file.txt",
		ContentType: "text/plain",
		Size:        1024,
		UploadedBy:  uuid.New(),
		UploadedAt:  time.Now(),
	}

	if info.Filename != "file.txt" {
		t.Errorf("expected filename 'file.txt', got %s", info.Filename)
	}
	if info.Size != 1024 {
		t.Errorf("expected size 1024, got %d", info.Size)
	}
}

// Test LocalBackend
func TestNewLocalBackend(t *testing.T) {
	tempDir := t.TempDir()

	backend, err := NewLocalBackend(tempDir, "http://localhost:8080/files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
	if backend.basePath != tempDir {
		t.Errorf("expected base path %s, got %s", tempDir, backend.basePath)
	}
}

func TestLocalBackendUploadDownload(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	content := []byte("test file content")
	path := "uploads/test.txt"

	// Upload
	url, err := backend.Upload(context.Background(), path, bytes.NewReader(content), "text/plain", int64(len(content)))
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}
	if url != "http://localhost:8080/files/"+path {
		t.Errorf("unexpected URL: %s", url)
	}

	// Verify file exists
	fullPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("file should exist")
	}

	// Download
	reader, err := backend.Download(context.Background(), path)
	if err != nil {
		t.Fatalf("download error: %v", err)
	}
	defer reader.Close()

	downloaded, _ := io.ReadAll(reader)
	if !bytes.Equal(downloaded, content) {
		t.Errorf("content mismatch: expected %s, got %s", content, downloaded)
	}
}

func TestLocalBackendDelete(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	content := []byte("test file content")
	path := "uploads/delete-me.txt"

	// Upload
	backend.Upload(context.Background(), path, bytes.NewReader(content), "text/plain", int64(len(content)))

	// Delete
	err := backend.Delete(context.Background(), path)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// Verify deleted
	fullPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestLocalBackendDeleteNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	// Should not error when deleting non-existent file
	err := backend.Delete(context.Background(), "does-not-exist.txt")
	if err != nil {
		t.Errorf("unexpected error deleting non-existent file: %v", err)
	}
}

func TestLocalBackendGetURL(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	url := backend.GetURL("path/to/file.txt")
	if url != "http://localhost:8080/files/path/to/file.txt" {
		t.Errorf("unexpected URL: %s", url)
	}
}

func TestLocalBackendGetSignedURL(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	// Local backend just returns regular URL
	url, err := backend.GetSignedURL(context.Background(), "path/to/file.txt", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:8080/files/path/to/file.txt" {
		t.Errorf("unexpected URL: %s", url)
	}
}

func TestLocalBackendDownloadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	backend, _ := NewLocalBackend(tempDir, "http://localhost:8080/files")

	_, err := backend.Download(context.Background(), "does-not-exist.txt")
	if err == nil {
		t.Error("expected error downloading non-existent file")
	}
}

// S3Backend tests (stub implementation)

func TestNewS3Backend(t *testing.T) {
	cfg := S3Config{
		Endpoint:       "https://s3.amazonaws.com",
		Bucket:         "test-bucket",
		Region:         "us-east-1",
		AccessKey:      "access-key",
		SecretKey:      "secret-key",
		PublicURL:      "https://cdn.example.com",
		ForcePathStyle: true,
	}

	backend, err := NewS3Backend(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestS3BackendGetURL(t *testing.T) {
	cfg := S3Config{
		Bucket:    "test-bucket",
		PublicURL: "https://cdn.example.com",
	}
	backend, _ := NewS3Backend(cfg)

	url := backend.GetURL("path/to/file.txt")
	expected := "https://cdn.example.com/path/to/file.txt"
	if url != expected {
		t.Errorf("expected URL %s, got %s", expected, url)
	}
}

func TestS3BackendUploadNotImplemented(t *testing.T) {
	cfg := S3Config{Bucket: "test-bucket"}
	backend, _ := NewS3Backend(cfg)

	_, err := backend.Upload(context.Background(), "path/file.txt", strings.NewReader("data"), "text/plain", 4)
	if err != ErrS3NotImplemented {
		t.Errorf("expected ErrS3NotImplemented, got %v", err)
	}
}

func TestS3BackendDownloadNotImplemented(t *testing.T) {
	cfg := S3Config{Bucket: "test-bucket"}
	backend, _ := NewS3Backend(cfg)

	_, err := backend.Download(context.Background(), "path/file.txt")
	if err != ErrS3NotImplemented {
		t.Errorf("expected ErrS3NotImplemented, got %v", err)
	}
}

func TestS3BackendDeleteNotImplemented(t *testing.T) {
	cfg := S3Config{Bucket: "test-bucket"}
	backend, _ := NewS3Backend(cfg)

	err := backend.Delete(context.Background(), "path/file.txt")
	if err != ErrS3NotImplemented {
		t.Errorf("expected ErrS3NotImplemented, got %v", err)
	}
}

func TestS3BackendGetSignedURLNotImplemented(t *testing.T) {
	cfg := S3Config{Bucket: "test-bucket"}
	backend, _ := NewS3Backend(cfg)

	_, err := backend.GetSignedURL(context.Background(), "path/file.txt", time.Hour)
	if err != ErrS3NotImplemented {
		t.Errorf("expected ErrS3NotImplemented, got %v", err)
	}
}

func TestErrS3NotImplemented(t *testing.T) {
	expectedMsg := "S3 storage requires Go 1.23+, use local storage for now"
	if ErrS3NotImplemented.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, ErrS3NotImplemented.Error())
	}
}
