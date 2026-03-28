package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// StickerHandler handles sticker endpoints
type StickerHandler struct {
	stickerService *services.StickerService
	serverService  *services.ServerService
	permService    *services.PermissionService
}

// NewStickerHandler creates a new sticker handler
func NewStickerHandler(
	stickerService *services.StickerService,
	serverService *services.ServerService,
	permService *services.PermissionService,
) *StickerHandler {
	return &StickerHandler{
		stickerService: stickerService,
		serverService:  serverService,
		permService:    permService,
	}
}

// ListGlobalStickers returns all global stickers
// @Summary List global stickers
// @Description Returns all global stickers available to all users
// @Tags Stickers
// @Produce json
// @Success 200 {array} models.StickerResponse "List of global stickers"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /stickers [get]
func (h *StickerHandler) ListGlobalStickers(c *fiber.Ctx) error {
	stickers, err := h.stickerService.GetGlobal(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stickers",
		})
	}

	response := make([]models.StickerResponse, 0, len(stickers))
	for _, s := range stickers {
		response = append(response, s.ToResponse())
	}

	return c.JSON(response)
}

// ListServerStickers returns all stickers for a server
// @Summary List server stickers
// @Description Returns all custom stickers for a specific server
// @Tags Stickers
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.StickerResponse "List of server stickers"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/stickers [get]
func (h *StickerHandler) ListServerStickers(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	stickers, err := h.stickerService.GetByServer(c.Context(), serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stickers",
		})
	}

	response := make([]models.StickerResponse, 0, len(stickers))
	for _, s := range stickers {
		response = append(response, s.ToResponse())
	}

	return c.JSON(response)
}

// GetSticker returns a specific sticker
// @Summary Get sticker by ID
// @Description Returns a specific sticker by its ID
// @Tags Stickers
// @Produce json
// @Param id path string true "Server ID"
// @Param stickerId path string true "Sticker ID"
// @Success 200 {object} models.StickerResponse "Sticker details"
// @Failure 400 {object} fiber.Map "Invalid server or sticker ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Sticker not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/stickers/{stickerId} [get]
func (h *StickerHandler) GetSticker(c *fiber.Ctx) error {
	stickerID, err := uuid.Parse(c.Params("stickerId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sticker id",
		})
	}

	sticker, err := h.stickerService.Get(c.Context(), stickerID)
	if err != nil {
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker",
		})
	}

	return c.JSON(sticker.ToResponse())
}

// CreateSticker uploads a new custom sticker to a server
// @Summary Create sticker
// @Description Uploads a new custom sticker image to a server
// @Tags Stickers
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Server ID"
// @Param name formData string true "Sticker name (2-30 chars)"
// @Param tags formData string false "Comma-separated tags (max 10)"
// @Param image formData file true "Sticker image file (PNG, APNG, GIF, max 512KB, max 100x100px)"
// @Success 201 {object} models.StickerResponse "Sticker created successfully"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/stickers [post]
func (h *StickerHandler) CreateSticker(c *fiber.Ctx) error {
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

	// Get sticker name
	name := c.FormValue("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sticker name is required",
		})
	}

	// Validate name length
	if len(name) < 2 || len(name) > 30 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sticker name must be 2-30 characters",
		})
	}

	// Validate name characters (alphanumeric, spaces, underscores)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '_') {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker name can only contain letters, numbers, spaces, and underscores",
			})
		}
	}

	// Parse tags
	var tags []string
	tagsStr := c.FormValue("tags")
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		// Clean up tags
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		// Limit to 10 tags
		if len(tags) > 10 {
			tags = tags[:10]
		}
	}

	// Get image file
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "image file is required",
		})
	}

	// Validate file size (512KB max)
	if file.Size > 512*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sticker image must be under 512KB",
		})
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/png":  true,
		"image/apng": true,
		"image/gif":  true,
	}
	if !allowedTypes[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid image type. Use PNG, APNG, or GIF",
		})
	}

	// Create sticker
	sticker, err := h.stickerService.Create(c.Context(), &serverID, name, tags, file, userID)
	if err != nil {
		if err == services.ErrStickerNameRequired {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker name is required",
			})
		}
		if err == services.ErrStickerNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker name must be 2-30 characters",
			})
		}
		if err == services.ErrStickerTooLarge {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker image must be under 512KB",
			})
		}
		if err == services.ErrStickerFormat {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid image type. Use PNG, APNG, or GIF",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create sticker",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(sticker.ToResponse())
}

// ModifySticker modifies a sticker's name or tags
// @Summary Modify sticker
// @Description Updates the name or tags of a sticker
// @Tags Stickers
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param stickerId path string true "Sticker ID"
// @Param body body models.UpdateStickerRequest true "Sticker update data"
// @Success 200 {object} models.StickerResponse "Updated sticker"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Sticker not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/stickers/{stickerId} [patch]
func (h *StickerHandler) ModifySticker(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	stickerID, err := uuid.Parse(c.Params("stickerId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sticker id",
		})
	}

	var req models.UpdateStickerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate name length if provided
	if req.Name != "" {
		if len(req.Name) < 2 || len(req.Name) > 30 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker name must be 2-30 characters",
			})
		}
	}

	// Get sticker to check ownership
	sticker, err := h.stickerService.Get(c.Context(), stickerID)
	if err != nil {
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker",
		})
	}

	// Check MANAGE_STICKERS permission on the server
	if h.permService != nil && sticker.ServerID != nil {
		if err := h.permService.RequirePermission(c.Context(), *sticker.ServerID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	updated, err := h.stickerService.Update(c.Context(), stickerID, req.Name, req.Tags)
	if err != nil {
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		if err == services.ErrStickerNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "sticker name must be 2-30 characters",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update sticker",
		})
	}

	return c.JSON(updated.ToResponse())
}

// DeleteSticker deletes a sticker
// @Summary Delete sticker
// @Description Deletes a custom sticker from a server
// @Tags Stickers
// @Param id path string true "Server ID"
// @Param stickerId path string true "Sticker ID"
// @Success 204 "Sticker deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid sticker ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Sticker not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/stickers/{stickerId} [delete]
func (h *StickerHandler) DeleteSticker(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	stickerID, err := uuid.Parse(c.Params("stickerId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sticker id",
		})
	}

	// Get sticker to check ownership
	sticker, err := h.stickerService.Get(c.Context(), stickerID)
	if err != nil {
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker",
		})
	}

	// Check MANAGE_STICKERS permission on the server
	if sticker.CreatedBy != userID {
		// Check if user has MANAGE_STICKERS permission
		if h.permService != nil && sticker.ServerID != nil {
			if err := h.permService.RequirePermission(c.Context(), *sticker.ServerID, userID, models.PermManageStickers); err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "missing MANAGE_STICKERS permission",
				})
			}
		}
	}

	err = h.stickerService.Delete(c.Context(), stickerID)
	if err != nil {
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete sticker",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
