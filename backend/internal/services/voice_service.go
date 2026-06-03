package services

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"hearth/internal/models"
)

var (
	ErrLiveKitNotConfigured = errors.New("livekit is not configured")
	ErrVoiceChannelNotFound = errors.New("voice channel not found")
	ErrNotVoiceChannel      = errors.New("channel is not a voice channel")
)

// Participant represents a user in a voice channel
type Participant struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	JoinedAt    int64  `json:"joined_at"`
	IsMuted     bool   `json:"is_muted"`
	IsSpeaking  bool   `json:"is_speaking"`
	HasVideo    bool   `json:"has_video"`
	HasScreen   bool   `json:"has_screen"`
}

// VoiceTokenResponse is the response from GenerateToken
type VoiceTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// VoiceService handles LiveKit voice channel operations
type VoiceService struct {
	apiKey      string
	apiSecret   string
	url         string
	roomClient  *lksdk.RoomServiceClient
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	tokenExpiry time.Duration
}

// NewVoiceService creates a new voice service
func NewVoiceService(
	apiKey string,
	apiSecret string,
	url string,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
) *VoiceService {
	var roomClient *lksdk.RoomServiceClient
	if apiKey != "" && apiSecret != "" && url != "" {
		roomClient = lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	}

	return &VoiceService{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		url:         url,
		roomClient:  roomClient,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		tokenExpiry: 24 * time.Hour,
	}
}

// IsConfigured returns true if LiveKit is properly configured
func (s *VoiceService) IsConfigured() bool {
	return s.apiKey != "" && s.apiSecret != "" && s.url != ""
}

// GenerateToken generates a LiveKit access token for a user to join a voice channel
func (s *VoiceService) GenerateToken(
	ctx context.Context,
	userID uuid.UUID,
	channelID uuid.UUID,
	userName string,
	displayName string,
	avatarURL string,
) (*VoiceTokenResponse, error) {
	if !s.IsConfigured() {
		return nil, ErrLiveKitNotConfigured
	}

	// Verify the channel exists and is a voice channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrVoiceChannelNotFound
	}

	// Check if it's a voice or stage channel
	if channel.Type != "voice" && channel.Type != "stage" {
		return nil, ErrNotVoiceChannel
	}

	// Verify user has access to the channel (is a member of the server)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// Room name is the channel ID
	roomName := channelID.String()

	// Create access token
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)

	// Set token validity
	at.SetValidFor(s.tokenExpiry)

	// Set identity and metadata
	// VideoGrant permissions:
	// - RoomJoin: allows joining the room
	// - CanPublish: allows publishing audio, video, and screen share tracks
	// - CanSubscribe: allows subscribing to other participants' tracks
	canPublish := true
	canSubscribe := true
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	at.SetVideoGrant(grant).
		SetIdentity(userID.String()).
		SetName(displayName).
		SetMetadata(s.buildParticipantMetadata(userName, displayName, avatarURL))

	token, err := at.ToJWT()
	if err != nil {
		return nil, err
	}

	return &VoiceTokenResponse{
		Token: token,
		URL:   s.url,
	}, nil
}

// buildParticipantMetadata creates JSON metadata for a participant
func (s *VoiceService) buildParticipantMetadata(userName, displayName, avatarURL string) string {
	// Simple JSON metadata - could use json.Marshal but this is straightforward
	if avatarURL != "" {
		return `{"username":"` + userName + `","display_name":"` + displayName + `","avatar_url":"` + avatarURL + `"}`
	}
	return `{"username":"` + userName + `","display_name":"` + displayName + `"}`
}

