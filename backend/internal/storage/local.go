package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalBackend implements StorageBackend for local filesystem
type LocalBackend struct {
	basePath  string
	publicURL string
}

// NewLocalBackend creates a new local filesystem storage backend
func NewLocalBackend(basePath, publicURL string) (*LocalBackend, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalBackend{
		basePath:  basePath,
		publicURL: publicURL,
	}, nil
}

// Upload saves a file to the local filesystem
func (b *LocalBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	fullPath := filepath.Join(b.basePath, path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy data
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return b.GetURL(path), nil
}

// Download retrieves a file from the local filesystem
func (b *LocalBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(b.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from the local filesystem
func (b *LocalBackend) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(b.basePath, path)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetURL returns a public URL for a file
func (b *LocalBackend) GetURL(path string) string {
	return b.publicURL + "/" + path
}

// GetSignedURL returns a signed URL (not implemented for local storage)
func (b *LocalBackend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Local storage doesn't support signed URLs
	// Just return the regular URL
	return b.GetURL(path), nil
}
