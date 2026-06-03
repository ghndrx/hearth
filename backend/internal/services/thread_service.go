package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrThreadNotFound     = errors.New("thread not found")
	ErrThreadArchived     = errors.New("thread is archived")
	ErrThreadLocked       = errors.New("thread is locked")
	ErrNotThreadMember    = errors.New("not a thread member")
	ErrNotThreadOwner     = errors.New("not the thread owner")
	ErrInvalidAutoArchive = errors.New("invalid auto archive duration")
)

// ThreadRepository defines thread data access
type ThreadRepository interface {
	Create(ctx context.Context, thread *models.Thread) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	GetByParentMessageID(ctx context.Context, messageID uuid.UUID) (*models.Thread, error)
	Update(ctx context.Context, thread *models.Thread) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	GetThreadsPaginated(ctx context.Context, channelID uuid.UUID, sortOrder int, limit, offset int, includeArchived bool) ([]models.Thread, int, error)
	GetThreadCount(ctx context.Context, channelID uuid.UUID, includeArchived bool) (int, error)
	GetTotalMessageCount(ctx context.Context, channelID uuid.UUID) (int, error)
	Archive(ctx context.Context, id uuid.UUID) error
	Unarchive(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, threadID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error
	IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error)
	GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error)
	CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error
	// Notification preferences
	GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error)
	SetNotificationPreference(ctx context.Context, pref *models.ThreadNotificationPreference) error
	DeleteNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) error
	// Presence
	SetPresence(ctx context.Context, threadID, userID uuid.UUID) error
	RemovePresence(ctx context.Context, threadID, userID uuid.UUID) error
	GetActiveViewers(ctx context.Context, threadID uuid.UUID) ([]models.ThreadPresenceUser, error)
	UpdatePresenceHeartbeat(ctx context.Context, threadID, userID uuid.UUID) error
}

// ThreadService handles thread-related business logic
type ThreadService struct {
	threadRepo  ThreadRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	eventBus    EventBus
}

// NewThreadService creates a new thread service
func NewThreadService(
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *ThreadService {
	return &ThreadService{
		threadRepo:  threadRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		eventBus:    eventBus,
	}
}

// CreateThread creates a new thread in a channel
func (s *ThreadService) CreateThread(
	ctx context.Context,
	channelID uuid.UUID,
	creatorID uuid.UUID,
	name string,
	autoArchive *int,
	parentMessageID *uuid.UUID,
	tagIDs []uuid.UUID,
) (*models.Thread, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// For server channels, verify membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, creatorID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// If a thread already exists for this parent message, return it
	if parentMessageID != nil {
		existing, err := s.threadRepo.GetByParentMessageID(ctx, *parentMessageID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	// Validate auto archive duration
	archiveDuration := models.AutoArchive24Hour // Default: 24 hours
	if autoArchive != nil {
		switch *autoArchive {
		case models.AutoArchive1Hour, models.AutoArchive24Hour, models.AutoArchive3Day, models.AutoArchive1Week:
			archiveDuration = *autoArchive
		default:
			return nil, ErrInvalidAutoArchive
		}
	}

	thread := &models.Thread{
		ID:              uuid.New(),
		ParentChannelID: channelID,
		ParentMessageID: parentMessageID,
		OwnerID:         creatorID,
		Name:            name,
		MessageCount:    0,
		MemberCount:     1,
		Archived:        false,
		AutoArchive:     archiveDuration,
		Locked:          false,
		CreatedAt:       time.Now(),
		AppliedTags:     tagIDs,
		IsPinned:        false,
		PinWeight:       0,
	}

	if err := s.threadRepo.Create(ctx, thread); err != nil {
		return nil, err
	}

	s.eventBus.Publish("thread.created", &ThreadCreatedEvent{
		Thread:    thread,
		ChannelID: channelID,
	})

	// Also publish forum.thread_created for WebSocket bridge
	s.eventBus.Publish("forum.thread_created", &ThreadCreatedEvent{
		Thread:    thread,
		ChannelID: channelID,
	})

	return thread, nil
}

// UpdateThread updates a thread's name, archived, or locked state
func (s *ThreadService) UpdateThread(ctx context.Context, threadID uuid.UUID, requesterID uuid.UUID, req models.UpdateThreadRequest) (*models.Thread, error) {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	// Only thread owner or server moderators can update
	if thread.OwnerID != requesterID {
		channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
		if err != nil {
			return nil, err
		}
		if channel != nil && channel.ServerID != nil {
			member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
			if err != nil || member == nil {
				return nil, ErrNotServerMember
			}
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageThreads); err != nil {
					return nil, err
				}
			}
		} else {
			return nil, ErrNotThreadOwner
		}
	}

	if req.Name != nil {
		thread.Name = *req.Name
	}
	if req.Archived != nil {
		thread.Archived = *req.Archived
		if *req.Archived {
			now := time.Now()
			thread.ArchivedAt = &now
		} else {
			thread.ArchivedAt = nil
		}
	}
	if req.Locked != nil {
		thread.Locked = *req.Locked
	}
	if req.AutoArchive != nil {
		switch *req.AutoArchive {
		case models.AutoArchive1Hour, models.AutoArchive24Hour, models.AutoArchive3Day, models.AutoArchive1Week:
			thread.AutoArchive = *req.AutoArchive
		default:
			return nil, ErrInvalidAutoArchive
		}
	}

	if err := s.threadRepo.Update(ctx, thread); err != nil {
		return nil, err
	}

	s.eventBus.Publish("thread.updated", &ThreadUpdatedEvent{
		Thread:    thread,
		ChannelID: thread.ParentChannelID,
	})

	// Also publish forum.thread_updated for WebSocket bridge
	s.eventBus.Publish("forum.thread_updated", &ThreadUpdatedEvent{
		Thread:    thread,
		ChannelID: thread.ParentChannelID,
	})

	return thread, nil
}

// GetThread retrieves a thread by ID
func (s *ThreadService) GetThread(ctx context.Context, threadID uuid.UUID) (*models.Thread, error) {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}
	return thread, nil
}

