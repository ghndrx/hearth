package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- mockBackend for testing Service methods ---

type mockBackend struct {
	uploaded    map[string]*bytes.Reader
	uploadErr   error
	downloadErr error
	deleteErr   error
	urlPrefix   string
	signedErr   error
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		uploaded: make(map[string]*bytes.Reader),
	}
}

func (m *mockBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	m.uploaded[path] = bytes.NewReader(data)
	return m.urlPrefix + "/" + path, nil
}

func (m *mockBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	r, ok := m.uploaded[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	data, _ := io.ReadAll(r)
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockBackend) Delete(ctx context.Context, path string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.uploaded, path)
	return nil
}

func (m *mockBackend) GetURL(path string) string {
	return m.urlPrefix + "/" + path
}

func (m *mockBackend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	if m.signedErr != nil {
		return "", m.signedErr
	}
	return m.urlPrefix + "/" + path + "?signed&expiry=" + expiry.String(), nil
}

// --- Service tests ---

func TestNewService(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"

	svc := NewService(backend, 10, []string{"exe", "bat", "sh"})

	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.backend == nil {
		t.Error("backend not set")
	}
	if svc.maxFileSize != 10*1024*1024 {
		t.Errorf("maxFileSize = %d, want %d", svc.maxFileSize, 10*1024*1024)
	}
	if !svc.blockedExts["exe"] || !svc.blockedExts["bat"] || !svc.blockedExts["sh"] {
		t.Error("blocked extensions not set correctly")
	}
}

func TestNewService_DefaultBlockedTypes(t *testing.T) {
	backend := newMockBackend()
	svc := NewService(backend, 0, nil)

	// Verify default blocked content types
	if !svc.blockedTypes["application/x-msdownload"] {
		t.Error("expected default blocked type application/x-msdownload")
	}
	if !svc.blockedTypes["application/x-msdos-program"] {
		t.Error("expected default blocked type application/x-msdos-program")
	}
	if !svc.blockedTypes["application/x-executable"] {
		t.Error("expected default blocked type application/x-executable")
	}
}

func TestService_UploadFile(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"
	svc := NewService(backend, 10, []string{"exe"})

	uploaderID := uuid.New()

	makeFormFile := func(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Use CreatePart to properly set content type
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
		h["Content-Type"] = []string{contentType}

		part, err := writer.CreatePart(h)
		if err != nil {
			t.Fatalf("CreatePart failed: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("ParseMultipartForm failed: %v", err)
		}

		f := req.MultipartForm.File["file"]
		if len(f) == 0 {
			t.Fatal("no file in form")
		}
		return f[0]
	}

	t.Run("successful upload", func(t *testing.T) {
		data := []byte("hello world")
		fileHeader := makeFormFile(t, "test.txt", "text/plain", data)

		info, err := svc.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")
		if err != nil {
			t.Fatalf("UploadFile failed: %v", err)
		}
		if info == nil {
			t.Fatal("expected FileInfo, got nil")
		}
		if info.Filename != "test.txt" {
			t.Errorf("Filename = %s, want test.txt", info.Filename)
		}
		if info.ContentType != "text/plain" {
			t.Errorf("ContentType = %s, want text/plain", info.ContentType)
		}
		if info.Size != int64(len(data)) {
			t.Errorf("Size = %d, want %d", info.Size, int64(len(data)))
		}
		if info.UploadedBy != uploaderID {
			t.Errorf("UploadedBy = %v, want %v", info.UploadedBy, uploaderID)
		}
		if info.ID == uuid.Nil {
			t.Error("expected non-nil UUID for ID")
		}
	})

	t.Run("file too large", func(t *testing.T) {
		svcSmall := NewService(backend, 1, nil) // 1 MB limit
		// Create data larger than 1MB
		data := make([]byte, 2*1024*1024)
		fileHeader := makeFormFile(t, "large.txt", "text/plain", data)

		_, err := svcSmall.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")
		if err == nil {
			t.Error("expected error for oversized file")
		}
	})

	t.Run("blocked extension", func(t *testing.T) {
		data := []byte("malicious")
		fileHeader := makeFormFile(t, "malware.exe", "application/octet-stream", data)

		_, err := svc.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")
		if err == nil {
			t.Error("expected error for blocked extension")
		}
	})

	t.Run("blocked content type", func(t *testing.T) {
		data := []byte("malicious")
		fileHeader := makeFormFile(t, "file.txt", "application/x-msdownload", data)

		_, err := svc.UploadFile(context.Background(), fileHeader, uploaderID, "attachments")
		if err == nil {
			t.Error("expected error for blocked content type")
		}
	})
}

func TestService_DeleteFile(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"
	svc := NewService(backend, 0, nil)

	// Pre-upload a file via the service
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "delete-me.txt")
	_, _ = part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(32 << 20)
	f := req.MultipartForm.File["file"][0]

	uploaderID := uuid.New()
	info, err := svc.UploadFile(context.Background(), f, uploaderID, "attachments")
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	t.Run("successful delete", func(t *testing.T) {
		err := svc.DeleteFile(context.Background(), info.Path)
		if err != nil {
			t.Errorf("DeleteFile failed: %v", err)
		}
	})

	t.Run("delete via backend error", func(t *testing.T) {
		backend.deleteErr = errors.New("delete failed")
		err := svc.DeleteFile(context.Background(), "any/path")
		if err == nil {
			t.Error("expected error when backend delete fails")
		}
		backend.deleteErr = nil
	})
}

