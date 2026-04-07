package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Video call types
const (
	VideoCallTypeDirect  = "direct"
	VideoCallTypeGroup   = "group"
	VideoCallTypeChannel = "channel"
)

// Video call states
const (
	VideoStateIdle         = "idle"
	VideoStateRingingOut   = "ringing_out"
	VideoStateRingingIn    = "ringing_in"
	VideoStateConnecting   = "connecting"
	VideoStateConnected    = "connected"
	VideoStateReconnecting = "reconnecting"
	VideoStateEnded        = "ended"
)

// Video signaling message types (client → server)
const (
	VideoSignalRing          = "VIDEO_RING"
	VideoSignalRingResponse  = "VIDEO_RING_RESPONSE"
	VideoSignalLeave         = "VIDEO_LEAVE"
	VideoSignalOffer         = "VIDEO_OFFER"
	VideoSignalAnswer        = "VIDEO_ANSWER"
	VideoSignalICECandidate  = "VIDEO_ICE_CANDIDATE"
	VideoSignalStateUpdate   = "VIDEO_STATE_UPDATE"
	VideoSignalScreenStart   = "VIDEO_SCREEN_START"
	VideoSignalScreenStop    = "VIDEO_SCREEN_STOP"
)

// Video event types (server → client)
const (
	EventTypeVideoRingStart     = "VIDEO_RING_START"
	EventTypeVideoRingAccept    = "VIDEO_RING_ACCEPT"
	EventTypeVideoRingDecline   = "VIDEO_RING_DECLINE"
	EventTypeVideoRingEnd       = "VIDEO_RING_END"
	EventTypeVideoStateUpdate    = "VIDEO_STATE_UPDATE"
	EventTypeVideoServerUpdate   = "VIDEO_SERVER_UPDATE"
	EventTypeVideoOffer         = "VIDEO_OFFER"
	EventTypeVideoAnswer        = "VIDEO_ANSWER"
	EventTypeVideoICECandidate  = "VIDEO_ICE_CANDIDATE"
)

// VideoCallData represents data for initiating a video call
type VideoCallData struct {
	ChannelID  uuid.UUID `json:"channel_id"`
	ServerID   uuid.UUID `json:"server_id,omitempty"`
	CallType   string    `json:"call_type"`
	ToUserID   uuid.UUID `json:"to_user_id,omitempty"`
	IsGroup    bool      `json:"is_group"`
}

// VideoRingResponseData represents accept/decline response
type VideoRingResponseData struct {
	CallID     uuid.UUID `json:"call_id"`
	Accept     bool      `json:"accept"`
	ToUserID   uuid.UUID `json:"to_user_id,omitempty"`
}

// VideoLeaveData represents leaving a video call
type VideoLeaveData struct {
	CallID    uuid.UUID `json:"call_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id,omitempty"`
}

// VideoOfferData represents an SDP offer
type VideoOfferData struct {
	CallID    uuid.UUID `json:"call_id"`
	SDP       string    `json:"sdp"`
	ToUserID  uuid.UUID `json:"to_user_id"`
}

// VideoAnswerData represents an SDP answer
type VideoAnswerData struct {
	CallID    uuid.UUID `json:"call_id"`
	SDP       string    `json:"sdp"`
	ToUserID  uuid.UUID `json:"to_user_id"`
}

// VideoICECandidateData represents an ICE candidate
type VideoICECandidateData struct {
	CallID         uuid.UUID `json:"call_id"`
	Candidate      string    `json:"candidate"`
	SDPMid         string    `json:"sdpMid"`
	SDPMLineIndex  int       `json:"sdpMLineIndex"`
	ToUserID       uuid.UUID `json:"to_user_id"`
}

// VideoStateUpdateData represents video state update
type VideoStateUpdateData struct {
	CallID       uuid.UUID `json:"call_id"`
	UserID       uuid.UUID `json:"user_id,omitempty"`
	IsCameraOn   bool      `json:"is_camera_on"`
	IsMuted      bool      `json:"is_muted"`
	IsScreenShare bool     `json:"is_screen_share"`
}

