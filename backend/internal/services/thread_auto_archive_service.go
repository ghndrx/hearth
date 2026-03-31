package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrAutoArchiveSettingsNotFound = errors.New("auto-archive settings not found")
	ErrAutoArchiveOverrideExists    = errors.New("auto-archive override already exists for this channel")
	ErrAutoArchiveNotAllowed        = errors.New("auto-archive override not allowed for this server")
	ErrInvalidAutoArchiveDuration   = errors.New("invalid auto-archive duration")
)

// ThreadAutoArchiveRepositoryInterface defines thread auto-archive data access
type ThreadAutoArchiveRepositoryInterface interface {
	CreateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error
	GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error)
	UpdateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error
	DeleteServerSettings(ctx context.Context, serverID uuid.UUID) error
	SetChannelOverride(ctx context.Context, override *models.ChannelAutoArchiveOverride) error
	GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error)
	DeleteChannelOverride(ctx context.Context, channelID uuid.UUID) error
	GetOrCreateThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error)
	UpdateThreadMeta(ctx context.Context, meta *models.ThreadAutoArchiveMeta) error
	SetThreadNextArchive(ctx context.Context, threadID uuid.UUID, nextArchiveAt *time.Time) error
	SetThreadArchiveEligible(ctx context.Context, threadID uuid.UUID, eligible bool) error
	BumpThreadOwnerActivity(ctx context.Context, threadID uuid.UUID) error
	GetThreadsReadyForArchive(ctx context.Context, limit int) ([]*models.ThreadAutoArchiveMeta, error)
	GetThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error)
	DeleteThreadMeta(ctx context.Context, threadID uuid.UUID) error
	GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error)
	GetChannelDuration(ctx context.Context, channelID, serverID uuid.UUID) (int, error)
}

// ThreadAutoArchiveService handles thread auto-archive business logic
type ThreadAutoArchiveService struct {
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface
	threadRepo      ThreadRepository
	channelRepo     ChannelRepository
	serverRepo      ServerRepository
	permService     PermissionServiceInterface
	eventBus        EventBus
}

// NewThreadAutoArchiveService creates a new thread auto-archive service
func NewThreadAutoArchiveService(
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface,
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService PermissionServiceInterface,
	eventBus EventBus,
) *ThreadAutoArchiveService {
	return &ThreadAutoArchiveService{
		autoArchiveRepo: autoArchiveRepo,
		threadRepo:       threadRepo,
		channelRepo:      channelRepo,
		serverRepo:       serverRepo,
		permService:      permService,
		eventBus:         eventBus,
	}
}

// GetOrCreateServerSettings retrieves or creates server auto-archive settings
func (s *ThreadAutoArchiveService) GetOrCreateServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	settings, err := s.autoArchiveRepo.GetServerSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if settings != nil {
		return settings, nil
	}

	// Create default settings
	now := time.Now()
	settings = &models.ThreadAutoArchiveSettings{
		ID:                    uuid.New(),
		ServerID:              serverID,
		DefaultDuration:       1440, // 24 hours
		AllowOverride:         true,
		ArchiveDurationOptions: []int{60, 1440, 4320, 10080},
		RequirePostAuthor:     false,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.autoArchiveRepo.CreateServerSettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// UpdateServerSettings updates server auto-archive settings
func (s *ThreadAutoArchiveService) UpdateServerSettings(ctx context.Context, serverID, requesterID uuid.UUID, req models.UpdateThreadAutoArchiveSettingsRequest) (*models.ThreadAutoArchiveSettings, error) {
	// Verify server exists and requester is an admin
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil || server == nil {
		return nil, ErrServerNotFound
	}

	// Check if user is owner
	isOwner := server.OwnerID == requesterID
	
	// Check if user has admin permission
	hasAdmin, err := s.permService.HasPermission(ctx, serverID, requesterID, models.PermAdministrator)
	if err != nil {
		return nil, err
	}
	if !isOwner && !hasAdmin {
		return nil, ErrMissingAdministrator
	}

	settings, err := s.autoArchiveRepo.GetServerSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		// Create default first
		settings, err = s.GetOrCreateServerSettings(ctx, serverID)
		if err != nil {
			return nil, err
		}
	}

	// Apply updates
	if req.DefaultDuration != nil {
		if !isValidDuration(*req.DefaultDuration) {
			return nil, ErrInvalidAutoArchiveDuration
		}
		settings.DefaultDuration = *req.DefaultDuration
	}
	if req.AllowOverride != nil {
		settings.AllowOverride = *req.AllowOverride
	}
	if req.ArchiveDurationOptions != nil {
		for _, d := range *req.ArchiveDurationOptions {
			if !isValidDuration(d) {
				return nil, ErrInvalidAutoArchiveDuration
			}
		}
		settings.ArchiveDurationOptions = *req.ArchiveDurationOptions
	}
	if req.RequirePostAuthor != nil {
		settings.RequirePostAuthor = *req.RequirePostAuthor
	}
	settings.UpdatedAt = time.Now()

	if err := s.autoArchiveRepo.UpdateServerSettings(ctx, settings); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish(EventThreadAutoArchiveSettingsUpdated, map[string]interface{}{
		"server_id": serverID,
		"settings":  settings,
	})

	return settings, nil
}