// GetThreadMessages retrieves messages from a thread with pagination
func (s *ThreadService) GetThreadMessages(
	ctx context.Context,
	threadID uuid.UUID,
	requesterID uuid.UUID,
	before *uuid.UUID,
	limit int,
) ([]*models.ThreadMessage, error) {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	// Get the parent channel to check permissions
	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil {
		return nil, err
	}

	// For server channels, verify membership
	if channel != nil && channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.threadRepo.GetMessages(ctx, threadID, before, limit)
}

// SendThreadMessage sends a message to a thread
func (s *ThreadService) SendThreadMessage(
	ctx context.Context,
	threadID uuid.UUID,
	authorID uuid.UUID,
	content string,
) (*models.ThreadMessage, error) {
	// Verify thread exists and is not archived/locked
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	if thread.Archived {
		return nil, ErrThreadArchived
	}
	if thread.Locked {
		return nil, ErrThreadLocked
	}

	// Get the parent channel to check permissions
	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil {
		return nil, err
	}

	// For server channels, verify membership
	if channel != nil && channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, authorID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// Add user to thread if not already a member
	isMember, err := s.threadRepo.IsMember(ctx, threadID, authorID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		if err := s.threadRepo.AddMember(ctx, threadID, authorID); err != nil {
			return nil, err
		}
	}

	msg, err := s.threadRepo.CreateMessage(ctx, threadID, authorID, content)
	if err != nil {
		return nil, err
	}

	s.eventBus.Publish("thread.message_created", &ThreadMessageCreatedEvent{
		Message:  msg,
		ThreadID: threadID,
	})

	return msg, nil
}

// ArchiveThread archives a thread
func (s *ThreadService) ArchiveThread(ctx context.Context, threadID uuid.UUID, requesterID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Only thread owner or server moderators can archive
	if thread.OwnerID != requesterID {
		// Check if user has manage threads permission in server
		channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
		if err != nil {
			return err
		}
		if channel != nil && channel.ServerID != nil {
			member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
			if err != nil || member == nil {
				return ErrNotServerMember
			}
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageThreads); err != nil {
					return err
				}
			}
		} else {
			return ErrNotThreadOwner
		}
	}

	if err := s.threadRepo.Archive(ctx, threadID); err != nil {
		return err
	}

	s.eventBus.Publish("thread.archived", &ThreadArchivedEvent{
		ThreadID:  threadID,
		ChannelID: thread.ParentChannelID,
	})

	return nil
}

// UnarchiveThread unarchives a thread
func (s *ThreadService) UnarchiveThread(ctx context.Context, threadID uuid.UUID, requesterID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Only thread owner or server moderators can unarchive
	if thread.OwnerID != requesterID {
		channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
		if err != nil {
			return err
		}
		if channel != nil && channel.ServerID != nil {
			member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
			if err != nil || member == nil {
				return ErrNotServerMember
			}
		} else {
			return ErrNotThreadOwner
		}
	}

	if err := s.threadRepo.Unarchive(ctx, threadID); err != nil {
		return err
	}

	s.eventBus.Publish("thread.unarchived", &ThreadUnarchivedEvent{
		ThreadID:  threadID,
		ChannelID: thread.ParentChannelID,
	})

	return nil
}

