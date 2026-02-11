package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StorageBackend defines the storage interface
type StorageBackend interface {
	// Upload uploads a file and returns its URL
	Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error)

	// Download retrieves a file
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file
	Delete(ctx context.Context, path string) error

	// GetURL returns a public URL for a file
	GetURL(path string) string

	// GetSignedURL returns a signed URL for temporary access
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}

// FileInfo contains metadata about an uploaded file
type FileInfo struct {
	ID          uuid.UUID `json:"id"`
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Service handles file storage operations
type Service struct {
	backend        StorageBackend
	maxFileSize    int64
	allowedTypes   map[string]bool
	blockedTypes   map[string]bool
	blockedExts    map[string]bool
}

// NewService creates a new storage service
func NewService(backend StorageBackend, maxFileSizeMB int64, blockedExts []string) *Service {
	blocked := make(map[string]bool)
	for _, ext := range blockedExts {
		blocked[strings.ToLower(ext)] = true
	}

	return &Service{
		backend:      backend,
		maxFileSize:  maxFileSizeMB * 1024 * 1024,
		blockedExts:  blocked,
		blockedTypes: map[string]bool{
			"application/x-msdownload": true,
			"application/x-msdos-program": true,
			"application/x-executable": true,
		},
	}
}

// UploadFile handles file upload with validation
func (s *Service) UploadFile(
	ctx context.Context,
	file *multipart.FileHeader,
	uploaderID uuid.UUID,
	category string, // "attachments", "avatars", "icons", etc.
) (*FileInfo, error) {
	// Validate size
	if s.maxFileSize > 0 && file.Size > s.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", file.Size, s.maxFileSize)
	}

	// Validate extension
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if s.blockedExts[ext] {
		return nil, fmt.Errorf("file type not allowed: %s", ext)
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if s.blockedTypes[contentType] {
		return nil, fmt.Errorf("content type not allowed: %s", contentType)
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate path
	fileID := uuid.New()
	path := fmt.Sprintf("%s/%s/%s/%s%s",
		category,
		uploaderID.String()[:8],
		time.Now().Format("2006/01"),
		fileID.String(),
		filepath.Ext(file.Filename),
	)

	// Upload
	url, err := s.backend.Upload(ctx, path, src, contentType, file.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &FileInfo{
		ID:          fileID,
		Path:        path,
		URL:         url,
		Filename:    file.Filename,
		ContentType: contentType,
		Size:        file.Size,
		UploadedBy:  uploaderID,
		UploadedAt:  time.Now(),
	}, nil
}

// DeleteFile deletes a file
func (s *Service) DeleteFile(ctx context.Context, path string) error {
	return s.backend.Delete(ctx, path)
}

// GetURL returns a public URL for a file
func (s *Service) GetURL(path string) string {
	return s.backend.GetURL(path)
}

// GetSignedURL returns a signed URL for temporary access
func (s *Service) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return s.backend.GetSignedURL(ctx, path, expiry)
}
