package services

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

type Attachment struct {
	ID          uuid.UUID
	MessageID   uuid.UUID
	Filename    string
	ContentType string
	Size        int64
	URL         string
}

type AttachmentService struct {
	mu          sync.RWMutex
	attachments map[uuid.UUID]*Attachment
}

func NewAttachmentService() *AttachmentService {
	return &AttachmentService{
		attachments: make(map[uuid.UUID]*Attachment),
	}
}

func (s *AttachmentService) Upload(ctx context.Context, messageID uuid.UUID, filename, contentType string, size int64) (*Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a := &Attachment{
		ID:          uuid.New(),
		MessageID:   messageID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		URL:         "/attachments/" + uuid.New().String(),
	}
	s.attachments[a.ID] = a
	return a, nil
}

func (s *AttachmentService) Get(ctx context.Context, attachmentID uuid.UUID) (*Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if a, ok := s.attachments[attachmentID]; ok {
		return a, nil
	}
	return nil, ErrAttachmentNotFound
}

func (s *AttachmentService) Delete(ctx context.Context, attachmentID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.attachments[attachmentID]; ok {
		delete(s.attachments, attachmentID)
		return nil
	}
	return ErrAttachmentNotFound
}
