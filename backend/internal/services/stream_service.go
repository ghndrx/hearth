package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrLiveStreamNotFound      = errors.New("live stream not found")
	ErrLiveAlreadyStreaming    = errors.New("user is already streaming in this channel")
	ErrLiveNoActiveStream      = errors.New("no active stream in this channel")
	ErrLiveNotStreamer         = errors.New("user is not the streamer")
	ErrLiveCannotJoinOwnStream = errors.New("cannot join your own stream")
	ErrLiveChannelNotVoice     = errors.New("channel is not a voice channel")
	ErrLiveAlreadyViewing      = errors.New("user is already viewing this stream")
	ErrLiveNotViewing          = errors.New("user is not viewing this stream")
	ErrLiveStreamEnded         = errors.New("stream has ended")
)

// StreamRepository defines data access for live streams
type StreamRepository interface {
	Create(ctx context.Context, stream *models.LiveStream) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.LiveStream, error)
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*models.LiveStream, error)
	GetActiveByUser(ctx context.Context, userID uuid.UUID) (*models.LiveStream, error)
	Update(ctx context.Context, stream *models.LiveStream) error
	End(ctx context.Context, id uuid.UUID) error
	AddViewer(ctx context.Context, streamID, viewerID uuid.UUID) error
	RemoveViewer(ctx context.Context, streamID, viewerID uuid.UUID) error
	GetViewers(ctx context.Context, streamID uuid.UUID) ([]uuid.UUID, error)
	IsViewing(ctx context.Context, streamID, viewerID uuid.UUID) (bool, error)
}

// LiveStreamService handles live streaming business logic
type LiveStreamService struct {
	repo        StreamRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	userRepo    UserRepository
	permService *PermissionService
	eventBus    EventBus

	// In-memory cache of active streams for fast lookup
	activeStreams   map[uuid.UUID]*models.LiveStream // channelID -> Stream
	activeStreamsMu sync.RWMutex
}

// NewLiveStreamService creates a new live stream service
func NewLiveStreamService(
	repo StreamRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	userRepo UserRepository,
	permService *PermissionService,
	eventBus EventBus,
) *LiveStreamService {
	return &LiveStreamService{
		repo:          repo,
		channelRepo:   channelRepo,
		serverRepo:    serverRepo,
		userRepo:      userRepo,
		permService:   permService,
		eventBus:      eventBus,
		activeStreams: make(map[uuid.UUID]*models.LiveStream),
	}
}

// StartStream starts a live stream in a voice channel
func (s *LiveStreamService) StartStream(
	ctx context.Context,
	channelID uuid.UUID,
	userID uuid.UUID,
	streamType models.LiveStreamType,
	quality models.LiveStreamQuality,
) (*models.LiveStreamInfo, error) {
	// Validate channel exists and is a voice channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Check if channel is voice or stage type
	if channel.Type != models.ChannelTypeVoice && channel.Type != models.ChannelTypeStage {
		return nil, ErrLiveChannelNotVoice
	}

	// Check if user is a member of the server (for server channels)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}

		// Check STREAM permission (video streaming permission)
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermVideo); err != nil {
				return nil, err
			}
		}
	}

	// Check if there's already an active stream in this channel
	s.activeStreamsMu.RLock()
	existingStream, err := s.repo.GetActiveByChannel(ctx, channelID)
	s.activeStreamsMu.RUnlock()
	if err != nil {
		return nil, err
	}
	if existingStream != nil {
		return nil, ErrLiveAlreadyStreaming
	}

	// Check if user is already streaming somewhere
	userStream, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if userStream != nil {
		return nil, ErrLiveAlreadyStreaming
	}

	// Set default quality
	if quality == 0 {
		quality = models.LiveStreamQuality720p
	}

	// Create stream
	var serverID uuid.UUID
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}

	stream := &models.LiveStream{
		ID:         uuid.New(),
		ChannelID:  channelID,
		ServerID:   serverID,
		StreamerID: userID,
		Type:       streamType,
		Quality:    quality,
		Status:     models.LiveStreamStatusActive,
		Viewers:    []uuid.UUID{},
		StartedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, stream); err != nil {
		return nil, err
	}

	// Update in-memory cache
	s.activeStreamsMu.Lock()
	s.activeStreams[channelID] = stream
	s.activeStreamsMu.Unlock()

	// Get streamer info
	var streamer *models.User
	if s.userRepo != nil {
		streamer, _ = s.userRepo.GetByID(ctx, userID)
	}

	info := &models.LiveStreamInfo{
		ID:          stream.ID,
		ChannelID:   stream.ChannelID,
		ServerID:    stream.ServerID,
		StreamerID:  stream.StreamerID,
		Streamer:    streamer,
		Type:        stream.Type,
		Quality:     stream.Quality,
		Status:      stream.Status,
		ViewerCount: 0,
		Viewers:     []uuid.UUID{},
		StartedAt:   stream.StartedAt,
	}

	// Emit stream start event
	s.eventBus.Publish("stream.started", &models.LiveStreamStartEvent{
		Stream:    info,
		ChannelID: channelID,
		ServerID:  serverID,
	})

	return info, nil
}

