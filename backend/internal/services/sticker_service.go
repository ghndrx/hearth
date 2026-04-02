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

	// Sticker Pack errors
	ErrPackNotFound       = errors.New("sticker pack not found")
	ErrPackNameRequired   = errors.New("pack name is required")
	ErrPackNameTooLong    = errors.New("pack name too long (max 100 chars)")
	ErrPackTierInvalid    = errors.New("invalid pack tier")
	ErrPackTierRequired   = errors.New("pack tier required")
	ErrPackNotOwned       = errors.New("user does not own this pack")
	ErrPackNotGlobal      = errors.New("can only create global packs")
	ErrPackTierAccess     = errors.New("user tier does not have access to this pack")
	ErrStickerAlreadyInPack = errors.New("sticker already in pack")
)

// StickerService handles sticker business logic
type StickerService struct {
	stickerRepo StickerRepository
	storage     *storage.Service
	premiumRepo PremiumRepository
}

// NewStickerService creates a new sticker service
func NewStickerService(stickerRepo StickerRepository, storageService *storage.Service) *StickerService {
	return &StickerService{
		stickerRepo: stickerRepo,
		storage:     storageService,
	}
}

// SetPremiumRepository sets the premium repository for subscription checks
func (s *StickerService) SetPremiumRepository(premiumRepo PremiumRepository) {
	s.premiumRepo = premiumRepo
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
	return s.CreateWithTier(ctx, serverID, name, tags, file, uploadedBy, models.StickerPackTierFree)
}

