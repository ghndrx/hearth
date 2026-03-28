package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// SlashCommandHandler handles slash command HTTP requests
type SlashCommandHandler struct {
	slashCmdService *services.SlashCommandService
	permService     services.PermissionServiceInterface
}

// NewSlashCommandHandler creates a new slash command handler
func NewSlashCommandHandler(slashCmdService *services.SlashCommandService, permService services.PermissionServiceInterface) *SlashCommandHandler {
	return &SlashCommandHandler{
		slashCmdService: slashCmdService,
		permService:     permService,
	}
}

// SlashCommandResponse represents a slash command in API responses
type SlashCommandResponse struct {
	ID          string                     `json:"id"`
	Type        int                        `json:"type"`
	AppID       string                     `json:"application_id"`
	ServerID    string                     `json:"guild_id,omitempty"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Options     []*models.CommandOption    `json:"options,omitempty"`
	Permissions *models.CommandPermissions `json:"permissions,omitempty"`
	Version     string                     `json:"version"`
	CreatorID   string                     `json:"creator_id,omitempty"`
	Default     bool                       `json:"default_permission"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
}

func commandToResponse(cmd *models.SlashCommand) SlashCommandResponse {
	resp := SlashCommandResponse{
		ID:          cmd.ID.String(),
		Type:        int(cmd.Type),
		AppID:       cmd.AppID.String(),
		Name:        cmd.Name,
		Description: cmd.Description,
		Options:     cmd.Options,
		Permissions: cmd.Permissions,
		Version:     cmd.Version,
		Default:     cmd.Default,
		CreatedAt:   cmd.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   cmd.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if cmd.ServerID != nil {
		resp.ServerID = cmd.ServerID.String()
	}
	if cmd.CreatorID != nil {
		resp.CreatorID = cmd.CreatorID.String()
	}
	return resp
}

// RegisterCommandRequest represents the request to create a slash command
type RegisterCommandRequest struct {
	Type        int                        `json:"type"`
	ServerID    *string                    `json:"guild_id,omitempty"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Options     []*models.CommandOption    `json:"options,omitempty"`
	Permissions *models.CommandPermissions `json:"permissions,omitempty"`
	Default     *bool                      `json:"default_permission,omitempty"`
}

// RegisterCommand creates a new slash command
// POST /api/v1/applications/:appId/commands
func (h *SlashCommandHandler) RegisterCommand(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	var req RegisterCommandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate name
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}
	if len(req.Name) > 32 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name must be 32 characters or less",
		})
	}

	// Validate description
	if req.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "description is required",
		})
	}
	if len(req.Description) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "description must be 100 characters or less",
		})
	}

	cmd := &models.SlashCommand{
		Type:        models.CommandType(req.Type),
		Name:        req.Name,
		Description: req.Description,
		Options:     req.Options,
		Permissions: req.Permissions,
		CreatorID:   &userID,
		Default:     true,
	}
	if req.Default != nil {
		cmd.Default = *req.Default
	}
	if req.ServerID != nil {
		serverID, err := uuid.Parse(*req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}
		cmd.ServerID = &serverID
	}

	if err := h.slashCmdService.RegisterCommand(c.Context(), appID, cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(commandToResponse(cmd))
}

// GetCommands gets all slash commands for an application
// GET /api/v1/applications/:appId/commands
func (h *SlashCommandHandler) GetCommands(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	// Optional: filter by server
	var serverID *uuid.UUID
	if serverIDStr := c.Query("guild_id"); serverIDStr != "" {
		id, err := uuid.Parse(serverIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}
		serverID = &id
	}

	var commands []*models.SlashCommand
	if serverID != nil {
		commands, err = h.slashCmdService.GetServerCommands(c.Context(), *serverID)
	} else {
		commands, err = h.slashCmdService.GetCommands(c.Context(), appID)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := make([]SlashCommandResponse, len(commands))
	for i, cmd := range commands {
		response[i] = commandToResponse(cmd)
	}
	return c.JSON(fiber.Map{
		"commands": response,
	})
}

// GetCommand gets a specific slash command
// GET /api/v1/applications/:appId/commands/:commandId
func (h *SlashCommandHandler) GetCommand(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}
	commandID, err := uuid.Parse(c.Params("commandId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	cmd, err := h.slashCmdService.GetCommand(c.Context(), appID, commandID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "command not found",
		})
	}

	return c.JSON(commandToResponse(cmd))
}

// UpdateCommand updates an existing slash command
// PUT /api/v1/applications/:appId/commands/:commandId
func (h *SlashCommandHandler) UpdateCommand(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}
	commandID, err := uuid.Parse(c.Params("commandId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	var req RegisterCommandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	cmd := &models.SlashCommand{
		Name:        req.Name,
		Description: req.Description,
		Options:     req.Options,
		Permissions: req.Permissions,
		Default:     true,
	}
	if req.Default != nil {
		cmd.Default = *req.Default
	}
	if req.ServerID != nil {
		serverID, err := uuid.Parse(*req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}
		cmd.ServerID = &serverID
	}

	if err := h.slashCmdService.UpdateCommand(c.Context(), appID, commandID, cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	updated, err := h.slashCmdService.GetCommand(c.Context(), appID, commandID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(commandToResponse(updated))
}

// DeleteCommand deletes a slash command
// DELETE /api/v1/applications/:appId/commands/:commandId
func (h *SlashCommandHandler) DeleteCommand(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}
	commandID, err := uuid.Parse(c.Params("commandId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	if err := h.slashCmdService.DeleteCommand(c.Context(), appID, commandID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// BulkRegisterCommands registers multiple commands at once
// POST /api/v1/applications/:appId/commands/bulk
func (h *SlashCommandHandler) BulkRegisterCommands(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	var requests []RegisterCommandRequest
	if err := c.BodyParser(&requests); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	created := make([]SlashCommandResponse, 0, len(requests))
	for _, req := range requests {
		cmd := &models.SlashCommand{
			Name:        req.Name,
			Description: req.Description,
			Options:     req.Options,
			Permissions: req.Permissions,
			Default:     true,
		}
		if req.Default != nil {
			cmd.Default = *req.Default
		}
		if req.ServerID != nil {
			serverID, err := uuid.Parse(*req.ServerID)
			if err == nil {
				cmd.ServerID = &serverID
			}
		}
		if err := h.slashCmdService.RegisterCommand(c.Context(), appID, cmd); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "failed to register command",
				"details": err.Error(),
			})
		}
		created = append(created, commandToResponse(cmd))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"commands": created,
	})
}
