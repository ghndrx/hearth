package handlers

import (
	"errors"
	"log"
	"net/url"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/websocket"
)

// validAudioExtensions contains the allowed audio file extensions
var validAudioExtensions = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".m4a":  true,
	".aac":  true,
	".opus": true,
}

// validateAudioURL validates that the given URL is a valid audio file URL
func validateAudioURL(audioURL string) error {
	if audioURL == "" {
		return errors.New("audio URL cannot be empty")
	}

	parsed, err := url.Parse(audioURL)
	if err != nil {
		return errors.New("invalid URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("audio URL must use http or https scheme")
	}

	ext := filepath.Ext(parsed.Path)
	if !validAudioExtensions[ext] {
		return errors.New("audio URL must point to a valid audio file (.mp3, .wav, .ogg, .m4a, .aac, .opus)")
	}

	return nil
}

// SoundboardHandler handles soundboard endpoints
type SoundboardHandler struct {
	soundboardService *services.SoundboardService
	serverService     *services.ServerService
	permService       *services.PermissionService
	gateway           *websocket.Gateway
}

// SetGateway sets the WebSocket gateway for broadcasting events
func (h *SoundboardHandler) SetGateway(gateway *websocket.Gateway) {
	h.gateway = gateway
}

// NewSoundboardHandler creates a new soundboard handler
func NewSoundboardHandler(soundboardService *services.SoundboardService, serverService *services.ServerService, permService *services.PermissionService) *SoundboardHandler {
	return &SoundboardHandler{
		soundboardService: soundboardService,
		serverService:     serverService,
		permService:       permService,
	}
}

// ListDefaultSounds returns all default/global soundboard sounds
func (h *SoundboardHandler) ListDefaultSounds(c *fiber.Ctx) error {
	sounds, err := h.soundboardService.GetDefault(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get default sounds"})
	}
	responses := make([]interface{}, 0, len(sounds))
	for _, sound := range sounds {
		responses = append(responses, sound.ToResponse())
	}
	return c.JSON(responses)
}

// ListServerSounds returns all soundboard sounds for a server
func (h *SoundboardHandler) ListServerSounds(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
	}

	sounds, err := h.soundboardService.GetByServer(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get server sounds"})
	}
	responses := make([]interface{}, 0, len(sounds))
	for _, sound := range sounds {
		responses = append(responses, sound.ToResponse())
	}
	return c.JSON(responses)
}

// GetSound returns a single soundboard sound by ID
func (h *SoundboardHandler) GetSound(c *fiber.Ctx) error {
	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	sound, err := h.soundboardService.Get(c.Context(), soundID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sound not found"})
	}
	return c.JSON(sound.ToResponse())
}