// GetChannelThreads retrieves all threads for a channel
func (s *ThreadService) GetChannelThreads(
	ctx context.Context,
	channelID uuid.UUID,
	requesterID uuid.UUID,
	includeArchived bool,
) ([]*models.Thread, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// For server channels, verify membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	if includeArchived {
		return s.threadRepo.GetByChannelID(ctx, channelID)
	}
	return s.threadRepo.GetActiveByChannelID(ctx, channelID)
}

// GetChannelThreadsPaginated retrieves threads for a channel with pagination
func (s *ThreadService) GetChannelThreadsPaginated(
	ctx context.Context,
	channelID uuid.UUID,
	requesterID uuid.UUID,
	sortOrder int,
	limit, offset int,
	includeArchived bool,
) ([]models.Thread, int, error) {
	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, 0, err
	}
	if channel == nil {
		return nil, 0, ErrChannelNotFound
	}

	// For server channels, verify membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
		if err != nil || member == nil {
			return nil, 0, ErrNotServerMember
		}
	}

	return s.threadRepo.GetThreadsPaginated(ctx, channelID, sortOrder, limit, offset, includeArchived)
}

// JoinThread adds a user to a thread
func (s *ThreadService) JoinThread(ctx context.Context, threadID, userID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	return s.threadRepo.AddMember(ctx, threadID, userID)
}

// LeaveThread removes a user from a thread
func (s *ThreadService) LeaveThread(ctx context.Context, threadID, userID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	return s.threadRepo.RemoveMember(ctx, threadID, userID)
}

// DeleteThread deletes a thread
func (s *ThreadService) DeleteThread(ctx context.Context, threadID, requesterID uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Only thread owner or server moderators can delete
	if thread.OwnerID != requesterID {
		channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
		if err != nil {
			return err
		}
		if channel != nil && channel.ServerID != nil {
			member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, requesterID)
			if err != nil || member == nil {
				return ErrNotServerMember
			}
			if s.permService != nil {
				if err := s.permService.RequirePermission(ctx, *channel.ServerID, requesterID, models.PermManageThreads); err != nil {
					return err
				}
			}
		} else {
			return ErrNotThreadOwner
		}
	}

	if err := s.threadRepo.Delete(ctx, threadID); err != nil {
		return err
	}

	s.eventBus.Publish("thread.deleted", &ThreadDeletedEvent{
		ThreadID:  threadID,
		ChannelID: thread.ParentChannelID,
	})

	// Also publish forum.thread_deleted for WebSocket bridge
	s.eventBus.Publish("forum.thread_deleted", &ThreadDeletedEvent{
		ThreadID:  threadID,
		ChannelID: thread.ParentChannelID,
	})

	return nil
}

// ============================================================================
// Thread Notification Preferences
// ============================================================================

// GetNotificationPreference gets a user's notification preference for a thread
func (s *ThreadService) GetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadNotificationPreference, error) {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	return s.threadRepo.GetNotificationPreference(ctx, threadID, userID)
}

// SetNotificationPreference sets a user's notification preference for a thread
func (s *ThreadService) SetNotificationPreference(ctx context.Context, threadID, userID uuid.UUID, level models.ThreadNotificationLevel) error {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Validate level
	switch level {
	case models.ThreadNotifyAll, models.ThreadNotifyMentions, models.ThreadNotifyNone:
		// Valid
	default:
		return errors.New("invalid notification level")
	}

	pref := &models.ThreadNotificationPreference{
		ThreadID: threadID,
		UserID:   userID,
		Level:    level,
	}

	if err := s.threadRepo.SetNotificationPreference(ctx, pref); err != nil {
		return err
	}

	s.eventBus.Publish("thread.notification_preference_updated", &ThreadNotificationUpdatedEvent{
		ThreadID: threadID,
		UserID:   userID,
		Level:    level,
	})

	return nil
}

// ============================================================================
// Thread Presence (Active Viewers)
// ============================================================================

