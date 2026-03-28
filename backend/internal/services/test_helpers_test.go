package services

import (
	"bytes"
	"context"
	"io"
	"time"
)

// mockStorageBackend implements storage.StorageBackend for testing
type mockStorageBackend struct {
	files map[string][]byte
}

func newMockStorageBackend() *mockStorageBackend {
	return &mockStorageBackend{
		files: make(map[string][]byte),
	}
}

func (m *mockStorageBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	m.files[path] = data
	return "http://test.com/" + path, nil
}

func (m *mockStorageBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if data, ok := m.files[path]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, io.EOF
}

func (m *mockStorageBackend) Delete(ctx context.Context, path string) error {
	delete(m.files, path)
	return nil
}

func (m *mockStorageBackend) GetURL(path string) string {
	return "http://test.com/" + path
}

func (m *mockStorageBackend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "http://test.com/signed/" + path, nil
}
