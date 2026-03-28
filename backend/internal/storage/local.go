package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalBackend implements StorageBackend for local filesystem
type LocalBackend struct {
	basePath  string
	publicURL string
}

// validatePath ensures the path doesn't escape the base directory
func (b *LocalBackend) validatePath(path string) error {
	// Clean the path to resolve any .. or . elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: contains parent directory reference")
	}

	// Ensure the resolved path is within basePath
	fullPath := filepath.Join(b.basePath, cleanPath)
	if !strings.HasPrefix(fullPath, filepath.Clean(b.basePath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid path: outside storage directory")
	}

	return nil
}

// NewLocalBackend creates a new local filesystem storage backend
func NewLocalBackend(basePath, publicURL string) (*LocalBackend, error) {
	// Ensure base path exists with secure permissions (0750)
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalBackend{
		basePath:  basePath,
		publicURL: publicURL,
	}, nil
}

// Upload saves a file to the local filesystem
func (b *LocalBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (retPath string, retErr error) {
	// Validate path to prevent directory traversal
	if err := b.validatePath(path); err != nil {
		return "", err
	}

	fullPath := filepath.Join(b.basePath, path)

	// Ensure directory exists with secure permissions (0750)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := dst.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close file: %w", err)
		}
	}()

	// Copy data
	if _, err := io.Copy(dst, file); err != nil {
		if removeErr := os.Remove(fullPath); removeErr != nil && !os.IsNotExist(removeErr) {
			// Log cleanup failure but return original error
			log.Printf("Failed to remove partial file %s: %v", fullPath, removeErr)
		}
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return b.GetURL(path), nil
}

// Download retrieves a file from the local filesystem
func (b *LocalBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	// Validate path to prevent directory traversal
	if err := b.validatePath(path); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(b.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from the local filesystem
func (b *LocalBackend) Delete(ctx context.Context, path string) error {
	// Validate path to prevent directory traversal
	if err := b.validatePath(path); err != nil {
		return err
	}

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
