package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// SearchRepository defines the interface for search data access
type SearchRepository interface {
	// SearchMessages performs advanced message search with filters
	SearchMessages(ctx context.Context, opts SearchMessageOptions) (*SearchResult, error)
	// SearchUsers searches for users
	SearchUsers(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.PublicUser, error)
	// SearchChannels searches for channels
	SearchChannels(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.Channel, error)
}

// SearchMessageOptions contains all possible search filters
type SearchMessageOptions struct {
	// Required parameters
	Query string

	// Scope filters
	ServerID   *uuid.UUID
	ChannelID  *uuid.UUID
	ChannelIDs []uuid.UUID

	// Author filter
	AuthorID *uuid.UUID

	// Time range filters
	Before *time.Time
	After  *time.Time

	// Content filters
	HasAttachments *bool
	HasEmbeds      *bool
	HasReactions   *bool
	Pinned         *bool
	Mentions       []uuid.UUID

	// Pagination
	Limit  int
	Offset int

	// Requester for permission checks
	RequesterID uuid.UUID
}

// SearchResult contains search results with pagination info
type SearchResult struct {
	Messages []*models.Message
	Users    []*models.PublicUser
	Channels []*models.Channel
	Total    int
	HasMore  bool
}

// SearchService handles search-related business logic
type SearchService struct {
	searchRepo  SearchRepository
	messageRepo MessageRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	userRepo    UserRepository
	cache       CacheService
}

// NewSearchService creates a new search service
func NewSearchService(
	searchRepo SearchRepository,
	messageRepo MessageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	userRepo UserRepository,
	cache CacheService,
) *SearchService {
	return &SearchService{
		searchRepo:  searchRepo,
		messageRepo: messageRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		userRepo:    userRepo,
		cache:       cache,
	}
}

// SearchMessages searches for messages with filters
func (s *SearchService) SearchMessages(ctx context.Context, opts SearchMessageOptions) (*SearchResult, error) {
	// Validate and set defaults
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}

	// If server ID is provided, validate membership and get accessible channels
	if opts.ServerID != nil {
		if err := s.validateServerAccess(ctx, opts.ServerID, &opts.RequesterID); err != nil {
			return nil, err
		}

		// If no specific channel provided, search all accessible channels
		if opts.ChannelID == nil && len(opts.ChannelIDs) == 0 {
			channels, err := s.getAccessibleChannels(ctx, *opts.ServerID, opts.RequesterID)
			if err != nil {
				return nil, err
			}
			opts.ChannelIDs = channels
		}
	} else if opts.ChannelID != nil {
		// Validate channel access
		if err := s.validateChannelAccess(ctx, *opts.ChannelID, opts.RequesterID); err != nil {
			return nil, err
		}
	}

	// Perform search
	result, err := s.searchRepo.SearchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Enrich results with author info
	if len(result.Messages) > 0 {
		if err := s.enrichMessages(ctx, result.Messages); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// SearchUsers searches for users by username/display name
func (s *SearchService) SearchUsers(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.PublicUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// Validate server access if searching within a server
	if serverID != nil {
		if err := s.validateServerAccess(ctx, serverID, &requesterID); err != nil {
			return nil, err
		}
	}

	return s.searchRepo.SearchUsers(ctx, query, serverID, limit)
}

// SearchChannels searches for channels by name
func (s *SearchService) SearchChannels(ctx context.Context, query string, serverID *uuid.UUID, requesterID uuid.UUID, limit int) ([]*models.Channel, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// Validate server access if searching within a server
	if serverID != nil {
		if err := s.validateServerAccess(ctx, serverID, &requesterID); err != nil {
			return nil, err
		}
	}

	return s.searchRepo.SearchChannels(ctx, query, serverID, limit)
}

// validateServerAccess checks if user can access the server
func (s *SearchService) validateServerAccess(ctx context.Context, serverID *uuid.UUID, requesterID *uuid.UUID) error {
	if serverID == nil {
		return nil
	}

	member, err := s.serverRepo.GetMember(ctx, *serverID, *requesterID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrNotServerMember
	}
	return nil
}

// validateChannelAccess checks if user can access the channel
func (s *SearchService) validateChannelAccess(ctx context.Context, channelID uuid.UUID, requesterID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	// Check server membership for server channels
	if channel.ServerID != nil {
		return s.validateServerAccess(ctx, channel.ServerID, &requesterID)
	}

	// For DM channels, check if user is a participant
	isParticipant := false
	for _, recipient := range channel.Recipients {
		if recipient == requesterID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return ErrNoPermission
	}

	return nil
}

// getAccessibleChannels returns all channels the user can access in a server
func (s *SearchService) getAccessibleChannels(ctx context.Context, serverID uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	channels, err := s.channelRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var accessible []uuid.UUID
	for _, ch := range channels {
		// Skip non-text channels
		if ch.Type != models.ChannelTypeText && ch.Type != models.ChannelTypeAnnouncement {
			continue
		}
		accessible = append(accessible, ch.ID)
	}

	return accessible, nil
}

// enrichMessages adds author information to messages
func (s *SearchService) enrichMessages(ctx context.Context, messages []*models.Message) error {
	// Collect unique author IDs
	authorIDs := make(map[uuid.UUID]bool)
	for _, msg := range messages {
		authorIDs[msg.AuthorID] = true
	}

	// Fetch authors
	for authorID := range authorIDs {
		user, err := s.userRepo.GetByID(ctx, authorID)
		if err != nil {
			continue
		}
		if user != nil {
			publicUser := user.ToPublic()
			for _, msg := range messages {
				if msg.AuthorID == authorID {
					msg.Author = &publicUser
				}
			}
		}
	}

	return nil
}

// ParseSearchQuery parses a search query string and extracts filters
// Supports Discord-like syntax: from:@user in:#channel has:attachment before:2024-01-01
func ParseSearchQuery(query string) SearchMessageOptions {
	opts := SearchMessageOptions{
		Query: query,
	}

	// TODO: Implement query parsing for advanced filters
	// This would parse patterns like:
	// - from:@username -> AuthorID
	// - in:#channel -> ChannelID
	// - has:attachment -> HasAttachments=true
	// - has:embed -> HasEmbeds=true
	// - pinned:true -> Pinned=true
	// - before:2024-01-01 -> Before
	// - after:2024-01-01 -> After
	// - mentions:@user -> Mentions

	return opts
}