// GetRoomParticipants returns the list of participants in a voice channel
func (s *VoiceService) GetRoomParticipants(ctx context.Context, channelID uuid.UUID) ([]Participant, error) {
	if !s.IsConfigured() {
		return nil, ErrLiveKitNotConfigured
	}

	roomName := channelID.String()

	res, err := s.roomClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: roomName,
	})
	if err != nil {
		// Room might not exist yet (no one has joined)
		// Return empty list instead of error
		return []Participant{}, nil
	}

	participants := make([]Participant, 0, len(res.Participants))
	for _, p := range res.Participants {
		participant := Participant{
			ID:          p.Sid,
			UserID:      p.Identity,
			DisplayName: p.Name,
			JoinedAt:    p.JoinedAt,
			IsMuted:     s.isParticipantMuted(p),
			IsSpeaking:  s.isParticipantSpeaking(p),
			HasVideo:    s.hasVideoTrack(p),
			HasScreen:   s.hasScreenTrack(p),
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

// isParticipantMuted checks if the participant has their audio muted
func (s *VoiceService) isParticipantMuted(p *livekit.ParticipantInfo) bool {
	for _, track := range p.Tracks {
		if track.Type == livekit.TrackType_AUDIO {
			return track.Muted
		}
	}
	return true // No audio track = muted
}

// isParticipantSpeaking checks if the participant is currently speaking
func (s *VoiceService) isParticipantSpeaking(p *livekit.ParticipantInfo) bool {
	// LiveKit doesn't expose speaking state directly in ParticipantInfo
	// This would need to be tracked via active speakers callback
	return false
}

// hasVideoTrack checks if the participant has a video track published
func (s *VoiceService) hasVideoTrack(p *livekit.ParticipantInfo) bool {
	for _, track := range p.Tracks {
		if track.Type == livekit.TrackType_VIDEO && track.Source == livekit.TrackSource_CAMERA {
			return !track.Muted
		}
	}
	return false
}

// hasScreenTrack checks if the participant has a screen share track published
func (s *VoiceService) hasScreenTrack(p *livekit.ParticipantInfo) bool {
	for _, track := range p.Tracks {
		if track.Type == livekit.TrackType_VIDEO && track.Source == livekit.TrackSource_SCREEN_SHARE {
			return !track.Muted
		}
	}
	return false
}

// DisconnectParticipant removes a participant from a voice channel (for moderation)
func (s *VoiceService) DisconnectParticipant(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error {
	if !s.IsConfigured() {
		return ErrLiveKitNotConfigured
	}

	roomName := channelID.String()

	_, err := s.roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: userID.String(),
	})

	return err
}

// MuteParticipant mutes a participant in a voice channel (for moderation)
func (s *VoiceService) MuteParticipant(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, muted bool) error {
	if !s.IsConfigured() {
		return ErrLiveKitNotConfigured
	}

	roomName := channelID.String()

	// Get participant's tracks
	res, err := s.roomClient.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: userID.String(),
	})
	if err != nil {
		return err
	}

	// Mute all audio tracks
	for _, track := range res.Tracks {
		if track.Type == livekit.TrackType_AUDIO {
			_, err = s.roomClient.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
				Room:     roomName,
				Identity: userID.String(),
				TrackSid: track.Sid,
				Muted:    muted,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var ErrUserNotInVoice = errors.New("user not in voice channel")

type VoiceState struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
	ServerID  uuid.UUID
	Muted     bool
	Deafened  bool
	Video     bool
	Streaming bool
}

type VoiceStateService struct {
	mu     sync.RWMutex
	states map[uuid.UUID]*VoiceState // userID -> state
}

// NewVoiceStateService creates a new voice state service
func NewVoiceStateService() *VoiceStateService {
	return &VoiceStateService{states: make(map[uuid.UUID]*VoiceState)}
}

func (s *VoiceStateService) Join(ctx context.Context, userID, channelID, serverID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[userID] = &VoiceState{UserID: userID, ChannelID: channelID, ServerID: serverID}
	return nil
}

func (s *VoiceStateService) Leave(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
	return nil
}

func (s *VoiceStateService) SetMuted(ctx context.Context, userID uuid.UUID, muted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[userID]
	if !ok {
		return ErrUserNotInVoice
	}
	state.Muted = muted
	return nil
}

func (s *VoiceStateService) SetDeafened(ctx context.Context, userID uuid.UUID, deafened bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[userID]
	if !ok {
		return ErrUserNotInVoice
	}
	state.Deafened = deafened
	return nil
}

func (s *VoiceStateService) SetVideo(ctx context.Context, userID uuid.UUID, video bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[userID]
	if !ok {
		return ErrUserNotInVoice
	}
	state.Video = video
	return nil
}

func (s *VoiceStateService) SetStreaming(ctx context.Context, userID uuid.UUID, streaming bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[userID]
	if !ok {
		return ErrUserNotInVoice
	}
	state.Streaming = streaming
	return nil
}

func (s *VoiceStateService) GetChannelUsers(ctx context.Context, channelID uuid.UUID) ([]*VoiceState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var users []*VoiceState
	for _, state := range s.states {
		if state.ChannelID == channelID {
			users = append(users, state)
		}
	}
	return users, nil
}

// CallRepositoryInterface defines the interface for call data access
type CallRepositoryInterface interface {
	Create(ctx context.Context, call *models.Call) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Call, error)
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) ([]*models.Call, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.CallStatus, endReason string) error
	AddParticipant(ctx context.Context, participant *models.CallParticipant) error
	RemoveParticipant(ctx context.Context, callID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, callID uuid.UUID) ([]models.CallParticipant, error)
	GetActiveParticipantCount(ctx context.Context, callID uuid.UUID) (int, error)
	CreateSession(ctx context.Context, session *models.CallSession) error
	EndSession(ctx context.Context, sessionID uuid.UUID) error
}

