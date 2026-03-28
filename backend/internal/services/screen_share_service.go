package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrStreamNotFound      = errors.New("stream not found")
	ErrAlreadyStreaming    = errors.New("user is already streaming in this channel")
	ErrNoActiveStream      = errors.New("no active stream in this channel")
	ErrNotStreamer         = errors.New("user is not the streamer")
	ErrCannotJoinOwnStream = errors.New("cannot join your own stream")
	ErrChannelNotVoice     = errors.New("channel is not a voice channel")
	ErrAlreadyViewing      = errors.New("user is already viewing this stream")
	ErrNotViewing          = errors.New("user is not viewing this stream")
)

// ScreenShareRepository defines data access for screen share
type ScreenShareRepository interface {
	CreateSession(ctx context.Context, session *models.StreamSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*models.StreamSession, error)
	GetActiveSessionByChannel(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error)
	UpdateSession(ctx context.Context, session *models.StreamSession) error
	EndSession(ctx context.Context, id uuid.UUID) error
	ListActiveSessions(ctx context.Context) ([]*models.StreamSession, error)
	ListActiveSessionsByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StreamSession, error)
	AddViewer(ctx context.Context, sessionID, userID uuid.UUID) error
	RemoveViewer(ctx context.Context, sessionID, userID uuid.UUID) error
	GetViewerCount(ctx context.Context, sessionID uuid.UUID) (int, error)
	GetViewers(ctx context.Context, sessionID uuid.UUID) ([]models.StreamViewer, error)
	IsViewing(ctx context.Context, sessionID, userID uuid.UUID) (bool, error)
	GetActiveStreamForUser(ctx context.Context, userID uuid.UUID) (*models.StreamSession, error)
}

// ScreenShareService handles screen share business logic
type ScreenShareService struct {
	repo        ScreenShareRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	eventBus    EventBus
}

// NewScreenShareService creates a new screen share service
func NewScreenShareService(
	repo ScreenShareRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *ScreenShareService {
	return &ScreenShareService{
		repo:        repo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		eventBus:    eventBus,
	}
}

// StartStream starts a screen share or application stream in a voice channel
func (s *ScreenShareService) StartStream(
	ctx context.Context,
	channelID uuid.UUID,
	userID uuid.UUID,
	streamType models.StreamType,
	resolution string,
	frameRate int,
) (*models.StreamInfo, error) {
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
		return nil, ErrChannelNotVoice
	}

	// Check if user is a member of the server (for server channels)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}

		// Check STREAM permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermVideo); err != nil {
				return nil, err
			}
		}
	}

	// Check if there's already an active stream in this channel
	existingStream, err := s.repo.GetActiveSessionByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if existingStream != nil {
		return nil, ErrAlreadyStreaming
	}

	// Check if user is already streaming somewhere
	userStream, err := s.repo.GetActiveStreamForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if userStream != nil {
		return nil, ErrAlreadyStreaming
	}

	// Set defaults
	if resolution == "" {
		resolution = "1080p"
	}
	if frameRate == 0 {
		frameRate = 30
	}

	// Create session
	var serverID uuid.UUID
	if channel.ServerID != nil {
		serverID = *channel.ServerID
	}
	session := &models.StreamSession{
		ID:         uuid.New(),
		ServerID:   serverID,
		ChannelID:  channelID,
		UserID:     userID,
		StreamType: streamType,
		Status:     models.StreamStatusActive,
		Resolution: resolution,
		FrameRate:  frameRate,
		StartedAt:  time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Get viewer count (0 initially)
	viewerCount, _ := s.repo.GetViewerCount(ctx, session.ID)

	info := &models.StreamInfo{
		ID:          session.ID,
		ServerID:    session.ServerID,
		ChannelID:   session.ChannelID,
		UserID:      session.UserID,
		StreamType:  session.StreamType,
		Status:      session.Status,
		Resolution:  session.Resolution,
		FrameRate:   session.FrameRate,
		ViewerCount: viewerCount,
		StartedAt:   session.StartedAt,
	}

	// Emit stream start event
	s.eventBus.Publish("stream.started", &StreamStartedEvent{
		Stream:    info,
		ServerID:  session.ServerID,
		ChannelID: session.ChannelID,
	})

	return info, nil
}

