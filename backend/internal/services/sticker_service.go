package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/storage"
)

var (
	ErrStickerNotFound     = errors.New("sticker not found")
	ErrStickerInvalid      = errors.New("invalid sticker")
	ErrStickerNameRequired = errors.New("sticker name is required")
	ErrStickerNameTooLong  = errors.New("sticker name too long (max 30 chars)")
	ErrStickerTooLarge     = errors.New("sticker file too large (max 512KB)")
	ErrStickerFormat       = errors.New("invalid sticker format (only PNG, APNG, GIF allowed)")
	ErrStickerDimensions   = errors.New("sticker dimensions too large (max 100x100px)")
)

// StickerService handles sticker business logic
type StickerService struct {
	stickerRepo StickerRepository
	storage     *storage.Service
}

// NewStickerService creates a new sticker service
func NewStickerService(stickerRepo StickerRepository, storageService *storage.Service) *StickerService {
	return &StickerService{
		stickerRepo: stickerRepo,
		storage:     storageService,
	}
}

// StickerFileInfo contains sticker upload validation info
type StickerFileInfo struct {
	Format       models.StickerFormat
	IsAnimated   bool
	Width        int
	Height       int
	Requirements string
}

// ValidateStickerUpload validates a sticker file
func (s *StickerService) ValidateStickerUpload(file *multipart.FileHeader) error {
	// Max size: 512KB
	if file.Size > 512*1024 {
		return ErrStickerTooLarge
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		// PNG is valid
	case "image/apng":
		// APNG is valid
	case "image/gif":
		// GIF is valid
	default:
		return ErrStickerFormat
	}

	return nil
}

// GetStickerFormatFromContentType returns the StickerFormat from content type
func GetStickerFormatFromContentType(contentType string) models.StickerFormat {
	switch contentType {
	case "image/png":
		return models.StickerFormatPNG
	case "image/apng":
		return models.StickerFormatAPNG
	case "image/gif":
		return models.StickerFormatGIF
	default:
		return models.StickerFormatPNG
	}
}

// Create creates a new sticker
func (s *StickerService) Create(
	ctx context.Context,
	serverID *uuid.UUID,
	name string,
	tags []string,
	file *multipart.FileHeader,
	uploadedBy uuid.UUID,
) (*models.Sticker, error) {
	if name == "" {
		return nil, ErrStickerNameRequired
	}
	if len(name) > 30 {
		return nil, ErrStickerNameTooLong
	}

	// Validate file
	if err := s.ValidateStickerUpload(file); err != nil {
		return nil, err
	}

	// Upload the file
	var url string
	if s.storage != nil {
		fileInfo, err := s.storage.UploadFile(ctx, file, uploadedBy, "stickers")
		if err != nil {
			return nil, fmt.Errorf("failed to upload sticker: %w", err)
		}
		url = fileInfo.URL
	} else {
		// Fallback for testing
		url = "/stickers/" + uuid.New().String() + ".png"
	}

	// Get format
	format := GetStickerFormatFromContentType(file.Header.Get("Content-Type"))

	sticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  serverID,
		Name:      name,
		Tags:      tags,
		URL:       url,
		Format:    format,
		CreatedBy: uploadedBy,
		CreatedAt: time.Now(),
	}

	if err := s.stickerRepo.Create(ctx, sticker); err != nil {
		return nil, fmt.Errorf("failed to create sticker: %w", err)
	}

	return sticker, nil
}

// Get retrieves a sticker by ID
func (s *StickerService) Get(ctx context.Context, stickerID uuid.UUID) (*models.Sticker, error) {
	sticker, err := s.stickerRepo.GetByID(ctx, stickerID)
	if err != nil {
		return nil, err
	}
	if sticker == nil {
		return nil, ErrStickerNotFound
	}
	return sticker, nil
}

// GetByServer retrieves all stickers for a server
func (s *StickerService) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Sticker, error) {
	return s.stickerRepo.GetByServer(ctx, serverID)
}

// GetGlobal retrieves all global stickers
func (s *StickerService) GetGlobal(ctx context.Context) ([]*models.Sticker, error) {
	return s.stickerRepo.GetGlobal(ctx)
}

// GetAvailable returns all stickers available to a user (global + server-specific)
func (s *StickerService) GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.Sticker, error) {
	return s.stickerRepo.GetAvailable(ctx, serverID)
}

// Update updates a sticker's name and tags
func (s *StickerService) Update(ctx context.Context, stickerID uuid.UUID, name string, tags []string) (*models.Sticker, error) {
	sticker, err := s.stickerRepo.GetByID(ctx, stickerID)
	if err != nil {
		return nil, err
	}
	if sticker == nil {
		return nil, ErrStickerNotFound
	}

	if name != "" {
		if len(name) > 30 {
			return nil, ErrStickerNameTooLong
		}
		sticker.Name = name
	}

	if tags != nil {
		sticker.Tags = tags
	}

	if err := s.stickerRepo.Update(ctx, sticker); err != nil {
		return nil, err
	}

	return sticker, nil
}

// Delete deletes a sticker
func (s *StickerService) Delete(ctx context.Context, stickerID uuid.UUID) error {
	sticker, err := s.stickerRepo.GetByID(ctx, stickerID)
	if err != nil {
		return err
	}
	if sticker == nil {
		return ErrStickerNotFound
	}

	return s.stickerRepo.Delete(ctx, stickerID)
}

// Search searches stickers by name or tags
func (s *StickerService) Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.Sticker, error) {
	return s.stickerRepo.Search(ctx, query, serverID)
}
