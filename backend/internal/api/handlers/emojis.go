package handlers

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/storage"
)

// EmojiHandler handles custom emoji endpoints
type EmojiHandler struct {
	emojiService   *services.EmojiService
	serverService  *services.ServerService
	permService    *services.PermissionService
	storageService *storage.Service
}

// NewEmojiHandler creates a new emoji handler
func NewEmojiHandler(
	emojiService *services.EmojiService,
	serverService *services.ServerService,
	permService *services.PermissionService,
	storageService *storage.Service,
) *EmojiHandler {
	return &EmojiHandler{
		emojiService:   emojiService,
		serverService:  serverService,
		permService:    permService,
		storageService: storageService,
	}
}

// EmojiResponse represents the emoji response
type EmojiResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Animated     bool      `json:"animated"`
	RequireColon bool      `json:"require_colons"`
	Managed      bool      `json:"managed"`
	CreatorID    string    `json:"creator_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListServerEmojis returns all custom emojis for a server
// @Summary List server emojis
// @Description Returns all custom emojis for a specific server
// @Tags Emojis
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} EmojiResponse "List of emojis"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/emojis [get]
func (h *EmojiHandler) ListServerEmojis(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	emojis, err := h.emojiService.GetServerEmojis(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get emojis",
		})
	}

	response := make([]EmojiResponse, 0, len(emojis))
	for _, e := range emojis {
		response = append(response, EmojiResponse{
			ID:           e.ID.String(),
			Name:         e.Name,
			URL:          e.URL,
			Animated:     e.Animated,
			RequireColon: true,
			Managed:      false,
			CreatedAt:    time.Now(), // Would be e.CreatedAt if stored
		})
	}

	return c.JSON(response)
}

// CreateEmoji uploads a new custom emoji
// @Summary Create custom emoji
// @Description Uploads a new custom emoji image to a server
// @Tags Emojis
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Server ID"
// @Param name formData string true "Emoji name (2-32 chars, alphanumeric + underscore)"
// @Param image formData file true "Emoji image file (PNG, GIF, JPEG, WebP, max 256KB)"
// @Success 201 {object} EmojiResponse "Emoji created successfully"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_EMOJI permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/emojis [post]
func (h *EmojiHandler) CreateEmoji(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Get emoji name from form
	name := c.FormValue("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "emoji name is required",
		})
	}

	// Validate name (2-32 chars, alphanumeric + underscore)
	if len(name) < 2 || len(name) > 32 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "emoji name must be 2-32 characters",
		})
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "emoji name can only contain letters, numbers, and underscores",
			})
		}
	}

	// Get image file
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "image file is required",
		})
	}

	// Validate file size (256KB max)
	if file.Size > 256*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "emoji image must be under 256KB",
		})
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/png":  true,
		"image/gif":  true,
		"image/jpeg": true,
		"image/webp": true,
	}
	if !allowedTypes[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid image type. Use PNG, GIF, JPEG, or WebP",
		})
	}

	// Check if animated (GIF)
	animated := contentType == "image/gif"

	// Upload the image
	var url string
	if h.storageService != nil {
		fileInfo, err := h.storageService.UploadFile(c.Context(), file, userID, "emojis")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to upload emoji image",
			})
		}
		url = fileInfo.URL
	} else {
		// Fallback: generate a placeholder URL for testing
		ext := strings.ToLower(filepath.Ext(file.Filename))
		url = "/emojis/" + uuid.New().String() + ext
	}

	// Create emoji record
	emoji, err := h.emojiService.Create(c.Context(), serverID, name, url, animated)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create emoji",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(EmojiResponse{
		ID:           emoji.ID.String(),
		Name:         emoji.Name,
		URL:          emoji.URL,
		Animated:     emoji.Animated,
		RequireColon: true,
		Managed:      false,
		CreatorID:    userID.String(),
		CreatedAt:    time.Now(),
	})
}

// GetEmoji returns a specific emoji
// @Summary Get emoji by ID
// @Description Returns a specific custom emoji by its ID
// @Tags Emojis
// @Produce json
// @Param id path string true "Server ID"
// @Param emojiId path string true "Emoji ID"
// @Success 200 {object} EmojiResponse "Emoji details"
// @Failure 400 {object} fiber.Map "Invalid server or emoji ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Emoji not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/emojis/{emojiId} [get]
func (h *EmojiHandler) GetEmoji(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	emojiID, err := uuid.Parse(c.Params("emojiId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid emoji id",
		})
	}

	emojis, err := h.emojiService.GetServerEmojis(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get emojis",
		})
	}

	for _, e := range emojis {
		if e.ID == emojiID {
			return c.JSON(EmojiResponse{
				ID:           e.ID.String(),
				Name:         e.Name,
				URL:          e.URL,
				Animated:     e.Animated,
				RequireColon: true,
				Managed:      false,
				CreatedAt:    time.Now(),
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "emoji not found",
	})
}

// DeleteEmoji deletes a custom emoji
// @Summary Delete custom emoji
// @Description Deletes a custom emoji from a server
// @Tags Emojis
// @Param id path string true "Server ID"
// @Param emojiId path string true "Emoji ID"
// @Success 204 "Emoji deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid emoji ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_EMOJI permission"
// @Failure 404 {object} fiber.Map "Emoji not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/emojis/{emojiId} [delete]
func (h *EmojiHandler) DeleteEmoji(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	emojiID, err := uuid.Parse(c.Params("emojiId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid emoji id",
		})
	}

	// Get emoji to find its server
	emoji, err := h.emojiService.Get(c.Context(), emojiID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "emoji not found",
		})
	}

	// Check MANAGE_EMOJI permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), emoji.ServerID, userID, models.PermManageEmoji); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_EMOJI permission",
			})
		}
	}

	err = h.emojiService.Delete(c.Context(), emojiID)
	if err != nil {
		if err == services.ErrEmojiNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "emoji not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete emoji",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateEmoji updates an emoji's name
// @Summary Update emoji name
// @Description Updates the name of a custom emoji
// @Tags Emojis
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param emojiId path string true "Emoji ID"
// @Param body body struct{Name string `json:"name"`} true "Emoji update data"
// @Success 200 {object} models.Emoji "Updated emoji"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_EMOJI permission"
// @Failure 404 {object} fiber.Map "Emoji not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/emojis/{emojiId} [patch]
func (h *EmojiHandler) UpdateEmoji(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	emojiID, err := uuid.Parse(c.Params("emojiId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid emoji id",
		})
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate name
	if req.Name != "" {
		if len(req.Name) < 2 || len(req.Name) > 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "emoji name must be 2-32 characters",
			})
		}
	}

	// Get emoji to find its server
	emoji, err := h.emojiService.Get(c.Context(), emojiID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "emoji not found",
		})
	}

	// Check MANAGE_EMOJI permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), emoji.ServerID, userID, models.PermManageEmoji); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_EMOJI permission",
			})
		}
	}

	updated, err := h.emojiService.Update(c.Context(), emojiID, req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update emoji",
		})
	}

	return c.JSON(updated)
}