// EndStream stops an active stream
func (s *ScreenShareService) EndStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID) error {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrStreamNotFound
	}

	// Only the streamer can end their own stream
	if session.UserID != userID {
		// Check if user has permission to end others' streams (e.g., moderators)
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, session.ServerID, userID, models.PermManageChannels); err != nil {
				return ErrNotStreamer
			}
		} else {
			return ErrNotStreamer
		}
	}

	// Check if stream is already ended
	if session.Status != models.StreamStatusActive {
		return ErrStreamNotFound
	}

	// End the session
	if err := s.repo.EndSession(ctx, streamID); err != nil {
		return err
	}

	// Emit stream end event
	s.eventBus.Publish("stream.ended", &StreamEndedEvent{
		StreamID:  streamID,
		ChannelID: session.ChannelID,
		ServerID:  session.ServerID,
		UserID:    session.UserID,
	})

	return nil
}

// GetStreamInfo returns information about a stream
func (s *ScreenShareService) GetStreamInfo(ctx context.Context, streamID uuid.UUID) (*models.StreamInfo, error) {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrStreamNotFound
	}

	viewerCount, _ := s.repo.GetViewerCount(ctx, streamID)

	info := &models.StreamInfo{
		ID:          session.ID,
		ServerID:    session.ServerID,
		ChannelID:   session.ChannelID,
		UserID:      session.UserID,
		StreamType:  session.StreamType,
		Status:      session.Status,
		Resolution:  session.Resolution,
		FrameRate:   session.FrameRate,
		ViewerCount: viewerCount,
		StartedAt:   session.StartedAt,
		EndedAt:     session.EndedAt,
	}

	return info, nil
}

// JoinStream allows a user to join viewing a stream
func (s *ScreenShareService) JoinStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID) error {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrStreamNotFound
	}

	// Check if stream is still active
	if session.Status != models.StreamStatusActive {
		return ErrNoActiveStream
	}

	// Cannot join your own stream
	if session.UserID == userID {
		return ErrCannotJoinOwnStream
	}

	// Check if already viewing
	viewing, err := s.repo.IsViewing(ctx, streamID, userID)
	if err != nil {
		return err
	}
	if viewing {
		return ErrAlreadyViewing
	}

	// Verify user is a member of the server (for server channels)
	if s.serverRepo != nil {
		member, err := s.serverRepo.GetMember(ctx, session.ServerID, userID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
	}

	// Add viewer
	if err := s.repo.AddViewer(ctx, streamID, userID); err != nil {
		return err
	}

	// Get updated viewer count
	viewerCount, _ := s.repo.GetViewerCount(ctx, streamID)

	// Emit viewer join event
	s.eventBus.Publish("stream.viewer_joined", &StreamViewerJoinedEvent{
		StreamID:    streamID,
		UserID:      userID,
		ViewerCount: viewerCount,
		ChannelID:   session.ChannelID,
		ServerID:    session.ServerID,
	})

	return nil
}

// LeaveStream allows a user to stop viewing a stream
func (s *ScreenShareService) LeaveStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID) error {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrStreamNotFound
	}

	// Check if viewing
	viewing, err := s.repo.IsViewing(ctx, streamID, userID)
	if err != nil {
		return err
	}
	if !viewing {
		return ErrNotViewing
	}

	// Remove viewer
	if err := s.repo.RemoveViewer(ctx, streamID, userID); err != nil {
		return err
	}

	// Get updated viewer count
	viewerCount, _ := s.repo.GetViewerCount(ctx, streamID)

	// Emit viewer leave event
	s.eventBus.Publish("stream.viewer_left", &StreamViewerLeftEvent{
		StreamID:    streamID,
		UserID:      userID,
		ViewerCount: viewerCount,
		ChannelID:   session.ChannelID,
		ServerID:    session.ServerID,
	})

	return nil
}

