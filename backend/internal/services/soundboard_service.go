package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/storage"
)

var (
	ErrSoundboardSoundNotFound = errors.New("soundboard sound not found")
	ErrSoundboardSoundInvalid  = errors.New("invalid soundboard sound")
	ErrSoundboardNameRequired  = errors.New("sound name is required")
	ErrSoundboardNameTooLong   = errors.New("sound name too long (max 100 chars)")
	ErrSoundboardTooLarge      = errors.New("sound file too large (max 5 seconds / 500KB)")
	ErrSoundboardFormat        = errors.New("invalid sound format (only MP3, OGG, WAV, OPUS allowed)")
	ErrSoundboardDuration      = errors.New("sound duration too long (max 5000ms)")
)

// SoundboardService handles soundboard business logic
type SoundboardService struct {
	mu      sync.RWMutex
	sounds  map[uuid.UUID]*models.SoundboardSound
	storage *storage.Service
}

// NewSoundboardService creates a new soundboard service
func NewSoundboardService(storageService *storage.Service) *SoundboardService {
	return &SoundboardService{
		sounds:  make(map[uuid.UUID]*models.SoundboardSound),
		storage: storageService,
	}
}

// AllowedAudioTypes defines the allowed audio MIME types for soundboard sounds
var AllowedAudioTypes = map[string]bool{
	"audio/mpeg":  true, // MP3
	"audio/ogg":   true, // OGG
	"audio/wave":  true, // WAV
	"audio/wav":   true, // WAV
	"audio/x-wav": true, // WAV
	"audio/opus":  true, // OPUS
	"audio/webm":  true, // WEBM (opus container)
}

// ValidateSoundUpload validates a sound file
func (s *SoundboardService) ValidateSoundUpload(file *multipart.FileHeader) error {
	// Max size: 500KB
	if file.Size > 500*1024 {
		return ErrSoundboardTooLarge
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !AllowedAudioTypes[contentType] {
		return ErrSoundboardFormat
	}

	return nil
}

// Create creates a new soundboard sound
func (s *SoundboardService) Create(
	ctx context.Context,
	serverID *uuid.UUID,
	name string,
	emojiName string,
	volume float64,
	file *multipart.FileHeader,
	uploadedBy uuid.UUID,
) (*models.SoundboardSound, error) {
	if name == "" {
		return nil, ErrSoundboardNameRequired
	}
	if len(name) > 100 {
		return nil, ErrSoundboardNameTooLong
	}

	// Validate file
	if err := s.ValidateSoundUpload(file); err != nil {
		return nil, err
	}

	// Set default volume
	if volume <= 0 {
		volume = 1.0
	}

	// Upload the file
	var url string
	var path string
	if s.storage != nil {
		fileInfo, err := s.storage.UploadFile(ctx, file, uploadedBy, "soundboard")
		if err != nil {
			return nil, fmt.Errorf("failed to upload sound: %w", err)
		}
		url = fileInfo.URL
		path = fileInfo.Path
	} else {
		// Fallback for testing
		ext := strings.ToLower(filepath.Ext(file.Filename))
		path = fmt.Sprintf("soundboard/%s/%s%s", uploadedBy.String()[:8], uuid.New().String(), ext)
		url = "/soundboard/" + uuid.New().String() + ext
		_ = path
	}

	// Estimate duration (placeholder - real implementation would use audio parsing)
	// Default to 1000ms if we can't determine actual duration
	durationMs := 1000

	sound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   serverID,
		Name:       name,
		EmojiName:  emojiName,
		Volume:     volume,
		AudioURL:   url,
		DurationMs: durationMs,
		Available:  true,
		CreatorID:  uploadedBy,
		CreatedAt:  time.Now(),
	}

	s.mu.Lock()
	s.sounds[sound.ID] = sound
	s.mu.Unlock()

	return sound, nil
}

// Get retrieves a soundboard sound by ID
func (s *SoundboardService) Get(ctx context.Context, soundID uuid.UUID) (*models.SoundboardSound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sound, ok := s.sounds[soundID]
	if !ok {
		return nil, ErrSoundboardSoundNotFound
	}
	return sound, nil
}

// GetByServer retrieves all soundboard sounds for a server
func (s *SoundboardService) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.SoundboardSound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sounds []*models.SoundboardSound
	for _, sound := range s.sounds {
		if sound.ServerID != nil && *sound.ServerID == serverID {
			sounds = append(sounds, sound)
		}
	}
	return sounds, nil
}

// GetDefault retrieves all default/global sounds
func (s *SoundboardService) GetDefault(ctx context.Context) ([]*models.SoundboardSound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sounds []*models.SoundboardSound
	for _, sound := range s.sounds {
		if sound.ServerID == nil {
			sounds = append(sounds, sound)
		}
	}
	return sounds, nil
}

// GetAvailable returns all sounds available to a user (default + server-specific)
func (s *SoundboardService) GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.SoundboardSound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sounds []*models.SoundboardSound
	for _, sound := range s.sounds {
		if !sound.Available {
			continue
		}
		if sound.ServerID == nil {
			// Default sound
			sounds = append(sounds, sound)
		} else if serverID != nil && *sound.ServerID == *serverID {
			// Server-specific sound
			sounds = append(sounds, sound)
		}
	}
	return sounds, nil
}

// Update updates a soundboard sound's properties
func (s *SoundboardService) Update(
	ctx context.Context,
	soundID uuid.UUID,
	name string,
	emojiName string,
	volume *float64,
	available *bool,
) (*models.SoundboardSound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sound, ok := s.sounds[soundID]
	if !ok {
		return nil, ErrSoundboardSoundNotFound
	}

	if name != "" {
		if len(name) > 100 {
			return nil, ErrSoundboardNameTooLong
		}
		sound.Name = name
	}

	if emojiName != "" {
		sound.EmojiName = emojiName
	}

	if volume != nil {
		sound.Volume = *volume
	}

	if available != nil {
		sound.Available = *available
	}

	return sound, nil
}

// Delete deletes a soundboard sound
func (s *SoundboardService) Delete(ctx context.Context, soundID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sounds[soundID]; !ok {
		return ErrSoundboardSoundNotFound
	}

	delete(s.sounds, soundID)
	return nil
}

// Search searches sounds by name
func (s *SoundboardService) Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.SoundboardSound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var sounds []*models.SoundboardSound

	for _, sound := range s.sounds {
		if !sound.Available {
			continue
		}

		// Filter by server if provided
		if serverID != nil {
			if sound.ServerID == nil {
				continue
			}
			if *sound.ServerID != *serverID {
				continue
			}
		}

		// Match name
		if strings.Contains(strings.ToLower(sound.Name), query) {
			sounds = append(sounds, sound)
			continue
		}

		// Match emoji name
		if strings.Contains(strings.ToLower(sound.EmojiName), query) {
			sounds = append(sounds, sound)
		}
	}

	return sounds, nil
}

// Add_Test is a test helper to add a sound directly
func (s *SoundboardService) Add_Test(sound *models.SoundboardSound) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sounds[sound.ID] = sound
}

// GetByID_Test is a test helper to get sound by ID directly
func (s *SoundboardService) GetByID_Test(soundID uuid.UUID) *models.SoundboardSound {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sounds[soundID]
}