// VideoStateData represents video state for broadcasting
type VideoStateData struct {
	CallID       uuid.UUID            `json:"call_id"`
	ChannelID    uuid.UUID            `json:"channel_id"`
	ServerID     uuid.UUID            `json:"server_id,omitempty"`
	CallType     string               `json:"call_type"`
	State        string               `json:"state"`
	InitiatorID  uuid.UUID            `json:"initiator_id,omitempty"`
	Participants []VideoParticipantData `json:"participants,omitempty"`
	StartedAt    *time.Time           `json:"started_at,omitempty"`
	EndedAt      *time.Time           `json:"ended_at,omitempty"`
}

// VideoParticipantData represents a participant in a video call
type VideoParticipantData struct {
	UserID        uuid.UUID `json:"user_id"`
	Username      string    `json:"username"`
	DisplayName   *string   `json:"display_name,omitempty"`
	Avatar        *string   `json:"avatar,omitempty"`
	IsCameraOn    bool      `json:"is_camera_on"`
	IsScreenShare bool      `json:"is_screen_share"`
	IsMuted       bool      `json:"is_muted"`
	JoinedAt      time.Time `json:"joined_at"`
	SessionID     string    `json:"session_id,omitempty"`
}

// VideoCallInfo represents an active video call
type VideoCallInfo struct {
	CallID       uuid.UUID
	ChannelID    uuid.UUID
	ServerID     uuid.UUID
	CallType     string
	State        string
	InitiatorID  uuid.UUID
	StartedAt    time.Time
	Participants map[uuid.UUID]*VideoParticipantData
	peers        map[uuid.UUID]*Client // userID -> client
	mu           sync.RWMutex
}

// VideoSignalingService handles WebRTC signaling for video calls
type VideoSignalingService struct {
	hub       HubInterface
	voiceRepo interface {
		GetByUser(ctx context.Context, userID uuid.UUID) (interface{}, error)
	}

	// Track active calls
	calls   map[uuid.UUID]*VideoCallInfo // callID -> call
	callsMu sync.RWMutex

	// Track pending ring requests (userID -> callID)
	pendingRings   map[uuid.UUID]uuid.UUID // userID -> callID
	pendingRingsMu sync.RWMutex

	// Track user to call mapping
	userCalls   map[uuid.UUID]uuid.UUID // userID -> callID
	userCallsMu sync.RWMutex
}

// NewVideoSignalingService creates a new video signaling service
func NewVideoSignalingService(hub HubInterface) *VideoSignalingService {
	return &VideoSignalingService{
		hub:          hub,
		calls:        make(map[uuid.UUID]*VideoCallInfo),
		pendingRings: make(map[uuid.UUID]uuid.UUID),
		userCalls:    make(map[uuid.UUID]uuid.UUID),
	}
}

// HandleVideoMessage handles incoming video-related messages
func (s *VideoSignalingService) HandleVideoMessage(ctx context.Context, client *Client, sessionID string, msgType string, data json.RawMessage) error {
	switch msgType {
	case VideoSignalRing:
		return s.handleRing(ctx, client, sessionID, data)
	case VideoSignalRingResponse:
		return s.handleRingResponse(ctx, client, sessionID, data)
	case VideoSignalLeave:
		return s.handleLeave(ctx, client, data)
	case VideoSignalOffer:
		return s.handleOffer(ctx, client, data)
	case VideoSignalAnswer:
		return s.handleAnswer(ctx, client, data)
	case VideoSignalICECandidate:
		return s.handleICECandidate(ctx, client, data)
	case VideoSignalStateUpdate:
		return s.handleStateUpdate(ctx, client, data)
	case VideoSignalScreenStart:
		return s.handleScreenStart(ctx, client, data)
	case VideoSignalScreenStop:
		return s.handleScreenStop(ctx, client, data)
	default:
		log.Printf("[Video] Unknown message type: %s", msgType)
		return nil
	}
}

// SignalVideo relays signaling data for WebRTC negotiation via HTTP.
// This is the HTTP equivalent of HandleVideoMessage, but accepts senderID
// from the auth context instead of extracting it from a WebSocket client.
// It handles VIDEO_OFFER, VIDEO_ANSWER, and VIDEO_ICE_CANDIDATE by relaying
// to the target user via the hub.
func (s *VideoSignalingService) SignalVideo(ctx context.Context, senderID uuid.UUID, signalType string, data json.RawMessage) error {
	switch signalType {
	case "VIDEO_OFFER":
		return s.signalOfferHTTP(ctx, senderID, data)
	case "VIDEO_ANSWER":
		return s.signalAnswerHTTP(ctx, senderID, data)
	case "VIDEO_ICE_CANDIDATE":
		return s.signalICECandidateHTTP(ctx, senderID, data)
	default:
		log.Printf("[Video] SignalVideo: unknown signal type: %s", signalType)
		return fmt.Errorf("unknown signal type: %s", signalType)
	}
}

