package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalBackend_validatePath(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	backend := &LocalBackend{
		basePath: tmpDir,
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		reason  string
	}{
		{
			name:    "valid relative path",
			path:    "files/test.txt",
			wantErr: false,
			reason:  "normal file path should be allowed",
		},
		{
			name:    "valid nested path",
			path:    "users/123/avatar.png",
			wantErr: false,
			reason:  "nested directories should be allowed",
		},
		{
			name:    "path with dot cleanup",
			path:    "files/./test.txt",
			wantErr: false,
			reason:  "single dot should be cleaned and allowed",
		},
		{
			name:    "directory traversal attempt - parent",
			path:    "../etc/passwd",
			wantErr: true,
			reason:  "should block path traversal to parent directory",
		},
		{
			name:    "directory traversal attempt - double parent",
			path:    "../../etc/passwd",
			wantErr: true,
			reason:  "should block multiple parent directory traversal",
		},
		{
			name:    "directory traversal attempt - nested",
			path:    "files/../../etc/passwd",
			wantErr: true,
			reason:  "should block traversal even when nested in valid path",
		},
		{
			name:    "cleaned relative path that stays within bounds",
			path:    "files/../other/test.txt",
			wantErr: false,
			reason:  "path that goes up then down but stays in bounds is OK",
		},
		{
			name:    "absolute path that would escape",
			path:    "/etc/passwd",
			wantErr: false,
			reason:  "absolute paths get joined and stay within bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v\nReason: %s\nPath: %s\nCleaned: %s\nFull: %s",
					err, tt.wantErr, tt.reason, tt.path,
					filepath.Clean(tt.path),
					filepath.Join(tmpDir, filepath.Clean(tt.path)))
			}
		})
	}
}

func TestLocalBackend_Upload(t *testing.T) {
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	t.Run("successful upload", func(t *testing.T) {
		content := []byte("test content")
		reader := bytes.NewReader(content)

		url, err := backend.Upload(ctx, "test.txt", reader, "text/plain", int64(len(content)))
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		expected := "http://example.com/files/test.txt"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}

		// Verify file exists
		fullPath := filepath.Join(tmpDir, "test.txt")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("File was not created")
		}
	})

	t.Run("path traversal blocked on upload", func(t *testing.T) {
		content := []byte("malicious content")
		reader := bytes.NewReader(content)

		_, err := backend.Upload(ctx, "../../../etc/passwd", reader, "text/plain", int64(len(content)))
		if err == nil {
			t.Error("Expected error for path traversal attempt, got nil")
		}
	})

	t.Run("nested directory creation", func(t *testing.T) {
		content := []byte("nested content")
		reader := bytes.NewReader(content)

		url, err := backend.Upload(ctx, "users/123/profile.jpg", reader, "image/jpeg", int64(len(content)))
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		expected := "http://example.com/files/users/123/profile.jpg"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}

		// Verify file exists
		fullPath := filepath.Join(tmpDir, "users", "123", "profile.jpg")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("Nested file was not created")
		}
	})
}

func TestLocalBackend_Download(t *testing.T) {
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	// Create a test file
	testContent := []byte("test content for download")
	testPath := filepath.Join(tmpDir, "download-test.txt")
	if err := os.WriteFile(testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("successful download", func(t *testing.T) {
		reader, err := backend.Download(ctx, "download-test.txt")
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		defer reader.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(reader); err != nil {
			t.Fatalf("Failed to read content: %v", err)
		}

		if !bytes.Equal(buf.Bytes(), testContent) {
			t.Errorf("Content mismatch: got %s, want %s", buf.String(), string(testContent))
		}
	})

	t.Run("path traversal blocked on download", func(t *testing.T) {
		_, err := backend.Download(ctx, "../../etc/passwd")
		if err == nil {
			t.Error("Expected error for path traversal attempt, got nil")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := backend.Download(ctx, "nonexistent.txt")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})
}

func TestLocalBackend_Delete(t *testing.T) {
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		// Create a test file
		testPath := filepath.Join(tmpDir, "delete-test.txt")
		if err := os.WriteFile(testPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Delete it
		if err := backend.Delete(ctx, "delete-test.txt"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify it's gone
		if _, err := os.Stat(testPath); !os.IsNotExist(err) {
			t.Error("File still exists after delete")
		}
	})

	t.Run("path traversal blocked on delete", func(t *testing.T) {
		err := backend.Delete(ctx, "../../../etc/passwd")
		if err == nil {
			t.Error("Expected error for path traversal attempt, got nil")
		}
	})

	t.Run("delete nonexistent file", func(t *testing.T) {
		// Should not return an error for nonexistent files (idempotent)
		if err := backend.Delete(ctx, "nonexistent.txt"); err != nil {
			t.Errorf("Expected no error for deleting nonexistent file, got: %v", err)
		}
	})
}

func TestLocalBackend_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	// Check that the base directory was created with correct permissions
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Failed to stat base directory: %v", err)
	}

	// Note: On some systems, the actual permissions may differ due to umask
	// We're mainly testing that it's not world-writable
	mode := info.Mode().Perm()
	t.Logf("Base directory permissions: %o", mode)

	// Create a subdirectory via upload
	ctx := context.Background()
	content := []byte("test")
	reader := bytes.NewReader(content)

	_, err = backend.Upload(ctx, "secure/test.txt", reader, "text/plain", int64(len(content)))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check the subdirectory permissions
	subDir := filepath.Join(tmpDir, "secure")
	subInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Failed to stat subdirectory: %v", err)
	}

	subMode := subInfo.Mode().Perm()
	t.Logf("Subdirectory permissions: %o", subMode)

	// Verify it's not world-writable (0o002 bit should not be set)
	if subMode&0o002 != 0 {
		t.Error("Subdirectory is world-writable, security risk!")
	}
}

