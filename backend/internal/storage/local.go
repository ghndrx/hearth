//go:build go1.25

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalBackend implements StorageBackend for local filesystem.
// All file I/O is scoped under basePath via os.Root (Go 1.24+) for
// kernel-level enforcement against path traversal and symlink escapes.
type LocalBackend struct {
	basePath  string
	publicURL string
	root      *os.Root
}

// validatePath performs a fast pre-check before any file operation.
func (b *LocalBackend) validatePath(path string) error {
	if path == "" || filepath.IsAbs(path) {
		return fmt.Errorf("invalid path: must be a relative, non-empty path")
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: contains parent directory reference")
	}
	return nil
}

// ensureDirs creates each directory component of relPath via root.Mkdir.
func (b *LocalBackend) ensureDirs(relPath string) error {
	relPath = filepath.Clean(relPath)
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "" {
		return nil
	}
	parts := strings.Split(dir, string(os.PathSeparator))
	for i := range parts {
		segment := strings.Join(parts[:i+1], string(os.PathSeparator))
		if segment == "" {
			continue
		}
		if err := b.root.Mkdir(segment, 0750); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create directory %s: %w", segment, err)
		}
	}
	return nil
}

// NewLocalBackend creates a new local filesystem storage backend using os.Root
// (Go 1.24+) for kernel-level path scoping.
func NewLocalBackend(basePath, publicURL string) (*LocalBackend, error) {
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	root, err := os.OpenRoot(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoped root: %w", err)
	}
	return &LocalBackend{basePath: basePath, publicURL: publicURL, root: root}, nil
}

// Upload saves a file scoped to basePath via root.Create.
func (b *LocalBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (retPath string, retErr error) {
	if err := b.validatePath(path); err != nil {
		return "", err
	}

	// Create parent directories scoped under b.root
	if err := b.root.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file scoped under b.root — kernel enforces no traversal
	dst, err := b.root.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := dst.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close file: %w", err)
		}
	}()

	if _, err := io.Copy(dst, file); err != nil {
		// Clean up partial file scoped under b.root
		if removeErr := b.root.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			retErr = fmt.Errorf("failed to write file: %w (cleanup also failed: %v)", err, removeErr)
		} else {
			retErr = fmt.Errorf("failed to write file: %w", err)
		}
		return "", retErr
	}

	return b.GetURL(path), nil
}

// Download retrieves a file from the local filesystem.
// The path is scoped under b.root — kernel enforces no traversal.
func (b *LocalBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := b.validatePath(path); err != nil {
		return nil, err
	}

	file, err := b.root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete removes a file from the local filesystem.
// The path is scoped under b.root — kernel enforces no traversal.
func (b *LocalBackend) Delete(ctx context.Context, path string) error {
	if err := b.validatePath(path); err != nil {
		return err
	}

	if err := b.root.Remove(path); err != nil && !os.IsNotExist(err) {
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
	return b.GetURL(path), nil
}