// EnterThread marks a user as actively viewing a thread
func (s *ThreadService) EnterThread(ctx context.Context, threadID, userID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	// Set presence
	if err := s.threadRepo.SetPresence(ctx, threadID, userID); err != nil {
		return nil, err
	}

	// Get all active viewers
	viewers, err := s.threadRepo.GetActiveViewers(ctx, threadID)
	if err != nil {
		return nil, err
	}

	response := &models.ThreadPresenceResponse{
		ThreadID:    threadID,
		ActiveUsers: viewers,
	}

	// Broadcast presence update
	s.eventBus.Publish("thread.presence_update", &ThreadPresenceUpdateEvent{
		ThreadID:    threadID,
		ActiveUsers: viewers,
	})

	return response, nil
}

// ExitThread removes a user's presence from a thread (stops viewing)
func (s *ThreadService) ExitThread(ctx context.Context, threadID, userID uuid.UUID) error {
	// Remove presence (don't error if thread doesn't exist)
	if err := s.threadRepo.RemovePresence(ctx, threadID, userID); err != nil {
		return err
	}

	// Get remaining active viewers
	viewers, _ := s.threadRepo.GetActiveViewers(ctx, threadID)

	// Broadcast presence update
	s.eventBus.Publish("thread.presence_update", &ThreadPresenceUpdateEvent{
		ThreadID:    threadID,
		ActiveUsers: viewers,
	})

	return nil
}

// GetActiveViewers gets users currently viewing a thread
func (s *ThreadService) GetActiveViewers(ctx context.Context, threadID uuid.UUID) (*models.ThreadPresenceResponse, error) {
	// Verify thread exists
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	viewers, err := s.threadRepo.GetActiveViewers(ctx, threadID)
	if err != nil {
		return nil, err
	}

	return &models.ThreadPresenceResponse{
		ThreadID:    threadID,
		ActiveUsers: viewers,
	}, nil
}

// HeartbeatPresence updates the last_seen_at for an active viewer
func (s *ThreadService) HeartbeatPresence(ctx context.Context, threadID, userID uuid.UUID) error {
	return s.threadRepo.UpdatePresenceHeartbeat(ctx, threadID, userID)
}

// Events

type ThreadCreatedEvent struct {
	Thread    *models.Thread
	ChannelID uuid.UUID
}

type ThreadUpdatedEvent struct {
	Thread    *models.Thread
	ChannelID uuid.UUID
}

type ThreadArchivedEvent struct {
	ThreadID  uuid.UUID
	ChannelID uuid.UUID
}

type ThreadUnarchivedEvent struct {
	ThreadID  uuid.UUID
	ChannelID uuid.UUID
}

type ThreadDeletedEvent struct {
	ThreadID  uuid.UUID
	ChannelID uuid.UUID
}

type ThreadPinnedEvent struct {
	ThreadID  uuid.UUID
	ChannelID uuid.UUID
	Pinned   bool
}

type ThreadMessageCreatedEvent struct {
	Message  *models.ThreadMessage
	ThreadID uuid.UUID
}

type ThreadNotificationUpdatedEvent struct {
	ThreadID uuid.UUID
	UserID   uuid.UUID
	Level    models.ThreadNotificationLevel
}

type ThreadPresenceUpdateEvent struct {
	ThreadID    uuid.UUID
	ActiveUsers []models.ThreadPresenceUser
}

var (
	ErrAutoArchiveSettingsNotFound = errors.New("auto-archive settings not found")
	ErrAutoArchiveOverrideExists   = errors.New("auto-archive override already exists for this channel")
	ErrAutoArchiveNotAllowed       = errors.New("auto-archive override not allowed for this server")
	ErrInvalidAutoArchiveDuration  = errors.New("invalid auto-archive duration")
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
		threadRepo:      threadRepo,
		channelRepo:     channelRepo,
		serverRepo:      serverRepo,
		permService:     permService,
		eventBus:        eventBus,
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
		ID:                     uuid.New(),
		ServerID:               serverID,
		DefaultDuration:        1440, // 24 hours
		AllowOverride:          true,
		ArchiveDurationOptions: []int{60, 1440, 4320, 10080},
		RequirePostAuthor:      false,
		CreatedAt:              now,
		UpdatedAt:              now,
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
		"thread_id":       threadID,
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
		"thread_id":    threadID,
		"channel_id":   thread.ParentChannelID,
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
	EventThreadAutoArchiveBumped          = "thread_auto_archive.bumped"
	EventThreadAutoArchived               = "thread_auto_archived"
)

// ThreadAutoArchiveWorker processes threads that are ready for auto-archive
type ThreadAutoArchiveWorker struct {
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface
	threadRepo      ThreadRepository
	channelRepo     ChannelRepository
	eventBus        EventBus

	stopCh        chan struct{}
	wg            sync.WaitGroup
	batchSize     int
	checkInterval time.Duration
	isRunning     bool
	mu            sync.Mutex
}