// signalOfferHTTP relays an SDP offer to a specific peer (HTTP variant)
func (s *VideoSignalingService) signalOfferHTTP(ctx context.Context, senderID uuid.UUID, data json.RawMessage) error {
	var offerData VideoOfferData
	if err := json.Unmarshal(data, &offerData); err != nil {
		return fmt.Errorf("invalid offer data: %w", err)
	}

	log.Printf("[Video] SignalVideo: relaying offer from %s to %s", senderID, offerData.ToUserID)

	s.hub.SendToUser(offerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoOffer,
		Data: map[string]interface{}{
			"call_id":     offerData.CallID,
			"from_user_id": senderID,
			"sdp":         offerData.SDP,
		},
	})

	return nil
}

// signalAnswerHTTP relays an SDP answer to a specific peer (HTTP variant)
func (s *VideoSignalingService) signalAnswerHTTP(ctx context.Context, senderID uuid.UUID, data json.RawMessage) error {
	var answerData VideoAnswerData
	if err := json.Unmarshal(data, &answerData); err != nil {
		return fmt.Errorf("invalid answer data: %w", err)
	}

	log.Printf("[Video] SignalVideo: relaying answer from %s to %s", senderID, answerData.ToUserID)

	s.hub.SendToUser(answerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoAnswer,
		Data: map[string]interface{}{
			"call_id":     answerData.CallID,
			"from_user_id": senderID,
			"sdp":         answerData.SDP,
		},
	})

	return nil
}

// signalICECandidateHTTP relays an ICE candidate to a specific peer (HTTP variant)
func (s *VideoSignalingService) signalICECandidateHTTP(ctx context.Context, senderID uuid.UUID, data json.RawMessage) error {
	var candidateData VideoICECandidateData
	if err := json.Unmarshal(data, &candidateData); err != nil {
		return fmt.Errorf("invalid ICE candidate data: %w", err)
	}

	log.Printf("[Video] SignalVideo: relaying ICE candidate from %s to %s", senderID, candidateData.ToUserID)

	s.hub.SendToUser(candidateData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoICECandidate,
		Data: map[string]interface{}{
			"call_id":       candidateData.CallID,
			"from_user_id":  senderID,
			"candidate":     candidateData.Candidate,
			"sdpMid":        candidateData.SDPMid,
			"sdpMLineIndex": candidateData.SDPMLineIndex,
		},
	})

	return nil
}

// handleRing initiates a video call (rings the target user)
func (s *VideoSignalingService) handleRing(ctx context.Context, client *Client, sessionID string, data json.RawMessage) error {
	var ringData VideoCallData
	if err := json.Unmarshal(data, &ringData); err != nil {
		return err
	}

	log.Printf("[Video] User %s initiating video call to %s in channel %s", 
		client.UserID, ringData.ToUserID, ringData.ChannelID)

	// Create a new call
	callID := uuid.New()
	startedAt := time.Now()
	
	call := &VideoCallInfo{
		CallID:       callID,
		ChannelID:    ringData.ChannelID,
		ServerID:     ringData.ServerID,
		CallType:     ringData.CallType,
		State:        VideoStateRingingOut,
		InitiatorID:  client.UserID,
		StartedAt:    startedAt,
		Participants: make(map[uuid.UUID]*VideoParticipantData),
		peers:        make(map[uuid.UUID]*Client),
	}

	// Add initiator as first participant
	call.Participants[client.UserID] = &VideoParticipantData{
		UserID:      client.UserID,
		Username:    client.Username,
		IsCameraOn:  false,
		IsScreenShare: false,
		IsMuted:     true,
		JoinedAt:    startedAt,
		SessionID:   sessionID,
	}
	call.peers[client.UserID] = client

	// Store the call
	s.callsMu.Lock()
	s.calls[callID] = call
	s.callsMu.Unlock()

	// Track user's call
	s.userCallsMu.Lock()
	s.userCalls[client.UserID] = callID
	s.userCallsMu.Unlock()

	// Store pending ring for the target user
	if ringData.ToUserID != uuid.Nil {
		s.pendingRingsMu.Lock()
		s.pendingRings[ringData.ToUserID] = callID
		s.pendingRingsMu.Unlock()

		// Send ring start event to the target user
		s.hub.SendToUser(ringData.ToUserID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVideoRingStart,
			Data: map[string]interface{}{
				"call_id":      callID,
				"channel_id":   ringData.ChannelID,
				"server_id":    ringData.ServerID,
				"call_type":    ringData.CallType,
				"from_user_id": client.UserID,
				"from_username": client.Username,
			},
		})
	}

	// Confirm to the initiator that ring was sent
	s.hub.SendToUser(client.UserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoServerUpdate,
		Data: VideoStateData{
			CallID:      callID,
			ChannelID:   ringData.ChannelID,
			ServerID:    ringData.ServerID,
			CallType:    ringData.CallType,
			State:       VideoStateRingingOut,
			InitiatorID: client.UserID,
			StartedAt:   &startedAt,
		},
	})

	return nil
}

