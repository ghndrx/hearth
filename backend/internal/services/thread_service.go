package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrThreadNotFound   = errors.New("thread not found")
	ErrThreadArchived   = errors.New("thread is archived")
	ErrThreadLocked     = errors.New("thread is locked")
	ErrNotThreadMember  = errors.New("not a thread member")
	ErrNotThreadOwner   = errors.New("not the thread owner")
	ErrInvalidAutoArchive = errors.New("invalid auto archive duration")
)

// ThreadRepository defines thread data access
type ThreadRepository interface {
	Create(ctx context.Context, thread *models.Thread) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	Update(ctx context.Context, thread *models.Thread) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	Archive(ctx context.Context, id uuid.UUID) error
	Unarchive(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, threadID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error
	IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error)
	GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error)
	CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error
}

// ThreadService handles thread-related business logic
type ThreadService struct {
	threadRepo  ThreadRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	eventBus    EventBus
}

// NewThreadService creates a new thread service
func NewThreadService(
	threadRepo ThreadRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	eventBus EventBus,
) *ThreadService {
	return &ThreadService{
		threadRepo:  threadRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
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
		OwnerID:         creatorID,
		Name:            name,
		MessageCount:    0,
		MemberCount:     1,
		Archived:        false,
		AutoArchive:     archiveDuration,
		Locked:          false,
		CreatedAt:       time.Now(),
	}

	if err := s.threadRepo.Create(ctx, thread); err != nil {
		return nil, err
	}

	s.eventBus.Publish("thread.created", &ThreadCreatedEvent{
		Thread:    thread,
		ChannelID: channelID,
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
			// TODO: Check MANAGE_THREADS permission
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
			// TODO: Check MANAGE_THREADS permission
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

	return nil
}

// Events

type ThreadCreatedEvent struct {
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

type ThreadMessageCreatedEvent struct {
	Message  *models.ThreadMessage
	ThreadID uuid.UUID
}