// NewThreadAutoArchiveWorker creates a new auto-archive worker
func NewThreadAutoArchiveWorker(
	autoArchiveRepo ThreadAutoArchiveRepositoryInterface,
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	eventBus EventBus,
) *ThreadAutoArchiveWorker {
	return &ThreadAutoArchiveWorker{
		autoArchiveRepo: autoArchiveRepo,
		threadRepo:      threadRepo,
		channelRepo:     channelRepo,
		eventBus:        eventBus,
		stopCh:          make(chan struct{}),
		batchSize:       50,
		checkInterval:   1 * time.Minute,
	}
}

// Start begins the background worker
func (w *ThreadAutoArchiveWorker) Start() {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	log.Println("Thread auto-archive worker started")
}

// Stop gracefully stops the worker
func (w *ThreadAutoArchiveWorker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()

	log.Println("Thread auto-archive worker stopped")
}

func (w *ThreadAutoArchiveWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processReadyThreads()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processReadyThreads()
		}
	}
}

func (w *ThreadAutoArchiveWorker) processReadyThreads() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get threads ready for archive
	metas, err := w.autoArchiveRepo.GetThreadsReadyForArchive(ctx, w.batchSize)
	if err != nil {
		log.Printf("Error fetching threads ready for archive: %v", err)
		return
	}

	if len(metas) == 0 {
		return
	}

	log.Printf("Processing %d threads for auto-archive", len(metas))

	for _, meta := range metas {
		select {
		case <-w.stopCh:
			return
		default:
			w.archiveThread(ctx, meta)
		}
	}
}

func (w *ThreadAutoArchiveWorker) archiveThread(ctx context.Context, meta *models.ThreadAutoArchiveMeta) {
	threadID := meta.ThreadID

	// Get thread details
	thread, err := w.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		log.Printf("Error getting thread %s: %v", threadID, err)
		return
	}

	// Skip if already archived
	if thread.Archived {
		// Clear the meta
		meta.NextArchiveAt = nil
		meta.ArchiveEligible = false
		w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
		return
	}

	// Check if owner has bumped since last check
	if meta.BumpedByOwner {
		// Reset bumped flag and update next archive time
		meta.BumpedByOwner = false
		meta.ArchiveEligible = false
		w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)

		// Emit bump event
		w.eventBus.Publish(EventThreadAutoArchiveBumped, map[string]interface{}{
			"thread_id":       threadID,
			"bumped_by_owner": true,
		})
		return
	}

	// Archive the thread
	if err := w.threadRepo.Archive(ctx, threadID); err != nil {
		log.Printf("Error archiving thread %s: %v", threadID, err)
		return
	}

	// Update meta
	meta.NextArchiveAt = nil
	meta.ArchiveEligible = false
	w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)

	// Emit archive event
	w.eventBus.Publish(EventThreadAutoArchived, map[string]interface{}{
		"thread_id":    threadID,
		"channel_id":   thread.ParentChannelID,
		"auto_archive": true,
	})

	log.Printf("Auto-archived thread %s", threadID)
}

// ProcessThreadActivity updates auto-archive metadata when a thread receives activity
func (w *ThreadAutoArchiveWorker) ProcessThreadActivity(ctx context.Context, threadID, userID uuid.UUID) error {
	thread, err := w.threadRepo.GetByID(ctx, threadID)
	if err != nil || thread == nil {
		return ErrThreadNotFound
	}

	// Skip if archived
	if thread.Archived {
		return nil
	}

	channel, err := w.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil || channel == nil {
		return ErrChannelNotFound
	}

	var serverID uuid.UUID
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}

	// Get the effective auto-archive duration
	duration := 1440 // Default 24 hours
	if serverID != uuid.Nil {
		duration, err = w.autoArchiveRepo.GetChannelDuration(ctx, thread.ParentChannelID, serverID)
		if err != nil {
			return err
		}
	}

	// Get or create meta
	meta, err := w.autoArchiveRepo.GetOrCreateThreadMeta(ctx, threadID)
	if err != nil {
		return err
	}

	// Update last activity
	meta.LastActivityAt = time.Now()
	meta.LastActivityUserID = &userID

	// Check if bumped by owner
	if thread.OwnerID == userID {
		meta.BumpedByOwner = true
	}

	// Calculate next archive time
	nextArchive := time.Now().Add(time.Duration(duration) * time.Minute)
	meta.NextArchiveAt = &nextArchive
	meta.ArchiveEligible = true

	return w.autoArchiveRepo.UpdateThreadMeta(ctx, meta)
}