// handleRingResponse handles accept/decline of incoming call
func (s *VideoSignalingService) handleRingResponse(ctx context.Context, client *Client, sessionID string, data json.RawMessage) error {
	var responseData VideoRingResponseData
	if err := json.Unmarshal(data, &responseData); err != nil {
		return err
	}

	log.Printf("[Video] User %s responding to video call %s: accept=%v", 
		client.UserID, responseData.CallID, responseData.Accept)

	s.callsMu.Lock()
	call, exists := s.calls[responseData.CallID]
	s.callsMu.Unlock()

	if !exists {
		log.Printf("[Video] Call %s not found", responseData.CallID)
		return nil
	}

	if responseData.Accept {
		// Accept the call
		s.callsMu.Lock()
		call.State = VideoStateConnecting
		call.Participants[client.UserID] = &VideoParticipantData{
			UserID:       client.UserID,
			Username:     client.Username,
			IsCameraOn:   false,
			IsScreenShare: false,
			IsMuted:      true,
			JoinedAt:     time.Now(),
			SessionID:    sessionID,
		}
		call.peers[client.UserID] = client
		s.callsMu.Unlock()

		// Track user's call
		s.userCallsMu.Lock()
		s.userCalls[client.UserID] = responseData.CallID
		s.userCallsMu.Unlock()

		// Remove pending ring
		s.pendingRingsMu.Lock()
		delete(s.pendingRings, client.UserID)
		s.pendingRingsMu.Unlock()

		// Send accept to initiator
		s.hub.SendToUser(call.InitiatorID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVideoRingAccept,
			Data: map[string]interface{}{
				"call_id":     call.CallID,
				"from_user_id": client.UserID,
				"from_username": client.Username,
			},
		})

		// Send full call state to the acceptor
		s.hub.SendToUser(client.UserID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVideoServerUpdate,
			Data: s.buildCallState(call),
		})

		// Notify all participants of state update
		s.broadcastToCall(call, EventTypeVideoStateUpdate, map[string]interface{}{
			"call_id":    call.CallID,
			"user_id":    client.UserID,
			"is_joined":   true,
		}, client.UserID)

	} else {
		// Decline the call
		// Remove pending ring
		s.pendingRingsMu.Lock()
		delete(s.pendingRings, client.UserID)
		s.pendingRingsMu.Unlock()

		// Notify initiator of decline
		s.hub.SendToUser(call.InitiatorID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVideoRingDecline,
			Data: map[string]interface{}{
				"call_id":     call.CallID,
				"from_user_id": client.UserID,
				"from_username": client.Username,
			},
		})

		// Clean up the call
		s.cleanupCall(responseData.CallID)
	}

	return nil
}