// CreateWithTier creates a new sticker with a specific tier requirement
func (s *StickerService) CreateWithTier(
	ctx context.Context,
	serverID *uuid.UUID,
	name string,
	tags []string,
	file *multipart.FileHeader,
	uploadedBy uuid.UUID,
	requiredTier models.StickerPackTier,
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
		ID:           uuid.New(),
		ServerID:     serverID,
		Name:         name,
		Tags:         tags,
		URL:          url,
		Format:       format,
		RequiredTier: requiredTier,
		CreatedBy:    uploadedBy,
		CreatedAt:    time.Now(),
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
func (s *StickerService) GetAvailable(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.Sticker, error) {
	allStickers, err := s.stickerRepo.GetAvailable(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Filter by tier access
	var filtered []*models.Sticker
	for _, sticker := range allStickers {
		if models.TierMeetsRequirement(userTier, sticker.RequiredTier) {
			filtered = append(filtered, sticker)
		}
	}
	return filtered, nil
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

// --- Sticker Pack Operations ---

// CreatePack creates a new sticker pack
func (s *StickerService) CreatePack(
	ctx context.Context,
	name string,
	description *string,
	iconURL *string,
	tier models.StickerPackTier,
	isGlobal bool,
	serverID *uuid.UUID,
	createdBy uuid.UUID,
) (*models.StickerPack, error) {
	if name == "" {
		return nil, ErrPackNameRequired
	}
	if len(name) > 100 {
		return nil, ErrPackNameTooLong
	}

	// For now, only global packs are supported unless serverID is provided
	if !isGlobal && serverID == nil {
		// Default to global pack if no server specified
		isGlobal = true
	}

	// Check user tier access for premium packs
	if tier != models.StickerPackTierFree {
		if s.premiumRepo != nil {
			userPremiumTier, err := s.premiumRepo.GetUserPremiumTier(ctx, createdBy)
			if err != nil || !models.TierMeetsRequirement(models.StickerTierFromString(string(userPremiumTier)), tier) {
				return nil, ErrPackTierAccess
			}
		}
	}

	pack := &models.StickerPack{
		ID:           uuid.New(),
		Name:         name,
		Description:  description,
		IconURL:      iconURL,
		Tier:         tier,
		StickerCount: 0,
		IsActive:     true,
		IsGlobal:     isGlobal,
		ServerID:     serverID,
		CreatedBy:    &createdBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.stickerRepo.CreatePack(ctx, pack); err != nil {
		return nil, fmt.Errorf("failed to create pack: %w", err)
	}

	return pack, nil
}

// GetPack retrieves a sticker pack by ID
func (s *StickerService) GetPack(ctx context.Context, packID uuid.UUID) (*models.StickerPack, error) {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return nil, err
	}
	if pack == nil {
		return nil, ErrPackNotFound
	}
	return pack, nil
}

// GetPackWithStickers retrieves a pack with all its stickers
func (s *StickerService) GetPackWithStickers(ctx context.Context, packID uuid.UUID, userTier models.StickerPackTier) (*models.StickerPack, error) {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return nil, err
	}
	if pack == nil {
		return nil, ErrPackNotFound
	}

	// Check tier access
	if !models.TierMeetsRequirement(userTier, pack.Tier) {
		return nil, ErrPackTierAccess
	}

	// Get stickers
	stickers, err := s.stickerRepo.GetStickersInPack(ctx, packID)
	if err != nil {
		return nil, err
	}

	// Filter stickers by user's tier
	var filteredStickers []*models.Sticker
	for _, sticker := range stickers {
		if models.TierMeetsRequirement(userTier, sticker.RequiredTier) {
			filteredStickers = append(filteredStickers, sticker)
		}
	}
	pack.Stickers = filteredStickers

	return pack, nil
}

// GetPacksByServer retrieves all packs for a server
func (s *StickerService) GetPacksByServer(ctx context.Context, serverID uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error) {
	packs, err := s.stickerRepo.GetPacksByServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Filter by tier access
	var filtered []*models.StickerPack
	for _, pack := range packs {
		if models.TierMeetsRequirement(userTier, pack.Tier) {
			filtered = append(filtered, pack)
		}
	}
	return filtered, nil
}

// GetGlobalPacks retrieves all global packs
func (s *StickerService) GetGlobalPacks(ctx context.Context, userTier models.StickerPackTier) ([]*models.StickerPack, error) {
	packs, err := s.stickerRepo.GetGlobalPacks(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by tier access
	var filtered []*models.StickerPack
	for _, pack := range packs {
		if models.TierMeetsRequirement(userTier, pack.Tier) {
			filtered = append(filtered, pack)
		}
	}
	return filtered, nil
}

// GetAvailablePacks retrieves all packs available to a user
func (s *StickerService) GetAvailablePacks(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error) {
	return s.stickerRepo.GetAvailablePacks(ctx, serverID, userTier)
}

// UpdatePack updates a sticker pack
func (s *StickerService) UpdatePack(ctx context.Context, packID uuid.UUID, name *string, description *string, iconURL *string, isActive *bool) (*models.StickerPack, error) {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return nil, err
	}
	if pack == nil {
		return nil, ErrPackNotFound
	}

	if name != nil {
		if len(*name) > 100 {
			return nil, ErrPackNameTooLong
		}
		pack.Name = *name
	}
	if description != nil {
		pack.Description = description
	}
	if iconURL != nil {
		pack.IconURL = iconURL
	}
	if isActive != nil {
		pack.IsActive = *isActive
	}

	if err := s.stickerRepo.UpdatePack(ctx, pack); err != nil {
		return nil, err
	}

	return pack, nil
}

// DeletePack deletes a sticker pack
func (s *StickerService) DeletePack(ctx context.Context, packID uuid.UUID) error {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return err
	}
	if pack == nil {
		return ErrPackNotFound
	}

	return s.stickerRepo.DeletePack(ctx, packID)
}

// AddStickerToPack adds a sticker to a pack
func (s *StickerService) AddStickerToPack(ctx context.Context, packID, stickerID uuid.UUID, position int, isDefault bool) error {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return err
	}
	if pack == nil {
		return ErrPackNotFound
	}

	sticker, err := s.stickerRepo.GetByID(ctx, stickerID)
	if err != nil {
		return err
	}
	if sticker == nil {
		return ErrStickerNotFound
	}

	return s.stickerRepo.AddStickerToPack(ctx, packID, stickerID, position, isDefault)
}

// RemoveStickerFromPack removes a sticker from a pack
func (s *StickerService) RemoveStickerFromPack(ctx context.Context, packID, stickerID uuid.UUID) error {
	pack, err := s.stickerRepo.GetPackByID(ctx, packID)
	if err != nil {
		return err
	}
	if pack == nil {
		return ErrPackNotFound
	}

	return s.stickerRepo.RemoveStickerFromPack(ctx, packID, stickerID)
}

// GetUserTierFromPremiumRepo gets the user's premium tier from the premium repository
func (s *StickerService) GetUserTierFromPremiumRepo(ctx context.Context, userID uuid.UUID) (models.StickerPackTier, error) {
	if s.premiumRepo == nil {
		return models.StickerPackTierFree, nil
	}

	tier, err := s.premiumRepo.GetUserPremiumTier(ctx, userID)
	if err != nil {
		return models.StickerPackTierFree, nil
	}

	return models.StickerTierFromString(string(tier)), nil
}