// CallService handles call business logic
type CallService struct {
	callRepo CallRepositoryInterface
}

// NewCallService creates a new call service
func NewCallService(callRepo CallRepositoryInterface) *CallService {
	return &CallService{
		callRepo: callRepo,
	}
}

var (
	ErrCallNotFound      = errors.New("call not found")
	ErrCallAlreadyEnded  = errors.New("call has already ended")
	ErrAlreadyInCall     = errors.New("user is already in this call")
	ErrNotInCall         = errors.New("user is not in this call")
	ErrInvalidCallType   = errors.New("invalid call type")
)

// CreateCall creates a new call
func (s *CallService) CreateCall(ctx context.Context, initiatorID, channelID uuid.UUID, serverID *uuid.UUID, callType models.CallType) (*models.Call, error) {
	if callType != models.CallTypeDirect && callType != models.CallTypeGroup && callType != models.CallTypeChannel {
		return nil, ErrInvalidCallType
	}

	call := &models.Call{
		ChannelID:   channelID,
		ServerID:    serverID,
		InitiatorID: initiatorID,
		Type:        callType,
		Status:      models.CallStatusRinging,
		StartedAt:   time.Now(),
	}

	if err := s.callRepo.Create(ctx, call); err != nil {
		return nil, err
	}

	// Add initiator as first participant
	participant := &models.CallParticipant{
		CallID:    call.ID,
		UserID:    initiatorID,
		IsMuted:   true,
		IsVideoOn: false,
	}
	if err := s.callRepo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	return s.callRepo.GetByID(ctx, call.ID)
}

// GetCall retrieves a call by ID
func (s *CallService) GetCall(ctx context.Context, callID uuid.UUID) (*models.Call, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, ErrCallNotFound
	}
	return call, nil
}

// JoinCall adds a user to an existing call
func (s *CallService) JoinCall(ctx context.Context, callID, userID uuid.UUID) (*models.JoinCallResponse, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, ErrCallNotFound
	}
	if call.Status == models.CallStatusEnded {
		return nil, ErrCallAlreadyEnded
	}

	// Update call status to active if it was ringing
	if call.Status == models.CallStatusRinging {
		if err := s.callRepo.UpdateStatus(ctx, callID, models.CallStatusActive, ""); err != nil {
			return nil, err
		}
	}

	participant := &models.CallParticipant{
		CallID:    callID,
		UserID:    userID,
		IsMuted:   true,
		IsVideoOn: false,
	}
	if err := s.callRepo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	return &models.JoinCallResponse{
		CallID:     callID,
		UserID:     userID,
		JoinedAt:   participant.JoinedAt,
		ICEServers: s.getICEServers(),
	}, nil
}

// LeaveCall removes a user from a call
func (s *CallService) LeaveCall(ctx context.Context, callID, userID uuid.UUID) error {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil {
		return ErrCallNotFound
	}

	if err := s.callRepo.RemoveParticipant(ctx, callID, userID); err != nil {
		return err
	}

	// Check if any participants remain
	count, err := s.callRepo.GetActiveParticipantCount(ctx, callID)
	if err != nil {
		return err
	}

	// End call if no participants remain
	if count == 0 {
		return s.callRepo.UpdateStatus(ctx, callID, models.CallStatusEnded, string(models.CallEndReasonCompleted))
	}

	return nil
}

// EndCall ends a call with a given reason
func (s *CallService) EndCall(ctx context.Context, callID uuid.UUID, reason models.CallEndReason) error {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil {
		return ErrCallNotFound
	}
	if call.Status == models.CallStatusEnded {
		return ErrCallAlreadyEnded
	}

	return s.callRepo.UpdateStatus(ctx, callID, models.CallStatusEnded, string(reason))
}

// GetActiveCallsForChannel returns active calls in a channel
func (s *CallService) GetActiveCallsForChannel(ctx context.Context, channelID uuid.UUID) ([]*models.Call, error) {
	return s.callRepo.GetActiveByChannel(ctx, channelID)
}