// handleLeave handles leaving a video call
func (s *VideoSignalingService) handleLeave(ctx context.Context, client *Client, data json.RawMessage) error {
	var leaveData VideoLeaveData
	if err := json.Unmarshal(data, &leaveData); err != nil {
		return err
	}

	log.Printf("[Video] User %s leaving video call %s", client.UserID, leaveData.CallID)

	s.callsMu.Lock()
	call, exists := s.calls[leaveData.CallID]
	s.callsMu.Unlock()

	if !exists {
		return nil
	}

	s.callsMu.Lock()
	// Remove participant
	delete(call.Participants, client.UserID)
	delete(call.peers, client.UserID)
	s.callsMu.Unlock()

	// Remove user from call tracking
	s.userCallsMu.Lock()
	delete(s.userCalls, client.UserID)
	s.userCallsMu.Unlock()

	// Notify other participants
	s.broadcastToCall(call, EventTypeVideoStateUpdate, map[string]interface{}{
		"call_id":  call.CallID,
		"user_id":  client.UserID,
		"is_left":  true,
	}, client.UserID)

	// If no more participants, end the call
	s.callsMu.Lock()
	if len(call.Participants) == 0 {
		call.State = VideoStateEnded
		s.callsMu.Unlock()
		s.cleanupCall(leaveData.CallID)
	} else if call.State == VideoStateRingingOut || call.State == VideoStateConnecting {
		// If the initiator left during ring/connect, end for everyone
		s.hub.SendToUser(call.InitiatorID, &Event{
			Op:   OpDispatch,
			Type: EventTypeVideoRingEnd,
			Data: map[string]interface{}{
				"call_id": call.CallID,
				"reason":  "user_left",
			},
		})
		s.callsMu.Unlock()
		s.cleanupCall(leaveData.CallID)
	} else {
		s.callsMu.Unlock()
	}

	return nil
}

// handleOffer relays an SDP offer to a specific peer
func (s *VideoSignalingService) handleOffer(ctx context.Context, client *Client, data json.RawMessage) error {
	var offerData VideoOfferData
	if err := json.Unmarshal(data, &offerData); err != nil {
		return err
	}

	log.Printf("[Video] Relaying video offer from %s to %s", client.UserID, offerData.ToUserID)

	// Send to target user
	s.hub.SendToUser(offerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoOffer,
		Data: map[string]interface{}{
			"call_id":     offerData.CallID,
			"from_user_id": client.UserID,
			"sdp":         offerData.SDP,
		},
	})

	return nil
}

// handleAnswer relays an SDP answer to a specific peer
func (s *VideoSignalingService) handleAnswer(ctx context.Context, client *Client, data json.RawMessage) error {
	var answerData VideoAnswerData
	if err := json.Unmarshal(data, &answerData); err != nil {
		return err
	}

	log.Printf("[Video] Relaying video answer from %s to %s", client.UserID, answerData.ToUserID)

	// Send to target user
	s.hub.SendToUser(answerData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoAnswer,
		Data: map[string]interface{}{
			"call_id":     answerData.CallID,
			"from_user_id": client.UserID,
			"sdp":         answerData.SDP,
		},
	})

	return nil
}

// handleICECandidate relays an ICE candidate to a specific peer
func (s *VideoSignalingService) handleICECandidate(ctx context.Context, client *Client, data json.RawMessage) error {
	var candidateData VideoICECandidateData
	if err := json.Unmarshal(data, &candidateData); err != nil {
		return err
	}

	// Send to target user
	s.hub.SendToUser(candidateData.ToUserID, &Event{
		Op:   OpDispatch,
		Type: EventTypeVideoICECandidate,
		Data: map[string]interface{}{
			"call_id":      candidateData.CallID,
			"from_user_id": client.UserID,
			"candidate":    candidateData.Candidate,
			"sdpMid":       candidateData.SDPMid,
			"sdpMLineIndex": candidateData.SDPMLineIndex,
		},
	})

	return nil
}

// handleStateUpdate handles video state updates (camera, mute, etc.)
func (s *VideoSignalingService) handleStateUpdate(ctx context.Context, client *Client, data json.RawMessage) error {
	var stateData VideoStateUpdateData
	if err := json.Unmarshal(data, &stateData); err != nil {
		return err
	}

	s.callsMu.Lock()
	call, exists := s.calls[stateData.CallID]
	s.callsMu.Unlock()

	if !exists {
		return nil
	}

	s.callsMu.Lock()
	if participant, ok := call.Participants[client.UserID]; ok {
		participant.IsCameraOn = stateData.IsCameraOn
		participant.IsMuted = stateData.IsMuted
	}
	s.callsMu.Unlock()

	// Broadcast to all participants
	s.broadcastToCall(call, EventTypeVideoStateUpdate, map[string]interface{}{
		"call_id":       call.CallID,
		"user_id":       client.UserID,
		"is_camera_on":  stateData.IsCameraOn,
		"is_muted":      stateData.IsMuted,
		"is_screen_share": stateData.IsScreenShare,
	}, client.UserID)

	return nil
}

