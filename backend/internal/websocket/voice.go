package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"

	"hearth/internal/database/postgres"
)

// Voice signaling message types
const (
	VoiceSignalOffer        = "VOICE_OFFER"
	VoiceSignalAnswer       = "VOICE_ANSWER"
	VoiceSignalICECandidate = "VOICE_ICE_CANDIDATE"
	VoiceSignalJoin         = "VOICE_JOIN"
	VoiceSignalLeave        = "VOICE_LEAVE"
	VoiceSignalSpeaking     = "VOICE_SPEAKING"
)

// Voice state event types (broadcast to server)
const (
	EventTypeVoiceStateUpdate  = "VOICE_STATE_UPDATE"
	EventTypeVoiceServerUpdate = "VOICE_SERVER_UPDATE"
)

// VoiceSignalMessage represents a WebRTC signaling message
type VoiceSignalMessage struct {
	Type       string          `json:"type"`
	ChannelID  uuid.UUID       `json:"channel_id"`
	ServerID   uuid.UUID       `json:"server_id"`
	FromUserID uuid.UUID       `json:"from_user_id,omitempty"`
	ToUserID   uuid.UUID       `json:"to_user_id,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// VoiceJoinData represents data for joining voice
type VoiceJoinData struct {
	ChannelID    uuid.UUID `json:"channel_id"`
	ServerID     uuid.UUID `json:"server_id"`
	SelfMuted    bool      `json:"self_muted"`
	SelfDeafened bool      `json:"self_deafened"`
}

// VoiceLeaveData represents data for leaving voice
type VoiceLeaveData struct {
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
}

// VoiceSpeakingData represents speaking state
type VoiceSpeakingData struct {
	UserID    uuid.UUID `json:"user_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	Speaking  bool      `json:"speaking"`
	SSRC      uint32    `json:"ssrc,omitempty"`
}

// VoiceStateData represents voice state update data for broadcasting
type VoiceStateData struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	DisplayName  *string   `json:"display_name,omitempty"`
	Avatar       *string   `json:"avatar,omitempty"`
	ChannelID    uuid.UUID `json:"channel_id"`
	ServerID     uuid.UUID `json:"server_id"`
	SelfMuted    bool      `json:"self_muted"`
	SelfDeafened bool      `json:"self_deafened"`
	SelfVideo    bool      `json:"self_video"`
	SelfStream   bool      `json:"self_stream"`
	Muted        bool      `json:"muted"`
	Deafened     bool      `json:"deafened"`
	SessionID    string    `json:"session_id,omitempty"`
}

// VoiceOfferData represents an SDP offer
type VoiceOfferData struct {
	SDP      string    `json:"sdp"`
	ToUserID uuid.UUID `json:"to_user_id"`
}

// VoiceAnswerData represents an SDP answer
type VoiceAnswerData struct {
	SDP      string    `json:"sdp"`
	ToUserID uuid.UUID `json:"to_user_id"`
}

// VoiceICECandidateData represents an ICE candidate
type VoiceICECandidateData struct {
	Candidate     string    `json:"candidate"`
	SDPMid        string    `json:"sdpMid"`
	SDPMLineIndex int       `json:"sdpMLineIndex"`
	ToUserID      uuid.UUID `json:"to_user_id"`
}

// VoiceSignalingService handles WebRTC signaling for voice channels
type VoiceSignalingService struct {
	hub       HubInterface
	voiceRepo *postgres.VoiceStateRepository

	// Track which clients are in which voice channels
	channelPeers   map[uuid.UUID]map[uuid.UUID]*Client // channelID -> userID -> client
	channelPeersMu sync.RWMutex

	// Track speaking states
	speakingStates   map[uuid.UUID]bool // userID -> speaking
	speakingStatesMu sync.RWMutex
}

// NewVoiceSignalingService creates a new voice signaling service
func NewVoiceSignalingService(hub HubInterface, voiceRepo *postgres.VoiceStateRepository) *VoiceSignalingService {
	return &VoiceSignalingService{
		hub:            hub,
		voiceRepo:      voiceRepo,
		channelPeers:   make(map[uuid.UUID]map[uuid.UUID]*Client),
		speakingStates: make(map[uuid.UUID]bool),
	}
}

// HandleVoiceMessage handles incoming voice-related messages
func (s *VoiceSignalingService) HandleVoiceMessage(ctx context.Context, client *Client, sessionID string, msgType string, data json.RawMessage) error {
	switch msgType {
	case VoiceSignalJoin:
		return s.handleJoin(ctx, client, sessionID, data)
	case VoiceSignalLeave:
		return s.handleLeave(ctx, client, data)
	case VoiceSignalOffer:
		return s.handleOffer(ctx, client, data)
	case VoiceSignalAnswer:
		return s.handleAnswer(ctx, client, data)
	case VoiceSignalICECandidate:
		return s.handleICECandidate(ctx, client, data)
	case VoiceSignalSpeaking:
		return s.handleSpeaking(ctx, client, data)
	default:
		log.Printf("[Voice] Unknown message type: %s", msgType)
		return nil
	}
}