// getICEServers returns the configured ICE servers.
// TURN server can be configured via environment variables:
// - TURN_SERVER_URL: The TURN server URL (e.g., "turn:turn.example.com:3478")
// - TURN_USERNAME: Optional username for TURN authentication
// - TURN_CREDENTIAL: Optional credential for TURN authentication
func (s *CallService) getICEServers() []models.ICEServer {
	servers := []models.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}

	turnURL := os.Getenv("TURN_SERVER_URL")
	if turnURL != "" {
		servers = append(servers, models.ICEServer{
			URLs:       []string{turnURL},
			Username:   os.Getenv("TURN_USERNAME"),
			Credential: os.Getenv("TURN_CREDENTIAL"),
		})
	}

	return servers
}

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

var (
	ErrStageNotFound            = errors.New("stage not found")
	ErrStageAlreadyExists       = errors.New("stage already exists for this channel")
	ErrStageNotActive           = errors.New("stage is not active")
	ErrStageNotLive             = errors.New("stage is not live")
	ErrStageNotPaused           = errors.New("stage is not paused")
	ErrNotStageHost             = errors.New("not the stage host")
	ErrNotStageModerator        = errors.New("not a stage moderator")
	ErrNotStageParticipant      = errors.New("not a stage participant")
	ErrCannotModifyHost         = errors.New("cannot modify host role")
	ErrMaxSpeakersReached       = errors.New("maximum speakers reached")
	ErrSpeakerRequestPending    = errors.New("speaker request already pending")
	ErrSpeakerRequestNotPending = errors.New("no pending speaker request")
	ErrNotAudienceMember        = errors.New("user is not an audience member")
	ErrNotSpeaker               = errors.New("user is not a speaker")
	ErrModeratorOnly            = errors.New("stage is moderator-only, cannot request to speak")
	ErrChannelNotStage          = errors.New("channel is not a stage channel")
	ErrNotStageHostOrMod        = errors.New("not the stage host or a moderator")
)

// StageRepository defines stage data access
type StageRepository interface {
	Create(ctx context.Context, stage *models.Stage) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Stage, error)
	GetByChannelID(ctx context.Context, channelID uuid.UUID) (*models.Stage, error)
	Update(ctx context.Context, stage *models.Stage) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddParticipant(ctx context.Context, p *models.StageParticipant) error
	GetParticipant(ctx context.Context, stageID, userID uuid.UUID) (*models.StageParticipant, error)
	UpdateParticipant(ctx context.Context, p *models.StageParticipant) error
	RemoveParticipant(ctx context.Context, stageID, userID uuid.UUID) error
	ListParticipants(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error)
	ListParticipantsByRole(ctx context.Context, stageID uuid.UUID, role models.StageRole) ([]models.StageParticipant, error)
	ListPendingRequests(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error)
	CountParticipantsByRole(ctx context.Context, stageID uuid.UUID) (speakers int, audience int, pending int, err error)
	UpdateParticipantMute(ctx context.Context, stageID, userID uuid.UUID, isMuted bool) error
	UpdateParticipantDeaf(ctx context.Context, stageID, userID uuid.UUID, isDeafened bool) error
	ApproveSpeakerRequest(ctx context.Context, stageID, userID uuid.UUID) error
	ClearAllParticipants(ctx context.Context, stageID uuid.UUID) error
}

// StageService handles stage channel business logic
type StageService struct {
	stageRepo    StageRepository
	channelRepo  ChannelRepository
	serverRepo   ServerRepository
	permService  *PermissionService
	voiceService *VoiceService
	eventBus     EventBus
}

// NewStageService creates a new stage service
func NewStageService(
	stageRepo StageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	voiceService *VoiceService,
	eventBus EventBus,
) *StageService {
	return &StageService{
		stageRepo:    stageRepo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		permService:  permService,
		voiceService: voiceService,
		eventBus:     eventBus,
	}
}

