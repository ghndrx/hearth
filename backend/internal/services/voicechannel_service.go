package services

import (
	"errors"
	"fmt"
)

// Token represents a unique identifier for a user session.
type Token string

// VoiceParticipant represents the state of a user in a voice channel.
type VoiceParticipant struct {
	Token       Token
	IsDeafened  bool
	IsMuted     bool
	IsSelfDeafened bool
	IsSelfMuted bool
}

// WSClient defines the interface for the websocket connection to handle client command routing.
type WSClient interface {
	// SendMessage sends a WebSocket message to the specific client.
	// The payload must be marshalled to JSON.
	SendMessage(payload interface{}) error
	
	// ClientID returns the unique identifier of the client this socket belongs to.
	ClientID() Token
}

// VoiceChannelService handles the state and lifecycle of a single voice channel.
type VoiceChannelService struct {
	participants map[Token]*VoiceParticipant
	// In a real application, you would subscribe to a EventBus here or use callbacks.
}

// NewVoiceChannelService creates a new voice channel instance.
func NewVoiceChannelService() *VoiceChannelService {
	return &VoiceChannelService{
		participants: make(map[Token]*VoiceParticipant),
	}
}

// Join attempts to join a channel. Overwrites if user already exists to sync state.
func (s *VoiceChannelService) Join(client WSClient) error {
	token := client.ClientID()
	
	// Initialize participant state
	participant := &VoiceParticipant{
		Token: token,
		// Default to deafening audio from self for privacy UX, 
		// though technically in Go logic we need to manage it.
	}

	s.participants[token] = participant

	// Emit an event to the channel so the frontend renders the new member.
	// This simulates a real backend broadcasting state changes.
	// e.g., emitStateChange(JoinEvent, participant)
	
	return nil
}

// Leave removes a client from the channel.
func (s *VoiceChannelService) Leave(client WSClient) error {
	token := client.ClientID()
	if _, exists := s.participants[token]; !exists {
		return errors.New("user not in channel")
	}

	delete(s.participants, token)
	// emitStateChange(LeaveEvent, token)...

	return nil
}

// ToggleMute toggles a user's mute state.
func (s *VoiceChannelService) ToggleMute(client WSClient) error {
	return s.updateAudioState(client, "Mute")
}

// ToggleDeafen toggles a user's deafen state.
func (s *VoiceChannelService) ToggleDeafen(client WSClient) error {
	return s.updateAudioState(client, "Deafen")
}

// ToggleSelfMute toggles a user's local mute state (Microphone toggle).
func (s *VoiceChannelService) ToggleSelfMute(client WSClient) error {
	return s.updateAudioState(client, "SelfMute")
}

// ToggleSelfDeafen toggles a user's local deafen state (Headphones mute toggle).
func (s *VoiceChannelService) ToggleSelfDeafen(client WSClient) error {
	return s.updateAudioState(client, "SelfDeafen")
}

// updateAudioState handles the logic for changing mute/deafen states.
func (s *VoiceChannelService) updateAudioState(client WSClient, mode string) error {
	token := client.ClientID()
	p, exists := s.participants[token]
	if !exists {
		return fmt.Errorf("cannot %s: user %s is not in channel", mode, token)
	}

	switch mode {
	case "Mute":
		p.IsMuted = !p.IsMuted
	case "Deafen":
		p.IsDeafened = !p.IsDeafened
	case "SelfMute":
		p.IsSelfMuted = !p.IsSelfMuted
	case "SelfDeafen":
		p.IsSelfDeafened = !p.IsSelfDeafened
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	// For the frontend config (embedded JS/Svelte), we might emit a specific 
	// structured payload containing the new flags.
	return nil
}

// GetParticipants returns a copy of the current channel participants.
func (s *VoiceChannelService) GetParticipants() []VoiceParticipant {
	list := make([]VoiceParticipant, 0, len(s.participants))
	for _, p := range s.participants {
		list = append(list, *p)
	}
	return list
}