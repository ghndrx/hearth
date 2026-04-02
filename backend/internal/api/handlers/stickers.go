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
	premiumService *services.PremiumService
}

// NewStickerHandler creates a new sticker handler
func NewStickerHandler(
	stickerService *services.StickerService,
	serverService *services.ServerService,
	permService *services.PermissionService,
	premiumService *services.PremiumService,
) *StickerHandler {
	return &StickerHandler{
		stickerService: stickerService,
		serverService:  serverService,
		permService:    permService,
		premiumService: premiumService,
	}
}

// getUserStickerTier gets the user's sticker tier based on their premium subscription
func (h *StickerHandler) getUserStickerTier(ctx *fiber.Ctx, userID uuid.UUID) models.StickerPackTier {
	if h.premiumService != nil {
		status, err := h.premiumService.GetUserPremiumStatus(ctx.Context(), userID)
		if err == nil && status != nil {
			return models.StickerTierFromString(string(status.Tier))
		}
	}
	return models.StickerPackTierFree
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
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	userTier := h.getUserStickerTier(c, userID)

	stickers, err := h.stickerService.GetAvailable(c.Context(), nil, userTier)
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

	userTier := h.getUserStickerTier(c, userID)

	stickers, err := h.stickerService.GetAvailable(c.Context(), &serverID, userTier)
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
// @Param tier formData string false "Required tier (free, basic, premium)" default("free")
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

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
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

	// Parse tier
	tierStr := c.FormValue("tier")
	requiredTier := models.StickerTierFromString(tierStr)

	// Create sticker
	sticker, err := h.stickerService.CreateWithTier(c.Context(), &serverID, name, tags, file, userID, requiredTier)
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
		if err == services.ErrPackTierAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "user tier does not have access to create premium stickers",
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

// --- Sticker Pack Endpoints ---

// ListGlobalStickerPacks returns all global sticker packs
// @Summary List global sticker packs
// @Description Returns all global sticker packs available to the user
// @Tags Sticker Packs
// @Produce json
// @Success 200 {array} models.StickerPackResponse "List of sticker packs"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /sticker-packs [get]
func (h *StickerHandler) ListGlobalStickerPacks(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	userTier := h.getUserStickerTier(c, userID)

	packs, err := h.stickerService.GetGlobalPacks(c.Context(), userTier)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker packs",
		})
	}

	response := make([]models.StickerPackResponse, 0, len(packs))
	for _, p := range packs {
		response = append(response, p.ToPackResponse())
	}

	return c.JSON(response)
}

// GetStickerPack returns a specific sticker pack with its stickers
// @Summary Get sticker pack
// @Description Returns a specific sticker pack with all its stickers
// @Tags Sticker Packs
// @Produce json
// @Param id path string true "Pack ID"
// @Success 200 {object} models.StickerPackResponse "Sticker pack details"
// @Failure 400 {object} fiber.Map "Invalid pack ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "User tier does not have access"
// @Failure 404 {object} fiber.Map "Pack not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /sticker-packs/{id} [get]
func (h *StickerHandler) GetStickerPack(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	packID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pack id",
		})
	}

	userTier := h.getUserStickerTier(c, userID)

	pack, err := h.stickerService.GetPackWithStickers(c.Context(), packID, userTier)
	if err != nil {
		if err == services.ErrPackNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker pack not found",
			})
		}
		if err == services.ErrPackTierAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "user tier does not have access to this pack",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker pack",
		})
	}

	return c.JSON(pack.ToPackResponse())
}

// ListServerStickerPacks returns all sticker packs for a server
// @Summary List server sticker packs
// @Description Returns all sticker packs for a specific server
// @Tags Sticker Packs
// @Produce json
// @Param id path string true "Server ID"
// @Success 200 {array} models.StickerPackResponse "List of sticker packs"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs [get]
func (h *StickerHandler) ListServerStickerPacks(c *fiber.Ctx) error {
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

	userTier := h.getUserStickerTier(c, userID)

	packs, err := h.stickerService.GetPacksByServer(c.Context(), serverID, userTier)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get sticker packs",
		})
	}

	response := make([]models.StickerPackResponse, 0, len(packs))
	for _, p := range packs {
		response = append(response, p.ToPackResponse())
	}

	return c.JSON(response)
}