// StopStream stops an active live stream
func (s *LiveStreamService) StopStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID) error {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil {
		return ErrLiveStreamNotFound
	}

	// Only the streamer can end their own stream (or moderators)
	if stream.StreamerID != userID {
		// Check if user has permission to end others' streams
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, stream.ServerID, userID, models.PermManageChannels); err != nil {
				return ErrLiveNotStreamer
			}
		} else {
			return ErrLiveNotStreamer
		}
	}

	// Check if stream is already ended
	if stream.Status != models.LiveStreamStatusActive {
		return ErrLiveStreamNotFound
	}

	// End the stream
	if err := s.repo.End(ctx, streamID); err != nil {
		return err
	}

	// Remove from in-memory cache
	s.activeStreamsMu.Lock()
	delete(s.activeStreams, stream.ChannelID)
	s.activeStreamsMu.Unlock()

	// Emit stream end event
	s.eventBus.Publish("stream.ended", &models.LiveStreamEndEvent{
		StreamID:  streamID,
		ChannelID: stream.ChannelID,
		ServerID:  stream.ServerID,
		UserID:    stream.StreamerID,
	})

	return nil
}

// GetStream returns information about a live stream
func (s *LiveStreamService) GetStream(ctx context.Context, streamID uuid.UUID) (*models.LiveStreamInfo, error) {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrLiveStreamNotFound
	}

	// Get streamer info
	var streamer *models.User
	if s.userRepo != nil {
		streamer, _ = s.userRepo.GetByID(ctx, stream.StreamerID)
	}

	info := &models.LiveStreamInfo{
		ID:          stream.ID,
		ChannelID:   stream.ChannelID,
		ServerID:    stream.ServerID,
		StreamerID:  stream.StreamerID,
		Streamer:    streamer,
		Type:        stream.Type,
		Quality:     stream.Quality,
		Status:      stream.Status,
		ViewerCount: stream.ViewerCount,
		Viewers:     stream.Viewers,
		StartedAt:   stream.StartedAt,
		EndedAt:     stream.EndedAt,
	}

	return info, nil
}

// GetActiveStreamForChannel returns the active stream for a channel if any
func (s *LiveStreamService) GetActiveStreamForChannel(ctx context.Context, channelID uuid.UUID) (*models.LiveStreamInfo, error) {
	// Check in-memory cache first
	s.activeStreamsMu.RLock()
	cached := s.activeStreams[channelID]
	s.activeStreamsMu.RUnlock()

	if cached != nil && cached.Status == models.LiveStreamStatusActive {
		return s.getStreamInfo(ctx, cached)
	}

	// Fallback to database
	stream, err := s.repo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, nil
	}

	// Update cache
	s.activeStreamsMu.Lock()
	s.activeStreams[channelID] = stream
	s.activeStreamsMu.Unlock()

	return s.getStreamInfo(ctx, stream)
}

