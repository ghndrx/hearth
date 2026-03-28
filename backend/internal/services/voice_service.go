package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
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
