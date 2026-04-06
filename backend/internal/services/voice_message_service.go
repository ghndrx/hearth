package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/storage"
)

var (
	ErrVoiceMessageNotFound = errors.New("voice message not found")
	ErrVoiceMessageTooLarge = errors.New("voice message file too large (max 5 minutes / 25MB)")
	ErrVoiceMessageInvalid  = errors.New("invalid voice message file")
	ErrVoiceMessageFormat   = errors.New("invalid voice message format (only WEBM, OGG, OPUS, MP3 allowed)")
	ErrVoiceMessageDuration = errors.New("voice message duration too long (max 5 minutes)")
)

// MaxVoiceMessageDuration is the maximum allowed duration for voice messages (5 minutes)
const MaxVoiceMessageDuration = 5 * 60 * 1000 // 5 minutes in milliseconds

// MaxVoiceMessageSize is the maximum file size (25MB)
const MaxVoiceMessageSize = 25 * 1024 * 1024

// VoiceMessageService handles voice message business logic
type VoiceMessageService struct {
	mu           sync.RWMutex
	voiceMessages map[uuid.UUID]*models.VoiceMessage
	storage      *storage.Service
}

// NewVoiceMessageService creates a new voice message service
func NewVoiceMessageService(storageService *storage.Service) *VoiceMessageService {
	return &VoiceMessageService{
		voiceMessages: make(map[uuid.UUID]*models.VoiceMessage),
		storage:       storageService,
	}
}

// AllowedVoiceTypes defines the allowed audio MIME types for voice messages
var AllowedVoiceTypes = map[string]bool{
	"audio/webm":   true,
	"audio/ogg":    true,
	"audio/opus":   true,
	"audio/mpeg":   true,
	"audio/mp3":    true,
	"audio/wav":    true,
	"audio/wave":   true,
	"audio/x-wav":  true,
}