// GetWorkerStatus returns the current status of the worker
func (w *ThreadAutoArchiveWorker) GetWorkerStatus() map[string]interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	return map[string]interface{}{
		"is_running":     w.isRunning,
		"batch_size":     w.batchSize,
		"check_interval": w.checkInterval.String(),
	}
}

// SetBatchSize sets the number of threads to process per batch
func (w *ThreadAutoArchiveWorker) SetBatchSize(size int) {
	if size > 0 && size <= 500 {
		w.batchSize = size
	}
}

// SetCheckInterval sets how often the worker checks for threads to archive
func (w *ThreadAutoArchiveWorker) SetCheckInterval(interval time.Duration) {
	if interval >= time.Second {
		w.checkInterval = interval
	}
}

var (
	ErrTagNotFound     = errors.New("tag not found")
	ErrTagNameExists   = errors.New("tag with this name already exists in this channel")
	ErrTagLimitReached = errors.New("maximum tag limit reached for this channel")
)

// ForumTagRepository defines forum tag data access
type ForumTagRepositoryInterface interface {
	Create(ctx context.Context, tag *models.ForumTag) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ForumTag, error)
	GetByChannel(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ForumTag, error)
	Update(ctx context.Context, tag *models.ForumTag) error
	Delete(ctx context.Context, id uuid.UUID) error
	ApplyTags(ctx context.Context, threadID uuid.UUID, tagIDs []uuid.UUID) error
	GetThreadTags(ctx context.Context, threadID uuid.UUID) ([]models.ForumTag, error)
	FilterThreads(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]models.Thread, int, error)
}

// ForumTagService handles forum tag business logic
type ForumTagService struct {
	tagRepo     ForumTagRepositoryInterface
	threadRepo  ThreadRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	eventBus   EventBus
}

// NewForumTagService creates a new forum tag service
func NewForumTagService(
	tagRepo ForumTagRepositoryInterface,
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *ForumTagService {
	return &ForumTagService{
		tagRepo:     tagRepo,
		threadRepo:  threadRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		eventBus:   eventBus,
	}
}

// MaxTagsPerChannel is the maximum number of tags allowed per forum channel
const MaxTagsPerChannel = 20

// CreateTag creates a new forum tag
func (s *ForumTagService) CreateTag(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateForumTagRequest) (*models.ForumTag, error) {
	// Verify channel exists and is a forum channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeForum {
		return nil, errors.New("channel is not a forum channel")
	}

	// Check permissions
	serverID := channel.ServerID
	if serverID == nil {
		return nil, errors.New("forum channel must belong to a server")
	}
	perms, err := s.permService.GetMemberPermissions(ctx, *serverID, userID)
	if err != nil {
		return nil, err
	}
	if perms&models.PermManageChannels == 0 {
		return nil, ErrForbidden
	}

	// Check tag limit
	existingTags, err := s.tagRepo.GetByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if len(existingTags) >= MaxTagsPerChannel {
		return nil, ErrTagLimitReached
	}

	// Check for duplicate name
	for _, t := range existingTags {
		if t.Name == req.Name {
			return nil, ErrTagNameExists
		}
	}

	// Calculate position
	position := len(existingTags) // Default to end
	if req.Position != nil && *req.Position >= 0 && *req.Position <= len(existingTags) {
		position = *req.Position
	}

	tag := &models.ForumTag{
		ID:        uuid.New(),
		ServerID:  *serverID,
		ChannelID: channelID,
		Name:      req.Name,
		Color:     req.Color,
		EmojiName: req.EmojiName,
		Moderated: req.Moderated,
		Position:  position,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}

	// Publish forum.tag_created event
	s.eventBus.Publish("forum.tag_created", &ForumTagCreatedEvent{
		Tag:       tag,
		ChannelID: channelID,
	})

	return tag, nil
}

// GetChannelTags returns all tags for a forum channel
func (s *ForumTagService) GetChannelTags(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error) {
	return s.tagRepo.GetByChannel(ctx, channelID)
}