func TestService_GetURL(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://cdn.example.com"
	svc := NewService(backend, 0, nil)

	url := svc.GetURL("attachments/abc123/test.png")
	expected := "http://cdn.example.com/attachments/abc123/test.png"
	if url != expected {
		t.Errorf("GetURL = %s, want %s", url, expected)
	}
}

func TestService_GetSignedURL(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://cdn.example.com"
	svc := NewService(backend, 0, nil)

	expiry := 15 * time.Minute
	url, err := svc.GetSignedURL(context.Background(), "attachments/abc/test.pdf", expiry)
	if err != nil {
		t.Errorf("GetSignedURL failed: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty signed URL")
	}

	t.Run("backend signed URL error", func(t *testing.T) {
		backend.signedErr = errors.New("presign failed")
		_, err := svc.GetSignedURL(context.Background(), "path", expiry)
		if err == nil {
			t.Error("expected error when backend fails")
		}
		backend.signedErr = nil
	})
}

func TestService_Download(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"
	svc := NewService(backend, 0, nil)

	// Upload a file first
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "download-test.txt")
	_, _ = part.Write([]byte("download content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(32 << 20)
	f := req.MultipartForm.File["file"][0]

	uploaderID := uuid.New()
	info, err := svc.UploadFile(context.Background(), f, uploaderID, "attachments")
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	t.Run("successful download", func(t *testing.T) {
		rc, err := svc.Download(context.Background(), info.Path)
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		if string(data) != "download content" {
			t.Errorf("content = %q, want %q", string(data), "download content")
		}
	})

	t.Run("download error", func(t *testing.T) {
		backend.downloadErr = errors.New("download failed")
		_, err := svc.Download(context.Background(), info.Path)
		if err == nil {
			t.Error("expected error when backend download fails")
		}
		backend.downloadErr = nil
	})
}

func TestService_UploadFile_BackendError(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"
	backend.uploadErr = errors.New("backend upload failed")
	svc := NewService(backend, 10, nil)

	uploaderID := uuid.New()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte("hello"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(32 << 20)
	f := req.MultipartForm.File["file"][0]

	_, err := svc.UploadFile(context.Background(), f, uploaderID, "attachments")
	if err == nil {
		t.Error("expected error when backend upload fails")
	}
}

func TestService_UploadFile_NoSizeLimit(t *testing.T) {
	backend := newMockBackend()
	backend.urlPrefix = "http://localhost/files"
	svc := NewService(backend, 0, nil) // maxFileSize 0 means no limit

	uploaderID := uuid.New()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "big.txt")
	_, _ = part.Write(make([]byte, 1024))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(32 << 20)
	f := req.MultipartForm.File["file"][0]

	info, err := svc.UploadFile(context.Background(), f, uploaderID, "avatars")
	if err != nil {
		t.Fatalf("UploadFile with no size limit failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected FileInfo, got nil")
	}
}

func TestNewService_EmptyBlockedExts(t *testing.T) {
	backend := newMockBackend()
	svc := NewService(backend, 5, nil)
	if len(svc.blockedExts) != 0 {
		t.Errorf("expected 0 blocked extensions, got %d", len(svc.blockedExts))
	}
}

func TestNewService_BlockedExtsCaseInsensitive(t *testing.T) {
	backend := newMockBackend()
	svc := NewService(backend, 5, []string{"EXE", "Bat", "SH"})

	for _, ext := range []string{"exe", "bat", "sh"} {
		if !svc.blockedExts[ext] {
			t.Errorf("expected blocked ext %q to be normalized to lowercase", ext)
		}
	}
}