// handleJoin handles a user joining a voice channel
func (s *VoiceSignalingService) handleJoin(ctx context.Context, client *Client, sessionID string, data json.RawMessage) error {
	var joinData VoiceJoinData
	if err := json.Unmarshal(data, &joinData); err != nil {
		return err
	}

	log.Printf("[Voice] User %s joining channel %s", client.UserID, joinData.ChannelID)

	// Store voice state in database
	state, err := s.voiceRepo.JoinChannel(ctx, client.UserID, joinData.ChannelID, joinData.ServerID, sessionID)
	if err != nil {
		log.Printf("[Voice] Failed to join channel: %v", err)
		return err
	}

	// Update self state if provided
	if joinData.SelfMuted || joinData.SelfDeafened {
		state, err = s.voiceRepo.UpdateSelfState(ctx, client.UserID, joinData.ServerID,
			joinData.SelfMuted, joinData.SelfDeafened, false, false)
		if err != nil {
			log.Printf("[Voice] Failed to update self state: %v", err)
		}
	}

	// Track peer in memory
	s.channelPeersMu.Lock()
	if s.channelPeers[joinData.ChannelID] == nil {
		s.channelPeers[joinData.ChannelID] = make(map[uuid.UUID]*Client)
	}
	s.channelPeers[joinData.ChannelID][client.UserID] = client
	s.channelPeersMu.Unlock()

	// Get existing peers to notify them
	peers, err := s.voiceRepo.GetPeersInChannel(ctx, joinData.ChannelID, client.UserID)
	if err != nil {
		log.Printf("[Voice] Failed to get peers: %v", err)
	}

	// Broadcast voice state update to server
	stateData := VoiceStateData{
		UserID:       client.UserID,
		Username:     client.Username,
		ChannelID:    joinData.ChannelID,
		ServerID:     joinData.ServerID,
		SelfMuted:    state.SelfMuted,
		SelfDeafened: state.SelfDeafened,
		SelfVideo:    state.SelfVideo,
		SelfStream:   state.SelfStream,
		Muted:        state.Muted,
		Deafened:     state.Deafened,
		SessionID:    sessionID,
	}

	s.hub.SendToServer(joinData.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVoiceStateUpdate,
		Data: stateData,
	})

	// Send list of existing peers to the joining user
	peerList := make([]VoiceStateData, 0, len(peers))
	for _, p := range peers {
		peerList = append(peerList, VoiceStateData{
			UserID:       p.UserID,
			Username:     p.Username,
			DisplayName:  p.DisplayName,
			Avatar:       p.Avatar,
			ChannelID:    p.ChannelID,
			ServerID:     p.ServerID,
			SelfMuted:    p.SelfMuted,
			SelfDeafened: p.SelfDeafened,
			SelfVideo:    p.SelfVideo,
			SelfStream:   p.SelfStream,
			Muted:        p.Muted,
			Deafened:     p.Deafened,
			SessionID:    p.SessionID,
		})
	}

	// Send voice server update to the joining user (includes peer list)
	s.hub.SendToUser(client.UserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVoiceServerUpdate,
		Data: map[string]interface{}{
			"channel_id": joinData.ChannelID,
			"server_id":  joinData.ServerID,
			"peers":      peerList,
		},
	})

	return nil
}

// handleLeave handles a user leaving a voice channel
func (s *VoiceSignalingService) handleLeave(ctx context.Context, client *Client, data json.RawMessage) error {
	var leaveData VoiceLeaveData
	if err := json.Unmarshal(data, &leaveData); err != nil {
		return err
	}

	log.Printf("[Voice] User %s leaving channel %s", client.UserID, leaveData.ChannelID)

	// Remove from database
	if err := s.voiceRepo.LeaveChannel(ctx, client.UserID, leaveData.ServerID); err != nil {
		log.Printf("[Voice] Failed to leave channel: %v", err)
		return err
	}

	// Remove from memory tracking
	s.channelPeersMu.Lock()
	if peers, ok := s.channelPeers[leaveData.ChannelID]; ok {
		delete(peers, client.UserID)
		if len(peers) == 0 {
			delete(s.channelPeers, leaveData.ChannelID)
		}
	}
	s.channelPeersMu.Unlock()

	// Broadcast voice state update (null channel indicates left)
	s.hub.SendToServer(leaveData.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVoiceStateUpdate,
		Data: map[string]interface{}{
			"user_id":    client.UserID,
			"channel_id": nil,
			"server_id":  leaveData.ServerID,
		},
	})

	return nil
}

// handleOffer relays an SDP offer to a specific peer
func (s *VoiceSignalingService) handleOffer(ctx context.Context, client *Client, data json.RawMessage) error {
	var offerData VoiceOfferData
	if err := json.Unmarshal(data, &offerData); err != nil {
		return err
	}

	log.Printf("[Voice] Relaying offer from %s to %s", client.UserID, offerData.ToUserID)

	// Send to target user
	s.hub.SendToUser(offerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: VoiceSignalOffer,
		Data: map[string]interface{}{
			"from_user_id": client.UserID,
			"sdp":          offerData.SDP,
		},
	})

	return nil
}