// UpdateTag updates a forum tag
func (s *ForumTagService) UpdateTag(ctx context.Context, tagID, userID uuid.UUID, req *models.UpdateForumTagRequest) (*models.ForumTag, error) {
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}

	// Check permissions
	perms, err := s.permService.GetMemberPermissions(ctx, tag.ServerID, userID)
	if err != nil {
		return nil, err
	}
	if perms&models.PermManageChannels == 0 {
		return nil, ErrForbidden
	}

	if req.Name != nil {
		tag.Name = *req.Name
	}
	if req.Color != nil {
		tag.Color = req.Color
	}
	if req.EmojiName != nil {
		tag.EmojiName = req.EmojiName
	}
	if req.Moderated != nil {
		tag.Moderated = *req.Moderated
	}
	if req.Position != nil {
		tag.Position = *req.Position
	}

	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, err
	}

	// Publish forum.tag_updated event
	s.eventBus.Publish("forum.tag_updated", &ForumTagUpdatedEvent{
		Tag:       tag,
		ChannelID: tag.ChannelID,
	})

	return tag, nil
}

// DeleteTag deletes a forum tag
func (s *ForumTagService) DeleteTag(ctx context.Context, tagID, userID uuid.UUID) error {
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return err
	}
	if tag == nil {
		return ErrTagNotFound
	}

	// Check permissions
	perms, err := s.permService.GetMemberPermissions(ctx, tag.ServerID, userID)
	if err != nil {
		return err
	}
	if perms&models.PermManageChannels == 0 {
		return ErrForbidden
	}

	// Publish forum.tag_deleted event before deleting
	s.eventBus.Publish("forum.tag_deleted", &ForumTagDeletedEvent{
		TagID:     tagID,
		ChannelID: tag.ChannelID,
	})

	return s.tagRepo.Delete(ctx, tagID)
}

// ApplyTagsToThread applies tags to a forum post (thread)
func (s *ForumTagService) ApplyTagsToThread(ctx context.Context, threadID, userID uuid.UUID, tagIDs []uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	// Check permissions - owner or manage permission
	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil {
		return err
	}
	if channel == nil || channel.Type != models.ChannelTypeForum {
		return errors.New("thread is not in a forum channel")
	}

	serverID := channel.ServerID
	if serverID == nil {
		return errors.New("forum channel must belong to a server")
	}

	isOwner := thread.OwnerID == userID
	perms, err := s.permService.GetMemberPermissions(ctx, *serverID, userID)
	if err != nil {
		return err
	}
	canManage := perms&models.PermManageChannels != 0

	// If tag is moderated, only mods can apply it
	if !isOwner && !canManage {
		existingTags, err := s.tagRepo.GetByChannel(ctx, thread.ParentChannelID)
		if err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			for _, t := range existingTags {
				if t.ID == tagID && t.Moderated {
					return errors.New("cannot apply moderated tag without permission")
				}
			}
		}
	}

	return s.tagRepo.ApplyTags(ctx, threadID, tagIDs)
}

// GetThreadTags returns tags applied to a thread
func (s *ForumTagService) GetThreadTags(ctx context.Context, threadID uuid.UUID) ([]models.ForumTag, error) {
	return s.tagRepo.GetThreadTags(ctx, threadID)
}

// FilterForumPosts returns threads filtered by tags and sort order
func (s *ForumTagService) FilterForumPosts(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]models.Thread, []models.ForumTag, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}

	threads, total, err := s.tagRepo.FilterThreads(ctx, channelID, filter, limit, offset)
	if err != nil {
		return nil, nil, 0, err
	}

	// Fetch tags for each thread
	allTagIDs := make([]uuid.UUID, 0)
	for _, t := range threads {
		allTagIDs = append(allTagIDs, t.AppliedTags...)
	}

	var tagsForThreads []models.ForumTag
	if len(allTagIDs) > 0 {
		tagsForThreads, err = s.tagRepo.GetByIDs(ctx, allTagIDs)
		if err != nil {
			return nil, nil, 0, err
		}
	}

	return threads, tagsForThreads, total, nil
}

// PinThread pins or unpins a forum post
func (s *ForumTagService) PinThread(ctx context.Context, threadID, userID uuid.UUID, pin bool) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil {
		return err
	}
	if channel == nil || channel.Type != models.ChannelTypeForum {
		return errors.New("thread is not in a forum channel")
	}

	serverID := channel.ServerID
	if serverID == nil {
		return errors.New("forum channel must belong to a server")
	}

	// Only manage permission can pin
	perms, err := s.permService.GetMemberPermissions(ctx, *serverID, userID)
	if err != nil {
		return err
	}
	if perms&models.PermManageChannels == 0 {
		return ErrForbidden
	}

	thread.IsPinned = pin
	if pin && thread.PinWeight == 0 {
		thread.PinWeight = 1
	}
	if err := s.threadRepo.Update(ctx, thread); err != nil {
		return err
	}

	// Publish forum.thread_pinned event
	s.eventBus.Publish("forum.thread_pinned", &ThreadPinnedEvent{
		ThreadID:  threadID,
		ChannelID: thread.ParentChannelID,
		Pinned:   pin,
	})

	return nil
}