// handleScreenStart handles screen share start
func (s *VideoSignalingService) handleScreenStart(ctx context.Context, client *Client, data json.RawMessage) error {
	var stateData VideoStateUpdateData
	if err := json.Unmarshal(data, &stateData); err != nil {
		return err
	}

	s.callsMu.Lock()
	call, exists := s.calls[stateData.CallID]
	s.callsMu.Unlock()

	if !exists {
		return nil
	}

	s.callsMu.Lock()
	if participant, ok := call.Participants[client.UserID]; ok {
		participant.IsScreenShare = true
	}
	s.callsMu.Unlock()

	// Broadcast screen share start
	s.broadcastToCall(call, EventTypeVideoStateUpdate, map[string]interface{}{
		"call_id":        call.CallID,
		"user_id":        client.UserID,
		"is_screen_share": true,
	}, client.UserID)

	return nil
}

// handleScreenStop handles screen share stop
func (s *VideoSignalingService) handleScreenStop(ctx context.Context, client *Client, data json.RawMessage) error {
	var stateData VideoStateUpdateData
	if err := json.Unmarshal(data, &stateData); err != nil {
		return err
	}

	s.callsMu.Lock()
	call, exists := s.calls[stateData.CallID]
	s.callsMu.Unlock()

	if !exists {
		return nil
	}

	s.callsMu.Lock()
	if participant, ok := call.Participants[client.UserID]; ok {
		participant.IsScreenShare = false
	}
	s.callsMu.Unlock()

	// Broadcast screen share stop
	s.broadcastToCall(call, EventTypeVideoStateUpdate, map[string]interface{}{
		"call_id":        call.CallID,
		"user_id":        client.UserID,
		"is_screen_share": false,
	}, client.UserID)

	return nil
}

// broadcastToCall sends an event to all participants in a call
func (s *VideoSignalingService) broadcastToCall(call *VideoCallInfo, eventType string, data interface{}, excludeUserID uuid.UUID) {
	call.mu.RLock()
	defer call.mu.RUnlock()

	for userID, client := range call.peers {
		if userID == excludeUserID {
			continue
		}

		eventData, _ := json.Marshal(data)
		select {
		case client.send <- eventData:
		default:
			log.Printf("[Video] Failed to send to client %s", userID)
		}
	}
}

// buildCallState builds a VideoStateData from a call
func (s *VideoSignalingService) buildCallState(call *VideoCallInfo) VideoStateData {
	call.mu.RLock()
	defer call.mu.RUnlock()

	participants := make([]VideoParticipantData, 0, len(call.Participants))
	for _, p := range call.Participants {
		if p != nil {
			participants = append(participants, *p)
		}
	}

	startedAt := call.StartedAt
	return VideoStateData{
		CallID:       call.CallID,
		ChannelID:    call.ChannelID,
		ServerID:     call.ServerID,
		CallType:    call.CallType,
		State:       call.State,
		InitiatorID: call.InitiatorID,
		Participants: participants,
		StartedAt:   &startedAt,
	}
}

// cleanupCall removes a call and cleans up all references
func (s *VideoSignalingService) cleanupCall(callID uuid.UUID) {
	s.callsMu.Lock()
	call, exists := s.calls[callID]
	if exists {
		// Remove all users from call tracking
		for userID := range call.Participants {
			delete(s.userCalls, userID)
		}
	}
	delete(s.calls, callID)
	s.callsMu.Unlock()

	log.Printf("[Video] Cleaned up call %s", callID)
}

// GetCall returns a call by ID
func (s *VideoSignalingService) GetCall(callID uuid.UUID) *VideoCallInfo {
	s.callsMu.RLock()
	defer s.callsMu.RUnlock()
	return s.calls[callID]
}

// GetUserCall returns the call ID a user is in
func (s *VideoSignalingService) GetUserCall(userID uuid.UUID) uuid.UUID {
	s.userCallsMu.RLock()
	defer s.userCallsMu.RUnlock()
	return s.userCalls[userID]
}

// GetPendingRing returns the pending call ID for a user
func (s *VideoSignalingService) GetPendingRing(userID uuid.UUID) uuid.UUID {
	s.pendingRingsMu.RLock()
	defer s.pendingRingsMu.RUnlock()
	return s.pendingRings[userID]
}

// SetVoiceRepo sets the voice repository for integration
func (s *VideoSignalingService) SetVoiceRepo(repo interface{ GetByUser(ctx context.Context, userID uuid.UUID) (interface{}, error) }) {
	s.voiceRepo = repo
}