// JoinStreamAsViewer allows a user to join viewing a stream
func (s *LiveStreamService) JoinStreamAsViewer(ctx context.Context, streamID uuid.UUID, viewerID uuid.UUID) error {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil {
		return ErrLiveStreamNotFound
	}

	// Check if stream is still active
	if stream.Status != models.LiveStreamStatusActive {
		return ErrLiveStreamEnded
	}

	// Cannot join your own stream
	if stream.StreamerID == viewerID {
		return ErrLiveCannotJoinOwnStream
	}

	// Check if already viewing
	viewing, err := s.repo.IsViewing(ctx, streamID, viewerID)
	if err != nil {
		return err
	}
	if viewing {
		return ErrLiveAlreadyViewing
	}

	// Verify user is a member of the server (for server channels)
	if s.serverRepo != nil && stream.ServerID != uuid.Nil {
		member, err := s.serverRepo.GetMember(ctx, stream.ServerID, viewerID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
	}

	// Add viewer
	if err := s.repo.AddViewer(ctx, streamID, viewerID); err != nil {
		return err
	}

	// Get updated viewers
	viewers, _ := s.repo.GetViewers(ctx, streamID)
	viewerCount := len(viewers)

	// Update cache
	s.activeStreamsMu.Lock()
	if s.activeStreams[stream.ChannelID] != nil {
		s.activeStreams[stream.ChannelID].ViewerCount = viewerCount
		s.activeStreams[stream.ChannelID].Viewers = viewers
	}
	s.activeStreamsMu.Unlock()

	// Emit viewer join event
	s.eventBus.Publish("stream.viewer_joined", &models.LiveStreamViewerJoinEvent{
		StreamID:    streamID,
		UserID:      viewerID,
		ViewerCount: viewerCount,
		Viewers:     viewers,
		ChannelID:   stream.ChannelID,
		ServerID:    stream.ServerID,
	})

	return nil
}

// LeaveStream allows a user to stop viewing a stream
func (s *LiveStreamService) LeaveStream(ctx context.Context, streamID uuid.UUID, viewerID uuid.UUID) error {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return err
	}
	if stream == nil {
		return ErrLiveStreamNotFound
	}

	// Check if viewing
	viewing, err := s.repo.IsViewing(ctx, streamID, viewerID)
	if err != nil {
		return err
	}
	if !viewing {
		return ErrLiveNotViewing
	}

	// Remove viewer
	if err := s.repo.RemoveViewer(ctx, streamID, viewerID); err != nil {
		return err
	}

	// Get updated viewers
	viewers, _ := s.repo.GetViewers(ctx, streamID)
	viewerCount := len(viewers)

	// Update cache
	s.activeStreamsMu.Lock()
	if s.activeStreams[stream.ChannelID] != nil {
		s.activeStreams[stream.ChannelID].ViewerCount = viewerCount
		s.activeStreams[stream.ChannelID].Viewers = viewers
	}
	s.activeStreamsMu.Unlock()

	// Emit viewer leave event
	s.eventBus.Publish("stream.viewer_left", &models.LiveStreamViewerLeaveEvent{
		StreamID:    streamID,
		UserID:      viewerID,
		ViewerCount: viewerCount,
		Viewers:     viewers,
		ChannelID:   stream.ChannelID,
		ServerID:    stream.ServerID,
	})

	return nil
}

// UpdateStream updates stream settings (quality)
func (s *LiveStreamService) UpdateStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID, updates *models.LiveStreamSettingsUpdate) (*models.LiveStreamInfo, error) {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrLiveStreamNotFound
	}

	// Only the streamer can update their stream
	if stream.StreamerID != userID {
		return nil, ErrLiveNotStreamer
	}

	// Check if stream is still active
	if stream.Status != models.LiveStreamStatusActive {
		return nil, ErrLiveStreamEnded
	}

	// Apply updates
	if updates.Quality != nil {
		stream.Quality = *updates.Quality
	}

	if err := s.repo.Update(ctx, stream); err != nil {
		return nil, err
	}

	return s.getStreamInfo(ctx, stream)
}

// GetStreamViewers returns all viewers of a stream
func (s *LiveStreamService) GetStreamViewers(ctx context.Context, streamID uuid.UUID) ([]uuid.UUID, error) {
	stream, err := s.repo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrLiveStreamNotFound
	}

	return s.repo.GetViewers(ctx, streamID)
}

// Helper to get stream info with user details
func (s *LiveStreamService) getStreamInfo(ctx context.Context, stream *models.LiveStream) (*models.LiveStreamInfo, error) {
	var streamer *models.User
	if s.userRepo != nil {
		streamer, _ = s.userRepo.GetByID(ctx, stream.StreamerID)
	}

	info := &models.LiveStreamInfo{
		ID:          stream.ID,
		ChannelID:   stream.ChannelID,
		ServerID:    stream.ServerID,
		StreamerID:  stream.StreamerID,
		Streamer:    streamer,
		Type:        stream.Type,
		Quality:     stream.Quality,
		Status:      stream.Status,
		ViewerCount: stream.ViewerCount,
		Viewers:     stream.Viewers,
		StartedAt:   stream.StartedAt,
		EndedAt:     stream.EndedAt,
	}

	return info, nil
}
