package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/storage"
)

var (
	ErrAttachmentNotFound    = errors.New("attachment not found")
	ErrFileTooLarge          = errors.New("file too large")
	ErrFileTypeNotAllowed    = errors.New("file type not allowed")
	ErrAttachmentAccessDenied = errors.New("access denied")
)

// Attachment represents a file attachment
type Attachment struct {
	ID          uuid.UUID `json:"id"`
	MessageID   uuid.UUID `json:"message_id,omitempty"`
	ChannelID   uuid.UUID `json:"channel_id,omitempty"`
	UploaderID  uuid.UUID `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	Path        string    `json:"path"`
	AltText     string    `json:"alt_text,omitempty"` // Accessibility: description for screen readers
	CreatedAt   time.Time `json:"created_at"`
}

// AttachmentService handles file attachments
type AttachmentService struct {
	mu          sync.RWMutex
	attachments map[uuid.UUID]*Attachment
	storage     *storage.Service
}

// NewAttachmentService creates a new attachment service
func NewAttachmentService(storageService *storage.Service) *AttachmentService {
	return &AttachmentService{
		attachments: make(map[uuid.UUID]*Attachment),
		storage:     storageService,
	}
}

// Upload handles file upload
func (s *AttachmentService) Upload(
	ctx context.Context,
	file *multipart.FileHeader,
	uploaderID uuid.UUID,
	channelID uuid.UUID,
) (*Attachment, error) {
	return s.UploadWithAltText(ctx, file, uploaderID, channelID, "")
}

// UploadWithAltText handles file upload with alt text for accessibility
func (s *AttachmentService) UploadWithAltText(
	ctx context.Context,
	file *multipart.FileHeader,
	uploaderID uuid.UUID,
	channelID uuid.UUID,
	altText string,
) (*Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use storage service if available
	if s.storage != nil {
		fileInfo, err := s.storage.UploadFile(ctx, file, uploaderID, "attachments")
		if err != nil {
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		a := &Attachment{
			ID:          fileInfo.ID,
			ChannelID:   channelID,
			UploaderID:  uploaderID,
			Filename:    fileInfo.Filename,
			ContentType: fileInfo.ContentType,
			Size:        fileInfo.Size,
			URL:         fileInfo.URL,
			Path:        fileInfo.Path,
			AltText:     altText,
			CreatedAt:   fileInfo.UploadedAt,
		}
		s.attachments[a.ID] = a
		return a, nil
	}

	// Fallback to in-memory storage for testing
	a := &Attachment{
		ID:          uuid.New(),
		ChannelID:   channelID,
		UploaderID:  uploaderID,
		Filename:    file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		Size:        file.Size,
		URL:         "/attachments/" + uuid.New().String() + filepath.Ext(file.Filename),
		AltText:     altText,
		CreatedAt:   time.Now(),
	}
	s.attachments[a.ID] = a
	return a, nil
}

// UploadForMessage uploads a file and associates it with a message
func (s *AttachmentService) UploadForMessage(
	ctx context.Context,
	file *multipart.FileHeader,
	uploaderID uuid.UUID,
	channelID uuid.UUID,
	messageID uuid.UUID,
) (*Attachment, error) {
	attachment, err := s.Upload(ctx, file, uploaderID, channelID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	attachment.MessageID = messageID
	s.mu.Unlock()

	return attachment, nil
}

// Get retrieves an attachment by ID
func (s *AttachmentService) Get(ctx context.Context, attachmentID uuid.UUID) (*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if a, ok := s.attachments[attachmentID]; ok {
		return a, nil
	}
	return nil, ErrAttachmentNotFound
}

// GetByChannel retrieves all attachments for a channel
func (s *AttachmentService) GetByChannel(ctx context.Context, channelID uuid.UUID) ([]*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Attachment
	for _, a := range s.attachments {
		if a.ChannelID == channelID {
			result = append(result, a)
		}
	}
	return result, nil
}

// GetByMessage retrieves all attachments for a message
func (s *AttachmentService) GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Attachment
	for _, a := range s.attachments {
		if a.MessageID == messageID {
			result = append(result, a)
		}
	}
	return result, nil
}

// Download returns a reader for the attachment file
func (s *AttachmentService) Download(ctx context.Context, attachmentID uuid.UUID) (io.ReadCloser, *Attachment, error) {
	a, err := s.Get(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}

	if s.storage == nil {
		return nil, nil, errors.New("storage not configured")
	}

	reader, err := s.storage.Download(ctx, a.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file: %w", err)
	}

	return reader, a, nil
}

// Delete removes an attachment
func (s *AttachmentService) Delete(ctx context.Context, attachmentID uuid.UUID, requesterID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.attachments[attachmentID]
	if !ok {
		return ErrAttachmentNotFound
	}

	// Check ownership
	if a.UploaderID != requesterID {
		return ErrAttachmentAccessDenied
	}

	// Delete from storage
	if s.storage != nil && a.Path != "" {
		if err := s.storage.DeleteFile(ctx, a.Path); err != nil {
			return fmt.Errorf("failed to delete file from storage: %w", err)
		}
	}

	delete(s.attachments, attachmentID)
	return nil
}

// DeleteByMessage deletes all attachments for a message
func (s *AttachmentService) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var toDelete []uuid.UUID
	for id, a := range s.attachments {
		if a.MessageID == messageID {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		a := s.attachments[id]
		if s.storage != nil && a.Path != "" {
			s.storage.DeleteFile(ctx, a.Path)
		}
		delete(s.attachments, id)
	}

	return nil
}

// GetSignedURL returns a signed URL for temporary access
func (s *AttachmentService) GetSignedURL(ctx context.Context, attachmentID uuid.UUID, expiry time.Duration) (string, error) {
	a, err := s.Get(ctx, attachmentID)
	if err != nil {
		return "", err
	}

	if s.storage == nil {
		return a.URL, nil
	}

	return s.storage.GetSignedURL(ctx, a.Path, expiry)
}

// ValidateContentType checks if a content type is allowed
func ValidateContentType(contentType string) bool {
	// Block dangerous content types
	blockedTypes := map[string]bool{
		"application/x-msdownload":    true,
		"application/x-msdos-program": true,
		"application/x-executable":    true,
		"application/x-dosexec":       true,
	}
	return !blockedTypes[strings.ToLower(contentType)]
}

// ValidateFileExtension checks if a file extension is allowed
func ValidateFileExtension(filename string) bool {
	blockedExtensions := map[string]bool{
		".exe":  true,
		".bat":  true,
		".cmd":  true,
		".com":  true,
		".msi":  true,
		".scr":  true,
		".pif":  true,
		".vbs":  true,
		".js":   true,
		".jar":  true,
		".ps1":  true,
		".sh":   true,
		".bash": true,
	}
	ext := strings.ToLower(filepath.Ext(filename))
	return !blockedExtensions[ext]
}

// Download helper for storage.Service
func (s *AttachmentService) downloadFromStorage(ctx context.Context, path string) (io.ReadCloser, error) {
	if s.storage == nil {
		return nil, errors.New("storage not configured")
	}

	backend := s.storage
	_ = backend // storage.Service wraps the backend
	return nil, errors.New("direct download not supported, use URL")
}

// Upload_Test_Add is a test helper to add an attachment directly (for testing only)
func (s *AttachmentService) Upload_Test_Add(id uuid.UUID, a *Attachment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attachments[id] = a
}