// handleAnswer relays an SDP answer to a specific peer
func (s *VoiceSignalingService) handleAnswer(ctx context.Context, client *Client, data json.RawMessage) error {
	var answerData VoiceAnswerData
	if err := json.Unmarshal(data, &answerData); err != nil {
		return err
	}

	log.Printf("[Voice] Relaying answer from %s to %s", client.UserID, answerData.ToUserID)

	// Send to target user
	s.hub.SendToUser(answerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: VoiceSignalAnswer,
		Data: map[string]interface{}{
			"from_user_id": client.UserID,
			"sdp":          answerData.SDP,
		},
	})

	return nil
}

// handleICECandidate relays an ICE candidate to a specific peer
func (s *VoiceSignalingService) handleICECandidate(ctx context.Context, client *Client, data json.RawMessage) error {
	var candidateData VoiceICECandidateData
	if err := json.Unmarshal(data, &candidateData); err != nil {
		return err
	}

	// Send to target user
	s.hub.SendToUser(candidateData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: VoiceSignalICECandidate,
		Data: map[string]interface{}{
			"from_user_id":  client.UserID,
			"candidate":     candidateData.Candidate,
			"sdpMid":        candidateData.SDPMid,
			"sdpMLineIndex": candidateData.SDPMLineIndex,
		},
	})

	return nil
}

// handleSpeaking handles speaking state updates
func (s *VoiceSignalingService) handleSpeaking(ctx context.Context, client *Client, data json.RawMessage) error {
	var speakingData VoiceSpeakingData
	if err := json.Unmarshal(data, &speakingData); err != nil {
		return err
	}

	// Update in-memory state
	s.speakingStatesMu.Lock()
	s.speakingStates[client.UserID] = speakingData.Speaking
	s.speakingStatesMu.Unlock()

	// Get user's current voice state to find channel/server
	if s.voiceRepo == nil {
		return nil
	}
	state, err := s.voiceRepo.GetByUser(ctx, client.UserID)
	if err != nil || state == nil {
		return err
	}

	// Broadcast to all users in the channel
	s.channelPeersMu.RLock()
	peers := s.channelPeers[state.ChannelID]
	s.channelPeersMu.RUnlock()

	for userID, peerClient := range peers {
		if userID == client.UserID {
			continue // Don't send to self
		}

		eventData, _ := json.Marshal(map[string]interface{}{
			"user_id":    client.UserID,
			"channel_id": state.ChannelID,
			"speaking":   speakingData.Speaking,
		})

		select {
		case peerClient.send <- eventData:
		default:
		}
	}

	return nil
}

// UpdateSelfState handles voice state updates (mute/deafen)
func (s *VoiceSignalingService) UpdateSelfState(ctx context.Context, client *Client, serverID uuid.UUID, selfMuted, selfDeafened, selfVideo, selfStream bool) error {
	state, err := s.voiceRepo.UpdateSelfState(ctx, client.UserID, serverID, selfMuted, selfDeafened, selfVideo, selfStream)
	if err != nil {
		return err
	}

	// Broadcast update
	s.hub.SendToServer(serverID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVoiceStateUpdate,
		Data: VoiceStateData{
			UserID:       client.UserID,
			Username:     client.Username,
			ChannelID:    state.ChannelID,
			ServerID:     state.ServerID,
			SelfMuted:    state.SelfMuted,
			SelfDeafened: state.SelfDeafened,
			SelfVideo:    state.SelfVideo,
			SelfStream:   state.SelfStream,
			Muted:        state.Muted,
			Deafened:     state.Deafened,
		},
	})

	return nil
}

// CleanupSession removes voice states for a disconnected session
func (s *VoiceSignalingService) CleanupSession(ctx context.Context, sessionID string) error {
	states, err := s.voiceRepo.LeaveBySession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Broadcast leave events for each cleaned up state
	for _, state := range states {
		// Remove from memory tracking
		s.channelPeersMu.Lock()
		if peers, ok := s.channelPeers[state.ChannelID]; ok {
			delete(peers, state.UserID)
			if len(peers) == 0 {
				delete(s.channelPeers, state.ChannelID)
			}
		}
		s.channelPeersMu.Unlock()

		// Broadcast leave
		s.hub.SendToServer(state.ServerID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVoiceStateUpdate,
			Data: map[string]interface{}{
				"user_id":    state.UserID,
				"channel_id": nil,
				"server_id":  state.ServerID,
			},
		})
	}

	return nil
}

// GetChannelVoiceStates returns all voice states for a channel
func (s *VoiceSignalingService) GetChannelVoiceStates(ctx context.Context, channelID uuid.UUID) ([]postgres.VoiceStateWithUser, error) {
	return s.voiceRepo.GetByChannel(ctx, channelID)
}

// GetServerVoiceStates returns all voice states for a server
func (s *VoiceSignalingService) GetServerVoiceStates(ctx context.Context, serverID uuid.UUID) ([]postgres.VoiceStateWithUser, error) {
	return s.voiceRepo.GetByServer(ctx, serverID)
}