// CreateStage creates and starts a new stage in a channel
func (s *StageService) CreateStage(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateStageRequest) (*models.StageInfo, error) {
	// Verify channel exists and is a stage channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeStage {
		return nil, ErrChannelNotStage
	}

	// Check if a stage already exists for this channel
	existing, err := s.stageRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != models.StageStatusEnded {
		return nil, ErrStageAlreadyExists
	}

	// Verify user is a member of the server (if this is a server channel)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// Build stage config
	discoveryDisabled := false
	requestToSpeak := true
	moderatorOnly := false
	maxSpeakers := 10

	if req.DiscoveryDisabled != nil {
		discoveryDisabled = *req.DiscoveryDisabled
	}
	if req.RequestToSpeak != nil {
		requestToSpeak = *req.RequestToSpeak
	}
	if req.ModeratorOnly != nil {
		moderatorOnly = *req.ModeratorOnly
	}
	if req.MaxSpeakers != nil {
		maxSpeakers = *req.MaxSpeakers
	}

	now := time.Now()
	stage := &models.Stage{
		ID:                uuid.New(),
		ChannelID:         channelID,
		Topic:             req.Topic,
		Description:       req.Description,
		Status:            models.StageStatusLive,
		HostUserID:        userID,
		DiscoveryDisabled: discoveryDisabled,
		RequestToSpeak:    requestToSpeak,
		ModeratorOnly:     moderatorOnly,
		MaxSpeakers:       maxSpeakers,
		CreatedAt:         now,
		StartedAt:         &now,
		UpdatedAt:         now,
	}

	if err := s.stageRepo.Create(ctx, stage); err != nil {
		return nil, err
	}

	// Add host as participant
	hostParticipant := &models.StageParticipant{
		StageID:  stage.ID,
		UserID:   userID,
		Role:     models.StageRoleHost,
		JoinedAt: now,
		IsMuted:  false, // Host can speak
	}
	if err := s.stageRepo.AddParticipant(ctx, hostParticipant); err != nil {
		return nil, err
	}

	// Emit stage created event
	s.emitStageEvent(ctx, models.WSEventStageCreated, stage)

	return s.buildStageInfo(ctx, stage)
}

// GetStage retrieves stage info for a channel
func (s *StageService) GetStage(ctx context.Context, channelID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}
	return s.buildStageInfo(ctx, stage)
}

// UpdateStage updates stage metadata (topic, description)
func (s *StageService) UpdateStage(ctx context.Context, stageID, userID uuid.UUID, req *models.UpdateStageRequest) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can update
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	// Apply updates
	if req.Topic != nil {
		stage.Topic = *req.Topic
	}
	if req.Description != nil {
		stage.Description = *req.Description
	}
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage updated event
	s.emitStageEvent(ctx, models.WSEventStageUpdated, stage)

	return s.buildStageInfo(ctx, stage)
}

// UpdateStageConfig updates stage configuration
func (s *StageService) UpdateStageConfig(ctx context.Context, stageID, userID uuid.UUID, req *models.StageConfig) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host can update config
	if stage.HostUserID != userID {
		return nil, ErrNotStageHost
	}

	// Apply updates
	if req.DiscoveryDisabled != nil {
		stage.DiscoveryDisabled = *req.DiscoveryDisabled
	}
	if req.RequestToSpeak != nil {
		stage.RequestToSpeak = *req.RequestToSpeak
	}
	if req.ModeratorOnly != nil {
		stage.ModeratorOnly = *req.ModeratorOnly
	}
	if req.MaxSpeakers != nil {
		stage.MaxSpeakers = *req.MaxSpeakers
	}
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage updated event
	s.emitStageEvent(ctx, models.WSEventStageUpdated, stage)

	return s.buildStageInfo(ctx, stage)
}

// PauseStage pauses a live stage
func (s *StageService) PauseStage(ctx context.Context, stageID, userID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can pause
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	if stage.Status != models.StageStatusLive {
		return nil, ErrStageNotLive
	}

	stage.Status = models.StageStatusPaused
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage paused event
	s.emitStageEvent(ctx, models.WSEventStagePaused, stage)

	return s.buildStageInfo(ctx, stage)
}

// ResumeStage resumes a paused stage
func (s *StageService) ResumeStage(ctx context.Context, stageID, userID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can resume
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	if stage.Status != models.StageStatusPaused {
		return nil, ErrStageNotPaused
	}

	stage.Status = models.StageStatusLive
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage resumed event
	s.emitStageEvent(ctx, models.WSEventStageResumed, stage)

	return s.buildStageInfo(ctx, stage)
}

// EndStage ends a stage
func (s *StageService) EndStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host or moderator can end
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return ErrNotStageHostOrMod
	}

	if stage.Status == models.StageStatusEnded {
		return nil // Already ended
	}

	now := time.Now()
	stage.Status = models.StageStatusEnded
	stage.EndedAt = &now
	stage.UpdatedAt = now

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return err
	}

	// Remove all participants
	if err := s.stageRepo.ClearAllParticipants(ctx, stageID); err != nil {
		return err
	}

	// Emit stage ended event
	s.emitStageEvent(ctx, models.WSEventStageEnded, stage)

	return nil
}

