package handlers

import (
	"time"

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

// checkAppOwnership verifies the authenticated user owns the application
func (h *SlashCommandHandler) checkAppOwnership(c *fiber.Ctx, appID uuid.UUID) (uuid.UUID, error) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return uuid.Nil, err
	}
	app, err := h.slashCmdService.GetApplication(c.Context(), appID)
	if err != nil || app == nil {
		return uuid.Nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "application not found",
		})
	}
	if app.OwnerID != userID {
		return uuid.Nil, c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you do not own this application",
		})
	}
	return userID, nil
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

	userID, err := h.checkAppOwnership(c, appID)
	if err != nil {
		return err
	}

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

	_, err = h.checkAppOwnership(c, appID)
	if err != nil {
		return err
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

	_, err = h.checkAppOwnership(c, appID)
	if err != nil {
		return err
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

	_, err = h.checkAppOwnership(c, appID)
	if err != nil {
		return err
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

// GetCommandByID gets a command by its ID (no appId prefix required)
// GET /api/v1/commands/:id
func (h *SlashCommandHandler) GetCommandByID(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	cmd, err := h.slashCmdService.GetCommandByID(c.Context(), commandID)
	if err != nil || cmd == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "command not found",
		})
	}

	return c.JSON(commandToResponse(cmd))
}

// UpdateCommandByID updates a command by its ID (no appId prefix required)
// PATCH /api/v1/commands/:id
func (h *SlashCommandHandler) UpdateCommandByID(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
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

	if err := h.slashCmdService.UpdateCommandByID(c.Context(), commandID, cmd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	updated, err := h.slashCmdService.GetCommandByID(c.Context(), commandID)
	if err != nil || updated == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve updated command",
		})
	}

	return c.JSON(commandToResponse(updated))
}

// DeleteCommandByID deletes a command by its ID (no appId prefix required)
// DELETE /api/v1/commands/:id
func (h *SlashCommandHandler) DeleteCommandByID(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	if err := h.slashCmdService.DeleteCommandByID(c.Context(), commandID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SetCommandPermissions sets guild-specific permissions for a command
// POST /api/v1/commands/:id/permissions
func (h *SlashCommandHandler) SetCommandPermissions(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	var req struct {
		GuildID     string                              `json:"guild_id"`
		Permissions []*models.CommandPermissionOverride `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	guildID, err := uuid.Parse(req.GuildID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid guild ID",
		})
	}

	if err := h.slashCmdService.SetCommandPermissions(c.Context(), commandID, guildID, req.Permissions); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"command_id":  commandID.String(),
		"guild_id":    guildID.String(),
		"permissions": req.Permissions,
	})
}

// GetCommandPermissions gets guild-specific permissions for a command
// GET /api/v1/commands/:id/permissions
func (h *SlashCommandHandler) GetCommandPermissions(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	// Optional guild filter
	var guildID *uuid.UUID
	if guildIDStr := c.Query("guild_id"); guildIDStr != "" {
		id, err := uuid.Parse(guildIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid guild ID",
			})
		}
		guildID = &id
	}

	var permissions []*models.CommandPermissionOverride

	if guildID != nil {
		permissions, err = h.slashCmdService.GetCommandPermissions(c.Context(), commandID, *guildID)
	} else {
		permissions, err = h.slashCmdService.GetAllCommandPermissions(c.Context(), commandID)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"command_id":  commandID.String(),
		"permissions": permissions,
	})
}

// GetCommandOptions returns autocomplete options for a command
// GET /api/v1/commands/:id/options
func (h *SlashCommandHandler) GetCommandOptions(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid command ID",
		})
	}

	// Get the focused option name from query params for autocomplete
	focusedOption := c.Query("focused_option")
	focusedValue := c.Query("focused_value")

	options, err := h.slashCmdService.GetCommandOptions(c.Context(), commandID, focusedOption, focusedValue)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"command_id": commandID.String(),
		"options":    options,
	})
}

// ApplicationResponse represents an application in API responses
type ApplicationResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	OwnerID     string `json:"owner_id"`
	Verified    bool   `json:"verified"`
	CreatedAt   string `json:"created_at"`
}

func applicationToResponse(app *models.Application) ApplicationResponse {
	resp := ApplicationResponse{
		ID:          app.ID.String(),
		Name:        app.Name,
		Description: app.Description,
		Icon:        app.Icon,
		OwnerID:     app.OwnerID.String(),
		Verified:    app.Verified,
		CreatedAt:   app.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return resp
}

// CreateApplication creates a new application
// POST /api/v1/applications
func (h *SlashCommandHandler) CreateApplication(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}
	if len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name must be 100 characters or less",
		})
	}

	app := &models.Application{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		OwnerID:     userID,
		Verified:    false,
		CreatedAt:   time.Now(),
	}

	if err := h.slashCmdService.CreateApplication(c.Context(), app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(applicationToResponse(app))
}

// GetApplications gets all applications for the current user
// GET /api/v1/applications
func (h *SlashCommandHandler) GetApplications(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	apps, err := h.slashCmdService.GetApplicationsByOwner(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := make([]ApplicationResponse, len(apps))
	for i, app := range apps {
		response[i] = applicationToResponse(app)
	}
	return c.JSON(fiber.Map{
		"applications": response,
	})
}

// GetApplication gets a specific application
// GET /api/v1/applications/:id
func (h *SlashCommandHandler) GetApplication(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	app, err := h.slashCmdService.GetApplication(c.Context(), appID)
	if err != nil || app == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "application not found",
		})
	}
	if app.OwnerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "you do not own this application",
		})
	}

	return c.JSON(applicationToResponse(app))
}

// UpdateApplication updates an application
// PATCH /api/v1/applications/:id
func (h *SlashCommandHandler) UpdateApplication(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	_, err = h.checkAppOwnership(c, appID)
	if err != nil {
		return err
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Icon        *string `json:"icon,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	app := &models.Application{
		ID: appID,
	}
	if req.Name != nil {
		if len(*req.Name) > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "name must be 100 characters or less",
			})
		}
		app.Name = *req.Name
	}
	if req.Description != nil {
		app.Description = *req.Description
	}
	if req.Icon != nil {
		app.Icon = *req.Icon
	}

	if err := h.slashCmdService.UpdateApplication(c.Context(), app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	updated, err := h.slashCmdService.GetApplication(c.Context(), appID)
	if err != nil || updated == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve updated application",
		})
	}

	return c.JSON(applicationToResponse(updated))
}

// DeleteApplication deletes an application
// DELETE /api/v1/applications/:id
func (h *SlashCommandHandler) DeleteApplication(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid application ID",
		})
	}

	if err := h.slashCmdService.DeleteApplication(c.Context(), appID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// PermissionOverrideRequest represents a permission override entry
type PermissionOverrideRequest struct {
	ID    string `json:"id"`
	Type  int    `json:"type"` // 1 = role, 2 = user
	Allow bool   `json:"allow"`
	Deny  bool   `json:"deny"`
}
