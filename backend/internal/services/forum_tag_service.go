package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

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
	FilterThreadsByTags(ctx context.Context, channelID uuid.UUID, tagIDs []uuid.UUID, sortOrder int, limit, offset int) ([]models.Thread, int, error)
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

	threads, total, err := s.tagRepo.FilterThreadsByTags(ctx, channelID, filter.TagIDs, filter.SortOrder, limit, offset)
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