// CreateSound creates a new soundboard sound for a server
func (h *SoundboardHandler) CreateSound(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
	}

	var req struct {
		Name      string  `json:"name"`
		EmojiName string  `json:"emoji_name"`
		Volume    float64 `json:"volume"`
		AudioURL  string  `json:"audio_url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Get uploader ID from authenticated user context
	uploaderIDRaw := c.Locals("userID")
	if uploaderIDRaw == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	uploaderID, ok := uploaderIDRaw.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user authentication"})
	}

	// Handle file upload or URL-based creation
	var sound *models.SoundboardSound
	if audioFile := c.FormValue("audio"); audioFile != "" {
		// File upload path
		fileHeader, err := c.FormFile("audio")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "audio file is required"})
		}
		name := req.Name
		if name == "" {
			name = c.FormValue("name")
		}
		emojiName := req.EmojiName
		if emojiName == "" {
			emojiName = c.FormValue("emoji_name")
		}
		volume := req.Volume
		if volume <= 0 {
			volume = 1.0
		}
		sound, err = h.soundboardService.Create(c.Context(), &serverID, name, emojiName, volume, fileHeader, uploaderID)
		if err != nil {
			if errors.Is(err, services.ErrSoundboardNameRequired) ||
				errors.Is(err, services.ErrSoundboardNameTooLong) ||
				errors.Is(err, services.ErrSoundboardFormat) ||
				errors.Is(err, services.ErrSoundboardTooLarge) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create sound"})
		}
	} else {
		// URL-based creation
		if req.AudioURL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "audio file or audio_url is required"})
		}
		if err := validateAudioURL(req.AudioURL); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		volume := req.Volume
		if volume <= 0 {
			volume = 1.0
		}
		sound, err = h.soundboardService.Create(c.Context(), &serverID, req.Name, req.EmojiName, volume, nil, uploaderID)
		if err != nil {
			if errors.Is(err, services.ErrSoundboardNameRequired) ||
				errors.Is(err, services.ErrSoundboardNameTooLong) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create sound"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(sound.ToResponse())
}

// ModifySound updates a soundboard sound's properties
func (h *SoundboardHandler) ModifySound(c *fiber.Ctx) error {
	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	var req struct {
		Name      string   `json:"name"`
		EmojiName string   `json:"emoji_name"`
		Volume    *float64 `json:"volume"`
		Available *bool    `json:"available"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	sound, err := h.soundboardService.Update(c.Context(), soundID, req.Name, req.EmojiName, req.Volume, req.Available)
	if err != nil {
		if errors.Is(err, services.ErrSoundboardSoundNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sound not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update sound"})
	}
	return c.JSON(sound.ToResponse())
}

// DeleteSound deletes a soundboard sound
func (h *SoundboardHandler) DeleteSound(c *fiber.Ctx) error {
	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	if err := h.soundboardService.Delete(c.Context(), soundID); err != nil {
		if errors.Is(err, services.ErrSoundboardSoundNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sound not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete sound"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// PlaySound plays a soundboard sound in a voice channel
func (h *SoundboardHandler) PlaySound(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
	}

	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	var req struct {
		ChannelID string  `json:"channel_id"`
		Volume    float64 `json:"volume"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.ChannelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "channel_id is required"})
	}

	channelID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel_id"})
	}

	volume := req.Volume
	if volume <= 0 {
		volume = 1.0
	}

	playEvent, err := h.soundboardService.PlaySoundInVoice(c.Context(), soundID, channelID, serverID, volume)
	if err != nil {
		if errors.Is(err, services.ErrSoundboardSoundNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sound not found"})
		}
		if errors.Is(err, services.ErrSoundboardSoundInvalid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sound not available"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to play sound"})
	}

	// Add user info to the event
	userIDRaw := c.Locals("userID")
	if userID, ok := userIDRaw.(uuid.UUID); ok {
		playEvent.UserID = userID.String()
	}

	// Broadcast to all users in the server via WebSocket
	if h.gateway != nil {
		hub := h.gateway.Hub()
		if hub != nil {
			hub.SendToServer(serverID, &websocket.Event{
				Op:   websocket.OpDispatch,
				Type: websocket.EventSoundboardPlay,
				Data: playEvent,
			})
			log.Printf("[Soundboard] Broadcast play via REST: sound=%s, server=%s", playEvent.SoundName, serverID)
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// --- Pack endpoints ---

// ListServerPacks returns all soundboard packs for a server
func (h *SoundboardHandler) ListServerPacks(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
	}

	packs, err := h.soundboardService.GetPacksByServer(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get server packs"})
	}

	responses := make([]interface{}, 0, len(packs))
	for _, pack := range packs {
		responses = append(responses, pack.ToResponse())
	}
	return c.JSON(responses)
}

// GetPack returns a single soundboard pack by ID
func (h *SoundboardHandler) GetPack(c *fiber.Ctx) error {
	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pack ID"})
	}

	pack, err := h.soundboardService.GetPack(c.Context(), packID)
	if err != nil {
		if errors.Is(err, services.ErrSoundboardPackNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pack not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get pack"})
	}
	return c.JSON(pack.ToResponse())
}

// CreatePack creates a new soundboard pack for a server
func (h *SoundboardHandler) CreatePack(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server ID"})
	}

	var req struct {
		Name      string `json:"name"`
		EmojiName string `json:"emoji_name"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	pack, err := h.soundboardService.CreatePack(c.Context(), &serverID, req.Name, req.EmojiName, req.IsDefault)
	if err != nil {
		if errors.Is(err, services.ErrSoundboardPackNameReq) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "pack name is required"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create pack"})
	}
	return c.Status(fiber.StatusCreated).JSON(pack.ToResponse())
}

// ModifyPack updates a soundboard pack's properties
func (h *SoundboardHandler) ModifyPack(c *fiber.Ctx) error {
	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pack ID"})
	}

	var req struct {
		Name      string `json:"name"`
		EmojiName string `json:"emoji_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	pack, err := h.soundboardService.UpdatePack(c.Context(), packID, req.Name, req.EmojiName)
	if err != nil {
		if errors.Is(err, services.ErrSoundboardPackNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pack not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update pack"})
	}
	return c.JSON(pack.ToResponse())
}

// DeletePack deletes a soundboard pack
func (h *SoundboardHandler) DeletePack(c *fiber.Ctx) error {
	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pack ID"})
	}

	if err := h.soundboardService.DeletePack(c.Context(), packID); err != nil {
		if errors.Is(err, services.ErrSoundboardPackNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pack not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete pack"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// AddSoundToPack adds a sound to a pack
func (h *SoundboardHandler) AddSoundToPack(c *fiber.Ctx) error {
	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pack ID"})
	}

	var req struct {
		SoundID   string `json:"sound_id"`
		Position  int    `json:"position"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	soundID, err := uuid.Parse(req.SoundID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	if err := h.soundboardService.AddSoundToPack(c.Context(), packID, soundID, req.Position, req.IsDefault); err != nil {
		if errors.Is(err, services.ErrSoundboardPackNotFound) ||
			errors.Is(err, services.ErrSoundboardSoundNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add sound to pack"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveSoundFromPack removes a sound from a pack
func (h *SoundboardHandler) RemoveSoundFromPack(c *fiber.Ctx) error {
	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pack ID"})
	}

	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	if err := h.soundboardService.RemoveSoundFromPack(c.Context(), packID, soundID); err != nil {
		if errors.Is(err, services.ErrSoundboardPackNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pack not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to remove sound from pack"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