// GetServerSettings retrieves server auto-archive settings
func (s *ThreadAutoArchiveService) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	return s.autoArchiveRepo.GetServerSettings(ctx, serverID)
}

// SetChannelOverride sets a channel-level auto-archive override
func (s *ThreadAutoArchiveService) SetChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID, req models.SetChannelAutoArchiveOverrideRequest) (*models.ChannelAutoArchiveOverride, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil || channel == nil {
		return nil, ErrChannelNotFound
	}

	if channel.ServerID == nil {
		return nil, ErrChannelTypeNotSupported
	}

	serverID := *channel.ServerID

	// Check if override is allowed
	settings, err := s.autoArchiveRepo.GetServerSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if settings == nil || !settings.AllowOverride {
		return nil, ErrAutoArchiveNotAllowed
	}

	// Validate duration
	if !isValidDuration(req.AutoArchiveDuration) {
		return nil, ErrInvalidAutoArchiveDuration
	}

	// Verify requester is admin
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil || server == nil {
		return nil, ErrServerNotFound
	}
	isOwner := server.OwnerID == requesterID
	hasAdmin, err := s.permService.HasPermission(ctx, serverID, requesterID, models.PermAdministrator)
	if err != nil {
		return nil, err
	}
	if !isOwner && !hasAdmin {
		return nil, ErrMissingAdministrator
	}

	now := time.Now()
	override := &models.ChannelAutoArchiveOverride{
		ID:                  uuid.New(),
		ChannelID:           channelID,
		AutoArchiveDuration: req.AutoArchiveDuration,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.autoArchiveRepo.SetChannelOverride(ctx, override); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish(EventThreadAutoArchiveSettingsUpdated, map[string]interface{}{
		"channel_id": channelID,
		"server_id":  serverID,
		"override":   override,
	})

	return override, nil
}

// GetChannelOverride retrieves channel-level auto-archive override
func (s *ThreadAutoArchiveService) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	return s.autoArchiveRepo.GetChannelOverride(ctx, channelID)
}

// DeleteChannelOverride removes channel-level auto-archive override
func (s *ThreadAutoArchiveService) DeleteChannelOverride(ctx context.Context, channelID, requesterID uuid.UUID) error {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil || channel == nil {
		return ErrChannelNotFound
	}

	if channel.ServerID == nil {
		return ErrChannelTypeNotSupported
	}

	// Verify requester is admin
	server, err := s.serverRepo.GetByID(ctx, *channel.ServerID)
	if err != nil || server == nil {
		return ErrServerNotFound
	}
	isOwner := server.OwnerID == requesterID
	hasAdmin, err := s.permService.HasPermission(ctx, *channel.ServerID, requesterID, models.PermAdministrator)
	if err != nil {
		return err
	}
	if !isOwner && !hasAdmin {
		return ErrMissingAdministrator
	}

	return s.autoArchiveRepo.DeleteChannelOverride(ctx, channelID)
}