// JoinStage adds a user as an audience member to a stage
func (s *StageService) JoinStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}
	if stage.Status == models.StageStatusEnded || stage.Status == models.StageStatusScheduled {
		return ErrStageNotActive
	}

	// Verify user isn't already a participant
	existing, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // Already participating
	}

	// Add as audience member (muted by default)
	now := time.Now()
	participant := &models.StageParticipant{
		StageID:  stage.ID,
		UserID:   userID,
		Role:     models.StageRoleAudience,
		JoinedAt: now,
		IsMuted:  true, // Audience always muted
	}

	if err := s.stageRepo.AddParticipant(ctx, participant); err != nil {
		return err
	}

	// Emit speaker added event
	s.emitParticipantEvent(ctx, models.WSEventStageAudienceEnter, stage, userID)

	return nil
}

// LeaveStage removes a user from a stage
func (s *StageService) LeaveStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}

	// Host cannot leave without ending the stage
	if participant.Role == models.StageRoleHost {
		return ErrCannotModifyHost
	}

	if err := s.stageRepo.RemoveParticipant(ctx, stageID, userID); err != nil {
		return err
	}

	// Emit participant removed event
	s.emitParticipantEvent(ctx, models.WSEventStageParticipantRemove, stage, userID)

	return nil
}

