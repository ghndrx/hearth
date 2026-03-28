package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// TemplateHandler handles template endpoints
type TemplateHandler struct {
	templateService *services.TemplateService
	serverService   *services.ServerService
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(
	templateService *services.TemplateService,
	serverService *services.ServerService,
) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		serverService:   serverService,
	}
}

// CreateTemplate creates a new template from a server
// @Summary Create template
// @Description Creates a new template from an existing server
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body models.CreateTemplateRequest true "Template data"
// @Success 201 {object} models.ServerTemplateResponse "Template created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Missing MANAGE_SERVER permission"
// @Failure 404 {object} fiber.Map "Server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/templates [post]
func (h *TemplateHandler) CreateTemplate(c *fiber.Ctx) error {
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

	var req models.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "template name is required",
		})
	}

	if len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "template name must be 100 characters or less",
		})
	}

	tmpl, err := h.templateService.CreateTemplate(
		c.Context(),
		serverID,
		userID,
		req.Name,
		req.Description,
		req.IsPublic,
	)
	if err != nil {
		if err == services.ErrServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}
		if err == services.ErrTemplateNameRequired {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "template name is required",
			})
		}
		if err == services.ErrTemplateNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "template name must be 100 characters or less",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create template",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(tmpl.ToResponse())
}

// GetTemplate returns a template by its code
// @Summary Get template
// @Description Returns a template by its unique code
// @Tags Templates
// @Produce json
// @Param code path string true "Template code"
// @Success 200 {object} models.ServerTemplateResponse "Template details"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /templates/{code} [get]
func (h *TemplateHandler) GetTemplate(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "template code is required",
		})
	}

	tmpl, err := h.templateService.GetTemplate(c.Context(), code)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get template",
		})
	}

	return c.JSON(tmpl.ToResponse())
}

// ListMyTemplates returns templates created by the current user
// @Summary List my templates
// @Description Returns templates created by the authenticated user
// @Tags Templates
// @Produce json
// @Success 200 {array} models.ServerTemplateResponse "List of templates"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/me/templates [get]
func (h *TemplateHandler) ListMyTemplates(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templates, err := h.templateService.ListMyTemplates(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list templates",
		})
	}

	response := make([]*models.ServerTemplateResponse, 0, len(templates))
	for _, t := range templates {
		response = append(response, t.ToResponse())
	}

	return c.JSON(response)
}

// ListPublicTemplates returns public templates with cursor pagination
// @Summary List public templates
// @Description Returns a paginated list of public templates
// @Tags Templates
// @Produce json
// @Param cursor query string false "Cursor for pagination"
// @Param limit query int false "Number of templates to return (max 50)"
// @Success 200 {object} models.TemplateListResponse "List of templates"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /templates [get]
func (h *TemplateHandler) ListPublicTemplates(c *fiber.Ctx) error {
	var cursor *uuid.UUID
	cursorStr := c.Query("cursor")
	if cursorStr != "" {
		id, err := uuid.Parse(cursorStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid cursor",
			})
		}
		cursor = &id
	}

	limit := c.QueryInt("limit", 20)
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	templates, nextID, err := h.templateService.ListPublicTemplates(c.Context(), cursor, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list templates",
		})
	}

	response := make([]*models.ServerTemplateResponse, 0, len(templates))
	for _, t := range templates {
		response = append(response, t.ToResponse())
	}

	return c.JSON(fiber.Map{
		"templates":   response,
		"next_cursor": nextID,
	})
}

// UpdateTemplate updates a template's metadata
// @Summary Update template
// @Description Updates a template's name, description, or visibility
// @Tags Templates
// @Accept json
// @Produce json
// @Param templateId path string true "Template ID"
// @Param body body models.UpdateTemplateRequest true "Template update data"
// @Success 200 {object} models.ServerTemplateResponse "Updated template"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Not the template creator"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /templates/{templateId} [patch]
func (h *TemplateHandler) UpdateTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := uuid.Parse(c.Params("templateId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	var req models.UpdateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get template to check ownership
	existing, err := h.templateService.GetTemplateByID(c.Context(), templateID)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get template",
		})
	}

	if existing.CreatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you can only update your own templates",
		})
	}

	tmpl, err := h.templateService.UpdateTemplate(c.Context(), templateID, req.Name, req.Description, req.IsPublic)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		if err == services.ErrTemplateNameTooLong {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "template name must be 100 characters or less",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update template",
		})
	}

	return c.JSON(tmpl.ToResponse())
}

// DeleteTemplate deletes a template
// @Summary Delete template
// @Description Deletes a template created by the authenticated user
// @Tags Templates
// @Param templateId path string true "Template ID"
// @Success 204 "Template deleted"
// @Failure 400 {object} fiber.Map "Invalid template ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Not the template creator"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /templates/{templateId} [delete]
func (h *TemplateHandler) DeleteTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	templateID, err := uuid.Parse(c.Params("templateId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid template id",
		})
	}

	// Get template to check ownership
	existing, err := h.templateService.GetTemplateByID(c.Context(), templateID)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get template",
		})
	}

	if existing.CreatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you can only delete your own templates",
		})
	}

	if err := h.templateService.DeleteTemplate(c.Context(), templateID); err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete template",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// UseTemplate creates a new server from a template
// @Summary Use template
// @Description Creates a new server using a template
// @Tags Templates
// @Accept json
// @Produce json
// @Param code path string true "Template code"
// @Param body body UseTemplateRequest true "Server creation data"
// @Success 201 {object} ServerResponse "Server created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /templates/{code}/use [post]
func (h *TemplateHandler) UseTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "template code is required",
		})
	}

	var req UseTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "server name is required",
		})
	}

	server, err := h.templateService.UseTemplate(c.Context(), code, userID, req.Name)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "template not found",
			})
		}
		if err == services.ErrInvalidTemplateData {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "template data is invalid",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create server from template",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         server.ID,
		"name":       server.Name,
		"icon_url":   server.IconURL,
		"owner_id":   server.OwnerID,
		"created_at": server.CreatedAt,
	})
}

// UseTemplateRequest is the request body for using a template to create a server
type UseTemplateRequest struct {
	Name string `json:"name"`
}