// UpdateStream updates stream settings (resolution, frame rate)
func (s *ScreenShareService) UpdateStream(ctx context.Context, streamID uuid.UUID, userID uuid.UUID, updates *models.StreamUpdate) (*models.StreamInfo, error) {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrStreamNotFound
	}

	// Only the streamer can update their stream
	if session.UserID != userID {
		return nil, ErrNotStreamer
	}

	// Check if stream is still active
	if session.Status != models.StreamStatusActive {
		return nil, ErrNoActiveStream
	}

	// Apply updates
	if updates.Resolution != nil {
		session.Resolution = *updates.Resolution
	}
	if updates.FrameRate != nil {
		session.FrameRate = *updates.FrameRate
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	viewerCount, _ := s.repo.GetViewerCount(ctx, streamID)

	info := &models.StreamInfo{
		ID:          session.ID,
		ServerID:    session.ServerID,
		ChannelID:   session.ChannelID,
		UserID:      session.UserID,
		StreamType:  session.StreamType,
		Status:      session.Status,
		Resolution:  session.Resolution,
		FrameRate:   session.FrameRate,
		ViewerCount: viewerCount,
		StartedAt:   session.StartedAt,
	}

	return info, nil
}

// GetActiveStreamForChannel returns the active stream for a channel if any
func (s *ScreenShareService) GetActiveStreamForChannel(ctx context.Context, channelID uuid.UUID) (*models.StreamInfo, error) {
	session, err := s.repo.GetActiveSessionByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	viewerCount, _ := s.repo.GetViewerCount(ctx, session.ID)

	info := &models.StreamInfo{
		ID:          session.ID,
		ServerID:    session.ServerID,
		ChannelID:   session.ChannelID,
		UserID:      session.UserID,
		StreamType:  session.StreamType,
		Status:      session.Status,
		Resolution:  session.Resolution,
		FrameRate:   session.FrameRate,
		ViewerCount: viewerCount,
		StartedAt:   session.StartedAt,
	}

	return info, nil
}

// GetActiveStreamsForServer returns all active streams for a server
func (s *ScreenShareService) GetActiveStreamsForServer(ctx context.Context, serverID uuid.UUID) ([]*models.StreamInfo, error) {
	sessions, err := s.repo.ListActiveSessionsByServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var result []*models.StreamInfo
	for _, session := range sessions {
		viewerCount, _ := s.repo.GetViewerCount(ctx, session.ID)
		result = append(result, &models.StreamInfo{
			ID:          session.ID,
			ServerID:    session.ServerID,
			ChannelID:   session.ChannelID,
			UserID:      session.UserID,
			StreamType:  session.StreamType,
			Status:      session.Status,
			Resolution:  session.Resolution,
			FrameRate:   session.FrameRate,
			ViewerCount: viewerCount,
			StartedAt:   session.StartedAt,
		})
	}

	return result, nil
}

// GetStreamViewers returns all viewers of a stream
func (s *ScreenShareService) GetStreamViewers(ctx context.Context, streamID uuid.UUID) ([]uuid.UUID, error) {
	session, err := s.repo.GetSession(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrStreamNotFound
	}

	viewers, err := s.repo.GetViewers(ctx, streamID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(viewers))
	for i, v := range viewers {
		userIDs[i] = v.UserID
	}

	return userIDs, nil
}

// Events

// StreamStartedEvent is published when a stream starts
type StreamStartedEvent struct {
	Stream    *models.StreamInfo
	ServerID  uuid.UUID
	ChannelID uuid.UUID
}

// StreamEndedEvent is published when a stream ends
type StreamEndedEvent struct {
	StreamID  uuid.UUID
	ChannelID uuid.UUID
	ServerID  uuid.UUID
	UserID    uuid.UUID
}

// StreamViewerJoinedEvent is published when a viewer joins a stream
type StreamViewerJoinedEvent struct {
	StreamID    uuid.UUID
	UserID      uuid.UUID
	ViewerCount int
	ChannelID   uuid.UUID
	ServerID    uuid.UUID
}

// StreamViewerLeftEvent is published when a viewer leaves a stream
type StreamViewerLeftEvent struct {
	StreamID    uuid.UUID
	UserID      uuid.UUID
	ViewerCount int
	ChannelID   uuid.UUID
	ServerID    uuid.UUID
}