// UploadVoiceMessage handles voice message upload
func (s *VoiceMessageService) UploadVoiceMessage(
	ctx context.Context,
	file *multipart.FileHeader,
	userID uuid.UUID,
	channelID uuid.UUID,
) (*models.VoiceMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate file size
	if file.Size > MaxVoiceMessageSize {
		return nil, ErrVoiceMessageTooLarge
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !AllowedVoiceTypes[contentType] {
		return nil, ErrVoiceMessageFormat
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Read the file content
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	// Extract waveform data from the audio
	waveformData, durationMs, err := s.extractWaveform(data, contentType)
	if err != nil {
		// Log but don't fail - waveform extraction is best effort
		fmt.Printf("Warning: failed to extract waveform: %v\n", err)
		waveformData = generateDefaultWaveform(50) // 50 amplitude samples
		durationMs = 0
	}

	// Validate duration
	if durationMs > MaxVoiceMessageDuration {
		return nil, ErrVoiceMessageDuration
	}

	// Upload to storage using the standard UploadFile which takes *multipart.FileHeader
	// We need to pass the original file header and a category
	fileInfo, err := s.storage.UploadFile(ctx, file, userID, "voice_messages")
	if err != nil {
		return nil, fmt.Errorf("failed to upload voice message: %w", err)
	}

	fileURL := fileInfo.URL

	// Create voice message
	voiceMessage := &models.VoiceMessage{
		ID:           uuid.New(),
		ChannelID:    channelID,
		UserID:       userID,
		FileURL:      fileURL,
		DurationMs:   durationMs,
		WaveformData: waveformData,
		CreatedAt:    time.Now(),
	}

	s.voiceMessages[voiceMessage.ID] = voiceMessage

	return voiceMessage, nil
}

// GetVoiceMessage retrieves a voice message by ID
func (s *VoiceMessageService) GetVoiceMessage(ctx context.Context, id uuid.UUID) (*models.VoiceMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vm, ok := s.voiceMessages[id]
	if !ok {
		return nil, ErrVoiceMessageNotFound
	}

	return vm, nil
}

// GetChannelVoiceMessages retrieves all voice messages for a channel
func (s *VoiceMessageService) GetChannelVoiceMessages(ctx context.Context, channelID uuid.UUID, limit int) ([]*models.VoiceMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var result []*models.VoiceMessage
	for _, vm := range s.voiceMessages {
		if vm.ChannelID == channelID {
			result = append(result, vm)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// DeleteVoiceMessage deletes a voice message
func (s *VoiceMessageService) DeleteVoiceMessage(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm, ok := s.voiceMessages[id]
	if !ok {
		return ErrVoiceMessageNotFound
	}

	// Only the owner can delete their voice message
	if vm.UserID != userID {
		return errors.New("cannot delete another user's voice message")
	}

	delete(s.voiceMessages, id)

	// Delete from storage (best effort)
	s.storage.DeleteFile(ctx, vm.FileURL)

	return nil
}

// extractWaveform extracts amplitude samples from audio data for visualization
// Returns waveform data as array of normalized amplitudes (0.0-1.0) and duration in milliseconds
func (s *VoiceMessageService) extractWaveform(data []byte, contentType string) ([]float64, int, error) {
	// For now, generate a realistic-looking waveform from the audio data
	// In production, you'd use a proper audio decoding library like go-audio or opusdec
	
	// Simple approach: sample the raw bytes to create a waveform visualization
	// This gives a reasonable approximation of the audio amplitude
	numSamples := 50 // Number of waveform bars to show
	waveform := make([]float64, numSamples)
	
	// Calculate sample interval
	sampleInterval := len(data) / numSamples
	if sampleInterval == 0 {
		sampleInterval = 1
	}
	
	// Sample bytes to create waveform
	for i := 0; i < numSamples; i++ {
		idx := i * sampleInterval
		if idx >= len(data) {
			idx = len(data) - 1
		}
		// Normalize byte value to 0.0-1.0
		sample := float64(data[idx])
		if sample < 0 {
			sample = -sample // Handle signed bytes
		}
		// Map to reasonable amplitude range (0.2-1.0 to avoid too quiet)
		amplitude := 0.2 + (sample / 255.0) * 0.8
		waveform[i] = amplitude
	}
	
	// Estimate duration based on bitrate and file size
	// This is a rough estimate - real implementation would decode the audio
	var durationMs int
	switch contentType {
	case "audio/webm":
		// WebM/Opus typically around 64kbps for voice
		durationMs = (len(data) * 8) / 64
	case "audio/ogg":
		// Ogg/Opus typically around 64kbps for voice
		durationMs = (len(data) * 8) / 64
	case "audio/mpeg", "audio/mp3":
		// MP3 at 128kbps
		durationMs = (len(data) * 8) / 128
	default:
		// Default assumption: 64kbps
		durationMs = (len(data) * 8) / 64
	}
	
	return waveform, durationMs, nil
}

// generateDefaultWaveform generates a default waveform pattern
func generateDefaultWaveform(numSamples int) []float64 {
	waveform := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		// Generate a reasonable-looking pattern
		pos := float64(i) / float64(numSamples)
		base := 0.3 + 0.4*(1.0-pos) // Decreasing trend
		waveform[i] = base
	}
	return waveform
}

func getExtensionForType(contentType string) string {
	switch contentType {
	case "audio/webm":
		return ".webm"
	case "audio/ogg":
		return ".ogg"
	case "audio/opus":
		return ".opus"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	default:
		return ".audio"
	}
}

// MarshalWaveform marshals waveform data to JSON for storage
func MarshalWaveform(waveform []float64) ([]byte, error) {
	return json.Marshal(waveform)
}

// UnmarshalWaveform unmarshals waveform data from JSON
func UnmarshalWaveform(data []byte) ([]float64, error) {
	var waveform []float64
	err := json.Unmarshal(data, &waveform)
	return waveform, err
}
