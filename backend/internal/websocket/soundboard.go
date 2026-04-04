package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// Soundboard signaling message types (client → server)
const (
	SoundboardSignalPlay  = "SOUNDBOARD_PLAY"
	SoundboardSignalStop = "SOUNDBOARD_STOP"
)

// SoundboardSoundData represents data for playing a sound
type SoundboardSoundData struct {
	SoundID   uuid.UUID `json:"sound_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
	Volume    float64   `json:"volume,omitempty"`
}

// SoundboardStopData represents data for stopping a sound
type SoundboardStopData struct {
	SoundID   uuid.UUID `json:"sound_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	ServerID  uuid.UUID `json:"server_id"`
}

// SoundboardSignalingService handles soundboard signaling within voice channels
type SoundboardSignalingService struct {
	hub HubInterface

	// SoundboardService for looking up sound details
	soundboardService interface {
		Get(ctx context.Context, soundID uuid.UUID) (*models.SoundboardSound, error)
	}

	// Track which sounds are currently playing in which channels
	// channelID -> soundID -> playingInfo
	activeSounds   map[uuid.UUID]map[uuid.UUID]*models.SoundboardPlayingSound
	activeSoundsMu sync.RWMutex
}

// NewSoundboardSignalingService creates a new soundboard signaling service
func NewSoundboardSignalingService(hub HubInterface, soundboardService interface {
	Get(ctx context.Context, soundID uuid.UUID) (*models.SoundboardSound, error)
}) *SoundboardSignalingService {
	return &SoundboardSignalingService{
		hub:              hub,
		soundboardService: soundboardService,
		activeSounds:     make(map[uuid.UUID]map[uuid.UUID]*models.SoundboardPlayingSound),
	}
}

// HandleSoundboardMessage handles incoming soundboard-related messages
func (s *SoundboardSignalingService) HandleSoundboardMessage(ctx context.Context, client *Client, sessionID string, msgType string, data json.RawMessage) error {
	switch msgType {
	case SoundboardSignalPlay:
		return s.handlePlay(ctx, client, data)
	case SoundboardSignalStop:
		return s.handleStop(ctx, client, data)
	default:
		log.Printf("[Soundboard] Unknown message type: %s", msgType)
		return nil
	}
}

// handlePlay handles a user playing a sound in a voice channel
func (s *SoundboardSignalingService) handlePlay(ctx context.Context, client *Client, data json.RawMessage) error {
	var playData SoundboardSoundData
	if err := json.Unmarshal(data, &playData); err != nil {
		return err
	}

	log.Printf("[Soundboard] User %s playing sound %s in channel %s", client.UserID, playData.SoundID, playData.ChannelID)

	// Get sound details
	sound, err := s.soundboardService.Get(ctx, playData.SoundID)
	if err != nil {
		log.Printf("[Soundboard] Sound not found: %s", playData.SoundID)
		return nil // Silently fail - don't expose not found to client
	}

	if !sound.Available {
		log.Printf("[Soundboard] Sound not available: %s", playData.SoundID)
		return nil
	}

	// Use provided volume or sound's default
	volume := playData.Volume
	if volume <= 0 {
		volume = sound.Volume
	}

	playingSound := &models.SoundboardPlayingSound{
		SoundID:    sound.ID,
		SoundName:  sound.Name,
		EmojiName:  sound.EmojiName,
		AudioURL:   sound.AudioURL,
		Volume:     volume,
		DurationMs: sound.DurationMs,
		StartedAt:  time.Now(),
		PlayedBy:   client.UserID,
	}

	// Track active sound
	s.activeSoundsMu.Lock()
	if s.activeSounds[playData.ChannelID] == nil {
		s.activeSounds[playData.ChannelID] = make(map[uuid.UUID]*models.SoundboardPlayingSound)
	}
	s.activeSounds[playData.ChannelID][playData.SoundID] = playingSound
	s.activeSoundsMu.Unlock()

	// Auto-cleanup after duration
	go func() {
		time.Sleep(time.Duration(sound.DurationMs) * time.Millisecond)
		s.activeSoundsMu.Lock()
		if channelSounds, ok := s.activeSounds[playData.ChannelID]; ok {
			delete(channelSounds, playData.SoundID)
			if len(channelSounds) == 0 {
				delete(s.activeSounds, playData.ChannelID)
			}
		}
		s.activeSoundsMu.Unlock()
	}()

	// Broadcast play event to all users in the server
	playEvent := models.SoundboardPlayEvent{
		SoundID:    sound.ID.String(),
		SoundName:  sound.Name,
		EmojiName:  sound.EmojiName,
		AudioURL:   sound.AudioURL,
		Volume:     volume,
		DurationMs: sound.DurationMs,
		UserID:     client.UserID.String(),
		ChannelID:  playData.ChannelID.String(),
		ServerID:   playData.ServerID.String(),
	}

	s.hub.SendToServer(playData.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventSoundboardPlay,
		Data: playEvent,
	})

	return nil
}

// handleStop handles a user stopping a sound in a voice channel
func (s *SoundboardSignalingService) handleStop(ctx context.Context, client *Client, data json.RawMessage) error {
	var stopData SoundboardStopData
	if err := json.Unmarshal(data, &stopData); err != nil {
		return err
	}

	log.Printf("[Soundboard] User %s stopping sound %s in channel %s", client.UserID, stopData.SoundID, stopData.ChannelID)

	// Remove from active sounds
	s.activeSoundsMu.Lock()
	if channelSounds, ok := s.activeSounds[stopData.ChannelID]; ok {
		if sound, ok := channelSounds[stopData.SoundID]; ok {
			// Only the user who started it can stop it
			if sound.PlayedBy != client.UserID {
				s.activeSoundsMu.Unlock()
				return nil
			}
			delete(channelSounds, stopData.SoundID)
			if len(channelSounds) == 0 {
				delete(s.activeSounds, stopData.ChannelID)
			}
		}
	}
	s.activeSoundsMu.Unlock()

	// Broadcast stop event to all users in the server
	stopEvent := map[string]interface{}{
		"sound_id":   stopData.SoundID.String(),
		"channel_id": stopData.ChannelID.String(),
		"server_id":  stopData.ServerID.String(),
		"user_id":    client.UserID.String(),
	}

	s.hub.SendToServer(stopData.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventSoundboardStop,
		Data: stopEvent,
	})

	return nil
}

// GetActiveSounds returns all sounds currently playing in a channel
func (s *SoundboardSignalingService) GetActiveSounds(ctx context.Context, channelID uuid.UUID) []*models.SoundboardPlayingSound {
	s.activeSoundsMu.RLock()
	defer s.activeSoundsMu.RUnlock()

	var sounds []*models.SoundboardPlayingSound
	if channelSounds, ok := s.activeSounds[channelID]; ok {
		for _, sound := range channelSounds {
			sounds = append(sounds, sound)
		}
	}
	return sounds
}