// Forum tag events

type ForumTagCreatedEvent struct {
	Tag       *models.ForumTag
	ChannelID uuid.UUID
}

type ForumTagUpdatedEvent struct {
	Tag       *models.ForumTag
	ChannelID uuid.UUID
}

type ForumTagDeletedEvent struct {
	TagID     uuid.UUID
	ChannelID uuid.UUID
}

// MarkThreadSolved marks a forum post as solved/answered or unmarks it
func (s *ForumTagService) MarkThreadSolved(ctx context.Context, threadID, userID uuid.UUID, solved bool, solvedMessageID *uuid.UUID) error {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread == nil {
		return ErrThreadNotFound
	}

	channel, err := s.channelRepo.GetByID(ctx, thread.ParentChannelID)
	if err != nil {
		return err
	}
	if channel == nil || channel.Type != models.ChannelTypeForum {
		return errors.New("thread is not in a forum channel")
	}

	serverID := channel.ServerID
	if serverID == nil {
		return errors.New("forum channel must belong to a server")
	}

	// Check permissions - only post owner, mods with PermMarkForumSolved, or mods with PermManageForumPosts
	perms, err := s.permService.GetMemberPermissions(ctx, *serverID, userID)
	if err != nil {
		return err
	}

	isOwner := thread.OwnerID == userID
	canMarkSolved := perms&models.PermMarkForumSolved != 0
	canManagePosts := perms&models.PermManageForumPosts != 0
	canManageChannels := perms&models.PermManageChannels != 0

	// Owner can mark their own posts as solved
	// Mods with appropriate permissions can mark any post as solved
	if !isOwner && !canMarkSolved && !canManagePosts && !canManageChannels {
		return ErrForbidden
	}

	thread.IsSolved = solved
	if solved {
		thread.SolvedBy = &userID
		thread.SolvedMessageID = solvedMessageID
		now := time.Now()
		thread.SolvedAt = &now
	} else {
		thread.SolvedBy = nil
		thread.SolvedAt = nil
		thread.SolvedMessageID = nil
	}

	return s.threadRepo.Update(ctx, thread)
}

// GetThreadSolvedMessage returns the message that solved the thread (if any)
func (s *ForumTagService) GetThreadSolvedMessage(ctx context.Context, threadID uuid.UUID) (*models.ThreadMessage, error) {
	thread, err := s.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, ErrThreadNotFound
	}

	if !thread.IsSolved || thread.SolvedMessageID == nil {
		return nil, nil
	}

	// We need to get the message from the thread_messages table
	// This requires a method on thread repo to get a specific message
	// For now, return nil if we don't have direct access
	return nil, nil
}

// GetForumChannelConfig returns the forum configuration for a channel
func (s *ForumTagService) GetForumChannelConfig(ctx context.Context, channelID uuid.UUID) (*models.ForumConfig, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeForum {
		return nil, errors.New("channel is not a forum channel")
	}

	// Parse forum_config JSONB if present
	if channel.ForumConfig == nil {
		return &models.ForumConfig{
			DefaultSortOrder: 0,
			DefaultLayout:    0,
			RequireTag:       false,
		}, nil
	}

	var config models.ForumConfig
	if err := json.Unmarshal(channel.ForumConfig, &config); err != nil {
		return nil, err
	}

	// Load available tags
	tags, err := s.tagRepo.GetByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	config.AvailableTags = tags

	return &config, nil
}

// UpdateForumChannelConfig updates the forum configuration for a channel
func (s *ForumTagService) UpdateForumChannelConfig(ctx context.Context, channelID, userID uuid.UUID, config *models.ForumConfig) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeForum {
		return errors.New("channel is not a forum channel")
	}

	serverID := channel.ServerID
	if serverID == nil {
		return errors.New("forum channel must belong to a server")
	}

	// Check permissions
	perms, err := s.permService.GetMemberPermissions(ctx, *serverID, userID)
	if err != nil {
		return err
	}
	if perms&models.PermManageChannels == 0 {
		return ErrForbidden
	}

	// Serialize config to JSONB
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return s.channelRepo.UpdateForumConfig(ctx, channelID, configJSON)
}