// CreateStickerPack creates a new sticker pack (server admins only)
// @Summary Create sticker pack
// @Description Creates a new sticker pack for a server
// @Tags Sticker Packs
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body models.CreateStickerPackRequest true "Sticker pack data"
// @Success 201 {object} models.StickerPackResponse "Sticker pack created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs [post]
func (h *StickerHandler) CreateStickerPack(c *fiber.Ctx) error {
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

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	var req models.CreateStickerPackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "pack name is required",
		})
	}

	if len(req.Name) < 2 || len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "pack name must be 2-100 characters",
		})
	}

	tier := models.StickerTierFromString(req.Tier)
	isGlobal := false
	if req.IsGlobal != nil {
		isGlobal = *req.IsGlobal
	}

	pack, err := h.stickerService.CreatePack(
		c.Context(),
		req.Name,
		req.Description,
		req.IconURL,
		tier,
		isGlobal,
		&serverID,
		userID,
	)
	if err != nil {
		if err == services.ErrPackNameRequired {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "pack name is required",
			})
		}
		if err == services.ErrPackNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "pack name must be 2-100 characters",
			})
		}
		if err == services.ErrPackTierAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "user tier does not have access to create premium packs",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create sticker pack",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(pack.ToPackResponse())
}

// UpdateStickerPack updates a sticker pack
// @Summary Update sticker pack
// @Description Updates a sticker pack's details
// @Tags Sticker Packs
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param packId path string true "Pack ID"
// @Param body body models.UpdateStickerPackRequest true "Update data"
// @Success 200 {object} models.StickerPackResponse "Updated pack"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Pack not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs/{packId} [patch]
func (h *StickerHandler) UpdateStickerPack(c *fiber.Ctx) error {
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

	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pack id",
		})
	}

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	var req models.UpdateStickerPackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	pack, err := h.stickerService.UpdatePack(c.Context(), packID, req.Name, req.Description, req.IconURL, req.IsActive)
	if err != nil {
		if err == services.ErrPackNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker pack not found",
			})
		}
		if err == services.ErrPackNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "pack name must be 2-100 characters",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update sticker pack",
		})
	}

	return c.JSON(pack.ToPackResponse())
}

// DeleteStickerPack deletes a sticker pack
// @Summary Delete sticker pack
// @Description Deletes a sticker pack and removes all sticker associations
// @Tags Sticker Packs
// @Param id path string true "Server ID"
// @Param packId path string true "Pack ID"
// @Success 204 "Pack deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Pack not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs/{packId} [delete]
func (h *StickerHandler) DeleteStickerPack(c *fiber.Ctx) error {
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

	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pack id",
		})
	}

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	err = h.stickerService.DeletePack(c.Context(), packID)
	if err != nil {
		if err == services.ErrPackNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker pack not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete sticker pack",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddStickerToPack adds a sticker to a pack
// @Summary Add sticker to pack
// @Description Adds an existing sticker to a sticker pack
// @Tags Sticker Packs
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param packId path string true "Pack ID"
// @Param body body models.AddStickerToPackRequest true "Sticker to add"
// @Success 200 {object} fiber.Map "Sticker added to pack"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Pack or sticker not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs/{packId}/stickers [post]
func (h *StickerHandler) AddStickerToPack(c *fiber.Ctx) error {
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

	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pack id",
		})
	}

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	var req models.AddStickerToPackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.StickerID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sticker_id is required",
		})
	}

	err = h.stickerService.AddStickerToPack(c.Context(), packID, req.StickerID, req.Position, req.IsDefault)
	if err != nil {
		if err == services.ErrPackNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker pack not found",
			})
		}
		if err == services.ErrStickerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to add sticker to pack",
		})
	}

	return c.JSON(fiber.Map{"message": "sticker added to pack"})
}

// RemoveStickerFromPack removes a sticker from a pack
// @Summary Remove sticker from pack
// @Description Removes a sticker from a sticker pack
// @Tags Sticker Packs
// @Param id path string true "Server ID"
// @Param packId path string true "Pack ID"
// @Param stickerId path string true "Sticker ID"
// @Success 204 "Sticker removed from pack"
// @Failure 400 {object} fiber.Map "Invalid ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_STICKERS permission"
// @Failure 404 {object} fiber.Map "Pack or sticker not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/sticker-packs/{packId}/stickers/{stickerId} [delete]
func (h *StickerHandler) RemoveStickerFromPack(c *fiber.Ctx) error {
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

	packID, err := uuid.Parse(c.Params("packId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid pack id",
		})
	}

	stickerID, err := uuid.Parse(c.Params("stickerId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sticker id",
		})
	}

	// Check MANAGE_STICKERS permission
	if h.permService != nil {
		if err := h.permService.RequirePermission(c.Context(), serverID, userID, models.PermManageStickers); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_STICKERS permission",
			})
		}
	}

	err = h.stickerService.RemoveStickerFromPack(c.Context(), packID, stickerID)
	if err != nil {
		if err == services.ErrPackNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "sticker pack not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to remove sticker from pack",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
