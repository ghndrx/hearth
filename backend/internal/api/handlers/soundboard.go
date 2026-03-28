package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// validAudioExtensions contains the allowed audio file extensions
var validAudioExtensions = map[string]bool{
	".mp3": true,
	".wav": true,
	".ogg": true,
	".m4a": true,
	".aac": true,
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
		return errors.New("audio URL must point to a valid audio file (.mp3, .wav, .ogg, .m4a, .aac)")
	}

	return nil
}

// SoundboardHandler handles soundboard endpoints
type SoundboardHandler struct {
	soundboardService *services.SoundboardService
	serverService     *services.ServerService
	permService       *services.PermissionService
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
	return c.JSON(sounds)
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
	return c.JSON(sounds)
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
	return c.JSON(sound)
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

	sound, err := h.soundboardService.Create(c.Context(), &serverID, req.Name, req.EmojiName, req.Volume, nil, uploaderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create sound"})
	}
	// Validate audio URL if provided
	if req.AudioURL != "" {
		if err := validateAudioURL(req.AudioURL); err != nil {
			return err
		}
	}
	return c.Status(fiber.StatusCreated).JSON(sound)
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update sound"})
	}
	return c.JSON(sound)
}

// DeleteSound deletes a soundboard sound
func (h *SoundboardHandler) DeleteSound(c *fiber.Ctx) error {
	soundID, err := uuid.Parse(c.Params("soundId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sound ID"})
	}

	if err := h.soundboardService.Delete(c.Context(), soundID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete sound"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ─── Legacy HTTP handler methods (net/http) ──────────────────────────────────

func (h *Handler) GetSounds(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("server_id")

	sounds, err := h.db.GetSounds(r.Context(), serverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, sounds, http.StatusOK)
}

func (h *Handler) CreateSound(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("server_id")
	user := getUser(r)

	var req struct {
		Name     string `json:"name"`
		AudioURL string `json:"audio_url"`
		Emoji    string `json:"emoji"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sound := &models.Sound{
		ID:        generateID(),
		ServerID:  serverID,
		Name:      req.Name,
		AudioURL:  req.AudioURL,
		Emoji:     req.Emoji,
		CreatedBy: user.ID,
		CreatedAt: time.Now(),
	}

	if err := h.db.CreateSound(r.Context(), sound); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, sound, http.StatusCreated)
}

func (h *Handler) DeleteSound(w http.ResponseWriter, r *http.Request) {
	soundID := r.PathValue("id")

	if err := h.db.DeleteSound(r.Context(), soundID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PlaySound(w http.ResponseWriter, r *http.Request) {
	soundID := r.PathValue("id")

	sound, err := h.db.GetSound(r.Context(), soundID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Broadcast to channel that this sound was triggered
	h.wsHub.BroadcastToChannel(sound.ServerID, map[string]interface{}{
		"type":      "sound_play",
		"sound_id":  sound.ID,
		"audio_url": sound.AudioURL,
		"emoji":     sound.Emoji,
	})

	w.WriteHeader(http.StatusNoContent)
}
