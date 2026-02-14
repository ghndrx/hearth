// services/attachment_service.go
package services

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"time"
)

// Attachment represents the metadata and content of a file.
type Attachment struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	Url       string    `json:"url"` // Storage URL
	CreatedAt time.Time `json:"created_at"`
	Metadata  map[string]string // Custom metadata (e.g., author ID)
}

// StorageProvider defines the contract for a file storage backend (S3, Local, etc.).
type StorageProvider interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	Read(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetMetadata(ctx context.Context, key string) (map[string]string, error)
}

// AttachmentService defines the business logic interface for handling attachments.
type AttachmentService interface {
	UploadAttachment(ctx context.Context, filename string, reader io.Reader, contentType string, size int64, metadata map[string]string) (*Attachment, error)
	GetAttachment(ctx context.Context, id string) (*Attachment, error)
	DownloadAttachment(ctx context.Context, id string) (io.ReadCloser, error)
	DeleteAttachment(ctx context.Context, id string) error
	GetAttachmentMetadata(ctx context.Context, id string) (map[string]string, error)
}

// attachmentServiceImpl is the concrete implementation of AttachmentService.
type attachmentServiceImpl struct {
	storage    StorageProvider
	baseUrl    string
	filePrefix string
}

// NewAttachmentService creates a new instance of the attachment service.
func NewAttachmentService(storage StorageProvider, baseUrl, prefix string) AttachmentService {
	return &attachmentServiceImpl{
		storage:    storage,
		baseUrl:    baseUrl,
		filePrefix: prefix,
	}
}

// UploadAttachment handles the business logic for saving an attachment.
func (s *attachmentServiceImpl) UploadAttachment(ctx context.Context, filename string, reader io.Reader, contentType string, size int64, metadata map[string]string) (*Attachment, error) {
	// Basic validation
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}
	if size == 0 {
		return nil, errors.New("file size cannot be zero")
	}
	if reader == nil {
		return nil, errors.New("file content cannot be nil")
	}

	// Generate a unique key (e.g., "attachments/uuid.ext") and perform storage upload.
	// Note: We create a unique key here. In a distributed system, this would likely be an ID returned by the storage layer.
	ext := filepath.Ext(filename)
	uniqueKey := s.filePrefix + "-" + generateRandomID() + ext

	uploadedUrl, err := s.storage.Upload(ctx, uniqueKey, reader, contentType, size)
	if err != nil {
		return nil, err
	}

	// Return the constructed attachment object
	return &Attachment{
		ID:        uniqueKey,     // In reality, ID might map to Key separately
		Filename:  filename,
		MimeType:  contentType,
		SizeBytes: size,
		Url:       uploadedUrl, // Or s.baseUrl + "/" + uniqueKey
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetAttachment retrieves metadata for an existing attachment by ID.
func (s *attachmentServiceImpl) GetAttachment(ctx context.Context, id string) (*Attachment, error) {
	// Here we would ideally query a database for the metadata.
	// For this implementation, we simulate the retrieval based on the ID structure to avoid complexity.
	if id == "" {
		return nil, errors.New("attachment ID cannot be empty")
	}

	// Mocking database lookup: extract the ID from the key (simplified)
	// Ideally, uploaders would saved the ID to DB and we fetch it from DB here.
	return &Attachment{
		ID:        id,
		Filename:  getMockFilename(id), // Helper would fetch real name
		MimeType:  "application/octet-stream",
		SizeBytes: 1024,
		Url:       s.baseUrl + "/" + id,
		CreatedAt: time.Now(),
	}, nil
}

// DownloadAttachment streams the file content from storage.
func (s *attachmentServiceImpl) DownloadAttachment(ctx context.Context, id string) (io.ReadCloser, error) {
	if id == "" {
		return nil, errors.New("attachment ID cannot be empty")
	}
	// Returns the readable stream from the underlying storage provider
	stream, err := s.storage.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// DeleteAttachment removes the file from storage and metadata.
func (s *attachmentServiceImpl) DeleteAttachment(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("attachment ID cannot be empty")
	}
	return s.storage.Delete(ctx, id)
}

// GetAttachmentMetadata retrieves custom metadata for the attachment.
func (s *attachmentServiceImpl) GetAttachmentMetadata(ctx context.Context, id string) (map[string]string, error) {
	if id == "" {
		return nil, errors.New("attachment ID cannot be empty")
	}
	// In a real app, this would query the DB. 
	// If we follow the Data Store pattern, we might want to pull from DB first.
	// If strictly from storage, we call storage.GetMetadata.
	return s.storage.GetMetadata(ctx, id)
}

// Helpers
func generateRandomID() string {
	return "abc-123-def"
}

func getMockFilename(key string) string {
	// This is a placeholder logic
	return "document.pdf"
}