// GetThreadAutoArchiveStatus retrieves auto-archive status for a thread
func (s *ThreadAutoArchiveService) GetThreadAutoArchiveStatus(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveResponse, error) {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		return nil, ErrThreadNotFound
	}

	meta, err := s.autoArchiveRepo.GetOrCreateThreadMeta(ctx, threadID)
	if err != nil {
		return nil, err
	}

	status := "active"
	if thread.Archived {
		status = "archived"
	} else if meta.NextArchiveAt != nil {
		if meta.ArchiveEligible && !meta.NextArchiveAt.After(time.Now()) {
			status = "ready"
		} else {
			status = "scheduled"
		}
	}

	return &models.ThreadAutoArchiveResponse{
		ThreadID:      threadID,
		NextArchiveAt: meta.NextArchiveAt,
		Eligible:      meta.ArchiveEligible,
		Status:        status,
	}, nil
}

// UpdateThreadAutoArchive updates a thread's auto-archive metadata after activity
func (s *ThreadAutoArchiveService) UpdateThreadAutoArchive(ctx context.Context, threadID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		return ErrThreadNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil || channel == nil {
		return ErrChannelNotFound
	}

	var serverID uuid.UUID
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}

	// Get the effective auto-archive duration
	duration := 1440 // Default
	if serverID != uuid.Nil {
		duration, err = s.autoArchiveRepo.GetChannelDuration(ctx, thread.ParentChannelID, serverID)
		if err != nil {
			return err
		}
	}

	// Get or create meta
	meta, err := s.autoArchiveRepo.GetOrCreateThreadMeta(ctx, threadID)
	if err != nil {
		return err
	}

	// Calculate next archive time based on last activity
	nextArchive := time.Now().Add(time.Duration(duration) * time.Minute)
	meta.NextArchiveAt = &nextArchive
	meta.ArchiveEligible = true

	// Check if bumped by owner
	if meta.LastActivityUserID != nil && thread.OwnerID == *meta.LastActivityUserID {
		meta.BumpedByOwner = true
	}

	if err := s.autoArchiveRepo.UpdateThreadMeta(ctx, meta); err != nil {
		return err
	}

	// Emit event
	s.eventBus.Publish(EventThreadAutoArchiveBumped, map[string]interface{}{
		"thread_id": threadID,
		"next_archive_at": nextArchive,
	})

	return nil
}

// GetServerStats retrieves auto-archive statistics for a server
func (s *ThreadAutoArchiveService) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	return s.autoArchiveRepo.GetServerStats(ctx, serverID)
}

// ArchiveThread archives a thread (called by background worker)
func (s *ThreadAutoArchiveService) ArchiveThread(ctx context.Context, threadID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		return ErrThreadNotFound
	}

	if thread.Archived {
		return nil // Already archived
	}

	if err := s.threadRepo.Archive(ctx, threadID); err != nil {
		return err
	}

	// Clear the meta
	meta, _ := s.autoArchiveRepo.GetThreadMeta(ctx, threadID)
	if meta != nil {
		meta.NextArchiveAt = nil
		meta.ArchiveEligible = false
		s.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
	}

	// Emit event
	s.eventBus.Publish(EventThreadAutoArchived, map[string]interface{}{
		"thread_id":   threadID,
		"channel_id":  thread.ParentChannelID,
		"auto_archive": true,
	})

	return nil
}

// Helper functions

func isValidDuration(d int) bool {
	validDurations := []int{60, 1440, 4320, 10080}
	for _, valid := range validDurations {
		if d == valid {
			return true
		}
	}
	return false
}

// Event types for thread auto-archive
const (
	EventThreadAutoArchiveSettingsUpdated = "thread_auto_archive.settings_updated"
	EventThreadAutoArchiveBumped         = "thread_auto_archive.bumped"
	EventThreadAutoArchived              = "thread_auto_archived"
)
