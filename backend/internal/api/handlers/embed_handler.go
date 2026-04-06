package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// EmbedServiceInterface defines methods needed from EmbedService
type EmbedServiceInterface interface {
	FetchURLMetadata(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error)
	CreateTemplate(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error)
	GetTemplates(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error)
	GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error)
	UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error)
	DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error
}

// EmbedHandler handles embed-related HTTP requests
type EmbedHandler struct {
	embedService EmbedServiceInterface
}

// NewEmbedHandler creates a new embed handler
func NewEmbedHandler(embedService EmbedServiceInterface) *EmbedHandler {
	return &EmbedHandler{
		embedService: embedService,
	}
}

// FetchURLMetadata fetches OpenGraph metadata for a URL
// @Summary Fetch URL metadata
// @Description Fetches OpenGraph/metadata for a URL to create an embed preview
// @Tags Embeds
// @Produce json
// @Param url query string true "URL to fetch metadata for"
// @Success 200 {object} models.LinkPreviewResponse "Metadata fetched"
// @Failure 400 {object} fiber.Map "Invalid URL"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/fetch [get]
func (h *EmbedHandler) FetchURLMetadata(c *fiber.Ctx) error {
	urlStr := c.Query("url")
	if urlStr == "" {
		return ValidationError(c, "url", "url is required")
	}

	preview, err := h.embedService.FetchURLMetadata(c.Context(), urlStr)
	if err != nil {
		if err == services.ErrInvalidURL {
			return ValidationError(c, "url", "invalid URL format")
		}
		if err == services.ErrUnreachableURL {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "URL is unreachable"})
		}
		return HandleServiceError(c, err)
	}

	return c.JSON(preview)
}

// CreateTemplate creates a new embed template
// @Summary Create an embed template
// @Description Creates a new embed template that can be reused
// @Tags Embeds
// @Accept json
// @Produce json
// @Param body body models.CreateEmbedTemplateRequest true "Template data"
// @Success 201 {object} models.EmbedTemplateResponse "Template created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/templates [post]
func (h *EmbedHandler) CreateTemplate(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.CreateEmbedTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" {
		return ValidationError(c, "name", "name is required")
	}

	template, err := h.embedService.CreateTemplate(c.Context(), userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(template.ToTemplateResponse())
}

// ListTemplates lists all embed templates for the current user
// @Summary List embed templates
// @Description Lists all embed templates owned by the current user
// @Tags Embeds
// @Produce json
// @Success 200 {array} models.EmbedTemplateResponse "Templates found"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/templates [get]
func (h *EmbedHandler) ListTemplates(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	templates, err := h.embedService.GetTemplates(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	response := make([]*models.EmbedTemplateResponse, len(templates))
	for i := range templates {
		response[i] = templates[i].ToTemplateResponse()
	}

	return c.JSON(response)
}

// GetTemplate retrieves a specific embed template
// @Summary Get an embed template
// @Description Retrieves a specific embed template by ID
// @Tags Embeds
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} models.EmbedTemplateResponse "Template found"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/templates/{id} [get]
func (h *EmbedHandler) GetTemplate(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ValidationError(c, "id", "invalid UUID format")
	}

	template, err := h.embedService.GetTemplate(c.Context(), userID, id)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}
		return HandleServiceError(c, err)
	}

	return c.JSON(template.ToTemplateResponse())
}

// UpdateTemplate updates an embed template
// @Summary Update an embed template
// @Description Updates an existing embed template
// @Tags Embeds
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param body body models.UpdateEmbedTemplateRequest true "Template update data"
// @Success 200 {object} models.EmbedTemplateResponse "Template updated"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/templates/{id} [put]
func (h *EmbedHandler) UpdateTemplate(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ValidationError(c, "id", "invalid UUID format")
	}

	var req models.UpdateEmbedTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	template, err := h.embedService.UpdateTemplate(c.Context(), userID, id, &req)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}
		return HandleServiceError(c, err)
	}

	return c.JSON(template.ToTemplateResponse())
}

// DeleteTemplate deletes an embed template
// @Summary Delete an embed template
// @Description Deletes an embed template
// @Tags Embeds
// @Param id path string true "Template ID"
// @Success 204 "Template deleted"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /embeds/templates/{id} [delete]
func (h *EmbedHandler) DeleteTemplate(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ValidationError(c, "id", "invalid UUID format")
	}

	if err := h.embedService.DeleteTemplate(c.Context(), userID, id); err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
		}
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