// RequestToSpeak requests speaker privileges
func (s *StageService) RequestToSpeak(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}
	if stage.Status != models.StageStatusLive && stage.Status != models.StageStatusPaused {
		return ErrStageNotActive
	}

	// Check if moderator only
	if stage.ModeratorOnly {
		return ErrModeratorOnly
	}

	// Check if request to speak is enabled
	if !stage.RequestToSpeak {
		return ErrModeratorOnly
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleAudience {
		return ErrNotAudienceMember
	}

	// Check if already has pending request
	if participant.RequestedAt != nil && participant.ApprovedAt == nil {
		return ErrSpeakerRequestPending
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	now := time.Now()
	participant.RequestedAt = &now

	if err := s.stageRepo.UpdateParticipant(ctx, participant); err != nil {
		return err
	}

	// Emit request to speak event
	s.emitParticipantEvent(ctx, models.WSEventStageRequestToSpeak, stage, userID)

	return nil
}

// CancelRequestToSpeak cancels a pending speaker request
func (s *StageService) CancelRequestToSpeak(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.RequestedAt == nil || participant.ApprovedAt != nil {
		return ErrSpeakerRequestNotPending
	}

	now := time.Time{}
	participant.RequestedAt = &now // Clear by setting to zero

	if err := s.stageRepo.UpdateParticipant(ctx, participant); err != nil {
		return err
	}

	return nil
}

// ApproveSpeaker approves a speaker request
func (s *StageService) ApproveSpeaker(ctx context.Context, stageID, approverID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check approver is host or moderator
	approver, err := s.stageRepo.GetParticipant(ctx, stageID, approverID)
	if err != nil {
		return err
	}
	if approver == nil || (approver.Role != models.StageRoleHost && approver.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Get target participant
	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.RequestedAt == nil || target.ApprovedAt != nil {
		return ErrSpeakerRequestNotPending
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	if err := s.stageRepo.ApproveSpeakerRequest(ctx, stageID, targetUserID); err != nil {
		return err
	}

	// Emit speaker approved event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerAdded, stage, targetUserID)

	return nil
}

// DenySpeaker denies a speaker request
func (s *StageService) DenySpeaker(ctx context.Context, stageID, denierID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check denier is host or moderator
	denier, err := s.stageRepo.GetParticipant(ctx, stageID, denierID)
	if err != nil {
		return err
	}
	if denier == nil || (denier.Role != models.StageRoleHost && denier.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Get target participant
	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	// Reset request
	now := time.Time{}
	target.RequestedAt = &now
	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	return nil
}

// PromoteToSpeaker promotes an audience member to speaker
func (s *StageService) PromoteToSpeaker(ctx context.Context, stageID, promoterID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check promoter is host or moderator
	promoter, err := s.stageRepo.GetParticipant(ctx, stageID, promoterID)
	if err != nil {
		return err
	}
	if promoter == nil || (promoter.Role != models.StageRoleHost && promoter.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleAudience {
		return ErrNotAudienceMember
	}

	// Promote directly (no request needed)
	now := time.Now()
	target.Role = models.StageRoleSpeaker
	target.IsMuted = false // Speakers can speak
	target.ApprovedAt = &now
	target.RequestedAt = nil

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	// Emit speaker added event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerAdded, stage, targetUserID)

	return nil
}

// DemoteToAudience demotes a speaker to audience
func (s *StageService) DemoteToAudience(ctx context.Context, stageID, demoterID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check demoter is host or moderator
	demoter, err := s.stageRepo.GetParticipant(ctx, stageID, demoterID)
	if err != nil {
		return err
	}
	if demoter == nil || (demoter.Role != models.StageRoleHost && demoter.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleSpeaker && target.Role != models.StageRoleModerator {
		return ErrNotSpeaker
	}
	if target.Role == models.StageRoleHost {
		return ErrCannotModifyHost
	}

	// Demote to audience
	target.Role = models.StageRoleAudience
	target.IsMuted = true
	target.ApprovedAt = nil
	now := time.Time{}
	target.RequestedAt = &now

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	// Emit speaker removed event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerRemove, stage, targetUserID)

	return nil
}

// AddModerator adds a moderator to the stage
func (s *StageService) AddModerator(ctx context.Context, stageID, hostID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host can add moderators
	if stage.HostUserID != hostID {
		return ErrNotStageHost
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	target.Role = models.StageRoleModerator
	target.IsMuted = false

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	s.emitParticipantEvent(ctx, models.WSEventStageModeratorAdded, stage, targetUserID)

	return nil
}

// RemoveModerator removes a moderator from the stage
func (s *StageService) RemoveModerator(ctx context.Context, stageID, hostID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host can remove moderators
	if stage.HostUserID != hostID {
		return ErrNotStageHost
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleModerator {
		return ErrNotStageModerator
	}

	// Demote to speaker
	target.Role = models.StageRoleSpeaker

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	s.emitParticipantEvent(ctx, models.WSEventStageModeratorRemoved, stage, targetUserID)

	return nil
}

// ListParticipants lists all participants in a stage
func (s *StageService) ListParticipants(ctx context.Context, stageID uuid.UUID) ([]models.ParticipantInfo, error) {
	participants, err := s.stageRepo.ListParticipants(ctx, stageID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ParticipantInfo, len(participants))
	for i, p := range participants {
		result[i] = models.ParticipantInfo{
			UserID:            p.UserID,
			Role:              p.Role,
			JoinedAt:          p.JoinedAt,
			IsMuted:           p.IsMuted,
			IsDeafened:        p.IsDeafened,
			HasPendingRequest: p.RequestedAt != nil && p.ApprovedAt == nil,
			RequestedAt:       p.RequestedAt,
		}
	}

	return result, nil
}

// ListPendingRequests lists pending speaker requests
func (s *StageService) ListPendingRequests(ctx context.Context, stageID, userID uuid.UUID) ([]models.ParticipantInfo, error) {
	// Only host or moderator can see requests
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	participants, err := s.stageRepo.ListPendingRequests(ctx, stageID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ParticipantInfo, len(participants))
	for i, p := range participants {
		result[i] = models.ParticipantInfo{
			UserID:            p.UserID,
			Role:              p.Role,
			JoinedAt:          p.JoinedAt,
			IsMuted:           p.IsMuted,
			IsDeafened:        p.IsDeafened,
			HasPendingRequest: true,
			RequestedAt:       p.RequestedAt,
		}
	}

	return result, nil
}

// MuteParticipant mutes a participant
func (s *StageService) MuteParticipant(ctx context.Context, stageID, moderatorID, targetUserID uuid.UUID, muted bool) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check moderator has permissions
	moderator, err := s.stageRepo.GetParticipant(ctx, stageID, moderatorID)
	if err != nil {
		return err
	}
	if moderator == nil || (moderator.Role != models.StageRoleHost && moderator.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	// Audience can't unmute themselves
	if target.Role == models.StageRoleAudience && !muted {
		return ErrNotSpeaker
	}

	return s.stageRepo.UpdateParticipantMute(ctx, stageID, targetUserID, muted)
}

// DeafenParticipant deafens a participant
func (s *StageService) DeafenParticipant(ctx context.Context, stageID, moderatorID, targetUserID uuid.UUID, deafened bool) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	moderator, err := s.stageRepo.GetParticipant(ctx, stageID, moderatorID)
	if err != nil {
		return err
	}
	if moderator == nil || (moderator.Role != models.StageRoleHost && moderator.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	return s.stageRepo.UpdateParticipantDeaf(ctx, stageID, targetUserID, deafened)
}

// GenerateVoiceToken generates a LiveKit token for stage participation
func (s *StageService) GenerateVoiceToken(ctx context.Context, stageID, userID uuid.UUID, userName, displayName, avatarURL string) (*VoiceTokenResponse, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}

	// Generate voice token with the channel ID as the room
	return s.voiceService.GenerateToken(ctx, userID, stage.ChannelID, userName, displayName, avatarURL)
}

// buildStageInfo builds a StageInfo response from a Stage
func (s *StageService) buildStageInfo(ctx context.Context, stage *models.Stage) (*models.StageInfo, error) {
	speakers, audience, pending, err := s.stageRepo.CountParticipantsByRole(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	return &models.StageInfo{
		ID:                stage.ID,
		ChannelID:         stage.ChannelID,
		Topic:             stage.Topic,
		Description:       stage.Description,
		Status:            stage.Status,
		HostUserID:        stage.HostUserID,
		DiscoveryDisabled: stage.DiscoveryDisabled,
		RequestToSpeak:    stage.RequestToSpeak,
		ModeratorOnly:     stage.ModeratorOnly,
		MaxSpeakers:       stage.MaxSpeakers,
		SpeakerCount:      speakers,
		AudienceCount:     audience,
		PendingCount:      pending,
		CreatedAt:         stage.CreatedAt,
		StartedAt:         stage.StartedAt,
		EndedAt:           stage.EndedAt,
	}, nil
}

// emitStageEvent emits a stage lifecycle event
func (s *StageService) emitStageEvent(ctx context.Context, eventType string, stage *models.Stage) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(eventType, stage)
}

// emitParticipantEvent emits a participant-specific event
func (s *StageService) emitParticipantEvent(ctx context.Context, eventType string, stage *models.Stage, userID uuid.UUID) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(eventType, map[string]interface{}{
		"stage_id":   stage.ID,
		"channel_id": stage.ChannelID,
		"user_id":    userID,
		"status":     stage.Status,
	})
}

// ServerAudioSettingsRepository defines the interface for server audio settings data access
type ServerAudioSettingsRepository interface {
	Get(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error)
	GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error)
	Upsert(ctx context.Context, settings *models.ServerAudioSettings) error
	Delete(ctx context.Context, userID, serverID uuid.UUID) error
}

// ServerAudioSettingsService handles per-server audio settings business logic
type ServerAudioSettingsService struct {
	repo     ServerAudioSettingsRepository
	eventBus EventBus
}

// NewServerAudioSettingsService creates a new server audio settings service
func NewServerAudioSettingsService(repo ServerAudioSettingsRepository, eventBus EventBus) *ServerAudioSettingsService {
	return &ServerAudioSettingsService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// GetSettings retrieves audio settings for a user in a server, returning defaults if none exist
func (s *ServerAudioSettingsService) GetSettings(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerAudioSettings, error) {
	settings, err := s.repo.Get(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return models.DefaultServerAudioSettings(userID, serverID), nil
	}
	return settings, nil
}

// GetAllForUser retrieves audio settings for all servers for a user
func (s *ServerAudioSettingsService) GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerAudioSettings, error) {
	return s.repo.GetAllForUser(ctx, userID)
}

// UpdateSettings updates audio settings for a user in a server
func (s *ServerAudioSettingsService) UpdateSettings(ctx context.Context, userID, serverID uuid.UUID, updates *models.UpdateServerAudioSettingsRequest) (*models.ServerAudioSettings, error) {
	settings, err := s.GetSettings(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	if updates.InputDeviceID != nil {
		settings.InputDeviceID = *updates.InputDeviceID
	}
	if updates.OutputDeviceID != nil {
		settings.OutputDeviceID = *updates.OutputDeviceID
	}
	if updates.InputVolume != nil {
		settings.InputVolume = *updates.InputVolume
	}
	if updates.OutputVolume != nil {
		settings.OutputVolume = *updates.OutputVolume
	}
	if updates.PushToTalkEnabled != nil {
		settings.PushToTalkEnabled = *updates.PushToTalkEnabled
	}
	if updates.PushToTalkKey != nil {
		settings.PushToTalkKey = *updates.PushToTalkKey
	}

	settings.UpdatedAt = time.Now()

	if err := s.repo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	s.eventBus.Publish("server.audio_settings_updated", &ServerAudioSettingsUpdatedEvent{
		UserID:   userID,
		ServerID: serverID,
		Settings: settings,
	})

	return settings, nil
}

// DeleteSettings deletes audio settings for a user in a server
func (s *ServerAudioSettingsService) DeleteSettings(ctx context.Context, userID, serverID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, serverID)
}

// ServerAudioSettingsUpdatedEvent is emitted when server audio settings are updated
type ServerAudioSettingsUpdatedEvent struct {
	UserID   uuid.UUID
	ServerID uuid.UUID
	Settings *models.ServerAudioSettings
}