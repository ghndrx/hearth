package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/models"
	"hearth/internal/services"
)

// ContentSafetyHandler handles content safety API endpoints
type ContentSafetyHandler struct {
	contentSafetyService *services.ContentSafetyService
	serverService        *services.ServerService
}

// NewContentSafetyHandler creates a new content safety handler
func NewContentSafetyHandler(contentSafetyService *services.ContentSafetyService, serverService *services.ServerService) *ContentSafetyHandler {
	return &ContentSafetyHandler{
		contentSafetyService: contentSafetyService,
		serverService:        serverService,
	}
}

// getContentSafetyUserID safely extracts userID from Fiber context
func getContentSafetyUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDVal := c.Locals("userID")
	if userIDVal == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user id")
	}
	return userID, nil
}

// ListContentFilters lists all content filters for a server
// GET /api/v1/servers/:id/content-filters
func (h *ContentSafetyHandler) ListContentFilters(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	filters, err := h.contentSafetyService.GetServerContentFilters(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(filters)
}

// CreateContentFilter creates a new content filter
// POST /api/v1/servers/:id/content-filters
func (h *ContentSafetyHandler) CreateContentFilter(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	var req models.CreateContentFilterRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" {
		return ValidationError(c, "name", "is required")
	}
	if req.Type == 0 {
		return ValidationError(c, "type", "is required")
	}
	if req.Action == 0 {
		return ValidationError(c, "action", "is required")
	}

	filter, err := h.contentSafetyService.CreateContentFilter(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(filter)
}

// GetContentFilter gets a specific content filter
// GET /api/v1/content-filters/:id
func (h *ContentSafetyHandler) GetContentFilter(c *fiber.Ctx) error {
	filterID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid filter id")
	}

	filter, err := h.contentSafetyService.GetContentFilter(c.Context(), filterID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if filter == nil {
		return fiber.NewError(fiber.StatusNotFound, "filter not found")
	}

	return c.JSON(filter)
}

// UpdateContentFilter updates an existing content filter
// PATCH /api/v1/content-filters/:id
func (h *ContentSafetyHandler) UpdateContentFilter(c *fiber.Ctx) error {
	filterID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid filter id")
	}

	_, err = getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Get existing filter to check ownership
	existing, err := h.contentSafetyService.GetContentFilter(c.Context(), filterID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if existing == nil {
		return fiber.NewError(fiber.StatusNotFound, "filter not found")
	}

	var req models.UpdateContentFilterRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	filter, err := h.contentSafetyService.UpdateContentFilter(c.Context(), filterID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(filter)
}

// DeleteContentFilter deletes a content filter
// DELETE /api/v1/content-filters/:id
func (h *ContentSafetyHandler) DeleteContentFilter(c *fiber.Ctx) error {
	filterID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid filter id")
	}

	if err := h.contentSafetyService.DeleteContentFilter(c.Context(), filterID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetAgeVerification gets age verification settings for a server
// GET /api/v1/servers/:id/age-verification
func (h *ContentSafetyHandler) GetAgeVerification(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	settings, err := h.contentSafetyService.GetAgeVerification(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	if settings == nil {
		return c.JSON(fiber.Map{
			"server_id": serverID,
			"enabled":   false,
		})
	}

	return c.JSON(settings)
}

// CreateAgeVerification creates age verification settings
// PUT /api/v1/servers/:id/age-verification
func (h *ContentSafetyHandler) CreateAgeVerification(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	var req models.CreateAgeVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.RequiredAge < 13 || req.RequiredAge > 100 {
		return ValidationError(c, "required_age", "must be between 13 and 100")
	}
	if req.VerificationType == "" {
		return ValidationError(c, "verification_type", "is required")
	}

	settings, err := h.contentSafetyService.CreateAgeVerification(c.Context(), serverID, userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(settings)
}

// UpdateAgeVerification updates age verification settings
// PATCH /api/v1/servers/:id/age-verification
func (h *ContentSafetyHandler) UpdateAgeVerification(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	_, err = getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.UpdateAgeVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.RequiredAge != nil && (*req.RequiredAge < 13 || *req.RequiredAge > 100) {
		return ValidationError(c, "required_age", "must be between 13 and 100")
	}

	settings, err := h.contentSafetyService.UpdateAgeVerification(c.Context(), serverID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(settings)
}

// DeleteAgeVerification deletes age verification settings
// DELETE /api/v1/servers/:id/age-verification
func (h *ContentSafetyHandler) DeleteAgeVerification(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	_, err = getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	settings, err := h.contentSafetyService.GetAgeVerification(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}
	if settings == nil {
		return fiber.NewError(fiber.StatusNotFound, "age verification not found")
	}

	if err := h.contentSafetyService.DeleteAgeVerification(c.Context(), settings.ID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetUserContentPreferences gets user content preferences
// GET /api/v1/users/@me/content-preferences
func (h *ContentSafetyHandler) GetUserContentPreferences(c *fiber.Ctx) error {
	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	prefs, err := h.contentSafetyService.GetUserContentPreference(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(prefs)
}

// UpdateUserContentPreferences updates user content preferences
// PUT /api/v1/users/@me/content-preferences
func (h *ContentSafetyHandler) UpdateUserContentPreferences(c *fiber.Ctx) error {
	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.UpdateUserContentPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	prefs, err := h.contentSafetyService.UpdateUserContentPreference(c.Context(), userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(prefs)
}

// GetServerSafetySettings gets comprehensive safety settings for a server
// GET /api/v1/servers/:id/content-safety
func (h *ContentSafetyHandler) GetServerSafetySettings(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	settings, err := h.contentSafetyService.GetServerSafetySettings(c.Context(), serverID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(settings)
}

// TestContentScan tests content against filters (for preview/admin)
// POST /api/v1/servers/:id/content-safety/test
func (h *ContentSafetyHandler) TestContentScan(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	userID, err := getContentSafetyUserID(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	// Check if user is a member of the server
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return fiber.NewError(fiber.StatusForbidden, "not a member of this server")
	}

	var req struct {
		Content   string `json:"content" validate:"required"`
		ChannelID string `json:"channel_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Content == "" {
		return ValidationError(c, "content", "is required")
	}

	// Default to server ID for channel
	channelID := serverID
	if req.ChannelID != "" {
		parsed, err := uuid.Parse(req.ChannelID)
		if err != nil {
			return ValidationError(c, "channel_id", "invalid format")
		}
		channelID = parsed
	}

	result, err := h.contentSafetyService.ScanContent(c.Context(), serverID, channelID, userID, nil, req.Content)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(result)
}