func TestNewLocalBackend_InvalidBasePath(t *testing.T) {
	// Use a path under /proc which cannot have directories created
	_, err := NewLocalBackend("/proc/nonexistent/storage", "http://example.com")
	if err == nil {
		t.Error("expected error for invalid base path")
	}
}

func TestLocalBackend_GetURL(t *testing.T) {
	backend := &LocalBackend{
		basePath:  "/tmp/storage",
		publicURL: "http://cdn.example.com",
	}

	got := backend.GetURL("files/test.txt")
	want := "http://cdn.example.com/files/test.txt"
	if got != want {
		t.Errorf("GetURL() = %s, want %s", got, want)
	}
}

func TestLocalBackend_GetSignedURL(t *testing.T) {
	backend := &LocalBackend{
		basePath:  "/tmp/storage",
		publicURL: "http://cdn.example.com",
	}

	url, err := backend.GetSignedURL(context.Background(), "files/test.txt", 15*time.Minute)
	if err != nil {
		t.Fatalf("GetSignedURL failed: %v", err)
	}

	want := "http://cdn.example.com/files/test.txt"
	if url != want {
		t.Errorf("GetSignedURL() = %s, want %s", url, want)
	}
}

// errReader is an io.Reader that always returns an error
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

func TestLocalBackend_Upload_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	_, err = backend.Upload(context.Background(), "fail.txt", errReader{}, "text/plain", 100)
	if err == nil {
		t.Error("expected error when reader fails")
	}

	// The partial file should be cleaned up
	fullPath := filepath.Join(tmpDir, "fail.txt")
	if _, statErr := os.Stat(fullPath); !os.IsNotExist(statErr) {
		t.Error("partial file was not cleaned up after read error")
	}
}

func TestLocalBackend_Upload_VerifyContent(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	content := []byte("verify this content is written correctly")
	_, err = backend.Upload(context.Background(), "verify.txt", bytes.NewReader(content), "text/plain", int64(len(content)))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Read the file back and verify content
	data, err := os.ReadFile(filepath.Join(tmpDir, "verify.txt"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

func TestLocalBackend_UploadDownloadRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()
	content := []byte("roundtrip test data")

	_, err = backend.Upload(ctx, "roundtrip/file.bin", bytes.NewReader(content), "application/octet-stream", int64(len(content)))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	rc, err := backend.Download(ctx, "roundtrip/file.bin")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("roundtrip content mismatch: got %q, want %q", got, content)
	}
}

func TestLocalBackend_UploadDeleteVerify(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()
	content := []byte("will be deleted")

	_, err = backend.Upload(ctx, "todelete.txt", bytes.NewReader(content), "text/plain", int64(len(content)))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if err := backend.Delete(ctx, "todelete.txt"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = backend.Download(ctx, "todelete.txt")
	if err == nil {
		t.Error("expected error downloading deleted file")
	}
}

func TestLocalBackend_validatePath_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	backend := &LocalBackend{basePath: tmpDir}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty path", "", true},
		{"just a filename", "file.txt", false},
		{"deeply nested", "a/b/c/d/e/f/g/file.txt", false},
		{"path with spaces", "my files/test doc.txt", false},
		{"double dot in filename", "file..txt", true},
		{"triple dot traversal", ".../etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestLocalBackend_GetURL_Variations(t *testing.T) {
	backend := &LocalBackend{
		basePath:  "/tmp/storage",
		publicURL: "http://localhost:8080",
	}

	tests := []struct {
		path string
		want string
	}{
		{"file.txt", "http://localhost:8080/file.txt"},
		{"a/b/c.png", "http://localhost:8080/a/b/c.png"},
		{"", "http://localhost:8080/"},
	}

	for _, tt := range tests {
		got := backend.GetURL(tt.path)
		if got != tt.want {
			t.Errorf("GetURL(%q) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

func TestNewLocalBackend_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")

	backend, err := NewLocalBackend(newDir, "http://example.com")
	if err != nil {
		t.Fatalf("NewLocalBackend failed: %v", err)
	}

	if backend.basePath != newDir {
		t.Errorf("basePath = %s, want %s", backend.basePath, newDir)
	}
	if backend.publicURL != "http://example.com" {
		t.Errorf("publicURL = %s, want http://example.com", backend.publicURL)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestLocalBackend_Upload_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := NewLocalBackend(tmpDir, "http://example.com/files")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	// Upload initial content
	_, err = backend.Upload(ctx, "overwrite.txt", bytes.NewReader([]byte("original")), "text/plain", 8)
	if err != nil {
		t.Fatalf("First upload failed: %v", err)
	}

	// Upload again to same path
	_, err = backend.Upload(ctx, "overwrite.txt", bytes.NewReader([]byte("replaced")), "text/plain", 8)
	if err != nil {
		t.Fatalf("Second upload failed: %v", err)
	}

	// Verify content is the new content
	data, _ := os.ReadFile(filepath.Join(tmpDir, "overwrite.txt"))
	if string(data) != "replaced" {
		t.Errorf("expected 'replaced', got %q", string(data))
	}
}

func TestLocalBackend_Download_PathTraversalVariants(t *testing.T) {
	tmpDir := t.TempDir()
	backend := &LocalBackend{basePath: tmpDir, publicURL: "http://example.com"}

	paths := []string{
		"../../../etc/shadow",
		"foo/../../etc/passwd",
		"..%2F..%2Fetc/passwd", // URL-encoded (should still be caught by filepath.Clean)
	}

	for _, p := range paths {
		_, err := backend.Download(context.Background(), p)
		if err == nil && strings.Contains(p, "..") {
			t.Errorf("expected error for traversal path %q", p)
		}
	}
}
